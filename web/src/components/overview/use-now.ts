import { useSyncExternalStore } from "react";

/** Relative timestamps only need to move at this granularity. */
const TICK_MS = 30_000;

const listeners = new Set<() => void>();
let timer: ReturnType<typeof setInterval> | null = null;

function subscribe(listener: () => void) {
  listeners.add(listener);
  if (timer === null) {
    timer = setInterval(() => {
      for (const l of listeners) l();
    }, TICK_MS);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && timer !== null) {
      clearInterval(timer);
      timer = null;
    }
  };
}

// Quantised so the snapshot is stable between ticks (useSyncExternalStore
// requires getSnapshot to return the same value until the store changes).
function getSnapshot(): number {
  return Math.floor(Date.now() / TICK_MS) * TICK_MS;
}

/** Current time in ms, refreshed every 30 seconds while mounted. */
export function useNow(): number {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
