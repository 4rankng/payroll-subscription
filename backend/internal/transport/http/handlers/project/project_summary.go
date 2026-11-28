package project

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
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

// 1. Validate user context
