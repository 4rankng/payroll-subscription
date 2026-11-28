import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth } from 'date-fns';
import {
  Users,
  UserCheck,
  Banknote,
  FolderKanban,
  ClipboardList,
  Hourglass,
  Wallet,
} from 'lucide-react';
import { UserCheck01, UserX01 } from '@untitledui/icons';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { MobilePageShell, MobileSurface } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import {
  MobileOperationsPanel,
  type MobileOperationAction,
  type MobileOperationMetric,
} from '@/components/shared/MobileOperationsPanel';
import { MissingMobileSection } from '@/components/employees/MissingMobileSection';
import { useDashboardSummary, useTopPaidEmployees } from '@/hooks/api/useDashboard';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import { PaidSeriesSection } from '@/components/admin-dashboard/PaidSeriesSection';
import { KyFeeSummaryTable } from '@/components/billing/KyFeeSummaryTable';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { cn } from '@/lib/utils';
import type { TopPaidEmployeeItem } from '@/types/api/dashboard.types';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

const ALL_VALUE = 'all';

/**
 * KPI wrap-safety (staging UI audit F-03): full VND strings beyond ~10 digits
 * ("18.151.395.905 ₫") overflow the narrow metric cells. Values that long get
 * the compact scale ("18,2 Tr ₫"); shorter ones keep their full digits.
 */
function formatKpiValue(value: number): string {
  const full = formatVND(value);
  if (full.replace(/[^\d]/g, '').length <= 10) return full;
  return `${new Intl.NumberFormat('vi-VN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)} ₫`;
}

/* ───────────────────────────────────────────────
   TopEmployeeRow — clean leaderboard row
   ─────────────────────────────────────────────── */

const RANK_STYLES: Record<number, { badge: string }> = {
  1: { badge: 'bg-warning/10 text-warning border-warning/30' },
  2: { badge: 'bg-muted text-muted-foreground border-border' },
  3: { badge: 'bg-primary/10 text-primary border-primary/30' },
};

function TopEmployeeRow({ item, maxPaid }: { item: TopPaidEmployeeItem; maxPaid: number }) {
  const pct = maxPaid > 0 ? (item.total_paid_vnd / maxPaid) * 100 : 0;
  const rank = RANK_STYLES[item.rank] ?? null;

  return (
    <div className="flex items-center gap-3 py-3 border-b border-border/30 last:border-0 group">
      {/* Rank badge */}
      <div
        className={cn(
          'flex items-center justify-center w-7 h-7 rounded-full text-xs font-bold shrink-0 border',
          rank?.badge ?? 'bg-muted/40 text-muted-foreground border-border/40',
        )}
      >
        {item.rank}
      </div>

      {/* Avatar */}
      <UserAvatar name={item.employee_name} size="sm" />

      {/* Name + progress */}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-sm font-medium text-foreground break-words">{item.employee_name}</span>
          {item.is_active ? (
            <Badge variant="outline" className="min-h-5 shrink-0 border-success/30 bg-success/10 px-1 py-0 text-xs text-success">
              <UserCheck01 className="w-2.5 h-2.5 mr-0.5" aria-hidden="true" />Đang làm
            </Badge>
          ) : (
            <Badge variant="outline" className="min-h-5 shrink-0 border-destructive/30 bg-destructive/10 px-1 py-0 text-xs text-destructive">
              <UserX01 className="w-2.5 h-2.5 mr-0.5" aria-hidden="true" />Nghỉ
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
          <span className="console-value shrink-0 text-right text-[13px] font-semibold text-foreground">
            {formatVND(item.total_paid_vnd)}
          </span>
        </div>
      </div>
    </div>
  );
}

/* ───────────────────────────────────────────────
   PartnerDashboardMobile — main page
   ─────────────────────────────────────────────── */

const PartnerDashboardMobile = () => {
  const navigate = useNavigate();
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM'),
  );

  const handleMonthChange = useCallback((v: string) => setSelectedMonth(v), []);

  // Employee rows in the mobile-warning section open the employee details
  // sheet through the URL-driven modal system — same wiring as the mobile
  // partner EmployeesPage's handleEmployeeClick.
  const { openEmployeeDetails } = useEmployeeModals();
  const handleEmployeeClick = useCallback(
    (employee: { id: number }) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const apiMonth = selectedMonth === ALL_VALUE ? undefined : selectedMonth;
  const { data: summary, isLoading: isSummaryLoading } = useDashboardSummary(apiMonth);
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

  const quickActions = useMemo<MobileOperationAction[]>(() => [
    {
      label: 'Dự án',
      icon: FolderKanban,
      onClick: () => navigate('/partner/projects'),
    },
    {
      label: 'Nhân viên',
      icon: Users,
      onClick: () => navigate('/partner/employees'),
    },
    {
      label: 'Bảng công',
      icon: ClipboardList,
      onClick: () => navigate('/partner/timesheet'),
    },
    {
      label: 'Thanh toán',
      icon: Wallet,
      onClick: () => navigate('/partner/billing'),
    },
  ], [navigate]);

  const operationMetrics = useMemo<MobileOperationMetric[]>(() => [
    {
      label: 'Tổng chi trả',
      value: formatKpiValue(summary?.total_paid_salary ?? 0),
      helper: periodLabel,
      icon: Banknote,
      tone: 'primary',
    },
    {
      label: 'Đang làm việc',
      value: (summary?.total_working_employees ?? 0).toLocaleString('vi-VN'),
      helper: `Tổng ${(summary?.total_employees ?? 0).toLocaleString('vi-VN')} NV`,
      icon: UserCheck,
      tone: 'success',
    },
    {
      label: 'Có thể nghỉ',
      value: Math.max(0, (summary?.total_employees ?? 0) - (summary?.total_working_employees ?? 0)).toLocaleString('vi-VN'),
      helper: 'Không hoạt động',
      icon: Users,
      tone: 'warning',
    },
    {
      label: 'Đã thanh toán',
      value: formatKpiValue(summary?.paid_salary_this_month ?? 0),
      helper: periodLabel,
      icon: Wallet,
      tone: 'neutral',
    },
  ], [summary, periodLabel]);

  return (
    <MobilePageShell className="space-y-2">
      <MobilePageHeader kicker="Đối tác" title="Tổng quan" subtitle={periodLabel} />
      <TimesheetMonthSelector value={selectedMonth} onChange={handleMonthChange} />

      {isSummaryLoading ? (
        <MobileSurface className="p-4">
          <div className="space-y-3">
            <Skeleton className="h-7 w-36" />
            <Skeleton className="h-8 w-44" />
            <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-[76px] rounded-lg" />
              ))}
            </div>
          </div>
        </MobileSurface>
      ) : (
        <MobileOperationsPanel
          eyebrow={periodLabel}
          title="Theo dõi chi trả"
          primaryLabel="Tổng chi trả"
          primaryValue={formatKpiValue(summary?.total_paid_salary ?? 0)}
          primaryHint={`Đã thanh toán ${formatKpiValue(summary?.paid_salary_this_month ?? 0)}`}
          metrics={operationMetrics}
          actions={quickActions}
          className="rounded-lg bg-card"
        />
      )}

      {/* Mobile warnings render nothing when clean (no dead card). */}
      <MissingMobileSection onEmployeeClick={handleEmployeeClick} />

      <KyFeeSummaryTable variant="mobile" />
      <PaidSeriesSection compactLegend />

      <MobileSurface className="p-4">
        <div className="mb-3 flex items-baseline gap-2">
          <h2 className="console-kicker">Chi trả theo nhân viên</h2>
          <span className="text-[11px] text-muted-foreground">— {periodLabel}</span>
        </div>

        <div className="divide-y divide-border/60">
          {isTopLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-3">
                <Skeleton className="w-7 h-7 rounded-full" />
                <Skeleton className="w-8 h-8 rounded-full" />
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
              <TopEmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
            ))
          )}
        </div>
      </MobileSurface>
    </MobilePageShell>
  );
};

export default PartnerDashboardMobile;
