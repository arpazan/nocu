import Link from "next/link";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";

export { ErrorPanel, errorMessage } from "@/components/list/error-panel";

/** Panel for a monitor that does not exist (404), with a way back to the list. */
export function NotFoundPanel({ children }: { children?: ReactNode }) {
  return (
    <div className="rounded-lg border border-border px-4 py-6">
      <p className="font-medium">This monitor does not exist</p>
      <p className="mt-1 text-sm text-muted-foreground">
        {children ?? "It may have been deleted, or the link is wrong."}
      </p>
      <Button variant="outline" size="sm" className="mt-4" asChild>
        <Link href="/monitors">Back to monitors</Link>
      </Button>
    </div>
  );
}
