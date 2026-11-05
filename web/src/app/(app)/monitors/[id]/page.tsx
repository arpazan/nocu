import type { Metadata } from "next";
import { Suspense } from "react";

import { MonitorDetail } from "@/components/monitors/monitor-detail";

export const metadata: Metadata = { title: "Monitor" };

/** Detail reads history filters from useSearchParams, which needs a Suspense boundary. */
export default async function MonitorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <Suspense>
      <MonitorDetail id={id} />
    </Suspense>
  );
}
