import { useState } from 'react';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useExportSaoke } from '@/hooks/api/useSaoke';
import type { SaokeSummary } from '@/utils/saoke';
import { SaokePreviewPanel } from '@/components/dialogs/SaokePreviewPanel';

interface PayrollReportExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Ledger figures previewed by "Xem trước". */
  summary: SaokeSummary;
}

/**
 * Xuất sao kê: pick the report date (Ngày báo cáo), preview the ledger
 * summary, then download the Excel workbook.
 */
export function PayrollReportExportDialog({
  open,
  onOpenChange,
  summary,
}: PayrollReportExportDialogProps) {
  const [reportDate, setReportDate] = useState(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  });
  const [showPreview, setShowPreview] = useState(false);
  const exportSaoke = useExportSaoke();

  const handleExport = async () => {
    try {
      await exportSaoke.mutateAsync(reportDate);
      onOpenChange(false);
    } catch {
      // Error notification is owned by the hook; keep the dialog open to retry.
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg gap-0 overflow-hidden rounded-xl p-0 sm:max-w-lg">
        <div className="flex items-start justify-between gap-3 border-b bg-card px-5 py-4">
          <DialogTitle className="text-base font-semibold text-foreground">
            Xuất sao kê
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            Chọn ngày báo cáo và tải file sao kê Excel.
          </DialogDescription>
        </div>

        <div className="space-y-4 p-5">
          <div>
            <Label
              htmlFor="saoke-export-report-date"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Ngày báo cáo
            </Label>
            <Input
              id="saoke-export-report-date"
              type="date"
              value={reportDate}
              onChange={(event) => {
                setReportDate(event.target.value);
                setShowPreview(false);
              }}
              className="h-9 w-48 tabular-nums"
            />
          </div>

          {showPreview && <SaokePreviewPanel summary={summary} reportDate={reportDate} />}
        </div>

        <DialogFooter className="justify-between border-t bg-muted/30 px-5 py-3.5">
          <Button
            type="button"
            variant="ghost"
            className="text-muted-foreground"
            onClick={() => onOpenChange(false)}
          >
            Đóng
          </Button>
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" onClick={() => setShowPreview(true)}>
              Xem trước
            </Button>
            <Button
              type="button"
              onClick={handleExport}
              disabled={exportSaoke.isPending || reportDate.length === 0}
            >
              {exportSaoke.isPending ? 'Đang xuất...' : 'Xuất Excel'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
