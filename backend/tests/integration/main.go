package main

import (
	"api-server/internal/pkg/clock"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// nonexistentID is a stable id far beyond any real row, for not-found
// negative tests across flows.
const nonexistentID = 999_999_999

// uniqueMobile returns a mobile number unique to this run so identity
// resolution never collides with leftover fixtures.
func uniqueMobile() string {
	return fmt.Sprintf("09%08d", time.Now().UnixNano()%100_000_000)
}

// rawRows decodes a list endpoint's data payload as raw JSON objects, for
// wire-format assertions that must not depend on typed models.
func rawRows(c *APIClient, path string) ([]map[string]any, *Pagination, error) {
	resp, statusCode, err := c.Get(path)
	if err != nil {
		return nil, nil, err
	}
	if resp.Status == "error" || statusCode >= 400 {
		return nil, resp.Pagination, fmt.Errorf("API error (HTTP %d): %s", statusCode, resp.Message)
	}
	if len(resp.Data) == 0 {
		return nil, resp.Pagination, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Data, &rows); err != nil {
		return nil, resp.Pagination, fmt.Errorf("decode rows: %w", err)
	}
	return rows, resp.Pagination, nil
}

func main() {
	// Serialize runs before touching any shared state: a second instance
	// blocks here until the first one exits, so two lanes can never pollute
	// each other's fixtures through the shared dev DB and asynq queue.
	lockFile := acquireSuiteLock()
	defer releaseSuiteLock(lockFile)

	cfg := LoadTestConfig()
	reporter := NewReporter()

	// Set up dual output: stdout + timestamped log file
	resultsDir := filepath.Join("tests", "integration", "results")
	_ = os.MkdirAll(resultsDir, 0o755)
	logFileName := fmt.Sprintf("test-result-%s.log", time.Now().Format("20060102-150405"))
	logFile, err := os.Create(filepath.Join(resultsDir, logFileName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create log file: %v\n", err)
	} else {
		defer func() { _ = logFile.Close() }()
		reporter.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	reporter.PrintHeader(cfg.BaseURL)

	client := NewAPIClient(cfg.BaseURL)

	// Reset server clock at start to ensure no state leakage from prior runs.
	// This is a no-op if the clock endpoint is unavailable (e.g., production).
	_ = ResetServerTime(client)

	// Phase 1: Discovery
	reporter.PrintSection("DISCOVERY: Test Data")
	testData, err := discoverTestData(client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  FATAL: discovery failed: %v\n\n", err)
		os.Exit(2)
	}
	reporter.PrintDiscovery(testData)

	// Phase 1.5: Clock manipulation (admin-only, tests time control endpoints)
	runClockManipulationTests(client, reporter)

	// Phase 3: Partner creates timesheets → Admin approves
	runPartnerTimesheetTests(client, testData, reporter)

	// Phase 3.5: Partner BCC Excel import
	runBCCImportTests(client, testData, reporter)

	// Phase 3.6: Weekly BCC (BCC-<shiftType> sheet format)
	runWeeklyBCCImportTests(client, testData, reporter)

	// Phase 7: Auth & User Management
	runAuthUserTests(client, testData, reporter, cfg)

	// Phase 7.4: Server-side role enforcement pin + cookie channel auth.
	runRoleEnforcementTests(client, testData, reporter, cfg)

	// Phase 7.5: Self-service password reset (no-token contracts; see flow file)
	runPasswordResetTests(client, testData, reporter, cfg)

	// Phase 7.55: Admin/partner employee password reset (the previously
	// untested PUT /employees/:id/change-password endpoint)
	runEmployeePasswordAdminTests(client, testData, reporter, cfg)

	// Phase 7.6: import-mobiles RBAC (admin|quanly pass, others 403)
	runEmployeeImportMobilesTests(client, testData, reporter, cfg)

	// Phase 7.65: "NV chưa có SĐT" template export (the workbook source half)
	runEmployeeMobileTemplateTests(client, testData, reporter, cfg)

	// Phase 8: CRUD flows
	runEmployeeCRUDTests(client, testData, reporter, cfg)
	runProjectCRUDTests(client, testData, reporter, cfg)
	runProjectPartnerSummaryTests(client, testData, reporter, cfg)
	runProjectUserAccessTests(client, testData, reporter, cfg)

	// Phase 9: Domain flows
	runPayrateCRUDTests(client, testData, reporter, cfg)
	runPayratePartnerVisibilityTests(client, testData, reporter, cfg)
	runSettingsTests(client, testData, reporter, cfg)
	runAssetTests(client, testData, reporter, cfg)

	// Phase 9.5: PAYSUB billing — upload verification read model + FR-6 surfaces
	runBillingTests(client, testData, reporter)

	// Phase 9.6: sao kê — Sổ cái export/recipients/email
	runSaoKeTests(client, testData, reporter)

	// Phase 10: Read-only flows
	runNotificationTests(client, testData, reporter, cfg)
	runDashboardTests(client, testData, reporter, cfg)
	runAuditTests(client, testData, reporter, cfg)
	runMetricsTests(client, testData, reporter, cfg)

	// Phase 11: Infrastructure invariants (schema, asynq migration guard)
	runInfraTests(reporter)

	// Phase 12: Extended flows
	runEmployeeSelfServiceTests(client, testData, reporter, cfg)
	runTimesheetExtendedTests(client, testData, reporter, cfg)

	// Summary
	reporter.PrintSummary()
	os.Exit(reporter.ExitCode())
}

func discoverTestData(client *APIClient, cfg *TestConfig) (*TestData, error) {
	data := &TestData{}

	// Login as admin
	loginResp, err := client.Login(cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("admin login: %w", err)
	}
	data.AdminToken = loginResp.AccessToken
	fmt.Printf("  Logged in as admin: %s (ID %d)\n", loginResp.User.Username, loginResp.User.ID)

	// Fetch projects
	if data.Projects, err = loadAllPages[ProjectResponse](client, "/api/v1/projects"); err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	fmt.Printf("  Found %d projects\n", len(data.Projects))

	// Fetch employees
	if data.Employees, err = loadAllPages[EmployeeResponse](client, "/api/v1/employees"); err != nil {
		return nil, fmt.Errorf("fetch employees: %w", err)
	}
	fmt.Printf("  Found %d employees\n", len(data.Employees))

	// Find an employee user account for self-service flows (nhanvien token).
	if err := findEmployeeForAdvance(client, data, cfg); err != nil {
		fmt.Printf("  Warning: could not find employee user account: %v\n", err)
	}

	// Find partner users
	findPartners(client, data, cfg)

	// Fresh-database bootstrap: the consolidated schema ships no data, so the
	// fixture set is created on the first run against an empty deployment.
	if len(data.Projects) == 0 {
		if err := bootstrapFixtures(client, data, cfg); err != nil {
			fmt.Printf("  Warning: fixture bootstrap failed: %v\n", err)
		}
		if data.Projects, err = loadAllPages[ProjectResponse](client, "/api/v1/projects"); err != nil {
			return nil, fmt.Errorf("re-fetch projects: %w", err)
		}
		if data.Employees, err = loadAllPages[EmployeeResponse](client, "/api/v1/employees"); err != nil {
			return nil, fmt.Errorf("re-fetch employees: %w", err)
		}
	}

	pickWeeklyFixture(data)

	// Discover payrate hour/day types for the fixture project (after bootstrap
	// so a fresh database has its payrate too).
	if data.WeeklyProject != nil {
		discoverPayrateTypes(client, data)
	}

	return data, nil
}

// pickWeeklyFixture selects the fixture project and its assigned employee
// after the project/employee lists are final.
func pickWeeklyFixture(data *TestData) {
	if len(data.Projects) > 0 {
		data.WeeklyProject = &data.Projects[0]
	}
	for i := range data.Employees {
		emp := &data.Employees[i]
		if data.WeeklyProject != nil {
			for _, proj := range emp.CurrentProjects {
				if proj.ProjectID == data.WeeklyProject.ID && data.WeeklyEmployee == nil {
					data.WeeklyEmployee = emp
					projCopy := proj
					data.WeeklyAssignment = &projCopy
				}
			}
		}
		if data.WeeklyEmployee != nil {
			break
		}
	}
	if data.WeeklyEmployee == nil && len(data.Employees) > 0 {
		data.WeeklyEmployee = &data.Employees[0]
	}
}

func findEmployeeForAdvance(client *APIClient, data *TestData, cfg *TestConfig) error {
	// Name kept for the two consumers (self-service + removal-visibility flows)
	// that predate the PAYSUB rebuild: find any employee with a user account
	// so the flows can log in as nhanvien.
	for _, emp := range data.Employees {
		var detail EmployeeDetailedResponse
		if _, err := client.GetInto(fmt.Sprintf("/api/v1/employees/%d", emp.ID), &detail); err != nil {
			continue
		}
		if detail.Username == nil || *detail.Username == "" {
			continue
		}

		empClient := NewAPIClient(client.BaseURL)
		loginResp, err := empClient.Login(*detail.Username, cfg.CommonPassword)
		if err != nil {
			continue
		}

		data.EmployeeForAdvance = &detail
		data.EmployeeTokenForAdv = loginResp.AccessToken
		fmt.Printf("  Found employee user account: %s (username: %s)\n",
			emp.Fullname, *detail.Username)
		return nil
	}

	return fmt.Errorf("no employee user accounts found")
}

func findPartners(client *APIClient, data *TestData, cfg *TestConfig) {
	for _, username := range cfg.PartnerUsernames {
		partnerClient := NewAPIClient(client.BaseURL)
		loginResp, err := partnerClient.Login(username, cfg.CommonPassword)
		if err != nil {
			fmt.Printf("  Could not login as partner %s: %v\n", username, err)
			continue
		}
		data.Partners = append(data.Partners, PartnerUser{
			ID:       loginResp.User.ID,
			Username: username,
			Fullname: loginResp.User.Fullname,
			Token:    loginResp.AccessToken,
		})
		fmt.Printf("  Found partner: %s (%s)\n", loginResp.User.Fullname, username)
	}
}

func today() string {
	return clock.Now().Format("2006-01-02")
}

// discoverPayrateTypes fetches the current payrate config for the weekly project
// and extracts the first available hour type and day type with a non-zero rate.
func discoverPayrateTypes(client *APIClient, data *TestData) {
	var payrates []PayrateResponse
	if _, err := client.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", data.WeeklyProject.ID), &payrates); err != nil {
		fmt.Printf("  Warning: could not fetch payrate for project %d: %v\n", data.WeeklyProject.ID, err)
		return
	}
	if len(payrates) == 0 {
		return
	}

	var rates map[string]map[string]map[string]float64
	if err := json.Unmarshal(payrates[0].Rates, &rates); err != nil {
		return
	}
	for _, dayTypes := range rates {
		for dayType, hourTypes := range dayTypes {
			for hourType, rate := range hourTypes {
				if rate > 0 {
					data.WeeklyDayType = dayType
					data.WeeklyHourType = hourType
					fmt.Printf("  Weekly project payrate: dayType=%q hourType=%q rate=%.0f\n", dayType, hourType, rate)
					return
				}
			}
		}
	}
}

func weekAgo() string {
	return clock.Now().AddDate(0, 0, -7).Format("2006-01-02")
}

func currentMonth() string {
	return time.Now().Format("2006-01")
}

// cleanupEmployeeTimesheets deletes non-paid timesheets for a given employee in a project.
// This ensures test idempotency by removing leftover approved-but-unpaid timesheets from prior runs.
func cleanupEmployeeTimesheets(client *APIClient, projectID, employeeID uint) {
	path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&employee_id=%d&pageSize=200",
		projectID, employeeID)
	var timesheets []TimesheetWithDetailsResponse
	if _, err := client.GetInto(path, &timesheets); err != nil {
		return
	}
	for _, ts := range timesheets {
		if ts.PaymentStatus == "paid" {
			continue
		}
		deletePath := fmt.Sprintf("/api/v1/timesheets/%d", ts.ID)
		_, _, _ = client.Delete(deletePath)
	}
}

func findAvailableDateWithMin(client *APIClient, projectID, employeeID uint, start time.Time, minDate *time.Time) string {
	// Get ALL timesheets for this employee in the project (no date filter due to timezone issues)
	path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&employee_id=%d&pageSize=200",
		projectID, employeeID)
	var timesheets []TimesheetWithDetailsResponse
	if _, err := client.GetInto(path, &timesheets); err != nil {
		return ""
	}

	// Build set of occupied dates (only non-deleted, since deleted timesheets free up the date)
	occupied := make(map[string]bool)
	for _, ts := range timesheets {
		occupied[ts.Date] = true
	}

	// Search backward from start, respecting minDate
	for i := 0; i < 365; i++ {
		d := start.AddDate(0, 0, -i)
		if d.Weekday() == time.Sunday {
			continue
		}
		if minDate != nil && d.Before(*minDate) {
			break
		}
		dateStr := d.Format("2006-01-02")
		if !occupied[dateStr] {
			return dateStr
		}
	}

	// Search forward (but not into the future)
	today := time.Now()
	for i := 1; i < 365; i++ {
		d := start.AddDate(0, 0, i)
		if d.After(today) {
			break
		}
		if d.Weekday() == time.Sunday {
			continue
		}
		dateStr := d.Format("2006-01-02")
		if !occupied[dateStr] {
			return dateStr
		}
	}
	return ""
}
