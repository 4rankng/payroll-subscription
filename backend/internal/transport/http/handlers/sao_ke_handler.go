package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"api-server/internal/app/services/billing"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"

	"github.com/gin-gonic/gin"
)

// saoKeContentType is the MIME type of the statement workbook download.
const saoKeContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// SaoKeHandler serves the Sổ cái sao kê surface (PRD §5.8, PAYR-41):
// statement workbook export, the persisted recipient list, and statement
// email dispatch — ketoan and giamdoc per the casbin policy.
type SaoKeHandler struct {
	saoke *billing.SaoKeService
	audit *infrastructure.AuditService
}

func NewSaoKeHandler(saoke *billing.SaoKeService, audit *infrastructure.AuditService) *SaoKeHandler {
	return &SaoKeHandler{saoke: saoke, audit: audit}
}

// Export streams the statement workbook for the Kỳ the report_date resolves
// to (pay-date match first, otherwise the payment window containing it).
func (h *SaoKeHandler) Export(c *gin.Context) {
	reportDate, err := billing.ParseSaoKeReportDate(c.Query("report_date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, workbook, filename, err := h.saoke.Export(c.Request.Context(), reportDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// The filename is ASCII (sao_ke_tt_<DD-MM-YYYY>.xlsx) so the plain
	// attachment form is always valid; escape defensively anyway.
	disposition := fmt.Sprintf("attachment; filename=%s", filename)
	if encoded := url.PathEscape(filename); encoded != filename {
		disposition = fmt.Sprintf("attachment; filename*=UTF-8''%s", encoded)
	}
	c.Header("Content-Disposition", disposition)
	h.auditExport(c, "sao_ke_export", len(data.Rows), filename)
	c.Data(http.StatusOK, saoKeContentType, workbook)
}

// GetRecipients returns the persisted statement mailing list.
func (h *SaoKeHandler) GetRecipients(c *gin.Context) {
	recipients, err := h.saoke.GetRecipients(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": recipients})
}

// saoKeRecipientsRequest is the PUT /billing/saoke/recipients body.
type saoKeRecipientsRequest struct {
	Recipients []string `json:"recipients"`
	CC         []string `json:"cc"`
}

// SaveRecipients validates and persists the statement mailing list. Empty
// lists are allowed (sending enforces ≥1 To recipient on its own).
func (h *SaoKeHandler) SaveRecipients(c *gin.Context) {
	var req saoKeRecipientsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "danh sách người nhận không hợp lệ"})
		return
	}
	if err := h.saoke.SaveRecipients(c.Request.Context(), &billing.SaoKeRecipients{
		Recipients: req.Recipients,
		CC:         req.CC,
	}, c.GetUint(constants.CtxUserID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": billing.SaoKeRecipients{
		Recipients: req.Recipients,
		CC:         req.CC,
	}})
}

// saoKeEmailRequest is the POST /billing/saoke/email body.
type saoKeEmailRequest struct {
	ReportDate string   `json:"report_date"`
	Recipients []string `json:"recipients"`
	CC         []string `json:"cc"`
}

// SendEmail builds the statement workbook for the resolved Kỳ and mails it.
// The dev sandbox provider captures the message in-process; production uses
// the configured provider.
func (h *SaoKeHandler) SendEmail(c *gin.Context) {
	var req saoKeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "yêu cầu gửi sao kê không hợp lệ"})
		return
	}
	reportDate, err := billing.ParseSaoKeReportDate(req.ReportDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.saoke.SendEmail(c.Request.Context(), reportDate, req.Recipients, req.CC); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{
		"message": "Email sao kê đã được gửi",
	}})
}

// auditExport emits the file-export audit event non-blocking. The request
// context dies with the handler, so the goroutine carries a background one.
func (h *SaoKeHandler) auditExport(c *gin.Context, dataType string, recordCount int, filename string) {
	if h.audit == nil {
		return
	}
	auditSvc := h.audit
	go func() {
		_ = auditSvc.LogFileExport(context.Background(), dataType, recordCount, filename)
	}()
}
