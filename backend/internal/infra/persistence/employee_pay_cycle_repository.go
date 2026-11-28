package persistence

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmployeePayCycleRepository persists the per-employee-per-Kỳ billing read model
// (FR-1). One row per (employee, Kỳ, work month); verification accumulates
// amounts onto the existing row instead of inserting duplicates.
type EmployeePayCycleRepository struct {
	*BaseRepository
}

// NewEmployeePayCycleRepository wires the repository onto the shared database,
// returning the domain interface so callers depend on the port.
func NewEmployeePayCycleRepository(db *Database) domain.EmployeePayCycleRepository {
	return &EmployeePayCycleRepository{BaseRepository: NewBaseRepository(db)}
}

// UpsertAmount inserts the record, or — when the employee already has a row
// for this (Kỳ, work month) — adds the incoming amount to the stored
// paid_amount. First-writer wins on project/timesheet identity (the billing
// atom is the employee, not the project; per-project detail lives on
// timesheets).
func (r *EmployeePayCycleRepository) UpsertAmount(ctx context.Context, record *domain.EmployeePayCycle) error {
	err := r.dbForContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "employee_id"},
				{Name: "cycle"},
				{Name: "work_month"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"paid_amount": gorm.Expr("paid_amount + VALUES(paid_amount)"),
			}),
		}).
		Create(record).Error
	if err != nil {
		return fmt.Errorf("employee_pay_cycles: upsert amount: %w", err)
	}
	return nil
}

// ListByWorkMonth returns every paid record of one work month, ordered by
// Kỳ then employee.
func (r *EmployeePayCycleRepository) ListByWorkMonth(ctx context.Context, workMonth string) ([]domain.EmployeePayCycle, error) {
	var rows []domain.EmployeePayCycle
	err := r.dbForContext(ctx).
		Where("work_month = ?", workMonth).
		Order("cycle ASC, employee_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("employee_pay_cycles: list by work_month: %w", err)
	}
	return rows, nil
}

// CountDistinctEmployeesByCycle returns distinct-employee counts per Kỳ for one
// work month — the FR-3 billing-report summary input. Rows are already the
// per-employee billing atom, so a plain COUNT per Kỳ is the distinct count.
func (r *EmployeePayCycleRepository) CountDistinctEmployeesByCycle(ctx context.Context, workMonth string) (map[int]int64, error) {
	var rows []struct {
		Cycle int   `gorm:"column:cycle"`
		Count int64 `gorm:"column:c"`
	}
	err := r.dbForContext(ctx).
		Model(&domain.EmployeePayCycle{}).
		Select("cycle, COUNT(*) AS c").
		Where("work_month = ?", workMonth).
		Group("cycle").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("employee_pay_cycles: count distinct by cycle: %w", err)
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.Cycle] = row.Count
	}
	return out, nil
}

// CountDistinctEmployeesByCycles returns one row per (work month, Kỳ) with its
// distinct-employee count for the inclusive [fromMonth, toMonth] window. The
// unique key (employee_id, cycle, work_month) makes COUNT(*) the distinct
// employee count of each bucket.
func (r *EmployeePayCycleRepository) CountDistinctEmployeesByCycles(ctx context.Context, fromMonth, toMonth string) ([]domain.CycleMonthCount, error) {
	var rows []domain.CycleMonthCount
	err := r.dbForContext(ctx).
		Model(&domain.EmployeePayCycle{}).
		Select("work_month, cycle, COUNT(*) AS count").
		Where("work_month >= ? AND work_month <= ?", fromMonth, toMonth).
		Group("work_month, cycle").
		Order("work_month ASC, cycle ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("employee_pay_cycles: count distinct by cycles: %w", err)
	}
	return rows, nil
}

// kyCycleExpr is the Kỳ of a work date under the fixed calendar: day 1–7 →
// Kỳ 1, 8–14 → Kỳ 2, 15–21 → Kỳ 3, 22–end → Kỳ 4 (clock.TimesheetKyCount).
const kyCycleExpr = "(CASE WHEN DAY(t.date) <= 7 THEN 1 WHEN DAY(t.date) <= 14 THEN 2 WHEN DAY(t.date) <= 21 THEN 3 ELSE 4 END)"

// BankTransferHistory returns the employee-Kỳ records with their per-posting
// bank trace for the rebuilt Lịch sử trả lương page (PRD §5.7), plus
// the summary strip over the whole filtered set. Postings are the employee's
// customer-paid timesheets inside the Kỳ window; the transfer code is the
// verifying upload's filename and the bank reference the timesheet's
// payment_reference (D22). All user input travels as query args.
func (r *EmployeePayCycleRepository) BankTransferHistory(ctx context.Context, f domain.BankTransferHistoryFilter) ([]domain.BankTransferHistoryRow, domain.BankTransferHistorySummary, error) {
	db := r.dbForContext(ctx)
	if f.PageSize <= 0 || f.PageSize > 200 {
		f.PageSize = 50
	}
	if f.Page < 1 {
		f.Page = 1
	}

	searchCond := ""
	var args []any
	if f.Search != "" {
		searchCond = ` AND (emp.fullname LIKE ? OR emp.cccd LIKE ? OR t.payment_reference LIKE ?
			OR EXISTS (SELECT 1 FROM transfer_uploads tu WHERE tu.id = epc.source_upload_id AND tu.original_filename LIKE ?))`
		s := "%" + f.Search + "%"
		args = append(args, s, s, s, s)
	}
	if f.Cycle >= 1 && f.Cycle <= 4 {
		searchCond += " AND epc.cycle = ?"
		args = append(args, f.Cycle)
	}

	// Summary over the whole filtered set: postings are counted per record.
	var summary domain.BankTransferHistorySummary
	sumArgs := append([]any{}, args...)
	err := db.Raw(`SELECT
			COALESCE(SUM(epc.paid_amount), 0) AS total_amount,
			COALESCE(SUM((SELECT COUNT(*) FROM timesheets t
				WHERE t.employee_id = epc.employee_id AND t.payment_status = 'paid'
				AND DATE_FORMAT(t.date, '%Y-%m') = epc.work_month
				AND `+kyCycleExpr+` = epc.cycle)), 0) AS transfer_count,
			COUNT(*) AS employee_count
		FROM employee_pay_cycles epc
		JOIN employees emp ON emp.id = epc.employee_id
		WHERE epc.work_month = ?`+searchCond,
		append([]any{f.WorkMonth}, sumArgs...)...).Scan(&summary).Error
	if err != nil {
		return nil, summary, fmt.Errorf("employee_pay_cycles: bank history summary: %w", err)
	}

	// One row per employee-Kỳ record, ordered Kỳ then employee.
	var keys []struct {
		EmployeeID uint   `gorm:"column:employee_id"`
		Cycle      int    `gorm:"column:cycle"`
		WorkMonth  string `gorm:"column:work_month"`
		PaidAmount int64  `gorm:"column:paid_amount"`
	}
	keyArgs := append([]any{f.WorkMonth}, args...)
	err = db.Raw(`SELECT epc.employee_id, epc.cycle, epc.work_month, epc.paid_amount
		FROM employee_pay_cycles epc
		JOIN employees emp ON emp.id = epc.employee_id
		WHERE epc.work_month = ?`+searchCond+`
		ORDER BY epc.cycle ASC, epc.employee_id ASC
		LIMIT ? OFFSET ?`,
		append(keyArgs, f.PageSize, (f.Page-1)*f.PageSize)...).Scan(&keys).Error
	if err != nil {
		return nil, summary, fmt.Errorf("employee_pay_cycles: bank history page: %w", err)
	}
	if len(keys) == 0 {
		return []domain.BankTransferHistoryRow{}, summary, nil
	}

	// Employee + project identity for the page.
	keyArgs2 := append([]any{f.WorkMonth}, args...)
	var identities []struct {
		EmployeeID   uint   `gorm:"column:employee_id"`
		EmployeeName string `gorm:"column:employee_name"`
		EmployeeCCCD string `gorm:"column:employee_cccd"`
		Cycle        int    `gorm:"column:cycle"`
		ProjectID    uint   `gorm:"column:project_id"`
		ProjectName  string `gorm:"column:project_name"`
	}
	err = db.Raw(`SELECT DISTINCT epc.employee_id, emp.fullname AS employee_name, emp.cccd AS employee_cccd,
			epc.cycle, p.id AS project_id, p.name AS project_name
		FROM employee_pay_cycles epc
		JOIN employees emp ON emp.id = epc.employee_id
		JOIN timesheets t ON t.employee_id = epc.employee_id
			AND t.payment_status = 'paid'
			AND DATE_FORMAT(t.date, '%Y-%m') = epc.work_month
			AND `+kyCycleExpr+` = epc.cycle
		JOIN projects p ON p.id = t.project_id
		WHERE epc.work_month = ?`+searchCond,
		keyArgs2...).Scan(&identities).Error
	if err != nil {
		return nil, summary, fmt.Errorf("employee_pay_cycles: bank history identities: %w", err)
	}

	// Postings for the page's records: one customer-paid timesheet each, with
	// the verifying upload's filename as the transfer code.
	type key struct {
		emp uint
		cyc int
	}
	want := map[key]bool{}
	for _, k := range keys {
		want[key{k.EmployeeID, k.Cycle}] = true
	}
	postArgs := []any{f.WorkMonth}
	postCond := ""
	if f.Cycle >= 1 && f.Cycle <= 4 {
		postCond = " AND epc.cycle = ?"
		postArgs = append(postArgs, f.Cycle)
	}
	var postings []struct {
		EmployeeID    uint64     `gorm:"column:employee_id"`
		Cycle         int        `gorm:"column:cycle"`
		TimesheetID   uint64     `gorm:"column:timesheet_id"`
		TransferCode  string     `gorm:"column:transfer_code"`
		BankReference *string    `gorm:"column:bank_reference"`
		Amount        int64      `gorm:"column:amount"`
		PaidAt        *time.Time `gorm:"column:paid_at"`
	}
	err = db.Raw(`SELECT epc.employee_id, epc.cycle, t.id AS timesheet_id,
			tu.original_filename AS transfer_code, t.payment_reference AS bank_reference,
			t.amount AS amount, t.paid_at AS paid_at
		FROM employee_pay_cycles epc
		JOIN timesheets t ON t.employee_id = epc.employee_id
			AND t.payment_status = 'paid'
			AND DATE_FORMAT(t.date, '%Y-%m') = epc.work_month
			AND `+kyCycleExpr+` = epc.cycle
		JOIN transfer_uploads tu ON tu.id = epc.source_upload_id
		WHERE epc.work_month = ?`+postCond,
		postArgs...).Scan(&postings).Error
	if err != nil {
		return nil, summary, fmt.Errorf("employee_pay_cycles: bank history postings: %w", err)
	}
	postByKey := map[key][]domain.BankTransferHistoryPosting{}
	for _, p := range postings {
		k := key{uint(p.EmployeeID), p.Cycle}
		if !want[k] {
			continue
		}
		ref := ""
		if p.BankReference != nil {
			ref = *p.BankReference
		}
		postByKey[k] = append(postByKey[k], domain.BankTransferHistoryPosting{
			TimesheetID:   p.TimesheetID,
			TransferCode:  p.TransferCode,
			BankReference: ref,
			Amount:        p.Amount,
			PaidAt:        p.PaidAt,
		})
	}

	identByIdKey := map[key]*domain.BankTransferHistoryRow{}
	rows := make([]domain.BankTransferHistoryRow, 0, len(keys))
	for _, k := range keys {
		row := domain.BankTransferHistoryRow{
			EmployeeID:  k.EmployeeID,
			WorkMonth:   k.WorkMonth,
			Cycle:       k.Cycle,
			TotalAmount: k.PaidAmount,
			Transfers:   []domain.BankTransferHistoryPosting{},
		}
		for _, id := range identities {
			if id.EmployeeID == k.EmployeeID && id.Cycle == k.Cycle {
				row.EmployeeName = id.EmployeeName
				row.EmployeeCCCD = id.EmployeeCCCD
				row.ProjectIDs = append(row.ProjectIDs, id.ProjectID)
				row.ProjectNames = append(row.ProjectNames, id.ProjectName)
			}
		}
		for _, p := range postByKey[key{k.EmployeeID, k.Cycle}] {
			if p.PaidAt != nil && row.PaymentDate == "" {
				row.PaymentDate = p.PaidAt.Format("2006-01-02")
			}
		}
		identByIdKey[key{k.EmployeeID, k.Cycle}] = &row
		rows = append(rows, row)
	}
	// Kỳ window labels from the fixed calendar: WorkStartDay opens the
	// window; Kỳ 4 closes at month end.
	for i := range rows {
		year, _ := strconv.Atoi(rows[i].WorkMonth[:4])
		month, _ := strconv.Atoi(rows[i].WorkMonth[5:7])
		monthTime := time.Month(month)
		startDay := clock.WorkStartDay(rows[i].Cycle)
		endDay := daysInMonthLocal(year, monthTime)
		if rows[i].Cycle < clock.TimesheetKyCount {
			endDay = clock.WorkStartDay(rows[i].Cycle+1) - 1
		}
		rows[i].FromDate = fmt.Sprintf("%02d/%02d/%d", startDay, month, year)
		rows[i].ToDate = fmt.Sprintf("%02d/%02d/%d", endDay, month, year)
	}
	return rows, summary, nil
}

// daysInMonthLocal mirrors clock.daysInMonth: day 0 of the next month.
func daysInMonthLocal(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
