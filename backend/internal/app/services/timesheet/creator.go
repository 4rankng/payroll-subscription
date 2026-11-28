package timesheet

import (
	"api-server/internal/domain"
)

type Creator struct {
	TimesheetRepo domain.TimesheetRepository
	PayrateRepo   domain.PayrateRepository
	Validator     *Validator
}

// CreateTimesheetRequestWithTypes holds timesheet data with separate hour and day types
type CreateTimesheetRequestWithTypes struct {
	Timesheet *domain.Timesheet
	HourType  string
	DayType   *string
}

// CreateTimesheet creates a timesheet with proper day type determination

// Validate employee assignment to project

// Validate that timesheet date is within employee's work period for the project

// Check if project status allows timesheet creation

// Determine day type based on date and rules

// Get position from employee assignment (level 1 of 3-level structure)

// Default position

// Construct the 3-level paytype: position.dayType.hourType (all lowercase for consistency)

// Log paytype construction for debugging

// Check for existing timesheet entry (upsert behavior) - lookup by hour type

// Check if trying to update an approved timesheet as a partner

// Validate one project per day constraint

// Validate daily hours limit (24 hours per day across all hour types)
// For upsert: calculate total after replacing existing hours with new hours

// Get active payrate configuration for the project

// Check if hourType rate is 0 (not allowed for timesheet creation)

// Prevent timesheet creation when hourType has rate of 0

// Set payrate using the 3-level path

// Set initial status and last updated by

// Perform upsert operation

// Update existing record

// Set status based on user role

// Partner submissions require approval

// Return the updated record

// Create new record - set status based on user role

// Partner submissions require approval

// BulkCreateTimesheets creates multiple timesheets in a single transaction
