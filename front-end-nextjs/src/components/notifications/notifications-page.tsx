"use client";

import { Button } from "@/components/ui/button";
import {
confirmSystemNotification,
deleteNotification,
dismissSystemNotification,
getNotificationUnreadCount,
getNotifications,
getSystemNotifications,
markAllNotificationsRead,
markNotificationRead,
} from "@/lib/api";
import type { NotificationsInitialData } from "@/lib/server/notifications-page-data";
import type {
NotificationItem,
NotificationUnreadCountPayload,
SystemNotificationItem
} from "@/lib/types";
import { cn } from "@/lib/utils";
import {
ArrowLeft,
MessageCircle,
RefreshCw
} from "lucide-react";
import Link from "next/link";
import { useCallback,useEffect,useMemo,useRef,useState } from "react";
import { toast } from "sonner";
import { EmptyState,NotificationDetailDialog,SystemNotificationCard,UserNotificationCard,buildVisibleItems } from "./notification-items";
import { notificationTabs,type NotificationTab } from "./notification-model";

type DetailNotificationState =
  | { kind: "system"; notification: SystemNotificationItem }
  | { kind: "user"; notification: NotificationItem };

export function NotificationsPage({
  initialData,
}: {
  initialData?: NotificationsInitialData | null;
} = {}) {
  const [activeTab, setActiveTab] = useState<NotificationTab>("all");
  const [notifications, setNotifications] = useState<NotificationItem[]>(
    () => initialData?.notifications ?? [],
  );
  const [systemNotifications, setSystemNotifications] = useState<SystemNotificationItem[]>(
    () => initialData?.systemNotifications ?? [],
  );
  const [unreadCount, setUnreadCount] = useState<NotificationUnreadCountPayload | null>(
    () => initialData?.unreadCount ?? null,
  );
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(() => !initialData);
  const [actingIds, setActingIds] = useState<Set<string>>(() => new Set());
  const actingIdsRef = useRef<Set<string>>(new Set());
  const [detailNotification, setDetailNotification] = useState<DetailNotificationState | null>(null);

  const beginAction = useCallback((actionKey: string) => {
    if (actingIdsRef.current.has(actionKey)) {
      return false;
    }
    actingIdsRef.current.add(actionKey);
    setActingIds(new Set(actingIdsRef.current));
    return true;
  }, []);

  const endAction = useCallback((actionKey: string) => {
    actingIdsRef.current.delete(actionKey);
    setActingIds(new Set(actingIdsRef.current));
  }, []);

  const loadNotifications = useCallback(async function loadNotifications() {
    try {
      setError(null);
      setIsLoading(true);
      const [userPayload, systemPayload, unreadPayload] = await Promise.all([
        getNotifications({ limit: 50 }),
        getSystemNotifications({ limit: 50 }),
        getNotificationUnreadCount(),
      ]);
      setNotifications(userPayload.data);
      setSystemNotifications(systemPayload.data);
      setUnreadCount(unreadPayload);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : "通知加载失败";
      setError(message);
      toast.error(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (initialData) {
      return;
    }
    queueMicrotask(() => {
      void loadNotifications();
    });
  }, [initialData, loadNotifications]);

  const visibleItems = useMemo(
    () => buildVisibleItems(activeTab, notifications, systemNotifications),
    [activeTab, notifications, systemNotifications],
  );

  async function handleMarkAllRead() {
    const actionKey = notificationActionKey("all", "read");
    if (!beginAction(actionKey)) {
      return;
    }
    const previousNotifications = notifications;
    const previousUnreadCount = unreadCount;
    const previousDetailNotification = detailNotification;
    setNotifications((items) => items.map((item) => ({ ...item, is_read: true })));
    setDetailNotification((current) =>
      current?.kind === "user"
        ? { ...current, notification: { ...current.notification, is_read: true } }
        : current,
    );
    setUnreadCount((current) =>
      current
        ? { ...current, notification_count: 0, total: current.system_notification_count }
        : current,
    );
    try {
      await markAllNotificationsRead();
      toast.success("已全部标记为已读");
    } catch (markError) {
      setNotifications(previousNotifications);
      setUnreadCount(previousUnreadCount);
      setDetailNotification(previousDetailNotification);
      toast.error(markError instanceof Error ? markError.message : "标记失败");
    } finally {
      endAction(actionKey);
    }
  }

  async function handleMarkRead(notification: NotificationItem) {
    if (notification.is_read) {
      return;
    }

    const actionKey = notificationActionKey("user", notification.id);
    if (!beginAction(actionKey)) {
      return;
    }
    const previousNotifications = notifications;
    const previousUnreadCount = unreadCount;
    setNotifications((items) =>
      items.map((item) => (item.id === notification.id ? { ...item, is_read: true } : item)),
    );
    setDetailNotification((current) =>
      isSameUserDetail(current, notification.id)
        ? { ...current, notification: { ...current.notification, is_read: true } }
        : current,
    );
    setUnreadCount((current) =>
      current
        ? {
            ...current,
            notification_count: Math.max(0, current.notification_count - 1),
            total: Math.max(0, current.total - 1),
          }
        : current,
    );
    try {
      await markNotificationRead(notification.id);
    } catch (markError) {
      setNotifications(previousNotifications);
      setUnreadCount(previousUnreadCount);
      setDetailNotification((current) =>
        isSameUserDetail(current, notification.id)
          ? { ...current, notification: { ...current.notification, is_read: notification.is_read } }
          : current,
      );
      toast.error(markError instanceof Error ? markError.message : "标记失败");
    } finally {
      endAction(actionKey);
    }
  }

  async function handleOpenUserNotification(notification: NotificationItem) {
    setDetailNotification({ kind: "user", notification });
    await handleMarkRead(notification);
  }

  async function handleDelete(notification: NotificationItem) {
    const actionKey = notificationActionKey("user", notification.id);
    if (!beginAction(actionKey)) {
      return;
    }
    const previousNotifications = notifications;
    const previousUnreadCount = unreadCount;
    const previousDetailNotification = detailNotification;
    setNotifications((items) => items.filter((item) => item.id !== notification.id));
    setDetailNotification((current) =>
      isSameUserDetail(current, notification.id)
        ? null
        : current,
    );
    if (!notification.is_read) {
      setUnreadCount((current) =>
        current
          ? {
              ...current,
              notification_count: Math.max(0, current.notification_count - 1),
              total: Math.max(0, current.total - 1),
            }
          : current,
      );
    }
    try {
      await deleteNotification(notification.id);
      toast.success("通知已删除");
    } catch (deleteError) {
      setNotifications(previousNotifications);
      setUnreadCount(previousUnreadCount);
      if (isSameUserDetail(previousDetailNotification, notification.id)) {
        setDetailNotification((current) => current ?? previousDetailNotification);
      }
      toast.error(deleteError instanceof Error ? deleteError.message : "删除失败");
    } finally {
      endAction(actionKey);
    }
  }

  async function handleConfirmSystem(notification: SystemNotificationItem) {
    const actionKey = notificationActionKey("system", notification.id);
    if (!beginAction(actionKey)) {
      return;
    }
    const previousNotifications = systemNotifications;
    const previousUnreadCount = unreadCount;
    setSystemNotifications((items) =>
      items.map((item) => (item.id === notification.id ? { ...item, is_read: true } : item)),
    );
    setDetailNotification((current) =>
      isSameSystemDetail(current, notification.id)
        ? { ...current, notification: { ...current.notification, is_read: true } }
        : current,
    );
    if (!notification.is_read) {
      setUnreadCount((current) =>
        current
          ? {
              ...current,
              system_notification_count: Math.max(0, current.system_notification_count - 1),
              total: Math.max(0, current.total - 1),
            }
          : current,
      );
    }
    try {
      await confirmSystemNotification(notification.id);
      toast.success("系统通知已确认");
    } catch (confirmError) {
      setSystemNotifications(previousNotifications);
      setUnreadCount(previousUnreadCount);
      setDetailNotification((current) =>
        isSameSystemDetail(current, notification.id)
          ? { ...current, notification: { ...current.notification, is_read: notification.is_read } }
          : current,
      );
      toast.error(confirmError instanceof Error ? confirmError.message : "确认失败");
    } finally {
      endAction(actionKey);
    }
  }

  async function handleDismissSystem(notification: SystemNotificationItem) {
    const actionKey = notificationActionKey("system", notification.id);
    if (!beginAction(actionKey)) {
      return;
    }
    const previousNotifications = systemNotifications;
    const previousUnreadCount = unreadCount;
    const previousDetailNotification = detailNotification;
    setSystemNotifications((items) => items.filter((item) => item.id !== notification.id));
    setDetailNotification((current) =>
      isSameSystemDetail(current, notification.id)
        ? null
        : current,
    );
    if (!notification.is_read) {
      setUnreadCount((current) =>
        current
          ? {
              ...current,
              system_notification_count: Math.max(0, current.system_notification_count - 1),
              total: Math.max(0, current.total - 1),
            }
          : current,
      );
    }
    try {
      await dismissSystemNotification(notification.id);
      toast.success("系统通知已移除");
    } catch (dismissError) {
      setSystemNotifications(previousNotifications);
      setUnreadCount(previousUnreadCount);
      if (isSameSystemDetail(previousDetailNotification, notification.id)) {
        setDetailNotification((current) => current ?? previousDetailNotification);
      }
      toast.error(dismissError instanceof Error ? dismissError.message : "移除失败");
    } finally {
      endAction(actionKey);
    }
  }

  return (
    <div className="theme-adaptive min-h-dvh bg-[#121212] text-white">
      <header className="sticky top-0 z-30 border-b border-white/[0.07] bg-[#121212]/96 backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-[1180px] items-center gap-3 px-4 sm:px-6 lg:px-8">
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label="返回"
            className="size-10 shrink-0 text-white hover:bg-white/[0.06]"
          >
            <Link href="/messages">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <div className="min-w-0 flex-1 text-center">
            <h1 className="truncate text-[24px] font-black leading-none tracking-normal md:text-lg md:font-semibold">
              消息通知
            </h1>
            <p className="mt-1 hidden text-xs text-white/40 md:block">
              {unreadCount?.total ? `${unreadCount.total} 条未读通知` : "暂无未读通知"}
            </p>
          </div>
          <button
            type="button"
            onClick={() => void handleMarkAllRead()}
            className="h-10 shrink-0 whitespace-nowrap px-1 text-sm font-black leading-none text-[#ff7ba6] transition hover:text-[#ff9abb] disabled:text-white/24"
            disabled={isLoading || actingIds.has(notificationActionKey("all", "read")) || !unreadCount?.notification_count}
          >
            全部已读
          </button>
        </div>

        <div className="mx-auto w-full max-w-[1180px] overflow-x-auto px-4 [scrollbar-width:none] sm:px-6 lg:px-8 [&::-webkit-scrollbar]:hidden">
          <div className="flex min-w-max items-end gap-7 border-t border-white/[0.03] sm:gap-9 md:h-12 md:border-t-0">
            {notificationTabs.map((tab) => {
              const active = activeTab === tab.key;

              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setActiveTab(tab.key)}
                  className={cn(
                    "relative h-12 whitespace-nowrap text-[15px] font-black leading-none text-white/42 transition-colors sm:text-base",
                    active && "text-[#ff7ba6]",
                  )}
                >
                  {tab.label}
                  {active ? (
                    <span className="absolute bottom-0 left-1/2 h-1 w-4 -translate-x-1/2 rounded-full bg-[#ff7ba6]" />
                  ) : null}
                </button>
              );
            })}
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-[1180px] pb-[calc(0.75rem+env(safe-area-inset-bottom))] md:px-6 md:py-6 lg:px-8">
        <div className="mb-3 hidden justify-end gap-2 md:flex">
          <Button
            asChild
            variant="outline"
            className="h-10 border-white/10 bg-white/[0.04] px-4 text-white hover:bg-white/[0.08]"
          >
            <Link href="/messages">
              <MessageCircle className="size-4" />
              消息
            </Link>
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="刷新通知"
            onClick={() => void loadNotifications()}
            disabled={isLoading}
            className="size-10 border-white/10 bg-white/[0.04] text-white hover:bg-white/[0.08]"
          >
            <RefreshCw className={cn("size-4", isLoading && "animate-spin")} />
          </Button>
        </div>

        {error ? (
          <EmptyState title="通知加载失败" description={error} />
        ) : isLoading ? (
          <EmptyState loading title="正在加载通知" description="稍等一下，新的提醒马上回来。" />
        ) : visibleItems.length > 0 ? (
          <div className="md:space-y-3">
            {visibleItems.map((item) =>
              item.kind === "user" ? (
                <UserNotificationCard
                  key={`user-${item.notification.id}`}
                  notification={item.notification}
                  acting={actingIds.has(notificationActionKey("user", item.notification.id))}
                  onDelete={handleDelete}
                  onMarkRead={handleMarkRead}
                  onOpen={handleOpenUserNotification}
                />
              ) : (
                <SystemNotificationCard
                  key={`system-${item.notification.id}`}
                  notification={item.notification}
                  acting={actingIds.has(notificationActionKey("system", item.notification.id))}
                  onConfirm={handleConfirmSystem}
                  onDismiss={handleDismissSystem}
                  onOpen={(notification) =>
                    setDetailNotification({ kind: "system", notification })
                  }
                />
              ),
            )}
          </div>
        ) : (
          <EmptyState
            title={`暂无${notificationTabs.find((tab) => tab.key === activeTab)?.label ?? ""}通知`}
            description="点赞、收藏、关注、评论和系统提醒会显示在这里。"
          />
        )}
      </main>

      {detailNotification ? (
        <NotificationDetailDialog
          notification={detailNotification.notification}
          onClose={() => setDetailNotification(null)}
        />
      ) : null}
    </div>
  );
}

function notificationActionKey(scope: "all" | "system" | "user", id: string | number) {
  return `${scope}:${String(id)}`;
}

function isSameUserDetail(
  detail: DetailNotificationState | null,
  id: string | number,
): detail is Extract<DetailNotificationState, { kind: "user" }> {
  return detail?.kind === "user" && String(detail.notification.id) === String(id);
}

function isSameSystemDetail(
  detail: DetailNotificationState | null,
  id: string | number,
): detail is Extract<DetailNotificationState, { kind: "system" }> {
  return detail?.kind === "system" && String(detail.notification.id) === String(id);
}
