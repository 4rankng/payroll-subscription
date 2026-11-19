import { useBillingUploads, useEmployeePayCycles } from '@/hooks/api/useBilling';
import { formatCurrency, formatDateTime } from '@/utils/formatters';
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
import { Badge } from '@/components/ui/badge';

const STATUS_LABELS: Record<string, { label: string; className: string }> = {
  verified: { label: 'Đã xác minh', className: 'bg-emerald-100 text-emerald-800' },
  pending: { label: 'Chờ xử lý', className: 'bg-slate-100 text-slate-700' },
  rejected: { label: 'Từ chối', className: 'bg-red-100 text-red-800' },
};

interface UploadHistoryPanelProps {
  workMonth: string;
}

/** FR-2: upload history for one work month. */
export function UploadHistoryPanel({ workMonth }: UploadHistoryPanelProps) {
  const { data: uploads, isLoading } = useBillingUploads(workMonth);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Lịch sử tải lên</CardTitle>
        <CardDescription>Kết quả chuyển khoản đã nộp trong tháng {workMonth}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="text-sm text-slate-500">Đang tải...</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kỳ</TableHead>
                <TableHead>File</TableHead>
                <TableHead>Trạng thái</TableHead>
                <TableHead>Thời gian</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(uploads ?? []).map((u) => {
                const status = STATUS_LABELS[u.status] ?? STATUS_LABELS.pending;
                return (
                  <TableRow key={u.id}>
                    <TableCell>Kỳ {u.cycle}</TableCell>
                    <TableCell className="max-w-64 truncate">{u.original_filename}</TableCell>
                    <TableCell>
                      <Badge className={status.className}>{status.label}</Badge>
                      {u.reject_reason && (
                        <span className="ml-2 text-xs text-red-700">{u.reject_reason}</span>
                      )}
                    </TableCell>
                    <TableCell>{formatDateTime(u.created_at)}</TableCell>
                  </TableRow>
                );
              })}
              {(uploads ?? []).length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-slate-500">
                    Chưa có file nào
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

interface EmployeePayCycleSummaryProps {
  workMonth: string;
  feeVnd: number;
}

/** FR-1/BR-2 view: distinct employees per Kỳ and the fee they bill. */
export function EmployeePayCycleSummary({ workMonth, feeVnd }: EmployeePayCycleSummaryProps) {
  const { data, isLoading } = useEmployeePayCycles(workMonth);
  const summary = data?.summaryByCycle ?? {};
  const cycles = [1, 2, 3, 4].filter((c) => (summary[String(c)] ?? 0) > 0);
  const total = cycles.reduce((sum, c) => sum + (summary[String(c)] ?? 0) * feeVnd, 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Số nhân viên theo kỳ</CardTitle>
        <CardDescription>
          Đếm distinct theo CCCD mỗi kỳ × {formatCurrency(feeVnd)} / nhân viên / kỳ
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="text-sm text-slate-500">Đang tải...</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kỳ</TableHead>
                <TableHead>Số nhân viên</TableHead>
                <TableHead className="text-right">Thành tiền</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cycles.map((c) => (
                <TableRow key={c}>
                  <TableCell>Kỳ {c}</TableCell>
                  <TableCell>{summary[String(c)]}</TableCell>
                  <TableCell className="text-right">
                    {formatCurrency((summary[String(c)] ?? 0) * feeVnd)}
                  </TableCell>
                </TableRow>
              ))}
              {cycles.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-slate-500">
                    Chưa có dữ liệu đã xác minh
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
        {cycles.length > 0 && (
          <p className="mt-3 text-sm font-medium">
            Tổng tháng {workMonth}: {formatCurrency(total)}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
