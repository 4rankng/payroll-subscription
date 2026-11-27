import { apiClient } from './client';

/**
 * Sao kê module (PRD §5.8, PAYR-41): statement export, persisted recipient
 * list and statement email — ketoan/giamdoc surface. Self-contained: owns
 * its endpoint paths and response types. API responses use the platform
 * envelope { status, data }.
 */

const ENDPOINTS = {
  export: '/billing/saoke/export',
  recipients: '/billing/saoke/recipients',
  email: '/billing/saoke/email',
} as const;

/** The persisted statement mailing list (To + CC). */
export interface SaoKeRecipients {
  recipients: string[];
  cc: string[];
}

export interface SaoKeEmailResult {
  message: string;
}

class SaoKeService {
  /** Read the persisted statement mailing list. Missing rows read as empty. */
  async getRecipients(): Promise<SaoKeRecipients> {
    const response = await apiClient.get<SaoKeRecipients>(ENDPOINTS.recipients);
    return response.data ?? { recipients: [], cc: [] };
  }

  /** Validate and persist the statement mailing list. */
  async saveRecipients(payload: SaoKeRecipients): Promise<SaoKeRecipients> {
    const response = await apiClient.put<SaoKeRecipients>(ENDPOINTS.recipients, payload);
    return response.data ?? payload;
  }

  /**
   * Download the statement workbook for the Kỳ the report date resolves to.
   * The backend answers with an xlsx attachment; apiClient.download triggers
   * the blob download with the server-provided filename.
   */
  async exportStatement(reportDate: string): Promise<void> {
    await apiClient.download(`${ENDPOINTS.export}?report_date=${encodeURIComponent(reportDate)}`);
  }

  /** Build the statement for the report date and email it to the recipients. */
  async sendStatementEmail(reportDate: string, recipients: string[], cc: string[]): Promise<SaoKeEmailResult> {
    const response = await apiClient.post<SaoKeEmailResult>(ENDPOINTS.email, {
      report_date: reportDate,
      recipients,
      cc,
    });
    return response.data ?? { message: 'Email sao kê đã được gửi' };
  }
}

export const saoKeService = new SaoKeService();
