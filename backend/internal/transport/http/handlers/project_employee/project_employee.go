package project_employee

import (
	"log/slog"
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	projectEmployeeService    *project.ProjectEmployeeService
	employeeService           *employee.EmployeeService
	projectService            *project.ProjectService
	projectPermissionService  *project.ProjectPermissionService
	employeePermissionService *employee.EmployeePermissionService
	logger                    *slog.Logger
}

func NewHandler(projectEmployeeService *project.ProjectEmployeeService, employeeService *employee.EmployeeService, projectService *project.ProjectService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService, logger *slog.Logger) *Handler {
	return &Handler{
		projectEmployeeService:    projectEmployeeService,
		employeeService:           employeeService,
		projectService:            projectService,
		projectPermissionService:  projectPermissionService,
		employeePermissionService: employeePermissionService,
		logger:                    logger,
	}
}

func (h *Handler) ListProjectEmployees(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Project ID must be a valid number")
		return
	}

	projectIDUint := uint(projectID)
	filters := domain.ProjectEmployeeFilters{
		ProjectID: &projectIDUint,
		Limit:     50,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 50
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 10000 {
			pageSize = size
		}
	}

	// Add status filter
	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	// Add free-text search across denormalized employee fields
	filters.Search = c.Query("search")

	// For partner users, check if they have limited employee-based access
	userRole := c.GetString(constants.CtxUserRole)
	var accessibleEmployeeIDs map[uint]bool // Use map for O(1) lookup
	var hasLimitedAccess bool

	if userRole == string(domain.RoleQuanly) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}

		// Check if partner has full project access (creator or explicit project_users access)
		// If they do, they can see all employees. If not, they only have employee-based access.
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), projectIDUint, uid)
		if err != nil {
			h.logger.ErrorContext(c.Request.Context(), "Failed to check project modify permission",
				"project_id", projectIDUint,
				"user_id", uid,
				"error", err,
			)
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}

		// If user doesn't have full project access, they only have employee-based access
		if !canModify {
			hasLimitedAccess = true
			// Build a set of accessible employee IDs
			accessibleEmployeeIDs = make(map[uint]bool)

			// Get employees this partner user has been granted access to
			grantedEmployeeIDs, err := h.employeePermissionService.GetAccessibleEmployeeIDs(c.Request.Context(), uid)
			if err != nil {
				h.logger.ErrorContext(c.Request.Context(), "Failed to get accessible employee IDs",
					"user_id", uid,
					"error", err,
				)
				response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
				return
			}

			for _, empID := range grantedEmployeeIDs {
				accessibleEmployeeIDs[empID] = true
			}
		}
	}

	// If user has limited access, we need to get all assignments and filter them
	// Otherwise, we can use pagination at the DB level
	if hasLimitedAccess {
		// Get all assignments for the project without pagination
		allFilters := filters
		allFilters.Limit = 10000 // Large limit to get all
		allFilters.Offset = 0

		assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), allFilters)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}

		// Filter assignments to only those the user can access
		var filteredAssignments []*domain.ProjectEmployee
		for _, assignment := range assignments {
			if accessibleEmployeeIDs[assignment.EmployeeID] {
				filteredAssignments = append(filteredAssignments, assignment)
			}
		}

		// Apply pagination to filtered results
		total := int64(len(filteredAssignments))
		startIdx := (page - 1) * pageSize
		endIdx := startIdx + pageSize

		if startIdx >= len(filteredAssignments) {
			startIdx = len(filteredAssignments)
			endIdx = len(filteredAssignments)
		} else if endIdx > len(filteredAssignments) {
			endIdx = len(filteredAssignments)
		}

		paginatedAssignments := filteredAssignments[startIdx:endIdx]

		// Build response
		var assignmentResponses []dto.ProjectEmployeeResponse
		for _, assignment := range paginatedAssignments {
			assignmentResponses = append(assignmentResponses, dto.ProjectEmployeeResponse{
				ID:           assignment.ID,
				ProjectID:    assignment.ProjectID,
				EmployeeID:   assignment.EmployeeID,
				EmployeeName: assignment.EmployeeName,
				EmployeeCCCD: assignment.EmployeeCCCD,
				EmployeeCode: assignment.EmployeeCode,
				Position:     assignment.Position,
				StartDate:    assignment.StartDate,
				CreatedBy:    assignment.CreatedBy,
				CreatedAt:    assignment.CreatedAt,
				UpdatedAt:    assignment.UpdatedAt,
			})
		}

		// Calculate pagination
		totalPages := (int(total) + pageSize - 1) / pageSize

		if len(assignmentResponses) == 0 {
			pagination := response.Pagination{
				Page:         page,
				PageSize:     pageSize,
				TotalPages:   totalPages,
				TotalRecords: int(total),
			}
			response.SuccessEmptyWithPagination(c, "No project employees found", pagination)
			return
		}

		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}

		message := "Project employees retrieved successfully"
		if page > 1 {
			message = "Page " + strconv.Itoa(page) + " of project employees retrieved successfully"
		}

		response.SuccessWithPagination(c, assignmentResponses, message, pagination)
		return
	}

	// User has full access - use normal pagination at DB level
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.projectEmployeeService.CountAssignments(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	var assignmentResponses []dto.ProjectEmployeeResponse
	for _, assignment := range assignments {
		employeeName := assignment.Employee.Fullname
		if employeeName == "" {
			employeeName = assignment.EmployeeName
		}

		employeeCCCD := assignment.Employee.CCCD
		if employeeCCCD == "" {
			employeeCCCD = assignment.EmployeeCCCD
		}

		assignmentResponses = append(assignmentResponses, dto.ProjectEmployeeResponse{
			ID:           assignment.ID,
			ProjectID:    assignment.ProjectID,
			EmployeeID:   assignment.EmployeeID,
			EmployeeName: employeeName,
			EmployeeCCCD: employeeCCCD,
			EmployeeCode: assignment.EmployeeCode,
			Position:     assignment.Position,
			StartDate:    assignment.StartDate,
			CreatedBy:    assignment.CreatedBy,
			CreatedAt:    assignment.CreatedAt,
			UpdatedAt:    assignment.UpdatedAt,
		})
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(assignmentResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, "No project employees found", pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := "Project employees retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of project employees retrieved successfully"
	}

	response.SuccessWithPagination(c, assignmentResponses, message, pagination)
}
