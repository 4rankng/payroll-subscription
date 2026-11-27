import { useEffect, useState } from 'react';

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogNavyHeader,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { TagInput } from '@/components/ui/tag-input';
import { useSaokeRecipients, useUpdateSaokeRecipients } from '@/hooks/api/useSaoke';
import { validateEmail } from '@/utils/validators';
import { seedSaokeRecipients } from '@/utils/saoke';

interface SaokeRecipientsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Danh sách người nhận sao kê: edit the persisted To/CC lists used to prefill
 * the export/email flows. Seeded with the canned defaults on first run.
 */
export function SaokeRecipientsDialog({ open, onOpenChange }: SaokeRecipientsDialogProps) {
  const [to, setTo] = useState<string[]>([]);
  const [cc, setCc] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);

  const recipientsQuery = useSaokeRecipients();
  const updateRecipients = useUpdateSaokeRecipients();

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

  const canSave = to.length > 0 && !updateRecipients.isPending;

  const handleSave = async () => {
    try {
      await updateRecipients.mutateAsync({ to, cc });
      onOpenChange(false);
    } catch {
      // Error notification is owned by the hook; keep the dialog open to retry.
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" hideCloseButton>
        <DialogNavyHeader
          title="Danh sách người nhận sao kê"
          description="Chỉnh sửa danh sách To và CC dùng cho các email sao kê."
        />

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="saoke-recipients-to">Người nhận (To)</Label>
            <TagInput
              id="saoke-recipients-to"
              value={to}
              onChange={(next) => {
                setTo(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="email1@congty.vn, email2@congty.vn"
              disabled={updateRecipients.isPending}
            />
            {to.length === 0 && (
              <p role="alert" className="text-xs text-destructive">
                Cần ít nhất 1 người nhận.
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="saoke-recipients-cc">Người nhận cc (CC)</Label>
            <TagInput
              id="saoke-recipients-cc"
              value={cc}
              onChange={(next) => {
                setCc(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Không bắt buộc"
              disabled={updateRecipients.isPending}
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Đóng
          </Button>
          <Button type="button" onClick={handleSave} disabled={!canSave}>
            {updateRecipients.isPending ? 'Đang lưu...' : 'Lưu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
