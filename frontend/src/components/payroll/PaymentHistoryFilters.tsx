import { memo, useMemo, useEffect, useRef } from 'react';
import { addMonths, endOfMonth, format, startOfMonth, subMonths } from 'date-fns';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { useProjects } from '@/hooks/api/useProjects';
import { useEmployees } from '@/hooks/api/useEmployees';
import { useProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { SearchBar } from '@/components/shared/SearchBar';
import { cn } from '@/lib/utils';
import { X } from 'lucide-react';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

/** Calendar bounds keep the ledger span within the hook's 36-month fan-out. */
const CALENDAR_MIN_DATE = startOfMonth(addMonths(new Date(), -35));
const CALENDAR_MAX_DATE = new Date();

interface PaymentHistoryFiltersProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
  /** Bump to programmatically focus the range inputs (empty-state quick action). */
  dateFocusSignal?: number;
}

export const PaymentHistoryFilters = memo(function PaymentHistoryFilters({
  filters,
  onFiltersChange,
  onClearFilters,
  hasFilters,
  dateFocusSignal,
}: PaymentHistoryFiltersProps) {
  const { data: projectsData } = useProjects({ pageSize: 100, status: ['active', 'paused', 'completed'] });

  const selectedProjectId = useMemo(() => filters.projectId?.[0], [filters.projectId]);

  const { data: projectEmployeesData, isLoading: isLoadingProjectEmployees } = useProjectEmployees(
    selectedProjectId!,
    { pageSize: 100, status: 'current' },
    !!selectedProjectId
  );
  const { data: allEmployeesData, isLoading: isLoadingAllEmployees } = useEmployees(
    !selectedProjectId ? { pageSize: 100, sortBy: 'fullname', sortOrder: 'asc' } : undefined
  );

  const employeesData = useMemo(() => {
    if (selectedProjectId && projectEmployeesData?.data) {
      return projectEmployeesData;
    }
    return allEmployeesData;
  }, [selectedProjectId, projectEmployeesData, allEmployeesData]);

  const isLoadingEmployees = selectedProjectId ? isLoadingProjectEmployees : isLoadingAllEmployees;
  const selectedEmployeeId = useMemo(() => filters.employeeId?.[0], [filters.employeeId]);

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  const handleProjectChange = (value: string) => {
    if (value === 'all') {
      onFiltersChange({ ...filters, projectId: undefined, employeeId: undefined, page: 1 });
    } else {
      onFiltersChange({ ...filters, projectId: [parseInt(value)], employeeId: undefined, page: 1 });
    }
  };

  const handleEmployeeChange = (value: string) => {
    set('employeeId', value === 'all' ? undefined : [parseInt(value)]);
  };

  // Keep the range valid when one bound moves past the other: picking a
  // "from" after the current "to" pulls "to" along (and vice versa).
  const handleStartDateChange = (date: string) => {
    const toDate = filters.toDate;
    onFiltersChange({
      ...filters,
      fromDate: date,
      toDate: toDate && date > toDate ? date : toDate,
      page: 1,
    });
  };

  const handleEndDateChange = (date: string) => {
    const fromDate = filters.fromDate;
    onFiltersChange({
      ...filters,
      fromDate: fromDate && date < fromDate ? date : fromDate,
      toDate: date,
      page: 1,
    });
  };

  const now = new Date();
  const presets = useMemo(
    () => [
      {
        label: 'Tháng này',
        from: format(startOfMonth(now), 'yyyy-MM-dd'),
        to: format(endOfMonth(now), 'yyyy-MM-dd'),
      },
      {
        label: 'Tháng trước',
        from: format(startOfMonth(subMonths(now, 1)), 'yyyy-MM-dd'),
        to: format(endOfMonth(subMonths(now, 1)), 'yyyy-MM-dd'),
      },
      {
        label: '3 tháng gần nhất',
        from: format(startOfMonth(subMonths(now, 2)), 'yyyy-MM-dd'),
        to: format(endOfMonth(now), 'yyyy-MM-dd'),
      },
    ],
    // `now` is intentionally read once per mount so preset labels stay stable.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const rangeRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!dateFocusSignal) return;
    const input = rangeRef.current?.querySelector<HTMLInputElement>(
      'input[aria-label="Ngày bắt đầu"]',
    );
    input?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
    input?.focus();
  }, [dateFocusSignal]);

  return (
    <div className="console-filter-bar overflow-x-auto">
      <SearchBar
        searchTerm={filters.search || ''}
        onSearchChange={(v) => set('search', v || undefined)}
        placeholder="Tìm tên, CCCD..."
        className="w-44 shrink-0"
      />

      <div ref={rangeRef} className="flex shrink-0 items-center gap-1">
        <DateRangePicker
          variant="compact"
          className="tabular-nums"
          startDate={filters.fromDate}
          endDate={filters.toDate}
          onStartDateChange={handleStartDateChange}
          onEndDateChange={handleEndDateChange}
          minDate={CALENDAR_MIN_DATE}
          maxDate={CALENDAR_MAX_DATE}
        />
        {presets.map((preset) => {
          const active = filters.fromDate === preset.from && filters.toDate === preset.to;
          return (
            <button
              key={preset.label}
              type="button"
              onClick={() =>
                onFiltersChange({ ...filters, fromDate: preset.from, toDate: preset.to, page: 1 })
              }
              aria-pressed={active}
              className={cn(
                'h-8 shrink-0 whitespace-nowrap rounded-md px-2 text-[12px] leading-none transition-colors',
                active
                  ? 'bg-accent font-medium text-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground',
              )}
            >
              {preset.label}
            </button>
          );
        })}
      </div>

      <SearchableSelect
        value={selectedProjectId?.toString() || 'all'}
        onChange={handleProjectChange}
        options={[
          { value: 'all', label: 'Tất cả dự án' },
          ...(projectsData?.data?.map((p) => ({ value: p.id.toString(), label: p.name })) ?? []),
        ]}
        placeholder="Tất cả dự án"
        searchPlaceholder="Tìm dự án..."
        triggerClassName="h-8 min-h-0 w-[148px] shrink-0 text-xs"
      />

      <SearchableSelect
        key={selectedProjectId || 'all'}
        value={selectedEmployeeId?.toString() || 'all'}
        onChange={handleEmployeeChange}
        options={[
          { value: 'all', label: 'Tất cả nhân viên' },
          ...(employeesData?.data?.map((e) => ({ value: e.id.toString(), label: e.name })) ?? []),
        ]}
        placeholder={isLoadingEmployees ? 'Đang tải...' : 'Tất cả nhân viên'}
        disabled={isLoadingEmployees}
        searchPlaceholder="Tìm nhân viên..."
        triggerClassName="h-8 min-h-0 w-[148px] shrink-0 text-xs"
      />

      {hasFilters && (
        <button
          onClick={onClearFilters}
          className="flex items-center gap-1 h-8 px-2 shrink-0 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <X className="h-3 w-3" />
          Xóa lọc
        </button>
      )}
    </div>
  );
});
