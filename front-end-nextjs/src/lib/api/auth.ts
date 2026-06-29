import type {
  AdminUser,
  AuthConfigPayload,
  AuthTokens,
  CaptchaPayload,
  OAuthAppTokenPayload,
  AuthUser,
  BackendApiSpec
} from "../types";
import {
  ApiError,
  ApiRequestContext,
  DEFAULT_SWAGGER_DOCS_PATH,
  apiAdminRequest,
  apiGet,
  apiPost,
  clearAdminSession,
  clearSession,
  extractEnvelope,
  normalizeAdminSession,
  normalizeTokens,
  parseResponse,
  storeAdminSession,
  storeOAuthCallbackTokens
} from "./core";

export async function loginAdmin(username: string, password: string) {
  const session = normalizeAdminSession(
    await apiPost<{ admin: AdminUser; tokens: Record<string, unknown> }>(
      "/api/auth/admin/login",
      { username, password },
      { auth: false },
    ),
  );
  storeAdminSession(session);
  return session;
}

type UserAuthSession = {
  tokens: AuthTokens;
  user: AuthUser;
};

type RawUserAuthSessionPayload = {
  tokens?: Record<string, unknown>;
  user?: AuthUser;
};

type RegisterUserInput = {
  captchaId?: string;
  captchaText?: string;
  email?: string;
  emailCode?: string;
  nickname: string;
  password: string;
  userID: string;
};

export async function loginUser(identifier: string, password: string) {
  const session = normalizeUserAuthSession(
    await apiPost<RawUserAuthSessionPayload>(
      "/api/auth/login",
      {
        identifier,
        user_id: identifier,
        password,
      },
      { auth: false },
    ),
  );
  storeUserAuthSession(session);
  return session;
}

export async function registerUser(input: RegisterUserInput) {
  const session = normalizeUserAuthSession(
    await apiPost<RawUserAuthSessionPayload>(
      "/api/auth/register",
      {
        user_id: input.userID,
        nickname: input.nickname,
        password: input.password,
        email: input.email,
        emailCode: input.emailCode,
        captchaId: input.captchaId,
        captchaText: input.captchaText,
      },
      { auth: false },
    ),
  );
  storeUserAuthSession(session);
  return session;
}

export function getAuthCaptcha() {
  return apiGet<CaptchaPayload>("/api/auth/captcha", undefined, {
    auth: false,
  });
}

export function sendAuthEmailCode(email: string) {
  return apiPost<Record<string, unknown>>(
    "/api/auth/send-email-code",
    { email },
    { auth: false },
  );
}

function normalizeUserAuthSession(data: RawUserAuthSessionPayload): UserAuthSession {
  const tokens = normalizeTokens(data.tokens);
  if (!tokens.accessToken || !tokens.refreshToken || !data.user) {
    throw new ApiError("error.auth_session_invalid");
  }

  return { tokens, user: data.user };
}

function storeUserAuthSession(session: UserAuthSession) {
  storeOAuthCallbackTokens({
    accessToken: session.tokens.accessToken,
    refreshToken: session.tokens.refreshToken,
    user: session.user,
  });
}


export function getAuthConfig(context: ApiRequestContext = {}) {
  return apiGet<AuthConfigPayload>("/api/auth/auth-config", undefined, {
    auth: false,
    context,
  });
}

export function exchangeOAuthAppToken(input: {
  code: string;
  appState: string;
  codeVerifier: string;
}) {
  return apiPost<OAuthAppTokenPayload>(
    "/api/auth/oauth2/app-token",
    {
      code: input.code,
      app_state: input.appState,
      code_verifier: input.codeVerifier,
    },
    { auth: false },
  );
}

export function exchangeOAuthMobileSession(ticket: string) {
  return apiPost<OAuthAppTokenPayload>(
    "/api/auth/oauth2/mobile-session",
    { ticket },
    { auth: false },
  );
}


export function getSwaggerDocsPath() {
  return process.env.NEXT_PUBLIC_SWAGGER_DOCS_PATH ?? DEFAULT_SWAGGER_DOCS_PATH;
}


export function getSwaggerDocsUrl() {
  return `/api/${getSwaggerDocsPath()}`;
}


export function getSwaggerJsonUrl() {
  return `/api/${getSwaggerDocsPath()}.json`;
}


export async function getBackendApiSpec() {
  const path = getSwaggerJsonUrl();
  if (typeof window === "undefined") {
    return apiGet<BackendApiSpec>(path, undefined, { auth: false });
  }

  const response = await fetch(path, {
    cache: "no-store",
    credentials: "include",
  });
  const payload = await parseResponse(response);
  if (!response.ok) {
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : `API request failed with status ${response.status}.`;
    throw new ApiError(message, { status: response.status, details: payload });
  }

  return extractEnvelope<BackendApiSpec>(payload, response.status);
}


export async function logout() {
  try {
    await apiPost("/api/auth/logout");
  } finally {
    clearSession();
  }
}


export function logoutAdmin() {
  clearAdminSession();
}


export function getCurrentUser(
  context: ApiRequestContext = {},
  options: { auth?: boolean } = {},
) {
  return apiGet<AuthUser>("/api/auth/me", undefined, {
    context,
    auth: options.auth ?? true,
  });
}


export function getCurrentAdmin(token?: string | null) {
  return apiAdminRequest<AdminUser>("/api/auth/admin/me", {
    clearSessionOnUnauthorized: true,
    method: "GET",
    token,
  });
}
