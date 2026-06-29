function checkFrontendCreatorWalletAdminStaticContract(checks) {
  const problems = [];
  const details = {
    routeMatrixPath: path.relative(repoRoot, backendRouteMatrixPath),
    frontendApiPath: path.relative(repoRoot, frontendApiPath),
    frontendPublishWorkbenchPath: path.relative(repoRoot, frontendPublishWorkbenchPath),
    frontendPublishRoutePath: path.relative(repoRoot, frontendPublishRoutePath),
    frontendWalletPagePath: path.relative(repoRoot, frontendWalletPagePath),
    frontendWalletRoutePath: path.relative(repoRoot, frontendWalletRoutePath),
    frontendAdminPagePath: path.relative(repoRoot, frontendAdminPagePath),
    frontendAdminRoutePath: path.relative(repoRoot, frontendAdminRoutePath),
  };

  const frontendApi = fileText(frontendApiPath);
  const publishWorkbench = fileText(frontendPublishWorkbenchPath);
  const publishRoute = fileText(frontendPublishRoutePath);
  const walletPage = fileText(frontendWalletPagePath);
  const walletRoute = fileText(frontendWalletRoutePath);
  const adminPage = fileText(frontendAdminPagePath);
  const adminRoute = fileText(frontendAdminRoutePath);

  try {
    const matrix = JSON.parse(fs.readFileSync(backendRouteMatrixPath, "utf8"));
    const routes = Array.isArray(matrix.routes) ? matrix.routes.filter(isRecord) : [];
    const expectedRoutes = [
      { method: "GET", path: "/api/balance/config", auth: "public" },
      { method: "GET", path: "/api/balance/recharge-config", auth: "public" },
      { method: "GET", path: "/api/balance/local-points", auth: "user" },
      { method: "GET", path: "/api/balance/user-balance", auth: "user" },
      { method: "GET", path: "/api/balance/orders", auth: "user" },
      { method: "GET", path: "/api/creator-center/config", auth: "public" },
      { method: "GET", path: "/api/creator-center/overview", auth: "user" },
      { method: "GET", path: "/api/creator-center/stats", auth: "user" },
      { method: "GET", path: "/api/creator-center/trends", auth: "user" },
      { method: "GET", path: "/api/creator-center/earnings-log", auth: "user" },
      { method: "GET", path: "/api/creator-center/paid-content", auth: "user" },
      { method: "GET", path: "/api/creator-center/quality-rewards", auth: "user" },
      { method: "POST", path: "/api/creator-center/withdraw", auth: "user" },
      { method: "GET", path: "/api/withdraw/wallet", auth: "user" },
      { method: "GET", path: "/api/withdraw/payment-code", auth: "user" },
      { method: "POST", path: "/api/withdraw/payment-code", auth: "user" },
      { method: "POST", path: "/api/withdraw/apply", auth: "user" },
      { method: "GET", path: "/api/withdraw/orders", auth: "user" },
      { method: "POST", path: "/api/auth/admin/login", auth: "public" },
      { method: "GET", path: "/api/auth/admin/me", auth: "user" },
      { method: "GET", path: "/api/admin/stats/overview", auth: "admin" },
      { method: "GET", path: "/api/admin/ai-review-status", auth: "admin" },
      { method: "POST", path: "/api/admin/ai-review-toggle", auth: "admin" },
      { method: "GET", path: "/api/admin/guest-access-status", auth: "admin" },
      { method: "POST", path: "/api/admin/guest-access-toggle", auth: "admin" },
      { method: "GET", path: "/api/admin/users", auth: "admin" },
      { method: "GET", path: "/api/admin/posts", auth: "admin" },
      { method: "GET", path: "/api/admin/content-review", auth: "admin" },
      { method: "GET", path: "/api/admin/reports", auth: "admin" },
      { method: "GET", path: "/api/withdraw/admin/orders", auth: "admin" },
      { method: "PUT", path: "/api/withdraw/admin/orders/:id/approve", auth: "admin" },
      { method: "PUT", path: "/api/withdraw/admin/orders/:id/reject", auth: "admin" },
      { method: "PUT", path: "/api/withdraw/admin/orders/:id/payout", auth: "admin" },
    ];
    const routeResults = [];

    for (const expected of expectedRoutes) {
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
      routeResults.push(result);
      if (!result.ok) {
        problems.push(`${expected.method} ${expected.path} route auth/status is not aligned`);
      }
    }

    details.creatorWalletAdminRoutes = routeResults;
  } catch (error) {
    problems.push(`backend route matrix cannot be read: ${error.message}`);
  }

  const requiredPatterns = [
    {
      fileText: frontendApi,
      pattern: 'apiGet<BalanceConfigPayload>("/api/balance/config", undefined, { auth: false })',
      problem: "getBalanceConfig does not call public /api/balance/config",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<BalanceRechargeConfigPayload>("/api/balance/recharge-config", undefined,',
      problem: "getBalanceRechargeConfig helper is missing",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<BalanceLocalPointsPayload>("/api/balance/local-points")',
      problem: "getBalanceLocalPoints does not call /api/balance/local-points",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<BalanceUserBalancePayload>("/api/balance/user-balance")',
      problem: "getBalanceUserBalance does not call /api/balance/user-balance",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<BalanceOrdersPayload>("/api/balance/orders"',
      problem: "getBalanceOrders does not call /api/balance/orders",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorConfigPayload>("/api/creator-center/config", undefined, { auth: false })',
      problem: "getCreatorConfig does not call public /api/creator-center/config",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorOverviewPayload>("/api/creator-center/overview")',
      problem: "getCreatorOverview does not call /api/creator-center/overview",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorStatsPayload>("/api/creator-center/stats", { days })',
      problem: "getCreatorStats does not call /api/creator-center/stats with days",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorTrendsPayload>("/api/creator-center/trends", { days })',
      problem: "getCreatorTrends does not call /api/creator-center/trends with days",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorEarningsLogPayload>("/api/creator-center/earnings-log"',
      problem: "getCreatorEarningsLog does not call /api/creator-center/earnings-log",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorPaidContentPayload>("/api/creator-center/paid-content"',
      problem: "getCreatorPaidContent does not call /api/creator-center/paid-content",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<CreatorQualityRewardsPayload>("/api/creator-center/quality-rewards"',
      problem: "getCreatorQualityRewards does not call /api/creator-center/quality-rewards",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<CreatorWithdrawPayload>("/api/creator-center/withdraw", { amount })',
      problem: "withdrawCreatorEarnings does not call POST /api/creator-center/withdraw",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<WithdrawWalletPayload>("/api/withdraw/wallet")',
      problem: "getWithdrawWallet does not call /api/withdraw/wallet",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<WithdrawPaymentCodePayload>("/api/withdraw/payment-code")',
      problem: "getWithdrawPaymentCode does not call GET /api/withdraw/payment-code",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<WithdrawPaymentCodePayload>("/api/withdraw/payment-code", input)',
      problem: "saveWithdrawPaymentCode does not call POST /api/withdraw/payment-code",
    },
    {
      fileText: frontendApi,
      pattern: 'apiPost<WithdrawApplyPayload>("/api/withdraw/apply", input)',
      problem: "applyWithdraw does not call POST /api/withdraw/apply",
    },
    {
      fileText: frontendApi,
      pattern: 'apiGet<WithdrawOrdersPayload>("/api/withdraw/orders"',
      problem: "getWithdrawOrders does not call /api/withdraw/orders",
    },
    {
      fileText: publishRoute,
      pattern: 'import { PublishWorkbench } from "@/components/publish/publish-workbench"',
      problem: "publish route does not render PublishWorkbench",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorOverview()",
      problem: "publish workbench does not load creator overview",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorStats(30)",
      problem: "publish workbench does not load creator stats",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorTrends(14)",
      problem: "publish workbench does not load creator trends",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorEarningsLog({ limit: creatorEarningsLogLimit })",
      problem: "publish workbench does not load creator earnings log",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorPaidContent({ limit: creatorPaidContentLimit })",
      problem: "publish workbench does not load creator paid content",
    },
    {
      fileText: publishWorkbench,
      pattern: "getCreatorQualityRewards({ limit: creatorQualityRewardsLimit })",
      problem: "publish workbench does not load creator quality rewards",
    },
    {
      fileText: publishWorkbench,
      pattern: "async function handleLoadMoreCreatorPanel(panel: CreatorListPanel)",
      problem: "publish workbench does not support creator list pagination",
    },
    {
      fileText: publishWorkbench,
      pattern: "page: nextCreatorListPage(creatorEarningsLog)",
      problem: "creator earnings pagination does not use nextCreatorListPage",
    },
    {
      fileText: publishWorkbench,
      pattern: "page: nextCreatorListPage(creatorPaidContent)",
      problem: "creator paid-content pagination does not use nextCreatorListPage",
    },
    {
      fileText: publishWorkbench,
      pattern: "page: nextCreatorListPage(creatorQualityRewards)",
      problem: "creator rewards pagination does not use nextCreatorListPage",
    },
    {
      fileText: walletRoute,
      pattern: 'import { WalletPage } from "@/components/wallet/wallet-page"',
      problem: "wallet route does not render WalletPage",
    },
    {
      fileText: walletPage,
      pattern: "const token = getStoredAccessToken()",
      problem: "wallet page does not gate private wallet calls with the stored user token",
    },
    {
      fileText: walletPage,
      pattern: '["balanceConfig", getBalanceConfig()]',
      problem: "wallet page does not load balance config",
    },
    {
      fileText: walletPage,
      pattern: '["rechargeConfig", getBalanceRechargeConfig()]',
      problem: "wallet page does not load recharge config",
    },
    {
      fileText: walletPage,
      pattern: "normalizeRechargeOptions(rechargeConfig?.options)",
      problem: "wallet page does not render recharge amount options from recharge-config",
    },
    {
      fileText: walletPage,
      pattern: "normalizeGiftCardOptions(rechargeConfig?.gift_card_purchase?.options)",
      problem: "wallet page does not render gift-card options from recharge-config",
    },
    {
      fileText: walletPage,
      pattern: "formatRechargeAmountRange(rechargeConfig?.min_amount, rechargeConfig?.max_amount)",
      problem: "wallet page does not surface the recharge custom amount range",
    },
    {
      fileText: walletPage,
      pattern: "<RechargeOptionList",
      problem: "wallet page does not include the recharge option list UI",
    },
    {
      fileText: walletPage,
      pattern: '["localPoints", getBalanceLocalPoints()]',
      problem: "wallet page does not load local points",
    },
    {
      fileText: walletPage,
      pattern: '["userBalance", getBalanceUserBalance()]',
      problem: "wallet page does not load external user balance",
    },
    {
      fileText: walletPage,
      pattern: '["creatorOverview", getCreatorOverview()]',
      problem: "wallet page does not load creator overview",
    },
    {
      fileText: walletPage,
      pattern: '["withdrawWallet", getWithdrawWallet()]',
      problem: "wallet page does not load withdraw wallet",
    },
    {
      fileText: walletPage,
      pattern: '["paymentCode", getWithdrawPaymentCode()]',
      problem: "wallet page does not load withdraw payment code",
    },
    {
      fileText: walletPage,
      pattern: '["withdrawOrders", getWithdrawOrders({ limit: 8 })]',
      problem: "wallet page does not load withdraw orders",
    },
    {
      fileText: walletPage,
      pattern: '["balanceOrders", getBalanceOrders({ limit: 5 })]',
      problem: "wallet page does not load balance orders",
    },
    {
      fileText: walletPage,
      pattern: "const nextCode = await saveWithdrawPaymentCode({",
      problem: "wallet page does not save withdraw payment codes",
    },
    {
      fileText: walletPage,
      pattern: "await applyWithdraw({ amount, type: withdrawType })",
      problem: "wallet page does not submit withdraw applications",
    },
    {
      fileText: walletPage,
      pattern: "const payload = await withdrawCreatorEarnings(amount)",
      problem: "wallet page does not transfer creator earnings",
    },
    {
      fileText: walletPage,
      pattern: "await loadWallet({ silent: true })",
      problem: "wallet page does not refresh wallet data after mutations",
    },
    {
      fileText: frontendApi,
      pattern: '"/api/auth/admin/login"',
      problem: "loginAdmin does not call /api/auth/admin/login",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<AdminUser>("/api/auth/admin/me"',
      problem: "getCurrentAdmin does not call /api/auth/admin/me",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<AdminStatsOverviewPayload>("/api/admin/stats/overview"',
      problem: "getAdminStatsOverview does not call /api/admin/stats/overview",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<AdminAiReviewStatusPayload>("/api/admin/ai-review-status"',
      problem: "getAdminAiReviewStatus does not call /api/admin/ai-review-status",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<unknown>("/api/admin/ai-review-toggle"',
      problem: "toggleAdminAiReview does not call /api/admin/ai-review-toggle",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<AdminGuestAccessStatusPayload>("/api/admin/guest-access-status"',
      problem: "getAdminGuestAccessStatus does not call /api/admin/guest-access-status",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<unknown>("/api/admin/guest-access-toggle"',
      problem: "toggleAdminGuestAccess does not call /api/admin/guest-access-toggle",
    },
    {
      fileText: frontendApi,
      pattern: "apiAdminEnvelope<unknown>(`/api/admin/${resource}`",
      problem: "getAdminList does not call /api/admin/:resource through the admin envelope",
    },
    {
      fileText: frontendApi,
      pattern: "pageSize: limit",
      problem: "getAdminList does not pass pageSize compatibility parameter",
    },
    {
      fileText: frontendApi,
      pattern: 'apiAdminRequest<AdminWithdrawOrdersPayload>("/api/withdraw/admin/orders"',
      problem: "getAdminWithdrawOrders does not call /api/withdraw/admin/orders",
    },
    {
      fileText: frontendApi,
      pattern: "`/api/withdraw/admin/orders/${orderId}/${action}`",
      problem: "updateAdminWithdrawOrder does not target withdraw admin action routes",
    },
    {
      fileText: adminRoute,
      pattern: 'import { AdminPage } from "@/components/admin/admin-page"',
      problem: "admin route does not render AdminPage",
    },
    {
      fileText: adminPage,
      pattern: "const storedToken = getStoredAdminAccessToken()",
      problem: "admin page does not bootstrap from stored admin token",
    },
    {
      fileText: adminPage,
      pattern: "setAdmin(getStoredAdminUser())",
      problem: "admin page does not use stored admin identity while booting",
    },
    {
      fileText: adminPage,
      pattern: "const currentAdmin = await getCurrentAdmin(storedToken)",
      problem: "admin page does not verify the stored admin token",
    },
    {
      fileText: adminPage,
      pattern: "const session = await loginAdmin(username.trim(), password)",
      problem: "admin page does not log in through loginAdmin",
    },
    {
      fileText: adminPage,
      pattern: '["stats", getAdminStatsOverview(nextToken)]',
      problem: "admin page does not load stats overview",
    },
    {
      fileText: adminPage,
      pattern: '["ai", getAdminAiReviewStatus(nextToken)]',
      problem: "admin page does not load AI review status",
    },
    {
      fileText: adminPage,
      pattern: '["guest", getAdminGuestAccessStatus(nextToken)]',
      problem: "admin page does not load guest access status",
    },
    {
      fileText: adminPage,
      pattern: "const payload = await getAdminList(resource, {",
      problem: "admin page does not load admin resource lists",
    },
    {
      fileText: adminPage,
      pattern: "const payload = await getAdminWithdrawOrders({",
      problem: "admin page does not load admin withdraw orders",
    },
    {
      fileText: adminPage,
      pattern: "await toggleAdminAiReview(nextEnabled, token)",
      problem: "admin page does not toggle AI review through the API helper",
    },
    {
      fileText: adminPage,
      pattern: "await toggleAdminGuestAccess(nextRestricted, token)",
      problem: "admin page does not toggle guest access through the API helper",
    },
    {
      fileText: adminPage,
      pattern: "await updateAdminWithdrawOrder(orderId, action, {}, token)",
      problem: "admin page does not update withdraw orders through the API helper",
    },
    {
      fileText: adminPage,
      pattern: 'resource: "users"',
      problem: "admin page does not expose the users resource",
    },
    {
      fileText: adminPage,
      pattern: 'resource: "posts"',
      problem: "admin page does not expose the posts resource",
    },
    {
      fileText: adminPage,
      pattern: 'resource: "content-review"',
      problem: "admin page does not expose the content-review resource",
    },
    {
      fileText: adminPage,
      pattern: 'resource: "reports"',
      problem: "admin page does not expose the reports resource",
    },
  ];

  for (const check of requiredPatterns) {
    if (!check.fileText.includes(check.pattern)) {
      problems.push(check.problem);
    }
  }

  addCheck(
    checks,
    "frontend-creator-wallet-admin-contract",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "frontend creator, wallet and admin page contract is aligned with backend routes"
      : "frontend creator, wallet and admin page contract is not aligned with backend routes",
    {
      ...details,
      problems,
    },
  );
}

