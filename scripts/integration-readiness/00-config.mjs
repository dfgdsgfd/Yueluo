const { spawnSync, dns, fs, net, path, repoRoot } = globalThis[Symbol.for("yuem.readiness.runtime")];

const backendEnvPath = path.join(repoRoot, "backend-gin", ".env");
const backendEnvExamplePath = path.join(repoRoot, "backend-gin", ".env.example");
const rootEnvPath = path.join(repoRoot, ".env");
const frontendEnvLocalPath = path.join(repoRoot, "front-end-nextjs", ".env.local");
const backendRouteMatrixPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "routes",
  "route-matrix.json",
);
const backendServerRouterPath = path.join(repoRoot, "backend-gin", "internal", "http", "server", "router.go");
const backendServerRouterTestPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "server",
  "router_test.go",
);
const backendMatrixModulesPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "matrix_modules.go",
);
const frontendEnvDocPath = path.join(repoRoot, "frontend-backend-api-integration-env.md");
const frontendRootPath = path.join(repoRoot, "front-end-nextjs");
const frontendEnvExamplePath = path.join(frontendRootPath, ".env.example");
const frontendApiAuditScriptPath = path.join(frontendRootPath, "scripts", "audit-api-routes.mjs");
const frontendNextConfigPath = path.join(frontendRootPath, "next.config.ts");
const frontendApiPath = path.join(repoRoot, "front-end-nextjs", "src", "lib", "api.ts");
const frontendTypesPath = path.join(repoRoot, "front-end-nextjs", "src", "lib", "types.ts");
const frontendLoginFormPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "auth",
  "login-form.tsx",
);
const frontendOAuthCallbackBootstrapPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "auth",
  "oauth-callback-bootstrap.tsx",
);
const frontendOAuthCallbackHandlerPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "auth",
  "oauth-callback-handler.tsx",
);
const frontendRootLayoutPath = path.join(repoRoot, "front-end-nextjs", "src", "app", "layout.tsx");
const frontendLoginPagePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "login", "page.tsx");
const frontendExplorePagePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "explore", "page.tsx");
const backendOAuthHandlerPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "matrix_auth_oauth.go",
);
const backendUploadHandlerPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "upload.go",
);
const backendFileHandlerPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "file.go",
);
const backendFileSigningHandlerPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "file_signing.go",
);
const backendFileHandlerTestPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "handlers",
  "file_test.go",
);
const backendConfigPath = path.join(repoRoot, "backend-gin", "internal", "config", "config.go");
const frontendPublishWorkbenchPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "publish",
  "publish-workbench.tsx",
);
const frontendPublishRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "publish", "page.tsx");
const frontendExploreFeedPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "feed",
  "explore-feed.tsx",
);
const frontendPostDetailDrawerPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "feed",
  "post-detail-drawer.tsx",
);
const frontendMessagesPagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "messages",
  "messages-page.tsx",
);
const frontendNotificationsPagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "notifications",
  "notifications-page.tsx",
);
const frontendSystemNotificationPopupPath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "notifications",
  "system-notification-popup.tsx",
);
const frontendMessagesRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "messages", "page.tsx");
const frontendNotificationsRoutePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "app",
  "notifications",
  "page.tsx",
);
const frontendWalletPagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "wallet",
  "wallet-page.tsx",
);
const frontendWalletRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "wallet", "page.tsx");
const frontendAdminPagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "admin",
  "admin-page.tsx",
);
const frontendAdminRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "admin", "page.tsx");
const frontendProfilePagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "profile",
  "profile-page.tsx",
);
const frontendProfileDataPagePath = path.join(
  repoRoot,
  "front-end-nextjs",
  "src",
  "components",
  "profile",
  "profile-data-page.tsx",
);
const frontendProfileRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "profile", "page.tsx");
const frontendUserRoutePath = path.join(repoRoot, "front-end-nextjs", "src", "app", "user", "[id]", "page.tsx");
const frontendUsersHelperPath = path.join(repoRoot, "front-end-nextjs", "src", "lib", "users.ts");
const DEFAULT_HTTP_TIMEOUT_MS = 5000;
const DEFAULT_HTTP_RETRY_COUNT = 1;
let httpTimeoutMs = positiveIntegerFromUnknown(
  process.env.INTEGRATION_HTTP_TIMEOUT_MS,
  DEFAULT_HTTP_TIMEOUT_MS,
);
let httpRetryCount = nonNegativeIntegerFromUnknown(
  process.env.INTEGRATION_HTTP_RETRY_COUNT,
  DEFAULT_HTTP_RETRY_COUNT,
);

