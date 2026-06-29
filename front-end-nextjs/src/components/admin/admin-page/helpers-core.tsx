"use client";
import { richTextToPlainText } from "@/lib/rich-text";
import type {
AdminDashboardOverviewPayload,
AdminListPayload,
AdminListResource,
AdminListRow,
AdminStatsOverviewPayload,
SystemUpdateConfigPayload,
SystemUpdateReleaseOptionPayload
} from "@/lib/types";
import type {
ReactNode
} from "react";
import {
pickerIDFromDraftValue,
pickerSelectionFromField
} from "./object-picker";
import {
StatusPill
} from "./resource-cells";
import {
SystemUpdateDraft
} from "./system-update-panel";
import {
shortHash,
systemUpdateReleaseHash
} from "./system-update-widgets";
import {
AdminView,
ColumnConfig,
CouponAdminItem,
FieldConfig,
ResourceConfig,
SelectOption,
Tone
} from "./types";

export function simpleColumns(keys: string[]): ColumnConfig[] {
  return keys.map((key) => ({
    key,
    label: columnLabel(key),
    render: (row) => key.endsWith("_at") || key.endsWith("_time") ? formatDateTime(row[key]) : smartCell(row[key]),
  }));
}


export function statsToOverview(stats: AdminStatsOverviewPayload): AdminDashboardOverviewPayload {
  return {
    generated_at: new Date().toISOString(),
    metrics: [
      { key: "users", label: "用户", value: stats.users, delta: 0, tone: "red" },
      { key: "posts", label: "内容", value: stats.posts, delta: 0, tone: "blue" },
      { key: "comments", label: "评论", value: stats.comments, delta: 0, tone: "green" },
      { key: "reports", label: "举报", value: stats.reports, delta: 0, tone: "amber" },
      { key: "feedback", label: "反馈", value: stats.feedback, delta: 0, tone: "purple" },
      { key: "announcements", label: "公告", value: stats.announcements, delta: 0, tone: "slate" },
    ],
    pending: {},
    finance: {},
    statuses: [],
  };
}


export function defaultDraft(fields: FieldConfig[], row: AdminListRow | null, mode: "create" | "edit" | "detail") {
  const draft: Record<string, unknown> = {};
  fields.forEach((field) => {
    if (mode === "create") {
      if (field.defaultValue !== undefined) draft[field.key] = field.defaultValue;
      else if (field.type === "boolean") draft[field.key] = true;
      else if (field.type === "number") draft[field.key] = "";
      else draft[field.key] = "";
      return;
    }
    const rowValue = nestedRecordValue(row, field.key);
    draft[field.key] = field.picker ? (pickerSelectionFromField(field, rowValue, row) ?? "") : (rowValue ?? "");
  });
  return draft;
}


export function appVersionLastFormDraft(value: Record<string, unknown>) {
  const draft: Record<string, unknown> = {};
  for (const key of ["app_name", "version_code", "version_name", "platform", "download_url", "size_mb", "update_log"]) {
    const text = String(value[key] ?? "").trim();
    if (text) draft[key] = text;
  }
  if (value.force_update !== undefined && value.force_update !== null && value.force_update !== "") {
    draft.force_update = truthy(value.force_update);
  }
  if (value.is_active !== undefined && value.is_active !== null && value.is_active !== "") {
    draft.is_active = truthy(value.is_active);
  }
  return draft;
}


export function sanitizeDraft(fields: FieldConfig[], draft: Record<string, unknown>, mode: "create" | "edit" | "detail" | null) {
  const out: Record<string, unknown> = {};
  fields.forEach((field) => {
    if (mode === "create" && field.editOnly) return;
    if (mode === "edit" && field.createOnly) return;
    let value = draft[field.key];
    if (field.picker) {
      value = pickerIDFromDraftValue(value);
      if (value === "" || value === undefined || value === null) return;
    }
    if (field.type === "password" && !value) return;
    if (value === "" || value === undefined) return;
    if (field.type === "number") {
      const numeric = Number(value);
      if (Number.isFinite(numeric)) value = numeric;
    }
    if (field.type === "json" && typeof value === "string") {
      const rawJson = value;
      try {
        value = JSON.parse(rawJson);
      } catch {
        value = rawJson.trim();
      }
    }
    if (field.uploadTarget === "json-list" && typeof value === "string") {
      value = value.split(/\r?\n|,|，/).map((item) => item.trim()).filter(Boolean);
      if (Array.isArray(value) && value.length === 0) return;
    }
    setNestedRecordValue(out, field.key, value);
  });
  return out;
}


function nestedRecordValue(record: Record<string, unknown> | null | undefined, key: string) {
  if (!record || !key.includes(".")) return record?.[key];
  const [root, ...path] = key.split(".");
  let value: unknown = record[root];
  if (typeof value === "string") {
    const parsed = parseJsonLike(value);
    if (parsed !== null) value = parsed;
  }
  for (const segment of path) {
    if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
    value = (value as Record<string, unknown>)[segment];
  }
  return value;
}


function setNestedRecordValue(record: Record<string, unknown>, key: string, value: unknown) {
  if (!key.includes(".")) {
    record[key] = value;
    return;
  }
  const [root, ...path] = key.split(".");
  const target = record[root] && typeof record[root] === "object" && !Array.isArray(record[root])
    ? record[root] as Record<string, unknown>
    : {};
  record[root] = target;
  let current = target;
  path.forEach((segment, index) => {
    if (index === path.length - 1) {
      current[segment] = value;
      return;
    }
    const next = current[segment];
    const child = next && typeof next === "object" && !Array.isArray(next)
      ? next as Record<string, unknown>
      : {};
    current[segment] = child;
    current = child;
  });
}


export function drawerTitle(config: ResourceConfig, mode: "create" | "edit" | "detail" | null) {
  if (mode === "create") return `新建${config.singular}`;
  if (mode === "edit") return `编辑${config.singular}`;
  if (mode === "detail") return `${config.singular}详情`;
  return "";
}


export function systemUpdateDraftFromConfig(config?: Partial<SystemUpdateConfigPayload>): SystemUpdateDraft {
  return {
    frontend_repo_url: String(config?.frontend_repo_url ?? ""),
    backend_repo_url: String(config?.backend_repo_url ?? ""),
    frontend_release_tag: String(config?.frontend_release_tag ?? "latest"),
    backend_release_tag: String(config?.backend_release_tag ?? "latest"),
    frontend_artifact_url: String(config?.frontend_artifact_url ?? ""),
    backend_artifact_url: String(config?.backend_artifact_url ?? ""),
    frontend_asset_pattern: String(config?.frontend_asset_pattern ?? "frontend-*-*.zip"),
    backend_asset_pattern: String(config?.backend_asset_pattern ?? ""),
    frontend_install_dir: String(config?.frontend_install_dir ?? ""),
    backend_install_path: String(config?.backend_install_path ?? ""),
    frontend_start_mode: String(config?.frontend_start_mode ?? "start"),
    artifact_dir: String(config?.artifact_dir ?? ""),
  };
}


export function systemUpdateStartModeLabel(value: unknown) {
  return String(value ?? "") === "build_start" ? "build + start" : "直接 start";
}


export function systemUpdateStartCommand(value: unknown) {
  return String(value ?? "") === "build_start" ? "npm ci && npm run build && npm run start" : "npm run start";
}


export function systemUpdateSelectedRelease(value: string, releases: SystemUpdateReleaseOptionPayload[]) {
  if (!releases.length) return null;
  if (value === "latest") return releases[0] ?? null;
  return releases.find((release) => release.tag_name === value) ?? null;
}


export function systemUpdateReleaseOptionLabel(release: SystemUpdateReleaseOptionPayload) {
  const pieces = [release.tag_name];
  const time = formatDateTime(release.published_at ?? release.created_at);
  if (time !== "-") pieces.push(time);
  if (systemUpdateReleaseHash(release)) pieces.push(shortHash(systemUpdateReleaseHash(release)));
  const asset = release.matching_assets?.[0];
  if (asset?.sha256) pieces.push(asset.sha256.slice(0, 12));
  return pieces.join(" · ");
}


export function sameView(a: AdminView, b: AdminView) {
  if (a.kind !== b.kind) return false;
  return a.kind !== "resource" || (b.kind === "resource" && a.resource === b.resource);
}


export function viewTitle(
  view: AdminView,
  resources?: Map<AdminListResource, ResourceConfig>,
) {
  if (view.kind === "dashboard") return "运营看板";
  if (view.kind === "withdraw") return "提现审核";
  if (view.kind === "invite") return "邀请裂变";
  if (view.kind === "coupon") return "优惠券运营";
  if (view.kind === "points") return "积分运营";
  if (view.kind === "onboarding-settings") return "引导设置";
  if (view.kind === "hidden-watermark-access") return "隐藏水印权限";
  if (view.kind === "settings") return "系统设置";
  if (view.kind === "system-update") return "系统更新";
  if (view.kind === "queues") return "队列状态";
  if (view.kind === "logs") return "日志中心";
  if (view.kind === "observability") return "系统观测";
  if (view.kind === "database") return "数据库管理";
  if (view.kind === "maintenance") return "维护模式";
  if (view.kind === "operations") return "运营控制台";
  if (view.kind === "component-check") return "组件检查与测试";
  if (view.kind === "resource") {
    return resources?.get(view.resource)?.label ?? "资源管理";
  }
  return "";
}


export function fieldText(row: AdminListRow, key: string) {
  const value = row[key];
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "object") return readableValue(value);
  return String(value);
}

export function fieldBytes(row: AdminListRow, key: string) {
  return formatBytes(row[key]);
}


export function smartCell(value: unknown) {
  if (typeof value === "boolean") return <StatusPill value={value ? "是" : "否"} tone={value ? "green" : "slate"} />;
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "number") return formatCompact(value);
  if (typeof value === "object") return renderReadableValue(value);
  if (typeof value === "string" && parseJsonLike(value) !== null) return renderReadableValue(value);
  return clampText(String(value));
}


export function clampText(value: string) {
  return <span className="line-clamp-2 break-words">{richTextSummary(value) || "-"}</span>;
}


export function richTextSummary(value: unknown) {
  const raw = typeof value === "string" ? value : fieldText({ value }, "value");
  if (!raw || raw === "-") return raw;
  return richTextToPlainText(raw).replace(/\s+/g, " ").trim();
}


export function compactMetrics(row: AdminListRow, keys: string[]) {
  return (
    <span className="text-xs text-[#6f7582]">
      {keys.map((key) => `${metricShortLabel(key)} ${formatCompact(row[key])}`).join(" / ")}
    </span>
  );
}


export function readableValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "是" : "否";
  if (typeof value === "number") return Number.isFinite(value) ? formatCompact(value) : "-";
  if (typeof value === "string") {
    const parsed = parseJsonLike(value);
    if (Array.isArray(parsed)) return parsed.length ? `${parsed.length} 项` : "空列表";
    if (parsed && typeof parsed === "object") return `${Object.keys(parsed as Record<string, unknown>).length} 个字段`;
    return value;
  }
  if (Array.isArray(value)) return value.length ? `${value.length} 项` : "空列表";
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    const preferred = ["name", "title", "nickname", "user_id", "id", "status", "type"]
      .map((key) => record[key])
      .find((item) => item !== undefined && item !== null && item !== "");
    if (preferred !== undefined) return String(preferred);
    return `${Object.keys(record).length} 个字段`;
  }
  return String(value);
}


export function selectOptionLabel(options: SelectOption[], value: unknown) {
  const raw = String(value ?? "");
  return options.find((option) => option.value === raw)?.label;
}


export function renderReadableValue(value: unknown): ReactNode {
  if (typeof value === "string") {
    const parsed = parseJsonLike(value);
    if (parsed !== null) return renderReadableValue(parsed);
    return <span>{value || "-"}</span>;
  }
  if (Array.isArray(value)) {
    if (!value.length) return <span className="text-[#8b919e]">空列表</span>;
    return (
      <div className="flex flex-wrap gap-1.5">
        {value.slice(0, 6).map((item, index) => (
          <span key={index} className="rounded-full bg-[#f0f2f6] px-2 py-1 text-xs text-[#555b66]">{readableValue(item)}</span>
        ))}
        {value.length > 6 ? <span className="rounded-full bg-[#fff5f6] px-2 py-1 text-xs text-[#d71935]">+{value.length - 6}</span> : null}
      </div>
    );
  }
  if (value && typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>).filter(([, item]) => item !== undefined && item !== null && item !== "").slice(0, 6);
    if (!entries.length) return <span className="text-[#8b919e]">空对象</span>;
    return (
      <div className="grid gap-1">
        {entries.map(([key, item]) => (
          <div key={key} className="flex min-w-0 items-center justify-between gap-2 rounded-md bg-[#fafbfe] px-2 py-1 text-xs">
            <span className="truncate text-[#8b919e]">{columnLabel(key)}</span>
            <span className="min-w-0 truncate text-right font-medium text-[#30333b]">{readableValue(item)}</span>
          </div>
        ))}
      </div>
    );
  }
  return <span>{readableValue(value)}</span>;
}


export function parseJsonLike(value: string) {
  const trimmed = value.trim();
  if ((!trimmed.startsWith("{") || !trimmed.endsWith("}")) && (!trimmed.startsWith("[") || !trimmed.endsWith("]"))) {
    return null;
  }
  try {
    return JSON.parse(trimmed) as unknown;
  } catch {
    return null;
  }
}


export function formatBytes(value: unknown) {
  const bytes = Number(value);
  if (!Number.isFinite(bytes) || bytes <= 0) return "-";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(size >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}


export function formatDuration(value: unknown) {
  const seconds = Number(value);
  if (!Number.isFinite(seconds) || seconds <= 0) return "-";
  if (seconds < 60) return `${Math.round(seconds)} 秒`;
  const minutes = Math.floor(seconds / 60);
  const rest = Math.round(seconds % 60);
  if (minutes < 60) return rest ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`;
  const hours = Math.floor(minutes / 60);
  return `${hours} 小时 ${minutes % 60} 分`;
}


export function activityTypeLabel(value: unknown) {
  const labels: Record<string, string> = {
    user_register: "新用户注册",
    post_create: "新笔记发布",
    comment_create: "新评论",
  };
  return labels[String(value ?? "")] ?? "系统活动";
}


export function recommendConfigEntries(config: Record<string, unknown>): Array<[string, unknown]> {
  const defaults: Record<string, unknown> = {
    like_weight: 1,
    collect_weight: 1,
    view_weight: 1,
    category_weight: 1,
    tag_weight: 1,
    following_weight: 1,
    mutual_follow_weight: 1,
    popularity_weight: 1,
    interest_weight: 1,
    time_decay_half_life: 72,
  };
  return Object.entries({ ...defaults, ...config }).filter(([, value]) => typeof value !== "object");
}


export function parseConfigValue(next: string, previous: unknown) {
  if (typeof previous === "number") {
    const numeric = Number(next);
    return Number.isFinite(numeric) ? numeric : previous;
  }
  if (typeof previous === "boolean") return next === "true";
  return next;
}


export function fallbackPagination(page: number, limit: number, itemCount: number): AdminListPayload["pagination"] {
  return {
    page,
    limit,
    pageSize: limit,
    total: itemCount,
    totalPages: page,
    hasNextPage: itemCount >= limit,
  };
}


export function paginationHasNext(pagination?: AdminListPayload["pagination"]) {
  if (!pagination) return false;
  if (pagination.hasNextPage !== undefined) return Boolean(pagination.hasNextPage);
  const totalPages = pagination.totalPages ?? pagination.pages ?? 0;
  return totalPages ? pagination.page < totalPages : false;
}


export function couponDefaultDraft(row: CouponAdminItem | null): Record<string, unknown> {
  const now = new Date();
  const nextMonth = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
  return {
    name: row?.name ?? "",
    description: row?.description ?? "",
    code: row?.code ?? "",
    type: row?.type ?? "amount",
    value: row?.value ?? "",
    min_order: row?.min_order ?? 0,
    max_discount: row?.max_discount ?? "",
    start_time: toDatetimeLocal(row?.start_time ?? now.toISOString()),
    end_time: toDatetimeLocal(row?.end_time ?? nextMonth.toISOString()),
    total_count: row?.total_count ?? -1,
    is_active: row?.is_active ?? true,
  };
}


export function couponDraftPayload(draft: Record<string, unknown>) {
  return {
    name: String(draft.name ?? "").trim(),
    description: String(draft.description ?? "").trim(),
    code: String(draft.code ?? "").trim(),
    type: String(draft.type ?? "amount"),
    value: Number(draft.value),
    min_order: Number(draft.min_order) || 0,
    max_discount: draft.max_discount === "" || draft.max_discount === null || draft.max_discount === undefined ? undefined : Number(draft.max_discount),
    start_time: datetimeLocalToRFC3339(String(draft.start_time ?? "")),
    end_time: datetimeLocalToRFC3339(String(draft.end_time ?? "")),
    total_count: Number.isFinite(Number(draft.total_count)) ? Number(draft.total_count) : -1,
    is_active: Boolean(draft.is_active),
  };
}


export function toDatetimeLocal(value: unknown) {
  if (!value) return "";
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return String(value).slice(0, 16);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}


export function datetimeLocalToRFC3339(value: string) {
  if (!value) return "";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}


export function couponValueLabel(row: CouponAdminItem) {
  if (row.type === "percent") return `${formatCompact(row.value ?? 0)} 折`;
  return `减 ${formatMoney(row.value ?? 0)}`;
}


export function couponUsageStatusLabel(value: unknown) {
  const raw = String(value ?? "");
  if (raw === "used") return "已使用";
  if (raw === "expired") return "已过期";
  if (raw === "claimed") return "已领取";
  return raw || "-";
}


export function couponUsageUser(row: AdminListRow) {
  const user = row.user;
  if (user && typeof user === "object" && !Array.isArray(user)) {
    const record = user as Record<string, unknown>;
    return String(record.nickname ?? record.user_id ?? record.id ?? "-");
  }
  return "-";
}


export function reporterName(row: AdminListRow) {
  return userObjectName(row.reporter);
}


export function userObjectName(value: unknown) {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    const record = value as Record<string, unknown>;
    return String(record.nickname ?? record.user_id ?? record.id ?? "-");
  }
  return "-";
}


export function activeLabel(value: unknown) {
  return truthy(value) ? "启用" : "停用";
}


export function truthy(value: unknown) {
  return value === true || value === 1 || value === "1" || value === "true";
}


export function auditStatusLabel(value: unknown) {
  const raw = String(value ?? "");
  if (raw === "0" || raw === "pending") return "待处理";
  if (raw === "1" || raw === "approved") return "已通过";
  if (raw === "2" || raw === "rejected") return "已驳回";
  return raw || "-";
}


export function verificationTypeLabel(value: unknown) {
  const raw = String(value ?? "");
  if (raw === "1") return "个人认证";
  if (raw === "2") return "官方认证";
  return raw || "-";
}


export function auditTone(value: unknown): Tone {
  const raw = String(value ?? "");
  if (raw === "1" || raw === "approved") return "green";
  if (raw === "2" || raw === "rejected") return "red";
  return "amber";
}


export function reportStatusLabel(value: unknown) {
  switch (String(value ?? "")) {
    case "resolved":
      return "已处理";
    case "rejected":
      return "已驳回";
    case "pending":
      return "待处理";
    default:
      return String(value ?? "-");
  }
}


export function reportTone(value: unknown): Tone {
  if (String(value) === "resolved") return "green";
  if (String(value) === "rejected") return "red";
  return "amber";
}


export function visibilityLabel(value: unknown) {
  switch (String(value ?? "")) {
    case "public":
      return "公开";
    case "private":
      return "私密";
    case "friends_only":
      return "好友";
    default:
      return String(value ?? "-");
  }
}


export function qualityLevelLabel(value: unknown) {
  switch (String(value ?? "")) {
    case "high":
      return "高质量";
    case "medium":
      return "中质量";
    case "low":
      return "低质量";
    case "excellent":
      return "精选";
    case "good":
      return "优质";
    case "normal":
      return "普通";
    case "none":
    case "":
      return "未标记";
    default:
      return String(value);
  }
}


export function qualityTone(value: unknown): Tone {
  switch (String(value ?? "")) {
    case "high":
      return "red";
    case "medium":
      return "green";
    case "low":
      return "blue";
    case "excellent":
      return "red";
    case "good":
      return "green";
    case "normal":
      return "blue";
    default:
      return "slate";
  }
}


export function withdrawStatusLabel(value: unknown) {
  switch (String(value ?? "")) {
    case "pending":
      return "待审核";
    case "approved":
      return "已通过";
    case "rejected":
      return "已驳回";
    case "paid":
      return "已打款";
    default:
      return String(value ?? "-");
  }
}


export function withdrawTone(value: unknown): Tone {
  if (String(value) === "approved" || String(value) === "paid") return "green";
  if (String(value) === "rejected") return "red";
  return "amber";
}


export function formatCompact(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return numeric.toLocaleString("zh-CN");
}


export function formatMoney(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return numeric.toLocaleString("zh-CN", { maximumFractionDigits: 2, minimumFractionDigits: 0 });
}


export function formatDateTime(value: unknown) {
  if (!value || typeof value !== "string") return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}



export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}


export function columnLabel(key: string) {
  const labels: Record<string, string> = {
    id: "ID",
    user_id: "用户",
    user_display_id: "用户",
    nickname: "昵称",
    title: "标题",
    content: "内容",
    type: "类型",
    status: "状态",
    created_at: "创建时间",
    last_active_at: "最近活跃",
    updated_at: "更新时间",
    published_at: "发布时间",
    audit_time: "审核时间",
    reason: "原因",
    audit_result: "认证材料",
    name: "名称",
    url: "URL",
    is_active: "启用",
    is_published: "发布",
    show_popup: "弹窗",
    use_count: "使用量",
    post_count: "内容数",
    word_count: "词数",
    category_title: "分类标题",
    category_name: "分类",
    template_key: "模板键",
    license_key: "授权码",
    machine_model: "机器型号",
    machine_id: "机器码",
    expires_at: "过期时间",
    api_key_prefix: "Key 前缀",
    permissions: "权限",
    last_used_at: "最近使用",
    platform: "平台",
    version_name: "版本名",
    version_code: "版本号",
    size_bytes: "安装包大小",
    size_mb: "安装包大小",
    force_update: "强更",
    reward_amount: "奖励",
    quality_level: "质量等级",
    description: "说明",
  };
  return labels[key] ?? key.replaceAll("_", " ");
}


export function metricShortLabel(key: string) {
  const labels: Record<string, string> = {
    view_count: "看",
    like_count: "赞",
    comment_count: "评",
    collect_count: "藏",
  };
  return labels[key] ?? key;
}


export function pendingLabel(key: string) {
  const labels: Record<string, string> = {
    content_review: "内容待审核",
    audit: "认证待审核",
    reports: "举报待处理",
    withdraw: "提现待审核",
    feedback: "反馈待回复",
    missing_covers: "缺封面视频",
  };
  return labels[key] ?? key;
}
