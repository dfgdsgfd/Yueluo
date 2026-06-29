"use client";
import type {
  ReactNode
} from "react";
import type {
  LucideIcon
} from "lucide-react";
import type {
  AdminListResource,
  AdminListRow
} from "@/lib/types";

export type AdminView =
  | { kind: "dashboard" }
  | { kind: "resource"; resource: AdminListResource }
  | { kind: "withdraw" }
  | { kind: "invite" }
  | { kind: "coupon" }
  | { kind: "points" }
  | { kind: "onboarding-settings" }
  | { kind: "ai-agent" }
  | { kind: "access-block" }
  | { kind: "hidden-watermark-access" }
  | { kind: "settings" }
  | { kind: "system-update" }
  | { kind: "queues" }
  | { kind: "logs" }
  | { kind: "observability" }
  | { kind: "maintenance" }
  | { kind: "database" }
  | { kind: "operations" }
  | { kind: "component-check" };


export type AdminSection = {
  id: string;
  label: string;
  items: NavItem[];
};


export type NavItem = {
  id: string;
  label: string;
  icon: LucideIcon;
  description?: string;
} & ({ view: AdminView; href?: never } | { href: string; view?: never });


export type SelectOption = {
  value: string;
  label: string;
  hint?: string;
};


export type FieldType = "text" | "number" | "boolean" | "textarea" | "richtext" | "json" | "datetime" | "select" | "password";

export type UploadKind = "image" | "video" | "attachment" | "apk" | "media";

export type UploadedAsset = {
  coverUrl?: string | null;
  coverSignedUrl?: string | null;
  filePath?: string;
  originalname?: string;
  signedUrl?: string;
  size?: number;
  url: string;
};


export type FieldConfig = {
  key: string;
  label: string;
  type?: FieldType;
  placeholder?: string;
  options?: SelectOption[];
  optionValueType?: "string" | "number" | "boolean";
  createOnly?: boolean;
  editOnly?: boolean;
  required?: boolean;
  upload?: UploadKind;
  uploadTarget?: "url" | "json-list";
  autoFilled?: boolean;
  defaultValue?: unknown;
  picker?: PickerResource;
  pickerPlaceholder?: string;
};


export type FilterConfig = {
  key: string;
  label: string;
  type?: "text" | "select" | "boolean";
  options?: SelectOption[];
};


export type ColumnConfig = {
  key: string;
  label?: string;
  labelKey?: string;
  className?: string;
  render?: (row: AdminListRow) => ReactNode;
};


export type ResourceConfig = {
  resource: AdminListResource;
  label: string;
  singular: string;
  description: string;
  icon: LucideIcon;
  tone: Tone;
  columns: ColumnConfig[];
  filters?: FilterConfig[];
  fields: FieldConfig[];
  defaultSort?: string;
  readOnly?: boolean;
  canCreate?: boolean;
  canEdit?: boolean;
  canDelete?: boolean;
  canBulkDelete?: boolean;
  limit?: number;
  basePath?: string;
  tableMinWidth?: number;
};


export type Tone = "red" | "green" | "blue" | "purple" | "amber" | "slate";


export type ActivityItem = Record<string, unknown> & {
  id?: string | number;
  type?: string;
  title?: string;
  content?: string;
  nickname?: string;
  user_id?: string;
  target_id?: string | number;
  created_at?: string;
};


export type ApkFileItem = Record<string, unknown> & {
  name?: string;
  size?: number;
  url?: string;
  createdAt?: string;
  created_at?: string;
};


export type MissingCoverStats = {
  total?: number;
  accessible?: number;
  remote?: number;
};


export type BatchUploadFilesPayload = {
  images?: ApkFileItem[];
  videos?: ApkFileItem[];
};


export type AppVersionStatsPayload = {
  total_users?: number;
  today_active_users?: number;
  version_updates?: Array<Record<string, unknown>>;
  usage_duration?: Record<string, unknown>;
  platform_stats?: Array<Record<string, unknown>>;
};


export type ContentReviewSettingsPayload = {
  ai_auto_review?: boolean;
  ai_username_review?: boolean;
  ai_content_review?: boolean;
};


export type OperationResult = {
  title: string;
  message?: string;
  values?: Record<string, unknown>;
};


export type PickerResource = Extract<AdminListResource, "users" | "posts" | "categories" | "banned-word-categories">;


export type PickerSelection = {
  id: string | number;
  label: string;
  description?: string;
  displayId?: string;
  avatar?: string | null;
};


export type InviteAdminItem = AdminListRow & {
  avatar?: string | null;
  click_count?: number;
  code?: string;
  invited_by_nickname?: string | null;
  is_active?: boolean;
  nickname?: string;
  register_count?: number;
  total_earnings?: number;
  user_id?: string;
};


export type InviteOverviewPayload = {
  total_codes?: number;
  total_clicks?: number;
  total_registers?: number;
  total_earnings?: number;
};


export type CouponAdminItem = AdminListRow & {
  code?: string | null;
  description?: string | null;
  end_time?: string;
  is_active?: boolean;
  issued_count?: number;
  max_discount?: number | null;
  min_order?: number;
  name?: string;
  start_time?: string;
  total_count?: number;
  type?: "amount" | "percent" | string;
  used_count?: number;
  value?: number;
};


export type CouponStatsPayload = {
  activeCoupons?: number;
  totalCoupons?: number;
  totalIssued?: number;
  totalUsed?: number;
};


export const defaultLimit = 12;


export const statusOptions = [
  { value: "0", label: "待处理" },
  { value: "1", label: "已通过" },
  { value: "2", label: "已驳回" },
];


export const booleanOptions = [
  { value: "true", label: "是" },
  { value: "false", label: "否" },
];


export const reportStatusOptions = [
  { value: "pending", label: "待处理" },
  { value: "resolved", label: "已处理" },
  { value: "rejected", label: "已驳回" },
];


export const withdrawStatusOptions = [
  { value: "pending", label: "待审核" },
  { value: "approved", label: "已通过" },
  { value: "rejected", label: "已驳回" },
  { value: "paid", label: "已打款" },
];


export const postTypeOptions = [
  { value: "1", label: "图文" },
  { value: "2", label: "视频" },
  { value: "3", label: "混合" },
];


export const visibilityOptions = [
  { value: "public", label: "公开" },
  { value: "private", label: "私密" },
  { value: "friends_only", label: "好友可见" },
];


export const qualityLevelOptions = [
  { value: "none", label: "未标记" },
  { value: "low", label: "低质量" },
  { value: "medium", label: "中质量" },
  { value: "high", label: "高质量" },
];


export const verifiedOptions = [
  { value: "0", label: "未认证" },
  { value: "1", label: "个人认证" },
  { value: "2", label: "机构认证" },
];


export const reportTargetTypeOptions = [
  { value: "post", label: "帖子" },
  { value: "user", label: "用户" },
  { value: "comment", label: "评论" },
];


export const likeTargetTypeOptions = [
  { value: "1", label: "帖子" },
  { value: "2", label: "评论" },
];


export const mediaTypeOptions = [
  { value: "image", label: "图片" },
  { value: "video", label: "视频" },
  { value: "apk", label: "APK" },
  { value: "file", label: "文件" },
];


export const announcementTypeOptions = [
  { value: "general", label: "普通公告" },
  { value: "system", label: "系统公告" },
  { value: "activity", label: "活动公告" },
  { value: "maintenance", label: "维护公告" },
];


export const systemNotificationTypeOptions = [
  { value: "system", label: "系统通知" },
  { value: "activity", label: "活动通知" },
  { value: "update", label: "更新通知" },
  { value: "warning", label: "风险提醒" },
];


export const notificationTemplateTypeOptions = [
  { value: "system", label: "系统通知" },
  { value: "email", label: "邮件" },
  { value: "discord", label: "Discord" },
  { value: "push", label: "推送" },
];


export const feedbackStatusOptions = [
  { value: "pending", label: "待处理" },
  { value: "processing", label: "处理中" },
  { value: "resolved", label: "已解决" },
  { value: "rejected", label: "已关闭" },
];


export const appPlatformOptions = [
  { value: "android", label: "Android" },
  { value: "android_fast", label: "Android 极速版" },
  { value: "ios", label: "iOS" },
];


export const pointsTaskTypeOptions = [
  { value: "comment", label: "评论", hint: "发布评论自动获得积分" },
  { value: "click", label: "点击", hint: "点击进入内容详情自动获得积分" },
  { value: "like", label: "点赞", hint: "点赞内容自动获得积分" },
  { value: "collect", label: "收藏", hint: "收藏内容自动获得积分" },
  { value: "view", label: "浏览", hint: "浏览内容自动获得积分" },
  { value: "post", label: "发帖", hint: "发布公开内容自动获得积分" },
  { value: "set_avatar", label: "设置头像", hint: "设置头像获得一次性积分" },
  { value: "set_background", label: "设置背景", hint: "设置个人背景获得一次性积分" },
  { value: "set_signature", label: "设置签名", hint: "设置个人签名获得一次性积分" },
  { value: "set_name", label: "设置名称", hint: "设置名称获得一次性积分" },
];


export const achievementTriggerOptions = [
  { value: "total_posts", label: "累计发帖" },
  { value: "consecutive_posts", label: "连续发帖" },
  { value: "total_points", label: "累计积分" },
];


export const couponTypeOptions = [
  { value: "amount", label: "满减金额" },
  { value: "percent", label: "折扣百分比" },
];


export const couponIssueTargetOptions = [
  { value: "users", label: "指定用户" },
  { value: "all", label: "全部活跃用户" },
  { value: "vip1", label: "VIP1 及以上" },
  { value: "vip2", label: "VIP2 及以上" },
];


export const systemUpdateStartModeOptions = [
  { value: "start", label: "直接 start", hint: "使用已构建的 .next，执行 npm run start" },
  { value: "build_start", label: "build + start", hint: "重启容器时先 npm ci / build，再 npm run start" },
];
