"use client";

import { Bell, ClipboardList, Megaphone, MessageSquare } from "lucide-react";
import type { ResourceConfig } from "./types";
import { announcementTypeOptions, booleanOptions, feedbackStatusOptions, notificationTemplateTypeOptions, systemNotificationTypeOptions } from "./types";
import { StatusPill } from "./resource-cells";
import { clampText, fieldText, formatDateTime, richTextSummary, simpleColumns, userObjectName } from "./helpers";

export const notificationResourceConfigs: ResourceConfig[] = [
  {
    resource: "announcements",
    label: "公告管理",
    singular: "公告",
    description: "站内公告与发布状态",
    icon: Megaphone,
    tone: "amber",
    columns: simpleColumns(["title", "type", "is_published", "published_at", "expires_at", "created_at"]),
    filters: [{ key: "type", label: "类型", type: "select", options: announcementTypeOptions }, { key: "is_published", label: "已发布", type: "boolean", options: booleanOptions }],
    fields: [
      { key: "title", label: "标题", required: true },
      { key: "content", label: "内容", type: "richtext" },
      { key: "type", label: "类型", type: "select", options: announcementTypeOptions },
      { key: "is_published", label: "发布", type: "boolean" },
      { key: "duration_days", label: "有效天数", type: "number", defaultValue: 7, placeholder: "编辑留空不修改；填 0 长期有效" },
    ],
    canBulkDelete: false,
  },
  {
    resource: "system-notifications",
    label: "系统通知",
    singular: "系统通知",
    description: "弹窗、站内信和确认状态",
    icon: Bell,
    tone: "blue",
    columns: simpleColumns(["title", "type", "show_popup", "is_active", "confirmed_count", "unread_count"]),
    filters: [{ key: "type", label: "类型", type: "select", options: systemNotificationTypeOptions }, { key: "is_active", label: "启用", type: "boolean", options: booleanOptions }],
    fields: [
      { key: "title", label: "标题", required: true },
      { key: "content", label: "内容", type: "richtext" },
      { key: "type", label: "类型", type: "select", options: systemNotificationTypeOptions },
      { key: "image_url", label: "图片", upload: "image" },
      { key: "link_url", label: "链接 URL" },
      { key: "show_popup", label: "弹窗", type: "boolean" },
      { key: "is_active", label: "启用", type: "boolean" },
      { key: "start_time", label: "开始时间", type: "datetime" },
      { key: "end_time", label: "结束时间", type: "datetime" },
    ],
  },
  {
    resource: "notification-templates",
    label: "通知模板",
    singular: "模板",
    description: "通知、邮件与测试模板",
    icon: ClipboardList,
    tone: "blue",
    columns: simpleColumns(["template_key", "name", "type", "is_active", "updated_at"]),
    filters: [{ key: "type", label: "类型", type: "select", options: notificationTemplateTypeOptions }, { key: "is_active", label: "启用", type: "boolean", options: booleanOptions }],
    fields: [
      { key: "template_key", label: "模板键", required: true },
      { key: "name", label: "名称", required: true },
      { key: "type", label: "类型", type: "select", options: notificationTemplateTypeOptions },
      { key: "subject", label: "主题" },
      { key: "content", label: "内容", type: "richtext" },
      { key: "is_active", label: "启用", type: "boolean" },
    ],
  },
  {
    resource: "feedback",
    label: "用户反馈",
    singular: "反馈",
    description: "反馈工单、回复与状态",
    icon: MessageSquare,
    tone: "green",
    columns: [
      { key: "content", label: "反馈", render: (row) => clampText(richTextSummary(row.content)) },
      { key: "user", label: "用户", render: (row) => userObjectName(row.user) },
      { key: "status", label: "状态", render: (row) => <StatusPill value={fieldText(row, "status")} tone={String(row.status) === "resolved" ? "green" : "amber"} /> },
      { key: "created_at", label: "提交时间", render: (row) => formatDateTime(row.created_at) },
    ],
    filters: [{ key: "status", label: "状态", type: "select", options: feedbackStatusOptions }],
    fields: [{ key: "status", label: "状态", type: "select", options: feedbackStatusOptions }, { key: "admin_reply", label: "管理员回复", type: "richtext" }],
    canCreate: false,
    canBulkDelete: false,
  },
];
