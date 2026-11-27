import { useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  billingService,
  type SaokeRecipients,
  type SendSaokeEmailParams,
} from '@/services/api/billing.service';
import { useBillingUploads, useEmployeePayCycles } from '@/hooks/api/useBilling';
import { buildLedgerBatches, buildSaokeSummary, type LedgerBatch, type SaokeSummary } from '@/utils/saoke';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

/**
 * Sổ cái / sao kê hooks (PAYR-41). Keys are module-scoped like useBilling —
 * this module owns its cache identity.
 */
export const saokeKeys = {
  all: ['saoke'] as const,
  recipients: ['saoke', 'recipients'] as const,
};

export const useSaokeRecipients = () =>
  useQuery({
    queryKey: saokeKeys.recipients,
    queryFn: () => billingService.getSaokeRecipients(),
  });

export const useUpdateSaokeRecipients = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { to: string[]; cc: string[] }) => billingService.updateSaokeRecipients(params),
    onSuccess: (data: SaokeRecipients) => {
      queryClient.setQueryData(saokeKeys.recipients, data);
      showSuccessNotification('Đã lưu danh sách người nhận sao kê');
    },
    onError: (error: Error) => showErrorNotification(error),
  });
};

export const useExportSaoke = () =>
  useMutation({
    mutationFn: (reportDate: string) => billingService.exportSaoke(reportDate),
    onSuccess: () => showSuccessNotification('Đã tải file sao kê'),
    onError: (error: Error) => showErrorNotification(error),
  });

export const useSendSaokeEmail = () =>
  useMutation({
    mutationFn: (params: SendSaokeEmailParams) => billingService.sendSaokeEmail(params),
    onSuccess: () => showSuccessNotification('Đã gửi email sao kê'),
    onError: (error: Error) => showErrorNotification(error),
  });

export interface LedgerBatchesResult {
  batches: LedgerBatch[];
  summary: SaokeSummary;
  isLoading: boolean;
  isError: boolean;
}

/**
 * The Sổ cái batch list: transfer uploads for one work month, each carrying
 * the settled amount summed from its per-employee paid rows, plus the compact
 * summary the sao kê dialogs preview.
 */
export const useLedgerBatches = (workMonth: string): LedgerBatchesResult => {
  const uploadsQuery = useBillingUploads(workMonth);
  const payCyclesQuery = useEmployeePayCycles(workMonth);

  const batches = useMemo(
    () => buildLedgerBatches(uploadsQuery.data ?? [], payCyclesQuery.data?.rows ?? []),
    [uploadsQuery.data, payCyclesQuery.data],
  );
  const summary = useMemo(() => buildSaokeSummary(workMonth, batches), [workMonth, batches]);

  return {
    batches,
    summary,
    isLoading: uploadsQuery.isLoading || payCyclesQuery.isLoading,
    isError: uploadsQuery.isError || payCyclesQuery.isError,
  };
};
