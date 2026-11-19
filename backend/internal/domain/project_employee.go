package domain

import (
	"context"
	"strings"
	"time"
)

// ProjectEmployee represents the assignment between a project and an employee
type ProjectEmployee struct {
	ID           uint      `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ProjectID    uint      `json:"project_id" gorm:"not null;type:bigint unsigned"`
	EmployeeID   uint      `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	EmployeeName string    `json:"employee_name" gorm:"not null;type:varchar(255)"`
	EmployeeCCCD string    `json:"employee_cccd" gorm:"not null;type:varchar(255);comment:'Citizen ID - Can cong cong dan (12 digits)'"`
	EmployeeCode string    `json:"employee_code" gorm:"type:varchar(255);comment:'Factory-assigned code, optional'"`
	Position     string    `json:"position" gorm:"type:varchar(100);not null;default:'phổ thông';comment:'Employee position for payrate calculation'"`
	StartDate    time.Time `json:"start_date" gorm:"type:date;not null"`

	CreatedBy uint      `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Project  Project  `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
	Employee Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`

	Creator User `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// ProjectEmployeeRepository defines the interface for project employee persistence operations
type ProjectEmployeeRepository interface {
	Create(ctx context.Context, assignment *ProjectEmployee) error
	GetByID(ctx context.Context, id uint) (*ProjectEmployee, error)
	GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*ProjectEmployee, error)
	GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*ProjectEmployee, error)
	GetCurrentAssignmentsForProjectForUpdate(ctx context.Context, projectID uint, asOfDate time.Time) ([]*ProjectEmployee, error)
	GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*ProjectEmployee, error)
	GetActiveAssignmentsByEmployeeIDs(ctx context.Context, employeeIDs []uint) ([]*ProjectEmployee, error)
	Update(ctx context.Context, assignment *ProjectEmployee) error
	// UpdatePosition updates only the position column, avoiding full-row Save() that
	// could overwrite concurrent changes to other fields.
	UpdatePosition(ctx context.Context, id uint, position string) error
	// UpdatePositionIfCurrent performs a compare-and-swap update so imports never
	// overwrite a concurrent assignment edit made after their initial read.
	UpdatePositionIfCurrent(ctx context.Context, id uint, currentPosition, newPosition string) error
	// BackdateStartDateIfLater moves an assignment's start date earlier to the
	// given date, but only while the stored start is still later
	// (compare-and-set). Returns whether the row changed. Used by imports that
	// prove earlier worked days than the assignment currently covers.
	BackdateStartDateIfLater(ctx context.Context, id uint, newStart time.Time) (bool, error)
	Delete(ctx context.Context, id uint) error
	DeleteByProjectID(ctx context.Context, projectID uint) error
	DeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error
	HardDeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error
	List(ctx context.Context, filters ProjectEmployeeFilters) ([]*ProjectEmployee, error)
	Count(ctx context.Context, filters ProjectEmployeeFilters) (int64, error)
	GetByProject(ctx context.Context, projectID uint) ([]*ProjectEmployee, error)
	GetByEmployee(ctx context.Context, employeeID uint) ([]*ProjectEmployee, error)
	GetActiveAssignments(ctx context.Context, projectID uint) ([]*ProjectEmployee, error)
	GetCurrentProjectForEmployee(ctx context.Context, employeeID uint) (*Project, error)
	GetCurrentProjectsForEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*Project, error)
	GetActiveProjectsForEmployee(ctx context.Context, employeeID uint) ([]*Project, error)
	GetActiveEmployeesForProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*Employee, error)
	GetOverlappingAssignments(ctx context.Context, employeeID uint, startDate, endDate *time.Time) ([]*ProjectEmployee, error)
	CountWorkingEmployees(ctx context.Context) (int, error)

	// HasAccessViaProject checks if user can access employee through project assignments
	HasAccessViaProject(ctx context.Context, employeeID, userID uint) (bool, error)
}

// ProjectEmployeeFilters represents filtering options for project employee queries
type ProjectEmployeeFilters struct {
	ProjectID  *uint
	EmployeeID *uint
	CreatedBy  *uint
	FromDate   *time.Time
	ToDate     *time.Time
	Status     string
	Search     string // Free-text search across employee_name, employee_cccd, employee_code
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// GetSearch implements common.SearchFilter so the persistence layer can route
// free-text search through common.FilterBuilder.ApplyVietnameseSearch (the
// shared chokepoint used by project_repository and others), keeping accent/
// case handling consistent across every search surface.
func (f ProjectEmployeeFilters) GetSearch() string {
	return strings.TrimSpace(f.Search)
}

// GetSearchFields implements common.SearchFilter — the denormalized employee
// columns on project_employees that the free-text search scans.
func (f ProjectEmployeeFilters) GetSearchFields() []string {
	return []string{"employee_name", "employee_cccd", "employee_code"}
}

// ProjectEmployeeWithDetails represents project employee with full project and employee details
type ProjectEmployeeWithDetails struct {
	ID           uint      `json:"id"`
	ProjectID    uint      `json:"project_id"`
	EmployeeID   uint      `json:"employee_id"`
	EmployeeCode string    `json:"employee_code"`
	Position     string    `json:"position"`
	StartDate    time.Time `json:"start_date"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Project details
	ProjectName   string        `json:"project_name"`
	ProjectCode   string        `json:"project_code"`
	ProjectStatus ProjectStatus `json:"project_status"`
	ClientName    string        `json:"client_name"`

	// Employee details
	EmployeeFullname string `json:"employee_fullname"`
	EmployeeCCCD     string `json:"employee_cccd"`
	EmployeeEmail    string `json:"employee_email"`
}

// ValidateProjectID validates the project ID
func (pe *ProjectEmployee) ValidateProjectID() error {
	if pe.ProjectID == 0 {
		return NewValidationError("Mã dự án là bắt buộc")
	}
	return nil
}

// ValidateEmployeeID validates the employee ID
func (pe *ProjectEmployee) ValidateEmployeeID() error {
	if pe.EmployeeID == 0 {
		return NewValidationError("Mã nhân viên là bắt buộc")
	}
	return nil
}

// ValidateDates validates start and end dates
func (pe *ProjectEmployee) ValidateDates() error {
	if pe.StartDate.IsZero() {
		return NewValidationError("Ngày bắt đầu là bắt buộc")
	}

	return nil
}

// ValidatePosition validates the position field
func (pe *ProjectEmployee) ValidatePosition() error {
	if pe.Position == "" {
		return NewValidationError("Vị trí làm việc là bắt buộc")
	}
	// Position can be any string value, no specific validation needed
	return nil
}

// IsValid validates the entire project employee assignment
func (pe *ProjectEmployee) IsValid() error {
	if err := pe.ValidateProjectID(); err != nil {
		return err
	}
	if err := pe.ValidateEmployeeID(); err != nil {
		return err
	}
	if err := pe.ValidatePosition(); err != nil {
		return err
	}
	if err := pe.ValidateDates(); err != nil {
		return err
	}
	return nil
}

// CanCreateTimesheet checks if timesheet can be created for this assignment on given date
func (pe *ProjectEmployee) CanCreateTimesheet(date time.Time) bool {
	// Date must be on or after start date
	return !date.Before(pe.StartDate)
}

// IsAssignedOnDate checks if employee was assigned to project on specific date
func (pe *ProjectEmployee) IsAssignedOnDate(date time.Time) bool {
	// Date must be on or after start date
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	startDateOnly := time.Date(pe.StartDate.Year(), pe.StartDate.Month(), pe.StartDate.Day(), 0, 0, 0, 0, pe.StartDate.Location())

	return !dateOnly.Before(startDateOnly)
}

// Additional types for repository operations

// ProjectAssignmentSummary represents assignment summary for a project
type ProjectAssignmentSummary struct {
	ProjectID         uint  `json:"project_id"`
	TotalAssignments  int64 `json:"total_assignments"`
	ActiveAssignments int64 `json:"active_assignments"`
}
