"use client";

import { useCallback, useEffect, useState } from "react";
import Image from "next/image";
import { Bell, CheckCheck, ExternalLink, Megaphone } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  AUTH_USER_EVENT,
  confirmSystemNotification,
  getActivityNotifications,
  getOptionalPopupSystemNotifications,
} from "@/lib/api";
import { MarkdownContent } from "@/components/markdown-content";
import type { SystemNotificationItem } from "@/lib/types";
import { cn } from "@/lib/utils";

const ACTIVITY_POPUP_DISMISSED_KEY = "yuem_dismissed_activity_popups";

type PopupNotification = SystemNotificationItem & {
  popupSource: "system" | "public_activity";
};

export function SystemNotificationPopup() {
  const t = useTranslations("systemNotificationPopup");
  const [notifications, setNotifications] = useState<PopupNotification[]>([]);
  const [actingId, setActingId] = useState<string | number | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadPopups = () => {
      Promise.allSettled([
        getActivityNotifications({ limit: 10 }),
        getOptionalPopupSystemNotifications(),
      ])
        .then(([activityResult, systemResult]) => {
          if (cancelled) {
            return;
          }

          const dismissedActivityIds = getDismissedActivityPopupIds();
          const systemItems: PopupNotification[] =
            systemResult.status === "fulfilled"
              ? systemResult.value
                  .filter((item) => item.show_popup !== false)
                  .map((item) => ({ ...item, popupSource: "system" }))
              : [];
          const activityItems: PopupNotification[] =
            activityResult.status === "fulfilled"
              ? activityResult.value.data
                  .filter((item) => item.show_popup === true)
                  .filter((item) => !dismissedActivityIds.has(String(item.id)))
                  .map((item) => ({
                    ...item,
                    type: item.type ?? "activity",
                    popupSource: "public_activity",
                  }))
              : [];

          setNotifications(mergePopupNotifications(systemItems, activityItems));
        })
        .catch(() => undefined);
    };

    const handleAuthUser = () => {
      loadPopups();
    };

    loadPopups();
    window.addEventListener(AUTH_USER_EVENT, handleAuthUser);

    return () => {
      cancelled = true;
      window.removeEventListener(AUTH_USER_EVENT, handleAuthUser);
    };
  }, []);

  const notification = notifications[0] ?? null;
  const isActing = notification ? actingId === notification.id : false;
  const isActivity =
    notification?.popupSource === "public_activity" || notification?.type === "activity";
  const isPublicActivity = notification?.popupSource === "public_activity";
  const externalLink = Boolean(
    notification?.link_url && /^[a-z][a-z\d+\-.]*:/i.test(notification.link_url),
  );

  const removeCurrent = useCallback((notificationId: string | number) => {
    setNotifications((items) => items.filter((item) => item.id !== notificationId));
  }, []);

  const closeCurrent = useCallback(
    (item: PopupNotification) => {
      if (item.popupSource === "public_activity") {
        rememberDismissedActivityPopupId(item.id);
      }
      removeCurrent(item.id);
    },
    [removeCurrent],
  );

  async function handleConfirm() {
    if (!notification) {
      return;
    }

    if (isPublicActivity) {
      closeCurrent(notification);
      toast.success(t("activityDismissed"));
      return;
    }

    setActingId(notification.id);
    try {
      await confirmSystemNotification(notification.id);
      removeCurrent(notification.id);
      toast.success(isActivity ? t("activityConfirmed") : t("systemConfirmed"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("confirmFailed"));
    } finally {
      setActingId(null);
    }
  }

  if (!notification) {
    return null;
  }

  return (
    <aside
      aria-live="polite"
      className="fixed inset-x-3 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-50 mx-auto flex max-h-[min(82dvh,42rem)] w-[calc(100vw-1.5rem)] max-w-[34rem] flex-col overflow-hidden rounded-lg border border-border bg-background text-foreground shadow-2xl md:inset-x-auto md:left-1/2 md:top-1/2 md:bottom-auto md:w-[min(calc(100vw-3rem),34rem)] md:-translate-x-1/2 md:-translate-y-1/2"
    >
      <div className="flex min-h-0 gap-3 overflow-y-auto overscroll-contain p-4 pb-3">
        <span
          className={cn(
            "mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-full text-white",
            isActivity ? "bg-[#f97316]" : "bg-primary",
          )}
        >
          {isActivity ? <Megaphone className="size-4" /> : <Bell className="size-4" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-start gap-2">
            <div className="min-w-0 flex-1">
              <span
                className={cn(
                  "inline-flex h-5 items-center rounded-full px-2 text-[11px] font-medium",
                  isActivity
                    ? "bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-200"
                    : "bg-primary/10 text-primary",
                )}
              >
                {isActivity ? t("activityBadge") : t("systemBadge")}
              </span>
              <h2 className="mt-1 min-w-0 text-sm font-semibold leading-5">
                {notification.title || (isActivity ? t("activityFallbackTitle") : t("systemFallbackTitle"))}
              </h2>
            </div>
          </div>
          {notification.image_url ? (
            <div className="relative mt-3 aspect-[16/9] overflow-hidden rounded-md border border-border bg-muted">
              <Image
                src={notification.image_url}
                alt={notification.title || (isActivity ? t("activityImageAlt") : t("systemImageAlt"))}
                fill
                unoptimized
                sizes="(max-width: 640px) calc(100vw - 96px), 360px"
                className="object-cover"
              />
            </div>
          ) : null}
          {notification.content ? (
            <NotificationRichText
              content={notification.content}
              className="mt-2 min-w-0 text-sm leading-6 text-muted-foreground [overflow-wrap:anywhere]"
            />
          ) : null}
        </div>
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-t border-border bg-background/95 px-4 py-3">
        <Button
          type="button"
          size="sm"
          onClick={() => void handleConfirm()}
          disabled={isActing}
          className="h-9 flex-1 px-3 sm:flex-none"
        >
          <CheckCheck className="size-4" />
          {t("acknowledge")}
        </Button>
        {notification.link_url ? (
          <Button asChild variant="ghost" size="sm" className="h-9 flex-1 px-3 sm:flex-none">
            <a
              href={notification.link_url}
              target={externalLink ? "_blank" : undefined}
              rel={externalLink ? "noreferrer" : undefined}
            >
              <ExternalLink className="size-4" />
              {isActivity ? t("viewActivity") : t("viewSystem")}
            </a>
          </Button>
        ) : null}
      </div>
    </aside>
  );
}

function mergePopupNotifications(
  systemItems: PopupNotification[],
  activityItems: PopupNotification[],
) {
  const byId = new Map<string, PopupNotification>();
  for (const item of systemItems) {
    byId.set(String(item.id), item);
  }
  for (const item of activityItems) {
    const key = String(item.id);
    if (!byId.has(key)) {
      byId.set(key, item);
    }
  }
  return Array.from(byId.values()).sort((a, b) => popupTime(b) - popupTime(a));
}

function popupTime(item: PopupNotification) {
  const time = Date.parse(item.created_at ?? item.start_time ?? "");
  return Number.isFinite(time) ? time : 0;
}

function getDismissedActivityPopupIds() {
  if (typeof window === "undefined") {
    return new Set<string>();
  }
  try {
    const raw = window.localStorage.getItem(ACTIVITY_POPUP_DISMISSED_KEY);
    const value = raw ? (JSON.parse(raw) as unknown) : [];
    if (!Array.isArray(value)) {
      return new Set<string>();
    }
    return new Set(value.map((item) => String(item)));
  } catch {
    return new Set<string>();
  }
}

function rememberDismissedActivityPopupId(notificationId: string | number) {
  if (typeof window === "undefined") {
    return;
  }
  const next = getDismissedActivityPopupIds();
  next.add(String(notificationId));
  try {
    window.localStorage.setItem(
      ACTIVITY_POPUP_DISMISSED_KEY,
      JSON.stringify(Array.from(next).slice(-100)),
    );
  } catch {
    // Ignore storage failures; the current popup is still removed from this page view.
  }
}

function NotificationRichText({ className, content }: { className?: string; content: string }) {
  return <MarkdownContent className={cn("notification-rich-text-content", className)} content={content} />;
}
