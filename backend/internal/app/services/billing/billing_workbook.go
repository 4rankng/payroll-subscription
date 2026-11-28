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

	styles, err := newSaoKeStyles(f)
	if err != nil {
		return nil, err
	}

	summary := f.GetSheetName(0)
	summaryTitle := fmt.Sprintf("Báo cáo phí subscription — tháng %s", workMonth)
	if err := f.SetCellValue(summary, "A1", summaryTitle); err != nil {
		return nil, err
	}
	if err := f.MergeCell(summary, "A1", "D1"); err != nil {
		return nil, fmt.Errorf("billing report: merge summary title: %w", err)
	}
	if err := f.SetCellStyle(summary, "A1", "A1", styles.title); err != nil {
		return nil, err
	}
	if err := f.SetRowHeight(summary, 1, 28); err != nil {
		return nil, fmt.Errorf("billing report: summary title height: %w", err)
	}

	headers := []string{"Kỳ", "Số nhân viên (distinct CCCD)", "Đơn giá / nhân viên / kỳ (VND)", "Thành tiền (VND)"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 3)
		if err := f.SetCellValue(summary, cell, header); err != nil {
			return nil, err
		}
	}
	rowIdx := 4
	lineStyles := []int{styles.center, styles.center, styles.money, styles.money}
	for _, line := range lines {
		values := []any{fmt.Sprintf("Kỳ %d", line.Cycle), line.DistinctCount, fee, line.AmountVnd}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
			if err := f.SetCellValue(summary, cell, v); err != nil {
				return nil, err
			}
			if err := f.SetCellStyle(summary, cell, cell, lineStyles[col]); err != nil {
				return nil, err
			}
			widths.add(col+1, v, false)
		}
		rowIdx++
	}
	totalRow := rowIdx
	totalValues := []any{"TỔNG", totalDistinct, nil, totalVnd}
	for col, v := range totalValues {
		cell, _ := excelize.CoordinatesToCellName(col+1, totalRow)
		if v != nil {
			if err := f.SetCellValue(summary, cell, v); err != nil {
				return nil, err
			}
		}
		style := styles.totalLabel
		if col == 3 {
			style = styles.totalMoney
		}
		if err := f.SetCellStyle(summary, cell, cell, style); err != nil {
			return nil, err
		}
		widths.add(col+1, v, false)
	}
	if err := f.SetRowHeight(summary, totalRow, 20); err != nil {
		return nil, fmt.Errorf("billing report: total row height: %w", err)
	}
	lastKyRow := rowIdx - 1
	filterEnd, _ := excelize.CoordinatesToCellName(4, lastKyRow)
	if err := f.AutoFilter(summary, "A3:"+filterEnd, []excelize.AutoFilterOptions{}); err != nil {
		return nil, fmt.Errorf("billing report: summary filter: %w", err)
	}
	if err := saoKeFreeze(f, summary, 4); err != nil {
		return nil, err
	}
	if err := saoKeHideGridlines(f, summary); err != nil {
		return nil, err
	}
	if err := saoKeTabColor(f, summary, saoKeNavy); err != nil {
		return nil, err
	}
	if err := widths.apply(f, summary); err != nil {
		return nil, err
	}

	for _, sheet := range sheets {
		safe := safeSheetName(fmt.Sprintf("%s (%d)", sheet.ProjectName, sheet.ProjectID))
		if _, err := f.NewSheet(safe); err != nil {
			return nil, err
		}
		header := []string{"STT", "Họ tên", "CCCD", "Kỳ", "Ngày làm việc", "Số tiền (VND)", "Ngày thanh toán", "Mã tham chiếu"}
		widths := newSaoKeWidths(8)
		for j, h := range header {
			cell, _ := excelize.CoordinatesToCellName(j+1, 1)
			if err := f.SetCellValue(safe, cell, h); err != nil {
				return nil, err
			}
			widths.add(j+1, h, false)
		}
		if err := f.SetCellStyle(safe, "A1", "H1", styles.header); err != nil {
			return nil, err
		}
		if err := f.SetRowHeight(safe, 1, 24); err != nil {
			return nil, fmt.Errorf("billing report: project header height: %w", err)
		}
		rowStyles := []int{styles.center, styles.data, styles.text, styles.center, styles.center, styles.money, styles.center, styles.data}
		for i, row := range sheet.Rows {
			r := i + 2
			values := []any{
				i + 1, row.EmployeeName, row.CCCD, fmt.Sprintf("Kỳ %d", row.Cycle),
				row.Date, row.Amount, row.PaymentDate, row.PaymentReference,
			}
			for j, v := range values {
				cell, _ := excelize.CoordinatesToCellName(j+1, r)
				if err := f.SetCellValue(safe, cell, v); err != nil {
					return nil, err
				}
				if err := f.SetCellStyle(safe, cell, cell, rowStyles[j]); err != nil {
					return nil, err
				}
				widths.add(j+1, v, false)
			}
		}
		lastRow := len(sheet.Rows) + 1
		filterEnd, _ := excelize.CoordinatesToCellName(8, lastRow)
		if err := f.AutoFilter(safe, "A1:"+filterEnd, []excelize.AutoFilterOptions{}); err != nil {
			return nil, fmt.Errorf("billing report: project filter: %w", err)
		}
		if err := saoKeFreeze(f, safe, 2); err != nil {
			return nil, err
		}
		if err := saoKeHideGridlines(f, safe); err != nil {
			return nil, err
		}
		if err := saoKeTabColor(f, safe, saoKeBlue); err != nil {
			return nil, err
		}
		if err := widths.apply(f, safe); err != nil {
			return nil, err
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
