import { useQuery } from '@tanstack/react-query';

import { apiClient } from '@/services/api/client';
import { billingService, type DashboardKySummary } from '@/services/api/billing.service';

/**
 * Tổng quan (Fee Insights) dashboard hooks. Module-scoped query keys, same
 * convention as useBilling.ts. The recon-status endpoint is not yet on the
 * billing service, so this module owns its fetch here (same apiClient and
 * { status, data } envelope as the service layer).
 */
export const billingDashboardKeys = {
  all: ['billing-dashboard'] as const,
  /** Reuse the billing module's ky-summary cache identity. */
  kySummary: ['billing', 'ky-summary'] as const,
  reconStatus: (workMonth?: string) =>
    ['billing-dashboard', 'recon-status', workMonth ?? 'current'] as const,
};

export interface ReconKyResponse {
  cycle: number;
  uploads: number;
  parsed_rows: number;
  matched_rows: number;
  state: 'waiting_file' | 'mismatch' | 'reconciled';
}

export interface ReconStatusResponse {
  work_month: string;
  kyes: ReconKyResponse[];
}

async function fetchReconStatus(workMonth?: string): Promise<ReconStatusResponse> {
  const qs = workMonth ? `?work_month=${encodeURIComponent(workMonth)}` : '';
  const response = await apiClient.get<ReconStatusResponse>(
    `/billing/dashboard/recon-status${qs}`,
  );
  return (
    response.data ?? { work_month: workMonth ?? '', kyes: [] }
  );
}

/** FR-6 Tổng quan: distinct employees per Kỳ for the last 12 months + fee. */
export const useBillingKySummary = () =>
  useQuery<DashboardKySummary>({
    queryKey: billingDashboardKeys.kySummary,
    queryFn: () => billingService.getKySummary(12),
  });

/** Đối chiếu (recon) status per Kỳ for one work month (defaults to current). */
export const useBillingReconStatus = (workMonth?: string) =>
  useQuery({
    queryKey: billingDashboardKeys.reconStatus(workMonth),
    queryFn: () => fetchReconStatus(workMonth),
    enabled: workMonth === undefined || workMonth.length === 7,
  });
