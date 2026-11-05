import type { ReactNode } from "react";

/** One grey sentence where the rows would be, optionally followed by a single way out. */
export function EmptyState({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-lg border border-border px-4 py-10 text-center">
      <p className="text-sm text-muted-foreground">{children}</p>
      {action}
    </div>
  );
}
