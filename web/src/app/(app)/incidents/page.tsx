import type { Metadata } from "next";
import { Suspense } from "react";

import { IncidentsTable, IncidentsTableSkeleton } from "@/components/incidents/incidents-table";

export const metadata: Metadata = { title: "Incidents" };

/** Suspense is required because the table reads useSearchParams. */
export default function IncidentsPage() {
  return (
    <Suspense fallback={<IncidentsTableSkeleton />}>
      <IncidentsTable />
    </Suspense>
  );
}
