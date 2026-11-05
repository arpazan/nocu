/** Display helpers for timestamps, durations and latencies. */

const EMPTY = "—"; // em dash

const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

const relative = new Intl.RelativeTimeFormat("en", { numeric: "always", style: "short" });

function parse(iso: string | null | undefined): Date | null {
  if (!iso) return null;
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? null : date;
}

/** "5 Nov 2026, 14:03" in the viewer's time zone; an em dash for empty or invalid input. */
export function formatDateTime(iso: string | null | undefined): string {
  const date = parse(iso);
  return date ? dateTime.format(date) : EMPTY;
}

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 24 * 60 * 60],
  ["month", 30 * 24 * 60 * 60],
  ["day", 24 * 60 * 60],
  ["hour", 60 * 60],
  ["minute", 60],
];

/**
 * "3 min ago" / "in 2 min" relative to `now`; "just now" within 30 seconds,
 * an em dash for empty or invalid input.
 */
export function formatRelative(iso: string | null | undefined, now: Date | number = Date.now()): string {
  const date = parse(iso);
  if (!date) return EMPTY;
  const seconds = Math.round((date.getTime() - Number(now)) / 1000);
  if (Math.abs(seconds) < 30) return "just now";
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) {
      // ICU short style abbreviates with a trailing dot ("3 min. ago"); drop it.
      return relative.format(Math.trunc(seconds / size), unit).replace(/\.(?=\s|$)/g, "");
    }
  }
  return relative.format(seconds, "second").replace(/\.(?=\s|$)/g, "");
}

/** "812 ms" below one second, otherwise "1.2 s". */
export function formatLatency(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || !Number.isFinite(ms)) return EMPTY;
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(1)} s`;
}

/** "60 s" for sub-two-minute or uneven values, "2 min" for whole minutes, "1 h" for whole hours. */
export function formatInterval(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined || !Number.isFinite(seconds)) return EMPTY;
  if (seconds >= 3600 && seconds % 3600 === 0) return `${seconds / 3600} h`;
  if (seconds >= 120 && seconds % 60 === 0) return `${seconds / 60} min`;
  return `${seconds} s`;
}
