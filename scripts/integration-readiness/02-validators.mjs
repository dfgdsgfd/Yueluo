function validateSearchPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["search data is not an object"];
  }

  const postsGroup = isRecord(data.posts) ? data.posts : null;
  const posts = Array.isArray(postsGroup?.data)
    ? postsGroup.data
    : Array.isArray(data.data)
      ? data.data
      : null;
  const pagination = isRecord(postsGroup?.pagination)
    ? postsGroup.pagination
    : isRecord(data.pagination)
      ? data.pagination
      : null;

  if (!posts) {
    problems.push("search posts are not exposed via posts.data or data");
  }
  if (!pagination) {
    problems.push("search pagination is not exposed via posts.pagination or pagination");
  }
  if (data.users !== undefined) {
    if (!isRecord(data.users)) {
      problems.push("search users is not an object");
    } else if (!Array.isArray(data.users.data)) {
      problems.push("search users.data is not an array");
    }
  }

  return problems;
}

function validateUserProfilePayload(payload, label) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (data.id === undefined && data.user_id === undefined && data.xise_id === undefined) {
    problems.push(`${label} does not expose id, user_id or xise_id`);
  }
  for (const field of ["nickname", "avatar", "background", "bio", "location"]) {
    if (data[field] !== undefined && data[field] !== null && typeof data[field] !== "string") {
      problems.push(`${label} ${field} is not a string or null`);
    }
  }
  for (const field of ["follow_count", "fans_count", "like_count", "collect_count", "post_count"]) {
    if (data[field] !== undefined && typeof data[field] !== "number") {
      problems.push(`${label} ${field} is not a number`);
    }
  }

  return problems;
}

function validateUserPostListPayload(payload, field, label) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (!Array.isArray(data[field])) {
    problems.push(`${label} ${field} is not an array`);
  }
  if (data.pagination !== undefined && !isRecord(data.pagination)) {
    problems.push(`${label} pagination is not an object`);
  }

  return problems;
}

function validatePostDetailPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["post detail data is not an object"];
  }
  if (data.id === undefined && data.post_id === undefined) {
    problems.push("post detail does not expose id or post_id");
  }
  if (data.title !== undefined && typeof data.title !== "string") {
    problems.push("post detail title is not a string");
  }
  if (data.type !== undefined && typeof data.type !== "number") {
    problems.push("post detail type is not a number");
  }
  for (const field of ["liked", "collected"]) {
    if (data[field] !== undefined && typeof data[field] !== "boolean") {
      problems.push(`post detail ${field} is not a boolean`);
    }
  }

  return problems;
}

function validateCommentsPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["comments data is not an object"];
  }
  if (!Array.isArray(data.comments)) {
    problems.push("comments data.comments is not an array");
  } else {
    const invalidComment = data.comments.find(
      (comment) =>
        !isRecord(comment) ||
        comment.id === undefined ||
        typeof comment.content !== "string",
    );
    if (invalidComment) {
      problems.push("comments items do not expose id and content");
    }
  }
  if (!isRecord(data.pagination)) {
    problems.push("comments pagination is not an object");
  }

  return problems;
}

function postIdFromFeedPayload(payload) {
  const data = unwrapApiPayload(payload);
  const posts = isRecord(data) && Array.isArray(data.posts) ? data.posts : [];

  for (const post of posts) {
    if (!isRecord(post)) {
      continue;
    }
    for (const field of ["id", "post_id"]) {
      const value = post[field];
      if (typeof value === "string" && value.trim()) {
        return value.trim();
      }
      if (typeof value === "number" && Number.isFinite(value)) {
        return String(value);
      }
    }
  }

  return "";
}

function validateFollowStatusPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["follow status data is not an object"];
  }
  if (
    data.isFollowing === undefined &&
    data.followed === undefined &&
    data.buttonType === undefined
  ) {
    problems.push("follow status does not expose isFollowing, followed or buttonType");
  }
  if (data.isFollowing !== undefined && typeof data.isFollowing !== "boolean") {
    problems.push("follow status isFollowing is not a boolean");
  }
  if (data.followed !== undefined && typeof data.followed !== "boolean") {
    problems.push("follow status followed is not a boolean");
  }
  if (data.buttonType !== undefined && data.buttonType !== null && typeof data.buttonType !== "string") {
    problems.push("follow status buttonType is not a string or null");
  }

  return problems;
}

function validateCategoriesPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!Array.isArray(data)) {
    return ["categories data is not an array"];
  }

  const invalidCategory = data.find(
    (item) =>
      !isRecord(item) ||
      typeof item.id !== "number" ||
      (typeof item.name !== "string" && typeof item.category_title !== "string"),
  );
  return invalidCategory ? ["categories items do not expose id plus name/category_title"] : [];
}

function validateUnreadCountPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["notification unread-count data is not an object"];
  }
  for (const field of ["notification_count", "system_notification_count", "total"]) {
    if (typeof data[field] !== "number") {
      problems.push(`notification unread-count ${field} is not a number`);
    }
  }

  return problems;
}

function validateNotificationListPayload(payload, label, { system = false } = {}) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (!Array.isArray(data.data)) {
    problems.push(`${label} data.data is not an array`);
  }
  if (!isRecord(data.pagination)) {
    problems.push(`${label} pagination is not an object`);
  }

  const invalidItem = Array.isArray(data.data)
    ? data.data.find((item) => {
        if (!isRecord(item)) {
          return true;
        }
        if (item.id === undefined || typeof item.title !== "string") {
          return true;
        }
        if (system) {
          return item.content !== undefined && item.content !== null && typeof item.content !== "string";
        }
        return typeof item.is_read !== "boolean";
      })
    : null;
  if (invalidItem) {
    problems.push(
      system
        ? `${label} items do not expose id/title and optional string content`
        : `${label} items do not expose id/title/is_read`,
    );
  }

  return problems;
}

function validateSystemNotificationPopupPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!Array.isArray(data)) {
    return ["system notification popup data is not an array"];
  }

  const invalidItem = data.find((item) => {
    if (!isRecord(item)) {
      return true;
    }
    if (item.id === undefined || typeof item.title !== "string") {
      return true;
    }
    if (item.content !== undefined && item.content !== null && typeof item.content !== "string") {
      return true;
    }
    if (item.show_popup !== undefined && typeof item.show_popup !== "boolean") {
      return true;
    }
    return item.is_active !== undefined && typeof item.is_active !== "boolean";
  });
  if (invalidItem) {
    problems.push("system notification popup items do not expose id/title plus valid content/show_popup/is_active fields");
  }

  return problems;
}

function validateArrayPayload(payload, label) {
  const data = unwrapApiPayload(payload);
  return Array.isArray(data) ? [] : [`${label} data is not an array`];
}

function firstConversationIdFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  const conversations = Array.isArray(data) ? data : [];

  for (const conversation of conversations) {
    if (!isRecord(conversation)) {
      continue;
    }
    for (const field of ["id", "external_id"]) {
      const value = conversation[field];
      if (typeof value === "string" && value.trim()) {
        return value.trim();
      }
      if (typeof value === "number" && Number.isFinite(value)) {
        return String(value);
      }
    }
  }

  return "";
}

function conversationIdFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return "";
  }

  for (const field of ["id", "external_id"]) {
    const value = data[field];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
    if (typeof value === "number" && Number.isFinite(value)) {
      return String(value);
    }
  }

  return "";
}

function numericUserIdFromAuthMe(meData) {
  if (!isRecord(meData)) {
    return null;
  }

  const id = meData.id;
  if (typeof id === "number" && Number.isFinite(id)) {
    return id;
  }
  if (typeof id === "string" && /^\d+$/.test(id.trim())) {
    return Number(id.trim());
  }

  return null;
}

function imMessagesArrayFromPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (Array.isArray(data)) {
    return data;
  }
  if (isRecord(data) && Array.isArray(data.messages)) {
    return data.messages;
  }
  return [];
}

function validateImMessagesPayload(payload) {
  const data = unwrapApiPayload(payload);
  const envelope = isRecord(payload) ? payload : {};
  const dataRecord = isRecord(data) ? data : null;
  const messages = Array.isArray(data)
    ? data
    : Array.isArray(dataRecord?.messages)
      ? dataRecord.messages
      : null;
  const pagination = isRecord(envelope.pagination)
    ? envelope.pagination
    : isRecord(dataRecord?.pagination)
      ? dataRecord.pagination
      : null;
  const problems = [];

  if (!messages) {
    problems.push("IM messages are not exposed via data or data.messages");
  } else {
    const invalidMessage = messages.find(
      (message) =>
        !isRecord(message) ||
        message.id === undefined ||
        message.conversation_id === undefined ||
        message.sender_id === undefined ||
        typeof message.content !== "string",
    );
    if (invalidMessage) {
      problems.push("IM message items do not expose id, conversation_id, sender_id and content");
    }
  }
  if (!pagination) {
    problems.push("IM messages pagination is not exposed at the top level or data.pagination");
  } else {
    if (pagination.limit !== undefined && typeof pagination.limit !== "number") {
      problems.push("IM messages pagination.limit is not a number");
    }
    if (pagination.has_more !== undefined && typeof pagination.has_more !== "boolean") {
      problems.push("IM messages pagination.has_more is not a boolean");
    }
  }

  return problems;
}

function validateImMessagePayload(payload, label = "IM message") {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (data.id === undefined) {
    problems.push(`${label} id is missing`);
  }
  if (data.conversation_id === undefined) {
    problems.push(`${label} conversation_id is missing`);
  }
  if (data.sender_id === undefined) {
    problems.push(`${label} sender_id is missing`);
  }
  if (typeof data.content !== "string") {
    problems.push(`${label} content is not a string`);
  }
  if (data.client_msg_id !== undefined && data.client_msg_id !== null && typeof data.client_msg_id !== "string") {
    problems.push(`${label} client_msg_id is not a string or null`);
  }

  return problems;
}

function validateImSyncPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["IM sync data is not an object"];
  }
  if (!Array.isArray(data.messages)) {
    problems.push("IM sync messages is not an array");
  }
  if (data.cursor === undefined) {
    problems.push("IM sync cursor is missing");
  }
  if (data.has_more !== undefined && typeof data.has_more !== "boolean") {
    problems.push("IM sync has_more is not a boolean");
  }

  return problems;
}

function validateNumberFieldsPayload(payload, label, fields) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  for (const field of fields) {
    if (typeof data[field] !== "number") {
      problems.push(`${label} ${field} is not a number`);
    }
  }

  return problems;
}

function validateBooleanFieldsPayload(payload, label, fields) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  for (const field of fields) {
    if (typeof data[field] !== "boolean") {
      problems.push(`${label} ${field} is not a boolean`);
    }
  }

  return problems;
}

function validateStringOrNullFieldsPayload(payload, label, fields) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  for (const field of fields) {
    if (data[field] !== undefined && data[field] !== null && typeof data[field] !== "string") {
      problems.push(`${label} ${field} is not a string or null`);
    }
  }

  return problems;
}

function validateListWithPaginationPayload(payload, label) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return [`${label} data is not an object`];
  }
  if (!Array.isArray(data.list)) {
    problems.push(`${label} list is not an array`);
  }
  if (!isRecord(data.pagination)) {
    problems.push(`${label} pagination is not an object`);
  }

  return problems;
}

function validateAdminListPayload(payload, label) {
  const envelope = isRecord(payload) ? payload : {};
  const data = unwrapApiPayload(payload);
  const dataRecord = isRecord(data) ? data : null;
  const itemsSource = Array.isArray(data)
    ? data
    : Array.isArray(dataRecord?.data)
      ? dataRecord.data
      : Array.isArray(dataRecord?.list)
        ? dataRecord.list
        : Array.isArray(dataRecord?.items)
          ? dataRecord.items
          : null;
  const paginationSource =
    isRecord(envelope.pagination)
      ? envelope.pagination
      : isRecord(dataRecord?.pagination)
        ? dataRecord.pagination
        : dataRecord && ("total" in dataRecord || "page" in dataRecord || "limit" in dataRecord)
          ? dataRecord
          : null;
  const problems = [];

  if (!itemsSource) {
    problems.push(`${label} does not expose an array via data, data.data, data.list or data.items`);
  }
  if (paginationSource !== null && !isRecord(paginationSource)) {
    problems.push(`${label} pagination is not an object`);
  }

  return problems;
}

function validateCreatorStatsPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["creator stats data is not an object"];
  }
  if (typeof data.window_days !== "number") {
    problems.push("creator stats window_days is not a number");
  }
  if (typeof data.generated_at !== "string") {
    problems.push("creator stats generated_at is not a string");
  }
  for (const field of ["fans", "post_totals", "interactions"]) {
    if (!isRecord(data[field])) {
      problems.push(`creator stats ${field} is not an object`);
    }
  }

  return problems;
}

function validateCreatorTrendsPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["creator trends data is not an object"];
  }
  if (typeof data.days !== "number") {
    problems.push("creator trends days is not a number");
  }
  if (!Array.isArray(data.labels) || data.labels.some((item) => typeof item !== "string")) {
    problems.push("creator trends labels is not a string array");
  }
  for (const field of ["views", "likes", "collects", "comments", "followers"]) {
    if (!Array.isArray(data[field]) || data[field].some((item) => typeof item !== "number")) {
      problems.push(`creator trends ${field} is not a number array`);
    }
  }

  return problems;
}

function validateCreatorQualityRewardsPayload(payload) {
  const problems = validateListWithPaginationPayload(payload, "creator quality rewards");
  const data = unwrapApiPayload(payload);

  if (isRecord(data)) {
    if (typeof data.total_earnings !== "number") {
      problems.push("creator quality rewards total_earnings is not a number");
    }
    if (!Array.isArray(data.stats)) {
      problems.push("creator quality rewards stats is not an array");
    }
  }

  return problems;
}

function validateCreatorConfigPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["creator config data is not an object"];
  }
  for (const field of ["platformFeeRate", "creatorShareRate", "minWithdrawAmount"]) {
    if (typeof data[field] !== "number") {
      problems.push(`creator config ${field} is not a number`);
    }
  }
  if (typeof data.withdrawEnabled !== "boolean") {
    problems.push("creator config withdrawEnabled is not a boolean");
  }
  if (data.extendedEarnings !== undefined) {
    if (!isRecord(data.extendedEarnings)) {
      problems.push("creator config extendedEarnings is not an object");
    } else {
      if (typeof data.extendedEarnings.enabled !== "boolean") {
        problems.push("creator config extendedEarnings.enabled is not a boolean");
      }
      if (typeof data.extendedEarnings.dailyCap !== "number") {
        problems.push("creator config extendedEarnings.dailyCap is not a number");
      }
      if (!isRecord(data.extendedEarnings.rates)) {
        problems.push("creator config extendedEarnings.rates is not an object");
      }
    }
  }

  return problems;
}

function validateBalanceConfigPayload(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return ["balance config data is not an object"];
  }
  return typeof data.enabled === "boolean" ? [] : ["balance config enabled is not a boolean"];
}

function validateRechargeConfigPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    return ["recharge config data is not an object"];
  }
  if (data.custom_amount_enable !== undefined && typeof data.custom_amount_enable !== "boolean") {
    problems.push("recharge config custom_amount_enable is not a boolean");
  }
  for (const field of ["min_amount", "max_amount"]) {
    if (data[field] !== undefined && typeof data[field] !== "number") {
      problems.push(`recharge config ${field} is not a number`);
    }
  }
  if (
    typeof data.min_amount === "number" &&
    typeof data.max_amount === "number" &&
    data.min_amount > data.max_amount
  ) {
    problems.push("recharge config min_amount is greater than max_amount");
  }
  if (data.recharge_url !== undefined) {
    if (typeof data.recharge_url !== "string") {
      problems.push("recharge config recharge_url is not a string");
    } else if (data.recharge_url.trim()) {
      try {
        new URL(data.recharge_url);
      } catch {
        problems.push("recharge config recharge_url is not an absolute URL");
      }
    }
  }
  if (data.options !== undefined) {
    if (!Array.isArray(data.options)) {
      problems.push("recharge config options is not an array");
    } else {
      const invalidOption = data.options.find(
        (item) =>
          !isRecord(item) ||
          typeof item.amount !== "number" ||
          (item.bonus !== undefined && typeof item.bonus !== "number"),
      );
      if (invalidOption) {
        problems.push("recharge config options items must expose numeric amount and optional numeric bonus");
      }
    }
  }
  if (data.gift_card_purchase !== undefined) {
    if (!isRecord(data.gift_card_purchase)) {
      problems.push("recharge config gift_card_purchase is not an object");
    } else {
      if (typeof data.gift_card_purchase.enabled !== "boolean") {
        problems.push("recharge config gift_card_purchase.enabled is not a boolean");
      }
      if (
        data.gift_card_purchase.options !== undefined &&
        !Array.isArray(data.gift_card_purchase.options)
      ) {
        problems.push("recharge config gift_card_purchase.options is not an array");
      } else if (Array.isArray(data.gift_card_purchase.options)) {
        const invalidGiftCardOption = data.gift_card_purchase.options.find(
          (item) =>
            !isRecord(item) ||
            typeof item.amount !== "number" ||
            typeof item.price !== "number" ||
            (item.discount !== undefined && typeof item.discount !== "number"),
        );
        if (invalidGiftCardOption) {
          problems.push("recharge config gift_card_purchase.options items must expose numeric amount, price and optional discount");
        }
      }
      if (
        data.gift_card_purchase.disallow_self_use !== undefined &&
        typeof data.gift_card_purchase.disallow_self_use !== "boolean"
      ) {
        problems.push("recharge config gift_card_purchase.disallow_self_use is not a boolean");
      }
      if (
        data.gift_card_purchase.expires_in_days !== undefined &&
        typeof data.gift_card_purchase.expires_in_days !== "number"
      ) {
        problems.push("recharge config gift_card_purchase.expires_in_days is not a number");
      }
      if (
        data.gift_card_purchase.user_discounts !== undefined &&
        !Array.isArray(data.gift_card_purchase.user_discounts)
      ) {
        problems.push("recharge config gift_card_purchase.user_discounts is not an array");
      }
    }
  }

  return problems;
}

