"use client";

import type { ReactNode } from "react";

import { useEventStream } from "@/lib/realtime/use-event-stream";

/**
 * Mounts the realtime WebSocket for the signed-in area. Rendered inside
 * AuthGate so it only runs once a session exists; it renders no DOM of its own.
 */
export function RealtimeProvider({ children }: { children: ReactNode }) {
  useEventStream();
  return children;
}
