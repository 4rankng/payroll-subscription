import { useMemo } from 'react';
import { DataTable } from '@/components/ui/data-table';
import { EmptyState } from '@/components/shared/EmptyState';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { RotateCw, Upload } from 'lucide-react';
import { formatDateVn, formatDateTimeVn, formatFullCurrency } from '@/utils/formatters';
import type { ColumnDef, SortingState, OnChangeFn } from '@tanstack/react-table';
import type { PaymentHistory } from '@/types/api/payroll.types';

/** Every row in this ledger passed transfer-upload verification. */
const RECONCILED_LABEL = 'Đã đối chiếu';

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

/** Sort keys for the client-side sort over merged month ledgers. */
const sortValueGetters: Record<string, (row: PaymentHistory) => string | number> = {
  processed_at: (row) => row.processed_at || row.paid_date || '',
  bank_transaction_code: (row) => row.bank_transaction_code ?? '',
  employee_name: (row) => row.employee_name ?? '',
  project_name: (row) => row.project_name ?? '',
  transfer_note: (row) => row.transfer_note ?? '',
  total_paid_amount: (row) => row.total_paid_amount ?? 0,
};

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
      id: 'processed_at',
      header: 'Ngày CK',
      enableSorting: true,
      size: 150,
      cell: ({ row }) => {
        const payment = row.original;
        const text = payment.processed_at
          ? formatDateTimeVn(payment.processed_at)
          : payment.paid_date
            ? formatDateVn(payment.paid_date)
            : '—';
        return (
          <time
            dateTime={payment.processed_at || payment.paid_date || undefined}
            className="whitespace-nowrap tabular-nums text-muted-foreground"
          >
            {text}
          </time>
        );
      },
    },
    {
      accessorKey: 'bank_transaction_code',
      header: 'Mã bút toán',
      enableSorting: true,
      size: 150,
      cell: ({ row }) => {
        const code = row.original.bank_transaction_code;
        if (!code) return <span className="text-muted-foreground">—</span>;
        return (
          <span
            className="block max-w-[150px] truncate font-mono text-[12px] text-foreground"
            title={code}
          >
            {code}
          </span>
        );
      },
    },
    {
      accessorKey: 'employee_name',
      header: 'Nhân viên',
      enableSorting: true,
      size: 170,
      cell: ({ row }) => (
        <div className="min-w-0">
          <p className="truncate font-medium text-foreground" title={row.original.employee_name}>
            {row.original.employee_name}
          </p>
          {row.original.employee_cccd && (
            <p
              className="truncate text-[11px] leading-tight text-muted-foreground tabular-nums"
              title={row.original.employee_cccd}
            >
              {row.original.employee_cccd}
            </p>
          )}
        </div>
      ),
    },
    {
      accessorKey: 'project_name',
      header: 'Dự án',
      enableSorting: true,
      size: 140,
      cell: ({ row }) => (
        <span className="line-clamp-2 text-foreground" title={row.original.project_name}>
          {row.original.project_name || '—'}
        </span>
      ),
    },
    {
      accessorKey: 'transfer_note',
      header: 'Ngân hàng',
      enableSorting: true,
      size: 160,
      cell: ({ row }) => {
        const note = row.original.transfer_note;
        if (!note) return <span className="text-muted-foreground">—</span>;
        return (
          <span className="block max-w-[160px] truncate text-muted-foreground" title={note}>
            {note}
          </span>
        );
      },
    },
    {
      accessorKey: 'total_paid_amount',
      header: 'Số tiền (₫)',
      enableSorting: true,
      size: 130,
      meta: { align: 'right', num: true },
      cell: ({ row }) => (
        <span className="whitespace-nowrap font-semibold tabular-nums text-emerald-700">
          {formatFullCurrency(row.original.total_paid_amount)}
        </span>
      ),
    },
    {
      id: 'status',
      header: 'Trạng thái',
      enableSorting: false,
      size: 110,
      cell: () => (
        <Badge variant="success" className="text-[11px] font-semibold leading-[1.35]">
          {RECONCILED_LABEL}
        </Badge>
      ),
    },
  ], []);

  // Months are merged in the hook and paginated client-side, so sorting also
  // applies here before slicing the requested page.
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
        description="Chưa có lịch sử thanh toán nào trong khoảng thời gian này."
        size="sm"
        action={
          onPickDateRange
            ? { label: 'Chọn khoảng thời gian khác', onClick: onPickDateRange }
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
            Tải lên sao kê
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
      caption="Bảng lịch sử đối chiếu chuyển khoản"
    />
  );
}
