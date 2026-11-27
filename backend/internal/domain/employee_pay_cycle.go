package domain

import (
	"context"
	"time"
)

// EmployeePayCycle is the per-employee-per-Kỳ billing atom (FR-1, founder
// decision: the billing unit is one distinct employee in one Kỳ). A row is
// created/accumulated by upload verification: `paid_amount` is the SUM of the
// employee's customer-paid timesheet amounts across all projects in that Kỳ —
// projects are internal detail, never a second billing row. One row per
// (employee, Kỳ, work month), hard delete only (D16).
type EmployeePayCycle struct {
	ID             uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Cycle          int       `json:"cycle" gorm:"column:cycle;type:tinyint unsigned;not null"`
	WorkMonth      string    `json:"work_month" gorm:"column:work_month;type:char(7);not null"`
	EmployeeID     uint      `json:"employee_id" gorm:"column:employee_id;type:bigint unsigned;not null"`
	ProjectID      uint      `json:"project_id" gorm:"column:project_id;type:bigint unsigned;not null"`
	TimesheetID    uint      `json:"timesheet_id" gorm:"column:timesheet_id;type:bigint unsigned;not null"`
	SourceUploadID uint64    `json:"source_upload_id" gorm:"column:source_upload_id;type:bigint unsigned;not null"`
	PaidAmount     int64     `json:"paid_amount" gorm:"column:paid_amount;not null;default:0"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
}

// TableName pins the consolidated-schema table name.
func (EmployeePayCycle) TableName() string { return "employee_pay_cycles" }

// BankTransferHistoryPosting is one bank posting inside an employee-Kỳ
// record: a single customer-paid timesheet with its transfer trace (PRD
// PAYR-42 §5.7 — one ledger row per posting, nothing collapsed).
type BankTransferHistoryPosting struct {
	TimesheetID   uint64     `json:"timesheet_id"`
	TransferCode  string     `json:"transfer_code"`
	BankReference string     `json:"bank_reference"`
	Amount        int64      `json:"amount"`
	PaidAt        *time.Time `json:"paid_at"`
}

// BankTransferHistoryRow is one employee-Kỳ record of the rebuilt payment
// history page: the employee/project identity plus every bank posting.
type BankTransferHistoryRow struct {
	EmployeeID   uint                         `json:"employee_id"`
	EmployeeName string                       `json:"employee_name"`
	EmployeeCCCD string                       `json:"employee_cccd"`
	ProjectIDs   []uint                       `json:"project_ids"`
	ProjectNames []string                     `json:"project_names"`
	WorkMonth    string                       `json:"work_month"`
	Cycle        int                          `json:"cycle"`
	FromDate     string                       `json:"from_date"`
	ToDate       string                       `json:"to_date"`
	PaymentDate  string                       `json:"payment_date"`
	TotalAmount  int64                        `json:"total_amount"`
	Transfers    []BankTransferHistoryPosting `json:"transfers"`
}

// BankTransferHistorySummary aggregates the filter result set (PRD §5.7
// summary strip): total transferred VND, bank posting count, employee count.
type BankTransferHistorySummary struct {
	TotalAmount   int64 `json:"total_amount"`
	TransferCount int64 `json:"transfer_count"`
	EmployeeCount int64 `json:"employee_count"`
}

// BankTransferHistoryFilter scopes the payment-history read model.
type BankTransferHistoryFilter struct {
	WorkMonth string
	Cycle     int
	Search    string
	Page      int
	PageSize  int
}

// EmployeePayCycleRepository persists the per-employee-per-Kỳ read model.
type EmployeePayCycleRepository interface {
	// UpsertAmount inserts the row, or — when the employee already has a row
	// for this (Kỳ, work month) — adds the amount to the existing row's
	// paid_amount. The unique key (employee_id, cycle, work_month) is the
	// conflict target.
	UpsertAmount(ctx context.Context, record *EmployeePayCycle) error
	// ListByWorkMonth returns every paid record of one work month.
	ListByWorkMonth(ctx context.Context, workMonth string) ([]EmployeePayCycle, error)
	// CountDistinctEmployeesByCycle returns distinct-employee counts per Kỳ for
	// one work month — the billing-report summary input (FR-3).
	CountDistinctEmployeesByCycle(ctx context.Context, workMonth string) (map[int]int64, error)
	// CountDistinctEmployeesByCycles returns one row per (work month, Kỳ) with
	// its distinct-employee count, for every work month in the inclusive
	// [fromMonth, toMonth] window (YYYY-MM) — the FR-6 Tổng quan per-month
	// table input. Months with no paid records simply have no rows.
	CountDistinctEmployeesByCycles(ctx context.Context, fromMonth, toMonth string) ([]CycleMonthCount, error)
	// BankTransferHistory serves the rebuilt Lịch sử trả lương page (PRD
	// PAYR-42 §5.7): employee-Kỳ records with their per-posting bank trace,
	// filtered by work month, Kỳ, and a free search across employee name,
	// CCCD, transfer code and bank reference; paginated, with the summary
	// strip aggregates for the WHOLE filtered set.
	BankTransferHistory(ctx context.Context, f BankTransferHistoryFilter) ([]BankTransferHistoryRow, BankTransferHistorySummary, error)
}

// CycleMonthCount is the distinct-employee count of one (work month, Kỳ)
// bucket of the FR-6 monthly table.
type CycleMonthCount struct {
	WorkMonth string
	Cycle     int
	Count     int64
}
