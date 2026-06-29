import Script from "next/script";

const oauthCallbackParams = [
  "oauth2_login",
  "access_token",
  "refresh_token",
  "is_new_user",
  "error",
  "message",
];

export function OAuthCallbackBootstrap() {
  return (
    <Script
      id="oauth-callback-bootstrap"
      dangerouslySetInnerHTML={{
        __html: `
(function () {
  var callbackParams = ${JSON.stringify(oauthCallbackParams)};
  var params = new URLSearchParams(window.location.search);
  if (params.get("oauth2_login") !== "success") {
    return;
  }

  var accessToken = params.get("access_token");
  var refreshToken = params.get("refresh_token");
  window.__YUEM_OAUTH_CALLBACK__ = {
    accessToken: accessToken,
    refreshToken: refreshToken
  };

  if (accessToken && refreshToken) {
    try {
      window.localStorage.setItem("yuem_access_token", accessToken);
      window.localStorage.setItem("yuem_refresh_token", refreshToken);
      window.localStorage.removeItem("yuem_user");
      document.cookie = "yuem_access_token=" + encodeURIComponent(accessToken) + "; path=/; max-age=604800; samesite=lax" + (window.location.protocol === "https:" ? "; secure" : "");
    } catch (error) {}
  }

  for (var i = 0; i < callbackParams.length; i += 1) {
    params.delete(callbackParams[i]);
  }

  var query = params.toString();
  window.history.replaceState(null, "", window.location.pathname + (query ? "?" + query : "") + window.location.hash);
}());
        `.trim(),
      }}
    />
  );
}
