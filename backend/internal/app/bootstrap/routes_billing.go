package bootstrap

import (
	"github.com/gin-gonic/gin"
)

// setupBillingRoutes mounts the subscription billing surface (FR-2/FR-3).
// All routes require authentication; writes additionally require the casbin
// policy (quanly uploads, ketoan/admin read — see configs/casbin_policy.csv).
func setupBillingRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Billing == nil {
		return
	}
	billing := v1.Group("/billing")
	billing.Use(container.Middleware.Auth.Authenticate())
	{
		billing.POST("/uploads", container.Middleware.Authorization.Authorize(), container.Handlers.Billing.Upload)
		billing.GET("/uploads", container.Handlers.Billing.ListUploads)
		billing.GET("/ky-paid-records", container.Handlers.Billing.KyPaidRecords)
		billing.GET("/dashboard/paid-series", container.Handlers.Billing.PaidSeries)
		billing.GET("/dashboard/ky-summary", container.Handlers.Billing.KySummary)
		billing.GET("/dashboard/recon-status", container.Handlers.Billing.ReconStatus)
	}
	reports := v1.Group("/billing/reports")
	reports.Use(container.Middleware.Auth.Authenticate())
	{
		reports.POST("/generate", container.Middleware.Authorization.Authorize(), container.Handlers.Billing.GenerateReport)
		reports.GET("", container.Handlers.Billing.GetReport)
	}

	// Sao kê (PRD §5.8, PAYR-41): statement export, recipients and email —
	// ketoan + giamdoc per the casbin policy; every route is casbin-gated.
	saoKe := v1.Group("/billing/saoke")
	saoKe.Use(container.Middleware.Auth.Authenticate())
	{
		saoKe.GET("/export", container.Middleware.Authorization.Authorize(), container.Handlers.SaoKe.Export)
		saoKe.GET("/recipients", container.Middleware.Authorization.Authorize(), container.Handlers.SaoKe.GetRecipients)
		saoKe.PUT("/recipients", container.Middleware.Authorization.Authorize(), container.Handlers.SaoKe.SaveRecipients)
		saoKe.POST("/email", container.Middleware.Authorization.Authorize(), container.Handlers.SaoKe.SendEmail)
	}
}

// setupPaymentHistoryRoutes mounts the read-only payment history (Lịch sử trả
// lương): the bank-entry ledger of a work month and its per-project Excel
// export. Every route is casbin-gated — quanly / ketoan / giamdoc (admin
// wildcard covers admin; see configs/casbin_policy.csv).
func setupPaymentHistoryRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.PaymentHistory == nil {
		return
	}
	payrolls := v1.Group("/payrolls")
	payrolls.Use(container.Middleware.Auth.Authenticate())
	{
		payrolls.GET("/bank-transfer-histories", container.Middleware.Authorization.Authorize(), container.Handlers.PaymentHistory.List)
		payrolls.GET("/bank-transfer-histories/export", container.Middleware.Authorization.Authorize(), container.Handlers.PaymentHistory.Export)
	}
}
