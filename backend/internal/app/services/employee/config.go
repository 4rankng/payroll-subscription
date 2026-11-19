package employee

import (
	"context"
	"log/slog"

	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// Config consolidates all dependencies for EmployeeService
// This reduces constructor complexity from 11 parameters to 1
type Config struct {
	// Repositories
	EmployeeRepo        domain.EmployeeRepository
	TimesheetRepo       domain.TimesheetRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	EmployeeUserRepo    domain.EmployeeUserRepository
	UserRepo            domain.UserRepository

	// Domain & Application Services
	EmployeeDomainService *domainServices.EmployeeDomainService
	UserService           *user.UserService

	// Infrastructure
	TransactionManager domain.TransactionManager
	Events             domain.EventBus
	Cache              domain.CacheServiceUseCase
	Logger             *slog.Logger

	// IsEmployeeMobileRequired reports the admin policy toggle that makes the
	// phone number mandatory on employee creation. Nil (tests, partial wiring)
	// means the policy is off. Wired at bootstrap to
	// config.SettingsConfigService.IsEmployeeMobileRequired.
	IsEmployeeMobileRequired func(ctx context.Context) bool
}
