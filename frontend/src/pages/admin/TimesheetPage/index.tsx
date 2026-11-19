import { parseTimesheetStatusFilter } from '@/utils/timesheetFilterHelpers';
import { useState, useMemo, useEffect, useCallback } from 'react';
import { useSearchParams, useLocation } from 'react-router-dom';
import { format, parse } from 'date-fns';
import { getConsoleKicker } from '@/lib/console';
import {
  CheckDone01,
  DotsVertical,
  Grid01,
  Plus,
  RefreshCw05,
  Trash01,
  UploadCloud01,
} from '@untitledui/icons';
import { PageHeader } from '@/components/shared/PageHeader';
import { MissingBankDetailsSection } from '@/components/employees/MissingBankDetailsSection';
import { TimesheetDisplaySection } from '@/components/timesheet/TimesheetDisplaySection';
import { UploadHistorySheet } from '@/components/timesheet/UploadHistorySheet';
import { BCCUploadModal } from '@/components/timesheet/BCCUploadModal';
import { RejectUnpaidTimesheetsDialog } from '@/components/timesheet/RejectUnpaidTimesheetsDialog';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';
import { useApproveAllTimesheets, useResetAllTimesheets, useCashReadiness, useTimesheetSummary } from '@/hooks/api/useTimesheets';
import { PayrollControlCenter } from '@/components/timesheet/PayrollControlCenter';
import { useEmployeeModals, useModalNavigation } from '@/hooks/useModalNavigation';
import { MODAL_IDS } from '@/constants/modalRegistry';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { Skeleton } from '@/components/ui/skeleton';
import type { Employee } from '@/types/api/employee.types';
import type { Timesheet } from '@/types/api/timesheet.types';

const TimesheetPage = () => {
  const location = useLocation();
  const [bulkApproveDialogOpen, setBulkApproveDialogOpen] = useState(false);
  const [resetAllDialogOpen, setResetAllDialogOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [rejectUnpaidDialogOpen, setRejectUnpaidDialogOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();

  // Get initial values from URL parameters
  const urlEmployeeId = searchParams.get('employee');
  const urlStatus = parseTimesheetStatusFilter(searchParams.get('status'));

  const timesheetManagement = useTimesheetManagement({ userRole: 'admin' });
  const { openModal } = useModalNavigation();
  const { openEmployeeDetails } = useEmployeeModals();

  // Global summary (no filters) for "Duyệt hết" confirm dialog — backend
  // approveAll() ignores filters and approves system-wide, so we show the
  // honest, unfiltered scope here.
  const { data: globalSummary, refetch: refetchGlobalSummary } = useTimesheetSummary({});
  const cashReadiness = useCashReadiness({});

  // Initialize employee and status selection from URL on mount only
  useEffect(() => {
    if (urlEmployeeId && urlEmployeeId !== timesheetManagement.selectedEmployee) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    if (urlStatus && urlStatus !== timesheetManagement.statusFilter) {
      timesheetManagement.setStatusFilter(urlStatus);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update URL when employee selection changes
  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);
    const isSpecific = timesheetManagement.selectedEmployee &&
                       timesheetManagement.selectedEmployee !== 'all' &&
                       timesheetManagement.selectedEmployee.trim() !== '';
    if (isSpecific) {
      newParams.set('employee', timesheetManagement.selectedEmployee);
    } else {
      newParams.delete('employee');
      newParams.delete('view');
    }
    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [timesheetManagement.selectedEmployee, searchParams, setSearchParams]);


  // Use the same filters for stats config to ensure consistent date range
  const statsFilters = useMemo(() => ({
    project_id: timesheetManagement.selectedProject !== 'all' ? parseInt(timesheetManagement.selectedProject) : undefined,
    fromDate: timesheetManagement.selectedMonth !== 'all' ? `${timesheetManagement.selectedMonth}-01` : undefined,
    toDate: timesheetManagement.selectedMonth !== 'all' ? (() => {
      const lastDay = new Date(
        parseInt(timesheetManagement.selectedMonth.split('-')[0]),
        parseInt(timesheetManagement.selectedMonth.split('-')[1]),
        0
      ).getDate();
      return `${timesheetManagement.selectedMonth}-${lastDay.toString().padStart(2, '0')}`;
    })() : undefined,
  }), [timesheetManagement.selectedProject, timesheetManagement.selectedMonth]);

  const { data: timesheetSummary, isLoading: isTimesheetSummaryLoading } = useTimesheetSummary(statsFilters);

  const approveAllMutation = useApproveAllTimesheets();
  const resetAllMutation = useResetAllTimesheets();

  const handleAddTimesheet = () => {
    openModal(MODAL_IDS.TIMESHEET_ENTRY);
  };

  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const handleEditTimesheet = (timesheet: Timesheet) => {
    openModal(MODAL_IDS.TIMESHEET_ENTRY, {
      projectId: timesheet.project_id?.toString(),
      employeeId: timesheet.employee_id?.toString(),
      entryId: timesheet.id?.toString()
    });
  };

  const handleBulkApprove = () => {
    refetchGlobalSummary();
    setBulkApproveDialogOpen(true);
  };

  const handleConfirmBulkApprove = async () => {
    try {
      await approveAllMutation.mutateAsync();
      setBulkApproveDialogOpen(false);
    } catch {
      // Error is handled by the mutation
    }
  };

  // "Bỏ duyệt hết" — resets all approved (unpaid) timesheets back to pending.
  // Mirrors the "Duyệt hết" flow: refresh the global summary first so the
  // confirm dialog shows accurate, unfiltered counts.
  const handleResetAll = () => {
    refetchGlobalSummary();
    setResetAllDialogOpen(true);
  };

  const handleConfirmResetAll = async () => {
    try {
      await resetAllMutation.mutateAsync();
      setResetAllDialogOpen(false);
    } catch {
      // Error is handled by the mutation
    }
  };

  const handleApprove = async (timesheet: Timesheet) => {
    await timesheetManagement.handleApprove(timesheet);
  };

  const handleDelete = async (timesheet: Timesheet) => {
    await timesheetManagement.handleDelete(timesheet);
  };

  const handleBccHistory = useCallback(() => {
    setBccHistoryOpen(true);
  }, []);

  const handleCloseBccHistory = useCallback(() => {
    setBccHistoryOpen(false);
  }, []);


  if (timesheetManagement.isLoading && timesheetManagement.timesheets.length === 0) {
    return (
      <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1.5">
            <Skeleton className="h-7 w-40" />
            <Skeleton className="h-4 w-56" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-9 w-24" />
            <Skeleton className="h-9 w-9" />
          </div>
        </div>
        <div className="console-panel">
          <div className="console-panel-body">
            <Skeleton className="h-24 w-full" />
          </div>
        </div>
        <div className="console-panel">
          <div className="console-panel-body space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-80 w-full" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
      {/* ── Header ── */}
      <PageHeader
        title="Bảng công"
        kicker={getConsoleKicker(location.pathname)}
        meta={
          timesheetManagement.selectedMonth !== 'all'
            ? format(parse(`${timesheetManagement.selectedMonth}-01`, 'yyyy-MM-dd', new Date()), 'MM/yyyy')
            : undefined
        }
      >
        {/* PAYR-33: one control cluster — period picker and bulk actions share a
            single bordered strip so they read (and wrap) as one unit. Stacks
            internally below lg where the sidebar leaves it no room. */}
        <div className="flex w-full max-w-full flex-col overflow-hidden rounded-xl border border-border bg-card lg:w-auto lg:flex-row lg:items-stretch">
          <TimesheetMonthSelector
            value={timesheetManagement.selectedMonth}
            onChange={timesheetManagement.setSelectedMonth}
            className="rounded-none border-0 shadow-none lg:w-auto lg:shrink-0"
          />
          <div className="flex flex-1 flex-wrap items-center gap-px border-t border-border lg:flex-none lg:border-l lg:border-t-0">
          <button
            onClick={handleBulkApprove}
            className="inline-flex items-center gap-1.5 h-9 px-3 bg-card text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors border-r border-border"
          >
            <CheckDone01 className="h-4 w-4 shrink-0" aria-hidden="true" />
            Duyệt hết
          </button>
          <button
            onClick={handleResetAll}
            className="inline-flex items-center gap-1.5 h-9 px-3 bg-card text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors border-r border-border"
          >
            <RefreshCw05 className="h-4 w-4 shrink-0" aria-hidden="true" />
            Bỏ duyệt
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                aria-label="Thêm tùy chọn"
                className="inline-flex items-center justify-center h-9 w-9 bg-card text-foreground hover:bg-muted transition-colors"
              >
                <DotsVertical className="h-4 w-4" aria-hidden="true" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              {/* Group 1: Nhập / Input */}
              <DropdownMenuItem onClick={handleAddTimesheet}>
                <Plus className="w-4 h-4 mr-2" aria-hidden="true" />
                Nhập công
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {/* Group 2: Import / Upload */}
              <DropdownMenuItem onClick={() => setBccUploadOpen(true)}>
                <UploadCloud01 className="w-4 h-4 mr-2" aria-hidden="true" />
                Tải lên BCC
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleBccHistory}>
                <Grid01 className="w-4 h-4 mr-2" aria-hidden="true" />
                Lịch sử BCC
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => setRejectUnpaidDialogOpen(true)}
                className="text-destructive focus:text-destructive"
              >
                <Trash01 className="w-4 h-4 mr-2" aria-hidden="true" />
                Loại công
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          </div>
        </div>
      </PageHeader>

      {/* ── Payroll Control Center: cash readiness + operational KPIs ── */}
      <PayrollControlCenter
        cashReadiness={{
          data: cashReadiness.data,
          isLoading: cashReadiness.isLoading,
          isError: cashReadiness.isError,
        }}
        stats={{
          summary: timesheetSummary,
          isLoading: isTimesheetSummaryLoading,
        }}
        activeFilter={timesheetManagement.statusFilter}
        onFilterChange={timesheetManagement.setStatusFilter}
      />

      <MissingBankDetailsSection onEmployeeClick={handleEmployeeClick} />

      <TimesheetDisplaySection
        timesheetManagement={timesheetManagement}
        onEdit={handleEditTimesheet}
        onApprove={handleApprove}
        onDelete={handleDelete}
        onAddTimesheet={handleAddTimesheet}
        onBulkApprove={handleBulkApprove}
        userRole="admin"
      />

      <ConfirmDialog
        open={bulkApproveDialogOpen}
        onOpenChange={setBulkApproveDialogOpen}
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

      {/* BCC Upload */}
      <BCCUploadModal
        open={bccUploadOpen}
        onClose={() => setBccUploadOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== 'all'
            ? parseInt(timesheetManagement.selectedProject, 10)
            : 0
        }
        projects={timesheetManagement.projects}
      />

      {/* BCC Upload History */}
      <UploadHistorySheet
        open={bccHistoryOpen}
        onClose={handleCloseBccHistory}
        projects={timesheetManagement.projects}
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
};

export default TimesheetPage;
