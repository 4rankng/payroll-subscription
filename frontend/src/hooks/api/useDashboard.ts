import { useQuery } from '@tanstack/react-query';

import { dashboardService } from '@/services/api/dashboard.service';
import { QueryKeys } from '@/lib/queryKeys';
import type { TopPaidEmployeesParams } from '@/types/api/dashboard.types';

// Get dashboard summary
export const useDashboardSummary = (month?: string) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.summary(month),
    queryFn: () => dashboardService.getDashboardSummary(month ? { month } : undefined),
  });
};

// Get top paid employees (admin dashboard metric)
export const useTopPaidEmployees = (params?: TopPaidEmployeesParams) => {
  return useQuery({
    queryKey: QueryKeys.dashboard.topPaidEmployees(params?.month, params?.limit),
    queryFn: () => dashboardService.getTopPaidEmployees(params),
  });
};
