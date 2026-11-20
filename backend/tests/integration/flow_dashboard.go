package main

import (
	"fmt"
)

const flowDashboard = "Dashboard"

func runDashboardTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Dashboard Read-Only")

	admin := client.WithToken(data.AdminToken)

	// Live dashboard surface (PAYSUB rebuild): the summary feeds the Tổng quan
	// metric strip, new-employees and top-paid feed its cards. The old-model
	// endpoints (financial-overview, monthly-financials, salary-distribution,
	// employee-activity, bank-usage, profitability, partner) are removed and
	// must stay absent — their absence is pinned by the routes file.
	endpoints := []struct {
		name string
		path string
	}{
		{"Summary", "/api/v1/dashboard/summary"},
		{"New employees", "/api/v1/dashboard/new-employees?pageSize=5"},
		{"Top paid employees", "/api/v1/dashboard/top-paid-employees"},
	}

	for _, ep := range endpoints {
		ep := ep
		reporter.RunTest(flowDashboard, "GET "+ep.name, func() error {
			var resp interface{}
			if _, err := admin.GetInto(ep.path, &resp); err != nil {
				return fmt.Errorf("%s: %w", ep.name, err)
			}
			fmt.Printf("    OK: %s\n", ep.name)
			return nil
		})
	}

	// Removed old-model dashboard endpoints must answer 404, not 200 with
	// stale aggregates (PAYSUB demolition pin).
	removed := []struct {
		name string
		path string
	}{
		{"Financial overview", "/api/v1/dashboard/financial-overview"},
		{"Salary distribution", "/api/v1/dashboard/salary-distribution"},
		{"Monthly financials", "/api/v1/dashboard/monthly-financials"},
		{"Employee activity", "/api/v1/dashboard/employee-activity"},
		{"Bank usage", "/api/v1/dashboard/bank-usage"},
		{"Project profitability", "/api/v1/dashboard/project-profitability"},
		{"Partner dashboard", "/api/v1/dashboard/partner"},
	}
	for _, ep := range removed {
		ep := ep
		reporter.RunTest(flowDashboard, "REMOVED "+ep.name, func() error {
			// The router answers plain-text "404 page not found" — check the
			// status code directly instead of parsing the body.
			_, statusCode, parseErr := admin.Get(ep.path)
			if parseErr != nil && statusCode == 0 {
				return fmt.Errorf("%s: %w", ep.name, parseErr)
			}
			if statusCode != 404 {
				return fmt.Errorf("%s: want 404 for removed endpoint, got HTTP %d", ep.name, statusCode)
			}
			return nil
		})
	}

	if cfg == nil {
		return
	}

	// Partner-role dashboard read access (quanly has /dashboard/* GET).
	if len(data.Partners) > 0 {
		partner := client.WithToken(data.Partners[0].Token)

		reporter.RunTest(flowDashboard, "Partner summary read", func() error {
			var resp interface{}
			if _, err := partner.GetInto("/api/v1/dashboard/summary", &resp); err != nil {
				return fmt.Errorf("partner summary: %w", err)
			}
			fmt.Printf("    OK: Partner summary\n")
			return nil
		})
	}
}
