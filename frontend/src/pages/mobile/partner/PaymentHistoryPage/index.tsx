import { useState, useMemo, useCallback } from 'react';
import { format, startOfMonth, endOfMonth } from 'date-fns';
import { Download } from 'lucide-react';
import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PaymentHistoryMobileList } from '@/components/payroll/mobile/PaymentHistoryMobileList';
import { PaymentHistoryFilters } from '@/components/payroll/PaymentHistoryFilters';
import { useBankTransferHistories, useExportBankTransferHistories } from '@/hooks/api/usePaymentHistory';
import { formatFullCurrency } from '@/utils/formatters';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

export default function PaymentHistoryMobilePage() {
  const [filters, setFilters] = useState<FiltersType>(() => ({
    fromDate: format(startOfMonth(new Date()), 'yyyy-MM-dd'),
    toDate: format(endOfMonth(new Date()), 'yyyy-MM-dd'),
    page: 1,
    pageSize: 50,
  }));

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
    return !!(filters.search || filters.projectId?.length || filters.employeeId?.length);
  }, [filters]);

  return (
    <MobilePageShell className="space-y-2">
      <MobilePageHeader
        title="Lịch sử trả lương"
        subtitle="Bút toán chuyển khoản đã đối chiếu"
        actions={
          <Button variant="outline" size="sm" onClick={handleExport} disabled={exportMutation.isPending} className="min-h-[44px]">
            <Download className="mr-1 size-3.5" />
            Excel
          </Button>
        }
      />

      {/* Summary strip */}
      {isLoading ? (
        <div className="grid grid-cols-3 gap-2">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-[68px] rounded-lg" />)}
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-2">
          <MobileSurface className="px-2.5 py-2">
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Tổng chuyển</p>
            <p className="mt-0.5 text-sm font-semibold tabular-nums">{formatFullCurrency(data?.summary?.total_amount_vnd ?? 0)}</p>
          </MobileSurface>
          <MobileSurface className="px-2.5 py-2">
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Bút toán</p>
            <p className="mt-0.5 text-sm font-semibold tabular-nums">{(data?.summary?.bank_entry_count ?? 0).toLocaleString('vi-VN')}</p>
          </MobileSurface>
          <MobileSurface className="px-2.5 py-2">
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Nhân viên</p>
            <p className="mt-0.5 text-sm font-semibold tabular-nums">{(data?.summary?.employee_count ?? 0).toLocaleString('vi-VN')}</p>
          </MobileSurface>
        </div>
      )}

      <PaymentHistoryFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
      />

      <PaymentHistoryMobileList
        data={data?.rows ?? []}
        isLoading={isLoading}
      />
    </MobilePageShell>
  );
}
