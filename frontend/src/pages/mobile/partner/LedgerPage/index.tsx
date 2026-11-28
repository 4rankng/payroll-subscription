import { useMemo, useState } from 'react';

import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import {
  PayrollReportEmailDialog,
  PayrollReportExportDialog,
  SaokeRecipientsDialog,
} from '@/components/dialogs';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useLedgerBatches, useSaokeRecipients } from '@/hooks/api/useSaoke';
import { formatDateTime, formatFullCurrency } from '@/utils/formatters';
import { seedSaokeRecipients } from '@/utils/saoke';

const currentWorkMonth = (): string => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
};

const STATUS_LABELS: Record<string, { label: string; className: string }> = {
  verified: { label: 'Đã xác minh', className: 'bg-emerald-100 text-emerald-800' },
  pending: { label: 'Chờ xử lý', className: 'bg-slate-100 text-slate-700' },
  rejected: { label: 'Từ chối', className: 'bg-red-100 text-red-800' },
};

/**
 * Sổ cái mobile: same data and actions as the desktop page —
 * transfer batches, sao kê export/email, recipient lists.
 */
const LedgerPageMobile = () => {
  const [workMonth, setWorkMonth] = useState(currentWorkMonth);
  const [exportOpen, setExportOpen] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const [recipientsOpen, setRecipientsOpen] = useState(false);

  const { batches, summary, isLoading, isError } = useLedgerBatches(workMonth);
  const recipientsQuery = useSaokeRecipients();
  // Same hydration the Chỉnh sửa dialog applies (canned defaults on first
  // run) so the card can never contradict the dialog it opens.
  const recipients = useMemo(
    () => (recipientsQuery.data === undefined ? undefined : seedSaokeRecipients(recipientsQuery.data)),
    [recipientsQuery.data],
  );

  return (
    <MobilePageShell>
      <MobilePageHeader
        title="Sổ cái"
        subtitle={`${workMonth === 'all' ? 'Tất cả các tháng' : `Tháng ${workMonth}`} · ${isLoading ? '…' : `${batches.length} file kết quả`}`}
      />

      <div className="space-y-3">
        <TimesheetMonthSelector value={workMonth} onChange={setWorkMonth} allowAll={false} />

        <div className="grid grid-cols-1 gap-2">
          <Button
            type="button"
            variant="outline"
            className="min-h-11 w-full sm:min-h-9"
            onClick={() => setExportOpen(true)}
          >
            Xuất sao kê
          </Button>
          <Button
            type="button"
            className="min-h-11 w-full sm:min-h-9"
            onClick={() => setEmailOpen(true)}
          >
            Gửi sao kê qua email
          </Button>
        </div>

        <MobileSurface>
          <div className="space-y-3 p-3">
            <div>
              <p className="console-kicker">CHUYỂN KHOẢN</p>
              <h2 className="font-display text-[13px] font-semibold text-foreground">
                File kết quả chuyển khoản
              </h2>
              <p className="mt-0.5 text-[11px] text-muted-foreground">
                Các lô đã tải lên trong tháng {workMonth}
              </p>
            </div>

            {isLoading && <p className="text-sm text-muted-foreground">Đang tải...</p>}
            {isError && (
              <p className="text-sm text-destructive">Không thể tải danh sách file. Hãy thử lại.</p>
            )}
            {!isLoading && !isError && batches.length === 0 && (
              <p className="text-sm text-muted-foreground">
                Chưa có file kết quả chuyển khoản nào trong tháng này.
              </p>
            )}
            {!isLoading &&
              !isError &&
              batches.map((batch) => {
                const status = STATUS_LABELS[batch.status] ?? STATUS_LABELS.pending;
                return (
                  <div key={batch.id} className="space-y-1.5 border-b border-border/70 pb-3 last:border-b-0 last:pb-0">
                    <div className="flex items-start justify-between gap-2">
                      <p className="break-all text-sm font-semibold text-foreground">
                        {batch.original_filename}
                      </p>
                      {batch.status !== 'verified' && (
                        <Badge className={status.className}>{status.label}</Badge>
                      )}
                    </div>
                    <dl className="space-y-1 text-sm">
                      <div className="flex items-baseline justify-between gap-3">
                        <dt className="text-xs text-muted-foreground">Ngày tải lên</dt>
                        <dd className="text-right text-foreground">{formatDateTime(batch.created_at)}</dd>
                      </div>
                      <div className="flex items-baseline justify-between gap-3">
                        <dt className="text-xs text-muted-foreground">Kỳ</dt>
                        <dd className="text-right text-foreground">Kỳ {batch.cycle}</dd>
                      </div>
                      <div className="flex items-baseline justify-between gap-3">
                        <dt className="text-xs text-muted-foreground">Dòng khớp</dt>
                        <dd className="console-value text-right text-foreground">{batch.matched_rows}</dd>
                      </div>
                      <div className="flex items-baseline justify-between gap-3">
                        <dt className="text-xs text-muted-foreground">Tổng số tiền</dt>
                        <dd className="console-value text-right font-semibold text-foreground">
                          {formatFullCurrency(batch.total_amount_vnd)}
                        </dd>
                      </div>
                    </dl>
                  </div>
                );
              })}
          </div>
        </MobileSurface>

        <MobileSurface>
          <div className="space-y-3 p-3">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="console-kicker">GỬI SAO KÊ</p>
                <h2 className="font-display text-[13px] font-semibold text-foreground">
                  Danh sách người nhận
                </h2>
                <p className="mt-0.5 text-[11px] text-muted-foreground">
                  Địa chỉ To/CC dùng cho email sao kê.
                </p>
              </div>
              <Button
                size="sm"
                variant="outline"
                className="min-h-11 shrink-0 sm:min-h-9"
                onClick={() => setRecipientsOpen(true)}
              >
                Chỉnh sửa
              </Button>
            </div>

            {recipientsQuery.isLoading ? (
              <p className="text-sm text-muted-foreground">Đang tải...</p>
            ) : recipientsQuery.isError ? (
              <p className="text-sm text-destructive">
                Không thể tải danh sách người nhận. Hãy thử lại.
              </p>
            ) : !recipients || (recipients.to.length === 0 && recipients.cc.length === 0) ? (
              <p className="text-sm text-muted-foreground">
                Chưa cấu hình người nhận. Bấm “Chỉnh sửa” để thiết lập.
              </p>
            ) : (
              <div className="space-y-2">
                <div>
                  <p className="console-kicker">To</p>
                  <p className="break-all text-sm text-foreground">{recipients.to.join(', ') || '—'}</p>
                </div>
                <div>
                  <p className="console-kicker">CC</p>
                  <p className="break-all text-sm text-foreground">{recipients.cc.join(', ') || '—'}</p>
                </div>
              </div>
            )}
          </div>
        </MobileSurface>
      </div>

      <PayrollReportExportDialog open={exportOpen} onOpenChange={setExportOpen} summary={summary} />
      <PayrollReportEmailDialog open={emailOpen} onOpenChange={setEmailOpen} summary={summary} />
      <SaokeRecipientsDialog open={recipientsOpen} onOpenChange={setRecipientsOpen} />
    </MobilePageShell>
  );
};

export default LedgerPageMobile;
