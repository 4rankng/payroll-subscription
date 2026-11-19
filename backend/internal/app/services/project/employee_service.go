package project

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	auditctx "api-server/internal/pkg/context"

	"gorm.io/gorm"
)

// CacheDeleter is a minimal interface for cache invalidation
type CacheDeleter interface {
	Delete(ctx context.Context, key string) error
}

// TimesheetRecalculator defines the interface for recalculating timesheets
type TimesheetRecalculator interface {
	RecalculateTimesheetsForAssignment(ctx context.Context, assignment *domain.ProjectEmployee, updatedBy uint) error
}

// ProjectEmployeeService demonstrates orchestration-only pattern
type ProjectEmployeeService struct {
	// Repositories for data access
	projectEmployeeRepo      domain.ProjectEmployeeRepository
	employeeRepo             domain.EmployeeRepository
	employeeUserRepo         domain.EmployeeUserRepository
	projectRepo              domain.ProjectRepository
	timesheetAssignmentCheck domain.TimesheetAssignmentChecker
	timesheetReader          domain.TimesheetReader
	auditLogRepo             domain.AuditLogRepository

	// Domain services for business logic
	assignmentService *domainServices.EmployeeAssignmentService

	// Application services
	timesheetRecalculator TimesheetRecalculator

	// Infrastructure services
	transactionManager *infrastructure.TransactionManager
	eventBus           domain.EventBus
	cache              CacheDeleter
}

// NewProjectEmployeeService creates a new refactored assignment service
func NewProjectEmployeeService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	employeeUserRepo domain.EmployeeUserRepository,
	projectRepo domain.ProjectRepository,
	timesheetRepo domain.TimesheetRepository,
	auditLogRepo domain.AuditLogRepository,
	transactionManager *infrastructure.TransactionManager,
	eventBus domain.EventBus,
	timesheetRecalculator TimesheetRecalculator,
	cache CacheDeleter,
) *ProjectEmployeeService {
	// Initialize domain services
	assignmentService := domainServices.NewEmployeeAssignmentService(projectEmployeeRepo, employeeRepo, projectRepo, timesheetRepo)

	return &ProjectEmployeeService{
		projectEmployeeRepo:      projectEmployeeRepo,
		employeeRepo:             employeeRepo,
		employeeUserRepo:         employeeUserRepo,
		projectRepo:              projectRepo,
		timesheetAssignmentCheck: timesheetRepo,
		timesheetReader:          timesheetRepo,
		auditLogRepo:             auditLogRepo,
		assignmentService:        assignmentService,
		timesheetRecalculator:    timesheetRecalculator,
		transactionManager:       transactionManager,
		eventBus:                 eventBus,
	}
}

// AssignEmployee orchestrates employee assignment (no business logic here)
func (s *ProjectEmployeeService) AssignEmployee(ctx context.Context, assignment *domain.ProjectEmployee, assignedBy uint) (*domain.ProjectEmployee, error) {
	// Preload employee and project for event publishing
	employee, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
	if err != nil {
		return nil, err
	}
	project, err := s.projectRepo.GetByID(ctx, assignment.ProjectID)
	if err != nil {
		return nil, err
	}

	// Orchestrate the operation within a transaction
	err = s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Validate assignment using domain service
		if err := s.assignmentService.ValidateAssignment(ctx, assignment); err != nil {
			return err
		}

		// 2. Set audit fields
		assignment.CreatedBy = assignedBy

		// 3. Create assignment using repository
		if err := s.projectEmployeeRepo.Create(ctx, assignment); err != nil {
			return err
		}

		// 4. Populate employee data explicitly (replaces GORM hooks)
		assignment.EmployeeName = employee.Fullname
		assignment.EmployeeCCCD = employee.CCCD
		if assignment.EmployeeCode == "" {
			assignment.EmployeeCode = employee.CCCD
		}

		// 5. Update the assignment with populated employee data
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 6. Publish ProjectEmployeeCreatedEvent for audit and notifications
		if s.eventBus != nil {
			actorFullName := auditctx.GetFullName(ctx)
			if actorFullName == "" {
				actorFullName = "Unknown" // fallback when no context available
			}
			event := domain.NewProjectEmployeeCreatedEvent(ctx, assignment, project.Name, employee.Fullname, auditctx.GetUserIDOrZero(ctx), actorFullName)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish ProjectEmployeeCreatedEvent", "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return assignment, nil
}

// UpdateAssignment orchestrates assignment updates
func (s *ProjectEmployeeService) UpdateAssignment(ctx context.Context, assignment *domain.ProjectEmployee, updatedBy uint) error {
	var existing *domain.ProjectEmployee

	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get existing assignment for audit
		var err error
		existing, err = s.projectEmployeeRepo.GetByID(ctx, assignment.ID)
		if err != nil {
			return err
		}

		// 2. Validate updated assignment using domain service (use update-specific validation)
		if err := s.assignmentService.ValidateAssignmentForUpdate(ctx, assignment); err != nil {
			return err
		}

		// 3. Update assignment using repository
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 4. Publish ProjectEmployeeUpdatedEvent
		if s.eventBus != nil {
			// Ensure related entities are available for the event
			employee, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if err == nil {
				assignment.Employee = *employee
			}
			project, err := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if err == nil {
				assignment.Project = *project
			}

			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (update)", "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 5. Invalidate assignment cache so subsequent validation sees updated data
	if s.cache != nil {
		cacheKey := fmt.Sprintf("assignment:%d:%d", assignment.ProjectID, assignment.EmployeeID)
		if err := s.cache.Delete(ctx, cacheKey); err != nil {
			observability.GetLogger().Warn("failed to invalidate assignment cache after update", "error", err)
		}
	}

	// 6. After successful transaction commit, recalculate editable timesheets if position or start date changed
	if s.timesheetRecalculator != nil && (existing.Position != assignment.Position || !existing.StartDate.Equal(assignment.StartDate)) {
		recalculator := s.timesheetRecalculator
		assignmentCopy := *assignment
		domain.RegisterAfterCommit(ctx, func() {
			if err := recalculator.RecalculateTimesheetsForAssignment(context.Background(), &assignmentCopy, updatedBy); err != nil {
				observability.GetLogger().Warn("failed to recalculate timesheets for assignment", "assignmentID", assignmentCopy.ID, "error", err)
			}
		})
	}

	return nil
}

// UpdateAssignmentPosition updates only the position field of an existing assignment,
// with full cache invalidation, event emission, and conditional timesheet recalculation.
// Uses targeted column update to avoid full-row Save() overwriting concurrent changes.
func (s *ProjectEmployeeService) UpdateAssignmentPosition(ctx context.Context, assignmentID uint, position string, updatedBy uint) error {
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.projectEmployeeRepo.GetByID(txCtx, assignmentID)
		if err != nil {
			return err
		}
		return s.updateAssignmentPosition(txCtx, existing, existing.Position, position, updatedBy)
	})
}

// UpdateAssignmentPositionIfCurrent updates a position only when it still
// matches the caller's snapshot. When ctx already carries a transaction, the
// compare-and-swap and all downstream writes join that transaction.
func (s *ProjectEmployeeService) UpdateAssignmentPositionIfCurrent(
	ctx context.Context,
	assignmentID uint,
	currentPosition string,
	newPosition string,
	updatedBy uint,
) error {
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.projectEmployeeRepo.GetByID(txCtx, assignmentID)
		if err != nil {
			return err
		}
		if existing.Position != currentPosition {
			return domain.NewConflictError("vị trí nhân viên đã thay đổi trong lúc nhập BCC")
		}
		return s.updateAssignmentPosition(txCtx, existing, currentPosition, newPosition, updatedBy)
	})
}

// BackdateAssignmentStartIfLater moves an assignment's start date EARLIER so
// it covers worked days an import proves, but only while the stored start is
// still later than the target (compare-and-set — a concurrent earlier edit
// wins). When ctx already carries a transaction, the update joins it. Cache
// invalidation, event publication, and timesheet recalculation run after
// commit, mirroring UpdateAssignmentPositionIfCurrent.
func (s *ProjectEmployeeService) BackdateAssignmentStartIfLater(
	ctx context.Context,
	assignmentID uint,
	newStart time.Time,
	updatedBy uint,
) (bool, error) {
	backdated := false
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.projectEmployeeRepo.GetByID(txCtx, assignmentID)
		if err != nil {
			return err
		}
		if !existing.StartDate.After(newStart) {
			return nil // already covers the earlier date
		}
		changed, err := s.projectEmployeeRepo.BackdateStartDateIfLater(txCtx, existing.ID, newStart)
		if err != nil || !changed {
			return err
		}
		backdated = true

		existing.StartDate = newStart
		assignmentCopy := *existing
		afterCommitCtx := domain.WithoutTransactionContext(context.WithoutCancel(ctx))
		// Register on txCtx so hooks fire only after the backdate commits —
		// registering on the outer ctx runs them immediately when no ambient
		// transaction exists, letting a concurrent validation re-cache the
		// stale start (ADR-007).
		domain.RegisterAfterCommit(txCtx, func() {
			if s.cache != nil {
				cacheKey := fmt.Sprintf("assignment:%d:%d", assignmentCopy.ProjectID, assignmentCopy.EmployeeID)
				if err := s.cache.Delete(afterCommitCtx, cacheKey); err != nil {
					observability.GetLogger().Warn("failed to invalidate assignment cache after start-date backdate", "error", err)
				}
			}

			if s.eventBus != nil {
				if employee, err := s.employeeRepo.GetByID(afterCommitCtx, assignmentCopy.EmployeeID); err == nil {
					assignmentCopy.Employee = *employee
				}
				if project, err := s.projectRepo.GetByID(afterCommitCtx, assignmentCopy.ProjectID); err == nil {
					assignmentCopy.Project = *project
				}
				event := domain.NewProjectEmployeeUpdatedEvent(afterCommitCtx, &assignmentCopy)
				if err := s.eventBus.Publish(afterCommitCtx, event); err != nil {
					observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (start-date backdate)", "error", err)
				}
			}

			if s.timesheetRecalculator != nil {
				if err := s.timesheetRecalculator.RecalculateTimesheetsForAssignment(afterCommitCtx, &assignmentCopy, updatedBy); err != nil {
					observability.GetLogger().Warn("failed to recalculate timesheets after start-date backdate", "assignmentID", assignmentCopy.ID, "error", err)
				}
			}
		})
		return nil
	})
	return backdated, err
}

func (s *ProjectEmployeeService) updateAssignmentPosition(
	ctx context.Context,
	existing *domain.ProjectEmployee,
	currentPosition string,
	newPosition string,
	updatedBy uint,
) error {
	if currentPosition == newPosition {
		return nil
	}
	if err := s.projectEmployeeRepo.UpdatePositionIfCurrent(ctx, existing.ID, currentPosition, newPosition); err != nil {
		return err
	}

	existing.Position = newPosition
	assignmentCopy := *existing
	afterCommitCtx := domain.WithoutTransactionContext(context.WithoutCancel(ctx))
	domain.RegisterAfterCommit(ctx, func() {
		if s.cache != nil {
			cacheKey := fmt.Sprintf("assignment:%d:%d", assignmentCopy.ProjectID, assignmentCopy.EmployeeID)
			if err := s.cache.Delete(afterCommitCtx, cacheKey); err != nil {
				observability.GetLogger().Warn("failed to invalidate assignment cache after position update", "error", err)
			}
		}

		if s.eventBus != nil {
			if employee, err := s.employeeRepo.GetByID(afterCommitCtx, assignmentCopy.EmployeeID); err == nil {
				assignmentCopy.Employee = *employee
			}
			if project, err := s.projectRepo.GetByID(afterCommitCtx, assignmentCopy.ProjectID); err == nil {
				assignmentCopy.Project = *project
			}
			event := domain.NewProjectEmployeeUpdatedEvent(afterCommitCtx, &assignmentCopy)
			if err := s.eventBus.Publish(afterCommitCtx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (position update)", "error", err)
			}
		}

		if s.timesheetRecalculator != nil {
			if err := s.timesheetRecalculator.RecalculateTimesheetsForAssignment(afterCommitCtx, &assignmentCopy, updatedBy); err != nil {
				observability.GetLogger().Warn("failed to recalculate timesheets after position update", "assignmentID", assignmentCopy.ID, "error", err)
			}
		}
	})

	return nil
}

// GetConflictingAssignments orchestrates conflict detection
func (s *ProjectEmployeeService) GetConflictingAssignments(ctx context.Context, employeeID uint, startDate time.Time, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to domain service - no transaction needed for read operation
	return s.assignmentService.GetConflictingAssignments(ctx, employeeID, startDate, endDate)
}

// GetAssignmentsByProject orchestrates data retrieval
func (s *ProjectEmployeeService) GetAssignmentsByProject(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByProject(ctx, projectID)
}

// GetAssignmentsByEmployee orchestrates data retrieval
func (s *ProjectEmployeeService) GetAssignmentsByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByEmployee(ctx, employeeID)
}

// ListAssignments orchestrates assignment listing with pagination
func (s *ProjectEmployeeService) ListAssignments(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.List(ctx, filters)
}

// CountAssignments orchestrates assignment counting
func (s *ProjectEmployeeService) CountAssignments(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.Count(ctx, filters)
}

// GetAssignmentByProjectAndEmployee orchestrates specific assignment retrieval
func (s *ProjectEmployeeService) GetAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByProjectAndEmployee(ctx, projectID, employeeID)
}

// GetAssignmentByID orchestrates assignment retrieval by primary key
func (s *ProjectEmployeeService) GetAssignmentByID(ctx context.Context, id uint) (*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByID(ctx, id)
}

// AssignEmployeesBatch orchestrates batch employee assignment
func (s *ProjectEmployeeService) AssignEmployeesBatch(ctx context.Context, assignments []*domain.ProjectEmployee, assignedBy uint) ([]*domain.ProjectEmployee, error) {
	var results []*domain.ProjectEmployee

	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, assignment := range assignments {
			// Delegate to single assignment method
			result, err := s.AssignEmployee(ctx, assignment, assignedBy)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})

	return results, err
}

// RemoveEmployeesBatch orchestrates batch employee removal. Assignments with
// recorded timesheets are kept; clean assignments are deleted outright.
func (s *ProjectEmployeeService) RemoveEmployeesBatch(ctx context.Context, projectID uint, requests []RemoveEmployeeRequest, removedBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, request := range requests {
			// Find the active assignment by project and employee
			assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, request.EmployeeID)
			if err != nil {
				// Return the error directly - it should already be a proper domain error with Vietnamese message
				return err
			}

			// Validate the retrieved assignment has proper data
			if assignment.EmployeeID == 0 {
				return domain.NewValidationError(constants.MsgInvalidAssignmentDataVN)
			}

			if err := s.projectEmployeeRepo.Delete(ctx, assignment.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveEmployeeRequest represents a request to remove an employee from a project
type RemoveEmployeeRequest struct {
	EmployeeID uint `json:"employee_id"`
}

// CleanupDuplicateAssignments removes duplicate assignment records for the same employee-project combination
// Keeps only the most recently updated one
func (s *ProjectEmployeeService) CleanupDuplicateAssignments(ctx context.Context) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Find all duplicate assignments (same project_id + employee_id combination)
		var duplicates []struct {
			ProjectID  uint `json:"project_id"`
			EmployeeID uint `json:"employee_id"`
			Count      int  `json:"count"`
		}

		err := tx.WithContext(ctx).
			Model(&domain.ProjectEmployee{}).
			Select("project_id, employee_id, COUNT(*) as count").
			Group("project_id, employee_id").
			Having("COUNT(*) > 1").
			Scan(&duplicates).Error

		if err != nil {
			return err
		}

		for _, dup := range duplicates {
			// Get all assignments for this project-employee combination
			assignments, err := s.projectEmployeeRepo.GetByEmployee(ctx, dup.EmployeeID)
			if err != nil {
				continue // Skip this one and continue with others
			}

			// Filter to only this project
			var projectAssignments []*domain.ProjectEmployee
			for _, assignment := range assignments {
				if assignment.ProjectID == dup.ProjectID {
					projectAssignments = append(projectAssignments, assignment)
				}
			}

			if len(projectAssignments) <= 1 {
				continue // No duplicates for this combination
			}

			// Keep the most recently updated assignment
			var keepAssignment *domain.ProjectEmployee
			for _, assignment := range projectAssignments {
				if keepAssignment == nil || assignment.UpdatedAt.After(keepAssignment.UpdatedAt) {
					keepAssignment = assignment
				}
			}

			// Delete all others
			for _, assignment := range projectAssignments {
				if assignment.ID != keepAssignment.ID {
					// Check if this assignment has timesheets
					hasTimesheets, err := s.timesheetAssignmentCheck.HasTimesheetsForAssignment(ctx, assignment.ProjectID, assignment.EmployeeID)
					if err != nil {
						continue // Skip on error
					}

					if !hasTimesheets {
						// Safe to delete
						if err := s.projectEmployeeRepo.Delete(ctx, assignment.ID); err != nil {
							continue // Skip on error, continue with others
						}
					}
				}
			}
		}

		return nil
	})
}
