package dto

import (
	"time"
)

// BatchAssignEmployeesRequest represents the request to assign employees to a project (batch)
type BatchAssignEmployeesRequest []AssignEmployeeToProjectRequest

// AssignEmployeeToProjectRequest represents a single employee assignment in batch operation
type AssignEmployeeToProjectRequest struct {
	EmployeeID   uint    `json:"employee_id" binding:"required"`
	EmployeeCode string  `json:"employee_code,omitempty"`
	Position     string  `json:"position" binding:"required"`
	StartDate    *string `json:"start_date,omitempty"`
}

// RemoveEmployeeFromProjectRequest represents the request to remove an employee from a project
type RemoveEmployeeFromProjectRequest struct {
	EmployeeID uint `json:"employee_id" binding:"required"`
}

// BatchRemoveEmployeesRequest represents the request to remove employees from a project (batch)
type BatchRemoveEmployeesRequest []RemoveEmployeeFromProjectRequest

// UpdateAssignmentRequest represents the request to update a project-employee assignment
// Used by both PUT /projects/:id/employees and PUT /employees/:id/projects endpoints
type UpdateAssignmentRequest struct {
	ProjectID    uint    `json:"project_id,omitempty"`
	EmployeeID   uint    `json:"employee_id,omitempty"`
	AssignmentID *uint   `json:"assignment_id,omitempty"`
	EmployeeCode *string `json:"employee_code,omitempty"`
	Position     *string `json:"position,omitempty"`
	StartDate    *string `json:"start_date,omitempty"`
}

// ProjectEmployeeAssignmentResponse represents the response for assign/update operations
// Follows API spec - does not include employee_name and employee_cccd
type ProjectEmployeeAssignmentResponse struct {
	ID           uint      `json:"id"`
	ProjectID    uint      `json:"project_id"`
	EmployeeID   uint      `json:"employee_id"`
	EmployeeCode string    `json:"employee_code"`
	Position     string    `json:"position"`
	StartDate    time.Time `json:"start_date"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectEmployeeResponse represents the response for listing employees (GET /projects/:id/employees)
// Includes employee_name and employee_cccd per API spec
type ProjectEmployeeResponse struct {
	ID           uint      `json:"id"`
	ProjectID    uint      `json:"project_id"`
	EmployeeID   uint      `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	EmployeeCCCD string    `json:"employee_cccd"`
	EmployeeCode string    `json:"employee_code"`
	Position     string    `json:"position"`
	StartDate    time.Time `json:"start_date"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
