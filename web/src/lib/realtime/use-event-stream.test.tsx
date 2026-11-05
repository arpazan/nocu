import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { incidentKeys } from "@/lib/api/incidents";
import { monitorKeys } from "@/lib/api/monitors";
import { sessionStore } from "@/lib/auth/session-store";
import { eventStore } from "@/lib/realtime/event-store";
import { useEventStream } from "@/lib/realtime/use-event-stream";

const refreshSession = vi.hoisted(() => vi.fn<() => Promise<null>>(() => Promise.resolve(null)));
vi.mock("@/lib/auth/refresh", () => ({ refreshSession }));

const toast = vi.hoisted(() => {
  const fn = vi.fn() as ReturnType<typeof vi.fn> & Record<string, ReturnType<typeof vi.fn>>;
  for (const k of ["success", "error", "info", "warning"]) fn[k] = vi.fn();
  return fn;
});
vi.mock("sonner", () => ({ toast }));

/** Minimal stand-in for the browser WebSocket: records instances, lets tests drive events. */
class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: FakeWebSocket[] = [];

  readyState = FakeWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  onclose: ((ev: { code: number }) => void) | null = null;
  onerror: (() => void) | null = null;
  close = vi.fn((code?: number) => {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code: code ?? 1000 });
  });

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  message(frame: unknown) {
    this.onmessage?.({ data: typeof frame === "string" ? frame : JSON.stringify(frame) });
  }

  serverClose(code: number) {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code });
  }
}

function setup() {
  const client = new QueryClient();
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const hook = renderHook(() => useEventStream(), { wrapper });
  return { ...hook, invalidate };
}

function latest(): FakeWebSocket {
  return FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
}

describe("useEventStream", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal("WebSocket", FakeWebSocket);
    FakeWebSocket.instances = [];
    sessionStore.set({ access_token: "tok-1", token_type: "Bearer", expires_in: 900 });
  });

  afterEach(() => {
    sessionStore.clear();
    eventStore.clear();
    vi.unstubAllGlobals();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("does nothing without a session", () => {
    sessionStore.clear();
    setup();
    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(eventStore.getConnection()).toBe("closed");
  });

  it("connects with the access token and reports the connection state", () => {
    setup();
    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(latest().url).toBe("ws://localhost:8080/api/v1/ws?token=tok-1");
    expect(eventStore.getConnection()).toBe("connecting");

    act(() => latest().open());
    expect(eventStore.getConnection()).toBe("open");
  });

  it("pushes events into the store and invalidates by subject", () => {
    const { invalidate } = setup();
    act(() => latest().open());

    act(() => {
      latest().message({ type: "hello", user_id: "u1", at: "now" });
      latest().message({
        type: "event",
        subject: "incident.updated",
        payload: {
          type: "incident.updated",
          occurred_at: "2026-11-05T10:00:00Z",
          source: "core-api",
          data: { id: "i1", title: "Checkout", severity: "high", status: "acknowledged" },
        },
      });
      latest().message("garbage");
    });

    const events = eventStore.getEvents();
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ subject: "incident.updated", source: "core-api" });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: incidentKeys.all });
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("toasts and invalidates the monitor detail on monitor.down", () => {
    const { invalidate } = setup();
    act(() => latest().open());

    act(() => {
      latest().message({
        type: "event",
        subject: "monitor.down",
        payload: {
          type: "monitor.down",
          occurred_at: "t",
          source: "scheduler",
          data: { monitor_id: "m1", name: "Blog", url: "https://blog", status: "down", message: "timeout", http_status: 0, latency_ms: 0, checked_at: "t" },
        },
      });
    });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: monitorKeys.all });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: monitorKeys.detail("m1") });
    expect(toast.error).toHaveBeenCalledWith("Blog is down", { description: "timeout" });
  });

  it("ignores scheduler jobs for toasts but keeps them in the store", () => {
    setup();
    act(() => latest().open());
    act(() => {
      latest().message({ type: "event", subject: "job.completed", payload: { type: "job.completed", occurred_at: "t", source: "scheduler", data: {} } });
    });
    expect(eventStore.getEvents()).toHaveLength(1);
    expect(toast).not.toHaveBeenCalled();
    expect(toast.info).not.toHaveBeenCalled();
  });

  it("schedules a reconnect with backoff after an unexpected close", async () => {
    setup();
    act(() => latest().open());
    // Past the early-close window so this is treated as a plain drop, not an auth failure.
    act(() => vi.advanceTimersByTime(5_000));
    act(() => latest().serverClose(1006));

    expect(eventStore.getConnection()).toBe("closed");
    expect(FakeWebSocket.instances).toHaveLength(1);
    expect(refreshSession).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(999));
    expect(FakeWebSocket.instances).toHaveLength(1);
    act(() => vi.advanceTimersByTime(1));
    expect(FakeWebSocket.instances).toHaveLength(2);
    expect(eventStore.getConnection()).toBe("connecting");

    // A second drop before the socket ever opens looks like a bad token:
    // refresh once, then wait twice as long.
    act(() => latest().serverClose(1006));
    expect(refreshSession).toHaveBeenCalledTimes(1);
    await act(async () => {
      await Promise.resolve();
    });
    act(() => vi.advanceTimersByTime(1_999));
    expect(FakeWebSocket.instances).toHaveLength(2);
    act(() => vi.advanceTimersByTime(1));
    expect(FakeWebSocket.instances).toHaveLength(3);
  });

  it("resets the backoff once a connection opens", () => {
    setup();
    act(() => latest().open());
    act(() => vi.advanceTimersByTime(5_000));
    act(() => latest().serverClose(1006));
    act(() => vi.advanceTimersByTime(1_000));
    expect(FakeWebSocket.instances).toHaveLength(2);

    act(() => latest().open());
    act(() => vi.advanceTimersByTime(5_000));
    act(() => latest().serverClose(1006));
    act(() => vi.advanceTimersByTime(1_000));
    expect(FakeWebSocket.instances).toHaveLength(3);
  });

  it("refreshes the session before reconnecting on an auth close", async () => {
    setup();
    act(() => latest().open());
    act(() => latest().serverClose(4401));

    expect(refreshSession).toHaveBeenCalledTimes(1);
    // Refresh resolved without changing the token: fall back to a backoff reconnect.
    await act(async () => {
      await Promise.resolve();
    });
    expect(FakeWebSocket.instances).toHaveLength(1);
    act(() => vi.advanceTimersByTime(1_000));
    expect(FakeWebSocket.instances).toHaveLength(2);
  });

  it("reconnects with the new token when the session changes", () => {
    setup();
    const first = latest();
    act(() => first.open());

    act(() => sessionStore.set({ access_token: "tok-2", token_type: "Bearer", expires_in: 900 }));

    expect(first.close).toHaveBeenCalled();
    expect(FakeWebSocket.instances).toHaveLength(2);
    expect(latest().url).toContain("token=tok-2");
  });

  it("closes the socket and cancels timers on unmount", () => {
    const { unmount } = setup();
    act(() => latest().open());
    const socket = latest();
    unmount();

    expect(socket.close).toHaveBeenCalledWith(1000, expect.any(String));
    expect(eventStore.getConnection()).toBe("closed");
    act(() => vi.advanceTimersByTime(60_000));
    expect(FakeWebSocket.instances).toHaveLength(1);
  });
});
