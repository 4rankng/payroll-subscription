package timesheet

import (
	"api-server/internal/domain"
)

type Operations struct {
	TimesheetRepo  domain.TimesheetRepository
	EmployeeRepo   domain.EmployeeRepository
	ProjectEmpRepo domain.ProjectEmployeeRepository
}

// Check if employee has active project assignments

// Get employee details

// Set employee ID filter

// Get timesheets for the employee

// Calculate summary

// Aggregate hours by paytype

// Track working days

// Count entries by status

// Track last entry date

// Calculate current week hours

// getCurrentWeekHours calculates total hours worked by employee in the current ISO week with comprehensive error handling

// Validate employee ID

// Use optimized database aggregation method

// Log error but don't fail the entire request
// In production, you might want to use a proper logger
