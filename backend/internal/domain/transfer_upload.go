package domain

import (
	"context"
	"time"
)

// TransferUploadStatus is the lifecycle of one uploaded bank-transfer result
// file for a Kỳ. Per-row processing means an upload is `verified` when at
// least one row matched; `rejected` only when every row failed (D10 gate for
// the all-admin push). `pending` is the pre-parse state.
type TransferUploadStatus string

const (
	TransferUploadStatusPending  TransferUploadStatus = "pending"
	TransferUploadStatusVerified TransferUploadStatus = "verified"
	TransferUploadStatusRejected TransferUploadStatus = "rejected"
)

// TransferUpload is one uploaded bank-transfer result file for one Kỳ of one
// work month (FR-2). The SHA-256 of the content is the idempotency key: a
// re-upload of identical content short-circuits to the prior result instead
// of re-touching the books (settlement_uploads pattern). Hard delete only —
// no soft-delete columns anywhere (D16).
type TransferUpload struct {
	ID               uint64               `gorm:"column:id;primaryKey;autoIncrement"`
	Ky               int                  `gorm:"column:ky;type:tinyint unsigned;not null"`
	WorkMonth        string               `gorm:"column:work_month;type:char(7);not null"`
	FileHash         string               `gorm:"column:file_hash;type:char(64);not null;uniqueIndex:uq_transfer_uploads_file_hash"`
	OriginalFilename string               `gorm:"column:original_filename;type:varchar(255);not null"`
	AssetID          *uint64              `gorm:"column:asset_id;type:bigint unsigned"`
	UploadedBy       uint                 `gorm:"column:uploaded_by;type:bigint unsigned;not null"`
	Status           TransferUploadStatus `gorm:"column:status;type:enum('pending','verified','rejected');not null;default:'pending'"`
	RejectReason     *string              `gorm:"column:reject_reason;type:varchar(500)"`
	VerifiedAt       *time.Time           `gorm:"column:verified_at;type:datetime(3)"`
	VerifiedBy       *uint                `gorm:"column:verified_by;type:bigint unsigned"`
	CreatedAt        time.Time            `gorm:"column:created_at"`
	UpdatedAt        time.Time            `gorm:"column:updated_at"`
}

// TableName pins the consolidated-schema table name.
func (TransferUpload) TableName() string { return "transfer_uploads" }

// TransferUploadRepository persists transfer_uploads rows.
type TransferUploadRepository interface {
	// GetByFileHash returns the prior upload for the given SHA-256, or
	// (nil, nil) when this content was never uploaded before.
	GetByFileHash(ctx context.Context, fileHash string) (*TransferUpload, error)
	// GetByID returns one upload with its outcome fields.
	GetByID(ctx context.Context, id uint64) (*TransferUpload, error)
	// Create inserts a new upload row; the unique file_hash index makes a
	// concurrent duplicate fail rather than double-count.
	Create(ctx context.Context, upload *TransferUpload) error
	// Update persists status/reject/verify mutations.
	Update(ctx context.Context, upload *TransferUpload) error
	// ListByWorkMonth returns the uploads of one work month, newest first.
	ListByWorkMonth(ctx context.Context, workMonth string, limit int) ([]TransferUpload, error)
}
