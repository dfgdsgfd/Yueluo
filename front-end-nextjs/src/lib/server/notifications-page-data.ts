import "server-only";

import {
  getNotificationUnreadCount,
  getNotifications,
  getRequestAccessToken,
  getSystemNotifications,
  type ApiRequestContext,
} from "@/lib/api";
import type {
  NotificationItem,
  NotificationUnreadCountPayload,
  SystemNotificationItem,
} from "@/lib/types";

export type NotificationsInitialData = {
  notifications: NotificationItem[];
  systemNotifications: SystemNotificationItem[];
  unreadCount: NotificationUnreadCountPayload | null;
};

export async function getNotificationsInitialData(
  context: ApiRequestContext,
): Promise<NotificationsInitialData | null> {
  if (!getRequestAccessToken(context)) {
    return null;
  }

  const [notifications, systemNotifications, unreadCount] =
    await Promise.allSettled([
      getNotifications({ limit: 50 }, context),
      getSystemNotifications({ limit: 50 }, context),
      getNotificationUnreadCount(context),
    ]);

  return {
    notifications:
      notifications.status === "fulfilled" ? notifications.value.data : [],
    systemNotifications:
      systemNotifications.status === "fulfilled"
        ? systemNotifications.value.data
        : [],
    unreadCount:
      unreadCount.status === "fulfilled" ? unreadCount.value : null,
  };
}
