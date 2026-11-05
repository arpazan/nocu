import type { ReactNode } from "react";

import { Label } from "@/components/ui/label";

/** A labelled control in the filter row above a list; the label sits in quiet grey over the select. */
export function FilterField({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id} className="text-xs text-muted-foreground">
        {label}
      </Label>
      {children}
    </div>
  );
}
