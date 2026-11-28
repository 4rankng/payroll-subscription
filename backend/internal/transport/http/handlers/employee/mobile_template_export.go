package employee

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportMissingMobileTemplate streams the "NV chưa có SĐT" workbook that the
// import-mobiles modal pairs with its upload: one row per employee with no
// mobile on file, the Mobile column left blank for the partner to fill.
// Admin or quanly, the same gate as ImportMobiles — the download and the
// upload are the two halves of one fix loop. Unpaginated on purpose: the
// monthly workbook is larger than one list page, and a truncated template
// would silently skip employees.
func (h *Handler) ExportMissingMobileTemplate(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			observability.GetLogger().Error("panic in ExportMissingMobileTemplate",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		}
	}()

	if userRole := c.GetString(constants.CtxUserRole); userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleQuanly) {
		response.Forbidden(c, constants.MsgMobileBackfillAdminOnlyVN)
		return
	}

	// Same scoping as the missing-mobile list: quanly sees only their set.
	filters, _, _, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}
	filters.Limit = 0
	filters.Offset = 0

	employees, err := h.employeeService.GetEmployeesWithMissingMobile(c.Request.Context(), filters)
	if err != nil {
		observability.GetLogger().Error("failed to list employees for mobile template", "error", err)
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	excelService := excel.NewExportService()
	sheet := "NV chua co SDT"
	f, err := excelService.CreateStyledWorkbook(sheet)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	// ParseMobileBackfillExcel locates "CCCD" and "Mobile" by normalized
	// header; the remaining columns only help the user identify the row.
	headers := []string{"STT", "Họ và tên", "CCCD", "Dự án", "Mobile"}
	if err := excelService.WriteHeaders(f, sheet, headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	var data [][]interface{}
	for i, emp := range employees {
		projectNames := make([]string, 0, len(emp.CurrentProjects))
		for _, project := range emp.CurrentProjects {
			projectNames = append(projectNames, project.Name)
		}
		data = append(data, []interface{}{
			i + 1,
			emp.FormattedFullname(),
			emp.CCCD,
			strings.Join(projectNames, ", "),
			"", // Mobile — the column the partner fills in
		})
	}

	for i, rowData := range data {
		rowNum := i + 2 // Row 1 is headers
		if err := excelService.WriteDataRow(f, sheet, rowNum, rowData, []int{}); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	if err := excelService.AutoSizeColumns(f, sheet, headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	filename := fmt.Sprintf("NV_chua_co_SDT_%s.xlsx", h.clock.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Audit the export (non-blocking); Background context because the request
	// context dies with the handler.
	if auditSvc, ok := h.auditService.(interface {
		LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error
	}); ok && auditSvc != nil {
		_ = auditSvc.LogFileExport(context.Background(), "employee_mobile_template", len(employees), filename)
	}

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
