import { EmptyState } from '@/components/shared/EmptyState';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { RotateCw, Upload } from 'lucide-react';
import { formatDateVn, formatDateTimeVn, formatFullCurrency } from '@/utils/formatters';
import type { PaymentHistory } from '@/types/api/payroll.types';

// PRD §5.7 mobile blocks: dense ledger lines — identity + amount first, then
// Kỳ window, pay/processing dates, note and bank code. Nothing hidden behind
// a tap-open; no cards, no icon chips.

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
            className="mt-3 min-h-[44px] text-[12px]"
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
          <Skeleton key={i} className="h-16 w-full rounded-md" />
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
            className="mt-2 min-h-[44px] text-[12px]"
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
    <div className="divide-y divide-utility-gray-300" role="list" aria-label="Sổ cái bút toán chuyển khoản">
      {data.map((payment) => (
        <div
          key={`${payment.employee_id}-${payment.stt ?? 0}-${payment.bank_transaction_code ?? ''}`}
          role="listitem"
          className="px-1 py-2"
        >
          <div className="flex items-baseline justify-between gap-3">
            <p className="min-w-0 break-words font-display text-[13px] font-bold leading-tight text-fg-primary">
              <span className="mr-1 tabular-nums text-fg-tertiary">
                {String(payment.stt ?? 0).padStart(2, '0')}
              </span>
              {payment.employee_name}
            </p>
            <p className="shrink-0 whitespace-nowrap text-right font-financial text-[15px] font-bold tabular-nums leading-tight text-emerald-700">
              {formatFullCurrency(payment.total_paid_amount)}
            </p>
          </div>

          <p className="mt-0.5 break-words text-[11px] leading-snug text-fg-tertiary">
            CCCD {payment.employee_cccd || '—'}
            {payment.project_name && <> · {payment.project_name}</>}
          </p>

          <p className="mt-0.5 flex flex-wrap items-center gap-x-2 text-[11px] leading-snug text-fg-tertiary">
            {payment.ky_window && (
              <span className="tabular-nums">
                {payment.ky_window}
                {payment.ky ? (
                  <span className="ml-1 font-semibold text-fg-secondary">Kỳ {payment.ky}</span>
                ) : null}
              </span>
            )}
            {payment.paid_date && (
              <time dateTime={payment.paid_date} className="tabular-nums">
                Trả {formatDateVn(payment.paid_date)}
              </time>
            )}
            {payment.processed_at && (
              <time dateTime={payment.processed_at} className="tabular-nums">
                XL {formatDateTimeVn(payment.processed_at)}
              </time>
            )}
          </p>

          <p className="mt-0.5 flex flex-wrap items-baseline gap-x-2 text-[11px] leading-snug text-fg-tertiary">
            {payment.transfer_note && (
              <span className="min-w-0 truncate" title={payment.transfer_note}>
                CK: {payment.transfer_note}
              </span>
            )}
            {payment.bank_transaction_code && (
              <span className="shrink-0 break-all font-mono text-fg-secondary">
                {payment.bank_transaction_code}
              </span>
            )}
            <span className="ml-auto shrink-0 whitespace-nowrap tabular-nums text-fg-secondary">
              Thực nhận:{' '}
              {payment.actual_received_vnd !== undefined && payment.actual_received_vnd !== null
                ? formatFullCurrency(payment.actual_received_vnd)
                : '—'}
            </span>
          </p>
        </div>
      ))}
    </div>
  );
}
