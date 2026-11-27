package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/services/billing"
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"

	"github.com/gin-gonic/gin"
)

// BillingHandler serves the subscription billing surface (FR-2/FR-3):
// transfer-result uploads with per-row verification outcomes, the upload
// history per work month, and the per-employee-per-Kỳ paid-records read
// model that feeds the billing report.
type BillingHandler struct {
	uploads *billing.TransferUploadService
	reports *billing.BillingReportService
}

func NewBillingHandler(uploads *billing.TransferUploadService, reports *billing.BillingReportService) *BillingHandler {
	return &BillingHandler{uploads: uploads, reports: reports}
}

// Upload processes one bank-transfer result workbook for (cycle, work_month).
// Per-row outcomes are returned inline; a whole-file rejection (no
// processable rows) already fired the D10 all-admin push inside the service.
func (h *BillingHandler) Upload(c *gin.Context) {
	cycle, err := strconv.Atoi(c.PostForm("cycle"))
	if err != nil || cycle < 1 || cycle > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cycle phải là số 1..4"})
		return
	}
	workMonth := strings.TrimSpace(c.PostForm("work_month"))
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "thiếu file kết quả chuyển khoản"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("mở file thất bại: %v", err)})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("đọc file thất bại: %v", err)})
		return
	}

	uploaderID := c.GetUint(constants.CtxUserID)
	result, err := h.uploads.ProcessUpload(c.Request.Context(), cycle, workMonth, fileHeader.Filename, data, uploaderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

// ListUploads returns the upload history of one work month, newest first.
func (h *BillingHandler) ListUploads(c *gin.Context) {
	workMonth := strings.TrimSpace(c.Query("work_month"))
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.uploads.ListUploads(c.Request.Context(), workMonth, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"items": rows}})
}

// ReconStatus serves the Tổng quan reconciliation block (PRD §5.1): per-Kỳ
// upload counts, parsed vs matched rows, and the derived state for one work
// month. Defaults to the current work month.
func (h *BillingHandler) ReconStatus(c *gin.Context) {
	now := clock.Now()
	workMonth := now.Format("2006-01")
	if v := c.Query("work_month"); v != "" {
		if len(v) != 7 || v[4] != '-' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
			return
		}
		workMonth = v
	}
	states, err := h.uploads.ReconStatus(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{
		"work_month": workMonth,
		"kyes":       states,
	}})
}

// KyPaidRecords returns the per-employee-per-Kỳ billing rows of one work
// month plus the per-Kỳ distinct summary — the FR-1 read model behind the
// FR-3 billing report.
func (h *BillingHandler) KyPaidRecords(c *gin.Context) {
	workMonth := strings.TrimSpace(c.Query("work_month"))
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	rows, err := h.uploads.ListEmployeePayCycles(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.uploads.MonthlySummary(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"items": rows, "summary_by_cycle": summary}})
}

// GenerateReport builds (or regenerates) the monthly billing report for one
// work month — admin and giamdoc per D19. The fee snapshot (FR-4) is taken
// at generation time.
func (h *BillingHandler) GenerateReport(c *gin.Context) {
	var body struct {
		WorkMonth string `json:"work_month" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month là bắt buộc (YYYY-MM)"})
		return
	}
	workMonth := strings.TrimSpace(body.WorkMonth)
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	generatedBy := c.GetUint(constants.CtxUserID)
	report, lines, err := h.reports.GenerateMonthly(c.Request.Context(), workMonth, generatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reports.NotifyReportReady(c.Request.Context(), workMonth)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"report": report, "lines": lines}})
}

// GetReport returns one monthly report with its Kỳ lines.
func (h *BillingHandler) GetReport(c *gin.Context) {
	workMonth := strings.TrimSpace(c.Query("work_month"))
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	report, lines, err := h.reports.GetReport(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chưa có báo cáo cho tháng này"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"report": report, "lines": lines}})
}

// PaidSeries serves the FR-6 Tổng quan time series: per-day distinct paid
// employees and paid amount per project. Defaults to the last 90 days.
func (h *BillingHandler) PaidSeries(c *gin.Context) {
	to := clock.Now()
	from := to.AddDate(0, 0, -90)
	if v := c.Query("from"); v != "" {
		parsed, err := time.ParseInLocation("2006-01-02", v, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from phải có dạng YYYY-MM-DD"})
			return
		}
		from = parsed
	}
	if v := c.Query("to"); v != "" {
		parsed, err := time.ParseInLocation("2006-01-02", v, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to phải có dạng YYYY-MM-DD"})
			return
		}
		to = parsed
	}
	var projectID *uint
	if v := c.Query("project_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "project_id không hợp lệ"})
			return
		}
		p := uint(id)
		projectID = &p
	}
	rows, err := h.reports.PaidSeries(c.Request.Context(), from, to, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"items": rows}})
}

// KySummary serves the FR-6 Tổng quan per-month table: distinct employees per
// Kỳ for the last N work months (query `months`, default 12, max 24) plus the
// current subscription fee (BR-1) so the UI can render VND totals.
func (h *BillingHandler) KySummary(c *gin.Context) {
	months := 12
	if v := c.Query("months"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "months phải là số nguyên dương"})
			return
		}
		months = parsed
	}
	summary, err := h.reports.DashboardKySummary(c.Request.Context(), months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": summary})
}
