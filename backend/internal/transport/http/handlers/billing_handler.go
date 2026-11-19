package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"api-server/internal/app/services/billing"
	"api-server/internal/constants"

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

// Upload processes one bank-transfer result workbook for (ky, work_month).
// Per-row outcomes are returned inline; a whole-file rejection (no
// processable rows) already fired the D10 all-admin push inside the service.
func (h *BillingHandler) Upload(c *gin.Context) {
	ky, err := strconv.Atoi(c.PostForm("ky"))
	if err != nil || ky < 1 || ky > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ky phải là số 1..4"})
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
	result, err := h.uploads.ProcessUpload(c.Request.Context(), ky, workMonth, fileHeader.Filename, data, uploaderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
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
	c.JSON(http.StatusOK, gin.H{"items": rows})
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
	rows, err := h.uploads.ListKyPaidRecords(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.uploads.MonthlySummary(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "summary_by_ky": summary})
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
	c.JSON(http.StatusOK, gin.H{"report": report, "lines": lines})
}

// GetReport returns one monthly report with its Kỳ lines.
func (h *BillingHandler) GetReport(c *gin.Context) {
	workMonth := strings.TrimSpace(c.Query("work_month"))
	if len(workMonth) != 7 || workMonth[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_month phải có dạng YYYY-MM"})
		return
	}
	report, _, err := h.reports.GetReport(c.Request.Context(), workMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chưa có báo cáo cho tháng này"})
		return
	}
	c.JSON(http.StatusOK, report)
}
