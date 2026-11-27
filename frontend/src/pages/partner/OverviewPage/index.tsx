import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { FileCheck2, FileWarning, FileX2 } from 'lucide-react';

import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useBillingKySummary, useBillingReconStatus } from '@/hooks/api/useBillingDashboard';
import {
  buildFeeTrendSeries,
  buildMonthRows,
  buildReconCards,
  formatVnd,
  KY_IDS,
  type ReconKyCard,
  type ReconState,
} from './overviewData';

const RECON_BADGE_VARIANT: Record<ReconState, 'secondary' | 'warning' | 'success'> = {
  waiting_file: 'secondary',
  mismatch: 'warning',
  reconciled: 'success',
};

const RECON_ICON: Record<ReconState, typeof FileCheck2> = {
  waiting_file: FileX2,
  mismatch: FileWarning,
  reconciled: FileCheck2,
};

function ReconCard({ card }: { card: ReconKyCard }) {
  const Icon = RECON_ICON[card.state];
  return (
    <Card className="min-w-0">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-semibold">Kỳ {card.ky}</CardTitle>
        <Badge variant={RECON_BADGE_VARIANT[card.state]}>{card.stateLabel}</Badge>
      </CardHeader>
      <CardContent className="space-y-1.5 text-sm">
        <p className="flex items-center gap-2 text-muted-foreground">
          <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
          {card.uploads} file tải lên
        </p>
        <p className="tabular-nums">
          {card.matchedRows.toLocaleString('vi-VN')} / {card.parsedRows.toLocaleString('vi-VN')} dòng khớp
        </p>
        {card.mismatchRows > 0 && (
          <p className="text-xs font-medium text-destructive tabular-nums">
            {card.mismatchRows.toLocaleString('vi-VN')} dòng lệch
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function KySummarySkeleton() {
  return (
    <div className="space-y-3" aria-busy="true" aria-label="Đang tải dữ liệu tổng quan">
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-64 w-full" />
      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        {KY_IDS.map((ky) => (
          <Skeleton key={ky} className="h-28 w-full" />
        ))}
      </div>
    </div>
  );
}

/**
 * Tổng quan (Fee Insights) — partner portal. Monthly Kỳ distinct table with
 * the tổng cộng row, the fee trend chart (recharts) and the per-Kỳ recon
 * status cards. Mobile variant lives in Mobile.tsx.
 */
const FeeInsightsPage = () => {
  const location = useLocation();
  const kySummary = useBillingKySummary();
  const recon = useBillingReconStatus();

  if (kySummary.isError || recon.isError) {
    const error = (kySummary.error ?? recon.error) as Error;
    return (
      <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
        <PageHeader
          title="Tổng quan"
          description="Insights phí subscription"
          kicker={getConsoleKicker(location.pathname)}
        />
        <Card>
          <CardContent className="py-10 text-center text-sm text-destructive">
            Không thể tải dữ liệu tổng quan: {error.message}
          </CardContent>
        </Card>
      </div>
    );
  }

  const feePerEmployeePerKyVnd = kySummary.data?.fee_per_employee_per_ky_vnd ?? 0;
  const { rows, total } = buildMonthRows(
    kySummary.data?.months ?? [],
    feePerEmployeePerKyVnd,
  );
  const trendSeries = buildFeeTrendSeries(
    kySummary.data?.months ?? [],
    feePerEmployeePerKyVnd,
  );
  const reconCards = buildReconCards(recon.data?.kyes ?? []);

  return (
    <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
      <PageHeader
        title="Tổng quan"
        description="Insights phí subscription"
        kicker={getConsoleKicker(location.pathname)}
      />

      {kySummary.isLoading ? (
        <KySummarySkeleton />
      ) : (
        <>
          <Card>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tháng</TableHead>
                    {KY_IDS.map((ky) => (
                      <TableHead key={ky} className="text-right">
                        Kỳ {ky}
                      </TableHead>
                    ))}
                    <TableHead className="text-right">NV×kỳ</TableHead>
                    <TableHead className="text-right">Phí tháng (VND)</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((row) => (
                    <TableRow key={row.workMonth}>
                      <TableCell className="font-medium">{row.label}</TableCell>
                      {row.kyCells.map((cell) => (
                        <TableCell key={cell.ky} className="text-right tabular-nums">
                          {cell.distinct.toLocaleString('vi-VN')}
                        </TableCell>
                      ))}
                      <TableCell className="text-right font-semibold tabular-nums">
                        {row.employeeKyTotal.toLocaleString('vi-VN')}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {formatVnd(row.feeVnd)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
                <TableFooter>
                  <TableRow>
                    <TableCell className="font-bold">{total.label}</TableCell>
                    {KY_IDS.map((ky) => (
                      <TableCell key={ky} className="text-right font-bold tabular-nums">
                        {total.kyTotals[ky].toLocaleString('vi-VN')}
                      </TableCell>
                    ))}
                    <TableCell className="text-right font-bold tabular-nums">
                      {total.employeeKyTotal.toLocaleString('vi-VN')}
                    </TableCell>
                    <TableCell className="text-right font-bold tabular-nums">
                      {formatVnd(total.feeVnd)}
                    </TableCell>
                  </TableRow>
                </TableFooter>
              </Table>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Xu hướng phí theo tháng (VND — đơn giá {formatVnd(feePerEmployeePerKyVnd)}/NV×kỳ)
              </CardTitle>
            </CardHeader>
            <CardContent>
              {trendSeries.length === 0 ? (
                <p className="py-12 text-center text-sm text-muted-foreground">
                  Chưa có dữ liệu phí
                </p>
              ) : (
                <div className="h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={trendSeries} margin={{ top: 8, right: 8, left: 8, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} />
                      <XAxis dataKey="label" tickLine={false} axisLine={false} fontSize={12} />
                      <YAxis
                        tickFormatter={(value: number) => formatVnd(value)}
                        tickLine={false}
                        axisLine={false}
                        fontSize={12}
                        width={90}
                      />
                      <Tooltip
                        formatter={(value) => [`${formatVnd(Number(value))} VND`, 'Phí tháng']}
                      />
                      <Bar dataKey="feeVnd" name="Phí tháng" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </CardContent>
          </Card>

          {recon.isLoading ? (
            <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
              {KY_IDS.map((ky) => (
                <Skeleton key={ky} className="h-28 w-full" />
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
              {reconCards.map((card) => (
                <ReconCard key={card.ky} card={card} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default FeeInsightsPage;
