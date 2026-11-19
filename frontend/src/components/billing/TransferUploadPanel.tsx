import { useRef, useState } from 'react';

import { useTransferUpload } from '@/hooks/api/useBilling';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import type { TransferUploadResult } from '@/services/api/billing.service';

const CYCLES = [1, 2, 3, 4] as const;

interface TransferUploadPanelProps {
  workMonth: string;
  onUploaded: () => void;
}

/**
 * FR-2 upload panel: one bank-transfer result workbook per cycle of a work
 * month. Per-row failures are shown inline (process-rows-in-file semantics);
 * a whole-file rejection means zero rows processed.
 */
export function TransferUploadPanel({ workMonth, onUploaded }: TransferUploadPanelProps) {
  const [cycle, setCycle] = useState<number>(1);
  const [file, setFile] = useState<File | null>(null);
  const [outcome, setOutcome] = useState<TransferUploadResult | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const upload = useTransferUpload(workMonth);

  const canSubmit = file !== null && workMonth.length === 7 && !upload.isPending;

  const handleSubmit = () => {
    if (!canSubmit || file === null) return;
    upload.mutate(
      { cycle, file },
      {
        onSuccess: (result) => {
          setOutcome(result);
          setFile(null);
          if (fileInputRef.current) fileInputRef.current.value = '';
          if (result.rows_ok > 0) onUploaded();
        },
      },
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tải lên kết quả chuyển khoản</CardTitle>
        <CardDescription>
          File Excel kết quả chuyển khoản cho một kỳ trong tháng {workMonth}. Xử lý theo từng dòng
          trong file — dòng lỗi không chặn các dòng khác.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="billing-cycle">Kỳ</Label>
            <Select value={String(cycle)} onValueChange={(v) => setCycle(Number(v))}>
              <SelectTrigger id="billing-cycle">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CYCLES.map((c) => (
                  <SelectItem key={c} value={String(c)}>
                    Kỳ {c}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="billing-file">File kết quả chuyển khoản (.xlsx)</Label>
            <Input
              id="billing-file"
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.xls"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </div>
        </div>

        <Button onClick={handleSubmit} disabled={!canSubmit}>
          {upload.isPending ? 'Đang xử lý...' : 'Tải lên và đối soát'}
        </Button>

        {outcome && <UploadOutcome outcome={outcome} />}
      </CardContent>
    </Card>
  );
}

function UploadOutcome({ outcome }: { outcome: TransferUploadResult }) {
  if (outcome.duplicate) {
    return (
      <div className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm text-slate-600">
        File trùng nội dung đã xử lý trước đó (#{outcome.upload_id}) — không xử lý lại.
      </div>
    );
  }

  const rejected = outcome.status === 'rejected';

  return (
    <div
      className={`rounded-md border p-3 text-sm ${
        rejected ? 'border-red-200 bg-red-50 text-red-900' : 'border-emerald-200 bg-emerald-50 text-emerald-900'
      }`}
    >
      <p className="font-medium">
        {rejected ? 'Từ chối toàn bộ file' : 'Đã xác minh'} — {outcome.rows_ok}/{outcome.rows_total} dòng
        hợp lệ
      </p>
      {rejected && (
        <p className="mt-1">Đã gửi thông báo đẩy đến tất cả quản trị viên. Kỳ bị đóng băng cho đến khi có file sửa đúng.</p>
      )}
      {outcome.failures.length > 0 && <RowFailures failures={outcome.failures} />}
    </div>
  );
}

function RowFailures({
  failures,
}: {
  failures: { row: number; cccd: string; reason: string }[];
}) {
  return (
    <table className="mt-2 w-full text-xs">
      <thead>
        <tr className="text-left">
          <th className="py-1 pr-2">Dòng</th>
          <th className="py-1 pr-2">CCCD</th>
          <th className="py-1">Lý do</th>
        </tr>
      </thead>
      <tbody>
        {failures.map((f) => (
          <tr key={`${f.row}-${f.cccd}`} className="border-t border-slate-200">
            <td className="py-1 pr-2 font-mono">{f.row}</td>
            <td className="py-1 pr-2 font-mono">{f.cccd}</td>
            <td className="py-1">{f.reason}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
