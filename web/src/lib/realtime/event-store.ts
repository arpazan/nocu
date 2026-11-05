import { useSyncExternalStore } from "react";

import type { RealtimeEvent } from "@/lib/realtime/events";

/**
 * In-memory feed of the most recent realtime events plus the socket state.
 * Same shape as session-store: a module singleton read through
 * useSyncExternalStore, so every subscriber sees one consistent list.
 */

export type ConnectionState = "connecting" | "open" | "closed";

/** How many events the feed keeps; older ones fall off the end. */
export const MAX_EVENTS = 100;

const EMPTY: readonly RealtimeEvent[] = Object.freeze([]);

let events: readonly RealtimeEvent[] = EMPTY;
let connection: ConnectionState = "closed";
const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

export const eventStore = {
  getEvents: () => events,
  getConnection: () => connection,
  /** Prepends an event (newest first) and trims to MAX_EVENTS. */
  push(event: RealtimeEvent) {
    events = [event, ...events].slice(0, MAX_EVENTS);
    emit();
  },
  setConnection(next: ConnectionState) {
    if (connection === next) return;
    connection = next;
    emit();
  },
  clear() {
    events = EMPTY;
    emit();
  },
  subscribe(listener: () => void) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};

/** Newest-first list of recent events. */
export function useRealtimeEvents(): readonly RealtimeEvent[] {
  return useSyncExternalStore(eventStore.subscribe, eventStore.getEvents, () => EMPTY);
}

/** Current WebSocket state. */
export function useRealtimeConnection(): ConnectionState {
  return useSyncExternalStore(eventStore.subscribe, eventStore.getConnection, () => "closed");
}
