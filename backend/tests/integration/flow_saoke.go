package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const flowSaoKe = "SaoKe"

// Dev-seed role credentials (seed.EnsureDevUsers, non-production only): the
// sao kê surface belongs to ketoan + giamdoc (PRD §5.8, casbin policy).
const (
	saoKeKetoanUser  = "ketoan"
	saoKeGiamdocUser = "giamdoc"
	saoKeDevPassword = "Abc123"
	saoKeTestEmail   = "saoke-it@example.com"
	saoKeTestCC      = "giamdoc-it@example.com"
	saoKePaidAmount  = 800000
)

// saoKeRecipientsPayload is the GET/PUT /billing/saoke/recipients body.
type saoKeRecipientsPayload struct {
	Recipients []string `json:"recipients"`
	CC         []string `json:"cc"`
}

// saoKeUploadRow is one row of GET /billing/uploads with the matched_amount
// enrichment (Sổ cái "Tổng tiền").
type saoKeUploadRow struct {
	ID            uint64 `json:"id"`
	Cycle         int    `json:"cycle"`
	WorkMonth     string `json:"work_month"`
	Status        string `json:"status"`
	MatchedRows   int    `json:"matched_rows"`
	MatchedAmount int64  `json:"matched_amount"`
}

type saoKeUploadsPayload struct {
	Items []saoKeUploadRow `json:"items"`
}

type saoKeEmailPayload struct {
	ReportDate string   `json:"report_date"`
	Recipients []string `json:"recipients"`
	CC         []string `json:"cc"`
}

// runSaoKeTests covers the PAYR-41 sao kê surface: recipient persistence,
// the Excel export and the statement email, plus the uploads-list
// matched_amount enrichment the Sổ cái page renders.
func runSaoKeTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: PAYR-41 Sao kê (Sổ cái)")

	ketoan, err := loginSaoKeRole(client, saoKeKetoanUser)
	if err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("ketoan login: %v", err))
		return
	}
	giamdoc, err := loginSaoKeRole(client, saoKeGiamdocUser)
	if err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("giamdoc login: %v", err))
		return
	}

	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowSaoKe, "All tests", "no fixture project/employee discovered")
		return
	}
	admin := client.WithToken(data.AdminToken)
	projectID := data.WeeklyProject.ID
	employeeID := data.WeeklyEmployee.ID

	// Kỳ 1 of the current work month: work day 03, pay date day 10.
	thisMonth := time.Now().Format("2006-01")
	workDate := thisMonth + "-03"
	reportDate := thisMonth + "-10"
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())

	cleanupSaoKeFixtures(projectID, employeeID, stamp)
	defer cleanupSaoKeFixtures(projectID, employeeID, stamp)

	// ── Seed: one paid timesheet in the Kỳ 1 window + verified upload + ────
	// employee_pay_cycles row — exactly what upload verification produces.
	if err := plantPaidTimesheet(projectID, employeeID, workDate); err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("plant timesheet: %v", err))
		return
	}
	// The statement window is payment_date/paid_at — stamp the payment onto
	// the Kỳ 1 pay date (day 10).
	if err := execPayrollSQL(fmt.Sprintf(
		"UPDATE timesheets SET payment_date = '%s', paid_at = NOW() WHERE project_id = %d AND employee_id = %d AND date = '%s'",
		reportDate, projectID, employeeID, workDate)); err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("stamp payment date: %v", err))
		return
	}
	tsID, err := queryPayrollScalar(fmt.Sprintf(
		"SELECT id FROM timesheets WHERE project_id = %d AND employee_id = %d AND date = '%s' LIMIT 1",
		projectID, employeeID, workDate))
	if err != nil || tsID == "" {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("resolve planted timesheet: err=%v id=%q", err, tsID))
		return
	}
	uploaderID, err := queryPayrollScalar("SELECT id FROM users ORDER BY id LIMIT 1")
	if err != nil || uploaderID == "" {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("resolve uploader user: %v", err))
		return
	}
	if err := execPayrollSQL(fmt.Sprintf(
		"INSERT INTO transfer_uploads (cycle, work_month, file_hash, original_filename, uploaded_by, status, parsed_rows, matched_rows, verified_at, created_at, updated_at) "+
			"VALUES (1, '%s', 'it-saoke-%s', 'it-saoke-%s.xlsx', %s, 'verified', 1, 1, NOW(), NOW(), NOW())",
		thisMonth, stamp, stamp, uploaderID)); err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("plant transfer upload: %v", err))
		return
	}
	uploadID, err := queryPayrollScalar(fmt.Sprintf(
		"SELECT id FROM transfer_uploads WHERE file_hash = 'it-saoke-%s'", stamp))
	if err != nil || uploadID == "" {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("resolve upload id: %v", err))
		return
	}
	if err := execPayrollSQL(fmt.Sprintf(
		"INSERT INTO employee_pay_cycles (cycle, work_month, employee_id, project_id, timesheet_id, source_upload_id, paid_amount, created_at) "+
			"VALUES (1, '%s', %d, %d, %s, %s, %d, NOW())",
		thisMonth, employeeID, projectID, tsID, uploadID, saoKePaidAmount)); err != nil {
		reporter.Skip(flowSaoKe, "All tests", fmt.Sprintf("plant employee_pay_cycles: %v", err))
		return
	}

	// ── Recipients PUT→GET roundtrip (ketoan) ──────────────────────────────
	reporter.RunTest(flowSaoKe, "Recipients roundtrip persists To and CC lists", func() error {
		want := saoKeRecipientsPayload{Recipients: []string{saoKeTestEmail}, CC: []string{saoKeTestCC}}
		var saved saoKeRecipientsPayload
		if _, err := ketoan.PutInto("/api/v1/billing/saoke/recipients", want, &saved); err != nil {
			return fmt.Errorf("PUT recipients: %w", err)
		}
		var got saoKeRecipientsPayload
		if _, err := ketoan.GetInto("/api/v1/billing/saoke/recipients", &got); err != nil {
			return fmt.Errorf("GET recipients: %w", err)
		}
		if len(got.Recipients) != 1 || got.Recipients[0] != saoKeTestEmail {
			return fmt.Errorf("recipients want [%s], got %v", saoKeTestEmail, got.Recipients)
		}
		if len(got.CC) != 1 || got.CC[0] != saoKeTestCC {
			return fmt.Errorf("cc want [%s], got %v", saoKeTestCC, got.CC)
		}
		return nil
	})

	// Giamdoc holds the same grants (PRD §5.8).
	reporter.RunTest(flowSaoKe, "Giamdoc reads the persisted recipients", func() error {
		var got saoKeRecipientsPayload
		if _, err := giamdoc.GetInto("/api/v1/billing/saoke/recipients", &got); err != nil {
			return fmt.Errorf("GET recipients as giamdoc: %w", err)
		}
		if len(got.Recipients) != 1 || got.Recipients[0] != saoKeTestEmail {
			return fmt.Errorf("recipients want [%s], got %v", saoKeTestEmail, got.Recipients)
		}
		return nil
	})

	// Invalid addresses are rejected at the boundary.
	reporter.RunTest(flowSaoKe, "Recipients save rejects a malformed address", func() error {
		_, status, err := ketoan.PutExpectError("/api/v1/billing/saoke/recipients", saoKeRecipientsPayload{
			Recipients: []string{"khong-phai-email"},
		})
		if err != nil {
			return fmt.Errorf("malformed-address save: %w", err)
		}
		if status < 400 {
			return fmt.Errorf("expected malformed-address save to fail, got status %d", status)
		}
		return nil
	})

	// ── Excel export ───────────────────────────────────────────────────────
	reporter.RunTest(flowSaoKe, "Export returns the sao kê workbook", func() error {
		body, header, status, err := ketoan.DownloadGet(fmt.Sprintf("/api/v1/billing/saoke/export?report_date=%s", reportDate))
		if err != nil {
			return fmt.Errorf("GET export: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("export status want 200, got %d", status)
		}
		if len(body) < 4 || string(body[:2]) != "PK" {
			return fmt.Errorf("export body is not an xlsx (len=%d)", len(body))
		}
		contentType := header.Get("Content-Type")
		if !strings.Contains(contentType, "spreadsheetml") {
			return fmt.Errorf("content type want spreadsheetml, got %q", contentType)
		}
		wantFilename := "sao_ke_tt_" + time.Now().Format("02-01-2006") + ".xlsx"
		disposition := header.Get("Content-Disposition")
		if !strings.Contains(disposition, wantFilename) {
			return fmt.Errorf("disposition want %s, got %q", wantFilename, disposition)
		}
		return nil
	})

	// ── Statement email (dev sandbox provider captures in-process) ─────────
	reporter.RunTest(flowSaoKe, "Email send succeeds with at least one recipient", func() error {
		resp, status, err := ketoan.Post("/api/v1/billing/saoke/email", saoKeEmailPayload{
			ReportDate: reportDate,
			Recipients: []string{saoKeTestEmail},
			CC:         []string{saoKeTestCC},
		})
		if err != nil {
			return fmt.Errorf("POST email: %w", err)
		}
		if status != http.StatusOK || resp.Status == "error" {
			return fmt.Errorf("email send want 200, got %d: %s", status, resp.Message)
		}
		return nil
	})

	reporter.RunTest(flowSaoKe, "Email send without recipients is rejected", func() error {
		_, status, err := ketoan.PostExpectError("/api/v1/billing/saoke/email", saoKeEmailPayload{
			ReportDate: reportDate,
			Recipients: []string{},
		})
		if err != nil {
			return fmt.Errorf("empty-recipient send: %w", err)
		}
		if status < 400 {
			return fmt.Errorf("expected empty-recipient send to fail, got status %d", status)
		}
		return nil
	})

	// ── Sổ cái uploads list: matched_amount enrichment ─────────────────────
	reporter.RunTest(flowSaoKe, "Uploads list carries matched_amount per upload", func() error {
		var resp saoKeUploadsPayload
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/billing/uploads?work_month=%s&limit=200", thisMonth), &resp); err != nil {
			return fmt.Errorf("GET uploads: %w", err)
		}
		for _, row := range resp.Items {
			if fmt.Sprintf("%d", row.ID) == uploadID {
				if row.MatchedRows != 1 {
					return fmt.Errorf("matched_rows want 1, got %d", row.MatchedRows)
				}
				if row.MatchedAmount != saoKePaidAmount {
					return fmt.Errorf("matched_amount want %d, got %d", saoKePaidAmount, row.MatchedAmount)
				}
				return nil
			}
		}
		return fmt.Errorf("planted upload %s missing from %s uploads list", uploadID, thisMonth)
	})

	// Leave the saved lists empty so later runs start clean.
	_, _, _ = ketoan.Put("/api/v1/billing/saoke/recipients", saoKeRecipientsPayload{Recipients: []string{}, CC: []string{}})
}

// loginSaoKeRole logs a dev-seed role account in on a fresh client so the
// caller's token channel stays untouched.
func loginSaoKeRole(client *APIClient, username string) (*APIClient, error) {
	roleClient := NewAPIClient(client.BaseURL)
	if _, err := roleClient.Login(username, saoKeDevPassword); err != nil {
		return nil, err
	}
	return roleClient, nil
}

// cleanupSaoKeFixtures removes this flow's planted rows (idempotent across
// runs — the file_hash/original_filename stamp scopes the delete).
func cleanupSaoKeFixtures(projectID, employeeID uint, stamp string) {
	_ = execPayrollSQL(fmt.Sprintf(
		"DELETE FROM employee_pay_cycles WHERE source_upload_id IN (SELECT id FROM transfer_uploads WHERE original_filename LIKE 'it-saoke-%s%%')",
		stamp))
	_ = execPayrollSQL(fmt.Sprintf(
		"DELETE FROM transfer_uploads WHERE original_filename LIKE 'it-saoke-%s%%'", stamp))
	_ = deleteTimesheetsForPair(projectID, employeeID)
}
