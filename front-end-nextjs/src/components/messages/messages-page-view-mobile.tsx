"use client";

import { MarkdownContent } from "@/components/markdown-content";
import { Avatar,AvatarFallback,AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { getStoredUser } from "@/lib/api";
import { prefetchConversationMessages } from "@/lib/im-cache";
import type { ImConversation,ImMessage } from "@/lib/types";
import { cn } from "@/lib/utils";
import {
ArrowLeft,
Bell,
CheckCheck,
ChevronRight,
ChevronUp,
Inbox,
Loader2,
MessageCircle,
RefreshCw,
Search,
Send
} from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { ConversationAvatar,EmptyState,formatConversationDate,formatMessageTime,getConversationTitle,getConversationUserHref } from "./messages-page-view-desktop";

export function MessageShell({
  actions,
  children,
  mobileContent,
  mobileDetail,
}: {
  actions?: ReactNode;
  children: ReactNode;
  mobileContent?: ReactNode;
  mobileDetail?: boolean;
}) {
  return (
    <div className="message-page theme-adaptive min-h-screen bg-[var(--message-bg)] text-[var(--message-text)]">
      <header
        className={cn(
          "sticky top-0 z-30 border-b border-[var(--message-border)] bg-[var(--message-header)] backdrop-blur md:block",
          mobileDetail ? "hidden" : "block",
        )}
      >
        <div className="mx-auto flex h-16 w-full max-w-[1180px] items-center gap-3 px-4 sm:px-6 lg:px-8">
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label="返回信息流"
            className="size-10 text-[var(--message-text)] hover:bg-[var(--message-control-hover)]"
          >
            <Link href="/">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <div className="min-w-0 flex-1 text-center">
            <h1 className="truncate text-[24px] font-black leading-none tracking-normal md:text-lg md:font-semibold">
              消息
            </h1>
            <p className="hidden text-xs text-[var(--message-subtle)] md:block">私信会话与实时消息</p>
          </div>
          <Button
            asChild
            variant="outline"
            className="hidden h-10 border-[var(--message-border)] bg-[var(--message-control)] px-4 text-[var(--message-text)] hover:bg-[var(--message-control-hover)] sm:inline-flex"
          >
            <Link href="/notifications">
              <Inbox className="size-4" />
              通知
            </Link>
          </Button>
          {actions}
        </div>
      </header>
      {mobileContent}
      {children}
    </div>
  );
}

export function MobileMessagesList({
  conversations,
  error,
  filteredConversations,
  isLoading,
  notificationUnreadCount,
  onQueryChange,
  onRefresh,
  query,
}: {
  conversations: ImConversation[];
  error: string | null;
  filteredConversations: ImConversation[];
  isLoading: boolean;
  notificationUnreadCount: number;
  onQueryChange: (value: string) => void;
  onRefresh: () => void | Promise<void>;
  query: string;
}) {
  return (
    <main className="flex min-h-[calc(100dvh-64px)] flex-col bg-[var(--message-bg)] pb-3 md:hidden">
      <div className="px-4 pb-2 pt-3">
        <label className="relative block">
          <Search className="pointer-events-none absolute left-4 top-1/2 size-4 -translate-y-1/2 text-[var(--message-faint)]" />
          <input
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="搜索会话"
            className="h-10 w-full rounded-full border-0 bg-[var(--message-control)] pl-10 pr-4 text-sm text-[var(--message-text)] outline-none placeholder:text-[var(--message-faint)]"
          />
        </label>
      </div>

      <section className="px-4 pb-1 pt-0">
        <Link
          href="/notifications"
          className="group grid min-h-16 grid-cols-[44px_minmax(0,1fr)_18px] items-center gap-3 rounded-2xl bg-[var(--message-control)] px-1"
        >
          <span className="relative flex size-11 items-center justify-center rounded-full bg-[var(--message-accent-soft)] text-[var(--message-accent-text)] shadow-[0_0_24px_rgba(255,36,92,0.1)]">
            <Bell className="size-[22px]" strokeWidth={2.4} />
            {notificationUnreadCount > 0 ? (
              <span className="absolute -right-0.5 -top-0.5 min-w-4 rounded-full bg-primary px-1 text-center text-[10px] font-black leading-4 text-white ring-2 ring-[var(--message-bg)]">
                {notificationUnreadCount > 99 ? "99+" : notificationUnreadCount}
              </span>
            ) : null}
          </span>
          <span className="min-w-0 truncate text-lg font-black leading-none tracking-normal text-[var(--message-text)]">
            消息通知
          </span>
          <ChevronRight className="size-5 text-[var(--message-faint)] transition group-active:translate-x-0.5" />
        </Link>
      </section>

      <section className="min-h-0 flex-1 overflow-y-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {error ? (
          <MobileListNotice title="会话加载失败" description={error} actionLabel="重试" onAction={onRefresh} />
        ) : conversations.length > 0 ? (
          filteredConversations.length > 0 ? (
            <div className="pt-1">
              {filteredConversations.map((conversation) => (
                <MobileConversationListItem key={conversation.id} conversation={conversation} />
              ))}
            </div>
          ) : (
            <MobileListNotice title="没有匹配会话" description="换个昵称、账号或消息关键词再试。" />
          )
        ) : isLoading ? (
          <MobileListNotice
            loading
            title="正在加载会话"
            description="稍等一下，消息列表马上回来。"
          />
        ) : (
          <MobileListNotice title="暂无会话" description="从用户主页发起私信后，会话会显示在这里。" />
        )}
      </section>

    </main>
  );
}

export function MobileConversationListItem({ conversation }: { conversation: ImConversation }) {
  return (
    <Link
      href={`/messages/${conversation.id}`}
      onFocus={() => void prefetchConversationMessages(conversation.id)}
      onMouseEnter={() => void prefetchConversationMessages(conversation.id)}
      onTouchStart={() => void prefetchConversationMessages(conversation.id)}
      className="grid min-h-[76px] grid-cols-[46px_minmax(0,1fr)] gap-3 px-4"
    >
      <div className="flex items-start pt-3.5">
        <ConversationAvatar conversation={conversation} mobile />
      </div>
      <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] border-b border-[var(--message-border)] py-3.5">
        <div className="min-w-0 self-center">
          <div className="flex min-w-0 items-center gap-2">
            <p className="truncate text-[20px] font-black leading-tight tracking-normal text-[var(--message-text)]">
              {getConversationTitle(conversation)}
            </p>
            {conversation.unread_count ? (
              <span className="min-w-5 rounded-full bg-primary px-1.5 text-center text-[11px] font-bold leading-5 text-white">
                {conversation.unread_count > 99 ? "99+" : conversation.unread_count}
              </span>
            ) : null}
          </div>
          <p className="mt-1 truncate text-[15px] font-semibold leading-tight text-[var(--message-subtle)]">
            {conversation.last_message?.content || "暂无消息"}
          </p>
        </div>
        <span className="ml-2 shrink-0 pt-1 text-sm font-semibold leading-none text-[var(--message-faint)]">
          {formatConversationDate(conversation.last_message?.created_at ?? conversation.updated_at)}
        </span>
      </div>
    </Link>
  );
}

export function MobileMessageDetail({
  conversation,
  draft,
  hasOlderMessages,
  isLoadingMessages,
  isLoadingOlderMessages,
  isSending,
  messages,
  onDraftChange,
  onLoadOlderMessages,
  onSendMessage,
  retryLabel,
  viewerId,
}: {
  conversation: ImConversation | null;
  draft: string;
  hasOlderMessages: boolean;
  isLoadingMessages: boolean;
  isLoadingOlderMessages: boolean;
  isSending: boolean;
  messages: ImMessage[];
  onDraftChange: (value: string) => void;
  onLoadOlderMessages: () => void | Promise<void>;
  onSendMessage: (contentOverride?: string, retryMessageId?: string | number) => void | Promise<void>;
  retryLabel: string;
  viewerId: string | null;
}) {
  const conversationTitle = conversation ? getConversationTitle(conversation) : "消息";
  const conversationUserHref = conversation ? getConversationUserHref(conversation) : null;

  return (
    <main className="theme-adaptive flex h-dvh flex-col overflow-hidden bg-[var(--message-bg)] text-[var(--message-text)] md:hidden">
      <header className="grid h-[clamp(76px,24vw,104px)] shrink-0 grid-cols-[44px_clamp(56px,18vw,92px)_minmax(0,1fr)] items-center gap-2 px-4 pt-3 sm:px-5 sm:pt-4">
        <Button
          asChild
          variant="ghost"
          size="icon"
          aria-label="返回消息列表"
          className="size-11 justify-self-start text-[var(--message-text)] hover:bg-[var(--message-control-hover)] sm:size-12"
        >
          <Link href="/messages">
            <ArrowLeft className="size-7 sm:size-9" strokeWidth={2.6} />
          </Link>
        </Button>
        {conversation ? (
          conversationUserHref ? (
            <Link
              href={conversationUserHref}
              aria-label={`Open ${conversationTitle} profile`}
              className="rounded-full outline-none transition hover:opacity-85 focus-visible:ring-2 focus-visible:ring-[var(--message-focus-ring)]"
            >
              <ConversationAvatar conversation={conversation} mobileLarge />
            </Link>
          ) : (
            <ConversationAvatar conversation={conversation} mobileLarge />
          )
        ) : (
          <Avatar className="size-[clamp(56px,17vw,76px)] bg-[var(--message-other-bubble)]" />
        )}
        {conversationUserHref ? (
          <Link
            href={conversationUserHref}
            aria-label={`Open ${conversationTitle} profile`}
            className="min-w-0 truncate rounded outline-none transition hover:text-[var(--message-accent-text)] focus-visible:ring-2 focus-visible:ring-[var(--message-focus-ring)]"
          >
            <h1 className="truncate text-[clamp(22px,7vw,34px)] font-black leading-none tracking-normal text-[var(--message-text)]">
              {conversationTitle}
            </h1>
          </Link>
        ) : (
          <h1 className="min-w-0 truncate text-[clamp(22px,7vw,34px)] font-black leading-none tracking-normal text-[var(--message-text)]">
            {conversationTitle}
          </h1>
        )}
      </header>

      <section className="min-h-0 flex-1 overflow-y-auto px-4 pb-4 pt-4 [scrollbar-width:none] sm:px-6 sm:pb-5 sm:pt-5 [&::-webkit-scrollbar]:hidden">
        {isLoadingMessages ? (
          <div className="flex h-full items-center justify-center text-sm text-[var(--message-subtle)]">
            <Loader2 className="mr-2 size-4 animate-spin" />
            正在加载消息...
          </div>
        ) : messages.length > 0 ? (
          <div className="flex flex-col gap-4 sm:gap-7">
            {hasOlderMessages || isLoadingOlderMessages ? (
              <div className="flex justify-center">
                <button
                  type="button"
                  onClick={() => void onLoadOlderMessages()}
                  disabled={isLoadingOlderMessages}
                  className="inline-flex h-9 items-center gap-2 rounded-full bg-[var(--message-control)] px-4 text-xs font-semibold text-[var(--message-muted)] disabled:opacity-60"
                >
                  {isLoadingOlderMessages ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <ChevronUp className="size-3.5" />
                  )}
                  加载更早消息
                </button>
              </div>
            ) : null}
            {messages.map((message) => {
              const mine = viewerId !== null && String(message.sender_id) === viewerId;
              return (
                <MobileMessageBubble
                  key={message.id}
                  conversation={conversation}
                  message={message}
                  mine={mine}
                  onRetry={() => void onSendMessage(message.content, message.id)}
                  retryLabel={retryLabel}
                />
              );
            })}
          </div>
        ) : (
          <div className="flex h-full items-center justify-center">
            <EmptyState compact title="还没有消息" description="发送第一条消息，开始这个会话。" />
          </div>
        )}
      </section>

      <footer className="shrink-0 border-t border-[var(--message-border)] bg-[var(--message-footer)] px-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 sm:px-4">
        <div className="flex min-h-[clamp(48px,14vw,56px)] items-end rounded-full bg-[var(--message-input-bg)] px-4 sm:px-5">
          <textarea
            value={draft}
            onChange={(event) => onDraftChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                void onSendMessage();
              }
            }}
            rows={1}
            maxLength={1000}
            placeholder="输入消息..."
            className="max-h-28 min-h-[clamp(48px,14vw,56px)] flex-1 resize-none bg-transparent py-3 text-[clamp(15px,4.6vw,22px)] font-semibold leading-7 text-[var(--message-text)] outline-none placeholder:text-[var(--message-faint)] sm:leading-8"
          />
          <button
            type="button"
            aria-label="发送消息"
            onClick={() => void onSendMessage()}
            disabled={!draft.trim() || isSending}
            className="mb-2 ml-2 flex size-9 shrink-0 items-center justify-center rounded-full bg-[var(--message-mine-bubble)] text-[var(--message-mine-text)] transition disabled:bg-transparent disabled:text-[var(--message-faint)] sm:size-10"
          >
            {isSending ? <Loader2 className="size-5 animate-spin" /> : <Send className="size-5" />}
          </button>
        </div>
      </footer>
    </main>
  );
}

export function MobileMessageBubble({
  conversation,
  message,
  mine,
  onRetry,
  retryLabel,
}: {
  conversation: ImConversation | null;
  message: ImMessage;
  mine: boolean;
  onRetry: () => void;
  retryLabel: string;
}) {
  const viewer = typeof window === "undefined" ? null : getStoredUser();
  const title = conversation ? getConversationTitle(conversation) : "用户";

  if (mine) {
    return (
      <div className="flex justify-end gap-2 pl-10 sm:gap-4 sm:pl-14">
        <div className="flex min-w-0 flex-col items-end">
          <div className="max-w-[72vw] rounded-[16px] rounded-tr-[3px] bg-[var(--message-mine-bubble)] px-4 py-3 text-[clamp(15px,4.6vw,22px)] font-bold leading-snug text-[var(--message-mine-text)] sm:max-w-[68vw] sm:px-7 sm:py-4">
            <MarkdownContent className="markdown-content-compact break-words" content={message.content} />
          </div>
          <span className="mt-2 text-[clamp(12px,3.8vw,18px)] font-semibold leading-none text-[var(--message-faint)]">
            {formatMessageTime(message.created_at)}
          </span>
          {message.status === "pending" ? (
            <Loader2 className="mt-3 size-5 animate-spin text-[var(--message-faint)] sm:mt-4 sm:size-6" />
          ) : message.status === "failed" ? (
            <button
              type="button"
              aria-label={retryLabel}
              title={retryLabel}
              onClick={onRetry}
              className="mt-3 rounded-full p-1 text-red-400 transition hover:bg-red-500/10 sm:mt-4"
            >
              <RefreshCw className="size-5 sm:size-6" strokeWidth={2.7} />
            </button>
          ) : (
            <CheckCheck className="mt-3 size-5 text-[var(--message-accent-text)] sm:mt-4 sm:size-6" strokeWidth={2.7} />
          )}
        </div>
        <Avatar className="size-[clamp(40px,12vw,58px)] shrink-0 bg-[var(--message-other-bubble)]">
          <AvatarImage src={viewer?.avatar ?? undefined} alt={viewer?.nickname ?? "我"} />
          <AvatarFallback className="bg-[var(--message-other-bubble)] text-lg font-bold text-[var(--message-muted)]">
            {(viewer?.nickname ?? "我").charAt(0).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      </div>
    );
  }

  return (
    <div className="flex justify-start gap-2 pr-10 sm:gap-4 sm:pr-14">
      {conversation ? (
        <ConversationAvatar conversation={conversation} mobileSmall />
      ) : (
        <Avatar className="size-[clamp(40px,12vw,58px)] shrink-0 bg-[var(--message-other-bubble)]" />
      )}
      <div className="flex min-w-0 flex-col items-start">
        <div className="max-w-[72vw] rounded-[16px] rounded-tl-[3px] bg-[var(--message-other-bubble)] px-4 py-3 text-[clamp(15px,4.6vw,22px)] font-bold leading-snug text-[var(--message-other-text)] sm:max-w-[68vw] sm:px-7 sm:py-4">
          <MarkdownContent className="markdown-content-compact break-words" content={message.content || title} />
        </div>
        <span className="mt-2 text-[clamp(12px,3.8vw,18px)] font-semibold leading-none text-[var(--message-faint)]">
          {formatMessageTime(message.created_at)}
        </span>
      </div>
    </div>
  );
}

export function MobileListNotice({
  actionLabel,
  description,
  loading,
  onAction,
  title,
}: {
  actionLabel?: string;
  description: string;
  loading?: boolean;
  onAction?: () => void | Promise<void>;
  title: string;
}) {
  return (
    <div className="flex min-h-[42vh] flex-col items-center justify-center px-8 text-center">
      {loading ? (
        <Loader2 className="size-8 animate-spin text-[var(--message-faint)]" />
      ) : (
        <MessageCircle className="size-9 text-[var(--message-faint)]" />
      )}
      <h2 className="mt-4 text-lg font-bold text-[var(--message-text)]">{title}</h2>
      <p className="mt-2 text-sm leading-6 text-[var(--message-muted)]">{description}</p>
      {actionLabel && onAction ? (
        <button
          type="button"
          onClick={() => void onAction()}
          className="mt-5 rounded-full bg-[var(--message-control)] px-5 py-2 text-sm font-bold text-[var(--message-text)]"
        >
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}
