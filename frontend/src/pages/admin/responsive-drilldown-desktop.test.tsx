import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ActivityUsersPage from "./DashboardPage/ActivityUsersPage";

const mocks = vi.hoisted(() => ({
  useIsMobile: vi.fn(() => false),
}));

vi.mock("@/hooks/useBreakpoint", () => ({
  useIsMobile: mocks.useIsMobile,
  useMediaQuery: () => true,
}));
vi.mock("@/components/shared/AdminPageFrame", () => ({
  AdminPageCanvas: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminPageHeaderCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminSectionCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminFilterRow: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/shared/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/employees/details/EmployeeDetailsSheet", () => ({
  EmployeeDetailsSheet: () => null,
}));
vi.mock("@/hooks/api/useDashboard", () => ({
  useActiveEmployeesBySchedule: () => ({
    data: [
      {
        user_id: 1,
        employee_id: 11,
        fullname: "Nguyễn Văn A",
        username: "nva",
        role: "nhanvien",
        payment_schedule: "weekly",
        last_login: "2026-10-28T10:00:00Z",
      },
    ],
    isLoading: false,
  }),
}));

describe("Drill-down desktop variants render dense tables (PAYR-23)", () => {
  const renderPage = (node: React.ReactNode) =>
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>{node}</MemoryRouter>
      </QueryClientProvider>,
    );

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useIsMobile.mockReturnValue(false);
  });

  it("ActivityUsersPage desktop render bảng", () => {
    renderPage(<ActivityUsersPage />);
    expect(screen.getByRole("table")).toBeTruthy();
    expect(screen.getByText("Nguyễn Văn A")).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Lịch lương" })).toBeTruthy();
  });
});
