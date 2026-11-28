// Payroll and payment history related types

export interface PaymentHistory {
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  project_id: number;
  project_name: string;
  position: string;
  total_paid_amount: number;
  paid_date: string; // ISO 8601 format (UTC)
  /** 1-based ledger line number from the backend bank-entry projection. */
  stt?: number;
  /** Pay cycle 1..4 this entry belongs to. */
  ky?: number;
  /** Cycle window label, e.g. "01/11 → 07/11". */
  ky_window?: string;
  /** Original transfer-upload filename that verified the payment (Ghi chú CK). */
  transfer_note?: string;
  /** Bank transaction reference code (Mã bút toán). */
  bank_transaction_code?: string;
  /** RFC 3339 instant of when the bank entry was processed (+07 offset). */
  processed_at?: string;
  /** Amount actually received by the employee, VND. */
  actual_received_vnd?: number;
}

export interface PaymentHistoryFilters {
  page?: number;
  pageSize?: number;
  sortBy?: 'paid_date' | 'employee_name' | 'amount' | 'project_name';
  sortOrder?: 'asc' | 'desc';
  fromDate?: string; // YYYY-MM-DD
  toDate?: string; // YYYY-MM-DD
  projectId?: number[];
  employeeId?: number[];
  position?: string;
  search?: string; // Search by employee name or CCCD
}

export interface PaymentHistoryListResponse {
  status: string;
  message: string;
  data: PaymentHistory[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface BankTransferHistoryTransfer {
  transfer_code: string;
  bank_reference: string;
  amount: number;
  paid_at?: string;
}

export interface BankTransferHistory {
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  project_ids: number[];
  project_names: string[];
  work_month: string;
  cycle: 1 | 2 | 3 | 4;
  from_date: string;
  to_date: string;
  payment_date: string;
  total_amount: number;
  transfers: BankTransferHistoryTransfer[];
}

export interface BankTransferHistoryFilters {
  month?: string;
  cycle?: number;
  projectId?: number[];
  employeeId?: number[];
  search?: string;
  page?: number;
  pageSize?: number;
}

export interface BankTransferHistoryListResponse {
  status: string;
  message: string;
  data: BankTransferHistory[];
  summary?: {
    total_amount: number;
    transfer_count: number;
    employee_count: number;
  };
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}
