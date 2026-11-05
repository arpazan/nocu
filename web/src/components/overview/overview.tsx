"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";

import { BackendHealth } from "@/components/backend-health";
import { SeverityBar, StatusDot, ToneDot } from "@/components/overview/marks";
import {
  attentionIncidents,
  attentionMonitors,
  situationSentence,
  summarize,
} from "@/components/overview/situation";
import { useNow } from "@/components/overview/use-now";
import { Skeleton } from "@/components/ui/skeleton";
import { incidentsQueryOptions } from "@/lib/api/incidents";
import { monitorsQueryOptions } from "@/lib/api/monitors";
import type { Incident, Monitor } from "@/lib/api/types";
import { formatRelative } from "@/lib/format";
import { incidentStatusLabel, severityLabel } from "@/lib/incidents/transitions";
import { useRealtimeConnection, useRealtimeEvents } from "@/lib/realtime/event-store";
import { describeEvent, type RealtimeEvent } from "@/lib/realtime/events";

/** How many feed rows the overview shows; the store keeps more for other screens. */
const FEED_LIMIT = 40;

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong";
}

function ConnectionLine() {
  const connection = useRealtimeConnection();
  const live = connection === "open";
  return (
    <p className="flex items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
      <span
        aria-hidden="true"
        className={
          live
            ? "size-1.5 rounded-full bg-tertiary motion-safe:animate-pulse motion-reduce:animate-none"
            : "size-1.5 rounded-full bg-primary"
        }
      />
      {live ? "Live" : "Reconnecting…"}
    </p>
  );
}

function Situation({ monitors, incidents }: { monitors: Monitor[]; incidents: Incident[] }) {
  const { lead, count, rest } = situationSentence(summarize(monitors, incidents));
  return (
    <h1 className="text-2xl font-semibold tracking-tight text-balance">
      {lead}
      {count && <span className="text-primary">{count}</span>}
      {rest}
    </h1>
  );
}

function NeedsAttention({ monitors, incidents }: { monitors: Monitor[]; incidents: Incident[] }) {
  const downMonitors = attentionMonitors(monitors);
  const openIncidents = attentionIncidents(incidents);

  if (downMonitors.length === 0 && openIncidents.length === 0) {
    return <p className="text-sm text-muted-foreground">Nothing needs attention right now.</p>;
  }

  return (
    <ul className="divide-y">
      {downMonitors.map((m) => (
        <li key={m.id}>
          <Link
            href={`/monitors/${encodeURIComponent(m.id)}`}
            className="flex items-start gap-3 rounded-md py-3 outline-none hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring"
          >
            <StatusDot status={m.last_status} className="mt-2" />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-medium">{m.name}</span>
              <span className="block truncate font-mono text-xs text-muted-foreground">{m.url}</span>
            </span>
            <span className="shrink-0 text-xs text-muted-foreground">
              {m.last_checked_at ? `down ${formatRelative(m.last_checked_at)}` : "down"}
            </span>
          </Link>
        </li>
      ))}
      {openIncidents.map((i) => (
        <li key={i.id}>
          <Link
            href={`/incidents/${encodeURIComponent(i.id)}`}
            className="flex items-stretch gap-3 rounded-md py-3 outline-none hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring"
          >
            <SeverityBar severity={i.severity} />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-medium">{i.title}</span>
              <span className="block text-xs text-muted-foreground">{severityLabel[i.severity]}</span>
            </span>
            <span className="shrink-0 self-center text-xs text-muted-foreground">{incidentStatusLabel[i.status]}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function LiveEvents({ events }: { events: readonly RealtimeEvent[] }) {
  const now = useNow();
  const visible: { event: RealtimeEvent; description: ReturnType<typeof describeEvent> }[] = [];
  for (const event of events) {
    const description = describeEvent(event);
    if (description.quiet) continue;
    visible.push({ event, description });
    if (visible.length >= FEED_LIMIT) break;
  }

  if (visible.length === 0) {
    return <p className="text-sm text-muted-foreground">Events appear here as they happen.</p>;
  }

  return (
    <ol className="divide-y">
      {visible.map(({ event, description }) => (
        <li key={event.id} className="flex items-start gap-3 py-3">
          <time
            dateTime={event.received_at}
            className="w-20 shrink-0 pt-0.5 font-mono text-xs text-muted-foreground"
          >
            {formatRelative(event.received_at, now)}
          </time>
          <ToneDot tone={description.tone} className="mt-2" />
          <span className="min-w-0 flex-1">
            <span className="block text-sm font-medium">{description.title}</span>
            {description.detail && (
              <span className="block truncate text-xs text-muted-foreground">{description.detail}</span>
            )}
          </span>
        </li>
      ))}
    </ol>
  );
}

function HeroSkeleton() {
  return (
    <div className="space-y-3" aria-busy="true" aria-label="Loading overview">
      <Skeleton className="h-8 w-2/3 max-w-md" />
      <Skeleton className="h-4 w-16" />
    </div>
  );
}

function ListSkeleton() {
  return (
    <div className="space-y-4 py-1">
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
      <Skeleton className="h-4 w-2/3" />
    </div>
  );
}

/** Overview: the live situation line, what needs attention, and the realtime feed. */
export function Overview() {
  const monitors = useQuery(monitorsQueryOptions({ limit: 200 }));
  const incidents = useQuery(incidentsQueryOptions({ limit: 200 }));
  const events = useRealtimeEvents();

  const loading = monitors.isPending || incidents.isPending;
  const error = monitors.error ?? incidents.error;
  const ready = monitors.isSuccess && incidents.isSuccess;

  return (
    <div className="space-y-10">
      <header className="space-y-2">
        {loading ? (
          <HeroSkeleton />
        ) : ready ? (
          <>
            <Situation monitors={monitors.data.items} incidents={incidents.data.items} />
            <ConnectionLine />
          </>
        ) : (
          <>
            <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
            <p className="text-sm text-primary" role="alert">
              Could not load the overview: {errorMessage(error)}
            </p>
          </>
        )}
      </header>

      <div className="grid gap-10 lg:grid-cols-2 lg:gap-12">
        <section aria-labelledby="attention-heading" className="min-w-0">
          <h2 id="attention-heading" className="mb-2 text-sm font-medium text-muted-foreground">
            Needs attention
          </h2>
          {loading ? (
            <ListSkeleton />
          ) : ready ? (
            <NeedsAttention monitors={monitors.data.items} incidents={incidents.data.items} />
          ) : null}
        </section>

        <section aria-labelledby="events-heading" className="min-w-0">
          <h2 id="events-heading" className="mb-2 text-sm font-medium text-muted-foreground">
            Live events
          </h2>
          <LiveEvents events={events} />
        </section>
      </div>

      <BackendHealth />
    </div>
  );
}
