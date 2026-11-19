package handlers

import (
	timesheetHandler "api-server/internal/transport/http/handlers/timesheet"

	"api-server/internal/app/services/config"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
)

type TimesheetHandler struct {
	*timesheetHandler.Handler
}

func NewTimesheetHandler(
	timesheetService *timesheet.TimesheetService,
	settingsConfigService *config.SettingsConfigService,
	projectService *project.ProjectService,
	projectEmployeeService *project.ProjectEmployeeService,
	payrateService *payroll.PayrateService,
	projectPermissionService *project.ProjectPermissionService,
	employeePermissionService *employee.EmployeePermissionService,
	auditService interface{},
	clk clock.Clock,
) *TimesheetHandler {
	return &TimesheetHandler{
		Handler: timesheetHandler.NewHandler(
			timesheetService,
			settingsConfigService,
			projectService,
			projectEmployeeService,
			payrateService,
			projectPermissionService,
			employeePermissionService,
			auditService,
			clk,
		),
	}
}
