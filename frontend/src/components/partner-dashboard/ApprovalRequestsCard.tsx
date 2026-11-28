import { Link } from 'react-router-dom';
import { CalendarCheck01 } from '@untitledui/icons';
import { useTimesheetSummary } from '@/hooks/api/useTimesheets';
import { cn } from '@/lib/utils';

/**
 * PRD §5.2: the Quản lý dashboard's approval-requests card — pending-approval
 * timesheet count with a deep link into Bảng công. Renders nothing while
 * loading; collapses to a calm zero state when the queue is empty.
 */
export function ApprovalRequestsCard({ className }: { className?: string }) {
  const summary = useTimesheetSummary();
  const pending = summary.data?.pendingApproval ?? 0;
  const isLoading = summary.isLoading;

  if (isLoading) {
    return (
      <div
        className={cn('console-panel flex h-[64px] animate-pulse items-center rounded-xl bg-muted/60', className)}
        aria-busy="true"
        aria-label="Đang tải yêu cầu phê duyệt"
      />
    );
  }

  const hasPending = pending > 0;

  return (
    <div
      className={cn(
        'console-panel flex min-h-[64px] flex-wrap items-center justify-between gap-2 rounded-xl px-4 py-2.5',
        hasPending ? 'border-warning/40 bg-warning/5' : 'bg-card',
        className,
      )}
      data-testid="approval-requests-card"
    >
      <div className="flex min-w-0 items-center gap-2.5">
        <CalendarCheck01
          className="size-5 shrink-0 text-muted-foreground"
          aria-hidden="true"
        />
        <p className="min-w-0 text-[13px] leading-snug text-muted-foreground">
          <span
            className={cn(
              'font-semibold tabular-nums',
              hasPending ? 'text-foreground' : 'text-muted-foreground',
            )}
          >
            {pending}
          </span>{' '}
          bảng công chờ duyệt
        </p>
      </div>
      <Link
        to="/timesheet"
        className={cn(
          'inline-flex h-9 shrink-0 items-center rounded-lg border border-border bg-card px-3 text-xs font-semibold',
          'text-foreground transition-colors hover:bg-muted focus-visible:outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
        )}
      >
        {hasPending ? 'Xem bảng công' : 'Mở bảng công'}
      </Link>
    </div>
  );
}
