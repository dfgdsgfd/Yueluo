function checkOAuthFrontendStaticContract(checks) {
  const problems = [];
  const details = {
    frontendLoginFormPath: path.relative(repoRoot, frontendLoginFormPath),
    frontendLoginPagePath: path.relative(repoRoot, frontendLoginPagePath),
    frontendRootLayoutPath: path.relative(repoRoot, frontendRootLayoutPath),
    frontendOAuthCallbackBootstrapPath: path.relative(repoRoot, frontendOAuthCallbackBootstrapPath),
    frontendOAuthCallbackHandlerPath: path.relative(repoRoot, frontendOAuthCallbackHandlerPath),
  };
  const loginForm = fileText(frontendLoginFormPath);
  const loginPage = fileText(frontendLoginPagePath);
  const rootLayout = fileText(frontendRootLayoutPath);
  const callbackBootstrap = fileText(frontendOAuthCallbackBootstrapPath);
  const callbackHandler = fileText(frontendOAuthCallbackHandlerPath);

  const requiredPatterns = [
    {
      fileText: loginPage,
      pattern: "<LoginForm />",
      problem: "/login page does not render LoginForm",
    },
    {
      fileText: loginForm,
      pattern: "getAuthConfig()",
      problem: "login form does not load auth-config",
    },
    {
      fileText: loginForm,
      pattern: "resolveUserCenterLoginUrl(authConfig.oauth2StartUrl)",
      problem: "login form does not prefer auth-config oauth2StartUrl",
    },
    {
      fileText: loginForm,
      pattern: 'startUrl?.trim() || "/api/auth/oauth2/login"',
      problem: "login form does not fall back to the backend OAuth2 start endpoint",
    },
    {
      fileText: loginForm,
      pattern: "NEXT_PUBLIC_API_BASE_URL",
      problem: "login form does not support direct backend OAuth2 start URLs",
    },
    {
      fileText: loginForm,
      pattern: "用户中心一键登录",
      problem: "login form does not expose the user-center one-click login label",
    },
    {
      fileText: loginForm,
      pattern: "oauth2OnlyLogin && userCenterLoginUrl",
      problem: "login form does not derive an OAuth2-only mode from auth-config",
    },
    {
      fileText: loginForm,
      pattern: "当前环境仅支持用户中心登录。",
      problem: "login form does not show the OAuth2-only notice",
    },
    {
      fileText: callbackBootstrap,
      pattern: 'id="oauth-callback-bootstrap"',
      problem: "OAuth callback bootstrap script id is missing",
    },
    {
      fileText: callbackBootstrap,
      pattern: 'window.localStorage.setItem("yuem_access_token", accessToken)',
      problem: "OAuth callback bootstrap does not save access token early",
    },
    {
      fileText: callbackBootstrap,
      pattern: "window.history.replaceState",
      problem: "OAuth callback bootstrap does not clean sensitive URL params early",
    },
    {
      fileText: callbackHandler,
      pattern: "getCurrentUser({ token: accessToken }, { auth: false })",
      problem: "OAuth callback handler does not hydrate current user with the returned token",
    },
    {
      fileText: callbackHandler,
      pattern: "const handledRef = useRef(false);",
      problem: "OAuth callback handler is not guarded against React dev StrictMode duplicate effects",
    },
    {
      fileText: callbackHandler,
      pattern: "handledRef.current = true",
      problem: "OAuth callback handler does not mark the callback as handled before async hydration",
    },
    {
      fileText: rootLayout,
      pattern: "<OAuthCallbackBootstrap />",
      problem: "root layout does not mount OAuthCallbackBootstrap",
    },
    {
      fileText: rootLayout,
      pattern: "<OAuthCallbackHandler />",
      problem: "root layout does not mount OAuthCallbackHandler",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }
  if (/\bcancelled\b/.test(callbackHandler)) {
    problems.push("OAuth callback handler still contains a cancellable async hydration path");
  }

  if (/resolveUserCenterLoginUrl\(authConfig\.oauth2LoginUrl\)/.test(loginForm)) {
    problems.push("login form points the button at oauth2LoginUrl instead of the backend start endpoint");
  }
  if (!/<head>\s*<OAuthCallbackBootstrap \/>/.test(rootLayout)) {
    problems.push("OAuth callback bootstrap is not mounted directly inside the root layout head");
  }

  addCheck(
    checks,
    "frontend-oauth2-ui-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "OAuth2 login UI and callback static contract is aligned"
      : "OAuth2 login UI and callback static contract is not aligned",
    {
      ...details,
      problems,
    },
  );
}

function checkOAuthStartUrlStaticContract(checks) {
  const loginForm = fileText(frontendLoginFormPath);
  const typesFile = fileText(frontendTypesPath);
  const problems = [];
  const details = {
    frontendLoginFormPath: path.relative(repoRoot, frontendLoginFormPath),
    frontendTypesPath: path.relative(repoRoot, frontendTypesPath),
    expectedFallbackStartPath: "/api/auth/oauth2/login",
    expectedDirectBackendExample: "http://localhost:3001/api/auth/oauth2/login",
  };

  const requiredPatterns = [
    {
      text: typesFile,
      pattern: "oauth2StartUrl?: string;",
      problem: "auth-config type does not expose oauth2StartUrl for the backend start endpoint",
    },
    {
      text: loginForm,
      pattern: "resolveUserCenterLoginUrl(authConfig.oauth2StartUrl)",
      problem: "login form does not resolve the button from oauth2StartUrl",
    },
    {
      text: loginForm,
      pattern: 'startUrl?.trim() || "/api/auth/oauth2/login"',
      problem: "login form does not default to the backend OAuth2 start endpoint",
    },
    {
      text: loginForm,
      pattern: "/^[a-z][a-z\\d+\\-.]*:/i.test(rawStartUrl)",
      problem: "login form does not preserve absolute OAuth2 start URLs",
    },
    {
      text: loginForm,
      pattern: 'rawStartUrl.startsWith("/") ? rawStartUrl : `/${rawStartUrl}`',
      problem: "login form does not normalize relative OAuth2 start paths",
    },
    {
      text: loginForm,
      pattern: "process.env.NEXT_PUBLIC_API_BASE_URL?.trim().replace(/\\/$/, \"\")",
      problem: "login form does not normalize NEXT_PUBLIC_API_BASE_URL for direct backend redirects",
    },
    {
      text: loginForm,
      pattern: "new URL(normalizedStartUrl, directApiBaseUrl).toString()",
      problem: "login form does not compose direct backend OAuth2 start URLs with URL parsing",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.text.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }
  if (/authConfig\.oauth2LoginUrl/.test(loginForm)) {
    problems.push("login form uses oauth2LoginUrl; the button must target the backend start endpoint instead");
  }

  addCheck(
    checks,
    "frontend-oauth2-start-url-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "OAuth2 one-click login button resolves to the backend start endpoint"
      : "OAuth2 one-click login button URL contract is not aligned",
    {
      ...details,
      problems,
    },
  );
}

function checkFrontendAuthSessionStaticContract(checks) {
  const frontendApi = fileText(frontendApiPath);
  const problems = [];
  const details = {
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
  };

  const sliceBetween = (startPattern, endPattern) => {
    const startIndex = frontendApi.indexOf(startPattern);
    if (startIndex < 0) {
      return "";
    }
    const endIndex = endPattern
      ? frontendApi.indexOf(endPattern, startIndex + startPattern.length)
      : -1;
    return frontendApi.slice(startIndex, endIndex > startIndex ? endIndex : undefined);
  };

  const storeSessionSource = sliceBetween("export function storeSession", "export function storeOAuthCallbackTokens");
  const storeOAuthSource = sliceBetween("export function storeOAuthCallbackTokens", "export function storeAdminSession");
  const storeAdminSource = sliceBetween("export function storeAdminSession", "export function clearSession");
  const clearSessionSource = sliceBetween("export function clearSession", "export function clearAdminSession");
  const clearAdminSessionSource = sliceBetween("export function clearAdminSession", "function normalizeTokens");
  const refreshSource = sliceBetween("async function refreshAccessToken", "function redirectToLogin");
  const apiRequestSource = sliceBetween("export async function apiRequest", "export async function apiFetch");
  const apiRequestEnvelopeSource = sliceBetween("async function apiRequestEnvelope", "export function apiGet");
  const apiUploadSource = sliceBetween("function apiUploadWithProgress", "export function apiUpload");
  const apiAdminRequestSource = sliceBetween("async function apiAdminRequest", "async function apiAdminEnvelope");
  const apiAdminEnvelopeSource = sliceBetween("async function apiAdminEnvelope", "function isRecord");
  const loginSource = sliceBetween("export async function loginWithPassword", "export async function loginAdmin");
  const loginAdminSource = sliceBetween("export async function loginAdmin", "export async function registerWithPassword");
  const registerSource = sliceBetween("export async function registerWithPassword", "export function getAuthConfig");
  const logoutSource = sliceBetween("export async function logout", "export function logoutAdmin");
  const getFeedPageSource = sliceBetween("export async function getFeedPage", "function normalizeSearchFeedPayload");
  const searchFeedSource = sliceBetween("export async function searchFeed", "export function hasNextFeedPage");
  const postReadSource = sliceBetween("export function getPostDetail", "export function createComment");
  const getHotCategoriesSource = sliceBetween("export async function getHotCategories", "export async function getInitialFeedData");

  const requiredPatterns = [
    {
      text: frontendApi,
      pattern: 'const ACCESS_TOKEN_KEY = "yuem_access_token";',
      problem: "ordinary user access-token storage key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: 'const REFRESH_TOKEN_KEY = "yuem_refresh_token";',
      problem: "ordinary user refresh-token storage key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: 'const USER_KEY = "yuem_user";',
      problem: "ordinary user snapshot storage key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: 'const ACCESS_TOKEN_COOKIE = "yuem_access_token";',
      problem: "ordinary user SSR auth cookie key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: "function getRequestAccessToken",
      problem: "ordinary user API requests do not centralize token lookup across context, cookies and storage",
    },
    {
      text: frontendApi,
      pattern: "getCookieValue(context?.cookie, ACCESS_TOKEN_COOKIE)",
      problem: "ordinary user API requests do not read the SSR auth cookie from request context",
    },
    {
      text: frontendApi,
      pattern: "function getRequestAuthorizationToken",
      problem: "ordinary user API requests do not centralize explicit context-token authorization",
    },
    {
      text: frontendApi,
      pattern: "return context?.token ?? (auth ? getRequestAccessToken(context) : null)",
      problem: "auth:false requests with an explicit context token do not send Authorization",
    },
    {
      text: frontendApi,
      pattern: "function persistAccessTokenCookie",
      problem: "ordinary user sessions do not persist an auth cookie for SSR page loads",
    },
    {
      text: frontendApi,
      pattern: "function clearAccessTokenCookie",
      problem: "ordinary user logout/401 cleanup does not clear the SSR auth cookie",
    },
    {
      text: frontendApi,
      pattern: 'const ADMIN_ACCESS_TOKEN_KEY = "yuem_admin_access_token";',
      problem: "administrator access-token storage key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: 'const ADMIN_REFRESH_TOKEN_KEY = "yuem_admin_refresh_token";',
      problem: "administrator refresh-token storage key is missing or renamed",
    },
    {
      text: frontendApi,
      pattern: 'const ADMIN_USER_KEY = "yuem_admin_user";',
      problem: "administrator snapshot storage key is missing or renamed",
    },
    {
      text: storeSessionSource,
      pattern: "window.localStorage.setItem(ACCESS_TOKEN_KEY, session.tokens.accessToken)",
      problem: "storeSession does not persist the ordinary user access token",
    },
    {
      text: storeSessionSource,
      pattern: "window.localStorage.setItem(REFRESH_TOKEN_KEY, session.tokens.refreshToken)",
      problem: "storeSession does not persist the ordinary user refresh token",
    },
    {
      text: storeSessionSource,
      pattern: "window.localStorage.setItem(USER_KEY, JSON.stringify(session.user))",
      problem: "storeSession does not persist the ordinary user snapshot",
    },
    {
      text: storeSessionSource,
      pattern: "persistAccessTokenCookie(session.tokens.accessToken)",
      problem: "storeSession does not persist the ordinary user auth cookie",
    },
    {
      text: storeOAuthSource,
      pattern: "window.localStorage.setItem(ACCESS_TOKEN_KEY, input.accessToken)",
      problem: "OAuth callback storage does not persist the ordinary user access token",
    },
    {
      text: storeOAuthSource,
      pattern: "window.localStorage.setItem(REFRESH_TOKEN_KEY, input.refreshToken)",
      problem: "OAuth callback storage does not persist the ordinary user refresh token",
    },
    {
      text: storeOAuthSource,
      pattern: "persistAccessTokenCookie(input.accessToken)",
      problem: "OAuth callback storage does not persist the ordinary user auth cookie",
    },
    {
      text: storeOAuthSource,
      pattern: "window.localStorage.removeItem(USER_KEY)",
      problem: "OAuth callback storage does not clear a stale user snapshot before hydration",
    },
    {
      text: storeAdminSource,
      pattern: "window.localStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, session.tokens.accessToken)",
      problem: "storeAdminSession does not persist the administrator access token",
    },
    {
      text: storeAdminSource,
      pattern: "window.localStorage.setItem(ADMIN_REFRESH_TOKEN_KEY, session.tokens.refreshToken)",
      problem: "storeAdminSession does not persist the administrator refresh token",
    },
    {
      text: storeAdminSource,
      pattern: "window.localStorage.setItem(ADMIN_USER_KEY, JSON.stringify(session.admin))",
      problem: "storeAdminSession does not persist the administrator snapshot",
    },
    {
      text: clearSessionSource,
      pattern: "window.localStorage.removeItem(ACCESS_TOKEN_KEY)",
      problem: "clearSession does not remove the ordinary user access token",
    },
    {
      text: clearSessionSource,
      pattern: "window.localStorage.removeItem(REFRESH_TOKEN_KEY)",
      problem: "clearSession does not remove the ordinary user refresh token",
    },
    {
      text: clearSessionSource,
      pattern: "window.localStorage.removeItem(USER_KEY)",
      problem: "clearSession does not remove the ordinary user snapshot",
    },
    {
      text: clearSessionSource,
      pattern: "clearAccessTokenCookie()",
      problem: "clearSession does not remove the ordinary user SSR auth cookie",
    },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(ADMIN_ACCESS_TOKEN_KEY)",
      problem: "clearAdminSession does not remove the administrator access token",
    },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(ADMIN_REFRESH_TOKEN_KEY)",
      problem: "clearAdminSession does not remove the administrator refresh token",
    },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(ADMIN_USER_KEY)",
      problem: "clearAdminSession does not remove the administrator snapshot",
    },
    {
      text: refreshSource,
      pattern: "const refreshToken = window.localStorage.getItem(REFRESH_TOKEN_KEY)",
      problem: "refreshAccessToken does not read the ordinary user refresh token",
    },
    {
      text: refreshSource,
      pattern: 'fetch(buildApiUrl("/api/auth/refresh")',
      problem: "refreshAccessToken does not call the backend refresh endpoint",
    },
    {
      text: refreshSource,
      pattern: "body: JSON.stringify({ refresh_token: refreshToken })",
      problem: "refreshAccessToken does not send refresh_token in the backend payload",
    },
    {
      text: refreshSource,
      pattern: "window.localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken)",
      problem: "refreshAccessToken does not persist the refreshed access token",
    },
    {
      text: refreshSource,
      pattern: "window.localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken)",
      problem: "refreshAccessToken does not persist the refreshed refresh token",
    },
    {
      text: refreshSource,
      pattern: "persistAccessTokenCookie(tokens.accessToken)",
      problem: "refreshAccessToken does not persist the refreshed ordinary user auth cookie",
    },
    {
      text: refreshSource,
      pattern: "return tokens.accessToken",
      problem: "refreshAccessToken does not return the refreshed access token for retry",
    },
    {
      text: apiRequestSource,
      pattern: "const token = getRequestAuthorizationToken(context, auth)",
      problem: "apiRequest does not read ordinary user tokens from context/cookie/storage",
    },
    {
      text: apiRequestSource,
      pattern: "const nextToken = await refreshAccessToken()",
      problem: "apiRequest does not attempt refresh on user 401 responses",
    },
    {
      text: apiRequestSource,
      pattern: "retryOnUnauthorized: false",
      problem: "apiRequest does not stop retry loops after one refreshed retry",
    },
    {
      text: apiRequestSource,
      pattern: "context: { ...context, token: nextToken }",
      problem: "apiRequest does not retry with the refreshed access token",
    },
    {
      text: apiRequestSource,
      pattern: "clearSession()",
      problem: "apiRequest does not clear ordinary user session on unrecoverable 401",
    },
    {
      text: apiRequestSource,
      pattern: "redirectToLogin()",
      problem: "apiRequest does not redirect ordinary user flows to login on unrecoverable 401",
    },
    {
      text: apiRequestEnvelopeSource,
      pattern: "const token = getRequestAuthorizationToken(context, auth)",
      problem: "apiRequestEnvelope does not read ordinary user tokens from context/cookie/storage",
    },
    {
      text: apiRequestEnvelopeSource,
      pattern: "const nextToken = await refreshAccessToken()",
      problem: "apiRequestEnvelope does not attempt refresh on user 401 responses",
    },
    {
      text: apiRequestEnvelopeSource,
      pattern: "retryOnUnauthorized: false",
      problem: "apiRequestEnvelope does not stop retry loops after one refreshed retry",
    },
    {
      text: apiUploadSource,
      pattern: "const nextToken = await refreshAccessToken()",
      problem: "upload-with-progress requests do not attempt refresh on user 401 responses",
    },
    {
      text: frontendApi,
      pattern: "const token = getRequestAuthorizationToken(context, auth)",
      problem: "download requests do not read ordinary user tokens from context/cookie/storage",
    },
    {
      text: apiUploadSource,
      pattern: "clearSession()",
      problem: "upload-with-progress requests do not clear ordinary user session on unrecoverable 401",
    },
    {
      text: loginSource,
      pattern: '"/api/auth/login"',
      problem: "loginWithPassword does not call the backend user login endpoint",
    },
    {
      text: loginSource,
      pattern: "storeSession(session)",
      problem: "loginWithPassword does not store the returned user session",
    },
    {
      text: registerSource,
      pattern: '"/api/auth/register"',
      problem: "registerWithPassword does not call the backend user register endpoint",
    },
    {
      text: registerSource,
      pattern: "storeSession(session)",
      problem: "registerWithPassword does not store the returned user session",
    },
    {
      text: loginAdminSource,
      pattern: '"/api/auth/admin/login"',
      problem: "loginAdmin does not call the backend admin login endpoint",
    },
    {
      text: loginAdminSource,
      pattern: "storeAdminSession(session)",
      problem: "loginAdmin does not store the returned admin session",
    },
    {
      text: logoutSource,
      pattern: 'await apiPost("/api/auth/logout")',
      problem: "logout does not call the backend user logout endpoint",
    },
    {
      text: logoutSource,
      pattern: "} finally {",
      problem: "logout does not clear local session in a finally block",
    },
    {
      text: logoutSource,
      pattern: "clearSession()",
      problem: "logout does not clear ordinary user session",
    },
    {
      text: getFeedPageSource,
      pattern: "auth: true",
      problem: "feed page requests do not send the stored ordinary-user token",
    },
    {
      text: searchFeedSource,
      pattern: "auth: true",
      problem: "search feed requests do not send the stored ordinary-user token",
    },
    {
      text: postReadSource,
      pattern: "auth: true",
      problem: "post detail/comment requests do not send the stored ordinary-user token",
    },
    {
      text: getHotCategoriesSource,
      pattern: "auth: true",
      problem: "hot category requests do not send the stored ordinary-user token",
    },
    {
      text: apiAdminRequestSource,
      pattern: "clearAdminSession()",
      problem: "apiAdminRequest does not clear administrator session on 401",
    },
    {
      text: apiAdminEnvelopeSource,
      pattern: "clearAdminSession()",
      problem: "apiAdminEnvelope does not clear administrator session on 401",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.text.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  const adminLeakPatterns = [
    { text: clearSessionSource, pattern: "ADMIN_", problem: "clearSession should not clear administrator storage keys" },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(ACCESS_TOKEN_KEY)",
      problem: "clearAdminSession should not clear ordinary user access token",
    },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(REFRESH_TOKEN_KEY)",
      problem: "clearAdminSession should not clear ordinary user refresh token",
    },
    {
      text: clearAdminSessionSource,
      pattern: "window.localStorage.removeItem(USER_KEY)",
      problem: "clearAdminSession should not clear ordinary user snapshot",
    },
    { text: apiRequestSource, pattern: "getStoredAdminAccessToken", problem: "ordinary user API requests should not read administrator tokens" },
    { text: apiRequestEnvelopeSource, pattern: "getStoredAdminAccessToken", problem: "ordinary user envelope requests should not read administrator tokens" },
    { text: apiUploadSource, pattern: "getStoredAdminAccessToken", problem: "ordinary user upload requests should not read administrator tokens" },
    { text: apiAdminRequestSource, pattern: "getStoredAccessToken", problem: "administrator API requests should not read ordinary user tokens" },
    { text: apiAdminEnvelopeSource, pattern: "getStoredAccessToken", problem: "administrator envelope requests should not read ordinary user tokens" },
  ];

  for (const check of adminLeakPatterns) {
    if (check.text.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  if (!/apiUploadWithProgress<T>\([\s\S]*false,\s*\)/.test(apiUploadSource)) {
    problems.push("upload-with-progress requests do not stop retry loops after one refreshed retry");
  }

  addCheck(
    checks,
    "frontend-auth-session-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend auth session storage, refresh and logout contract is aligned"
      : "frontend auth session storage, refresh and logout contract is not aligned",
    {
      ...details,
      problems,
    },
  );
}

