import { useState, useMemo, useCallback, useEffect } from 'react';
import { format, startOfMonth, endOfMonth } from 'date-fns';
import { useNavigate } from 'react-router-dom';
import { Download } from 'lucide-react';
import { PageHeader } from '@/components/shared/PageHeader';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PaymentHistoryTable } from '@/components/payroll/PaymentHistoryTable';
import { PaymentHistoryFilters } from '@/components/payroll/PaymentHistoryFilters';
import { useBankTransferHistories, useExportBankTransferHistories } from '@/hooks/api/usePaymentHistory';
import { formatFullCurrency, formatInt } from '@/utils/formatters';
import type { SortingState } from '@tanstack/react-table';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

/** The statement-upload entry (Tải lên kết quả chuyển khoản) lives here. */
const STATEMENT_UPLOAD_PATH = '/timesheet';

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
        <p className="console-value mt-1 text-lg font-semibold">{formatInt(summary?.bank_entry_count ?? 0)}</p>
      </div>
      <div className="console-panel px-3 py-2">
        <p className="console-kicker">Nhân viên</p>
        <p className="console-value mt-1 text-lg font-semibold">{formatInt(summary?.employee_count ?? 0)}</p>
      </div>
    </div>
  );
}

export default function PaymentHistoryPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<FiltersType>(() => ({
    fromDate: format(startOfMonth(new Date()), 'yyyy-MM-dd'),
    toDate: format(endOfMonth(new Date()), 'yyyy-MM-dd'),
    page: 1,
    pageSize: 50,
  }));
  const [sorting, setSorting] = useState<SortingState>([{ id: 'processed_at', desc: true }]);
  const [dateFocusSignal, setDateFocusSignal] = useState(0);

  const { data, isLoading, isError, error, refetch } = useBankTransferHistories(filters);
  const exportMutation = useExportBankTransferHistories();

  // The UI shows a fixed Vietnamese message — the raw failure is
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

  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }));
  }, []);

  const handlePageSizeChange = useCallback((pageSize: number) => {
    setFilters((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const handlePickDateRange = useCallback(() => {
    setDateFocusSignal((signal) => signal + 1);
  }, []);

  const handleUploadStatement = useCallback(() => {
    navigate(STATEMENT_UPLOAD_PATH);
  }, [navigate]);

  const hasFilters = useMemo(() => {
    return !!(filters.search || filters.projectId?.length || filters.employeeId?.length || filters.position);
  }, [filters]);

  const tableError = isError
    ? error instanceof Error
      ? error
      : new Error('Không thể truy vấn dữ liệu từ máy chủ.')
    : null;

  return (
    <div className="space-y-3 p-3 sm:p-4">
      <PageHeader
        title="Lịch sử trả lương"
        meta="Bút toán chuyển khoản đã đối chiếu"
        metaVariant="sans"
      >
        <Button variant="outline" size="sm" onClick={handleExport} disabled={exportMutation.isPending}>
          <Download className="mr-1.5 size-3.5" />
          Xuất Excel
        </Button>
      </PageHeader>

      {!isError && <SummaryStrip summary={data?.summary} isLoading={isLoading} />}

      <PaymentHistoryFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        dateFocusSignal={dateFocusSignal}
      />

      <PaymentHistoryTable
        data={data?.rows ?? []}
        isLoading={isLoading}
        slowLoading={slowLoading}
        error={tableError}
        onRetry={() => void refetch()}
        onPickDateRange={handlePickDateRange}
        onUploadStatement={handleUploadStatement}
        pagination={{
          page: data?.page ?? 1,
          pageSize: data?.page_size ?? 50,
          totalPages: Math.ceil((data?.total ?? 0) / (data?.page_size ?? 50)),
          totalRecords: data?.total ?? 0,
        }}
        sorting={sorting}
        onSortingChange={setSorting}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
      />
    </div>
  );
}
