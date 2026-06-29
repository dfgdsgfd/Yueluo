"use client";

import { KeyRound, Radio, ShieldCheck, UserCog } from "lucide-react";
import type { ResourceConfig } from "./types";
import { booleanOptions } from "./types";
import { simpleColumns } from "./helpers";

export const securityResourceConfigs: ResourceConfig[] = [
  {
    resource: "open-apis",
    label: "开放 API",
    singular: "API Key",
    description: "外部调用密钥与权限",
    icon: KeyRound,
    tone: "purple",
    columns: simpleColumns(["name", "api_key_prefix", "permissions", "is_active", "last_used_at"]),
    filters: [{ key: "is_active", label: "启用", type: "boolean", options: booleanOptions }],
    fields: [
      { key: "name", label: "名称", required: true },
      { key: "permissions", label: "权限配置", type: "json" },
      { key: "is_active", label: "启用", type: "boolean" },
    ],
  },
  {
    resource: "admins",
    label: "管理员",
    singular: "管理员",
    description: "后台管理员账号",
    icon: UserCog,
    tone: "slate",
    columns: simpleColumns(["username", "created_at"]),
    filters: [{ key: "username", label: "账号" }],
    fields: [{ key: "username", label: "账号", required: true }, { key: "password", label: "密码", type: "password", required: true }],
  },
  {
    resource: "sessions",
    label: "会话管理",
    singular: "会话",
    description: "登录会话、过期时间与设备信息",
    icon: Radio,
    tone: "slate",
    columns: simpleColumns(["id", "user_id", "type", "last_active_at", "expires_at", "created_at"]),
    filters: [{ key: "is_active", label: "启用", type: "boolean", options: booleanOptions }, { key: "user_display_id", label: "用户" }],
    fields: [
      { key: "user_id", label: "用户", type: "number", picker: "users", pickerPlaceholder: "搜索昵称、账号或邮箱" },
      { key: "is_active", label: "启用", type: "boolean", defaultValue: true },
      { key: "expires_at", label: "过期时间", type: "datetime" },
    ],
    canCreate: false,
    canEdit: false,
    canDelete: true,
    canBulkDelete: false,
  },
  {
    resource: "banned-word-categories",
    label: "敏感词分类",
    singular: "敏感词分类",
    description: "敏感词分组与说明",
    icon: ShieldCheck,
    tone: "amber",
    columns: simpleColumns(["name", "description", "word_count", "created_at"]),
    filters: [{ key: "name", label: "名称" }],
    fields: [{ key: "name", label: "名称", required: true }, { key: "description", label: "描述", type: "textarea" }],
    canBulkDelete: false,
  },
  {
    resource: "banned-words",
    label: "敏感词",
    singular: "敏感词",
    description: "敏感词、正则与启用状态",
    icon: ShieldCheck,
    tone: "amber",
    columns: simpleColumns(["word", "category_name", "is_regex", "enabled", "created_at"]),
    filters: [{ key: "enabled", label: "启用", type: "boolean", options: booleanOptions }, { key: "is_regex", label: "正则", type: "boolean", options: booleanOptions }],
    fields: [
      { key: "word", label: "词条", required: true },
      { key: "category_id", label: "分类", type: "number", picker: "banned-word-categories", pickerPlaceholder: "搜索敏感词分类" },
      { key: "is_regex", label: "正则", type: "boolean" },
      { key: "enabled", label: "启用", type: "boolean" },
    ],
  },
];
