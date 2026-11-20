// Dashboard API Types — mirrors backend/internal/app/dto/dashboard.go.

// Dashboard Summary Types (GET /dashboard/summary)
export interface DashboardSummaryParams {
  month?: string; // Format: YYYY-MM (e.g., "2025-04")
}

export interface DashboardSummary {
  total_employees: number;
  total_working_employees: number;
  employees_hired_this_month: number;
  total_paid_salary: number;
  pending_salary_this_month: number;
  paid_salary_this_month: number;
}

export interface NewEmployeeProject {
  id: number;
  name: string;
  code: string | null;
}

export interface NewEmployee {
  id: number;
  fullname: string;
  email: string | null;
  cccd: string;
  date_of_birth: string | null;
  status: 'active' | 'inactive';
  current_project: NewEmployeeProject | null;
  created_at: string;
  created_by_name: string;
  avatar: string | null;
}

export interface NewEmployeesParams {
  page?: number;
  pageSize?: number; // max 20
  days?: number;
}

export interface PaginationData {
  page: number;
  pageSize: number;
  totalRecords: number;
  totalPages: number;
}

// API Response wrapper types
export interface NewEmployeesResponse {
  status: 'success' | 'error';
  data: NewEmployee[];
  pagination: PaginationData;
  message: string;
}

export interface TopPaidEmployeeItem {
  rank: number;
  employee_id: number;
  employee_name: string;
  total_paid_vnd: number;
  is_active: boolean;
  last_timesheet_date: string;
}

export interface TopPaidEmployeesParams {
  month?: string; // YYYY-MM, optional; omit for all-time
  limit?: number;
}

export interface TopPaidEmployeesResponse {
  employees: TopPaidEmployeeItem[];
  period: string; // "all_time" or "YYYY-MM"
}
