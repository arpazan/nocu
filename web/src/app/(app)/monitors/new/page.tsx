import type { Metadata } from "next";

import { MonitorForm } from "@/components/monitors/monitor-form";

export const metadata: Metadata = { title: "New monitor" };

export default function NewMonitorPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">New monitor</h1>
        <p className="text-muted-foreground">Only a name and a URL are needed; everything else has a default.</p>
      </div>
      <MonitorForm />
    </div>
  );
}
