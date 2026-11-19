package dto

import (
	"time"
)

type RecentActivitiesResponse []ActivityItem

// ActivityItem represents a single system activity in the dashboard feed
type ActivityItem struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// SystemNotificationsResponse represents system notifications
type SystemNotificationsResponse struct {
	Notifications []NotificationItem `json:"notifications"`
	UnreadCount   int                `json:"unread_count"`
}

// NotificationItem represents a single system notification
type NotificationItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	ActionURL *string   `json:"action_url"`
	CreatedAt time.Time `json:"created_at"`
	Icon      string    `json:"icon"`
}

// NewEmployeesResponse represents recently added employees
type NewEmployeesResponse []NewEmployeeItem

// NewEmployeeItem represents a recently added employee
type NewEmployeeItem struct {
	ID             int          `json:"id"`
	Fullname       string       `json:"fullname"`
	Email          *string      `json:"email"`
	CCCD           string       `json:"cccd"`
	CurrentProject *ProjectInfo `json:"current_project"`
	CreatedAt      time.Time    `json:"created_at"`
	CreatedByName  string       `json:"created_by_name"`
	Avatar         *string      `json:"avatar"`
}

// ProjectInfo represents basic project information
type ProjectInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// RecentActivitiesRequest represents query parameters for activities
type RecentActivitiesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"pageSize,default=10" binding:"min=1,max=100"`
	SortBy    string `form:"sortBy,default=created_at"`
	SortOrder string `form:"sortOrder,default=desc"`
}

// SystemNotificationsRequest represents query parameters for notifications
type SystemNotificationsRequest struct {
	Page        int  `form:"page,default=1" binding:"min=1"`
	PageSize    int  `form:"pageSize,default=5" binding:"min=1,max=20"`
	IncludeRead bool `form:"include_read,default=false"`
}

// NewEmployeesRequest represents query parameters for new employees
type NewEmployeesRequest struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"pageSize,default=5" binding:"min=1,max=20"`
	Days     int `form:"days,default=30" binding:"min=1"`
}

// DashboardSummaryRequest represents query parameters for the dashboard summary
type DashboardSummaryRequest struct {
	Month *string `form:"month" binding:"omitempty" time_format:"2006-01"` // Format: YYYY-MM (e.g., "2025-04")
}

// DashboardSummaryResponse represents the dashboard summary data
type DashboardSummaryResponse struct {
	// TotalEmployees is the count of employees currently assigned to projects
	// (excludes employees without active project assignments)
	TotalEmployees int `json:"total_employees"`
	// TotalWorkingEmployees is the count of distinct employees with a paid
	// timesheet in the last 30 days
	TotalWorkingEmployees int `json:"total_working_employees"`
	// EmployeesHiredThisMonth counts distinct employees whose project assignment
	// start_date falls within the current month (based on project_employees),
	// not employees created this month.
	EmployeesHiredThisMonth int   `json:"employees_hired_this_month"`
	TotalPaidSalary         int64 `json:"total_paid_salary"`
	PendingSalaryThisMonth  int64 `json:"pending_salary_this_month"`
	PaidSalaryThisMonth     int64 `json:"paid_salary_this_month"`
}

// TopPaidEmployeeItem represents a single employee in the top paid list
type TopPaidEmployeeItem struct {
	Rank              int    `json:"rank"`
	EmployeeID        uint   `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	TotalPaidVND      int64  `json:"total_paid_vnd"`
	IsActive          bool   `json:"is_active"` // true if has timesheet in last 14 days
	LastTimesheetDate string `json:"last_timesheet_date,omitempty"`
}

// TopPaidEmployeesRequest represents query parameters for top paid employees
type TopPaidEmployeesRequest struct {
	Month string `form:"month"` // YYYY-MM, optional; if empty returns all-time
	Limit int    `form:"limit,default=10" binding:"min=1,max=50"`
}

// TopPaidEmployeesResponse represents the top paid employees response
type TopPaidEmployeesResponse struct {
	Employees []TopPaidEmployeeItem `json:"employees"`
	Period    string                `json:"period"` // "all_time" or "YYYY-MM"
}
