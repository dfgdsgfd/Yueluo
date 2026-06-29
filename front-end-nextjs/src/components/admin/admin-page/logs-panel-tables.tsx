"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import type { AdminAccessLogItem, AdminBalanceAuditLogItem, AdminPointsAuditLogItem, AdminSecurityAuditLogItem } from "@/lib/types";
import { formatCompact, formatDateTime } from "./helpers";
import type { AdminLogsTranslator } from "./logs-panel-types";

export function PointsAuditTable({ items }: { items: AdminPointsAuditLogItem[] }) {
  const t = useTranslations("adminLogs");
  if (!items.length) return <p className="rounded-lg bg-[#f8fafc] px-3 py-6 text-center text-sm text-[#7a8190]">{t("emptyPointsLogs")}</p>;
  return (
    <div className="overflow-x-auto">
      <table className="min-w-[1280px] w-full text-left text-sm">
        <thead className="text-xs uppercase tracking-normal text-[#7a8190]">
          <tr>
            <th className="px-3 py-2">{t("time")}</th>
            <th className="px-3 py-2">{t("user")}</th>
            <th className="px-3 py-2">{t("type")}</th>
            <th className="px-3 py-2">{t("entryRole")}</th>
            <th className="px-3 py-2">{t("post")}</th>
            <th className="px-3 py-2">{t("counterparty")}</th>
            <th className="px-3 py-2">{t("amount")}</th>
            <th className="px-3 py-2">{t("balanceAfter")}</th>
            <th className="px-3 py-2">{t("reason")}</th>
            <th className="px-3 py-2">{t("auditFlag")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={String(item.id)} className={cn("border-t border-black/[0.05]", item.is_anomaly && "bg-red-50/70")}>
              <td className="whitespace-nowrap px-3 py-2 text-[#59606c]">{formatDateTime(item.created_at)}</td>
              <td className="px-3 py-2">
                <span className="block font-semibold text-[#252932]">{item.nickname || item.user_display_id || `#${item.user_id}`}</span>
                <span className="text-xs text-[#8a90a0]">{item.user_display_id || item.user_id || "-"}</span>
              </td>
              <td className="px-3 py-2 text-[#59606c]">{item.type || "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{ledgerRoleLabel(item.entry_role, t)}</td>
              <td className="max-w-[260px] px-3 py-2 text-[#59606c]">
                <span className="block break-words">{item.post_title || "-"}</span>
                {item.post_id ? <span className="text-xs text-[#8a90a0]">#{item.post_id}</span> : null}
              </td>
              <td className="px-3 py-2 text-[#59606c]">{counterpartyLabel(item.counterparty)}</td>
              <td className={cn("px-3 py-2 font-semibold", Number(item.amount ?? 0) < 0 ? "text-red-600" : "text-emerald-600")}>{formatAuditAmount(item.amount)}</td>
              <td className="px-3 py-2 text-[#59606c]">{formatAuditAmount(item.balance_after, false)}</td>
              <td className="max-w-[300px] break-words px-3 py-2 text-[#59606c]">{item.reason || "-"}</td>
              <td className="px-3 py-2">{item.is_anomaly ? <AuditBadge anomaly label={t("anomaly")} /> : <AuditBadge label={t("normal")} />}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function BalanceAuditTable({ items }: { items: AdminBalanceAuditLogItem[] }) {
  const t = useTranslations("adminLogs");
  if (!items.length) return <p className="rounded-lg bg-[#f8fafc] px-3 py-6 text-center text-sm text-[#7a8190]">{t("emptyBalanceLogs")}</p>;
  return (
    <div className="overflow-x-auto">
      <table className="min-w-[1660px] w-full text-left text-sm">
        <thead className="text-xs uppercase tracking-normal text-[#7a8190]">
          <tr>
            <th className="px-3 py-2">{t("time")}</th>
            <th className="px-3 py-2">{t("user")}</th>
            <th className="px-3 py-2">{t("oauth2Id")}</th>
            <th className="px-3 py-2">{t("entryRole")}</th>
            <th className="px-3 py-2">{t("post")}</th>
            <th className="px-3 py-2">{t("counterparty")}</th>
            <th className="px-3 py-2">{t("amount")}</th>
            <th className="px-3 py-2">{t("status")}</th>
            <th className="px-3 py-2">{t("balanceAfter")}</th>
            <th className="px-3 py-2">{t("platformFee")}</th>
            <th className="px-3 py-2">{t("attempts")}</th>
            <th className="px-3 py-2">{t("reason")}</th>
            <th className="px-3 py-2">{t("lastError")}</th>
            <th className="px-3 py-2">{t("operationKey")}</th>
            <th className="px-3 py-2">{t("auditFlag")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={String(item.id)} className={cn("border-t border-black/[0.05]", item.is_anomaly && "bg-red-50/70")}>
              <td className="whitespace-nowrap px-3 py-2 text-[#59606c]">{formatDateTime(item.created_at)}</td>
              <td className="px-3 py-2">
                <span className="block font-semibold text-[#252932]">{item.nickname || item.user_display_id || `#${item.user_id}`}</span>
                <span className="text-xs text-[#8a90a0]">{item.user_display_id || item.user_id || "-"}</span>
              </td>
              <td className="px-3 py-2 text-[#59606c]">{item.oauth2_id ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{ledgerRoleLabel(item.entry_role, t)}</td>
              <td className="max-w-[260px] px-3 py-2 text-[#59606c]">
                <span className="block break-words">{item.post_title || "-"}</span>
                {item.post_id ? <span className="text-xs text-[#8a90a0]">#{item.post_id}</span> : null}
              </td>
              <td className="px-3 py-2 text-[#59606c]">{counterpartyLabel(item.counterparty)}</td>
              <td className={cn("px-3 py-2 font-semibold", Number(item.amount ?? 0) < 0 ? "text-red-600" : "text-emerald-600")}>{formatAuditAmount(item.amount)}</td>
              <td className="px-3 py-2"><AuditBadge anomaly={item.is_anomaly} label={item.status || "-"} /></td>
              <td className="px-3 py-2 text-[#59606c]">{item.remote_balance_after == null ? "-" : formatAuditAmount(item.remote_balance_after, false)}</td>
              <td className="px-3 py-2 text-[#59606c]">{formatAuditAmount(item.platform_fee, false)}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.attempts ?? 0}</td>
              <td className="max-w-[260px] break-words px-3 py-2 text-[#59606c]">{item.reason || "-"}</td>
              <td className="max-w-[300px] break-words px-3 py-2 text-red-700">{item.last_error || "-"}</td>
              <td className="max-w-[260px] truncate px-3 py-2 font-mono text-xs text-[#59606c]" title={item.operation_key || ""}>{item.operation_key || "-"}</td>
              <td className="px-3 py-2">{item.is_anomaly ? <AuditBadge anomaly label={t("anomaly")} /> : <AuditBadge label={t("normal")} />}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ledgerRoleLabel(role: string | null | undefined, t: AdminLogsTranslator) {
  switch (role) {
    case "buyer_debit":
      return t("buyerExpense");
    case "author_credit":
      return t("authorIncome");
    default:
      return role || "-";
  }
}

function counterpartyLabel(counterparty: { user_id?: number | string | null; user_display_id?: string | null; nickname?: string | null } | null | undefined) {
  if (!counterparty) return "-";
  return counterparty.nickname || counterparty.user_display_id || (counterparty.user_id ? `#${counterparty.user_id}` : "-");
}

function AuditBadge({ anomaly = false, label }: { anomaly?: boolean; label: string }) {
  return (
    <span className={cn(
      "inline-flex rounded-full px-2 py-1 text-xs font-semibold",
      anomaly ? "bg-red-100 text-red-700" : "bg-emerald-100 text-emerald-700",
    )}>
      {label}
    </span>
  );
}

function formatAuditAmount(value: unknown, signed = true) {
  const amount = Number(value ?? 0);
  if (!Number.isFinite(amount)) return "-";
  const formatted = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(amount);
  return signed && amount > 0 ? `+${formatted}` : formatted;
}

function ipLabel(item: { ip?: string | null; country_code?: string | null; country_name?: string | null; country_flag?: string | null }) {
  const ip = item.ip ?? "-";
  const code = item.country_code?.trim();
  const name = item.country_name?.trim();
  const flag = item.country_flag?.trim();
  if (!code && !name && !flag) return ip;
  const prefix = [flag, code ? `[${code}]` : null].filter(Boolean).join(" ");
  return `${prefix ? `${prefix} ` : ""}${ip}${name ? ` · ${name}` : ""}`;
}

export function AccessLogTable({ items }: { items: AdminAccessLogItem[] }) {
  const t = useTranslations("adminLogs");
  if (!items.length) return <p className="rounded-lg bg-[#f8fafc] px-3 py-6 text-center text-sm text-[#7a8190]">{t("emptyLogs")}</p>;
  return (
    <div className="overflow-x-auto">
      <table className="min-w-[1180px] w-full text-left text-sm">
        <thead className="text-xs uppercase tracking-normal text-[#7a8190]">
          <tr>
            <th className="px-3 py-2">{t("time")}</th>
            <th className="px-3 py-2">{t("behavior")}</th>
            <th className="px-3 py-2">{t("user")}</th>
            <th className="px-3 py-2">{t("ip")}</th>
            <th className="px-3 py-2">{t("language")}</th>
            <th className="px-3 py-2">{t("ua")}</th>
            <th className="px-3 py-2">{t("path")}</th>
            <th className="px-3 py-2">{t("status")}</th>
            <th className="px-3 py-2">{t("latency")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={String(item.id)} className="border-t border-black/[0.05]">
              <td className="whitespace-nowrap px-3 py-2 text-[#59606c]">{formatDateTime(item.created_at)}</td>
              <td className="px-3 py-2 font-semibold text-[#252932]">{item.behavior ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.user_display_id ?? item.principal_type ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{ipLabel(item)}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.browser_language ?? "-"}</td>
              <td className="max-w-[260px] truncate px-3 py-2 text-[#59606c]" title={item.user_agent ?? ""}>{item.user_agent ?? "-"}</td>
              <td className="max-w-[280px] truncate px-3 py-2 text-[#59606c]">{item.method} {item.path}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.status ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{formatCompact(Number(item.latency_ms ?? 0))}ms</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function SecurityLogTable({ items }: { items: AdminSecurityAuditLogItem[] }) {
  const t = useTranslations("adminLogs");
  if (!items.length) return <p className="rounded-lg bg-[#f8fafc] px-3 py-6 text-center text-sm text-[#7a8190]">{t("emptyLogs")}</p>;
  return (
    <div className="overflow-x-auto">
      <table className="min-w-[1320px] w-full text-left text-sm">
        <thead className="text-xs uppercase tracking-normal text-[#7a8190]">
          <tr>
            <th className="px-3 py-2">{t("time")}</th>
            <th className="px-3 py-2">{t("category")}</th>
            <th className="px-3 py-2">{t("action")}</th>
            <th className="px-3 py-2">{t("outcome")}</th>
            <th className="px-3 py-2">{t("actor")}</th>
            <th className="px-3 py-2">{t("ip")}</th>
            <th className="px-3 py-2">{t("ua")}</th>
            <th className="px-3 py-2">{t("reason")}</th>
            <th className="px-3 py-2">{t("metadata")}</th>
            <th className="px-3 py-2">{t("path")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={String(item.id)} className="border-t border-black/[0.05]">
              <td className="whitespace-nowrap px-3 py-2 text-[#59606c]">{formatDateTime(item.created_at)}</td>
              <td className="px-3 py-2 font-semibold text-[#252932]">{item.category ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.action ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.outcome ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.actor_display_id ?? item.actor_type ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{ipLabel(item)}</td>
              <td className="max-w-[260px] truncate px-3 py-2 text-[#59606c]" title={item.user_agent ?? ""}>{item.user_agent ?? "-"}</td>
              <td className="px-3 py-2 text-[#59606c]">{item.reason_code ?? "-"}</td>
              <td className="max-w-[260px] truncate px-3 py-2 text-[#59606c]" title={metadataTitle(item.metadata)}>{metadataSummary(item.metadata, t)}</td>
              <td className="max-w-[280px] truncate px-3 py-2 text-[#59606c]">{item.method} {item.path}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function metadataSummary(metadata: Record<string, unknown> | null | undefined, t: AdminLogsTranslator) {
  const root = metadataRecord(metadata);
  if (!root) return "-";
  const fileDelete = metadataRecord(root.file_delete);
  if (fileDelete) {
    return t("fileDeleteSummary", {
      deleted: metadataNumber(fileDelete.deleted_count),
      failed: metadataNumber(fileDelete.failed_count),
      skipped: metadataNumber(fileDelete.skipped_count),
      missing: metadataNumber(fileDelete.missing_count),
    });
  }
  const source = metadataText(root.source);
  if (source) return source;
  const text = metadataTitle(root);
  return text.length > 120 ? `${text.slice(0, 117)}...` : text || "-";
}

function metadataTitle(metadata: Record<string, unknown> | null | undefined) {
  if (!metadata) return "";
  try {
    return JSON.stringify(metadata);
  } catch {
    return "";
  }
}

function metadataRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : null;
}

function metadataNumber(value: unknown) {
  const numberValue = Number(value ?? 0);
  return Number.isFinite(numberValue) ? numberValue : 0;
}

function metadataText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}
