import { ApiClient } from "@/lib/api/client";
import { env } from "@/lib/env";

export { ApiClient, ApiError } from "@/lib/api/client";
export type { ListResult, RequestOptions } from "@/lib/api/client";
export type * from "@/lib/api/types";

/** Browser-side clients for the two HTTP services. */
export const identityApi = new ApiClient(env.identityUrl);
export const coreApi = new ApiClient(env.coreApiUrl);
