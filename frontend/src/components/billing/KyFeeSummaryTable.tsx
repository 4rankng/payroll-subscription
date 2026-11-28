import { memo, useMemo } from 'react';
import { format, parseISO } from 'date-fns';
import { vi } from 'date-fns/locale';
import { UploadCloud01 } from '@untitledui/icons';

import { useKySummary } from '@/hooks/api/useBilling';
import { formatFullCurrency as formatVND } from '@/utils/formatters';
import { ConsolePanel } from '@/components/shared/console-panel';
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/lib/utils';

const KY_COUNT = 4;

type KyVariant = 'desktop' | 'mobile';

interface KyFeeSummaryTableProps {
  /** Rendering context — controls density and typography only. */
  variant?: KyVariant;
  /** How many trailing work months to show, newest last. */
  months?: number;
  className?: string;
}

function formatMonthLabel(workMonth: string): string {
  try {
    return format(parseISO(`${workMonth}-01`), 'MMM yyyy', { locale: vi });
  } catch {
    return workMonth;
  }
}

function cycleCountsRow(counts: Record<string, number>): number[] {
  const row: number[] = [];
  for (let cycle = 1; cycle <= KY_COUNT; cycle++) {
    row.push(counts[String(cycle)] ?? 0);
  }
  return row;
}

/**
 * FR-6 (3): per-month per-Kỳ table — distinct paid employees per Kỳ of each
 * work month, priced at the admin-configured subscription fee (BR-1, default
 * 10,000 VND/employee/Kỳ). Fed by GET /billing/dashboard/ky-summary, which
 * snapshots the fee server-side.
 */
export const KyFeeSummaryTable = memo(function KyFeeSummaryTable({
  variant = 'desktop',
  months = 12,
  className,
}: KyFeeSummaryTableProps) {
  const isMobile = variant === 'mobile';
  const { data, isLoading, isError } = useKySummary(months);

  const rows = useMemo(
    () =>
      (data?.months ?? []).map((month) => {
        const counts = cycleCountsRow(month.counts);
        const employeeKy = counts.reduce((sum, count) => sum + count, 0);
        return {
          workMonth: month.work_month,
          label: formatMonthLabel(month.work_month),
          counts,
          employeeKy,
          feeVnd: employeeKy * (data?.fee_per_employee_per_ky_vnd ?? 0),
        };
      }),
    [data],
  );

  const totalFeeVnd = useMemo(() => rows.reduce((sum, row) => sum + row.feeVnd, 0), [rows]);
  const hasAnyRecord = rows.some((row) => row.employeeKy > 0);

  const cellSize = isMobile ? 'px-2 py-1.5 text-xs' : 'px-2.5 py-2 text-[12px]';
  const headSize = isMobile ? 'px-2 py-1.5 text-[11px]' : 'px-2.5 py-2 text-[11px] font-semibold leading-[1.35] tracking-[0.03em]';

  return (
    /* DASH-03: desktop variant fills the grid row height so both
       summary cards share bottom edges; mobile keeps its natural height. */
    <ConsolePanel className={cn('flex flex-col gap-3', !isMobile && 'h-full', className)}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className={cn('font-bold text-foreground', isMobile ? 'text-[13px]' : 'text-sm')}>
          NV theo kỳ trả và phí
        </p>
        {data !== undefined && (
          <p className="text-xs text-muted-foreground">
            Phí: <span className="font-semibold text-foreground">{formatVND(data.fee_per_employee_per_ky_vnd)}</span>
            /NV/kỳ
          </p>
        )}
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Đang tải số liệu kỳ…</p>
      ) : isError ? (
        <p className="text-sm text-destructive">Không thể tải số liệu theo kỳ. Hãy làm mới để thử lại.</p>
      ) : !hasAnyRecord ? (
        isMobile ? (
          <p className="text-sm text-muted-foreground">
            Chưa có kỳ nào được xác nhận thanh toán — số liệu xuất hiện sau khi kết quả chuyển khoản được tải lên và
            đối chiếu.
          </p>
        ) : (
          /* Desktop empty state: centered icon + informative copy only.
             Upload/đối soát chuyển khoản stays on its own surface — it is
             not a dashboard action (owner ruling). */
          <div className="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-center">
            <UploadCloud01 className="h-6 w-6 text-muted-foreground/60" aria-hidden="true" />
            <p className="max-w-xs text-xs leading-4 text-muted-foreground">
              Chưa có kỳ nào được xác nhận thanh toán — số liệu xuất hiện sau khi kết quả chuyển khoản được tải lên và
              đối chiếu.
            </p>
          </div>
        )
      ) : (
        <div className={cn('overflow-x-auto', !isMobile && 'flex-1')}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className={cn(headSize, 'sticky left-0 bg-card')}>Tháng</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>Kỳ 1</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>Kỳ 2</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>Kỳ 3</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>Kỳ 4</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>NV × kỳ</TableHead>
                <TableHead className={cn(headSize, 'text-right')}>Phí tháng</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.workMonth}>
                  <TableCell className={cn(cellSize, 'sticky left-0 bg-card font-medium')}>{row.label}</TableCell>
                  {row.counts.map((count, index) => (
                    <TableCell key={index} className={cn(cellSize, 'text-right tabular-nums')}>
                      {count > 0 ? count.toLocaleString('vi-VN') : '—'}
                    </TableCell>
                  ))}
                  <TableCell className={cn(cellSize, 'text-right tabular-nums')}>{row.employeeKy}</TableCell>
                  <TableCell className={cn(cellSize, 'text-right font-semibold tabular-nums')}>
                    {row.feeVnd > 0 ? formatVND(row.feeVnd) : '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
            <TableFooter>
              <TableRow>
                <TableCell className={cn(cellSize, 'sticky left-0 bg-card')}>Tổng {rows.length} tháng</TableCell>
                <TableCell colSpan={5} className={cellSize} />
                <TableCell className={cn(cellSize, 'text-right font-bold tabular-nums')}>
                  {formatVND(totalFeeVnd)}
                </TableCell>
              </TableRow>
            </TableFooter>
          </Table>
        </div>
      )}
    </ConsolePanel>
  );
});
