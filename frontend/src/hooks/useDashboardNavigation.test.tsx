import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useDashboardNavigation } from "./useDashboardNavigation";

const navigate = vi.fn();

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigate,
}));

describe("useDashboardNavigation", () => {
  beforeEach(() => {
    navigate.mockReset();
  });

  it("routes the pending-approval queue to the timesheet filter contract", () => {
    const { result } = renderHook(() => useDashboardNavigation());

    act(() => result.current.navigateToPendingApprovals());

    expect(navigate).toHaveBeenCalledWith("/admin/timesheet?status=pending_approval");
  });

  it("routes active-employee drill-down to the working status filter", () => {
    const { result } = renderHook(() => useDashboardNavigation());

    act(() => result.current.navigateToActiveEmployees());

    expect(navigate).toHaveBeenCalledWith("/admin/employees?status=working");
  });
});
