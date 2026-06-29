function checkBackendRouteMatrixContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    expectedSummary: {
      totalApiRoutes: 506,
      mountedModules: 27,
      webSockets: 1,
    },
  };

  if (!fs.existsSync(backendRouteMatrixPath)) {
    addCheck(
      checks,
      "backend-route-matrix-contract",
      "fail",
      "backend route matrix is missing",
      details,
    );
    return;
  }

  let matrix;
  try {
    matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
  } catch (error) {
    addCheck(
      checks,
      "backend-route-matrix-contract",
      "fail",
      "backend route matrix is not valid JSON",
      {
        ...details,
        error: error.message,
      },
    );
    return;
  }

  const summary = isRecord(matrix.summary) ? matrix.summary : {};
  const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
  const webSockets = Array.isArray(matrix.webSockets) ? matrix.webSockets.filter(isRecord) : [];
  const criticalRoutes = [
    { method: "GET", path: "/api/health", auth: "public" },
    { method: "GET", path: "/api/auth/auth-config", auth: "public" },
    { method: "GET", path: "/api/auth/oauth2/login", auth: "public" },
    { method: "GET", path: "/api/auth/oauth2/callback", auth: "public" },
    { method: "POST", path: "/api/auth/oauth2/mobile-token", auth: "public" },
    { method: "POST", path: "/api/auth/login", auth: "public" },
    { method: "POST", path: "/api/auth/register", auth: "public" },
    { method: "POST", path: "/api/auth/refresh", auth: "public" },
    { method: "POST", path: "/api/auth/logout", auth: "user" },
    { method: "GET", path: "/api/auth/me", auth: "user" },
    { method: "GET", path: "/api/posts/recommended", auth: "optional-note-guest-restricted" },
    { method: "GET", path: "/api/posts/:id", auth: "optional-note-guest-restricted" },
    { method: "PUT", path: "/api/posts/:id", auth: "user" },
    { method: "DELETE", path: "/api/posts/:id", auth: "user" },
    { method: "GET", path: "/api/posts/:id/comments", auth: "optional-note-guest-restricted" },
    { method: "GET", path: "/api/comments", auth: "optional-note-guest-restricted" },
    { method: "POST", path: "/api/comments", auth: "user" },
    { method: "POST", path: "/api/likes", auth: "user" },
    { method: "DELETE", path: "/api/likes", auth: "user" },
    { method: "POST", path: "/api/posts/:id/collect", auth: "user" },
    { method: "POST", path: "/api/upload/single", auth: "user" },
    { method: "POST", path: "/api/upload/video", auth: "user" },
    { method: "POST", path: "/api/upload/attachment", auth: "user" },
    { method: "GET", path: "/api/notifications", auth: "user" },
    { method: "GET", path: "/api/notifications/unread-count", auth: "user" },
    { method: "GET", path: "/api/notifications/system", auth: "user" },
    { method: "GET", path: "/api/notifications/system/popup", auth: "user" },
    { method: "GET", path: "/api/im/conversations", auth: "user" },
    { method: "POST", path: "/api/im/conversations", auth: "user" },
    { method: "GET", path: "/api/im/conversations/:id/messages", auth: "user" },
    { method: "POST", path: "/api/im/conversations/:id/messages", auth: "user" },
    { method: "POST", path: "/api/im/messages/:id/read", auth: "user" },
    { method: "GET", path: "/api/creator-center/config", auth: "public" },
    { method: "GET", path: "/api/creator-center/overview", auth: "user" },
    { method: "GET", path: "/api/balance/config", auth: "public" },
    { method: "GET", path: "/api/balance/user-balance", auth: "user" },
    { method: "GET", path: "/api/withdraw/wallet", auth: "user" },
    { method: "GET", path: "/api/auth/admin/me", auth: "user" },
    { method: "GET", path: "/api/admin/stats/overview", auth: "admin" },
    { method: "GET", path: "/api/admin/performance", auth: "admin" },
    { method: "GET", path: "/api/admin/observability/access-log", auth: "admin" },
    { method: "GET", path: "/api/admin/database/vacuum-config", auth: "admin" },
    { method: "PUT", path: "/api/admin/database/vacuum-config", auth: "admin" },
    { method: "POST", path: "/api/admin/database/vacuum-analyze", auth: "admin" },
    { method: "GET", path: "/api/withdraw/admin/orders", auth: "admin" },
  ];
  const missingCriticalRoutes = [];
  const mismatchedCriticalRoutes = [];
  const routeKeys = new Set();
  const duplicateRoutes = [];

  if (summary.totalApiRoutes !== details.expectedSummary.totalApiRoutes) {
    problems.push("backend route matrix totalApiRoutes changed unexpectedly");
  }
  if (summary.mountedModules !== details.expectedSummary.mountedModules) {
    problems.push("backend route matrix mountedModules changed unexpectedly");
  }
  if (routes.length !== details.expectedSummary.totalApiRoutes) {
    problems.push("backend route matrix route count does not match the expected total");
  }
  if (summary.totalApiRoutes !== undefined && routes.length !== summary.totalApiRoutes) {
    problems.push("backend route matrix route count does not match summary.totalApiRoutes");
  }
  if (webSockets.length !== details.expectedSummary.webSockets) {
    problems.push("backend route matrix WebSocket count changed unexpectedly");
  }

  for (const route of routes) {
    const key = `${route.method} ${route.path}`;
    if (routeKeys.has(key)) {
      duplicateRoutes.push(key);
    }
    routeKeys.add(key);
  }

  const nonNativeRoutes = routes
    .filter((route) => route.status !== "native-gin")
    .map((route) => ({
      method: route.method,
      path: route.path,
      status: route.status ?? null,
    }));
  const nonNativeWebSockets = webSockets
    .filter((entry) => entry.status !== "native-gin")
    .map((entry) => ({
      path: entry.path,
      status: entry.status ?? null,
    }));

  if (duplicateRoutes.length > 0) {
    problems.push("backend route matrix contains duplicate method/path entries");
  }
  if (nonNativeRoutes.length > 0 || nonNativeWebSockets.length > 0) {
    problems.push("backend route matrix contains routes that are not native-gin");
  }

  for (const expected of criticalRoutes) {
    const matches = routes.filter(
      (route) => route.method === expected.method && route.path === expected.path,
    );
    const actual = matches[0] ?? null;
    const result = {
      method: expected.method,
      path: expected.path,
      expectedAuth: expected.auth,
      actualAuth: actual?.auth ?? null,
      actualStatus: actual?.status ?? null,
      ok:
        matches.length === 1 &&
        actual?.auth === expected.auth &&
        actual?.status === "native-gin",
    };

    if (matches.length === 0) {
      missingCriticalRoutes.push(`${expected.method} ${expected.path}`);
      continue;
    }
    if (matches.length > 1 || !result.ok) {
      mismatchedCriticalRoutes.push(result);
    }
  }

  if (missingCriticalRoutes.length > 0) {
    problems.push("backend route matrix is missing critical frontend integration routes");
  }
  if (mismatchedCriticalRoutes.length > 0) {
    problems.push("backend route matrix critical route auth/status does not match expectations");
  }

  const imWebSocket = webSockets.find((entry) => entry.path === "/api/im/ws") ?? null;
  if (!imWebSocket) {
    problems.push("backend route matrix does not expose /api/im/ws WebSocket");
  } else if (
    imWebSocket.auth !== "query-token-and-redis-session" ||
    imWebSocket.status !== "native-gin"
  ) {
    problems.push("backend route matrix /api/im/ws WebSocket auth/status is unexpected");
  }

  addCheck(
    checks,
    "backend-route-matrix-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "backend route matrix preserves native-gin critical integration routes"
      : "backend route matrix does not preserve the expected integration contract",
    {
      ...details,
      actualSummary: {
        totalApiRoutes: summary.totalApiRoutes ?? null,
        mountedModules: summary.mountedModules ?? null,
        webSockets: webSockets.length,
      },
      duplicateRoutes: duplicateRoutes.slice(0, 10),
      nonNativeRoutes: nonNativeRoutes.slice(0, 10),
      nonNativeWebSockets: nonNativeWebSockets.slice(0, 10),
      criticalRouteCount: criticalRoutes.length,
      missingCriticalRoutes,
      mismatchedCriticalRoutes,
      imWebSocket: imWebSocket
        ? {
            path: imWebSocket.path,
            auth: imWebSocket.auth ?? null,
            status: imWebSocket.status ?? null,
          }
        : null,
      problems,
    },
  );
}

function checkBackendAuthBootstrapNoDbContract(checks) {
  const problems = [];
  const details = {
    backendServerRouterPath: path.relative(repoRoot, backendServerRouterPath),
    backendMatrixModulesPath: path.relative(repoRoot, backendMatrixModulesPath),
    backendServerRouterTestPath: path.relative(repoRoot, backendServerRouterTestPath),
  };
  const router = fileText(backendServerRouterPath);
  const matrixModules = fileText(backendMatrixModulesPath);
  const routerTest = fileText(backendServerRouterTestPath);
  const databaseAvailabilityBody = goFunctionBody(router, "databaseAvailability");
  const databaseOptionalPathBody = goFunctionBody(router, "databaseOptionalPath");
  const requiredOptionalPaths = [
    "/api/health",
    "/_gin_migration/status",
    "/api/auth/auth-config",
    "/api/auth/email-config",
    "/api/auth/captcha",
    "/api/auth/oauth2/login",
    "/api/auth/oauth2/callback",
  ];
  const forbiddenOptionalPaths = [
    "/api/auth/login",
    "/api/auth/register",
    "/api/auth/me",
    "/api/posts",
  ];
  const authDispatchPatterns = [
    {
      pattern: 'method == http.MethodGet && path == "/api/auth/auth-config"',
      problem: "AuthMatrix does not dispatch GET /api/auth/auth-config",
    },
    {
      pattern: "h.authConfig(c)",
      problem: "AuthMatrix does not call authConfig",
    },
    {
      pattern: 'method == http.MethodGet && path == "/api/auth/email-config"',
      problem: "AuthMatrix does not dispatch GET /api/auth/email-config",
    },
    {
      pattern: "h.authEmailConfig(c)",
      problem: "AuthMatrix does not call authEmailConfig",
    },
    {
      pattern: 'method == http.MethodGet && path == "/api/auth/captcha"',
      problem: "AuthMatrix does not dispatch GET /api/auth/captcha",
    },
    {
      pattern: "h.authCaptcha(c)",
      problem: "AuthMatrix does not call authCaptcha",
    },
    {
      pattern: 'method == http.MethodGet && path == "/api/auth/oauth2/login"',
      problem: "AuthMatrix does not dispatch GET /api/auth/oauth2/login",
    },
    {
      pattern: "h.authOAuthLogin(c)",
      problem: "AuthMatrix does not call authOAuthLogin",
    },
    {
      pattern: 'method == http.MethodGet && path == "/api/auth/oauth2/callback"',
      problem: "AuthMatrix does not dispatch GET /api/auth/oauth2/callback",
    },
    {
      pattern: "h.authOAuthCallback(c)",
      problem: "AuthMatrix does not call authOAuthCallback",
    },
  ];
  const testPatterns = [
    "TestDatabaseOptionalAuthBootstrapRoutes",
    'path: "/api/auth/auth-config"',
    'path: "/api/auth/email-config"',
    'path: "/api/auth/captcha"',
    'httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/login", nil)',
    'httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/callback?code=sample&state=missing", nil)',
    'httptest.NewRequest(http.MethodGet, "/api/posts?page=1&limit=1", nil)',
    "want 500",
  ];
  const missingOptionalPaths = requiredOptionalPaths.filter(
    (routePath) => !databaseOptionalPathBody.includes(`"${routePath}"`),
  );
  const wronglyOptionalPaths = forbiddenOptionalPaths.filter((routePath) =>
    databaseOptionalPathBody.includes(`"${routePath}"`),
  );
  const missingAuthDispatchPatterns = authDispatchPatterns
    .filter((check) => !matrixModules.includes(check.pattern))
    .map((check) => check.problem);
  const missingTestPatterns = testPatterns.filter((pattern) => !routerTest.includes(pattern));

  if (!databaseAvailabilityBody) {
    problems.push("databaseAvailability function is missing or cannot be parsed");
  }
  if (!databaseOptionalPathBody) {
    problems.push("databaseOptionalPath function is missing or cannot be parsed");
  }
  if (
    databaseAvailabilityBody &&
    !databaseAvailabilityBody.includes("databaseOptionalPath(c.Request.URL.Path, cfg)")
  ) {
    problems.push("databaseAvailability does not call databaseOptionalPath");
  }
  if (
    databaseAvailabilityBody &&
    !databaseAvailabilityBody.includes("AbortWithStatusJSON(http.StatusInternalServerError")
  ) {
    problems.push("databaseAvailability does not reject non-optional requests with HTTP 500");
  }
  if (missingOptionalPaths.length > 0) {
    problems.push("databaseOptionalPath does not allow all auth bootstrap routes");
  }
  if (wronglyOptionalPaths.length > 0) {
    problems.push("databaseOptionalPath allows routes that still require the database");
  }
  if (missingAuthDispatchPatterns.length > 0) {
    problems.push("AuthMatrix does not dispatch all auth bootstrap routes");
  }
  if (missingTestPatterns.length > 0) {
    problems.push("router tests do not cover the no-database auth bootstrap contract");
  }

  addCheck(
    checks,
    "backend-auth-bootstrap-no-db-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "backend no-database auth bootstrap routes remain available"
      : "backend no-database auth bootstrap route contract is not preserved",
    {
      ...details,
      requiredOptionalPaths,
      missingOptionalPaths,
      forbiddenOptionalPaths,
      wronglyOptionalPaths,
      missingAuthDispatchPatterns,
      missingTestPatterns,
      problems,
    },
  );
}
