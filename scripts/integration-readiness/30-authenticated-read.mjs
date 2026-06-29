async function getUserAccessToken(env, backendOrigin, prefix) {
  const directToken = env[`${prefix}_ACCESS_TOKEN`]?.trim();
  if (directToken) {
    return { ok: true, source: "access-token", token: directToken };
  }

  const userId = env[`${prefix}_ID`]?.trim();
  const password = env[`${prefix}_PASSWORD`]?.trim();
  if (!userId || !password) {
    return { ok: false, source: "missing" };
  }

  const loginUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/login");
  if (!loginUrl) {
    return { ok: false, source: "password", error: "login URL cannot be built" };
  }

  const result = await requestJsonWithTimeout(loginUrl, {
    method: "POST",
    body: JSON.stringify({ user_id: userId, password }),
  });
  const accessToken = extractAccessToken(result.payload);

  if (!result.ok || !accessToken) {
    return {
      ok: false,
      source: "password",
      status: result.status,
      code: isRecord(result.payload) ? result.payload.code ?? null : null,
      message: isRecord(result.payload) ? result.payload.message ?? null : null,
    };
  }

  return {
    ok: true,
    source: "password",
    token: accessToken,
    accountId: userId,
  };
}

async function getAdminAccessToken(env, backendOrigin) {
  const directToken = env.INTEGRATION_ADMIN_ACCESS_TOKEN?.trim();
  if (directToken) {
    return { ok: true, source: "access-token", token: directToken };
  }

  const username = env.INTEGRATION_ADMIN_USERNAME?.trim();
  const password = env.INTEGRATION_ADMIN_PASSWORD?.trim();
  if (!username || !password) {
    return { ok: false, source: "missing" };
  }

  const loginUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/admin/login");
  if (!loginUrl) {
    return { ok: false, source: "password", error: "admin login URL cannot be built" };
  }

  const result = await requestJsonWithTimeout(loginUrl, {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  const accessToken = extractAccessToken(result.payload);

  if (!result.ok || !accessToken) {
    return {
      ok: false,
      source: "password",
      status: result.status,
      code: isRecord(result.payload) ? result.payload.code ?? null : null,
      message: isRecord(result.payload) ? result.payload.message ?? null : null,
    };
  }

  return {
    ok: true,
    source: "password",
    token: accessToken,
    username,
  };
}

async function checkAuthenticatedCreatorWalletReadSurfaces(backendOrigin, token) {
  const headers = { authorization: `Bearer ${token}` };
  const endpointChecks = [
    {
      id: "creatorOverview",
      path: "/api/creator-center/overview",
      validate: (payload) =>
        validateNumberFieldsPayload(payload, "creator overview", [
          "balance",
          "total_earnings",
          "withdrawn_amount",
          "today_earnings",
          "month_earnings",
        ]),
    },
    {
      id: "creatorStats",
      path: "/api/creator-center/stats?days=30",
      validate: validateCreatorStatsPayload,
    },
    {
      id: "creatorTrends",
      path: "/api/creator-center/trends?days=14",
      validate: validateCreatorTrendsPayload,
    },
    {
      id: "creatorEarningsLog",
      path: "/api/creator-center/earnings-log?page=1&limit=1",
      validate: (payload) => validateListWithPaginationPayload(payload, "creator earnings-log"),
    },
    {
      id: "creatorPaidContent",
      path: "/api/creator-center/paid-content?page=1&limit=1",
      validate: (payload) => validateListWithPaginationPayload(payload, "creator paid-content"),
    },
    {
      id: "creatorQualityRewards",
      path: "/api/creator-center/quality-rewards?page=1&limit=1",
      validate: validateCreatorQualityRewardsPayload,
    },
    {
      id: "balanceLocalPoints",
      path: "/api/balance/local-points",
      validate: (payload) => validateNumberFieldsPayload(payload, "balance local-points", ["points"]),
    },
    {
      id: "balanceUserBalance",
      path: "/api/balance/user-balance",
      validate: (payload) => validateNumberFieldsPayload(payload, "balance user-balance", ["balance"]),
    },
    {
      id: "balanceOrders",
      path: "/api/balance/orders?page=1&limit=1",
      validate: (payload) => validateListWithPaginationPayload(payload, "balance orders"),
    },
    {
      id: "withdrawWallet",
      path: "/api/withdraw/wallet",
      validate: (payload) =>
        validateNumberFieldsPayload(payload, "withdraw wallet", [
          "moon_coin",
          "cash_balance",
          "total_income",
          "frozen_amount",
        ]),
    },
    {
      id: "withdrawPaymentCode",
      path: "/api/withdraw/payment-code",
      validate: (payload) =>
        validateStringOrNullFieldsPayload(payload, "withdraw payment-code", [
          "wechat_url",
          "alipay_url",
        ]),
    },
    {
      id: "withdrawOrders",
      path: "/api/withdraw/orders?page=1&limit=1",
      validate: (payload) => validateListWithPaginationPayload(payload, "withdraw orders"),
    },
  ];

  const urls = endpointChecks.map((check) => ({
    ...check,
    url: apiUrlFromOrigin(backendOrigin, check.path),
  }));
  const missingUrls = urls.filter((check) => !check.url).map((check) => check.id);
  if (missingUrls.length > 0) {
    return {
      problems: [`authenticated creator/wallet URLs cannot be built: ${missingUrls.join(", ")}`],
      statuses: {},
    };
  }

  const responses = await Promise.all(
    urls.map(async (check) => {
      const result = await requestJsonWithTimeout(check.url, {
        method: "GET",
        headers,
      });
      const problems = result.ok
        ? check.validate(result.payload)
        : [`${check.id} returned status ${result.status}`];
      return {
        id: check.id,
        status: result.status,
        code: isRecord(result.payload) ? result.payload.code ?? null : null,
        problems,
      };
    }),
  );

  return {
    problems: responses.flatMap((response) => response.problems),
    statuses: Object.fromEntries(
      responses.map((response) => [
        response.id,
        {
          status: response.status,
          code: response.code,
        },
      ]),
    ),
  };
}

function profileUserIdFromAuthMe(meData, fallbackAccountId = null) {
  if (!isRecord(meData)) {
    return fallbackAccountId ? String(fallbackAccountId) : "";
  }

  for (const field of ["user_id", "xise_id", "id"]) {
    const value = meData[field];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
    if (typeof value === "number" && Number.isFinite(value)) {
      return String(value);
    }
  }

  return fallbackAccountId ? String(fallbackAccountId) : "";
}

async function checkAuthenticatedProfileUserReadSurfacesForUserId(
  backendOrigin,
  token,
  userId,
  label = "profile/user",
) {
  if (!userId) {
    return {
      userId: null,
      problems: [`${label} user id cannot be derived`],
      statuses: {},
    };
  }

  const encodedUserId = encodeURIComponent(userId);
  const headers = { authorization: `Bearer ${token}` };
  const endpointChecks = [
    {
      id: "userProfile",
      path: `/api/users/${encodedUserId}`,
      validate: (payload) => validateUserProfilePayload(payload, `${label} profile`),
    },
    {
      id: "userPosts",
      path: `/api/users/${encodedUserId}/posts?page=1&limit=1`,
      validate: (payload) => validateUserPostListPayload(payload, "posts", `${label} posts`),
    },
    {
      id: "userCollections",
      path: `/api/users/${encodedUserId}/collections?page=1&limit=1`,
      validate: (payload) =>
        validateUserPostListPayload(payload, "collections", `${label} collections`),
    },
    {
      id: "userLikes",
      path: `/api/users/${encodedUserId}/likes?page=1&limit=1`,
      validate: (payload) => validateUserPostListPayload(payload, "posts", `${label} likes`),
    },
    {
      id: "userFollowStatus",
      path: `/api/users/${encodedUserId}/follow-status`,
      validate: validateFollowStatusPayload,
    },
  ];

  const urls = endpointChecks.map((check) => ({
    ...check,
    url: apiUrlFromOrigin(backendOrigin, check.path),
  }));
  const missingUrls = urls.filter((check) => !check.url).map((check) => check.id);
  if (missingUrls.length > 0) {
    return {
      userId,
      problems: [`${label} URLs cannot be built: ${missingUrls.join(", ")}`],
      statuses: {},
    };
  }

  const responses = await Promise.all(
    urls.map(async (check) => {
      const result = await requestJsonWithTimeout(check.url, {
        method: "GET",
        headers,
      });
      const problems = result.ok
        ? check.validate(result.payload)
        : [`${label} ${check.id} returned status ${result.status}`];
      return {
        id: check.id,
        status: result.status,
        code: isRecord(result.payload) ? result.payload.code ?? null : null,
        problems,
      };
    }),
  );

  return {
    userId,
    problems: responses.flatMap((response) => response.problems),
    statuses: Object.fromEntries(
      responses.map((response) => [
        response.id,
        {
          status: response.status,
          code: response.code,
        },
      ]),
    ),
  };
}

async function checkAuthenticatedProfileUserReadSurfaces(
  backendOrigin,
  token,
  meData,
  fallbackAccountId = null,
) {
  const userId = profileUserIdFromAuthMe(meData, fallbackAccountId);
  if (!userId) {
    return {
      userId: null,
      problems: ["authenticated profile user id cannot be derived"],
      statuses: {},
    };
  }

  return checkAuthenticatedProfileUserReadSurfacesForUserId(
    backendOrigin,
    token,
    userId,
    "authenticated profile/user",
  );
}

async function checkAuthenticatedContentDetailReadSurfaces(backendOrigin, token, feedPayload) {
  const postId = postIdFromFeedPayload(feedPayload);
  if (!postId) {
    return {
      postId: null,
      skipped: true,
      skipReason: "authenticated feed did not return a post id",
      problems: [],
      statuses: {},
    };
  }

  const encodedPostId = encodeURIComponent(postId);
  const headers = { authorization: `Bearer ${token}` };
  const endpointChecks = [
    {
      id: "postDetail",
      path: `/api/posts/${encodedPostId}`,
      validate: validatePostDetailPayload,
    },
    {
      id: "postComments",
      path: `/api/posts/${encodedPostId}/comments?page=1&limit=1`,
      validate: validateCommentsPayload,
    },
    {
      id: "dislikeStatus",
      path: `/api/dislikes?post_id=${encodedPostId}`,
      validate: (payload) => validateBooleanFieldsPayload(payload, "dislike status", ["disliked"]),
    },
    {
      id: "reportStatus",
      path: `/api/reports/check?target_type=post&target_id=${encodedPostId}`,
      validate: (payload) => validateBooleanFieldsPayload(payload, "report status", ["reported"]),
    },
  ];

  const urls = endpointChecks.map((check) => ({
    ...check,
    url: apiUrlFromOrigin(backendOrigin, check.path),
  }));
  const missingUrls = urls.filter((check) => !check.url).map((check) => check.id);
  if (missingUrls.length > 0) {
    return {
      postId,
      skipped: false,
      problems: [`authenticated content detail URLs cannot be built: ${missingUrls.join(", ")}`],
      statuses: {},
    };
  }

  const responses = await Promise.all(
    urls.map(async (check) => {
      const result = await requestJsonWithTimeout(check.url, {
        method: "GET",
        headers,
      });
      const problems = result.ok
        ? check.validate(result.payload)
        : [`${check.id} returned status ${result.status}`];
      return {
        id: check.id,
        status: result.status,
        code: isRecord(result.payload) ? result.payload.code ?? null : null,
        problems,
      };
    }),
  );

  return {
    postId,
    skipped: false,
    problems: responses.flatMap((response) => response.problems),
    statuses: Object.fromEntries(
      responses.map((response) => [
        response.id,
        {
          status: response.status,
          code: response.code,
        },
      ]),
    ),
  };
}

async function checkAuthenticatedImReadSurfaces(backendOrigin, token, conversationsPayload) {
  const headers = { authorization: `Bearer ${token}` };
  const conversationId = firstConversationIdFromPayload(conversationsPayload);
  const syncUrl = apiUrlFromOrigin(backendOrigin, "/api/im/sync?since=0&limit=1");
  const problems = [];
  const statuses = {};

  if (!syncUrl) {
    return {
      conversationId: null,
      skippedMessages: !conversationId,
      problems: ["IM sync URL cannot be built"],
      statuses,
    };
  }

  const syncResult = await requestJsonWithTimeout(syncUrl, {
    method: "GET",
    headers,
  });
  statuses.imSync = {
    status: syncResult.status,
    code: isRecord(syncResult.payload) ? syncResult.payload.code ?? null : null,
  };
  if (!syncResult.ok) {
    problems.push(`IM sync returned status ${syncResult.status}`);
  } else {
    problems.push(...validateImSyncPayload(syncResult.payload));
  }

  if (!conversationId) {
    return {
      conversationId: null,
      skippedMessages: true,
      skipReason: "IM conversations did not return a conversation id",
      problems,
      statuses,
    };
  }

  const messagesUrl = apiUrlFromOrigin(
    backendOrigin,
    `/api/im/conversations/${encodeURIComponent(conversationId)}/messages?limit=1`,
  );
  if (!messagesUrl) {
    return {
      conversationId,
      skippedMessages: false,
      problems: [...problems, "IM messages URL cannot be built"],
      statuses,
    };
  }

  const messagesResult = await requestJsonWithTimeout(messagesUrl, {
    method: "GET",
    headers,
  });
  statuses.imMessages = {
    status: messagesResult.status,
    code: isRecord(messagesResult.payload) ? messagesResult.payload.code ?? null : null,
  };
  if (!messagesResult.ok) {
    problems.push(`IM messages returned status ${messagesResult.status}`);
  } else {
    problems.push(...validateImMessagesPayload(messagesResult.payload));
  }

  return {
    conversationId,
    skippedMessages: false,
    problems,
    statuses,
  };
}

async function checkUserAuthenticatedSmoke(checks, env, backendOrigin, prefix, id, label, includeFeed) {
  if (
    !env[`${prefix}_ACCESS_TOKEN`]?.trim() &&
    (!env[`${prefix}_ID`]?.trim() || !env[`${prefix}_PASSWORD`]?.trim())
  ) {
    return;
  }

  try {
    const tokenResult = await getUserAccessToken(env, backendOrigin, prefix);
    if (!tokenResult.ok) {
      addCheck(checks, id, "fail", `${label} cannot obtain an access token`, {
        source: tokenResult.source,
        status: tokenResult.status ?? null,
        code: tokenResult.code ?? null,
        message: tokenResult.message ?? tokenResult.error ?? null,
      });
      return;
    }

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(checks, id, "fail", `${label} /api/auth/me URL cannot be built`);
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${tokenResult.token}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    const problems = [];

    if (!meResult.ok) {
      problems.push(`/api/auth/me returned status ${meResult.status}`);
    }
    if (!isRecord(meData) || (meData.id === undefined && meData.user_id === undefined)) {
      problems.push("/api/auth/me did not return a user identity");
    }

    const details = {
      tokenSource: tokenResult.source,
      accountId: tokenResult.accountId ?? null,
      meStatus: meResult.status,
      userIdentityPresent:
        isRecord(meData) && (meData.id !== undefined || meData.user_id !== undefined),
    };

    if (isRecord(meData) && (meData.id !== undefined || meData.user_id !== undefined)) {
      const profileUserResult = await checkAuthenticatedProfileUserReadSurfaces(
        backendOrigin,
        tokenResult.token,
        meData,
        tokenResult.accountId,
      );
      problems.push(...profileUserResult.problems);
      details.profileUserId = profileUserResult.userId;
      details.profileUserStatuses = profileUserResult.statuses;
    }

    if (includeFeed) {
      const feedUrl = apiUrlFromOrigin(backendOrigin, "/api/posts/recommended?page=1&limit=1");
      const categoriesUrl = apiUrlFromOrigin(backendOrigin, "/api/categories/hot?limit=12");
      const searchUrl = apiUrlFromOrigin(
        backendOrigin,
        "/api/search?keyword=smoke&type=all&page=1&limit=1",
      );
      if (!feedUrl || !categoriesUrl || !searchUrl) {
        problems.push("authenticated feed/search URLs cannot be built");
      } else {
        const [feedResult, categoriesResult, searchResult] = await Promise.all([
          requestJsonWithTimeout(feedUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(categoriesUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(searchUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
        ]);

        if (!feedResult.ok) {
          problems.push(`authenticated recommended feed returned status ${feedResult.status}`);
        } else {
          problems.push(...validateFeedPayload(feedResult.payload));
        }
        if (!categoriesResult.ok) {
          problems.push(`authenticated hot categories returned status ${categoriesResult.status}`);
        } else {
          problems.push(...validateCategoriesPayload(categoriesResult.payload));
        }
        if (!searchResult.ok) {
          problems.push(`authenticated search returned status ${searchResult.status}`);
        } else {
          problems.push(...validateSearchPayload(searchResult.payload));
        }

        details.feedStatus = feedResult.status;
        details.categoriesStatus = categoriesResult.status;
        details.searchStatus = searchResult.status;

        if (feedResult.ok) {
          const contentDetailResult = await checkAuthenticatedContentDetailReadSurfaces(
            backendOrigin,
            tokenResult.token,
            feedResult.payload,
          );
          problems.push(...contentDetailResult.problems);
          details.contentDetailPostId = contentDetailResult.postId;
          details.contentDetailSkipped = contentDetailResult.skipped;
          if (contentDetailResult.skipReason) {
            details.contentDetailSkipReason = contentDetailResult.skipReason;
          }
          details.contentDetailStatuses = contentDetailResult.statuses;
        }
      }

      const notificationUnreadUrl = apiUrlFromOrigin(backendOrigin, "/api/notifications/unread-count");
      const notificationsUrl = apiUrlFromOrigin(backendOrigin, "/api/notifications?page=1&limit=1");
      const systemNotificationsUrl = apiUrlFromOrigin(
        backendOrigin,
        "/api/notifications/system?page=1&limit=1",
      );
      const systemPopupNotificationsUrl = apiUrlFromOrigin(
        backendOrigin,
        "/api/notifications/system/popup",
      );
      const imConversationsUrl = apiUrlFromOrigin(backendOrigin, "/api/im/conversations");
      if (
        !notificationUnreadUrl ||
        !notificationsUrl ||
        !systemNotificationsUrl ||
        !systemPopupNotificationsUrl ||
        !imConversationsUrl
      ) {
        problems.push("authenticated read-surface URLs cannot be built");
      } else {
        const [
          notificationUnreadResult,
          notificationsResult,
          systemNotificationsResult,
          systemPopupNotificationsResult,
          imConversationsResult,
        ] = await Promise.all([
          requestJsonWithTimeout(notificationUnreadUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(notificationsUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(systemNotificationsUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(systemPopupNotificationsUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
          requestJsonWithTimeout(imConversationsUrl, {
            method: "GET",
            headers: { authorization: `Bearer ${tokenResult.token}` },
          }),
        ]);

        if (!notificationUnreadResult.ok) {
          problems.push(`notification unread-count returned status ${notificationUnreadResult.status}`);
        } else {
          problems.push(...validateUnreadCountPayload(notificationUnreadResult.payload));
        }
        if (!notificationsResult.ok) {
          problems.push(`notification list returned status ${notificationsResult.status}`);
        } else {
          problems.push(...validateNotificationListPayload(notificationsResult.payload, "notification list"));
        }
        if (!systemNotificationsResult.ok) {
          problems.push(`system notification list returned status ${systemNotificationsResult.status}`);
        } else {
          problems.push(
            ...validateNotificationListPayload(systemNotificationsResult.payload, "system notification list", {
              system: true,
            }),
          );
        }
        if (!systemPopupNotificationsResult.ok) {
          problems.push(`system notification popup returned status ${systemPopupNotificationsResult.status}`);
        } else {
          problems.push(...validateSystemNotificationPopupPayload(systemPopupNotificationsResult.payload));
        }
        if (!imConversationsResult.ok) {
          problems.push(`IM conversations returned status ${imConversationsResult.status}`);
        } else {
          problems.push(...validateArrayPayload(imConversationsResult.payload, "IM conversations"));
          const imReadResult = await checkAuthenticatedImReadSurfaces(
            backendOrigin,
            tokenResult.token,
            imConversationsResult.payload,
          );
          problems.push(...imReadResult.problems);
          details.imReadConversationId = imReadResult.conversationId;
          details.imReadMessagesSkipped = imReadResult.skippedMessages;
          if (imReadResult.skipReason) {
            details.imReadSkipReason = imReadResult.skipReason;
          }
          details.imReadStatuses = imReadResult.statuses;
        }

        details.notificationUnreadStatus = notificationUnreadResult.status;
        details.notificationListStatus = notificationsResult.status;
        details.systemNotificationListStatus = systemNotificationsResult.status;
        details.systemNotificationPopupStatus = systemPopupNotificationsResult.status;
        details.imConversationsStatus = imConversationsResult.status;
      }

      const creatorWalletResult = await checkAuthenticatedCreatorWalletReadSurfaces(
        backendOrigin,
        tokenResult.token,
      );
      problems.push(...creatorWalletResult.problems);
      details.creatorWalletStatuses = creatorWalletResult.statuses;
    }

    addCheck(
      checks,
      id,
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? `${label} authenticated smoke check passed`
        : `${label} authenticated smoke check failed`,
      {
        ...details,
        problems,
      },
    );
  } catch (error) {
    addCheck(checks, id, "fail", `${label} authenticated smoke check is not reachable`, {
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

