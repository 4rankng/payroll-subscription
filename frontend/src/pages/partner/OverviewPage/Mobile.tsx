import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useBillingKySummary, useBillingReconStatus } from '@/hooks/api/useBillingDashboard';
import {
  buildMonthRows,
  buildReconCards,
  formatVnd,
  KY_IDS,
  type ReconState,
} from './overviewData';

const RECON_BADGE_VARIANT: Record<ReconState, 'secondary' | 'warning' | 'success'> = {
  waiting_file: 'secondary',
  mismatch: 'warning',
  reconciled: 'success',
};

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
          NV×kỳ: {employeeKyTotal.toLocaleString('vi-VN')} · Phí: {formatVnd(feeVnd)} VND
        </p>
      </CardContent>
    </Card>
  );
}

/**
 * Mobile variant of Tổng quan (Fee Insights): stacked month cards (4 Kỳ
 * chips + NV×kỳ + fee) and the recon cards. Fits 320px without horizontal
 * overflow; touch targets ≥ 44px.
 */
export const FeeInsightsPageMobile = () => {
  const location = useLocation();
  const kySummary = useBillingKySummary();
  const recon = useBillingReconStatus();

  const feePerEmployeePerKyVnd = kySummary.data?.fee_per_employee_per_ky_vnd ?? 0;
  const { rows, total } = buildMonthRows(kySummary.data?.months ?? [], feePerEmployeePerKyVnd);
  const reconCards = buildReconCards(recon.data?.kyes ?? []);

  return (
    <div className="mx-auto w-full max-w-full space-y-3 overflow-x-hidden p-3">
      <PageHeader
        title="Tổng quan"
        description="Insights phí subscription"
        kicker={getConsoleKicker(location.pathname)}
      />

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
                NV×kỳ: {total.employeeKyTotal.toLocaleString('vi-VN')}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {recon.isLoading ? (
        <div className="space-y-2">
          {KY_IDS.map((ky) => (
            <Skeleton key={ky} className="h-20 w-full" />
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          {reconCards.map((card) => (
            <Card key={card.ky} className="min-w-0">
              <CardContent className="min-h-[44px] flex items-center justify-between gap-2 py-2.5">
                <span className="text-sm font-semibold">Kỳ {card.ky}</span>
                <div className="flex min-w-0 flex-col items-end gap-0.5">
                  <Badge variant={RECON_BADGE_VARIANT[card.state]}>{card.stateLabel}</Badge>
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {card.matchedRows.toLocaleString('vi-VN')} / {card.parsedRows.toLocaleString('vi-VN')} dòng khớp
                  </span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};
