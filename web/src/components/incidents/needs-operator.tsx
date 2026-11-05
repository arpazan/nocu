"use client";

import type { ReactNode } from "react";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

export const NEEDS_OPERATOR = "Needs the operator role";

/**
 * Wraps a control that the caller has already disabled for viewers so the
 * reason shows on hover and focus. Disabled buttons swallow pointer events, so
 * the tooltip hangs off a focusable span around them. Carries its own provider
 * so it works outside the app shell (tests, bare pages).
 */
export function NeedsOperator({ allowed, children }: { allowed: boolean; children: ReactNode }) {
  if (allowed) return <>{children}</>;
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span tabIndex={0} className="inline-flex rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-ring">
            {children}
          </span>
        </TooltipTrigger>
        <TooltipContent>{NEEDS_OPERATOR}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
