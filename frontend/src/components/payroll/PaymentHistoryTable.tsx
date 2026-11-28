import { useMemo } from 'react';
import { DataTable } from '@/components/ui/data-table';
import { EmptyState } from '@/components/shared/EmptyState';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { RotateCw, Upload } from 'lucide-react';
import { formatDateVn, formatDateTimeVn, formatFullCurrency } from '@/utils/formatters';
import type { ColumnDef, SortingState, OnChangeFn } from '@tanstack/react-table';
import type { PaymentHistory } from '@/types/api/payroll.types';

interface PaymentHistoryTableProps {
  data: PaymentHistory[];
  isLoading?: boolean;
  /** True once the request has been loading for 3s — shows a status label. */
  slowLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  onPickDateRange?: () => void;
  onUploadStatement?: () => void;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}

/** Sort keys for the client-side sort over the merged month ledger. */
const sortValueGetters: Record<string, (row: PaymentHistory) => string | number> = {
  stt: (row) => row.stt ?? 0,
  employee_name: (row) => row.employee_name ?? '',
  ky: (row) => row.ky ?? 0,
  paid_date: (row) => row.paid_date || '',
  transfer_note: (row) => row.transfer_note ?? '',
  bank_transaction_code: (row) => row.bank_transaction_code ?? '',
  processed_at: (row) => row.processed_at || '',
  total_paid_amount: (row) => row.total_paid_amount ?? 0,
  actual_received_vnd: (row) => row.actual_received_vnd ?? 0,
};

const STT_CELL = 'tabular-nums text-muted-foreground';
const NUMERIC_CELL = 'whitespace-nowrap tabular-nums';

/** PRD §5.7 ledger columns: STT · Nhân viên · Kỳ · Ngày · Ghi chú CK · Mã GD · Xử lý · Số tiền · Thực nhận. */
export function PaymentHistoryTable({
  data,
  isLoading,
  slowLoading,
  error,
  onRetry,
  onPickDateRange,
  onUploadStatement,
  pagination,
  sorting,
  onSortingChange,
  onPageChange,
  onPageSizeChange,
}: PaymentHistoryTableProps) {
  const columns: ColumnDef<PaymentHistory>[] = useMemo(() => [
    {
      accessorKey: 'stt',
      header: 'STT',
      enableSorting: true,
      size: 40,
      cell: ({ row }) => (
        <span className={STT_CELL}>{String(row.original.stt ?? 0).padStart(2, '0')}</span>
      ),
    },
    {
      accessorKey: 'employee_name',
      header: 'Nhân viên',
      enableSorting: true,
      size: 200,
      cell: ({ row }) => (
        <div>
          <p className="max-w-[170px] truncate font-medium text-foreground" title={row.original.employee_name}>
            {row.original.employee_name}
          </p>
          <p
            className="max-w-[170px] truncate text-[11px] leading-tight tabular-nums text-muted-foreground"
            title={`${row.original.employee_cccd || '—'} · ${row.original.project_name || ''}`}
          >
            {row.original.employee_cccd || '—'}
            {row.original.project_name && (
              <span> · {row.original.project_name}</span>
            )}
          </p>
        </div>
      ),
    },
    {
      accessorKey: 'ky',
      header: 'Kỳ thanh toán',
      enableSorting: true,
      size: 110,
      cell: ({ row }) => (
        <div className="min-w-0">
          <p className={`whitespace-nowrap ${STT_CELL}`}>{row.original.ky_window || '—'}</p>
          {row.original.ky ? (
            <span className="mt-0.5 inline-block rounded-full border border-utility-gray-300 bg-utility-gray-50 px-1.5 py-0 text-[10px] font-bold text-foreground/80">
              Kỳ {row.original.ky}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      accessorKey: 'paid_date',
      header: 'Ngày thanh toán',
      enableSorting: true,
      size: 100,
      cell: ({ row }) => {
        const paidDate = row.original.paid_date;
        if (!paidDate) return <span className="text-muted-foreground">—</span>;
        return (
          <time dateTime={paidDate} className={NUMERIC_CELL} title={paidDate}>
            {formatDateVn(paidDate)}
          </time>
        );
      },
    },
    {
      accessorKey: 'transfer_note',
      header: 'Ghi chú CK',
      enableSorting: true,
      size: 120,
      cell: ({ row }) => {
        const note = row.original.transfer_note;
        if (!note) return <span className="text-muted-foreground">—</span>;
        return (
          <span className="block max-w-[110px] truncate text-muted-foreground" title={note}>
            {note}
          </span>
        );
      },
    },
    {
      accessorKey: 'bank_transaction_code',
      header: 'Mã GD ngân hàng',
      enableSorting: true,
      size: 130,
      cell: ({ row }) => {
        const code = row.original.bank_transaction_code;
        if (!code) return <span className="text-muted-foreground">—</span>;
        return (
          <span
            className="block max-w-[130px] truncate font-mono text-[12px] text-foreground"
            title={code}
          >
            {code}
          </span>
        );
      },
    },
    {
      accessorKey: 'processed_at',
      header: 'Thời gian xử lý',
      enableSorting: true,
      size: 110,
      cell: ({ row }) => {
        const processedAt = row.original.processed_at;
        if (!processedAt) return <span className="text-muted-foreground">—</span>;
        return (
          <time dateTime={processedAt} className={NUMERIC_CELL}>
            {formatDateTimeVn(processedAt)}
          </time>
        );
      },
    },
    {
      accessorKey: 'total_paid_amount',
      header: 'Số tiền (₫)',
      enableSorting: true,
      size: 105,
      meta: { align: 'right', num: true },
      cell: ({ row }) => (
        <span className={`${NUMERIC_CELL} font-semibold text-emerald-700`}>
          {formatFullCurrency(row.original.total_paid_amount)}
        </span>
      ),
    },
    {
      accessorKey: 'actual_received_vnd',
      header: 'Thực nhận (kỳ)',
      enableSorting: true,
      size: 105,
      meta: { align: 'right', num: true },
      cell: ({ row }) => {
        const actual = row.original.actual_received_vnd;
        if (actual === undefined || actual === null) {
          return <span className="text-muted-foreground">—</span>;
        }
        return (
          <span className={`${NUMERIC_CELL} text-foreground`}>{formatFullCurrency(actual)}</span>
        );
      },
    },
  ], []);

  // The ledger is paginated client-side, so sorting applies before slicing
  // the requested page.
  const pageRows = useMemo(() => {
    const first = sorting?.[0];
    const getter = first ? sortValueGetters[first.id] : undefined;
    const rows = getter
      ? [...data].sort((a, b) => {
          const valueA = getter(a);
          const valueB = getter(b);
          const cmp =
            typeof valueA === 'number' && typeof valueB === 'number'
              ? valueA - valueB
              : String(valueA).localeCompare(String(valueB), 'vi');
          return first!.desc ? -cmp : cmp;
        })
      : data;
    const page = pagination?.page ?? 1;
    const pageSize = pagination?.pageSize ?? rows.length;
    return pageSize > 0 ? rows.slice((page - 1) * pageSize, page * pageSize) : rows;
  }, [data, sorting, pagination]);

  if (error) {
    return (
      <div role="alert" className="console-panel p-4">
        <p className="text-[13px] font-semibold text-foreground">
          Không tải được lịch sử đối chiếu
        </p>
        <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground">
          Không thể truy vấn dữ liệu từ máy chủ. Vui lòng thử lại.
        </p>
        {onRetry && (
          <Button variant="outline" size="sm" className="mt-3 h-8 text-[12px]" onClick={onRetry}>
            <RotateCw className="mr-1.5 size-3.5" />
            Thử lại
          </Button>
        )}
      </div>
    );
  }

  // Loading skeleton with a 3s slow-load status label (never an endless skeleton).
  if (isLoading) {
    return (
      <div className="space-y-2">
        <div className="space-y-1">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded-md" />
          ))}
        </div>
        {slowLoading && (
          <p role="status" aria-live="polite" className="px-1 text-[12px] tabular-nums text-muted-foreground">
            Đang tải lịch sử đối chiếu…
          </p>
        )}
      </div>
    );
  }

  // Empty state with quick actions.
  if (data.length === 0) {
    return (
      <EmptyState
        title="Không tìm thấy lịch sử thanh toán"
        description="Chưa có bút toán nào được đối chiếu trong tháng này."
        size="sm"
        action={
          onPickDateRange
            ? { label: 'Chọn tháng khác', onClick: onPickDateRange }
            : undefined
        }
      >
        {onUploadStatement && (
          <Button
            variant="outline"
            size="sm"
            className="mt-2 h-8 text-[12px]"
            onClick={onUploadStatement}
          >
            <Upload className="mr-1.5 size-3.5" />
            Tải lên kết quả chuyển khoản
          </Button>
        )}
      </EmptyState>
    );
  }

  return (
    <DataTable
      data={pageRows}
      columns={columns}
      sorting={sorting}
      onSortingChange={onSortingChange}
      showPagination
      pagination={pagination ?? undefined}
      onPageChange={onPageChange}
      onPageSizeChange={onPageSizeChange}
      density="dense"
      caption="Sổ cái bút toán chuyển khoản đã đối chiếu"
    />
  );
}
