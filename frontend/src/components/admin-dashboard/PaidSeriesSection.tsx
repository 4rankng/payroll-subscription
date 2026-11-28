import { useAllProjects } from '@/hooks/api/useProjects';
import { usePaidSeries } from '@/hooks/api/useBilling';
import { PaidAmountChart, PaidEmployeesChart } from '@/components/billing/PaidSeriesChart';
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
 * from the billing module, over the last 12 months.
 */
interface PaidSeriesSectionProps {
  /**
   * Opt-in (mobile dashboards): cap each chart's legend to the top 8 projects
   * + a "Khác" aggregate in a bounded scroll container. Default off, so the
   * desktop admin surface keeps its full legend.
   */
  compactLegend?: boolean;
}

export function PaidSeriesSection({ compactLegend }: PaidSeriesSectionProps) {
  const from = monthStartIso(11);
  const to = todayIso();
  const { data: projects } = useAllProjects();
  const { data: rows, isLoading } = usePaidSeries(from, to);
  const labels = (projects ?? []).map((p) => ({ id: p.id, name: p.name }));

  return (
    <ConsolePanel className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div>
        <p className="mb-2 text-[12px] font-bold text-foreground">Nhân viên đã trả theo dự án</p>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Đang tải...</p>
        ) : (
          <PaidEmployeesChart rows={rows ?? []} projects={labels} legendLimit={compactLegend ? 8 : undefined} />
        )}
      </div>
      <div>
        <p className="mb-2 text-[12px] font-bold text-foreground">Số tiền đã trả theo dự án</p>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Đang tải...</p>
        ) : (
          <PaidAmountChart rows={rows ?? []} projects={labels} legendLimit={compactLegend ? 8 : undefined} />
        )}
      </div>
    </ConsolePanel>
  );
}
