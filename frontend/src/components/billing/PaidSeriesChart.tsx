import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { format, parseISO } from 'date-fns';

import { formatCurrency } from '@/utils/formatters';
import type { PaidSeriesRow } from '@/services/api/billing.service';

export interface ProjectLabel {
  id: number;
  name: string;
}

interface PaidSeriesChartProps {
  rows: PaidSeriesRow[];
  projects: ProjectLabel[];
  metric: 'employee_count' | 'amount';
  label: string;
  /**
   * Opt-in legend cap (mobile dashboards): rank projects by their total over
   * the window, keep the top `legendLimit` series and aggregate everything
   * else into one "Khác" series. The legend renders in a bounded scroll
   * container so it can never overflow into the section below. Default is
   * off — the desktop admin surface keeps the full per-project legend.
   */
  legendLimit?: number;
}

function metricOf(row: PaidSeriesRow, metric: 'employee_count' | 'amount'): number {
  return metric === 'amount' ? row.amount : row.employee_count;
}

function projectNameOf(projects: ProjectLabel[], projectId: number, fallback?: string): string {
  return projects.find((p) => p.id === projectId)?.name ?? fallback ?? '(dự án)';
}

/**
 * FR-6 time-series chart: one bar series per project on the payment-date
 * axis. Metric selects employees vs amount; explicit variant components
 * below are the only intended entry points.
 */
function formatChartDate(iso: string): string {
  try {
    return format(parseISO(iso), 'dd/MM');
  } catch {
    return iso.slice(0, 10);
  }
}

interface SeriesDef {
  name: string;
  /** Project ids aggregated into this series (one, or many for "Khác"). */
  projectIds: number[];
  color: string;
}

/** Bounded vertical legend: scrolls instead of pushing into the next section. */
function BoundedLegend({ payload }: { payload?: Array<{ value?: string; color?: string }> }) {
  if (!payload?.length) return null;
  return (
    <ul className="mx-auto mt-1 flex max-h-24 max-w-full flex-col gap-0.5 overflow-y-auto px-2 text-[11px] leading-4">
      {payload.map((entry) => (
        <li key={entry.value} className="flex min-w-0 items-center gap-1.5">
          <span
            aria-hidden="true"
            className="h-2 w-2 shrink-0 rounded-full"
            style={{ backgroundColor: entry.color }}
          />
          <span className="min-w-0 truncate text-slate-600">{entry.value}</span>
        </li>
      ))}
    </ul>
  );
}

function SeriesChart({ rows, projects, metric, label, legendLimit }: PaidSeriesChartProps) {
  if (rows.length === 0) {
    return <p className="text-sm text-slate-600">Chưa có dữ liệu thanh toán</p>;
  }

  const dates = [...new Set(rows.map((r) => r.date))].sort();
  const byProject = new Map<number, Record<string, number>>();
  // The API enriches each row with the project name/code (PAYR-61) — the
  // projects prop is only a last-resort lookup for older cached responses.
  const rowNames = new Map<number, string>();
  for (const row of rows) {
    const name = row.project_name ?? row.project_code;
    if (name && !rowNames.has(row.project_id)) rowNames.set(row.project_id, name);
  }
  for (const row of rows) {
    const slot = byProject.get(row.project_id) ?? {};
    slot[row.date] = (slot[row.date] ?? 0) + metricOf(row, metric);
    byProject.set(row.project_id, slot);
  }

  // Rank projects by their window total so the legend cap keeps the
  // heaviest payers and folds the long tail into "Khác".
  const totals = new Map<number, number>();
  for (const [projectId, slot] of byProject) {
    totals.set(projectId, Object.values(slot).reduce((sum, v) => sum + v, 0));
  }
  const ranked = [...byProject.keys()].sort(
    (a, b) => (totals.get(b) ?? 0) - (totals.get(a) ?? 0),
  );

  let series: SeriesDef[];
  if (legendLimit && ranked.length > legendLimit) {
    const top = ranked.slice(0, legendLimit);
    const rest = ranked.slice(legendLimit);
    series = [
      ...top.map((projectId, i) => ({
        name: projectNameOf(projects, projectId, rowNames.get(projectId)),
        projectIds: [projectId],
        color: PROJECT_COLORS[i % PROJECT_COLORS.length],
      })),
      {
        name: 'Khác',
        projectIds: rest,
        color: PROJECT_COLORS[legendLimit % PROJECT_COLORS.length],
      },
    ];
  } else {
    series = ranked.map((projectId, i) => ({
      name: projectNameOf(projects, projectId, rowNames.get(projectId)),
      projectIds: [projectId],
      color: PROJECT_COLORS[i % PROJECT_COLORS.length],
    }));
  }

  const data = dates.map((d) => {
    const point: Record<string, string | number> = { date: formatChartDate(d) };
    for (const def of series) {
      point[def.name] = def.projectIds.reduce(
        (sum, projectId) => sum + (byProject.get(projectId)?.[d] ?? 0),
        0,
      );
    }
    return point;
  });

  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" tickFormatter={formatChartDate} fontSize={12} />
          <YAxis
            fontSize={12}
            width={metric === 'amount' ? 76 : 40}
            tickFormatter={(v: number) => (metric === 'amount' ? formatCurrency(v) : String(v))}
          />
          <Tooltip />
          {legendLimit ? (
            <Legend content={<BoundedLegend />} />
          ) : (
            <Legend />
          )}
          {series.map((def) => (
            <Bar key={def.name} dataKey={def.name} fill={def.color} />
          ))}
        </BarChart>
      </ResponsiveContainer>
      <p className="mt-1 text-xs text-slate-600">{label}</p>
    </div>
  );
}

const PROJECT_COLORS = ['#0ea5e9', '#22c55e', '#f59e0b', '#a855f7', '#ef4444', '#14b8a6'];

/** FR-6 (1): number of paid employees per project over time. */
export function PaidEmployeesChart(props: Omit<PaidSeriesChartProps, 'metric' | 'label'>) {
  return <SeriesChart {...props} metric="employee_count" label="Nhân viên đã trả" />;
}

/** FR-6 (2): paid amount per project over time. */
export function PaidAmountChart(props: Omit<PaidSeriesChartProps, 'metric' | 'label'>) {
  return <SeriesChart {...props} metric="amount" label="Số tiền đã trả" />;
}
