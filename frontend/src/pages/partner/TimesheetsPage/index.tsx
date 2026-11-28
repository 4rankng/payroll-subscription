import { useState, useEffect, useCallback, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, Clock3, FileUp, Plus, X } from "lucide-react";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
import { TimesheetFilters } from "@/components/timesheet/TimesheetFilters";
import { TimesheetListTable } from "@/components/timesheet/TimesheetListTable";
import { TimesheetMobileList } from "@/components/timesheet/TimesheetMobileList";
import { TimesheetProvider } from "@/components/timesheet/TimesheetContext";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Textarea } from "@/components/ui/textarea";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useEmployeeModals, useTimesheetModals } from "@/hooks/useModalNavigation";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import { Clock, Download01 } from "@untitledui/icons";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { TransferUploadPanel } from "@/components/billing/TransferUploadPanel";
import { billingKeys } from "@/hooks/api/useBilling";
import { billingService } from "@/services/api/billing.service";
import { useAuth } from "@/contexts";
import { cn } from "@/lib/utils";
import { formatFullCurrency } from "@/utils/formatters";
import { formatTimesheetHours } from "@/components/timesheet/utils/timesheetGrouping";
import type { Employee } from "@/types/api/employee.types";
import type { Timesheet } from "@/types/api/timesheet.types";

/* ------------------------------------------------------------------ */
/*  TimesheetStatsRow — single 44px horizontal metric strip            */
/* ------------------------------------------------------------------ */

interface TimesheetStatsRowProps {
  isLoading: boolean;
  totalHours: number;
  totalPay: number;
  pendingCount: number;
  approvedCount: number;
  statusFilter: string;
  onToggleStatus: (status: string) => void;
}

function TimesheetStatsRow({
  isLoading,
  totalHours,
  totalPay,
  pendingCount,
  approvedCount,
  statusFilter,
  onToggleStatus,
}: TimesheetStatsRowProps) {
  if (isLoading) {
    return (
      <div className="flex h-11 items-center gap-4 rounded-xl border border-input bg-card px-3 shadow-xs">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <div className="h-2.5 w-14 bg-muted rounded animate-pulse" />
            <div className="h-4 w-12 bg-muted rounded animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  const cells = [
    {
      key: "total_hours",
      label: "Tổng giờ",
      value: `${formatTimesheetHours(totalHours)} giờ`,
      active: false,
      onClick: undefined as (() => void) | undefined,
      dot: "",
    },
    {
      key: "total_pay",
      label: "Tổng chi trả",
      value: formatFullCurrency(totalPay),
      active: false,
      onClick: undefined,
      dot: "",
    },
    {
      key: "pending_approval",
      label: "Chờ duyệt",
      value: pendingCount.toLocaleString("vi-VN"),
      active: statusFilter === "pending_approval",
      onClick: pendingCount > 0
        ? () => onToggleStatus(statusFilter === "pending_approval" ? "all" : "pending_approval")
        : undefined,
      dot: pendingCount > 0 ? "bg-amber-400" : "bg-muted-foreground/40",
    },
    {
      key: "approved",
      label: "Đã duyệt",
      value: approvedCount.toLocaleString("vi-VN"),
      active: statusFilter === "approved",
      onClick: approvedCount > 0
        ? () => onToggleStatus(statusFilter === "approved" ? "all" : "approved")
        : undefined,
      dot: approvedCount > 0 ? "bg-green-500" : "bg-muted-foreground/40",
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-input bg-border/40 shadow-xs sm:flex sm:h-11 sm:gap-0 sm:bg-card">
      {cells.map(({ key, label, value, active, onClick, dot }) => (
        <div
          key={key}
          onClick={onClick}
          role={onClick ? "button" : undefined}
          tabIndex={onClick ? 0 : undefined}
          onKeyDown={onClick ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onClick(); } } : undefined}
          aria-pressed={onClick ? active : undefined}
          className={cn(
            "flex min-h-11 min-w-0 items-center justify-center gap-1.5 bg-card px-2.5 py-1.5 outline-brand transition-colors sm:h-full sm:min-h-full sm:flex-1 sm:py-0",
            onClick && "cursor-pointer hover:bg-muted/40 focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
            active && "bg-primary/5",
          )}
        >
          <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", dot)} aria-hidden="true" />
          <span className="text-[11px] font-semibold leading-[1.35] tracking-[0.03em] text-muted-foreground whitespace-nowrap">
            {label}
          </span>
          <span className={cn(
            "truncate text-sm font-bold tabular-nums leading-none",
            active ? "text-primary" : "text-foreground",
          )}>
            {value}
          </span>
        </div>
      ))}
    </div>
  );
}

export default function TimesheetsPage() {
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [bulkApproveOpen, setBulkApproveOpen] = useState(false);
  const [bulkRejectOpen, setBulkRejectOpen] = useState(false);
  const [bulkRejectReason, setBulkRejectReason] = useState("");
  const [searchParams, setSearchParams] = useSearchParams();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openEmployeeDetails } = useEmployeeModals();
  const isMobile = useIsMobile();

  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const urlEmployeeId = searchParams.get("employee");
  const urlProjectId = searchParams.get("project");

  const timesheetManagement = useTimesheetManagement({
    userRole: "quanly",
    useYearToDate: false,
  });

  const currentMonthStr = useMemo(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  }, []);

  useEffect(() => {
    if (timesheetManagement.selectedMonth === "all") {
      timesheetManagement.setSelectedMonth(currentMonthStr);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (urlProjectId && urlProjectId !== timesheetManagement.selectedProject) {
      timesheetManagement.setSelectedProject(urlProjectId);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const displayMonth = timesheetManagement.selectedMonth === "all"
    ? currentMonthStr
    : timesheetManagement.selectedMonth;

  // Bank-transfer workflow (ketoan only): export the bulk-transfer file for
  // the selected month and upload the bank's result workbook for reconciliation.
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const isKetoan = user?.role === "ketoan";
  const isQuanly = user?.role === "quanly";

  const bankTransferExport = useMutation({
    mutationFn: async () => {
      const blob = await billingService.exportBankTransferHistories({ month: displayMonth });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `chuyen-khoan-${displayMonth}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    },
  });

  const handleTransferUploaded = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: billingKeys.all });
  }, [queryClient]);

  const statsFilters = useMemo(() => ({
    project_id: timesheetManagement.selectedProject !== "all" ? parseInt(timesheetManagement.selectedProject) : undefined,
    fromDate: displayMonth ? `${displayMonth}-01` : undefined,
    toDate: displayMonth ? (() => {
      const lastDay = new Date(
        parseInt(displayMonth.split("-")[0]),
        parseInt(displayMonth.split("-")[1]),
        0,
      ).getDate();
      return `${displayMonth}-${lastDay.toString().padStart(2, "0")}`;
    })() : undefined,
  }), [timesheetManagement.selectedProject, displayMonth]);

  const timesheetStats = useTimesheetStatsConfig(statsFilters);

  useEffect(() => {
    if (
      urlEmployeeId &&
      urlEmployeeId !== timesheetManagement.selectedEmployee
    ) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);
    const isSpecific =
      timesheetManagement.selectedEmployee &&
      timesheetManagement.selectedEmployee !== "all" &&
      timesheetManagement.selectedEmployee.trim() !== "";
    if (isSpecific) {
      newParams.set("employee", timesheetManagement.selectedEmployee);
    } else {
      newParams.delete("employee");
      newParams.delete("view");
    }
    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [timesheetManagement.selectedEmployee, searchParams, setSearchParams]);

  const handleDelete = useCallback(
    async (timesheet: Timesheet) => {
      await timesheetManagement.handleDelete(timesheet);
    },
    [timesheetManagement],
  );

  // Hours/pay strip values are summed over the timesheet rows currently
  // loaded for the active filters (the grouped endpoint pages by employee).
  const loadedTimesheets = timesheetManagement.timesheets;
  const totalHours = useMemo(
    () => loadedTimesheets.reduce((sum, t) => sum + (t.hours_worked || 0), 0),
    [loadedTimesheets],
  );
  const totalPay = useMemo(
    () => loadedTimesheets.reduce((sum, t) => sum + (t.amount || 0), 0),
    [loadedTimesheets],
  );

  const pendingTimesheets = useMemo(
    () => loadedTimesheets.filter((t) => t.status === "pending_approval"),
    [loadedTimesheets],
  );

  const handleBulkApprove = useCallback(async () => {
    await timesheetManagement.handleBulkApprove(pendingTimesheets.map((t) => t.id));
    setBulkApproveOpen(false);
  }, [timesheetManagement, pendingTimesheets]);

  const handleBulkReject = useCallback(async () => {
    const reason = bulkRejectReason.trim();
    if (!reason) return;
    await timesheetManagement.handleBulkReject(
      pendingTimesheets.map((t) => t.id),
      reason,
    );
    setBulkRejectReason("");
    setBulkRejectOpen(false);
  }, [timesheetManagement, pendingTimesheets, bulkRejectReason]);

  const summaryStats = useMemo(() => {
    const s = timesheetStats.summary;
    if (!s) return null;
    return {
      pendingCount: s.pendingApproval,
      approvedCount: s.approvedEntries,
    };
  }, [timesheetStats.summary]);

  if (
    timesheetManagement.isLoading &&
    timesheetManagement.timesheets.length === 0
  ) {
    return (
      <div className="min-h-full bg-[radial-gradient(circle_at_100%_0%,rgba(8,120,62,0.12),transparent_29rem)] p-3 sm:p-4">
        <div className="mx-auto max-w-[1480px] space-y-2">
          <Skeleton className="h-11 w-full rounded-xl" />
          <Skeleton className="h-11 w-full rounded-xl" />
          <Skeleton className="h-10 w-full rounded-xl" />
          <Skeleton className="h-[420px] w-full rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-[radial-gradient(circle_at_100%_0%,rgba(8,120,62,0.12),transparent_29rem)] px-3 py-2 sm:px-4 sm:py-3">
      <div className="mx-auto max-w-[1480px] space-y-2">

        {/* Header — single 44px chrome row (title + record count) */}
        <div className="flex h-11 items-center justify-between gap-2 rounded-xl border border-input bg-card px-3 shadow-xs motion-safe:animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:both]">
          <div className="flex min-w-0 items-center gap-2">
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-inset ring-primary/15">
              <Clock className="h-4 w-4" aria-hidden="true" />
            </span>
            <h1 className="text-lg font-bold tracking-tight text-fg-primary whitespace-nowrap">
              Bảng công
            </h1>
            <span className="hidden min-w-0 truncate text-xs text-muted-foreground lg:inline">
              Theo dõi bảng công, yêu cầu sửa và lịch sử chi trả theo tháng
            </span>
          </div>
          <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[11px] font-semibold tabular-nums text-muted-foreground">
            {timesheetManagement.timesheets.length} bản ghi
          </span>
        </div>

        {/* Stats — one dense horizontal strip: Tổng giờ · Tổng chi trả · Chờ duyệt · Đã duyệt */}
        {summaryStats && (
          <div className="motion-safe:animate-fade-in-up [animation-delay:80ms] [animation-fill-mode:both]">
            <TimesheetStatsRow
              isLoading={timesheetStats.isLoading}
              totalHours={totalHours}
              totalPay={totalPay}
              pendingCount={summaryStats.pendingCount}
              approvedCount={summaryStats.approvedCount}
              statusFilter={timesheetManagement.statusFilter}
              onToggleStatus={timesheetManagement.setStatusFilter}
            />
          </div>
        )}

        <TimesheetProvider
          management={timesheetManagement}
          onEdit={timesheetManagement.handleEdit}
          onDelete={handleDelete}
          userRole="quanly"
          onExportExcel={() => timesheetManagement.handleExportExcel()}
        >
          {/* Toolbar — one 40px row on desktop (nowrap + horizontal scroll
              when actions overflow; TIME-01 chrome target): month + filters
              left, actions right. Mobile keeps wrapping. */}
          <div className="flex flex-wrap items-center justify-between gap-2 overflow-x-auto rounded-xl border border-input bg-card px-2 py-1 shadow-xs motion-safe:animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:both] sm:flex-nowrap">
            <div className="flex min-w-0 flex-wrap items-center gap-1.5 sm:shrink-0 sm:flex-nowrap">
              <TimesheetMonthSelector
                value={timesheetManagement.selectedMonth}
                onChange={timesheetManagement.setSelectedMonth}
              />
              <TimesheetFilters />
            </div>
            <div className="flex flex-wrap items-center gap-1.5 sm:flex-nowrap">
              {isQuanly && (
                <>
                  <Button
                    type="button"
                    size="sm"
                    onClick={() => openTimesheetEntry()}
                    className="h-8 gap-1.5 px-2.5 text-xs"
                  >
                    <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                    Nhập công
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setBccUploadOpen(true)}
                    className="h-8 gap-1.5 px-2.5 text-xs"
                  >
                    <FileUp className="h-3.5 w-3.5" aria-hidden="true" />
                    Tải lên BCC
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setBccHistoryOpen(true)}
                    className="h-8 gap-1.5 px-2.5 text-xs text-muted-foreground"
                  >
                    <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
                    Lịch sử BCC
                  </Button>
                </>
              )}
              {/* Duyệt/loại công: quanly and ketoan (backend casbin grants
                  single + pending-selection approval to both roles). */}
              {(isQuanly || isKetoan) && pendingTimesheets.length > 0 && (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setBulkApproveOpen(true)}
                    disabled={timesheetManagement.isLoading}
                    className="h-8 gap-1.5 border-green-600/40 px-2.5 text-xs text-green-700 hover:bg-green-50"
                  >
                    <Check className="h-3.5 w-3.5" aria-hidden="true" />
                    Duyệt {pendingTimesheets.length}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => { setBulkRejectReason(""); setBulkRejectOpen(true); }}
                    disabled={timesheetManagement.isLoading}
                    className="h-8 gap-1.5 border-red-600/40 px-2.5 text-xs text-red-700 hover:bg-red-50"
                  >
                    <X className="h-3.5 w-3.5" aria-hidden="true" />
                    Loại {pendingTimesheets.length}
                  </Button>
                </>
              )}
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void timesheetManagement.handleExportExcel()}
                disabled={timesheetManagement.isLoading}
                className="h-8 gap-1.5 px-2.5 text-xs"
              >
                <Download01 className="h-3.5 w-3.5" aria-hidden="true" />
                Xuất Excel
              </Button>
            </div>
          </div>

          {/* Grid — sticky header + frozen name column live in TimesheetGroupedTable */}
          <div className="motion-safe:animate-fade-in-up [animation-delay:130ms] [animation-fill-mode:both]">
            {isMobile ? <TimesheetMobileList /> : <TimesheetListTable />}
          </div>
        </TimesheetProvider>

        {/* Chuyển khoản ngân hàng — ketoan only; workflow panel sits below the grid */}
        {isKetoan && (
          <div className="rounded-xl border border-input bg-card shadow-xs motion-safe:animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:both]">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/55 bg-muted/25 px-3 py-2 sm:px-4">
              <div>
                <p className="text-[12px] font-bold text-foreground">Chuyển khoản ngân hàng</p>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Xuất file chuyển khoản hàng loạt tháng {displayMonth} và tải lên kết quả đối soát từ ngân hàng
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void bankTransferExport.mutateAsync()}
                disabled={bankTransferExport.isPending}
                className="h-8 gap-1.5 text-xs"
              >
                <Download01 className="h-3.5 w-3.5" aria-hidden="true" />
                {bankTransferExport.isPending ? "Đang xuất..." : "Xuất file chuyển khoản"}
              </Button>
            </div>
            <div className="px-3 py-3 sm:px-4">
              <TransferUploadPanel workMonth={displayMonth} onUploaded={handleTransferUploaded} />
            </div>
          </div>
        )}

      </div>

      <BCCUploadModal
        open={bccUploadOpen}
        onClose={() => setBccUploadOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== "all"
            ? parseInt(timesheetManagement.selectedProject, 10)
            : 0
        }
        projects={timesheetManagement.projects}
      />

      <UploadHistorySheet
        open={bccHistoryOpen}
        onClose={() => setBccHistoryOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== "all"
            ? parseInt(timesheetManagement.selectedProject, 10)
            : undefined
        }
        projects={timesheetManagement.projects}
      />

      <ConfirmDialog
        open={bulkApproveOpen}
        onOpenChange={setBulkApproveOpen}
        title="Duyệt bảng công hàng loạt"
        description={`Duyệt tất cả ${pendingTimesheets.length} bảng công đang chờ trong phạm vi bộ lọc hiện tại?`}
        confirmText="Duyệt"
        onConfirm={handleBulkApprove}
        loading={timesheetManagement.isLoading}
      />

      <ConfirmDialog
        open={bulkRejectOpen}
        onOpenChange={setBulkRejectOpen}
        title="Loại bảng công hàng loạt"
        description={
          <div className="space-y-3">
            <p>
              Loại tất cả {pendingTimesheets.length} bảng công đang chờ trong phạm vi bộ lọc hiện tại?
            </p>
            <Textarea
              value={bulkRejectReason}
              onChange={(e) => setBulkRejectReason(e.target.value)}
              placeholder="Lý do loại (bắt buộc)"
              rows={3}
              className="text-sm"
            />
          </div>
        }
        confirmText="Loại"
        confirmVariant="destructive"
        disabled={!bulkRejectReason.trim()}
        onConfirm={handleBulkReject}
        loading={timesheetManagement.isLoading}
      />
    </div>
  );
}
