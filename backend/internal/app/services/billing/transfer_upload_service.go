package billing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// BillingNotifier pushes notifications to a whole role — the D10 all-admin
// fan-out (wallet-balance-alert mechanism, generic notifier).
type BillingNotifier interface {
	NotifyUsersByRole(ctx context.Context, role domain.UserRole, notificationType domain.NotificationType, title, message string) error
}

// RowFailure is one processable-but-failed row of an upload, reported per
// row (founder decision: process whatever the file contains — failed rows
// never block other rows).
type RowFailure struct {
	Row    int    `json:"row"`
	CCCD   string `json:"cccd"`
	Reason string `json:"reason"`
}

// UploadResult is the outcome of processing one transfer result upload.
type UploadResult struct {
	UploadID   uint64                      `json:"upload_id"`
	Status     domain.TransferUploadStatus `json:"status"`
	Duplicate  bool                        `json:"duplicate"`
	RowsTotal  int                         `json:"rows_total"`
	RowsOK     int                         `json:"rows_ok"`
	RowsFailed int                         `json:"rows_failed"`
	Failures   []RowFailure                `json:"failures"`
}

// TransferUploadService ingests and verifies uploaded bank-transfer results
// (FR-2). Per-row processing (current donor behaviour): matched rows mark
// their approved timesheets customer-paid and accumulate the per-employee-
// per-Kỳ billing atom; failed rows are reported; timesheets not in the file
// stay pending. A whole upload is rejected — with the D10 all-admin push —
// only when it has no processable rows at all.
type TransferUploadService struct {
	uploads    domain.TransferUploadRepository
	kyPaid     domain.EmployeePayCycleRepository
	timesheets domain.TimesheetRepository
	employees  domain.EmployeeRepository
	txManager  domain.TransactionManager
	notifier   BillingNotifier
	logger     *slog.Logger
}

func NewTransferUploadService(
	uploads domain.TransferUploadRepository,
	kyPaid domain.EmployeePayCycleRepository,
	timesheets domain.TimesheetRepository,
	employees domain.EmployeeRepository,
	txManager domain.TransactionManager,
	notifier BillingNotifier,
	logger *slog.Logger,
) *TransferUploadService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TransferUploadService{
		uploads:    uploads,
		kyPaid:     kyPaid,
		timesheets: timesheets,
		employees:  employees,
		txManager:  txManager,
		notifier:   notifier,
		logger:     logger,
	}
}

// ProcessUpload ingests one uploaded transfer result for (cycle, workMonth).
// Idempotent on the file content: an identical re-upload returns the prior
// outcome without touching the books.
func (s *TransferUploadService) ProcessUpload(ctx context.Context, cycle int, workMonth string, filename string, data []byte, uploadedBy uint) (*UploadResult, error) {
	sum := sha256.Sum256(data)
	fileHash := hex.EncodeToString(sum[:])

	prior, err := s.uploads.GetByFileHash(ctx, fileHash)
	if err != nil {
		return nil, fmt.Errorf("billing: idempotency lookup: %w", err)
	}
	if prior != nil {
		return &UploadResult{UploadID: prior.ID, Status: prior.Status, Duplicate: true}, nil
	}

	rows, parseErr := ParseTransferResultExcel(bytes.NewReader(data))
	upload := &domain.TransferUpload{
		Cycle:            cycle,
		WorkMonth:        workMonth,
		FileHash:         fileHash,
		OriginalFilename: filename,
		UploadedBy:       uploadedBy,
		Status:           domain.TransferUploadStatusPending,
		CreatedAt:        clock.Now(),
		UpdatedAt:        clock.Now(),
	}
	if err := s.uploads.Create(ctx, upload); err != nil {
		return nil, fmt.Errorf("billing: create upload: %w", err)
	}

	if parseErr != nil {
		reason := truncate(parseErr.Error(), 500)
		upload.Status = domain.TransferUploadStatusRejected
		upload.RejectReason = &reason
		if err := s.uploads.Update(ctx, upload); err != nil {
			return nil, fmt.Errorf("billing: mark rejected: %w", err)
		}
		s.notifyRejected(ctx, upload, reason)
		return &UploadResult{UploadID: upload.ID, Status: upload.Status, Failures: []RowFailure{{Reason: reason}}}, nil
	}

	result := &UploadResult{UploadID: upload.ID, RowsTotal: len(rows), Failures: []RowFailure{}}

	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.processRows(txCtx, upload, rows, result)
	})
	if err != nil {
		return nil, fmt.Errorf("billing: process rows: %w", err)
	}

	now := clock.Now()
	if result.RowsOK > 0 {
		upload.Status = domain.TransferUploadStatusVerified
		upload.VerifiedAt = &now
		by := uploadedBy
		upload.VerifiedBy = &by
	} else {
		reason := fmt.Sprintf("no processable rows: %d rows all failed", result.RowsTotal)
		upload.Status = domain.TransferUploadStatusRejected
		upload.RejectReason = &reason
	}
	upload.UpdatedAt = now
	if err := s.uploads.Update(ctx, upload); err != nil {
		return nil, fmt.Errorf("billing: finalize upload: %w", err)
	}
	result.Status = upload.Status

	if upload.Status == domain.TransferUploadStatusRejected {
		s.notifyRejected(ctx, upload, deref(upload.RejectReason))
	}
	s.logger.Info("transfer upload processed",
		"upload_id", upload.ID, "cycle", cycle, "work_month", workMonth,
		"rows_total", result.RowsTotal, "rows_ok", result.RowsOK, "rows_failed", result.RowsFailed,
		"status", upload.Status,
	)
	return result, nil
}

// processRows walks the parsed rows inside the caller's transaction. A row
// succeeds only when its CCCD resolves to an employee with ≥1 pending
// approved timesheet inside the Kỳ window; all matched timesheets of the
// employee are marked customer-paid (D11) and the billing atom is
// accumulated (founder decision: one row per employee per Kỳ).
func (s *TransferUploadService) processRows(ctx context.Context, upload *domain.TransferUpload, rows []ParsedTransferRow, result *UploadResult) error {
	from, to := window(upload.Cycle, upload.WorkMonth)
	now := clock.Now()
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, row := range rows {
		employee, err := s.employees.GetByCCCD(ctx, row.CCCD)
		if err != nil || employee == nil {
			result.Failures = append(result.Failures, RowFailure{Row: row.RowNumber, CCCD: row.CCCD, Reason: "không tìm thấy nhân viên theo CCCD"})
			result.RowsFailed++
			continue
		}

		timesheets, err := s.timesheets.GetByEmployee(ctx, employee.ID, from, to)
		if err != nil {
			return fmt.Errorf("load timesheets for employee %d: %w", employee.ID, err)
		}

		var matched []domain.Timesheet
		var matchedAmount int64
		for _, ts := range timesheets {
			if ts.Status == domain.TimesheetStatusApproved && ts.PaymentStatus == domain.PaymentStatusPending {
				matched = append(matched, *ts)
				matchedAmount += ts.Amount
			}
		}
		if len(matched) == 0 {
			result.Failures = append(result.Failures, RowFailure{Row: row.RowNumber, CCCD: row.CCCD, Reason: "không có timesheet đã duyệt chờ thanh toán trong kỳ"})
			result.RowsFailed++
			continue
		}

		updates := make([]domain.PaymentStatusUpdate, 0, len(matched))
		for i := range matched {
			ts := &matched[i]
			ref := row.Reference
			amount := ts.Amount
			updates = append(updates, domain.PaymentStatusUpdate{
				TimesheetID:      ts.ID,
				PaymentStatus:    domain.PaymentStatusPaid,
				PaymentReference: &ref,
				PaymentDate:      &nowDate,
				PaidAmount:       &amount,
				PaidAt:           &now,
			})
			if err := s.kyPaid.UpsertAmount(ctx, &domain.EmployeePayCycle{
				Cycle:          upload.Cycle,
				WorkMonth:      upload.WorkMonth,
				EmployeeID:     employee.ID,
				ProjectID:      ts.ProjectID,
				TimesheetID:    ts.ID,
				SourceUploadID: upload.ID,
				PaidAmount:     amount,
				CreatedAt:      now,
			}); err != nil {
				return fmt.Errorf("accumulate ky paid record (employee %d, ky %d): %w", employee.ID, upload.Cycle, err)
			}
		}
		if err := s.timesheets.BulkUpdatePaymentStatus(ctx, updates); err != nil {
			return fmt.Errorf("mark %d timesheets paid (employee %d): %w", len(updates), employee.ID, err)
		}
		result.RowsOK++
	}
	return nil
}

// notifyRejected fires the D10 all-admin push for a rejected upload. A failed
// notification is logged, never fatal — the upload outcome is already saved.
func (s *TransferUploadService) notifyRejected(ctx context.Context, upload *domain.TransferUpload, reason string) {
	title := "Kết quả chuyển khoản bị từ chối"
	message := fmt.Sprintf(
		"File kết quả chuyển khoản kỳ %d tháng %s (%s) bị từ chối: %s. Số đếm của kỳ bị đóng băng cho đến khi có file sửa đúng.",
		upload.Cycle, upload.WorkMonth, upload.OriginalFilename, reason,
	)
	if err := s.notifier.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeTransferUploadRejected, title, message); err != nil {
		s.logger.Error("transfer upload rejection push failed", "upload_id", upload.ID, "error", err)
	}
}

// window resolves the Kỳ work window (clock is the canonical calendar).
func window(cycle int, workMonth string) (time.Time, time.Time) {
	monthTime, err := time.ParseInLocation("2006-01", workMonth, clock.DefaultLocation)
	if err != nil {
		now := clock.Now()
		monthTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, clock.DefaultLocation)
	}
	return (clock.TimesheetPayCycle{Ky: cycle, WorkMonth: monthTime}).CycleWindow()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ListUploads returns the upload history of one work month, newest first.
func (s *TransferUploadService) ListUploads(ctx context.Context, workMonth string, limit int) ([]domain.TransferUpload, error) {
	return s.uploads.ListByWorkMonth(ctx, workMonth, limit)
}

// ListEmployeePayCycles returns the per-employee-per-Kỳ billing rows of one work
// month — the FR-1 read model behind the FR-3 billing report.
func (s *TransferUploadService) ListEmployeePayCycles(ctx context.Context, workMonth string) ([]domain.EmployeePayCycle, error) {
	return s.kyPaid.ListByWorkMonth(ctx, workMonth)
}

// MonthlySummary returns distinct-employee counts per Kỳ for one work month.
func (s *TransferUploadService) MonthlySummary(ctx context.Context, workMonth string) (map[int]int64, error) {
	return s.kyPaid.CountDistinctEmployeesByCycle(ctx, workMonth)
}
