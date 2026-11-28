import { useState, useEffect, useCallback, useMemo } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { TimesheetPageHeaderMobile } from "@/components/timesheet/mobile/TimesheetPageHeaderMobile";
import { TimesheetDisplaySection } from "@/components/timesheet/TimesheetDisplaySection";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { RejectUnpaidTimesheetsDialog } from "@/components/timesheet/RejectUnpaidTimesheetsDialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { GroupedStatCard } from "@/components/shared/GroupedStatCard";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
import { Calendar, AlertCircle } from "lucide-react";
import { useAuth } from "@/contexts";
import { toast } from "@/components/ui/sonner";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useTimesheetModals, useModalNavigation } from "@/hooks/useModalNavigation";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import {
  useApproveAllTimesheets,
  useResetAllTimesheets,
  useTimesheetSummary,
} from "@/hooks/api/useTimesheets";
import { MobileTimesheetEntry } from "@/components/sheets/timesheet-entry/mobile/MobileTimesheetEntry";
import { createProjectBulkApprovalContent } from "@/utils/timesheetBulkHelpers";
import { parseTimesheetStatusFilter } from "@/utils/timesheetFilterHelpers";
import type { Timesheet } from "@/types/api/timesheet.types";

export default function TimesheetPageMobile() {
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  // Provider/hook role semantics preserved from the pre-merge pages
  // (PAYR-134): admin → 'admin'; every partner role → 'quanly'.
  const userRole: "admin" | "quanly" = isAdmin ? "admin" : "quanly";

  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  // Admin-only global flows
  const [bulkApproveDialogOpen, setBulkApproveDialogOpen] = useState(false);
  const [resetAllDialogOpen, setResetAllDialogOpen] = useState(false);
  const [projectBulkApproveDialogOpen, setProjectBulkApproveDialogOpen] = useState(false);
  const [rejectUnpaidDialogOpen, setRejectUnpaidDialogOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openModal } = useModalNavigation();

  // Full-page entry mode when modal=timesheet_entry
  const isEntryMode = searchParams.get("modal") === "timesheet_entry";
  const urlProjectId = searchParams.get("projectId");
  const urlProjectFilter = searchParams.get("project");
  const entryEmployeeId = searchParams.get("employeeId");

  const urlViewMode = searchParams.get("view") as "table" | "calendar" | null;
  const viewEmployeeId = searchParams.get("employee");
  const urlEmployeeId = searchParams.get("employee");
  const urlStatus = parseTimesheetStatusFilter(searchParams.get("status"));

  const handleEntryClose = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete("modal");
    params.delete("projectId");
    params.delete("employeeId");
    navigate(`/timesheet${params.toString() ? `?${params}` : ""}`, {
      replace: true,
    });
  }, [navigate, searchParams]);

  // View mode state
  const [viewMode, setViewMode] = useState<"table" | "calendar">(() => {
    if (urlViewMode === "table" || urlViewMode === "calendar")
      return urlViewMode;
    return viewEmployeeId ? "calendar" : "table";
  });

  const timesheetManagement = useTimesheetManagement({
    userRole,
    useYearToDate: !isAdmin,
  });

  const approveAllMutation = useApproveAllTimesheets();
  const resetAllMutation = useResetAllTimesheets();

  // Global (unfiltered) summary for the "Duyệt hết" confirm — backend
  // approveAll() ignores filters and approves system-wide, so we show the
  // honest, unfiltered scope here.
  const { data: globalSummary, refetch: refetchGlobalSummary } =
    useTimesheetSummary({});

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

  // Open the safe "Duyệt hết" flow — refresh the global summary first so the
  // confirm dialog shows accurate, unfiltered counts.
  const handleBulkApprove = useCallback(() => {
    refetchGlobalSummary();
    setBulkApproveDialogOpen(true);
  }, [refetchGlobalSummary]);

  const handleConfirmBulkApprove = async () => {
    try {
      await approveAllMutation.mutateAsync();
      setBulkApproveDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  // "Bỏ duyệt hết" — reset all approved (unpaid) timesheets to pending.
  const handleResetAll = useCallback(() => {
    refetchGlobalSummary();
    setResetAllDialogOpen(true);
  }, [refetchGlobalSummary]);

  const handleConfirmResetAll = async () => {
    try {
      await resetAllMutation.mutateAsync();
      setResetAllDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  const handleProjectBulkApprove = useCallback(() => {
    if (timesheetManagement.projectPendingTimesheets.length === 0) {
      toast({
        title: "Thông báo",
        description: "Không có bảng công pending nào cho dự án này.",
      });
      return;
    }
    setProjectBulkApproveDialogOpen(true);
  }, [timesheetManagement.projectPendingTimesheets]);

  const handleConfirmProjectBulkApprove = async () => {
    try {
      await timesheetManagement.handleProjectBulkApprove();
      setProjectBulkApproveDialogOpen(false);
      toast({ title: "Duyệt dự án thành công" });
    } catch {
      toast({
        title: "Lỗi",
        description: "Không thể duyệt bảng công dự án.",
        variant: "destructive",
      });
    }
  };

  const statsFilters = useMemo(() => ({
    project_id: timesheetManagement.selectedProject !== "all" ? parseInt(timesheetManagement.selectedProject) : undefined,
    fromDate: timesheetManagement.selectedMonth !== "all" ? `${timesheetManagement.selectedMonth}-01` : undefined,
    toDate: timesheetManagement.selectedMonth !== "all" ? (() => {
      const parts = timesheetManagement.selectedMonth.split("-");
      const lastDay = new Date(parseInt(parts[0]), parseInt(parts[1]), 0).getDate();
      return `${timesheetManagement.selectedMonth}-${lastDay.toString().padStart(2, "0")}`;
    })() : undefined,
  }), [timesheetManagement.selectedProject, timesheetManagement.selectedMonth]);

  const timesheetStats = useTimesheetStatsConfig(statsFilters);

  // Initialize employee/status/project selection from URL on mount only
  useEffect(() => {
    if (
      urlEmployeeId &&
      urlEmployeeId !== timesheetManagement.selectedEmployee
    ) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    if (urlStatus && urlStatus !== timesheetManagement.statusFilter) {
      timesheetManagement.setStatusFilter(urlStatus);
    }
    if (
      urlProjectFilter &&
      urlProjectFilter !== timesheetManagement.selectedProject
    ) {
      timesheetManagement.setSelectedProject(urlProjectFilter);
    }
    // Only initialize from the incoming route; subsequent changes are owned by
    // the page filters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update URL when selection changes
  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);
    const isSpecific =
      timesheetManagement.selectedEmployee &&
      timesheetManagement.selectedEmployee !== "all" &&
      timesheetManagement.selectedEmployee.trim() !== "";

    if (isSpecific) {
      newParams.set("employee", timesheetManagement.selectedEmployee);
      newParams.set("view", viewMode);
    } else {
      newParams.delete("employee");
      newParams.delete("view");
      if (viewMode !== "table") setViewMode("table");
    }

    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [
    timesheetManagement.selectedEmployee,
    viewMode,
    searchParams,
    setSearchParams,
  ]);

  // All hooks must be called BEFORE any early returns

  const handleDelete = useCallback(
    async (timesheet: Timesheet) => {
      await timesheetManagement.handleDelete(timesheet);
    },
    [timesheetManagement],
  );

  // Render full-page entry view AFTER all hooks are defined
  if (isEntryMode) {
    return (
      <div className="flex min-h-[100dvh] flex-col overflow-x-clip bg-card pb-[env(safe-area-inset-bottom)]">
        <MobileTimesheetEntry
          isOpen={true}
          onClose={handleEntryClose}
          projectId={urlProjectId ? Number(urlProjectId) : undefined}
          employeeId={entryEmployeeId ? Number(entryEmployeeId) : undefined}
        />
        <div id="datepicker-portal" />
      </div>
    );
  }

  if (
    timesheetManagement.isLoading &&
    timesheetManagement.timesheets.length === 0
  ) {
    return (
      <MobilePageShell className="space-y-3">
        {/* Mobile header skeleton */}
        <div className="flex justify-between gap-2">
          <div className="space-y-1 flex-1">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-3.5 w-48" />
          </div>
          <Skeleton className="h-9 w-20 shrink-0" />
          <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
        </div>
        {/* Filters skeleton */}
        <div className="flex gap-2">
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 flex-1" />
          <Skeleton className="h-9 flex-1" />
        </div>
        {/* List skeleton */}
        <Skeleton className="h-80" />
      </MobilePageShell>
    );
  }

  return (
    <MobilePageShell className="space-y-3">
      <TimesheetPageHeaderMobile
        onAddTimesheet={handleAddTimesheet}
        onBccUpload={() => setBccUploadOpen(true)}
        onBccHistory={() => setBccHistoryOpen(true)}
        onBulkApprove={isAdmin ? handleBulkApprove : undefined}
        onResetAll={isAdmin ? handleResetAll : undefined}
        onRejectUnpaid={isAdmin ? () => setRejectUnpaidDialogOpen(true) : undefined}
        userRole={userRole}
        monthValue={timesheetManagement.selectedMonth}
        onMonthChange={timesheetManagement.setSelectedMonth}
      />

      {/* Summary stats — partner-role view (admin uses its own dialogs instead) */}
      {!isAdmin && timesheetStats.summary && (
        <div className="space-y-3">
          <GroupedStatCard
            title="Tổng quan"
            icon={Calendar}
            stats={[
              { label: "Nhân viên", value: timesheetStats.summary.totalEmployees ?? 0 },
              { label: "Tổng công", value: timesheetStats.summary.totalEntries },
              {
                label: "Đã duyệt",
                value: timesheetStats.summary.approvedEntries,
                variant: timesheetManagement.statusFilter === "approved"
                  ? "accent" as const
                  : "default" as const,
                isPressed: timesheetManagement.statusFilter === "approved",
                onClick: timesheetStats.summary.approvedEntries > 0
                  ? () => timesheetManagement.setStatusFilter(
                      timesheetManagement.statusFilter === "approved"
                        ? "all"
                        : "approved",
                    )
                  : undefined,
              },
            ]}
          />
          {(timesheetStats.summary.pendingApproval > 0 || (timesheetStats.summary.pendingEmployees ?? 0) > 0) && (
            <GroupedStatCard
              title="Cần xử lý"
              icon={AlertCircle}
              stats={[
                {
                  label: "Chờ duyệt",
                  value: timesheetStats.summary.pendingApproval,
                  variant: "accent" as const,
                  onClick: () => timesheetManagement.setStatusFilter(
                    timesheetManagement.statusFilter === "pending_approval" ? "all" : "pending_approval"
                  ),
                },
                {
                  label: "NV chờ TT",
                  value: timesheetStats.summary.pendingEmployees ?? 0,
                  variant: "accent" as const,
                  onClick: () => timesheetManagement.setStatusFilter(
                    timesheetManagement.statusFilter === "pending_payment" ? "all" : "pending_payment"
                  ),
                },
              ]}
            />
          )}
        </div>
      )}

      <MobileSurface className="overflow-hidden p-3">
        <TimesheetDisplaySection
          timesheetManagement={timesheetManagement}
          onEdit={handleEditTimesheet}
          onDelete={handleDelete}
          onAddTimesheet={handleAddTimesheet}
          onApprove={isAdmin ? timesheetManagement.handleApprove : undefined}
          onBulkApprove={isAdmin ? handleBulkApprove : undefined}
          userRole={userRole}
        />
      </MobileSurface>

      {/* Admin-only global dialogs */}
      {isAdmin && (
        <>
          <ConfirmDialog
            open={bulkApproveDialogOpen}
            onOpenChange={setBulkApproveDialogOpen}
            title="Duyệt hết bảng công"
            description={
              <div className="space-y-0 divide-y divide-border/60">
                <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="text-muted-foreground">Bảng công chờ duyệt</span>
                  <span className="font-semibold tabular-nums min-[380px]:text-right">
                    {globalSummary?.pendingApproval ?? 0}
                  </span>
                </div>
                <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="text-muted-foreground">Số NV liên quan</span>
                  <span className="font-semibold tabular-nums min-[380px]:text-right">
                    {globalSummary?.pendingEmployees ?? "—"}
                  </span>
                </div>
              </div>
            }
            confirmText="Duyệt hết"
            onConfirm={handleConfirmBulkApprove}
            loading={approveAllMutation.isPending}
            disabled={(globalSummary?.pendingApproval ?? 0) === 0}
          />
          <ConfirmDialog
            open={resetAllDialogOpen}
            onOpenChange={setResetAllDialogOpen}
            title="Bỏ duyệt tất cả bảng công"
            description={
              <div className="space-y-0 divide-y divide-border/60">
                <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="text-muted-foreground">Bảng công đã duyệt</span>
                  <span className="font-semibold tabular-nums min-[380px]:text-right">
                    {globalSummary?.approvedEntries ?? 0}
                  </span>
                </div>
                <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="text-muted-foreground">Bảng công đã thanh toán</span>
                  <span className="font-semibold tabular-nums min-[380px]:text-right">
                    {globalSummary?.paidEntries ?? 0}
                  </span>
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
          <ConfirmDialog
            open={projectBulkApproveDialogOpen}
            onOpenChange={setProjectBulkApproveDialogOpen}
            title="Duyệt bảng công theo dự án"
            description={createProjectBulkApprovalContent(
              timesheetManagement.selectedProjectDetails?.name || "Dự án đã chọn",
              timesheetManagement.projectPendingTimesheets,
            )}
            confirmText="Duyệt dự án"
            onConfirm={handleConfirmProjectBulkApprove}
            loading={timesheetManagement.isLoading}
          />
          <RejectUnpaidTimesheetsDialog
            open={rejectUnpaidDialogOpen}
            onOpenChange={setRejectUnpaidDialogOpen}
            initialProjectId={
              timesheetManagement.selectedProject !== "all"
                ? Number(timesheetManagement.selectedProject)
                : undefined
            }
          />
        </>
      )}

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
        projects={timesheetManagement.projects}
      />
    </MobilePageShell>
  );
}
