import { useMemo } from 'react';
import { SearchBar } from '@/components/shared/SearchBar';
import { SearchableDropdown } from '@/components/ui/searchable-dropdown';
import { EmployeeDropdownAdapter } from '@/components/timesheet/EmployeeDropdownAdapter';
import type { Employee } from '@/types/api/employee.types';
import { Badge } from '@/components/ui/badge';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { SlidersHorizontal } from 'lucide-react';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useTimesheetContext } from '@/components/timesheet/TimesheetContext';
import { cn } from '@/lib/utils';

const STATUS_OPTIONS = [
  { value: 'pending_approval', label: 'Chờ duyệt' },
  { value: 'pending_payment', label: 'Chờ TT' },
  { value: 'paid', label: 'Đã thanh toán' },
  { value: 'rejected', label: 'Bị loại' },
];

/** Solid status dropdown: opaque white surface, hard border and a strong
 *  shadow so table text never bleeds through while it is open. */
const SolidStatusFilter = ({ className }: { className?: string }) => {
  const { filters } = useTimesheetContext();
  const { statusFilter, onStatusChange } = filters;
  const activeLabel = STATUS_OPTIONS.find(o => o.value === statusFilter)?.label;

  return (
    <Select value={statusFilter} onValueChange={onStatusChange}>
      <SelectTrigger
        aria-label="Trạng thái"
        className={cn(
          'inline-flex h-11 min-h-11 w-auto items-center gap-1.5 rounded-lg border px-3 py-0 text-sm font-medium whitespace-nowrap select-none outline-brand transition-colors duration-100 sm:h-9 sm:min-h-9 sm:px-2.5',
          statusFilter !== 'all'
            ? 'border-primary/30 bg-primary/8 text-primary hover:bg-primary/12'
            : 'border-border/60 bg-card text-muted-foreground hover:border-border hover:text-foreground hover:bg-accent/40',
          className,
        )}
      >
        <span>{statusFilter !== 'all' ? activeLabel : 'Trạng thái'}</span>
      </SelectTrigger>
      <SelectContent className="z-40 rounded-lg border border-gray-200 bg-white shadow-lg">
        <SelectItem value="all">Trạng thái</SelectItem>
        {STATUS_OPTIONS.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}
      </SelectContent>
    </Select>
  );
};

export const TimesheetFilters = () => {
  const { filters } = useTimesheetContext();
  const {
    selectedProject,
    onProjectChange,
    selectedEmployee,
    onEmployeeChange,
    statusFilter,
    projects,
    projectEmployees,
    searchTerm = '',
    onSearchChange,
  } = filters;

  const isMobile = useIsMobile();

  const projectOptions = useMemo(() =>
    projects.map(p => {
      const parts = p.name.split(' - ');
      return { value: p.id.toString(), label: `${parts[0]} (${p.code})` };
    }), [projects]);

  const shouldUseProjectEmployees = selectedProject !== 'all' && projectEmployees && projectEmployees.length > 0;

  // The month control lives in the page header / toolbar (TimesheetMonthSelector),
  // so only the filters owned here count toward the badge.
  const activeFilterCount = [
    statusFilter !== 'all',
    selectedProject !== 'all',
    selectedEmployee !== 'all',
  ].filter(Boolean).length;

  // ── Mobile ─────────────────────────────────────────────────────────────────
  if (isMobile) {
    return (
      <div className="flex flex-col gap-1.5">
        {onSearchChange && (
          <SearchBar
            searchTerm={searchTerm}
            onSearchChange={onSearchChange}
            placeholder="Tìm kiếm..."
            className="w-full"
          />
        )}
        <div className="flex items-center gap-1.5">
        <SolidStatusFilter className="shrink-0" />
        <Sheet>
          <SheetTrigger asChild>
            <button
              className="relative inline-flex h-11 w-11 items-center justify-center rounded-lg border border-border/60 text-muted-foreground transition-colors hover:border-border hover:text-foreground"
              aria-label="Bộ lọc"
            >
              <SlidersHorizontal className="h-3.5 w-3.5" />
              {activeFilterCount > 0 && (
                <Badge className="absolute -top-1.5 -right-1.5 h-4 min-w-4 p-0 text-xs flex items-center justify-center rounded-full">
                  {activeFilterCount}
                </Badge>
              )}
            </button>
          </SheetTrigger>
          <SheetContent side="bottom" className="h-auto max-h-[85dvh] overflow-y-auto px-4 pt-4">
            <SheetHeader className="mb-4">
              <SheetTitle className="text-base">Bộ lọc</SheetTitle>
            </SheetHeader>
            <div className="space-y-3 pb-4" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Trạng thái</Label>
                <Select value={statusFilter} onValueChange={filters.onStatusChange}>
                  <SelectTrigger aria-label="Trạng thái bảng công" className="h-11"><SelectValue placeholder="Tất cả trạng thái" /></SelectTrigger>
                  <SelectContent className="z-40 border border-gray-200 bg-white shadow-lg">
                    <SelectItem value="all">Tất cả trạng thái</SelectItem>
                    {STATUS_OPTIONS.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Dự án</Label>
                <SearchableDropdown
                  value={selectedProject} onValueChange={onProjectChange}
                  options={projectOptions}
                  placeholder="Tất cả dự án" searchPlaceholder="Tìm dự án..."
                  emptyMessage="Không tìm thấy dự án nào." className="w-full"
                  allOption={{ value: 'all', label: 'Tất cả dự án' }}
                  mobileTitle="Chọn dự án"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Nhân viên</Label>
                <EmployeeDropdownAdapter
                  value={selectedEmployee} onChange={onEmployeeChange}
                  selectedProject={selectedProject} placeholder="Tất cả nhân viên"
                  className="w-full"
                  availableEmployees={(shouldUseProjectEmployees ? projectEmployees : []) as unknown as Employee[]}
                  includeAvailableOnly={shouldUseProjectEmployees}
                />
              </div>
            </div>
          </SheetContent>
        </Sheet>
        </div>
      </div>
    );
  }

  // ── Desktop ────────────────────────────────────────────────────────────────
  return (
    <div className="console-filter-bar">
      {onSearchChange && (
        <SearchBar
          searchTerm={searchTerm}
          onSearchChange={onSearchChange}
          placeholder="Tìm nhân viên, dự án..."
          className="w-64"
        />
      )}

      <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

      <SolidStatusFilter />

      <SearchableDropdown
        value={selectedProject}
        onValueChange={onProjectChange}
        options={projectOptions}
        placeholder="Dự án"
        searchPlaceholder="Tìm dự án..."
        emptyMessage="Không tìm thấy dự án nào."
        className="h-11 min-h-11 rounded-lg bg-card px-3 text-sm sm:h-9 sm:min-h-9 sm:px-2.5"
        allOption={{ value: 'all', label: 'Dự án' }}
        pillStyle
      />

      <EmployeeDropdownAdapter
        value={selectedEmployee}
        onChange={onEmployeeChange}
        selectedProject={selectedProject}
        placeholder="Nhân viên"
        className="inline-flex h-11 min-h-11 items-center gap-1 rounded-lg border border-border/60 bg-card px-3 py-0 text-sm font-medium whitespace-nowrap text-muted-foreground transition-colors duration-100 hover:border-border hover:bg-accent/40 hover:text-foreground sm:h-9 sm:min-h-9 sm:px-2.5 [&>svg]:h-3.5 [&>svg]:w-3.5"
        availableEmployees={(shouldUseProjectEmployees ? projectEmployees : []) as unknown as Employee[]}
        includeAvailableOnly={shouldUseProjectEmployees}
        pillStyle
      />
    </div>
  );
};
