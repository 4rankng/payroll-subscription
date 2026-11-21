package domain

import (
	"context"
	"time"
)

// BillingReport is the month-end subscription billing artifact (FR-3): the
// fee is snapshotted at generation time (FR-4) so later price changes never
// rewrite history. One report per work month; regenerated reports replace
// their lines but keep the same row.
type BillingReport struct {
	ID                     uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	WorkMonth              string    `json:"work_month" gorm:"column:work_month;type:char(7);not null;uniqueIndex:uq_billing_reports_work_month"`
	FeePerEmployeePerKyVnd int64     `json:"fee_per_employee_per_ky_vnd" gorm:"column:fee_per_employee_per_ky_vnd;not null;comment:'Fee snapshot at generation time (FR-4)'"`
	TotalDistinct          int       `json:"total_distinct" gorm:"column:total_distinct;not null;default:0"`
	TotalVnd               int64     `json:"total_vnd" gorm:"column:total_vnd;not null;default:0"`
	ExcelPath              string    `json:"excel_path" gorm:"column:excel_path;type:varchar(500)"`
	GeneratedBy            uint      `json:"generated_by" gorm:"column:generated_by;type:bigint unsigned;not null"`
	GeneratedAt            time.Time `json:"generated_at" gorm:"column:generated_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)"`
}

// TableName pins the consolidated-schema table name.
func (BillingReport) TableName() string { return "billing_reports" }

// BillingReportLine is one Kỳ row of a report: the distinct-employee count
// and the amount for that Kỳ (distinct × fee).
type BillingReportLine struct {
	ID            uint64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ReportID      uint64 `json:"report_id" gorm:"column:report_id;type:bigint unsigned;not null"`
	Cycle         int    `json:"cycle" gorm:"column:cycle;type:tinyint unsigned;not null"`
	DistinctCount int    `json:"distinct_count" gorm:"column:distinct_count;not null;default:0"`
	AmountVnd     int64  `json:"amount_vnd" gorm:"column:amount_vnd;not null;default:0"`
}

// TableName pins the consolidated-schema table name.
func (BillingReportLine) TableName() string { return "billing_report_lines" }

// BillingReportRepository persists billing_reports and their Kỳ lines.
type BillingReportRepository interface {
	// UpsertMonthly inserts the report or, when the work month already has
	// one, updates it in place (regeneration keeps the same row).
	UpsertMonthly(ctx context.Context, report *BillingReport) error
	// ReplaceLines deletes the report's lines and inserts the new set, in one
	// transaction with the report upsert's caller.
	ReplaceLines(ctx context.Context, reportID uint64, lines []BillingReportLine) error
	// GetByWorkMonth returns the report of one work month, or (nil, nil).
	GetByWorkMonth(ctx context.Context, workMonth string) (*BillingReport, error)
	// LinesOf returns the Kỳ lines of one report, ordered by Kỳ.
	LinesOf(ctx context.Context, reportID uint64) ([]BillingReportLine, error)
}
