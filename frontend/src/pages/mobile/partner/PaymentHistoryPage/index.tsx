import { useState, useMemo, useCallback, useEffect } from 'react';
import { format, startOfMonth, endOfMonth } from 'date-fns';
import { useNavigate } from 'react-router-dom';
import { Download } from 'lucide-react';
import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PaymentHistoryMobileList } from '@/components/payroll/mobile/PaymentHistoryMobileList';
import { PaymentHistoryFilters } from '@/components/payroll/PaymentHistoryFilters';
import { useBankTransferHistories, useExportBankTransferHistories } from '@/hooks/api/usePaymentHistory';
import { formatFullCurrency, formatInt } from '@/utils/formatters';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

/** The statement-upload entry (Tải lên kết quả chuyển khoản) lives here. */
const STATEMENT_UPLOAD_PATH = '/timesheet';

export default function PaymentHistoryMobilePage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<FiltersType>(() => ({
    fromDate: format(startOfMonth(new Date()), 'yyyy-MM-dd'),
    toDate: format(endOfMonth(new Date()), 'yyyy-MM-dd'),
    page: 1,
    pageSize: 50,
  }));
  const [dateFocusSignal, setDateFocusSignal] = useState(0);

  const { data, isLoading, isError, error, refetch } = useBankTransferHistories(filters);
  const exportMutation = useExportBankTransferHistories();

  // The UI shows a fixed Vietnamese message (PAYR-122) — the raw failure is
  // logged here only, for diagnosis.
  useEffect(() => {
    if (error) console.error('[payment-history] ledger load failed', error);
  }, [error]);

  // 3s slow-load status: flips the "Đang tải lịch sử đối chiếu…" label on.
  const [slowLoading, setSlowLoading] = useState(false);
  useEffect(() => {
    if (!isLoading) {
      setSlowLoading(false);
      return;
    }
    const timer = setTimeout(() => setSlowLoading(true), 3000);
    return () => clearTimeout(timer);
  }, [isLoading]);

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

  const handlePickDateRange = useCallback(() => {
    setDateFocusSignal((signal) => signal + 1);
  }, []);

  const handleUploadStatement = useCallback(() => {
    navigate(STATEMENT_UPLOAD_PATH);
  }, [navigate]);

  const hasFilters = useMemo(() => {
    return !!(filters.search || filters.projectId?.length || filters.employeeId?.length);
  }, [filters]);

  const listError = isError
    ? error instanceof Error
      ? error
      : new Error('Không thể truy vấn dữ liệu từ máy chủ.')
    : null;

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
      {!isError && (
        isLoading ? (
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
              <p className="mt-0.5 text-sm font-semibold tabular-nums">{formatInt(data?.summary?.bank_entry_count ?? 0)}</p>
            </MobileSurface>
            <MobileSurface className="px-2.5 py-2">
              <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Nhân viên</p>
              <p className="mt-0.5 text-sm font-semibold tabular-nums">{formatInt(data?.summary?.employee_count ?? 0)}</p>
            </MobileSurface>
          </div>
        )
      )}

      <PaymentHistoryFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        dateFocusSignal={dateFocusSignal}
      />

      <PaymentHistoryMobileList
        data={data?.rows ?? []}
        isLoading={isLoading}
        slowLoading={slowLoading}
        error={listError}
        onRetry={() => void refetch()}
        onPickDateRange={handlePickDateRange}
        onUploadStatement={handleUploadStatement}
      />
    </MobilePageShell>
  );
}
