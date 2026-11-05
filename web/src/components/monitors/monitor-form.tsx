"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, Trash2Icon } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import {
  type ArrayPath,
  type Control,
  Controller,
  type FieldPath,
  useFieldArray,
  useForm,
  type UseFormSetError,
  useWatch,
} from "react-hook-form";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api/client";
import { createMonitor, monitorKeys, updateMonitor } from "@/lib/api/monitors";
import type { Monitor, NotificationType } from "@/lib/api/types";
import { authMethodLabel, bodyEncodingLabel, notificationTypeLabel } from "@/lib/monitors/labels";
import {
  AUTH_METHODS,
  BODY_ENCODINGS,
  fromMonitor,
  HTTP_METHODS,
  type MonitorFormValues,
  monitorFormDefaults,
  monitorFormSchema,
  newNotification,
  toCreateRequest,
  toUpdateRequest,
} from "@/lib/monitors/schemas";

type FormControl = Control<MonitorFormValues>;
type Name = FieldPath<MonitorFormValues>;

// ---------------------------------------------------------------------------
// Field helpers. Values are cast at the edge because the generic path type
// cannot tell TypeScript which leaf is a string, number or boolean.
// ---------------------------------------------------------------------------

function TextField({
  control,
  name,
  label,
  description,
  type = "text",
  placeholder,
  autoComplete,
  mono,
}: {
  control: FormControl;
  name: Name;
  label: string;
  description?: ReactNode;
  type?: "text" | "url" | "password";
  placeholder?: string;
  autoComplete?: string;
  mono?: boolean;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field data-invalid={fieldState.invalid}>
          <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
          <Input
            name={field.name}
            ref={field.ref}
            onBlur={field.onBlur}
            onChange={field.onChange}
            value={(field.value as string | undefined) ?? ""}
            id={field.name}
            type={type}
            placeholder={placeholder}
            autoComplete={autoComplete}
            className={mono ? "font-mono" : undefined}
            aria-invalid={fieldState.invalid}
          />
          {description ? <FieldDescription>{description}</FieldDescription> : null}
          <FieldError errors={[fieldState.error]} />
        </Field>
      )}
    />
  );
}

function TextareaField({
  control,
  name,
  label,
  description,
  placeholder,
  rows = 4,
}: {
  control: FormControl;
  name: Name;
  label: string;
  description?: ReactNode;
  placeholder?: string;
  rows?: number;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field data-invalid={fieldState.invalid}>
          <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
          <Textarea
            name={field.name}
            ref={field.ref}
            onBlur={field.onBlur}
            onChange={field.onChange}
            value={(field.value as string | undefined) ?? ""}
            id={field.name}
            rows={rows}
            placeholder={placeholder}
            className="font-mono text-sm"
            aria-invalid={fieldState.invalid}
          />
          {description ? <FieldDescription>{description}</FieldDescription> : null}
          <FieldError errors={[fieldState.error]} />
        </Field>
      )}
    />
  );
}

function NumberField({
  control,
  name,
  label,
  description,
  min,
  unit,
}: {
  control: FormControl;
  name: Name;
  label: string;
  description?: ReactNode;
  min?: number;
  unit?: string;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const value = field.value as number | undefined;
        return (
          <Field data-invalid={fieldState.invalid}>
            <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
            <div className="flex items-center gap-2">
              <Input
                name={field.name}
                ref={field.ref}
                onBlur={field.onBlur}
                id={field.name}
                type="number"
                inputMode="numeric"
                min={min}
                step={1}
                className="w-32 tabular-nums"
                value={value === undefined || Number.isNaN(value) ? "" : value}
                onChange={(event) => {
                  const raw = event.target.value;
                  // Empty stays undefined so zod reports the field as required.
                  field.onChange(raw === "" ? undefined : Number(raw));
                }}
                aria-invalid={fieldState.invalid}
              />
              {unit ? <span className="text-sm text-muted-foreground">{unit}</span> : null}
            </div>
            {description ? <FieldDescription>{description}</FieldDescription> : null}
            <FieldError errors={[fieldState.error]} />
          </Field>
        );
      }}
    />
  );
}

function SwitchField({
  control,
  name,
  label,
  description,
}: {
  control: FormControl;
  name: Name;
  label: string;
  description?: ReactNode;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field orientation="horizontal" data-invalid={fieldState.invalid}>
          <Switch
            id={field.name}
            name={field.name}
            ref={field.ref}
            checked={Boolean(field.value)}
            onCheckedChange={field.onChange}
            onBlur={field.onBlur}
            aria-invalid={fieldState.invalid}
          />
          <div className="flex flex-col gap-0.5">
            <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
            {description ? <FieldDescription>{description}</FieldDescription> : null}
          </div>
        </Field>
      )}
    />
  );
}

function SelectField({
  control,
  name,
  label,
  options,
  description,
}: {
  control: FormControl;
  name: Name;
  label: string;
  options: readonly { value: string; label: string }[];
  description?: ReactNode;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <Field data-invalid={fieldState.invalid}>
          <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
          <Select value={(field.value as string | undefined) ?? ""} onValueChange={field.onChange} name={field.name}>
            <SelectTrigger id={field.name} className="w-56" aria-invalid={fieldState.invalid} onBlur={field.onBlur}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {description ? <FieldDescription>{description}</FieldDescription> : null}
          <FieldError errors={[fieldState.error]} />
        </Field>
      )}
    />
  );
}

/** Editable key/value rows for a header map. */
function HeaderRows({
  control,
  name,
  addLabel = "Add header",
}: {
  control: FormControl;
  name: ArrayPath<MonitorFormValues>;
  addLabel?: string;
}) {
  const rows = useFieldArray({ control, name });
  return (
    <div className="flex flex-col gap-2">
      {rows.fields.map((row, index) => (
        <div key={row.id} className="flex items-start gap-2">
          <Controller
            control={control}
            name={`${name}.${index}.key` as Name}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid} className="flex-1">
                <Input
                  name={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  onChange={field.onChange}
                  value={(field.value as string | undefined) ?? ""}
                  placeholder="Header name"
                  aria-label={`Header ${index + 1} name`}
                  className="font-mono"
                  aria-invalid={fieldState.invalid}
                />
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />
          <Controller
            control={control}
            name={`${name}.${index}.value` as Name}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid} className="flex-1">
                <Input
                  name={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  onChange={field.onChange}
                  value={(field.value as string | undefined) ?? ""}
                  placeholder="Value"
                  aria-label={`Header ${index + 1} value`}
                  className="font-mono"
                  aria-invalid={fieldState.invalid}
                />
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Remove header ${index + 1}`}
            onClick={() => rows.remove(index)}
          >
            <Trash2Icon />
          </Button>
        </div>
      ))}
      <div>
        <Button type="button" variant="outline" size="sm" onClick={() => rows.append({ key: "", value: "" })}>
          <PlusIcon data-icon="inline-start" />
          {addLabel}
        </Button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sections
// ---------------------------------------------------------------------------

function Section({ title, help, children }: { title: string; help: string; children: ReactNode }) {
  return (
    <section className="space-y-5 border-t border-border pt-8 first:border-t-0 first:pt-0">
      <div>
        <h2 className="text-base font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{help}</p>
      </div>
      {children}
    </section>
  );
}

const METHOD_OPTIONS = HTTP_METHODS.map((m) => ({ value: m, label: m }));
const ENCODING_OPTIONS = BODY_ENCODINGS.map((e) => ({ value: e, label: bodyEncodingLabel[e] }));
const AUTH_OPTIONS = AUTH_METHODS.map((a) => ({ value: a, label: authMethodLabel[a] }));
const MESSAGE_FORMAT_OPTIONS = [
  { value: "text", label: "Text" },
  { value: "embed", label: "Embed" },
];
const WEBHOOK_METHOD_OPTIONS = ["POST", "PUT", "PATCH"].map((m) => ({ value: m, label: m }));
const REQUEST_BODY_OPTIONS = [
  { value: "json", label: "JSON" },
  { value: "form", label: "Form (urlencoded)" },
  { value: "custom", label: "Custom" },
];

function AuthenticationFields({ control, editing }: { control: FormControl; editing: boolean }) {
  const authMethod = useWatch({ control, name: "auth_method" });
  const hasPassword = useWatch({ control, name: "has_basic_password" }) === true;
  const hasToken = useWatch({ control, name: "has_bearer_token" }) === true;

  return (
    <>
      <SelectField control={control} name="auth_method" label="Auth method" options={AUTH_OPTIONS} />
      {authMethod === "basic" ? (
        <>
          <TextField control={control} name="basic_username" label="Username" autoComplete="off" />
          <TextField
            control={control}
            name="basic_password"
            label="Password"
            type="password"
            autoComplete="new-password"
            description={editing && hasPassword ? "A password is saved. Leave blank to keep it." : undefined}
          />
          {editing && hasPassword ? (
            <SwitchField control={control} name="clear_basic_password" label="Clear saved password" />
          ) : null}
        </>
      ) : null}
      {authMethod === "bearer" ? (
        <>
          <TextField
            control={control}
            name="bearer_token"
            label="Token"
            type="password"
            autoComplete="off"
            description={editing && hasToken ? "A token is saved. Leave blank to keep it." : undefined}
          />
          {editing && hasToken ? (
            <SwitchField control={control} name="clear_bearer_token" label="Clear saved token" />
          ) : null}
        </>
      ) : null}
    </>
  );
}

function DiscordFields({ control, index }: { control: FormControl; index: number }) {
  const base = `notifications.${index}.config` as const;
  return (
    <>
      <TextField
        control={control}
        name={`${base}.webhook_url`}
        label="Webhook URL"
        type="url"
        placeholder="https://discord.com/api/webhooks/…"
        mono
      />
      <div className="grid gap-5 sm:grid-cols-2">
        <TextField control={control} name={`${base}.bot_display_name`} label="Bot display name" />
        <SelectField
          control={control}
          name={`${base}.message_format`}
          label="Message format"
          options={MESSAGE_FORMAT_OPTIONS}
        />
      </div>
      <TextField
        control={control}
        name={`${base}.prefix_custom_message`}
        label="Message prefix"
        description="Added in front of every message, for example a team mention."
      />
      <SwitchField control={control} name={`${base}.disable_url`} label="Leave the monitor URL out of messages" />
    </>
  );
}

function WebhookFields({ control, index }: { control: FormControl; index: number }) {
  const base = `notifications.${index}.config` as const;
  const requestBody = useWatch({ control, name: `${base}.request_body` });
  return (
    <>
      <TextField control={control} name={`${base}.post_url`} label="URL" type="url" placeholder="https://" mono />
      <div className="grid gap-5 sm:grid-cols-2">
        <SelectField control={control} name={`${base}.http_method`} label="Method" options={WEBHOOK_METHOD_OPTIONS} />
        <SelectField
          control={control}
          name={`${base}.request_body`}
          label="Request body"
          options={REQUEST_BODY_OPTIONS}
        />
      </div>
      {requestBody === "custom" ? (
        <TextareaField
          control={control}
          name={`${base}.custom_body`}
          label="Custom body"
          description={
            <>
              You can use <code>{"{{monitor_name}}"}</code>, <code>{"{{monitor_url}}"}</code>,{" "}
              <code>{"{{status}}"}</code> and <code>{"{{message}}"}</code>.
            </>
          }
        />
      ) : null}
      <div className="space-y-2">
        <p className="text-sm font-medium">Additional headers</p>
        <HeaderRows control={control} name={`${base}.additional_headers`} />
      </div>
    </>
  );
}

function WebPushSummary({ control, index }: { control: FormControl; index: number }) {
  const endpoint = useWatch({ control, name: `notifications.${index}.config.endpoint` });
  let host = "";
  try {
    host = new URL(endpoint).host;
  } catch {
    host = endpoint;
  }
  return (
    <p className="text-sm text-muted-foreground">
      Browser push, configured elsewhere.
      {host ? (
        <>
          {" "}
          Delivers to <span className="font-mono">{host}</span>.
        </>
      ) : null}
    </p>
  );
}

function NotificationCard({
  control,
  index,
  type,
  onRemove,
}: {
  control: FormControl;
  index: number;
  type: NotificationType;
  onRemove: () => void;
}) {
  return (
    <div className="space-y-5 rounded-lg border border-border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="font-medium">{notificationTypeLabel[type]}</p>
        <div className="flex items-center gap-3">
          <SwitchField control={control} name={`notifications.${index}.enabled`} label="Enabled" />
          <Button type="button" variant="ghost" size="sm" onClick={onRemove}>
            <Trash2Icon data-icon="inline-start" />
            Remove
          </Button>
        </div>
      </div>
      {type === "discord" ? <DiscordFields control={control} index={index} /> : null}
      {type === "webhook" ? <WebhookFields control={control} index={index} /> : null}
      {type === "webpush" ? <WebPushSummary control={control} index={index} /> : null}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

const FIELD_HINTS: [RegExp, FieldPath<MonitorFormValues>][] = [
  [/interval/i, "interval_seconds"],
  [/timeout/i, "timeout_seconds"],
  [/\burl\b/i, "url"],
  [/\bname\b/i, "name"],
  [/retries/i, "retries"],
  [/redirect/i, "max_redirects"],
];

function applyApiError(err: unknown, setError: UseFormSetError<MonitorFormValues>, fallback: string) {
  if (err instanceof ApiError && err.status === 403) {
    setError("root", { message: "You need the operator role to do this" });
    return;
  }
  if (err instanceof ApiError && err.status === 400) {
    setError("root", { message: err.message });
    for (const [pattern, field] of FIELD_HINTS) {
      if (pattern.test(err.message)) {
        setError(field, { message: err.message });
        break;
      }
    }
    return;
  }
  setError("root", { message: err instanceof Error && err.message ? err.message : fallback });
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

/** Create and edit form for a monitor. Pass `monitor` to edit it. */
export function MonitorForm({ monitor }: { monitor?: Monitor }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const editing = monitor !== undefined;

  const form = useForm<MonitorFormValues>({
    resolver: zodResolver(monitorFormSchema),
    defaultValues: monitor ? fromMonitor(monitor) : monitorFormDefaults,
  });
  const { control } = form;
  const notifications = useFieldArray({ control, name: "notifications" });

  const save = useMutation({
    mutationFn: async (values: MonitorFormValues): Promise<string> => {
      if (monitor) {
        const patch = toUpdateRequest(values, monitor);
        if (Object.keys(patch).length === 0) return monitor.id;
        const updated = await updateMonitor(monitor.id, patch);
        return updated.id;
      }
      const created = await createMonitor(toCreateRequest(values));
      return created.id;
    },
    onSuccess: async (id) => {
      toast.success(editing ? "Changes saved" : "Monitor created");
      await queryClient.invalidateQueries({ queryKey: monitorKeys.all });
      router.push(`/monitors/${id}`);
    },
    onError: (err) => applyApiError(err, form.setError, editing ? "Could not save changes" : "Could not create the monitor"),
  });

  const cancelHref = monitor ? `/monitors/${monitor.id}` : "/monitors";

  return (
    <form noValidate onSubmit={form.handleSubmit((values) => save.mutate(values))} className="max-w-3xl">
      <div className="space-y-8">
        <Section title="General" help="What to check and how often.">
          <FieldGroup>
            <TextField control={control} name="name" label="Name" placeholder="Public website" />
            <TextField
              control={control}
              name="url"
              label="URL"
              type="url"
              placeholder="https://example.com/health"
              mono
            />
            <div className="grid gap-5 sm:grid-cols-3">
              <NumberField control={control} name="interval_seconds" label="Interval" unit="seconds" min={20} />
              <NumberField
                control={control}
                name="timeout_seconds"
                label="Timeout"
                unit="seconds"
                min={1}
                description="Must be shorter than the interval"
              />
              <NumberField
                control={control}
                name="retries"
                label="Retries"
                min={0}
                description="Failures tolerated before the monitor is down"
              />
            </div>
          </FieldGroup>
        </Section>

        <Section title="Advanced" help="Extra warnings and what to keep from failed checks.">
          <FieldGroup>
            <SwitchField
              control={control}
              name="cert_expiry_notification"
              label="Warn before the TLS certificate expires"
            />
            <SwitchField control={control} name="domain_expiry_notification" label="Warn before the domain expires" />
            <SwitchField
              control={control}
              name="save_error_response"
              label="Keep the response body of failed checks"
            />
            <NumberField
              control={control}
              name="max_redirects"
              label="Max redirects"
              min={0}
              description="0 disables following redirects"
            />
          </FieldGroup>
        </Section>

        <Section title="Request" help="How the check calls the URL.">
          <FieldGroup>
            <div className="grid gap-5 sm:grid-cols-2">
              <SelectField control={control} name="method" label="Method" options={METHOD_OPTIONS} />
              <SelectField control={control} name="body_encoding" label="Body encoding" options={ENCODING_OPTIONS} />
            </div>
            <TextareaField control={control} name="body" label="Body" placeholder="Sent with POST, PUT and PATCH" />
            <div className="space-y-2">
              <p className="text-sm font-medium">Headers</p>
              <HeaderRows control={control} name="headers" />
            </div>
          </FieldGroup>
        </Section>

        <Section title="Authentication" help="Credentials sent with every check. They are stored, never shown again.">
          <FieldGroup>
            <AuthenticationFields control={control} editing={editing} />
          </FieldGroup>
        </Section>

        <Section title="Notifications" help="Where to send a message when the status changes.">
          <div className="space-y-4">
            {notifications.fields.length === 0 ? (
              <p className="text-sm text-muted-foreground">No channels yet. Status changes stay in the app.</p>
            ) : null}
            {notifications.fields.map((field, index) => (
              <NotificationCard
                key={field.id}
                control={control}
                index={index}
                type={field.type}
                onRemove={() => notifications.remove(index)}
              />
            ))}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button type="button" variant="outline" size="sm">
                  <PlusIcon data-icon="inline-start" />
                  Add channel
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuItem onSelect={() => notifications.append(newNotification("discord"))}>
                  Discord
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => notifications.append(newNotification("webhook"))}>
                  Webhook
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </Section>
      </div>

      <div className="sticky bottom-0 mt-8 flex flex-wrap items-center justify-between gap-3 border-t border-border bg-background py-4">
        <FieldError errors={[form.formState.errors.root]} />
        <div className="ml-auto flex items-center gap-2">
          <Button type="button" variant="ghost" asChild>
            <Link href={cancelHref}>Cancel</Link>
          </Button>
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? "Saving…" : editing ? "Save changes" : "Create monitor"}
          </Button>
        </div>
      </div>
    </form>
  );
}
