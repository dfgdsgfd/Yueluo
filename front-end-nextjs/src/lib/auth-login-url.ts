export function resolveUserCenterLoginUrl(startUrl?: string) {
  const rawStartUrl = startUrl?.trim() || "/api/auth/oauth2/login";

  if (/^[a-z][a-z\d+\-.]*:/i.test(rawStartUrl)) {
    return rawStartUrl;
  }

  const normalizedStartUrl = rawStartUrl.startsWith("/") ? rawStartUrl : `/${rawStartUrl}`;
  const directApiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.trim().replace(/\/$/, "");

  if (!directApiBaseUrl) {
    return normalizedStartUrl;
  }

  try {
    return new URL(normalizedStartUrl, directApiBaseUrl).toString();
  } catch {
    return `${directApiBaseUrl}${normalizedStartUrl}`;
  }
}
