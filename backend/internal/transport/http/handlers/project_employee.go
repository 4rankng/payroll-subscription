package handlers

import (
	"log/slog"

	projectEmployeeHandler "api-server/internal/transport/http/handlers/project_employee"

	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
)

type ProjectEmployeeHandler struct {
	*projectEmployeeHandler.Handler
}

func NewProjectEmployeeHandler(projectEmployeeService *project.ProjectEmployeeService, employeeService *employee.EmployeeService, projectService *project.ProjectService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService, logger *slog.Logger) *ProjectEmployeeHandler {
	return &ProjectEmployeeHandler{
		Handler: projectEmployeeHandler.NewHandler(projectEmployeeService, employeeService, projectService, projectPermissionService, employeePermissionService, logger),
	}
}
