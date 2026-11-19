import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EmployeeRouter from "./index";

const profileQuery = vi.hoisted(() => ({
  data: undefined as { payment_schedule: string } | undefined,
  isLoading: false,
  isError: false,
  isFetching: false,
  refetch: vi.fn(),
}));

vi.mock("@/hooks/api/useEmployeePortal", () => ({ useEmployeeProfile: () => profileQuery }));
vi.mock("@/pages/employee/EmployeePage", () => ({ default: () => <p>Trang bảng công</p> }));

describe("EmployeeRouter", () => {
  beforeEach(() => {
    profileQuery.data = undefined;
    profileQuery.isLoading = false;
    profileQuery.isError = false;
    profileQuery.isFetching = false;
    profileQuery.refetch.mockReset();
  });

  it("renders non-interactive shell chrome while loading", () => {
    profileQuery.isLoading = true;
    render(<EmployeeRouter />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Đang tải thông tin nhân viên")).toBeInTheDocument();
  });

  it("guards retry while a profile refetch is already running", () => {
    profileQuery.isError = true;
    profileQuery.isFetching = true;
    render(<EmployeeRouter />);

    const retryButton = screen.getByRole("button", { name: "Đang tải lại…" });
    expect(retryButton).toBeDisabled();
    fireEvent.click(retryButton);
    expect(profileQuery.refetch).not.toHaveBeenCalled();
  });

  it("keeps rendering the timesheet page after a background refetch fails", () => {
    profileQuery.data = { payment_schedule: "flexible" };
    profileQuery.isError = true;
    render(<EmployeeRouter />);

    expect(screen.getByText("Trang bảng công")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Chưa tải được hồ sơ" })).not.toBeInTheDocument();
  });
});
