import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { toast } from "sonner";

import { incidentKeys } from "@/lib/api/incidents";
import { monitorKeys } from "@/lib/api/monitors";
import { refreshSession } from "@/lib/auth/refresh";
import { getAccessToken, useSession } from "@/lib/auth/session-store";
import { env } from "@/lib/env";
import { eventStore } from "@/lib/realtime/event-store";
import {
  describeEvent,
  isIncidentEvent,
  isMonitorStatusEvent,
  parseFrame,
  toRealtimeEvent,
  type RealtimeEvent,
} from "@/lib/realtime/events";

/** Reconnect delay bounds: 1s doubling per attempt, capped at 30s, reset on a successful open. */
export const BACKOFF_MIN_MS = 1000;
export const BACKOFF_MAX_MS = 30_000;

/** A close this soon after open (or before any open) is treated as an auth failure. */
const EARLY_CLOSE_MS = 1000;
/** Do not hit the refresh endpoint more often than this from the socket loop. */
const REFRESH_COOLDOWN_MS = 60_000;

/** Close codes core-api uses for a rejected or expired token. */
const AUTH_CLOSE_CODES = new Set([1008, 4401]);

export function backoffDelay(attempt: number): number {
  return Math.min(BACKOFF_MAX_MS, BACKOFF_MIN_MS * 2 ** attempt);
}

/** Builds the socket URL with the access token as the query param core-api expects. */
export function socketUrl(token: string): string {
  const url = new URL(env.wsUrl);
  url.searchParams.set("token", token);
  return url.toString();
}

function notify(event: RealtimeEvent) {
  if (isMonitorStatusEvent(event)) {
    const { title, detail } = describeEvent(event);
    if (event.subject === "monitor.down") toast.error(title, { description: detail });
    else toast.success(title, { description: detail });
    return;
  }
  if (event.subject === "incident.created" && isIncidentEvent(event)) {
    const { title, detail } = describeEvent(event);
    const severe = event.data.severity === "critical" || event.data.severity === "high";
    (severe ? toast.warning : toast.info)(title, { description: detail });
  }
}

/**
 * Keeps one WebSocket to core-api open for as long as there is a session.
 *
 * Every event lands in the event store (for the feed) and invalidates the
 * TanStack queries it affects, so lists and details re-fetch themselves.
 * The effect is keyed on the access token: a refresh replaces the token,
 * which tears this socket down and opens a new one with the fresh token.
 */
export function useEventStream(): void {
  const token = useSession()?.access_token ?? null;
  const queryClient = useQueryClient();

  const socketRef = useRef<WebSocket | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Survive re-keys so a refresh-induced reconnect does not restart the backoff from zero.
  const attemptRef = useRef(0);
  const refreshedAtRef = useRef(0);

  useEffect(() => {
    if (typeof window === "undefined" || typeof WebSocket === "undefined" || !token) {
      eventStore.setConnection("closed");
      return;
    }

    let disposed = false;
    let openedAt = 0;

    const clearTimer = () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };

    const handleEvent = (event: RealtimeEvent) => {
      eventStore.push(event);

      if (event.subject.startsWith("incident.")) {
        void queryClient.invalidateQueries({ queryKey: incidentKeys.all });
      } else if (event.subject.startsWith("monitor.")) {
        void queryClient.invalidateQueries({ queryKey: monitorKeys.all });
        if (isMonitorStatusEvent(event)) {
          // monitorKeys.all already covers these; being explicit keeps the intent visible.
          void queryClient.invalidateQueries({ queryKey: monitorKeys.detail(event.data.monitor_id) });
        }
      }

      notify(event);
    };

    const scheduleReconnect = () => {
      if (disposed) return;
      clearTimer();
      const delay = backoffDelay(attemptRef.current);
      attemptRef.current += 1;
      timerRef.current = setTimeout(() => {
        timerRef.current = null;
        connect();
      }, delay);
    };

    const handleClose = (code: number) => {
      socketRef.current = null;
      eventStore.setConnection("closed");
      if (disposed) return;

      const now = Date.now();
      const early = openedAt === 0 || now - openedAt < EARLY_CLOSE_MS;
      const authSuspect = AUTH_CLOSE_CODES.has(code) || early;
      const canRefresh = now - refreshedAtRef.current > REFRESH_COOLDOWN_MS;

      if (!authSuspect || !canRefresh) {
        scheduleReconnect();
        return;
      }

      // The token may have expired: get a new one before trying again, so an
      // expired access token does not turn into a reject/reconnect loop.
      refreshedAtRef.current = now;
      refreshSession()
        .then(() => {
          if (disposed) return;
          // A changed (or cleared) token re-keys this effect, which reconnects on its own.
          if (getAccessToken() !== token) return;
          scheduleReconnect();
        })
        .catch(() => scheduleReconnect());
    };

    const connect = () => {
      if (disposed) return;
      openedAt = 0;
      eventStore.setConnection("connecting");

      let socket: WebSocket;
      try {
        socket = new WebSocket(socketUrl(token));
      } catch {
        scheduleReconnect();
        return;
      }
      socketRef.current = socket;

      socket.onopen = () => {
        if (disposed) return;
        openedAt = Date.now();
        attemptRef.current = 0;
        eventStore.setConnection("open");
      };

      socket.onmessage = (message: MessageEvent) => {
        if (disposed || typeof message.data !== "string") return;
        const frame = parseFrame(message.data);
        if (!frame || frame.type !== "event") return;
        handleEvent(toRealtimeEvent(frame));
      };

      socket.onclose = (ev: CloseEvent) => {
        if (socketRef.current !== socket) return;
        handleClose(ev.code);
      };

      // onerror is always followed by onclose; nothing to do here.
      socket.onerror = () => {};
    };

    connect();

    return () => {
      disposed = true;
      clearTimer();
      const socket = socketRef.current;
      socketRef.current = null;
      if (socket) {
        socket.onopen = null;
        socket.onmessage = null;
        socket.onclose = null;
        socket.onerror = null;
        if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
          socket.close(1000, "client navigated or signed out");
        }
      }
      eventStore.setConnection("closed");
    };
  }, [token, queryClient]);
}
