package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

func loadAllPages[T any](client *APIClient, path string) ([]T, error) {
	endpoint, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("pageSize", "200")
	var records []T
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		var batch []T
		response, err := client.GetInto(endpoint.String(), &batch)
		if err != nil {
			return nil, fmt.Errorf("discover %s page %d: %w", path, page, err)
		}
		records = append(records, batch...)
		if response.Pagination == nil || page >= response.Pagination.TotalPages {
			return records, nil
		}
		if len(batch) == 0 || response.Pagination.Page != page {
			return nil, fmt.Errorf("discover %s: inconsistent pagination at page %d", path, page)
		}
	}
}

// partnerWeeklyProject returns a project the given partner can actually
// upload into. Discovery picks the fixture project globally while partners
// come from config, so the two are not guaranteed to overlap; a partner upload
// to a project outside their scope is refused by design. Prefer the discovered
// project when it is visible, otherwise fall back to any project the partner
// sees, and report false when there is none (caller skips).
func partnerWeeklyProject(client *APIClient, preferred *ProjectResponse) (*ProjectResponse, bool) {
	projects, err := loadAllPages[ProjectResponse](client, "/api/v1/projects")
	if err != nil {
		return preferred, preferred != nil
	}
	if preferred != nil {
		for i := range projects {
			if projects[i].ID == preferred.ID {
				return &projects[i], true
			}
		}
	}
	for i := range projects {
		return &projects[i], true
	}
	return nil, false
}

// bootstrapFixtures creates the minimal operating set on a fresh database
// (consolidated schema.sql deployments start empty): one fixture project, one
// employee, the assignment pairing, a payrate, and project access for every
// configured partner. Idempotent by discovery — the caller only invokes it
// when no projects exist.
func bootstrapFixtures(client *APIClient, data *TestData, cfg *TestConfig) error {
	admin := client.WithToken(data.AdminToken)

	var project ProjectResponse
	createBody := CreateProjectRequest{
		ClientName: "Khách kiểm thử",
		Name:       "Dự án fixture integration",
		Code:       "FIX" + cfg.UniquePrefix(),
	}
	if _, err := admin.PostInto("/api/v1/projects", createBody, &project); err != nil {
		return fmt.Errorf("create fixture project: %w", err)
	}

	// Auto-activate so summary/eligibility surfaces see an active project.
	if _, status, err := admin.Post("/api/v1/projects/activate", map[string]any{}); err != nil {
		return fmt.Errorf("activate fixture projects: %w (HTTP %d)", err, status)
	}

	payrateBody := map[string]any{
		"rates":          json.RawMessage(`{"Nhân viên":{"Ngày thường":{"Ca ngày":250000}}}`),
		"effective_from": today(),
	}
	if _, status, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/payrate", project.ID), payrateBody); err != nil {
		return fmt.Errorf("create fixture payrate: %w (HTTP %d)", err, status)
	}

	empBody := CreateEmployeeRequest{
		Fullname: "NV Fixture " + cfg.UniquePrefix(),
		CCCD:     "99" + cfg.UniquePrefix() + "000001",
		Mobile:   uniqueMobile(),
	}
	var employee EmployeeResponse
	if _, err := admin.PostInto("/api/v1/employees", empBody, &employee); err != nil {
		return fmt.Errorf("create fixture employee: %w", err)
	}

	assignBody := []map[string]any{{
		"employee_id": employee.ID,
		"position":    "Nhân viên",
		"start_date":  today(),
	}}
	if _, status, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees", project.ID), assignBody); err != nil {
		return fmt.Errorf("assign fixture employee: %w (HTTP %d)", err, status)
	}

	for _, p := range data.Partners {
		if _, status, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/users", project.ID),
			map[string]any{"user_id": p.ID}); err != nil {
			return fmt.Errorf("grant partner %d access: %w (HTTP %d)", p.ID, err, status)
		}
	}

	fmt.Printf("  Bootstrapped fixture project %d + employee %d\n", project.ID, employee.ID)
	return nil
}
