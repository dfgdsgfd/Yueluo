"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import {
  ChevronUp,
  Inbox,
  Loader2,
  RefreshCw,
  Search,
  Send,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  getStoredAccessToken,
  getStoredUser,
  markImMessageRead,
  sendImMessage,
} from "@/lib/api";
import {
  getCachedConversationMessages,
  getCachedMessagesPageData,
  loadConversationMessagesData,
  loadMessagesPageData,
  setCachedConversationMessages,
  setCachedMessagesPageData,
  updateCachedMessagesPageData,
} from "@/lib/im-cache";
import {
  emitMessageBadgeCount,
  getConversationUnreadTotal,
  getMessageBadgeCount,
  markLatestImMessageRead,
} from "@/lib/im-unread";
import type { ImConversation, ImMessage } from "@/lib/types";
import type { MessagesInitialData } from "@/lib/server/messages-page-data";
import { cn } from "@/lib/utils";

import {
  ConversationAvatar,
  ConversationListItem,
  EmptyState,
  MessageBubble,
  MessageShell,
  MobileMessageDetail,
  MobileMessagesList,
  SocketStatus,
  filterConversations,
  getConversationTitle,
  getRequestedConversationId,
  mergeMessages,
  resolveInitialSelectedConversationId,
  type SocketState,
} from "./messages-page-view";
import { useImSocketRefresh } from "./use-im-socket-refresh";

const messagePageSize = 50;

export function MessagesPage({
  conversationId,
  initialData,
}: {
  conversationId?: string | number;
  initialData?: MessagesInitialData | null;
} = {}) {
  const feedT = useTranslations("feed");
  const [initialMessagesState] = useState(() => {
    const pageData = initialData
      ? {
          conversations: initialData.conversations,
          notificationUnreadCount: initialData.notificationUnreadCount,
        }
      : getCachedMessagesPageData();
    const selectedConversationId = resolveInitialSelectedConversationId(
      initialData?.selectedConversationId ?? conversationId,
      pageData?.conversations ?? [],
    );
    const thread =
      initialData?.thread ??
      (selectedConversationId === null
        ? null
        : getCachedConversationMessages(selectedConversationId));

    return { pageData, selectedConversationId, thread };
  });
  const [authToken, setAuthToken] = useState<string | null | undefined>(
    undefined,
  );
  const [conversations, setConversations] = useState<ImConversation[]>(
    () => initialMessagesState.pageData?.conversations ?? [],
  );
  const [selectedConversationId, setSelectedConversationId] = useState<
    string | number | null
  >(() => initialMessagesState.selectedConversationId);
  const [messages, setMessages] = useState<ImMessage[]>(
    () => initialMessagesState.thread?.messages ?? [],
  );
  const [notificationUnreadCount, setNotificationUnreadCount] = useState(
    () => initialMessagesState.pageData?.notificationUnreadCount ?? 0,
  );
  const [conversationQuery, setConversationQuery] = useState("");
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [hasOlderMessages, setHasOlderMessages] = useState(
    () => initialMessagesState.thread?.pagination.has_more ?? false,
  );
  const [isLoadingConversations, setIsLoadingConversations] = useState(
    () => !initialMessagesState.pageData,
  );
  const [isLoadingMessages, setIsLoadingMessages] = useState(() =>
    Boolean(
      initialMessagesState.selectedConversationId &&
      !initialMessagesState.thread,
    ),
  );
  const [isLoadingOlderMessages, setIsLoadingOlderMessages] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [nextBefore, setNextBefore] = useState<string | number | null>(
    () => initialMessagesState.thread?.pagination.next_before ?? null,
  );
  const [socketState, setSocketState] = useState<SocketState>("idle");
  const selectedConversationIdRef = useRef<string | number | null>(null);
  const notificationUnreadCountRef = useRef(
    initialMessagesState.pageData?.notificationUnreadCount ?? 0,
  );

  const viewerId = useMemo(() => {
    if (!authToken) {
      return null;
    }

    const user = getStoredUser();
    return user?.id === undefined || user.id === null ? null : String(user.id);
  }, [authToken]);

  const selectedConversation = useMemo(
    () =>
      conversations.find(
        (conversation) =>
          String(conversation.id) === String(selectedConversationId),
      ) ?? null,
    [conversations, selectedConversationId],
  );

  const filteredConversations = useMemo(
    () => filterConversations(conversations, conversationQuery),
    [conversationQuery, conversations],
  );

  const markConversationRead = useCallback(
    (conversationId: string | number, threadMessages: readonly ImMessage[]) => {
      setConversations((items) => {
        const nextItems = items.map((item) =>
          String(item.id) === String(conversationId)
            ? { ...item, unread_count: 0 }
            : item,
        );
        updateCachedMessagesPageData((current) => ({
          ...current,
          conversations: current.conversations.map((item) =>
            String(item.id) === String(conversationId)
              ? { ...item, unread_count: 0 }
              : item,
          ),
        }));
        emitMessageBadgeCount(
          getConversationUnreadTotal(nextItems) +
            notificationUnreadCountRef.current,
        );
        return nextItems;
      });

      void markLatestImMessageRead(threadMessages, markImMessageRead)
        .then((readMessageId) => {
          if (readMessageId === null) {
            return;
          }
          void getMessageBadgeCount({ background: true })
            .then(emitMessageBadgeCount)
            .catch(() => undefined);
        })
        .catch(() => undefined);
    },
    [],
  );

  const loadConversations = useCallback(
    async (options: { silent?: boolean } = {}) => {
      try {
        const cachedPageData = getCachedMessagesPageData();
        if (cachedPageData && !options.silent) {
          setConversations(cachedPageData.conversations);
          setNotificationUnreadCount(cachedPageData.notificationUnreadCount);
          notificationUnreadCountRef.current =
            cachedPageData.notificationUnreadCount;
          setIsLoadingConversations(false);
        } else if (!options.silent) {
          setIsLoadingConversations(true);
        }
        setError(null);
        const {
          conversations: nextConversations,
          notificationUnreadCount: nextNotificationUnreadCount,
        } = await loadMessagesPageData({
          background: options.silent,
          refresh: true,
        });
        setConversations(nextConversations);
        setNotificationUnreadCount(nextNotificationUnreadCount);
        notificationUnreadCountRef.current = nextNotificationUnreadCount;
        emitMessageBadgeCount(
          getConversationUnreadTotal(nextConversations) +
            nextNotificationUnreadCount,
        );
        setSelectedConversationId((currentId) => {
          const requestedId = getRequestedConversationId(conversationId);
          if (
            requestedId &&
            nextConversations.some(
              (conversation) => String(conversation.id) === requestedId,
            )
          ) {
            return requestedId;
          }

          if (
            currentId !== null &&
            nextConversations.some(
              (conversation) => String(conversation.id) === String(currentId),
            )
          ) {
            return currentId;
          }

          return conversationId ? null : (nextConversations[0]?.id ?? null);
        });
      } catch (loadError) {
        const message =
          loadError instanceof Error ? loadError.message : "会话加载失败";
        setError(message);
        toast.error(message);
      } finally {
        if (!options.silent) {
          setIsLoadingConversations(false);
        }
      }
    },
    [conversationId],
  );

  const loadMessages = useCallback(
    async (
      conversationId: string | number,
      options: { silent?: boolean } = {},
    ) => {
      try {
        const cachedThread = getCachedConversationMessages(conversationId);
        if (cachedThread && !options.silent) {
          setMessages(cachedThread.messages);
          setHasOlderMessages(Boolean(cachedThread.pagination.has_more));
          setNextBefore(cachedThread.pagination.next_before ?? null);
          setIsLoadingMessages(false);
        } else if (!options.silent) {
          setIsLoadingMessages(true);
        }
        const payload = await loadConversationMessagesData(conversationId, {
          background: options.silent,
          limit: messagePageSize,
          refresh: true,
        });
        setMessages(payload.messages);
        setHasOlderMessages(Boolean(payload.pagination.has_more));
        setNextBefore(payload.pagination.next_before ?? null);

        markConversationRead(conversationId, payload.messages);
      } catch (loadError) {
        const message =
          loadError instanceof Error ? loadError.message : "消息加载失败";
        toast.error(message);
      } finally {
        if (!options.silent) {
          setIsLoadingMessages(false);
        }
      }
    },
    [markConversationRead],
  );

  useEffect(() => {
    queueMicrotask(() => {
      const token = getStoredAccessToken();
      setAuthToken(token);
      if (token) {
        if (initialData) {
          setCachedMessagesPageData({
            conversations: initialData.conversations,
            notificationUnreadCount: initialData.notificationUnreadCount,
          });
        } else {
          void loadConversations();
        }
      } else {
        setIsLoadingConversations(false);
      }
    });
  }, [initialData, loadConversations]);

  useEffect(() => {
    selectedConversationIdRef.current = selectedConversationId;
    if (selectedConversationId === null) {
      return;
    }

    if (
      initialData?.thread &&
      String(initialData.selectedConversationId) ===
        String(selectedConversationId)
    ) {
      setCachedConversationMessages(selectedConversationId, initialData.thread);
      markConversationRead(selectedConversationId, initialData.thread.messages);
      return;
    }

    queueMicrotask(() => {
      void loadMessages(selectedConversationId);
    });
  }, [initialData, loadMessages, markConversationRead, selectedConversationId]);

  // Contract marker for readiness: new WebSocket(getImWebSocketUrl(socketToken))
  useImSocketRefresh({
    authToken,
    loadConversations,
    loadMessages,
    selectedConversationIdRef,
    setSocketState,
  });

  async function handleRefresh() {
    await Promise.all([
      loadConversations(),
      selectedConversationId !== null
        ? loadMessages(selectedConversationId)
        : Promise.resolve(),
    ]);
  }

  async function handleLoadOlderMessages() {
    if (!selectedConversationId || !nextBefore || isLoadingOlderMessages) {
      return;
    }

    setIsLoadingOlderMessages(true);
    try {
      const payload = await loadConversationMessagesData(
        selectedConversationId,
        {
          before: nextBefore,
          limit: messagePageSize,
        },
      );
      setMessages((items) => {
        const nextMessages = mergeMessages(payload.messages, items);
        setCachedConversationMessages(selectedConversationId, {
          messages: nextMessages,
          pagination: payload.pagination,
        });
        return nextMessages;
      });
      setHasOlderMessages(Boolean(payload.pagination.has_more));
      setNextBefore(payload.pagination.next_before ?? null);
    } catch (loadError) {
      toast.error(
        loadError instanceof Error ? loadError.message : "历史消息加载失败",
      );
    } finally {
      setIsLoadingOlderMessages(false);
    }
  }

  async function handleSendMessage(
    contentOverride?: string,
    retryMessageId?: string | number,
  ) {
    const content = (contentOverride ?? draft).trim();
    if (!content || !selectedConversationId || isSending) {
      return;
    }

    const optimisticId = retryMessageId ?? `pending-${crypto.randomUUID()}`;
    const optimisticMessage: ImMessage = {
      id: optimisticId,
      conversation_id: selectedConversationId,
      sender_id: viewerId ?? "current-user",
      content,
      client_msg_id: String(optimisticId),
      created_at: new Date().toISOString(),
      status: "pending",
    };

    if (contentOverride === undefined) {
      setDraft("");
    }
    setMessages((items) => {
      const nextMessages = retryMessageId
        ? items.map((item) =>
            String(item.id) === String(retryMessageId)
              ? { ...item, status: "pending" }
              : item,
          )
        : [...items, optimisticMessage];
      setCachedConversationMessages(selectedConversationId, {
        messages: nextMessages,
        pagination: {
          has_more: hasOlderMessages,
          limit: messagePageSize,
          next_before: nextBefore,
        },
      });
      return nextMessages;
    });

    setIsSending(true);
    try {
      const sentMessage = await sendImMessage(selectedConversationId, content);
      setMessages((items) => {
        const nextMessages = mergeMessages(
          items
            .filter((item) => String(item.id) !== String(sentMessage.id))
            .map((item) =>
              String(item.id) === String(optimisticId) ? sentMessage : item,
            ),
          [],
        );
        setCachedConversationMessages(selectedConversationId, {
          messages: nextMessages,
          pagination: {
            has_more: hasOlderMessages,
            limit: messagePageSize,
            next_before: nextBefore,
          },
        });
        return nextMessages;
      });
      void loadConversations({ silent: true });
    } catch (sendError) {
      setMessages((items) => {
        const nextMessages = items.map((item) =>
          String(item.id) === String(optimisticId)
            ? { ...item, status: "failed" }
            : item,
        );
        setCachedConversationMessages(selectedConversationId, {
          messages: nextMessages,
          pagination: {
            has_more: hasOlderMessages,
            limit: messagePageSize,
            next_before: nextBefore,
          },
        });
        return nextMessages;
      });
      toast.error(sendError instanceof Error ? sendError.message : "发送失败");
    } finally {
      setIsSending(false);
    }
  }

  if (authToken === null) {
    return (
      <MessageShell mobileDetail={Boolean(conversationId)}>
        <EmptyState
          actionHref="/login"
          actionLabel="去登录"
          title="需要登录后查看私信"
          description="登录后可以查看会话、发送消息并接收实时提醒。"
        />
      </MessageShell>
    );
  }

  return (
    <MessageShell
      mobileDetail={Boolean(conversationId)}
      mobileContent={
        conversationId ? (
          <MobileMessageDetail
            conversation={selectedConversation}
            messages={messages}
            viewerId={viewerId}
            draft={draft}
            isLoadingMessages={isLoadingMessages}
            isLoadingOlderMessages={isLoadingOlderMessages}
            isSending={isSending}
            hasOlderMessages={hasOlderMessages}
            onDraftChange={setDraft}
            onLoadOlderMessages={handleLoadOlderMessages}
            onSendMessage={handleSendMessage}
            retryLabel={feedT("retry")}
          />
        ) : (
          <MobileMessagesList
            conversations={conversations}
            filteredConversations={filteredConversations}
            query={conversationQuery}
            error={error}
            isLoading={isLoadingConversations}
            notificationUnreadCount={notificationUnreadCount}
            onQueryChange={setConversationQuery}
            onRefresh={handleRefresh}
          />
        )
      }
      actions={
        <>
          <SocketStatus state={socketState} />
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="刷新会话"
            onClick={() => void handleRefresh()}
            disabled={isLoadingConversations || isLoadingMessages}
            className="size-10 border-[var(--message-border)] bg-[var(--message-control)] text-[var(--message-text)] hover:bg-[var(--message-control-hover)]"
          >
            <RefreshCw
              className={cn(
                "size-4",
                (isLoadingConversations || isLoadingMessages) && "animate-spin",
              )}
            />
          </Button>
        </>
      }
    >
      <main className="mx-auto hidden min-h-[calc(100vh-64px)] w-full max-w-[1180px] gap-0 px-4 py-5 md:grid md:grid-cols-[minmax(280px,340px)_minmax(0,1fr)] sm:px-6 lg:px-8">
        <aside className="min-h-[220px] border-[var(--message-border)] lg:border-r lg:pr-4">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-[var(--message-muted)]">
              会话
            </h2>
            {isLoadingConversations ? (
              <Loader2 className="size-4 animate-spin text-[var(--message-subtle)]" />
            ) : null}
          </div>

          <label className="relative mb-3 block">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--message-subtle)]" />
            <input
              value={conversationQuery}
              onChange={(event) => setConversationQuery(event.target.value)}
              placeholder="搜索会话"
              className="h-10 w-full rounded-lg border border-[var(--message-border)] bg-[var(--message-surface)] pl-9 pr-3 text-sm text-[var(--message-text)] outline-none transition placeholder:text-[var(--message-faint)] focus:border-primary"
            />
          </label>

          {error ? (
            <div className="rounded-lg border border-[var(--message-border)] bg-[var(--message-surface)] px-4 py-5 text-sm text-[var(--message-muted)]">
              {error}
            </div>
          ) : conversations.length > 0 ? (
            filteredConversations.length > 0 ? (
              <div className="space-y-2">
                {filteredConversations.map((conversation) => (
                  <ConversationListItem
                    key={conversation.id}
                    conversation={conversation}
                    selected={
                      String(conversation.id) === String(selectedConversationId)
                    }
                    onSelect={() => setSelectedConversationId(conversation.id)}
                  />
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-dashed border-[var(--message-border)] px-4 py-8 text-center">
                <Search className="mx-auto size-8 text-[var(--message-faint)]" />
                <p className="mt-3 text-sm font-semibold text-[var(--message-muted)]">
                  没有匹配会话
                </p>
                <p className="mt-1 text-xs leading-5 text-[var(--message-subtle)]">
                  换个昵称、账号或消息关键词再试。
                </p>
              </div>
            )
          ) : isLoadingConversations ? (
            <div className="rounded-lg border border-[var(--message-border)] bg-[var(--message-surface)] px-4 py-5 text-sm text-[var(--message-subtle)]">
              正在加载会话...
            </div>
          ) : (
            <div className="rounded-lg border border-dashed border-[var(--message-border)] px-4 py-8 text-center">
              <Inbox className="mx-auto size-8 text-[var(--message-faint)]" />
              <p className="mt-3 text-sm font-semibold text-[var(--message-muted)]">
                暂无会话
              </p>
              <p className="mt-1 text-xs leading-5 text-[var(--message-subtle)]">
                从用户主页发起私信后，会话会显示在这里。
              </p>
            </div>
          )}
        </aside>

        <section className="mt-5 flex min-h-[560px] flex-col rounded-lg border border-[var(--message-border)] bg-[var(--message-surface)] lg:ml-4 lg:mt-0">
          {selectedConversation ? (
            <>
              <div className="flex h-16 items-center gap-3 border-b border-[var(--message-border)] px-4">
                <ConversationAvatar conversation={selectedConversation} />
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-sm font-semibold text-[var(--message-text)]">
                    {getConversationTitle(selectedConversation)}
                  </h2>
                  <p className="text-xs text-[var(--message-subtle)]">
                    {selectedConversation.type === "group" ? "群聊" : "私信"}
                  </p>
                </div>
              </div>

              <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4 py-5">
                {isLoadingMessages ? (
                  <div className="flex flex-1 items-center justify-center text-sm text-[var(--message-subtle)]">
                    <Loader2 className="mr-2 size-4 animate-spin" />
                    正在加载消息...
                  </div>
                ) : messages.length > 0 ? (
                  <>
                    {hasOlderMessages || isLoadingOlderMessages ? (
                      <div className="flex justify-center">
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => void handleLoadOlderMessages()}
                          disabled={isLoadingOlderMessages}
                          className="h-8 border-[var(--message-border)] bg-[var(--message-control)] px-3 text-xs text-[var(--message-muted)] hover:bg-[var(--message-control-hover)] hover:text-[var(--message-text)]"
                        >
                          {isLoadingOlderMessages ? (
                            <Loader2 className="size-3.5 animate-spin" />
                          ) : (
                            <ChevronUp className="size-3.5" />
                          )}
                          加载更早消息
                        </Button>
                      </div>
                    ) : null}
                    {messages.map((message) => (
                      <MessageBubble
                        key={message.id}
                        message={message}
                        mine={
                          viewerId !== null &&
                          String(message.sender_id) === viewerId
                        }
                        onRetry={() =>
                          void handleSendMessage(message.content, message.id)
                        }
                        retryLabel={feedT("retry")}
                      />
                    ))}
                  </>
                ) : (
                  <div className="flex flex-1 items-center justify-center">
                    <EmptyState
                      compact
                      title="还没有消息"
                      description="发送第一条消息，开始这个会话。"
                    />
                  </div>
                )}
              </div>

              <div className="border-t border-[var(--message-border)] p-3">
                <div className="flex items-end gap-2 rounded-lg border border-[var(--message-border)] bg-[var(--message-input-bg)] p-2">
                  <textarea
                    value={draft}
                    onChange={(event) => setDraft(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault();
                        void handleSendMessage();
                      }
                    }}
                    rows={1}
                    maxLength={1000}
                    placeholder="输入消息..."
                    className="max-h-32 min-h-10 flex-1 resize-none bg-transparent px-2 py-2 text-sm leading-6 text-[var(--message-text)] outline-none placeholder:text-[var(--message-faint)]"
                  />
                  <Button
                    type="button"
                    size="icon"
                    aria-label="发送消息"
                    onClick={() => void handleSendMessage()}
                    disabled={!draft.trim() || isSending}
                    className="size-10"
                  >
                    {isSending ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Send className="size-4" />
                    )}
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center px-6">
              <EmptyState
                title="选择一个会话"
                description="左侧会话列表加载完成后，选择会话即可查看消息。"
              />
            </div>
          )}
        </section>
      </main>
    </MessageShell>
  );
}
