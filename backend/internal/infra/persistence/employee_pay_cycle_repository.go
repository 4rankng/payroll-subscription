package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"

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
