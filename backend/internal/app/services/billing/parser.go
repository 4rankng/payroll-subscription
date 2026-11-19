package billing

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParsedTransferRow is one payment row from an uploaded bank-transfer result
// workbook. Identity is CCCD (founder decision: rows are keyed by CCCD, the
// donor's weekly-payment matching); amount is VND; reference is the optional
// bank transaction reference preserved onto the timesheet.
type ParsedTransferRow struct {
	RowNumber int    `json:"row"`
	CCCD      string `json:"cccd"`
	FullName  string `json:"full_name,omitempty"`
	Amount    int64  `json:"amount"`
	Reference string `json:"reference,omitempty"`
}

// ParseTransferResultExcel reads the uploaded workbook. Format follows the
// current weekly-payment result convention: one header row whose cells are
// matched case/whitespace-insensitively ("CCCD" required; amount column
// recognised by "số tiền"/"amount"/"thanh toán"; reference by
// "tham chiếu"/"reference"; name by "họ tên"/"ho ten"/"name"), then one data
// row per payment. Row numbers are 1-based Excel row numbers so per-row
// failures point straight at the file.
func ParseTransferResultExcel(r io.Reader) ([]ParsedTransferRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("transfer result: open workbook: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("transfer result: workbook has no sheets")
	}

	rows, err := f.GetRows(sheets[0], excelize.Options{RawCellValue: false})
	if err != nil {
		return nil, fmt.Errorf("transfer result: read rows: %w", err)
	}

	colCCCD, colAmount, colRef, colName, headerRow := -1, -1, -1, -1, -1
	for i, row := range rows {
		for j, cell := range row {
			norm := normalizeHeader(cell)
			if norm == "" {
				continue
			}
			if norm == "cccd" {
				colCCCD, headerRow = j, i
			} else if colAmount < 0 && isAmountHeader(norm) {
				colAmount = j
			} else if colRef < 0 && isReferenceHeader(norm) {
				colRef = j
			} else if colName < 0 && isNameHeader(norm) {
				colName = j
			}
		}
		if colCCCD >= 0 {
			break
		}
	}
	if colCCCD < 0 {
		return nil, fmt.Errorf("transfer result: no CCCD column found in header")
	}

	var out []ParsedTransferRow
	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]
		cccd := cellAt(row, colCCCD)
		if normalizeHeader(cccd) == "" {
			continue // blank separator row
		}
		rowNum := i + 1
		amount, parseErr := parseVND(cellAt(row, colAmount))
		if parseErr != nil {
			return nil, fmt.Errorf("transfer result: row %d: unreadable amount: %w", rowNum, parseErr)
		}
		out = append(out, ParsedTransferRow{
			RowNumber: rowNum,
			CCCD:      strings.TrimSpace(cccd),
			FullName:  strings.TrimSpace(cellAt(row, colName)),
			Amount:    amount,
			Reference: strings.TrimSpace(cellAt(row, colRef)),
		})
	}
	return out, nil
}

func cellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return row[col]
}

func normalizeHeader(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isAmountHeader(norm string) bool {
	return norm == "amount" || strings.Contains(norm, "số tiền") ||
		strings.Contains(norm, "thanh toan") || strings.Contains(norm, "thanh toán")
}

func isReferenceHeader(norm string) bool {
	return strings.Contains(norm, "tham chiếu") || strings.Contains(norm, "tham chieu") ||
		strings.Contains(norm, "reference")
}

func isNameHeader(norm string) bool {
	return norm == "họ tên" || norm == "ho ten" || norm == "fullname" || norm == "full name" || norm == "name"
}

// parseVND accepts "1234567", "1.234.567", "1,234,567" and " 1 234 567 đ".
func parseVND(s string) (int64, error) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case '.', ',', ' ', '₫', 'đ':
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	if clean == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v < 0 || v > math.MaxInt64 {
		return 0, fmt.Errorf("not a valid VND amount: %q", s)
	}
	return int64(v), nil
}
