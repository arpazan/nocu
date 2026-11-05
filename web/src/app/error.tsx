"use client";

import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";

export default function ErrorPage({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  const message =
    error instanceof ApiError
      ? `${error.code}: ${error.message}`
      : error instanceof Error
        ? error.message
        : "Something went wrong";

  return (
    <div className="flex flex-col items-center gap-4 py-24 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">Something went wrong</h1>
      <p className="max-w-md font-mono text-sm text-muted-foreground">{message}</p>
      <Button onClick={() => retry()}>Try again</Button>
    </div>
  );
}
