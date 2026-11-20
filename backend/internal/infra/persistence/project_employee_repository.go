package persistence

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectEmployeeRepository struct {
	*BaseRepository
	filterBuilder *common.FilterBuilder
}

func NewProjectEmployeeRepository(db *Database) domain.ProjectEmployeeRepository {
	return &ProjectEmployeeRepository{
		BaseRepository: NewBaseRepository(db),
		filterBuilder:  common.NewFilterBuilder(db.DB),
	}
}

// getDB returns the transaction-aware DB session. If a transaction is active in
// the context (set via domain.WithTransactionContext), it uses that TX; otherwise
// it falls back to the repository's own DB connection.
func (r *ProjectEmployeeRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func withAssignmentUpdateLock(ctx context.Context, query *gorm.DB) *gorm.DB {
	if _, inTransaction := domain.GetTransactionFromContext(ctx); inTransaction {
		return query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return query
}

// applyCommonPreloads centralizes the loading of common related entities
func (r *ProjectEmployeeRepository) applyCommonPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Project").
		Preload("Employee").
		Preload("Creator")
}

func (r *ProjectEmployeeRepository) Create(ctx context.Context, assignment *domain.ProjectEmployee) error {
	return r.SafeCreate(ctx, assignment)
}

func (r *ProjectEmployeeRepository) GetByID(ctx context.Context, id uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	query := r.getDB(ctx)
	err := r.applyCommonPreloads(query).First(&assignment, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

func (r *ProjectEmployeeRepository) GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("project_id = ? AND employee_id = ?", projectID, employeeID)
	err := r.applyCommonPreloads(query).First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

// GetActiveAssignmentByProjectAndEmployee returns the employee's current
// assignment row for the project. Assignment rows are open-ended and unique
// per (project, employee) pair, so the newest row is the active one.
func (r *ProjectEmployeeRepository) GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee

	query := r.getDB(ctx).
		Where("project_id = ? AND employee_id = ?", projectID, employeeID).
		Order("created_at DESC, id DESC")
	query = withAssignmentUpdateLock(ctx, query)
	err := query.First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgActiveAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

func (r *ProjectEmployeeRepository) GetCurrentAssignmentsForProjectForUpdate(
	ctx context.Context,
	projectID uint,
	asOfDate time.Time,
) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	err := r.getDB(ctx).
		Where("project_id = ?", projectID).
		Where("start_date <= ?", asOfDate).
		Order("id ASC").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&assignments).Error
	return assignments, err
}

func (r *ProjectEmployeeRepository) GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*domain.ProjectEmployee, error) {
	if len(projectIDs) == 0 || len(employeeIDs) == 0 {
		return []*domain.ProjectEmployee{}, nil
	}

	var assignments []*domain.ProjectEmployee

	// Assignment rows are open-ended, so every row covering the given ids is
	// active. The composite index idx_project_employees_project_employee
	// serves this lookup.
	query := r.getDB(ctx).
		Where("project_id IN ? AND employee_id IN ?", projectIDs, employeeIDs)

	err := query.Find(&assignments).Error

	if err != nil {
		return nil, err
	}

	return assignments, nil
}

// GetActiveAssignmentsByEmployeeIDs returns every active assignment for the
// given employees. Used by the vendor read API to attach project codes to an
// employee page in one query instead of one lookup per row.
func (r *ProjectEmployeeRepository) GetActiveAssignmentsByEmployeeIDs(ctx context.Context, employeeIDs []uint) ([]*domain.ProjectEmployee, error) {
	if len(employeeIDs) == 0 {
		return []*domain.ProjectEmployee{}, nil
	}

	var assignments []*domain.ProjectEmployee

	query := r.getDB(ctx).
		Where("employee_id IN ?", employeeIDs)

	if err := query.Find(&assignments).Error; err != nil {
		return nil, err
	}

	return assignments, nil
}

func (r *ProjectEmployeeRepository) Update(ctx context.Context, assignment *domain.ProjectEmployee) error {
	// Omit relationship structs to prevent Save() from overwriting foreign keys
	// with zero values when the assignment was fetched without preloads.
	db := r.getDB(ctx).Omit("Project", "Employee", "Creator")
	if err := db.Save(assignment).Error; err != nil {
		return fmt.Errorf("failed to update project employee: %w", err)
	}
	return nil
}

func (r *ProjectEmployeeRepository) UpdatePosition(ctx context.Context, id uint, position string) error {
	result := r.getDB(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id = ?", id).
		Update("position", position)
	if result.Error != nil {
		return fmt.Errorf("failed to update assignment position: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewNotFoundError("phân công không tồn tại")
	}
	return nil
}

func (r *ProjectEmployeeRepository) UpdatePositionIfCurrent(
	ctx context.Context,
	id uint,
	currentPosition string,
	newPosition string,
) error {
	result := r.getDB(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id = ? AND position = ?", id, currentPosition).
		Update("position", newPosition)
	if result.Error != nil {
		return fmt.Errorf("failed to update assignment position: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewConflictError("vị trí nhân viên đã thay đổi trong lúc nhập BCC")
	}
	return nil
}

func (r *ProjectEmployeeRepository) BackdateStartDateIfLater(
	ctx context.Context,
	id uint,
	newStart time.Time,
) (bool, error) {
	result := r.getDB(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id = ? AND start_date > ?", id, newStart).
		Update("start_date", newStart)
	if result.Error != nil {
		return false, fmt.Errorf("failed to backdate assignment start: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *ProjectEmployeeRepository) Delete(ctx context.Context, id uint) error {
	return r.SafeDelete(ctx, &domain.ProjectEmployee{}, id)
}

func (r *ProjectEmployeeRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return r.DB.WithContext(ctx).
		Where("project_id = ?", projectID).
		Delete(&domain.ProjectEmployee{}).Error
}

func (r *ProjectEmployeeRepository) List(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee

	// Special handling for project employee listing to avoid duplicates
	if filters.ProjectID != nil {
		return r.getDistinctProjectEmployees(ctx, filters)
	}

	query := r.DB.WithContext(ctx)
	query = r.applyCommonPreloads(query)

	// Apply filters
	query = r.applyFilters(query, filters)

	// Apply sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&assignments).Error
	return assignments, err
}

func (r *ProjectEmployeeRepository) Count(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	var count int64

	// Special handling for project employee counting to avoid duplicates
	if filters.ProjectID != nil {
		return r.countDistinctProjectEmployees(ctx, filters)
	}

	query := r.DB.WithContext(ctx).Model(&domain.ProjectEmployee{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *ProjectEmployeeRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("project_id = ?", projectID)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) GetByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("employee_id = ?", employeeID)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) GetActiveAssignments(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee

	query := r.getDB(ctx).Where("project_id = ?", projectID)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) applyFilters(query *gorm.DB, filters domain.ProjectEmployeeFilters) *gorm.DB {
	// Filter by project
	if filters.ProjectID != nil {
		query = query.Where("project_id = ?", *filters.ProjectID)

		today := timeutil.StartOfDay(clock.NowUTC())
		// Filter by status
		switch filters.Status {
		case "current":
			query = query.Where("start_date <= ?", today)
		case "upcoming":
			query = query.Where("start_date > ?", today)
		default:
			// Rows are open-ended: every assignment is current or upcoming.
		}
	}

	// Filter by employee
	if filters.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filters.EmployeeID)
	}

	// Filter by creator
	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}

	// Filter by date range
	if filters.FromDate != nil {
		query = query.Where("start_date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("start_date <= ?", *filters.ToDate)
	}

	// Free-text search across denormalized employee fields. Routed through the
	// shared FilterBuilder.ApplyVietnameseSearch chokepoint (same as
	// project_repository) so case/accent handling is consistent across every
	// search surface and the OR clause is grouped safely.
	query = r.filterBuilder.ApplyVietnameseSearch(query, &filters)

	return query
}

func (r *ProjectEmployeeRepository) GetActiveEmployeesForProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*domain.Employee, error) {
	var employees []*domain.Employee

	query := r.DB.WithContext(ctx).
		Model(&domain.Employee{}).
		Select("employees.*").
		Joins("INNER JOIN project_employees pe ON employees.id = pe.employee_id").
		Where("pe.project_id = ?", projectID)

	// Apply creator filter if provided
	if createdBy != nil {
		query = query.Where("employees.created_by = ?", *createdBy)
	}

	err := query.Find(&employees).Error
	return employees, err
}

func (r *ProjectEmployeeRepository) GetActiveProjectsForEmployee(ctx context.Context, employeeID uint) ([]*domain.Project, error) {
	var projects []*domain.Project

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("projects.*").
		Joins("INNER JOIN project_employees pe ON projects.id = pe.project_id").
		Where("pe.employee_id = ? AND projects.project_status = ?",
			employeeID, domain.ProjectStatusRunning).
		Find(&projects).Error

	return projects, err
}

// GetOverlappingAssignments returns assignments that overlap with given date range (pure data access)
func (r *ProjectEmployeeRepository) GetOverlappingAssignments(ctx context.Context, employeeID uint, startDate, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("employee_id = ?", employeeID)
	query = r.applyCommonPreloads(query)

	// Assignment rows are open-ended, so an assignment overlaps the window
	// whenever it starts on or before the window's end; an open-ended window
	// overlaps every assignment of the employee.
	if endDate != nil {
		query = query.Where("start_date <= ?", *endDate)
	}

	err := query.Find(&assignments).Error
	return assignments, err
}

// GetCurrentProjectForEmployee returns the current active project for an employee
func (r *ProjectEmployeeRepository) GetCurrentProjectForEmployee(ctx context.Context, employeeID uint) (*domain.Project, error) {
	var project domain.Project

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("projects.*").
		Joins("INNER JOIN project_employees pe ON projects.id = pe.project_id").
		Where("pe.employee_id = ?", employeeID).
		Preload("Creator").
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No current project
		}
		return nil, err
	}

	return &project, nil
}

// GetCurrentProjectsForEmployees retrieves current projects for multiple employees in batch
func (r *ProjectEmployeeRepository) GetCurrentProjectsForEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*domain.Project, error) {
	if len(employeeIDs) == 0 {
		return make(map[uint]*domain.Project), nil
	}

	type ProjectEmployeeMapping struct {
		EmployeeID uint
		ProjectID  uint
	}

	var mappings []ProjectEmployeeMapping
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select("employee_id, project_id").
		Where("employee_id IN ?", employeeIDs).
		Scan(&mappings).Error

	if err != nil {
		return nil, err
	}

	if len(mappings) == 0 {
		return make(map[uint]*domain.Project), nil
	}

	// Extract unique project IDs
	projectIDs := make([]uint, 0, len(mappings))
	projectIDSet := make(map[uint]bool)
	for _, m := range mappings {
		if !projectIDSet[m.ProjectID] {
			projectIDs = append(projectIDs, m.ProjectID)
			projectIDSet[m.ProjectID] = true
		}
	}

	// Fetch all projects at once
	var projects []*domain.Project
	err = r.DB.WithContext(ctx).
		Where("id IN ?", projectIDs).
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	// Create project lookup map
	projectMap := make(map[uint]*domain.Project)
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	// Map employees to their projects
	result := make(map[uint]*domain.Project)
	for _, m := range mappings {
		if project, exists := projectMap[m.ProjectID]; exists {
			result[m.EmployeeID] = project
		}
	}

	return result, nil
}

// getDistinctProjectEmployees returns unique employees for a project using window function
func (r *ProjectEmployeeRepository) getDistinctProjectEmployees(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	today := timeutil.StartOfDay(clock.NowUTC())

	// Subquery with window function to rank assignments per employee
	subQuery := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`id, ROW_NUMBER() OVER(
			PARTITION BY employee_id
			ORDER BY start_date DESC
		) as rn`).
		Where("project_id = ?", *filters.ProjectID)

	// Apply status filters to the subquery
	switch filters.Status {
	case "current":
		subQuery = subQuery.Where("start_date <= ?", today)
	case "upcoming":
		subQuery = subQuery.Where("start_date > ?", today)
	default:
		// Rows are open-ended: every assignment is current or upcoming.
	}

	// Free-text search across denormalized employee fields
	subQuery = r.filterBuilder.ApplyVietnameseSearch(subQuery, &filters)

	// Main query joins with the subquery to get only the top-ranked assignment per employee
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).
		Joins("JOIN (?) AS ranked_assignments ON project_employees.id = ranked_assignments.id AND ranked_assignments.rn = 1", subQuery).
		// Left join to get the most recent timesheet date per employee for this project
		Joins(`LEFT JOIN (
			SELECT employee_id, MAX(date) AS last_timesheet_date
			FROM timesheets
			WHERE project_id = ?
			GROUP BY employee_id
		) AS ts_summary ON ts_summary.employee_id = project_employees.employee_id`, *filters.ProjectID)

	// Apply creator filter if provided
	if filters.CreatedBy != nil {
		query = query.Joins("INNER JOIN employees e ON project_employees.employee_id = e.id").
			Where("e.created_by = ?", *filters.CreatedBy)
	}

	query = r.applyCommonPreloads(query)

	// Sort by the most recent of: assignment created_at vs last timesheet date.
	// This puts recently active employees first regardless of when they were assigned.
	query = query.Order("GREATEST(project_employees.created_at, COALESCE(ts_summary.last_timesheet_date, '0001-01-01')) DESC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&assignments).Error
	return assignments, err
}

// countDistinctProjectEmployees returns count of unique employees for a project
func (r *ProjectEmployeeRepository) countDistinctProjectEmployees(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	today := timeutil.StartOfDay(clock.NowUTC())

	query := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select("COUNT(DISTINCT project_employees.employee_id)").
		Where("project_employees.project_id = ?", *filters.ProjectID)

	// Apply creator filter if provided
	if filters.CreatedBy != nil {
		query = query.Joins("INNER JOIN employees e ON project_employees.employee_id = e.id").
			Where("e.created_by = ?", *filters.CreatedBy)
	}

	// Apply status filters
	switch filters.Status {
	case "current":
		query = query.Where("start_date <= ?", today)
	case "upcoming":
		query = query.Where("start_date > ?", today)
	default:
		// Rows are open-ended: every assignment is current or upcoming.
	}

	// Free-text search across denormalized employee fields
	query = r.filterBuilder.ApplyVietnameseSearch(query, &filters)

	var count int64
	err := query.Scan(&count).Error
	return count, err
}

// DeleteAssignmentsByEmployeeID deletes all project assignments for an employee
func (r *ProjectEmployeeRepository) DeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error {
	return r.getDB(ctx).
		Where("employee_id = ?", employeeID).
		Delete(&domain.ProjectEmployee{}).Error
}

func (r *ProjectEmployeeRepository) HardDeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error {
	return r.dbForContext(ctx).
		Unscoped().
		Where("employee_id = ?", employeeID).
		Delete(&domain.ProjectEmployee{}).Error
}

// CountWorkingEmployees returns the count of distinct employees with assignments that have started
func (r *ProjectEmployeeRepository) CountWorkingEmployees(ctx context.Context) (int, error) {
	// Use DB/local timezone and include only assignments that have started
	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("start_date <= ?", today).
		Distinct("employee_id").
		Count(&count).Error

	return int(count), err
}

// HasAccessViaProject checks if user can access employee through project assignments.
//
// Access via a project is granted if the employee is ACTIVELY assigned to a project
// the user can access. A user "can access a project" when they are an explicit project
// member (project_users) OR they CREATED the project. Project creators are intentionally
// not stored in project_users — see ProjectPermissionService.CanUserAccessProject, which
// special-cases projects.created_by, and GetProjectUsers, which synthesizes a virtual
// owner entry. This query must mirror that rule: without the projects.created_by branch,
// a partner cannot view employees assigned to their OWN project (regression: partner 403
// "No access to this employee" on GET /employees/:id).
func (r *ProjectEmployeeRepository) HasAccessViaProject(ctx context.Context, employeeID, userID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Joins("INNER JOIN projects ON projects.id = project_employees.project_id").
		// LEFT JOIN so a project creator matches even with no project_users row.
		Joins("LEFT JOIN project_users ON project_users.project_id = project_employees.project_id AND project_users.user_id = ?", userID).
		Where("project_employees.employee_id = ?", employeeID).
		Where("project_employees.start_date <= NOW()").
		// Creator of the project OR explicit project member.
		Where("projects.created_by = ? OR project_users.user_id IS NOT NULL", userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}
