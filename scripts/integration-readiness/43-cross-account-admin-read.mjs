async function checkUserCrossAccountReadSmoke(checks, env, backendOrigin) {
  if (
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_A") ||
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_B")
  ) {
    return;
  }

  try {
    const [userATokenResult, userBTokenResult] = await Promise.all([
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A"),
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_B"),
    ]);
    const details = {
      userATokenSource: userATokenResult.source,
      userBTokenSource: userBTokenResult.source,
      userAAccountId: userATokenResult.accountId ?? null,
      userBAccountId: userBTokenResult.accountId ?? null,
    };
    const problems = [];

    if (!userATokenResult.ok) {
      problems.push(
        `ordinary user A cannot obtain an access token: ${
          userATokenResult.message ?? userATokenResult.error ?? userATokenResult.status ?? "unknown error"
        }`,
      );
    }
    if (!userBTokenResult.ok) {
      problems.push(
        `ordinary user B cannot obtain an access token: ${
          userBTokenResult.message ?? userBTokenResult.error ?? userBTokenResult.status ?? "unknown error"
        }`,
      );
    }
    if (problems.length > 0) {
      addCheck(checks, "user-cross-account-read-smoke", "fail", "ordinary user cross-account read smoke failed", {
        ...details,
        problems,
      });
      return;
    }

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(
        checks,
        "user-cross-account-read-smoke",
        "fail",
        "ordinary user cross-account /api/auth/me URL cannot be built",
        details,
      );
      return;
    }

    const [userAMeResult, userBMeResult] = await Promise.all([
      requestJsonWithTimeout(meUrl, {
        method: "GET",
        headers: { authorization: `Bearer ${userATokenResult.token}` },
      }),
      requestJsonWithTimeout(meUrl, {
        method: "GET",
        headers: { authorization: `Bearer ${userBTokenResult.token}` },
      }),
    ]);
    const userAMeData = unwrapApiPayload(userAMeResult.payload);
    const userBMeData = unwrapApiPayload(userBMeResult.payload);
    const userAId = profileUserIdFromAuthMe(userAMeData, userATokenResult.accountId);
    const userBId = profileUserIdFromAuthMe(userBMeData, userBTokenResult.accountId);

    details.userAMeStatus = userAMeResult.status;
    details.userBMeStatus = userBMeResult.status;
    details.userAProfileUserId = userAId || null;
    details.userBProfileUserId = userBId || null;

    if (!userAMeResult.ok) {
      problems.push(`ordinary user A /api/auth/me returned status ${userAMeResult.status}`);
    }
    if (!userBMeResult.ok) {
      problems.push(`ordinary user B /api/auth/me returned status ${userBMeResult.status}`);
    }
    if (!userAId) {
      problems.push("ordinary user A profile id cannot be derived");
    }
    if (!userBId) {
      problems.push("ordinary user B profile id cannot be derived");
    }

    if (userAId && userBId) {
      const [userAReadsUserB, userBReadsUserA] = await Promise.all([
        checkAuthenticatedProfileUserReadSurfacesForUserId(
          backendOrigin,
          userATokenResult.token,
          userBId,
          "ordinary user A reading ordinary user B",
        ),
        checkAuthenticatedProfileUserReadSurfacesForUserId(
          backendOrigin,
          userBTokenResult.token,
          userAId,
          "ordinary user B reading ordinary user A",
        ),
      ]);
      problems.push(...userAReadsUserB.problems, ...userBReadsUserA.problems);
      details.userAReadsUserBStatuses = userAReadsUserB.statuses;
      details.userBReadsUserAStatuses = userBReadsUserA.statuses;
    }

    addCheck(
      checks,
      "user-cross-account-read-smoke",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "ordinary user cross-account read smoke check passed"
        : "ordinary user cross-account read smoke check failed",
      {
        ...details,
        problems,
      },
    );
  } catch (error) {
    addCheck(checks, "user-cross-account-read-smoke", "fail", "ordinary user cross-account read smoke is not reachable", {
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

async function checkAuthenticatedAdminReadSurfaces(backendOrigin, token) {
  const headers = { authorization: `Bearer ${token}` };
  const endpointChecks = [
    {
      id: "adminUsers",
      path: "/api/admin/users?page=1&limit=1&pageSize=1",
      validate: (payload) => validateAdminListPayload(payload, "admin users"),
    },
    {
      id: "adminPosts",
      path: "/api/admin/posts?page=1&limit=1&pageSize=1",
      validate: (payload) => validateAdminListPayload(payload, "admin posts"),
    },
    {
      id: "adminContentReview",
      path: "/api/admin/content-review?page=1&limit=1&pageSize=1",
      validate: (payload) => validateAdminListPayload(payload, "admin content-review"),
    },
    {
      id: "adminReports",
      path: "/api/admin/reports?page=1&limit=1&pageSize=1",
      validate: (payload) => validateAdminListPayload(payload, "admin reports"),
    },
    {
      id: "adminWithdrawOrders",
      path: "/api/withdraw/admin/orders?page=1&limit=1",
      validate: (payload) => validateListWithPaginationPayload(payload, "admin withdraw orders"),
    },
  ];
  const urls = endpointChecks.map((check) => ({
    ...check,
    url: apiUrlFromOrigin(backendOrigin, check.path),
  }));
  const missingUrls = urls.filter((check) => !check.url).map((check) => check.id);
  if (missingUrls.length > 0) {
    return {
      problems: [`administrator read-surface URLs cannot be built: ${missingUrls.join(", ")}`],
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

function booleanFieldFromPayload(payload, field) {
  const data = unwrapApiPayload(payload);
  return isRecord(data) && typeof data[field] === "boolean" ? data[field] : null;
}

async function requestAdminBooleanStatus(backendOrigin, token, config) {
  const url = apiUrlFromOrigin(backendOrigin, config.statusPath);
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      value: null,
      problems: [`${config.label} status URL cannot be built`],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "GET",
    headers: { authorization: `Bearer ${token}` },
  });
  const validationProblems = result.ok
    ? validateBooleanFieldsPayload(result.payload, config.label, config.requiredFields)
    : [];
  const value = result.ok ? booleanFieldFromPayload(result.payload, config.field) : null;

  return {
    ok: result.ok && validationProblems.length === 0 && typeof value === "boolean",
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    value,
    problems: result.ok ? validationProblems : [`${config.label} returned status ${result.status}`],
  };
}

async function requestAdminToggleMutation(backendOrigin, token, config, value) {
  const url = apiUrlFromOrigin(backendOrigin, config.togglePath);
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      problems: [`${config.label} toggle URL cannot be built`],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
    body: JSON.stringify({ [config.bodyField]: value }),
  });

  return {
    ok: result.ok,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    problems: result.ok ? [] : [`${config.label} toggle returned status ${result.status}`],
  };
}

async function runAdminRuntimeToggleSmoke(backendOrigin, token, config) {
  const problems = [];
  const details = {};
  let initialValue = null;
  let mutationAttempted = false;

  const initialStatus = await requestAdminBooleanStatus(backendOrigin, token, config);
  details.initial = {
    status: initialStatus.status,
    code: initialStatus.code,
    value: initialStatus.value,
  };
  problems.push(...initialStatus.problems);
  if (!initialStatus.ok || typeof initialStatus.value !== "boolean") {
    return { problems, details };
  }

  initialValue = initialStatus.value;
  const toggledValue = !initialValue;

  try {
    mutationAttempted = true;
    const toggleMutation = await requestAdminToggleMutation(backendOrigin, token, config, toggledValue);
    details.toggle = {
      requestedValue: toggledValue,
      status: toggleMutation.status,
      code: toggleMutation.code,
    };
    problems.push(...toggleMutation.problems);

    if (toggleMutation.ok) {
      const toggledStatus = await requestAdminBooleanStatus(backendOrigin, token, config);
      details.toggled = {
        status: toggledStatus.status,
        code: toggledStatus.code,
        value: toggledStatus.value,
      };
      problems.push(...toggledStatus.problems);
      if (toggledStatus.value !== toggledValue) {
        problems.push(`${config.label} did not change to ${String(toggledValue)}`);
      }
    }
  } catch (error) {
    problems.push(`${config.label} toggle request failed: ${fetchErrorMessage(error)}`);
  } finally {
    if (mutationAttempted && typeof initialValue === "boolean") {
      details.restoreAttempted = true;
      try {
        const restoreMutation = await requestAdminToggleMutation(backendOrigin, token, config, initialValue);
        details.restore = {
          requestedValue: initialValue,
          status: restoreMutation.status,
          code: restoreMutation.code,
        };
        problems.push(...restoreMutation.problems);

        const restoredStatus = await requestAdminBooleanStatus(backendOrigin, token, config);
        details.restored = {
          status: restoredStatus.status,
          code: restoredStatus.code,
          value: restoredStatus.value,
        };
        problems.push(...restoredStatus.problems);
        if (restoredStatus.value !== initialValue) {
          problems.push(`${config.label} was not restored to ${String(initialValue)}`);
        }
      } catch (error) {
        problems.push(`${config.label} restore request failed: ${fetchErrorMessage(error)}`);
      }
    }
  }

  return { problems, details };
}

async function checkAdminAuthenticatedSmoke(checks, env, backendOrigin) {
  if (!hasAdminIntegrationCredentials(env)) {
    return;
  }

  const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/admin/me");
  const statsUrl = apiUrlFromOrigin(backendOrigin, "/api/admin/stats/overview");
  const aiReviewUrl = apiUrlFromOrigin(backendOrigin, "/api/admin/ai-review-status");
  const guestAccessUrl = apiUrlFromOrigin(backendOrigin, "/api/admin/guest-access-status");
  if (!meUrl || !statsUrl || !aiReviewUrl || !guestAccessUrl) {
    addCheck(checks, "admin-authenticated-smoke", "fail", "administrator smoke URLs cannot be built");
    return;
  }

  try {
    const tokenResult = await getAdminAccessToken(env, backendOrigin);
    if (!tokenResult.ok) {
      addCheck(checks, "admin-authenticated-smoke", "fail", "administrator cannot obtain an access token", {
        tokenSource: tokenResult.source,
        status: tokenResult.status ?? null,
        code: tokenResult.code ?? null,
        message: tokenResult.message ?? tokenResult.error ?? null,
      });
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${tokenResult.token}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    const problems = [];

    if (!meResult.ok) {
      problems.push(`/api/auth/admin/me returned status ${meResult.status}`);
    }
    if (!isRecord(meData) || (meData.id === undefined && meData.username === undefined)) {
      problems.push("/api/auth/admin/me did not return an admin identity");
    }

    const [statsResult, aiReviewResult, guestAccessResult] = await Promise.all([
      requestJsonWithTimeout(statsUrl, {
        method: "GET",
        headers: { authorization: `Bearer ${tokenResult.token}` },
      }),
      requestJsonWithTimeout(aiReviewUrl, {
        method: "GET",
        headers: { authorization: `Bearer ${tokenResult.token}` },
      }),
      requestJsonWithTimeout(guestAccessUrl, {
        method: "GET",
        headers: { authorization: `Bearer ${tokenResult.token}` },
      }),
    ]);

    if (!statsResult.ok) {
      problems.push(`admin stats overview returned status ${statsResult.status}`);
    } else {
      problems.push(
        ...validateNumberFieldsPayload(statsResult.payload, "admin stats overview", [
          "users",
          "posts",
          "comments",
          "reports",
          "feedback",
          "announcements",
        ]),
      );
    }
    if (!aiReviewResult.ok) {
      problems.push(`admin AI review status returned status ${aiReviewResult.status}`);
    } else {
      problems.push(
        ...validateBooleanFieldsPayload(aiReviewResult.payload, "admin AI review status", [
          "enabled",
          "username_enabled",
          "content_enabled",
        ]),
      );
    }
    if (!guestAccessResult.ok) {
      problems.push(`admin guest access status returned status ${guestAccessResult.status}`);
    } else {
      problems.push(
        ...validateBooleanFieldsPayload(guestAccessResult.payload, "admin guest access status", [
          "restricted",
          "note_restricted",
          "video_restricted",
          "admin_restricted",
        ]),
      );
    }

    const adminReadSurfacesResult = await checkAuthenticatedAdminReadSurfaces(
      backendOrigin,
      tokenResult.token,
    );
    problems.push(...adminReadSurfacesResult.problems);

    addCheck(
      checks,
      "admin-authenticated-smoke",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "administrator authenticated smoke check passed"
        : "administrator authenticated smoke check failed",
      {
        tokenSource: tokenResult.source,
        username: tokenResult.username ?? env.INTEGRATION_ADMIN_USERNAME?.trim() ?? null,
        meStatus: meResult.status,
        statsStatus: statsResult.status,
        aiReviewStatus: aiReviewResult.status,
        guestAccessStatus: guestAccessResult.status,
        adminReadSurfaceStatuses: adminReadSurfacesResult.statuses,
        adminIdentityPresent:
          isRecord(meData) && (meData.id !== undefined || meData.username !== undefined),
        problems,
      },
    );
  } catch (error) {
    addCheck(checks, "admin-authenticated-smoke", "fail", "administrator authenticated smoke check is not reachable", {
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
    });
  }
}

async function checkAdminRuntimeToggleWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  if (!hasAdminIntegrationCredentials(env)) {
    addCheck(
      checks,
      "admin-runtime-toggle-write-smoke",
      "fail",
      "administrator runtime toggle write smoke is enabled but administrator credentials are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_ADMIN_ACCESS_TOKEN"],
          ["INTEGRATION_ADMIN_USERNAME", "INTEGRATION_ADMIN_PASSWORD"],
        ],
      },
    );
    return;
  }

  const details = { writeSmokeEnabled: true };
  const problems = [];

  try {
    const tokenResult = await getAdminAccessToken(env, backendOrigin);
    details.tokenSource = tokenResult.source;
    details.username = tokenResult.username ?? env.INTEGRATION_ADMIN_USERNAME?.trim() ?? null;

    if (!tokenResult.ok) {
      addCheck(checks, "admin-runtime-toggle-write-smoke", "fail", "administrator runtime toggle write smoke failed", {
        ...details,
        problems: [
          `administrator cannot obtain an access token: ${
            tokenResult.message ?? tokenResult.error ?? tokenResult.status ?? "unknown error"
          }`,
        ],
      });
      return;
    }

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/admin/me");
    if (!meUrl) {
      addCheck(
        checks,
        "admin-runtime-toggle-write-smoke",
        "fail",
        "administrator runtime toggle write smoke /api/auth/admin/me URL cannot be built",
        details,
      );
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${tokenResult.token}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    details.meStatus = meResult.status;
    details.adminIdentityPresent =
      isRecord(meData) && (meData.id !== undefined || meData.username !== undefined);

    if (!meResult.ok) {
      problems.push(`/api/auth/admin/me returned status ${meResult.status}`);
    }
    if (!details.adminIdentityPresent) {
      problems.push("/api/auth/admin/me did not return an admin identity");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "admin-runtime-toggle-write-smoke",
        "fail",
        "administrator runtime toggle write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const toggleChecks = [
      {
        id: "aiReview",
        label: "admin AI review status",
        statusPath: "/api/admin/ai-review-status",
        togglePath: "/api/admin/ai-review-toggle",
        field: "enabled",
        bodyField: "enabled",
        requiredFields: ["enabled", "username_enabled", "content_enabled"],
      },
      {
        id: "guestAccess",
        label: "admin guest access status",
        statusPath: "/api/admin/guest-access-status",
        togglePath: "/api/admin/guest-access-toggle",
        field: "restricted",
        bodyField: "restricted",
        requiredFields: ["restricted", "note_restricted", "video_restricted", "admin_restricted"],
      },
    ];
    details.toggles = {};

    for (const toggleCheck of toggleChecks) {
      const result = await runAdminRuntimeToggleSmoke(
        backendOrigin,
        tokenResult.token,
        toggleCheck,
      );
      details.toggles[toggleCheck.id] = result.details;
      problems.push(...result.problems);
    }

    addCheck(
      checks,
      "admin-runtime-toggle-write-smoke",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "administrator runtime toggle write smoke check passed"
        : "administrator runtime toggle write smoke check failed",
      { ...details, problems },
    );
  } catch (error) {
    addCheck(
      checks,
      "admin-runtime-toggle-write-smoke",
      "fail",
      "administrator runtime toggle write smoke is not reachable",
      {
        ...details,
        error: fetchErrorMessage(error),
        timeoutMs: httpTimeoutMs,
      },
    );
  }
}

