import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import EmployeesPageMobile from "./index";

const mocks = vi.hoisted(() => ({
  exportEmployees: vi.fn(),
}));

vi.mock("@/hooks/employees/useEmployeeDataInfinite", () => ({
  useEmployeeDataInfinite: () => ({
    employees: [],
    loading: false,
    searchEmployees: vi.fn(),
    searchTerm: "",
    updateFilters: vi.fn(),
    clearFilters: vi.fn(),
    currentFilters: {},
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 0 },
    hasMore: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
}));
vi.mock("@/hooks/api/useEmployees", () => ({
  useEmployeesSummary: () => ({ data: undefined, isLoading: false }),
}));
vi.mock("@/hooks/api/useProjects", () => ({
  useAssignableProjects: () => ({ data: { data: [] } }),
}));
vi.mock("@/hooks/employees/useEmployeeExport", () => ({
  useEmployeeExport: () => ({
    exportEmployees: mocks.exportEmployees,
    isExporting: false,
  }),
}));
vi.mock("@/hooks/useModalNavigation", () => ({
  useEmployeeModals: () => ({ openEmployeeDetails: vi.fn(), openAddEmployee: vi.fn() }),
}));
vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: ({ actions }: { actions: React.ReactNode }) => <div>{actions}</div>,
}));
vi.mock("@/components/shared/MobileStatStrip", () => ({ MobileStatStrip: () => null }));
vi.mock("@/components/shared/MobileSearchInput", () => ({
  MobileSearchInput: () => <div />,
}));
vi.mock("@/components/shared/MobilePagination", () => ({ MobilePagination: () => null }));
vi.mock("@/components/employees/EmployeeMobileCard", () => ({
  EmployeeMobileCard: () => <div>card</div>,
}));
vi.mock("@/components/employees/EmployeeEmptyStates", () => ({ EmployeeEmptyStates: () => null }));
vi.mock("@/components/employees/MissingBankDetailsSection", () => ({
  MissingBankDetailsSection: () => null,
}));
vi.mock("@/components/ui/searchable-select", () => ({ SearchableSelect: () => null }));
vi.mock("@/components/modals/ExportEmployeesModal", () => ({ ExportEmployeesModal: () => null }));
vi.mock("@/components/modals/ImportMobilesModal", () => ({
  ImportMobilesModal: ({ open }: { open: boolean }) =>
    open ? <p>modal-import-mobiles</p> : null,
}));

describe("EmployeesPageMobile admin actions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("action sheet có 'Nhập SĐT'; mở đúng modal import", () => {
    render(
      <MemoryRouter>
        <EmployeesPageMobile />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Thao tác khác" }));
    fireEvent.click(screen.getByRole("button", { name: /Nhập SĐT/ }));
    expect(screen.getByText("modal-import-mobiles")).toBeTruthy();
  });
});
