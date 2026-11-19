package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func setupHealthRoutes(router *gin.Engine, container *Container) {
	api := router.Group("/api")
	{
		api.GET("/healthz", container.Handlers.Health.HealthCheck)
		api.GET("/ready", container.Handlers.Health.ReadinessCheck)
	}
}

func setupLegacyCacheRoutes(router *gin.Engine, container *Container) {
	if container == nil {
		return
	}
	if container.Handlers == nil || container.Handlers.Cache == nil {
		return
	}

	cache := router.Group("/v1/cache")
	if container.Middleware != nil && container.Middleware.Auth != nil {
		cache.Use(container.Middleware.Auth.Authenticate())
	}
	cache.DELETE("", container.Handlers.Cache.DeleteAllCache)
}

func setupCacheRoutes(v1 *gin.RouterGroup, container *Container) {
	cache := v1.Group("/cache")
	cache.Use(container.Middleware.Auth.Authenticate())
	{
		cache.DELETE("", container.Handlers.Cache.DeleteAllCache)
	}
}

func setupMobileRoutes(v1 *gin.RouterGroup, container *Container) {
	mobile := v1.Group("/mobile")
	mobile.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		mobile.Use(container.Middleware.Authorization.Authorize())
	}

}

func setupNotificationRoutes(protected *gin.RouterGroup, container *Container) {
	// Admin-only endpoint for sending custom notifications
	protected.POST("/notification", container.Handlers.Notification.CreateCustomNotification)

	notifications := protected.Group("/notifications")
	{
		notifications.GET("", container.Handlers.Notification.GetNotifications)
		notifications.GET("/unread", container.Handlers.Notification.GetUnreadNotifications)
		notifications.GET("/unread/count", container.Handlers.Notification.GetUnreadCount)
		notifications.PUT("/:id/read", container.Handlers.Notification.MarkAsRead)
		notifications.PUT("/read-all", container.Handlers.Notification.MarkAllAsRead)
		notifications.POST("/no-default-password", container.Handlers.Notification.NotifyDefaultPasswordUsers)
	}
}

func setupPushRoutes(v1 *gin.RouterGroup, protected *gin.RouterGroup, container *Container) {
	// Public: VAPID key is needed before subscription and is not sensitive
	v1.GET("/push/vapid-key", container.Handlers.Push.GetVAPIDKey)

	push := protected.Group("/push")
	{
		push.POST("/subscribe", container.Handlers.Push.Subscribe)
		push.POST("/unsubscribe", container.Handlers.Push.Unsubscribe)
	}
}

func setupAssetRoutes(protected *gin.RouterGroup, container *Container) {
	assets := protected.Group("/assets")
	{
		assets.POST("/upload", container.Handlers.Asset.UploadAsset)
		assets.GET("", container.Handlers.Asset.ListAssets)
		assets.GET("/:id/download", container.Handlers.Asset.DownloadAsset)
		assets.GET("/:id", container.Handlers.Asset.GetAsset)
	}
}

func setupEmailRoutes(protected *gin.RouterGroup, container *Container) {
	emails := protected.Group("/email")
	{
		emails.GET("/senders", container.Handlers.Email.GetAvailableSenders)
		emails.GET("/history", container.Handlers.Email.GetEmailHistory)
		emails.POST("/send", container.Handlers.Email.SendGenericEmail)
	}
}

func setupDBRoutes(protected *gin.RouterGroup, container *Container) {
	// Removed migration routes as they are no longer needed
}

func setupAuditRoutes(protected *gin.RouterGroup, container *Container) {
	audit := protected.Group("/audit")
	{
		audit.GET("/logs", container.Handlers.Audit.ListAuditLogs)
		audit.GET("/logs/:id", container.Handlers.Audit.GetAuditLog)
	}
}

func setupTestRoutes(v1 *gin.RouterGroup) {
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
}
