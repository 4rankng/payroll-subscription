import { format } from "date-fns";
import { useState, useMemo, useCallback } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { PageHeader } from "@/components/shared/PageHeader";
import { InlineStatStrip } from "@/components/shared/InlineStatStrip";
import { SearchBar } from "@/components/shared/SearchBar";
import { FilterPill } from "@/components/shared/FilterPill";
import { EmptyState } from "@/components/shared/EmptyState";
import { usePartnerEmployeesData } from "@/hooks/partner-employees/usePartnerEmployeesData";
import { useEmployeesSummary, useRequestEmployeeAccess } from "@/hooks/api/useEmployees";
import { useEmployeeModals } from "@/hooks/useModalNavigation";
import { useEmployeeExport } from "@/hooks/employees/useEmployeeExport";
import { useAssignableProjects } from "@/hooks/api/useProjects";
import { ExportEmployeesModal } from "@/components/modals/ExportEmployeesModal";
import { useTableSorting } from "@/utils/sorting";
import { Skeleton } from "@/components/ui/skeleton";
import { UserAvatar } from "@/components/ui/user-avatar";
import { AlertTriangle, Plus, Download } from "lucide-react";
import {
  Bank,
  CheckCircle,
  Clock,
  CreditCard01,
  Globe01,
  PhoneCall01,
  UserCheck01,
  UserPlus01,
  XClose,
} from "@untitledui/icons";
import {
  Employee,
  getEmployeeProjects,
  getEmployeeProjectCount,
  hasCompleteBankDetails,
} from "@/types/api/employee.types";
import { generateMonthOptions } from "@/utils/dateHelpers";
import { getConsoleKicker } from "@/lib/console";
import { useMediaQuery } from "@/hooks/useBreakpoint";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { canClaimEmployee, canPartnerManageProject } from "@/lib/permissions";
import { cn } from "@/lib/utils";
import type { ColumnDef } from "@tanstack/react-table";

const SCHEDULE_STYLES: Record<string, string> = {
  weekly: "bg-info/10 text-info border-info/30",
  monthly: "bg-primary/10 text-primary border-primary/30",
  flexible: "bg-success/10 text-success border-success/30",
};

const SCHEDULE_LABELS: Record<string, string> = {
  weekly: "lương tuần",
  monthly: "lương tháng",
  flexible: "linh động",
};

const monthOptions = generateMonthOptions(12).map(o => ({
  value: o.value,
  label: `T${o.value.split("-")[1]}/${o.value.split("-")[0]}`,
}));

const EmployeesPage = () => {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const [poolView, setPoolView] = useState<"mine" | "global">("mine");

  // "Tất cả NV" pool: scope=global is part of the query key, and useEmployees
  // stays enabled for every role in canViewEmployees, so quanly/ketoan/giamdoc
  // all fetch the paginated pool. Read-only roles render the same table with
  // the claim affordances hidden (canClaimEmployee gates the "Yêu cầu quản lý"
  // button), never a blank region: an empty result falls through to the
  // structured empty state below.
  const {
    employees,
    pagination,
    isLoading,
    error: listError,
    refetch,
    searchEmployees,
    searchTerm,
    projectId,
    filterByProject,
    statusFilter,
    month,
    updateStatusFilter,
    updateMonth,
    clearAllFilters,
    handlePageChange,
    handlePageSizeChange,
    sortBy,
    sortOrder,
    handleSortChange,
  } = usePartnerEmployeesData(poolView === "global" ? "global" : undefined);

  const canClaim = canClaimEmployee();
  const canCreateEmployee = canPartnerManageProject();

  const { openEmployeeDetails, openAddEmployee } = useEmployeeModals();
  const requestAccess = useRequestEmployeeAccess();
  const handleClaim = useCallback(
    (employee: Employee) => {
      requestAccess.mutate(employee.id, {
        // A successful claim grants immediate detail access. Open the detail
        // sheet only after the server has persisted that permission, rather
        // than sending the partner to a predictable 403 first.
        onSuccess: () => openEmployeeDetails(employee.id.toString()),
      });
    },
    [openEmployeeDetails, requestAccess],
  );

  const { sorting, onSortingChange } = useTableSorting(
    sortBy,
    sortOrder,
    handleSortChange,
  );

  const { data: employeesSummary, isLoading: summaryLoading } =
    useEmployeesSummary();
  const { data: projectsData } = useAssignableProjects();
  const { exportEmployees, isExporting } = useEmployeeExport();
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const handleEmployeeClick = useCallback(
    (employee: Employee) => {
      if (canClaim && poolView === "global" && employee.is_accessible === false) {
        handleClaim(employee);
        return;
      }
      openEmployeeDetails(employee.id.toString());
    },
    [canClaim, handleClaim, openEmployeeDetails, poolView],
  );

  const activeEmployees = useMemo(
    () => employees.filter((emp) => emp.status === "active").length,
    [employees],
  );

  const projects = useMemo(
    () =>
      projectsData?.data?.map((p) => ({
        id: p.id,
        name: p.name,
        code: p.code,
      })) ?? [],
    [projectsData],
  );

  const columns: ColumnDef<Employee>[] = useMemo(
    () => [
      // Global-pool view: management status + claim action replaces bank column
      ...(poolView === "global"
        ? [{
            id: "managed_by",
            header: "Quản lý",
            size: 220,
            minSize: 190,
            accessorFn: (row: Employee) => row.creator_name || "",
            cell: ({ row }: { row: { original: Employee } }) => {
              const employee = row.original;
              const accessible = employee.is_accessible;
              return (
                <div className="flex items-center justify-between gap-3 min-w-0">
                  <div className="min-w-0">
                    <span className="text-[12px] leading-[1.35] text-foreground/80 truncate block" title={employee.creator_name}>
                      {employee.creator_name || "—"}
                    </span>
                    <span className="typography-label-medium text-muted-foreground">
                      {employee.created_at
                        ? format(new Date(employee.created_at), 'dd/MM/yyyy')
                        : ""}
                    </span>
                  </div>
                  {accessible ? (
                    <span className="inline-flex items-center gap-1 shrink-0 text-xs font-semibold text-success bg-success/10 border border-success/30 rounded-full px-2 py-0.5">
                      <CheckCircle className="h-3 w-3" aria-hidden="true" />
                      Đang quản lý
                    </span>
                  ) : canClaim ? (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleClaim(employee);
                      }}
                      disabled={requestAccess.isPending}
                      className="inline-flex items-center gap-1.5 shrink-0 rounded-lg bg-primary px-2.5 py-1.5 text-xs font-semibold text-primary-foreground outline-brand transition-colors hover:bg-primary/90 focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 disabled:opacity-50"
                    >
                      <UserPlus01 className="h-3.5 w-3.5" aria-hidden="true" />
                      Yêu cầu quản lý
                    </button>
                  ) : null}
                </div>
              );
            },
          } satisfies ColumnDef<Employee>]
        : []),
      {
        accessorKey: "fullname",
        header: "Nhân viên",
        size: 260,
        minSize: 220,
        cell: ({ row }) => {
          const employee = row.original;
          const pendingCount = employee.timesheet_summary?.pending_timesheets ?? 0;
          const bankOk = hasCompleteBankDetails(employee);
          // In the global pool, bank fields are deliberately masked until this
          // partner manages the employee. Masked data is not missing data.
          const showBankWarning = employee.is_accessible !== false && !bankOk;
          return (
            <div
              className="flex items-center gap-3 cursor-pointer -mx-2 px-2 py-2 rounded-xl transition-all hover:bg-primary/5 min-w-0 group"
              onClick={(e) => {
                e.stopPropagation();
                handleEmployeeClick(employee);
              }}
            >
              <UserAvatar name={employee.fullname} email={employee.email} cccd={employee.cccd} size="md" />
              <div className="min-w-0 flex-1 space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-[12px] leading-[1.35] font-bold text-foreground truncate group-hover:text-primary transition-colors">
                    {employee.fullname}
                  </span>
                  {pendingCount > 0 && (
                    <TooltipProvider delayDuration={200}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate(`/partner/timesheet?employee=${employee.id}`);
                            }}
                            className="inline-flex items-center gap-0.5 shrink-0 text-xs font-semibold text-warning bg-warning/10 border border-warning/30 rounded px-1.5 py-0.5 outline-brand hover:bg-warning/20 transition-colors cursor-pointer focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2"
                          >
                            <Clock className="h-2.5 w-2.5" aria-hidden="true" />
                            {pendingCount}
                          </button>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                          {pendingCount} bảng công chờ duyệt
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                  {showBankWarning && (
                    <TooltipProvider delayDuration={200}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex items-center gap-1 shrink-0 rounded-full border border-amber-200 bg-amber-50 px-1.5 py-px text-[11px] font-semibold leading-[1.35] text-amber-700">
                            <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
                            Thiếu NH
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                          Chưa cập nhật tài khoản ngân hàng
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                </div>
                <div className="flex items-center gap-2.5 text-muted-foreground">
                  <div className="flex items-center gap-1">
                    <CreditCard01 className="h-3 w-3 shrink-0" aria-hidden="true" />
                    <span className="typography-label-medium font-mono truncate">{employee.cccd}</span>
                  </div>
                  {employee.mobile && (
                    <div className="flex items-center gap-1">
                      <PhoneCall01 className="h-3 w-3 shrink-0" aria-hidden="true" />
                      <span className="typography-label-medium tabular-nums">{employee.mobile}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        },
      },
      {
        id: "project_info",
        header: "Dự án",
        size: 320,
        minSize: 240,
        accessorFn: (row) => {
          const projects = getEmployeeProjects(row);
          return projects.length > 0 ? projects[0].name : "";
        },
        cell: ({ row }) => {
          const employee = row.original;
          const projects = getEmployeeProjects(employee);
          const projectCount = getEmployeeProjectCount(employee);

          if (projectCount === 0) {
            return (
              <span className="inline-flex items-center gap-1 text-xs text-muted-foreground px-2 py-1 rounded-lg bg-muted/45">
                Chưa phân công
              </span>
            );
          }

          const first = projects[0];
          const more = projectCount - 1;

          return (
            <div className="space-y-1 min-w-0">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-[12px] leading-[1.35] text-foreground truncate">
                  {first.name}
                </span>
                {first.payment_schedule && !first.last_date && (
                  <span className={cn(
                    "inline-flex items-center px-1.5 py-px rounded-md text-xs font-semibold border shrink-0",
                    SCHEDULE_STYLES[first.payment_schedule],
                  )}>
                    {SCHEDULE_LABELS[first.payment_schedule]}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-1.5">
                <span className="typography-label-medium text-muted-foreground font-mono truncate">
                  {first.code}
                  {first.position && ` · ${first.position}`}
                </span>
                {more > 0 && (
                  <TooltipProvider delayDuration={200}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex items-center justify-center h-4 min-w-[1.25rem] px-1 rounded text-xs font-bold bg-primary/5 text-primary border border-primary/10 shrink-0">
                          +{more}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs max-w-[200px]">
                        {projects.slice(1).map(p => p.name).join(', ')}
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </div>
            </div>
          );
        },
      },
      ...(poolView !== "global" ? [{
        id: "bank",
        header: "Ngân hàng",
        size: 220,
        accessorFn: (row) => row.bank?.branch_name || "",
        cell: ({ row }) => {
          const employee = row.original;
          const branch = employee.bank?.branch_name;
          if (!branch) return <span className="text-muted-foreground">—</span>;
          return (
            <div className="space-y-0.5 min-w-0">
              <div className="flex items-center gap-2">
                <div className="flex h-5 w-5 items-center justify-center rounded bg-primary/5 border border-primary/10 shrink-0">
                  <Bank className="h-2.5 w-2.5 text-primary" aria-hidden="true" />
                </div>
                <span className="text-[12px] leading-[1.35] text-foreground/80 truncate" title={branch}>
                  {branch}
                </span>
              </div>
              {employee.bank_account_number && (
                <span className="typography-label-medium text-muted-foreground tabular-nums pl-7 truncate block">
                  {employee.bank_account_number}
                </span>
              )}
            </div>
          );
        },
      } satisfies ColumnDef<Employee>] : []),
      {
        accessorKey: "created_at",
        header: "Ngày tạo",
        size: 110,
        cell: ({ row }) => (
          <span className="text-[12px] leading-[1.35] text-muted-foreground tabular-nums">
            {row.original.created_at
              ? format(new Date(row.original.created_at), 'dd/MM/yyyy')
              : "—"}
          </span>
        ),
      },
    ],
    [navigate, handleEmployeeClick, poolView, handleClaim, requestAccess.isPending, canClaim],
  );

  // Console table contract: Ngày tạo is the lowest-value column — hide it at
  // narrow desktop widths instead of a horizontal rail (the detail sheet and
  // export keep the field).
  const isBelowLg = useMediaQuery("(max-width: 1023.98px)");
  const desktopColumns = useMemo(
    () => columns.filter((column) => !(isBelowLg && column.id === "created_at")),
    [columns, isBelowLg],
  );

  const mobileConfig = useMemo(
    () => ({
      mobileFields: [
        {
          key: "project_info",
          label: "Dự án",
          render: (row: Employee) => {
            const projects = getEmployeeProjects(row);
            const count = getEmployeeProjectCount(row);
            if (count === 0) return <span className="text-muted-foreground">Chưa phân công</span>;
            const p = projects[0];
            return (
              <div className="space-y-0.5">
                <span className="typography-body-medium text-foreground">{p.name}</span>
                {p.payment_schedule && (
                  <span className={cn(
                    "ml-1.5 inline-flex items-center px-1.5 py-px rounded text-xs font-semibold border",
                    SCHEDULE_STYLES[p.payment_schedule],
                  )}>
                    {SCHEDULE_LABELS[p.payment_schedule]}
                  </span>
                )}
                {count > 1 && (
                  <span className="ml-1 text-xs text-primary font-bold">+{count - 1}</span>
                )}
              </div>
            );
          },
        },
        {
          key: "bank",
          label: "Ngân hàng",
          render: (row: Employee) => (
            <span className="typography-body-medium text-foreground/80">
              {row.bank?.branch_name || "—"}
            </span>
          ),
        },
      ],
      rowTitle: (row: Employee) => (
        <div className="flex items-center gap-2.5">
          <UserAvatar name={row.fullname} email={row.email} cccd={row.cccd} size="sm" />
          <div className="min-w-0">
            <p className="typography-body-medium font-semibold text-foreground truncate">{row.fullname}</p>
            <div className="flex items-center gap-2 text-muted-foreground">
              <div className="flex items-center gap-1">
                <CreditCard01 className="h-3 w-3" aria-hidden="true" />
                <span className="typography-label-medium font-mono">{row.cccd || "—"}</span>
              </div>
            </div>
          </div>
        </div>
      ),
      rowSubtitle: () => null,
    }),
    [],
  );

  const hasFilters = !!(statusFilter || month || projectId || searchTerm);

  // Only show full-page skeleton on initial load (no data yet).
  // During search/refetch, keep the SearchBar & filters visible so the user
  // doesn't see their input vanish.
  const isInitialLoad = isLoading && employees.length === 0 && !searchTerm;

  if (isInitialLoad) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
        <div className="space-y-2">
          <Skeleton className="h-7 w-36" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-11 w-full max-w-md rounded-lg" />
        <div className="console-panel">
          <div className="console-panel-header">
            <Skeleton className="h-3 w-32" />
          </div>
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex items-center gap-3 border-b border-border/55 px-4 py-3 last:border-b-0">
              <div className="h-8 w-8 rounded-full bg-muted animate-pulse" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3.5 w-36" />
                <Skeleton className="h-2.5 w-28" />
              </div>
              <Skeleton className="h-3.5 w-32" />
              <Skeleton className="h-3.5 w-20" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full px-4 py-5 lg:px-8 lg:py-7">
      <div className="mx-auto max-w-[1480px] space-y-4">

        <div className="motion-safe:animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:both]">
          <PageHeader
            kicker={getConsoleKicker(pathname)}
            title="Nhân viên dự án"
            meta={`${(pagination?.totalRecords ?? employees.length).toLocaleString("vi-VN")} nhân viên`}
            actions={[
              {
                label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
                onClick: () => setExportModalOpen(true),
                icon: Download,
                variant: 'outline' as const,
                className: isExporting ? 'opacity-50 pointer-events-none' : '',
              },
              ...(canCreateEmployee ? [{
                label: 'Thêm',
                onClick: () => openAddEmployee(),
                icon: Plus,
                variant: 'default' as const,
              }] : []),
            ]}
          />
        </div>

        <div className={cn(
          "motion-safe:animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:both]",
          poolView === "global" && "hidden",
        )}>
          <InlineStatStrip
            isLoading={summaryLoading}
            items={
              employeesSummary
                ? [
                    {
                      label: "Tổng",
                      value: employeesSummary.total_employees,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(undefined); },
                    },
                    {
                      label: "Chưa phân công",
                      value: employeesSummary.total_employees - employeesSummary.total_working_employees,
                      highlight: (employeesSummary.total_employees - employeesSummary.total_working_employees) > 0,
                      onClick: () => { updateStatusFilter(statusFilter === "unassigned" ? undefined : "unassigned"); updateMonth(undefined); },
                    },
                    {
                      // Scoped label — the partner dashboard's "Đang làm việc"
                      // counts the selected month's workforce, this strip
                      // counts employment status across all time.
                      label: "Đang làm việc (tổng)",
                      value: employeesSummary.total_working_employees,
                      onClick: () => { updateStatusFilter(statusFilter === "working" ? undefined : "working"); updateMonth(undefined); },
                    },
                    {
                      label: "Thêm tháng này",
                      value: employeesSummary.employees_hired_this_month,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(month ? undefined : new Date().toISOString().slice(0, 7)); },
                    },
                  ]
                : [
                    { label: "Tổng", value: employees.length, onClick: () => { updateStatusFilter(undefined); updateMonth(undefined); } },
                    {
                      label: "Chưa phân công",
                      value: employees.length - activeEmployees,
                      highlight: (employees.length - activeEmployees) > 0,
                      onClick: () => { updateStatusFilter(statusFilter === "unassigned" ? undefined : "unassigned"); updateMonth(undefined); },
                    },
                    {
                      label: "Đang làm việc (tổng)",
                      value: activeEmployees,
                      onClick: () => { updateStatusFilter(statusFilter === "working" ? undefined : "working"); updateMonth(undefined); },
                    },
                    {
                      label: "Thêm tháng này",
                      value: 0,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(month ? undefined : new Date().toISOString().slice(0, 7)); },
                    },
                  ]
            }
          />
        </div>

        <div className="motion-safe:animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:both]">
          <div className="console-filter-bar">
            <div className="inline-flex shrink-0 items-center rounded-lg border border-border/70 bg-muted/40 p-0.5">
              <button
                onClick={() => setPoolView("mine")}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-semibold outline-brand transition-all focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
                  poolView === "mine"
                    ? "bg-card text-foreground shadow-sm ring-1 ring-inset ring-border/60"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <UserCheck01 className="h-3.5 w-3.5" aria-hidden="true" />
                Nhân viên của tôi
              </button>
              <button
                onClick={() => setPoolView("global")}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-semibold outline-brand transition-all focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2",
                  poolView === "global"
                    ? "bg-card text-foreground shadow-sm ring-1 ring-inset ring-border/60"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <Globe01 className="h-3.5 w-3.5" aria-hidden="true" />
                Tất cả NV
              </button>
            </div>

            <SearchBar
              searchTerm={searchTerm}
              onSearchChange={searchEmployees}
              placeholder="Tìm tên, CCCD hoặc số điện thoại..."
              className="w-56 min-w-0 sm:w-64"
            />

            <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

            {projects.length > 0 && (
              <FilterPill
                value={projectId ? projectId.toString() : "all"}
                onChange={(v) => filterByProject(v === "all" ? null : parseInt(v))}
                placeholder="Dự án"
                options={projects.map((p) => ({ value: p.id.toString(), label: p.name }))}
              />
            )}

            <FilterPill
              value={statusFilter ?? "all"}
              onChange={(v) => updateStatusFilter(v === "all" ? undefined : (v as "working" | "unassigned"))}
              placeholder="Trạng thái"
              options={[
                { value: "working", label: "Đang làm việc" },
                { value: "unassigned", label: "Chưa phân công" },
              ]}
            />

            <FilterPill
              value={month ?? "all"}
              onChange={(v) => updateMonth(v === "all" ? undefined : v)}
              placeholder="Tháng"
              options={monthOptions}
            />

            {hasFilters && (
              <button
                onClick={clearAllFilters}
                className="inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-lg px-2 text-xs font-semibold text-muted-foreground outline-brand transition-colors hover:bg-muted hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2"
              >
                <XClose className="h-3 w-3" aria-hidden="true" />
                Xóa lọc
              </button>
            )}
          </div>
        </div>

        {poolView === "global" && (
          <div className="flex items-center gap-2.5 rounded-lg border border-info/25 bg-info/5 px-4 py-3">
            <Globe01 className="h-4 w-4 shrink-0 text-info" aria-hidden="true" />
            <p className="typography-body-medium text-foreground/80">
              {canClaim
                ? (<>Toàn bộ nhân viên trong hệ thống. Nhấn <strong>Yêu cầu quản lý</strong> để thêm nhân viên vào danh sách của bạn thay vì tạo mới trùng lặp.</>)
                : "Toàn bộ nhân viên trong hệ thống."}
            </p>
          </div>
        )}

        <div className={cn(
          "console-panel motion-safe:animate-fade-in-up [animation-delay:200ms] [animation-fill-mode:both]",
          // Dense console table contract for this panel only: 34px sticky
          // header row with capped tracking, 12px body cells. Scoped overrides
          // so the shared table chrome keeps its defaults for other routes.
          "[&_.console-table_thead_tr]:h-[34px]",
          "[&_.console-table_thead_th]:h-[34px]",
          "[&_.console-table_thead_th]:py-[3px]",
          "[&_.console-table_thead_th]:leading-[1.35]",
          "[&_.console-table_thead_th]:tracking-[0.03em]",
          "[&_.console-table_td]:text-[12px]",
          // Zero rows → nothing to scroll: suppress the phantom horizontal
          // scrollbar the sized columns would otherwise force.
          employees.length === 0 && "[&_[data-slot=data-table-desktop]]:overflow-x-hidden",
        )}>
          <div className="console-panel-header justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2">
              <span className="font-financial text-[11px] font-semibold uppercase leading-[1.35] tracking-[0.03em] text-muted-foreground">{poolView === "global" ? "Toàn bộ nhân viên" : "Danh sách nhân sự"}</span>
              <span className="hidden truncate text-xs text-muted-foreground min-[420px]:block">
                {poolView === "global"
                  ? "Danh sách mọi nhân viên trong hệ thống"
                  : "Chọn một nhân viên để xem hồ sơ chi tiết"}
              </span>
            </div>
            <span className="shrink-0 font-financial text-xs tabular-nums text-muted-foreground">{(pagination?.totalRecords ?? employees.length).toLocaleString("vi-VN")} người</span>
          </div>
          {listError && employees.length === 0 ? (
            <EmptyState
              title="Không thể tải danh sách nhân viên"
              description="Đã xảy ra lỗi khi tải dữ liệu. Vui lòng thử lại."
              size="sm"
              action={{ label: "Thử lại", onClick: () => refetch() }}
            >
              {hasFilters && (
                <button
                  onClick={clearAllFilters}
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-border px-3 text-xs font-semibold text-muted-foreground outline-brand transition-colors hover:bg-muted hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2"
                >
                  <XClose className="h-3 w-3" aria-hidden="true" />
                  Xóa lọc
                </button>
              )}
            </EmptyState>
          ) : (
            <ResponsiveTable
              data={employees}
              columns={desktopColumns}
              mobileFields={mobileConfig.mobileFields}
              rowTitle={mobileConfig.rowTitle}
              rowSubtitle={mobileConfig.rowSubtitle}
              getRowId={(row) => row.id.toString()}
              onRowClick={handleEmployeeClick}
              pagination={pagination}
              onPageChange={handlePageChange}
              onPageSizeChange={handlePageSizeChange}
              sorting={sorting}
              onSortingChange={onSortingChange}
              variant="console"
              embedded
              emptyState={
                <EmptyState
                  title={hasFilters ? "Không có nhân viên phù hợp" : poolView === "global" ? "Hệ thống chưa có nhân viên nào" : "Không có nhân viên nào"}
                  description={
                    hasFilters
                      ? "Không có kết quả phù hợp với bộ lọc của bạn."
                      : poolView === "global"
                        ? "Chưa có nhân viên nào trong hệ thống."
                        : "Chưa có nhân viên nào trong các dự án được phân quyền."
                  }
                  size="sm"
                  action={
                    hasFilters
                      ? { label: "Đặt lại bộ lọc", onClick: () => clearAllFilters() }
                      : undefined
                  }
                />
              }
              accordionType="single"
            />
          )}
        </div>

      </div>

      <ExportEmployeesModal
        open={exportModalOpen}
        onClose={() => setExportModalOpen(false)}
        onExport={(projectIds) => exportEmployees(projectIds)}
        projects={projects}
        isExporting={isExporting}
      />
    </div>
  );
};

export default EmployeesPage;
