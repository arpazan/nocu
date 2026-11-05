import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";

/** Human message for a failed query. */
export function errorMessage(error: unknown, fallback = "Something went wrong"): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

/** Quiet inline panel for a failed query with a retry action. */
export function ErrorPanel({
  error,
  onRetry,
  title = "Could not load this list",
}: {
  error: unknown;
  onRetry: () => void;
  title?: string;
}) {
  return (
    <div role="alert" className="rounded-lg border border-border px-4 py-6">
      <p className="font-medium">{title}</p>
      <p className="mt-1 text-sm text-muted-foreground">{errorMessage(error)}</p>
      <Button variant="outline" size="sm" className="mt-4" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
