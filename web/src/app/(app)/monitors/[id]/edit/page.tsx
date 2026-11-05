import type { Metadata } from "next";
import Link from "next/link";

import { MonitorEditLoader } from "@/components/monitors/monitor-edit-loader";

export const metadata: Metadata = { title: "Edit monitor" };

export default async function EditMonitorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <div className="space-y-8">
      <div>
        <Link
          href={`/monitors/${id}`}
          className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
        >
          Back to monitor
        </Link>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Edit monitor</h1>
        <p className="text-muted-foreground">Only the fields you change are sent.</p>
      </div>
      <MonitorEditLoader id={id} />
    </div>
  );
}
