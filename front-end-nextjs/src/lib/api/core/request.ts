import { emitPointsAwardFromResponsePayload } from "../../points-award-events";
import type { AdminListPayload, AdminListRow } from "../../types/admin";
import type { ApiEnvelope } from "../../types/auth";
import type {
  ApiEnvelopeWithExtras,
  ApiRequestOptions,
  QueryValue,
} from "./contracts";
import {
  ACCESS_TOKEN_KEY,
  ApiError,
  ApiUnauthorizedError,
  DEFAULT_ADMIN_API_TIMEOUT_MS,
  DEFAULT_API_TIMEOUT_MS,
  REFRESH_TOKEN_KEY,
} from "./contracts";
import { throwAccessBlockResponse } from "./access-block";
import {
  applyAuthorizationHeader,
  applyForwardedHeaders,
  buildApiUrl,
  createRequestSignal,
  normalizeAuthToken,
} from "./http";
import {
  extractEnvelope,
  parseResponse,
  responseErrorCode,
  responseErrorDetails,
} from "./response";
import {
  canUseStorage,
  clearAdminSession,
  clearSession,
  getRequestAuthorizationToken,
  getStoredAccessToken,
  getStoredAdminAccessToken,
  getStoredRefreshToken,
  normalizeTokens,
  persistAccessTokenCookie,
} from "./session";

let refreshAccessTokenPromise: Promise<string | null> | null = null;

export async function refreshAccessToken() {
  if (!canUseStorage()) {
    return null;
  }

  if (!refreshAccessTokenPromise) {
    refreshAccessTokenPromise = refreshAccessTokenNow().finally(() => {
      refreshAccessTokenPromise = null;
    });
  }

  return refreshAccessTokenPromise;
}

async function refreshAccessTokenNow() {
  const refreshToken = getStoredRefreshToken();
  const url = buildApiUrl("/api/auth/refresh");

  const response = await fetch(url, {
    method: "POST",
    credentials: "include",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(refreshToken ? { refresh_token: refreshToken } : {}),
    cache: "no-store",
    redirect: "manual",
  });

  throwAccessBlockResponse(response, url);

  if (!response.ok) {
    clearSession();
    return null;
  }

  const payload = await parseResponse(response);
  const data = extractEnvelope<Record<string, unknown>>(
    payload,
    response.status,
  );
  const tokens = normalizeTokens(data);
  if (!tokens.accessToken || !tokens.refreshToken) {
    clearSession();
    return null;
  }

  window.localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
  window.localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  persistAccessTokenCookie(tokens.accessToken);
  return tokens.accessToken;
}

export function getUpdatedAccessTokenForRetry(previousToken: string | null | undefined) {
  if (typeof window === "undefined") {
    return null;
  }
  const storedToken = canUseStorage()
    ? normalizeAuthToken(window.localStorage.getItem(ACCESS_TOKEN_KEY))
    : null;
  if (storedToken && storedToken !== previousToken) {
    return storedToken;
  }
  const latestToken = getStoredAccessToken();
  return latestToken && latestToken !== previousToken ? latestToken : null;
}

export async function getUnauthorizedRetryToken(
  previousToken: string | null | undefined,
  retryOnUnauthorized: boolean,
) {
  const updatedToken = getUpdatedAccessTokenForRetry(previousToken);
  if (updatedToken) {
    return updatedToken;
  }
  if (!retryOnUnauthorized) {
    return null;
  }
  return refreshAccessToken();
}

export function redirectToLogin() {
  if (typeof window !== "undefined") {
    window.location.assign("/login");
  }
}

export function isMutationMethod(method: string | undefined) {
  const normalizedMethod = method?.toUpperCase() ?? "GET";
  return normalizedMethod !== "GET" && normalizedMethod !== "HEAD";
}

export async function resolveMutationTokenOrRedirect(
  auth: boolean,
  token: string | null,
  method: string | undefined,
  retryOnUnauthorized: boolean,
  redirectOnUnauthorized = true,
) {
  if (
    !auth ||
    token ||
    !isMutationMethod(method) ||
    typeof window === "undefined"
  ) {
    return token;
  }

  if (retryOnUnauthorized) {
    const nextToken = await refreshAccessToken();
    if (nextToken) {
      return nextToken;
    }
  }

  if (redirectOnUnauthorized) {
    clearSession();
    redirectToLogin();
  }
  throw new ApiUnauthorizedError();
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const {
    query,
    headers: optionHeaders,
    auth = true,
    context,
    redirectOnUnauthorized = true,
    retryOnUnauthorized = true,
    signal: optionSignal,
    timeoutMs = DEFAULT_API_TIMEOUT_MS,
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);
  const token = await resolveMutationTokenOrRedirect(
    auth,
    getRequestAuthorizationToken(context, auth),
    fetchOptions.method,
    retryOnUnauthorized,
    redirectOnUnauthorized,
  );

  applyAuthorizationHeader(requestHeaders, token);
  applyForwardedHeaders(requestHeaders, context);

  if (context?.cookie && !requestHeaders.has("cookie")) {
    requestHeaders.set("cookie", context.cookie);
  }

  const isFormData = fetchOptions.body instanceof FormData;
  if (fetchOptions.body && !isFormData && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    credentials: "include",
    headers: requestHeaders,
    cache: fetchOptions.cache ?? "no-store",
    redirect: fetchOptions.redirect ?? "manual",
    signal: createRequestSignal(timeoutMs, optionSignal, context?.signal),
  });

  throwAccessBlockResponse(response, url);

  if (response.status === 401) {
    if (auth && typeof window !== "undefined") {
      const nextToken = await getUnauthorizedRetryToken(token, retryOnUnauthorized);
      if (nextToken) {
        return apiRequest<T>(path, {
          ...options,
          retryOnUnauthorized: false,
          context: { ...context, token: nextToken },
        });
      }
    }

    if (auth && redirectOnUnauthorized) {
      clearSession();
      redirectToLogin();
    }
    throw new ApiUnauthorizedError();
  }

  const payload = await parseResponse(response);

  if (!response.ok) {
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `API request failed with status ${response.status}.`;
    throw new ApiError(message, {
      code: responseErrorCode(payload),
      status: response.status,
      details: responseErrorDetails(
        payload,
        response.headers.get("X-Request-ID"),
      ),
    });
  }

  emitPointsAwardFromResponsePayload(payload);
  return extractEnvelope<T>(
    payload,
    response.status,
    response.headers.get("X-Request-ID"),
  );
}

export async function apiFetch<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<ApiEnvelope<T>> {
  const data = await apiRequest<T>(path, options);
  return { code: 200, message: "success", data };
}

export async function apiRequestEnvelope<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<ApiEnvelopeWithExtras<T>> {
  const {
    query,
    headers: optionHeaders,
    auth = true,
    context,
    redirectOnUnauthorized = true,
    retryOnUnauthorized = true,
    signal: optionSignal,
    timeoutMs = DEFAULT_API_TIMEOUT_MS,
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);
  const token = await resolveMutationTokenOrRedirect(
    auth,
    getRequestAuthorizationToken(context, auth),
    fetchOptions.method,
    retryOnUnauthorized,
    redirectOnUnauthorized,
  );

  applyAuthorizationHeader(requestHeaders, token);
  applyForwardedHeaders(requestHeaders, context);

  if (context?.cookie && !requestHeaders.has("cookie")) {
    requestHeaders.set("cookie", context.cookie);
  }

  const isFormData = fetchOptions.body instanceof FormData;
  if (fetchOptions.body && !isFormData && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    credentials: "include",
    headers: requestHeaders,
    cache: fetchOptions.cache ?? "no-store",
    redirect: fetchOptions.redirect ?? "manual",
    signal: createRequestSignal(timeoutMs, optionSignal, context?.signal),
  });

  throwAccessBlockResponse(response, url);

  if (response.status === 401) {
    if (auth && typeof window !== "undefined") {
      const nextToken = await getUnauthorizedRetryToken(token, retryOnUnauthorized);
      if (nextToken) {
        return apiRequestEnvelope<T>(path, {
          ...options,
          retryOnUnauthorized: false,
          context: { ...context, token: nextToken },
        });
      }
    }

    if (auth && redirectOnUnauthorized) {
      clearSession();
      redirectToLogin();
    }
    throw new ApiUnauthorizedError();
  }

  const payload = await parseResponse(response);

  if (!response.ok) {
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `API request failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  emitPointsAwardFromResponsePayload(payload);
  const envelope = payload as Partial<ApiEnvelope<T>> | null;
  if (!envelope || typeof envelope !== "object") {
    return { code: response.status, message: "success", data: payload as T };
  }

  if ("code" in envelope && envelope.code !== 200) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      code: envelope.code,
      status: response.status,
      details: payload,
    });
  }

  if ("success" in envelope && envelope.success === false) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      status: response.status,
      details: payload,
    });
  }

  if ("data" in envelope) {
    return payload as ApiEnvelopeWithExtras<T>;
  }

  if ("code" in envelope || "success" in envelope) {
    return {
      ...(payload as Record<string, unknown>),
      code: Number(envelope.code ?? response.status),
      message: envelope.message ?? "success",
      data: {} as T,
    };
  }

  return { code: response.status, message: "success", data: payload as T };
}

export function apiGet<T>(
  path: string,
  query?: Record<string, QueryValue>,
  options?: Omit<ApiRequestOptions, "method" | "query">,
) {
  return apiRequest<T>(path, { ...options, method: "GET", query });
}

export function apiPost<T>(
  path: string,
  body?: unknown,
  options?: Omit<ApiRequestOptions, "body" | "method" | "query">,
) {
  return apiRequest<T>(path, {
    ...options,
    method: "POST",
    body: body instanceof FormData ? body : JSON.stringify(body ?? {}),
  });
}

export function apiPut<T>(
  path: string,
  body?: unknown,
  options?: Omit<ApiRequestOptions, "body" | "method" | "query">,
) {
  return apiRequest<T>(path, {
    ...options,
    method: "PUT",
    body: body instanceof FormData ? body : JSON.stringify(body ?? {}),
  });
}

export function apiDelete<T>(
  path: string,
  body?: unknown,
  options?: Omit<ApiRequestOptions, "body" | "method" | "query">,
) {
  return apiRequest<T>(path, {
    ...options,
    method: "DELETE",
    body: body ? JSON.stringify(body) : undefined,
  });
}

export async function apiAdminRequest<T>(
  path: string,
  options: RequestInit & {
    clearSessionOnUnauthorized?: boolean;
    query?: Record<string, QueryValue>;
    token?: string | null;
    timeoutMs?: number;
  } = {},
): Promise<T> {
  const {
    clearSessionOnUnauthorized = false,
    query,
    headers: optionHeaders,
    signal,
    timeoutMs = DEFAULT_ADMIN_API_TIMEOUT_MS,
    token = getStoredAdminAccessToken(),
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);

  applyAuthorizationHeader(requestHeaders, token);

  const isFormData = fetchOptions.body instanceof FormData;
  if (fetchOptions.body && !isFormData && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    credentials: "include",
    headers: requestHeaders,
    cache: fetchOptions.cache ?? "no-store",
    redirect: fetchOptions.redirect ?? "manual",
    signal: createRequestSignal(timeoutMs, signal),
  });

  throwAccessBlockResponse(response, url);

  if (response.status === 401) {
    if (clearSessionOnUnauthorized) {
      clearAdminSession();
    }
    throw new ApiUnauthorizedError("Admin authentication is required.");
  }

  const payload = await parseResponse(response);

  if (!response.ok) {
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `API request failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  return extractEnvelope<T>(payload, response.status);
}

export async function apiAdminEnvelope<T>(
  path: string,
  options: RequestInit & {
    clearSessionOnUnauthorized?: boolean;
    query?: Record<string, QueryValue>;
    token?: string | null;
    timeoutMs?: number;
  } = {},
): Promise<ApiEnvelopeWithExtras<T>> {
  const {
    clearSessionOnUnauthorized = false,
    query,
    headers: optionHeaders,
    signal,
    timeoutMs = DEFAULT_ADMIN_API_TIMEOUT_MS,
    token = getStoredAdminAccessToken(),
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);

  applyAuthorizationHeader(requestHeaders, token);

  const isFormData = fetchOptions.body instanceof FormData;
  if (fetchOptions.body && !isFormData && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    credentials: "include",
    headers: requestHeaders,
    cache: fetchOptions.cache ?? "no-store",
    redirect: fetchOptions.redirect ?? "manual",
    signal: createRequestSignal(timeoutMs, signal),
  });

  throwAccessBlockResponse(response, url);

  if (response.status === 401) {
    if (clearSessionOnUnauthorized) {
      clearAdminSession();
    }
    throw new ApiUnauthorizedError("Admin authentication is required.");
  }

  const payload = await parseResponse(response);

  if (!response.ok) {
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `API request failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  const envelope = payload as Partial<ApiEnvelope<T>> | null;
  if (!envelope || typeof envelope !== "object") {
    return { code: response.status, message: "success", data: payload as T };
  }

  if ("code" in envelope && envelope.code !== 200) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      code: envelope.code,
      status: response.status,
      details: payload,
    });
  }

  if ("success" in envelope && envelope.success === false) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      status: response.status,
      details: payload,
    });
  }

  if ("data" in envelope) {
    return payload as ApiEnvelopeWithExtras<T>;
  }

  if ("code" in envelope || "success" in envelope) {
    return {
      ...(payload as Record<string, unknown>),
      code: Number(envelope.code ?? response.status),
      message: envelope.message ?? "success",
      data: {} as T,
    };
  }

  return { code: response.status, message: "success", data: payload as T };
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

export function numberFromUnknown(value: unknown, fallback = 0) {
  const numeric = typeof value === "number" ? value : Number(value);
  return Number.isFinite(numeric) ? numeric : fallback;
}

export function normalizeAdminPagination(
  source: unknown,
  fallback: { page: number; limit: number; itemCount: number },
) {
  const record = isRecord(source) ? source : {};
  const page = numberFromUnknown(record.page, fallback.page);
  const limit = numberFromUnknown(
    record.limit ?? record.pageSize,
    fallback.limit,
  );
  const total =
    record.total === undefined ? undefined : numberFromUnknown(record.total);
  const rawPages = record.pages ?? record.totalPages;
  const pages =
    rawPages === undefined
      ? total !== undefined && limit > 0
        ? Math.ceil(total / limit)
        : undefined
      : numberFromUnknown(rawPages);
  const hasNextPage =
    pages !== undefined
      ? page < pages
      : total !== undefined && limit > 0
        ? page * limit < total
        : fallback.itemCount >= limit;

  return {
    page,
    limit,
    pageSize: limit,
    total,
    pages,
    totalPages: pages,
    hasNextPage,
  };
}

export function normalizeAdminListEnvelope<T extends AdminListRow>(
  envelope: ApiEnvelopeWithExtras<unknown>,
  fallback: { page: number; limit: number },
): AdminListPayload<T> {
  const data = envelope.data;
  const dataRecord = isRecord(data) ? data : null;
  const itemsSource = Array.isArray(data)
    ? data
    : Array.isArray(dataRecord?.data)
      ? dataRecord.data
      : Array.isArray(dataRecord?.list)
        ? dataRecord.list
        : Array.isArray(dataRecord?.items)
          ? dataRecord.items
          : [];
  const items = itemsSource as T[];
  const topPagination = isRecord(envelope.pagination)
    ? envelope.pagination
    : null;
  const nestedPagination = isRecord(dataRecord?.pagination)
    ? dataRecord.pagination
    : null;
  const dataAsPagination =
    dataRecord &&
    ("total" in dataRecord || "page" in dataRecord || "limit" in dataRecord)
      ? dataRecord
      : null;

  return {
    items,
    pagination: normalizeAdminPagination(
      topPagination ?? nestedPagination ?? dataAsPagination,
      {
        ...fallback,
        itemCount: items.length,
      },
    ),
  };
}
