"use client";

import { MarkdownContent } from "@/components/markdown-content";
import { Avatar,AvatarFallback,AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { prefetchConversationMessages } from "@/lib/im-cache";
import type { ImConversation,ImMessage,ImUser } from "@/lib/types";
import { getUserHref } from "@/lib/users";
import { cn } from "@/lib/utils";
import {
Loader2,
MessageCircle,
RefreshCw,
Wifi,
WifiOff
} from "lucide-react";
import Link from "next/link";
import type { SocketState } from "./messages-page-view-types";

export function ConversationListItem({
  conversation,
  selected,
  onSelect,
}: {
  conversation: ImConversation;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      onFocus={() => void prefetchConversationMessages(conversation.id)}
      onMouseEnter={() => void prefetchConversationMessages(conversation.id)}
      className={cn(
        "flex w-full items-center gap-3 rounded-lg border px-3 py-3 text-left transition-colors",
        selected
          ? "border-primary/70 bg-primary/10"
          : "border-[var(--message-border)] bg-[var(--message-surface)] hover:bg-[var(--message-control)]",
      )}
    >
      <ConversationAvatar conversation={conversation} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-semibold text-[var(--message-text)]">
            {getConversationTitle(conversation)}
          </p>
          {conversation.unread_count ? (
            <span className="min-w-4 rounded-full bg-primary px-1 text-center text-[10px] font-bold leading-4 text-white">
              {conversation.unread_count > 99 ? "99+" : conversation.unread_count}
            </span>
          ) : null}
        </div>
        <p className="mt-1 truncate text-xs text-[var(--message-muted)]">
          {conversation.last_message?.content || "暂无消息"}
        </p>
      </div>
      <span className="shrink-0 text-[11px] text-[var(--message-subtle)]">
        {formatDateTime(conversation.last_message?.created_at ?? conversation.updated_at)}
      </span>
    </button>
  );
}

export function ConversationAvatar({
  conversation,
  mobile,
  mobileLarge,
  mobileSmall,
}: {
  conversation: ImConversation;
  mobile?: boolean;
  mobileLarge?: boolean;
  mobileSmall?: boolean;
}) {
  const member = conversation.members?.[0];
  const title = getConversationTitle(conversation);

  return (
    <Avatar
      className={cn(
        "border-[var(--message-border)] bg-[var(--message-other-bubble)]",
        mobileLarge
          ? "size-[clamp(56px,17vw,76px)]"
          : mobileSmall
            ? "size-[clamp(40px,12vw,58px)]"
            : mobile
              ? "size-[46px]"
            : "size-11",
      )}
    >
      <AvatarImage src={member?.avatar ?? undefined} alt={title} />
      <AvatarFallback
        className={cn(
          "bg-[var(--message-other-bubble)] font-bold text-[var(--message-muted)]",
          mobile || mobileLarge || mobileSmall ? "text-[clamp(16px,5vw,20px)]" : "text-sm",
        )}
      >
        {title.charAt(0).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
}

export function MessageBubble({
  message,
  mine,
  onRetry,
  retryLabel,
}: {
  message: ImMessage;
  mine: boolean;
  onRetry: () => void;
  retryLabel: string;
}) {
  return (
    <div className={cn("flex", mine ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[82%] rounded-lg px-3 py-2 sm:max-w-[68%]",
          mine
            ? "bg-[var(--message-mine-bubble)] text-[var(--message-mine-text)]"
            : "bg-[var(--message-other-bubble)] text-[var(--message-other-text)]",
        )}
      >
        <MarkdownContent className="markdown-content-compact break-words text-sm leading-6" content={message.content} />
        <div
          className={cn(
            "mt-1 flex items-center gap-2 text-[11px]",
            mine ? "justify-end text-[var(--message-mine-meta)]" : "text-[var(--message-subtle)]",
          )}
        >
          <span>{formatDateTime(message.created_at)}</span>
          {mine ? (
            message.status === "pending" ? (
              <Loader2 className="size-3 animate-spin" />
            ) : message.status === "failed" ? (
              <button
                type="button"
                aria-label={retryLabel}
                title={retryLabel}
                onClick={onRetry}
                className="rounded p-0.5 text-red-300 transition hover:bg-white/10"
              >
                <RefreshCw className="size-3" />
              </button>
            ) : (
              <span>{message.status === "read" ? "已读" : "已发送"}</span>
            )
          ) : null}
        </div>
      </div>
    </div>
  );
}

export function EmptyState({
  actionHref,
  actionLabel,
  compact,
  description,
  title,
}: {
  actionHref?: string;
  actionLabel?: string;
  compact?: boolean;
  description: string;
  title: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center px-6 text-center",
        compact ? "py-10" : "min-h-[52vh]",
      )}
    >
      <MessageCircle className="size-10 text-[var(--message-faint)]" />
      <h2 className="mt-4 text-base font-semibold text-[var(--message-text)]">{title}</h2>
      <p className="mt-2 max-w-[360px] text-sm leading-6 text-[var(--message-muted)]">{description}</p>
      {actionHref && actionLabel ? (
        <Button asChild className="mt-5 h-10 px-5">
          <Link href={actionHref}>{actionLabel}</Link>
        </Button>
      ) : null}
    </div>
  );
}

export function SocketStatus({ state }: { state: SocketState }) {
  const connected = state === "open";

  return (
    <div
      className={cn(
        "hidden h-10 items-center gap-2 rounded-full border px-3 text-xs font-semibold sm:flex",
        connected
          ? "border-emerald-400/20 bg-emerald-400/10 text-emerald-200"
          : "border-[var(--message-border)] bg-[var(--message-control)] text-[var(--message-muted)]",
      )}
    >
      {connected ? <Wifi className="size-4" /> : <WifiOff className="size-4" />}
      {connected ? "实时连接" : state === "connecting" ? "连接中" : "未连接"}
    </div>
  );
}

export function getConversationTitle(conversation: ImConversation) {
  if (conversation.name) {
    return conversation.name;
  }

  const memberNames = (conversation.members ?? [])
    .map((member) => getUserDisplayName(member))
    .filter(Boolean);

  if (memberNames.length > 0) {
    return memberNames.join("、");
  }

  if (conversation.member_ids && conversation.member_ids.length > 1) {
    return `成员 ${conversation.member_ids.join("、")}`;
  }

  return `会话 ${conversation.id}`;
}

export function getConversationUserHref(conversation: ImConversation) {
  if (conversation.type === "group") {
    return null;
  }

  const member = conversation.members?.[0];
  const userId =
    normalizeUserIdentifier(member?.username) ??
    normalizeUserIdentifier(member?.id) ??
    normalizeUserIdentifier(conversation.member_ids?.[0]);

  return userId ? getUserHref(userId) : null;
}

export function normalizeUserIdentifier(value: number | string | null | undefined) {
  if (value === undefined || value === null) {
    return null;
  }

  const normalized = String(value).trim();
  return normalized || null;
}

export function filterConversations(conversations: ImConversation[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return conversations;
  }

  return conversations.filter((conversation) => {
    const searchable = [
      getConversationTitle(conversation),
      conversation.last_message?.content,
      conversation.type,
      ...(conversation.member_ids ?? []).map(String),
      ...(conversation.members ?? []).flatMap((member) => [
        member.id,
        member.nickname,
        member.username,
      ]),
    ]
      .filter((value) => value !== undefined && value !== null)
      .join(" ")
      .toLowerCase();

    return searchable.includes(normalizedQuery);
  });
}

export function mergeMessages(previousMessages: ImMessage[], currentMessages: ImMessage[]) {
  const seen = new Set<string>();
  const merged: ImMessage[] = [];

  for (const message of [...previousMessages, ...currentMessages]) {
    const key = String(message.id);
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    merged.push(message);
  }

  return merged;
}

export function getUserDisplayName(user: ImUser) {
  return user.nickname ?? user.username ?? (user.id ? `用户 ${user.id}` : "");
}

export function resolveInitialSelectedConversationId(
  conversationId: string | number | undefined,
  conversations: ImConversation[],
) {
  const requestedId = getRequestedConversationId(conversationId);
  if (requestedId) {
    return conversationId ?? requestedId;
  }

  return conversations[0]?.id ?? null;
}

export function getRequestedConversationId(conversationId?: string | number) {
  if (conversationId !== undefined && conversationId !== null) {
    return String(conversationId);
  }

  if (typeof window === "undefined") {
    return null;
  }

  return new URLSearchParams(window.location.search).get("conversation");
}

export function formatDateTime(value?: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function formatMessageTime(value?: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function formatConversationDate(value?: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const startOfDate = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
  const dayDistance = Math.round((startOfToday - startOfDate) / 86400000);

  if (dayDistance === 0) {
    return formatMessageTime(value);
  }

  if (dayDistance === 1) {
    return "昨天";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
  }).format(date);
}
