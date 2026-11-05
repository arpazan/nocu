import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LoginForm } from "@/components/login-form";
import { sessionStore } from "@/lib/auth/session-store";

const push = vi.fn();
let search = "";
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
  useSearchParams: () => new URLSearchParams(search),
}));

function renderForm() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LoginForm />
    </QueryClientProvider>,
  );
}

describe("LoginForm", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    push.mockClear();
    search = "";
    sessionStore.clear();
  });

  it("shows zod validation messages without calling the API", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    renderForm();

    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect(await screen.findByText("Enter a valid email address")).toBeInTheDocument();
    expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("stores the session and redirects on success", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ data: { access_token: "tok", token_type: "Bearer", expires_in: 900 } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    renderForm();

    await userEvent.type(screen.getByLabelText("Email"), "ops@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "correct horse");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/"));
    expect(sessionStore.get()?.access_token).toBe("tok");
  });

  it("maps a 401 from the backend to a root form error", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "unauthorized", message: "invalid credentials" } }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    );
    renderForm();

    await userEvent.type(screen.getByLabelText("Email"), "ops@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "wrong password");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect(await screen.findByText("Email or password is incorrect")).toBeInTheDocument();
  });

  it("redirects to the same-origin `next` path after signing in", async () => {
    search = "next=%2Fincidents%3Fstatus%3Dopen";
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ data: { access_token: "tok", token_type: "Bearer", expires_in: 900 } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    renderForm();

    await userEvent.type(screen.getByLabelText("Email"), "ops@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "correct horse");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/incidents?status=open"));
  });

  it("ignores an off-site `next` and links to registration", () => {
    search = "next=https%3A%2F%2Fevil.example";
    renderForm();

    expect(screen.getByRole("link", { name: "Create an account" })).toHaveAttribute("href", "/register");
  });
});
