import { useBillingReport, useGenerateBillingReport } from '@/hooks/api/useBilling';
import { formatCurrency, formatDateTime } from '@/utils/formatters';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface ReportPanelProps {
  workMonth: string;
  canGenerate: boolean;
}

/**
 * FR-3/D24: month-end billing report. Generate (giamdoc/admin) then read the
 * per-Kỳ lines; the Excel file path comes back on the report row for the
 * export flow.
 */
export function ReportPanel({ workMonth, canGenerate }: ReportPanelProps) {
  const { data: report } = useBillingReport(workMonth);
  const lines = report?.lines;
  const generate = useGenerateBillingReport(workMonth);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <div className="space-y-1.5">
          <CardTitle>Báo cáo phí tháng {workMonth}</CardTitle>
          <CardDescription>
            Tổng hợp số nhân viên distinct theo kỳ × đơn giá. File Excel dùng để gửi yêu cầu thanh
            toán.
          </CardDescription>
        </div>
        {canGenerate && (
          <Button onClick={() => generate.mutate()} disabled={generate.isPending}>
            {generate.isPending ? 'Đang tạo...' : 'Tạo / cập nhật báo cáo'}
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {report === null || report === undefined ? (
          <p className="text-sm text-slate-500">Chưa có báo cáo cho tháng này.</p>
        ) : (
          <div className="space-y-3">
            <div className="text-sm text-slate-600">
              Tạo lúc {formatDateTime(report.generated_at)} — đơn giá{" "}
              {formatCurrency(report.fee_per_employee_per_ky_vnd)}/nhân viên/kỳ
              {report.excel_path !== null && report.excel_path !== "" && (
                <span className="ml-2 font-mono text-xs">{report.excel_path}</span>
              )}
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kỳ</TableHead>
                  <TableHead>Số nhân viên</TableHead>
                  <TableHead className="text-right">Thành tiền</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(lines ?? []).map((line) => (
                  <TableRow key={line.cycle}>
                    <TableCell>Kỳ {line.cycle}</TableCell>
                    <TableCell>{line.distinct_count}</TableCell>
                    <TableCell className="text-right">
                      {formatCurrency(line.amount_vnd)}
                    </TableCell>
                  </TableRow>
                ))}
                <TableRow>
                  <TableCell className="font-medium">Tổng</TableCell>
                  <TableCell className="font-medium">{report.total_distinct}</TableCell>
                  <TableCell className="text-right font-medium">
                    {formatCurrency(report.total_vnd)}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
