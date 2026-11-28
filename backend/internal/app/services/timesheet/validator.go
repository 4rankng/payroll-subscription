package timesheet

import (
	"time"

	"api-server/internal/domain"
)

type Validator struct {
	TimesheetRepo  domain.TimesheetRepository
	ProjectEmpRepo domain.ProjectEmployeeRepository
	HolidayService HolidayServiceInterface
}

type HolidayServiceInterface interface {
	IsSaturday(date time.Time) bool
	GetDayType(date time.Time) string
}

// DetermineDayType determines the day type based on Vietnam rules
// If user provides DayType, validate and use it; otherwise auto-infer

// If user provided day type, validate and use it

// Auto-infer day type if not provided by user

// Normalize to lowercase for consistency

// ParsePaytype extracts position, day type, and hour type from a paytype string
// Format: "position.dayType.hourType"

// Handle hour types with dots like "08:00-17:00"

// Fallback for old format

// Very old or invalid format

// ValidateEmployeeAssignment checks if employee is assigned to project

// ValidateDailyHoursLimitForUpsert checks if upsert operation would exceed 24 hours per day limit

// First check: single entry cannot exceed 24 hours

// Get all existing timesheets for the employee on the specified date

// Calculate total hours after the upsert operation

// Replace existing hours with new hours (upsert behavior)

// If this is a new record (not an update), add the new hours

// Check if total would exceed 24 hours

// ValidateNoDuplicateEntry checks for duplicate timesheet entry

// ValidateOneProjectPerDay checks if employee already has timesheet for a different project on the same date

// If no existing projects, validation passes

// Check if any existing project is different from the requested project

// ValidateEmployeeWorkPeriod validates that the timesheet date is within employee's work period for the project

// Use the domain method to check if timesheet can be created on this date

// Format date strings for the error message
