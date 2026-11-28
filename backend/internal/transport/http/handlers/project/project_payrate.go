package project

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ListProjectPayrates gets payrates for a specific project

// Check if project exists

// Parse query parameters

// default
// default

// Handle page/pageSize pagination (new model)

// Convert to offset/limit for repository

// No toDate filter: payrates.to_date was dropped by migration 138 and end
// dates are derived for display from each config's successor. The FE list
// params never carried toDate.

// Get payrates using the injected payrate service

// Convert payrates to response format

// Calculate pagination

// CreateProjectPayrate creates a new payrate for a project
func (h *Handler) CreateProjectPayrate(c *gin.Context) {
	logger := observability.GetLogger()

	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		logger.Error("Invalid project ID parameter",
			"project_id_param", c.Param("id"),
			"error", err.Error(),
			"endpoint", "CreateProjectPayrate",
		)
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	logger.Info("Creating payrate for project",
		"project_id", projectID,
		"user_agent", c.GetHeader("User-Agent"),
	)

	// Check if project exists and validate status
	project, err := h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		logger.Error("Project not found or error accessing project",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Check if project status allows modifications
	if project.ProjectStatus == domain.ProjectStatusCompleted || project.ProjectStatus == domain.ProjectStatusCancelled {
		logger.Warn("Attempted to create payrate for project with restricted status",
			"project_id", projectID,
			"project_status", project.ProjectStatus,
		)
		response.BadRequest(c, constants.MsgCannotCreatePayrateForProjectVN+": "+string(project.ProjectStatus))
		return
	}

	// Log the raw request body for debugging
	body, _ := c.GetRawData()
	logger.Info("Raw request body received",
		"project_id", projectID,
		"body_size", len(body),
		"body", string(body),
	)

	// Reset body for binding (GetRawData consumes it)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var req dto.CreatePayrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind JSON request body",
			"error", err.Error(),
			"project_id", projectID,
			"endpoint", "CreateProjectPayrate",
			"raw_body", string(body),
		)
		response.BadRequest(c, fmt.Sprintf(constants.MsgInvalidRequestBodyFormatVN, err.Error()))
		return
	}

	logger.Info("Successfully parsed request body",
		"project_id", projectID,
		"rates_count", len(req.Rates),
		"effective_from", req.EffectiveFrom,
	)

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole, roleExists := c.Get(constants.CtxUserRole)
	if !roleExists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}
	if userRole.(string) == string(domain.RoleQuanly) {
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), uint(projectID), userID.(uint))
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}
		if !canModify {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	// Set project ID from URL
	req.ProjectID = uint(projectID)

	// Convert DTO to domain entity
	logger.Info("Converting DTO to domain entity",
		"project_id", projectID,
	)

	payrate, err := req.ToDomain()
	if err != nil {
		logger.Error("Failed to convert DTO to domain",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Normalize keys (trim + NFC) and reject Vietnamese fold-collisions so
	// divergent spellings of one position cannot be persisted.
	normalizedRates, normErr := payrate.Payrate.NormalizedKeys()
	if normErr != nil {
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, normErr.Error()))
		return
	}
	payrate.Payrate = normalizedRates

	// Validate rate structure using basic JSON validation
	logger.Info("Validating payrate configuration",
		"project_id", projectID,
	)

	if err := payrate.Payrate.Validate(); err != nil {
		logger.Error("Payrate configuration validation failed",
			"project_id", projectID,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("%s: %s", constants.MsgInvalidPayrateConfigurationVN, err.Error()))
		return
	}

	// Parse from date string to time.Time for database
	logger.Info("Parsing effective date",
		"project_id", projectID,
		"effective_from", req.EffectiveFrom,
	)

	fromDate, err := time.Parse(timeutil.DateFormat, req.EffectiveFrom)
	if err != nil {
		logger.Error("Failed to parse effective_from date",
			"project_id", projectID,
			"effective_from", req.EffectiveFrom,
			"expected_format", timeutil.DateFormat,
			"error", err.Error(),
		)
		response.BadRequest(c, fmt.Sprintf("Invalid effective_from format. Use %s format. Error: %s", timeutil.DateFormat, err.Error()))
		return
	}

	// From-date-only model: there are no end dates (migration 138 dropped the
	// column). A supplied effective_to is a contract violation — reject it
	// loudly instead of silently ignoring.
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		response.BadRequest(c, constants.MsgEffectiveToNotSupportedVN)
		return
	}

	// Set additional fields on the converted payrate.
	payrate.FromDate = fromDate
	payrate.CreatedBy = userID.(uint)

	logger.Info("Creating payrate in service layer",
		"project_id", projectID,
		"user_id", userID.(uint),
		"user_role", userRole.(string),
		"from_date", fromDate.Format(timeutil.DateFormat),
	)

	createdPayrate, err := h.payrateService.CreatePayrate(c.Request.Context(), payrate, userID.(uint), userRole.(string))
	if err != nil {
		logger.Error("Failed to create payrate in service layer",
			"project_id", projectID,
			"user_id", userID.(uint),
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
		)
		response.HandleDomainError(c, err)
		return
	}

	logger.Info("Successfully created payrate",
		"project_id", projectID,
		"payrate_id", createdPayrate.ID,
		"user_id", userID.(uint),
	)

	// Convert dates back to string for response
	payrateResponse := dto.PayrateResponse{
		ID:        createdPayrate.ID,
		ProjectID: createdPayrate.ProjectID,
		Rates:     createdPayrate.Payrate,
		FromDate:  createdPayrate.FromDate.Format(timeutil.DateFormat),
		CreatedBy: createdPayrate.CreatedBy,
		CreatedAt: createdPayrate.CreatedAt,
		UpdatedAt: createdPayrate.UpdatedAt,
	}

	response.SuccessCreated(c, payrateResponse, constants.MsgProjectPayrateCreatedSuccessfullyVN)
}

// GetCurrentProjectPayrate gets the currently active or nearest upcoming payrate for a project
func (h *Handler) GetCurrentProjectPayrate(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	// Check if project exists
	_, err = h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get current or upcoming payrate
	payrate, err := h.payrateService.GetCurrentOrUpcomingPayrateByProject(c.Request.Context(), uint(projectID))
	if err != nil {
		// No payrate is not an error — return 200 with null data
		if domain.IsNotFoundError(err) {
			response.Success(c, nil, "No active payrate found for this project")
			return
		}
		response.HandleDomainError(c, err)
		return
	}

	// Convert to response format
	payrateResponse := dto.PayrateResponse{
		ID:        payrate.ID,
		ProjectID: payrate.ProjectID,
		Rates:     payrate.Payrate,
		FromDate:  payrate.FromDate.Format(timeutil.DateFormat),
		CreatedBy: payrate.CreatedBy,
		CreatedAt: payrate.CreatedAt,
		UpdatedAt: payrate.UpdatedAt,
	}

	response.Success(c, payrateResponse, constants.MsgCurrentProjectPayrateRetrievedVN)
}
