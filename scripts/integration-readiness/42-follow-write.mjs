async function requestFollowStatus(backendOrigin, token, targetUserId) {
  const url = apiUrlFromOrigin(
    backendOrigin,
    `/api/users/${encodeURIComponent(targetUserId)}/follow-status`,
  );
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      following: null,
      problems: ["follow-status URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "GET",
    headers: { authorization: `Bearer ${token}` },
  });
  const data = unwrapApiPayload(result.payload);
  const validationProblems = result.ok ? validateFollowStatusPayload(result.payload) : [];
  const following = isRecord(data) ? Boolean(data.isFollowing ?? data.followed) : null;

  return {
    ok: result.ok && validationProblems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    following,
    problems: result.ok ? validationProblems : [`follow-status returned status ${result.status}`],
  };
}

async function requestFollowMutation(backendOrigin, token, targetUserId, method) {
  const url = apiUrlFromOrigin(backendOrigin, `/api/users/${encodeURIComponent(targetUserId)}/follow`);
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      problems: [`${method} follow URL cannot be built`],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method,
    headers: { authorization: `Bearer ${token}` },
  });
  return {
    ok: result.ok,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    problems: result.ok ? [] : [`${method} follow returned status ${result.status}`],
  };
}

async function checkUserCrossAccountFollowWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  if (
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_A") ||
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_B")
  ) {
    addCheck(
      checks,
      "user-cross-account-follow-write-smoke",
      "fail",
      "ordinary user cross-account follow write smoke is enabled but user A/B credentials are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_USER_A_ACCESS_TOKEN"],
          ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
          ["INTEGRATION_USER_B_ACCESS_TOKEN"],
          ["INTEGRATION_USER_B_ID", "INTEGRATION_USER_B_PASSWORD"],
        ],
      },
    );
    return;
  }

  const problems = [];
  const details = { writeSmokeEnabled: true };

  try {
    const [userATokenResult, userBTokenResult] = await Promise.all([
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A"),
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_B"),
    ]);
    details.userATokenSource = userATokenResult.source;
    details.userBTokenSource = userBTokenResult.source;

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
      addCheck(
        checks,
        "user-cross-account-follow-write-smoke",
        "fail",
        "ordinary user cross-account follow write smoke failed",
        { ...details, problems },
      );
      return;
    }

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    if (!meUrl) {
      addCheck(
        checks,
        "user-cross-account-follow-write-smoke",
        "fail",
        "ordinary user cross-account follow write smoke /api/auth/me URL cannot be built",
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
    const userAId = profileUserIdFromAuthMe(
      unwrapApiPayload(userAMeResult.payload),
      userATokenResult.accountId,
    );
    const userBId = profileUserIdFromAuthMe(
      unwrapApiPayload(userBMeResult.payload),
      userBTokenResult.accountId,
    );
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
    if (userAId && userBId && userAId === userBId) {
      problems.push("ordinary user A and B resolve to the same profile id");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-cross-account-follow-write-smoke",
        "fail",
        "ordinary user cross-account follow write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const initialStatus = await requestFollowStatus(backendOrigin, userATokenResult.token, userBId);
    details.initialFollowStatus = {
      status: initialStatus.status,
      code: initialStatus.code,
      following: initialStatus.following,
    };
    if (!initialStatus.ok || typeof initialStatus.following !== "boolean") {
      problems.push(...initialStatus.problems);
      addCheck(
        checks,
        "user-cross-account-follow-write-smoke",
        "fail",
        "ordinary user cross-account follow write smoke cannot read initial state",
        { ...details, problems },
      );
      return;
    }

    const steps = [];
    const desiredMutationOrder = initialStatus.following
      ? [
          { method: "DELETE", expectedFollowing: false },
          { method: "POST", expectedFollowing: true },
        ]
      : [
          { method: "POST", expectedFollowing: true },
          { method: "DELETE", expectedFollowing: false },
        ];

    for (const step of desiredMutationOrder) {
      const mutation = await requestFollowMutation(
        backendOrigin,
        userATokenResult.token,
        userBId,
        step.method,
      );
      const followStatus = mutation.ok
        ? await requestFollowStatus(backendOrigin, userATokenResult.token, userBId)
        : null;
      steps.push({
        method: step.method,
        mutationStatus: mutation.status,
        mutationCode: mutation.code,
        followStatus: followStatus?.status ?? null,
        followCode: followStatus?.code ?? null,
        following: followStatus?.following ?? null,
      });
      problems.push(...mutation.problems);
      if (followStatus) {
        problems.push(...followStatus.problems);
        if (followStatus.following !== step.expectedFollowing) {
          problems.push(
            `${step.method} follow did not produce expected follow state ${String(step.expectedFollowing)}`,
          );
        }
      }
      if (problems.length > 0) {
        break;
      }
    }

    let finalStatus = await requestFollowStatus(backendOrigin, userATokenResult.token, userBId);
    if (finalStatus.following !== null && finalStatus.following !== initialStatus.following) {
      const restoreMethod = initialStatus.following ? "POST" : "DELETE";
      const restoreMutation = await requestFollowMutation(
        backendOrigin,
        userATokenResult.token,
        userBId,
        restoreMethod,
      );
      const restoredStatus = restoreMutation.ok
        ? await requestFollowStatus(backendOrigin, userATokenResult.token, userBId)
        : finalStatus;
      details.restoreAttempt = {
        method: restoreMethod,
        mutationStatus: restoreMutation.status,
        mutationCode: restoreMutation.code,
        followStatus: restoredStatus.status,
        followCode: restoredStatus.code,
        following: restoredStatus.following,
      };
      problems.push(...restoreMutation.problems);
      if (restoredStatus !== finalStatus) {
        problems.push(...restoredStatus.problems);
      }
      finalStatus = restoredStatus;
    }
    details.followSteps = steps;
    details.finalFollowStatus = {
      status: finalStatus.status,
      code: finalStatus.code,
      following: finalStatus.following,
    };
    problems.push(...finalStatus.problems);
    if (finalStatus.following !== initialStatus.following) {
      problems.push("follow write smoke did not restore the initial follow state");
    }

    addCheck(
      checks,
      "user-cross-account-follow-write-smoke",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "ordinary user cross-account follow write smoke check passed"
        : "ordinary user cross-account follow write smoke check failed",
      { ...details, problems },
    );
  } catch (error) {
    addCheck(
      checks,
      "user-cross-account-follow-write-smoke",
      "fail",
      "ordinary user cross-account follow write smoke is not reachable",
      {
        ...details,
        error: fetchErrorMessage(error),
        timeoutMs: httpTimeoutMs,
      },
    );
  }
}

async function checkUserCrossAccountImWriteSmoke(checks, env, backendOrigin) {
  if (!shouldRunWriteSmoke(env)) {
    return;
  }

  if (
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_A") ||
    !hasUserIntegrationCredentials(env, "INTEGRATION_USER_B")
  ) {
    addCheck(
      checks,
      "user-cross-account-im-write-smoke",
      "fail",
      "ordinary user cross-account IM write smoke is enabled but user A/B credentials are missing",
      {
        requiredVariableSets: [
          ["INTEGRATION_USER_A_ACCESS_TOKEN"],
          ["INTEGRATION_USER_A_ID", "INTEGRATION_USER_A_PASSWORD"],
          ["INTEGRATION_USER_B_ACCESS_TOKEN"],
          ["INTEGRATION_USER_B_ID", "INTEGRATION_USER_B_PASSWORD"],
        ],
      },
    );
    return;
  }

  const details = { writeSmokeEnabled: true };
  const problems = [];

  try {
    const [userATokenResult, userBTokenResult] = await Promise.all([
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_A"),
      getUserAccessToken(env, backendOrigin, "INTEGRATION_USER_B"),
    ]);
    details.userATokenSource = userATokenResult.source;
    details.userBTokenSource = userBTokenResult.source;

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
      addCheck(checks, "user-cross-account-im-write-smoke", "fail", "ordinary user cross-account IM write smoke failed", {
        ...details,
        problems,
      });
      return;
    }

    const meUrl = apiUrlFromOrigin(backendOrigin, "/api/auth/me");
    const createConversationUrl = apiUrlFromOrigin(backendOrigin, "/api/im/conversations");
    if (!meUrl || !createConversationUrl) {
      addCheck(
        checks,
        "user-cross-account-im-write-smoke",
        "fail",
        "ordinary user cross-account IM write smoke URLs cannot be built",
        {
          ...details,
          meUrlBuilt: Boolean(meUrl),
          createConversationUrlBuilt: Boolean(createConversationUrl),
        },
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
    const userAInternalId = numericUserIdFromAuthMe(userAMeData);
    const userBInternalId = numericUserIdFromAuthMe(userBMeData);
    const userAProfileId = profileUserIdFromAuthMe(userAMeData, userATokenResult.accountId);
    const userBProfileId = profileUserIdFromAuthMe(userBMeData, userBTokenResult.accountId);

    details.userAMeStatus = userAMeResult.status;
    details.userBMeStatus = userBMeResult.status;
    details.userAProfileUserId = userAProfileId || null;
    details.userBProfileUserId = userBProfileId || null;
    details.userAInternalIdPresent = userAInternalId !== null;
    details.userBInternalIdPresent = userBInternalId !== null;

    if (!userAMeResult.ok) {
      problems.push(`ordinary user A /api/auth/me returned status ${userAMeResult.status}`);
    }
    if (!userBMeResult.ok) {
      problems.push(`ordinary user B /api/auth/me returned status ${userBMeResult.status}`);
    }
    if (userAInternalId === null) {
      problems.push("ordinary user A internal numeric id cannot be derived from /api/auth/me");
    }
    if (userBInternalId === null) {
      problems.push("ordinary user B internal numeric id cannot be derived from /api/auth/me");
    }
    if (userAInternalId !== null && userBInternalId !== null && userAInternalId === userBInternalId) {
      problems.push("ordinary user A and B resolve to the same internal id");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-cross-account-im-write-smoke",
        "fail",
        "ordinary user cross-account IM write smoke failed before mutation",
        { ...details, problems },
      );
      return;
    }

    const createConversationResult = await requestJsonWithTimeout(createConversationUrl, {
      method: "POST",
      headers: { authorization: `Bearer ${userATokenResult.token}` },
      body: JSON.stringify({ member_ids: [userBInternalId] }),
    });
    const conversationId = conversationIdFromPayload(createConversationResult.payload);
    details.createConversationStatus = createConversationResult.status;
    details.createConversationCode = isRecord(createConversationResult.payload)
      ? createConversationResult.payload.code ?? null
      : null;
    details.conversationId = conversationId || null;

    if (!createConversationResult.ok) {
      problems.push(`IM conversation create returned status ${createConversationResult.status}`);
    }
    if (!conversationId) {
      problems.push("IM conversation create did not return a conversation id");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-cross-account-im-write-smoke",
        "fail",
        "ordinary user cross-account IM write smoke failed while creating conversation",
        { ...details, problems },
      );
      return;
    }

    const clientMsgId = `codex-im-write-smoke-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const content = `codex IM write smoke ${clientMsgId}`;
    const sendMessageUrl = apiUrlFromOrigin(
      backendOrigin,
      `/api/im/conversations/${encodeURIComponent(conversationId)}/messages`,
    );
    const readMessagesUrl = apiUrlFromOrigin(
      backendOrigin,
      `/api/im/conversations/${encodeURIComponent(conversationId)}/messages?limit=20`,
    );
    if (!sendMessageUrl || !readMessagesUrl) {
      addCheck(
        checks,
        "user-cross-account-im-write-smoke",
        "fail",
        "ordinary user cross-account IM write smoke message URLs cannot be built",
        {
          ...details,
          sendMessageUrlBuilt: Boolean(sendMessageUrl),
          readMessagesUrlBuilt: Boolean(readMessagesUrl),
        },
      );
      return;
    }

    const sendMessageResult = await requestJsonWithTimeout(sendMessageUrl, {
      method: "POST",
      headers: { authorization: `Bearer ${userATokenResult.token}` },
      body: JSON.stringify({ content, client_msg_id: clientMsgId }),
    });
    const sentMessageData = unwrapApiPayload(sendMessageResult.payload);
    const sentMessageId = isRecord(sentMessageData) ? sentMessageData.id : null;
    details.sendMessageStatus = sendMessageResult.status;
    details.sendMessageCode = isRecord(sendMessageResult.payload)
      ? sendMessageResult.payload.code ?? null
      : null;
    details.clientMsgId = clientMsgId;
    details.sentMessageId = sentMessageId ?? null;

    if (!sendMessageResult.ok) {
      problems.push(`IM send message returned status ${sendMessageResult.status}`);
    } else {
      problems.push(...validateImMessagePayload(sendMessageResult.payload, "sent IM message"));
      if (isRecord(sentMessageData) && sentMessageData.client_msg_id !== undefined && sentMessageData.client_msg_id !== clientMsgId) {
        problems.push("sent IM message client_msg_id does not match the request");
      }
    }
    if (sentMessageId === null || sentMessageId === undefined) {
      problems.push("sent IM message id is missing");
    }
    if (problems.length > 0) {
      addCheck(
        checks,
        "user-cross-account-im-write-smoke",
        "fail",
        "ordinary user cross-account IM write smoke failed while sending message",
        { ...details, problems },
      );
      return;
    }

    const userBMessagesResult = await requestJsonWithTimeout(readMessagesUrl, {
      method: "GET",
      headers: { authorization: `Bearer ${userBTokenResult.token}` },
    });
    const userBMessages = imMessagesArrayFromPayload(userBMessagesResult.payload);
    const receivedMessage = userBMessages.find(
      (message) =>
        isRecord(message) &&
        (message.client_msg_id === clientMsgId || message.content === content || String(message.id) === String(sentMessageId)),
    );
    details.userBMessagesStatus = userBMessagesResult.status;
    details.userBMessagesCode = isRecord(userBMessagesResult.payload)
      ? userBMessagesResult.payload.code ?? null
      : null;
    details.userBReceivedMessage = Boolean(receivedMessage);

    if (!userBMessagesResult.ok) {
      problems.push(`ordinary user B IM messages returned status ${userBMessagesResult.status}`);
    } else {
      problems.push(...validateImMessagesPayload(userBMessagesResult.payload));
      if (!receivedMessage) {
        problems.push("ordinary user B did not receive the sent IM smoke message in recent conversation messages");
      }
    }

    if (receivedMessage && isRecord(receivedMessage)) {
      const markReadUrl = apiUrlFromOrigin(
        backendOrigin,
        `/api/im/messages/${encodeURIComponent(receivedMessage.id)}/read`,
      );
      if (!markReadUrl) {
        problems.push("IM mark-read URL cannot be built");
      } else {
        const markReadResult = await requestJsonWithTimeout(markReadUrl, {
          method: "POST",
          headers: { authorization: `Bearer ${userBTokenResult.token}` },
        });
        details.markReadStatus = markReadResult.status;
        details.markReadCode = isRecord(markReadResult.payload) ? markReadResult.payload.code ?? null : null;
        if (!markReadResult.ok) {
          problems.push(`IM mark-read returned status ${markReadResult.status}`);
        }
      }
    }

    addCheck(
      checks,
      "user-cross-account-im-write-smoke",
      problems.length === 0 ? "pass" : "fail",
      problems.length === 0
        ? "ordinary user cross-account IM write smoke check passed"
        : "ordinary user cross-account IM write smoke check failed",
      { ...details, problems },
    );
  } catch (error) {
    addCheck(
      checks,
      "user-cross-account-im-write-smoke",
      "fail",
      "ordinary user cross-account IM write smoke is not reachable",
      {
        ...details,
        error: fetchErrorMessage(error),
        timeoutMs: httpTimeoutMs,
      },
    );
  }
}

