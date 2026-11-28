import { useMemo } from 'react';
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
}

export function PaidSeriesSection({ month }: PaidSeriesSectionProps) {
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
  const paidRows = rows ?? [];
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
        </>
      )}
    </ConsolePanel>
  );
}
