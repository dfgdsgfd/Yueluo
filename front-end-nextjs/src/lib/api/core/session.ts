import type { AdminSession,AdminUser } from "../../types/admin";
import type { AuthTokens,AuthUser } from "../../types/auth";
import type { ApiRequestContext } from "./contracts";
import { ACCESS_TOKEN_COOKIE,ACCESS_TOKEN_COOKIE_FALLBACKS,ACCESS_TOKEN_KEY,ADMIN_ACCESS_TOKEN_KEY,ADMIN_REFRESH_TOKEN_KEY,ADMIN_USER_KEY,AUTH_COOKIE_MAX_AGE_SECONDS,AUTH_USER_EVENT,HTTP_ACCESS_TOKEN_COOKIE,HTTP_REFRESH_TOKEN_COOKIE,REFRESH_TOKEN_COOKIE,REFRESH_TOKEN_COOKIE_FALLBACKS,REFRESH_TOKEN_KEY,USER_KEY } from "./contracts";
import { getCookieValue,getCookieValueFromNames,normalizeAuthToken } from "./http";

let memoryAccessToken: string | null = null;

let memoryRefreshToken: string | null = null;

let memoryAuthenticatedUser: AuthUser | null = null;

export function canUseStorage() {
  if (typeof window === "undefined") {
    return false;
  }
  try {
    return Boolean(window.localStorage);
  } catch {
    return false;
  }
}

export function getRequestAccessToken(context?: ApiRequestContext) {
  return (
    normalizeAuthToken(context?.token) ??
    normalizeAuthToken(getCookieValue(context?.cookie, HTTP_ACCESS_TOKEN_COOKIE)) ??
    normalizeAuthToken(getCookieValue(context?.cookie, ACCESS_TOKEN_COOKIE)) ??
    normalizeAuthToken(getCookieValueFromNames(context?.cookie, ACCESS_TOKEN_COOKIE_FALLBACKS)) ??
    getStoredAccessToken()
  );
}

export function getRequestAuthorizationToken(context: ApiRequestContext | undefined, auth: boolean) {
  return normalizeAuthToken(context?.token) ?? (auth ? getRequestAccessToken(context) : null);
}

export function getBrowserCookieValue(name: string) {
  if (typeof document === "undefined") {
    return null;
  }

  return getCookieValue(document.cookie, name);
}

export function getBrowserCookieValueFromNames(names: readonly string[]) {
  for (const name of names) {
    const value = getBrowserCookieValue(name);
    if (value) {
      return value;
    }
  }

  return null;
}

export function persistAccessTokenCookie(accessToken: string) {
  if (typeof document === "undefined") {
    return;
  }

  const secure = window.location.protocol === "https:" ? "; secure" : "";
  document.cookie = `${ACCESS_TOKEN_COOKIE}=${encodeURIComponent(
    accessToken,
  )}; path=/; max-age=${AUTH_COOKIE_MAX_AGE_SECONDS}; samesite=lax${secure}`;
}

export function clearAccessTokenCookie() {
  if (typeof document === "undefined") {
    return;
  }

  const secure = window.location.protocol === "https:" ? "; secure" : "";
  for (const name of [
    ACCESS_TOKEN_COOKIE,
    HTTP_ACCESS_TOKEN_COOKIE,
    HTTP_REFRESH_TOKEN_COOKIE,
    REFRESH_TOKEN_COOKIE,
    ...ACCESS_TOKEN_COOKIE_FALLBACKS,
    ...REFRESH_TOKEN_COOKIE_FALLBACKS,
  ]) {
    document.cookie = `${name}=; path=/; max-age=0; samesite=lax${secure}`;
  }
}

export function getStoredRefreshToken() {
  const cookieToken =
    normalizeAuthToken(getBrowserCookieValue(REFRESH_TOKEN_COOKIE)) ??
    normalizeAuthToken(getBrowserCookieValueFromNames(REFRESH_TOKEN_COOKIE_FALLBACKS));
  if (cookieToken) {
    return cookieToken;
  }

  if (!canUseStorage()) {
    return memoryRefreshToken;
  }

  return normalizeAuthToken(window.localStorage.getItem(REFRESH_TOKEN_KEY)) ?? memoryRefreshToken;
}

export function getStoredAccessToken() {
  const cookieToken =
    normalizeAuthToken(getBrowserCookieValue(ACCESS_TOKEN_COOKIE)) ??
    normalizeAuthToken(getBrowserCookieValueFromNames(ACCESS_TOKEN_COOKIE_FALLBACKS));
  if (cookieToken) {
    return cookieToken;
  }

  if (canUseStorage()) {
    const storedToken = normalizeAuthToken(window.localStorage.getItem(ACCESS_TOKEN_KEY));
    if (storedToken) {
      return storedToken;
    }
  }

  return memoryAccessToken;
}

export function getStoredAdminAccessToken() {
  if (!canUseStorage()) {
    return null;
  }

  return window.localStorage.getItem(ADMIN_ACCESS_TOKEN_KEY);
}

export function getStoredUser(): AuthUser | null {
  const raw = canUseStorage() ? window.localStorage.getItem(USER_KEY) : null;
  if (!raw) {
    return memoryAuthenticatedUser;
  }

  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return memoryAuthenticatedUser;
  }
}

export function getStoredAdminUser(): AdminUser | null {
  if (!canUseStorage()) {
    return null;
  }

  const raw = window.localStorage.getItem(ADMIN_USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as AdminUser;
  } catch {
    return null;
  }
}

function dispatchAuthenticatedUserUpdated(user: AuthUser | null) {
  if (typeof window === "undefined") {
    return;
  }
  window.dispatchEvent(new CustomEvent<AuthUser | null>(AUTH_USER_EVENT, { detail: user }));
}

export function storeOAuthCallbackTokens(input: {
  accessToken: string;
  refreshToken?: string | null;
  user?: AuthUser | null;
}) {
  if (typeof window === "undefined") {
    return;
  }

  memoryAccessToken = input.accessToken;
  memoryRefreshToken = input.refreshToken ?? null;
  memoryAuthenticatedUser = input.user ?? null;
  persistAccessTokenCookie(input.accessToken);
  if (canUseStorage()) {
    window.localStorage.setItem(ACCESS_TOKEN_KEY, input.accessToken);
    if (input.refreshToken) {
      window.localStorage.setItem(REFRESH_TOKEN_KEY, input.refreshToken);
    } else {
      window.localStorage.removeItem(REFRESH_TOKEN_KEY);
    }
    if (input.user) {
      window.localStorage.setItem(USER_KEY, JSON.stringify(input.user));
    } else {
      window.localStorage.removeItem(USER_KEY);
    }
  }
  dispatchAuthenticatedUserUpdated(input.user ?? null);
}

export function storeAuthenticatedUser(user: AuthUser | null) {
  if (typeof window === "undefined") {
    return;
  }

  memoryAuthenticatedUser = user;
  if (canUseStorage()) {
    if (user) {
      window.localStorage.setItem(USER_KEY, JSON.stringify(user));
    } else {
      window.localStorage.removeItem(USER_KEY);
    }
  }
  dispatchAuthenticatedUserUpdated(user);
}

export function storeAdminSession(session: AdminSession) {
  if (!canUseStorage()) {
    return;
  }

  window.localStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, session.tokens.accessToken);
  window.localStorage.setItem(ADMIN_REFRESH_TOKEN_KEY, session.tokens.refreshToken);
  window.localStorage.setItem(ADMIN_USER_KEY, JSON.stringify(session.admin));
}

export function clearSession() {
  if (typeof window === "undefined") {
    return;
  }

  memoryAccessToken = null;
  memoryRefreshToken = null;
  memoryAuthenticatedUser = null;
  if (canUseStorage()) {
    window.localStorage.removeItem(ACCESS_TOKEN_KEY);
    window.localStorage.removeItem(REFRESH_TOKEN_KEY);
    window.localStorage.removeItem(USER_KEY);
  }
  clearAccessTokenCookie();
  dispatchAuthenticatedUserUpdated(null);
}

export function clearAdminSession() {
  if (!canUseStorage()) {
    return;
  }

  window.localStorage.removeItem(ADMIN_ACCESS_TOKEN_KEY);
  window.localStorage.removeItem(ADMIN_REFRESH_TOKEN_KEY);
  window.localStorage.removeItem(ADMIN_USER_KEY);
}

export function normalizeTokens(raw: Record<string, unknown> | undefined): AuthTokens {
  return {
    accessToken: String(raw?.access_token ?? raw?.token ?? ""),
    refreshToken: String(raw?.refresh_token ?? ""),
    expiresIn: typeof raw?.expires_in === "number" ? raw.expires_in : undefined,
  };
}

export function normalizeAdminSession(data: { admin?: AdminUser; tokens?: Record<string, unknown> }): AdminSession {
  return {
    admin: data.admin ?? { id: "", username: "" },
    tokens: normalizeTokens(data.tokens),
  };
}
