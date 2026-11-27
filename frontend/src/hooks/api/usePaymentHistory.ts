import { useQuery, useMutation } from '@tanstack/react-query';
import { billingService } from '@/services/api/billing.service';
import type { PaymentHistoryFilters, PaymentHistory } from '@/types/api/payroll.types';

interface BankTransferHistoriesResult {
  rows: PaymentHistory[];
  summary: { total_amount_vnd: number; bank_entry_count: number; employee_count: number };
  total: number;
  page: number;
  page_size: number;
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
  };
}

export function useBankTransferHistories(filters: PaymentHistoryFilters) {
  return useQuery<BankTransferHistoriesResult>({
    queryKey: ['bank-transfer-histories', filters],
    queryFn: async () => {
      const params: Record<string, unknown> = {};
      if (filters.fromDate) params.month = filters.fromDate.slice(0, 7);
      if (filters.search) params.search = filters.search;
      if (filters.page) params.page = filters.page;
      if (filters.pageSize) params.page_size = filters.pageSize;

      const res = await billingService.getBankTransferHistories(params);
      const data = res as unknown as Record<string, unknown>;
      return {
        rows: ((data.rows as Record<string, unknown>[]) ?? []).map(mapBackendRow),
        summary: (data.summary as BankTransferHistoriesResult['summary']) ?? { total_amount_vnd: 0, bank_entry_count: 0, employee_count: 0 },
        total: (data.total as number) ?? 0,
        page: (data.page as number) ?? 1,
        page_size: (data.page_size as number) ?? 50,
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
