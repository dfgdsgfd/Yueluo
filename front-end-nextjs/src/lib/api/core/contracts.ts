import type { ApiEnvelope } from "../../types/auth";

export const DEFAULT_SERVER_API_ORIGIN = "https://xse.yuelk.com";

export const DEFAULT_SWAGGER_DOCS_PATH =
  "swagger-MYQD6LuH0heYgcK5DT10Al00dj6OW8Wc";

export const ACCESS_TOKEN_KEY = "yuem_access_token";

export const REFRESH_TOKEN_KEY = "yuem_refresh_token";

export const USER_KEY = "yuem_user";

export const AUTH_USER_EVENT = "yuem:auth-user";

export const ACCESS_TOKEN_COOKIE = "yuem_access_token";

export const HTTP_ACCESS_TOKEN_COOKIE = "yuem_http_access_token";

export const ACCESS_TOKEN_COOKIE_FALLBACKS = ["access_token", "token"];

export const REFRESH_TOKEN_COOKIE = "yuem_refresh_token";

export const HTTP_REFRESH_TOKEN_COOKIE = "yuem_http_refresh_token";

export const REFRESH_TOKEN_COOKIE_FALLBACKS = ["refresh_token"];

export const ADMIN_ACCESS_TOKEN_KEY = "yuem_admin_access_token";

export const ADMIN_REFRESH_TOKEN_KEY = "yuem_admin_refresh_token";

export const ADMIN_USER_KEY = "yuem_admin_user";

export const AUTH_COOKIE_MAX_AGE_SECONDS = 7 * 24 * 60 * 60;

export const truthyEnvValues = new Set(["1", "true", "yes", "on"]);

export const DEFAULT_API_TIMEOUT_MS = 15_000;

export const DEFAULT_ADMIN_API_TIMEOUT_MS = 20_000;

export const DEFAULT_DOWNLOAD_TIMEOUT_MS = 60_000;

export const DEFAULT_UPLOAD_TIMEOUT_MS = 10 * 60 * 1000;

export type QueryValue = string | number | boolean | null | undefined;

export type ApiEnvelopeWithExtras<T> = ApiEnvelope<T> & Record<string, unknown>;

export type ApiRequestContext = {
  cookie?: string;
  forwardedHeaders?: Record<string, string | undefined>;
  signal?: AbortSignal;
  token?: string;
};

export type ApiRequestOptions = RequestInit & {
  auth?: boolean;
  context?: ApiRequestContext;
  query?: Record<string, QueryValue>;
  redirectOnUnauthorized?: boolean;
  retryOnUnauthorized?: boolean;
  timeoutMs?: number;
};

export type UploadProgress = {
  chunkNumber?: number;
  chunkPercent?: number;
  fileName?: string;
  loaded: number;
  message?: string;
  percent?: number;
  stage?:
    | "preparing"
    | "verifying"
    | "uploading"
    | "processing"
    | "chunking"
    | "merging"
    | "thumbnail"
    | "publishing"
    | "complete";
  total?: number;
  totalChunks?: number;
  uploadedChunks?: number;
};

export type UploadChunkConfigPayload = {
  chunkSize?: number;
  imageChunkThreshold?: number;
  imageMaxSize?: number;
  maxFileSize?: number;
};

export type UploadChunkVerifyPayload = {
  exists?: boolean;
  valid?: boolean;
};

export type UploadChunkPayload = {
  chunkNumber?: number;
  complete?: boolean;
  total?: number;
  uploaded?: number;
};

export type ApiUploadOptions = {
  auth?: boolean;
  context?: ApiRequestContext;
  onProgress?: (progress: UploadProgress) => void;
  purpose?: "content" | "avatar" | "background" | "cover" | "feedback" | "ai_analysis";
  query?: Record<string, QueryValue>;
  signal?: AbortSignal;
  timeoutMs?: number;
};

export class ApiError extends Error {
  code?: number;
  status?: number;
  details?: unknown;

  constructor(
    message: string,
    options: { code?: number; status?: number; details?: unknown } = {},
  ) {
    super(message);
    this.name = "ApiError";
    this.code = options.code;
    this.status = options.status;
    this.details = options.details;
  }
}

export class ApiUnauthorizedError extends ApiError {
  constructor(message = "Authentication is required.") {
    super(message, { code: 401, status: 401 });
    this.name = "ApiUnauthorizedError";
  }
}
