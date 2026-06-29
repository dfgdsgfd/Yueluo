import { getImConversations, getImMessages } from "./api/im";
import { getNotificationUnreadCount } from "./api/notifications";
import { IM_BACKGROUND_REQUEST_OPTIONS } from "./im-background-request";
import type { ImConversation, ImMessagesPayload } from "./types";

const MESSAGE_PAGE_CACHE_TTL_MS = 60 * 1000;
const MESSAGE_THREAD_CACHE_TTL_MS = 30 * 1000;
const IM_BACKGROUND_CIRCUIT_OPEN_MS = 15000;

export type MessagesPageData = {
  conversations: ImConversation[];
  notificationUnreadCount: number;
};

type MessagesPageCacheEntry = MessagesPageData & {
  updatedAt: number;
};

type MessageThreadCacheEntry = ImMessagesPayload & {
  updatedAt: number;
};

let messagesPageCache: MessagesPageCacheEntry | null = null;
let messagesPageRequest: Promise<MessagesPageData> | null = null;
let messagesPageBackgroundCircuitOpenedAt = 0;
const messageThreadCache = new Map<string, MessageThreadCacheEntry>();
const messageThreadRequests = new Map<string, Promise<ImMessagesPayload>>();
const messageThreadBackgroundCircuitOpenedAt = new Map<string, number>();

const emptyNotificationUnreadCount = {
  notification_count: 0,
  system_notification_count: 0,
  total: 0,
};

export function getCachedMessagesPageData(): MessagesPageData | null {
  if (!messagesPageCache) {
    return null;
  }

  if (Date.now() - messagesPageCache.updatedAt > MESSAGE_PAGE_CACHE_TTL_MS) {
    messagesPageCache = null;
    return null;
  }

  return {
    conversations: messagesPageCache.conversations,
    notificationUnreadCount: messagesPageCache.notificationUnreadCount,
  };
}

export function setCachedMessagesPageData(data: MessagesPageData) {
  messagesPageCache = {
    ...data,
    updatedAt: Date.now(),
  };
}

export function updateCachedMessagesPageData(
  updater: (current: MessagesPageData) => MessagesPageData,
) {
  const current = getCachedMessagesPageData();
  if (!current) {
    return;
  }

  setCachedMessagesPageData(updater(current));
}

function isCircuitOpen(openedAt: number) {
  return openedAt > 0 && Date.now() - openedAt < IM_BACKGROUND_CIRCUIT_OPEN_MS;
}

function openMessagesPageBackgroundCircuit() {
  messagesPageBackgroundCircuitOpenedAt = Date.now();
}

function closeMessagesPageBackgroundCircuit() {
  messagesPageBackgroundCircuitOpenedAt = 0;
}

function getThreadCircuitKey(conversationId: string | number) {
  return String(conversationId);
}

function openThreadBackgroundCircuit(conversationId: string | number) {
  messageThreadBackgroundCircuitOpenedAt.set(
    getThreadCircuitKey(conversationId),
    Date.now(),
  );
}

function closeThreadBackgroundCircuit(conversationId: string | number) {
  messageThreadBackgroundCircuitOpenedAt.delete(
    getThreadCircuitKey(conversationId),
  );
}

export async function loadMessagesPageData(
  options: { background?: boolean; refresh?: boolean } = {},
) {
  if (!options.refresh) {
    const cached = getCachedMessagesPageData();
    if (cached) {
      return cached;
    }
  }

  if (messagesPageRequest) {
    return messagesPageRequest;
  }

  if (
    options.background &&
    isCircuitOpen(messagesPageBackgroundCircuitOpenedAt)
  ) {
    return (
      getCachedMessagesPageData() ?? {
        conversations: [],
        notificationUnreadCount: 0,
      }
    );
  }

  let imUnavailable = false;
  const requestOptions = options.background
    ? IM_BACKGROUND_REQUEST_OPTIONS
    : {};

  messagesPageRequest = Promise.all([
    getImConversations({}, requestOptions).catch(() => {
      imUnavailable = true;
      if (options.background) {
        openMessagesPageBackgroundCircuit();
      }
      return getCachedMessagesPageData()?.conversations ?? [];
    }),
    getNotificationUnreadCount(
      {},
      options.background ? IM_BACKGROUND_REQUEST_OPTIONS : {},
    ).catch(() => emptyNotificationUnreadCount),
  ])
    .then(([conversations, unreadCount]) => {
      const data: MessagesPageData = {
        conversations,
        notificationUnreadCount: unreadCount.total ?? 0,
      };
      if (!imUnavailable) {
        closeMessagesPageBackgroundCircuit();
      }
      if (!options.background || !imUnavailable) {
        setCachedMessagesPageData(data);
      }
      return data;
    })
    .finally(() => {
      messagesPageRequest = null;
    });

  return messagesPageRequest;
}

export function prefetchMessagesPageData() {
  return loadMessagesPageData({ background: true });
}

export function getCachedConversationMessages(
  conversationId: string | number,
): ImMessagesPayload | null {
  const cached = messageThreadCache.get(String(conversationId));
  if (!cached) {
    return null;
  }

  if (Date.now() - cached.updatedAt > MESSAGE_THREAD_CACHE_TTL_MS) {
    messageThreadCache.delete(String(conversationId));
    return null;
  }

  return {
    messages: cached.messages,
    pagination: cached.pagination,
  };
}

export function setCachedConversationMessages(
  conversationId: string | number,
  payload: ImMessagesPayload,
) {
  messageThreadCache.set(String(conversationId), {
    ...payload,
    updatedAt: Date.now(),
  });
}

export async function loadConversationMessagesData(
  conversationId: string | number,
  options: {
    background?: boolean;
    before?: string | number | null;
    limit?: number;
    refresh?: boolean;
  } = {},
) {
  const canUseThreadCache =
    options.before === undefined || options.before === null;
  if (canUseThreadCache && !options.refresh) {
    const cached = getCachedConversationMessages(conversationId);
    if (cached) {
      return cached;
    }
  }

  const requestKey = canUseThreadCache
    ? String(conversationId)
    : `${conversationId}:${options.before ?? ""}:${options.limit ?? ""}`;
  const inFlightRequest = messageThreadRequests.get(requestKey);
  if (inFlightRequest) {
    return inFlightRequest;
  }

  if (
    options.background &&
    isCircuitOpen(
      messageThreadBackgroundCircuitOpenedAt.get(
        getThreadCircuitKey(conversationId),
      ) ?? 0,
    )
  ) {
    return (
      getCachedConversationMessages(conversationId) ?? {
        messages: [],
        pagination: {
          has_more: false,
          limit: options.limit ?? 50,
          next_before: null,
        },
      }
    );
  }

  const requestOptions = options.background
    ? IM_BACKGROUND_REQUEST_OPTIONS
    : {};
  const request = getImMessages(
    conversationId,
    {
      before: options.before,
      limit: options.limit,
    },
    {},
    requestOptions,
  )
    .then((payload) => {
      if (canUseThreadCache) {
        setCachedConversationMessages(conversationId, payload);
      }
      closeThreadBackgroundCircuit(conversationId);
      return payload;
    })
    .catch((error) => {
      if (options.background) {
        openThreadBackgroundCircuit(conversationId);
      }
      throw error;
    })
    .finally(() => {
      messageThreadRequests.delete(requestKey);
    });

  messageThreadRequests.set(requestKey, request);
  return request;
}

export function prefetchConversationMessages(conversationId: string | number) {
  return loadConversationMessagesData(conversationId);
}
