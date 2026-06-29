import "server-only";

import {
  getImConversations,
  getImMessages,
  getNotificationUnreadCount,
  getRequestAccessToken,
  type ApiRequestContext,
} from "@/lib/api";
import type { ImConversation, ImMessagesPayload } from "@/lib/types";

export type MessagesInitialData = {
  conversations: ImConversation[];
  notificationUnreadCount: number;
  selectedConversationId: string | number | null;
  thread: ImMessagesPayload | null;
};

export async function getMessagesInitialData(
  context: ApiRequestContext,
  requestedConversationId?: string | number,
): Promise<MessagesInitialData | null> {
  if (!getRequestAccessToken(context)) {
    return null;
  }

  const conversationsPromise = getImConversations(context);
  const unreadCountPromise = getNotificationUnreadCount(context).catch(() => ({
    notification_count: 0,
    system_notification_count: 0,
    total: 0,
  }));
  const requestedThreadPromise =
    requestedConversationId === undefined
      ? null
      : getImMessages(requestedConversationId, { limit: 50 }, context).catch(
          () => null,
        );

  const [conversations, unreadCount] = await Promise.all([
    conversationsPromise,
    unreadCountPromise,
  ]);
  const selectedConversationId = resolveSelectedConversationId(
    conversations,
    requestedConversationId,
  );
  let thread: ImMessagesPayload | null = null;
  if (selectedConversationId !== null) {
    thread =
      requestedThreadPromise &&
      String(selectedConversationId) === String(requestedConversationId)
        ? await requestedThreadPromise
        : await getImMessages(selectedConversationId, { limit: 50 }, context).catch(
            () => null,
          );
  }

  return {
    conversations,
    notificationUnreadCount: unreadCount.total ?? 0,
    selectedConversationId,
    thread,
  };
}

function resolveSelectedConversationId(
  conversations: ImConversation[],
  requestedConversationId?: string | number,
) {
  if (requestedConversationId !== undefined) {
    return (
      conversations.find(
        (conversation) =>
          String(conversation.id) === String(requestedConversationId),
      )?.id ?? null
    );
  }

  return conversations[0]?.id ?? null;
}
