package persistence

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BillingReportRepository persists billing_reports and their Kỳ lines (FR-3).
type BillingReportRepository struct {
	*BaseRepository
}

// NewBillingReportRepository wires the repository onto the shared database,
// returning the domain interface so callers depend on the port.
func NewBillingReportRepository(db *Database) domain.BillingReportRepository {
	return &BillingReportRepository{BaseRepository: NewBaseRepository(db)}
}

// UpsertMonthly inserts the report or updates the existing row for the same
// work month in place (regeneration keeps the same id).
func (r *BillingReportRepository) UpsertMonthly(ctx context.Context, report *domain.BillingReport) error {
	err := r.dbForContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "work_month"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"fee_per_employee_per_ky_vnd": report.FeePerEmployeePerKyVnd,
				"total_distinct":              report.TotalDistinct,
				"total_vnd":                   report.TotalVnd,
				"excel_path":                  report.ExcelPath,
				"asset_id":                    report.AssetID,
				"generated_by":                report.GeneratedBy,
				"generated_at":                report.GeneratedAt,
			}),
		}).
		Create(report).Error
	if err != nil {
		return fmt.Errorf("billing_reports: upsert monthly: %w", err)
	}
	return nil
}

// ReplaceLines deletes the report's lines and inserts the new set.
func (r *BillingReportRepository) ReplaceLines(ctx context.Context, reportID uint64, lines []domain.BillingReportLine) error {
	return r.dbForContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("report_id = ?", reportID).Delete(&domain.BillingReportLine{}).Error; err != nil {
			return fmt.Errorf("billing_report_lines: clear: %w", err)
		}
		for i := range lines {
			lines[i].ReportID = reportID
		}
		if len(lines) == 0 {
			return nil
		}
		if err := tx.Create(&lines).Error; err != nil {
			return fmt.Errorf("billing_report_lines: insert: %w", err)
		}
		return nil
	})
}

// GetByWorkMonth returns the report of one work month, or (nil, nil).
func (r *BillingReportRepository) GetByWorkMonth(ctx context.Context, workMonth string) (*domain.BillingReport, error) {
	var row domain.BillingReport
	err := r.dbForContext(ctx).Where("work_month = ?", workMonth).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("billing_reports: get by work_month: %w", err)
	}
	return &row, nil
}

// LinesOf returns the Kỳ lines of one report, ordered by Kỳ.
func (r *BillingReportRepository) LinesOf(ctx context.Context, reportID uint64) ([]domain.BillingReportLine, error) {
	var rows []domain.BillingReportLine
	err := r.dbForContext(ctx).
		Where("report_id = ?", reportID).
		Order("cycle ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("billing_report_lines: list: %w", err)
	}
	return rows, nil
}
