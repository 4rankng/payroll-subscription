// Payroll and payment history related types

export interface PaymentHistory {
  /** 1-based ledger line number from the backend bank-entry projection. */
  stt?: number;
  employee_id: number;
  employee_name: string;
  employee_cccd: string;
  project_id: number;
  project_name: string;
  position: string;
  total_paid_amount: number;
  paid_date: string; // ISO 8601 format (UTC)
  /** Pay cycle 1..4 this entry belongs to. */
  ky?: number;
  /** Cycle window label, e.g. "01/11 → 07/11". */
  ky_window?: string;
  /** Original transfer-upload filename that verified the payment (Ghi chú CK). */
  transfer_note?: string;
  /** Bank transaction reference code (Mã GD ngân hàng). */
  bank_transaction_code?: string;
  /** RFC 3339 instant of when the bank entry was processed (+07 offset). */
  processed_at?: string;
  /** Amount actually received by the employee, VND (Thực nhận kỳ). */
  actual_received_vnd?: number;
}

/** PRD §5.7 filter set: month + Kỳ + free-text search. */
export interface PaymentHistoryFilters {
  page?: number;
  pageSize?: number;
  /** Work month, YYYY-MM. Defaults to the current month. */
  month?: string;
  /** Kỳ 1..4; undefined = all Kỳ of the month. */
  ky?: number;
  /** Search by employee name / CCCD / transfer note / bank transaction code. */
  search?: string;
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
