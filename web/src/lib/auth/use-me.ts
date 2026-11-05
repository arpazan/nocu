import { useQuery } from "@tanstack/react-query";

import { meQueryOptions } from "@/lib/api/identity";
import { useSession } from "@/lib/auth/session-store";

/** The signed-in user from identity /me; disabled while there is no access token. */
export function useMe() {
  const session = useSession();
  return useQuery({ ...meQueryOptions(), enabled: session !== null });
}

/** Roles of the signed-in user, empty while unknown or signed out. */
export function useRoles(): string[] {
  const me = useMe();
  return me.data?.roles ?? [];
}
