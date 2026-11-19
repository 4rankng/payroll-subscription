package persistence

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

// projectFinancialAggregateSQL aggregates the denormalised project payout totals
// from the project's timesheets, which are the authoritative per-project money
// record. `total_payout_vnd` covers timesheets the customer has paid (verified
// from an uploaded transfer result, D11); `pending_payable_vnd` covers approved
// timesheets not yet covered by a verified upload.
//
// Placeholder order: paid status, approved status, pending status, project ID.
const projectFinancialAggregateSQL = `
	SELECT
		COALESCE(SUM(CASE WHEN payment_status = ? THEN COALESCE(paid_amount, 0) ELSE 0 END), 0) AS total_payout_vnd,
		COALESCE(SUM(CASE WHEN timesheet_status = ? AND payment_status = ? THEN COALESCE(amount, 0) ELSE 0 END), 0) AS pending_payable_vnd
	FROM timesheets
	WHERE project_id = ?
`

// GetProjectFinancialAggregate aggregates the denormalised project payout totals
// for one project. The meaning of each total is documented on domain.Project.
func (r *ProjectRepository) GetProjectFinancialAggregate(ctx context.Context, projectID uint) (*domain.ProjectFinancialAggregate, error) {
	var aggregate domain.ProjectFinancialAggregate

	err := r.DB.WithContext(ctx).
		Raw(projectFinancialAggregateSQL,
			domain.PaymentStatusPaid,
			domain.TimesheetStatusApproved,
			domain.PaymentStatusPending,
			projectID,
		).
		Scan(&aggregate).Error

	if err != nil {
		return nil, fmt.Errorf("aggregate timesheet financials for project %d: %w", projectID, err)
	}

	return &aggregate, nil
}

// UpdateProjectFinancialAggregate writes the denormalised payout totals in a
// single UPDATE statement and stamps updated_at with the caller's business time.
//
// It is a no-op when projectID does not exist: the statement simply matches no
// row. Unknown IDs are deliberately not an error, so repeating the recompute is
// always safe.
func (r *ProjectRepository) UpdateProjectFinancialAggregate(
	ctx context.Context,
	projectID uint,
	aggregate domain.ProjectFinancialAggregate,
	updatedAt time.Time,
) error {
	result := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Where("id = ?", projectID).
		Updates(map[string]any{
			"total_payout_vnd":    aggregate.TotalPayoutVND,
			"pending_payable_vnd": aggregate.PendingPayableVND,
			"updated_at":          updatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("update financial aggregate for project %d: %w", projectID, result.Error)
	}

	return nil
}
