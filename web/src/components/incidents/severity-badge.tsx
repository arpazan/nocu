import type { Severity } from "@/lib/api/types";
import { severityLabel } from "@/lib/incidents/transitions";
import { cn } from "@/lib/utils";

/** Bar colour per severity: destructive is reserved for critical, orange for high, blue for medium. */
const BAR: Record<Severity, string> = {
  critical: "bg-destructive",
  high: "bg-primary",
  medium: "bg-tertiary",
  low: "bg-muted-foreground/40",
};

/** A 2px vertical bar coloured by severity, followed by its label. */
export function SeverityMark({
  severity,
  className,
}: {
  severity: Severity;
  className?: string;
}) {
  return (
    <span className={cn("inline-flex items-center gap-2 text-sm", className)}>
      <span aria-hidden="true" className={cn("h-4 w-0.5 shrink-0 rounded-full", BAR[severity])} />
      {severityLabel[severity]}
    </span>
  );
}
