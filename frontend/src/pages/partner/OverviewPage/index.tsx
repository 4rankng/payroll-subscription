import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import type { ApiError } from '@/services/api/client';
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
import { useBillingKySummary } from '@/hooks/api/useBillingDashboard';
import {
  buildFeeTrendSeries,
  buildMonthRows,
  formatVnd,
  KY_IDS,
} from './overviewData';

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

  if (kySummary.isError) {
    const error = kySummary.error as unknown as ApiError;
    // Non-JSON failure bodies (proxy 502/504 HTML, empty panic response)
    // carry no message — show the HTTP status or a connection fallback
    // instead of an empty string after the colon.
    const detail =
      error?.message?.trim() ||
      (error?.http_status ? `Lỗi máy chủ (HTTP ${error.http_status})` : "Lỗi kết nối đến máy chủ.");
    return (
      <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
        <PageHeader
          title="Tổng quan"
          description="Insights phí subscription"
          kicker={getConsoleKicker(location.pathname)}
        />
        <Card>
          <CardContent className="py-10 text-center text-sm text-destructive">
            Không thể tải dữ liệu tổng quan: {detail}
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
                    <TableHead className="text-right">Tổng</TableHead>
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
                Xu hướng phí theo tháng (VND — đơn giá {formatVnd(feePerEmployeePerKyVnd)}/Tổng)
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
        </>
      )}
    </div>
  );
};

export default FeeInsightsPage;
