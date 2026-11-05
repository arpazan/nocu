"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { Controller, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api/client";
import { createIncident, incidentKeys, updateIncident } from "@/lib/api/incidents";
import type { Incident } from "@/lib/api/types";
import { createIncidentSchema, updateIncidentSchema, type CreateIncidentValues } from "@/lib/incidents/schemas";
import { SEVERITIES, severityLabel } from "@/lib/incidents/transitions";

export const OPERATOR_ROLE_MESSAGE = "You need the operator role to do this";

type FormValues = CreateIncidentValues;

/** Creates an incident when `incident` is absent, otherwise edits it in place. */
export function IncidentFormDialog({
  open,
  onOpenChange,
  incident,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  incident?: Incident;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{incident ? "Edit incident" : "New incident"}</DialogTitle>
          <DialogDescription>
            {incident
              ? "Change the title, severity or description. Status is changed from the incident page."
              : "Describe what is wrong and how urgent it is."}
          </DialogDescription>
        </DialogHeader>
        {/* Keyed so reopening for a different incident starts from fresh values. */}
        <IncidentForm key={incident?.id ?? "new"} incident={incident} onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

function IncidentForm({ incident, onDone }: { incident?: Incident; onDone: () => void }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const isEdit = incident !== undefined;

  // The update schema makes every field optional; the form still holds the full shape.
  const resolver = isEdit
    ? (zodResolver(updateIncidentSchema) as unknown as Resolver<FormValues>)
    : (zodResolver(createIncidentSchema) as Resolver<FormValues>);

  const form = useForm<FormValues>({
    resolver,
    defaultValues: {
      title: incident?.title ?? "",
      description: incident?.description ?? "",
      severity: incident?.severity,
      source: incident?.source ?? "",
    },
  });

  const save = useMutation({
    mutationFn: (values: FormValues) =>
      incident
        ? updateIncident(incident.id, {
            title: values.title,
            description: values.description ?? "",
            severity: values.severity,
          })
        : createIncident({
            title: values.title,
            severity: values.severity,
            description: values.description || undefined,
            source: values.source || undefined,
          }),
    onSuccess: (saved) => {
      void queryClient.invalidateQueries({ queryKey: incidentKeys.all });
      toast.success(isEdit ? "Changes saved" : "Incident created");
      onDone();
      if (!isEdit) router.push(`/incidents/${saved.id}`);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 403) {
        form.setError("root", { message: OPERATOR_ROLE_MESSAGE });
        return;
      }
      form.setError("root", {
        message: err instanceof Error ? err.message : isEdit ? "Could not save changes" : "Could not create the incident",
      });
    },
  });

  return (
    <form noValidate onSubmit={form.handleSubmit((values) => save.mutate(values))}>
      <FieldGroup>
        <Controller
          control={form.control}
          name="title"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor={field.name}>Title</FieldLabel>
              <Input
                {...field}
                id={field.name}
                autoFocus
                placeholder="Core router down"
                aria-invalid={fieldState.invalid}
              />
              <FieldError errors={[fieldState.error]} />
            </Field>
          )}
        />
        <Controller
          control={form.control}
          name="severity"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor={field.name}>Severity</FieldLabel>
              <Select name={field.name} value={field.value ?? ""} onValueChange={field.onChange}>
                <SelectTrigger
                  id={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  className="w-full"
                  aria-invalid={fieldState.invalid}
                >
                  <SelectValue placeholder="Select a severity" />
                </SelectTrigger>
                <SelectContent>
                  {SEVERITIES.map((severity) => (
                    <SelectItem key={severity} value={severity}>
                      {severityLabel[severity]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldError errors={[fieldState.error]} />
            </Field>
          )}
        />
        <Controller
          control={form.control}
          name="description"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor={field.name}>Description</FieldLabel>
              <Textarea
                {...field}
                value={field.value ?? ""}
                id={field.name}
                rows={3}
                placeholder="What is affected and what you have seen so far"
                aria-invalid={fieldState.invalid}
              />
              <FieldError errors={[fieldState.error]} />
            </Field>
          )}
        />
        {!isEdit && (
          <Controller
            control={form.control}
            name="source"
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldLabel htmlFor={field.name}>Source</FieldLabel>
                <Input
                  {...field}
                  value={field.value ?? ""}
                  id={field.name}
                  placeholder="snmp, manual, pagerduty…"
                  aria-invalid={fieldState.invalid}
                />
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />
        )}
        <FieldError errors={[form.formState.errors.root]} />
      </FieldGroup>
      <DialogFooter className="mt-4">
        <Button type="button" variant="outline" onClick={onDone} disabled={save.isPending}>
          Cancel
        </Button>
        <Button type="submit" disabled={save.isPending}>
          {isEdit ? (save.isPending ? "Saving…" : "Save changes") : save.isPending ? "Creating…" : "Create incident"}
        </Button>
      </DialogFooter>
    </form>
  );
}
