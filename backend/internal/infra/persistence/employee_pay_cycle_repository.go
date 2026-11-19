package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// KyPaidRecordRepository persists the per-employee-per-Kỳ billing read model
// (FR-1). One row per (employee, Kỳ, work month); verification accumulates
// amounts onto the existing row instead of inserting duplicates.
type KyPaidRecordRepository struct {
	*BaseRepository
}

// NewKyPaidRecordRepository wires the repository onto the shared database,
// returning the domain interface so callers depend on the port.
func NewKyPaidRecordRepository(db *Database) domain.KyPaidRecordRepository {
	return &KyPaidRecordRepository{BaseRepository: NewBaseRepository(db)}
}

// UpsertAmount inserts the record, or — when the employee already has a row
// for this (Kỳ, work month) — adds the incoming amount to the stored
// paid_amount. First-writer wins on project/timesheet identity (the billing
// atom is the employee, not the project; per-project detail lives on
// timesheets).
func (r *KyPaidRecordRepository) UpsertAmount(ctx context.Context, record *domain.KyPaidRecord) error {
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
		return fmt.Errorf("ky_paid_records: upsert amount: %w", err)
	}
	return nil
}

// ListByWorkMonth returns every paid record of one work month, ordered by
// Kỳ then employee.
func (r *KyPaidRecordRepository) ListByWorkMonth(ctx context.Context, workMonth string) ([]domain.KyPaidRecord, error) {
	var rows []domain.KyPaidRecord
	err := r.dbForContext(ctx).
		Where("work_month = ?", workMonth).
		Order("ky ASC, employee_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ky_paid_records: list by work_month: %w", err)
	}
	return rows, nil
}

// CountDistinctEmployeesByKy returns distinct-employee counts per Kỳ for one
// work month — the FR-3 billing-report summary input. Rows are already the
// per-employee billing atom, so a plain COUNT per Kỳ is the distinct count.
func (r *KyPaidRecordRepository) CountDistinctEmployeesByKy(ctx context.Context, workMonth string) (map[int]int64, error) {
	var rows []struct {
		Ky    int   `gorm:"column:ky"`
		Count int64 `gorm:"column:c"`
	}
	err := r.dbForContext(ctx).
		Model(&domain.KyPaidRecord{}).
		Select("ky, COUNT(*) AS c").
		Where("work_month = ?", workMonth).
		Group("cycle").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ky_paid_records: count distinct by ky: %w", err)
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.Ky] = row.Count
	}
	return out, nil
}
