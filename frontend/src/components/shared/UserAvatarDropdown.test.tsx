import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { authService } from "@/services/api/auth.service";
import { UserAvatarDropdown } from "./UserAvatarDropdown";

vi.mock("@/services/api/auth.service", () => ({
  authService: {
    logout: vi.fn(),
    clearLocalSession: vi.fn(),
  },
}));

vi.mock("@/contexts", () => ({
  useAuth: () => ({
    user: {
      id: "1",
      email: "cuongnv@example.com",
      name: "Nguyễn Văn Cường",
      username: "cuongnv",
      role: "admin" as const,
    },
    logout: vi.fn(),
  }),
}));

vi.mock("@/hooks/useModalNavigation", () => ({
  useModalNavigation: () => ({ openModal: vi.fn() }),
}));

vi.mock("@/components/sheets/UserProfileSheet", () => ({
  UserProfileSheet: () => null,
}));

const renderDropdown = () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <UserAvatarDropdown />
      </QueryClientProvider>
    </MemoryRouter>
  );
};

describe("UserAvatarDropdown", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("revokes the server session when logging out from the account menu", async () => {
    vi.mocked(authService.logout).mockResolvedValue({} as never);
    renderDropdown();

    fireEvent.pointerDown(screen.getByRole("button", { name: "Menu người dùng" }), { button: 0 });
    fireEvent.click(await screen.findByText("Đăng xuất"));

    // The logout must reach the server (POST /auth/logout) so the token is
    // blacklisted — clearing local state alone leaves the session alive.
    await waitFor(() => expect(authService.logout).toHaveBeenCalledTimes(1));
  });
});
