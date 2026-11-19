package repositories

import (
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
)

type Repositories struct {
	User               domain.UserRepository
	Project            domain.ProjectRepository
	ProjectUser        domain.ProjectUserRepository
	Employee           domain.EmployeeRepository
	EmployeeUser       domain.EmployeeUserRepository
	ProjectEmployee    domain.ProjectEmployeeRepository
	Payrate            domain.PayrateRepository
	Timesheet          domain.TimesheetRepository
	Notification       domain.NotificationRepository
	Settings           domain.SettingsRepository
	Asset              domain.AssetRepository
	PushSubscription   domain.PushSubscriptionRepository
	TransferUpload     domain.TransferUploadRepository
	EmployeePayCycle   domain.EmployeePayCycleRepository
	BillingReport      domain.BillingReportRepository
	TimesheetImportJob domain.TimesheetImportJobRepository
	AuditLog           domain.AuditLogRepository
	BlacklistedToken   domain.BlacklistedTokenRepository
}

func Initialize(db *persistence.Database, eventBus domain.EventBus) *Repositories {
	repos := &Repositories{
		User:               persistence.NewUserRepository(db),
		Project:            persistence.NewProjectRepository(db),
		ProjectUser:        persistence.NewProjectUserRepository(db),
		Employee:           persistence.NewEmployeeRepository(db),
		EmployeeUser:       persistence.NewEmployeeUserRepository(db),
		ProjectEmployee:    persistence.NewProjectEmployeeRepository(db),
		Payrate:            persistence.NewPayrateRepository(db),
		Notification:       persistence.NewNotificationRepository(db),
		Settings:           persistence.NewSettingsRepository(db),
		Asset:              persistence.NewAssetRepository(db, eventBus),
		PushSubscription:   persistence.NewPushSubscriptionRepository(db.DB),
		TransferUpload:     persistence.NewTransferUploadRepository(db),
		EmployeePayCycle:   persistence.NewEmployeePayCycleRepository(db),
		BillingReport:      persistence.NewBillingReportRepository(db),
		TimesheetImportJob: persistence.NewTimesheetImportJobRepository(db),
		AuditLog:           persistence.NewAuditLogRepository(db),
		BlacklistedToken:   persistence.NewBlacklistedTokenRepository(db),
	}

	repos.Timesheet = persistence.NewTimesheetRepository(db, repos.ProjectEmployee)

	return repos
}
