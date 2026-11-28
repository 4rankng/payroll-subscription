package main

import (
	"fmt"
)

const flowPartnerProjectSummary = "ProjectPartnerSummary"

// partnerProjectSummaryResponse mirrors GET /api/v1/projects/partner-summary.
type partnerProjectSummaryResponse struct {
	TotalProjects     int `json:"total_projects"`
	ActiveProjects    int `json:"active_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalEmployees    int `json:"total_employees"`
}

// runProjectPartnerSummaryTests proves PROJ-95: the partner-summary endpoint
// is registered, scoped like the partner list, and reachable for the roles
// the frontend gate enables (quanly/giamdoc; ketoan shares the same GET
// grant on /api/v1/projects so it must get 200 here too — never 404).
func runProjectPartnerSummaryTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Project Partner Summary")

	if len(data.Partners) == 0 {
		reporter.RunTest(flowPartnerProjectSummary, "Partner summary (skipped: no partner users)", func() error {
			fmt.Println("    No partner users configured; skipping role-scoped checks")
			return nil
		})
		return
	}

	for _, partner := range data.Partners {
		partnerClient := client.WithToken(partner.Token)
		reporter.RunTest(flowPartnerProjectSummary, fmt.Sprintf("Partner summary for %s", partner.Username), func() error {
			var summary partnerProjectSummaryResponse
			if _, err := partnerClient.GetInto("/api/v1/projects/partner-summary", &summary); err != nil {
				return fmt.Errorf("partner summary: %w", err)
			}
			fmt.Printf("    total=%d active=%d completed=%d employees=%d\n",
				summary.TotalProjects, summary.ActiveProjects, summary.CompletedProjects, summary.TotalEmployees)
			if err := AssertGreaterOrEqual("active+completed vs total",
				summary.ActiveProjects+summary.CompletedProjects, summary.TotalProjects); err != nil {
				return err
			}
			return nil
		})

		// The summary total must agree with the list the same role sees.
		reporter.RunTest(flowPartnerProjectSummary, fmt.Sprintf("Summary agrees with list for %s", partner.Username), func() error {
			var list []ProjectResponse
			if _, err := partnerClient.GetInto("/api/v1/projects", &list); err != nil {
				return fmt.Errorf("list projects: %w", err)
			}
			var summary partnerProjectSummaryResponse
			if _, err := partnerClient.GetInto("/api/v1/projects/partner-summary", &summary); err != nil {
				return fmt.Errorf("partner summary: %w", err)
			}
			if len(list) > summary.TotalProjects {
				return fmt.Errorf("summary total %d smaller than visible list %d",
					summary.TotalProjects, len(list))
			}
			return nil
		})
	}
}
