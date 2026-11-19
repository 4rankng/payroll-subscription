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
}

function metricOf(row: PaidSeriesRow, metric: 'employee_count' | 'amount'): number {
  return metric === 'amount' ? row.amount : row.employee_count;
}

function projectNameOf(projects: ProjectLabel[], projectId: number): string {
  return projects.find((p) => p.id === projectId)?.name ?? `Dự án #${projectId}`;
}

/**
 * FR-6 time-series chart: one bar series per project on the payment-date
 * axis. Metric selects employees vs amount; explicit variant components
 * below are the only intended entry points.
 */
function SeriesChart({ rows, projects, metric, label }: PaidSeriesChartProps) {
  if (rows.length === 0) {
    return <p className="text-sm text-slate-500">Chưa có dữ liệu thanh toán</p>;
  }

  const dates = [...new Set(rows.map((r) => r.date))].sort();
  const byProject = new Map<number, Record<string, number>>();
  for (const row of rows) {
    const slot = byProject.get(row.project_id) ?? {};
    slot[row.date] = (slot[row.date] ?? 0) + metricOf(row, metric);
    byProject.set(row.project_id, slot);
  }
  const data = dates.map((d) => {
    const point: Record<string, string | number> = { date: d };
    for (const [projectId, slot] of byProject) {
      point[projectNameOf(projects, projectId)] = slot[d] ?? 0;
    }
    return point;
  });
  const names = [...byProject.keys()].map((id) => projectNameOf(projects, id));

  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" fontSize={12} />
          <YAxis
            fontSize={12}
            tickFormatter={(v: number) => (metric === 'amount' ? formatCurrency(v) : String(v))}
          />
          <Tooltip />
          <Legend />
          {names.map((name, i) => (
            <Bar key={name} dataKey={name} fill={PROJECT_COLORS[i % PROJECT_COLORS.length]} />
          ))}
        </BarChart>
      </ResponsiveContainer>
      <p className="mt-1 text-xs text-slate-500">{label}</p>
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
