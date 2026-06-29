import type { FeedPagination } from "./content";

export type NotificationUnreadCountPayload = {
  notification_count: number;
  system_notification_count: number;
  total: number;
};

export type NotificationSender = {
  id: number | string;
  nickname?: string | null;
  avatar?: string | null;
  user_id?: string | null;
};

export type NotificationComment = {
  id: number | string;
  content: string;
  post_id?: number | string;
  parent_id?: number | string | null;
};

export type GiftCardRedemptionNotification = {
  product_name: string;
  face_value: string;
  points_spent: number;
  code: string;
  redeemed_at?: string;
};

export type NotificationItem = {
  id: number | string;
  user_id?: number | string;
  sender_id?: number | string;
  type?: number;
  title: string;
  target_id?: number | string | null;
  comment_id?: number | string | null;
  is_read: boolean;
  created_at?: string;
  detail?: string | null;
  gift_card_redemption?: GiftCardRedemptionNotification | null;
  sender?: NotificationSender | null;
  comment?: NotificationComment | null;
  post_cover?: string | null;
  post_title?: string | null;
};

export type SystemNotificationItem = {
  id: number | string;
  title: string;
  content?: string | null;
  type?: string | null;
  content_format?: string | null;
  image_url?: string | null;
  link_url?: string | null;
  show_popup?: boolean;
  is_active?: boolean;
  is_read?: boolean;
  confirmed_at?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  created_at?: string;
  updated_at?: string | null;
};

export type NotificationListPayload = {
  data: NotificationItem[];
  pagination: FeedPagination;
};

export type SystemNotificationListPayload = {
  data: SystemNotificationItem[];
  pagination: FeedPagination;
};

export type ActivityNotificationListPayload = SystemNotificationListPayload;
