package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"api-server/internal/app/services/billing"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// PaymentHistoryHandler serves the read-only payment history (Lịch sử trả
// lương): the paginated bank-entry ledger of a work month and its per-project
// Excel export.
type PaymentHistoryHandler struct {
	payments *billing.PaymentHistoryService
	audit    *infrastructure.AuditService
}

func NewPaymentHistoryHandler(payments *billing.PaymentHistoryService, audit *infrastructure.AuditService) *PaymentHistoryHandler {
	return &PaymentHistoryHandler{payments: payments, audit: audit}
}

// List — GET /payrolls/bank-transfer-histories
// Query: month (YYYY-MM, default current), ky (1..4, optional), search,
// page (default 1), page_size (default 50).
func (h *PaymentHistoryHandler) List(c *gin.Context) {
	q, ok := h.parseQuery(c)
	if !ok {
		return
	}
	result, err := h.payments.ListPaymentHistory(c.Request.Context(), q)
	if err != nil {
		writePaymentHistoryError(c, err)
		return
	}
	// Standard {status, data} envelope — apiClient-based services read
	// response.data, so a bare c.JSON(result) leaves it undefined client-side.
	response.Success(c, result, "")
}

// Export — GET /payrolls/bank-transfer-histories/export
// Query: month (YYYY-MM, default current), ky (1..4, optional), search.
// Returns the per-project Excel workbook (STT | Nhân viên | CCCD | Vị trí |
// Số tiền | Ngày thanh toán per sheet).
func (h *PaymentHistoryHandler) Export(c *gin.Context) {
	q, ok := h.parseQuery(c)
	if !ok {
		return
	}
	q.Page, q.PageSize = 0, 0 // export covers the whole filtered set

	out, err := h.payments.ExportPaymentHistory(c.Request.Context(), q)
	if err != nil {
		writePaymentHistoryError(c, err)
		return
	}

	filename := fmt.Sprintf("lich-su-tra-luong_%s", q.Month)
	if q.Ky != 0 {
		filename += fmt.Sprintf("_ky%d", q.Ky)
	}
	filename += ".xlsx"

	if h.audit != nil {
		_ = h.audit.LogFileExport(c.Request.Context(), "payment_history", out.RecordCount, filename)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(out.Workbook)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", out.Workbook)
}

// parseQuery validates the shared query parameters. ok=false means the error
// response has already been written.
func (h *PaymentHistoryHandler) parseQuery(c *gin.Context) (billing.PaymentHistoryQuery, bool) {
	q := billing.PaymentHistoryQuery{
		Month:    strings.TrimSpace(c.Query("month")),
		Search:   strings.TrimSpace(c.Query("search")),
		Page:     1,
		PageSize: 50,
	}

	if q.Month == "" {
		q.Month = clock.Now().Format("2006-01")
	} else if len(q.Month) != 7 || q.Month[4] != '-' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month phải có dạng YYYY-MM"})
		return q, false
	}

	if v := strings.TrimSpace(c.Query("ky")); v != "" {
		ky, err := strconv.Atoi(v)
		if err != nil || ky < 1 || ky > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ky phải là số 1..4"})
			return q, false
		}
		q.Ky = ky
	}

	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 {
			q.Page = p
		}
	}
	if v := strings.TrimSpace(c.Query("page_size")); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps >= 1 {
			q.PageSize = ps
		}
	}
	return q, true
}

func writePaymentHistoryError(c *gin.Context, err error) {
	if domain.IsValidationError(err) {
		response.BadRequest(c, err.Error())
		return
	}
	response.InternalServerError(c, err.Error())
}
