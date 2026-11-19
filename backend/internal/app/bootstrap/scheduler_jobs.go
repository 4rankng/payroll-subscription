package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"api-server/internal/app/services/billing"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/scheduler"
	"api-server/internal/pkg/clock"
)

func registerSchedulerJobs(
	s *scheduler.Scheduler,
	aggregateRecomputeService *project.AggregateRecomputeService,
	logger *slog.Logger,
	billingReportService *billing.BillingReportService,
) {
	// Monthly billing report (FR-3/PAYSUB-4): generate the previous work
	// month's report on day 1 at 09:00 (after Kỳ 4's pay date). Regeneration
	// replaces lines/workbook; the fee snapshot follows the generation-time rule.
	s.AddJob(scheduler.Job{
		Name:    "generate_monthly_billing_report",
		Cron:    "0 9 1 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			workMonth := clock.Now().In(time.Local).AddDate(0, -1, 0).Format("2006-01")
			logger.Info("Generating monthly billing report", "work_month", workMonth)
			if _, _, err := billingReportService.GenerateMonthly(ctx, workMonth, 0); err != nil {
				logger.Error("Failed to generate monthly billing report", "work_month", workMonth, "error", err)
				return
			}
			billingReportService.NotifyReportReady(ctx, workMonth)
			logger.Info("Monthly billing report generated", "work_month", workMonth)
		},
	})

	// Reconcile the denormalised project payout totals - daily.
	//
	// The projects.total_payout_vnd / pending_payable_vnd totals have no HTTP
	// writer: they are projections of the timesheet rows, so a lost write
	// leaves a total stale with nothing to correct it. This job re-derives
	// every project's totals from its timesheets, which is a pure function of
	// those rows and therefore idempotent: a daily run turns silent drift into
	// a bounded, self-healing lag instead of a manual backfill.
	s.AddJob(scheduler.Job{
		Name:    "recompute_project_aggregates",
		Cron:    "0 3 * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting project payout aggregate reconciliation")
			result, err := aggregateRecomputeService.RecomputeAllProjects(ctx)
			if err != nil {
				// A failed run either listed no projects (result is nil) or
				// recomputed some and accumulated the rest; log both shapes.
				if result == nil {
					logger.Error("Failed to reconcile project payout aggregates", "error", err)
					return
				}
				logger.Error("Project payout aggregate reconciliation reported failures",
					"error", err,
					"recomputed", result.Recomputed,
					"failed", result.Failed)
				return
			}
			logger.Info("Project payout aggregates reconciled",
				"recomputed", result.Recomputed)
		},
	})
}
