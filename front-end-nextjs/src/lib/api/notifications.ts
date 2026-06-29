import type {
  ActivityNotificationListPayload,
  NotificationListPayload,
  NotificationUnreadCountPayload,
  SystemNotificationItem,
  SystemNotificationListPayload,
} from "../types";
import {
  ApiRequestContext,
  type ApiRequestOptions,
  ApiUnauthorizedError,
  apiDelete,
  apiGet,
  apiPost,
  apiPut,
  getStoredAccessToken,
} from "./core";

export function getNotificationUnreadCount(
  context: ApiRequestContext = {},
  options: Omit<
    ApiRequestOptions,
    "body" | "context" | "method" | "query"
  > = {},
) {
  return apiGet<NotificationUnreadCountPayload>(
    "/api/notifications/unread-count",
    undefined,
    {
      context,
      ...options,
    },
  );
}

export function getNotifications(
  query: {
    page?: number;
    limit?: number;
    type?: string;
    is_read?: string;
  } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<NotificationListPayload>(
    "/api/notifications",
    {
      page: query.page ?? 1,
      limit: query.limit ?? 20,
      type: query.type,
      is_read: query.is_read,
    },
    { context },
  );
}

export function getSystemNotifications(
  query: { page?: number; limit?: number; type?: string } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<SystemNotificationListPayload>(
    "/api/notifications/system",
    {
      page: query.page ?? 1,
      limit: query.limit ?? 20,
      type: query.type,
    },
    { context },
  );
}

export function getActivityNotifications(
  query: { page?: number; limit?: number } = {},
) {
  return apiGet<ActivityNotificationListPayload>(
    "/api/notifications/activities",
    {
      page: query.page ?? 1,
      limit: query.limit ?? 20,
    },
    { auth: false },
  );
}

export function getPopupSystemNotifications() {
  return apiGet<SystemNotificationItem[]>("/api/notifications/system/popup");
}

export async function getOptionalPopupSystemNotifications() {
  const token = getStoredAccessToken();
  if (!token) {
    return [];
  }

  try {
    return await apiGet<SystemNotificationItem[]>(
      "/api/notifications/system/popup",
      undefined,
      { auth: false, context: { token } },
    );
  } catch (error) {
    if (error instanceof ApiUnauthorizedError) {
      return [];
    }
    throw error;
  }
}

export function markNotificationRead(notificationId: string | number) {
  return apiPut(`/api/notifications/${notificationId}/read`);
}

export function markAllNotificationsRead() {
  return apiPut("/api/notifications/read-all");
}

export function deleteNotification(notificationId: string | number) {
  return apiDelete(`/api/notifications/${notificationId}`);
}

export function confirmSystemNotification(notificationId: string | number) {
  return apiPost(`/api/notifications/system/${notificationId}/confirm`);
}

export function dismissSystemNotification(notificationId: string | number) {
  return apiDelete(`/api/notifications/system/${notificationId}/dismiss`);
}
