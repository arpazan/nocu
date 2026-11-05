import type { Metadata } from "next";
import { Suspense } from "react";

import { MonitorsTable } from "@/components/monitors/monitors-table";

export const metadata: Metadata = { title: "Monitors" };

/** The table reads its filters from useSearchParams, which needs a Suspense boundary. */
export default function MonitorsPage() {
  return (
    <Suspense>
      <MonitorsTable />
    </Suspense>
  );
}
