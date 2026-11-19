package handlers

import (
	"api-server/internal/app/dto"
	"api-server/internal/app/services/dashboard"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardService *dashboard.Service
}

func NewDashboardHandler(dashboardService *dashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetRecentActivities retrieves recent system activities
// @Summary Get recent activities
// @Description Get a feed of recent system activities for the dashboard activity section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of activities per page (default: 10, max: 100)"
// @Param sortBy query string false "Sort field (default: created_at)"
// @Param sortOrder query string false "Sort order - asc or desc (default: desc)"
// @Success 200 {object} dto.RecentActivitiesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/recent-activities [get]
func (h *DashboardHandler) GetRecentActivities(c *gin.Context) {
	var req dto.RecentActivitiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	activities, pagination, err := h.dashboardService.GetRecentActivities(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, activities, constants.MsgRecentActivitiesRetrievedVN, *pagination)
}

// GetSystemNotifications retrieves system notifications
// @Summary Get system notifications
// @Description Get system notifications with unread count for the dashboard notifications section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of notifications per page (default: 5, max: 20)"
// @Param include_read query bool false "Include read notifications (default: false)"
// @Success 200 {object} dto.SystemNotificationsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/notifications [get]
func (h *DashboardHandler) GetSystemNotifications(c *gin.Context) {
	var req dto.SystemNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	notifications, pagination, err := h.dashboardService.GetSystemNotifications(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, notifications, constants.MsgSystemNotificationsRetrievedVN, *pagination)
}

// GetNewEmployees retrieves recently added employees
// @Summary Get new employees
// @Description Get recently added employees for the dashboard new employees section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of employees per page (default: 5, max: 20)"
// @Param days query int false "Number of days to look back (default: 30)"
// @Success 200 {object} dto.NewEmployeesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/new-employees [get]
func (h *DashboardHandler) GetNewEmployees(c *gin.Context) {
	var req dto.NewEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	employees, pagination, err := h.dashboardService.GetNewEmployees(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, employees, constants.MsgNewEmployeesRetrievedVN, *pagination)
}

// GetDashboardSummary retrieves dashboard summary statistics
// ADMIN only - This endpoint provides sensitive financial and employee metrics
// @Summary Get dashboard summary
// @Description Get dashboard summary statistics including employee counts, salary information, and profit metrics (ADMIN only)
// @Tags dashboard
// @Accept json
// @Produce json
// @Param month query string false "Month in YYYY-MM format (e.g., 2025-04). Defaults to current month."
// @Success 200 {object} dto.DashboardSummaryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/summary [get]
func (h *DashboardHandler) GetDashboardSummary(c *gin.Context) {
	var req dto.DashboardSummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	summary, err := h.dashboardService.GetDashboardSummary(c.Request.Context(), req.Month)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, summary, constants.MsgDashboardSummaryRetrievedVN)
}

// GetTopPaidEmployees returns the top N employees by total paid amount
// @Summary Get top paid employees
// @Description Get top employees ranked by total paid amount for a given month or all-time
// @Tags dashboard
// @Produce json
// @Param month query string false "Month in YYYY-MM format (optional; omit for all-time)"
// @Param limit query int false "Number of employees to return (default: 10, max: 50)"
// @Success 200 {object} dto.TopPaidEmployeesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/top-paid-employees [get]
func (h *DashboardHandler) GetTopPaidEmployees(c *gin.Context) {
	var req dto.TopPaidEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	data, err := h.dashboardService.GetTopPaidEmployees(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy danh sách nhân viên được trả lương cao nhất thành công")
}
