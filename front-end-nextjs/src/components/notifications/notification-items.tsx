"use client";

import { MarkdownContent } from "@/components/markdown-content";
import { Avatar,AvatarFallback,AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { richTextToPlainText } from "@/lib/rich-text";
import type {
GiftCardRedemptionNotification,
NotificationItem,
SystemNotificationItem
} from "@/lib/types";
import { getUserHref } from "@/lib/users";
import { cn } from "@/lib/utils";
import {
Bookmark,
CheckCheck,
Copy,
Gift as GiftIcon,
Heart,
Inbox,
MessageCircle,
RefreshCw,
ShieldCheck,
Star,
Trash2,
UserPlus,
X
} from "lucide-react";
import { useLocale,useTranslations } from "next-intl";
import Image from "next/image";
import Link from "next/link";
import { toast } from "sonner";
import { notificationTabs,notificationTypeGiftCardRedeemed,type NotificationFeedItem,type NotificationTab } from "./notification-model";

export function UserNotificationCard({
  acting,
  notification,
  onDelete,
  onMarkRead,
  onOpen,
}: {
  acting: boolean;
  notification: NotificationItem;
  onDelete: (notification: NotificationItem) => void | Promise<void>;
  onMarkRead: (notification: NotificationItem) => void | Promise<void>;
  onOpen: (notification: NotificationItem) => void | Promise<void>;
}) {
  const t = useTranslations();
  const meta = getUserNotificationMeta(notification, t);
  const postHref = getNotificationPostHref(notification);
  const senderHref = getNotificationSenderHref(notification);
  const content = (
    <>
      <p className="mt-1 truncate text-[15px] font-semibold leading-tight text-white/38 md:text-sm md:text-white/55">
        {meta.actionText}
      </p>
      {meta.detail ? (
        <p className="mt-1 line-clamp-1 text-[13px] font-medium leading-tight text-white/28 md:text-xs">
          {meta.detail}
        </p>
      ) : null}
    </>
  );

  return (
    <article
      className={cn(
        "grid min-h-[86px] grid-cols-[46px_minmax(0,1fr)] gap-3 px-4 md:min-h-0 md:grid-cols-[52px_minmax(0,1fr)] md:rounded-lg md:border md:border-white/[0.08] md:bg-[#181818] md:px-4 md:py-4",
        isOpenableNotification(notification) && "cursor-pointer transition hover:bg-white/[0.04]",
        !notification.is_read && "bg-white/[0.02] md:ring-1 md:ring-[#ff7ba6]/25",
      )}
      onClick={() => {
        if (isOpenableNotification(notification)) {
          void onOpen(notification);
        }
      }}
    >
      <div className="flex items-start pt-3.5 md:pt-0">
        <NotificationAvatar
          avatar={notification.sender?.avatar}
          href={senderHref}
          icon={meta.icon}
          initial={meta.senderInitial}
          label={meta.senderName}
        />
      </div>

      <div className="min-w-0 border-b border-white/[0.07] py-3.5 md:border-b-0 md:py-0">
        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] gap-3">
          <div className="min-w-0 self-center">
            <div className="flex min-w-0 items-center gap-2">
              {senderHref ? (
                <Link
                  href={senderHref}
                  aria-label={`Open ${meta.senderName} profile`}
                  className="min-w-0 truncate rounded outline-none transition hover:text-[#ff9abb] focus-visible:ring-2 focus-visible:ring-[#ff7ba6]/60"
                >
                  <h2 className="truncate text-[20px] font-black leading-tight tracking-normal text-white md:text-base">
                    {meta.senderName}
                  </h2>
                </Link>
              ) : (
                <h2 className="min-w-0 truncate text-[20px] font-black leading-tight tracking-normal text-white md:text-base">
                  {meta.senderName}
                </h2>
              )}
              {!notification.is_read ? (
                <span className="min-w-5 rounded-full bg-[#ff2f69] px-1.5 text-center text-[10px] font-black leading-5 text-white">
                  未读
                </span>
              ) : null}
            </div>
            {postHref ? (
              <Link
                href={postHref}
                aria-label={`Open post ${notification.post_title ?? notification.target_id}`}
                className="block min-w-0 rounded outline-none transition hover:text-[#ff9abb] focus-visible:ring-2 focus-visible:ring-[#ff7ba6]/60"
              >
                {content}
              </Link>
            ) : (
              <div className="min-w-0">{content}</div>
            )}
          </div>

          <div className="flex min-w-[52px] shrink-0 flex-col items-end gap-2">
            <span className="whitespace-nowrap pt-1 text-sm font-semibold leading-none text-white/26 md:text-xs">
              {formatShortDate(notification.created_at)}
            </span>
            {notification.post_cover ? (
              postHref ? (
                <Link
                  href={postHref}
                  aria-label={`Open post ${notification.post_title ?? notification.target_id}`}
                  className="rounded-md outline-none transition hover:opacity-85 focus-visible:ring-2 focus-visible:ring-[#ff7ba6]/60"
                >
                  <Image
                    src={notification.post_cover}
                    alt={notification.post_title ?? "通知封面"}
                    width={56}
                    height={56}
                    className="size-[clamp(46px,13vw,56px)] rounded-md object-cover md:size-14"
                  />
                </Link>
              ) : (
                <Image
                  src={notification.post_cover}
                  alt={notification.post_title ?? "通知封面"}
                  width={56}
                  height={56}
                  className="size-[clamp(46px,13vw,56px)] rounded-md object-cover md:size-14"
                />
              )
            ) : null}
            <div className="flex gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="标记已读"
                onClick={(event) => {
                  event.stopPropagation();
                  void onMarkRead(notification);
                }}
                disabled={notification.is_read || acting}
                className="size-7 text-white/32 hover:bg-white/[0.06] hover:text-white md:size-8"
              >
                <CheckCheck className="size-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="删除通知"
                onClick={(event) => {
                  event.stopPropagation();
                  void onDelete(notification);
                }}
                disabled={acting}
                className="size-7 text-white/32 hover:bg-white/[0.06] hover:text-white md:size-8"
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </article>
  );
}

export function NotificationAvatar({
  avatar,
  href,
  icon: Icon,
  initial,
  label,
}: {
  avatar?: string | null;
  href: string | null;
  icon: typeof Heart;
  initial: string;
  label: string;
}) {
  const avatarElement = (
    <Avatar className="size-[46px] bg-[#451626] md:size-[52px]">
      <AvatarImage src={avatar ?? undefined} alt={label} />
      <AvatarFallback className="bg-[#451626] text-[#ff7ba6]">
        {initial ? (
          <span className="text-[18px] font-black md:text-xl">{initial}</span>
        ) : (
          <Icon className="size-6 md:size-7" />
        )}
      </AvatarFallback>
    </Avatar>
  );

  if (!href) {
    return avatarElement;
  }

  return (
    <Link
      href={href}
      aria-label={`Open ${label} profile`}
      className="block rounded-full outline-none transition hover:opacity-85 focus-visible:ring-2 focus-visible:ring-[#ff7ba6]/60"
    >
      {avatarElement}
    </Link>
  );
}

export function SystemNotificationCard({
  acting,
  notification,
  onConfirm,
  onDismiss,
  onOpen,
}: {
  acting: boolean;
  notification: SystemNotificationItem;
  onConfirm: (notification: SystemNotificationItem) => void | Promise<void>;
  onDismiss: (notification: SystemNotificationItem) => void | Promise<void>;
  onOpen: (notification: SystemNotificationItem) => void;
}) {
  return (
    <article
      className="grid min-h-[86px] cursor-pointer grid-cols-[46px_minmax(0,1fr)] gap-3 px-4 transition hover:bg-white/[0.04] md:min-h-0 md:grid-cols-[52px_minmax(0,1fr)] md:rounded-lg md:border md:border-white/[0.08] md:bg-[#181818] md:px-4 md:py-4"
      onClick={() => onOpen(notification)}
    >
      <div className="flex items-start pt-3.5 md:pt-0">
        <span className="flex size-[46px] items-center justify-center rounded-full bg-white/[0.06] text-white/64 md:size-[52px]">
          <Inbox className="size-6 md:size-7" />
        </span>
      </div>

      <div className="min-w-0 border-b border-white/[0.07] py-3.5 md:border-b-0 md:py-0">
        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] gap-3">
          <div className="min-w-0 self-center">
            <div className="flex min-w-0 items-center gap-2">
              <h2 className="truncate text-[20px] font-black leading-tight tracking-normal text-white md:text-base">
                {notification.title}
              </h2>
              {notification.is_read ? (
                <span className="rounded-full bg-white/[0.06] px-2 py-0.5 text-[10px] font-bold text-white/42">
                  已确认
                </span>
              ) : null}
            </div>
            <NotificationRichText
              content={notification.content || "系统通知"}
              className="mt-1 line-clamp-2 text-[15px] font-semibold leading-snug text-white/38 md:text-sm md:text-white/55"
            />
          </div>

          <div className="flex min-w-[52px] shrink-0 flex-col items-end gap-2">
            <span className="whitespace-nowrap pt-1 text-sm font-semibold leading-none text-white/26 md:text-xs">
              {formatShortDate(notification.created_at)}
            </span>
            <div className="flex gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="确认系统通知"
                onClick={(event) => {
                  event.stopPropagation();
                  void onConfirm(notification);
                }}
                disabled={Boolean(notification.is_read) || acting}
                className="size-7 text-white/32 hover:bg-white/[0.06] hover:text-white md:size-8"
              >
                <CheckCheck className="size-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="移除系统通知"
                onClick={(event) => {
                  event.stopPropagation();
                  void onDismiss(notification);
                }}
                disabled={acting}
                className="size-7 text-white/32 hover:bg-white/[0.06] hover:text-white md:size-8"
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </article>
  );
}

export function NotificationDetailDialog({
  notification,
  onClose,
}: {
  notification: NotificationItem | SystemNotificationItem;
  onClose: () => void;
}) {
  const t = useTranslations();
  const content = notificationDetailContent(notification);
  const giftCard = "gift_card_redemption" in notification ? notification.gift_card_redemption : null;
  const title = giftCard
    ? t("notification.giftCardRedeemed.title", { name: giftCard.product_name })
    : translateNotificationTitle(notification.title, t);
  return (
    <div className="fixed inset-0 z-50 grid place-items-end bg-black/60 px-0 md:place-items-center md:px-4">
      <button type="button" aria-label={t("notification.actions.closeOverlay")} className="absolute inset-0" onClick={onClose} />
      <section className="relative flex max-h-[82dvh] w-full flex-col overflow-hidden rounded-t-lg border border-white/[0.08] bg-[#181818] text-white shadow-2xl md:max-w-[520px] md:rounded-lg">
        <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-white/[0.08] px-4">
          <div className="min-w-0">
            <h2 className="truncate text-base font-black tracking-normal">{title}</h2>
            <p className="mt-0.5 text-xs font-semibold text-white/32">{formatShortDate(notification.created_at)}</p>
          </div>
          <Button type="button" variant="ghost" size="icon" onClick={onClose} aria-label={t("notification.actions.closeDetail")} className="size-9 shrink-0 text-white/55 hover:bg-white/[0.08] hover:text-white">
            <X className="size-4" />
          </Button>
        </div>
        <div className="min-h-0 overflow-y-auto px-4 py-4">
          {giftCard ? (
            <GiftCardNotificationDetail detail={giftCard} />
          ) : (
            <p className="whitespace-pre-line text-sm font-semibold leading-7 text-white/72">{content}</p>
          )}
        </div>
      </section>
    </div>
  );
}

export function GiftCardNotificationDetail({ detail }: { detail: GiftCardRedemptionNotification }) {
  const t = useTranslations();
  const locale = useLocale();

  async function copyCode() {
    try {
      await navigator.clipboard.writeText(detail.code);
      toast.success(t("notification.giftCardRedeemed.copySuccess"));
    } catch {
      toast.error(t("notification.giftCardRedeemed.copyFailed"));
    }
  }

  return (
    <div className="space-y-4" data-testid="gift-card-notification-detail">
      <p className="text-sm font-semibold leading-6 text-white/72">
        {t("notification.giftCardRedeemed.description", { name: detail.product_name })}
      </p>
      <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-3 rounded-lg bg-white/[0.04] p-4 text-sm">
        <dt className="text-white/42">{t("notification.giftCardRedeemed.product")}</dt>
        <dd className="min-w-0 break-words text-right font-bold text-white/82">{detail.product_name}</dd>
        <dt className="text-white/42">{t("notification.giftCardRedeemed.faceValue")}</dt>
        <dd className="min-w-0 break-words text-right font-bold text-white/82">{detail.face_value}</dd>
        <dt className="text-white/42">{t("notification.giftCardRedeemed.pointsSpent")}</dt>
        <dd className="text-right font-bold text-white/82">{detail.points_spent}</dd>
        <dt className="text-white/42">{t("notification.giftCardRedeemed.redeemedAt")}</dt>
        <dd className="text-right font-bold text-white/82">{formatNotificationDate(detail.redeemed_at, locale)}</dd>
      </dl>
      <div className="rounded-lg border border-[#ff7ba6]/25 bg-[#451626]/45 p-4">
        <p className="text-xs font-bold uppercase tracking-wider text-[#ff9abb]">{t("notification.giftCardRedeemed.code")}</p>
        <code className="mt-2 block break-all text-base font-black leading-7 text-white" data-testid="gift-card-notification-code">
          {detail.code}
        </code>
        <Button type="button" onClick={() => void copyCode()} className="mt-4 h-10 w-full bg-[#ff2f69] font-black text-white hover:bg-[#ff497c]">
          <Copy className="size-4" />
          {t("notification.giftCardRedeemed.copy")}
        </Button>
      </div>
    </div>
  );
}

export function NotificationRichText({ className, content }: { className?: string; content: string }) {
  return <MarkdownContent className={cn("notification-rich-text-content", className)} content={content} />;
}

export function EmptyState({
  description,
  loading,
  title,
}: {
  description: string;
  loading?: boolean;
  title: string;
}) {
  return (
    <div className="flex min-h-[42vh] flex-col items-center justify-center px-8 text-center md:rounded-lg md:border md:border-dashed md:border-white/[0.1]">
      {loading ? (
        <RefreshCw className="size-10 animate-spin text-white/28" />
      ) : (
        <Inbox className="size-10 text-white/28" />
      )}
      <h2 className="mt-4 text-base font-semibold text-white">{title}</h2>
      <p className="mt-2 max-w-[360px] text-sm leading-6 text-white/45">{description}</p>
    </div>
  );
}

export function buildVisibleItems(
  activeTab: NotificationTab,
  notifications: NotificationItem[],
  systemNotifications: SystemNotificationItem[],
): NotificationFeedItem[] {
  const systemUserItems = notifications
    .filter((notification) => Number(notification.type) === notificationTypeGiftCardRedeemed)
    .map((notification) => ({ kind: "user" as const, notification }));

  if (activeTab === "system") {
    return [
      ...systemUserItems,
      ...systemNotifications.map((notification) => ({ kind: "system" as const, notification })),
    ].sort((a, b) => getItemTime(b) - getItemTime(a));
  }

  const tab = notificationTabs.find((item) => item.key === activeTab);
  const typeSet = new Set(
    tab?.type
      ?.split(",")
      .map((value) => Number(value.trim()))
      .filter((value) => Number.isFinite(value)) ?? [],
  );
  const userItems = notifications
    .filter((notification) => Number(notification.type) !== notificationTypeGiftCardRedeemed)
    .filter((notification) => typeSet.size === 0 || typeSet.has(Number(notification.type)))
    .map((notification) => ({ kind: "user" as const, notification }));

  if (activeTab !== "all") {
    return userItems;
  }

  return [
    ...userItems,
    ...systemUserItems,
    ...systemNotifications.map((notification) => ({ kind: "system" as const, notification })),
  ].sort((a, b) => getItemTime(b) - getItemTime(a));
}

export function getUserNotificationMeta(
  notification: NotificationItem,
  t: ReturnType<typeof useTranslations>,
) {
  const type = Number(notification.type);
  const senderName =
    notification.sender?.nickname?.trim() ||
    notification.sender?.user_id?.trim() ||
    notification.title ||
    "用户";
  const senderInitial = senderName === "用户" ? "" : senderName.charAt(0).toUpperCase();

  if (type === 1) {
    return {
      actionText: "赞了你的笔记",
      detail: notification.post_title ?? "",
      icon: Heart,
      senderInitial,
      senderName,
    };
  }

  if (type === 2) {
    return {
      actionText: "赞了你的评论",
      detail: notification.comment?.content ?? notification.post_title ?? "",
      icon: Heart,
      senderInitial,
      senderName,
    };
  }

  if (type === 3) {
    return {
      actionText: "评论了你的笔记",
      detail: notification.comment?.content ?? notification.post_title ?? "",
      icon: MessageCircle,
      senderInitial,
      senderName,
    };
  }

  if (type === 4) {
    return {
      actionText: "回复了你的评论",
      detail: notification.comment?.content ?? notification.post_title ?? "",
      icon: MessageCircle,
      senderInitial,
      senderName,
    };
  }

  if (type === 6) {
    return {
      actionText: "收藏了你的笔记",
      detail: notification.post_title ?? "",
      icon: Bookmark,
      senderInitial,
      senderName,
    };
  }

  if (type === 5) {
    return {
      actionText: notification.title || "关注了你",
      detail: "",
      icon: UserPlus,
      senderInitial,
      senderName,
    };
  }

  if (type === 11) {
    return {
      actionText: notification.title || "发布了新的笔记",
      detail: notification.post_title ?? "",
      icon: UserPlus,
      senderInitial,
      senderName,
    };
  }

  if (type === 12) {
    return {
      actionText: translateNotificationTitle(notification.title, t),
      detail: notification.post_title ?? "",
      icon: ShieldCheck,
      senderInitial,
      senderName,
    };
  }

  if (type === notificationTypeGiftCardRedeemed) {
    const giftCard = notification.gift_card_redemption;
    return {
      actionText: t("notification.giftCardRedeemed.action"),
      detail: giftCard
        ? t("notification.giftCardRedeemed.summary", { name: giftCard.product_name, value: giftCard.face_value })
        : t("notification.giftCardRedeemed.legacySummary"),
      icon: GiftIcon,
      senderInitial: "",
      senderName: t("notification.giftCardRedeemed.sender"),
    };
  }

  return {
    actionText: translateNotificationTitle(notification.title, t) || "新的互动通知",
    detail: notification.comment?.content ?? notification.post_title ?? "",
    icon: Star,
    senderInitial,
    senderName,
  };
}

export function translateNotificationTitle(
  title: string | null | undefined,
  t: ReturnType<typeof useTranslations>,
) {
  if (title === "notification.imageProtectionReady.title") {
    return t("notification.imageProtectionReady.title");
  }
  if (title === "notification.giftCardRedeemed.title") {
    return t("notification.giftCardRedeemed.fallbackTitle");
  }
  return title ?? "";
}

export function isOpenableNotification(notification: NotificationItem) {
  return Number(notification.type) === notificationTypeGiftCardRedeemed || Boolean(notification.detail);
}

export function notificationDetailContent(notification: NotificationItem | SystemNotificationItem) {
  if ("detail" in notification && notification.detail) {
    return notification.detail;
  }
  if ("content" in notification && notification.content) {
    return richTextToPlainText(notification.content);
  }
  return notification.title || "系统消息";
}

export function getNotificationSenderHref(notification: NotificationItem) {
  if (Number(notification.type) === notificationTypeGiftCardRedeemed) {
    return null;
  }
  const senderId =
    notification.sender?.user_id?.trim() ||
    normalizeNotificationIdentifier(notification.sender_id) ||
    normalizeNotificationIdentifier(notification.sender?.id);

  return senderId ? getUserHref(senderId) : null;
}

export function getNotificationPostHref(notification: NotificationItem) {
  if (Number(notification.type) === notificationTypeGiftCardRedeemed) {
    return null;
  }
  const postId =
    normalizeNotificationIdentifier(notification.target_id) ??
    normalizeNotificationIdentifier(notification.comment?.post_id);

  if (!postId) {
    return null;
  }

  const params = new URLSearchParams({ id: postId });
  const commentId = getNotificationCommentId(notification);
  if (commentId) {
    params.set("comment", commentId);
    const parentCommentId = getNotificationCommentParentId(notification);
    if (parentCommentId) {
      params.set("parentComment", parentCommentId);
    }
  }

  return `/post?${params.toString()}${commentId ? `#comment-${encodeURIComponent(commentId)}` : ""}`;
}

export function getNotificationCommentId(notification: NotificationItem) {
  return (
    normalizeNotificationIdentifier(notification.comment_id) ??
    normalizeNotificationIdentifier(notification.comment?.id)
  );
}

export function getNotificationCommentParentId(notification: NotificationItem) {
  return normalizeNotificationIdentifier(notification.comment?.parent_id);
}

export function normalizeNotificationIdentifier(value: number | string | null | undefined) {
  if (value === undefined || value === null) {
    return null;
  }

  const normalized = String(value).trim();
  return normalized || null;
}

export function getItemTime(item: NotificationFeedItem) {
  const raw = item.notification.created_at;
  if (!raw) {
    return 0;
  }

  const date = new Date(raw);
  return Number.isNaN(date.getTime()) ? 0 : date.getTime();
}

export function formatShortDate(value?: string) {
  if (!value) {
    return "刚刚";
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
    return new Intl.DateTimeFormat("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }

  if (dayDistance === 1) {
    return "昨天";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
  }).format(date);
}

export function formatNotificationDate(value: string | undefined, locale: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}
