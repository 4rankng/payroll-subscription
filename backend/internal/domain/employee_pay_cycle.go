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
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Cycle          int       `gorm:"column:cycle;type:tinyint unsigned;not null"`
	WorkMonth      string    `gorm:"column:work_month;type:char(7);not null"`
	EmployeeID     uint      `gorm:"column:employee_id;type:bigint unsigned;not null"`
	CCCD           string    `gorm:"column:cccd;type:varchar(255);not null"`
	ProjectID      uint      `gorm:"column:project_id;type:bigint unsigned;not null"`
	TimesheetID    uint      `gorm:"column:timesheet_id;type:bigint unsigned;not null"`
	SourceUploadID uint64    `gorm:"column:source_upload_id;type:bigint unsigned;not null"`
	PaidAmount     int64     `gorm:"column:paid_amount;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

// TableName pins the consolidated-schema table name.
func (EmployeePayCycle) TableName() string { return "employee_pay_cycles" }

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
}
