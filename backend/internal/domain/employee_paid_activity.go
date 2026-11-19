package domain

import "time"

// EmployeePaidActivity is one row of the /employees/export-paid-without-mobile
// export: an employee who received salary within the query window but has no
// mobile number on file.
type EmployeePaidActivity struct {
	EmployeeID       uint       `json:"employee_id"`
	Fullname         string     `json:"fullname"`
	CCCD             string     `json:"cccd"`
	IsWorking        bool       `json:"is_working"`
	SalaryTotal      int64      `json:"salary_total"`
	SalaryCount      int64      `json:"salary_count"`
	LastSalaryPaidAt *time.Time `json:"last_salary_paid_at"`
	LastActivityAt   *time.Time `json:"last_activity_at"`
	Projects         string     `json:"projects"`
}
