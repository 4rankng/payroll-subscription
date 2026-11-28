import { useMemo, useState } from 'react';
import { useAllProjects } from '@/hooks/api/useProjects';
import { usePaidSeries } from '@/hooks/api/useBilling';
import {
  PaidAmountChart,
  PaidEmployeesChart,
  PaidSeriesLegend,
  buildPaidSeriesDefs,
  type PaidSeriesDef,
  type ProjectLabel,
} from '@/components/billing/PaidSeriesChart';
import { ConsolePanel } from '@/components/shared/console-panel';
import { cn } from '@/lib/utils';

/** First day of the month `monthsBack` before today, ISO yyyy-mm-dd. */
function monthStartIso(monthsBack: number): string {
  const d = new Date();
  d.setDate(1);
  d.setMonth(d.getMonth() - monthsBack);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`;
}

function todayIso(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/**
 * FR-6 Tổng quan section: per-day paid employees and paid amount per project
 * from the billing module. With `month`, both charts synchronize to that
 * month's window (DASH-01); without it they cover the last 12 months.
 *
 * Both charts share one series set (top projects + "Khác", ranked by paid
 * amount) and one HTML legend rendered above them in normal flow, so the
 * legend can never steal chart height or collide with the section below.
 */
interface PaidSeriesSectionProps {
  /** yyyy-mm period from the page's month selector; omit for 12 months. */
  month?: string;
  /**
   * 'split' renders both charts side by side (admin dashboard default).
   * 'segmented' renders one full-width chart with a metric toggle
   * (partner dashboard, DASH-01).
   */
  layout?: 'split' | 'segmented';
}

export function PaidSeriesSection({ month, layout = 'split' }: PaidSeriesSectionProps) {
  const [metric, setMetric] = useState<'amount' | 'employee_count'>('amount');
  const range = useMemo(() => {
    if (!month) return { from: monthStartIso(11), to: todayIso() };
    const [y, m] = month.split('-').map(Number);
    const lastDay = new Date(y, m, 0).getDate();
    const to = `${month}-${String(lastDay).padStart(2, '0')}`;
    const today = todayIso();
    return { from: `${month}-01`, to: to > today ? today : to };
  }, [month]);
  const { data: projects } = useAllProjects();
  const { data: rows, isLoading } = usePaidSeries(range.from, range.to);
  const labels = useMemo<ProjectLabel[]>(
    () => (projects ?? []).map((p) => ({ id: p.id, name: p.name })),
    [projects],
  );
  const paidRows = useMemo(() => rows ?? [], [rows]);
  const series = useMemo<PaidSeriesDef[]>(
    () => (paidRows.length > 0 ? buildPaidSeriesDefs(paidRows, labels) : []),
    [paidRows, labels],
  );

  return (
    <ConsolePanel className="flex flex-col gap-3">
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Đang tải...</p>
      ) : paidRows.length === 0 ? (
        <p className="text-sm text-muted-foreground">Chưa có dữ liệu thanh toán</p>
      ) : (
        <>
          <PaidSeriesLegend series={series} />
          {layout === 'segmented' ? (
            /* DASH-01 (partner): one full-width chart, metric picked by the
               h-8 segmented toggle — no duplicate headings or half-width
               canvases. */
            <div className="flex flex-col gap-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-[12px] font-bold text-foreground">Đã trả theo dự án</p>
                <div
                  className="flex h-8 items-center rounded-lg bg-muted/60 p-1"
                  role="group"
                  aria-label="Chọn chỉ số biểu đồ"
                >
                  {(
                    [
                      ['amount', 'Số tiền đã trả (₫)'],
                      ['employee_count', 'Nhân viên đã trả'],
                    ] as const
                  ).map(([value, label]) => (
                    <button
                      type="button"
                      key={value}
                      onClick={() => setMetric(value)}
                      aria-pressed={metric === value}
                      className={cn(
                        'h-7 rounded-md px-2.5 text-[11px] font-semibold transition-[background-color,color,box-shadow]',
                        metric === value
                          ? 'bg-card text-foreground shadow-sm ring-1 ring-inset ring-border/60'
                          : 'text-muted-foreground hover:bg-card/70 hover:text-foreground',
                      )}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </div>
              {metric === 'amount' ? (
                <PaidAmountChart rows={paidRows} series={series} />
              ) : (
                <PaidEmployeesChart rows={paidRows} series={series} />
              )}
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <div className="min-w-0">
                <p className="mb-2 text-[12px] font-bold text-foreground">
                  Nhân viên đã trả theo dự án
                </p>
                <PaidEmployeesChart rows={paidRows} series={series} />
              </div>
              <div className="min-w-0">
                <p className="mb-2 text-[12px] font-bold text-foreground">Số tiền đã trả theo dự án</p>
                <PaidAmountChart rows={paidRows} series={series} />
              </div>
            </div>
          )}
        </>
      )}
    </ConsolePanel>
  );
}
