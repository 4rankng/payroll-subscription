package persistence

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// TransferUploadRepository persists transfer_uploads rows (FR-2).
type TransferUploadRepository struct {
	*BaseRepository
}

// NewTransferUploadRepository wires the repository onto the shared database,
// returning the domain interface so callers depend on the port.
func NewTransferUploadRepository(db *Database) domain.TransferUploadRepository {
	return &TransferUploadRepository{BaseRepository: NewBaseRepository(db)}
}

// GetByFileHash returns the prior upload for the given SHA-256, or
// (nil, nil) when this content has never been uploaded — the caller treats
// absence as "not previously processed" without a sentinel error.
func (r *TransferUploadRepository) GetByFileHash(ctx context.Context, fileHash string) (*domain.TransferUpload, error) {
	var row domain.TransferUpload
	err := r.dbForContext(ctx).Where("file_hash = ?", fileHash).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("transfer_uploads: get by file_hash: %w", err)
	}
	return &row, nil
}

// GetByID returns one upload with its outcome fields.
func (r *TransferUploadRepository) GetByID(ctx context.Context, id uint64) (*domain.TransferUpload, error) {
	var row domain.TransferUpload
	err := r.dbForContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("transfer_uploads: get by id: %w", err)
	}
	return &row, nil
}

// Create inserts a new upload row. The unique index on file_hash makes a
// concurrent duplicate fail rather than double-count.
// Delete removes a residue row (e.g. a pending upload whose synchronous
// processing never completed). Hard delete — D16.
func (r *TransferUploadRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.dbForContext(ctx).Delete(&domain.TransferUpload{}, id).Error; err != nil {
		return fmt.Errorf("transfer_uploads: delete: %w", err)
	}
	return nil
}

func (r *TransferUploadRepository) Create(ctx context.Context, upload *domain.TransferUpload) error {
	if err := r.dbForContext(ctx).Create(upload).Error; err != nil {
		return fmt.Errorf("transfer_uploads: create: %w", err)
	}
	return nil
}

// Update persists status/reject/verify and row-counter mutations on the
// primary key. The counters must stay in the column map: ProcessUpload
// finalizes parsed_rows/matched_rows through here, and a dropped column
// silently leaves the Sổ cái "Dòng khớp" list at 0.
func (r *TransferUploadRepository) Update(ctx context.Context, upload *domain.TransferUpload) error {
	if err := r.dbForContext(ctx).Model(&domain.TransferUpload{}).
		Where("id = ?", upload.ID).
		Updates(map[string]interface{}{
			"status":        upload.Status,
			"reject_reason": upload.RejectReason,
			"verified_at":   upload.VerifiedAt,
			"verified_by":   upload.VerifiedBy,
			"asset_id":      upload.AssetID,
			"parsed_rows":   upload.ParsedRows,
			"matched_rows":  upload.MatchedRows,
			"updated_at":    upload.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("transfer_uploads: update: %w", err)
	}
	return nil
}

// ListByWorkMonth returns the uploads of one work month, newest first.
func (r *TransferUploadRepository) ListByWorkMonth(ctx context.Context, workMonth string, limit int) ([]domain.TransferUpload, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []domain.TransferUpload
	err := r.dbForContext(ctx).
		Where("work_month = ?", workMonth).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("transfer_uploads: list by work_month: %w", err)
	}
	return rows, nil
}
