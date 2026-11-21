package main

import (
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/pkg/clock"
)

const flowBilling = "Billing"

// billingKySummary is the FR-6 per-month table payload of
// GET /billing/dashboard/ky-summary.
type billingKySummary struct {
	Months []struct {
		WorkMonth string         `json:"work_month"`
		Counts    map[string]int64 `json:"counts"`
	} `json:"months"`
	FeePerEmployeePerKyVnd int64 `json:"fee_per_employee_per_ky_vnd"`
}

// billingKyPaidRecords is the GET /billing/ky-paid-records payload.
type billingKyPaidRecords struct {
	Items []struct {
		ID        uint   `json:"id"`
		Cycle     int    `json:"cycle"`
		WorkMonth string `json:"work_month"`
		EmployeeID uint  `json:"employee_id"`
		ProjectID uint   `json:"project_id"`
		PaidAmount int64 `json:"paid_amount"`
	} `json:"items"`
	SummaryByCycle map[string]int64 `json:"summary_by_cycle"`
}

// paidSeriesItem is one day×project bucket of GET /billing/dashboard/paid-series.
type paidSeriesItem struct {
	Date          string `json:"date"`
	ProjectID     uint   `json:"project_id"`
	EmployeeCount int64  `json:"employee_count"`
	Amount        int64  `json:"amount"`
}

// paidSeriesResponse wraps the handler's {items: [...]} payload.
type paidSeriesResponse struct {
	Items []paidSeriesItem `json:"items"`
}

func runBillingTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: PAYSUB Billing (FR-1/FR-6)")

	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowBilling, "All tests", "no fixture project/employee discovered")
		return
	}

	admin := client.WithToken(data.AdminToken)
	projectID := data.WeeklyProject.ID
	employeeID := data.WeeklyEmployee.ID

	now := clock.Now()
	thisMonth := now.Format("2006-01")
	prevMonth := now.AddDate(0, -1, 0).Format("2006-01")
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())

	cleanupBillingFixtures(projectID, employeeID, stamp)

	// ── Seed: paid timesheets in three (month, Kỳ) buckets ─────────────────
	// plantPaidTimesheet writes approved + paid rows — exactly what upload
	// verification produces, so the read model sees a realistic substrate.
	plantDates := []struct{ month, date string }{
		{prevMonth, prevMonth + "-03"}, // Kỳ 1 of the previous work month
		{prevMonth, prevMonth + "-10"}, // Kỳ 2 of the previous work month
		{thisMonth, thisMonth + "-03"}, // Kỳ 1 of the current work month
	}
	for _, pd := range plantDates {
		if err := plantPaidTimesheet(projectID, employeeID, pd.date); err != nil {
			reporter.Skip(flowBilling, "All tests", fmt.Sprintf("plant timesheet %s: %v", pd.date, err))
			cleanupBillingFixtures(projectID, employeeID, stamp)
			return
		}
	}
	// FR-6's paid series buckets by payment_date (fallback paid_at) — the
	// shared planter leaves both NULL, so stamp the payment onto the work date.
	if err := execPayrollSQL(fmt.Sprintf(
		"UPDATE timesheets SET payment_date = date, paid_at = NOW() WHERE project_id = %d AND employee_id = %d AND date IN ('%s','%s','%s')",
		projectID, employeeID, plantDates[0].date, plantDates[1].date, plantDates[2].date)); err != nil {
		reporter.Skip(flowBilling, "All tests", fmt.Sprintf("stamp payment dates: %v", err))
		cleanupBillingFixtures(projectID, employeeID, stamp)
		return
	}

	// Resolve the planted timesheet ids for the employee_pay_cycles FKs.
	tsIDs, err := queryPayrollScalar(fmt.Sprintf(
		"SELECT GROUP_CONCAT(id ORDER BY date) FROM timesheets WHERE project_id = %d AND employee_id = %d AND date IN ('%s','%s','%s')",
		projectID, employeeID, plantDates[0].date, plantDates[1].date, plantDates[2].date))
	if err != nil || len(splitCSV(tsIDs)) != len(plantDates) {
		reporter.Skip(flowBilling, "All tests", fmt.Sprintf("resolve planted timesheets: err=%v ids=%q", err, tsIDs))
		cleanupBillingFixtures(projectID, employeeID, stamp)
		return
	}
	ids := splitCSV(tsIDs)

	uploaderID, err := queryPayrollScalar("SELECT id FROM users ORDER BY id LIMIT 1")
	if err != nil || uploaderID == "" {
		reporter.Skip(flowBilling, "All tests", fmt.Sprintf("resolve uploader user: %v", err))
		cleanupBillingFixtures(projectID, employeeID, stamp)
		return
	}

	// One transfer upload per (month, Kỳ): prev month Kỳ 1+2, this month Kỳ 1.
	uploads := []struct{ month, cycle string }{
		{prevMonth, "1"},
		{prevMonth, "2"},
		{thisMonth, "1"},
	}
	uploadIDs := make([]string, 0, len(uploads))
	for i, up := range uploads {
		if err := execPayrollSQL(fmt.Sprintf(
			"INSERT INTO transfer_uploads (cycle, work_month, file_hash, original_filename, uploaded_by, status, verified_at, created_at, updated_at) "+
				"VALUES (%s, '%s', 'it-billing-%s-%d', 'it-billing-%s-%d.xlsx', %s, 'verified', NOW(), NOW(), NOW())",
			up.cycle, up.month, stamp, i, stamp, i, uploaderID)); err != nil {
			reporter.Skip(flowBilling, "All tests", fmt.Sprintf("plant transfer upload %d: %v", i, err))
			cleanupBillingFixtures(projectID, employeeID, stamp)
			return
		}
		uid, uErr := queryPayrollScalar(fmt.Sprintf(
			"SELECT id FROM transfer_uploads WHERE file_hash = 'it-billing-%s-%d'", stamp, i))
		if uErr != nil || uid == "" {
			reporter.Skip(flowBilling, "All tests", fmt.Sprintf("resolve upload id %d: %v", i, uErr))
			cleanupBillingFixtures(projectID, employeeID, stamp)
			return
		}
		uploadIDs = append(uploadIDs, uid)
	}

	// employee_pay_cycles: one row per planted timesheet (unique per
	// employee + Kỳ + month — COUNT(*) is the distinct-employee count).
	for i, up := range uploads {
		if err := execPayrollSQL(fmt.Sprintf(
			"INSERT INTO employee_pay_cycles (cycle, work_month, employee_id, project_id, timesheet_id, source_upload_id, paid_amount, created_at) "+
				"VALUES (%s, '%s', %d, %d, %s, %s, 800000, NOW())",
			up.cycle, up.month, employeeID, projectID, ids[i], uploadIDs[i])); err != nil {
			reporter.Skip(flowBilling, "All tests", fmt.Sprintf("plant employee_pay_cycles %d: %v", i, err))
			cleanupBillingFixtures(projectID, employeeID, stamp)
			return
		}
	}

	// ── FR-6 (3): per-month per-Kỳ summary + fee ───────────────────────────
	reporter.RunTest(flowBilling, "Ky-summary returns per-Kỳ distinct counts and fee", func() error {
		var summary billingKySummary
		if _, err := admin.GetInto("/api/v1/billing/dashboard/ky-summary?months=3", &summary); err != nil {
			return fmt.Errorf("ky-summary: %w", err)
		}
		if summary.FeePerEmployeePerKyVnd <= 0 {
			return fmt.Errorf("fee_per_employee_per_ky_vnd must be positive, got %d", summary.FeePerEmployeePerKyVnd)
		}
		byMonth := make(map[string]map[string]int64, len(summary.Months))
		for _, m := range summary.Months {
			byMonth[m.WorkMonth] = m.Counts
		}
		prev, ok := byMonth[prevMonth]
		if !ok {
			return fmt.Errorf("ky-summary missing work month %s (payload: %s)", prevMonth, marshal(summary))
		}
		// Shared DB: other flows may add rows for the same months — assert the
		// planted Kỳ buckets are present, not exact totals.
		if prev["1"] < 1 || prev["2"] < 1 {
			return fmt.Errorf("%s counts want Kỳ1>=1 and Kỳ2>=1, got %s", prevMonth, marshal(prev))
		}
		cur, ok := byMonth[thisMonth]
		if !ok || cur["1"] < 1 {
			return fmt.Errorf("%s counts want Kỳ1>=1, got %s (present=%v)", thisMonth, marshal(cur), ok)
		}
		return nil
	})

	// ── FR-1: per-employee-per-Kỳ rows + per-Kỳ summary ────────────────────
	reporter.RunTest(flowBilling, "Ky-paid-records returns rows and per-Kỳ summary", func() error {
		var resp billingKyPaidRecords
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/billing/ky-paid-records?work_month=%s", prevMonth), &resp); err != nil {
			return fmt.Errorf("ky-paid-records: %w", err)
		}
		// Shared DB: scope the assertion to the flow's own planted employee.
		mine := 0
		for _, row := range resp.Items {
			if row.EmployeeID == employeeID {
				mine++
			}
		}
		if mine != 2 {
			return fmt.Errorf("%s wants 2 pay-cycle rows for employee %d, got %d", prevMonth, employeeID, mine)
		}
		return nil
	})

	// ── FR-6 (1)/(2): per-day paid time series ─────────────────────────────
	reporter.RunTest(flowBilling, "Paid-series covers the planted days", func() error {
		from := prevMonth + "-01"
		var resp paidSeriesResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/billing/dashboard/paid-series?from=%s&to=%s", from, today()), &resp); err != nil {
			return fmt.Errorf("paid-series: %w", err)
		}
		totalEmployees := int64(0)
		totalAmount := int64(0)
		for _, row := range resp.Items {
			if row.ProjectID == projectID {
				totalEmployees += row.EmployeeCount
				totalAmount += row.Amount
			}
		}
		if totalEmployees != 3 {
			return fmt.Errorf("paid employees for project %d want 3, got %d", projectID, totalEmployees)
		}
		if totalAmount != 3*800000 {
			return fmt.Errorf("paid amount want %d, got %d", 3*800000, totalAmount)
		}
		return nil
	})

	cleanupBillingFixtures(projectID, employeeID, stamp)
}

// cleanupBillingFixtures removes this flow's planted rows (idempotent across
// runs — the file_hash/original_filename stamp scopes the delete).
func cleanupBillingFixtures(projectID, employeeID uint, stamp string) {
	_ = execPayrollSQL(fmt.Sprintf(
		"DELETE FROM employee_pay_cycles WHERE source_upload_id IN (SELECT id FROM transfer_uploads WHERE original_filename LIKE 'it-billing-%s%%')",
		stamp))
	_ = execPayrollSQL(fmt.Sprintf(
		"DELETE FROM transfer_uploads WHERE original_filename LIKE 'it-billing-%s%%'", stamp))
	_ = deleteTimesheetsForPair(projectID, employeeID)
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func marshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}
