package validation

import (
	"strconv"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// RequestValidator provides standardized request validation utilities
type RequestValidator struct{}

// NewRequestValidator creates a new request validator
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{}
}

// ValidateIDParam validates and parses ID from URL parameter
func (v *RequestValidator) ValidateIDParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	if idStr == "" {
		return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		switch paramName {
		case "id":
			return 0, domain.NewValidationError(constants.MsgInvalidEmployeeIDVN)
		case "project_id":
			return 0, domain.NewValidationError(constants.MsgInvalidProjectIDVN)
		case "payrate_id":
			return 0, domain.NewValidationError(constants.MsgInvalidPayrateIDVN)
		default:
			return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
		}
	}

	if id == 0 {
		return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
	}

	return uint(id), nil
}

// ValidateUserContext extracts and validates user ID from context
func (v *RequestValidator) ValidateUserContext(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, domain.NewForbiddenError(constants.MsgUserIDNotFoundInContextVN)
	}

	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		return 0, domain.NewForbiddenError(constants.MsgInvalidUserIDVN)
	}

	return uid, nil
}

// ValidateDateParam validates and parses date parameter

// Optional parameter

// ValidateRequiredDateParam validates required date parameter

// ValidatePaginationParams validates pagination query parameters

// Default values

// Parse limit

// Max limit of 1000

// Parse offset

// ValidateSearchQuery validates search query parameters

// Alternative parameter name

// ValidateOptionalSearchQuery validates optional search query parameters

// Alternative parameter name

// ValidateSortParams validates sorting parameters

// Validate sort_by

// Validate sort_order
