async function checkInitialFeedAccess(checks, backendOrigin) {
  const feedUrl = apiUrlFromOrigin(backendOrigin, "/api/posts/recommended?page=1&limit=1");
  const categoriesUrl = apiUrlFromOrigin(backendOrigin, "/api/categories/hot?limit=12");

  if (!feedUrl || !categoriesUrl) {
    addCheck(checks, "frontend-initial-feed-access", "fail", "initial feed URLs cannot be built", {
      origin: backendOrigin,
    });
    return;
  }

  try {
    const [feedResult, categoriesResult] = await Promise.all([
      fetchJsonWithTimeout(feedUrl),
      fetchJsonWithTimeout(categoriesUrl),
    ]);
    const feedUnauthorized = isUnauthorizedApiResult(feedResult);
    const categoriesUnauthorized = isUnauthorizedApiResult(categoriesResult);
    const problems = [];
    let accessMode = "public";

    if (feedUnauthorized && categoriesUnauthorized) {
      accessMode = "guest-restricted";
    } else if (feedUnauthorized || categoriesUnauthorized) {
      accessMode = "mixed";
      problems.push("initial feed endpoints have mixed guest-access behavior");
    } else {
      if (!feedResult.ok) {
        problems.push(`recommended feed returned status ${feedResult.status}`);
      }
      if (!categoriesResult.ok) {
        problems.push(`hot categories returned status ${categoriesResult.status}`);
      }

      if (feedResult.ok) {
        problems.push(...validateFeedPayload(feedResult.payload));
      }
      if (categoriesResult.ok) {
        problems.push(...validateCategoriesPayload(categoriesResult.payload));
      }
    }

    addCheck(
      checks,
      "frontend-initial-feed-access",
      problems.length === 0 ? "pass" : "fail",
      accessMode === "guest-restricted"
        ? "initial feed endpoints consistently require login for guest access"
        : problems.length === 0
          ? "initial feed endpoints are publicly readable and match the frontend contract"
          : "initial feed endpoints do not match the frontend contract",
      {
        accessMode,
        problems,
        feed: {
          url: feedUrl,
          status: feedResult.status,
          code: isRecord(feedResult.payload) ? feedResult.payload.code ?? null : null,
        },
        categories: {
          url: categoriesUrl,
          status: categoriesResult.status,
          code: isRecord(categoriesResult.payload) ? categoriesResult.payload.code ?? null : null,
        },
      },
    );
  } catch (error) {
    addCheck(checks, "frontend-initial-feed-access", "fail", "initial feed endpoints are not reachable", {
      feedUrl,
      categoriesUrl,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

async function checkSearchAccess(checks, backendOrigin) {
  const searchUrl = apiUrlFromOrigin(
    backendOrigin,
    "/api/search?keyword=smoke&type=all&page=1&limit=1",
  );

  if (!searchUrl) {
    addCheck(checks, "frontend-search-access", "fail", "search URL cannot be built", {
      origin: backendOrigin,
    });
    return;
  }

  try {
    const result = await fetchJsonWithTimeout(searchUrl);
    const unauthorized = isUnauthorizedApiResult(result);
    const problems = [];
    let accessMode = "public";

    if (unauthorized) {
      accessMode = "guest-restricted";
    } else {
      if (!result.ok) {
        problems.push(`search returned status ${result.status}`);
      } else {
        problems.push(...validateSearchPayload(result.payload));
      }
    }

    addCheck(
      checks,
      "frontend-search-access",
      problems.length === 0 ? "pass" : "fail",
      accessMode === "guest-restricted"
        ? "search endpoint requires login for guest access"
        : problems.length === 0
          ? "search endpoint is publicly readable and matches the frontend contract"
          : "search endpoint does not match the frontend contract",
      {
        accessMode,
        problems,
        search: {
          url: searchUrl,
          status: result.status,
          code: isRecord(result.payload) ? result.payload.code ?? null : null,
        },
      },
    );
  } catch (error) {
    addCheck(checks, "frontend-search-access", "fail", "search endpoint is not reachable", {
      searchUrl,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

async function checkGuestProtectedReadSurfaces(checks, backendOrigin) {
  const endpointChecks = [
    { id: "authMe", path: "/api/auth/me", audience: "user" },
    { id: "notificationsUnread", path: "/api/notifications/unread-count", audience: "user" },
    { id: "notificationsList", path: "/api/notifications?page=1&limit=1", audience: "user" },
    { id: "notificationsSystem", path: "/api/notifications/system?page=1&limit=1", audience: "user" },
    { id: "notificationsSystemPopup", path: "/api/notifications/system/popup", audience: "user" },
    { id: "imConversations", path: "/api/im/conversations?page=1&limit=1", audience: "user" },
    { id: "creatorOverview", path: "/api/creator-center/overview", audience: "user" },
    { id: "balanceUserBalance", path: "/api/balance/user-balance", audience: "user" },
    { id: "withdrawWallet", path: "/api/withdraw/wallet", audience: "user" },
    { id: "adminMe", path: "/api/auth/admin/me", audience: "admin" },
    { id: "adminStats", path: "/api/admin/stats/overview", audience: "admin" },
    { id: "adminUsers", path: "/api/admin/users?page=1&limit=1&pageSize=1", audience: "admin" },
    { id: "adminWithdrawOrders", path: "/api/withdraw/admin/orders?page=1&limit=1", audience: "admin" },
  ];
  const endpoints = endpointChecks.map((check) => ({
    ...check,
    url: apiUrlFromOrigin(backendOrigin, check.path),
  }));
  const missingUrls = endpoints.filter((check) => !check.url).map((check) => check.id);

  if (missingUrls.length > 0) {
    addCheck(
      checks,
      "frontend-guest-protected-access",
      "fail",
      "guest protected endpoint URLs cannot be built",
      {
        origin: backendOrigin,
        missingUrls,
      },
    );
    return;
  }

  try {
    const responses = await Promise.all(
      endpoints.map(async (check) => {
        const result = await fetchJsonWithTimeout(check.url);
        return {
          id: check.id,
          path: check.path,
          audience: check.audience,
          status: result.status,
          code: isRecord(result.payload) ? result.payload.code ?? null : null,
          rejected: isGuestAuthRejection(result),
          parseError: result.parseError ?? null,
        };
      }),
    );
    const exposedEndpoints = responses.filter((response) => !response.rejected);
    const problems = exposedEndpoints.map(
      (response) =>
        `${response.id} did not reject guest access (status ${response.status}, code ${
          response.code ?? "n/a"
        })`,
    );

    addCheck(
      checks,
      "frontend-guest-protected-access",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "protected user/admin read endpoints reject guest access"
        : "protected user/admin read endpoints do not consistently reject guest access",
      {
        problems,
        endpoints: responses,
      },
    );
  } catch (error) {
    addCheck(checks, "frontend-guest-protected-access", "fail", "guest protected endpoints are not reachable", {
      origin: backendOrigin,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
      retries: httpRetryCount,
    });
  }
}

async function checkMonetizationConfigContract(checks, backendOrigin) {
  const creatorConfigUrl = apiUrlFromOrigin(backendOrigin, "/api/creator-center/config");
  const balanceConfigUrl = apiUrlFromOrigin(backendOrigin, "/api/balance/config");
  const rechargeConfigUrl = apiUrlFromOrigin(backendOrigin, "/api/balance/recharge-config");

  if (!creatorConfigUrl || !balanceConfigUrl || !rechargeConfigUrl) {
    addCheck(checks, "frontend-monetization-config-contract", "fail", "monetization config URLs cannot be built", {
      origin: backendOrigin,
    });
    return;
  }

  try {
    const [creatorConfigResult, balanceConfigResult, rechargeConfigResult] = await Promise.all([
      fetchJsonWithTimeout(creatorConfigUrl),
      fetchJsonWithTimeout(balanceConfigUrl),
      fetchJsonWithTimeout(rechargeConfigUrl),
    ]);
    const problems = [];

    if (!creatorConfigResult.ok) {
      problems.push(`creator config returned status ${creatorConfigResult.status}`);
    } else {
      problems.push(...validateCreatorConfigPayload(creatorConfigResult.payload));
    }
    if (!balanceConfigResult.ok) {
      problems.push(`balance config returned status ${balanceConfigResult.status}`);
    } else {
      problems.push(...validateBalanceConfigPayload(balanceConfigResult.payload));
    }
    if (!rechargeConfigResult.ok) {
      problems.push(`recharge config returned status ${rechargeConfigResult.status}`);
    } else {
      problems.push(...validateRechargeConfigPayload(rechargeConfigResult.payload));
    }

    addCheck(
      checks,
      "frontend-monetization-config-contract",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "creator and wallet public config endpoints match the frontend contract"
        : "creator and wallet public config endpoints do not match the frontend contract",
      {
        problems,
        creatorConfig: {
          url: creatorConfigUrl,
          status: creatorConfigResult.status,
          code: isRecord(creatorConfigResult.payload) ? creatorConfigResult.payload.code ?? null : null,
        },
        balanceConfig: {
          url: balanceConfigUrl,
          status: balanceConfigResult.status,
          code: isRecord(balanceConfigResult.payload) ? balanceConfigResult.payload.code ?? null : null,
        },
        rechargeConfig: {
          url: rechargeConfigUrl,
          status: rechargeConfigResult.status,
          code: isRecord(rechargeConfigResult.payload) ? rechargeConfigResult.payload.code ?? null : null,
        },
      },
    );
  } catch (error) {
    addCheck(checks, "frontend-monetization-config-contract", "fail", "monetization config endpoints are not reachable", {
      creatorConfigUrl,
      balanceConfigUrl,
      rechargeConfigUrl,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

