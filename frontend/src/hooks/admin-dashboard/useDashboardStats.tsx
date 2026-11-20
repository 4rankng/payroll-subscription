import { useMemo } from 'react';

// Types
import { StatItem } from '@/components/shared/GroupedStatCard';

/** Live fields of GET /dashboard/summary (profit/revenue aggregates died
 * with the old model — FR-6 surfaces fee data via the billing module). */
interface DashboardSummary {
  total_employees: number;
  total_working_employees: number;
  employees_hired_this_month: number;
  pending_salary_this_month: number;
  paid_salary_this_month: number;
  total_paid_salary: number;
}

interface DashboardNavigation {
  navigateToActiveEmployees: () => void;
  navigateToNewEmployees: () => void;
  navigateToPendingApprovals: () => void;
}

interface UseDashboardStatsParams {
  dashboardSummary: DashboardSummary | null;
  totalPendingApprovals: number;
  dashboardNav: DashboardNavigation;
}

interface UseDashboardStatsReturn {
  employeeStats: StatItem[];
  salaryStats: StatItem[];
}

export const useDashboardStats = ({
  dashboardSummary,
  totalPendingApprovals,
  dashboardNav,
}: UseDashboardStatsParams): UseDashboardStatsReturn => {
  // Employee statistics
  const employeeStats = useMemo(() => {
    if (!dashboardSummary) return [];
    return [
      {
        label: 'Tổng',
        value: dashboardSummary.total_employees,
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Đang làm việc',
        value: dashboardSummary.total_working_employees,
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Mới',
        value: dashboardSummary.employees_hired_this_month,
        onClick: dashboardNav.navigateToNewEmployees,
      },
    ] as StatItem[];
  }, [dashboardSummary, dashboardNav]);

  // Combined salary statistics
  const salaryStats = useMemo(() => {
    const stats: StatItem[] = [];

    if (dashboardSummary) {
      stats.push(
        {
          label: 'Chờ trả',
          value: dashboardSummary.pending_salary_this_month,
          unit: ' ₫',
        },
        {
          label: 'Đã trả',
          value: dashboardSummary.paid_salary_this_month,
          unit: ' ₫',
        },
        {
          label: 'Tổng đã trả',
          value: dashboardSummary.total_paid_salary,
          unit: ' ₫',
        }
      );
    }

    stats.push({
      label: 'Chờ duyệt',
      value: totalPendingApprovals,
      onClick: dashboardNav.navigateToPendingApprovals,
    });

    return stats;
  }, [dashboardSummary, totalPendingApprovals, dashboardNav]);

  return {
    employeeStats,
    salaryStats,
  };
};
