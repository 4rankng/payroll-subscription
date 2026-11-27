package billing

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// saoKeContentType is the MIME type of the statement workbook, used both for
// the download response and the email attachment.
const saoKeContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// renderSaoKeWorkbook renders the sao kê workbook (PRD §6.2):
//
//   - the summary sheet holds ONLY the CHI TIẾT GIAO DỊCH table
//     (STT | Kỳ lương tháng | Dự án | Số tiền) — no summary block, no
//     beneficiary block, no branding;
//   - one sheet per project (tên dự án, tổng nhân viên, kỳ lương tháng,
//     tổng tiền + per-employee STT|Họ tên|CCCD|Dự án|Số tiền|Ngày trả);
//   - an INTERNAL sheet listing the timesheet ids behind the statement.
//
// Returns the xlsx bytes and the download filename sao_ke_tt_<DD-MM-YYYY>.xlsx.
func renderSaoKeWorkbook(data *SaoKeData) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	boldStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, "", fmt.Errorf("sao ke: header style: %w", err)
	}

	summary := f.GetSheetName(0)
	if err := renderSaoKeSummarySheet(f, summary, data, boldStyle); err != nil {
		return nil, "", err
	}
	for _, group := range data.Groups {
		if _, err := f.NewSheet(safeSheetName(group.ProjectName)); err != nil {
			return nil, "", fmt.Errorf("sao ke: project sheet %q: %w", group.ProjectName, err)
		}
		if err := renderSaoKeProjectSheet(f, safeSheetName(group.ProjectName), data.Cycle, group, boldStyle); err != nil {
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
		if err := f.SetCellStyle("INTERNAL", "A1", "A1", boldStyle); err != nil {
			return nil, "", err
		}
		for i, id := range data.TimesheetIDs {
			cell, _ := excelize.CoordinatesToCellName(1, i+2)
			if err := f.SetCellValue("INTERNAL", cell, id); err != nil {
				return nil, "", err
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("sao ke: write workbook: %w", err)
	}
	return buf.Bytes(), saoKeFilename(data.ReportDate), nil
}

// renderSaoKeSummarySheet writes the transaction-detail table — and nothing
// else — on the summary sheet.
func renderSaoKeSummarySheet(f *excelize.File, sheet string, data *SaoKeData, boldStyle int) error {
	if err := f.SetCellValue(sheet, "A1", "CHI TIẾT GIAO DỊCH"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", boldStyle); err != nil {
		return err
	}
	headers := []string{"STT", "Kỳ lương tháng", "Dự án", "Số tiền"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 3)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, cell, cell, boldStyle); err != nil {
			return err
		}
	}
	kyLabel := data.Cycle.KyLabel()
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
	return f.SetCellStyle(sheet, totalLabelCell, totalCell, boldStyle)
}

// renderSaoKeProjectSheet writes one project's statement sheet: the project
// heading block (tên dự án, tổng nhân viên, kỳ lương tháng, tổng tiền)
// followed by the per-employee payment table.
func renderSaoKeProjectSheet(f *excelize.File, sheet string, cycle SaoKeCycle, group SaoKeProjectGroup, boldStyle int) error {
	if err := f.SetCellValue(sheet, "A1", group.ProjectName); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", "A1", boldStyle); err != nil {
		return err
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
	if err := f.SetCellStyle(sheet, "A4", "A4", boldStyle); err != nil {
		return err
	}

	headers := []string{"STT", "Họ tên", "CCCD", "Dự án", "Số tiền", "Ngày trả"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 6)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, cell, cell, boldStyle); err != nil {
			return err
		}
	}
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
		}
	}
	return nil
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
