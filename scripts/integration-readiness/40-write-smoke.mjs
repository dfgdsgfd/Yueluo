function hasUserIntegrationCredentials(env, prefix) {
  return Boolean(
    env[`${prefix}_ACCESS_TOKEN`]?.trim() ||
      (env[`${prefix}_ID`]?.trim() && env[`${prefix}_PASSWORD`]?.trim()),
  );
}

function hasAdminIntegrationCredentials(env) {
  return Boolean(
    env.INTEGRATION_ADMIN_ACCESS_TOKEN?.trim() ||
      (env.INTEGRATION_ADMIN_USERNAME?.trim() && env.INTEGRATION_ADMIN_PASSWORD?.trim()),
  );
}

function shouldRunWriteSmoke(env) {
  return isTruthyEnvValue(env.INTEGRATION_ENABLE_WRITE_SMOKE ?? "");
}

function writeSmokePostId(env) {
  return env.INTEGRATION_WRITE_SMOKE_POST_ID?.trim() || "";
}

function postInteractionStateFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return { id: null, liked: null, collected: null };
  }

  return {
    id: data.id ?? data.post_id ?? null,
    liked: typeof data.liked === "boolean" ? data.liked : null,
    collected: typeof data.collected === "boolean" ? data.collected : null,
  };
}

function commentIdFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return null;
  }

  return data.id ?? data.comment_id ?? null;
}

function postIdFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return null;
  }

  return data.id ?? data.post_id ?? null;
}

function validateSingleCommentPayload(payload, label) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (data.id === undefined) {
    problems.push(`${label} id is missing`);
  }
  if (typeof data.content !== "string") {
    problems.push(`${label} content is not a string`);
  }

  return problems;
}

async function requestPostInteractionDetail(backendOrigin, token, postId) {
  const url = apiUrlFromOrigin(backendOrigin, `/api/posts/${encodeURIComponent(postId)}`);
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      state: { id: null, liked: null, collected: null },
      problems: ["post detail URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "GET",
    headers: { authorization: `Bearer ${token}` },
  });
  const state = postInteractionStateFromPayload(result.payload);
  const validationProblems = result.ok ? validatePostDetailPayload(result.payload) : [];

  return {
    ok:
      result.ok &&
      validationProblems.length === 0 &&
      typeof state.liked === "boolean" &&
      typeof state.collected === "boolean",
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    state,
    problems: result.ok ? validationProblems : [`post detail returned status ${result.status}`],
  };
}

async function requestPostLikeToggle(backendOrigin, token, postId) {
  const url = apiUrlFromOrigin(backendOrigin, "/api/likes");
  if (!url) {
    return { ok: false, status: null, code: null, liked: null, problems: ["like toggle URL cannot be built"] };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
    body: JSON.stringify({ target_type: 1, target_id: postId }),
  });
  const liked = booleanFieldFromPayload(result.payload, "liked");
  const problems = result.ok ? [] : [`like toggle returned status ${result.status}`];
  if (result.ok && typeof liked !== "boolean") {
    problems.push("like toggle response does not expose liked boolean");
  }

  return {
    ok: result.ok && problems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    liked,
    problems,
  };
}

async function requestPostCollectToggle(backendOrigin, token, postId) {
  const url = apiUrlFromOrigin(backendOrigin, `/api/posts/${encodeURIComponent(postId)}/collect`);
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      collected: null,
      problems: ["collect toggle URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
  });
  const collected = booleanFieldFromPayload(result.payload, "collected");
  const problems = result.ok ? [] : [`collect toggle returned status ${result.status}`];
  if (result.ok && typeof collected !== "boolean") {
    problems.push("collect toggle response does not expose collected boolean");
  }

  return {
    ok: result.ok && problems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    collected,
    problems,
  };
}

async function requestCreateSmokeComment(backendOrigin, token, postId, content) {
  const url = apiUrlFromOrigin(backendOrigin, "/api/comments");
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      commentId: null,
      problems: ["comment create URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
    body: JSON.stringify({ post_id: postId, content }),
  });
  const commentId = commentIdFromPayload(result.payload);
  const problems = result.ok ? validateSingleCommentPayload(result.payload, "created comment") : [];
  if (!result.ok) {
    problems.push(`comment create returned status ${result.status}`);
  }
  if (result.ok && (commentId === null || commentId === undefined)) {
    problems.push("comment create response does not expose a comment id");
  }

  return {
    ok: result.ok && problems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    commentId,
    problems,
  };
}

async function requestDeleteSmokeComment(backendOrigin, token, commentId) {
  const url = apiUrlFromOrigin(backendOrigin, `/api/comments/${encodeURIComponent(commentId)}`);
  if (!url) {
    return { ok: false, status: null, code: null, problems: ["comment delete URL cannot be built"] };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "DELETE",
    headers: { authorization: `Bearer ${token}` },
  });

  return {
    ok: result.ok,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    problems: result.ok ? [] : [`comment delete returned status ${result.status}`],
  };
}

async function requestCreateDraftPostSmoke(backendOrigin, token) {
  const url = apiUrlFromOrigin(backendOrigin, "/api/posts");
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      postId: null,
      problems: ["draft post create URL cannot be built"],
    };
  }

  const smokeSuffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
    body: JSON.stringify({
      title: `codex draft smoke ${smokeSuffix}`,
      content: `codex draft write smoke ${smokeSuffix}`,
      type: 1,
      is_draft: true,
      visibility: "public",
      tags: ["codex-smoke"],
    }),
  });
  const postId = postIdFromPayload(result.payload);
  const problems = result.ok ? [] : [`draft post create returned status ${result.status}`];
  if (result.ok && (postId === null || postId === undefined)) {
    problems.push("draft post create response does not expose a post id");
  }

  return {
    ok: result.ok && problems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    postId,
    problems,
  };
}

async function requestDeleteDraftPostSmoke(backendOrigin, token, postId) {
  const url = apiUrlFromOrigin(backendOrigin, `/api/posts/${encodeURIComponent(postId)}`);
  if (!url) {
    return { ok: false, status: null, code: null, problems: ["draft post delete URL cannot be built"] };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "DELETE",
    headers: { authorization: `Bearer ${token}` },
  });

  return {
    ok: result.ok,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    problems: result.ok ? [] : [`draft post delete returned status ${result.status}`],
  };
}

function paymentCodeStateFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return { wechatUrl: null, alipayUrl: null };
  }

  return {
    wechatUrl: typeof data.wechat_url === "string" && data.wechat_url.trim() ? data.wechat_url : null,
    alipayUrl: typeof data.alipay_url === "string" && data.alipay_url.trim() ? data.alipay_url : null,
  };
}

function hasPaymentCodeValue(state) {
  return Boolean(state.wechatUrl || state.alipayUrl);
}

async function requestWithdrawPaymentCode(backendOrigin, token) {
  const url = apiUrlFromOrigin(backendOrigin, "/api/withdraw/payment-code");
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      state: { wechatUrl: null, alipayUrl: null },
      problems: ["withdraw payment-code URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "GET",
    headers: { authorization: `Bearer ${token}` },
  });
  const state = paymentCodeStateFromPayload(result.payload);
  const validationProblems = result.ok
    ? validateStringOrNullFieldsPayload(result.payload, "withdraw payment-code", [
        "wechat_url",
        "alipay_url",
      ])
    : [];

  return {
    ok: result.ok && validationProblems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    state,
    problems: result.ok ? validationProblems : [`withdraw payment-code returned status ${result.status}`],
  };
}

async function requestSaveWithdrawPaymentCode(backendOrigin, token, state) {
  const url = apiUrlFromOrigin(backendOrigin, "/api/withdraw/payment-code");
  if (!url) {
    return {
      ok: false,
      status: null,
      code: null,
      state: { wechatUrl: null, alipayUrl: null },
      problems: ["withdraw payment-code save URL cannot be built"],
    };
  }

  const result = await requestJsonWithTimeout(url, {
    method: "POST",
    headers: { authorization: `Bearer ${token}` },
    body: JSON.stringify({
      wechat_url: state.wechatUrl,
      alipay_url: state.alipayUrl,
    }),
  });
  const savedState = paymentCodeStateFromPayload(result.payload);
  const validationProblems = result.ok
    ? validateStringOrNullFieldsPayload(result.payload, "withdraw payment-code save", [
        "wechat_url",
        "alipay_url",
      ])
    : [];

  return {
    ok: result.ok && validationProblems.length === 0,
    status: result.status,
    code: isRecord(result.payload) ? result.payload.code ?? null : null,
    state: savedState,
    problems: result.ok
      ? validationProblems
      : [`withdraw payment-code save returned status ${result.status}`],
  };
}

