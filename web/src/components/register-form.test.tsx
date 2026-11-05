import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { RegisterForm } from "@/components/register-form";

const push = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

function renderForm() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RegisterForm />
    </QueryClientProvider>,
  );
}

async function fillForm(name?: string) {
  await userEvent.type(screen.getByLabelText("Email"), "ops@example.com");
  await userEvent.type(screen.getByLabelText("Password"), "correct horse");
  if (name) await userEvent.type(screen.getByLabelText("Name (optional)"), name);
}

describe("RegisterForm", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    push.mockClear();
  });

  it("shows zod validation messages without calling the API", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    renderForm();

    await userEvent.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("Enter a valid email address")).toBeInTheDocument();
    expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("posts to identity and redirects to /login on success", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          data: { id: "u1", email: "ops@example.com", name: "Ops", is_active: true, roles: ["viewer"] },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      ),
    );
    renderForm();

    await fillForm("Ops");
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/login"));
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/auth\/register$/);
    expect(init?.body).toBe(JSON.stringify({ email: "ops@example.com", password: "correct horse", name: "Ops" }));
  });

  it("maps a 409 from identity to an email field error", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "conflict", message: "email already registered" } }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    renderForm();

    await fillForm();
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("That email is already registered")).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
  });

  it("maps a 400 from identity to a root error with the API message", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "bad_request", message: "invalid email" } }), {
        status: 400,
        headers: { "Content-Type": "application/json" },
      }),
    );
    renderForm();

    await fillForm();
    await userEvent.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("invalid email")).toBeInTheDocument();
  });

  it("links to the sign-in page", () => {
    renderForm();
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute("href", "/login");
  });
});
