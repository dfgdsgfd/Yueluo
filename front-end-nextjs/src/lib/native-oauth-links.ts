export const nativeAndroidPackageName = "com.yuelk.xsewebfast";
export const nativeAppOrigin = normalizeOrigin(
  process.env.NEXT_PUBLIC_YUEM_MOBILE_SERVER_URL ??
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    "https://xse.yuelk.com",
);
export const nativeOAuthCallbackPath = "/app/oauth/callback";

const nativeOAuthScheme = "xsewebfast";
const nativeOAuthSchemeHost = "oauth";
const nativeOAuthSchemePath = "/callback";

export type NativeOAuthLinkInput = {
  appState?: string | null;
  code?: string | null;
  error?: string | null;
};

export function buildNativeOAuthDeepLinks(input: NativeOAuthLinkInput) {
  const query = new URLSearchParams();
  if (input.code) query.set("code", input.code);
  if (input.appState) query.set("app_state", input.appState);
  if (input.error) query.set("error", input.error);

  const queryString = query.toString();
  const suffix = queryString ? `?${queryString}` : "";
  const schemeUrl = `${nativeOAuthScheme}://${nativeOAuthSchemeHost}${nativeOAuthSchemePath}${suffix}`;
  const fallbackUrl = `${nativeAppOrigin}${nativeOAuthCallbackPath}${suffix}`;
  const intentUrl = `intent://${nativeOAuthSchemeHost}${nativeOAuthSchemePath}${suffix}#Intent;scheme=${nativeOAuthScheme};package=${nativeAndroidPackageName};S.browser_fallback_url=${encodeURIComponent(fallbackUrl)};end`;

  return { schemeUrl, intentUrl };
}

export function selectNativeOAuthAutoOpenLink(input: {
  deepLink: string;
  intentLink?: string;
}) {
  return input.intentLink ?? input.deepLink;
}

function normalizeOrigin(raw: string) {
  try {
    return new URL(raw).origin;
  } catch {
    return "https://xse.yuelk.com";
  }
}
