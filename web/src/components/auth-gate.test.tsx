import { act, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AuthGate } from "@/components/auth-gate";
import { refreshSession } from "@/lib/auth/refresh";
import { sessionStore } from "@/lib/auth/session-store";

const replace = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
  usePathname: () => "/incidents",
  useSearchParams: () => new URLSearchParams("status=open"),
}));

vi.mock("@/lib/auth/refresh", () => ({ refreshSession: vi.fn() }));

const refresh = vi.mocked(refreshSession);
const session = { access_token: "tok", token_type: "Bearer" as const, expires_in: 900 };

describe("AuthGate", () => {
  afterEach(() => {
    sessionStore.clear();
    vi.clearAllMocks();
  });

  it("renders children immediately when a session is already held", () => {
    sessionStore.set(session);
    render(
      <AuthGate>
        <p>secret</p>
      </AuthGate>,
    );

    expect(screen.getByText("secret")).toBeInTheDocument();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("shows a busy skeleton while refreshing, then the children", async () => {
    let settle!: () => void;
    refresh.mockReturnValue(
      new Promise((resolve) => {
        settle = () => {
          sessionStore.set(session);
          resolve(session);
        };
      }),
    );
    render(
      <AuthGate>
        <p>secret</p>
      </AuthGate>,
    );

    expect(screen.getByLabelText("Restoring session")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("secret")).not.toBeInTheDocument();

    await act(async () => settle());
    expect(await screen.findByText("secret")).toBeInTheDocument();
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(replace).not.toHaveBeenCalled();
  });

  it("redirects to /login with the current location when the refresh fails", async () => {
    refresh.mockResolvedValue(null);
    render(
      <AuthGate>
        <p>secret</p>
      </AuthGate>,
    );

    await waitFor(() =>
      expect(replace).toHaveBeenCalledWith(`/login?next=${encodeURIComponent("/incidents?status=open")}`),
    );
    expect(screen.queryByText("secret")).not.toBeInTheDocument();
  });
});
