package project

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetProjectSummary gets aggregated project metrics
// @Summary Get project summary
// @Description Get aggregated metrics for all projects
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} dto.ProjectSummaryResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/summary [get]
func (h *Handler) GetProjectSummary(c *gin.Context) {
	summary, err := h.projectService.GetProjectSummary(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetProjectSummaryVN)
		return
	}

	summaryResponse := dto.ProjectSummaryResponse{
		TotalActiveProjects:    summary.TotalActiveProjects,
		TotalPayoutVND:         summary.TotalPayoutVND,
		TotalPendingPayableVND: summary.TotalPendingPayableVND,
	}

	response.Success(c, summaryResponse, constants.MsgProjectSummaryRetrievedVN)
}

// GetPartnerProjectSummary gets project statistics for partner users
// @Summary Get partner project summary
// @Description Get project statistics for the partner dashboard (same accessible set as the partner project list: owned + shared projects)
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} dto.PartnerProjectSummaryResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/partner-summary [get]

func (h *Handler) GetPartnerProjectSummary(c *gin.Context) {
	// 1. Validate user context and scope like the partner project list
	// (ListProjects): quanly counts their accessible set; every other role
	// sees the same unscoped list on /projects, so their summary counts the
	// whole table.
	var userID uint
	if c.GetString(constants.CtxUserRole) == string(domain.RoleQuanly) {
		raw, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}
		uid, ok := raw.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
		userID = uid
	}

	// 2. Compute the summary over the scoped set
	summary, err := h.projectService.GetPartnerProjectSummary(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetProjectSummaryVN)
		return
	}

	summaryResponse := dto.PartnerProjectSummaryResponse{
		TotalProjects:     summary.TotalProjects,
		ActiveProjects:    summary.ActiveProjects,
		CompletedProjects: summary.CompletedProjects,
		TotalEmployees:    summary.TotalEmployees,
	}
	response.Success(c, summaryResponse, constants.MsgProjectSummaryRetrievedVN)
}
