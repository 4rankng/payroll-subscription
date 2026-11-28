import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { format, parseISO } from 'date-fns';

import { formatCurrency, formatFullCurrency } from '@/utils/formatters';
import type { PaidSeriesRow } from '@/services/api/billing.service';
import { cn } from '@/lib/utils';

export interface ProjectLabel {
  id: number;
  name: string;
}

/** Named stacked series: one project, or the aggregated "Khác" tail. */
export interface PaidSeriesDef {
  name: string;
  /** Project ids aggregated into this series (one, or many for "Khác"). */
  projectIds: number[];
  color: string;
}

const TOP_PROJECT_COUNT = 8;
const SERIES_COLORS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
  'hsl(var(--chart-6))',
  'hsl(var(--chart-7))',
  'hsl(var(--chart-8))',
];
const OTHER_COLOR = 'hsl(var(--chart-other))';

function projectNameOf(projects: ProjectLabel[], projectId: number, fallback?: string): string {
  return projects.find((p) => p.id === projectId)?.name ?? fallback ?? '(dự án)';
}

/**
 * Shared series definition for both FR-6 charts: rank projects by their
 * paid-amount total over the window, keep the top N as named series and fold
 * the long tail into one neutral "Khác". Both charts must render the exact
 * same series set so one HTML legend covers them (PAYR ticket: sliver bars +
 * duplicated rainbow legend made the section unreadable).
 */
export function buildPaidSeriesDefs(
  rows: PaidSeriesRow[],
  projects: ProjectLabel[],
  topN: number = TOP_PROJECT_COUNT,
): PaidSeriesDef[] {
  const totals = new Map<number, { amount: number; employees: number }>();
  const rowNames = new Map<number, string>();
  for (const row of rows) {
    const name = row.project_name ?? row.project_code;
    if (name && !rowNames.has(row.project_id)) rowNames.set(row.project_id, name);
    const slot = totals.get(row.project_id) ?? { amount: 0, employees: 0 };
    slot.amount += row.amount;
    slot.employees += row.employee_count;
    totals.set(row.project_id, slot);
  }

  const ranked = [...totals.keys()].sort((a, b) => {
    const ta = totals.get(a) ?? { amount: 0, employees: 0 };
    const tb = totals.get(b) ?? { amount: 0, employees: 0 };
    return tb.amount - ta.amount || tb.employees - ta.employees;
  });

  const named = (projectIds: number[]): PaidSeriesDef[] => {
    const seen = new Map<string, number>();
    return projectIds.map((projectId) => {
      const base = projectNameOf(projects, projectId, rowNames.get(projectId));
      // Series names double as chart data keys — keep them unique, but the
      // owner banned raw database ids on any user-facing surface, so unnamed
      // projects get an ordinal suffix instead of "#<id>".
      const count = seen.get(base) ?? 0;
      seen.set(base, count + 1);
      const name = count === 0 ? base : `${base} ${count + 1}`;
      return { name, projectIds: [projectId], color: SERIES_COLORS[(count) % SERIES_COLORS.length] };
    });
  };

  if (ranked.length <= topN) return named(ranked);
  const rest = ranked.slice(topN);
  return [
    ...named(ranked.slice(0, topN)),
    { name: `Khác (${rest.length} dự án)`, projectIds: rest, color: OTHER_COLOR },
  ];
}

/** Shared HTML legend — normal flow above both charts, can never overlap them. */
export function PaidSeriesLegend({
  series,
  className,
}: {
  series: PaidSeriesDef[];
  className?: string;
}) {
  if (series.length === 0) return null;
  return (
    <ul className={cn('mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5', className)}>
      {series.map((def) => (
        <li
          key={def.name}
          className="inline-flex min-w-0 items-center gap-1.5 text-[11px] leading-4 text-muted-foreground"
        >
          <span
            aria-hidden="true"
            className="h-2 w-2 shrink-0 rounded-full"
            style={{ backgroundColor: def.color }}
          />
          <span className="max-w-[180px] truncate">{def.name}</span>
        </li>
      ))}
    </ul>
  );
}

function formatChartDate(iso: string): string {
  try {
    return format(parseISO(iso), 'dd/MM');
  } catch {
    return iso.slice(0, 10);
  }
}

interface TooltipPayloadEntry {
  name?: string;
  value?: number | string;
  color?: string;
}

/**
 * DASH-02: opaque, viewport-safe tooltip — zero-value series filtered,
 * rows sorted descending, bold Tổng cộng footer. Series are already capped
 * at top-8 + "Khác", so the list can never balloon to one row per project.
 */
function PaidSeriesTooltip({
  active,
  payload,
  metric,
}: {
  active?: boolean;
  payload?: TooltipPayloadEntry[];
  metric: 'employee_count' | 'amount';
}) {
  if (!active || !payload?.length) return null;
  const rows = payload
    .filter((entry) => Number(entry.value ?? 0) !== 0)
    .sort((a, b) => Number(b.value ?? 0) - Number(a.value ?? 0));
  if (rows.length === 0) return null;
  const total = rows.reduce((sum, entry) => sum + Number(entry.value ?? 0), 0);
  const formatValue = (value: number) => (metric === 'amount' ? formatCurrency(value) : String(value));

  return (
    <div className="z-40 max-w-[260px] rounded-lg border border-gray-200 bg-white px-3 py-2 text-[11px] shadow-lg">
      <ul className="flex flex-col gap-0.5">
        {rows.map((entry) => (
          <li key={entry.name} className="flex items-center gap-1.5">
            <span
              aria-hidden="true"
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: entry.color }}
            />
            <span className="min-w-0 flex-1 truncate text-slate-600">{entry.name}</span>
            <span className="font-medium tabular-nums text-slate-700">
              {formatValue(Number(entry.value ?? 0))}
            </span>
          </li>
        ))}
      </ul>
      <p className="mt-1.5 flex items-center justify-between gap-3 border-t border-gray-100 pt-1 font-semibold tabular-nums text-slate-900">
        <span>Tổng cộng</span>
        <span>{formatValue(total)}</span>
      </p>
    </div>
  );
}

interface SeriesChartProps {
  rows: PaidSeriesRow[];
  series: PaidSeriesDef[];
  metric: 'employee_count' | 'amount';
}

function SeriesChart({ rows, series, metric }: SeriesChartProps) {
  const isAmount = metric === 'amount';
  const data = useMemo(() => {
    const metricByKey = new Map<string, number>();
    for (const r of rows) {
      const key = `${r.project_id}|${r.date}`;
      metricByKey.set(key, (metricByKey.get(key) ?? 0) + (isAmount ? r.amount : r.employee_count));
    }
    const dates = [...new Set(rows.map((r) => r.date))].sort();
    return dates.map((date) => {
      const point: Record<string, string | number> = { date: formatChartDate(date) };
      for (const def of series) {
        point[def.name] = def.projectIds.reduce(
          (sum, projectId) => sum + (metricByKey.get(`${projectId}|${date}`) ?? 0),
          0,
        );
      }
      return point;
    });
  }, [rows, series, isAmount]);

  if (series.length === 0) return null;

  return (
    <div className="h-72 w-full min-w-0">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 8, right: 12, left: 12, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" vertical={false} />
          <XAxis dataKey="date" fontSize={11} tickMargin={8} minTickGap={24} />
          <YAxis
            fontSize={11}
            width={isAmount ? 88 : 32}
            axisLine={false}
            tickLine={false}
            allowDecimals={false}
            tickFormatter={
              isAmount
                ? (v: number) => formatFullCurrency(v, { showSymbol: false })
                : (v: number) => String(v)
            }
          />
          <Tooltip
            cursor={{ fill: 'rgba(100, 116, 139, 0.08)' }}
            wrapperStyle={{ zIndex: 40, maxWidth: 260 }}
            isAnimationActive={false}
            content={<PaidSeriesTooltip metric={metric} />}
          />
          {series.map((def) => (
            <Bar key={def.name} dataKey={def.name} stackId="paid" fill={def.color} maxBarSize={28} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

/** FR-6 (1): number of paid employees per project over time. */
export function PaidEmployeesChart(props: Omit<SeriesChartProps, 'metric'>) {
  return <SeriesChart {...props} metric="employee_count" />;
}

/** FR-6 (2): paid amount per project over time. */
export function PaidAmountChart(props: Omit<SeriesChartProps, 'metric'>) {
  return <SeriesChart {...props} metric="amount" />;
}
