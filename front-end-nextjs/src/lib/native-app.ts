"use client";

import { App } from "@capacitor/app";
import { Browser } from "@capacitor/browser";
import { Capacitor } from "@capacitor/core";
import { Share } from "@capacitor/share";
import { nativeAppOrigin } from "./native-oauth-links";

export {
  buildNativeOAuthDeepLinks,
  nativeAppOrigin,
  nativeOAuthCallbackPath,
  selectNativeOAuthAutoOpenLink,
} from "./native-oauth-links";

const oauthCallbackClaimsKey = "yuem_android_oauth_callback_claims";
const oauthStatusKey = "yuem_android_oauth_status";
const oauthMaxAgeMs = 10 * 60 * 1000;
const oauthCallbackClaimMaxAgeMs = 24 * 60 * 60 * 1000;
const oauthCallbackClaimLimit = 12;
const nativeOAuthFallbackPath = "/explore";
const nativeOAuthScheme =
  process.env.NEXT_PUBLIC_YUEM_MOBILE_CALLBACK_SCHEME?.trim().replace(/:.*$/u, "") ||
  "xsewebfast";
const nativeOAuthCallbackHost = "auth-return";
const nativeOAuthStatusMessageKeys = {
  callback_received: "nativeStatus.callbackReceived",
  missing_code: "nativeStatus.missingCode",
  oauth_error: "nativeStatus.oauthError",
  profile_failed: "nativeStatus.profileFailed",
  signed_in: "nativeStatus.signedIn",
  state_failed: "nativeStatus.stateFailed",
  token_exchange_failed: "nativeStatus.tokenExchangeFailed",
  token_storage_failed: "nativeStatus.tokenStorageFailed",
} as const satisfies Record<NativeOAuthStatusStep, string>;

export type NativeOAuthCallback = {
  error: string;
  returnUrl: string;
  ticket: string;
};

export type NativeOAuthStatusStep =
  | "callback_received"
  | "state_failed"
  | "oauth_error"
  | "missing_code"
  | "token_exchange_failed"
  | "token_storage_failed"
  | "profile_failed"
  | "signed_in";

export type NativeOAuthStatus = {
  at: number;
  detail?: string;
  ok: boolean;
  step: NativeOAuthStatusStep;
};

type NativeOAuthCallbackClaim = {
  at: number;
  key: string;
};

let memoryOAuthStatus: NativeOAuthStatus | null = null;
let memoryOAuthCallbackClaims: NativeOAuthCallbackClaim[] = [];

export function isNativeAndroidApp() {
  return Capacitor.isNativePlatform() && Capacitor.getPlatform() === "android";
}

export async function startNativeOAuth(
  startUrl = "/api/auth/oauth2/login",
  returnUrl =
    typeof window === "undefined"
      ? new URL(nativeOAuthFallbackPath, nativeAppOrigin).toString()
      : resolveNativeOAuthDefaultReturnUrl(window.location.href),
) {
  if (!isNativeAndroidApp()) {
    return false;
  }

  clearNativeOAuthStatus();
  const loginUrl = buildNativeOAuthStartUrl(startUrl, returnUrl);
  await Browser.open({ url: loginUrl.toString() });
  return true;
}

export function buildNativeOAuthStartUrl(
  startUrl: string,
  returnUrl: string,
) {
  const loginUrl = new URL(startUrl, nativeAppOrigin);
  const safeReturnUrl = sanitizeNativeOAuthReturnUrl(returnUrl);
  const callbackUrl = new URL(`${nativeOAuthScheme}://${nativeOAuthCallbackHost}`);
  callbackUrl.searchParams.set("url", safeReturnUrl);
  loginUrl.searchParams.set(
    "app_callback",
    callbackUrl.toString(),
  );
  loginUrl.searchParams.set(
    "app_return_url",
    safeReturnUrl,
  );
  return loginUrl;
}

export function parseNativeOAuthCallback(rawUrl: string): NativeOAuthCallback | null {
  let parsed: URL;
  try {
    parsed = new URL(rawUrl);
  } catch {
    return null;
  }

  if (parsed.protocol !== `${nativeOAuthScheme}:`) {
    return null;
  }

  const explicitUrl =
    parsed.searchParams.get("url") ??
    parsed.searchParams.get("target") ??
    parsed.searchParams.get("next") ??
    "";
  return {
    error: parsed.searchParams.get("error") ?? "",
    returnUrl: sanitizeNativeOAuthReturnUrl(
      explicitUrl || nativeOAuthPathReturnUrl(parsed),
    ),
    ticket: parsed.searchParams.get("ticket") ?? "",
  };
}

export function sanitizeNativeOAuthReturnUrl(rawUrl: string) {
  const fallback = new URL(nativeOAuthFallbackPath, nativeAppOrigin).toString();
  let parsed: URL;
  try {
    parsed = new URL(rawUrl || fallback, nativeAppOrigin);
  } catch {
    return fallback;
  }
  if (!nativeOAuthAllowedOrigins().has(parsed.origin)) {
    return fallback;
  }
  parsed.username = "";
  parsed.password = "";
  return parsed.toString();
}

export function resolveNativeOAuthDefaultReturnUrl(rawUrl: string) {
  const fallback = new URL(nativeOAuthFallbackPath, nativeAppOrigin).toString();
  const safeCurrentUrl = sanitizeNativeOAuthReturnUrl(rawUrl || fallback);
  let currentUrl: URL;
  try {
    currentUrl = new URL(safeCurrentUrl);
  } catch {
    return fallback;
  }

  if (!isNativeOAuthLoginUrl(currentUrl)) {
    return currentUrl.toString();
  }

  for (const name of ["next", "return_url", "returnUrl", "redirect"]) {
    const candidate = currentUrl.searchParams.get(name);
    if (!candidate) continue;
    const safeCandidate = sanitizeNativeOAuthReturnUrl(candidate);
    try {
      const parsedCandidate = new URL(safeCandidate);
      if (!isNativeOAuthLoginUrl(parsedCandidate)) {
        return parsedCandidate.toString();
      }
    } catch {
      // Ignore invalid candidates and keep looking for a safe fallback.
    }
  }

  return fallback;
}

export function claimNativeOAuthCallback(callback: NativeOAuthCallback, now = Date.now()) {
  const key = nativeOAuthCallbackClaimKey(callback);
  const claims = readOAuthCallbackClaims(now);
  if (claims.some((claim) => claim.key === key)) {
    return false;
  }
  const nextClaims = [...claims, { at: now, key }].slice(-oauthCallbackClaimLimit);
  memoryOAuthCallbackClaims = nextClaims;
  writeJSON(oauthCallbackClaimsKey, nextClaims);
  return true;
}

export function recordNativeOAuthStatus(
  status: Omit<NativeOAuthStatus, "at">,
  now = Date.now(),
) {
  const nextStatus: NativeOAuthStatus = { ...status, at: now };
  memoryOAuthStatus = nextStatus;
  writeJSON(oauthStatusKey, nextStatus);
}

export function readNativeOAuthStatus(now = Date.now()) {
  const status = readJSON<NativeOAuthStatus>(oauthStatusKey) ?? memoryOAuthStatus;
  if (!isNativeOAuthStatus(status)) {
    return null;
  }
  if (now - status.at > oauthMaxAgeMs) {
    clearNativeOAuthStatus();
    return null;
  }
  return status;
}

export function clearNativeOAuthStatus() {
  memoryOAuthStatus = null;
  removeStorageItem(oauthStatusKey);
}

export function nativeOAuthStatusMessageKey(step: string) {
  if (step in nativeOAuthStatusMessageKeys) {
    return nativeOAuthStatusMessageKeys[step as NativeOAuthStatusStep];
  }
  return "nativeStatus.unknown";
}

export function describeNativeOAuthError(error: unknown) {
  if (!error) {
    return "unknown_error";
  }

  if (error instanceof Error) {
    return describeErrorRecord({
      message: error.message,
      status: (error as { status?: unknown }).status,
      code: (error as { code?: unknown }).code,
    });
  }

  if (typeof error === "object") {
    return describeErrorRecord(error as Record<string, unknown>);
  }

  return String(error);
}

export async function getNativeLaunchUrl() {
  if (!isNativeAndroidApp()) {
    return null;
  }
  return (await App.getLaunchUrl())?.url ?? null;
}

export async function shareNativeUrl(input: { title?: string; text?: string; url: string }) {
  if (!isNativeAndroidApp()) {
    return false;
  }
  const availability = await Share.canShare();
  if (!availability.value) {
    return false;
  }
  await Share.share(input);
  return true;
}

export async function openNativeBrowser(url: string) {
  if (!isNativeAndroidApp()) {
    return false;
  }
  await Browser.open({ url });
  return true;
}

function describeErrorRecord(record: {
  code?: unknown;
  message?: unknown;
  status?: unknown;
}) {
  const message = String(record.message || "unknown_error");
  const status = Number(record.status);
  const code = Number(record.code);
  const details: string[] = [];
  if (Number.isFinite(status) && status > 0) {
    details.push("HTTP " + status);
  }
  if (Number.isFinite(code) && code > 0) {
    details.push("code " + code);
  }
  return details.length ? message + " (" + details.join(", ") + ")" : message;
}

function nativeOAuthCallbackClaimKey(callback: NativeOAuthCallback) {
  return [callback.ticket, callback.error, callback.returnUrl].join("\n");
}

function nativeOAuthPathReturnUrl(parsed: URL) {
  const pathPrefix = parsed.hostname ? `/${parsed.hostname}${parsed.pathname}` : parsed.pathname;
  const path = `${pathPrefix}${parsed.search}${parsed.hash}`;
  return path && path !== "/" ? path : "";
}

function nativeOAuthAllowedOrigins() {
  const origins = new Set<string>([new URL(nativeAppOrigin).origin]);
  const configuredHosts = (
    process.env.NEXT_PUBLIC_YUEM_IN_APP_HOSTS ?? "xse.yuelk.com,cs.yuelk.com"
  )
    .split(",")
    .map((host) => host.trim())
    .filter(Boolean);
  for (const host of configuredHosts) {
    try {
      const candidate = host.includes("://") ? host : `https://${host}`;
      origins.add(new URL(candidate).origin);
    } catch {
      // Invalid build-time host entries are ignored.
    }
  }
  if (
    typeof window !== "undefined" &&
    window.location?.hostname &&
    configuredHosts.includes(window.location.hostname)
  ) {
    origins.add(window.location.origin);
  }
  return origins;
}

function isNativeOAuthLoginUrl(url: URL) {
  return (url.pathname.replace(/\/+$/u, "") || "/") === "/login";
}

function readOAuthCallbackClaims(now: number) {
  const storedClaims =
    readJSON<NativeOAuthCallbackClaim[]>(oauthCallbackClaimsKey) ??
    memoryOAuthCallbackClaims;
  const claims = Array.isArray(storedClaims) ? storedClaims : [];
  const activeClaims = claims.filter(
    (claim) =>
      claim &&
      typeof claim.key === "string" &&
      typeof claim.at === "number" &&
      Number.isFinite(claim.at) &&
      now - claim.at <= oauthCallbackClaimMaxAgeMs,
  );
  memoryOAuthCallbackClaims = activeClaims;
  if (activeClaims.length !== claims.length) {
    writeJSON(oauthCallbackClaimsKey, activeClaims);
  }
  return activeClaims;
}

function isNativeOAuthStatus(value: unknown): value is NativeOAuthStatus {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return false;
  }
  const record = value as Partial<NativeOAuthStatus>;
  return (
    typeof record.at === "number" &&
    Number.isFinite(record.at) &&
    typeof record.ok === "boolean" &&
    typeof record.step === "string"
  );
}

function canUseLocalStorage() {
  if (typeof window === "undefined") {
    return false;
  }
  try {
    return Boolean(window.localStorage);
  } catch {
    return false;
  }
}

function readJSON<T>(key: string) {
  if (!canUseLocalStorage()) {
    return null;
  }
  const raw = window.localStorage.getItem(key);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

function writeJSON(key: string, value: unknown) {
  if (!canUseLocalStorage()) {
    return;
  }
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Keep the in-memory fallback so the current WebView session can still
    // report the latest OAuth state even when persistent storage is blocked.
  }
}

function removeStorageItem(key: string) {
  if (!canUseLocalStorage()) {
    return;
  }
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Storage removal is best-effort; stale diagnostics are also age-limited.
  }
}
