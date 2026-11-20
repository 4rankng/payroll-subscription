package main

import (
	"encoding/json"
	"fmt"

)

const flowProject = "ProjectCRUD"

// projectDateFields captures the removed project date fields; API responses
// must no longer carry them.
type projectDateFields struct {
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

func runProjectCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Project CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testProjectID uint

	// --- Create ---

	reporter.RunTest(flowProject, "Create test project", func() error {
		body := CreateProjectRequest{
			ClientName: prefix + " Client",
			Name:       prefix + " Project",
			Code:       prefix,
		}
		var resp ProjectResponse
		apiResp, err := admin.PostInto("/api/v1/projects", body, &resp)
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		testProjectID = resp.ID
		fmt.Printf("    Created project ID %d\n", resp.ID)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		if err := AssertEqual("code", prefix, resp.Code); err != nil {
			return err
		}
		var dates projectDateFields
		if err := json.Unmarshal(apiResp.Data, &dates); err != nil {
			return fmt.Errorf("decode created project payload: %w", err)
		}
		if dates.StartDate != nil || dates.EndDate != nil {
			return fmt.Errorf("created project must not expose start_date/end_date, got %v/%v", dates.StartDate, dates.EndDate)
		}
		return nil
	})

	defer func() {
		if testProjectID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/projects/%d", testProjectID))
		}
	}()

	// --- Read ---

	reporter.RunTest(flowProject, "Get project by ID", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp ProjectResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d", testProjectID), &resp); err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		if err := AssertEqual("id", testProjectID, resp.ID); err != nil {
			return err
		}
		return AssertHasPrefix("code", resp.Code, prefix)
	})

	reporter.RunTest(flowProject, "List projects", func() error {
		var list []ProjectResponse
		if _, err := admin.GetInto("/api/v1/projects", &list); err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		fmt.Printf("    Found %d projects\n", len(list))
		return AssertSliceMinLen("projects", len(list), 1)
	})

	reporter.RunTest(flowProject, "Get project summary", func() error {
		// Auto-activate first: the suite creates draft projects, and the
		// summary counts active ones only.
		if _, _, err := admin.Post("/api/v1/projects/activate", map[string]any{}); err != nil {
			return fmt.Errorf("activate projects: %w", err)
		}
		var summary ProjectSummaryResponse
		if _, err := admin.GetInto("/api/v1/projects/summary", &summary); err != nil {
			return fmt.Errorf("project summary: %w", err)
		}
		fmt.Printf("    Active Projects: %d\n", summary.TotalActiveProjects)
		return AssertGreaterThan("total_projects", int64(0), summary.TotalActiveProjects)
	})

	// --- Assignment Update Tests (via PUT /employees/:id/projects) ---

	var updateAssignmentID uint
	var updateEmployee EmployeeResponse

	reporter.RunTest(flowProject, "Setup: assign employee for update tests", func() error {
		if testProjectID == 0 || len(data.Employees) < 2 {
			return fmt.Errorf("need test project and at least 2 employees")
		}
		updateEmployee = data.Employees[1]
		req := []map[string]any{
			{
				"employee_id": updateEmployee.ID,
				"position":    "Phổ thông",
				"start_date":  "2026-05-01",
			},
		}
		var resp []ProjectEmployeeResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/employees", testProjectID), req, &resp); err != nil {
			return fmt.Errorf("assign employee for update test: %w", err)
		}
		if len(resp) == 0 {
			return fmt.Errorf("no assignment returned")
		}
		updateAssignmentID = resp[0].ID
		fmt.Printf("    Assigned employee %d for update tests (assignment %d)\n", updateEmployee.ID, updateAssignmentID)
		return nil
	})

	reporter.RunTest(flowProject, "Update assignment position", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		newPos := "Co tay nghe"
		body := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"position":    newPos,
		}
		var resp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body, &resp); err != nil {
			return fmt.Errorf("update position: %w", err)
		}
		if err := AssertEqual("position", newPos, resp.Position); err != nil {
			return err
		}
		fmt.Printf("    Updated position: %s\n", resp.Position)
		return nil
	})

	reporter.RunTest(flowProject, "Update assignment start_date", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		body := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"start_date":  "2026-05-05",
		}
		var resp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body, &resp); err != nil {
			return fmt.Errorf("update start_date: %w", err)
		}
		if err := AssertContains("start_date", resp.StartDate, "2026-05-05"); err != nil {
			return err
		}
		fmt.Printf("    Updated start_date to 2026-05-05\n")
		return nil
	})

	reporter.RunTest(flowProject, "Cleanup: remove update test employee", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return nil
		}
		body := map[string]any{
			"employee_ids": []uint{updateEmployee.ID},
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees/remove", testProjectID), body); err != nil {
			return fmt.Errorf("cleanup remove employee: %w", err)
		}
		fmt.Printf("    Cleaned up employee %d from project %d\n", updateEmployee.ID, testProjectID)
		return nil
	})

	// --- Project Users ---

	reporter.RunTest(flowProject, "Get project users", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var users interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/users", testProjectID), &users); err != nil {
			return fmt.Errorf("get project users: %w", err)
		}
		return nil
	})

	// --- Project Timesheets ---

	reporter.RunTest(flowProject, "Get project timesheets", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/timesheets?pageSize=5", testProjectID), &resp); err != nil {
			return fmt.Errorf("get project timesheets: %w", err)
		}
		return nil
	})

	// --- Project Payrate ---

	reporter.RunTest(flowProject, "Get project payrates", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/payrate", testProjectID), &resp); err != nil {
			// May return 404 if no payrate exists - that's ok
			fmt.Printf("    No payrate for test project (expected): %v\n", err)
			return nil
		}
		return nil
	})

	// --- Delete (blocked while employees are assigned, allowed after) ---

	reporter.RunTest(flowProject, "Delete blocked while employee assigned", func() error {
		if testProjectID == 0 || len(data.Employees) == 0 {
			return fmt.Errorf("missing test project or employees")
		}
		emp := data.Employees[0]
		req := []map[string]any{
			{
				"employee_id": emp.ID,
				"position":    "Nhân viên test",
			},
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees", testProjectID), req); err != nil {
			return fmt.Errorf("assign employee for delete guard: %w", err)
		}
		apiErr, statusCode, err := admin.DeleteExpectError(fmt.Sprintf("/api/v1/projects/%d", testProjectID))
		if err != nil {
			return fmt.Errorf("delete expect error: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 400, statusCode); err != nil {
			return err
		}
		fmt.Printf("    Correctly blocked delete with assigned employee (HTTP %d)\n", statusCode)
		return AssertContains("delete error message", apiErr.Message, "Không thể xóa dự án đang có nhân viên được giao")
	})

	reporter.RunTest(flowProject, "Delete succeeds after removing the employee", func() error {
		if testProjectID == 0 || len(data.Employees) == 0 {
			return fmt.Errorf("missing test project or employees")
		}
		emp := data.Employees[0]
		body := map[string]any{
			"employee_ids": []uint{emp.ID},
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees/remove", testProjectID), body); err != nil {
			return fmt.Errorf("remove employee before delete: %w", err)
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/projects/%d", testProjectID)); err != nil {
			return fmt.Errorf("delete project after unassigning: %w", err)
		}
		testProjectID = 0
		fmt.Printf("    Test project deleted after employee removed\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowProject, "Edge: create project with missing required fields", func() error {
		body := map[string]any{}
		_, statusCode, err := admin.PostExpectError("/api/v1/projects", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowProject, "Edge: get non-existent project", func() error {
		_, statusCode, _ := admin.Get(fmt.Sprintf("/api/v1/projects/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowProject, "Edge: update non-existent project", func() error {
		newName := "Ghost"
		body := UpdateProjectRequest{Name: &newName}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/projects/%d", nonexistentID), body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowProject, "Edge: delete non-existent project", func() error {
		_, statusCode, _ := admin.Delete(fmt.Sprintf("/api/v1/projects/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	// --- Partner Project Creation ---

	// --- Partner Project Update (non-salary fields) ---

	if len(data.Partners) > 0 {
		partner := client.WithToken(data.Partners[0].Token)

		reporter.RunTest(flowProject, "Partner can update non-salary project fields", func() error {
			// Create a project first as the partner
			createBody := CreateProjectRequest{
				ClientName: prefix + " Partner Client",
				Name:       prefix + " Partner Project to Update",
				Code:       prefix + "PTU",
			}
			var created ProjectResponse
			if _, err := partner.PostInto("/api/v1/projects", createBody, &created); err != nil {
				return fmt.Errorf("partner create project for update test: %w", err)
			}
			defer func() {
				_, _, _ = partner.Delete(fmt.Sprintf("/api/v1/projects/%d", created.ID))
			}()

			body := UpdateProjectRequest{Name: strPtrPtr(prefix + " Partner Renamed")}
			var resp ProjectResponse
			_, err := partner.PutInto(fmt.Sprintf("/api/v1/projects/%d", created.ID), body, &resp)
			if err != nil {
				return fmt.Errorf("partner update non-salary field: %w", err)
			}
			if err := AssertEqual("name", prefix+" Partner Renamed", resp.Name); err != nil {
				return err
			}
			return nil
		})
	}

	
	reporter.RunTest(flowProject, "D8: no salary-period keys on the project wire format", func() error {
		// The fixed 4-Kỳ calendar (D8) removed the per-project salary period
		// overrides; the project payload must not carry any salary_period key.
		rows, _, err := rawRows(admin, fmt.Sprintf("/api/v1/projects?pageSize=5"))
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		for i, row := range rows {
			for key := range row {
				if key == "salary_period_from" || key == "salary_period_to" {
					return fmt.Errorf("row %d carries stale key %q — salary periods are removed (D8)", i, key)
				}
			}
		}
		return nil
	})
}

func strPtrPtr(s string) *string { return &s }
