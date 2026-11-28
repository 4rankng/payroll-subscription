import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { memo, useCallback, useMemo } from "react";
import {
  Star,
  ChevronRight,
  ChevronDown,
  Clock,
  Building2,
  Moon,
  SunMedium,
  TimerReset,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Timesheet } from "@/types/api/timesheet.types";
import {
  TIMESHEET_STRIP_COLORS,
  PAYMENT_STRIP_COLORS,
} from "../utils/timesheetStatusColors";
import {
  groupTimesheetsByEmployee,
  groupEmployeeEntriesByProject,
  formatTimesheetHours,
  getPaytypeDetail,
  EmployeeGroupedTimesheet,
  ProjectTimesheetSection,
} from "../utils/timesheetGrouping";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { UserAvatar } from "@/components/ui/user-avatar";
import { cn } from "@/lib/utils";
import { EmptyState } from "@/components/shared/EmptyState";
import { formatFullCurrency } from "@/utils/formatters";
import { getPayrateUnit } from "./timesheet-pay-unit";

export interface TimesheetGroupedTableProps {
  timesheets: Timesheet[];
  userRole: "admin" | "quanly";
  expandedGroups: Set<string>;
  onToggleGroup: (groupKey: string) => void;
  onRowClick: (timesheet: Timesheet) => void;
}

function getStatusStripBg(status: string, paymentStatus?: string): string {
  if (paymentStatus === "paid" || status === "paid") return "bg-green-500";
  switch (status) {
    case "approved": return "bg-blue-500";
    case "pending_approval": return "bg-amber-400";
    case "rejected": return "bg-red-500";
    default: return "bg-gray-300";
  }
}

export const TimesheetGroupedTable = memo(function TimesheetGroupedTable({
  timesheets,
  userRole,
  expandedGroups,
  onToggleGroup,
  onRowClick,
}: TimesheetGroupedTableProps) {
  const grouped = useMemo(
    () => groupTimesheetsByEmployee(timesheets),
    [timesheets],
  );

  if (grouped.length === 0) {
    return (
      <div className="rounded-2xl border border-border/60 bg-utility-gray-100 p-20 text-center">
        <EmptyState icon={Clock} title="Không có dữ liệu bảng công" />
      </div>
    );
  }

  return (
    // Single scroll container: vertical scroll drives the sticky header and
    // the frozen Nhân viên column; horizontal scroll drives the frozen column.
    <div className="max-h-[calc(100vh-190px)] overflow-auto rounded-xl border border-gray-200 bg-white">
      <Table variant="console" className="table-fixed">
        <colgroup>
          <col style={{ width: "8px" }} />
          <col style={{ width: "170px" }} />
          <col style={{ width: "130px" }} />
          <col style={{ width: "150px" }} />
          <col style={{ width: "90px" }} />
          <col style={{ width: "125px" }} />
          <col style={{ width: "145px" }} className="hidden lg:table-column" />
          <col style={{ width: "36px" }} />
        </colgroup>
        {/* Low-value preview column collapses below lg — tablets get the
            remaining columns without a horizontal rail. */}
        <TableHeader>
          <TableRow className="bg-slate-50 hover:bg-slate-50 border-b border-gray-200">
            <TableHead className="sticky top-0 z-20 bg-slate-50 p-0"><span className="sr-only">Trạng thái</span></TableHead>
            <TableHead className="sticky left-0 top-0 z-30 border-b border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground whitespace-nowrap" style={{ letterSpacing: "0.03em" }}>
              {userRole === "quanly" ? "Nhân viên · Ngày" : "Nhân viên"}
            </TableHead>
            <TableHead className="sticky top-0 z-20 border-b border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground whitespace-nowrap" style={{ letterSpacing: "0.03em" }}>
              {userRole === "quanly" ? "Dự án · Loại ngày" : "Loại ngày"}
            </TableHead>
            <TableHead className="sticky top-0 z-20 border-b border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground whitespace-nowrap" style={{ letterSpacing: "0.03em" }}>
              Ca làm · Trạng thái
            </TableHead>
            <TableHead className="sticky top-0 z-20 border-b border-l border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground text-right whitespace-nowrap" style={{ letterSpacing: "0.03em" }}>
              Giờ
            </TableHead>
            <TableHead className="sticky top-0 z-20 hidden border-b border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground text-right whitespace-nowrap lg:table-cell" style={{ letterSpacing: "0.03em" }}>
              {userRole === "admin" ? "Dự án · Đơn giá" : "Đơn giá"}
            </TableHead>
            <TableHead className="sticky top-0 z-20 border-b border-gray-200 bg-slate-50 px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.03em] leading-[1.35] text-muted-foreground text-right whitespace-nowrap" style={{ letterSpacing: "0.03em" }}>
              Thành tiền
            </TableHead>
            <TableHead className="sticky top-0 z-20 bg-slate-50 border-b border-gray-200 px-2 py-2"><span className="sr-only">Chi tiết</span></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {grouped.map((group, idx) => (
            <GroupRow
              key={group.groupKey}
              group={group}
              idx={idx}
              userRole={userRole}
              isExpanded={expandedGroups.has(group.groupKey)}
              onToggle={() => onToggleGroup(group.groupKey)}
              onRowClick={onRowClick}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
});

// ── Group row (collapsible header + children) ──────────────────────────────

interface GroupRowProps {
  group: EmployeeGroupedTimesheet;
  idx: number;
  userRole: "admin" | "quanly";
  isExpanded: boolean;
  onToggle: () => void;
  onRowClick: (timesheet: Timesheet) => void;
}

const GroupRow = memo(function GroupRow({
  group,
  idx,
  userRole,
  isExpanded,
  onToggle,
  onRowClick,
}: GroupRowProps) {
  const groupStripBg = getStatusStripBg(group.aggregatedStatus);
  const projectSections = useMemo(
    () => groupEmployeeEntriesByProject(group.entries),
    [group.entries],
  );

  return (
    <>
          <TableRow
            onClick={onToggle}
            className={cn(
              "group cursor-pointer transition-colors hover:bg-utility-gray-200/60",
              idx !== 0 && "border-t border-utility-gray-200",
            )}
          >
            {/* [0] status strip — inset from the card edge so the rounded
                border never clips it */}
            <TableCell className="p-0">
              <div className={cn("ml-[3px] h-full min-h-[44px] w-1 rounded-r-full transition-colors", groupStripBg)} />
            </TableCell>
            {/* [1] employee — frozen pane */}
            <TableCell className="sticky left-0 z-[15] bg-white px-3 py-2.5">
              <button
                type="button"
                className="flex min-h-11 w-full min-w-0 items-center gap-2.5 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                aria-expanded={isExpanded}
                aria-label={`${isExpanded ? "Thu gọn" : "Xem"} bảng công của ${group.employeeName || group.employeeCode || "nhân viên"}`}
                onClick={(event) => { event.stopPropagation(); onToggle(); }}
              >
                <UserAvatar email={group.employeeCode} name={group.employeeName} size="sm" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-semibold text-foreground leading-tight truncate">
                    {group.employeeName || "-"}
                  </p>
                  <p className="text-xs text-muted-foreground leading-tight mt-0.5 tabular-nums truncate">
                    {group.employeeCode || "-"}
                  </p>
                </div>
              </button>
            </TableCell>
            {/* [2] role-specific column */}
            <TableCell className="px-3 py-2.5">
              {userRole === "quanly" ? (
                <PartnerGroupColumn group={group} />
              ) : (
                <span className="inline-flex items-center gap-1 px-2.5 h-6 rounded-full bg-utility-gray-100 text-xs font-semibold text-muted-foreground tabular-nums">
                  {group.entries.length} ca
                </span>
              )}
            </TableCell>
            {/* [3] status summary */}
            <TableCell className="px-3 py-2.5">
              <GroupStatusSummary group={group} />
            </TableCell>
            {/* [4] total hours */}
            <TableCell className="num border-l border-border/30 px-3 py-2.5 text-right">
              <span className="typography-body-medium font-bold text-foreground tabular-nums whitespace-nowrap">
                {formatTimesheetHours(group.totalHours)} giờ
              </span>
            </TableCell>
            {/* [5] Rate summary for Partner; project preview for Admin. Rate lives in expanded rows. */}
            <TableCell className="hidden px-3 py-2.5 text-right lg:table-cell">
              {userRole === "admin" ? (
                <AdminGroupProjectColumn group={group} />
              ) : (
                <GroupRateColumn group={group} />
              )}
            </TableCell>
            {/* [6] total amount */}
            <TableCell className="num px-3 py-2.5 text-right">
              <span className="typography-body-medium font-bold text-foreground tabular-nums whitespace-nowrap">
                {formatFullCurrency(group.totalAmount)}
              </span>
            </TableCell>
            {/* [7] chevron */}
            <TableCell className="px-2 py-2.5">
              <div className="flex justify-center text-muted-foreground group-hover:text-foreground transition-colors">
                {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
              </div>
            </TableCell>
          </TableRow>

            {isExpanded && projectSections.map((project, projectIndex) => (
              <ProjectSection
                key={project.projectId}
                project={project}
                showHeader={projectSections.length > 1}
                isFirstProject={projectIndex === 0}
                userRole={userRole}
                onRowClick={onRowClick}
              />
            ))}
    </>
  );
});

// ── Partner-specific group column ──────────────────────────────────────────

/** Parent-row rate summary: single rate → `48.750 ₫/giờ`, several rates →
 *  shift count, none → honest dash (the ⚠ badge flags missing rates). */
const GroupRateColumn = memo(function GroupRateColumn({ group }: { group: EmployeeGroupedTimesheet }) {
  const rates = [...new Set(group.entries.map((e) => e.payrate).filter((r): r is number => !!r && r > 0))];
  if (rates.length === 1) {
    const isFlexible = group.entries.some((e) => e.projectIsFlexible);
    return (
      <span className="whitespace-nowrap text-xs font-medium tabular-nums text-muted-foreground">
        {formatFullCurrency(rates[0], { showSymbol: false })} {getPayrateUnit(isFlexible)}
      </span>
    );
  }
  if (rates.length > 1) {
    return (
      <span className="whitespace-nowrap text-xs font-semibold tabular-nums text-muted-foreground">
        {group.entries.length} ca
      </span>
    );
  }
  return <span className="text-xs text-muted-foreground">—</span>;
});

const PartnerGroupColumn = memo(function PartnerGroupColumn({ group }: { group: EmployeeGroupedTimesheet }) {
  return (
    <div className="flex flex-col gap-1 min-w-0">
      {(() => {
        const projects = [...new Set(group.entries.map(e => e.projectName).filter(Boolean))];
        return (
          <p className="text-xs font-medium text-foreground truncate" title={projects.join(", ")}>
            {projects.length === 1 ? projects[0] : projects.length > 1 ? `${projects[0]} +${projects.length - 1}` : "—"}
          </p>
        );
      })()}
    </div>
  );
});

const AdminGroupProjectColumn = memo(function AdminGroupProjectColumn({ group }: { group: EmployeeGroupedTimesheet }) {
  const projects = [...new Set(group.entries.map((entry) => entry.projectName).filter(Boolean))];
  const projectLabel = projects.length === 1
    ? projects[0]
    : projects.length > 1
      ? `${projects[0]} +${projects.length - 1}`
      : "—";

  return (
    <div
      className="ml-auto flex w-full min-w-0 items-center justify-end gap-1.5 text-left"
      title={projects.join(", ")}
    >
      <Building2 aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-primary" />
      <span className="min-w-0 truncate text-xs font-semibold text-foreground">
        {projectLabel}
      </span>
    </div>
  );
});

const GroupStatusSummary = memo(function GroupStatusSummary({ group }: { group: EmployeeGroupedTimesheet }) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
        {group.statusBreakdown.pending_approval > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-amber-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" />
            {group.statusBreakdown.pending_approval} chờ
          </span>
        )}
        {group.statusBreakdown.approved > 0 && group.aggregatedStatus !== "paid" && (
          <span className="inline-flex items-center gap-1 text-xs text-blue-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-500 shrink-0" />
            {group.statusBreakdown.approved} duyệt
          </span>
        )}
        {group.aggregatedStatus === "paid" && (
          <span className="inline-flex items-center gap-1 text-xs text-green-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 shrink-0" />
            {group.entries.length} đã TT
          </span>
        )}
        {group.statusBreakdown.rejected > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-red-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
            {group.statusBreakdown.rejected} loại
          </span>
        )}
        {group.statusBreakdown.draft > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-slate-600 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-slate-400 shrink-0" />
            {group.statusBreakdown.draft} nháp
          </span>
        )}
        {group.entries.some(e => !e.payrate || e.payrate === 0) && (
          <span className="inline-flex items-center gap-1 text-xs text-red-700 font-semibold whitespace-nowrap">
            ⚠ đơn giá
          </span>
        )}
    </div>
  );
});

const ProjectSection = memo(function ProjectSection({
  project,
  showHeader,
  isFirstProject,
  userRole,
  onRowClick,
}: {
  project: ProjectTimesheetSection;
  showHeader: boolean;
  isFirstProject: boolean;
  userRole: "admin" | "quanly";
  onRowClick: (timesheet: Timesheet) => void;
}) {
  return (
    <>
      {showHeader && (
        <TableRow className={cn("bg-primary/[0.06] hover:bg-primary/[0.06]", isFirstProject && "border-t border-utility-gray-200")}>
          <TableCell className="p-0" />
          <TableCell colSpan={3} className="px-3 py-2.5 pl-6">
            <div className="flex min-w-0 items-center gap-2.5">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Building2 className="h-3.5 w-3.5" />
              </span>
              <div className="min-w-0">
                <p className="truncate text-xs font-bold text-foreground">{project.projectName}</p>
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  {[project.projectCode, project.positions.join(', '), `${project.entries.length} mục`].filter(Boolean).join(' · ')}
                </p>
              </div>
            </div>
          </TableCell>
          <TableCell className="num px-4 py-2.5 text-right text-xs font-bold tabular-nums text-foreground">{formatTimesheetHours(project.totalHours)} giờ</TableCell>
          <TableCell className="hidden px-4 py-2.5 text-right text-xs text-muted-foreground lg:table-cell">—</TableCell>
          <TableCell className="num px-4 py-2.5 text-right text-xs font-bold tabular-nums text-foreground">
            {formatFullCurrency(project.totalAmount)}
          </TableCell>
          <TableCell className="p-0" />
        </TableRow>
      )}
      {project.entries.map((entry, entryIndex) => (
        <EntryRow
          key={entry.id}
          entry={entry}
          entryIdx={entryIndex}
          showDate={entryIndex === 0 || project.entries[entryIndex - 1]?.date !== entry.date}
          startsAfterProjectHeader={showHeader && entryIndex === 0}
          userRole={userRole}
          onClick={() => onRowClick(entry)}
        />
      ))}
    </>
  );
});

// ── Entry row (expanded child) ─────────────────────────────────────────────

interface EntryRowProps {
  entry: Timesheet;
  entryIdx: number;
  showDate: boolean;
  startsAfterProjectHeader: boolean;
  userRole: "admin" | "quanly";
  onClick: () => void;
}

const EntryRow = memo(function EntryRow({ entry, entryIdx, showDate, startsAfterProjectHeader, userRole, onClick }: EntryRowProps) {
  const navigate = useNavigate();
  const entryDate = new Date(entry.date);
  const hoursWorked = entry.hours_worked || 0;
  const dayTypeLabel = entry.day_type || "—";
  const normalizedDayType = dayTypeLabel.toLocaleLowerCase("vi-VN");
  const shiftLabel = entry.hour_type || getPaytypeDetail(entry.paytype) || "—";
  const normalizedShift = shiftLabel.toLocaleLowerCase("vi-VN");
  const isOvertime = normalizedShift.includes("tăng ca");
  const isNightShift = normalizedShift.includes("đêm");
  const isShortShift = hoursWorked > 0 && hoursWorked < 8;

  return (
    <TableRow
      onClick={onClick}
      className={cn(
        "cursor-pointer transition-colors hover:bg-utility-gray-200",
        "bg-utility-gray-200/60",
        entryIdx === 0 && !startsAfterProjectHeader ? "border-t border-utility-gray-300" : "border-t border-utility-gray-300/60",
      )}
    >
      <TableCell className="p-0 relative">
        {(!entry.payrate || entry.payrate === 0) && <span className="absolute left-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />}
      </TableCell>
      <TableCell className="sticky left-0 z-[15] bg-white py-2 px-3 pl-6">
        <div className="flex min-h-9 items-center gap-2">
          {showDate ? (
            <div className="flex items-center gap-2">
              <span className="inline-flex h-7 min-w-9 items-center justify-center rounded-md bg-utility-gray-50 px-1.5 text-xs font-bold uppercase text-slate-600 ring-1 ring-inset ring-utility-gray-300">
                {format(entryDate, "EEE", { locale: vi }).replace("Th ", "T")}
              </span>
              <p className="text-xs font-bold tabular-nums text-foreground">{format(entryDate, "dd/MM/yyyy")}</p>
            </div>
          ) : (
            <span className="ml-4 flex items-center gap-1.5 text-xs font-medium text-muted-foreground" aria-label="Cùng ngày">
              <span className="h-4 w-px bg-border" />
              <span className="h-1.5 w-1.5 rounded-full bg-border" />
            </span>
          )}
          {entry.force_payroll && <Star className="h-3 w-3 text-yellow-800 fill-yellow-500 shrink-0" />}
        </div>
      </TableCell>
      <TableCell className="py-2 px-3">
        <span
          className={cn(
            "inline-flex h-7 items-center rounded-md px-2.5 text-xs font-semibold capitalize ring-1 ring-inset",
            normalizedDayType.includes("lễ")
              ? "bg-utility-gray-50 text-rose-700 ring-rose-200"
              : normalizedDayType.includes("nghỉ")
                ? "bg-utility-gray-50 text-teal-700 ring-teal-200"
                : "bg-utility-gray-50 text-slate-700 ring-utility-gray-300",
          )}
        >
          {dayTypeLabel}
        </span>
      </TableCell>
      <TableCell className="py-2 px-3">
        <span
          className={cn(
            "inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-semibold capitalize",
            isOvertime
              ? "bg-utility-gray-50 text-amber-700 ring-1 ring-inset ring-amber-200"
              : isNightShift
                ? "bg-utility-gray-50 text-teal-700 ring-1 ring-inset ring-teal-200"
                : "bg-utility-gray-50 text-sky-700 ring-1 ring-inset ring-sky-200",
          )}
        >
          {isOvertime ? (
            <TimerReset className="h-3.5 w-3.5" />
          ) : isNightShift ? (
            <Moon className="h-3.5 w-3.5" />
          ) : (
            <SunMedium className="h-3.5 w-3.5" />
          )}
          {shiftLabel}
        </span>
      </TableCell>
      <TableCell className="num border-l border-border/30 py-2 px-3 text-right">
        <span
          className={cn(
            "text-xs font-bold tabular-nums whitespace-nowrap",
            isShortShift ? "text-amber-700" : "text-foreground",
          )}
        >
          {formatTimesheetHours(hoursWorked)} giờ
        </span>
      </TableCell>
      <TableCell className="num hidden py-2 px-3 text-right lg:table-cell">
        <p className={cn(
          "text-xs tabular-nums",
          (!entry.payrate || entry.payrate === 0) ? "text-red-700 font-semibold" : "font-medium text-muted-foreground",
        )}>
          {(!entry.payrate || entry.payrate === 0) && userRole === "quanly" ? (
            <span
              className="cursor-pointer hover:underline"
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/partner/projects/${entry.project_id}/payrates/new/edit`);
              }}
            >
              ⚠ chưa có
            </span>
          ) : `${formatFullCurrency(entry.payrate ?? 0, { showSymbol: false })} ${getPayrateUnit(!!entry.projectIsFlexible)}`}
        </p>
      </TableCell>
      <TableCell className="num py-2 px-3 text-right">
        <div className="flex items-center justify-end gap-1">
          <span className="text-xs font-bold text-foreground tabular-nums whitespace-nowrap">
            {formatFullCurrency(entry.amount || 0)}
          </span>
        </div>
      </TableCell>
      <TableCell className="py-2 px-2">
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground mx-auto" />
      </TableCell>
    </TableRow>
  );
});
