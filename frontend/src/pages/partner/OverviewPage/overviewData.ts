/**
 * Tổng quan (Fee Insights) — pure data shaping for the partner overview
 * page. No React, no side effects: table rows, chart series and the recon
 * card model are derived here so both desktop and mobile render from the
 * same logic.
 */

/** Kỳ ids are the pay cycles 1..4; API counts keys are strings. */
export const KY_IDS = [1, 2, 3, 4] as const;

export type KyId = (typeof KY_IDS)[number];

export interface KyMonthCounts {
  work_month: string;
  counts: Record<string, number>;
}

export interface KySummaryData {
  months: KyMonthCounts[];
  fee_per_employee_per_ky_vnd: number;
}

export type ReconState = 'waiting_file' | 'mismatch' | 'reconciled';

export interface ReconKy {
  cycle: number;
  uploads: number;
  parsed_rows: number;
  matched_rows: number;
  state: ReconState;
}

export interface ReconStatusData {
  work_month: string;
  kyes: ReconKy[];
}

export interface KyColumnCell {
  ky: KyId;
  /** Distinct employees billed in this Kỳ for the month (0 when absent). */
  distinct: number;
}

/** One month row of the Tổng quan table. */
export interface OverviewMonthRow {
  /** "2026-11" raw value, used as React key and chart axis id. */
  workMonth: string;
  /** "T11/2026" display label. */
  label: string;
  kyCells: KyColumnCell[];
  /** Sum of distinct employees across Kỳ 1–4. */
  employeeKyTotal: number;
  /** employeeKyTotal × fee per employee per Kỳ, in VND. */
  feeVnd: number;
}

/** Final "Tổng cộng" row aggregating all months. */
export interface OverviewTotalRow {
  label: string;
  kyTotals: Record<KyId, number>;
  employeeKyTotal: number;
  feeVnd: number;
}

/** Chart point: one per month, ascending. */
export interface FeeTrendPoint {
  workMonth: string;
  label: string;
  feeVnd: number;
}

export interface ReconKyCard {
  /** Pay-cycle id; widened to number since the API may send any cycle. */
  ky: number;
  uploads: number;
  parsedRows: number;
  matchedRows: number;
  state: ReconState;
  stateLabel: string;
  /** Rows found in the workbook but not matched to timesheets. */
  mismatchRows: number;
}

/** "2026-11" → "T11/2026". */
export function formatWorkMonth(workMonth: string): string {
  const [year, month] = workMonth.split('-');
  if (!year || !month) return workMonth;
  return `T${month}/${year}`;
}

const vndFormatter = new Intl.NumberFormat('vi-VN', {
  style: 'decimal',
  maximumFractionDigits: 0,
});

/** Fee formatting per contract: vi-VN, no decimals, "4.900.000" style. */
export function formatVnd(amount: number): string {
  return vndFormatter.format(Math.round(amount));
}

function distinctOf(counts: Record<string, number>, ky: KyId): number {
  return counts[String(ky)] ?? 0;
}

/** Sort months ascending (oldest first) so the table reads chronologically. */
export function sortMonthsAscending(months: KyMonthCounts[]): KyMonthCounts[] {
  return [...months].sort((a, b) => a.work_month.localeCompare(b.work_month));
}

/** Build the month table rows (ascending) plus the final tổng cộng row. */
export function buildMonthRows(
  months: KyMonthCounts[],
  feePerEmployeePerKyVnd: number,
): { rows: OverviewMonthRow[]; total: OverviewTotalRow } {
  const rows: OverviewMonthRow[] = sortMonthsAscending(months).map((month) => {
    const kyCells: KyColumnCell[] = KY_IDS.map((ky) => ({
      ky,
      distinct: distinctOf(month.counts, ky),
    }));
    const employeeKyTotal = kyCells.reduce((sum, cell) => sum + cell.distinct, 0);
    return {
      workMonth: month.work_month,
      label: formatWorkMonth(month.work_month),
      kyCells,
      employeeKyTotal,
      feeVnd: employeeKyTotal * feePerEmployeePerKyVnd,
    };
  });

  const kyTotals: Record<KyId, number> = { 1: 0, 2: 0, 3: 0, 4: 0 };
  for (const row of rows) {
    for (const cell of row.kyCells) {
      kyTotals[cell.ky] += cell.distinct;
    }
  }
  const employeeKyTotal = rows.reduce((sum, row) => sum + row.employeeKyTotal, 0);

  return {
    rows,
    total: {
      label: 'Tổng cộng',
      kyTotals,
      employeeKyTotal,
      feeVnd: employeeKyTotal * feePerEmployeePerKyVnd,
    },
  };
}

/** Fee trend series for the chart, ascending by month. */
export function buildFeeTrendSeries(
  months: KyMonthCounts[],
  feePerEmployeePerKyVnd: number,
): FeeTrendPoint[] {
  return sortMonthsAscending(months).map((month) => {
    const employeeKyTotal = KY_IDS.reduce(
      (sum, ky) => sum + distinctOf(month.counts, ky),
      0,
    );
    return {
      workMonth: month.work_month,
      label: formatWorkMonth(month.work_month),
      feeVnd: employeeKyTotal * feePerEmployeePerKyVnd,
    };
  });
}

export const RECON_STATE_LABELS: Record<ReconState, string> = {
  waiting_file: 'Chờ file',
  mismatch: 'Có dòng lệch',
  reconciled: 'Đã đối chiếu',
};

/** Recon card model with Vietnamese state labels and mismatch counts. */
export function buildReconCards(kyes: ReconKy[]): ReconKyCard[] {
  return kyes.map((ky) => ({
    ky: ky.cycle,
    uploads: ky.uploads,
    parsedRows: ky.parsed_rows,
    matchedRows: ky.matched_rows,
    state: ky.state,
    stateLabel: RECON_STATE_LABELS[ky.state] ?? ky.state,
    mismatchRows: Math.max(0, ky.parsed_rows - ky.matched_rows),
  }));
}
