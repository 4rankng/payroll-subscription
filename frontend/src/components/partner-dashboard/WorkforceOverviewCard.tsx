import { useMemo } from 'react';
import { formatFullCurrency } from '@/utils/formatters';
import { ConsoleDonut } from '@/components/shared/console-charts';
import { ConsolePanel } from '@/components/shared/console-panel';
import { useDashboardSummary } from '@/hooks/api/useDashboard';
import { Skeleton } from '@/components/ui/skeleton';

interface WorkforceOverviewCardProps {
  month?: string;
}

export function WorkforceOverviewCard({ month }: WorkforceOverviewCardProps) {
  const { data: summary, isLoading } = useDashboardSummary(month);

  const segments = useMemo(() => {
    if (!summary) return [];
    const working = summary.total_working_employees;
    const inactive = Math.max(0, summary.total_employees - summary.total_working_employees);
    return [
      {
        label: 'Đang làm việc',
        value: working,
        display: `${working} NV`,
        color: 'hsl(var(--chart-1))',
      },
      {
        label: 'Nghỉ / không hoạt động',
        value: inactive,
        display: `${inactive} NV`,
        color: 'hsl(var(--chart-2))',
      },
    ];
  }, [summary]);

  return (
    <ConsolePanel kicker="Nhân sự" title="Tổng quan nhân sự" isFlush>
      {isLoading ? (
        <div className="flex items-center justify-center gap-4 px-4 py-6">
          <Skeleton className="size-32 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-4 px-4 py-5 sm:flex-row sm:justify-center">
          <ConsoleDonut
            segments={segments}
            centreValue={String(summary?.total_employees ?? 0)}
            centreLabel="Tổng NV"
          />
          <div className="grid w-full grid-cols-2 gap-x-6 gap-y-2 sm:w-auto">
            {segments.map((seg) => (
              <div key={seg.label} className="flex items-center gap-2">
                <span
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: seg.color }}
                  aria-hidden
                />
                <div className="min-w-0">
                  <p className="text-xs text-muted-foreground truncate">{seg.label}</p>
                  <p className="text-sm font-semibold tabular-nums">{seg.display}</p>
                </div>
              </div>
            ))}
            <div className="col-span-2 border-t pt-2">
              <p className="text-xs text-muted-foreground">Tổng chi trả</p>
              <p className="text-sm font-semibold tabular-nums">
                {formatFullCurrency(summary?.total_paid_salary ?? 0)}
              </p>
            </div>
          </div>
        </div>
      )}
    </ConsolePanel>
  );
}
