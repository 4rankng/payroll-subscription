package excel

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// PaymentHistoryExportRow is one payment line on a per-project sheet of the
// payment-history workbook (Lịch sử trả lương).
type PaymentHistoryExportRow struct {
	EmployeeName string
	CCCD         string
	Position     string
	AmountVnd    int64
	PayDate      string // DD/MM/YYYY, empty when unknown
}

// PaymentHistoryProjectSheet is one project's payment lines, rendered as one
// workbook sheet.
type PaymentHistoryProjectSheet struct {
	ProjectID   uint
	ProjectName string
	Rows        []PaymentHistoryExportRow
}

// paymentHistoryHeaders is the fixed column contract of every project sheet.
var paymentHistoryHeaders = []string{"STT", "Nhân viên", "CCCD", "Vị trí", "Số tiền", "Ngày thanh toán"}

// RenderPaymentHistoryWorkbook renders one sheet per project — STT | Nhân
// viên | CCCD | Vị trí | Số tiền | Ngày thanh toán — and returns the xlsx
// bytes. A workbook with no project sheets still carries one header-only
// sheet so the export is never an empty file.
func RenderPaymentHistoryWorkbook(sheets []PaymentHistoryProjectSheet) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if len(sheets) == 0 {
		empty := f.GetSheetName(0)
		if err := f.SetSheetName(empty, "payment-history"); err != nil {
			return nil, fmt.Errorf("payment history workbook: %w", err)
		}
		if err := writePaymentHistorySheet(f, "payment-history", nil); err != nil {
			return nil, err
		}
	} else {
		if err := f.SetSheetName(f.GetSheetName(0), paymentHistorySheetName(sheets[0])); err != nil {
			return nil, fmt.Errorf("payment history workbook: %w", err)
		}
		for i, sheet := range sheets {
			name := paymentHistorySheetName(sheet)
			if i > 0 {
				if _, err := f.NewSheet(name); err != nil {
					return nil, fmt.Errorf("payment history workbook: %w", err)
				}
			}
			if err := writePaymentHistorySheet(f, name, sheet.Rows); err != nil {
				return nil, err
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("payment history workbook: %w", err)
	}
	return buf.Bytes(), nil
}

func writePaymentHistorySheet(f *excelize.File, sheet string, rows []PaymentHistoryExportRow) error {
	for j, h := range paymentHistoryHeaders {
		cell, err := excelize.CoordinatesToCellName(j+1, 1)
		if err != nil {
			return fmt.Errorf("payment history workbook: %w", err)
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return fmt.Errorf("payment history workbook: %w", err)
		}
	}
	for i, row := range rows {
		values := []interface{}{
			i + 1, row.EmployeeName, row.CCCD, row.Position, row.AmountVnd, row.PayDate,
		}
		for j, v := range values {
			cell, err := excelize.CoordinatesToCellName(j+1, i+2)
			if err != nil {
				return fmt.Errorf("payment history workbook: %w", err)
			}
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return fmt.Errorf("payment history workbook: %w", err)
			}
		}
	}
	return nil
}

// paymentHistorySheetName builds an Excel-safe per-project sheet name:
// "ProjectName (id)" stripped of forbidden characters and truncated to 31
// characters (Excel's hard limit).
func paymentHistorySheetName(sheet PaymentHistoryProjectSheet) string {
	out := make([]rune, 0, len(sheet.ProjectName)+8)
	for _, r := range fmt.Sprintf("%s (%d)", sheet.ProjectName, sheet.ProjectID) {
		switch r {
		case ':', '\\', '/', '?', '*', '[', ']':
			continue
		}
		out = append(out, r)
	}
	if len(out) > 31 {
		out = out[:31]
	}
	return string(out)
}
