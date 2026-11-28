package billing

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// saoKeContentType is the MIME type of the statement workbook, used both for
// the download response and the email attachment.
const saoKeContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// saoKeSummarySheet is the statement's first sheet, per the approved template.
// Project sheets are named after their project (via safeSheetName), so a
// project with the same safe name makes NewSheet fail and the export errors —
// accepted behaviour.
const saoKeSummarySheet = "Summary"

// Visual identity of the statement workbooks, matching the approved business
// specimen: navy headers with white bold text, thin light-blue borders, and
// thousand-grouped VND amounts.
const (
	saoKeFontName    = "Arial"
	saoKeNavy        = "1F3864"
	saoKeBlue        = "2E75B6"
	saoKeGray        = "808080"
	saoKeBorderColor = "B7C3D0"
	saoKeTotalFill   = "D9E2F3"
	saoKeMoneyFmt    = "#,##0"
	saoKeMoneyVndFmt = `#,##0" ₫"`
	saoKeVndSuffix   = " ₫"
	saoKeTextFmt     = "@"
	saoKeWidthPad    = 4
	saoKeWidthMin    = 8.0
	saoKeWidthMax    = 36.0
)

// saoKeStyles bundles the workbooks' cell styles, created once per workbook.
type saoKeStyles struct {
	title      int // sheet titles: bold 14 navy
	header     int // table header rows: bold white on navy, bordered
	data       int // left-aligned bordered cells
	center     int // centered bordered cells (STT, dates)
	text       int // CCCD: text format keeps leading zeros
	money      int // #,##0 bordered cells
	moneyVnd   int // ₫ bordered cells (Số tiền columns)
	totalMoney int // totals row: bold #,##0 on light-blue fill
	totalLabel int // totals-row labels and counts: bold on light-blue fill
	vndBold    int // heading-block total (Tổng tiền ứng): bold ₫, no border
	section    int // section titles (CHI TIẾT ...): bold navy, no border
	plain      int // unbordered info values
	plainBold  int // unbordered bold labels (Tên dự án, INTERNAL header)
}

// renderSaoKeWorkbook renders the sao kê workbook per the approved template
// (PRD §6.2):
//
//   - the "Summary" sheet holds one row per project (STT | from | to |
//     Dự án | Số tiền) with the Kỳ payment-window dates as separate cells;
//   - one sheet per project: a heading block (tên dự án, tổng nhân viên,
//     kỳ lương tháng, tổng tiền ứng), the CHI TIẾT GIAO DỊCH payment table
//     (one row per employee per payment date) and the CHI TIẾT CHẤM CÔNG
//     per-timesheet detail section;
//   - an INTERNAL sheet listing the timesheet ids behind the statement.
//
// Cell positions and totals are a stable contract consumed by reconciliation;
// the styling layer (fonts, fills, borders, widths, panes, filters, tab
// colors) never moves or rewrites a cell.
//
// Returns the xlsx bytes and the download filename sao_ke_tt_<DD-MM-YYYY>.xlsx.
func renderSaoKeWorkbook(data *SaoKeData) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	styles, err := newSaoKeStyles(f)
	if err != nil {
		return nil, "", err
	}

	summary := f.GetSheetName(0)
	if err := f.SetSheetName(summary, saoKeSummarySheet); err != nil {
		return nil, "", fmt.Errorf("sao ke: rename summary sheet: %w", err)
	}
	if err := renderSaoKeSummarySheet(f, saoKeSummarySheet, data, styles); err != nil {
		return nil, "", err
	}
	for _, group := range data.Groups {
		if _, err := f.NewSheet(safeSheetName(group.ProjectName)); err != nil {
			return nil, "", fmt.Errorf("sao ke: project sheet %q: %w", group.ProjectName, err)
		}
		if err := renderSaoKeProjectSheet(f, safeSheetName(group.ProjectName), data.Cycle, group, styles); err != nil {
			return nil, "", err
		}
	}
	if len(data.TimesheetIDs) > 0 {
		if _, err := f.NewSheet("INTERNAL"); err != nil {
			return nil, "", fmt.Errorf("sao ke: internal sheet: %w", err)
		}
		if err := f.SetCellValue("INTERNAL", "A1", "type:timesheet"); err != nil {
			return nil, "", err
		}
		if err := f.SetCellStyle("INTERNAL", "A1", "A1", styles.plainBold); err != nil {
			return nil, "", err
		}
		for i, id := range data.TimesheetIDs {
			cell, _ := excelize.CoordinatesToCellName(1, i+2)
			if err := f.SetCellValue("INTERNAL", cell, id); err != nil {
				return nil, "", err
			}
		}
		if err := f.SetColWidth("INTERNAL", "A", "A", 14); err != nil {
			return nil, "", fmt.Errorf("sao ke: internal col width: %w", err)
		}
		if err := saoKeTabColor(f, "INTERNAL", saoKeGray); err != nil {
			return nil, "", err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("sao ke: write workbook: %w", err)
	}
	return buf.Bytes(), saoKeFilename(data.ReportDate), nil
}

// newSaoKeStyles creates every cell style used by the workbooks.
func newSaoKeStyles(f *excelize.File) (saoKeStyles, error) {
	var out saoKeStyles
	thinBorders := []excelize.Border{
		{Type: "left", Color: saoKeBorderColor, Style: 1},
		{Type: "right", Color: saoKeBorderColor, Style: 1},
		{Type: "top", Color: saoKeBorderColor, Style: 1},
		{Type: "bottom", Color: saoKeBorderColor, Style: 1},
	}
	totalFill := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{saoKeTotalFill}}
	headerFill := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{saoKeNavy}}
	var err error
	set := func(id *int, style *excelize.Style, name string) {
		if err != nil {
			return
		}
		if *id, err = f.NewStyle(style); err != nil {
			err = fmt.Errorf("sao ke: %s style: %w", name, err)
		}
	}
	set(&out.title, &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: saoKeNavy, Family: saoKeFontName},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	}, "title")
	set(&out.header, &excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: saoKeFontName},
		Fill:      headerFill,
		Border:    thinBorders,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	}, "header")
	set(&out.data, &excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: saoKeFontName},
		Border:    thinBorders,
		Alignment: &excelize.Alignment{Vertical: "center"},
	}, "data")
	set(&out.center, &excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: saoKeFontName},
		Border:    thinBorders,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	}, "center")
	set(&out.text, &excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: saoKeFontName},
		Border:       thinBorders,
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		CustomNumFmt: new(saoKeTextFmt),
	}, "text")
	set(&out.money, &excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: saoKeFontName},
		Border:       thinBorders,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: new(saoKeMoneyFmt),
	}, "money")
	set(&out.moneyVnd, &excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: saoKeFontName},
		Border:       thinBorders,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: new(saoKeMoneyVndFmt),
	}, "moneyVnd")
	set(&out.totalMoney, &excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Family: saoKeFontName},
		Fill:         totalFill,
		Border:       thinBorders,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: new(saoKeMoneyFmt),
	}, "totalMoney")
	set(&out.totalLabel, &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: saoKeFontName},
		Fill:      totalFill,
		Border:    thinBorders,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	}, "totalLabel")
	set(&out.vndBold, &excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Family: saoKeFontName},
		Alignment:    &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		CustomNumFmt: new(saoKeMoneyVndFmt),
	}, "vndBold")
	set(&out.section, &excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: saoKeNavy, Family: saoKeFontName},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	}, "section")
	set(&out.plain, &excelize.Style{Font: &excelize.Font{Size: 11, Family: saoKeFontName}}, "plain")
	set(&out.plainBold, &excelize.Style{Font: &excelize.Font{Bold: true, Size: 11, Family: saoKeFontName}}, "plainBold")
	return out, err
}

// saoKeDisplayLen measures the runes a cell value occupies when displayed
// under its number format: strings verbatim, numbers thousand-grouped, totals
// with the trailing ₫.
func saoKeDisplayLen(v any, vnd bool) int {
	switch t := v.(type) {
	case string:
		return utf8.RuneCountInString(t)
	case int:
		return len(strconv.Itoa(t))
	case int64:
		n := len(saoKeThousandGroup(t))
		if vnd {
			n += utf8.RuneCountInString(saoKeVndSuffix)
		}
		return n
	case float64:
		return len(strconv.FormatFloat(t, 'f', -1, 64))
	default:
		return 0
	}
}

// saoKeThousandGroup renders v with comma group separators (1234567 →
// "1,234,567"), matching the #,##0 display format.
func saoKeThousandGroup(v int64) string {
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i := range len(s) {
		b.WriteByte(s[i])
		if rem := len(s) - i - 1; rem > 0 && rem%3 == 0 {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// saoKeWidths accumulates the longest display length per column and applies
// them as column widths (longest + padding, clamped).
type saoKeWidths struct {
	max []int
}

func newSaoKeWidths(cols int) *saoKeWidths {
	return &saoKeWidths{max: make([]int, cols)}
}

func (w *saoKeWidths) add(col int, v any, vnd bool) {
	if col > len(w.max) || col < 1 {
		return
	}
	if n := saoKeDisplayLen(v, vnd); n > w.max[col-1] {
		w.max[col-1] = n
	}
}

func (w *saoKeWidths) apply(f *excelize.File, sheet string) error {
	for i, n := range w.max {
		width := float64(n + saoKeWidthPad)
		if width < saoKeWidthMin {
			width = saoKeWidthMin
		}
		if width > saoKeWidthMax {
			width = saoKeWidthMax
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheet, col, col, width); err != nil {
			return fmt.Errorf("sao ke: %s col %s width: %w", sheet, col, err)
		}
	}
	return nil
}

// saoKeFreeze freezes everything above aboveRow on sheet.
func saoKeFreeze(f *excelize.File, sheet string, aboveRow int) error {
	topLeft, _ := excelize.CoordinatesToCellName(1, aboveRow)
	panes := &excelize.Panes{
		Freeze:      true,
		YSplit:      aboveRow - 1,
		TopLeftCell: topLeft,
		ActivePane:  "bottomRight",
	}
	if err := f.SetPanes(sheet, panes); err != nil {
		return fmt.Errorf("sao ke: %s freeze panes: %w", sheet, err)
	}
	return nil
}

// saoKeHideGridlines turns off the sheet's gridline display; tables carry
// their own borders.
func saoKeHideGridlines(f *excelize.File, sheet string) error {
	if err := f.SetSheetView(sheet, -1, &excelize.ViewOptions{ShowGridLines: new(false)}); err != nil {
		return fmt.Errorf("sao ke: %s hide gridlines: %w", sheet, err)
	}
	return nil
}

// saoKeTabColor sets the sheet's tab color from an RGB hex string.
func saoKeTabColor(f *excelize.File, sheet, rgb string) error {
	if err := f.SetSheetProps(sheet, &excelize.SheetPropsOptions{TabColorRGB: new(rgb)}); err != nil {
		return fmt.Errorf("sao ke: %s tab color: %w", sheet, err)
	}
	return nil
}

// saoKyWindowLabel renders the Kỳ payment window the way the template does:
// "21/10 -> 20/11/26".
func saoKyWindowLabel(cycle SaoKeCycle) string {
	return cycle.From.Format("02/01") + " -> " + cycle.To.Format("02/01/06")
}

// renderSaoKeSummarySheet writes the per-project transaction overview — and
// nothing else — on the summary sheet.
func renderSaoKeSummarySheet(f *excelize.File, sheet string, data *SaoKeData, styles saoKeStyles) error {
	if err := f.SetCellValue(sheet, "A1", "T\u1ed4NG QUAN GIAO D\u1ecaCH"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "A1", "E1"); err != nil {
		return fmt.Errorf("sao ke: merge summary title: %w", err)
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", styles.title); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 1, 28); err != nil {
		return fmt.Errorf("sao ke: summary title height: %w", err)
	}
	if err := f.SetCellValue(sheet, "A3", "STT"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "B3", "K\u1ef3 l\u01b0\u01a1ng th\u00e1ng"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "B3", "C3"); err != nil {
		return fmt.Errorf("sao ke: merge ky header: %w", err)
	}
	if err := f.SetCellValue(sheet, "D3", "D\u1ef1 \u00e1n"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "E3", "S\u1ed1 ti\u1ec1n"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A3", "E3", styles.header); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 3, 24); err != nil {
		return fmt.Errorf("sao ke: summary header height: %w", err)
	}

	widths := newSaoKeWidths(5)
	widths.add(1, "STT", false)
	widths.add(2, "K\u1ef3 l\u01b0\u01a1ng th\u00e1ng", false) // merged B3:C3 counts on B only
	widths.add(4, "D\u1ef1 \u00e1n", false)
	widths.add(5, "S\u1ed1 ti\u1ec1n", false)
	dataStyles := []int{styles.center, styles.center, styles.center, styles.data, styles.moneyVnd}
	for i, group := range data.Groups {
		rowNum := i + 4
		fromDate := data.Cycle.From.Format("02/01/2006")
		toDate := data.Cycle.To.Format("02/01/2006")
		cells := []struct {
			col   int
			value any
		}{{1, i + 1}, {2, fromDate}, {3, toDate}, {4, group.ProjectName}, {5, group.Total}}
		for _, v := range cells {
			cell, _ := excelize.CoordinatesToCellName(v.col, rowNum)
			if err := f.SetCellValue(sheet, cell, v.value); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, cell, cell, dataStyles[v.col-1]); err != nil {
				return err
			}
			widths.add(v.col, v.value, v.col == 5)
		}
	}
	lastRow := len(data.Groups) + 3
	filterEnd, _ := excelize.CoordinatesToCellName(5, lastRow)
	if err := f.AutoFilter(sheet, "A3:"+filterEnd, []excelize.AutoFilterOptions{}); err != nil {
		return fmt.Errorf("sao ke: summary filter: %w", err)
	}
	if err := saoKeFreeze(f, sheet, 4); err != nil {
		return err
	}
	if err := saoKeHideGridlines(f, sheet); err != nil {
		return err
	}
	if err := saoKeTabColor(f, sheet, saoKeNavy); err != nil {
		return err
	}
	return widths.apply(f, sheet)
}

// renderSaoKeProjectSheet writes one project's statement sheet per the
// template: the heading block, the aggregated payment table and the
// per-timesheet detail section.
func renderSaoKeProjectSheet(f *excelize.File, sheet string, cycle SaoKeCycle, group SaoKeProjectGroup, styles saoKeStyles) error {
	if err := f.SetCellValue(sheet, "A1", group.ProjectName); err != nil {
		return err
	}
	for _, h := range heading {
		labelCell, _ := excelize.CoordinatesToCellName(1, h.row)
		valueCell, _ := excelize.CoordinatesToCellName(3, h.row)
		labelEnd, _ := excelize.CoordinatesToCellName(2, h.row)
		if err := f.SetCellValue(sheet, labelCell, h.label); err != nil {
			return err
		}
		if err := f.MergeCell(sheet, labelCell, labelEnd); err != nil {
			return fmt.Errorf("sao ke: merge heading row %d: %w", h.row, err)
		}
		if err := f.SetCellStyle(sheet, labelCell, labelCell, styles.plainBold); err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, valueCell, h.value); err != nil {
			return err
		}
		valueStyle := styles.plain
		if h.vnd {
			valueStyle = styles.vndBold
		}
		if err := f.SetCellStyle(sheet, valueCell, valueCell, valueStyle); err != nil {
			return err
		}
		widths.add(1, h.label, false)
		widths.add(3, h.value, h.vnd)
	}

	if err := f.SetCellValue(sheet, "A7", "CHI TI\u1ebeT GIAO D\u1ecaCH"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "A7", "F7"); err != nil {
		return fmt.Errorf("sao ke: merge payment section title: %w", err)
	}
	if err := f.SetCellStyle(sheet, "A7", "A7", styles.section); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 7, 22); err != nil {
		return fmt.Errorf("sao ke: payment section height: %w", err)
	}

	paymentHeaders := []string{"STT", "H\u1ecd t\u00ean", "CCCD", "D\u1ef1 \u00e1n", "S\u1ed1 ti\u1ec1n", "Ng\u00e0y tr\u1ea3"}
	for col, header := range paymentHeaders {
		cell, _ := excelize.CoordinatesToCellName(col+1, 8)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
		widths.add(col+1, header, false)
	}
	if err := f.SetCellStyle(sheet, "A8", "F8", styles.header); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 8, 24); err != nil {
		return fmt.Errorf("sao ke: payment header height: %w", err)
	}

	paymentStyles := []int{styles.center, styles.data, styles.text, styles.data, styles.moneyVnd, styles.center}
	for i, row := range group.Payments {
		rowNum := i + 9
		cells := []struct {
			col   int
			value any
		}{{1, i + 1}, {2, row.EmployeeName}, {3, row.CCCD}, {4, row.ProjectName}, {5, row.Amount}, {6, row.PaymentDate}}
		for _, v := range cells {
			cell, _ := excelize.CoordinatesToCellName(v.col, rowNum)
			if err := f.SetCellValue(sheet, cell, v.value); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, cell, cell, paymentStyles[v.col-1]); err != nil {
				return err
			}
			widths.add(v.col, v.value, v.col == 5)
		}
	}
	lastPaymentRow := len(group.Payments) + 8
	filterEnd, _ := excelize.CoordinatesToCellName(6, lastPaymentRow)
	if err := f.AutoFilter(sheet, "A8:"+filterEnd, []excelize.AutoFilterOptions{}); err != nil {
		return fmt.Errorf("sao ke: project filter: %w", err)
	}
	if err := saoKeFreeze(f, sheet, 9); err != nil {
		return err
	}

	if len(group.TimesheetRows) > 0 {
		titleRow := lastPaymentRow + 3
		titleStart, _ := excelize.CoordinatesToCellName(1, titleRow)
		titleEnd, _ := excelize.CoordinatesToCellName(8, titleRow)
		if err := f.SetCellValue(sheet, titleStart, "CHI TI\u1ebeT CH\u1ea4M C\u00d4NG"); err != nil {
			return err
		}
		if err := f.MergeCell(sheet, titleStart, titleEnd); err != nil {
			return fmt.Errorf("sao ke: merge timesheet section title: %w", err)
		}
		if err := f.SetCellStyle(sheet, titleStart, titleStart, styles.section); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheet, titleRow, 22); err != nil {
			return fmt.Errorf("sao ke: timesheet section height: %w", err)
		}

		timesheetHeaders := []string{"STT", "H\u1ecd t\u00ean", "Ng\u00e0y c\u00f4ng", "V\u1ecb tr\u00ed", "Ng\u00e0y l\u00e0m", "Ca l\u00e0m", "Gi\u1edd", "S\u1ed1 ti\u1ec1n"}
		headerRow := titleRow + 1
		for col, header := range timesheetHeaders {
			cell, _ := excelize.CoordinatesToCellName(col+1, headerRow)
			if err := f.SetCellValue(sheet, cell, header); err != nil {
				return err
			}
			widths.add(col+1, header, false)
		}
		headerStart, _ := excelize.CoordinatesToCellName(1, headerRow)
		headerEnd, _ := excelize.CoordinatesToCellName(8, headerRow)
		if err := f.SetCellStyle(sheet, headerStart, headerEnd, styles.header); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheet, headerRow, 24); err != nil {
			return fmt.Errorf("sao ke: timesheet header height: %w", err)
		}

		timesheetStyles := []int{styles.center, styles.data, styles.center, styles.center, styles.center, styles.center, styles.center, styles.moneyVnd}
		for i, row := range group.TimesheetRows {
			rowNum := headerRow + 1 + i
			cells := []struct {
				col   int
				value any
			}{{1, i + 1}, {2, row.EmployeeName}, {3, row.WorkDate}, {4, row.Position}, {5, row.DayType}, {6, row.Shift}, {7, row.Hours}, {8, row.Amount}}
			for _, v := range cells {
				cell, _ := excelize.CoordinatesToCellName(v.col, rowNum)
				if err := f.SetCellValue(sheet, cell, v.value); err != nil {
					return err
				}
				if err := f.SetCellStyle(sheet, cell, cell, timesheetStyles[v.col-1]); err != nil {
					return err
				}
				widths.add(v.col, v.value, v.col == 8)
			}
		}
	}

	if err := saoKeHideGridlines(f, sheet); err != nil {
		return err
	}
	if err := saoKeTabColor(f, sheet, saoKeBlue); err != nil {
		return err
	}
	return widths.apply(f, sheet)
}

// saoKeFilename is the download/attachment filename for a report date:
// sao_ke_tt_<DD-MM-YYYY>.xlsx.
func saoKeFilename(reportDate interface{ Format(string) string }) string {
	return "sao_ke_tt_" + reportDate.Format("02-01-2006") + ".xlsx"
}

// renderSaoKeEmailBody is the statement email body: one line pointing at the
// attachment followed by the CHI TIẾT GIAO DỊCH table.
func renderSaoKeEmailBody(data *SaoKeData) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"<p>File Excel sao kê thanh toán lương ngày %s được đính kèm theo email này để Quý Công ty tiện theo dõi và đối chiếu.</p>",
		data.ReportDate.Format("02/01/2006")))
	b.WriteString(`<table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;">`)
	b.WriteString("<tr><th>STT</th><th>Kỳ lương tháng</th><th>Dự án</th><th>Số tiền</th></tr>")
	kyLabel := data.Cycle.KyLabel()
	// PRD §6.3: the body carries ONLY the CHI TIẾT GIAO DỊCH table — one
	// aggregated row per project (the Summary-sheet table), never the
	// per-timesheet detail. Detail rows live in the attached workbook, and
	// the aggregate keeps the body far under the 100KB email cap.
	projects := make([]SaoKeProjectGroup, 0, len(data.Groups))
	projects = append(projects, data.Groups...)
	sort.Slice(projects, func(i, j int) bool { return projects[i].ProjectID < projects[j].ProjectID })
	for i, g := range projects {
		fmt.Fprintf(&b,
			"<tr><td>%d</td><td>%s</td><td>%s</td><td style=\"text-align:right;\">%d</td></tr>",
			i+1, kyLabel, escapeHTMLText(g.ProjectName), g.Total)
	}
	fmt.Fprintf(&b, "<tr><td colspan=\"3\"><strong>TỔNG CỘNG</strong></td><td style=\"text-align:right;\"><strong>%d</strong></td></tr>", data.Total)
	b.WriteString("</table>")
	return b.String()
}

// escapeHTMLText escapes the five HTML-sensitive characters so project names
// from user data cannot inject markup into the statement email body.
func escapeHTMLText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
