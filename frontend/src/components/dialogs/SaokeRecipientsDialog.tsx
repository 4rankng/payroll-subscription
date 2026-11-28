import { useEffect, useState } from 'react';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
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
      <DialogContent className="max-w-lg gap-0 overflow-hidden rounded-xl p-0 sm:max-w-lg">
        <div className="flex flex-col items-start gap-1 border-b bg-card px-5 py-4">
          <DialogTitle className="text-base font-semibold text-foreground">
            Danh sách người nhận sao kê
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            Chỉnh sửa danh sách To và CC dùng cho các email sao kê.
          </DialogDescription>
        </div>

        <div className="space-y-4 p-5">
          <div>
            <Label
              htmlFor="saoke-recipients-to"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Người nhận (To) {to.length > 0 && <span className="font-normal">· {to.length}</span>}
            </Label>
            <TagInput
              id="saoke-recipients-to"
              value={to}
              onChange={(next) => {
                setTo(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Nhập email..."
              disabled={updateRecipients.isPending}
            />
            {to.length === 0 && (
              <p role="alert" className="mt-1.5 text-xs text-destructive">
                Cần ít nhất 1 người nhận.
              </p>
            )}
          </div>

          <div>
            <Label
              htmlFor="saoke-recipients-cc"
              className="mb-1.5 block text-xs font-medium text-muted-foreground"
            >
              Người nhận (CC) {cc.length > 0 && <span className="font-normal">· {cc.length}</span>}
            </Label>
            <TagInput
              id="saoke-recipients-cc"
              value={cc}
              onChange={(next) => {
                setCc(next);
                setDirty(true);
              }}
              validateToken={validateEmail}
              invalidTokenMessage={(token) => `"${token}" không phải email hợp lệ`}
              placeholder="Nhập email..."
              disabled={updateRecipients.isPending}
            />
          </div>
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
          <Button type="button" onClick={handleSave} disabled={!canSave}>
            {updateRecipients.isPending ? 'Đang lưu...' : 'Lưu'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
