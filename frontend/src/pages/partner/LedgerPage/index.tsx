import { useState } from 'react';
import { useLocation } from 'react-router-dom';

import { PageHeader } from '@/components/shared/PageHeader';
import { ConsolePanel } from '@/components/shared/console-panel';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import {
  PayrollReportEmailDialog,
  PayrollReportExportDialog,
  SaokeRecipientsDialog,
} from '@/components/dialogs';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useLedgerBatches, useSaokeRecipients } from '@/hooks/api/useSaoke';
import { getConsoleKicker } from '@/lib/console';
import { formatDateTime, formatFullCurrency } from '@/utils/formatters';

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
 * Sổ cái (PAYR-41, ketoan/giamdoc): company transfer-upload batches with the
 * money they settled, plus the sao kê export/email flows and their recipient
 * lists.
 */
const LedgerPage = () => {
  const location = useLocation();
  const [workMonth, setWorkMonth] = useState(currentWorkMonth);
  const [exportOpen, setExportOpen] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const [recipientsOpen, setRecipientsOpen] = useState(false);

  const { batches, summary, isLoading, isError } = useLedgerBatches(workMonth);
  const recipientsQuery = useSaokeRecipients();
  const recipients = recipientsQuery.data;

  return (
    <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
      <PageHeader
        title="Sổ cái"
        kicker={getConsoleKicker(location.pathname)}
        meta={`Tháng ${workMonth} · ${isLoading ? '…' : `${batches.length} file kết quả`}`}
        actions={[
          { label: 'Xuất sao kê', onClick: () => setExportOpen(true), variant: 'outline' },
          { label: 'Gửi sao kê qua email', onClick: () => setEmailOpen(true) },
        ]}
      >
        <TimesheetMonthSelector
          value={workMonth}
          onChange={setWorkMonth}
          className="lg:w-auto lg:shrink-0"
        />
      </PageHeader>

      <ConsolePanel
        kicker="CHUYỂN KHOẢN"
        title="File kết quả chuyển khoản"
        description={`Các lô đã tải lên trong tháng ${workMonth}`}
        isFlush
      >
        <div className="overflow-x-auto">
          <Table variant="console">
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-56">File</TableHead>
                <TableHead>Ngày tải lên</TableHead>
                <TableHead>Kỳ</TableHead>
                <TableHead className="text-right">Dòng khớp</TableHead>
                <TableHead className="text-right">Tổng số tiền</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && (
                <TableRow>
                  <TableCell colSpan={5} className="text-muted-foreground">
                    Đang tải...
                  </TableCell>
                </TableRow>
              )}
              {isError && (
                <TableRow>
                  <TableCell colSpan={5} className="text-destructive">
                    Không thể tải danh sách file. Hãy thử lại.
                  </TableCell>
                </TableRow>
              )}
              {!isLoading && !isError && batches.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-muted-foreground">
                    Chưa có file kết quả chuyển khoản nào trong tháng này.
                  </TableCell>
                </TableRow>
              )}
              {!isLoading &&
                !isError &&
                batches.map((batch) => {
                  const status = STATUS_LABELS[batch.status] ?? STATUS_LABELS.pending;
                  return (
                    <TableRow key={batch.id}>
                      <TableCell>
                        <span className="block truncate" title={batch.original_filename}>
                          {batch.original_filename}
                        </span>
                        {batch.status !== 'verified' && (
                          <Badge className={`${status.className} mt-1`}>{status.label}</Badge>
                        )}
                      </TableCell>
                      <TableCell>{formatDateTime(batch.created_at)}</TableCell>
                      <TableCell>Kỳ {batch.cycle}</TableCell>
                      <TableCell className="num">{batch.matched_rows}</TableCell>
                      <TableCell className="num">{formatFullCurrency(batch.total_amount_vnd)}</TableCell>
                    </TableRow>
                  );
                })}
            </TableBody>
          </Table>
        </div>
      </ConsolePanel>

      <ConsolePanel
        kicker="GỬI SAO KÊ"
        title="Danh sách người nhận"
        description="Địa chỉ To/CC dùng cho email sao kê."
        actions={
          <Button size="sm" variant="outline" onClick={() => setRecipientsOpen(true)}>
            Chỉnh sửa
          </Button>
        }
      >
        {recipientsQuery.isLoading ? (
          <p className="text-sm text-muted-foreground">Đang tải...</p>
        ) : recipientsQuery.isError ? (
          <p className="text-sm text-destructive">Không thể tải danh sách người nhận. Hãy thử lại.</p>
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
      </ConsolePanel>

      <PayrollReportExportDialog open={exportOpen} onOpenChange={setExportOpen} summary={summary} />
      <PayrollReportEmailDialog open={emailOpen} onOpenChange={setEmailOpen} summary={summary} />
      <SaokeRecipientsDialog open={recipientsOpen} onOpenChange={setRecipientsOpen} />
    </div>
  );
};

export default LedgerPage;
