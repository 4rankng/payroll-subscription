import { useQuery, useMutation } from '@tanstack/react-query';
import { billingService } from '@/services/api/billing.service';
import { getCurrentMonthValue } from '@/utils/bankTransferHistoryHelpers';
import type { PaymentHistory, PaymentHistoryFilters } from '@/types/api/payroll.types';

export interface PaymentHistoryResult {
  rows: PaymentHistory[];
  summary: { total_amount_vnd: number; bank_entry_count: number; employee_count: number };
  total: number;
  page: number;
  page_size: number;
}

/** Page size requested per query — the backend caps page_size at 500. */
const QUERY_PAGE_SIZE = 500;

function mapBackendRow(row: Record<string, unknown>): PaymentHistory {
  return {
    stt: (row.stt as number) ?? undefined,
    employee_id: (row.employee_id as number) ?? 0,
    employee_name: (row.employee_name as string) ?? '',
    employee_cccd: (row.cccd as string) ?? '',
    project_id: (row.project_id as number) ?? 0,
    project_name: (row.project_name as string) ?? '',
    position: (row.position as string) ?? '',
    total_paid_amount: (row.amount_vnd as number) ?? 0,
    paid_date: (row.pay_date as string) ?? '',
    ky: (row.ky as number) ?? undefined,
    ky_window: (row.ky_window as string) ?? undefined,
    transfer_note: (row.transfer_note as string) ?? undefined,
    bank_transaction_code: (row.bank_transaction_code as string) ?? undefined,
    processed_at: (row.processed_at as string) ?? undefined,
    actual_received_vnd: (row.actual_received_vnd as number) ?? undefined,
  };
}

/** Filters minus the paging keys — the query fetches everything, then slices. */
function ledgerParams(filters: PaymentHistoryFilters): Record<string, unknown> {
  const params: Record<string, unknown> = {
    month: filters.month ?? getCurrentMonthValue(),
    page: 1,
    page_size: QUERY_PAGE_SIZE,
  };
  if (filters.ky) params.ky = filters.ky;
  if (filters.search) params.search = filters.search;
  return params;
}

/** PRD §5.7: one work month per query, Kỳ 1-4 + name/CCCD/code search server-side. */
export function useBankTransferHistories(filters: PaymentHistoryFilters) {
  return useQuery<PaymentHistoryResult>({
    queryKey: ['bank-transfer-histories', filters],
    // Surface failures after one automatic retry — the UI offers an explicit
    // Retry button instead of a long silent backoff.
    retry: 1,
    queryFn: async () => {
      const res = await billingService.getBankTransferHistories(ledgerParams(filters));
      const data = res as unknown as Record<string, unknown> | undefined;
      const rows = ((data?.rows as Record<string, unknown>[] | undefined) ?? []).map(mapBackendRow);
      const summary =
        (data?.summary as PaymentHistoryResult['summary'] | undefined) ?? {
          total_amount_vnd: 0,
          bank_entry_count: 0,
          employee_count: 0,
        };

      // Merged ledger is paginated client-side at the PRD's 50 lines/page.
      const page = filters.page ?? 1;
      const pageSize = filters.pageSize ?? 50;
      return {
        rows: rows.slice((page - 1) * pageSize, page * pageSize),
        summary,
        total: (data?.total as number | undefined) ?? rows.length,
        page,
        page_size: pageSize,
      };
    },
  });
}

export function useExportBankTransferHistories() {
  return useMutation({
    mutationFn: async (filters: PaymentHistoryFilters) => {
      const params: Record<string, unknown> = {
        month: filters.month ?? getCurrentMonthValue(),
      };
      if (filters.ky) params.ky = filters.ky;
      if (filters.search) params.search = filters.search;

      const blob = await billingService.exportBankTransferHistories(params);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `lich-su-tra-luong-${params.month}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    },
  });
}
