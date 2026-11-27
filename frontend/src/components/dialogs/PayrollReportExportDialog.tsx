import { useState } from 'react';

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogNavyHeader,
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
      <DialogContent className="max-w-lg" hideCloseButton>
        <DialogNavyHeader
          title="Xuất sao kê"
          description="Chọn ngày báo cáo và tải file sao kê Excel."
        />

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="saoke-export-report-date">Ngày báo cáo</Label>
            <Input
              id="saoke-export-report-date"
              type="date"
              value={reportDate}
              onChange={(event) => {
                setReportDate(event.target.value);
                setShowPreview(false);
              }}
            />
          </div>

          {showPreview && <SaokePreviewPanel summary={summary} reportDate={reportDate} />}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Đóng
          </Button>
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
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
