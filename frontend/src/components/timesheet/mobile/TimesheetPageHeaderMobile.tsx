import { useState } from 'react';
import { useLocation } from 'react-router-dom';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { getConsoleKicker } from '@/lib/console';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { Plus, MoreHorizontal, CheckSquare, X, FileUp, FileSpreadsheet, FileSpreadsheet as TableIcon, Trash2, Undo2 } from 'lucide-react';

interface TimesheetPageHeaderMobileProps {
  onAddTimesheet: () => void;
  onBulkApprove?: () => void;
  /** Admin-only "Bỏ duyệt hết" — reset all approved timesheets to pending. */
  onResetAll?: () => void;
  onBccHistory?: () => void;
  onBccUpload?: () => void;
  onRejectUnpaid?: () => void;
  userRole?: 'admin' | 'quanly';
  monthValue: string;
  onMonthChange: (value: string) => void;
}

export function TimesheetPageHeaderMobile({
  onAddTimesheet,
  onBulkApprove,
  onResetAll,
  onBccHistory,
  onBccUpload,
  onRejectUnpaid,
  userRole = 'admin',
  monthValue,
  onMonthChange,
}: TimesheetPageHeaderMobileProps) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);

  return (
    <section className="console-panel overflow-hidden">
      <MobilePageHeader
        title="Bảng công"
        kicker={getConsoleKicker(useLocation().pathname)}
        icon={TableIcon}
        sticky={false}
        bordered={false}
        className="bg-transparent"
        actionsLayout="stacked"
        actions={
          <div className="flex w-full min-w-0 items-center gap-2 sm:w-auto">
            <Button
              onClick={onAddTimesheet}
              className="h-11 min-w-0 flex-1 rounded-xl px-4 shadow-sm touch-manipulation sm:flex-none"
            >
              <Plus className="mr-1 h-4 w-4" />
              Nhập
            </Button>

            <Sheet open={open} onOpenChange={setOpen}>
              <SheetTrigger asChild>
                <Button variant="outline" size="icon" className="h-11 w-11 shrink-0 rounded-xl border-border bg-card shadow-none touch-manipulation" aria-label="Thêm tùy chọn">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </SheetTrigger>
              <SheetContent side="bottom" className="h-auto max-h-[80dvh] overflow-y-auto rounded-t-3xl border-[hsl(var(--surface-border))] bg-card px-4 pt-3">
              <div className="mx-auto mb-3 h-1 w-10 rounded-full bg-border" />
              <SheetHeader><SheetTitle>Tùy chọn</SheetTitle></SheetHeader>
              <div className="space-y-1 py-3" style={{ paddingBottom: "max(12px, calc(12px + env(safe-area-inset-bottom)))" }}>

                {/* Actions */}
                {(onBulkApprove || onResetAll || (onRejectUnpaid && userRole === 'admin')) && (
                  <div className="h-px bg-border mx-1 my-1" />
                )}

                {onBulkApprove && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBulkApprove(); close(); }}
                  >
                    <CheckSquare className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Duyệt hết</span>
                  </Button>
                )}

                {onResetAll && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onResetAll(); close(); }}
                  >
                    <Undo2 className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Bỏ duyệt hết</span>
                  </Button>
                )}

                {onBccUpload && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBccUpload(); close(); }}
                  >
                    <FileUp className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Tải lên BCC</span>
                  </Button>
                )}

                {onBccHistory && (
                  <Button
                    variant="ghost"
                    className="w-full min-h-11 justify-start h-auto px-2 py-3"
                    onClick={() => { onBccHistory(); close(); }}
                  >
                    <FileSpreadsheet className="h-5 w-5 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">Lịch sử BCC</span>
                  </Button>
                )}

                {onRejectUnpaid && userRole === 'admin' && (
                  <Button
                    variant="ghost"
                    className="h-auto min-h-11 w-full justify-start px-2 py-3 text-destructive hover:text-destructive"
                    onClick={() => { onRejectUnpaid(); close(); }}
                  >
                    <Trash2 className="h-5 w-5 shrink-0" />
                    <span className="text-sm font-medium">Loại công</span>
                  </Button>
                )}

                {/* Close button at bottom */}
                <div className="h-px bg-border mx-1 my-1" />
                <Button
                  variant="ghost"
                  className="w-full min-h-11 justify-start h-auto px-2 py-3 text-muted-foreground"
                  onClick={close}
                >
                  <X className="h-5 w-5 shrink-0" />
                  <span className="text-sm font-medium">Đóng</span>
                </Button>
              </div>
              </SheetContent>
            </Sheet>
          </div>
        }
      />

      <div className="flex items-center gap-3 border-t border-border/70 bg-muted/25 px-3 py-2.5 sm:px-4">
        <span className="console-kicker hidden shrink-0 sm:inline">
          Kỳ công
        </span>
        <TimesheetMonthSelector
          value={monthValue}
          onChange={onMonthChange}
          className="min-w-0 flex-1 sm:flex-none"
        />
      </div>
    </section>
  );
}
