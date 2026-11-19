package repositories

import (
	"api-server/internal/pkg/clock"
	"context"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/infra/persistence/query_builders"

	"gorm.io/gorm"
)

// TimesheetAnalyticsRepository handles analytics and reporting operations for timesheets
type TimesheetAnalyticsRepository struct {
	db           *gorm.DB
	queryBuilder *query_builders.TimesheetQueryBuilder
	errorHandler *common.RepoErrorHandler
}

type pendingPaymentSummaryResult struct {
	PendingPaymentAmount    int64 `gorm:"column:pending_payment_amount"`
	PendingPaymentEmployees int   `gorm:"column:pending_payment_employees"`
}

// NewTimesheetAnalyticsRepository creates a new analytics repository
func NewTimesheetAnalyticsRepository(db *gorm.DB) *TimesheetAnalyticsRepository {
	return &TimesheetAnalyticsRepository{
		db:           db,
		queryBuilder: query_builders.NewTimesheetQueryBuilder(db),
		errorHandler: common.NewRepoErrorHandler(),
	}
}

// GetSummaryStats retrieves summary statistics for timesheets matching filters.
// Collapses the original 7 separate queries into 2 round-trips:
//   - Query 1: all counts, paid totals, employee counts, and last_updated in one pass.
//   - Query 2: pending payment amount + employee count (requires different status filters).
func (r *TimesheetAnalyticsRepository) GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	var stats domain.TimesheetSummaryStats

	// Query 1: aggregate everything that shares the same base filter set.
	type mainResult struct {
		PendingApproval int       `gorm:"column:pending_approval"`
		ApprovedEntries int       `gorm:"column:approved_entries"`
		RejectedEntries int       `gorm:"column:rejected_entries"`
		PaidEntries     int       `gorm:"column:paid_entries"`
		PaidAmount      int64     `gorm:"column:paid_amount"`
		PaidEmployees   int       `gorm:"column:paid_employees"`
		TotalEmployees  int       `gorm:"column:total_employees"`
		LastUpdated     time.Time `gorm:"column:last_updated"`
	}
	var main mainResult
	err := r.queryBuilder.BuildSummaryQuery(filters).
		Select(`
			COALESCE(SUM(timesheet_status = 'pending_approval'), 0)                                                        AS pending_approval,
			COALESCE(SUM(timesheet_status = 'approved'), 0)                                                               AS approved_entries,
			COALESCE(SUM(timesheet_status = 'rejected'), 0)                                                                AS rejected_entries,
			COALESCE(SUM(payment_status = 'paid'), 0)                                                                      AS paid_entries,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN paid_amount ELSE 0 END), 0)                                AS paid_amount,
			COUNT(DISTINCT CASE WHEN payment_status = 'paid' THEN timesheets.employee_id END)                              AS paid_employees,
			COUNT(DISTINCT timesheets.employee_id)                                                                         AS total_employees,
			MAX(timesheets.updated_at)                                                                                     AS last_updated
		`).
		Scan(&main).Error
	if err != nil {
		return nil, err
	}

	stats.PendingApproval = main.PendingApproval
	stats.ApprovedEntries = main.ApprovedEntries
	stats.RejectedEntries = main.RejectedEntries
	stats.TotalEntries = main.PendingApproval + main.ApprovedEntries + main.RejectedEntries
	stats.PaidEntries = main.PaidEntries
	stats.PaidAmount = main.PaidAmount
	stats.PaidEmployees = main.PaidEmployees
	stats.TotalEmployees = main.TotalEmployees
	if main.LastUpdated.IsZero() {
		stats.LastUpdated = clock.Now()
	} else {
		stats.LastUpdated = main.LastUpdated
	}

	pending, err := r.getPendingPaymentSummary(filters)
	if err != nil {
		return nil, err
	}
	stats.PendingPaymentAmount = pending.PendingPaymentAmount
	stats.PendingEmployees = pending.PendingPaymentEmployees

	return &stats, nil
}

func (r *TimesheetAnalyticsRepository) getPendingPaymentSummary(filters domain.TimesheetFilters) (pendingPaymentSummaryResult, error) {
	var pending pendingPaymentSummaryResult
	pendingPaymentFilters := domain.NewOperationalPaymentBacklogTimesheetFilters()
	err := r.queryBuilder.BuildSummaryQuery(filters).
		Where("timesheet_status IN ?", pendingPaymentFilters.TimesheetStatus).
		Where("payment_status IN ?", pendingPaymentFilters.PaymentStatus).
		Select(`
			COALESCE(SUM(amount), 0)                       AS pending_payment_amount,
			COUNT(DISTINCT timesheets.employee_id)         AS pending_payment_employees
		`).
		Scan(&pending).Error
	return pending, err
}

// GetSummaryByProject retrieves summary statistics for a specific project
func (r *TimesheetAnalyticsRepository) GetSummaryByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*domain.TimesheetSummary, error) {
	var summary domain.TimesheetSummary

	// Get totals
	var result struct {
		TotalHours    float64
		TotalAmount   float64
		EntryCount    int
		EmployeeCount int
	}

	// Get all summary statistics in a single query with multiple aggregations
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select(`
			COALESCE(SUM(hours_worked), 0) as total_hours,
			COALESCE(SUM(amount), 0) as total_amount,
			COUNT(*) as entry_count,
			COUNT(DISTINCT employee_id) as employee_count
		`).
		Where("project_id = ? AND date BETWEEN ? AND ? AND timesheet_status = ?",
			projectID, fromDate, toDate, domain.TimesheetStatusApproved).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	summary.TotalHours = result.TotalHours
	summary.TotalAmount = int64(result.TotalAmount)
	summary.EntryCount = result.EntryCount
	summary.EmployeeCount = result.EmployeeCount

	// Get hours and amount by pay type
	type PayTypeResult struct {
		PayType string  `json:"paytype"`
		Hours   float64 `json:"hours"`
		Amount  float64 `json:"amount"`
	}

	var payTypeResults []PayTypeResult
	err = r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("paytype, COALESCE(SUM(hours_worked), 0) as hours, COALESCE(SUM(amount), 0) as amount").
		Where("project_id = ? AND date BETWEEN ? AND ? AND timesheet_status = ?",
			projectID, fromDate, toDate, domain.TimesheetStatusApproved).
		Group("paytype").
		Scan(&payTypeResults).Error
	if err != nil {
		return nil, err
	}

	summary.HoursByPayType = make(map[string]float64)
	summary.AmountByPayType = make(map[string]int64)

	for _, result := range payTypeResults {
		payType := result.PayType
		summary.HoursByPayType[payType] = result.Hours
		summary.AmountByPayType[payType] = int64(result.Amount)
	}

	return &summary, nil
}

func (r *TimesheetAnalyticsRepository) GetPendingApprovalCount(ctx context.Context, startDate, endDate time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ?", domain.TimesheetStatusPendingApproval, startDate, endDate).
		Count(&count).Error

	return int(count), err
}

// GetApprovedSalaryTotal returns total approved salary amount within date range
func (r *TimesheetAnalyticsRepository) GetApprovedSalaryTotal(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ?", domain.TimesheetStatusApproved, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

// GetTotalPaidSalary returns the total paid salary of all time
func (r *TimesheetAnalyticsRepository) GetTotalPaidSalary(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Select("COALESCE(SUM(paid_amount), 0)").
		Scan(&total).Error

	return total, err
}

// GetPendingSalaryForMonth returns pending salary for the current month
func (r *TimesheetAnalyticsRepository) GetPendingSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("date BETWEEN ?", startDate, endDate).
		Where("timesheet_status IN ?", []domain.TimesheetStatus{
			domain.TimesheetStatusApproved,
			domain.TimesheetStatusPendingApproval,
		}).
		Where("payment_status != ?", domain.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

// GetPaidSalaryForMonth returns paid salary for timesheets with work date in the given month
func (r *TimesheetAnalyticsRepository) GetPaidSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("date BETWEEN ?", startDate, endDate).
		Select("COALESCE(SUM(paid_amount), 0)").
		Scan(&total).Error

	return total, err
}

// GetFirstPaidTimesheetDate returns the earliest paid_at date
func (r *TimesheetAnalyticsRepository) GetFirstPaidTimesheetDate(ctx context.Context) (*time.Time, error) {
	var res struct {
		PaidAt *time.Time
	}
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Order("paid_at ASC").
		Limit(1).
		Select("paid_at").
		Scan(&res).Error

	if err != nil {
		return nil, err
	}
	if res.PaidAt == nil {
		return nil, nil
	}
	return res.PaidAt, nil
}

// GetPaidSalaryByEmployee returns total paid salary for each employee in the given period
func (r *TimesheetAnalyticsRepository) GetPaidSalaryByEmployee(ctx context.Context, startDate, endDate time.Time) (map[uint]int64, error) {
	var results []struct {
		EmployeeID uint
		TotalPaid  int64
	}

	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("paid_at BETWEEN ?", startDate, endDate).
		Select("employee_id, SUM(paid_amount) as total_paid").
		Group("employee_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	salaryMap := make(map[uint]int64)
	for _, r := range results {
		salaryMap[r.EmployeeID] = r.TotalPaid
	}
	return salaryMap, nil
}

// TopPaidEmployeeRow holds aggregated paid data for a single employee.
type TopPaidEmployeeRow struct {
	EmployeeID   uint    `gorm:"column:employee_id"`
	EmployeeName string  `gorm:"column:employee_name"`
	TotalPaid    float64 `gorm:"column:total_paid"`
	// LastTimesheetDate is the most recent timesheet date for this employee (used for active check)
	LastTimesheetDate *string `gorm:"column:last_timesheet_date"`
}

// GetTopPaidEmployees returns the top N employees by total paid amount.
// If startDate/endDate are provided, filters by timesheet work date; otherwise all-time.
func (r *TimesheetAnalyticsRepository) GetTopPaidEmployees(ctx context.Context, startDate, endDate *time.Time, limit int) ([]TopPaidEmployeeRow, error) {
	if limit <= 0 {
		limit = 10
	}

	q := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select(`
			timesheets.employee_id,
			employees.fullname AS employee_name,
			COALESCE(SUM(timesheets.paid_amount), 0) AS total_paid,
			DATE_FORMAT(MAX(timesheets.date), '%Y-%m-%d') AS last_timesheet_date
		`).
		Joins("JOIN employees ON employees.id = timesheets.employee_id").
		Where("timesheets.payment_status = ?", domain.PaymentStatusPaid)

	if startDate != nil && endDate != nil {
		q = q.Where("timesheets.date >= ? AND timesheets.date < ?", *startDate, *endDate)
	}

	var rows []TopPaidEmployeeRow
	err := q.
		Group("timesheets.employee_id, employees.fullname").
		Order("total_paid DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}
