"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

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
import { deleteMonitor, monitorKeys, pauseMonitor, resumeMonitor } from "@/lib/api/monitors";
import { ApiError } from "@/lib/api/client";
import type { Monitor } from "@/lib/api/types";

function mutationError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.status === 403) return "You need the operator role to do this";
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

/** Pause or resume checks for one monitor; both invalidate every monitor query. */
export function usePauseResume() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (monitor: Pick<Monitor, "id" | "active">) =>
      monitor.active ? pauseMonitor(monitor.id) : resumeMonitor(monitor.id),
    onSuccess: (_data, monitor) => {
      toast.success(monitor.active ? "Checks paused" : "Checks resumed");
      return queryClient.invalidateQueries({ queryKey: monitorKeys.all });
    },
    onError: (err, monitor) => {
      toast.error(mutationError(err, monitor.active ? "Could not pause checks" : "Could not resume checks"));
    },
  });
}

/** Soft-deletes one monitor; optionally navigates away afterwards. */
export function useDeleteMonitor(options: { redirectTo?: string } = {}) {
  const queryClient = useQueryClient();
  const router = useRouter();
  return useMutation({
    mutationFn: (id: string) => deleteMonitor(id),
    onSuccess: () => {
      toast.success("Monitor deleted");
      if (options.redirectTo) router.push(options.redirectTo);
      return queryClient.invalidateQueries({ queryKey: monitorKeys.all });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        toast.error("You need the admin role to delete a monitor");
        return;
      }
      toast.error(mutationError(err, "Could not delete the monitor"));
    },
  });
}

/** Confirmation dialog for deleting a monitor. Controlled so a menu can open it. */
export function DeleteMonitorDialog({
  open,
  onOpenChange,
  onConfirm,
  pending,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  pending?: boolean;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this monitor?</AlertDialogTitle>
          <AlertDialogDescription>Its check history goes with it.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={pending}
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}
          >
            {pending ? "Deleting…" : "Delete monitor"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
