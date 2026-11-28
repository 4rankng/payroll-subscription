package employee

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/utils"
)

type EmployeeService struct {
	// Repositories - for data access only
	EmployeeRepo        domain.EmployeeRepository
	BankRepo            domain.BankRepository
	TimesheetRepo       domain.TimesheetRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	EmployeeUserRepo    domain.EmployeeUserRepository
	UserRepo            domain.UserRepository

	// Domain Services - for business logic
	EmployeeDomainService *domainServices.EmployeeDomainService

	// Application Services
	UserService *user.UserService

	// Infrastructure
	TransactionManager domain.TransactionManager
	events             domain.EventBus
	cache              domain.CacheServiceUseCase
}

func NewEmployeeService(cfg *Config) *EmployeeService {
	return &EmployeeService{
		EmployeeRepo:          cfg.EmployeeRepo,
		BankRepo:              cfg.BankRepo,
		TimesheetRepo:         cfg.TimesheetRepo,
		ProjectEmployeeRepo:   cfg.ProjectEmployeeRepo,
		EmployeeUserRepo:      cfg.EmployeeUserRepo,
		UserRepo:              cfg.UserRepo,
		EmployeeDomainService: cfg.EmployeeDomainService,
		UserService:           cfg.UserService,
		TransactionManager:    cfg.TransactionManager,
		events:                cfg.Events,
		cache:                 cfg.Cache,
	}
}

// CreateEmployee orchestrates employee creation using domain services and transaction management
func (s *EmployeeService) CreateEmployee(ctx context.Context, employee *domain.Employee, createdBy uint) (*domain.Employee, error) {
	return s.createEmployee(ctx, employee, createdBy)
}

// CreateEmployeeFromImport creates an employee from a trusted bulk-import
// workflow. It shares the transactional core with the manual write path.
func (s *EmployeeService) CreateEmployeeFromImport(ctx context.Context, employee *domain.Employee, createdBy uint) (*domain.Employee, error) {
	return s.createEmployee(ctx, employee, createdBy)
}

// ListBanks returns the seeded branch reference list for the /banks endpoint.
func (s *EmployeeService) ListBanks(ctx context.Context) ([]domain.Bank, error) {
	return s.BankRepo.List(ctx)
}

// ResolveBankName maps a raw bank name from an Excel import row to a seeded
// branch id. Unresolvable names return nil — non-fatal: the row stays
// bankless rather than half-filled.
func (s *EmployeeService) ResolveBankName(ctx context.Context, bankName string) *uint {
	if bankName == "" || s.BankRepo == nil {
		return nil
	}
	banks, err := s.BankRepo.SearchByBranchName(ctx, bankName, 1)
	if err != nil {
		observability.GetLogger().Warn("bank search failed", "bank_name", bankName, "error", err)
		return nil
	}
	if len(banks) == 0 {
		return nil
	}
	return &banks[0].ID
}

// EnsureEmployeeUserAccount creates and links a login account for an existing
// employee that predates the transactional employee-creation flow. Both writes
// share one transaction so an employee is never left without a linked account
// after a successful repair.
func (s *EmployeeService) EnsureEmployeeUserAccount(ctx context.Context, employeeID uint) error {
	return s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		employee, err := s.EmployeeRepo.GetByIDForUpdate(txCtx, employeeID)
		if err != nil {
			return err
		}
		if employee.UserID != nil {
			return nil
		}

		userID, err := s.createUserForEmployee(txCtx, employee)
		if err != nil {
			return err
		}

		employee.UserID = &userID
		if err := s.EmployeeRepo.Update(txCtx, employee); err != nil {
			return fmt.Errorf("%s: %w", constants.MsgFailedToUpdateEmployeeVN, err)
		}

		return nil
	})
}

func (s *EmployeeService) createEmployee(
	ctx context.Context,
	employee *domain.Employee,
	createdBy uint,
) (*domain.Employee, error) {
	// Orchestrate employee creation within a transaction
	result, err := s.TransactionManager.WithTransactionResult(ctx, func(txCtx context.Context) (any, error) {
		// Set created by
		employee.CreatedBy = createdBy
		return s.createEmployeeCore(txCtx, employee)
	})

	if err != nil {
		return nil, err
	}

	createdEmployee := result.(*domain.Employee)

	// Imports may own an outer transaction. Publish only after it commits so
	// consumers never observe a missing employee or audit a rolled-back import.
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, createdBy)
	event := domain.NewEmployeeCreatedEvent(ctx, createdEmployee, createdBy, actorFullName)
	afterCommitCtx := domain.WithoutTransactionContext(context.WithoutCancel(ctx))
	domain.RegisterAfterCommit(ctx, func() {
		if err := s.events.Publish(afterCommitCtx, event); err != nil {
			logger := observability.GetLogger()
			logger.Warn("Failed to publish EmployeeCreatedEvent", "employee_id", createdEmployee.ID, "error", err)
		}
	})

	return createdEmployee, nil
}

// createEmployeeCore contains the core transactional logic for creating an employee.
// It is shared between the main CreateEmployee flow and any asynchronous handlers that
// may need to participate in the same transaction in the future.
func (s *EmployeeService) createEmployeeCore(
	ctx context.Context,
	employee *domain.Employee,
) (*domain.Employee, error) {
	// Sanitize CCCD and email before persistence
	// Sanitize identifiers and contact fields
	employee.CCCD = strings.TrimSpace(employee.CCCD)
	employee.Mobile = strings.TrimSpace(employee.Mobile)
	employee.Address = strings.TrimSpace(employee.Address)
	if employee.Email != nil {
		trimmedEmail := strings.TrimSpace(*employee.Email)
		if trimmedEmail == "" {
			employee.Email = nil
		} else {
			employee.Email = &trimmedEmail
		}
	}

	// Optional bank capture is complete-or-none, and a supplied bank must be
	// a seeded branch so a bogus id surfaces as a 400 instead of a FK failure.
	if err := employee.ValidateBankingInfo(); err != nil {
		return nil, err
	}
	if employee.BankID != nil {
		if _, err := s.BankRepo.GetByID(ctx, *employee.BankID); err != nil {
			return nil, domain.NewValidationError(fmt.Sprintf("bank %d does not exist", *employee.BankID))
		}
	}

	// Normalize employee fullname to Vietnamese title case
	employee.Fullname = utils.ToVietnameseTitleCase(employee.Fullname)

	// 1. Validate employee data (domain entity validation)
	if err := employee.IsValid(); err != nil {
		return nil, err
	}

	// Duplicate mobile guard (always on).
	if employee.Mobile != "" {
		holders, err := s.EmployeeRepo.ListByMobile(ctx, employee.Mobile)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate mobile: %w", err)
		}
		if len(holders) > 0 {
			return nil, domain.NewConflictError(constants.MsgEmployeeMobileTakenVN)
		}
	}

	// 1c. Check for duplicate employee by CCCD before creating user account
	// This ensures we don't create a user if the employee already exists
	if existingEmployee, err := s.EmployeeRepo.GetByCCCD(ctx, employee.CCCD); err == nil && existingEmployee != nil {
		logger := observability.GetLogger()
		logger.Warn("Employee with CCCD already exists", "cccd", employee.CCCD, "existing_id", existingEmployee.ID)
		return nil, domain.NewConflictError(fmt.Sprintf("Employee with CCCD '%s' already exists", employee.CCCD))
	}

	// 2. Create user account for the employee
	userID, err := s.createUserForEmployee(ctx, employee)
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to create user for employee", "employee_name", employee.Fullname, "error", err)
		// Return error as-is to preserve the actual error message from the user service
		// This avoids hiding the root cause (e.g., duplicate username) behind generic error wrapping
		return nil, err
	}

	// 3. Link user to employee
	employee.UserID = &userID

	// 4. Persist employee (repository operation)
	if err := s.EmployeeRepo.Create(ctx, employee); err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to create employee", "cccd", employee.CCCD, "error", err)
		return nil, fmt.Errorf("%s: %w", constants.MsgFailedToCreateEmployeeVN, err)
	}

	// Load the referenced branch so the create response carries bank info.
	if employee.BankID != nil {
		if bank, bankErr := s.BankRepo.GetByID(ctx, *employee.BankID); bankErr == nil {
			employee.Bank = bank
		}
	}

	return employee, nil
}

// UpdateEmployee orchestrates employee update using domain services and transaction management
func (s *EmployeeService) UpdateEmployee(ctx context.Context, employee *domain.Employee, updatedBy uint) error {
	var originalEmployee *domain.Employee
	var nameChanged bool
	var cccdChanged bool

	// Orchestrate employee update within a transaction
	err := s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Get original employee data for change detection
		var err error
		originalEmployee, err = s.EmployeeRepo.GetByID(txCtx, employee.ID)
		if err != nil {
			return fmt.Errorf("failed to get original employee data: %w", err)
		}

		// Store original name for comparison after normalization
		originalName := originalEmployee.Fullname

		// Sanitize CCCD and email before persistence
		// Sanitize identifiers and contact fields
		employee.CCCD = strings.TrimSpace(employee.CCCD)
		employee.Mobile = strings.TrimSpace(employee.Mobile)
		if employee.Email != nil {
			trimmedEmail := strings.TrimSpace(*employee.Email)
			if trimmedEmail == "" {
				employee.Email = nil
			} else {
				employee.Email = &trimmedEmail
			}
		}

		// Normalize employee fullname to Vietnamese title case
		employee.Fullname = utils.ToVietnameseTitleCase(employee.Fullname)

		// Check if name changed after normalization
		nameChanged = originalName != employee.Fullname

		// Check if the CCCD changed — the assignment rows carry a copy the
		// BCC import resolves by, so it must cascade on update.
		cccdChanged = originalEmployee.CCCD != employee.CCCD

		// 1. Validate employee data (domain entity validation)
		if err := employee.IsValid(); err != nil {
			return err
		}

		// 1a. Duplicate mobile guard (self-excluded): only fires when the edit
		// moves the employee onto a number another active employee holds.
		if employee.Mobile != "" && employee.Mobile != originalEmployee.Mobile {
			holders, err := s.EmployeeRepo.ListByMobile(txCtx, employee.Mobile)
			if err != nil {
				return fmt.Errorf("failed to check duplicate mobile: %w", err)
			}
			for _, holder := range holders {
				if holder.ID != employee.ID {
					return domain.NewConflictError(constants.MsgEmployeeMobileTakenVN)
				}
			}
		}

		// 2. Update employee (repository operation)
		if err := s.EmployeeRepo.Update(txCtx, employee); err != nil {
			logger := observability.GetLogger()
			logger.Error("Failed to update employee", "employee_id", employee.ID, "error", err)
			return fmt.Errorf("%s: %w", constants.MsgFailedToUpdateEmployeeVN, err)
		}

		// 3. Sync email to linked User account (required for Google OAuth login)
		emailChanged := (employee.Email == nil) != (originalEmployee.Email == nil) ||
			(employee.Email != nil && originalEmployee.Email != nil && *employee.Email != *originalEmployee.Email)
		if emailChanged && employee.UserID != nil {
			linkedUser, err := s.UserRepo.GetByID(txCtx, *employee.UserID)
			if err != nil {
				logger := observability.GetLogger()
				logger.Warn("Failed to fetch linked user for email sync", "user_id", *employee.UserID, "error", err)
			} else {
				linkedUser.Email = employee.Email
				if err := s.UserRepo.Update(txCtx, linkedUser); err != nil {
					logger := observability.GetLogger()
					logger.Error("Failed to sync email to linked user", "user_id", *employee.UserID, "error", err)
					return fmt.Errorf("failed to sync email to user account: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Publish domain events (outside transaction)
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, updatedBy)

	// If the name or the CCCD changed, publish EmployeeNameUpdatedEvent first
	// to trigger the assignment copy sync (the event carries both values).
	if nameChanged || cccdChanged {
		nameChangeEvent := domain.NewEmployeeNameUpdatedEvent(
			ctx,
			employee.ID,
			originalEmployee.Fullname,
			employee.Fullname,
			employee.CCCD,
			updatedBy,
			actorFullName,
		)

		if err := s.events.Publish(ctx, nameChangeEvent); err != nil {
			logger := observability.GetLogger()
			logger.Warn("Failed to publish EmployeeNameUpdatedEvent", "employee_id", employee.ID, "error", err)
		}
	}

	// Always publish EmployeeUpdatedEvent for general audit purposes
	event := domain.NewEmployeeUpdatedEvent(ctx, employee, updatedBy, actorFullName, originalEmployee)
	if err := s.events.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish EmployeeUpdatedEvent", "employee_id", employee.ID, "error", err)
	}

	return nil
}

// DeleteEmployeeResult reports whether the employee's financial history required
// the record to be retained after its project assignments were removed.
type DeleteEmployeeResult struct {
	RetainedForFinancialHistory bool
}

func (s *EmployeeService) DeleteEmployee(ctx context.Context, id uint, deletedBy uint) (*DeleteEmployeeResult, error) {
	var deletedEmployee *domain.Employee
	result := &DeleteEmployeeResult{}

	// Orchestrate employee deletion within a transaction
	err := s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Get employee details before deletion for event
		var err error
		deletedEmployee, err = s.EmployeeRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to get employee for deletion: %w", err)
		}

		// Financial outcomes and approved payroll are immutable. An approved row
		// can be settled asynchronously after its bank export is created, so keep
		// the employee and only remove current project memberships in either case.
		hasProtectedTimesheets, err := s.TimesheetRepo.HasProtectedTimesheetsByEmployeeID(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to check employee protected timesheets: %w", err)
		}
		if hasProtectedTimesheets {
			if err := s.ProjectEmployeeRepo.DeleteAssignmentsByEmployeeID(txCtx, id); err != nil {
				return fmt.Errorf("%s: %w", constants.MsgFailedToDeleteEmployeeAssignmentsVN, err)
			}
			result.RetainedForFinancialHistory = true
			return nil
		}

		// No financial or approved-payroll record exists, so operational records
		// can be removed together before permanently removing the employee row.
		if err := s.ProjectEmployeeRepo.HardDeleteAssignmentsByEmployeeID(txCtx, id); err != nil {
			return fmt.Errorf("%s: %w", constants.MsgFailedToDeleteEmployeeAssignmentsVN, err)
		}
		if err := s.TimesheetRepo.HardDeleteOperationalByEmployeeID(txCtx, id); err != nil {
			return fmt.Errorf("failed to delete employee operational timesheets: %w", err)
		}
		if err := s.EmployeeRepo.Delete(txCtx, id); err != nil {
			return fmt.Errorf("%s: %w", constants.MsgFailedToDeleteEmployeeVN, err)
		}

		// The login is no longer associated with an employee, so disable it too.
		if deletedEmployee.UserID != nil {
			if err := s.UserRepo.Delete(txCtx, *deletedEmployee.UserID); err != nil {
				logger := observability.GetLogger()
				logger.Error("Failed to delete employee's user account", "employee_id", id, "user_id", *deletedEmployee.UserID, "error", err)
				return fmt.Errorf("failed to delete employee's user account: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if result.RetainedForFinancialHistory {
		actorFullName := audit.GetActorFullName(ctx, s.UserRepo, deletedBy)
		event := domain.NewEmployeeProjectAssignmentsRemovedEvent(ctx, deletedEmployee, deletedBy, actorFullName)
		if err := s.events.Publish(ctx, event); err != nil {
			logger := observability.GetLogger()
			logger.Warn("Failed to publish EmployeeProjectAssignmentsRemovedEvent", "employee_id", id, "error", err)
		}
		return result, nil
	}

	// Publish domain event (outside transaction)
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, deletedBy)
	event := domain.NewEmployeeDeletedEvent(ctx, deletedEmployee.ID, deletedEmployee.Fullname, deletedEmployee.CCCD, deletedBy, actorFullName)
	if err := s.events.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish EmployeeDeletedEvent", "employee_id", id, "error", err)
	}

	return result, nil
}

// createUserForEmployee creates a user account for an employee with auto-generated username and default password
func (s *EmployeeService) createUserForEmployee(ctx context.Context, employee *domain.Employee) (uint, error) {
	logger := observability.GetLogger()
	logger.Info("Creating user account for employee", "employee_name", employee.Fullname)

	// Generate username from employee's fullname
	baseUsername := utils.GenerateUsername(employee.Fullname)
	if baseUsername == "" {
		logger.Error("Failed to generate username", "employee_name", employee.Fullname)
		return 0, fmt.Errorf("failed to generate username for employee: %s", employee.Fullname)
	}

	logger.Info("Generated base username", "base_username", baseUsername)

	// Ensure username is unique. An error from the prefix lookup must abort
	// account creation: silently falling back to the base username guarantees
	// a duplicate-key failure at INSERT time.
	username, err := s.ensureUniqueUsername(ctx, baseUsername)
	if err != nil {
		return 0, err
	}
	logger.Info("Ensured unique username", "final_username", username)

	// Prepare email for user account
	var userEmail string
	if employee.Email != nil && *employee.Email != "" {
		userEmail = *employee.Email
	}

	// Create user using UserService
	logger.Info("Calling CreateUser", "username", username, "email", userEmail)
	user, err := s.UserService.CreateUser(ctx, dto.CreateUserRequest{
		Username: username,
		Email:    userEmail,
		Password: DefaultEmployeePassword,
		Fullname: employee.Fullname,
		Role:     string(domain.RoleNhanvien),
	})
	// Return error as-is to preserve the actual error message from the repository layer
	// This avoids hiding the root cause (e.g., duplicate username) behind generic error wrapping
	if err != nil {
		logger.Error("Failed to create user for employee",
			"employee_name", employee.Fullname,
			"username", username,
			"error_type", fmt.Sprintf("%T", err),
			"error_message", err.Error())
		return 0, err
	}

	logger.Info("User account created successfully for employee",
		"employee_name", employee.Fullname,
		"user_id", user.ID,
		"username", username)

	return user.ID, nil
}

// ensureUniqueUsername generates a unique username by appending numbers if needed.
// Uses a single DB query to fetch all existing variants with the same prefix,
// then picks the lowest available suffix in memory — O(1) DB round-trips.
// The lookup runs on the caller's transaction connection so a base username
// generated earlier in the same import transaction is detected and suffixed.
func (s *EmployeeService) ensureUniqueUsername(ctx context.Context, baseUsername string) (string, error) {
	logger := observability.GetLogger()

	existing, err := s.UserRepo.GetUsernamesByPrefix(ctx, baseUsername)
	if err != nil {
		return "", fmt.Errorf("failed to list existing usernames for %q: %w", baseUsername, err)
	}
	if len(existing) == 0 {
		// No conflicts found — base username is available.
		return baseUsername, nil
	}

	// Build a set of taken usernames for O(1) lookup.
	taken := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		taken[u] = struct{}{}
	}

	// Base username available?
	if _, conflict := taken[baseUsername]; !conflict {
		return baseUsername, nil
	}

	// Find the lowest available numeric suffix.
	for counter := 2; counter <= 1000; counter++ {
		candidate := baseUsername + strconv.Itoa(counter)
		if _, conflict := taken[candidate]; !conflict {
			logger.Debug("Generated username", "base", baseUsername, "result", candidate)
			return candidate, nil
		}
	}

	// Fallback: append a short random suffix (extremely unlikely to be needed).
	fallback := baseUsername + strconv.FormatInt(clock.Now().UnixNano()%10000, 10)
	logger.Warn("Username counter exceeded 1000, using timestamp fallback", "base", baseUsername, "fallback", fallback)
	return fallback, nil
}
