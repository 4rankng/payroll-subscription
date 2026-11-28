import { useEffect, useState } from 'react';

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
      <DialogContent className="max-w-lg gap-0 overflow-hidden rounded-xl p-0 sm:max-w-lg">
        {/* Header — bleeds to the rounded edges, white, hairline divider */}
        <div className="flex items-start justify-between gap-3 border-b bg-card px-5 py-4">
          <DialogTitle className="text-base font-semibold text-foreground">
            Gửi sao kê qua email
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            Chọn ngày báo cáo và danh sách người nhận, xem trước rồi gửi.
          </DialogDescription>
        </div>

        {/* Body */}
        <div className="space-y-4 p-5">
          <div>
            <Label
              htmlFor="saoke-email-report-date"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Ngày báo cáo
            </Label>
            <Input
              id="saoke-email-report-date"
              type="date"
              value={reportDate}
              onChange={(event) => {
                setReportDate(event.target.value);
                setShowPreview(false);
              }}
              className="h-9 w-48 tabular-nums"
              required
            />
          </div>

          <div>
            <Label
              htmlFor="saoke-email-to"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Người nhận (To) {to.length > 0 && <span className="font-normal">· {to.length}</span>}
            </Label>
            <TagInput
              id="saoke-email-to"
              value={to}
              onChange={(next) => {
                setTo(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Nhập email..."
              disabled={sendEmail.isPending}
            />
            {to.length === 0 && (
              <p role="alert" className="mt-1.5 text-xs text-destructive">
                Cần ít nhất 1 người nhận.
              </p>
            )}
          </div>

          <div>
            <Label
              htmlFor="saoke-email-cc"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Người nhận (CC) {cc.length > 0 && <span className="font-normal">· {cc.length}</span>}
            </Label>
            <TagInput
              id="saoke-email-cc"
              value={cc}
              onChange={(next) => {
                setCc(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Nhập email..."
              disabled={sendEmail.isPending}
            />
          </div>

          {showPreview && <SaokePreviewPanel summary={summary} reportDate={reportDate} />}
        </div>

        {/* Footer — Đóng separated left, actions grouped right */}
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
              Xem sao kê
            </Button>
            <Button type="button" onClick={handleSend} disabled={!canSend}>
              {sendEmail.isPending ? 'Đang gửi...' : 'Gửi email'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
