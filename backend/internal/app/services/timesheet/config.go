package timesheet

import (
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	domainServices "api-server/internal/domain/services"
	"log/slog"
)

// Config consolidates all dependencies for TimesheetService
type Config struct {
	// Repositories
	TimesheetRepo       domain.TimesheetRepository
	EmployeeRepo        domain.EmployeeRepository
	ProjectRepo         domain.ProjectRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	PayrateRepo         domain.PayrateRepository

	// Domain Services
	ValidationService          *domainServices.TimesheetValidationService
	CalculationService         *domainServices.PayrollCalculationService
	AssignmentService          *domainServices.EmployeeAssignmentService
	PaytypeConstructionService *domainServices.PaytypeConstructionService
	TimesheetDomainService     *domainServices.TimesheetDomainService

	// Infrastructure
	TransactionManager      *infrastructure.TransactionManager
	TransactionOrchestrator *TransactionOrchestrator
	Events                  domain.EventBus
	Cache                   infraports.CachePort
	Logger                  *slog.Logger
}

// NewTimesheetServiceFromConfig creates a TimesheetService from config

// Initialize bulk operations with transaction support
