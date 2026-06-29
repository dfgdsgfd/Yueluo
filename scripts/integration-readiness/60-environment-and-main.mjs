function checkFrontendApiRouteAudit(checks) {
  const details = {
    frontendRootPath: path.relative(repoRoot, frontendRootPath),
    auditScriptPath: path.relative(repoRoot, frontendApiAuditScriptPath),
  };

  if (!fs.existsSync(frontendApiAuditScriptPath)) {
    addCheck(
      checks,
      "frontend-api-route-audit",
      "fail",
      "frontend API route audit script is missing",
      details,
    );
    return;
  }

  const result = spawnSync(process.execPath, [frontendApiAuditScriptPath], {
    cwd: frontendRootPath,
    encoding: "utf8",
    maxBuffer: 10 * 1024 * 1024,
  });

  if (result.error) {
    addCheck(
      checks,
      "frontend-api-route-audit",
      "fail",
      "frontend API route audit could not be executed",
      {
        ...details,
        error: result.error.message,
      },
    );
    return;
  }

  const stdout = result.stdout.trim();
  let report = null;
  try {
    report = JSON.parse(stdout);
  } catch (error) {
    addCheck(
      checks,
      "frontend-api-route-audit",
      "fail",
      "frontend API route audit did not emit valid JSON",
      {
        ...details,
        exitStatus: result.status,
        error: error.message,
        stderr: result.stderr.trim(),
      },
    );
    return;
  }

  const summary = isRecord(report.summary) ? report.summary : {};
  const unmatchedApiCalls = Number(summary.unmatchedApiCalls);
  const broadDynamicMatches = Number(summary.broadDynamicMatches);
  const problems = [];

  if (result.status !== 0) {
    problems.push(`audit exited with status ${result.status}`);
  }
  if (report.status !== "pass") {
    problems.push(`audit status is ${String(report.status)}`);
  }
  if (!Number.isFinite(unmatchedApiCalls)) {
    problems.push("audit summary is missing unmatchedApiCalls");
  } else if (unmatchedApiCalls > 0) {
    problems.push(`${unmatchedApiCalls} frontend API call(s) are unmatched by backend routes`);
  }
  if (!Number.isFinite(broadDynamicMatches)) {
    problems.push("audit summary is missing broadDynamicMatches");
  } else if (broadDynamicMatches > 0) {
    problems.push(`${broadDynamicMatches} frontend API call(s) only matched broad dynamic backend routes`);
  }

  addCheck(
    checks,
    "frontend-api-route-audit",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend API calls are covered by backend routes"
      : "frontend API route audit failed",
    {
      ...details,
      auditStatus: report.status,
      exitStatus: result.status,
      frontendApiCalls: Number(summary.frontendApiCalls),
      unmatchedApiCalls,
      broadDynamicMatches,
      backendApiRoutes: Number(summary.backendApiRoutes),
      backendWebSockets: Number(summary.backendWebSockets),
      unmatchedSamples: Array.isArray(report.unmatched) ? report.unmatched.slice(0, 5) : [],
      broadDynamicMatchSamples: Array.isArray(report.broadDynamicMatches)
        ? report.broadDynamicMatches.slice(0, 5)
        : [],
      stderr: result.stderr.trim(),
      problems,
    },
  );
}

function checkFrontendEnvExampleContract(checks) {
  const details = {
    envExamplePath: path.relative(repoRoot, frontendEnvExamplePath),
  };

  if (!fs.existsSync(frontendEnvExamplePath)) {
    addCheck(checks, "frontend-env-example-contract", "fail", "frontend .env.example is missing", details);
    return;
  }

  const text = fileText(frontendEnvExamplePath);
  const activeValues = loadEnvFile(frontendEnvExamplePath);
  const requiredKeys = [
    "BACKEND_ORIGIN",
    "NEXT_PUBLIC_BACKEND_ORIGIN",
    "NEXT_PUBLIC_API_BASE_URL",
    "API_BASE_URL",
    "FEED_FIXTURE_FALLBACK",
    "INTEGRATION_HTTP_TIMEOUT_MS",
    "INTEGRATION_HTTP_RETRY_COUNT",
    "INTEGRATION_ENABLE_WRITE_SMOKE",
    "INTEGRATION_WRITE_SMOKE_POST_ID",
    "INTEGRATION_USER_A_ACCESS_TOKEN",
    "INTEGRATION_USER_A_ID",
    "INTEGRATION_USER_A_PASSWORD",
    "INTEGRATION_USER_B_ACCESS_TOKEN",
    "INTEGRATION_USER_B_ID",
    "INTEGRATION_USER_B_PASSWORD",
    "INTEGRATION_ADMIN_ACCESS_TOKEN",
    "INTEGRATION_ADMIN_USERNAME",
    "INTEGRATION_ADMIN_PASSWORD",
  ];
  const backendUrlKeys = [
    "BACKEND_ORIGIN",
    "NEXT_PUBLIC_BACKEND_ORIGIN",
    "NEXT_PUBLIC_API_BASE_URL",
    "API_BASE_URL",
  ];
  const requiredActiveBackendOrigins = ["BACKEND_ORIGIN", "NEXT_PUBLIC_BACKEND_ORIGIN"];
  const fixtureFallbackKeys = ["FEED_FIXTURE_FALLBACK", "NEXT_PUBLIC_FEED_FIXTURE_FALLBACK"];
  const sensitiveKeys = [
    "INTEGRATION_USER_A_ACCESS_TOKEN",
    "INTEGRATION_USER_A_PASSWORD",
    "INTEGRATION_USER_B_ACCESS_TOKEN",
    "INTEGRATION_USER_B_PASSWORD",
    "INTEGRATION_ADMIN_ACCESS_TOKEN",
    "INTEGRATION_ADMIN_PASSWORD",
  ];
  const missingKeys = requiredKeys.filter((key) => !exampleEnvIncludesKey(text, key));
  const invalidUrlKeys = [];
  const urlSamples = {};
  const problems = [];

  for (const key of backendUrlKeys) {
    const sample = exampleEnvValue(text, key);
    if (!sample) {
      continue;
    }
    urlSamples[key] = sample;
    try {
      const parsed = new URL(sample);
      if (!["http:", "https:"].includes(parsed.protocol)) {
        invalidUrlKeys.push(key);
      }
    } catch {
      invalidUrlKeys.push(key);
    }
  }

  for (const key of requiredActiveBackendOrigins) {
    const value = activeValues[key]?.trim();
    if (!value) {
      problems.push(`${key} should be active in .env.example for local Next rewrites`);
      continue;
    }
    try {
      new URL(value);
    } catch {
      problems.push(`${key} is not a valid URL`);
    }
  }

  const activeFixtureFallbackVariables = fixtureFallbackKeys.filter((key) => {
    const value = activeValues[key]?.trim();
    return value ? isTruthyEnvValue(value) : false;
  });
  const activeSensitiveVariables = sensitiveKeys.filter((key) => activeValues[key]?.trim());

  if (missingKeys.length > 0) {
    problems.push("frontend .env.example is missing integration variables");
  }
  if (invalidUrlKeys.length > 0) {
    problems.push("frontend .env.example contains invalid backend URL samples");
  }
  if (activeFixtureFallbackVariables.length > 0) {
    problems.push("frontend .env.example enables fixture fallback by default");
  }
  if (activeSensitiveVariables.length > 0) {
    problems.push("frontend .env.example contains active token/password placeholders");
  }

  addCheck(
    checks,
    "frontend-env-example-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend .env.example documents backend origin and integration variables"
      : "frontend .env.example does not cover integration configuration",
    {
      ...details,
      requiredKeys,
      missingKeys,
      backendUrlKeys,
      urlSamples,
      invalidUrlKeys,
      activeBackendOrigins: requiredActiveBackendOrigins,
      activeFixtureFallbackVariables,
      activeSensitiveVariables,
      problems,
    },
  );
}

function checkFrontendEnvDocContract(checks) {
  const details = {
    envDocPath: path.relative(repoRoot, frontendEnvDocPath),
    envExamplePath: path.relative(repoRoot, frontendEnvExamplePath),
  };

  if (!fs.existsSync(frontendEnvDocPath)) {
    addCheck(checks, "frontend-env-doc-contract", "fail", "frontend integration environment doc is missing", details);
    return;
  }

  const doc = fileText(frontendEnvDocPath);
  const envExample = fileText(frontendEnvExamplePath);
  const requiredPatterns = [
    "front-end-nextjs/.env.local",
    "backend-gin/.env",
    "BACKEND_ORIGIN",
    "NEXT_PUBLIC_BACKEND_ORIGIN",
    "NEXT_PUBLIC_API_BASE_URL",
    "API_BASE_URL",
    "OAUTH2_REDIRECT_URI",
    "OAUTH2_REDIRECT_BASE_URL",
    "OAUTH2_CALLBACK_PATH",
    "INTEGRATION_USER_A_ACCESS_TOKEN",
    "INTEGRATION_USER_A_ID",
    "INTEGRATION_USER_A_PASSWORD",
    "INTEGRATION_USER_B_ACCESS_TOKEN",
    "INTEGRATION_USER_B_ID",
    "INTEGRATION_USER_B_PASSWORD",
    "INTEGRATION_ADMIN_ACCESS_TOKEN",
    "INTEGRATION_ADMIN_USERNAME",
    "INTEGRATION_ADMIN_PASSWORD",
    "INTEGRATION_ENABLE_WRITE_SMOKE",
    "INTEGRATION_WRITE_SMOKE_POST_ID",
    "user-withdraw-payment-code-write-smoke",
    "user-draft-post-write-smoke",
    "user-post-interaction-write-smoke",
    "user-cross-account-follow-write-smoke",
    "user-cross-account-im-write-smoke",
    "admin-runtime-toggle-write-smoke",
    "负向验证",
    "不要把真实 token",
  ];
  const missingPatterns = requiredPatterns.filter((pattern) => !doc.includes(pattern));
  const problems = [];
  const envExampleReferencesDoc = envExample.includes("frontend-backend-api-integration-env.md");

  if (missingPatterns.length > 0) {
    problems.push("frontend integration environment doc is missing required configuration notes");
  }
  if (!envExampleReferencesDoc) {
    problems.push("frontend .env.example does not link to the integration environment doc");
  }

  addCheck(
    checks,
    "frontend-env-doc-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend integration environment doc covers runtime and smoke variables"
      : "frontend integration environment doc is incomplete",
    {
      ...details,
      requiredPatterns,
      missingPatterns,
      envExampleReferencesDoc,
      problems,
    },
  );
}

function checkFrontendBackendAddressStaticContract(checks) {
  const problems = [];
  const details = {
    frontendNextConfigPath: path.relative(repoRoot, frontendNextConfigPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
  };
  const nextConfig = fileText(frontendNextConfigPath);
  const frontendApi = fileText(frontendApiPath);
  const requiredPatterns = [
    {
      fileText: nextConfig,
      pattern: "async rewrites()",
      problem: "next.config.ts does not define rewrites",
    },
    {
      fileText: nextConfig,
      pattern: "process.env.BACKEND_ORIGIN",
      problem: "next.config.ts rewrite does not read BACKEND_ORIGIN",
    },
    {
      fileText: nextConfig,
      pattern: "process.env.NEXT_PUBLIC_BACKEND_ORIGIN",
      problem: "next.config.ts rewrite does not read NEXT_PUBLIC_BACKEND_ORIGIN",
    },
    {
      fileText: nextConfig,
      pattern: 'source: "/api/:path*"',
      problem: "next.config.ts rewrite does not proxy /api/:path*",
    },
    {
      fileText: nextConfig,
      pattern: 'destination: `${backendOrigin.replace(/\\/$/, "")}/api/:path*`',
      problem: "next.config.ts rewrite does not normalize the backend origin for /api/:path*",
    },
    {
      fileText: nextConfig,
      pattern: "DEFAULT_DEV_BACKEND_ORIGIN",
      problem: "next.config.ts does not provide a local development backend fallback",
    },
    {
      fileText: frontendApi,
      pattern: "function getApiBaseUrl()",
      problem: "frontend API client does not centralize backend base URL resolution",
    },
    {
      fileText: frontendApi,
      pattern: "process.env.NEXT_PUBLIC_API_BASE_URL ?? \"\"",
      problem: "browser API calls do not support NEXT_PUBLIC_API_BASE_URL direct backend mode",
    },
    {
      fileText: frontendApi,
      pattern: "process.env.API_BASE_URL ??",
      problem: "server API calls do not prefer API_BASE_URL",
    },
    {
      fileText: frontendApi,
      pattern: "process.env.BACKEND_ORIGIN ??",
      problem: "server API calls do not fall back to BACKEND_ORIGIN",
    },
    {
      fileText: frontendApi,
      pattern: "process.env.NEXT_PUBLIC_BACKEND_ORIGIN ??",
      problem: "server API calls do not fall back to NEXT_PUBLIC_BACKEND_ORIGIN",
    },
    {
      fileText: frontendApi,
      pattern: "new URL(path, apiBaseUrl || window.location.origin)",
      problem: "frontend API client does not keep browser requests on the Next origin when no direct API base is set",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  addCheck(
    checks,
    "frontend-backend-address-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend backend address configuration is wired into rewrites and API client"
      : "frontend backend address configuration is not fully wired",
    {
      ...details,
      problems,
    },
  );
}

async function checkFrontendEnv(checks, frontendEnv, integrationEnv = {}) {
  const backendOrigin = frontendEnv.BACKEND_ORIGIN || frontendEnv.NEXT_PUBLIC_BACKEND_ORIGIN;
  if (!backendOrigin) {
    addCheck(checks, "frontend-backend-origin", "fail", "frontend backend origin is not configured");
  } else {
    const normalizedOrigin = backendOrigin.replace(/\/$/, "");
    addCheck(checks, "frontend-backend-origin", "pass", "frontend backend origin is configured", {
      origin: normalizedOrigin,
      directBrowserOrigin: frontendEnv.NEXT_PUBLIC_API_BASE_URL
        ? frontendEnv.NEXT_PUBLIC_API_BASE_URL.replace(/\/$/, "")
        : null,
    });

    const frontendBackendHealthUrl = healthUrlFromOrigin(normalizedOrigin);
    if (!frontendBackendHealthUrl) {
      addCheck(checks, "frontend-backend-health", "fail", "frontend backend origin is not a valid URL", {
        origin: normalizedOrigin,
      });
    } else {
      await checkHttpHealth(
        checks,
        "frontend-backend-health",
        "frontend-configured backend",
        frontendBackendHealthUrl,
      );
      const authConfigResult = await checkAuthConfigContract(checks, normalizedOrigin);
      await checkAuthPublicAuxContract(checks, normalizedOrigin, authConfigResult);
      const oauthStartResult = await checkOAuthStartRedirect(checks, authConfigResult);
      checkOAuthCallbackRedirectContract(checks, normalizedOrigin, oauthStartResult, integrationEnv);
      await checkInitialFeedAccess(checks, normalizedOrigin);
      await checkSearchAccess(checks, normalizedOrigin);
      await checkGuestProtectedReadSurfaces(checks, normalizedOrigin);
      await checkMonetizationConfigContract(checks, normalizedOrigin);
    }
  }

  const fixtureFallbackVariables = [
    "FEED_FIXTURE_FALLBACK",
    "NEXT_PUBLIC_FEED_FIXTURE_FALLBACK",
  ];
  const enabledFixtureFallbackVariables = fixtureFallbackVariables.filter((key) => {
    const value = frontendEnv[key]?.trim();
    return value ? isTruthyEnvValue(value) : false;
  });

  addCheck(
    checks,
    "frontend-feed-fixture-fallback",
    enabledFixtureFallbackVariables.length === 0 ? "pass" : "fail",
    enabledFixtureFallbackVariables.length === 0
      ? "feed fixture fallback is disabled for integration"
      : "feed fixture fallback is enabled and would mask backend failures",
    {
      checkedVariables: fixtureFallbackVariables,
      enabledVariables: enabledFixtureFallbackVariables,
    },
  );

  checkImWebSocketStaticContract(checks, frontendEnv);
  checkBackendRouteMatrixContract(checks);
  checkBackendAuthBootstrapNoDbContract(checks);
  checkOAuthFrontendStaticContract(checks);
  checkOAuthStartUrlStaticContract(checks);
  checkFrontendAuthSessionStaticContract(checks);
  checkFrontendPublishUploadStaticContract(checks);
  checkFrontendContentInteractionStaticContract(checks);
  checkFrontendProfileUserStaticContract(checks);
  checkFrontendNotificationsImStaticContract(checks);
  checkFrontendCreatorWalletAdminStaticContract(checks);
  checkFrontendApiRouteAudit(checks);
  checkFrontendEnvExampleContract(checks);
  checkFrontendEnvDocContract(checks);
  checkFrontendBackendAddressStaticContract(checks);
}

function checkAccountGroup(checks, id, label, env, credentialSets) {
  const satisfiedSet = credentialSets.find((keys) => keys.every((key) => env[key]?.trim()));
  const missingSets = credentialSets.map((keys) => keys.filter((key) => !env[key]?.trim()));
  addCheck(
    checks,
    id,
    satisfiedSet ? "pass" : "fail",
    satisfiedSet
      ? `${label} variables are present`
      : `${label} variables are missing`,
    {
      requiredVariableSets: credentialSets,
      satisfiedVariableSet: satisfiedSet ?? null,
      missingVariableSets: missingSets,
    },
  );
}

async function main() {
  const checks = [];
  const backendEnv = mergeEnvInBackendOrder();
  const frontendEnv = loadFrontendEnv();
  const integrationEnv = {
    ...backendEnv,
    ...frontendEnv,
    ...process.env,
  };
  const frontendBackendOrigin = (
    integrationEnv.BACKEND_ORIGIN || integrationEnv.NEXT_PUBLIC_BACKEND_ORIGIN || ""
  )
    .trim()
    .replace(/\/$/, "");
  httpTimeoutMs = positiveIntegerFromUnknown(
    process.env.INTEGRATION_HTTP_TIMEOUT_MS ??
      integrationEnv.INTEGRATION_HTTP_TIMEOUT_MS,
    DEFAULT_HTTP_TIMEOUT_MS,
  );
  httpRetryCount = nonNegativeIntegerFromUnknown(
    process.env.INTEGRATION_HTTP_RETRY_COUNT ??
      integrationEnv.INTEGRATION_HTTP_RETRY_COUNT,
    DEFAULT_HTTP_RETRY_COUNT,
  );

  for (const command of ["rg", "git", "node", "npm", "go"]) {
    addCheck(
      checks,
      `tool-${command}`,
      commandExists(command) ? "pass" : "fail",
      commandExists(command) ? `${command} is available` : `${command} is missing from PATH`,
    );
  }
  for (const command of ["docker", "psql", "mysql"]) {
    addCheck(
      checks,
      `optional-tool-${command}`,
      commandExists(command) ? "pass" : "warn",
      commandExists(command)
        ? `${command} is available`
        : `${command} is missing; local database bootstrap or manual inspection may be limited`,
    );
  }

  await checkNetworkTarget(checks, "database", "database", databaseTarget(backendEnv));
  await checkNetworkTarget(checks, "redis", "Redis", redisTarget(backendEnv));
  await checkBackendHealth(checks, backendEnv);
  await checkFrontendEnv(checks, frontendEnv, integrationEnv);

  checkAccountGroup(checks, "user-a-account", "ordinary user A test account", integrationEnv, [
    ["INTEGRATION_USER_A_ACCESS_TOKEN"],
    ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
  ]);
  checkAccountGroup(checks, "user-b-account", "ordinary user B test account", integrationEnv, [
    ["INTEGRATION_USER_B_ACCESS_TOKEN"],
    ["INTEGRATION_USER_B_ID", "INTEGRATION_USER_B_PASSWORD"],
  ]);
  checkAccountGroup(checks, "admin-account", "administrator test account", integrationEnv, [
    ["INTEGRATION_ADMIN_ACCESS_TOKEN"],
    ["INTEGRATION_ADMIN_USERNAME", "INTEGRATION_ADMIN_PASSWORD"],
  ]);

  if (!frontendBackendOrigin && shouldRunWriteSmoke(integrationEnv)) {
    for (const id of [
      "user-withdraw-payment-code-write-smoke",
      "user-draft-post-write-smoke",
      "user-post-interaction-write-smoke",
      "user-cross-account-follow-write-smoke",
      "user-cross-account-im-write-smoke",
      "admin-runtime-toggle-write-smoke",
    ]) {
      addCheck(
        checks,
        id,
        "fail",
        "write smoke is enabled but BACKEND_ORIGIN or NEXT_PUBLIC_BACKEND_ORIGIN is missing",
        {
          requiredVariables: ["BACKEND_ORIGIN", "NEXT_PUBLIC_BACKEND_ORIGIN"],
          writeSmokeEnabled: true,
        },
      );
    }
  } else if (frontendBackendOrigin) {
    await checkUserAuthenticatedSmoke(
      checks,
      integrationEnv,
      frontendBackendOrigin,
      "INTEGRATION_USER_A",
      "user-a-authenticated-smoke",
      "ordinary user A",
      true,
    );
    await checkUserAuthenticatedSmoke(
      checks,
      integrationEnv,
      frontendBackendOrigin,
      "INTEGRATION_USER_B",
      "user-b-authenticated-smoke",
      "ordinary user B",
      false,
    );
    await checkUserCrossAccountReadSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkUserWithdrawPaymentCodeWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkUserDraftPostWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkUserPostInteractionWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkUserCrossAccountFollowWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkUserCrossAccountImWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkAdminAuthenticatedSmoke(checks, integrationEnv, frontendBackendOrigin);
    await checkAdminRuntimeToggleWriteSmoke(checks, integrationEnv, frontendBackendOrigin);
  }

  const failed = checks.filter((check) => check.status === "fail");
  const warned = checks.filter((check) => check.status === "warn");
  const report = {
    status: failed.length === 0 ? "ready" : "not-ready",
    summary: {
      checks: checks.length,
      passed: checks.filter((check) => check.status === "pass").length,
      warnings: warned.length,
      failed: failed.length,
    },
    checks,
  };

  console.log(JSON.stringify(report, null, 2));
  if (failed.length > 0) {
    process.exitCode = 1;
  }
}

main().catch((error) => {
  console.error(JSON.stringify({ status: "error", error: error.message }, null, 2));
  process.exitCode = 1;
});
