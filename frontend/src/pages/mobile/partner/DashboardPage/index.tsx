import { useState, useMemo, useCallback } from 'react';
import { format, startOfMonth } from 'date-fns';
import { UserCheck01, UserX01 } from '@untitledui/icons';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import { MissingMobileSection } from '@/components/employees/MissingMobileSection';
import { ApprovalRequestsCard } from '@/components/partner-dashboard/ApprovalRequestsCard';
import { useTopPaidEmployees } from '@/hooks/api/useDashboard';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import { PaidSeriesSection } from '@/components/admin-dashboard/PaidSeriesSection';
import { KyFeeSummaryTable } from '@/components/billing/KyFeeSummaryTable';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { WorkforceOverviewCard } from '@/components/partner-dashboard/WorkforceOverviewCard';
import { ROLE_EYEBROW, rankBadgeClass } from '@/components/partner-dashboard/leaderboard';
import { useAuth } from '@/contexts';
import { cn } from '@/lib/utils';
import type { TopPaidEmployeeItem } from '@/types/api/dashboard.types';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

const ALL_VALUE = 'all';

/* ───────────────────────────────────────────────
   TopEmployeeRow — clean leaderboard row
   ─────────────────────────────────────────────── */

function TopEmployeeRow({
  item,
  maxPaid,
  onOpen,
}: {
  item: TopPaidEmployeeItem;
  maxPaid: number;
  onOpen: (employeeId: string) => void;
}) {
  const pct = maxPaid > 0 ? (item.total_paid_vnd / maxPaid) * 100 : 0;

  return (
    <button
      type="button"
      onClick={() => onOpen(String(item.employee_id))}
      aria-label={`Mở thông tin nhân viên ${item.employee_name}`}
      className="flex w-full items-center gap-3 border-b border-border/30 py-3 text-left transition-colors last:border-0 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
    >
      {/* Rank badge — podium tokens from the shared helper (DASH-07). */}
      <div
        className={cn(
          'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold',
          rankBadgeClass(item.rank),
        )}
      >
        {item.rank}
      </div>

      {/* Avatar */}
      <UserAvatar name={item.employee_name} size="sm" />

      {/* Name + progress */}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-sm font-medium break-words text-foreground">{item.employee_name}</span>
          {item.is_active ? (
            <Badge variant="outline" className="min-h-5 shrink-0 border-success/30 bg-success/10 px-1 py-0 text-xs text-success">
              <UserCheck01 className="mr-0.5 h-2.5 w-2.5" aria-hidden="true" />Đang làm
            </Badge>
          ) : (
            <Badge variant="outline" className="min-h-5 shrink-0 border-destructive/30 bg-destructive/10 px-1 py-0 text-xs text-destructive">
              <UserX01 className="mr-0.5 h-2.5 w-2.5" aria-hidden="true" />Nghỉ
            </Badge>
          )}
        </div>
        <div className="mt-1.5 flex items-end gap-3">
          <div className="min-w-0 flex-1 pb-1">
            <div className="h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-gradient-to-r from-primary/40 to-primary/80 transition-all duration-700 ease-out"
                style={{ width: `${Math.max(pct, 2)}%` }}
              />
            </div>
          </div>
          <span className="console-value shrink-0 text-right text-[13px] font-semibold tabular-nums text-foreground">
            {formatVND(item.total_paid_vnd)}
          </span>
        </div>
      </div>
    </button>
  );
}

/* ───────────────────────────────────────────────
   PartnerDashboardMobile — main page
   ─────────────────────────────────────────────── */

const PartnerDashboardMobile = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM'),
  );

  const handleMonthChange = useCallback((v: string) => setSelectedMonth(v), []);

  const { user } = useAuth();

  // Employee rows in the mobile-warning section open the employee details
  // sheet through the URL-driven modal system — same wiring as the mobile
  // partner EmployeesPage's handleEmployeeClick.
  const { openEmployeeDetails } = useEmployeeModals();
  const handleEmployeeClick = useCallback(
    (employee: { id: number }) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );
  const handleEmployeeIdClick = useCallback(
    (employeeId: string) => openEmployeeDetails(employeeId),
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
    if (selectedMonth === ALL_VALUE) return 'Tất cả';
    const [y, m] = selectedMonth.split('-');
    return `T${m}/${y}`;
  }, [selectedMonth]);

  const topEmployees = useMemo(() => topPaid?.employees ?? [], [topPaid?.employees]);
  const maxPaid = topEmployees[0]?.total_paid_vnd ?? 0;

  return (
    <MobilePageShell className="space-y-2">
      <MobilePageHeader
        title="Tổng quan"
        subtitle={periodLabel}
        kicker={user?.role ? ROLE_EYEBROW[user.role] : undefined}
      />
      <TimesheetMonthSelector value={selectedMonth} onChange={handleMonthChange} />

      {/* Mobile warnings render nothing when clean (no dead card). */}
      <MissingMobileSection onEmployeeClick={handleEmployeeFixClick} />

      {/* PRD §5.2: pending-approval requests card. */}
      <ApprovalRequestsCard />

      {/* DASH-04: the 3-card KPI strip is gone — the workforce overview card
          (donut + breakdown + financial footer) is the single summary block. */}
      <WorkforceOverviewCard month={apiMonth} />

      <KyFeeSummaryTable variant="mobile" />
      <PaidSeriesSection month={apiMonth} layout="segmented" />

      <MobileSurface className="p-4">
        <div className="mb-3 flex items-baseline gap-2">
          <h2 className="console-kicker">Chi trả theo nhân viên</h2>
          <span className="text-[11px] text-muted-foreground">— {periodLabel}</span>
        </div>

        <div className="divide-y divide-border/60">
          {isTopLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-3">
                <Skeleton className="h-7 w-7 rounded-full" />
                <Skeleton className="h-8 w-8 rounded-full" />
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-1.5 w-full rounded-full" />
                </div>
                <Skeleton className="h-3.5 w-16" />
              </div>
            ))
          ) : topEmployees.length === 0 ? (
            <EmptyState
              title="Chưa có dữ liệu thanh toán"
              description="Chọn tháng khác để xem"
              size="sm"
            />
          ) : (
            topEmployees.map((item) => (
              <TopEmployeeRow
                key={item.employee_id}
                item={item}
                maxPaid={maxPaid}
                onOpen={handleEmployeeIdClick}
              />
            ))
          )}
        </div>
      </MobileSurface>
    </MobilePageShell>
  );
};

export default PartnerDashboardMobile;
