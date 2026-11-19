package timesheet

import (
	"context"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportTimesheets exports timesheets for a date range to Excel format
func (h *Handler) ExportTimesheets(c *gin.Context) {
	var req dto.ExportTimesheetsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	if err := req.ParseProjectIDs(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid project IDs: %v", err))
		return
	}

	if err := req.ParseStatus(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid status: %v", err))
		return
	}

	if err := req.ValidateDateRange(); err != nil {
		response.BadRequest(c, fmt.Sprintf("Invalid date range: %v", err))
		return
	}

	fromDate, toDate, err := req.GetParsedDates()
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	var statusFilter []domain.TimesheetStatus
	if len(req.Statuses) > 0 {
		for _, status := range req.Statuses {
			statusFilter = append(statusFilter, domain.TimesheetStatus(status))
		}
	}

	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		ProjectIDs:      req.ProjectIDs,
		EmployeeID:      req.EmployeeID,
		TimesheetStatus: statusFilter,
		Limit:           10000,
		Offset:          0,
		SortBy:          "date",
		SortOrder:       "asc",
	}

	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetTimesheetsVN)
		return
	}

	headers := []string{
		"STT", "Ngày công", "Họ tên", "Mã NV", "Dự án", "Ca làm việc",
		"Loại thanh toán", "Mức lương", "Tổng tiền", "Tiền đã trả", "Trạng thái", "Duyệt bởi", "Ngày duyệt",
	}
	currencyColumns := []int{7, 8, 9}

	data := h.buildTimesheetExportData(timesheets)
	filename := fmt.Sprintf("bang_cong_%s_%s.xlsx", req.FromDate, req.ToDate)

	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		go func() {
			_ = auditSvc.LogFileExport(c.Request.Context(), "timesheets", len(timesheets), filename)
		}()
	}

	h.generateExcelResponse(c, "Timesheets", headers, data, currencyColumns, filename)
}

func translateStatusToVietnamese(status string) string {
	statusTranslations := map[string]string{
		"pending_approval": "Chờ phê duyệt",
		"approved":         "Đã phê duyệt",
		"rejected":         "Bị từ chối",
		"pending":          "Chờ thanh toán",
		"paid":             "Đã thanh toán",
		"cancelled":        "Đã hủy",
	}

	if translated, exists := statusTranslations[status]; exists {
		return translated
	}
	return status
}

func (h *Handler) buildTimesheetExportData(timesheets []*domain.Timesheet) [][]interface{} {
	excelService := excel.NewExportService()
	var data [][]interface{}

	for i, ts := range timesheets {
		employeeName, employeeCode := h.extractEmployeeInfo(ts)
		projectName := h.extractProjectName(ts)
		approvedBy, approvedDate := h.extractApprovalInfo(ts, excelService)

		rowData := []interface{}{
			i + 1, excelService.FormatDateValue(ts.Date), employeeName, employeeCode,
			projectName, ts.HoursWorked, ts.PayType, ts.PayRate, ts.Amount,
			ts.PaidAmount, translateStatusToVietnamese(string(ts.PaymentStatus)), approvedBy, approvedDate,
		}
		data = append(data, rowData)
	}
	return data
}

func (h *Handler) generateExcelResponse(c *gin.Context, sheetName string, headers []string, data [][]interface{}, currencyColumns []int, filename string) {
	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook(sheetName)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	if err := excelService.WriteHeaders(f, sheetName, headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	for i, rowData := range data {
		rowNum := i + 2
		if err := excelService.WriteDataRow(f, sheetName, rowNum, rowData, currencyColumns); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	if err := excelService.AutoSizeColumns(f, sheetName, headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}

func (h *Handler) extractEmployeeInfo(ts *domain.Timesheet) (string, string) {
	if ts.Employee != nil {
		return ts.Employee.FormattedFullname(), ts.Employee.CCCD
	}
	return "", ""
}

func (h *Handler) extractProjectName(ts *domain.Timesheet) string {
	if ts.Project != nil {
		return ts.Project.Name
	}
	return ""
}

func (h *Handler) extractApprovalInfo(ts *domain.Timesheet, excelService *excel.ExportService) (string, string) {
	approvedBy := ""
	if ts.ApprovedBy != nil && ts.ApprovedUser != nil {
		approvedBy = ts.ApprovedUser.Fullname
	}

	approvedDate := ""
	if ts.ApprovedAt != nil {
		approvedDate = excelService.FormatDate(ts.ApprovedAt)
	}

	return approvedBy, approvedDate
}
