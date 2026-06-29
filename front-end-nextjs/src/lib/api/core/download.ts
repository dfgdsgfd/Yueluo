import type { DownloadPayload } from "../../types/content";
import type { ApiRequestOptions, QueryValue } from "./contracts";
import { throwAccessBlockResponse } from "./access-block";
import { ApiError,ApiUnauthorizedError,DEFAULT_DOWNLOAD_TIMEOUT_MS } from "./contracts";
import { applyAuthorizationHeader,applyForwardedHeaders,buildApiUrl,createRequestSignal } from "./http";
import { redirectToLogin } from "./request";
import { filenameFromContentDisposition,parseResponse } from "./response";
import { clearAdminSession,clearSession,getRequestAuthorizationToken,getStoredAdminAccessToken } from "./session";

export async function apiDownload(
  path: string,
  options: ApiRequestOptions = {},
): Promise<DownloadPayload> {
  const {
    query,
    headers: optionHeaders,
    auth = true,
    context,
    signal,
    timeoutMs = DEFAULT_DOWNLOAD_TIMEOUT_MS,
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);
  const token = getRequestAuthorizationToken(context, auth);

  applyAuthorizationHeader(requestHeaders, token);
  applyForwardedHeaders(requestHeaders, context);

  if (context?.cookie && !requestHeaders.has("cookie")) {
    requestHeaders.set("cookie", context.cookie);
  }

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    method: fetchOptions.method ?? "GET",
    credentials: "include",
    headers: requestHeaders,
    cache: fetchOptions.cache ?? "no-store",
    redirect: fetchOptions.redirect ?? "manual",
    signal: createRequestSignal(timeoutMs, signal, context?.signal),
  });

  throwAccessBlockResponse(response, url);

  if (response.status === 401) {
    if (auth) {
      clearSession();
      redirectToLogin();
    }
    throw new ApiUnauthorizedError();
  }

  if (!response.ok) {
    const payload = await parseResponse(response);
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `Download failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  const contentDisposition = response.headers.get("content-disposition");
  return {
    blob: await response.blob(),
    contentDisposition,
    contentType: response.headers.get("content-type"),
    filename: filenameFromContentDisposition(contentDisposition),
  };
}

export async function apiAdminDownload(
  path: string,
  options: RequestInit & {
    clearSessionOnUnauthorized?: boolean;
    query?: Record<string, QueryValue>;
    token?: string | null;
    timeoutMs?: number;
  } = {},
): Promise<DownloadPayload> {
  const {
    clearSessionOnUnauthorized = false,
    query,
    headers: optionHeaders,
    signal,
    timeoutMs = DEFAULT_DOWNLOAD_TIMEOUT_MS,
    token = getStoredAdminAccessToken(),
    ...fetchOptions
  } = options;
  const requestHeaders = new Headers(optionHeaders);
  applyAuthorizationHeader(requestHeaders, token);

  const url = buildApiUrl(path, query);
  const response = await fetch(url, {
    ...fetchOptions,
    method: fetchOptions.method ?? "GET",
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

  if (!response.ok) {
    const payload = await parseResponse(response);
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `Download failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  const contentDisposition = response.headers.get("content-disposition");
  return {
    blob: await response.blob(),
    contentDisposition,
    contentType: response.headers.get("content-type"),
    filename: filenameFromContentDisposition(contentDisposition),
  };
}
