import { useState, useCallback, useMemo } from 'react';
import { format, startOfMonth } from 'date-fns';
import { RefreshCw05, UserX01 } from '@untitledui/icons';
import { useQueryClient } from '@tanstack/react-query';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageHeader } from '@/components/shared/PageHeader';
import { ConsolePanel } from '@/components/shared/console-panel';
import { useTopPaidEmployees } from '@/hooks/api/useDashboard';
import { billingKeys } from '@/hooks/api/useBilling';
import { QueryKeys } from '@/lib/queryKeys';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import { useAuth } from '@/contexts';
import { MissingMobileSection } from '@/components/employees/MissingMobileSection';
import { PaidSeriesSection } from '@/components/admin-dashboard/PaidSeriesSection';
import { KyFeeSummaryTable } from '@/components/billing/KyFeeSummaryTable';
import { WorkforceOverviewCard } from '@/components/partner-dashboard/WorkforceOverviewCard';
import { ApprovalRequestsCard } from '@/components/partner-dashboard/ApprovalRequestsCard';
import { ROLE_EYEBROW, rankBadgeClass } from '@/components/partner-dashboard/leaderboard';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/shared/EmptyState';
import type { TopPaidEmployeeItem } from '@/types/api/dashboard.types';
import { monthOptions, formatVND } from './utils';

const ALL_VALUE = 'all';
const OTHER_MONTH_VALUE = 'other-month';

// ─── Month selector ────────────────────────────────────────────────────────
function MonthSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const primaryOptions = monthOptions.slice(0, 4);
  const otherOptions = monthOptions.slice(4);
  const isOtherMonth = otherOptions.some((option) => option.value === value);

  return (
    <div
      className="console-filter-bar rounded-lg bg-muted/60 p-1"
      role="group"
      aria-label="Chọn kỳ dữ liệu"
    >
      {primaryOptions.map((opt) => (
        <button
          type="button"
          key={opt.value}
          onClick={() => onChange(opt.value)}
          aria-pressed={value === opt.value}
          className={cn(
            'h-9 shrink-0 rounded-md px-3 text-xs font-semibold transition-[background-color,color,box-shadow]',
            'outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2',
            value === opt.value
              ? 'bg-card text-foreground shadow-sm ring-1 ring-inset ring-border/60'
              : 'text-muted-foreground hover:bg-card/70 hover:text-foreground',
          )}
        >
          {opt.label}
        </button>
      ))}
      <Select
        value={isOtherMonth ? value : OTHER_MONTH_VALUE}
        onValueChange={(nextValue) => {
          if (nextValue !== OTHER_MONTH_VALUE) onChange(nextValue);
        }}
      >
        <SelectTrigger
          aria-label="Chọn tháng khác"
          className={cn(
            'h-9 min-h-9 w-[132px] shrink-0 border-0 bg-transparent px-3 text-xs font-semibold shadow-none focus:ring-1 focus:ring-ring focus:ring-offset-0',
            isOtherMonth && 'bg-card text-foreground shadow-sm ring-1 ring-border/60',
          )}
        >
          <SelectValue placeholder="Tháng khác" />
        </SelectTrigger>
        {/* Opaque overlay contract: solid white menu, never translucent. */}
        <SelectContent align="end" className="bg-white shadow-lg">
          <SelectItem value={OTHER_MONTH_VALUE} disabled>
            Tháng khác
          </SelectItem>
          {otherOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

// ─── Leaderboard row ───────────────────────────────────────────────────────
// DASH-07: podium tokens come from the shared rankBadgeClass helper
// so desktop and mobile badges always match.

function LeaderRow({
  item,
  rank,
  maxAmount,
  onOpen,
}: {
  item: TopPaidEmployeeItem;
  rank: number;
  maxAmount: number;
  onOpen: (employeeId: string) => void;
}) {
  const badge = rankBadgeClass(rank);
  const barWidth = maxAmount > 0 ? Math.max(4, Math.round((item.total_paid_vnd / maxAmount) * 100)) : 0;

  return (
    <button
      type="button"
      onClick={() => onOpen(String(item.employee_id))}
      aria-label={`Mở thông tin nhân viên ${item.employee_name}`}
      className="group grid w-full grid-cols-[1.5rem_2rem_minmax(0,1fr)_auto] items-center gap-2 rounded-md px-2 py-1.5 text-left outline-brand transition-colors hover:bg-muted/40 focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2"
    >
      <span
        className={cn(
          'flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px] font-bold tabular-nums',
          badge,
        )}
      >
        {rank}
      </span>
      <UserAvatar name={item.employee_name} size="sm" className="shrink-0" />
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="truncate text-[13px] font-semibold text-foreground">{item.employee_name}</span>
          {!item.is_active && (
            <span className="inline-flex shrink-0 items-center gap-0.5 rounded-full bg-destructive/10 px-1.5 py-0.5 text-xs font-semibold text-destructive">
              <UserX01 className="h-2.5 w-2.5" aria-hidden="true" />
              Nghỉ
            </span>
          )}
        </div>
        {/* Proportional bar: % of the #1 amount. */}
        <div className="mt-1 h-1 w-full overflow-hidden rounded-full bg-muted/70">
          <div
            className={cn(
              'h-1 rounded-full',
              rank <= 3 ? 'bg-amber-400/80' : 'bg-muted-foreground/30',
            )}
            style={{ width: `${barWidth}%` }}
            aria-hidden
          />
        </div>
      </div>
      <span className="whitespace-nowrap text-xs font-bold tabular-nums text-foreground">
        {formatVND(item.total_paid_vnd)}
      </span>
    </button>
  );
}

// ─── Skeletons ─────────────────────────────────────────────────────────────
function LeaderboardSkeleton() {
  return (
    <div className="space-y-2 pt-2">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-2 py-2.5">
          <Skeleton className="h-6 w-6 rounded-full" />
          <Skeleton className="h-8 w-8 rounded-full" />
          <div className="flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-1 w-full rounded-full max-w-[140px]" />
          </div>
          <Skeleton className="h-3.5 w-20" />
        </div>
      ))}
    </div>
  );
}

// ─── Banner header ─────────────────────────────────────────────────────────
function BannerHeader({
  periodLabel,
  monthValue,
  onMonthChange,
  onRefresh,
}: {
  periodLabel: string;
  monthValue: string;
  onMonthChange: (v: string) => void;
  onRefresh: () => void;
}) {
  const { user } = useAuth();
  const eyebrow = user?.role ? ROLE_EYEBROW[user.role] : undefined;

  return (
    <PageHeader
      eyebrow={eyebrow}
      title="Tổng quan"
      meta={`Nhân sự và thanh toán · ${periodLabel}`}
      metaVariant="sans"
    >
      <button
        type="button"
        onClick={onRefresh}
        aria-label="Làm mới"
        title="Làm mới"
        className={cn(
          'inline-flex h-8 w-8 items-center justify-center rounded-md border border-border bg-card text-muted-foreground',
          'transition-colors hover:bg-muted hover:text-foreground',
          'outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2',
        )}
      >
        <RefreshCw05 className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
      <MonthSelector value={monthValue} onChange={onMonthChange} />
    </PageHeader>
  );
}

// ─── Main page ─────────────────────────────────────────────────────────────
const PartnerDashboardPage = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM'),
  );
  const handleMonthChange = useCallback((value: string) => setSelectedMonth(value), []);

  const queryClient = useQueryClient();
  const handleRefresh = useCallback(() => {
    // The dashboard renders dashboard, billing (FR-6 charts + kỳ fee table)
    // and the phone-warning banner data — refresh all three families.
    void queryClient.invalidateQueries({ queryKey: QueryKeys.dashboard.all });
    void queryClient.invalidateQueries({ queryKey: billingKeys.all });
    void queryClient.invalidateQueries({ queryKey: QueryKeys.employees.all });
  }, [queryClient]);

  // Employee rows open the employee details sheet through the URL-driven
  // modal system — same wiring as the partner EmployeesPage.
  const { openEmployeeDetails } = useEmployeeModals();
  const handleEmployeeClick = useCallback(
    (employee: { id: number }) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );
  // DASH-06: banner rows deep-link the phone field (?focus=mobile) — the
  // banner context IS the fix context.
  const handleEmployeeFixClick = useCallback(
    (employee: { id: number }) => openEmployeeDetails(employee.id.toString(), undefined, 'mobile'),
    [openEmployeeDetails],
  );

  const apiMonth = selectedMonth === ALL_VALUE ? undefined : selectedMonth;
  const { data: topPaid, isLoading: isTopLoading } = useTopPaidEmployees(
    apiMonth ? { month: apiMonth } : undefined,
  );

  const periodLabel = useMemo(() => {
    if (selectedMonth === ALL_VALUE) return 'tất cả thời gian';
    const [y, m] = selectedMonth.split('-');
    return `tháng ${m}/${y}`;
  }, [selectedMonth]);

  const topEmployees = useMemo(() => topPaid?.employees ?? [], [topPaid?.employees]);
  const maxPaidAmount = topEmployees[0]?.total_paid_vnd ?? 0;

  return (
    <div className="min-h-full p-3 sm:p-4">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-3">
        <BannerHeader
          periodLabel={periodLabel}
          monthValue={selectedMonth}
          onMonthChange={handleMonthChange}
          onRefresh={handleRefresh}
        />

        {/* Mobile warnings render nothing when clean (no dead card). */}
        <MissingMobileSection onEmployeeClick={handleEmployeeFixClick} />

        {/* PRD §5.2: pending-approval requests card. */}
        <ApprovalRequestsCard />

        {/* ── Summary row: workforce card + fee table share bottom edges ── */}
        <section className="flex flex-col gap-3">
          <div className="grid grid-cols-1 items-stretch gap-4 lg:grid-cols-12">
            <div className="flex lg:col-span-5">
              <WorkforceOverviewCard month={apiMonth} />
            </div>
            <div className="flex lg:col-span-7">
              <KyFeeSummaryTable className="h-full w-full" />
            </div>
          </div>
          <PaidSeriesSection month={apiMonth} layout="segmented" />
        </section>

        {/* ── Leaderboard ── */}
        <ConsolePanel
          kicker="Chi trả"
          title="Top nhân viên"
          description={periodLabel}
          isFlush
          bodyClassName="flex flex-1 flex-col"
          actions={
            !isTopLoading && topEmployees.length > 0 ? (
              <span className="console-kicker">{topEmployees.length} nhân viên</span>
            ) : undefined
          }
        >
          <div
            className={cn(
              'px-3 py-2',
              !isTopLoading && topEmployees.length === 0 && 'flex flex-1 items-center justify-center',
            )}
          >
            {isTopLoading ? (
              <LeaderboardSkeleton />
            ) : topEmployees.length === 0 ? (
              <EmptyState
                showIllustration={false}
                title="Chưa có dữ liệu thanh toán"
                description={`Chưa có chi trả trong ${periodLabel}.`}
                size="sm"
              />
            ) : (
              <div className="grid grid-cols-1 gap-x-4 gap-y-0.5 min-[1120px]:grid-cols-2">
                {topEmployees.map((item, idx) => (
                  <LeaderRow
                    key={item.employee_id}
                    item={item}
                    rank={idx + 1}
                    maxAmount={maxPaidAmount}
                    onOpen={openEmployeeDetails}
                  />
                ))}
              </div>
            )}
          </div>
        </ConsolePanel>
      </div>
    </div>
  );
};

export default PartnerDashboardPage;
