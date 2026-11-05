import type { Metadata } from "next";

import { IncidentDetail } from "@/components/incidents/incident-detail";

export const metadata: Metadata = { title: "Incident" };

export default async function IncidentPage({ params }: PageProps<"/incidents/[id]">) {
  const { id } = await params;
  return <IncidentDetail id={id} />;
}
