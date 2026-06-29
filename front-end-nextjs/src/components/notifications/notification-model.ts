import type { NotificationItem,SystemNotificationItem } from "@/lib/types/notifications";

export type NotificationTab = "all" | "like" | "collect" | "follow" | "comment" | "system";
export type NotificationFeedItem =
  | { kind: "user"; notification: NotificationItem }
  | { kind: "system"; notification: SystemNotificationItem };

export const notificationTypeGiftCardRedeemed = 21;

export const notificationTabs: Array<{ key: NotificationTab; label: string; type?: string }> = [
  { key: "all", label: "全部" },
  { key: "like", label: "赞", type: "1,2" },
  { key: "collect", label: "收藏", type: "6" },
  { key: "follow", label: "关注", type: "5" },
  { key: "comment", label: "评论", type: "3,4" },
  { key: "system", label: "系统" },
];
