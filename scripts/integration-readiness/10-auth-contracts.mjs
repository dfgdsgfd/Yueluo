function resolveOAuthStartUrl(origin, startUrl) {
  const rawStartUrl =
    typeof startUrl === "string" && startUrl.trim()
      ? startUrl.trim()
      : "/api/auth/oauth2/login";

  try {
    if (/^[a-z][a-z\d+\-.]*:/i.test(rawStartUrl)) {
      return new URL(rawStartUrl).toString();
    }

    const normalizedStartUrl = rawStartUrl.startsWith("/") ? rawStartUrl : `/${rawStartUrl}`;
    return new URL(normalizedStartUrl, origin).toString();
  } catch {
    return null;
  }
}

async function checkAuthConfigContract(checks, backendOrigin) {
  const url = authConfigUrlFromOrigin(backendOrigin);
  if (!url) {
    addCheck(checks, "frontend-auth-config-contract", "fail", "auth-config URL cannot be built", {
      origin: backendOrigin,
    });
    return null;
  }

  try {
    const result = await fetchJsonWithTimeout(url);
    if (!result.ok) {
      addCheck(
        checks,
        "frontend-auth-config-contract",
        "fail",
        "frontend-configured backend auth-config endpoint is not usable",
        {
          url,
          status: result.status,
          parseError: result.parseError,
          textSnippet: result.textSnippet,
        },
      );
      return null;
    }

    const authConfig = unwrapAuthConfigPayload(result.payload);
    const requiredBooleanFields = [
      "emailEnabled",
      "oauth2Enabled",
      "oauth2OnlyLogin",
      "geetestEnabled",
    ];
    const invalidBooleanFields = requiredBooleanFields.filter(
      (field) => typeof authConfig?.[field] !== "boolean",
    );
    const problems = [];

    if (!authConfig) {
      problems.push("auth-config payload is not an object or envelope code is not successful");
    }
    if (invalidBooleanFields.length > 0) {
      problems.push("required boolean fields are missing or not boolean");
    }
    if (authConfig?.oauth2OnlyLogin === true && authConfig.oauth2Enabled !== true) {
      problems.push("oauth2OnlyLogin is true while oauth2Enabled is not true");
    }

    const oauthStartUrl =
      authConfig?.oauth2Enabled === true
        ? resolveOAuthStartUrl(backendOrigin, authConfig.oauth2StartUrl)
        : null;
    if (authConfig?.oauth2Enabled === true && !oauthStartUrl) {
      problems.push("OAuth2 is enabled but the start URL cannot be resolved");
    }
    if (
      authConfig?.geetestEnabled === true &&
      (typeof authConfig.geetestCaptchaId !== "string" || !authConfig.geetestCaptchaId.trim())
    ) {
      problems.push("Geetest is enabled but geetestCaptchaId is missing");
    }

    addCheck(
      checks,
      "frontend-auth-config-contract",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "auth-config contract supports the frontend login options"
        : "auth-config contract does not match the frontend login requirements",
      {
        url,
        problems,
        invalidBooleanFields,
        oauth2Enabled: authConfig?.oauth2Enabled ?? null,
        oauth2OnlyLogin: authConfig?.oauth2OnlyLogin ?? null,
        oauthStartUrl,
        oauthStartUrlSource: authConfig?.oauth2StartUrl ? "auth-config" : "frontend-fallback",
        geetestEnabled: authConfig?.geetestEnabled ?? null,
        hasGeetestCaptchaId:
          typeof authConfig?.geetestCaptchaId === "string" &&
          Boolean(authConfig.geetestCaptchaId.trim()),
      },
    );

    return {
      authConfig,
      oauthStartUrl,
      ok: problems.length === 0,
    };
  } catch (error) {
    addCheck(checks, "frontend-auth-config-contract", "fail", "auth-config endpoint is not reachable", {
      url,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
    return null;
  }
}

async function checkAuthPublicAuxContract(checks, backendOrigin, authConfigResult) {
  const emailConfigUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/email-config");
  const captchaUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/captcha");

  if (!emailConfigUrl || !captchaUrl) {
    addCheck(
      checks,
      "frontend-auth-public-aux-contract",
      "fail",
      "auth public auxiliary URLs cannot be built",
      {
        origin: backendOrigin,
        emailConfigUrl,
        captchaUrl,
      },
    );
    return;
  }

  try {
    const [emailConfigResult, captchaResult] = await Promise.all([
      fetchJsonWithTimeout(emailConfigUrl),
      fetchJsonWithTimeout(captchaUrl),
    ]);
    const emailConfig = unwrapApiPayload(emailConfigResult.payload);
    const captcha = unwrapApiPayload(captchaResult.payload);
    const problems = [];

    if (!emailConfigResult.ok) {
      problems.push(`email-config returned status ${emailConfigResult.status}`);
    }
    if (!captchaResult.ok) {
      problems.push(`captcha returned status ${captchaResult.status}`);
    }
    if (!isRecord(emailConfig) || typeof emailConfig.emailEnabled !== "boolean") {
      problems.push("email-config payload does not include boolean emailEnabled");
    }
    if (
      authConfigResult?.authConfig &&
      isRecord(emailConfig) &&
      typeof emailConfig.emailEnabled === "boolean" &&
      emailConfig.emailEnabled !== authConfigResult.authConfig.emailEnabled
    ) {
      problems.push("email-config emailEnabled does not match auth-config emailEnabled");
    }
    if (!isRecord(captcha)) {
      problems.push("captcha payload is not an object");
    } else {
      if (typeof captcha.captchaId !== "string" || !captcha.captchaId.trim()) {
        problems.push("captcha payload does not include a non-empty captchaId");
      }
      if (
        typeof captcha.captchaSvg !== "string" ||
        !captcha.captchaSvg.includes("<svg") ||
        !captcha.captchaSvg.includes("</svg>")
      ) {
        problems.push("captcha payload does not include SVG markup");
      }
      if (typeof captcha.captchaSvg === "string" && /<script[\s>]/i.test(captcha.captchaSvg)) {
        problems.push("captcha SVG contains a script tag");
      }
    }

    addCheck(
      checks,
      "frontend-auth-public-aux-contract",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "auth public email-config and captcha endpoints match the frontend contract"
        : "auth public auxiliary endpoints do not match the frontend contract",
      {
        emailConfigUrl,
        captchaUrl,
        problems,
        emailConfig: {
          status: emailConfigResult.status,
          code: isRecord(emailConfigResult.payload) ? emailConfigResult.payload.code ?? null : null,
          emailEnabled: isRecord(emailConfig) ? emailConfig.emailEnabled ?? null : null,
        },
        captcha: {
          status: captchaResult.status,
          code: isRecord(captchaResult.payload) ? captchaResult.payload.code ?? null : null,
          hasCaptchaId: isRecord(captcha) && typeof captcha.captchaId === "string" && Boolean(captcha.captchaId.trim()),
          hasCaptchaSvg:
            isRecord(captcha) &&
            typeof captcha.captchaSvg === "string" &&
            captcha.captchaSvg.includes("<svg") &&
            captcha.captchaSvg.includes("</svg>"),
          svgLength: isRecord(captcha) && typeof captcha.captchaSvg === "string" ? captcha.captchaSvg.length : null,
        },
      },
    );
  } catch (error) {
    addCheck(checks, "frontend-auth-public-aux-contract", "fail", "auth public auxiliary endpoints are not reachable", {
      emailConfigUrl,
      captchaUrl,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
      retries: httpRetryCount,
    });
  }
}

async function checkOAuthStartRedirect(checks, authConfigResult) {
  const authConfig = authConfigResult?.authConfig;
  if (!authConfig) {
    addCheck(
      checks,
      "frontend-oauth2-start-redirect",
      "fail",
      "OAuth2 start redirect cannot be checked without a valid auth-config",
    );
    return;
  }

  if (authConfig.oauth2Enabled !== true) {
    addCheck(
      checks,
      "frontend-oauth2-start-redirect",
      "pass",
      "OAuth2 is disabled; start redirect is not required",
    );
    return { disabled: true, ok: true };
  }

  if (!authConfigResult.oauthStartUrl) {
    addCheck(checks, "frontend-oauth2-start-redirect", "fail", "OAuth2 start URL cannot be resolved");
    return null;
  }

  try {
    const response = await fetchManualRedirectWithTimeout(authConfigResult.oauthStartUrl);
    const redirectStatusCodes = new Set([301, 302, 303, 307, 308]);
    const location = response.headers.get("location") || "";
    let redirectUrl = null;
    let redirectUriUrl = null;
    const problems = [];

    if (!redirectStatusCodes.has(response.status)) {
      problems.push("OAuth2 start endpoint did not return an HTTP redirect");
    }
    if (!location) {
      problems.push("OAuth2 start endpoint did not return a Location header");
    } else {
      try {
        redirectUrl = new URL(location, authConfigResult.oauthStartUrl);
      } catch {
        problems.push("OAuth2 start Location header is not a valid URL");
      }
    }

    const searchParams = redirectUrl?.searchParams;
    const redirectUri = searchParams?.get("redirect_uri") ?? "";
    if (redirectUri) {
      try {
        redirectUriUrl = new URL(redirectUri);
      } catch {
        problems.push("OAuth2 start redirect_uri is not a valid URL");
      }
    }
    const requiredParams = [
      "client_id",
      "state",
      "code_challenge",
      "code_challenge_method",
      "redirect_uri",
      "dpop_jkt",
    ];
    const missingParams = searchParams
      ? requiredParams.filter((param) => !searchParams.get(param))
      : requiredParams;
    if (missingParams.length > 0) {
      problems.push("OAuth2 start redirect is missing required authorization parameters");
    }
    if (searchParams && searchParams.get("code_challenge_method") !== "S256") {
      problems.push("OAuth2 start redirect must use PKCE S256");
    }
    if (redirectUrl && !redirectUrl.pathname.endsWith("/oauth2.1/authorize")) {
      problems.push("OAuth2 start redirect does not target the OAuth2.1 authorize endpoint");
    }

    addCheck(
      checks,
      "frontend-oauth2-start-redirect",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "OAuth2 start endpoint redirects with state, PKCE and DPoP parameters"
        : "OAuth2 start endpoint redirect does not match the expected OAuth2.1 contract",
      {
        startUrl: authConfigResult.oauthStartUrl,
        status: response.status,
        problems,
        redirectOrigin: redirectUrl?.origin ?? null,
        redirectPath: redirectUrl?.pathname ?? null,
        queryParameters: redirectUrl ? [...redirectUrl.searchParams.keys()].sort() : [],
        missingParams,
        codeChallengeMethod: searchParams?.get("code_challenge_method") ?? null,
        redirectUri: redirectUri || null,
        redirectUriOrigin: redirectUriUrl?.origin ?? null,
        redirectUriPath: redirectUriUrl?.pathname ?? null,
      },
    );
    return {
      ok: problems.length === 0,
      oauthStartUrl: authConfigResult.oauthStartUrl,
      redirectUrl,
      redirectUri,
      redirectUriUrl,
    };
  } catch (error) {
    addCheck(checks, "frontend-oauth2-start-redirect", "fail", "OAuth2 start endpoint is not reachable", {
      startUrl: authConfigResult.oauthStartUrl,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
    return null;
  }
}

function normalizePathname(value, fallback) {
  const text = String(value ?? "").trim() || fallback;
  return text.startsWith("/") ? text : `/${text}`;
}

function configuredOAuthRedirectURL(integrationEnv) {
  const callbackPath = normalizePathname(
    integrationEnv.OAUTH2_CALLBACK_PATH,
    "/api/auth/oauth2/callback",
  );
  const fullRedirectUri = integrationEnv.OAUTH2_REDIRECT_URI?.trim();
  if (fullRedirectUri) {
    try {
      return { url: new URL(fullRedirectUri), source: "OAUTH2_REDIRECT_URI", callbackPath, explicit: true };
    } catch {
      return { url: null, source: "OAUTH2_REDIRECT_URI", callbackPath, explicit: true };
    }
  }

  const baseValue = integrationEnv.OAUTH2_REDIRECT_BASE_URL?.trim();
  const base = (baseValue || "http://localhost:3000").replace(/\/$/, "");
  try {
    return {
      url: new URL(`${base}${callbackPath}`),
      source: "OAUTH2_REDIRECT_BASE_URL",
      callbackPath,
      explicit: Boolean(baseValue),
    };
  } catch {
    return {
      url: null,
      source: "OAUTH2_REDIRECT_BASE_URL",
      callbackPath,
      explicit: Boolean(baseValue),
    };
  }
}

function checkOAuthCallbackRedirectContract(checks, backendOrigin, oauthStartResult, integrationEnv = {}) {
  const problems = [];
  const details = {
    backendOAuthHandlerPath: path.relative(repoRoot, backendOAuthHandlerPath),
    frontendExplorePagePath: path.relative(repoRoot, frontendExplorePagePath),
    frontendRootLayoutPath: path.relative(repoRoot, frontendRootLayoutPath),
    frontendOAuthCallbackBootstrapPath: path.relative(repoRoot, frontendOAuthCallbackBootstrapPath),
    frontendOAuthCallbackHandlerPath: path.relative(repoRoot, frontendOAuthCallbackHandlerPath),
    backendEnvExamplePath: path.relative(repoRoot, backendEnvExamplePath),
  };

  if (oauthStartResult?.disabled) {
    addCheck(
      checks,
      "frontend-oauth2-callback-contract",
      "pass",
      "OAuth2 is disabled; callback redirect contract is not required",
      details,
    );
    return;
  }

  if (!oauthStartResult) {
    addCheck(
      checks,
      "frontend-oauth2-callback-contract",
      "fail",
      "OAuth2 callback redirect contract cannot be checked without a valid start redirect",
      details,
    );
    return;
  }

  let backendOriginUrl = null;
  try {
    backendOriginUrl = new URL(backendOrigin);
  } catch {
    problems.push("frontend-configured backend origin is not a valid URL");
  }
  const configuredRedirect = configuredOAuthRedirectURL(integrationEnv);
  if (!configuredRedirect.url) {
    problems.push("configured OAuth2 redirect URI is not a valid URL");
  }

  if (!oauthStartResult.redirectUriUrl) {
    problems.push("OAuth2 start redirect did not expose a parseable redirect_uri");
  } else {
    if (
      configuredRedirect.explicit &&
      configuredRedirect.url &&
      oauthStartResult.redirectUriUrl.origin !== configuredRedirect.url.origin
    ) {
      problems.push("OAuth2 redirect_uri origin does not match the configured OAuth2 redirect origin");
    }
    if (oauthStartResult.redirectUriUrl.pathname !== configuredRedirect.callbackPath) {
      problems.push("OAuth2 redirect_uri does not target the backend OAuth2 callback endpoint");
    }
  }

  const backendOAuthHandler = fileText(backendOAuthHandlerPath);
  const backendEnvExample = fileText(backendEnvExamplePath);
  const explorePage = fileText(frontendExplorePagePath);
  const rootLayout = fileText(frontendRootLayoutPath);
  const callbackBootstrap = fileText(frontendOAuthCallbackBootstrapPath);
  const callbackHandler = fileText(frontendOAuthCallbackHandlerPath);
  const requiredPatterns = [
    {
      fileText: backendOAuthHandler,
      pattern: '"redirect_uri":          {h.oauthCallbackURL(c)}',
      problem: "backend OAuth2 login does not send the backend callback URL as redirect_uri",
    },
    {
      fileText: backendOAuthHandler,
      pattern: "h.Config.OAuth2.RedirectBaseURL",
      problem: "backend OAuth2 callback URL does not support OAUTH2_REDIRECT_BASE_URL",
    },
    {
      fileText: backendOAuthHandler,
      pattern: 'path = firstNonEmpty(path, "/api/auth/oauth2/callback")',
      problem: "backend OAuth2 callback URL fallback path is not /api/auth/oauth2/callback",
    },
    {
      fileText: backendEnvExample,
      pattern: "OAUTH2_REDIRECT_BASE_URL=http://localhost:3000",
      problem: "backend .env.example does not document the local OAuth2 redirect base URL default",
    },
    {
      fileText: backendOAuthHandler,
      pattern: 'c.Redirect(http.StatusFound, "/explore?"+values.Encode())',
      problem: "backend OAuth2 callback does not redirect successful local-token logins to /explore",
    },
    {
      fileText: backendOAuthHandler,
      pattern: '"oauth2_login":  {"success"}',
      problem: "backend OAuth2 callback does not mark successful frontend redirects with oauth2_login=success",
    },
    {
      fileText: backendOAuthHandler,
      pattern: '"access_token":  {localAccess}',
      problem: "backend OAuth2 callback does not pass the local access token to the frontend callback",
    },
    {
      fileText: backendOAuthHandler,
      pattern: '"refresh_token": {localRefresh}',
      problem: "backend OAuth2 callback does not pass the local refresh token to the frontend callback",
    },
    {
      fileText: explorePage,
      pattern: "export default async function ExplorePage",
      problem: "frontend /explore page is missing",
    },
    {
      fileText: explorePage,
      pattern: "<ExploreFeed initialData={initialData} />",
      problem: "frontend /explore page does not render the exploration feed",
    },
    {
      fileText: explorePage,
      pattern: 'redirect("/login")',
      problem: "frontend /explore page does not redirect unauthorized SSR loads to /login",
    },
    {
      fileText: explorePage,
      pattern: "isOAuthSuccessCallback(resolvedSearchParams)",
      problem: "frontend /explore page does not let OAuth2 success callbacks render before SSR login redirects",
    },
    {
      fileText: explorePage,
      pattern: "return emptyInitialFeedData",
      problem: "frontend /explore page does not provide a safe shell for client-side OAuth token bootstrap",
    },
    {
      fileText: rootLayout,
      pattern: "<OAuthCallbackBootstrap />",
      problem: "root layout does not mount the OAuth callback bootstrap script",
    },
    {
      fileText: callbackBootstrap,
      pattern: 'window.localStorage.setItem("yuem_access_token", accessToken)',
      problem: "OAuth callback bootstrap does not save the access token before hydration",
    },
    {
      fileText: callbackBootstrap,
      pattern: "document.cookie",
      problem: "OAuth callback bootstrap does not expose the access token to SSR through an auth cookie",
    },
    {
      fileText: callbackBootstrap,
      pattern: "yuem_access_token=",
      problem: "OAuth callback bootstrap does not write the expected ordinary-user auth cookie",
    },
    {
      fileText: callbackBootstrap,
      pattern: "window.history.replaceState",
      problem: "OAuth callback bootstrap does not clean token-bearing URL parameters",
    },
    {
      fileText: callbackHandler,
      pattern: 'params.get("oauth2_login") === "success"',
      problem: "OAuth callback handler does not detect oauth2_login=success callbacks",
    },
    {
      fileText: callbackHandler,
      pattern: "getCurrentUser({ token: accessToken }, { auth: false })",
      problem: "OAuth callback handler does not hydrate the current user with the returned access token",
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
      fileText: callbackHandler,
      pattern: 'router.replace("/")',
      problem: "OAuth callback handler does not leave the token-bearing callback URL after hydration",
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

  addCheck(
    checks,
    "frontend-oauth2-callback-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "OAuth2 backend callback and frontend callback target are aligned"
      : "OAuth2 backend callback and frontend callback target are not aligned",
    {
      ...details,
      redirectUri: oauthStartResult.redirectUri || null,
      redirectUriOrigin: oauthStartResult.redirectUriUrl?.origin ?? null,
      redirectUriPath: oauthStartResult.redirectUriUrl?.pathname ?? null,
      expectedBackendOrigin: backendOriginUrl?.origin ?? null,
      expectedRedirectUri: configuredRedirect.url?.toString() ?? null,
      expectedRedirectSource: configuredRedirect.source,
      problems,
    },
  );
}

