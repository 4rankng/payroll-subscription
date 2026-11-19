package billing

import (
	"bytes"
	"fmt"

	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// renderBillingWorkbook renders the D24 workbook: a Summary sheet (per-Kỳ
// distinct × fee + total) followed by one sheet per project listing that
// project's payment-history rows. Returns the xlsx bytes.
func renderBillingWorkbook(workMonth string, fee int64, lines []domain.BillingReportLine, totalDistinct int, totalVnd int64, sheets []projectSheet) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	summary := f.GetSheetName(0)
	summaryTitle := fmt.Sprintf("Báo cáo phí subscription — tháng %s", workMonth)
	cells := []struct{ cell, value string }{
		{"A1", summaryTitle},
		{"A3", "Kỳ"},
		{"B3", "Số nhân viên (distinct CCCD)"},
		{"C3", "Đơn giá / nhân viên / kỳ (VND)"},
		{"D3", "Thành tiền (VND)"},
	}
	for _, c := range cells {
		if err := f.SetCellValue(summary, c.cell, c.value); err != nil {
			return nil, err
		}
	}
	rowIdx := 4
	for _, line := range lines {
		if err := f.SetCellValue(summary, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("Kỳ %d", line.Cycle)); err != nil {
			return nil, err
		}
		if err := f.SetCellValue(summary, fmt.Sprintf("B%d", rowIdx), line.DistinctCount); err != nil {
			return nil, err
		}
		if err := f.SetCellValue(summary, fmt.Sprintf("C%d", rowIdx), fee); err != nil {
			return nil, err
		}
		if err := f.SetCellValue(summary, fmt.Sprintf("D%d", rowIdx), line.AmountVnd); err != nil {
			return nil, err
		}
		rowIdx++
	}
	if err := f.SetCellValue(summary, fmt.Sprintf("A%d", rowIdx), "TỔNG"); err != nil {
		return nil, err
	}
	if err := f.SetCellValue(summary, fmt.Sprintf("B%d", rowIdx), totalDistinct); err != nil {
		return nil, err
	}
	if err := f.SetCellValue(summary, fmt.Sprintf("D%d", rowIdx), totalVnd); err != nil {
		return nil, err
	}

	for _, sheet := range sheets {
		safe := safeSheetName(fmt.Sprintf("%s (%d)", sheet.ProjectName, sheet.ProjectID))
		if _, err := f.NewSheet(safe); err != nil {
			return nil, err
		}
		header := []string{"STT", "Họ tên", "CCCD", "Kỳ", "Ngày làm việc", "Số tiền (VND)", "Ngày thanh toán", "Mã tham chiếu"}
		for j, h := range header {
			if err := f.SetCellValue(safe, fmt.Sprintf("%s1", colLetter(j)), h); err != nil {
				return nil, err
			}
		}
		for i, row := range sheet.Rows {
			r := i + 2
			values := []interface{}{
				i + 1, row.EmployeeName, row.CCCD, fmt.Sprintf("Kỳ %d", row.Cycle),
				row.Date, row.Amount, row.PaymentDate, row.PaymentReference,
			}
			for j, v := range values {
				if err := f.SetCellValue(safe, fmt.Sprintf("%s%d", colLetter(j), r), v); err != nil {
					return nil, err
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// safeSheetName strips characters Excel forbids in sheet names and truncates
// to 31 characters (Excel's hard limit).
func safeSheetName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
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

func colLetter(i int) string {
	if i < 0 {
		return "A"
	}
	name := ""
	for {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
		if i < 0 {
			break
		}
	}
	return name
}
