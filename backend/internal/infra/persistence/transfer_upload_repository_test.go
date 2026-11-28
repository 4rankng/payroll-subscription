package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTransferUploadUpdatePersistsRowCounters guards the Sổ cái "Dòng khớp"
// contract: ProcessUpload finalizes parsed_rows/matched_rows through Update,
// so the update map must write both columns — a dropped column silently
// leaves the upload list at 0 while the ky rows are all there.
func TestTransferUploadUpdatePersistsRowCounters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE transfer_uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cycle INTEGER NOT NULL,
			work_month TEXT NOT NULL,
			file_hash TEXT NOT NULL UNIQUE,
			original_filename TEXT NOT NULL,
			asset_id INTEGER,
			uploaded_by INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			parsed_rows INTEGER NOT NULL DEFAULT 0,
			matched_rows INTEGER NOT NULL DEFAULT 0,
			reject_reason TEXT,
			verified_at DATETIME,
			verified_by INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create transfer_uploads table: %v", err)
	}
	repo := &TransferUploadRepository{BaseRepository: NewBaseRepository(&Database{DB: db})}

	now := time.Now()
	upload := &domain.TransferUpload{
		Cycle:            4,
		WorkMonth:        "2026-11",
		FileHash:         "itest-hash",
		OriginalFilename: "itest.xlsx",
		UploadedBy:       1,
		Status:           domain.TransferUploadStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := repo.Create(context.Background(), upload); err != nil {
		t.Fatalf("create upload: %v", err)
	}

	upload.Status = domain.TransferUploadStatusVerified
	upload.ParsedRows = 20
	upload.MatchedRows = 20
	upload.VerifiedAt = &now
	by := uint(1)
	upload.VerifiedBy = &by
	upload.UpdatedAt = now
	if err := repo.Update(context.Background(), upload); err != nil {
		t.Fatalf("update upload: %v", err)
	}

	got, err := repo.GetByID(context.Background(), upload.ID)
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	if got.ParsedRows != 20 || got.MatchedRows != 20 {
		t.Fatalf("row counters after Update = parsed %d, matched %d; want 20, 20", got.ParsedRows, got.MatchedRows)
	}
	if got.Status != domain.TransferUploadStatusVerified {
		t.Fatalf("status after Update = %q; want verified", got.Status)
	}
}
