package domain

import (
	"context"
	"time"
)

// ProjectStatus represents project status enum
type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusRunning   ProjectStatus = "active" // Fixed: was "running", should be "active" to match DB schema
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
)

// Project represents a client project in the domain
type Project struct {
	ID          uint   `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ClientName  string `json:"client_name" gorm:"type:varchar(255);comment:'e.g. Nha may san xuat hoa my pham VERICO'"`
	Name        string `json:"name" gorm:"type:varchar(255);not null;comment:'e.g. San xuat xa phong'"`
	Code        string `json:"code" gorm:"type:varchar(255);uniqueIndex;comment:'Initials of client_name and YYMM of created_at date or manually input by users'"`
	Description string `json:"description" gorm:"type:text;comment:'Rich text format supported'"`
	OffDays     int    `json:"off_days" gorm:"column:off_days;type:tinyint unsigned;not null;default:0;comment:'Bitmask of weekly off days: bit0=Sun,bit1=Mon,...,bit6=Sat. Default 0 = no fixed off days'"` // bitmask

	// Denormalised financial totals, scoped to this project's timesheets.
	// They are derived data: no create/update request field writes them and
	// they are recomputed — never mutated in place — by
	// services/project.AggregateRecomputeService.
	//
	// Definitions (authoritative source: timesheets):
	//   - TotalPayoutVND    = SUM(paid_amount) of timesheets with payment_status = 'paid'
	//                         — employee salary already covered by a verified customer
	//                         transfer upload (D11).
	//   - PendingPayableVND = SUM(amount) of approved timesheets whose payment_status is
	//                         still 'pending' — salary awaiting a verified upload.
	TotalPayoutVND    float64       `json:"total_payout_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Total paid to employees to date'"`
	PendingPayableVND float64       `json:"pending_payable_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Pending payment to employees to date'"`
	ProjectStatus     ProjectStatus `json:"project_status" gorm:"column:project_status;type:enum('draft','active','paused','completed','cancelled');not null;default:'draft'"`
	CreatedBy         uint          `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`

	// OwnerRevokedAt, when set, records that an admin revoked the intrinsic
	// owner access of the creator; the creator keeps only what explicit
	// project_users membership grants.
	OwnerRevokedAt *time.Time `json:"-" gorm:"column:owner_revoked_at;type:datetime(3)"`

	// Relationships
	Creator User `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// IsRunning returns true if project status is running
func (p *Project) IsRunning() bool {
	return p.ProjectStatus == ProjectStatusRunning
}

// IsDraft returns true if project status is draft
func (p *Project) IsDraft() bool {
	return p.ProjectStatus == ProjectStatusDraft
}

// GetAuditEntityType implements Auditable interface
func (p Project) GetAuditEntityType() string {
	return "project"
}

// GetAuditEntityID implements Auditable interface
func (p Project) GetAuditEntityID() uint {
	return p.ID
}

// ProjectRepository defines the interface for project persistence operations
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uint) (*Project, error)
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*Project, error)
	GetByIDWithEmployeeCount(ctx context.Context, id uint) (*ProjectWithEmployeeCount, error)
	GetByCode(ctx context.Context, code string) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters ProjectFilters) ([]*Project, error)
	ListWithEmployeeCount(ctx context.Context, filters ProjectFilters) ([]*ProjectWithEmployeeCount, error)
	Count(ctx context.Context, filters ProjectFilters) (int64, error)
	GetSummary(ctx context.Context) (*ProjectSummary, error)
	GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error)
	GetStatusCounts(ctx context.Context) (map[string]int, error)
	GetStatusCountsForCreator(ctx context.Context, createdBy uint) (map[string]int, error)
	GetStatusCountsForAccessible(ctx context.Context, userID uint) (map[string]int, error)
	SearchProjects(ctx context.Context, search string, limit int) ([]*Project, error)
	GetPendingActivatedProjects(ctx context.Context) ([]*Project, error)
	CodeExistsIncludingDeleted(ctx context.Context, code string) bool
	GetProjectFinancialAggregate(ctx context.Context, projectID uint) (*ProjectFinancialAggregate, error)
	UpdateProjectFinancialAggregate(ctx context.Context, projectID uint, aggregate ProjectFinancialAggregate, updatedAt time.Time) error
}

// ProjectFilters represents filtering options for project queries
type ProjectFilters struct {
	ProjectStatus           []ProjectStatus
	CreatedBy               *uint
	AccessibleBy            *uint // New field for filtering accessible projects (owned or shared)
	ModifiableBy            *uint // creator or explicit project_users access; excludes employee-only read paths
	SkipAccessibilityFilter bool  // Skip accessibility filter for separate query approach
	FromDate                *time.Time
	ToDate                  *time.Time
	Search                  string
	Limit                   int
	Offset                  int
	SortBy                  string
	SortOrder               string
}

// GetStatuses implements common.StatusFilter interface
func (pf *ProjectFilters) GetStatuses() []string {
	if len(pf.ProjectStatus) == 0 {
		return nil
	}
	statuses := make([]string, len(pf.ProjectStatus))
	for i, status := range pf.ProjectStatus {
		statuses[i] = string(status)
	}
	return statuses
}

// GetStatusField implements common.StatusFilter interface
func (pf *ProjectFilters) GetStatusField() string {
	return "project_status"
}

// GetFromDate implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetFromDate() *time.Time {
	return pf.FromDate
}

// GetToDate implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetToDate() *time.Time {
	return pf.ToDate
}

// GetDateField implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetDateField() string {
	return "created_at"
}

// GetCreatedBy implements common.CreatorFilter interface
func (pf *ProjectFilters) GetCreatedBy() *uint {
	return pf.CreatedBy
}

// GetCreatorField implements common.CreatorFilter interface
func (pf *ProjectFilters) GetCreatorField() string {
	return "created_by"
}

// GetSearch implements common.SearchFilter interface
func (pf *ProjectFilters) GetSearch() string {
	return pf.Search
}

// GetSearchFields implements common.SearchFilter interface
func (pf *ProjectFilters) GetSearchFields() []string {
	return []string{"name", "code", "client_name", "description"}
}

// ProjectSummary represents aggregated project metrics
type ProjectSummary struct {
	TotalActiveProjects    int64   `json:"total_active_projects"`
	TotalPayoutVND         float64 `json:"total_payout_vnd"`
	TotalPendingPayableVND float64 `json:"total_pending_payable_vnd"`
}

// ProjectFinancialAggregate carries the denormalised Project payout totals
// recomputed from a project's timesheets. See the field documentation on Project
// for what each total means; services/project.AggregateRecomputeService is the
// writer.
type ProjectFinancialAggregate struct {
	TotalPayoutVND    float64 `json:"total_payout_vnd"`
	PendingPayableVND float64 `json:"pending_payable_vnd"`
}

// PartnerProjectSummary represents project statistics for partner users
type PartnerProjectSummary struct {
	TotalProjects     int `json:"total_projects"`
	ActiveProjects    int `json:"active_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalEmployees    int `json:"total_employees"`
}

// ValidateName validates the project name
func (p *Project) ValidateName() error {
	if p.Name == "" {
		return NewValidationError("project name is required")
	}
	return nil
}

// ValidateStatus validates the project status
func (p *Project) ValidateStatus() error {
	validStatuses := map[ProjectStatus]bool{
		ProjectStatusDraft:     true,
		ProjectStatusRunning:   true,
		ProjectStatusPaused:    true,
		ProjectStatusCompleted: true,
		ProjectStatusCancelled: true,
	}

	if !validStatuses[p.ProjectStatus] {
		return NewValidationError("invalid project status")
	}
	return nil
}

// IsValid validates the entire project entity
func (p *Project) IsValid() error {
	if err := p.ValidateName(); err != nil {
		return err
	}
	if err := p.ValidateStatus(); err != nil {
		return err
	}
	return nil
}

// CanTransitionTo checks if project status transition is allowed
func (p *Project) CanTransitionTo(newStatus ProjectStatus) bool {
	currentStatus := p.ProjectStatus

	// Define allowed transitions
	allowedTransitions := map[ProjectStatus][]ProjectStatus{
		ProjectStatusDraft: {
			ProjectStatusRunning,
			ProjectStatusPaused,
			ProjectStatusCancelled,
		},
		ProjectStatusRunning: {
			ProjectStatusPaused,
			ProjectStatusCompleted,
			ProjectStatusCancelled,
		},
		ProjectStatusPaused: {
			ProjectStatusRunning,
			ProjectStatusCompleted,
			ProjectStatusCancelled,
		},
		ProjectStatusCompleted: {}, // No transitions allowed from completed
		ProjectStatusCancelled: {}, // No transitions allowed from cancelled
	}

	allowedTargets, exists := allowedTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// Additional types for repository operations

// ProjectWithEmployeeCount represents project with employee count
type ProjectWithEmployeeCount struct {
	Project
	CreatorFullname string `json:"creator_fullname"`
	EmployeeCount   int    `json:"employee_count"`
}
