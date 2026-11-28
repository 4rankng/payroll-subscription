import { apiClient, createIdempotencyKey } from './client';

/**
 * Billing module (FR-2/FR-3/FR-6, PAYSUB): transfer-result uploads,
 * per-employee-per-Kỳ paid records, monthly billing reports, Tổng quan
 * paid series, sao kê export/email. Self-contained: owns its endpoint paths
 * and response types. API responses use the platform envelope { status, data }.
 */

export interface KyReconStatusRow {
  cycle: number;
  uploads: number;
  parsed_rows: number;
  matched_rows: number;
  /** waiting_file | mismatch | reconciled (backend derives from upload outcomes). */
  state: string;
}

export interface ReconStatus {
  work_month: string;
  kyes: KyReconStatusRow[];
}

const ENDPOINTS = {
  uploads: '/billing/uploads',
  kyPaidRecords: '/billing/ky-paid-records',
  paidSeries: '/billing/dashboard/paid-series',
  kySummary: '/billing/dashboard/ky-summary',
  reconStatus: '/billing/dashboard/recon-status',
  reports: '/billing/reports',
  generateReport: '/billing/reports/generate',
  saokeRecipients: '/billing/saoke/recipients',
  saokeExport: '/billing/saoke/export',
  saokeSendEmail: '/billing/saoke/email',
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
  /** Rows parsed from the workbook / matched to approved timesheets. */
  parsed_rows: number;
  matched_rows: number;
  /** SUM(employee_pay_cycles.paid_amount) for this upload (Sổ cái Tổng tiền). */
  matched_amount: number;
  /** Stored original workbook; null for rows from before workbook storage. */
  asset_id: number | null;
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
  asset_id: number | null;
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
  project_code?: string;
  project_name?: string;
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

/** Persisted To/CC lists for sao kê email (GET/PUT /billing/saoke/recipients). */
export interface SaokeRecipients {
  to: string[];
  cc: string[];
}

export interface SendSaokeEmailParams {
  report_date: string; // YYYY-MM-DD
  to: string[];
  cc?: string[];
}

export interface SendSaokeEmailResult {
  status: string;
  message: string;
}

class BillingService {
  /** Upload one bank-transfer result workbook for a cycle of a work month. */
  async uploadTransferResult(cycle: number, workMonth: string, file: File): Promise<TransferUploadResult> {
    const form = new FormData();
    form.append('cycle', String(cycle));
    form.append('work_month', workMonth);
    form.append('file', file);
    // The apiClient defaults Content-Type to application/json — FormData needs
    // an explicit multipart type or the server cannot parse the fields.
    const response = await apiClient.post<TransferUploadResult>(ENDPOINTS.uploads, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
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

  /** PRD §5.1: per-Kỳ reconciliation status of one work month. */
  async getReconStatus(workMonth: string): Promise<ReconStatus> {
    const response = await apiClient.get<ReconStatus>(`${ENDPOINTS.reconStatus}?work_month=${workMonth}`);
    return response.data ?? { work_month: workMonth, kyes: [] };
  }

  /** FR-6 Tổng quan: distinct employees per Kỳ for the last N months + fee (BR-1). */
  async getKySummary(months = 12): Promise<DashboardKySummary> {
    const response = await apiClient.get<DashboardKySummary>(`${ENDPOINTS.kySummary}?months=${months}`);
    return response.data ?? { months: [], fee_per_employee_per_ky_vnd: 0 };
  }

  /** Generate (or regenerate) the monthly billing report. Admin/giamdoc/ketoan. */
  async generateReport(workMonth: string): Promise<{ report: BillingReport; lines: BillingReportLine[] }> {
    const response = await apiClient.post<{ report: BillingReport; lines: BillingReportLine[] }>(
      ENDPOINTS.generateReport,
      { work_month: workMonth },
    );
    return response.data as { report: BillingReport; lines: BillingReportLine[] };
  }

  /** One monthly report with its Kỳ lines. */
  async getReport(workMonth: string): Promise<(BillingReport & { lines?: BillingReportLine[] }) | null> {
    const response = await apiClient.get<{ report: BillingReport; lines?: BillingReportLine[] }>(
      `${ENDPOINTS.reports}?work_month=${encodeURIComponent(workMonth)}`,
    );
    const payload = response.data;
    if (!payload?.report) return null;
    return { ...payload.report, lines: payload.lines ?? [] };
  }

  /**
   * Sao kê recipient lists (To/CC) shared by the export/email dialogs.
   * The API persists a JSON array under the wire key `recipients`; the
   * frontend speaks `to` — mapped here so dialogs stay unchanged.
   */
  async getSaokeRecipients(): Promise<SaokeRecipients> {
    const response = await apiClient.get<{ recipients?: string[]; cc?: string[] }>(ENDPOINTS.saokeRecipients);
    return { to: response.data?.recipients ?? [], cc: response.data?.cc ?? [] };
  }

  /** Replace the persisted sao kê recipient lists. */
  async updateSaokeRecipients(params: { to: string[]; cc: string[] }): Promise<SaokeRecipients> {
    const response = await apiClient.put<{ recipients?: string[]; cc?: string[] }>(
      ENDPOINTS.saokeRecipients,
      { recipients: params.to, cc: params.cc },
    );
    return {
      to: response.data?.recipients ?? params.to,
      cc: response.data?.cc ?? params.cc,
    };
  }

  /** Download the sao kê Excel workbook for one report date. */
  async exportSaoke(date: string): Promise<void> {
    const qs = new URLSearchParams({ report_date: date });
    await apiClient.download(`${ENDPOINTS.saokeExport}?${qs}`, `saoke_${date}.xlsx`);
  }

  /**
   * Send the sao kê email for one report date to the given To/CC lists.
   * The API body keys are `recipients`/`cc` — mapped from the frontend `to`.
   */
  async sendSaokeEmail(params: SendSaokeEmailParams): Promise<SendSaokeEmailResult> {
    const response = await apiClient.post<SendSaokeEmailResult>(
      ENDPOINTS.saokeSendEmail,
      {
        report_date: params.report_date,
        recipients: params.to,
        cc: params.cc ?? [],
      },
      {
        headers: {
          'Idempotency-Key': createIdempotencyKey('saoke-email'),
        },
      },
    );
    return response.data ?? { status: response.status, message: response.message || '' };
  }

  /** Get bank transfer payment histories with filters. */
  async getBankTransferHistories(params?: Record<string, unknown>): Promise<unknown> {
    const response = await apiClient.get('/payrolls/bank-transfer-histories', { params });
    return response.data;
  }

  /** Export bank transfer payment histories as Excel. */
  async exportBankTransferHistories(params?: Record<string, unknown>): Promise<Blob> {
    const response = await apiClient.get<Blob>('/payrolls/bank-transfer-histories/export', {
      params,
      responseType: 'blob',
    });
    return response.data;
  }
}

export const billingService = new BillingService();
