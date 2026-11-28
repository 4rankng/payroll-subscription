package employee

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

func buildProjectInfo(project domain.CurrentProject) dto.EmployeeProjectInfo {
	startDate := ""
	if !project.StartDate.IsZero() {
		startDate = project.StartDate.Format("2006-01-02")
	}

	return dto.EmployeeProjectInfo{
		ProjectID:         project.ProjectID,
		ProjectEmployeeID: project.ProjectEmployeeID,
		Name:              project.Name,
		Code:              project.Code,
		ClientName:        project.ClientName,
		Position:          project.Position,
		StartDate:         startDate,
	}
}

// allowedEmployeeSortFields is a set of valid sort fields to prevent SQL injection.
var allowedEmployeeSortFields = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "fullname": {}, "email": {}, "cccd": {},
	"mobile": {},
}

// parseEmployeeFilters builds EmployeeFilters from query params, applying pagination,
// sorting, search, date range, and role-based access control.
func (h *Handler) parseEmployeeFilters(c *gin.Context) (domain.EmployeeFilters, int, int, bool) {
	pg := helpers.ParsePagination(c, 100)

	filters := domain.EmployeeFilters{
		Limit:     pg.Limit,
		Offset:    pg.Offset,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Support both camelCase and snake_case sort params
	for _, key := range []string{"sortBy", "sort_by"} {
		if v := c.Query(key); v != "" {
			if _, ok := allowedEmployeeSortFields[v]; ok {
				filters.SortBy = v
				break
			}
		}
	}
	for _, key := range []string{"sortOrder", "sort_order"} {
		if v := c.Query(key); v != "" {
			switch v {
			case "asc", "desc", "ASC", "DESC":
				filters.SortOrder = v
			}
			break
		}
	}

	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	if fromDate := c.Query("fromDate"); fromDate != "" {
		if parsed, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return filters, 0, 0, false
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if parsed, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return filters, 0, 0, false
		}
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RoleQuanly) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return filters, 0, 0, false
		}
		uid, ok := userID.(uint)
		if !ok {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return filters, 0, 0, false
		}
		filters.AccessibleBy = &uid
	}

	return filters, pg.Page, pg.PageSize, true
}

// buildEmployeeListResponse converts a slice of EmployeeWithProjects to response DTOs.
// When maskInaccessible is true (global-pool listing for a partner), rows outside
// the requester's accessible set are annotated is_accessible=false and their
// bank/contact details are withheld: a preview row only carries identity.
func buildEmployeeListResponse(employees []*domain.EmployeeWithProjects, accessibleIDs map[uint]bool, maskInaccessible bool, claimedIDs map[uint]bool) []dto.EmployeeResponse {
	result := make([]dto.EmployeeResponse, 0, len(employees))
	for _, emp := range employees {
		r := dto.EmployeeResponse{
			ID:                emp.ID,
			Fullname:          emp.Fullname,
			Email:             emp.Email,
			CCCD:              emp.CCCD,
			Address:           emp.Address,
			Mobile:            emp.Mobile,
			Bank:              buildEmployeeBankInfo(emp.Bank),
			BankAccountNumber: emp.BankAccountNumber,
			BankAccountName:   emp.BankAccountName,
			DateOfBirth:       formatOptionalDate(emp.DateOfBirth),
			CreatedBy:         emp.CreatedBy,
			CreatedAt:         emp.CreatedAt,
			UpdatedAt:         emp.UpdatedAt,
			CanDelete:         true,
			CurrentProjects:   make([]dto.EmployeeProjectInfo, 0, len(emp.CurrentProjects)),
		}

		// Global-pool annotation: flag accessibility for rows the partner does
		// not manage (cross-partner guard) and strip the details a preview is
		// not entitled to.
		if maskInaccessible {
			accessible := accessibleIDs[emp.ID]
			r.IsAccessible = &accessible
			creatorName := emp.Creator.Fullname
			r.CreatorName = &creatorName
			if !accessible {
				r.Address = ""
				r.DateOfBirth = nil
				r.Bank = nil
				r.BankAccountNumber = ""
				r.BankAccountName = ""
			}
		}

		// Scoped-partner annotation: flag rows the partner manages via an
		// employee_users claim — the only rows the self-service remove action
		// applies to (created and project-visible rows cannot leave the list
		// through a claim release).
		if claimedIDs != nil {
			claimed := claimedIDs[emp.ID]
			r.IsClaimed = &claimed
		}

		for _, p := range emp.CurrentProjects {
			r.CurrentProjects = append(r.CurrentProjects, buildProjectInfo(p))
		}
		result = append(result, r)
	}
	return result
}

// ListEmployees lists all employees with pagination and filtering
// @Summary List employees
// @Description Get a paginated list of employees with filtering options. Inactive employees are not returned by default.
// @Tags employees
// @Accept json
// @Produce json
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param status query string false "Filter by assignment status: working (assigned to projects) or unassigned"
// @Param search query string false "Search by name, email, or CCCD"
// @Param projectId query int false "Filter employees assigned to specific project"
// @Param fromDate query string false "Filter employees created from this date (YYYY-MM-DD)"
// @Param toDate query string false "Filter employees created until this date (YYYY-MM-DD)"
// @Param scope query string false "Partner-only: 'global' lists ALL employees (global pool); default lists only accessible employees"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees [get]
func (h *Handler) ListEmployees(c *gin.Context) {
	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	// Partner global-pool scope: list all employees, annotated with is_accessible,
	// with bank/contact details withheld for rows the partner does not manage.
	// Only this endpoint honors scope=global — export/unassigned/summary stay scoped.
	globalPool := false
	if c.Query("scope") == "global" && c.GetString(constants.CtxUserRole) == string(domain.RoleQuanly) {
		globalPool = true
		filters.AccessibleBy = nil
	}

	if status := c.Query("status"); status != "" {
		if status != "working" && status != "unassigned" {
			response.BadRequest(c, constants.MsgInvalidEmployeeStatusFilterVN)
			return
		}
		filters.Status = status
	}

	if projectID := c.Query("projectId"); projectID != "" {
		if id, err := strconv.ParseUint(projectID, 10, 64); err == nil {
			uid := uint(id)
			filters.ProjectID = &uid
		}
	}

	employees, err := h.employeeService.ListEmployeesWithAllProjects(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.employeeService.CountEmployees(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	// For the global pool, resolve the partner's accessible set to annotate rows.
	accessibleIDs := map[uint]bool{}
	if globalPool {
		uid, err := h.validator.ValidateUserContext(c)
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
		ids, err := h.employeeService.ListAccessibleIDs(c.Request.Context(), uid)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}
		for _, id := range ids {
			accessibleIDs[id] = true
		}
	}

	// For the scoped partner list, resolve the partner's claim set so rows can
	// carry is_claimed (drives the self-service remove affordance).
	claimedIDs := map[uint]bool{}
	if !globalPool && h.employeePermissionService != nil {
		uid, err := h.validator.ValidateUserContext(c)
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
		ids, err := h.employeePermissionService.ListClaimedEmployeeIDs(c.Request.Context(), uid)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
			return
		}
		for _, id := range ids {
			claimedIDs[id] = true
		}
	}

	employeeResponses := buildEmployeeListResponse(employees, accessibleIDs, globalPool, claimedIDs)
	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No records found", pagination)
		return
	}

	message := constants.MsgEmployeesRetrievedSuccessfully
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of employees retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}

// GetEmployeesMissingMobile retrieves employees with no mobile number on file
// @Summary Get employees with missing mobile numbers
// @Description Get employees who have no phone number on file — the data-quality nudge list (phone login, Zalo OTP password reset and self check-in all key off the number)
// @Tags employees
// @Accept json
// @Produce json
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param search query string false "Search by name, email, or CCCD"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/missing-mobile [get]
func (h *Handler) GetEmployeesMissingMobile(c *gin.Context) {
	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	employees, err := h.employeeService.GetEmployeesWithMissingMobile(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	total, err := h.employeeService.CountEmployeesWithMissingMobile(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountEmployeesVN)
		return
	}

	employeeResponses := buildEmployeeListResponse(employees, nil, false, nil)
	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No records found", pagination)
		return
	}

	message := "Employees with missing mobile numbers retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of employees with missing mobile numbers retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}

// GetUnassignedEmployees retrieves employees not assigned to any project at a given date
// @Summary Get unassigned employees
// @Description Get employees not assigned to any project at a specific date
// @Tags employees
// @Accept json
// @Produce json
// @Param at_date query string false "Date to check assignments (YYYY-MM-DD format, defaults to today)" default(today)
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(100)
// @Param sortBy query string false "Field to sort by" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param search query string false "Search by name, email, or CCCD"
// @Success 200 {object} dto.ListEmployeesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/unassigned [get]
func (h *Handler) GetUnassignedEmployees(c *gin.Context) {
	atDate := h.clock.Now()
	if atDateStr := c.Query("at_date"); atDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", atDateStr); err == nil {
			atDate = parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidAtDateFormatVN)
			return
		}
	}

	filters, page, pageSize, ok := h.parseEmployeeFilters(c)
	if !ok {
		return
	}

	employees, err := h.employeeService.GetUnassignedEmployeesAtDate(c.Request.Context(), atDate, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveUnassignedEmployeesVN)
		return
	}

	total, err := h.employeeService.CountUnassignedEmployeesAtDate(c.Request.Context(), atDate, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountUnassignedEmployeesVN)
		return
	}

	var employeeResponses []dto.EmployeeResponse
	for _, emp := range employees {
		employeeResponses = append(employeeResponses, h.buildEmployeeResponse(c.Request.Context(), emp))
	}

	pagination := helpers.CalculatePagination(page, pageSize, total)

	if len(employeeResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "No unassigned employees found", pagination)
		return
	}

	message := "Unassigned employees retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of unassigned employees retrieved successfully"
	}
	response.SuccessWithPagination(c, employeeResponses, message, pagination)
}
