package billing

import (
	"bytes"
	"testing"

	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

func TestBillingWorkbookSummaryAndProjects(t *testing.T) {
	lines := []domain.BillingReportLine{
		{Cycle: 1, DistinctCount: 3, AmountVnd: 75000},
		{Cycle: 2, DistinctCount: 2, AmountVnd: 50000},
	}
	sheets := []projectSheet{{
		ProjectID:   7,
		ProjectName: "Dự án Alpha",
		Rows: []PaymentRow{{
			EmployeeName: "Nguyễn Văn An", CCCD: "031301008158", Cycle: 1,
			Date: "20/10/2026", Amount: 25000, PaymentDate: "01/11/2026", PaymentReference: "FT2137",
		}},
	}}
	raw, err := renderBillingWorkbook("2026-10", 25000, lines, 5, 125000, sheets)
	if err != nil {
		t.Fatalf("renderBillingWorkbook: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	summary := f.GetSheetName(0)
	if got, _ := f.GetCellValue(summary, "A1"); got != "Báo cáo phí subscription — tháng 2026-10" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellValue(summary, "B3"); got != "Số nhân viên (distinct CCCD)" {
		t.Fatalf("B3 = %q", got)
	}
	// Kỳ rows 4–5, TỔNG row 6
	if got, _ := f.GetCellValue(summary, "D4"); got != "75,000" {
		t.Fatalf("D4 = %q, want #,##0 applied", got)
	}
	if got, _ := f.GetCellValue(summary, "A6"); got != "TỔNG" {
		t.Fatalf("A6 = %q", got)
	}
	if got, _ := f.GetCellValue(summary, "D6"); got != "125,000" {
		t.Fatalf("D6 = %q, want totals unchanged", got)
	}

	panes, err := f.GetPanes(summary)
	if err != nil || !panes.Freeze || panes.YSplit != 3 {
		t.Fatalf("summary panes = %+v (err %v), want frozen above row 4", panes, err)
	}
	view, err := f.GetSheetView(summary, -1)
	if err != nil || view.ShowGridLines == nil || *view.ShowGridLines {
		t.Fatalf("summary gridlines = %+v (err %v), want hidden", view.ShowGridLines, err)
	}
	props, err := f.GetSheetProps(summary)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "1F3864" {
		t.Fatalf("summary tab color = %+v (err %v), want 1F3864", props.TabColorRGB, err)
	}

	sheet := f.GetSheetName(1)
	if got, _ := f.GetCellValue(sheet, "A1"); got != "STT" {
		t.Fatalf("project A1 = %q", got)
	}
	if got, _ := f.GetCellValue(sheet, "F2"); got != "25,000" {
		t.Fatalf("project F2 = %q, want #,##0 applied", got)
	}
	if got, _ := f.GetCellValue(sheet, "C2"); got != "031301008158" {
		t.Fatalf("project C2 = %q, want leading zeros preserved", got)
	}
	panes, err = f.GetPanes(sheet)
	if err != nil || !panes.Freeze || panes.YSplit != 1 {
		t.Fatalf("project panes = %+v (err %v), want frozen above row 2", panes, err)
	}
	view, err = f.GetSheetView(sheet, -1)
	if err != nil || view.ShowGridLines == nil || *view.ShowGridLines {
		t.Fatalf("project gridlines = %+v (err %v), want hidden", view.ShowGridLines, err)
	}
	props, err = f.GetSheetProps(sheet)
	if err != nil || props.TabColorRGB == nil || *props.TabColorRGB != "2E75B6" {
		t.Fatalf("project tab color = %+v (err %v), want 2E75B6", props.TabColorRGB, err)
	}
}
