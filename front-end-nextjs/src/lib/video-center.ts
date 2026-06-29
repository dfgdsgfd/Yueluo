import type {
  AuthConfigPayload,
  VideoCenterVisibilityConfig
} from "./types";

export const DEFAULT_VIDEO_CENTER_ACCOUNT_CUTOFF = "2026-06-12T00:00:00Z";

export function videoCenterConfigFromAuthConfig(
  payload?: AuthConfigPayload | null,
): VideoCenterVisibilityConfig {
  return normalizeVideoCenterConfig({
    enabled: payload?.videoCenterEnabled ?? true,
    accountCutoff: payload?.videoCenterAccountCutoff ?? DEFAULT_VIDEO_CENTER_ACCOUNT_CUTOFF,
    guestRestricted: payload?.videoCenterGuestRestricted ?? false,
    requestCountryCode: payload?.videoCenterRequestCountryCode ?? null,
  });
}

export function normalizeVideoCenterConfig(
  config?: Partial<VideoCenterVisibilityConfig> | null,
): VideoCenterVisibilityConfig {
  return {
    enabled: config?.enabled ?? true,
    accountCutoff: config?.accountCutoff ?? DEFAULT_VIDEO_CENTER_ACCOUNT_CUTOFF,
    guestRestricted: config?.guestRestricted ?? false,
    requestCountryCode: config?.requestCountryCode ?? null,
  };
}

export function shouldShowVideoCenterForUser(
  createdAt?: string | null,
  config?: Partial<VideoCenterVisibilityConfig> | null,
) {
  const normalized = normalizeVideoCenterConfig(config);
  if (!normalized.enabled) {
    return false;
  }

  if (normalized.guestRestricted && !createdAt) {
    return false;
  }

  const cutoff = normalized.accountCutoff?.trim();
  if (!cutoff) {
    return true;
  }

  const countryCode = normalized.requestCountryCode?.trim().toUpperCase();
  if (countryCode && countryCode !== "CN") {
    return true;
  }

  if (!createdAt) {
    return true;
  }

  const createdTime = Date.parse(createdAt);
  const cutoffTime = Date.parse(cutoff);
  if (Number.isNaN(createdTime) || Number.isNaN(cutoffTime)) {
    return true;
  }

  return createdTime < cutoffTime;
}
