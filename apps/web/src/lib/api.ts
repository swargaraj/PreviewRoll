import type { SavedConnection } from "./connection";
import { buildServerUrl } from "./connection";

export interface ApiRequestOptions {
  method?: string;
  path: string;
  body?: unknown;
  headers?: Record<string, string>;
}

export interface ApiResponse<T = unknown> {
  ok: boolean;
  status: number;
  data: T | null;
  error?: string;
}

export async function apiRequest<T = unknown>(
  connection: SavedConnection,
  options: ApiRequestOptions
): Promise<ApiResponse<T>> {
  const { method = "GET", path, body, headers = {} } = options;

  const url = buildServerUrl(connection, path);

  try {
    const res = await fetch(url, {
      method,
      headers: {
        "Content-Type": "application/json",
        ...headers,
      },
      credentials: "include",
      body: body ? JSON.stringify(body) : undefined,
    });

    const data = await res.json().catch(() => null);

    if (!res.ok) {
      return {
        ok: false,
        status: res.status,
        data: null,
        error: data?.message || `Request failed (${res.status})`,
      };
    }

    return {
      ok: true,
      status: res.status,
      data: data as T,
    };
  } catch (err) {
    return {
      ok: false,
      status: 0,
      data: null,
      error: err instanceof Error ? err.message : "Network error",
    };
  }
}
