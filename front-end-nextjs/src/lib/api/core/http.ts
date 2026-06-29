import type { ApiRequestContext,QueryValue } from "./contracts";
import { DEFAULT_SERVER_API_ORIGIN } from "./contracts";

export function getApiBaseUrl() {
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_BASE_URL ?? "";
  }

  return (
    process.env.API_BASE_URL ??
    process.env.BACKEND_ORIGIN ??
    process.env.NEXT_PUBLIC_BACKEND_ORIGIN ??
    DEFAULT_SERVER_API_ORIGIN
  ).replace(/\/$/, "");
}

export function buildApiUrl(path: string, query?: Record<string, QueryValue>) {
  const apiBaseUrl = getApiBaseUrl();
  const url = new URL(path, apiBaseUrl || window.location.origin);

  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== null && value !== undefined && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }

  if (typeof window !== "undefined" && !apiBaseUrl) {
    return `${url.pathname}${url.search}`;
  }

  return url.toString();
}

export function normalizeAuthToken(token: string | null | undefined) {
  const trimmed = token?.trim();
  if (!trimmed) {
    return null;
  }

  return trimmed.replace(/^Bearer\s+/i, "").trim() || null;
}

export function applyAuthorizationHeader(headers: Headers, token: string | null | undefined) {
  const normalizedToken = normalizeAuthToken(token);
  if (normalizedToken && !headers.has("authorization")) {
    headers.set("authorization", `Bearer ${normalizedToken}`);
  }
}

export function createRequestSignal(
  timeoutMs: number,
  ...signals: Array<AbortSignal | null | undefined>
) {
  const activeSignals = signals.filter(
    (signal): signal is AbortSignal => Boolean(signal),
  );
  if (timeoutMs > 0) {
    activeSignals.push(createTimeoutSignal(timeoutMs));
  }
  if (activeSignals.length === 0) {
    return undefined;
  }
  if (activeSignals.length === 1) {
    return activeSignals[0];
  }
  if (typeof AbortSignal.any === "function") {
    return AbortSignal.any(activeSignals);
  }

  const controller = new AbortController();
  const abort = (signal: AbortSignal) => {
    for (const activeSignal of activeSignals) {
      const listener = listeners.get(activeSignal);
      if (listener) {
        activeSignal.removeEventListener("abort", listener);
      }
    }
    controller.abort(signal.reason);
  };
  const listeners = new Map<AbortSignal, () => void>();

  for (const signal of activeSignals) {
    if (signal.aborted) {
      abort(signal);
      break;
    }
    const listener = () => abort(signal);
    listeners.set(signal, listener);
    signal.addEventListener("abort", listener, { once: true });
  }

  return controller.signal;
}

function createTimeoutSignal(timeoutMs: number) {
  if (typeof AbortSignal.timeout === "function") {
    return AbortSignal.timeout(timeoutMs);
  }
  const controller = new AbortController();
  setTimeout(() => controller.abort(), timeoutMs);
  return controller.signal;
}

export function createAbortError() {
  return new DOMException("The operation was aborted.", "AbortError");
}

const defaultClientIPHeaderNames = [
  "x-forwarded-for",
  "x-real-ip",
  "cf-connecting-ip",
  "x-custom-real-ip",
  "true-client-ip",
  "x-client-ip",
  "forwarded",
] as const;

const alwaysForwardHeaderNames = [
  "user-agent",
  "accept-language",
] as const;

const headerNamePattern = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;

type HeaderReader = {
  get(name: string): string | null;
};

export function apiRequestContextFromHeaders(headers: HeaderReader): ApiRequestContext {
  const forwardedHeaders: Record<string, string> = {};
  for (const name of serverForwardHeaderNames()) {
    const value = headers.get(name)?.trim();
    if (value) {
      forwardedHeaders[name] = value;
    }
  }
  return {
    cookie: headers.get("cookie") ?? undefined,
    forwardedHeaders,
  };
}

function serverForwardHeaderNames() {
  return uniqueHeaderNames([
    ...alwaysForwardHeaderNames,
    ...clientIPHeaderNamesFromEnv(process.env.CLIENT_IP_HEADERS),
  ]);
}

function clientIPHeaderNamesFromEnv(value: string | undefined) {
  const configured = splitHeaderNames(value);
  return configured.length > 0 ? configured : [...defaultClientIPHeaderNames];
}

function splitHeaderNames(value: string | undefined) {
  return uniqueHeaderNames(
    (value ?? "")
      .split(",")
      .map((name) => name.trim().toLowerCase())
      .filter((name) => name && headerNamePattern.test(name)),
  );
}

function uniqueHeaderNames(names: readonly string[]) {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const name of names) {
    if (!seen.has(name)) {
      seen.add(name);
      out.push(name);
    }
  }
  return out;
}

export function applyForwardedHeaders(headers: Headers, context?: ApiRequestContext) {
  for (const [name, value] of Object.entries(context?.forwardedHeaders ?? {})) {
    if (!value || headers.has(name)) {
      continue;
    }
    headers.set(name, value);
  }
  if (!headers.has("x-app-locale")) {
    const cookieHeader = context?.cookie ?? (typeof document !== "undefined" ? document.cookie : undefined);
    const locale = getCookieValue(cookieHeader, "xse.locale");
    if (locale) {
      headers.set("x-app-locale", locale);
    }
  }
}

export function getCookieValue(cookieHeader: string | undefined, name: string) {
  if (!cookieHeader) {
    return null;
  }

  for (const part of cookieHeader.split(";")) {
    const [rawKey, ...rawValue] = part.trim().split("=");
    if (rawKey !== name) {
      continue;
    }

    const value = rawValue.join("=");
    try {
      return decodeURIComponent(value);
    } catch {
      return value;
    }
  }

  return null;
}

export function getCookieValueFromNames(cookieHeader: string | undefined, names: readonly string[]) {
  for (const name of names) {
    const value = getCookieValue(cookieHeader, name);
    if (value) {
      return value;
    }
  }

  return null;
}
