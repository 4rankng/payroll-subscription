package bootstrap

import "github.com/gin-gonic/gin"

func setupDashboardRoutes(protected *gin.RouterGroup, container *Container) {
	dashboard := protected.Group("/dashboard")
	{
		dashboard.GET("/summary", container.Handlers.Dashboard.GetDashboardSummary)
		dashboard.GET("/recent-activities", container.Handlers.Dashboard.GetRecentActivities)
		dashboard.GET("/notifications", container.Handlers.Dashboard.GetSystemNotifications)
		dashboard.GET("/new-employees", container.Handlers.Dashboard.GetNewEmployees)
		dashboard.GET("/top-paid-employees", container.Handlers.Dashboard.GetTopPaidEmployees)
	}
}

// setupClockRoutes mounts the admin clock manipulation endpoints.
// Only available in non-production environments. Allows integration tests
// to manipulate the server's internal clock for time-dependent scenarios.
func setupClockRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.AdminClock == nil {
		return
	}
	if container.Config.App.Env == "production" {
		return
	}
	admin := v1.Group("/admin/clock")
	admin.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		admin.Use(container.Middleware.Authorization.Authorize())
	}
	admin.GET("", container.Handlers.AdminClock.GetTime)
	admin.POST("/set", container.Handlers.AdminClock.SetTime)
	admin.POST("/advance", container.Handlers.AdminClock.AdvanceTime)
	admin.POST("/reset", container.Handlers.AdminClock.ResetTime)
}


// setupProviderTransactionsAdminRoutes mounts the read-only stats endpoint
// for provider_transactions. Casbin-protects to admin via the standard
// Authorize() middleware. Always mounted — even when 9pay is dormant,
// the table exists and reports an empty (or historical) result set.

// setupZnsAdminRoutes mounts the admin ZNS tracking list over zns_messages —
// every ZNS the system sends, whatever feature. Casbin-protects to admin via
// the standard Authorize() middleware (admin-only, 403 for everyone else).

func setupSettingsRoutes(v1 *gin.RouterGroup, container *Container) {
	settings := v1.Group("/settings")
	settings.Use(container.Middleware.Auth.Authenticate())
	{
		// GET endpoints - accessible by all authenticated users
		settings.GET("", container.Handlers.Settings.ListSettings)
		settings.GET("/:id", container.Handlers.Settings.GetSetting)
		settings.GET("/key/:key", container.Handlers.Settings.GetSettingByKey)

		// Write endpoints - admin only
		settings.POST("", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.CreateSetting)
		settings.PUT("/:id", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.UpdateSetting)
		settings.DELETE("/:id", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.DeleteSetting)
	}
}


func setupDBExportRoutes(protected *gin.RouterGroup, container *Container) {
	dbExport := protected.Group("/db-export")
	{
		dbExport.POST("", container.Handlers.DBExport.StartExport)
		dbExport.GET("", container.Handlers.DBExport.ListExportJobs)
		dbExport.GET("/:jobId/status", container.Handlers.DBExport.GetExportStatus)
		dbExport.GET("/:jobId/download", container.Handlers.DBExport.DownloadExport)
	}
}
