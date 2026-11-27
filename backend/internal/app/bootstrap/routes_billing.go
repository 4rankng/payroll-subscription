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
}
