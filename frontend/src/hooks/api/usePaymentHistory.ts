import { useQuery, useMutation } from '@tanstack/react-query';
import { addMonths, format, parse, startOfMonth } from 'date-fns';
import { billingService } from '@/services/api/billing.service';
import type { PaymentHistoryFilters, PaymentHistory } from '@/types/api/payroll.types';

interface BankTransferHistoriesResult {
  rows: PaymentHistory[];
  summary: { total_amount_vnd: number; bank_entry_count: number; employee_count: number };
  total: number;
  page: number;
  page_size: number;
}

/** Page size requested per month — the backend caps page_size at 500. */
const MONTH_PAGE_SIZE = 500;
/**
 * Longest month span the ledger queries. The filter's calendar enforces the
 * same bound (minDate = 35 months back), so ranges never exceed this.
 */
const MAX_MONTHS = 36;

interface MonthLedger {
  rows: Record<string, unknown>[];
  total: number;
  summary: BankTransferHistoriesResult['summary'];
}

function mapBackendRow(row: Record<string, unknown>): PaymentHistory {
  return {
    employee_id: (row.employee_id as number) ?? 0,
    employee_name: (row.employee_name as string) ?? '',
    employee_cccd: (row.cccd as string) ?? '',
    project_id: (row.project_id as number) ?? 0,
    project_name: (row.project_name as string) ?? '',
    position: (row.position as string) ?? '',
    total_paid_amount: (row.amount_vnd as number) ?? 0,
    paid_date: (row.pay_date as string) ?? '',
    stt: (row.stt as number) ?? undefined,
    ky: (row.ky as number) ?? undefined,
    ky_window: (row.ky_window as string) ?? undefined,
    transfer_note: (row.transfer_note as string) ?? undefined,
    bank_transaction_code: (row.bank_transaction_code as string) ?? undefined,
    processed_at: (row.processed_at as string) ?? undefined,
    actual_received_vnd: (row.actual_received_vnd as number) ?? undefined,
  };
}

/**
 * List the YYYY-MM months covered by an inclusive yyyy-MM-dd range. The
 * backend queries one work month at a time, so a range fans out over these.
 */
function monthsInRange(fromDate?: string, toDate?: string): string[] {
  const fallback = [format(startOfMonth(new Date()), 'yyyy-MM')];
  if (!fromDate) return fallback;
  const from = parse(fromDate, 'yyyy-MM-dd', new Date());
  const to = toDate ? parse(toDate, 'yyyy-MM-dd', new Date()) : from;
  if (Number.isNaN(from.getTime()) || Number.isNaN(to.getTime())) return fallback;

  const months: string[] = [];
  let cursor = startOfMonth(from);
  const last = startOfMonth(to);
  while (cursor <= last && months.length < MAX_MONTHS) {
    months.push(format(cursor, 'yyyy-MM'));
    cursor = addMonths(cursor, 1);
  }
  return months.length > 0 ? months : fallback;
}

export function useBankTransferHistories(filters: PaymentHistoryFilters) {
  return useQuery<BankTransferHistoriesResult>({
    queryKey: ['bank-transfer-histories', filters],
    // Surface failures after one automatic retry — the UI offers an explicit
    // Retry button instead of a long silent backoff.
    retry: 1,
    queryFn: async () => {
      // The endpoint takes a single work month, so a selected range fans out
      // into one request per month and merges client-side.
      const months = monthsInRange(filters.fromDate, filters.toDate);
      const monthLedgers = await Promise.all(
        months.map(async (month): Promise<MonthLedger> => {
          const params: Record<string, unknown> = {
            month,
            page: 1,
            page_size: MONTH_PAGE_SIZE,
          };
          if (filters.search) params.search = filters.search;

          const res = await billingService.getBankTransferHistories(params);
          const data = res as unknown as Record<string, unknown> | undefined;
          return {
            rows: (data?.rows as Record<string, unknown>[] | undefined) ?? [],
            total: (data?.total as number | undefined) ?? 0,
            summary:
              (data?.summary as BankTransferHistoriesResult['summary'] | undefined) ?? {
                total_amount_vnd: 0,
                bank_entry_count: 0,
                employee_count: 0,
              },
          };
        }),
      );

      const allRows = monthLedgers.flatMap((ledger) => ledger.rows).map(mapBackendRow);
      const summary = monthLedgers.reduce(
        (acc, ledger) => ({
          total_amount_vnd: acc.total_amount_vnd + ledger.summary.total_amount_vnd,
          bank_entry_count: acc.bank_entry_count + ledger.summary.bank_entry_count,
          employee_count: acc.employee_count,
        }),
        { total_amount_vnd: 0, bank_entry_count: 0, employee_count: 0 },
      );
      summary.employee_count = new Set(allRows.map((row) => row.employee_id)).size;

      // Merged months are paginated client-side.
      const page = filters.page ?? 1;
      const pageSize = filters.pageSize ?? 50;
      const rows = allRows.slice((page - 1) * pageSize, page * pageSize);

      return {
        rows,
        summary,
        total: allRows.length,
        page,
        page_size: pageSize,
      };
    },
  });
}

export function useExportBankTransferHistories() {
  return useMutation({
    mutationFn: async (filters: PaymentHistoryFilters) => {
      const params: Record<string, unknown> = {};
      if (filters.fromDate) params.month = filters.fromDate.slice(0, 7);
      if (filters.search) params.search = filters.search;

      const blob = await billingService.exportBankTransferHistories(params);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `lich-su-tra-luong-${filters.fromDate?.slice(0, 7) ?? 'export'}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    },
  });
}
