import { apiClient } from './client';

/**
 * Billing module (FR-2/FR-3/FR-6, PAYSUB): transfer-result uploads,
 * per-employee-per-Kỳ paid records, monthly billing reports, Tổng quan
 * paid series. Self-contained: owns its endpoint paths and response types.
 * API responses use the platform envelope { status, data }.
 */

const ENDPOINTS = {
  uploads: '/billing/uploads',
  kyPaidRecords: '/billing/ky-paid-records',
  paidSeries: '/billing/dashboard/paid-series',
  kySummary: '/billing/dashboard/ky-summary',
  reports: '/billing/reports',
  generateReport: '/billing/reports/generate',
} as const;

export interface TransferUploadResult {
  upload_id: number;
  status: 'pending' | 'verified' | 'rejected';
  duplicate: boolean;
  rows_total: number;
  rows_ok: number;
  rows_failed: number;
  failures: { row: number; cccd: string; reason: string }[];
}

export interface TransferUploadSummary {
  id: number;
  cycle: number;
  work_month: string;
  original_filename: string;
  status: 'pending' | 'verified' | 'rejected';
  reject_reason: string | null;
  verified_at: string | null;
  created_at: string;
}

export interface EmployeePayCycleRow {
  id: number;
  cycle: number;
  work_month: string;
  employee_id: number;
  project_id: number;
  timesheet_id: number;
  source_upload_id: number;
  paid_amount: number;
}

export interface BillingReport {
  id: number;
  work_month: string;
  fee_per_employee_per_ky_vnd: number;
  total_distinct: number;
  total_vnd: number;
  excel_path: string | null;
  generated_at: string;
}

export interface BillingReportLine {
  cycle: number;
  distinct_count: number;
  amount_vnd: number;
}

export interface PaidSeriesRow {
  date: string;
  project_id: number;
  employee_count: number;
  amount: number;
}

/** One work month of the FR-6 per-month table: distinct employees per Kỳ. */
export interface KyMonthSummary {
  work_month: string;
  counts: Record<string, number>;
}

/** FR-6 per-month table payload: last N months + current subscription fee. */
export interface DashboardKySummary {
  months: KyMonthSummary[];
  fee_per_employee_per_ky_vnd: number;
}

export interface UploadOutcome {
  result: TransferUploadResult;
}

class BillingService {
  /** Upload one bank-transfer result workbook for a cycle of a work month. */
  async uploadTransferResult(cycle: number, workMonth: string, file: File): Promise<TransferUploadResult> {
    const form = new FormData();
    form.append('cycle', String(cycle));
    form.append('work_month', workMonth);
    form.append('file', file);
    const response = await apiClient.post<TransferUploadResult>(ENDPOINTS.uploads, form);
    return response.data as TransferUploadResult;
  }

  /** Upload history for one work month, newest first. */
  async listUploads(workMonth: string, limit = 100): Promise<TransferUploadSummary[]> {
    const response = await apiClient.get<{ items: TransferUploadSummary[] }>(
      `${ENDPOINTS.uploads}?work_month=${encodeURIComponent(workMonth)}&limit=${limit}`,
    );
    return response.data?.items ?? [];
  }

  /** Per-employee-per-Kỳ billing rows + per-cycle distinct summary (FR-1). */
  async getEmployeePayCycles(workMonth: string): Promise<{
    rows: EmployeePayCycleRow[];
    summaryByCycle: Record<string, number>;
  }> {
    const response = await apiClient.get<{
      items: EmployeePayCycleRow[];
      summary_by_cycle: Record<string, number>;
    }>(`${ENDPOINTS.kyPaidRecords}?work_month=${encodeURIComponent(workMonth)}`);
    return {
      rows: response.data?.items ?? [],
      summaryByCycle: response.data?.summary_by_cycle ?? {},
    };
  }

  /** FR-6 Tổng quan: per-day distinct paid employees + amount per project. */
  async getPaidSeries(params: { from: string; to: string; projectId?: number }): Promise<PaidSeriesRow[]> {
    const qs = new URLSearchParams({ from: params.from, to: params.to });
    if (params.projectId !== undefined) qs.set('project_id', String(params.projectId));
    const response = await apiClient.get<{ items: PaidSeriesRow[] }>(`${ENDPOINTS.paidSeries}?${qs}`);
    return response.data?.items ?? [];
  }

  /** FR-6 Tổng quan: distinct employees per Kỳ for the last N months + fee (BR-1). */
  async getKySummary(months = 12): Promise<DashboardKySummary> {
    const response = await apiClient.get<DashboardKySummary>(`${ENDPOINTS.kySummary}?months=${months}`);
    return response.data ?? { months: [], fee_per_employee_per_ky_vnd: 0 };
  }

  /** Generate (or regenerate) the monthly billing report. Giamdoc/admin. */
  async generateReport(workMonth: string): Promise<{ report: BillingReport; lines: BillingReportLine[] }> {
    const response = await apiClient.post<{ report: BillingReport; lines: BillingReportLine[] }>(
      ENDPOINTS.generateReport,
      { work_month: workMonth },
    );
    return response.data as { report: BillingReport; lines: BillingReportLine[] };
  }

  /** One monthly report with its Kỳ lines. */
  async getReport(workMonth: string): Promise<(BillingReport & { lines?: BillingReportLine[] }) | null> {
    const response = await apiClient.get<BillingReport & { lines?: BillingReportLine[] }>(
      `${ENDPOINTS.reports}?work_month=${encodeURIComponent(workMonth)}`,
    );
    return response.data ?? null;
  }
}

export const billingService = new BillingService();
