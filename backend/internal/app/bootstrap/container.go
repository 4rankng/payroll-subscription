package bootstrap

import (
	"context"
	"log/slog"

	bootstrapInfra "api-server/internal/app/bootstrap/infrastructure"
	bootstrapRepos "api-server/internal/app/bootstrap/repositories"
	bootstrapServices "api-server/internal/app/bootstrap/services"
	"api-server/internal/app/services/auth"
	dbExportSvc "api-server/internal/app/services/db_export"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/scheduler"
	"api-server/internal/app/workers"
	"api-server/internal/config"
	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/events"
	"api-server/internal/infra/persistence"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"
	"api-server/internal/seed"
	"api-server/internal/transport/http/handlers"
	adminHandlers "api-server/internal/transport/http/handlers/admin"
	employeeHandlers "api-server/internal/transport/http/handlers/employee"
	pushHandlers "api-server/internal/transport/http/handlers/push"
	timesheetHandlers "api-server/internal/transport/http/handlers/timesheet"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
)

type Container struct {
	Config      *config.Config
	Logger      *slog.Logger
	DB          *persistence.Database
	Redis       *persistence.RedisClient
	Scheduler   *scheduler.Scheduler
	AsynqClient *asynqinfra.Client
	AsynqServer *asynqinfra.Server

	Repos      *bootstrapRepos.Repositories
	Services   *bootstrapServices.Services
	Handlers   *Handlers
	Middleware *Middleware
}

type Handlers struct {
	User            *handlers.UserHandler
	Auth            *handlers.AuthHandler
	Dashboard       *handlers.DashboardHandler
	Project         *handlers.ProjectHandler
	Employee        *handlers.EmployeeHandler
	EmployeeUsers   *employeeHandlers.EmployeeUsersHandler
	EmployeeProfile *handlers.EmployeeProfileHandler
	ProjectEmployee *handlers.ProjectEmployeeHandler
	AdminClock      *adminHandlers.ClockHandler
	Timesheet       *handlers.TimesheetHandler
	Payrate         *handlers.PayrateHandler
	Health          *handlers.HealthHandler
	Notification    *handlers.NotificationHandler
	Settings        *handlers.SettingsHandler
	Billing         *handlers.BillingHandler
	PaymentHistory  *handlers.PaymentHistoryHandler
	SaoKe           *handlers.SaoKeHandler
	Asset           *handlers.AssetHandler
	Cache           *handlers.CacheHandler
	Email           *handlers.EmailHandler
	Audit           *handlers.AuditHandler
	EmployeeImport  *employeeHandlers.ImportHandler
	DBExport        *handlers.DBExportHandler
	Push            *pushHandlers.Handler
	BCCImport       *timesheetHandlers.BCCImportHandler
}

type Middleware struct {
	Auth                   *middleware.AuthMiddleware
	Authorization          *middleware.AuthorizationMiddleware
	LoginRateLimit         gin.HandlerFunc
	PasswordResetRateLimit gin.HandlerFunc // per-normalized-email cap
	APIRateLimit           gin.HandlerFunc
	StrictRateLimit        gin.HandlerFunc
	TenantSemaphore        *middleware.TenantSemaphoreMiddleware
}

func NewContainer(cfg *config.Config, version string) (*Container, error) {
	infra, err := bootstrapInfra.Initialize(cfg)
	if err != nil {
		return nil, err
	}

	eventBus := events.NewWorkerPoolEventBus(100, 10000)
	repos := bootstrapRepos.Initialize(infra.DB, eventBus)

	// Seal integration secrets that were stored before a cipher key existed.
	// Without this backfill, configuring SETTINGS_SECRET_KEY later would only
	// protect future writes: the admin API returns protected values as null
	// (treated as "unchanged" on save), so an existing credential row can never
	// be rewritten through the API. Best-effort — a failure must not block boot.
	if sealed, err := repos.Settings.EncryptProtectedValues(context.Background()); err != nil {
		infra.Logger.Warn("settings secret backfill failed", "error", err)
	} else if len(sealed) > 0 {
		infra.Logger.Info("settings secret backfill sealed plaintext values", "keys", sealed)
	}

	// Ensure boot-flag settings rows exist (idempotent FirstOrCreate). A
	// never-saved optional key must read as its documented default (200)
	// instead of 404 — those 404s fed the high-error-rate alert for keys the
	// frontend fetches before any admin saves the owning panel. Best-effort —
	// a failure must not block boot.
	if err := seed.EnsureSettings(context.Background(), infra.DB.DB); err != nil {
		infra.Logger.Warn("settings boot seed failed", "error", err)
	}

	// Local-dev login accounts (admin/quanly/giamdoc/ketoan, shared dev
	// password). Non-production only — production never gets seeded
	// credentials.
	if env := cfg.App.Env; env != "production" {
		if err := seed.EnsureDevUsers(context.Background(), infra.DB.DB, cfg); err != nil {
			infra.Logger.Warn("dev users seed failed", "error", err)
		}
	}

	// Shared clock instance — must be created before services so services that
	// need deterministic time in tests receive the correct FakeClock.
	var clk clock.Clock
	env := cfg.App.Env
	if env == "production" {
		clk = clock.New()
	} else {
		clk = clock.NewAutoFake()
		clock.SetGlobal(clk)
	}

	// Initialize Asynq client before services so enqueuing services can use it.
	asynqClient, err := asynqinfra.NewClient(cfg.Asynq)
	if err != nil {
		return nil, err
	}
	asynqServer := asynqinfra.NewServer(cfg.Asynq)

	services := bootstrapServices.Initialize(repos, cfg, infra.Logger, infra.DB, infra.Redis, eventBus, asynqClient, clk)

	h := initHandlers(services, repos, infra.DB, infra.Redis, infra.Logger, version, eventBus, cfg, asynqClient, clk)
	middlewares := initMiddleware(services.Auth, services.Authorization, services.ProjectPermission, services.EmployeePermission, cfg, infra.Logger)

	aggregateRecomputeSvc := project.NewAggregateRecomputeService(repos.Project, clock.New(), infra.Logger)
	sched := initScheduler(services, aggregateRecomputeSvc, cfg, infra.Logger)

	// Register Asynq handlers and periodic tasks
	asynqHandlers := asynqinfra.NewHandlers(
		workers.NewEmployeeImportWorker(
			services.EmployeeImport,
			services.EmployeeImportProgress,
			cfg.Asset.StoragePath,
		),
		workers.NewBCCImportWorker(services.BCCImport, services.Audit),
		workers.NewAuditLogWriteWorker(repos.AuditLog),
	)
	asynqinfra.RegisterHandlers(asynqServer, asynqHandlers)
	if err := asynqinfra.RegisterPeriodicTasks(asynqServer, asynqClient); err != nil {
		return nil, err
	}

	return &Container{
		Config:      cfg,
		Logger:      infra.Logger,
		DB:          infra.DB,
		Redis:       infra.Redis,
		Scheduler:   sched,
		AsynqClient: asynqClient,
		AsynqServer: asynqServer,
		Repos:       repos,
		Services:    services,
		Handlers:    h,
		Middleware:  middlewares,
	}, nil
}

func initHandlers(services *bootstrapServices.Services, repos *bootstrapRepos.Repositories, db *persistence.Database, redis *persistence.RedisClient, logger *slog.Logger, version string, eventBus domain.EventBus, cfg *config.Config, asynqClient *asynqinfra.Client, clk clock.Clock) *Handlers {
	healthCheckers := map[string]handlers.HealthChecker{
		"database": db,
		"redis":    redis,
	}

	// Create fileStorage for handlers (used by employee/timesheet import).
	fileStorage := storage.NewLocalFileStorage(cfg.Asset.StoragePath, cfg.Asset.BaseURL)

	// Create employee import handler with Asynq client
	employeeImportHandler := employeeHandlers.NewImportHandler(
		services.EmployeeImport,
		services.EmployeeImportProgress,
		fileStorage,
		asynqClient,
		services.Audit,
	)

	return &Handlers{
		User:            handlers.NewUserHandler(services.User, services.PasswordResetJobManager, services.Audit),
		Auth:            handlers.NewAuthHandler(services.Auth, services.User, services.EmailPasswordReset, cfg.Auth.AccessTTL, cfg.App.Env == "production"),
		Dashboard:       handlers.NewDashboardHandler(services.Dashboard),
		Project:         handlers.NewProjectHandlerWithServices(services.Project, services.Payrate, services.Timesheet, services.ProjectEmployee, services.ProjectPermission, clk),
		Employee:        handlers.NewEmployeeHandlerWithServices(services.Employee, services.Timesheet, services.ProjectEmployee, services.EmployeePermission, services.Audit, clk),
		EmployeeUsers:   employeeHandlers.NewEmployeeUsersHandler(services.EmployeePermission),
		EmployeeProfile: handlers.NewEmployeeProfileHandler(services.EmployeeProfile, services.EmployeeUser),
		ProjectEmployee: handlers.NewProjectEmployeeHandler(services.ProjectEmployee, services.Employee, services.Project, services.ProjectPermission, services.EmployeePermission, logger),
		Timesheet:       handlers.NewTimesheetHandler(services.Timesheet, services.SettingsConfig, services.Project, services.ProjectEmployee, services.Payrate, services.ProjectPermission, services.EmployeePermission, services.Audit, clk),
		Payrate:         handlers.NewPayrateHandler(services.Payrate, services.Project, services.ProjectPermission, clk),
		Health:          handlers.NewHealthHandler(healthCheckers, version),
		Notification:    handlers.NewNotificationHandler(services.Notification),
		Settings:        handlers.NewSettingsHandler(services.Settings),
		Billing:         handlers.NewBillingHandler(services.Billing, services.BillingReport),
		PaymentHistory:  handlers.NewPaymentHistoryHandler(services.PaymentHistory, services.Audit),
		SaoKe:           handlers.NewSaoKeHandler(services.SaoKe, services.Audit),
		Asset:           handlers.NewAssetHandler(services.Asset),
		EmployeeImport:  employeeImportHandler,
		Cache:           handlers.NewCacheHandler(services.Cache, logger),
		Email:           handlers.NewEmailHandler(services.Email),
		Audit:           handlers.NewAuditHandler(repos.AuditLog),
		DBExport:        handlers.NewDBExportHandler(dbExportSvc.NewDBExportService(db, services.CacheService, cfg.Asset.StoragePath), services.Audit),
		Push:            pushHandlers.NewHandler(services.Push),
		AdminClock:      adminHandlers.NewClockHandler(clk, cfg.App.Env),
		BCCImport:       timesheetHandlers.NewBCCImportHandler(services.BCCImport, repos.Asset, fileStorage, services.ProjectPermission, services.Audit),
	}
}

func initMiddleware(authService *auth.AuthService, authorizationService *auth.AuthorizationService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService, cfg *config.Config, logger *slog.Logger) *Middleware {
	tenantLimit := 10
	if cfg.TenantConcurrencyLimit > 0 {
		tenantLimit = cfg.TenantConcurrencyLimit
	}

	var authorizationMiddleware *middleware.AuthorizationMiddleware
	if authorizationService != nil {
		authorizationMiddleware = middleware.NewAuthorizationMiddleware(authorizationService, projectPermissionService, employeePermissionService, cfg.OTP)
	} else {
		authorizationMiddleware = middleware.NewAuthorizationMiddleware(nil, projectPermissionService, employeePermissionService, cfg.OTP)
	}

	logger.Info("middleware initialized",
		"authorization", authorizationService != nil,
		"tenant_concurrency_limit", tenantLimit,
	)

	return &Middleware{
		Auth:                   middleware.NewAuthMiddleware(authService, cfg.App.Env == "production"),
		Authorization:          authorizationMiddleware,
		LoginRateLimit:         middleware.CreateLoginRateLimit(cfg.Redis.Addr),
		PasswordResetRateLimit: middleware.CreatePasswordResetRateLimit(cfg.Redis.Addr, cfg.PasswordReset.RateLimitPerHour),
		APIRateLimit:           middleware.CreateAPIRateLimit(cfg.Redis.Addr),
		StrictRateLimit:        middleware.CreateStrictRateLimit(cfg.Redis.Addr),
		TenantSemaphore:        middleware.NewTenantSemaphoreMiddleware(tenantLimit),
	}
}

func initScheduler(services *bootstrapServices.Services, aggregateRecomputeSvc *project.AggregateRecomputeService, cfg *config.Config, logger *slog.Logger) *scheduler.Scheduler {
	s := scheduler.NewScheduler(
		logger,
		cfg.Scheduler.Timezone,
		cfg.Scheduler.Enabled,
	)

	registerSchedulerJobs(s, aggregateRecomputeSvc, logger, services.BillingReport)

	return s
}
