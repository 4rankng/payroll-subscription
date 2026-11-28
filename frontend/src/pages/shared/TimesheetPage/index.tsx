import { useState, useEffect, useCallback, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, FileUp, Plus, X } from "lucide-react";
import { Clock, DotsVertical, Download01 } from "@untitledui/icons";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
import { TimesheetFilters } from "@/components/timesheet/TimesheetFilters";
import { TimesheetListTable } from "@/components/timesheet/TimesheetListTable";
import { TimesheetMobileList } from "@/components/timesheet/TimesheetMobileList";
import { TimesheetEmptyState } from "@/components/timesheet/TimesheetEmptyState";
import { TimesheetProvider } from "@/components/timesheet/TimesheetContext";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Textarea } from "@/components/ui/textarea";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useTimesheetModals, useModalNavigation } from "@/hooks/useModalNavigation";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import { useApproveAllTimesheets, useResetAllTimesheets, useTimesheetSummary } from "@/hooks/api/useTimesheets";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { RejectUnpaidTimesheetsDialog } from "@/components/timesheet/RejectUnpaidTimesheetsDialog";
import { TransferUploadPanel } from "@/components/billing/TransferUploadPanel";
import { billingKeys } from "@/hooks/api/useBilling";
import { billingService } from "@/services/api/billing.service";
import { useAuth } from "@/contexts";
import { cn } from "@/lib/utils";
import { formatFullCurrency } from "@/utils/formatters";
import { formatTimesheetHours } from "@/components/timesheet/utils/timesheetGrouping";
import { parseTimesheetStatusFilter } from "@/utils/timesheetFilterHelpers";
import type { Timesheet } from "@/types/api/timesheet.types";

/* ------------------------------------------------------------------ */
/*  TimesheetStatsRow — inline KPI strip inside the filter card        */
/*  (PAYR-134: left-aligned, divide-x, clickable pending/approved)     */
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
      <div className="flex items-center gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-1.5">
            <div className="h-2.5 w-12 bg-muted rounded animate-pulse" />
            <div className="h-3.5 w-10 bg-muted rounded animate-pulse" />
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
    <div className="flex flex-wrap items-center gap-y-1 text-xs leading-[1.45] tabular-nums divide-y divide-border/60 max-sm:w-full sm:divide-y-0 sm:divide-x">
      {cells.map(({ key, label, value, active, onClick, dot }) => {
        const content = (
          <>
            {dot && <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", dot)} aria-hidden="true" />}
            <span className="text-muted-foreground">{label}</span>
            <span className={cn("font-semibold", active ? "text-primary" : "text-foreground")}>{value}</span>
          </>
        );
        const className = cn(
          "flex min-w-0 items-center gap-1.5 whitespace-nowrap px-2.5 first:pl-0 last:pr-0 outline-brand transition-colors",
          onClick && "cursor-pointer hover:text-primary focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
          active && "text-primary",
        );
        if (!onClick) {
          return <div key={key} className={className}>{content}</div>;
        }
        return (
          <button
            key={key}
            type="button"
            onClick={onClick}
            aria-pressed={active}
            className={className}
          >
            {content}
          </button>
        );
      })}
    </div>
  );
}

export default function TimesheetPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const isQuanly = user?.role === "quanly";
  const isKetoan = user?.role === "ketoan";
  // Provider/hook role semantics preserved from the pre-merge pages
  // (PAYR-134): admin → 'admin'; every partner role → 'quanly'.
  const userRole: "admin" | "quanly" = isAdmin ? "admin" : "quanly";

  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [bulkApproveOpen, setBulkApproveOpen] = useState(false);
  const [bulkRejectOpen, setBulkRejectOpen] = useState(false);
  const [bulkRejectReason, setBulkRejectReason] = useState("");
  // Admin-only global flows (Duyệt hết / Bỏ duyệt / Loại công)
  const [globalApproveOpen, setGlobalApproveOpen] = useState(false);
  const [resetAllDialogOpen, setResetAllDialogOpen] = useState(false);
  const [rejectUnpaidDialogOpen, setRejectUnpaidDialogOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openModal } = useModalNavigation();
  const isMobile = useIsMobile();

  const urlEmployeeId = searchParams.get("employee");
  const urlProjectId = searchParams.get("project");
  const urlStatus = parseTimesheetStatusFilter(searchParams.get("status"));

  const timesheetManagement = useTimesheetManagement({
    userRole,
    useYearToDate: false,
  });

  const handleAddTimesheet = useCallback(() => {
    if (isAdmin) {
      openModal(MODAL_IDS.TIMESHEET_ENTRY);
    } else {
      openTimesheetEntry();
    }
  }, [isAdmin, openModal, openTimesheetEntry]);

  const handleEditTimesheet = useCallback(
    (timesheet: Timesheet) => {
      if (isAdmin) {
        openModal(MODAL_IDS.TIMESHEET_ENTRY, {
          projectId: timesheet.project_id?.toString(),
          employeeId: timesheet.employee_id?.toString(),
          entryId: timesheet.id?.toString(),
        });
      } else {
        timesheetManagement.handleEdit(timesheet);
      }
    },
    [isAdmin, openModal, timesheetManagement],
  );

  // Admin global flows show the honest, unfiltered scope: backend
  // approveAll()/resetAll() ignore filters, so refresh the global summary
  // before opening the confirm dialogs.
  const { data: globalSummary, refetch: refetchGlobalSummary } = useTimesheetSummary({});
  const approveAllMutation = useApproveAllTimesheets();
  const resetAllMutation = useResetAllTimesheets();

  const handleGlobalApprove = useCallback(() => {
    refetchGlobalSummary();
    setGlobalApproveOpen(true);
  }, [refetchGlobalSummary]);

  const handleConfirmGlobalApprove = async () => {
    try {
      await approveAllMutation.mutateAsync();
      setGlobalApproveOpen(false);
    } catch {
      // Error is handled by the mutation
    }
  };

  const handleResetAll = useCallback(() => {
    refetchGlobalSummary();
    setResetAllDialogOpen(true);
  }, [refetchGlobalSummary]);

  const handleConfirmResetAll = async () => {
    try {
      await resetAllMutation.mutateAsync();
      setResetAllDialogOpen(false);
    } catch {
      // Error is handled by the mutation
    }
  };

  // Initialize employee/status/project selection from URL on mount only
  useEffect(() => {
    if (urlEmployeeId && urlEmployeeId !== timesheetManagement.selectedEmployee) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    if (urlStatus && urlStatus !== timesheetManagement.statusFilter) {
      timesheetManagement.setStatusFilter(urlStatus);
    }
    if (urlProjectId && urlProjectId !== timesheetManagement.selectedProject) {
      timesheetManagement.setSelectedProject(urlProjectId);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Partner-page semantics (PAYR-134): the ketoan bank-transfer export and
  // the KPI strip stay month-scoped — "Tất cả" displays the calendar month.
  const currentMonthStr = useMemo(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  }, []);
  const displayMonth = timesheetManagement.selectedMonth === "all"
    ? currentMonthStr
    : timesheetManagement.selectedMonth;

  // Bank-transfer workflow (ketoan only): export the bulk-transfer file for
  // the selected month and upload the bank's result workbook for reconciliation.
  const queryClient = useQueryClient();

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
  const canBulkReview = isQuanly || isKetoan;

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

  const hasActiveFilter =
    (timesheetManagement.searchTerm?.trim() || '') !== '' ||
    (timesheetManagement.selectedProject !== 'all' && timesheetManagement.selectedProject !== '') ||
    (timesheetManagement.selectedEmployee !== 'all' && timesheetManagement.selectedEmployee !== '') ||
    timesheetManagement.statusFilter !== 'all';

  const handleClearFilters = () => {
    timesheetManagement.setSearchTerm('');
    timesheetManagement.setSelectedProject('all');
    timesheetManagement.setSelectedEmployee('all');
    timesheetManagement.setStatusFilter('all');
  };

  if (
    timesheetManagement.isLoading &&
    timesheetManagement.timesheets.length === 0
  ) {
    return (
      <div className="min-h-full bg-[radial-gradient(circle_at_100%_0%,rgba(8,120,62,0.12),transparent_29rem)] p-3 sm:p-4">
        <div className="mx-auto max-w-[1480px] space-y-2">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-[420px] w-full rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-[radial-gradient(circle_at_100%_0%,rgba(8,120,62,0.12),transparent_29rem)] px-3 py-2 sm:px-4 sm:py-3">
      <div className="mx-auto max-w-[1480px] space-y-2">

        {/* Tier 1 — unboxed page header: title left, actions right (PAYR-134) */}
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border pb-3 motion-safe:animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:both]">
          <div className="flex min-w-0 items-baseline gap-2.5">
            <h1 className="whitespace-nowrap text-lg font-bold tracking-tight text-fg-primary">
              Bảng công
            </h1>
            <span className="hidden min-w-0 truncate text-xs text-muted-foreground lg:inline">
              Theo dõi bảng công, yêu cầu sửa và lịch sử chi trả theo tháng
            </span>
            <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[11px] font-semibold tabular-nums text-muted-foreground">
              {timesheetManagement.timesheets.length} NV
            </span>
          </div>

          <div className="flex shrink-0 items-center gap-1.5">
            {(isQuanly || isAdmin) && (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setBccUploadOpen(true)}
                  className="h-8 gap-1.5 rounded-md px-3 text-xs font-medium"
                >
                  <FileUp className="h-3.5 w-3.5" aria-hidden="true" />
                  Tải lên BCC
                </Button>
                <Button
                  type="button"
                  size="sm"
                  onClick={handleAddTimesheet}
                  className="h-8 gap-1.5 rounded-md px-3 text-xs font-medium"
                >
                  <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                  Nhập công
                </Button>
              </>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  aria-label="Tác vụ khác"
                  className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-border bg-card text-muted-foreground transition-colors hover:bg-accent/40 hover:text-foreground"
                >
                  <DotsVertical className="h-4 w-4" aria-hidden="true" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="z-40 w-52 rounded-lg border-gray-200 bg-white shadow-lg">
                {(isQuanly || isAdmin) && (
                  <DropdownMenuItem onClick={() => setBccHistoryOpen(true)}>
                    <Clock className="h-4 w-4 mr-2" aria-hidden="true" />
                    Lịch sử BCC
                  </DropdownMenuItem>
                )}
                {!isAdmin && (
                  <DropdownMenuItem
                    onClick={() => void timesheetManagement.handleExportExcel()}
                    disabled={timesheetManagement.isLoading}
                  >
                    <Download01 className="h-4 w-4 mr-2" aria-hidden="true" />
                    Xuất Excel
                  </DropdownMenuItem>
                )}
                {isAdmin && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleGlobalApprove}>
                      <Check className="h-4 w-4 mr-2" aria-hidden="true" />
                      Duyệt hết
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={handleResetAll}>
                      <X className="h-4 w-4 mr-2" aria-hidden="true" />
                      Bỏ duyệt hết
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => setRejectUnpaidDialogOpen(true)}
                      className="text-destructive focus:text-destructive"
                    >
                      <X className="h-4 w-4 mr-2" aria-hidden="true" />
                      Loại công
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Contextual bulk bar — filter-scoped pending count (quanly/ketoan) */}
        {canBulkReview && pendingTimesheets.length > 0 && (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-1.5 motion-safe:animate-fade-in-up [animation-delay:70ms] [animation-fill-mode:both]">
            <span className="text-xs font-medium text-amber-800 dark:text-amber-300">
              {pendingTimesheets.length} bảng công đang chờ trong bộ lọc hiện tại
            </span>
            <div className="ml-auto flex items-center gap-1.5">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setBulkApproveOpen(true)}
                disabled={timesheetManagement.isLoading}
                className="h-8 gap-1.5 rounded-md border-green-600/40 px-2.5 text-xs text-green-700 hover:bg-green-50"
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
                className="h-8 gap-1.5 rounded-md border-red-600/40 px-2.5 text-xs text-red-700 hover:bg-red-50"
              >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
                Loại {pendingTimesheets.length}
              </Button>
            </div>
          </div>
        )}

        <TimesheetProvider
          management={timesheetManagement}
          onEdit={handleEditTimesheet}
          onDelete={handleDelete}
          userRole={userRole}
          onExportExcel={() => timesheetManagement.handleExportExcel()}
        >
          {/* Tier 2 — one filter card: month + filters left, inline KPI strip right.
              flex-wrap everywhere: no horizontal scrollbar (PAYR-134). */}
          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 rounded-xl border border-input bg-card px-2.5 py-1.5 shadow-xs motion-safe:animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:both]">
            <div className="flex min-w-0 flex-wrap items-center gap-1.5">
              <TimesheetMonthSelector
                value={timesheetManagement.selectedMonth}
                onChange={timesheetManagement.setSelectedMonth}
              />
              <TimesheetFilters />
            </div>
            {summaryStats && (
              <TimesheetStatsRow
                isLoading={timesheetStats.isLoading}
                totalHours={totalHours}
                totalPay={totalPay}
                pendingCount={summaryStats.pendingCount}
                approvedCount={summaryStats.approvedCount}
                statusFilter={timesheetManagement.statusFilter}
                onToggleStatus={timesheetManagement.setStatusFilter}
              />
            )}
          </div>

          {/* Admin-only empty states (previously in TimesheetDisplaySection) */}
          {isAdmin && !isMobile && timesheetManagement.timesheets.length === 0 && hasActiveFilter && (
            <TimesheetEmptyState variant="no-filter-results" onClearFilters={handleClearFilters} />
          )}
          {isAdmin && !isMobile && timesheetManagement.timesheets.length === 0 && !hasActiveFilter && (globalSummary?.totalEntries ?? 0) === 0 && (
            <TimesheetEmptyState variant="no-data" onAddTimesheet={handleAddTimesheet} />
          )}

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

      {/* Filter-scoped bulk confirm dialogs (quanly/ketoan) */}
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

      {/* Admin global dialogs */}
      <ConfirmDialog
        open={globalApproveOpen}
        onOpenChange={setGlobalApproveOpen}
        title="Duyệt hết bảng công"
        description={
          <div className="space-y-0 divide-y divide-border/60">
            <div className="flex items-center justify-between py-2 text-sm">
              <span className="text-muted-foreground">Bảng công chờ duyệt</span>
              <span className="font-semibold tabular-nums">{globalSummary?.pendingApproval ?? 0}</span>
            </div>
            <div className="flex items-center justify-between py-2 text-sm">
              <span className="text-muted-foreground">Số NV liên quan</span>
              <span className="font-semibold tabular-nums">{globalSummary?.pendingEmployees ?? '—'}</span>
            </div>
          </div>
        }
        confirmText="Duyệt"
        onConfirm={handleConfirmGlobalApprove}
        loading={approveAllMutation.isPending}
        disabled={(globalSummary?.pendingApproval ?? 0) === 0}
      />

      <ConfirmDialog
        open={resetAllDialogOpen}
        onOpenChange={setResetAllDialogOpen}
        title="Bỏ duyệt tất cả bảng công"
        description={
          <div className="space-y-0 divide-y divide-border/60">
            <div className="flex items-center justify-between py-2 text-sm">
              <span className="text-muted-foreground">Bảng công đã duyệt</span>
              <span className="font-semibold tabular-nums">{globalSummary?.approvedEntries ?? 0}</span>
            </div>
            <div className="flex items-center justify-between gap-4 py-2 text-sm">
              <span className="text-muted-foreground">Bảng công đã thanh toán</span>
              <span className="font-semibold tabular-nums">{globalSummary?.paidEntries ?? 0}</span>
            </div>
            <p className="py-2 text-xs text-muted-foreground">
              Toàn bộ bảng công đã duyệt (chưa thanh toán) sẽ trở về trạng thái chờ duyệt. Bảng công đã thanh toán không bị ảnh hưởng.
            </p>
          </div>
        }
        confirmText="Bỏ duyệt"
        confirmVariant="destructive"
        onConfirm={handleConfirmResetAll}
        loading={resetAllMutation.isPending}
        disabled={(globalSummary?.approvedEntries ?? 0) === 0}
      />

      <RejectUnpaidTimesheetsDialog
        open={rejectUnpaidDialogOpen}
        onOpenChange={setRejectUnpaidDialogOpen}
        initialProjectId={
          timesheetManagement.selectedProject !== 'all'
            ? Number(timesheetManagement.selectedProject)
            : undefined
        }
      />
    </div>
  );
}
