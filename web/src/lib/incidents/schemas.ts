import { z } from "zod";

/** Mirrors the create rules in bruno/core-api/incidents/create.bru and the column sizes in shared/model/incident.go. */
export const createIncidentSchema = z.object({
  title: z.string().trim().min(1, "Enter a title").max(255, "Title must be 255 characters or fewer"),
  description: z.string().optional(),
  severity: z.enum(["critical", "high", "medium", "low"], { error: "Choose a severity" }),
  source: z.string().trim().max(128, "Source must be 128 characters or fewer").optional(),
});

export type CreateIncidentValues = z.infer<typeof createIncidentSchema>;

/** PATCH body: every field optional, but a present title may not be empty. */
export const updateIncidentSchema = z.object({
  title: z.string().trim().min(1, "Enter a title").max(255, "Title must be 255 characters or fewer").optional(),
  description: z.string().optional(),
  severity: z.enum(["critical", "high", "medium", "low"], { error: "Choose a severity" }).optional(),
  status: z.enum(["open", "acknowledged", "resolved", "closed"], { error: "Choose a status" }).optional(),
});

export type UpdateIncidentValues = z.infer<typeof updateIncidentSchema>;
