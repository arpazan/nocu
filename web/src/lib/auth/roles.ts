/**
 * Role checks mirroring the identity seed roles (viewer, operator, admin) and
 * the core-api middleware: operators and admins may mutate monitors and
 * incidents, only admins may delete. The backend still enforces these; the UI
 * uses them to hide controls that would fail anyway.
 */

export const WRITE_ROLES = ["operator", "admin"] as const;

/** True when the user may create, update, pause or resume resources. */
export function canWrite(roles: readonly string[] | undefined): boolean {
  return (roles ?? []).some((role) => (WRITE_ROLES as readonly string[]).includes(role));
}

/** True when the user may delete resources (admin only). */
export function canDelete(roles: readonly string[] | undefined): boolean {
  return (roles ?? []).includes("admin");
}
