import { useState, useEffect, useCallback, useMemo } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { TimesheetPageHeaderMobile } from "@/components/timesheet/mobile/TimesheetPageHeaderMobile";
import { TimesheetDisplaySection } from "@/components/timesheet/TimesheetDisplaySection";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { Skeleton } from "@/components/ui/skeleton";
import { GroupedStatCard } from "@/components/shared/GroupedStatCard";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
import { Calendar, AlertCircle } from "lucide-react";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useEmployeeModals, useTimesheetModals } from "@/hooks/useModalNavigation";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import { MobileTimesheetEntry } from "@/components/sheets/timesheet-entry/mobile/MobileTimesheetEntry";
import type { Employee } from "@/types/api/employee.types";
import type { Timesheet } from "@/types/api/timesheet.types";

export default function TimesheetsPageMobile() {
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openEmployeeDetails } = useEmployeeModals();

  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  // Full-page entry mode when modal=timesheet_entry
  const isEntryMode = searchParams.get("modal") === "timesheet_entry";
  const urlProjectId = searchParams.get("projectId");
  const urlProjectFilter = searchParams.get("project");
  const entryEmployeeId = searchParams.get("employeeId");

  const urlViewMode = searchParams.get("view") as "table" | "calendar" | null;
  const viewEmployeeId = searchParams.get("employee");

  const handleEntryClose = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete("modal");
    params.delete("projectId");
    params.delete("employeeId");
    navigate(`/partner/timesheet${params.toString() ? `?${params}` : ""}`, {
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
    userRole: "quanly",
    useYearToDate: true,
  });

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

  // Preserve the project-scoped shortcut used by the desktop project list.
  useEffect(() => {
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

  // Sync employee selection from URL
  useEffect(() => {
    if (
      viewEmployeeId &&
      viewEmployeeId !== timesheetManagement.selectedEmployee
    ) {
      timesheetManagement.setSelectedEmployee(viewEmployeeId);
    }
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

  const handleDelete = async (timesheet: Timesheet) => {
    await timesheetManagement.handleDelete(timesheet);
  };

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
        onAddTimesheet={() => openTimesheetEntry()}
        onBccUpload={() => setBccUploadOpen(true)}
        onBccHistory={() => setBccHistoryOpen(true)}
        userRole="quanly"
        monthValue={timesheetManagement.selectedMonth}
        onMonthChange={timesheetManagement.setSelectedMonth}
      />

      {/* Summary stats */}
      {timesheetStats.summary && (
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
          onEdit={timesheetManagement.handleEdit}
          onDelete={handleDelete}
          onAddTimesheet={() => openTimesheetEntry()}
          userRole="quanly"
        />
      </MobileSurface>

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
    </MobilePageShell>
  );
}
