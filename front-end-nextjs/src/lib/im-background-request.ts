import type { ApiRequestOptions } from "./api/core";

export const IM_BACKGROUND_REQUEST_TIMEOUT_MS = 2500;

export type ImRequestOptions = Omit<
  ApiRequestOptions,
  "body" | "context" | "method" | "query"
>;

export const IM_BACKGROUND_REQUEST_OPTIONS = {
  redirectOnUnauthorized: false,
  retryOnUnauthorized: false,
  timeoutMs: IM_BACKGROUND_REQUEST_TIMEOUT_MS,
} satisfies ImRequestOptions;
