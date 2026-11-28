import { useMemo } from 'react';
import { formatFullCurrency } from '@/utils/formatters';
import { ConsoleDonut } from '@/components/shared/console-charts';
import { ConsolePanel } from '@/components/shared/console-panel';
import { useDashboardSummary } from '@/hooks/api/useDashboard';
import { Skeleton } from '@/components/ui/skeleton';

interface WorkforceOverviewCardProps {
  month?: string;
}

/**
 * DASH-03: left summary card. Compact donut + 12px two-row
 * breakdown (label · count · share, dot-separated) and a full-width
 * two-column financial footer that always fills the panel height.
 */
export function WorkforceOverviewCard({ month }: WorkforceOverviewCardProps) {
  const { data: summary, isLoading } = useDashboardSummary(month);

  const { segments, totalEmployees } = useMemo(() => {
    if (!summary) return { segments: [], totalEmployees: 0 };
    const total = summary.total_employees;
    const working = summary.total_working_employees;
    const inactive = Math.max(0, total - working);
    const share = (value: number) =>
      total > 0 ? `${Math.round((value / total) * 100)}%` : '0%';
    return {
      totalEmployees: total,
      segments: [
        {
          label: 'Đang làm việc',
          value: working,
          display: `${working.toLocaleString('vi-VN')} NV · ${share(working)}`,
          // High-contrast pair — emerald for active, neutral slate for
          // inactive; chart tokens 1/2 were near-identical greens.
          color: '#059669',
        },
        {
          label: 'Nghỉ / không hoạt động',
          value: inactive,
          display: `${inactive.toLocaleString('vi-VN')} NV · ${share(inactive)}`,
          color: '#94A3B8',
        },
      ],
    };
  }, [summary]);

  return (
    <ConsolePanel
      kicker="Nhân sự"
      title="Tổng quan nhân sự"
      isFlush
      className="h-full w-full"
      bodyClassName="flex flex-1 flex-col justify-center"
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-4 px-4 py-6">
          <Skeleton className="size-28 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      ) : (
        <div className="flex flex-1 flex-col justify-center gap-4 px-4 py-5">
          <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-center sm:gap-5">
            <ConsoleDonut
              segments={segments}
              centreValue={totalEmployees.toLocaleString('vi-VN')}
              centreLabel="Tổng NV"
            />
            {/* 12px breakdown rows: label · count · share on one line each. */}
            <div className="w-full space-y-1.5">
              {segments.map((seg) => (
                <div key={seg.label} className="flex items-center gap-2">
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ backgroundColor: seg.color }}
                    aria-hidden
                  />
                  <span className="truncate text-xs text-muted-foreground">{seg.label}</span>
                  <span className="mx-0.5 text-muted-foreground/60" aria-hidden>
                    ·
                  </span>
                  <span className="ml-auto text-xs font-semibold tabular-nums text-foreground">
                    {seg.display}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Financial footer — single-line tabular figures, no orphan ₫. */}
          <div className="mt-auto grid grid-cols-2 gap-3 border-t pt-3">
            <div className="min-w-0">
              <p className="text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">
                Tổng chi trả
              </p>
              <p className="truncate text-xs font-semibold tabular-nums text-foreground" title={formatFullCurrency(summary?.total_paid_salary ?? 0)}>
                {formatFullCurrency(summary?.total_paid_salary ?? 0)}
              </p>
            </div>
            <div className="min-w-0">
              <p className="text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground">
                Đã thanh toán
              </p>
              <p className="truncate text-xs font-semibold tabular-nums text-foreground" title={formatFullCurrency(summary?.paid_salary_this_month ?? 0)}>
                {formatFullCurrency(summary?.paid_salary_this_month ?? 0)}
              </p>
            </div>
          </div>
        </div>
      )}
    </ConsolePanel>
  );
}
