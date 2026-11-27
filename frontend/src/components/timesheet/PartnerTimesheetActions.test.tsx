import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PartnerTimesheetActions } from "./PartnerTimesheetActions";

const mocks = vi.hoisted(() => ({
  useAuth: vi.fn(),
}));

vi.mock("@/contexts", () => ({
  useAuth: mocks.useAuth,
}));

describe("PartnerTimesheetActions", () => {
  it("keeps every timesheet action visible and connected for quanly", () => {
    mocks.useAuth.mockReturnValue({ user: { role: "quanly" } });
    const callbacks = {
      onAddTimesheet: vi.fn(),
      onOpenBccHistory: vi.fn(),
      onUploadBcc: vi.fn(),
    };

    render(<PartnerTimesheetActions {...callbacks} />);

    const actions = [
      ["Nhập công", callbacks.onAddTimesheet],
      ["Lịch sử BCC", callbacks.onOpenBccHistory],
      ["Tải lên BCC", callbacks.onUploadBcc],
    ] as const;

    actions.forEach(([label, callback]) => {
      fireEvent.click(screen.getByRole("button", { name: new RegExp(label) }));
      expect(callback).toHaveBeenCalledOnce();
    });
  });

  it("renders nothing for read-only roles (ketoan/giamdoc hold no timesheet writes)", () => {
    mocks.useAuth.mockReturnValue({ user: { role: "ketoan" } });
    const { container } = render(
      <PartnerTimesheetActions onAddTimesheet={vi.fn()} onOpenBccHistory={vi.fn()} onUploadBcc={vi.fn()} />
    );
    expect(container).toBeEmptyDOMElement();
  });
});
