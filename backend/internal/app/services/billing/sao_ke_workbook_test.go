package billing

import (
	"bytes"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// saoKeTestData builds a two-row single-project statement with deterministic
// labels for the workbook-rendering assertions below.
func saoKeTestData() *SaoKeData {
	rows := []SaoKeRow{
		{TimesheetID: 11, EmployeeID: 1, EmployeeName: "Nguyễn Văn An", CCCD: "031301008158", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 1234567, PaymentDate: "25/11/2026"},
		{TimesheetID: 12, EmployeeID: 2, EmployeeName: "Trần Thị Bình", CCCD: "001171012172", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 89000, PaymentDate: "25/11/2026"},
	}
	cycle := SaoKeCycle{
		Ky:        4,
		WorkMonth: "2026-11",
		PayDate:   time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		From:      time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
	}
	return &SaoKeData{
		Cycle:        cycle,
		ReportDate:   time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC),
		Rows:         rows,
		Groups:       []SaoKeProjectGroup{{ProjectID: 7, ProjectName: "Dự án Alpha", EmployeeCount: 2, Total: 1323567, Rows: rows}},
		Total:        1323567,
		TimesheetIDs: []uint{11, 12},
	}
}

func openSaoKeWorkbook(t *testing.T, data *SaoKeData) *excelize.File {
	t.Helper()
	raw, _, err := renderSaoKeWorkbook(data)
	if err != nil {
		t.Fatalf("renderSaoKeWorkbook: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func saoKeCell(t *testing.T, f *excelize.File, sheet, cell string, raw bool) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: raw})
	if err != nil {
		t.Fatalf("cell %s!%s: %v", sheet, cell, err)
	}
	return v
}

func saoKeWantWidth(t *testing.T, f *excelize.File, sheet, col string, want float64) {
	t.Helper()
	got, err := f.GetColWidth(sheet, col)
	if err != nil {
		t.Fatalf("col width %s!%s: %v", sheet, col, err)
	}
	if got != want {
		t.Fatalf("col width %s!%s = %v, want %v", sheet, col, got, want)
	}
}

func TestSaoKeWorkbookSheetsAndFilename(t *testing.T) {
	data := saoKeTestData()
	raw, filename, err := renderSaoKeWorkbook(data)
	if err != nil {
		t.Fatalf("renderSaoKeWorkbook: %v", err)
	}
	if filename != "sao_ke_tt_28-11-2026.xlsx" {
		t.Fatalf("filename = %q", filename)
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if got := f.GetSheetName(0); got != "Chi tiết giao dịch" {
		t.Fatalf("sheet 0 = %q", got)
	}
	if got := f.GetSheetName(1); got != "Dự án Alpha" {
		t.Fatalf("sheet 1 = %q", got)
	}
	if got := f.GetSheetName(2); got != "INTERNAL" {
		t.Fatalf("sheet 2 = %q", got)
	}

	// a statement without timesheet ids has no INTERNAL sheet
	noInternal := *data
	noInternal.TimesheetIDs = nil
	raw2, _, err := renderSaoKeWorkbook(&noInternal)
	if err != nil {
		t.Fatalf("render without timesheet ids: %v", err)
	}
	f2, err := excelize.OpenReader(bytes.NewReader(raw2))
	if err != nil {
		t.Fatalf("open without INTERNAL: %v", err)
	}
	defer f2.Close()
	if got := f2.GetSheetName(2); got != "" {
		t.Fatalf("unexpected sheet 2 without timesheet ids: %q", got)
	}
}

func TestSaoKeWorkbookSummarySheet(t *testing.T) {
	f := openSaoKeWorkbook(t, saoKeTestData())
	sheet := "Chi tiết giao dịch"

	// stable table: two data rows (4–5) and the total row (6)
	if got := saoKeCell(t, f, sheet, "A3", true); got != "STT" {
		t.Fatalf("A3 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "D3", true); got != "Số tiền (₫)" {
		t.Fatalf("D3 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "B4", true); got != "Kỳ 4 (22/11/2026 – 30/11/2026)" {
		t.Fatalf("B4 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "A4", true); got != "1" {
		t.Fatalf("A4 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "D4", true); got != "1234567" {
		t.Fatalf("D4 raw = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "D4", false); got != "1,234,567" {
		t.Fatalf("D4 formatted = %q, want #,##0 applied", got)
	}
	if got := saoKeCell(t, f, sheet, "C6", true); got != "TỔNG CỘNG" {
		t.Fatalf("C6 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "D6", false); got != "1,323,567 ₫" {
		t.Fatalf("D6 formatted = %q, want total ₫ format applied", got)
	}

	// widths: longest region-cell display length + 4, clamped to [8, 36]
	saoKeWantWidth(t, f, sheet, "A", 8)
	saoKeWantWidth(t, f, sheet, "B", 34) // 30-rune Kỳ label + 4
	saoKeWantWidth(t, f, sheet, "C", 15) // 11-rune "Dự án Alpha" + 4
	saoKeWantWidth(t, f, sheet, "D", 15) // 11-rune header/total display + 4

	panes, err := f.GetPanes(sheet)
	if err != nil || !panes.Freeze || panes.YSplit != 3 {
		t.Fatalf("summary panes = %+v (err %v), want frozen above row 4", panes, err)
	}
	view, err := f.GetSheetView(sheet, -1)
	if err != nil || view.ShowGridLines == nil || *view.ShowGridLines {
		t.Fatalf("summary gridlines = %+v (err %v), want hidden", view.ShowGridLines, err)
	}
	props, err := f.GetSheetProps(sheet)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "1F3864" {
		t.Fatalf("summary tab color = %+v (err %v), want 1F3864", props.TabColorRGB, err)
	}
}

func TestSaoKeWorkbookProjectSheet(t *testing.T) {
	f := openSaoKeWorkbook(t, saoKeTestData())
	sheet := "Dự án Alpha"

	if got := saoKeCell(t, f, sheet, "E6", true); got != "Số tiền (₫)" {
		t.Fatalf("E6 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "B4", false); got != "1,323,567 ₫" {
		t.Fatalf("B4 formatted = %q, want ₫ format applied", got)
	}
	if got := saoKeCell(t, f, sheet, "C7", false); got != "031301008158" {
		t.Fatalf("C7 = %q, want leading zeros preserved as text", got)
	}
	if got := saoKeCell(t, f, sheet, "B7", true); got != "Nguyễn Văn An" {
		t.Fatalf("B7 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "F7", true); got != "25/11/2026" {
		t.Fatalf("F7 = %q", got)
	}

	saoKeWantWidth(t, f, sheet, "A", 8)
	saoKeWantWidth(t, f, sheet, "B", 17) // 13-rune employee name beats 11-rune B4 total
	saoKeWantWidth(t, f, sheet, "C", 16) // 12-rune CCCD + 4
	saoKeWantWidth(t, f, sheet, "D", 15) // 11-rune project name + 4
	saoKeWantWidth(t, f, sheet, "E", 15) // 11-rune header + 4
	saoKeWantWidth(t, f, sheet, "F", 14) // 10-rune payment date + 4

	panes, err := f.GetPanes(sheet)
	if err != nil || !panes.Freeze || panes.YSplit != 6 {
		t.Fatalf("project panes = %+v (err %v), want frozen above row 7", panes, err)
	}
	view, err := f.GetSheetView(sheet, -1)
	if err != nil || view.ShowGridLines == nil || *view.ShowGridLines {
		t.Fatalf("project gridlines = %+v (err %v), want hidden", view.ShowGridLines, err)
	}
	props, err := f.GetSheetProps(sheet)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "2E75B6" {
		t.Fatalf("project tab color = %+v (err %v), want 2E75B6", props.TabColorRGB, err)
	}
}
