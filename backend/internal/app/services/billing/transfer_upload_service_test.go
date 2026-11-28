package billing

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// uploadRepoStub records the mutations ProcessUpload makes so the tests can
// assert what would reach the database.
type uploadRepoStub struct {
	prior   *domain.TransferUpload
	updates []*domain.TransferUpload
	creates []*domain.TransferUpload
	deletes []uint64
}

func (s *uploadRepoStub) GetByFileHash(ctx context.Context, fileHash string) (*domain.TransferUpload, error) {
	return s.prior, nil
}

func (s *uploadRepoStub) GetByID(ctx context.Context, id uint64) (*domain.TransferUpload, error) {
	return s.prior, nil
}

func (s *uploadRepoStub) Create(ctx context.Context, upload *domain.TransferUpload) error {
	s.creates = append(s.creates, upload)
	return nil
}

func (s *uploadRepoStub) Delete(ctx context.Context, id uint64) error {
	s.deletes = append(s.deletes, id)
	return nil
}

func (s *uploadRepoStub) Update(ctx context.Context, upload *domain.TransferUpload) error {
	s.updates = append(s.updates, upload)
	return nil
}

func (s *uploadRepoStub) ListByWorkMonth(ctx context.Context, workMonth string, limit int) ([]domain.TransferUpload, error) {
	return nil, nil
}

// kyPaidStub serves the billing atoms of one work month.
type kyPaidStub struct {
	rows    []domain.EmployeePayCycle
	upserts []*domain.EmployeePayCycle
}

func (s *kyPaidStub) UpsertAmount(ctx context.Context, record *domain.EmployeePayCycle) error {
	s.upserts = append(s.upserts, record)
	return nil
}

func (s *kyPaidStub) ListByWorkMonth(ctx context.Context, workMonth string) ([]domain.EmployeePayCycle, error) {
	return s.rows, nil
}

func (s *kyPaidStub) CountDistinctEmployeesByCycle(ctx context.Context, workMonth string) (map[int]int64, error) {
	return nil, nil
}

func (s *kyPaidStub) CountDistinctEmployeesByCycles(ctx context.Context, fromMonth, toMonth string) ([]domain.CycleMonthCount, error) {
	return nil, nil
}

func (s *kyPaidStub) BankTransferHistory(ctx context.Context, f domain.BankTransferHistoryFilter) ([]domain.BankTransferHistoryRow, domain.BankTransferHistorySummary, error) {
	return nil, domain.BankTransferHistorySummary{}, nil
}

// assetStorerStub records what the service stores for re-download.
type assetStorerStub struct {
	calls []storedResultFile
}

type storedResultFile struct {
	data       []byte
	filename   string
	uploadType string
	uploadedBy uint
}

func (s *assetStorerStub) UploadAssetFromBytes(ctx context.Context, data []byte, filename string, uploadType string, uploadedBy uint) (*domain.Asset, error) {
	s.calls = append(s.calls, storedResultFile{data: data, filename: filename, uploadType: uploadType, uploadedBy: uploadedBy})
	return &domain.Asset{ID: uint(len(s.calls)), Filename: filename, UploadType: uploadType, UploadedBy: uploadedBy}, nil
}

// transferResultBytes builds a one-row transfer result workbook in the
// upload format the parser accepts.
func transferResultBytes(t *testing.T, reference string) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	rows := [][]interface{}{
		{"Họ tên", "CCCD", "Số tiền", "Mã tham chiếu"},
		{"Nguyễn Văn A", "012345678901", "800000", reference},
	}
	for i, row := range rows {
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &row); err != nil {
			t.Fatalf("set row: %v", err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	return buf.Bytes()
}

// TestProcessUploadDuplicateHealsStaleCounters covers the Sổ cái "Dòng
// khớp" residue path (PAYSUB-23): an identical re-upload of a verified file
// must return the full outcome and repair counters a pre-fix build left at
// zero, without touching the books.
func TestProcessUploadDuplicateHealsStaleCounters(t *testing.T) {
	now := time.Now()
	prior := &domain.TransferUpload{
		ID: 7, Cycle: 4, WorkMonth: "2026-11", FileHash: "prior-hash",
		OriginalFilename: "proof.xlsx", UploadedBy: 1,
		Status: domain.TransferUploadStatusVerified, CreatedAt: now, UpdatedAt: now,
	}
	uploads := &uploadRepoStub{prior: prior}
	ky := &kyPaidStub{rows: []domain.EmployeePayCycle{{
		Cycle: 4, WorkMonth: "2026-11", EmployeeID: 1, ProjectID: 1,
		TimesheetID: 1, SourceUploadID: prior.ID, PaidAmount: 800000, CreatedAt: now,
	}}}
	svc := NewTransferUploadService(uploads, ky, nil, nil, nil, nil, nil, nil)

	result, err := svc.ProcessUpload(context.Background(), 4, "2026-11", "proof.xlsx", transferResultBytes(t, "FT-HEAL"), 1)
	if err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if !result.Duplicate || result.UploadID != prior.ID || result.Status != domain.TransferUploadStatusVerified {
		t.Fatalf("duplicate outcome mismatch: %+v", result)
	}
	if result.RowsTotal != 1 || result.RowsOK != 1 || result.RowsFailed != 0 {
		t.Fatalf("repaired outcome = total %d, ok %d, failed %d; want 1, 1, 0", result.RowsTotal, result.RowsOK, result.RowsFailed)
	}
	if len(uploads.updates) != 1 {
		t.Fatalf("want exactly 1 counter repair update, got %d", len(uploads.updates))
	}
	repaired := uploads.updates[0]
	if repaired.ParsedRows != 1 || repaired.MatchedRows != 1 {
		t.Fatalf("repaired counters = parsed %d, matched %d; want 1, 1", repaired.ParsedRows, repaired.MatchedRows)
	}
	if len(uploads.creates) != 0 || len(uploads.deletes) != 0 || len(ky.upserts) != 0 {
		t.Fatalf("duplicate upload touched the books: creates=%d deletes=%d upserts=%d",
			len(uploads.creates), len(uploads.deletes), len(ky.upserts))
	}
}

// TestProcessUploadDuplicateKeepsCorrectCounters: an identical re-upload of
// a healthy upload returns its outcome without any write.
func TestProcessUploadDuplicateKeepsCorrectCounters(t *testing.T) {
	now := time.Now()
	prior := &domain.TransferUpload{
		ID: 7, Cycle: 4, WorkMonth: "2026-11", FileHash: "prior-hash",
		OriginalFilename: "proof.xlsx", UploadedBy: 1,
		Status: domain.TransferUploadStatusVerified, ParsedRows: 1, MatchedRows: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	uploads := &uploadRepoStub{prior: prior}
	ky := &kyPaidStub{rows: []domain.EmployeePayCycle{{
		Cycle: 4, WorkMonth: "2026-11", EmployeeID: 1, ProjectID: 1,
		TimesheetID: 1, SourceUploadID: prior.ID, PaidAmount: 800000, CreatedAt: now,
	}}}
	svc := NewTransferUploadService(uploads, ky, nil, nil, nil, nil, nil, nil)

	result, err := svc.ProcessUpload(context.Background(), 4, "2026-11", "proof.xlsx", transferResultBytes(t, "FT-HEAL"), 1)
	if err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if !result.Duplicate || result.RowsTotal != 1 || result.RowsOK != 1 || result.RowsFailed != 0 {
		t.Fatalf("outcome mismatch: %+v", result)
	}
	if len(uploads.updates) != 0 {
		t.Fatalf("healthy counters must not be rewritten, got %d updates", len(uploads.updates))
	}
}

// TestProcessUploadDuplicateBackfillsStoredWorkbook (PAYSUB-24): a verified
// upload whose row predates workbook storage must gain its asset on an
// identical re-upload, so the Sổ cái file name becomes downloadable.
func TestProcessUploadDuplicateBackfillsStoredWorkbook(t *testing.T) {
	now := time.Now()
	prior := &domain.TransferUpload{
		ID: 7, Cycle: 4, WorkMonth: "2026-11", FileHash: "prior-hash",
		OriginalFilename: "proof.xlsx", UploadedBy: 1,
		Status: domain.TransferUploadStatusVerified, ParsedRows: 1, MatchedRows: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	uploads := &uploadRepoStub{prior: prior}
	ky := &kyPaidStub{rows: []domain.EmployeePayCycle{{
		Cycle: 4, WorkMonth: "2026-11", EmployeeID: 1, ProjectID: 1,
		TimesheetID: 1, SourceUploadID: prior.ID, PaidAmount: 800000, CreatedAt: now,
	}}}
	assets := &assetStorerStub{}
	svc := NewTransferUploadService(uploads, ky, nil, nil, nil, assets, nil, nil)

	data := transferResultBytes(t, "FT-ASSET")
	result, err := svc.ProcessUpload(context.Background(), 4, "2026-11", "proof.xlsx", data, 1)
	if err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if !result.Duplicate || result.RowsTotal != 1 || result.RowsOK != 1 {
		t.Fatalf("outcome mismatch: %+v", result)
	}
	if len(assets.calls) != 1 {
		t.Fatalf("want exactly 1 stored workbook, got %d", len(assets.calls))
	}
	stored := assets.calls[0]
	if stored.filename != "proof.xlsx" || stored.uploadType != domain.UploadTypeBulkTransferResult || string(stored.data) != string(data) {
		t.Fatalf("stored workbook mismatch: filename=%q type=%q bytes_equal=%v",
			stored.filename, stored.uploadType, string(stored.data) == string(data))
	}
	if len(uploads.updates) != 1 || uploads.updates[0].AssetID == nil || *uploads.updates[0].AssetID != 1 {
		t.Fatalf("asset link must be repaired onto the upload row, got %+v", uploads.updates)
	}
}

// TestProcessUploadDuplicateKeepsStoredWorkbook: an upload that already has
// its asset is never re-stored or rewritten on a duplicate re-upload.
func TestProcessUploadDuplicateKeepsStoredWorkbook(t *testing.T) {
	now := time.Now()
	assetID := uint64(9)
	prior := &domain.TransferUpload{
		ID: 7, Cycle: 4, WorkMonth: "2026-11", FileHash: "prior-hash",
		OriginalFilename: "proof.xlsx", AssetID: &assetID, UploadedBy: 1,
		Status: domain.TransferUploadStatusVerified, ParsedRows: 1, MatchedRows: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	uploads := &uploadRepoStub{prior: prior}
	ky := &kyPaidStub{rows: []domain.EmployeePayCycle{{
		Cycle: 4, WorkMonth: "2026-11", EmployeeID: 1, ProjectID: 1,
		TimesheetID: 1, SourceUploadID: prior.ID, PaidAmount: 800000, CreatedAt: now,
	}}}
	assets := &assetStorerStub{}
	svc := NewTransferUploadService(uploads, ky, nil, nil, nil, assets, nil, nil)

	if _, err := svc.ProcessUpload(context.Background(), 4, "2026-11", "proof.xlsx", transferResultBytes(t, "FT-ASSET"), 1); err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if len(assets.calls) != 0 || len(uploads.updates) != 0 {
		t.Fatalf("healthy upload must not be rewritten: stores=%d updates=%d", len(assets.calls), len(uploads.updates))
	}
}

// TestProcessUploadDuplicateUnparsableFallsBackToStoredOutcome: the identical
// bytes verified before, so a parse failure must fall back to the stored
// outcome instead of guessing.
func TestProcessUploadDuplicateUnparsableFallsBackToStoredOutcome(t *testing.T) {
	now := time.Now()
	prior := &domain.TransferUpload{
		ID: 7, Cycle: 4, WorkMonth: "2026-11", FileHash: "prior-hash",
		OriginalFilename: "proof.xlsx", UploadedBy: 1,
		Status: domain.TransferUploadStatusVerified, ParsedRows: 2, MatchedRows: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	uploads := &uploadRepoStub{prior: prior}
	svc := NewTransferUploadService(uploads, &kyPaidStub{}, nil, nil, nil, nil, nil, nil)

	result, err := svc.ProcessUpload(context.Background(), 4, "2026-11", "proof.xlsx", []byte("not a workbook"), 1)
	if err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if !result.Duplicate || result.RowsTotal != 2 || result.RowsOK != 1 || result.RowsFailed != 1 {
		t.Fatalf("fallback outcome mismatch: %+v", result)
	}
	if len(uploads.updates) != 0 {
		t.Fatalf("fallback must not rewrite counters, got %d updates", len(uploads.updates))
	}
}
