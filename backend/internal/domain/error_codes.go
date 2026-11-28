package domain

// Assignment-related error codes
const (
	ErrCodeAssignmentOverlap    = "ASSIGNMENT_OVERLAP"
	ErrCodeAssignmentReplaced   = "ASSIGNMENT_REPLACED"
	ErrCodeAssignmentTerminated = "ASSIGNMENT_TERMINATED"
	ErrCodeTimesheetExists      = "TIMESHEET_EXISTS"
	// ErrCodeMultipleProjectsInCycle refuses a transfer batch that would pay one
	// employee for two projects in the same cycle (two rows, same bank account).
	ErrCodeMultipleProjectsInCycle = "MULTIPLE_PROJECTS_IN_CYCLE"
	ErrCodeInvalidDateRange        = "INVALID_DATE_RANGE"
	ErrCodeInvalidTransition       = "INVALID_TRANSITION"
	ErrCodeConflictResolution      = "CONFLICT_RESOLUTION_FAILED"

	// Payrate timeline codes (from-date-only model)
	ErrCodePayrateNotLatest       = "PAYRATE_NOT_LATEST"
	ErrCodePayrateImmutableLinked = "PAYRATE_IMMUTABLE_LINKED"
)

// NewValidationErrorWithCode creates a validation error with an error code
func NewValidationErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "VALIDATION_ERROR",
		Code:    code,
		Message: message,
		Cause:   ErrValidation,
	}
}

// NewConflictErrorWithCode creates a conflict error with an error code

// Common error builders for timesheet domain
