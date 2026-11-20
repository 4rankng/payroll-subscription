import { apiClient, buildQueryString } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type {
  DashboardSummary,
  DashboardSummaryParams,
  NewEmployee,
  NewEmployeesResponse,
  NewEmployeesParams,
  TopPaidEmployeesParams,
  TopPaidEmployeesResponse,
} from '@/types/api/dashboard.types';

class DashboardService {

  /**
   * Get Dashboard Summary
   * Returns summary statistics: employee counts and customer-paid salary totals.
   */
  async getDashboardSummary(params?: DashboardSummaryParams): Promise<DashboardSummary> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<DashboardSummary>(
      `${API_ENDPOINTS.dashboard.summary}${queryString}`
    );
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return response.data;
  }

  /**
   * Get New Employees
   * Returns recently added employees
   */
  async getNewEmployees(params?: NewEmployeesParams): Promise<NewEmployeesResponse> {
    const queryString = buildQueryString(params);
    const response = await apiClient.get<NewEmployee[]>(`${API_ENDPOINTS.dashboard.newEmployees}${queryString}`);
    if (!response.data) {
      throw new Error('API response missing expected data');
    }
    return {
      status: response.status || 'success',
      data: response.data,
      pagination: response.pagination!,
      message: response.message || 'Recent activities retrieved successfully'
    };
  }

  async getTopPaidEmployees(params?: TopPaidEmployeesParams): Promise<TopPaidEmployeesResponse> {
    const queryString = params ? buildQueryString(params as Record<string, unknown>) : '';
    const response = await apiClient.get<TopPaidEmployeesResponse>(
      `${API_ENDPOINTS.dashboard.topPaidEmployees}${queryString}`
    );
    if (!response.data) throw new Error('API response missing expected data');
    return response.data;
  }
}

export const dashboardService = new DashboardService();
