import { useCallback, useMemo, useState } from 'react';
import { format, startOfMonth } from 'date-fns';
import { AlertTriangle } from '@untitledui/icons';

import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { DashboardHeader } from '@/components/admin-dashboard/DashboardHeader';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { PaidSeriesSection } from '@/components/admin-dashboard/PaidSeriesSection';
import { KyFeeSummaryTable } from '@/components/billing/KyFeeSummaryTable';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import {
  DashboardMetricStrip,
  DashboardPriorityList,
  type DashboardMetricItem,
  type DashboardPriorityItem,
} from '@/components/admin-dashboard/DashboardOverview';
import { ConsoleSectionHeading } from '@/components/shared/console-panel';
import { MasonryGrid } from '@/components/shared/masonry-grid';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

const AdminDashboard = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);

  const handleMonthChange = useCallback((value: string) => {
    setSelectedMonth(value);
  }, []);

  const monthParam = selectedMonth === 'all' ? undefined : selectedMonth;
  const monthLabel =
    selectedMonth === 'all'
      ? 'Tất cả'
      : (() => {
          const [year, month] = selectedMonth.split('-');
          return `${month}/${year}`;
        })();

  const { data, loading, employees, refetch, isRefreshing } = useDashboardData(monthParam);
  const actions = useDashboardActions();
  const {
    data: pendingData,
    isError: hasPendingError,
    isFetching: isPendingFetching,
    refetch: refetchPending,
  } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });
  const dashboardNav = useDashboardNavigation();
  const pendingApprovals = pendingData?.pagination?.totalRecords ?? null;

  const handleRefresh = useCallback(async () => {
    setIsManualRefreshing(true);
    try {
      await Promise.all([
        refetch(),
        refetchPending(),
        Promise.resolve(actions.handleRefresh?.()),
      ]);
    } finally {
      setIsManualRefreshing(false);
    }
  }, [actions, refetch, refetchPending]);

  const isDashboardRefreshing = isManualRefreshing || isRefreshing || isPendingFetching;

  const { employeeStats } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingApprovals ?? 0,
    dashboardNav,
  });

  const metricItems = useMemo<DashboardMetricItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    // Zone ownership: the strip owns period OUTCOMES only. Pending payouts and
    // approval-queue counts belong to DashboardPriorityList below — never
    // repeat a metric between the two adjacent zones.
    return [
      {
        label: 'Đã trả kỳ này',
        value: formatVND(data.dashboardSummary.paid_salary_this_month),
        fullValue: formatVND(data.dashboardSummary.paid_salary_this_month),
        context: `Tổng đã trả ${formatVND(data.dashboardSummary.total_paid_salary)}`,
        tone: 'primary',
      },
      {
        label: 'Chờ giải ngân',
        value: formatVND(data.dashboardSummary.pending_salary_this_month),
        fullValue: formatVND(data.dashboardSummary.pending_salary_this_month),
        context: `Lương đã duyệt chờ thanh toán ${monthLabel}`,
        tone: 'warning',
      },
      {
        label: 'Nhân sự đang làm',
        value: data.dashboardSummary.total_working_employees.toLocaleString('vi-VN'),
        context: 'Đang hoạt động trong kỳ',
        tone: 'neutral',
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Tổng nhân sự',
        value: data.dashboardSummary.total_employees.toLocaleString('vi-VN'),
        context: `${data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN')} mới trong kỳ`,
        tone: 'neutral',
      },
      {
        label: 'Nhân sự mới',
        value: data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN'),
        context: `Tuyển trong ${monthLabel}`,
        tone: 'neutral',
        onClick: dashboardNav.navigateToNewEmployees,
      },
    ];
  }, [dashboardNav, data.dashboardSummary, monthLabel]);

  const priorityItems = useMemo<DashboardPriorityItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    const items: DashboardPriorityItem[] = [];
    const pendingKnown = pendingApprovals !== null;
    const pendingCount = pendingApprovals ?? 0;

    // Non-zero row law: a completed queue is not a task. Only a real value,
    // a load failure, or a pending fetch earns a row.
    if (hasPendingError || !pendingKnown || pendingCount > 0) {
      items.push({
        title: 'Bảng công chờ duyệt',
        detail: hasPendingError
          ? 'Không thể tải hàng đợi toàn hệ thống.'
          : !pendingKnown
            ? 'Đang tải hàng đợi bảng công.'
            : 'Xử lý trước khi khóa sổ lương.',
        value: !pendingKnown ? '--' : pendingCount.toLocaleString('vi-VN'),
        statusLabel: hasPendingError ? 'Lỗi tải' : !pendingKnown ? 'Đang tải' : 'Cần duyệt',
        icon: AlertTriangle,
        tone: hasPendingError || !pendingKnown ? 'neutral' : 'warning',
        onClick:
          pendingKnown && pendingCount > 0
            ? dashboardNav.navigateToPendingApprovals
            : undefined,
      });
    }

    return items;
  }, [
    dashboardNav,
    data.dashboardSummary,
    hasPendingError,
    pendingApprovals,
  ]);

  if (loading) {
    return (
      <div className="admin-dashboard-page p-3 sm:p-4">
        <DashboardLoadingSkeleton />
      </div>
    );
  }

  return (
    <div className="admin-dashboard-page min-h-full bg-transparent">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-3 overflow-x-hidden p-3 sm:p-4">
        <div className="motion-reduce:animate-none motion-reduce:opacity-100">
          <DashboardHeader value={selectedMonth} onChange={handleMonthChange} />
        </div>

        <DashboardMetricStrip items={metricItems} />

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading
            kicker="Ưu tiên"
            title="Vận hành"
            meta={`Kỳ ${monthLabel}`}
          />
          <DashboardPriorityList
            items={priorityItems}
            isRefreshing={isDashboardRefreshing}
            onRefresh={handleRefresh}
            className="lg:col-span-12"
          />
        </section>

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading
            kicker="PAYSUB"
            title="Phí"
            meta="NV theo kỳ trả · 12 tháng"
          />
          <div className="flex flex-col gap-3">
            <KyFeeSummaryTable />
            <PaidSeriesSection />
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading kicker="Con người" title="Nhân sự" meta={monthLabel} />
          <MasonryGrid>
            <TopPaidEmployeesCard month={monthParam} className="lg:col-span-4" />
            <RecentEmployeesCard
              employees={employees.employees}
              isLoading={employees.isLoading}
              isLoadingMore={employees.isLoadingMore}
              totalEmployees={employees.totalEmployees}
              weeks={employees.weeks}
              onEmployeeClick={(employee) => actions.handleEmployeeClick?.(employee)}
              onLoadMore={employees.loadMore}
              className="lg:col-span-8"
            />
          </MasonryGrid>
        </section>
      </div>
    </div>
  );
};

export default AdminDashboard;
