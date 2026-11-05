"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

import { IncidentFormDialog, OPERATOR_ROLE_MESSAGE } from "@/components/incidents/incident-form-dialog";
import { NeedsOperator } from "@/components/incidents/needs-operator";
import { SeverityMark } from "@/components/incidents/severity-badge";
import { IncidentStatusBadge } from "@/components/incidents/status-badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError } from "@/lib/api/client";
import { deleteIncident, incidentKeys, incidentQueryOptions, updateIncident } from "@/lib/api/incidents";
import type { Incident, IncidentStatus } from "@/lib/api/types";
import { canDelete, canWrite } from "@/lib/auth/roles";
import { useRoles } from "@/lib/auth/use-me";
import { formatDateTime } from "@/lib/format";
import { nextStatuses, transitionVerb } from "@/lib/incidents/transitions";
import { cn } from "@/lib/utils";

/** Toast after a successful status change. */
const transitionDone: Record<IncidentStatus, string> = {
  open: "Reopened",
  acknowledged: "Acknowledged",
  resolved: "Resolved",
  closed: "Closed",
};

export function IncidentDetail({ id }: { id: string }) {
  const incident = useQuery(incidentQueryOptions(id));

  if (incident.isPending) return <DetailSkeleton />;

  if (incident.isError) {
    const error = incident.error;
    if (error instanceof ApiError && error.status === 404) {
      return (
        <Panel title="Incident not found" message="This incident does not exist or was deleted.">
          <Button asChild variant="outline">
            <Link href="/incidents">Back to incidents</Link>
          </Button>
        </Panel>
      );
    }
    return (
      <Panel title="Could not load this incident" message={error.message}>
        <Button variant="outline" onClick={() => void incident.refetch()}>
          Try again
        </Button>
        <Button asChild variant="ghost">
          <Link href="/incidents">Back to incidents</Link>
        </Button>
      </Panel>
    );
  }

  return <LoadedIncident incident={incident.data} />;
}

function LoadedIncident({ incident }: { incident: Incident }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const roles = useRoles();
  const allowed = canWrite(roles);
  const [editing, setEditing] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  const transition = useMutation({
    mutationFn: (status: IncidentStatus) => updateIncident(incident.id, { status }),
    onSuccess: (_, status) => {
      toast.success(transitionDone[status]);
      void queryClient.invalidateQueries({ queryKey: incidentKeys.all });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) {
        toast.error("That change is no longer allowed; the incident was updated elsewhere.");
        void queryClient.invalidateQueries({ queryKey: incidentKeys.detail(incident.id) });
        return;
      }
      if (err instanceof ApiError && err.status === 403) {
        toast.error(OPERATOR_ROLE_MESSAGE);
        return;
      }
      toast.error(err instanceof Error ? err.message : "Could not update the incident");
    },
  });

  const remove = useMutation({
    mutationFn: () => deleteIncident(incident.id),
    onSuccess: () => {
      toast.success("Incident deleted");
      void queryClient.invalidateQueries({ queryKey: incidentKeys.all });
      router.push("/incidents");
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Could not delete the incident");
    },
  });

  const targets = nextStatuses(incident.status);
  const forward = targets.find((t) => t !== "open" && t !== "closed");
  const busy = transition.isPending || remove.isPending;

  const timeline: { label: string; at: string | null | undefined }[] = [
    { label: "Opened", at: incident.created_at },
    { label: "Acknowledged", at: incident.acknowledged_at },
    { label: "Resolved", at: incident.resolved_at },
    { label: "Closed", at: incident.closed_at },
  ];

  return (
    <div className="space-y-8">
      <div className="space-y-4">
        <Link
          href="/incidents"
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
        >
          <ArrowLeftIcon className="size-4" aria-hidden="true" />
          Incidents
        </Link>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold tracking-tight text-balance">{incident.title}</h1>
          <div className="flex flex-wrap items-center gap-3">
            <SeverityMark severity={incident.severity} />
            <IncidentStatusBadge status={incident.status} />
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {targets.map((target) => (
            <NeedsOperator key={target} allowed={allowed}>
              <Button
                variant={target === forward ? "default" : "outline"}
                disabled={!allowed || busy}
                onClick={() => transition.mutate(target)}
              >
                {transitionVerb[target]}
              </Button>
            </NeedsOperator>
          ))}
          <NeedsOperator allowed={allowed}>
            <Button variant="outline" disabled={!allowed || busy} onClick={() => setEditing(true)}>
              Edit
            </Button>
          </NeedsOperator>
          {canDelete(roles) && (
            <Button variant="destructive" disabled={busy} onClick={() => setConfirmingDelete(true)}>
              Delete
            </Button>
          )}
        </div>
      </div>

      {incident.description ? (
        <p className="max-w-prose whitespace-pre-line">{incident.description}</p>
      ) : (
        <p className="text-muted-foreground">No description</p>
      )}

      <div className="grid gap-8 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <section aria-labelledby="timeline-heading" className="space-y-4">
          <h2 id="timeline-heading" className="text-sm font-medium">
            Timeline
          </h2>
          <ol className="relative ml-1 space-y-5 border-l pl-5">
            {timeline.map(({ label, at }) => (
              <li key={label} className={cn("relative", !at && "text-muted-foreground")}>
                <span
                  aria-hidden="true"
                  className={cn(
                    "absolute top-1.5 -left-[25px] size-2 rounded-full ring-4 ring-background",
                    at ? "bg-foreground" : "bg-border",
                  )}
                />
                <p className="text-sm font-medium">{label}</p>
                <p className="text-sm text-muted-foreground">{at ? formatDateTime(at) : "—"}</p>
              </li>
            ))}
          </ol>
        </section>

        <section aria-labelledby="details-heading" className="space-y-4">
          <h2 id="details-heading" className="text-sm font-medium">
            Details
          </h2>
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-6 gap-y-3 text-sm">
            <dt className="text-muted-foreground">Source</dt>
            <dd className="font-mono text-xs leading-5">{incident.source || "—"}</dd>
            <dt className="text-muted-foreground">Reporter</dt>
            <dd className="truncate font-mono text-xs leading-5" title={incident.reporter_id ?? undefined}>
              {incident.reporter_id || "—"}
            </dd>
            <dt className="text-muted-foreground">Updated</dt>
            <dd>{formatDateTime(incident.updated_at)}</dd>
          </dl>
        </section>
      </div>

      <IncidentFormDialog open={editing} onOpenChange={setEditing} incident={incident} />

      <AlertDialog open={confirmingDelete} onOpenChange={setConfirmingDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this incident?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes it from every list. The event feed still records that it existed.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => remove.mutate()}>
              Delete incident
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function Panel({ title, message, children }: { title: string; message: string; children: React.ReactNode }) {
  return (
    <div role="alert" className="max-w-md space-y-4 rounded-lg border px-4 py-4">
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{message}</p>
      </div>
      <div className="flex gap-2">{children}</div>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-8" aria-busy="true" aria-label="Loading incident">
      <div className="space-y-4">
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-8 w-2/3" />
        <div className="flex gap-3">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-5 w-20 rounded-4xl" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-8 w-20" />
          <Skeleton className="h-8 w-16" />
        </div>
      </div>
      <Skeleton className="h-4 w-3/4" />
      <div className="grid gap-8 md:grid-cols-2">
        <Skeleton className="h-40" />
        <Skeleton className="h-24" />
      </div>
    </div>
  );
}
