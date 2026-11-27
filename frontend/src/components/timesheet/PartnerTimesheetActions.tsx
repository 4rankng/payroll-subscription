import {
  Clock3,
  Files,
  FileUp,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/contexts";

interface PartnerTimesheetActionsProps {
  onAddTimesheet: () => void;
  onOpenBccHistory: () => void;
  onUploadBcc: () => void;
}

const supportingActions = [
  {
    label: "Lịch sử BCC",
    description: "Kiểm tra các lần tải lên",
    icon: Clock3,
    handler: "onOpenBccHistory",
  },
  {
    label: "Tải lên BCC",
    description: "Nhập dữ liệu từ Excel",
    icon: FileUp,
    handler: "onUploadBcc",
  },
] as const;

export function PartnerTimesheetActions({
  onAddTimesheet,
  onOpenBccHistory,
  onUploadBcc,
}: PartnerTimesheetActionsProps) {
  // D28: timesheet operations (manual entry, BCC upload/history) are quanly-only —
  // ketoan/giamdoc hold read-only timesheet access, so hide the whole panel
  // instead of serving 403 toasts on click.
  const { user } = useAuth();
  if (user?.role !== "quanly") return null;

  const handlers = {
    onOpenBccHistory,
    onUploadBcc,
  };

  return (
    <section
      aria-labelledby="partner-timesheet-actions-title"
      className="overflow-hidden rounded-2xl border border-border/60 bg-card"
    >
      <div className="flex flex-col gap-3 border-b border-border/60 bg-muted/20 p-3 sm:flex-row sm:items-center sm:justify-between sm:px-4">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Files className="h-5 w-5" aria-hidden="true" />
          </span>
          <div className="min-w-0">
            <h2
              id="partner-timesheet-actions-title"
              className="text-sm font-bold text-foreground"
            >
              Tác vụ bảng công
            </h2>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Quản lý, xuất và đối soát dữ liệu
            </p>
          </div>
        </div>

        <Button
          type="button"
          onClick={onAddTimesheet}
          className="w-full gap-2 sm:w-auto sm:min-w-[132px]"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          Nhập công
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-1.5 p-2 sm:grid-cols-2">
        {supportingActions.map((action) => {
          const Icon = action.icon;

          return (
            <Button
              key={action.label}
              type="button"
              variant="ghost"
              onClick={handlers[action.handler]}
              className="h-auto min-h-[68px] justify-start gap-3 whitespace-normal rounded-xl px-3 py-2.5 text-left"
            >
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-card text-muted-foreground">
                <Icon className="h-4 w-4" aria-hidden="true" />
              </span>
              <span className="min-w-0">
                <span className="block text-[13px] font-semibold leading-5 text-foreground">
                  {action.label}
                </span>
                <span className="block text-xs font-normal leading-4 text-muted-foreground">
                  {action.description}
                </span>
              </span>
            </Button>
          );
        })}
      </div>
    </section>
  );
}
