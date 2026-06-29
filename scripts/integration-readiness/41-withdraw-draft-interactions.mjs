async function checkUserWithdrawPaymentCodeWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  if (!hasUserIntegrationCredentials(env, "INTEGRATION_USER_A")) {
    addCheck(
      checks,
      "user-withdraw-payment-code-write-smoke",
      "fail",
      "ordinary user withdraw payment-code write smoke is enabled but user A credentials are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_USER_A_ACCESS_TOKEN"],
          ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
        ],
      },
    );
    return;
  }

  const details = { writeSmokeEnabled: true };
  const problems = [];
  let userAToken = null;
  let initialState = null;
  let saveAttempted = false;

  try {
    const tokenResult = await getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A");
    details.userATokenSource = tokenResult.source;

    if (!tokenResult.ok) {
      addCheck(
        checks,
        "user-withdraw-payment-code-write-smoke",
        "fail",
        "ordinary user withdraw payment-code write smoke failed",
        {
          ...details,
          problems: [
            `ordinary user A cannot obtain an access token: ${
              tokenResult.message ?? tokenResult.error ?? tokenResult.status ?? "unknown error"
            }`,
          ],
        },
      );
      return;
    }
    userAToken = tokenResult.token;

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(
        checks,
        "user-withdraw-payment-code-write-smoke",
        "fail",
        "ordinary user withdraw payment-code write smoke /api/auth/me URL cannot be built",
        details,
      );
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${userAToken}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    details.userAMeStatus = meResult.status;
    details.userAIdentityPresent = isRecord(meData) && (meData.id !== undefined || meData.user_id !== undefined);
    if (!meResult.ok) {
      problems.push(`ordinary user A /api/auth/me returned status ${meResult.status}`);
    }
    if (!details.userAIdentityPresent) {
      problems.push("ordinary user A /api/auth/me did not return a user identity");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-withdraw-payment-code-write-smoke",
        "fail",
        "ordinary user withdraw payment-code write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const initialResult = await requestWithdrawPaymentCode(backendOrigin, userAToken);
    initialState = initialResult.state;
    details.initialPaymentCode = {
      status: initialResult.status,
      code: initialResult.code,
      hasWechatUrl: Boolean(initialState.wechatUrl),
      hasAlipayUrl: Boolean(initialState.alipayUrl),
    };
    problems.push(...initialResult.problems);
    if (!initialResult.ok) {
      addCheck(
        checks,
        "user-withdraw-payment-code-write-smoke",
        "fail",
        "ordinary user withdraw payment-code write smoke cannot read initial payment-code state",
        { ...details, problems },
      );
      return;
    }
    if (!hasPaymentCodeValue(initialState)) {
      addCheck(
        checks,
        "user-withdraw-payment-code-write-smoke",
        "fail",
        "ordinary user withdraw payment-code write smoke requires a pre-existing payment code to restore",
        {
          ...details,
          problems: [
            "initial payment-code has no wechat_url or alipay_url; the backend save endpoint requires at least one non-empty URL, so the smoke will not create an irreversible first payment-code record",
          ],
        },
      );
      return;
    }

    const smokeSuffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const smokeState = {
      wechatUrl: `https://example.invalid/codex-smoke/wechat-${smokeSuffix}.png`,
      alipayUrl: `https://example.invalid/codex-smoke/alipay-${smokeSuffix}.png`,
    };
    const saveResult = await requestSaveWithdrawPaymentCode(backendOrigin, userAToken, smokeState);
    saveAttempted = saveResult.ok;
    details.savePaymentCode = {
      status: saveResult.status,
      code: saveResult.code,
      hasWechatUrl: Boolean(saveResult.state.wechatUrl),
      hasAlipayUrl: Boolean(saveResult.state.alipayUrl),
    };
    problems.push(...saveResult.problems);
    if (saveResult.ok) {
      if (saveResult.state.wechatUrl !== smokeState.wechatUrl) {
        problems.push("withdraw payment-code save did not persist smoke wechat_url");
      }
      if (saveResult.state.alipayUrl !== smokeState.alipayUrl) {
        problems.push("withdraw payment-code save did not persist smoke alipay_url");
      }
    }
  } catch (error) {
    problems.push(`withdraw payment-code write smoke failed: ${fetchErrorMessage(error)}`);
  } finally {
    if (userAToken && initialState && saveAttempted) {
      try {
        const restoreResult = await requestSaveWithdrawPaymentCode(backendOrigin, userAToken, initialState);
        details.restorePaymentCode = {
          status: restoreResult.status,
          code: restoreResult.code,
          hasWechatUrl: Boolean(restoreResult.state.wechatUrl),
          hasAlipayUrl: Boolean(restoreResult.state.alipayUrl),
        };
        problems.push(...restoreResult.problems);
        if (restoreResult.state.wechatUrl !== initialState.wechatUrl) {
          problems.push("withdraw payment-code restore did not return wechat_url to its initial value");
        }
        if (restoreResult.state.alipayUrl !== initialState.alipayUrl) {
          problems.push("withdraw payment-code restore did not return alipay_url to its initial value");
        }
      } catch (error) {
        problems.push(`withdraw payment-code restore failed: ${fetchErrorMessage(error)}`);
      }
    }
  }

  addCheck(
    checks,
    "user-withdraw-payment-code-write-smoke",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "ordinary user withdraw payment-code write smoke check passed"
      : "ordinary user withdraw payment-code write smoke check failed",
    { ...details, problems },
  );
}

async function checkUserDraftPostWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  if (!hasUserIntegrationCredentials(env, "INTEGRATION_USER_A")) {
    addCheck(
      checks,
      "user-draft-post-write-smoke",
      "fail",
      "ordinary user draft post write smoke is enabled but user A credentials are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_USER_A_ACCESS_TOKEN"],
          ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
        ],
      },
    );
    return;
  }

  const details = { writeSmokeEnabled: true };
  const problems = [];
  let userAToken = null;
  let draftPostId = null;

  try {
    const tokenResult = await getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A");
    details.userATokenSource = tokenResult.source;

    if (!tokenResult.ok) {
      addCheck(checks, "user-draft-post-write-smoke", "fail", "ordinary user draft post write smoke failed", {
        ...details,
        problems: [
          `ordinary user A cannot obtain an access token: ${
            tokenResult.message ?? tokenResult.error ?? tokenResult.status ?? "unknown error"
          }`,
        ],
      });
      return;
    }
    userAToken = tokenResult.token;

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(
        checks,
        "user-draft-post-write-smoke",
        "fail",
        "ordinary user draft post write smoke /api/auth/me URL cannot be built",
        details,
      );
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${userAToken}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    details.userAMeStatus = meResult.status;
    details.userAIdentityPresent = isRecord(meData) && (meData.id !== undefined || meData.user_id !== undefined);
    if (!meResult.ok) {
      problems.push(`ordinary user A /api/auth/me returned status ${meResult.status}`);
    }
    if (!details.userAIdentityPresent) {
      problems.push("ordinary user A /api/auth/me did not return a user identity");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-draft-post-write-smoke",
        "fail",
        "ordinary user draft post write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const createResult = await requestCreateDraftPostSmoke(backendOrigin, userAToken);
    draftPostId = createResult.postId;
    details.createDraftPost = {
      status: createResult.status,
      code: createResult.code,
      postId: draftPostId ?? null,
    };
    problems.push(...createResult.problems);
  } catch (error) {
    problems.push(`draft post write smoke failed: ${fetchErrorMessage(error)}`);
  } finally {
    if (userAToken && draftPostId !== null && draftPostId !== undefined) {
      try {
        const deleteResult = await requestDeleteDraftPostSmoke(backendOrigin, userAToken, draftPostId);
        details.deleteDraftPost = {
          status: deleteResult.status,
          code: deleteResult.code,
        };
        problems.push(...deleteResult.problems);
      } catch (error) {
        problems.push(`draft post delete restore failed: ${fetchErrorMessage(error)}`);
      }
    }
  }

  addCheck(
    checks,
    "user-draft-post-write-smoke",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "ordinary user draft post write smoke check passed"
      : "ordinary user draft post write smoke check failed",
    { ...details, problems },
  );
}

async function checkUserPostInteractionWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  const postId = writeSmokePostId(env);
  if (!hasUserIntegrationCredentials(env, "INTEGRATION_USER_A") || !postId) {
    addCheck(
      checks,
      "user-post-interaction-write-smoke",
      "fail",
      "ordinary user post interaction write smoke is enabled but user A credentials or a write-smoke post id are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_USER_A_ACCESS_TOKEN"],
          ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
        ],
        requiredVariables: ["INTEGRATION_WRITE_SMOKE_POST_ID"],
      },
    );
    return;
  }

  const details = { writeSmokeEnabled: true, postId };
  const problems = [];
  let commentId = null;
  let likeMayNeedRestore = false;
  let collectMayNeedRestore = false;
  let initialState = null;
  let userAToken = null;

  try {
    const tokenResult = await getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A");
    details.userATokenSource = tokenResult.source;

    if (!tokenResult.ok) {
      addCheck(checks, "user-post-interaction-write-smoke", "fail", "ordinary user post interaction write smoke failed", {
        ...details,
        problems: [
          `ordinary user A cannot obtain an access token: ${
            tokenResult.message ?? tokenResult.error ?? tokenResult.status ?? "unknown error"
          }`,
        ],
      });
      return;
    }
    userAToken = tokenResult.token;

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(
        checks,
        "user-post-interaction-write-smoke",
        "fail",
        "ordinary user post interaction write smoke /api/auth/me URL cannot be built",
        details,
      );
      return;
    }

    const meResult = await requestJsonWithTimeout(meUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${userAToken}` },
    });
    const meData = unwrapApiPayload(meResult.payload);
    details.userAMeStatus = meResult.status;
    details.userAIdentityPresent = isRecord(meData) && (meData.id !== undefined || meData.user_id !== undefined);
    if (!meResult.ok) {
      problems.push(`ordinary user A /api/auth/me returned status ${meResult.status}`);
    }
    if (!details.userAIdentityPresent) {
      problems.push("ordinary user A /api/auth/me did not return a user identity");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-post-interaction-write-smoke",
        "fail",
        "ordinary user post interaction write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const initialDetail = await requestPostInteractionDetail(backendOrigin, userAToken, postId);
    details.initialPostDetail = {
      status: initialDetail.status,
      code: initialDetail.code,
      liked: initialDetail.state.liked,
      collected: initialDetail.state.collected,
    };
    problems.push(...initialDetail.problems);
    if (!initialDetail.ok) {
      addCheck(
        checks,
        "user-post-interaction-write-smoke",
        "fail",
        "ordinary user post interaction write smoke cannot read initial post interaction state",
        { ...details, problems },
      );
      return;
    }
    initialState = initialDetail.state;

    const likeToggle = await requestPostLikeToggle(backendOrigin, userAToken, postId);
    likeMayNeedRestore = likeToggle.ok;
    details.likeToggle = {
      status: likeToggle.status,
      code: likeToggle.code,
      liked: likeToggle.liked,
    };
    problems.push(...likeToggle.problems);
    if (likeToggle.ok && likeToggle.liked !== !initialState.liked) {
      problems.push(`like toggle did not change liked to ${String(!initialState.liked)}`);
    }

    const collectToggle = await requestPostCollectToggle(backendOrigin, userAToken, postId);
    collectMayNeedRestore = collectToggle.ok;
    details.collectToggle = {
      status: collectToggle.status,
      code: collectToggle.code,
      collected: collectToggle.collected,
    };
    problems.push(...collectToggle.problems);
    if (collectToggle.ok && collectToggle.collected !== !initialState.collected) {
      problems.push(`collect toggle did not change collected to ${String(!initialState.collected)}`);
    }

    const commentContent = `codex post interaction write smoke ${Date.now()}-${Math.random()
      .toString(36)
      .slice(2, 8)}`;
    const commentCreate = await requestCreateSmokeComment(
      backendOrigin,
      userAToken,
      postId,
      commentContent,
    );
    commentId = commentCreate.commentId;
    details.commentCreate = {
      status: commentCreate.status,
      code: commentCreate.code,
      commentId: commentId ?? null,
    };
    problems.push(...commentCreate.problems);
  } catch (error) {
    problems.push(`post interaction write smoke failed: ${fetchErrorMessage(error)}`);
  } finally {
    if (userAToken && commentId !== null && commentId !== undefined) {
      try {
        const commentDelete = await requestDeleteSmokeComment(backendOrigin, userAToken, commentId);
        details.commentDelete = {
          status: commentDelete.status,
          code: commentDelete.code,
        };
        problems.push(...commentDelete.problems);
      } catch (error) {
        problems.push(`comment delete restore failed: ${fetchErrorMessage(error)}`);
      }
    }

    if (userAToken && initialState && likeMayNeedRestore) {
      try {
        const currentDetail = await requestPostInteractionDetail(backendOrigin, userAToken, postId);
        details.likeRestorePrecheck = {
          status: currentDetail.status,
          code: currentDetail.code,
          liked: currentDetail.state.liked,
        };
        if (currentDetail.state.liked !== initialState.liked) {
          const likeRestore = await requestPostLikeToggle(backendOrigin, userAToken, postId);
          details.likeRestore = {
            status: likeRestore.status,
            code: likeRestore.code,
            liked: likeRestore.liked,
          };
          problems.push(...likeRestore.problems);
          if (likeRestore.liked !== initialState.liked) {
            problems.push(`like restore did not return liked to ${String(initialState.liked)}`);
          }
        }
      } catch (error) {
        problems.push(`like restore failed: ${fetchErrorMessage(error)}`);
      }
    }

    if (userAToken && initialState && collectMayNeedRestore) {
      try {
        const currentDetail = await requestPostInteractionDetail(backendOrigin, userAToken, postId);
        details.collectRestorePrecheck = {
          status: currentDetail.status,
          code: currentDetail.code,
          collected: currentDetail.state.collected,
        };
        if (currentDetail.state.collected !== initialState.collected) {
          const collectRestore = await requestPostCollectToggle(backendOrigin, userAToken, postId);
          details.collectRestore = {
            status: collectRestore.status,
            code: collectRestore.code,
            collected: collectRestore.collected,
          };
          problems.push(...collectRestore.problems);
          if (collectRestore.collected !== initialState.collected) {
            problems.push(`collect restore did not return collected to ${String(initialState.collected)}`);
          }
        }
      } catch (error) {
        problems.push(`collect restore failed: ${fetchErrorMessage(error)}`);
      }
    }
  }

  addCheck(
    checks,
    "user-post-interaction-write-smoke",
    problems.length === 0 ? "pass" : "fail",
    problems.length === 0
      ? "ordinary user post interaction write smoke check passed"
      : "ordinary user post interaction write smoke check failed",
    { ...details, problems },
  );
}

