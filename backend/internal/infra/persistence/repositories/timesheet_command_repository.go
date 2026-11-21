package repositories

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
)

// TimesheetCommandRepository handles write operations for timesheets
type TimesheetCommandRepository struct {
	db             *gorm.DB
	batchProcessor *common.BatchProcessor
	errorHandler   *common.RepoErrorHandler
}

// NewTimesheetCommandRepository creates a new command repository
func NewTimesheetCommandRepository(db *gorm.DB) *TimesheetCommandRepository {
	return &TimesheetCommandRepository{
		db:             db,
		batchProcessor: common.NewBatchProcessor(common.DefaultBatchConfig()),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

// Create creates a new timesheet
func (r *TimesheetCommandRepository) Create(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.getDB(ctx).Create(timesheet).Error
}

// Update updates an existing timesheet
func (r *TimesheetCommandRepository) Update(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.getDB(ctx).Save(timesheet).Error
}

// Delete soft deletes a timesheet
func (r *TimesheetCommandRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Timesheet{}, id).Error
}

// HardDelete permanently removes a timesheet (no soft-delete). BCC replacement
// may only remove a row that is still pending approval and unpaid; the exact
// predicate prevents a concurrent review/payment transition from being lost.
func (r *TimesheetCommandRepository) HardDelete(ctx context.Context, id uint) error {
	result := r.getDB(ctx).
		Unscoped().
		Where("id = ? AND timesheet_status = ? AND payment_status = ?",
			id, domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending).
		Delete(&domain.Timesheet{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("timesheet %d became protected during replacement", id)
	}
	return nil
}

// BulkCreate creates multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkCreate(ctx context.Context, timesheets []*domain.Timesheet) error {
	if len(timesheets) == 0 {
		return nil
	}

	return r.getDB(ctx).CreateInBatches(timesheets, 100).Error
}

// BulkUpdate updates multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkUpdate(ctx context.Context, timesheets []*domain.Timesheet) error {
	if len(timesheets) == 0 {
		return nil
	}

	// Updates are more expensive than reads, so use smaller batches
	updateConfig := common.BatchConfig{
		DefaultSize:    100,
		MaxSize:        100,
		QuerySizeLimit: 32768,
	}
	updateProcessor := common.NewBatchProcessor(updateConfig)

	return updateProcessor.ProcessInBatches(ctx, timesheets, func(batch interface{}) error {
		batchTimesheets := batch.([]*domain.Timesheet)

		// Use transaction for batch consistency
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, timesheet := range batchTimesheets {
				if err := tx.Save(timesheet).Error; err != nil {
					return err
				}
			}
			return nil
		})
	})
}

// Approve approves a timesheet
func (r *TimesheetCommandRepository) Approve(ctx context.Context, id uint, approvedBy uint) error {
	now := clock.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusApproved,
			"approved_by":      approvedBy,
			"approved_at":      now,
			"rejection_reason": "",
		}).Error
}

// BulkApprove approves multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error {
	if len(ids) == 0 {
		return nil
	}

	now := clock.Now()

	// Use BatchProcessor for batch approval
	return r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return r.getDB(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Updates(map[string]interface{}{
				"timesheet_status": domain.TimesheetStatusApproved,
				"approved_by":      approvedBy,
				"approved_at":      now,
				"rejection_reason": "",
					}).Error
	})
}

// BulkReject rejects multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkReject(ctx context.Context, ids []uint, rejectionReason string) error {
	if len(ids) == 0 {
		return nil
	}

	return r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return r.db.WithContext(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Updates(map[string]interface{}{
				"timesheet_status": domain.TimesheetStatusRejected,
				"rejection_reason": rejectionReason,
				"approved_by":      nil,
				"approved_at":      nil,
			}).Error
	})
}

// Reject rejects a timesheet
func (r *TimesheetCommandRepository) Reject(ctx context.Context, id uint, rejectionReason string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusRejected,
			"rejection_reason": rejectionReason,
			"approved_by":      nil,
			"approved_at":      nil,
		}).Error
}

// Reset resets a timesheet to pending approval status
func (r *TimesheetCommandRepository) Reset(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusPendingApproval,
			"approved_by":      nil,
			"approved_at":      nil,
			"rejection_reason": "",
			"force_payroll":    false,
		}).Error
}

// ResetAllApproved mass-resets every approved, unpaid timesheet back to
// pending approval in a single set-based UPDATE (same field updates as Reset).
// The status and payment predicates are part of the UPDATE, so a row that
// becomes paid before this statement acquires its write lock cannot be reset.
// Returns the number of timesheets reset.
func (r *TimesheetCommandRepository) ResetAllApproved(ctx context.Context) (int64, error) {
	result := r.getDB(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ?", domain.TimesheetStatusApproved).
		Where("payment_status IS NULL OR payment_status <> ?", domain.PaymentStatusPaid).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusPendingApproval,
			"approved_by":      nil,
			"approved_at":      nil,
			"rejection_reason": "",
		})
	return result.RowsAffected, result.Error
}

// BulkUpdatePaymentStatus updates payment status for multiple timesheets in a single transaction
func (r *TimesheetCommandRepository) BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	// A transfer result can carry several rows for the same timesheet (two
	// payments inside one Kỳ). The per-row update below matches each id once —
	// the second duplicate update would see payment_status already paid and
	// trip the conflict check — so collapse duplicates to the last update per
	// id before executing.
	seen := make(map[uint]struct{}, len(updates))
	deduped := make([]domain.PaymentStatusUpdate, 0, len(updates))
	for i := len(updates) - 1; i >= 0; i-- {
		id := updates[i].TimesheetID
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, updates[i])
	}
	updates = deduped

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			updateFields := map[string]interface{}{
				"payment_status": update.PaymentStatus,
			}

			if update.PaymentReference != nil {
				updateFields["payment_reference"] = *update.PaymentReference
			}

			if update.PaymentDate != nil {
				updateFields["payment_date"] = *update.PaymentDate
			}

			if update.PaidAmount != nil {
				updateFields["paid_amount"] = *update.PaidAmount
			}

			if update.PaidAt != nil {
				updateFields["paid_at"] = *update.PaidAt
			} else if update.PaymentStatus == domain.PaymentStatusPaid {
				// Automatically set paid_at to current time when status is set to paid
				updateFields["paid_at"] = clock.Now()
			} else {
				// Clear paid_at when status is not paid
				updateFields["paid_at"] = nil
			}

			query := tx.Model(&domain.Timesheet{}).
				Where("id = ?", update.TimesheetID).
				Where("payment_status <> ?", domain.PaymentStatusPaid)
			if paymentUpdateRequiresApprovedStatus(update.PaymentStatus) {
				query = query.Where("timesheet_status = ?", domain.TimesheetStatusApproved)
			}
			result := query.Updates(updateFields)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return domain.NewConflictError(fmt.Sprintf("bảng chấm công %d đã thay đổi trước khi cập nhật thanh toán", update.TimesheetID))
			}
		}
		return nil
	})
}

// paymentUpdateRequiresApprovedStatus distinguishes workflow transitions from
// authoritative settlement facts. Paid/cancelled outcomes must still be
// recorded after a concurrent rejection; only reopening payment to pending
// remains approval-gated.
func paymentUpdateRequiresApprovedStatus(status domain.PaymentStatus) bool {
	switch status {
	case domain.PaymentStatusPaid, domain.PaymentStatusCancelled:
		return false
	default:
		return true
	}
}

// getDB gets the appropriate DB instance from context or falls back to default
func (r *TimesheetCommandRepository) getDB(ctx context.Context) *gorm.DB {
	// Check if there's a transaction context
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}
