import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CheckInToggle } from "./CheckInToggle";
import type { ProjectEmployeeAssignment } from "@/types/api/project-employee.types";

const mocks = vi.hoisted(() => ({
  toggleMutate: vi.fn(),
  cancelMutate: vi.fn(),
}));

vi.mock("@/hooks/api/useProjectEmployees", () => ({
  useToggleCheckInEnabled: () => ({ mutate: mocks.toggleMutate, isPending: false }),
  useCancelPendingCheckInEnable: () => ({ mutate: mocks.cancelMutate, isPending: false }),
}));

const baseAssignment: ProjectEmployeeAssignment = {
  id: 1,
  project_id: 7,
  employee_id: 101,
  employee_name: "Nguyễn Văn A",
  employee_cccd: "001001001001",
  employee_code: "nva",
  position: "Thợ",
  check_in_enabled: false,
  start_date: "2026-01-01",
  status: "current",
  created_by: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("CheckInToggle", () => {
  beforeEach(() => {
    mocks.toggleMutate.mockClear();
    mocks.cancelMutate.mockClear();
  });

  it("enables the service immediately when switched on", () => {
    render(<CheckInToggle assignment={{ ...baseAssignment, check_in_enabled: false }} />);

    fireEvent.click(screen.getByRole("switch"));

    expect(mocks.toggleMutate).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 101,
      enabled: true,
    });
    expect(mocks.cancelMutate).not.toHaveBeenCalled();
  });

  it("still turns the service off immediately", () => {
    render(<CheckInToggle assignment={{ ...baseAssignment, check_in_enabled: true }} />);

    fireEvent.click(screen.getByRole("switch"));

    expect(mocks.toggleMutate).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 101,
      enabled: false,
    });
  });

  it("shows the pending activation badge and cancels it", () => {
    render(
      <CheckInToggle
        assignment={{
          ...baseAssignment,
          check_in_enabled: false,
          pending_check_in_enabled: true,
          check_in_effective_from: "2026-11-01",
        }}
      />,
    );

    expect(screen.getByText(/Kích hoạt 01\/11/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Hủy" }));

    expect(mocks.cancelMutate).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 101,
    });
    expect(mocks.toggleMutate).not.toHaveBeenCalled();
  });
});
