import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { billingService } from '@/services/api/billing.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

/**
 * Billing module hooks (FR-2/FR-3/FR-6). Keys are module-scoped: this module
 * owns its cache identity — no god-file registration needed.
 */
export const billingKeys = {
  all: ['billing'] as const,
  uploads: (workMonth: string) => ['billing', 'uploads', workMonth] as const,
  payCycles: (workMonth: string) => ['billing', 'pay-cycles', workMonth] as const,
  paidSeries: (from: string, to: string, projectId?: number) =>
    ['billing', 'paid-series', from, to, projectId ?? 'all'] as const,
  kySummary: ['billing', 'ky-summary'] as const,
  report: (workMonth: string) => ['billing', 'report', workMonth] as const,
};

export const useBillingUploads = (workMonth: string) =>
  useQuery({
    queryKey: billingKeys.uploads(workMonth),
    queryFn: () => billingService.listUploads(workMonth),
    enabled: workMonth.length === 7,
  });

export const useEmployeePayCycles = (workMonth: string) =>
  useQuery({
    queryKey: billingKeys.payCycles(workMonth),
    queryFn: () => billingService.getEmployeePayCycles(workMonth),
    enabled: workMonth.length === 7,
  });

export const usePaidSeries = (from: string, to: string, projectId?: number) =>
  useQuery({
    queryKey: billingKeys.paidSeries(from, to, projectId),
    queryFn: () => billingService.getPaidSeries({ from, to, projectId }),
    enabled: from.length === 10 && to.length === 10,
  });

export const useKySummary = (months = 12) =>
  useQuery({
    queryKey: billingKeys.kySummary,
    queryFn: () => billingService.getKySummary(months),
  });

export const useTransferUpload = (workMonth: string) => {
  const upload = useMutation({
    mutationFn: ({ cycle, file }: { cycle: number; file: File }) =>
      billingService.uploadTransferResult(cycle, workMonth, file),
    onError: (error: Error) => showErrorNotification(error),
  });
  return upload;
};

export const useGenerateBillingReport = (workMonth: string) => {
  const queryClient = useQueryClient();
  const generate = useMutation({
    mutationFn: () => billingService.generateReport(workMonth),
    onSuccess: ({ report, lines }) => {
      // Seed the report cache so the panel shows the fresh report (and its
      // Excel link) right away instead of staying on "Chưa có báo cáo".
      queryClient.setQueryData(billingKeys.report(workMonth), { ...report, lines });
      showSuccessNotification(
        `Báo cáo phí tháng ${report.work_month} đã tạo: ${report.total_distinct} nhân viên × các kỳ, tổng ${report.total_vnd} VND`,
      );
    },
    onError: (error: Error) => showErrorNotification(error),
  });
  return generate;
};

export const useBillingReport = (workMonth: string) =>
  useQuery({
    queryKey: billingKeys.report(workMonth),
    queryFn: () => billingService.getReport(workMonth),
    enabled: workMonth.length === 7,
  });
