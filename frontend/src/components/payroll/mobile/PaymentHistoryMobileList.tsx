import { EmptyState } from '@/components/shared/EmptyState';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { RotateCw, Upload } from 'lucide-react';
import { formatDateVn, formatDateTimeVn, formatFullCurrency } from '@/utils/formatters';
import type { PaymentHistory } from '@/types/api/payroll.types';

/** Every row in this ledger passed transfer-upload verification. */
const RECONCILED_LABEL = 'Đã đối chiếu';

// Dense ledger rows (UU bridge tokens): identity + amount on the first line,
// metadata on the second — no cards, no icon chips, no wasted vertical space.

interface PaymentHistoryMobileListProps {
  data: PaymentHistory[];
  isLoading?: boolean;
  /** True once the request has been loading for 3s — shows a status label. */
  slowLoading?: boolean;
  error?: Error | null;
  onRetry?: () => void;
  onPickDateRange?: () => void;
  onUploadStatement?: () => void;
}

export function PaymentHistoryMobileList({
  data,
  isLoading,
  slowLoading,
  error,
  onRetry,
  onPickDateRange,
  onUploadStatement,
}: PaymentHistoryMobileListProps) {
  if (error) {
    return (
      <div role="alert" className="console-panel p-3">
        <p className="text-[13px] font-semibold text-foreground">
          Không tải được lịch sử đối chiếu
        </p>
        <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground">
          Không thể truy vấn dữ liệu từ máy chủ. Vui lòng thử lại.
        </p>
        {onRetry && (
          <Button
            variant="outline"
            size="sm"
            className="mt-3 h-8 text-[12px]"
            onClick={onRetry}
          >
            <RotateCw className="mr-1.5 size-3.5" />
            Thử lại
          </Button>
        )}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-1" aria-label="Đang tải lịch sử thanh toán" aria-busy="true">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-14 w-full rounded-md" />
        ))}
        {slowLoading && (
          <p role="status" aria-live="polite" className="px-1 text-[12px] tabular-nums text-muted-foreground">
            Đang tải lịch sử đối chiếu…
          </p>
        )}
      </div>
    );
  }

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
            className="mt-2 min-h-[44px] text-[12px]"
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
    <div className="divide-y divide-utility-gray-300" role="list" aria-label="Lịch sử thanh toán">
      {data.map((payment, index) => (
        <div
          key={`${payment.employee_id}-${payment.paid_date}-${payment.bank_transaction_code ?? ''}-${index}`}
          role="listitem"
          className="px-1 py-2"
        >
          <div className="flex items-baseline justify-between gap-3">
            <p className="min-w-0 break-words font-display text-[13px] font-bold leading-tight text-fg-primary">
              {payment.employee_name}
            </p>
            <p className="shrink-0 whitespace-nowrap font-financial text-[15px] font-bold tabular-nums leading-tight text-emerald-700">
              {formatFullCurrency(payment.total_paid_amount)}
            </p>
          </div>

          <p className="mt-0.5 break-words text-[11px] leading-snug text-fg-tertiary">
            <time
              dateTime={payment.processed_at || payment.paid_date || undefined}
              className="tabular-nums"
            >
              {payment.processed_at
                ? formatDateTimeVn(payment.processed_at)
                : payment.paid_date
                  ? formatDateVn(payment.paid_date)
                  : '—'}
            </time>
            {payment.project_name && <> · {payment.project_name}</>}
          </p>

          {(payment.bank_transaction_code || payment.transfer_note) && (
            <p className="mt-0.5 flex items-center gap-1.5 text-[11px] leading-snug text-fg-tertiary">
              {payment.bank_transaction_code && (
                <span className="shrink-0 font-mono">{payment.bank_transaction_code}</span>
              )}
              {payment.transfer_note && (
                <span className="min-w-0 truncate" title={payment.transfer_note}>
                  {payment.transfer_note}
                </span>
              )}
              <Badge
                variant="success"
                className="ml-auto shrink-0 text-[11px] font-semibold leading-[1.35]"
              >
                {RECONCILED_LABEL}
              </Badge>
            </p>
          )}
        </div>
      ))}
    </div>
  );
}
