import { formatFullCurrency } from '@/utils/formatters';
import { formatReportDate, type SaokeSummary } from '@/utils/saoke';

interface SaokePreviewPanelProps {
  summary: SaokeSummary;
  reportDate: string;
}

/**
 * The "Xem trước" block shared by the export and email dialogs: the report
 * date plus what the Sổ cái currently holds (batches, matched rows, money).
 */
export function SaokePreviewPanel({ summary, reportDate }: SaokePreviewPanelProps) {
  return (
    <div className="space-y-2 rounded-lg border border-border bg-muted/30 p-3">
      <p className="console-kicker">Xem trước</p>
      <p className="text-sm text-foreground">
        Ngày báo cáo:{' '}
        <span className="console-value font-semibold">{formatReportDate(reportDate)}</span>
      </p>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-1.5 text-sm sm:grid-cols-3">
        <div className="flex items-baseline justify-between gap-2 sm:block">
          <dt className="text-xs text-muted-foreground">File kết quả</dt>
          <dd className="console-value font-semibold text-foreground">{summary.batchCount}</dd>
        </div>
        <div className="flex items-baseline justify-between gap-2 sm:block">
          <dt className="text-xs text-muted-foreground">Dòng khớp</dt>
          <dd className="console-value font-semibold text-foreground">{summary.matchedRows}</dd>
        </div>
        <div className="flex items-baseline justify-between gap-2 sm:block">
          <dt className="text-xs text-muted-foreground">Tổng số tiền</dt>
          <dd className="console-value font-semibold text-foreground">
            {formatFullCurrency(summary.totalAmountVnd)}
          </dd>
        </div>
      </dl>
      <p className="text-xs leading-4 text-muted-foreground">
        Số liệu sổ cái tháng {summary.workMonth} — tổng hợp từ các file kết quả chuyển khoản đã tải
        lên.
      </p>
    </div>
  );
}
