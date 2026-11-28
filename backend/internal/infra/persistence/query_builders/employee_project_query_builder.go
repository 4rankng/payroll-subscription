package query_builders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// EmployeeProjectQueryBuilder handles complex query building for employee-project operations
type EmployeeProjectQueryBuilder struct {
	db *gorm.DB
}

// NewEmployeeProjectQueryBuilder creates a new employee project query builder
func NewEmployeeProjectQueryBuilder(db *gorm.DB) *EmployeeProjectQueryBuilder {
	return &EmployeeProjectQueryBuilder{
		db: db,
	}
}

// ProjectAssignment represents a project assignment for an employee
type ProjectAssignment struct {
	EmployeeID        uint      `gorm:"column:employee_id"`
	ProjectID         uint      `gorm:"column:project_id"`
	ProjectName       string    `gorm:"column:project_name"`
	ProjectCode       string    `gorm:"column:project_code"`
	ProjectClientName string    `gorm:"column:project_client_name"`
	Position          string    `gorm:"column:position"`
	StartDate         time.Time `gorm:"column:start_date"`
}

// ProjectAssignmentWithPayment represents a project assignment with payment info
type ProjectAssignmentWithPayment struct {
	EmployeeID        uint      `gorm:"column:employee_id"`
	ProjectEmployeeID uint      `gorm:"column:project_employee_id"`
	ProjectID         uint      `gorm:"column:project_id"`
	ProjectName       string    `gorm:"column:project_name"`
	ProjectCode       string    `gorm:"column:project_code"`
	ProjectClientName string    `gorm:"column:project_client_name"`
	Position          string    `gorm:"column:position"`
	StartDate         time.Time `gorm:"column:start_date"`
}

// FetchPositionsByProjectEmployee returns the position label of each given
// (project, employee) assignment from project_employees. Missing assignments
// are absent from the returned map.
func (b *EmployeeProjectQueryBuilder) FetchPositionsByProjectEmployee(ctx context.Context, pairs []domain.ProjectEmployeePair) (map[domain.ProjectEmployeePair]string, error) {
	out := make(map[domain.ProjectEmployeePair]string, len(pairs))
	if len(pairs) == 0 {
		return out, nil
	}
	projectIDs := make([]uint, 0, 8)
	employeeIDs := make([]int64, 0, 16)
	seenProjects := make(map[uint]struct{}, len(pairs))
	seenEmployees := make(map[int64]struct{}, len(pairs))
	for _, p := range pairs {
		if _, ok := seenProjects[p.ProjectID]; !ok {
			seenProjects[p.ProjectID] = struct{}{}
			projectIDs = append(projectIDs, p.ProjectID)
		}
		id := int64(p.EmployeeID)
		if _, ok := seenEmployees[id]; !ok {
			seenEmployees[id] = struct{}{}
			employeeIDs = append(employeeIDs, id)
		}
	}
	var rows []struct {
		ProjectID  uint
		EmployeeID uint
		Position   string
	}
	err := b.db.WithContext(ctx).
		Table("project_employees").
		Select("project_id, employee_id, position").
		Where("project_id IN ?", projectIDs).
		Where("employee_id IN ?", employeeIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[domain.ProjectEmployeePair{ProjectID: row.ProjectID, EmployeeID: row.EmployeeID}] = row.Position
	}
	return out, nil
}

// BuildListWithProjectsQuery builds the employee query for ListWithProjects
func (b *EmployeeProjectQueryBuilder) BuildListWithProjectsQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).Preload("User").Preload("Creator").Preload("Bank")

	// Apply filters for employee selection
	query = b.applyFilters(query, filters)

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

	return query
}

// BuildListWithAllProjectsQuery builds the employee query for ListWithAllProjects
func (b *EmployeeProjectQueryBuilder) BuildListWithAllProjectsQuery(filters domain.EmployeeFilters) *gorm.DB {
	return b.BuildListWithProjectsQuery(filters)
}

// BuildProjectAssignmentsQuery builds a query to fetch project assignments for employees
func (b *EmployeeProjectQueryBuilder) BuildProjectAssignmentsQuery(ctx context.Context, employeeIDs []uint) *gorm.DB {
	return b.db.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`project_employees.employee_id, project_employees.project_id, p.name as project_name, p.code as project_code,
			p.client_name as project_client_name, project_employees.position, project_employees.start_date`).
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.employee_id IN ? AND project_employees.start_date <= CURDATE()", employeeIDs)
}

// BuildProjectAssignmentsWithPaymentQuery builds a query to fetch project assignments with payment info
func (b *EmployeeProjectQueryBuilder) BuildProjectAssignmentsWithPaymentQuery(ctx context.Context, employeeIDs []uint) *gorm.DB {
	return b.db.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`project_employees.id as project_employee_id, project_employees.employee_id, project_employees.project_id, p.name as project_name, p.code as project_code,
			p.client_name as project_client_name, project_employees.position, project_employees.start_date`).
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.employee_id IN ? AND project_employees.start_date <= CURDATE()", employeeIDs)
}

// FetchProjectAssignments fetches project assignments for the given employee IDs
func (b *EmployeeProjectQueryBuilder) FetchProjectAssignments(ctx context.Context, employeeIDs []uint) ([]ProjectAssignment, error) {
	var assignments []ProjectAssignment
	err := b.BuildProjectAssignmentsQuery(ctx, employeeIDs).Find(&assignments).Error
	return assignments, err
}

// FetchProjectAssignmentsWithPayment fetches project assignments with payment info for the given employee IDs
func (b *EmployeeProjectQueryBuilder) FetchProjectAssignmentsWithPayment(ctx context.Context, employeeIDs []uint) ([]ProjectAssignmentWithPayment, error) {
	var assignments []ProjectAssignmentWithPayment
	err := b.BuildProjectAssignmentsWithPaymentQuery(ctx, employeeIDs).Find(&assignments).Error
	return assignments, err
}

// GroupAssignmentsByEmployee groups project assignments by employee ID
func (b *EmployeeProjectQueryBuilder) GroupAssignmentsByEmployee(assignments []ProjectAssignment) map[uint][]ProjectAssignment {
	grouped := make(map[uint][]ProjectAssignment)
	for _, assignment := range assignments {
		grouped[assignment.EmployeeID] = append(grouped[assignment.EmployeeID], assignment)
	}
	return grouped
}

// GroupAssignmentsWithPaymentByEmployee groups project assignments with payment info by employee ID
func (b *EmployeeProjectQueryBuilder) GroupAssignmentsWithPaymentByEmployee(assignments []ProjectAssignmentWithPayment) map[uint][]ProjectAssignmentWithPayment {
	grouped := make(map[uint][]ProjectAssignmentWithPayment)
	for _, assignment := range assignments {
		grouped[assignment.EmployeeID] = append(grouped[assignment.EmployeeID], assignment)
	}
	return grouped
}

// MapToEmployeeWithProject maps employees and assignments to EmployeeWithProject slice
func (b *EmployeeProjectQueryBuilder) MapToEmployeeWithProject(employees []*domain.Employee, assignments []ProjectAssignment) []*domain.EmployeeWithProject {
	projectsByEmployee := b.GroupAssignmentsByEmployee(assignments)

	var empWithProjects []*domain.EmployeeWithProject
	for _, emp := range employees {
		empWithProject := &domain.EmployeeWithProject{
			Employee: *emp,
		}

		if projects, exists := projectsByEmployee[emp.ID]; exists && len(projects) > 0 {
			firstProject := projects[0]
			empWithProject.ProjectID = &firstProject.ProjectID
			empWithProject.ProjectName = &firstProject.ProjectName
			empWithProject.ProjectCode = &firstProject.ProjectCode
			empWithProject.ProjectClientName = &firstProject.ProjectClientName
			empWithProject.Position = &firstProject.Position
			empWithProject.StartDate = &firstProject.StartDate
		}

		empWithProjects = append(empWithProjects, empWithProject)
	}

	return empWithProjects
}

// MapToEmployeeWithProjects maps employees and assignments to EmployeeWithProjects slice
func (b *EmployeeProjectQueryBuilder) MapToEmployeeWithProjects(employees []*domain.Employee, assignments []ProjectAssignmentWithPayment) []*domain.EmployeeWithProjects {
	projectsByEmployee := b.GroupAssignmentsWithPaymentByEmployee(assignments)

	var empWithProjects []*domain.EmployeeWithProjects
	for _, emp := range employees {
		empWithAllProjects := &domain.EmployeeWithProjects{
			Employee:        *emp,
			CurrentProjects: []domain.CurrentProject{},
		}

		if assignments, exists := projectsByEmployee[emp.ID]; exists {
			for _, assignment := range assignments {
				project := domain.CurrentProject{
					ProjectID:         assignment.ProjectID,
					ProjectEmployeeID: assignment.ProjectEmployeeID,
					Name:              assignment.ProjectName,
					Code:              assignment.ProjectCode,
					ClientName:        assignment.ProjectClientName,
					Position:          assignment.Position,
					StartDate:         assignment.StartDate,
				}
				empWithAllProjects.CurrentProjects = append(empWithAllProjects.CurrentProjects, project)
			}
		}

		empWithProjects = append(empWithProjects, empWithAllProjects)
	}

	return empWithProjects
}

// BuildGetByProjectQuery builds a query for getting employees by project
func (b *EmployeeProjectQueryBuilder) BuildGetByProjectQuery(projectID uint, createdBy *uint) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).
		Select("employees.*").
		Joins("INNER JOIN project_employees pe ON employees.id = pe.employee_id").
		Where("pe.project_id = ?", projectID)

	// Apply creator filter if provided
	if createdBy != nil {
		query = query.Where("employees.created_by = ?", *createdBy)
	}

	return query.Preload("Creator")
}

// missingMobilePredicate matches employees with no usable phone number on
// file. COALESCE+TRIM: legacy rows may hold NULL or whitespace even though
// the column default is ”.
const missingMobilePredicate = `COALESCE(TRIM(employees.mobile), '') = ''`

// BuildMissingMobileQuery lists employees with no mobile number on file.
// There is no pending-work gating: phone login, Zalo OTP password reset and
// self check-in all key off the number regardless of payroll activity, so
// every gap the caller can fix must surface.
func (b *EmployeeProjectQueryBuilder) BuildMissingMobileQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{})
	query = b.applyFilters(query, filters)
	query = query.Where(missingMobilePredicate)

	// Sorting/pagination follow the shared list contract.
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

// BuildMissingMobileCountQuery builds a query for counting employees with no
// mobile number on file.
func (b *EmployeeProjectQueryBuilder) BuildMissingMobileCountQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{})
	query = b.applyFilters(query, filters)
	return query.Where(missingMobilePredicate)
}

// BuildUnassignedAtDateQuery builds a query for getting unassigned employees at a specific date
func (b *EmployeeProjectQueryBuilder) BuildUnassignedAtDateQuery(atDate time.Time, filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).Preload("Creator")

	// Apply search filter if provided
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	// Find employees NOT assigned to any active project at the given date
	query = query.Where(`NOT EXISTS (
		SELECT 1 FROM project_employees pe
		WHERE pe.employee_id = employees.id
		AND pe.start_date <= ?
	)`, atDate)

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

	return query
}

// BuildUnassignedAtDateCountQuery builds a query for counting unassigned employees at a specific date
func (b *EmployeeProjectQueryBuilder) BuildUnassignedAtDateCountQuery(atDate time.Time, filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{})

	// Apply search filter if provided
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	// Find employees NOT assigned to any active project at the given date
	query = query.Where(`NOT EXISTS (
		SELECT 1 FROM project_employees pe
		WHERE pe.employee_id = employees.id
		AND pe.start_date <= ?
	)`, atDate)

	return query
}

// applyFilters applies filters to a project query
func (b *EmployeeProjectQueryBuilder) applyFilters(query *gorm.DB, filters domain.EmployeeFilters) *gorm.DB {
	// Filter by creator
	if filters.CreatedBy != nil {
		query = query.Where("employees.created_by = ?", *filters.CreatedBy)
	}

	// Filter for accessible employees (owned, shared, or assigned to projects)
	if filters.AccessibleBy != nil {
		if !filters.SkipAccessibilityFilter {
			userID := *filters.AccessibleBy
			ids, err := resolveAccessibleEmployeeIDs(b.db, userID)
			if err != nil {
				// Resolution failed: fall back to the row-local predicate (no
				// shared-ID prequery) so the request still returns correctly
				// scoped rows.
				slog.Warn("employee_access_resolution_failed", "error", err, "user_id", userID)
				query = query.Where(
					"employees.created_by = ? OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ?) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.start_date <= NOW())",
					userID, userID, userID,
				)
			} else if len(ids) == 0 {
				query = query.Where("1 = 0")
			} else {
				query = query.Where("employees.id IN (?)", ids)
			}
		}
	}

	// Filter by project (requires join)
	if filters.ProjectID != nil {
		subQuery := b.db.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id = ?", *filters.ProjectID)
		query = query.Where("employees.id IN (?)", subQuery)
	}

	// Filter by multiple projects
	if len(filters.ProjectIDs) > 0 {
		subQuery := b.db.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id IN ?", filters.ProjectIDs)
		query = query.Where("employees.id IN (?)", subQuery)
	}

	// Filter by status (working or unassigned)
	switch filters.Status {
	case "working":
		// Employees currently assigned to at least one project
		query = query.Where(`EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.start_date <= NOW()
		)`)
	case "unassigned":
		// Employees NOT assigned to any project
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.start_date <= NOW()
		)`)
	}

	// Filter by date range on created_at
	if filters.FromDate != nil {
		query = query.Where("employees.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		// Include the entire day by adding 23:59:59
		endOfDay := timeutil.EndOfDay(*filters.ToDate)
		query = query.Where("employees.created_at <= ?", endOfDay)
	}

	// Search functionality with Vietnamese normalization using virtual column
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	return query
}
