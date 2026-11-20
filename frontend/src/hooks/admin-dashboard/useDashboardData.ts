import { useMemo, useState } from 'react';
import { useDashboardNewEmployees } from './useDashboardNewEmployees';
import { useDashboardSummary } from '@/hooks/api/useDashboard';

export const useDashboardData = (month?: string) => {
  const [employeeWeeks, setEmployeeWeeks] = useState(1);

  const newEmployeesQuery = useDashboardNewEmployees({ pageSize: 20, days: employeeWeeks * 7 });
  const dashboardSummaryQuery = useDashboardSummary(month);

  const loading = newEmployeesQuery.isLoading || dashboardSummaryQuery.isLoading;

  const recentEmployees = useMemo(() => {
    return newEmployeesQuery.data?.employees || [];
  }, [newEmployeesQuery.data]);

  const error = newEmployeesQuery.error || dashboardSummaryQuery.error;

  const employees = useMemo(() => {
    const totalEmployees = newEmployeesQuery.data?.pagination?.totalRecords || 0;

    return {
      employees: recentEmployees,
      isLoading: newEmployeesQuery.isLoading,
      isLoadingMore: newEmployeesQuery.isFetching && !newEmployeesQuery.isLoading,
      weeks: employeeWeeks,
      totalEmployees,
      loadMore: () => setEmployeeWeeks((prev) => prev + 1),
      refresh: () => newEmployeesQuery.refetch(),
    };
  }, [newEmployeesQuery, recentEmployees, employeeWeeks]);

  return {
    data: {
      dashboardSummary: dashboardSummaryQuery.data || null,
    },
    loading,
    error: error?.message || error,
    employees,
    refetch: () =>
      Promise.all([newEmployeesQuery.refetch(), dashboardSummaryQuery.refetch()]),
    isRefreshing: newEmployeesQuery.isFetching || dashboardSummaryQuery.isFetching,
  };
};
