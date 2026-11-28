package persistence

import (
	"context"

	"api-server/internal/domain"
)

// GetPositionsByProjectEmployee loads the position label of the given
// (project, employee) assignments from project_employees. Assignments not
// found are absent from the result; callers apply the schema default
// ("phổ thông").
func (r *EmployeeRepository) GetPositionsByProjectEmployee(ctx context.Context, pairs []domain.ProjectEmployeePair) (map[domain.ProjectEmployeePair]string, error) {
	return r.projectQueryBuilder.FetchPositionsByProjectEmployee(ctx, pairs)
}
