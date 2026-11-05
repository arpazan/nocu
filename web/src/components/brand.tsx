import { ActivityIcon } from "lucide-react";
import Link from "next/link";

import { cn } from "@/lib/utils";

/** The Nocu mark: the Activity icon in Signal Orange next to the wordmark. Links home. */
export function Brand({ className }: { className?: string }) {
  return (
    <Link
      href="/"
      className={cn(
        "inline-flex items-center gap-2 rounded-md font-semibold tracking-tight outline-none focus-visible:ring-2 focus-visible:ring-ring",
        className,
      )}
    >
      <ActivityIcon className="size-5 text-primary" aria-hidden="true" />
      Nocu
    </Link>
  );
}
