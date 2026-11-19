package bootstrap

import "github.com/gin-gonic/gin"

func setupTimesheetRoutes(protected *gin.RouterGroup, container *Container) {
	timesheets := protected.Group("/timesheets")
	{
		timesheets.GET("/summary", container.Handlers.Timesheet.GetSummary)
		timesheets.POST("", container.Handlers.Timesheet.BulkCreateTimesheets)
		timesheets.POST("/preview", container.Handlers.Timesheet.PreviewTimesheets)
		timesheets.GET("", container.Handlers.Timesheet.ListTimesheets)
		timesheets.GET("/grouped", container.Handlers.Timesheet.ListGroupedTimesheets)
		timesheets.GET("/export", container.Handlers.Timesheet.ExportTimesheets)
		timesheets.POST("/export-entries-template", container.Handlers.Timesheet.ExportTimesheetTemplate)
		timesheets.POST("/upload-entries-excel", container.Handlers.Timesheet.UploadTimesheetEntries)
		timesheets.GET("/projects/:id", container.Handlers.Timesheet.GetTimesheetsByProjectAndDate)
		timesheets.POST("/bulk-approve", container.Handlers.Timesheet.BulkApprove)
		timesheets.POST("/bulk-reject", container.Handlers.Timesheet.BulkReject)
		timesheets.POST("/bulk-reset", container.Handlers.Timesheet.BulkReset)
		timesheets.POST("/approve-all", container.Handlers.Timesheet.ApproveAllTimesheets)
		timesheets.POST("/reset-all", container.Handlers.Timesheet.ResetAllTimesheets)

		// Partner BCC import endpoints
		partnerImport := timesheets.Group("/partner-import")
		{
			partnerImport.POST("", container.Handlers.BCCImport.UploadBCC)
			partnerImport.GET("", container.Handlers.BCCImport.ListPartnerImports)
			partnerImport.GET("/:id", container.Handlers.BCCImport.GetPartnerImport)
			partnerImport.GET("/:id/download", container.Handlers.BCCImport.DownloadPartnerImport)
		}

		timesheets.GET("/:id", container.Handlers.Timesheet.GetTimesheet)
		timesheets.PUT("/:id", container.Handlers.Timesheet.UpdateTimesheet)
		timesheets.DELETE("/:id", container.Handlers.Timesheet.DeleteTimesheet)
		timesheets.PUT("/:id/approve", container.Handlers.Timesheet.ApproveTimesheet)
		timesheets.PUT("/:id/reject", container.Handlers.Timesheet.RejectTimesheet)
	}
}
