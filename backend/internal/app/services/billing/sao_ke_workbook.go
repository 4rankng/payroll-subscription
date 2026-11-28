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

// saoKeSummarySheet is the statement's first sheet. Project sheets are named
// after their project (via safeSheetName), so a project with the same safe
// name makes NewSheet fail and the export errors — accepted behaviour.
const saoKeSummarySheet = "Chi tiết giao dịch"

// Visual identity of the statement workbook, matching the approved business
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

// saoKeStyles bundles the workbook's cell styles, created once per workbook.
type saoKeStyles struct {
	title      int // sheet titles: bold 14 navy
	header     int // table header rows: bold white on navy, bordered
	data       int // left-aligned bordered cells
	center     int // centered bordered cells (STT, dates)
	text       int // CCCD: text format keeps leading zeros
	money      int // #,##0 bordered cells
	totalMoney int // statement totals: bold ₫ on light-blue fill
	totalLabel int // total-row labels: bold on light-blue fill
	vndBold    int // heading-block total (project B4): bold ₫, no border
	plain      int // unbordered info lines
	plainBold  int // unbordered bold labels (Tổng tiền:, INTERNAL header)
}

// renderSaoKeWorkbook renders the sao kê workbook (PRD §6.2):
//
//   - the summary sheet holds ONLY the CHI TIẾT GIAO DỊCH table
//     (STT | Kỳ lương tháng | Dự án | Số tiền) — no summary block, no
//     beneficiary block, no branding;
//   - one sheet per project (tên dự án, tổng nhân viên, kỳ lương tháng,
//     tổng tiền + per-employee STT|Họ tên|CCCD|Dự án|Số tiền|Ngày trả);
//   - an INTERNAL sheet listing the timesheet ids behind the statement.
//
// Cell values and positions are a stable contract consumed by reconciliation;
// the styling layer (fonts, fills, borders, widths, panes, filters, tab
// colors) never moves or rewrites a cell except the amount header label and
// the summary sheet name.
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
		if err := f.SetCellValue("INTERNAL", "A1", "Timesheet IDs"); err != nil {
			return nil, "", err
		}
		if err := f.SetCellStyle("INTERNAL", "A1", "A1", styles.plainBold); err != nil {
			return nil, "", err
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

// newSaoKeStyles creates every cell style used by the statement workbook.
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
	set(&out.totalMoney, &excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Family: saoKeFontName},
		Fill:         totalFill,
		Border:       thinBorders,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: new(saoKeMoneyVndFmt),
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

// renderSaoKeSummarySheet writes the transaction-detail table — and nothing
// else — on the summary sheet.
func renderSaoKeSummarySheet(f *excelize.File, sheet string, data *SaoKeData, styles saoKeStyles) error {
	if err := f.SetCellValue(sheet, "A1", "CHI TIẾT GIAO DỊCH"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "A1", "D1"); err != nil {
		return fmt.Errorf("sao ke: merge summary title: %w", err)
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", styles.title); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 1, 28); err != nil {
		return fmt.Errorf("sao ke: summary title height: %w", err)
	}
	if err := f.SetRowHeight(sheet, 3, 24); err != nil {
		return fmt.Errorf("sao ke: summary header height: %w", err)
	}
	headers := []string{"STT", "Kỳ lương tháng", "Dự án", "Số tiền (₫)"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 3)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}
	if err := f.SetCellStyle(sheet, "A3", "D3", styles.header); err != nil {
		return err
	}
	kyLabel := data.Cycle.KyLabel()
	dataStyles := []int{styles.center, styles.data, styles.data, styles.money}
	widths := newSaoKeWidths(4)
	for col, header := range headers {
		widths.add(col+1, header, false)
	}
	for i, row := range data.Rows {
		rowNum := i + 4
		sttCell, _ := excelize.CoordinatesToCellName(1, rowNum)
		kyCell, _ := excelize.CoordinatesToCellName(2, rowNum)
		projectCell, _ := excelize.CoordinatesToCellName(3, rowNum)
		amountCell, _ := excelize.CoordinatesToCellName(4, rowNum)
		for _, v := range []struct {
			cell  string
			value any
		}{{sttCell, i + 1}, {kyCell, kyLabel}, {projectCell, row.ProjectName}, {amountCell, row.Amount}} {
			if err := f.SetCellValue(sheet, v.cell, v.value); err != nil {
				return err
			}
		}
		// styles differ per column, so set them individually
		for col := 1; col <= 4; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowNum)
			if err := f.SetCellStyle(sheet, cell, cell, dataStyles[col-1]); err != nil {
				return err
			}
		}
		widths.add(1, i+1, false)
		widths.add(2, kyLabel, false)
		widths.add(3, row.ProjectName, false)
		widths.add(4, row.Amount, false)
	}
	totalRow := len(data.Rows) + 4
	totalLabelCell, _ := excelize.CoordinatesToCellName(3, totalRow)
	totalCell, _ := excelize.CoordinatesToCellName(4, totalRow)
	if err := f.SetCellValue(sheet, totalLabelCell, "TỔNG CỘNG"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, totalCell, data.Total); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A"+strconv.Itoa(totalRow), "B"+strconv.Itoa(totalRow), styles.totalLabel); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, totalLabelCell, totalLabelCell, styles.totalLabel); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, totalCell, totalCell, styles.totalMoney); err != nil {
		return err
	}
	widths.add(3, "TỔNG CỘNG", false)
	widths.add(4, data.Total, true)
	filterEnd, _ := excelize.CoordinatesToCellName(4, len(data.Rows)+3)
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

// renderSaoKeProjectSheet writes one project's statement sheet: the project
// heading block (tên dự án, tổng nhân viên, kỳ lương tháng, tổng tiền)
// followed by the per-employee payment table.
func renderSaoKeProjectSheet(f *excelize.File, sheet string, cycle SaoKeCycle, group SaoKeProjectGroup, styles saoKeStyles) error {
	if err := f.SetCellValue(sheet, "A1", group.ProjectName); err != nil {
		return err
	}
	if err := f.MergeCell(sheet, "A1", "F1"); err != nil {
		return fmt.Errorf("sao ke: merge project title: %w", err)
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", styles.title); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 1, 28); err != nil {
		return fmt.Errorf("sao ke: project title height: %w", err)
	}
	if err := f.SetCellValue(sheet, "A2", fmt.Sprintf("Tổng nhân viên: %d", group.EmployeeCount)); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "A3", fmt.Sprintf("Kỳ lương tháng: %s", cycle.KyLabel())); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "A4", "Tổng tiền:"); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "B4", group.Total); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A2", "A3", styles.plain); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A4", "A4", styles.plainBold); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "B4", "B4", styles.vndBold); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 6, 24); err != nil {
		return fmt.Errorf("sao ke: project header height: %w", err)
	}
	headers := []string{"STT", "Họ tên", "CCCD", "Dự án", "Số tiền (₫)", "Ngày trả"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 6)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}
	if err := f.SetCellStyle(sheet, "A6", "F6", styles.header); err != nil {
		return err
	}
	dataStyles := []int{styles.center, styles.data, styles.text, styles.data, styles.money, styles.center}
	widths := newSaoKeWidths(6)
	for col, header := range headers {
		widths.add(col+1, header, false)
	}
	widths.add(2, group.Total, true)
	for i, row := range group.Rows {
		rowNum := i + 7
		cells := []struct {
			col   int
			value any
		}{{1, i + 1}, {2, row.EmployeeName}, {3, row.CCCD}, {4, row.ProjectName}, {5, row.Amount}, {6, row.PaymentDate}}
		for _, v := range cells {
			cell, _ := excelize.CoordinatesToCellName(v.col, rowNum)
			if err := f.SetCellValue(sheet, cell, v.value); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, cell, cell, dataStyles[v.col-1]); err != nil {
				return err
			}
			widths.add(v.col, v.value, false)
		}
	}
	filterEnd, _ := excelize.CoordinatesToCellName(6, len(group.Rows)+6)
	if err := f.AutoFilter(sheet, "A6:"+filterEnd, []excelize.AutoFilterOptions{}); err != nil {
		return fmt.Errorf("sao ke: project filter: %w", err)
	}
	if err := saoKeFreeze(f, sheet, 7); err != nil {
		return err
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
