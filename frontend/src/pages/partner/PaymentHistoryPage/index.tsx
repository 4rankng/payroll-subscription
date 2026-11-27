import { useState, useMemo, useCallback } from 'react';
import { format, startOfMonth, endOfMonth } from 'date-fns';
import { Download } from 'lucide-react';
import { PageHeader } from '@/components/shared/PageHeader';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PaymentHistoryTable } from '@/components/payroll/PaymentHistoryTable';
import { PaymentHistoryFilters } from '@/components/payroll/PaymentHistoryFilters';
import { useBankTransferHistories, useExportBankTransferHistories } from '@/hooks/api/usePaymentHistory';
import { formatFullCurrency } from '@/utils/formatters';
import type { SortingState } from '@tanstack/react-table';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

function SummaryStrip({ summary, isLoading }: { summary?: { total_amount_vnd: number; bank_entry_count: number; employee_count: number }; isLoading: boolean }) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-3 gap-2">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-[72px] rounded-lg" />)}
      </div>
    );
  }
  return (
    <div className="grid grid-cols-3 gap-2">
      <div className="console-panel px-3 py-2">
        <p className="console-kicker">Tổng đã chuyển</p>
        <p className="console-value mt-1 text-lg font-semibold">{formatFullCurrency(summary?.total_amount_vnd ?? 0)}</p>
      </div>
      <div className="console-panel px-3 py-2">
        <p className="console-kicker">Bút toán ngân hàng</p>
        <p className="console-value mt-1 text-lg font-semibold">{(summary?.bank_entry_count ?? 0).toLocaleString('vi-VN')}</p>
      </div>
      <div className="console-panel px-3 py-2">
        <p className="console-kicker">Nhân viên</p>
        <p className="console-value mt-1 text-lg font-semibold">{(summary?.employee_count ?? 0).toLocaleString('vi-VN')}</p>
      </div>
    </div>
  );
}

export default function PaymentHistoryPage() {
  const [filters, setFilters] = useState<FiltersType>(() => ({
    fromDate: format(startOfMonth(new Date()), 'yyyy-MM-dd'),
    toDate: format(endOfMonth(new Date()), 'yyyy-MM-dd'),
    page: 1,
    pageSize: 50,
  }));
  const [sorting, setSorting] = useState<SortingState>([{ id: 'paid_date', desc: true }]);

  const { data, isLoading } = useBankTransferHistories(filters);
  const exportMutation = useExportBankTransferHistories();

  const handleFiltersChange = useCallback((newFilters: FiltersType) => {
    setFilters((prev) => ({ ...prev, ...newFilters, page: 1 }));
  }, []);

  const handleClearFilters = useCallback(() => {
    setFilters({
      fromDate: format(startOfMonth(new Date()), 'yyyy-MM-dd'),
      toDate: format(endOfMonth(new Date()), 'yyyy-MM-dd'),
      page: 1,
      pageSize: 50,
    });
  }, []);

  const handleExport = useCallback(() => {
    exportMutation.mutate(filters);
  }, [exportMutation, filters]);

  const hasFilters = useMemo(() => {
    return !!(filters.search || filters.projectId?.length || filters.employeeId?.length || filters.position);
  }, [filters]);

  return (
    <div className="space-y-3 p-3 sm:p-4">
      <PageHeader
        title="Lịch sử trả lương"
        meta="Bút toán chuyển khoản đã đối chiếu"
      >
        <Button variant="outline" size="sm" onClick={handleExport} disabled={exportMutation.isPending}>
          <Download className="mr-1.5 size-3.5" />
          Xuất Excel
        </Button>
      </PageHeader>

      <SummaryStrip summary={data?.summary} isLoading={isLoading} />

      <PaymentHistoryFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
      />

      <PaymentHistoryTable
        data={data?.rows ?? []}
        isLoading={isLoading}
        pagination={{
          page: data?.page ?? 1,
          pageSize: data?.page_size ?? 50,
          totalPages: Math.ceil((data?.total ?? 0) / (data?.page_size ?? 50)),
          totalRecords: data?.total ?? 0,
        }}
        sorting={sorting}
        onSortingChange={setSorting}
      />
    </div>
  );
}
