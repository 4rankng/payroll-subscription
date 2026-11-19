package dashboard

import (
	"log/slog"

	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/repositories"
)

type Service struct {
	UserRepo               domain.UserRepository
	ProjectRepo            domain.ProjectRepository
	EmployeeRepo           domain.EmployeeRepository
	TimesheetDashboard     domain.TimesheetDashboardReader
	TimesheetPayment       domain.TimesheetPaymentUpdater
	TimesheetAnalyticsRepo *repositories.TimesheetAnalyticsRepository
	NotificationRepo       domain.NotificationRepository
	AuditLogRepo           domain.AuditLogRepository
	ProjectEmployeeRepo    domain.ProjectEmployeeRepository
	SettingsConfigSvc      *config.SettingsConfigService
	CacheService           *infrastructure.CacheService
	logger                 *slog.Logger
}

func NewService(
	userRepo domain.UserRepository,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
	timesheetRepo domain.TimesheetRepository,
	timesheetAnalyticsRepo *repositories.TimesheetAnalyticsRepository,
	notificationRepo domain.NotificationRepository,
	auditLogRepo domain.AuditLogRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	settingsConfigSvc *config.SettingsConfigService,
	cacheSvc *infrastructure.CacheService,
	logger *slog.Logger,
) *Service {
	return &Service{
		UserRepo:               userRepo,
		ProjectRepo:            projectRepo,
		EmployeeRepo:           employeeRepo,
		TimesheetDashboard:     timesheetRepo,
		TimesheetPayment:       timesheetRepo,
		TimesheetAnalyticsRepo: timesheetAnalyticsRepo,
		NotificationRepo:       notificationRepo,
		AuditLogRepo:           auditLogRepo,
		ProjectEmployeeRepo:    projectEmployeeRepo,
		SettingsConfigSvc:      settingsConfigSvc,
		CacheService:           cacheSvc,
		logger:                 logger,
	}
}
