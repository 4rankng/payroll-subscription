package billing

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// saoKeTestData builds one project with four paid timesheets covering the
// template's edge cases: two timesheets of one employee collapsing into a
// single payment row, a second payment date for the same employee, an
// unknown payment date, a weekend work date and a PaidAmount==0 fallback.
func saoKeTestData() *SaoKeData {
	rows := []SaoKeRow{
		{TimesheetID: 11, EmployeeID: 1, EmployeeName: "Nguyễn Văn An", CCCD: "031301008158", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 300000, PaymentDate: "24/10/2026"},
		{TimesheetID: 12, EmployeeID: 1, EmployeeName: "Nguyễn Văn An", CCCD: "031301008158", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 225000, PaymentDate: "24/10/2026"},
		{TimesheetID: 13, EmployeeID: 1, EmployeeName: "Nguyễn Văn An", CCCD: "031301008158", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 225000, PaymentDate: "31/10/2026"},
		{TimesheetID: 14, EmployeeID: 2, EmployeeName: "Trần Thị Bình", CCCD: "001171012172", ProjectID: 7, ProjectName: "Dự án Alpha", Amount: 300000},
	}
	cycle := SaoKeCycle{
		Ky:        4,
		WorkMonth: "2026-10",
		PayDate:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		From:      time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC),
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

func saoKeWantGridlinesHidden(t *testing.T, f *excelize.File, sheet string) {
	t.Helper()
	view, err := f.GetSheetView(sheet, -1)
	if err != nil || view.ShowGridLines == nil || *view.ShowGridLines {
		t.Fatalf("%s gridlines = %+v (err %v), want hidden", sheet, view.ShowGridLines, err)
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
	if got := f.GetSheetName(0); got != "Summary" {
		t.Fatalf("sheet 0 = %q", got)
	}
	if got := f.GetSheetName(1); got != "Dự án Alpha" {
		t.Fatalf("sheet 1 = %q", got)
	}
	if got := f.GetSheetName(2); got != "INTERNAL" {
		t.Fatalf("sheet 2 = %q", got)
	}

	// INTERNAL carries the type marker and the timesheet ids behind the rows.
	if got := saoKeCell(t, f, "INTERNAL", "A1", true); got != "type:timesheet" {
		t.Fatalf("INTERNAL A1 = %q", got)
	}
	if got := saoKeCell(t, f, "INTERNAL", "A2", true); got != "11" {
		t.Fatalf("INTERNAL A2 = %q", got)
	}
	if got := saoKeCell(t, f, "INTERNAL", "A5", true); got != "14" {
		t.Fatalf("INTERNAL A5 = %q", got)
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
	sheet := "Summary"

	if got := saoKeCell(t, f, sheet, "A1", true); got != "TỔNG QUAN GIAO DỊCH" {
		t.Fatalf("A1 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "B3", true); got != "Kỳ lương tháng" {
		t.Fatalf("B3 = %q", got)
	}
	// only the header merges B:C; the data rows carry separate date cells
	merged := map[string]bool{}
	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		t.Fatalf("merge cells: %v", err)
	}
	for _, m := range mergeCells {
		merged[m.GetStartAxis()+".."+m.GetEndAxis()] = true
	}
	if !merged["B3..C3"] {
		t.Fatalf("B3:C3 not merged: %v", merged)
	}
	if merged["B4..C4"] {
		t.Fatalf("data row B4:C4 must not be merged")
	}
	if got := saoKeCell(t, f, sheet, "B4", true); got != "21/10/2026" {
		t.Fatalf("B4 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "C4", true); got != "20/11/2026" {
		t.Fatalf("C4 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "D4", true); got != "Dự án Alpha" {
		t.Fatalf("D4 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "E4", false); got != "1,050,000 ₫" {
		t.Fatalf("E4 formatted = %q, want ₫ format applied", got)
	}
	// one row per project, no grand-total row
	if got := saoKeCell(t, f, sheet, "A5", true); got != "" {
		t.Fatalf("summary must end after the project rows, found A5 = %q", got)
	}

	saoKeWantWidth(t, f, sheet, "A", 8)
	saoKeWantWidth(t, f, sheet, "B", 18) // header "Kỳ lương tháng" (14) + 4
	saoKeWantWidth(t, f, sheet, "C", 14) // 10-rune date + 4
	saoKeWantWidth(t, f, sheet, "D", 15) // 11-rune project name + 4
	saoKeWantWidth(t, f, sheet, "E", 15) // 11-rune total display + 4

	panes, err := f.GetPanes(sheet)
	if err != nil || !panes.Freeze || panes.YSplit != 3 {
		t.Fatalf("summary panes = %+v (err %v), want frozen above row 4", panes, err)
	}
	saoKeWantGridlinesHidden(t, f, sheet)
	props, err := f.GetSheetProps(sheet)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "1F3864" {
		t.Fatalf("summary tab color = %+v (err %v), want 1F3864", props.TabColorRGB, err)
	}
}

func TestSaoKeWorkbookProjectSheet(t *testing.T) {
	f := openSaoKeWorkbook(t, saoKeTestData())
	sheet := "Dự án Alpha"

	// heading block: labels merged A:B, values in C
	if got := saoKeCell(t, f, sheet, "A1", true); got != "Tên dự án" {
		t.Fatalf("A1 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "C1", true); got != "Dự án Alpha" {
		t.Fatalf("C1 = %q", got)
	}
	// Tổng nhân viên = number of payment rows (3), not distinct employees (2)
	if got := saoKeCell(t, f, sheet, "C2", true); got != "3" {
		t.Fatalf("C2 = %q, want payment-row count", got)
	}
	if got := saoKeCell(t, f, sheet, "C3", true); got != "21/10 -> 20/11/26" {
		t.Fatalf("C3 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "C4", false); got != "1,050,000 ₫" {
		t.Fatalf("C4 = %q, want ₫ format applied", got)
	}
	if got := saoKeCell(t, f, sheet, "A7", true); got != "CHI TIẾT GIAO DỊCH" {
		t.Fatalf("A7 = %q", got)
	}

	// payment table rows 9–11: one row per (employee, payment date)
	if got := saoKeCell(t, f, sheet, "B9", true); got != "Nguyễn Văn An" {
		t.Fatalf("B9 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "E9", false); got != "525,000 ₫" {
		t.Fatalf("E9 = %q, want two timesheets summed", got)
	}
	if got := saoKeCell(t, f, sheet, "F9", true); got != "24/10/2026" {
		t.Fatalf("F9 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "B10", true); got != "Nguyễn Văn An" {
		t.Fatalf("B10 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "F10", true); got != "31/10/2026" {
		t.Fatalf("F10 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "B11", true); got != "Trần Thị Bình" {
		t.Fatalf("B11 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "F11", true); got != "" {
		t.Fatalf("F11 = %q, want empty Ngày trả for unknown payment date", got)
	}
	if got := saoKeCell(t, f, sheet, "E11", false); got != "300,000 ₫" {
		t.Fatalf("E11 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "C9", false); got != "031301008158" {
		t.Fatalf("C9 = %q, want leading zeros preserved as text", got)
	}

	// timesheet detail section: title at last payment row + 3
	if got := saoKeCell(t, f, sheet, "A14", true); got != "CHI TIẾT CHẤM CÔNG" {
		t.Fatalf("A14 = %q", got)
	}
	if got := saoKeCell(t, f, sheet, "C15", true); got != "Ngày công" {
		t.Fatalf("C15 = %q", got)
	}
	type tsRow struct {
		name, date, position, dayType, shift, hours, amount string
	}
	want := []tsRow{
		{"Nguyễn Văn An", "20/10/2026", "phổ thông", "ngày thường", "06:00-14:00", "8", "300,000 ₫"},
		{"Nguyễn Văn An", "25/10/2026", "phổ thông", "ngày nghỉ", "tăng ca", "4", "225,000 ₫"},
		{"Nguyễn Văn An", "26/10/2026", "phổ thông", "ngày thường", "tăng ca", "4", "225,000 ₫"},
		{"Trần Thị Bình", "21/10/2026", "phổ thông", "ngày thường", "06:00-14:00", "8", "300,000 ₫"},
	}
	for i, w := range want {
		r := 16 + i
		cols := map[string]string{"B": w.name, "C": w.date, "D": w.position, "E": w.dayType, "F": w.shift, "G": w.hours, "H": w.amount}
		for col, v := range cols {
			if got := saoKeCell(t, f, sheet, col+strconv.Itoa(r), false); got != v {
				t.Fatalf("timesheet row %d col %s = %q, want %q", r, col, got, v)
			}
		}
	}

	saoKeWantWidth(t, f, sheet, "A", 18) // "Tổng nhân viên" label (14) + 4
	saoKeWantWidth(t, f, sheet, "B", 17) // 13-rune employee name + 4
	saoKeWantWidth(t, f, sheet, "C", 21) // 17-rune window label "21/10 -> 20/11/26" + 4
	saoKeWantWidth(t, f, sheet, "E", 15) // 11-rune "ngày thường" + 4
	saoKeWantWidth(t, f, sheet, "F", 15) // 11-rune shift label + 4
	saoKeWantWidth(t, f, sheet, "H", 13) // 9-rune amount display + 4

	panes, err := f.GetPanes(sheet)
	if err != nil || !panes.Freeze || panes.YSplit != 8 {
		t.Fatalf("project panes = %+v (err %v), want frozen above row 9", panes, err)
	}
	saoKeWantGridlinesHidden(t, f, sheet)
	props, err := f.GetSheetProps(sheet)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "2E75B6" {
		t.Fatalf("project tab color = %+v (err %v), want 2E75B6", props.TabColorRGB, err)
	}
}
