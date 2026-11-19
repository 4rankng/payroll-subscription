package services

import (
	"log/slog"
	"time"

	cacheAdapter "api-server/internal/adapters/cache"
	bootstrapRepos "api-server/internal/app/bootstrap/repositories"
	appHandlers "api-server/internal/app/handlers"
	"api-server/internal/app/services"
	"api-server/internal/app/services/asset"
	"api-server/internal/app/services/auth"
	"api-server/internal/app/services/billing"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/dashboard"
	"api-server/internal/app/services/employee"
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/otp"
	"api-server/internal/app/services/passwordreset"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	pushService "api-server/internal/app/services/push"
	"api-server/internal/app/services/reporting"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/app/services/user"
	appConfig "api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/email"
	"api-server/internal/infra/events"
	"api-server/internal/infra/persistence"
	"api-server/internal/infra/persistence/repositories"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"
)

type Services struct {
	User                    *user.UserService
	PasswordResetJobManager *user.PasswordResetJobManager
	EmailPasswordReset      *passwordreset.Service
	Auth                    *auth.AuthService
	Authorization           *auth.AuthorizationService
	Dashboard               *dashboard.Service
	Project                 *project.ProjectService
	ProjectPermission       *project.ProjectPermissionService
	Employee                *employee.EmployeeService
	EmployeePermission      *employee.EmployeePermissionService
	EmployeeUser            *employee.EmployeeUserService
	EmployeeProfile         *employee.EmployeeProfileService
	ProjectEmployee         *project.ProjectEmployeeService
	Payrate                 *payroll.PayrateService
	Timesheet               *timesheet.TimesheetService
	Notification            *notification.NotificationService
	EmployeeNotifier        notification.EmployeeNotifier
	Billing                 *billing.TransferUploadService
	BillingReport           *billing.BillingReportService
	Settings                *config.SettingsService
	SettingsConfig          *config.SettingsConfigService
	ExcelConverter          *reporting.ExcelConverterService
	Asset                   *asset.AssetService
	Cache                   domain.CacheServiceUseCase
	CacheService            *infraServices.CacheService
	Email                   *notification.EmailService
	Audit                   *infraServices.AuditService
	EmployeeImport          *employee.ImportService
	EmployeeImportProgress  *infraServices.EmployeeImportProgressService
	Push                    *pushService.PushService
	BCCImport               *services.BCCImportService
	TransactionManager      *infraServices.TransactionManager
}

func Initialize(repos *bootstrapRepos.Repositories, cfg *appConfig.Config, logger *slog.Logger, db *persistence.Database, redis *persistence.RedisClient, eventBus domain.EventBus, asynqClient *asynqinfra.Client, clk clock.Clock) *Services {
	// Step 1: Initialize concrete infrastructure services first
	cacheService := infraServices.NewCacheService(redis)

	// Step 2: Create infrastructure adapters (ports)
	cachePort := cacheAdapter.NewRedisAdapter(cacheService)

	// Step 3: Create base services needed for other services
	userService := user.NewUserService(repos.User, repos.AuditLog, eventBus, cfg.Security.HashSecret, cfg.Security.HashSalt, clk)
	passwordResetJobManager := user.NewPasswordResetJobManager(userService, cacheService)
	pushSvc := pushService.NewPushService(repos.PushSubscription, cfg.Notification, logger)
	notificationService := notification.NewNotificationService(repos.Notification, repos.User, repos.Project, userService, pushSvc, logger)
	employeeNotificationService := notification.NewEmployeeNotificationService(notificationService, repos.Employee, logger)
	transactionManager := infraServices.NewTransactionManager(db.DB)
	settingsService := config.NewSettingsService(repos.Settings, transactionManager, cacheService, eventBus)
	billingService := billing.NewTransferUploadService(repos.TransferUpload, repos.EmployeePayCycle, repos.Timesheet, repos.Employee, transactionManager, notificationService, logger)
	settingsConfigService := config.NewSettingsConfigService(settingsService)
	// A committed settings change must reach the in-process business-settings
	// cache immediately: exports and statements read the beneficiary details
	// from it, and a stale value would print the previous account.
	settingsService.SetConfigCacheInvalidator(settingsConfigService)

	excelConverterService := reporting.NewExcelConverterService()
	fileStorage := storage.NewLocalFileStorage(cfg.Asset.StoragePath, cfg.Asset.BaseURL)
	assetService := asset.NewAssetService(repos.Asset, fileStorage, cfg.Asset.MaxFileSize, eventBus)
	billingReportService := billing.NewBillingReportService(repos.BillingReport, repos.EmployeePayCycle, repos.Timesheet, repos.Employee, settingsConfigService, assetService, notificationService, logger)

	// Step 4: Create domain services using infrastructure ports
	timesheetService := timesheet.NewTimesheetService(repos.Timesheet, repos.Employee, repos.Project, repos.ProjectEmployee, repos.Payrate, transactionManager, cachePort, eventBus)

	// Create email service for notification adapter (needed before adapter creation)
	emailNotificationPublisher := notification.NewEmailNotificationPublisher(repos.Notification, logger)

	// Select email provider based on environment
	var emailProvider domain.EmailDeliveryPort
	if cfg.App.Env != "production" {
		emailProvider = email.NewSandboxProvider(logger)
	} else {
		emailProvider = email.NewResendProvider(cfg.Notification.ResendAPIKey)
	}
	emailService := notification.NewEmailService(cfg.Notification, emailProvider, repos.Notification, repos.User, emailNotificationPublisher, assetService, logger)

	// Create query repositories
	timesheetAnalyticsRepo := repositories.NewTimesheetAnalyticsRepository(db.DB)

	dashboardService := dashboard.NewService(repos.User, repos.Project, repos.Employee, repos.Timesheet, timesheetAnalyticsRepo, repos.Notification, repos.AuditLog, repos.ProjectEmployee, settingsConfigService, cacheService, logger)

	// Create remaining services
	projectPermissionService := project.NewProjectPermissionService(repos.Project, repos.ProjectUser, repos.ProjectEmployee, repos.EmployeeUser, repos.User, eventBus)
	employeePermissionService := employee.NewEmployeePermissionService(repos.Employee, repos.EmployeeUser, repos.ProjectEmployee, repos.User, eventBus)
	authorizationService, err := auth.NewAuthorizationService(
		auth.GetDefaultModelPath(),
		auth.GetDefaultPolicyPath(),
		timesheetService,
		logger,
	)
	if err != nil {
		logger.Error("Failed to initialize authorization service", "error", err)
		authorizationService = nil
	}

	employeeUserService := employee.NewEmployeeUserService(repos.Employee, userService)
	employeeProfileService := employee.NewEmployeeProfileService(repos.Employee, repos.Timesheet, repos.ProjectEmployee, repos.Payrate, userService, eventBus)

	// Register global event handlers after all core services are constructed.
	auditEventHandler := events.NewAuditEventHandler(asynqClient)
	eventBus.SubscribeAll(auditEventHandler)

	// Register cache invalidation handler for async cache updates
	cacheInvalidationHandler := events.NewCacheInvalidationHandler(cacheService)
	eventBus.SubscribeAll(cacheInvalidationHandler)

	// Register employee user creation handler to decouple user creation from HTTP request path
	employeeUserCreatedHandler := events.NewEmployeeUserCreatedHandler(repos.Employee, employeeUserService)
	eventBus.SubscribeAll(employeeUserCreatedHandler)

	// Cascade employee profile identity (fullname, CCCD) onto project
	// assignment denormalized copies. The BCC import resolver indexes
	// assignments by the copied CCCD; without this subscription a profile edit
	// strands the copies and the next import reports "Không tìm thấy nhân
	// viên" for an employee who plainly exists.
	employeeIdentitySyncService := employee.NewEmployeeSyncService(repos.ProjectEmployee, repos.Employee, transactionManager, eventBus)
	eventBus.SubscribeAll(appHandlers.NewEmployeeSyncHandler(employeeIdentitySyncService))

	logger.Info("Event bus initialized with handlers",
		"audit", true,
		"cache_invalidation", true,
		"employee_user_created", true,
		"employee_identity_sync", true,
	)

	// Initialize employee import progress service
	employeeImportProgressService := infraServices.NewEmployeeImportProgressService(cacheService)

	// Create employee service config
	employeeConfig := &employee.Config{
		EmployeeRepo:             repos.Employee,
		TimesheetRepo:            repos.Timesheet,
		ProjectEmployeeRepo:      repos.ProjectEmployee,
		EmployeeUserRepo:         repos.EmployeeUser,
		UserRepo:                 repos.User,
		EmployeeDomainService:    domainServices.NewEmployeeDomainService(repos.Employee, repos.ProjectEmployee, repos.Timesheet),
		UserService:              userService,
		TransactionManager:       transactionManager,
		Events:                   eventBus,
		Cache:                    cacheService,
		Logger:                   logger,
		IsEmployeeMobileRequired: settingsConfigService.IsEmployeeMobileRequired,
	}

	employeeService := employee.NewEmployeeService(employeeConfig)

	// Initialize employee import service
	employeeImportService := employee.NewImportService(
		employeeService,
		repos.Employee,
		repos.Project,
		repos.ProjectEmployee,
		employeeImportProgressService,
	)

	projectEmployeeSvc := project.NewProjectEmployeeService(
		repos.ProjectEmployee,
		repos.Employee,
		repos.EmployeeUser,
		repos.Project,
		repos.Timesheet,
		repos.AuditLog,
		transactionManager,
		eventBus,
		timesheetService,
		cacheService,
	)

	// Email-OTP 2FA: build the pending-session store (Redis) + OTP service.
	//
	// OTP delivery ALWAYS uses the real Resend provider when OTP is enabled,
	// regardless of environment. Login depends on the code reaching the inbox,
	// and a sandbox/no-op provider would silently swallow the code and lock
	// every account out once MaxAttempts is hit. When OTP_ENABLE=false the
	// service is still constructed (cheap) but requiresOTP() short-circuits
	// and login behaves as before.
	otpEmailSender := emailProvider
	if cfg.OTP.Enabled {
		otpEmailSender = email.NewResendProvider(cfg.Notification.ResendAPIKey)
	}
	otpPendingStore := cache.NewOTPPendingStore(redis.Client, cfg.OTP.CodeTTL)
	otpFromEmail := constants.DefaultEmailSenderName + " <" + constants.DefaultEmailSenderAddress + ">"
	otpService := otp.NewOTPService(otpPendingStore, repos.User, otpEmailSender, otpFromEmail, cfg.OTP, clk, logger)

	// Self-service email password-reset (reuses otpEmailSender so reset emails
	// reach real inboxes under the same env-gating as OTP).
	pwresetTokenStore := cache.NewPasswordResetTokenStore(redis.Client, cfg.PasswordReset.TokenTTL)
	emailPasswordResetService := passwordreset.NewService(
		pwresetTokenStore, repos.User, userService, otpEmailSender, eventBus,
		constants.DefaultEmailSenderAddress, cfg.PasswordReset.ResetURL, clk, logger,
	)

	// Google OIDC nonce store (Redis) — single-use replay defense for id_tokens.
	nonceStore := cache.NewNonceStore(redis.Client, 10*time.Minute)

	// Self-hosted ALTCHA proof-of-work CAPTCHA (signed challenges + Redis
	// single-use solutions). Reuses the shared Redis client. When
	// CAPTCHA_ENABLE=false the service is still constructed (cheap) but
	// RequiredForFailures() short-circuits.
	captchaService := auth.NewCaptchaService(redis.Client, auth.CaptchaConfig{
		Enabled:       cfg.Captcha.Enabled,
		Threshold:     cfg.Captcha.Threshold,
		ChallengeTTL:  cfg.Captcha.ChallengeTTL,
		HMACSecret:    cfg.Captcha.HMACSecret,
		PoWCost:       cfg.Captcha.PoWCost,
		PoWDifficulty: cfg.Captcha.PoWDifficulty,
	})

	// Create audit service
	auditService := infraServices.NewAuditService(eventBus)

	servicesStruct := &Services{
		User:                    userService,
		PasswordResetJobManager: passwordResetJobManager,
		EmailPasswordReset:      emailPasswordResetService,
		Auth:                    auth.NewAuthService(userService, repos.Employee, repos.BlacklistedToken, eventBus, cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, otpService, cfg.OTP, cfg.Google.ClientID, nonceStore, captchaService, cacheService, logger),
		Authorization:           authorizationService,
		Dashboard:               dashboardService,
		Project:                 project.NewProjectService(repos.Project, repos.Employee, repos.ProjectEmployee, repos.Timesheet, eventBus, cacheService, logger),
		ProjectPermission:       projectPermissionService,
		Employee:                employeeService,
		EmployeePermission:      employeePermissionService,
		EmployeeUser:            employeeUserService,
		EmployeeProfile:         employeeProfileService,
		ProjectEmployee:         projectEmployeeSvc,
		Payrate:                 payroll.NewPayrateService(repos.Payrate, eventBus, db.DB),
		Timesheet:               timesheetService,
		Notification:            notificationService,
		EmployeeNotifier:        employeeNotificationService,
		Billing:                 billingService,
		BillingReport:           billingReportService,
		Settings:                settingsService,
		SettingsConfig:          settingsConfigService,
		ExcelConverter:          excelConverterService,
		Asset:                   assetService,
		Cache:                   cacheService,
		CacheService:            cacheService,
		Email:                   emailService,
		Audit:                   auditService,
		EmployeeImport:          employeeImportService,
		EmployeeImportProgress:  employeeImportProgressService,
		Push:                    pushSvc,
		BCCImport: services.NewBCCImportService(
			repos.Payrate,
			repos.Timesheet,
			timesheetService,
			repos.Asset,
			fileStorage,
			transactionManager,
			redis.Client,
			employeeService,
			employeeUserService,
			projectEmployeeSvc,
			repos.TimesheetImportJob,
			asynqClient,
		),
		TransactionManager: transactionManager,
	}

	return servicesStruct
}
