import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useBillingKySummary, useBillingReconStatus } from '@/hooks/api/useBillingDashboard';
import { authManager } from '@/lib/auth';
import { ReportPanel } from '@/components/billing/ReportPanel';
import { format, subMonths } from 'date-fns';
import { getCurrentMonthValue } from '@/utils/bankTransferHistoryHelpers';
import {
  buildMonthRows,
  formatVnd,
  KY_IDS,
} from './overviewData';

function MonthCard({
  label,
  kyCells,
  employeeKyTotal,
  feeVnd,
}: {
  label: string;
  kyCells: { ky: number; distinct: number }[];
  employeeKyTotal: number;
  feeVnd: number;
}) {
  return (
    <Card className="min-w-0">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-semibold">{label}</CardTitle>
        <span className="text-sm font-bold tabular-nums">{formatVnd(feeVnd)}</span>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="grid grid-cols-4 gap-1.5">
          {kyCells.map((cell) => (
            <div
              key={cell.ky}
              className="flex min-h-[44px] flex-col items-center justify-center rounded-md bg-muted/60 px-1 py-1"
            >
              <span className="text-[11px] font-medium text-muted-foreground">Kỳ {cell.ky}</span>
              <span className="text-sm font-semibold tabular-nums">
                {cell.distinct.toLocaleString('vi-VN')}
              </span>
            </div>
          ))}
        </div>
        <p className="text-xs text-muted-foreground tabular-nums">
          Tổng: {employeeKyTotal.toLocaleString('vi-VN')} · Phí: {formatVnd(feeVnd)} VND
        </p>
      </CardContent>
    </Card>
  );
}


const RECON_LABEL: Record<string, string> = {
  reconciled: 'Đã đối chiếu',
  waiting_file: 'Chờ file',
  mismatch: 'Có dòng lệch',
};

/** PRD §5.1 mobile: compact per-Kỳ reconciliation chips for the work month. */
function ReconStatusStrip({ workMonth }: { workMonth: string }) {
  const recon = useBillingReconStatus(workMonth);
  const byCycle = new Map((recon.data?.kyes ?? []).map((row) => [row.cycle, row]));

  return (
    <div className="grid grid-cols-2 gap-2">
      {[1, 2, 3, 4].map((ky) => {
        const row = byCycle.get(ky);
        const state = row?.state ?? 'waiting_file';
        return (
          <Card key={ky} className="min-w-0">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-1">
              <CardTitle className="text-[13px] font-semibold">Kỳ {ky}</CardTitle>
              <span
                className={`rounded-full px-2 py-0.5 text-[11px] font-semibold ${
                  state === 'reconciled'
                    ? 'bg-emerald-100 text-emerald-800'
                    : state === 'mismatch'
                      ? 'bg-amber-100 text-amber-800'
                      : 'bg-muted text-muted-foreground'
                }`}
              >
                {RECON_LABEL[state]}
              </span>
            </CardHeader>
            <CardContent className="text-[11px] tabular-nums text-muted-foreground">
              File: {row?.uploads ?? 0} · Khớp: {row ? `${row.matched_rows.toLocaleString('vi-VN')}/${row.parsed_rows.toLocaleString('vi-VN')}` : '0/0'}
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}

/**
 * Mobile variant of Tổng quan (Fee Insights): stacked month cards (4 Kỳ
 * chips + Tổng + fee). Fits 320px without horizontal
 * overflow; touch targets ≥ 44px.
 */
export const FeeInsightsPageMobile = () => {
  const location = useLocation();
  const kySummary = useBillingKySummary();

  const feePerEmployeePerKyVnd = kySummary.data?.fee_per_employee_per_ky_vnd ?? 0;
  const { rows, total } = buildMonthRows(kySummary.data?.months ?? [], feePerEmployeePerKyVnd);

  return (
    <div className="mx-auto w-full max-w-full space-y-3 overflow-x-hidden p-3">
      <PageHeader
        title="Tổng quan"
        description="Thống kê phí theo tháng"
        kicker={getConsoleKicker(location.pathname)}
      />

      <ReconStatusStrip workMonth={getCurrentMonthValue()} />

      {(authManager.getUserRole() === 'admin' || authManager.getUserRole() === 'giamdoc') && (
        <ReportPanel workMonth={format(subMonths(new Date(), 1), 'yyyy-MM')} canGenerate />
      )}

      {kySummary.isLoading ? (
        <div className="space-y-3" aria-busy="true" aria-label="Đang tải dữ liệu tổng quan">
          {KY_IDS.map((ky) => (
            <Skeleton key={ky} className="h-32 w-full" />
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          {rows.map((row) => (
            <MonthCard
              key={row.workMonth}
              label={row.label}
              kyCells={row.kyCells}
              employeeKyTotal={row.employeeKyTotal}
              feeVnd={row.feeVnd}
            />
          ))}
          <Card className="min-w-0 bg-muted/40">
            <CardContent className="py-3 text-sm">
              <div className="flex items-center justify-between font-bold">
                <span>Tổng cộng</span>
                <span className="tabular-nums">{formatVnd(total.feeVnd)}</span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground tabular-nums">
                Tổng: {total.employeeKyTotal.toLocaleString('vi-VN')}
              </p>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
};
