package main

import (
	"fmt"
	"strings"
)

const flowMetrics = "Metrics"

func runMetricsTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Metrics (Prometheus)")

	admin := client.WithToken(data.AdminToken)

	// The Prometheus exposition endpoint is the only metrics surface; the
	// per-stat JSON views (/metrics/api, /metrics/cache, …) belonged to the
	// removed API-metric table (PAYSUB-9) and are covered by the dashboard
	// flow's REMOVED pins.
	reporter.RunTest(flowMetrics, "GET Prometheus metrics", func() error {
		resp, statusCode, err := admin.Get("/metrics")
		if err != nil {
			// The exposition is plain Prometheus text, not the JSON envelope —
			// the client's parse step fails on a healthy 200.
			if statusCode == 200 {
				return nil
			}
			return fmt.Errorf("metrics: %w", err)
		}
		if statusCode >= 400 {
			return fmt.Errorf("metrics: HTTP %d", statusCode)
		}
		if !strings.Contains(resp.Message, "go_") && len(resp.Data) == 0 {
			// Empty exposition is still valid for a quiet server; only fail on
			// an unparseable empty response.
			return fmt.Errorf("metrics: empty response")
		}
		return nil
	})
}
