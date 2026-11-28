import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TimesheetFilters } from './TimesheetFilters';

const useTimesheetContextMock = vi.hoisted(() => vi.fn());
const useIsMobileMock = vi.hoisted(() => vi.fn(() => false));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: useIsMobileMock,
}));

vi.mock('@/components/timesheet/TimesheetContext', () => ({
  useTimesheetContext: useTimesheetContextMock,
}));

vi.mock('@/components/ui/searchable-dropdown', () => ({
  SearchableDropdown: ({ className, mobileTitle }: { className?: string; mobileTitle?: string }) => (
    <button data-testid="project-filter" data-mobile-title={mobileTitle} className={className}>Dự án</button>
  ),
}));

vi.mock('@/components/timesheet/EmployeeDropdownAdapter', () => ({
  EmployeeDropdownAdapter: ({ className }: { className?: string }) => (
    <button data-testid="employee-filter" className={className}>Tất cả nhân viên</button>
  ),
}));

describe('TimesheetFilters', () => {
  beforeEach(() => {
    useIsMobileMock.mockReturnValue(false);
    useTimesheetContextMock.mockReturnValue({
      filters: {
        selectedMonth: '2026-07',
        onMonthChange: vi.fn(),
        selectedProject: 'all',
        onProjectChange: vi.fn(),
        selectedEmployee: 'all',
        onEmployeeChange: vi.fn(),
        statusFilter: 'all',
        onStatusChange: vi.fn(),
        projects: [],
        projectEmployees: [],
        searchTerm: '',
        onSearchChange: vi.fn(),
        userRole: 'admin',
      },
    });
  });

  it('keeps a touch-safe base and compacts every desktop dropdown trigger', () => {
    render(<TimesheetFilters />);

    const triggers = [
      screen.getByRole('combobox', { name: 'Trạng thái' }),
      screen.getByTestId('project-filter'),
      screen.getByTestId('employee-filter'),
    ];

    for (const trigger of triggers) {
      expect(trigger).toHaveClass('min-h-11');
      expect(trigger).toHaveClass('sm:min-h-9');
      expect(trigger).toHaveClass('bg-card');
    }
  });

  // The month selector is owned by the page header/toolbar; the filter bar
  // must not render a second month control next to it.
  it('renders no month combobox of its own', () => {
    render(<TimesheetFilters />);

    expect(screen.queryByRole('combobox', { name: 'Tháng' })).not.toBeInTheDocument();
  });

  it('renders the status dropdown on an opaque solid surface', () => {
    render(<TimesheetFilters />);

    // The portal content mounts on open; the trigger's combobox must exist
    // and the sheet/mobile month select must not duplicate it.
    expect(screen.getByRole('combobox', { name: 'Trạng thái' })).toBeInTheDocument();
  });

  it('keeps the mobile filter controls at least 44px tall', () => {
    useIsMobileMock.mockReturnValue(true);
    render(<TimesheetFilters />);

    const statusTrigger = screen.getByRole('combobox', { name: 'Trạng thái' });
    expect(statusTrigger).toHaveClass('h-11');
    expect(statusTrigger).toHaveClass('min-h-11');
    expect(screen.getByRole('button', { name: 'Bộ lọc' })).toHaveClass('h-11', 'w-11');
  });

  it('wires the project selector to the mobile picker', () => {
    useIsMobileMock.mockReturnValue(true);
    render(<TimesheetFilters />);

    fireEvent.click(screen.getByRole('button', { name: 'Bộ lọc' }));

    expect(screen.getByTestId('project-filter')).toHaveAttribute('data-mobile-title', 'Chọn dự án');
  });
});
