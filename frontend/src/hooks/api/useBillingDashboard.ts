import { useQuery } from '@tanstack/react-query';

import { billingService, type DashboardKySummary } from '@/services/api/billing.service';

/**
 * Tổng quan (Fee Insights) dashboard hooks. Module-scoped query keys, same
 * convention as useBilling.ts.
 */
export const billingDashboardKeys = {
  all: ['billing-dashboard'] as const,
  /** Reuse the billing module's ky-summary cache identity. */
  kySummary: ['billing', 'ky-summary'] as const,
};

/** FR-6 Tổng quan: distinct employees per Kỳ for the last 12 months + fee. */
export const useBillingKySummary = () =>
  useQuery<DashboardKySummary>({
    queryKey: billingDashboardKeys.kySummary,
    queryFn: () => billingService.getKySummary(12),
  });
