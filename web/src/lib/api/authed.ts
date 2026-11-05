import { ApiClient, ApiError, type RequestOptions } from "@/lib/api/client";
import { refreshSession } from "@/lib/auth/refresh";
import { getAccessToken } from "@/lib/auth/session-store";
import { env } from "@/lib/env";

/**
 * ApiClient that attaches the in-memory access token and transparently
 * refreshes it once when the backend answers 401. Every public method of
 * ApiClient funnels through `raw`, so overriding it is enough; the actual
 * fetch is delegated to the wrapped client so an injected fetchImpl keeps
 * working.
 */
class AuthedApiClient extends ApiClient {
  constructor(private readonly inner: ApiClient) {
    super(inner.baseUrl);
  }

  override async raw<T>(path: string, options: RequestOptions = {}): Promise<T> {
    // An explicit token (even null) means the caller owns auth for this call.
    if (options.token !== undefined) return this.inner.raw<T>(path, options);

    try {
      return await this.inner.raw<T>(path, { ...options, token: getAccessToken() });
    } catch (err) {
      if (!(err instanceof ApiError) || !err.isUnauthorized) throw err;
      const session = await refreshSession();
      if (!session) throw err;
      return this.inner.raw<T>(path, { ...options, token: session.access_token });
    }
  }
}

/** Wraps a client so every request carries the current access token. */
export function withAuth(client: ApiClient): ApiClient {
  return new AuthedApiClient(client);
}

/** Browser-side authenticated clients for the two HTTP services. */
export const authedCoreApi = withAuth(new ApiClient(env.coreApiUrl));
export const authedIdentityApi = withAuth(new ApiClient(env.identityUrl));
