import type { ApiErrorBody, Envelope, Meta } from "@/lib/api/types";

/**
 * Error thrown for every non-2xx response. `code` and `message` come from the
 * backend envelope when present, so callers can switch on `code` the same way
 * Go callers use errors.Is.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  /** JSON-encoded as the request body. */
  body?: unknown;
  /** Bearer token for the Authorization header. */
  token?: string | null;
  /** Query parameters appended to the URL. Undefined values are skipped. */
  query?: Record<string, string | number | boolean | undefined>;
}

export interface ListResult<T> {
  items: T[];
  meta: Meta;
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  return (
    typeof value === "object" &&
    value !== null &&
    "error" in value &&
    typeof (value as ApiErrorBody).error?.code === "string"
  );
}

function buildUrl(baseUrl: string, path: string, query?: RequestOptions["query"]): string {
  const url = new URL(path.startsWith("/") ? path : `/${path}`, baseUrl);
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined) url.searchParams.set(key, String(value));
  }
  return url.toString();
}

async function parseBody(res: Response): Promise<unknown> {
  if (res.status === 204) return undefined;
  const text = await res.text();
  if (text.length === 0) return undefined;
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

/**
 * ApiClient is a thin fetch wrapper bound to one service base URL. It encodes
 * JSON, attaches the bearer token and converts the backend's `{error}` body
 * into an ApiError so call sites only deal with typed success values.
 */
export class ApiClient {
  constructor(
    readonly baseUrl: string,
    private readonly fetchImpl: typeof fetch = (...args) => fetch(...args),
  ) {}

  /** Performs a request and returns the raw decoded body (no envelope). */
  async raw<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { body, token, query, headers, ...init } = options;
    const res = await this.fetchImpl(buildUrl(this.baseUrl, path, query), {
      ...init,
      headers: {
        Accept: "application/json",
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...headers,
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    const payload = await parseBody(res);
    if (!res.ok) {
      if (isErrorBody(payload)) {
        throw new ApiError(res.status, payload.error.code, payload.error.message);
      }
      throw new ApiError(res.status, "http_error", res.statusText || `HTTP ${res.status}`);
    }
    return payload as T;
  }

  /** Performs a request and unwraps the `{data}` envelope. */
  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const envelope = await this.raw<Envelope<T> | undefined>(path, options);
    return envelope?.data as T;
  }

  /** Performs a list request and returns both items and pagination meta. */
  async list<T>(path: string, options: RequestOptions = {}): Promise<ListResult<T>> {
    const envelope = await this.raw<Envelope<T[]>>(path, options);
    return {
      items: envelope.data,
      meta: envelope.meta ?? { total: envelope.data.length, limit: envelope.data.length, offset: 0 },
    };
  }

  get<T>(path: string, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: "GET" });
  }

  post<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: "POST", body });
  }

  put<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: "PUT", body });
  }

  patch<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: "PATCH", body });
  }

  delete<T = void>(path: string, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: "DELETE" });
  }
}
