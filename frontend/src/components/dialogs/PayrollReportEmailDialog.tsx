import { useEffect, useState } from 'react';

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogNavyHeader,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { TagInput } from '@/components/ui/tag-input';
import { useSaokeRecipients, useSendSaokeEmail } from '@/hooks/api/useSaoke';
import { validateEmail } from '@/utils/validators';
import { seedSaokeRecipients, type SaokeSummary } from '@/utils/saoke';
import { SaokePreviewPanel } from '@/components/dialogs/SaokePreviewPanel';

interface PayrollReportEmailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Ledger figures previewed by "Xem sao kê". */
  summary: SaokeSummary;
}

/**
 * Gửi sao kê qua email: report date (required), To (min 1) and CC chip
 * lists prefilled from the persisted recipients, preview, then send.
 */
export function PayrollReportEmailDialog({
  open,
  onOpenChange,
  summary,
}: PayrollReportEmailDialogProps) {
  const now = new Date();
  const [reportDate, setReportDate] = useState(
    `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`,
  );
  const [to, setTo] = useState<string[]>([]);
  const [cc, setCc] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  const recipientsQuery = useSaokeRecipients();
  const sendEmail = useSendSaokeEmail();

  // Prefill To/CC from the persisted lists (seeded defaults on first run)
  // until the user edits them for this email.
  useEffect(() => {
    if (!open) {
      setDirty(false);
      return;
    }
    if (dirty) return;
    const seeded = seedSaokeRecipients(recipientsQuery.data);
    setTo(seeded.to);
    setCc(seeded.cc);
  }, [open, dirty, recipientsQuery.data]);

  const canSend = reportDate.length > 0 && to.length > 0 && !sendEmail.isPending;

  const handleSend = async () => {
    try {
      await sendEmail.mutateAsync({
        report_date: reportDate,
        to,
        cc,
      });
      onOpenChange(false);
    } catch {
      // Error notification is owned by the hook; keep the dialog open to retry.
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" hideCloseButton>
        <DialogNavyHeader
          title="Gửi sao kê qua email"
          description="Chọn ngày báo cáo và danh sách người nhận, xem trước rồi gửi."
        />

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="saoke-email-report-date">Ngày báo cáo</Label>
            <Input
              id="saoke-email-report-date"
              type="date"
              value={reportDate}
              onChange={(event) => {
                setReportDate(event.target.value);
                setShowPreview(false);
              }}
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="saoke-email-to">Người nhận (To)</Label>
            <TagInput
              id="saoke-email-to"
              value={to}
              onChange={(next) => {
                setTo(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="email1@congty.vn, email2@congty.vn"
              disabled={sendEmail.isPending}
            />
            {to.length === 0 && (
              <p role="alert" className="text-xs text-destructive">
                Cần ít nhất 1 người nhận.
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="saoke-email-cc">Người nhận cc (CC)</Label>
            <TagInput
              id="saoke-email-cc"
              value={cc}
              onChange={(next) => {
                setCc(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Không bắt buộc"
              disabled={sendEmail.isPending}
            />
          </div>

          {showPreview && <SaokePreviewPanel summary={summary} reportDate={reportDate} />}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Đóng
          </Button>
          <Button type="button" variant="outline" onClick={() => setShowPreview(true)}>
            Xem sao kê
          </Button>
          <Button type="button" onClick={handleSend} disabled={!canSend}>
            {sendEmail.isPending ? 'Đang gửi...' : 'Gửi email'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
