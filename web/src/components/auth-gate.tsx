"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { useSessionBootstrap } from "@/lib/auth/use-session-bootstrap";

/**
 * Client-side guard for the signed-in area. The session only exists in memory,
 * so on first render it restores it from the refresh cookie, shows a skeleton
 * meanwhile and sends signed-out visitors to /login with a `next` to return to.
 */
export function AuthGate({ children }: { children: ReactNode }) {
  const status = useSessionBootstrap();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  useEffect(() => {
    if (status !== "signed-out") return;
    const search = searchParams.toString();
    const next = search.length > 0 ? `${pathname}?${search}` : pathname;
    router.replace(`/login?next=${encodeURIComponent(next)}`);
  }, [status, router, pathname, searchParams]);

  if (status === "signed-in") return children;
  if (status === "signed-out") return null;

  return (
    <div aria-busy="true" aria-label="Restoring session" className="flex flex-1 items-center justify-center py-12">
      <div className="w-full max-w-sm space-y-3">
        <Skeleton className="h-6 w-1/2" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-24 w-full" />
      </div>
    </div>
  );
}
