"use client";

import { Settings, SlidersHorizontal, Star, Zap } from "lucide-react";
import type { ResourceConfig } from "./types";
import { booleanOptions, postTypeOptions, qualityLevelOptions } from "./types";
import { PostCell, StatusPill } from "./resource-cells";
import { fieldText, formatDateTime, formatMoney, qualityLevelLabel, qualityTone, simpleColumns, smartCell } from "./helpers";

export const recommendationResourceConfigs: ResourceConfig[] = [
  {
    resource: "post-configs",
    label: "帖子推荐规则",
    singular: "帖子推荐规则",
    description: "单篇内容推荐加权、置顶与抑制",
    icon: Zap,
    tone: "amber",
    columns: [
      { key: "post_id", label: "帖子", render: (row) => <PostCell row={row} /> },
      { key: "boost_score", label: "加权分", render: (row) => smartCell(row.boost_score) },
      { key: "is_pinned", label: "置顶", render: (row) => smartCell(row.is_pinned) },
      { key: "is_suppressed", label: "抑制", render: (row) => smartCell(row.is_suppressed) },
      { key: "is_active", label: "启用", render: (row) => smartCell(row.is_active) },
    ],
    fields: [
      { key: "post_id", label: "内容", type: "number", required: true, picker: "posts", pickerPlaceholder: "搜索标题、正文或作者" },
      { key: "boost_score", label: "加权分", type: "number" },
      { key: "is_pinned", label: "置顶", type: "boolean" },
      { key: "is_suppressed", label: "抑制", type: "boolean" },
      { key: "reason", label: "原因", type: "textarea" },
      { key: "is_active", label: "启用", type: "boolean" },
    ],
    basePath: "/api/admin/recommendation/post-configs",
    canBulkDelete: false,
  },
  {
    resource: "user-configs",
    label: "用户推荐规则",
    singular: "用户推荐规则",
    description: "用户个性化推荐权重",
    icon: SlidersHorizontal,
    tone: "purple",
    columns: simpleColumns(["user_id", "like_weight", "collect_weight", "view_weight", "is_active"]),
    fields: [
      { key: "user_id", label: "用户", type: "number", picker: "users", pickerPlaceholder: "搜索昵称、账号或邮箱" },
      { key: "like_weight", label: "点赞权重", type: "number" },
      { key: "collect_weight", label: "收藏权重", type: "number" },
      { key: "view_weight", label: "浏览权重", type: "number" },
      { key: "is_active", label: "启用", type: "boolean" },
    ],
    basePath: "/api/admin/recommendation/user-configs",
    canBulkDelete: false,
  },
  {
    resource: "posts-quality",
    label: "原创激励",
    singular: "原创激励",
    description: "为单篇作品发放原创激励，质量等级仅作兼容标记",
    icon: Star,
    tone: "amber",
    columns: [
      { key: "title", label: "内容", render: (row) => <PostCell row={row} /> },
      { key: "nickname", label: "作者", render: (row) => fieldText(row, "nickname") },
      { key: "quality_level", label: "质量等级", render: (row) => <StatusPill value={qualityLevelLabel(row.quality_level)} tone={qualityTone(row.quality_level)} /> },
      { key: "quality_reward", label: "原创激励", render: (row) => formatMoney(row.quality_reward) },
      { key: "created_at", label: "发布时间", render: (row) => formatDateTime(row.created_at) },
    ],
    filters: [
      { key: "title", label: "标题" },
      { key: "quality_level", label: "质量等级", type: "select", options: qualityLevelOptions },
      { key: "type", label: "类型", type: "select", options: postTypeOptions },
    ],
    fields: [
      { key: "quality_reward", label: "原创激励金额", type: "number" },
      { key: "quality_level", label: "质量等级（可选）", type: "select", options: qualityLevelOptions },
    ],
    basePath: "/api/admin/posts-quality",
    canCreate: false,
    canDelete: false,
    canBulkDelete: false,
  },
  {
    resource: "quality-reward-settings",
    label: "质量奖励",
    singular: "质量奖励",
    description: "内容质量等级与创作者奖励",
    icon: Star,
    tone: "amber",
    columns: simpleColumns(["quality_level", "reward_amount", "description", "is_active"]),
    fields: [{ key: "reward_amount", label: "奖励金额", type: "number" }, { key: "description", label: "说明" }, { key: "is_active", label: "启用", type: "boolean" }],
    canCreate: false,
    canDelete: false,
    canBulkDelete: false,
  },
  {
    resource: "user-toolbar",
    label: "用户工具栏",
    singular: "工具栏项",
    description: "前台快捷入口与排序",
    icon: Settings,
    tone: "green",
    columns: simpleColumns(["name", "url", "sort_order", "is_active", "created_at"]),
    filters: [{ key: "is_active", label: "启用", type: "boolean", options: booleanOptions }],
    fields: [{ key: "name", label: "名称", required: true }, { key: "icon", label: "图标", type: "select", options: [{ value: "link", label: "链接" }, { value: "external", label: "外链" }] }, { key: "url", label: "URL" }, { key: "sort_order", label: "排序", type: "number" }, { key: "is_active", label: "启用", type: "boolean" }],
  },
];
