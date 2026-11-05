import { queryOptions } from "@tanstack/react-query";

import { authedIdentityApi } from "@/lib/api/authed";
import { ApiClient } from "@/lib/api/client";
import type { RegisterRequest, User } from "@/lib/api/types";
import { env } from "@/lib/env";

/** Unauthenticated identity client for the public endpoints. */
const identityApi = new ApiClient(env.identityUrl);

/** Query options for identity /me. Cached per tab; a 401 is handled by the authed client. */
export function meQueryOptions() {
  return queryOptions({
    queryKey: ["me"] as const,
    queryFn: () => authedIdentityApi.get<User>("/api/v1/me"),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}

/** Creates an account. Throws ApiError 409 (conflict) when the email is taken, 400 on validation. */
export function register(values: RegisterRequest): Promise<User> {
  return identityApi.post<User>("/api/v1/auth/register", values);
}
