/**
 * Sổ cái / sao kê data shaping (PAYR-41). Pure functions only — pages and
 * dialogs render the results; the .ts = logic rule keeps derivation here.
 */
import type { EmployeePayCycleRow, SaokeRecipients, TransferUploadSummary } from '@/services/api/billing.service';
import { DEFAULT_SAOKE_CC, DEFAULT_SAOKE_RECIPIENTS } from '@/constants/emailDefaults';
import { format, parseISO } from 'date-fns';

/** YYYY-MM-DD → dd/MM/yyyy without UTC day-shift. */
export function formatReportDate(date: string): string {
  return format(parseISO(date), 'dd/MM/yyyy');
}

/** One transfer-upload batch plus the money it settled (from ky-paid rows). */
export interface LedgerBatch extends TransferUploadSummary {
  /** Sum of paid_amount of rows sourced from this upload (VND). */
  total_amount_vnd: number;
}

/** Compact "what the ledger currently holds" figure set for the dialogs. */
export interface SaokeSummary {
  /** YYYY-MM scope of the ledger data behind the figures. */
  workMonth: string;
  batchCount: number;
  matchedRows: number;
  totalAmountVnd: number;
}

/** Groups per-employee payments by the upload that produced them. */
export function batchAmountTotals(paidRows: EmployeePayCycleRow[]): Map<number, number> {
  const totals = new Map<number, number>();
  for (const row of paidRows) {
    totals.set(row.source_upload_id, (totals.get(row.source_upload_id) ?? 0) + row.paid_amount);
  }
  return totals;
}

/** Attaches the settled total to each upload batch, newest upload first. */
export function buildLedgerBatches(
  uploads: TransferUploadSummary[],
  paidRows: EmployeePayCycleRow[],
): LedgerBatch[] {
  const totals = batchAmountTotals(paidRows);
  return uploads.map((upload) => ({
    ...upload,
    total_amount_vnd: totals.get(upload.id) ?? 0,
  }));
}

export function buildSaokeSummary(workMonth: string, batches: LedgerBatch[]): SaokeSummary {
  return {
    workMonth,
    batchCount: batches.length,
    matchedRows: batches.reduce((sum, batch) => sum + batch.matched_rows, 0),
    totalAmountVnd: batches.reduce((sum, batch) => sum + batch.total_amount_vnd, 0),
  };
}

/**
 * First-run seeding: once the API has no lists yet, the canned defaults are
 * what the dialogs prefill — the first save then persists them.
 */
export function seedSaokeRecipients(current?: SaokeRecipients): SaokeRecipients {
  if (current && (current.to.length > 0 || current.cc.length > 0)) {
    return { to: [...current.to], cc: [...current.cc] };
  }
  return { to: [...DEFAULT_SAOKE_RECIPIENTS], cc: [...DEFAULT_SAOKE_CC] };
}
