package billing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"api-server/internal/app/services/config"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// assetStorer is the file-storage slice the report saver needs (AssetService
// implements it; fakes in tests).
type assetStorer interface {
	UploadAssetFromBytes(ctx context.Context, data []byte, filename string, uploadType string, uploadedBy uint) (*domain.Asset, error)
}

// BillingReportService generates the month-end billing report (FR-3): the
// Summary sheet (distinct employees per Kỳ × fee) plus one sheet per project
// listing that project's payment history (D24). The fee is snapshotted into
// the report (FR-4) — later price changes never rewrite history.
type BillingReportService struct {
	reports    domain.BillingReportRepository
	kyPaid     domain.EmployeePayCycleRepository
	timesheets domain.TimesheetRepository
	employees  domain.EmployeeRepository
	projects   domain.ProjectRepository
	feeSource  *config.SettingsConfigService
	storage    assetStorer
	notifier   BillingNotifier
	logger     *slog.Logger
}

func NewBillingReportService(
	reports domain.BillingReportRepository,
	kyPaid domain.EmployeePayCycleRepository,
	timesheets domain.TimesheetRepository,
	employees domain.EmployeeRepository,
	projects domain.ProjectRepository,
	feeSource *config.SettingsConfigService,
	storage assetStorer,
	notifier BillingNotifier,
	logger *slog.Logger,
) *BillingReportService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BillingReportService{
		reports:    reports,
		kyPaid:     kyPaid,
		timesheets: timesheets,
		employees:  employees,
		projects:   projects,
		feeSource:  feeSource,
		storage:    storage,
		notifier:   notifier,
		logger:     logger,
	}
}

// projectSheet is one project's payment-history rows for the per-project
// sheets of the D24 workbook.
type projectSheet struct {
	ProjectID   uint
	ProjectName string
	Rows        []PaymentRow
}

// PaymentRow is one customer-paid timesheet rendered on a project sheet.
type PaymentRow struct {
	EmployeeName     string
	CCCD             string
	Cycle            int
	Date             string
	Amount           int64
	PaymentDate      string
	PaymentReference string
}

// GenerateMonthly builds (or regenerates) the billing report of one work
// month. Idempotent: regenerating keeps the same report row and replaces its
// Kỳ lines and workbook.
func (s *BillingReportService) GenerateMonthly(ctx context.Context, workMonth string, generatedBy uint) (*domain.BillingReport, []domain.BillingReportLine, error) {
	fee := s.feeSource.GetSubscriptionFeePerEmployeePerKy(ctx)

	byCycle, err := s.kyPaid.CountDistinctEmployeesByCycle(ctx, workMonth)
	if err != nil {
		return nil, nil, fmt.Errorf("billing report: distinct counts: %w", err)
	}

	report := &domain.BillingReport{
		WorkMonth:              workMonth,
		FeePerEmployeePerKyVnd: fee,
		GeneratedBy:            generatedBy,
		GeneratedAt:            clock.Now(),
	}
	var lines []domain.BillingReportLine
	for ky := 1; ky <= clock.TimesheetKyCount; ky++ {
		distinct := byCycle[ky]
		if distinct == 0 {
			continue
		}
		lines = append(lines, domain.BillingReportLine{
			Cycle:         ky,
			DistinctCount: int(distinct),
			AmountVnd:     int64(distinct) * fee,
		})
		report.TotalDistinct += int(distinct)
		report.TotalVnd += int64(distinct) * fee
	}

	sheets, err := s.projectSheets(ctx, workMonth)
	if err != nil {
		return nil, nil, err
	}

	workbook, err := renderBillingWorkbook(workMonth, fee, lines, report.TotalDistinct, report.TotalVnd, sheets)
	if err != nil {
		return nil, nil, fmt.Errorf("billing report: render workbook: %w", err)
	}
	if s.storage != nil {
		stored, err := s.storage.UploadAssetFromBytes(ctx, workbook,
			fmt.Sprintf("bao-cao-phi-%s.xlsx", workMonth), "billing_report", generatedBy)
		if err != nil {
			return nil, nil, fmt.Errorf("billing report: store workbook: %w", err)
		}
		report.ExcelPath = stored.FilePath
		assetID := uint64(stored.ID)
		report.AssetID = &assetID
	}

	if err := s.reports.UpsertMonthly(ctx, report); err != nil {
		return nil, nil, fmt.Errorf("billing report: persist: %w", err)
	}
	if report.ID == 0 {
		saved, err := s.reports.GetByWorkMonth(ctx, workMonth)
		if err != nil {
			return nil, nil, err
		}
		report = saved
	}
	if err := s.reports.ReplaceLines(ctx, report.ID, lines); err != nil {
		return nil, nil, fmt.Errorf("billing report: persist lines: %w", err)
	}

	s.logger.Info("billing report generated",
		"work_month", workMonth, "fee", fee,
		"total_distinct", report.TotalDistinct, "total_vnd", report.TotalVnd,
		"projects", len(sheets),
	)
	return report, lines, nil
}

// NotifyReportReady pushes admins that a report is ready (portal notice).
func (s *BillingReportService) NotifyReportReady(ctx context.Context, workMonth string) {
	if s.notifier == nil {
		return
	}
	title := "Báo cáo phí tháng " + workMonth + " đã sẵn sàng"
	message := "Báo cáo phí subscription tháng " + workMonth + " đã được tạo. Vui lòng xuất file Excel và gửi yêu cầu thanh toán."
	if err := s.notifier.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeBillingReportGenerated, title, message); err != nil {
		s.logger.Error("billing report ready push failed", "work_month", workMonth, "error", err)
	}
}

// projectSheets groups the month's customer-paid timesheets per project,
// sorted by project id for deterministic sheet order.
func (s *BillingReportService) projectSheets(ctx context.Context, workMonth string) ([]projectSheet, error) {
	monthTime, err := parseWorkMonth(workMonth)
	if err != nil {
		return nil, fmt.Errorf("billing report: %w", err)
	}
	first := monthTime
	last := monthTime.AddDate(0, 1, -1)

	var paid []*domain.Timesheet
	if err := s.timesheets.GetPaidTimesheetsInDateRange(ctx, first, last, &paid); err != nil {
		return nil, fmt.Errorf("billing report: load paid timesheets: %w", err)
	}
	if len(paid) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(paid))
	seen := map[uint]struct{}{}
	for i := range paid {
		if _, ok := seen[paid[i].EmployeeID]; ok {
			continue
		}
		seen[paid[i].EmployeeID] = struct{}{}
		ids = append(ids, int64(paid[i].EmployeeID))
	}
	employees, err := s.employees.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("billing report: load employees: %w", err)
	}
	names := make(map[uint]string, len(employees))
	cccds := make(map[uint]string, len(employees))
	for _, e := range employees {
		names[e.ID] = e.Fullname
		cccds[e.ID] = e.CCCD
	}

	projectIDs := make(map[uint]struct{}, 8)
	for i := range paid {
		projectIDs[paid[i].ProjectID] = struct{}{}
	}
	// Sheet titles use the real project name (code as fallback) — the owner
	// banned "Dự án #<id>" on any user-facing surface.
	titles := make(map[uint]string, len(projectIDs))
	if len(projectIDs) > 0 {
		ids := make([]uint, 0, len(projectIDs))
		for id := range projectIDs {
			ids = append(ids, id)
		}
		projects, err := s.projects.GetByIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("billing report: load projects: %w", err)
		}
		for id, p := range projects {
			switch {
			case p != nil && p.Name != "":
				titles[id] = p.Name
			case p != nil && p.Code != "":
				titles[id] = p.Code
			}
		}
	}
	for id := range projectIDs {
		if titles[id] == "" {
			titles[id] = "(dự án đã xoá)"
		}
	}

	sheets := map[uint]*projectSheet{}
	for i := range paid {
		ts := paid[i] // *domain.Timesheet
		sheet, ok := sheets[ts.ProjectID]
		if !ok {
			sheet = &projectSheet{ProjectID: ts.ProjectID, ProjectName: titles[ts.ProjectID]}
			sheets[ts.ProjectID] = sheet
		}
		paymentDate := ""
		if ts.PaymentDate != nil {
			paymentDate = ts.PaymentDate.Format("02/01/2006")
		}
		ref := ""
		if ts.PaymentReference != nil {
			ref = *ts.PaymentReference
		}
		sheet.Rows = append(sheet.Rows, PaymentRow{
			EmployeeName:     names[ts.EmployeeID],
			CCCD:             cccds[ts.EmployeeID],
			Cycle:            clock.KyFromWorkDay(ts.Date.Day()),
			Date:             ts.Date.Format("02/01/2006"),
			Amount:           ts.PaidAmount,
			PaymentDate:      paymentDate,
			PaymentReference: ref,
		})
	}

	out := make([]projectSheet, 0, len(sheets))
	for _, sheet := range sheets {
		out = append(out, *sheet)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProjectID < out[j].ProjectID })
	return out, nil
}

func parseWorkMonth(workMonth string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01", workMonth, clock.DefaultLocation)
	if err != nil {
		return time.Time{}, fmt.Errorf("work_month phải có dạng YYYY-MM: %q", workMonth)
	}
	return t, nil
}

// GetReport returns the report of one work month with its Kỳ lines, or
// (nil, nil, nil) when the month has no report yet.
func (s *BillingReportService) GetReport(ctx context.Context, workMonth string) (*domain.BillingReport, []domain.BillingReportLine, error) {
	report, err := s.reports.GetByWorkMonth(ctx, workMonth)
	if err != nil || report == nil {
		return nil, nil, err
	}
	lines, err := s.reports.LinesOf(ctx, report.ID)
	if err != nil {
		return nil, nil, err
	}
	return report, lines, nil
}

// PaidSeries returns the FR-6 per-day paid time series (distinct employees
// and amount) per project. projectID nil = all projects. Rows are enriched
// with the project code/name so the UI never renders "Dự án #<id>".
func (s *BillingReportService) PaidSeries(ctx context.Context, from, to time.Time, projectID *uint) ([]domain.PaidSeriesRow, error) {
	rows, err := s.timesheets.GetPaidSeries(ctx, from, to, projectID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, 8)
	seen := make(map[uint]struct{}, 8)
	for i := range rows {
		if _, ok := seen[rows[i].ProjectID]; ok {
			continue
		}
		seen[rows[i].ProjectID] = struct{}{}
		ids = append(ids, rows[i].ProjectID)
	}
	if len(ids) == 0 {
		return rows, nil
	}
	projects, err := s.projects.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("billing report: load paid-series projects: %w", err)
	}
	for i := range rows {
		if p := projects[rows[i].ProjectID]; p != nil {
			rows[i].ProjectName = p.Name
			rows[i].ProjectCode = p.Code
		}
	}
	return rows, nil
}

// KyMonthSummary is one work month of the FR-6 Tổng quan per-month table:
// distinct-employee counts per Kỳ of that month.
type KyMonthSummary struct {
	WorkMonth string        `json:"work_month"`
	Counts    map[int]int64 `json:"counts"`
}

// DashboardKySummary is the FR-6 per-month view payload: the last `Months`
// work months ending at the current one, each with per-Kỳ distinct counts,
// plus the current subscription fee so the table can render VND totals
// without a second call.
type DashboardKySummary struct {
	Months                 []KyMonthSummary `json:"months"`
	FeePerEmployeePerKyVnd int64            `json:"fee_per_employee_per_ky_vnd"`
}

// DashboardKySummary builds the FR-6 monthly table input. months is clamped
// to [1, 24] and defaults to 12 at the handler.
func (s *BillingReportService) DashboardKySummary(ctx context.Context, months int) (*DashboardKySummary, error) {
	if months < 1 {
		months = 1
	}
	if months > 24 {
		months = 24
	}

	now := clock.Now()
	// Walk back to the first work month of the window, inclusive of the
	// current month. char(7) YYYY-MM sorts lexicographically == chronologically.
	firstYear, firstMonth := now.Year(), int(now.Month())
	firstMonth -= months - 1
	for firstMonth < 1 {
		firstMonth += 12
		firstYear--
	}
	fromMonth := fmt.Sprintf("%04d-%02d", firstYear, firstMonth)
	toMonth := fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))

	rows, err := s.kyPaid.CountDistinctEmployeesByCycles(ctx, fromMonth, toMonth)
	if err != nil {
		return nil, fmt.Errorf("billing: dashboard ky summary: %w", err)
	}

	byMonth := make(map[string]map[int]int64, months)
	for _, row := range rows {
		counts, ok := byMonth[row.WorkMonth]
		if !ok {
			counts = make(map[int]int64, clock.TimesheetKyCount)
			byMonth[row.WorkMonth] = counts
		}
		counts[row.Cycle] = row.Count
	}

	monthItems := make([]KyMonthSummary, 0, months)
	year, month := firstYear, firstMonth
	for range months {
		workMonth := fmt.Sprintf("%04d-%02d", year, month)
		counts := byMonth[workMonth]
		if counts == nil {
			counts = map[int]int64{}
		}
		monthItems = append(monthItems, KyMonthSummary{WorkMonth: workMonth, Counts: counts})
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return &DashboardKySummary{
		Months:                 monthItems,
		FeePerEmployeePerKyVnd: s.feeSource.GetSubscriptionFeePerEmployeePerKy(ctx),
	}, nil
}
