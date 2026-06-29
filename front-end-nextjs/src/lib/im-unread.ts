import { loadMessagesPageData } from "./im-cache";
import type { ImConversation, ImMessage } from "./types";

export const MESSAGE_BADGE_COUNT_EVENT = "yuem:message-badge-count";

export type ImSocketMessageEvent = {
  conversation_id?: string | number;
  conversationId?: string | number;
  type?: string;
};

type MessageBadgeCountEventDetail = {
  count: number;
};

export function getConversationUnreadTotal(
  conversations: readonly Pick<ImConversation, "unread_count">[],
) {
  return conversations.reduce(
    (total, conversation) => total + (conversation.unread_count ?? 0),
    0,
  );
}

export async function markLatestImMessageRead(
  messages: readonly ImMessage[],
  markRead: (messageId: string | number) => Promise<unknown>,
) {
  const latestMessage = messages.at(-1);

  if (!latestMessage) {
    return null;
  }

  await markRead(latestMessage.id);
  return latestMessage.id;
}

export async function getMessageBadgeCount(
  options: { background?: boolean } = {},
) {
  try {
    const data = await loadMessagesPageData({ background: options.background });

    return (
      getConversationUnreadTotal(data.conversations) +
      data.notificationUnreadCount
    );
  } catch {
    return 0;
  }
}

export function emitMessageBadgeCount(count: number) {
  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(
    new CustomEvent<MessageBadgeCountEventDetail>(MESSAGE_BADGE_COUNT_EVENT, {
      detail: { count },
    }),
  );
}

export function subscribeMessageBadgeCount(listener: (count: number) => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  function handleMessageBadgeCount(event: Event) {
    const count = (event as CustomEvent<MessageBadgeCountEventDetail>).detail
      ?.count;
    if (typeof count === "number" && Number.isFinite(count)) {
      listener(count);
    }
  }

  window.addEventListener(MESSAGE_BADGE_COUNT_EVENT, handleMessageBadgeCount);
  return () =>
    window.removeEventListener(
      MESSAGE_BADGE_COUNT_EVENT,
      handleMessageBadgeCount,
    );
}

export function parseImSocketMessage(
  data: unknown,
): ImSocketMessageEvent | null {
  if (typeof data !== "string") {
    return null;
  }

  try {
    return JSON.parse(data) as ImSocketMessageEvent;
  } catch {
    return null;
  }
}

export function getImSocketConversationId(
  payload: ImSocketMessageEvent | null,
) {
  return payload?.conversation_id ?? payload?.conversationId;
}
