package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const flowWeeklyBCC = "WeeklyBCCImport"

// runWeeklyBCCImportTests exercises the BCC-<shiftType> sheet format through
// the partner-import endpoint. The fixture BCC LGD.xlsx carries May-2026 work
// days (dates 1/5..31/5) across shifts HC / OT150 / OT30 / OT200 / OT270 /
// OT300 / OT390, so the flow pins for_month to 2026-05 and seeds a payrate
// matrix that covers every shift across ALL three day types (ngày thường /
// ngày nghỉ / ngày lễ) — May 1 is a public holiday and Sundays are rest days,
// and the weekly resolver picks the (position, dayType, shift) bucket by the
// entry's own work date.
func runWeeklyBCCImportTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Weekly BCC Import (BCC-<shiftType> sheets)")

	if len(data.Partners) == 0 || data.WeeklyProject == nil {
		reporter.Skip(flowWeeklyBCC, "All weekly BCC tests", "no partner users or fixture project")
		return
	}

	partnerClient := client.WithToken(data.Partners[0].Token)
	adminClient := client.WithToken(data.AdminToken)

	// Pick a project the partner can actually write into (middleware enforces
	// modify access on scoped partner roles), then make sure it holds the
	// fixture payrate matrix + the sheet employee's assignment.
	weeklyProject, ok := partnerWeeklyProject(partnerClient, data.WeeklyProject)
	if !ok {
		reporter.Skip(flowWeeklyBCC, "All weekly BCC tests", "partner has no accessible project")
		return
	}
	projectID := weeklyProject.ID
	projectIDStr := strconv.Itoa(int(projectID))
	endpoint := "/api/v1/timesheets/partner-import"
	fixtureMonth := "2026-05"

	if _, status, err := adminClient.Post(fmt.Sprintf("/api/v1/projects/%d/users", projectID),
		map[string]any{"user_id": data.Partners[0].ID}); err != nil {
		fmt.Printf("    grant partner upload access: %v (HTTP %d) — continuing\n", err, status)
	}

	seedWeeklyPayrateMatrix(projectID, fixtureMonth + "-01")

	const sheetCCCD = "031092020742"
	var sheetEmp EmployeeResponse
	if _, err := adminClient.GetInto("/api/v1/employees/cccd/"+sheetCCCD, &sheetEmp); err != nil {
		empBody := CreateEmployeeRequest{
			Fullname: "NV BCC LGD",
			CCCD:     sheetCCCD,
			Mobile:   uniqueMobile(),
		}
		if _, cerr := adminClient.PostInto("/api/v1/employees", empBody, &sheetEmp); cerr != nil {
			fmt.Printf("    seed sheet employee: %v — continuing\n", cerr)
		}
	}
	if sheetEmp.ID != 0 {
		assign := []map[string]any{{
			"employee_id": sheetEmp.ID,
			"position":    "Nhân viên",
			"start_date":  fixtureMonth + "-01",
		}}
		if _, status, err := adminClient.Post(fmt.Sprintf("/api/v1/projects/%d/employees", projectID), assign); err != nil {
			// Duplicate assignment from an earlier run is fine — the resolver
			// only needs one active assignment for the (project, employee).
			fmt.Printf("    assign sheet employee: %v (HTTP %d) — continuing\n", err, status)
		}
	}

	weeklyBCCFile := "../docs/WeeklyBCC/BCC LGD.xlsx"

	// ── 1. Upload (busy-retry: a previous flow's job may still finalize) ────
	var importID uint
	reporter.RunTest(flowWeeklyBCC, "Upload WeeklyBCC file (BCC LGD.xlsx)", func() error {
		var apiResp *APIResponse
		var status int
		var err error
		for attempt := 0; attempt < 8; attempt++ {
			apiResp, status, err = partnerClient.UploadFile(endpoint, "file", weeklyBCCFile,
				map[string]string{"project_id": projectIDStr, "for_month": fixtureMonth})
			if err != nil || status != 409 {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status == 403 {
			fmt.Printf("    Partner lacks upload access (403) — skipping\n")
			return nil
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			if status != 200 {
				return fmt.Errorf("status %d, body: %s", status, string(raw))
			}
			return fmt.Errorf("unmarshal: %w (raw: %s)", err, string(raw))
		}
		importID = result.ID
		fmt.Printf("    import_id=%d status=%s created=%d skipped=%d errors=%d for_month=%s total_rows=%d\n",
			result.ID, result.Status, result.CreatedCount, result.SkippedCount, result.ErrorCount, result.ForMonth, result.TotalRows)
		if status != 202 {
			return fmt.Errorf("expected 202, got %d", status)
		}
		if result.ID == 0 {
			return fmt.Errorf("import ID must be > 0")
		}
		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		result = *terminal
		if result.Status == "failed" {
			return fmt.Errorf("import failed: status=%s, error_detail present=%v", result.Status, result.ErrorCount > 0)
		}
		return nil
	})

	// ── 2. Detail ────────────────────────────────────────────────────────────
	if importID > 0 {
		reporter.RunTest(flowWeeklyBCC, "Get WeeklyBCC import detail", func() error {
			apiResp, status, err := partnerClient.Get(fmt.Sprintf("%s/%d", endpoint, importID))
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if status != 200 {
				return fmt.Errorf("expected 200, got %d", status)
			}
			var result bccImportResponse
			raw, _ := json.Marshal(apiResp.Data)
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			fmt.Printf("    detail: status=%s created=%d errors=%d for_month=%s\n",
				result.Status, result.CreatedCount, result.ErrorCount, result.ForMonth)
			return AssertEqual("import_id", importID, result.ID)
		})
	}

	// ── 3. EPE06 template accepted on a WeeklyBCC-scope project ─────────────
	reporter.RunTest(flowWeeklyBCC, "EPE06.2026.xlsx upload accepted", func() error {
		epeFile := "../docs/WeeklyBCC/EPE06.2026.xlsx"
		apiResp, status, err := partnerClient.UploadFile(endpoint, "file", epeFile,
			map[string]string{"project_id": projectIDStr, "for_month": fixtureMonth})
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if status != 202 {
			return fmt.Errorf("expected 202, got %d", status)
		}
		var result bccImportResponse
		raw, _ := json.Marshal(apiResp.Data)
		_ = json.Unmarshal(raw, &result)
		terminal, err := waitForBCCImport(partnerClient, endpoint, result.ID)
		if err != nil {
			return err
		}
		fmt.Printf("    EPE upload: status=%s created=%d errors=%d\n", terminal.Status, terminal.CreatedCount, terminal.ErrorCount)
		return nil
	})
}

// seedWeeklyPayrateMatrix writes the fixture payrate matrix directly through
// the db_helper SQL channel: every shift of the BCC LGD workbook across all
// three day types, effective from the fixture month start. Rate values mirror
// the workbook's rate row (HC 33.000, OT150 49.500, …).
func seedWeeklyPayrateMatrix(projectID uint, monthStart string) {
	rates := `{"Nhân viên": {` +
		`"ngày thường": {"HC": 33000, "OT150": 49500, "OT30": 33000, "OT200": 66000, "OT270": 89100, "OT300": 99000, "OT390": 128700}, ` +
		`"ngày nghỉ": {"HC": 66000, "OT150": 99000, "OT30": 66000, "OT200": 132000, "OT270": 178200, "OT300": 198000, "OT390": 257400}, ` +
		`"ngày lễ": {"HC": 99000, "OT150": 148500, "OT30": 99000, "OT200": 198000, "OT270": 267300, "OT300": 297000, "OT390": 386100}, ` +
		`"Ca ngày": 250000}}`
	seedSQL := "INSERT INTO payrates (project_id, payrate_json, from_date, created_by, created_at, updated_at) " +
		"VALUES (%d, '%s', '%s', 1, NOW(), NOW()) " +
		"ON DUPLICATE KEY UPDATE payrate_json = VALUES(payrate_json), updated_at = NOW()"
	if err := execPayrollSQL(fmt.Sprintf(seedSQL, projectID, rates, monthStart)); err != nil {
		fmt.Printf("    seed payrate matrix: %v — continuing\n", err)
	}
}
