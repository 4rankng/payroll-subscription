import { TimesheetFilters } from '@/components/timesheet/TimesheetFilters';
import { TimesheetListTable } from '@/components/timesheet/TimesheetListTable';
import { TimesheetMobileList } from '@/components/timesheet/TimesheetMobileList';
import { TimesheetEmptyState } from '@/components/timesheet/TimesheetEmptyState';
import { TimesheetProvider, useTimesheetContext } from '@/components/timesheet/TimesheetContext';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useTimesheetSummary } from '@/hooks/api/useTimesheets';
import type { Timesheet } from '@/types/api/timesheet.types';
import type { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';

interface TimesheetDisplaySectionProps {
  timesheetManagement: ReturnType<typeof useTimesheetManagement>;
  onEdit: (timesheet: Timesheet) => void;
  onApprove?: (timesheet: Timesheet) => void;
  onDelete?: (timesheet: Timesheet) => void;
  onAddTimesheet?: () => void;
  onBulkApprove?: () => void;
  bulkTransferPercentage?: number;
  userRole?: 'admin' | 'quanly';
  onExportExcel?: () => void;
  onPaymentHistory?: () => void;
  isExportLoading?: boolean;
}

export function TimesheetDisplaySection({
  timesheetManagement,
  onEdit,
  onApprove,
  onDelete,
  onAddTimesheet,
  bulkTransferPercentage = 0,
  userRole = 'admin',
  onExportExcel,
  onPaymentHistory,
  isExportLoading = false,
}: TimesheetDisplaySectionProps) {
  return (
    <TimesheetProvider
      management={timesheetManagement}
      onEdit={onEdit}
      onApprove={onApprove}
      onDelete={onDelete}
      userRole={userRole}
      bulkTransferPercentage={bulkTransferPercentage}
      onExportExcel={onExportExcel}
      onPaymentHistory={onPaymentHistory}
      isExportLoading={isExportLoading}
    >
      <TimesheetDisplayContent
        onAddTimesheet={onAddTimesheet}
      />
    </TimesheetProvider>
  );
}

function TimesheetDisplayContent({
  onAddTimesheet,
}: {
  onAddTimesheet?: () => void;
}) {
  const { state, filters } = useTimesheetContext();
  const isMobile = useIsMobile();

  const hasActiveFilter =
    (filters.searchTerm?.trim() || '') !== '' ||
    (filters.selectedProject !== 'all' && filters.selectedProject !== '') ||
    (filters.selectedEmployee !== 'all' && filters.selectedEmployee !== '') ||
    filters.statusFilter !== 'all';

  const { data: globalSummary } = useTimesheetSummary({});

  const isListEmpty =
    !state.isLoading && state.timesheets.length === 0;

  const handleClearFilters = () => {
    filters.onSearchChange?.('');
    filters.onProjectChange('all');
    filters.onEmployeeChange('all');
    filters.onStatusChange('all');
  };

  return (
    <>
      <TimesheetFilters />

      {state.userRole === 'admin' && !isMobile && (
        <>
          {isListEmpty && hasActiveFilter && (
            <TimesheetEmptyState variant="no-filter-results" onClearFilters={handleClearFilters} />
          )}
          {isListEmpty && !hasActiveFilter && (globalSummary?.totalEntries ?? 0) === 0 && (
            <TimesheetEmptyState variant="no-data" onAddTimesheet={onAddTimesheet} />
          )}
        </>
      )}

      <div
        className={isMobile ? '' : 'pb-6'}
        hidden={!isMobile && state.userRole === 'admin' && isListEmpty}
      >
        {isMobile ? <TimesheetMobileList /> : <TimesheetListTable />}
      </div>
    </>
  );
}


