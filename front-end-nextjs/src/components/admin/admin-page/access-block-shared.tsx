"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export type AccessBlockRule = {
  id: number | string;
  import_source_id?: number;
  kind: "ip" | "ua" | string;
  match_type: "ip" | "cidr" | "ua_contains" | "ua_regex" | string;
  pattern: string;
  enabled: boolean;
  priority: number;
  action: "status" | "redirect" | string;
  status_code: number;
  redirect_url?: string;
  note?: string;
  created_at?: string;
  updated_at?: string;
};

export type AccessBlockImportSource = {
  id: number | string;
  url: string;
  enabled: boolean;
  priority: number;
  action: "status" | "redirect" | string;
  status_code: number;
  redirect_url?: string;
  note?: string;
  update_interval_seconds: number;
  last_sync_at?: string;
  next_sync_at?: string;
  last_status?: string;
  last_error?: string;
  last_count?: number;
  created_at?: string;
  updated_at?: string;
};

export type AccessBlockDraft = {
  kind: "ip" | "ua";
  match_type: "ip" | "cidr" | "ua_contains" | "ua_regex";
  pattern: string;
  batch: string;
  enabled: boolean;
  priority: number;
  action: "status" | "redirect";
  status_code: number;
  redirect_url: string;
  note: string;
  force: boolean;
};

export type AccessBlockImportDraft = {
  url: string;
  enabled: boolean;
  priority: number;
  action: "status" | "redirect";
  status_code: number;
  redirect_url: string;
  note: string;
  update_interval_minutes: number;
  force: boolean;
};

export const defaultDraft: AccessBlockDraft = {
  kind: "ip",
  match_type: "cidr",
  pattern: "",
  batch: "",
  enabled: true,
  priority: 1000,
  action: "status",
  status_code: 444,
  redirect_url: "",
  note: "",
  force: false,
};

export const defaultImportDraft: AccessBlockImportDraft = {
  url: "",
  enabled: true,
  priority: 1000,
  action: "status",
  status_code: 444,
  redirect_url: "",
  note: "",
  update_interval_minutes: 60,
  force: false,
};

export function importSourceBody(draft: AccessBlockImportDraft) {
  return {
    url: draft.url,
    enabled: draft.enabled,
    priority: draft.priority,
    action: draft.action,
    status_code: draft.status_code,
    redirect_url: draft.redirect_url,
    note: draft.note,
    update_interval_seconds: Math.max(300, Math.round(draft.update_interval_minutes * 60)),
  };
}

export function actionLabel(rule: Pick<AccessBlockRule, "action" | "status_code" | "redirect_url">, t: (key: string, values?: Record<string, string | number>) => string) {
  if (rule.action === "redirect") return t("actionRedirect", { url: rule.redirect_url || "-" });
  return t("actionStatus", { code: rule.status_code || 444 });
}

export function accessBlockMatchType(rule: AccessBlockRule): AccessBlockDraft["match_type"] {
  if (rule.match_type === "ip" || rule.match_type === "cidr" || rule.match_type === "ua_regex") return rule.match_type;
  return "ua_contains";
}

export function statusLabel(status: string | undefined, t: (key: string) => string) {
  if (status === "success" || status === "failed") return t(`importStatuses.${status}`);
  return t("importStatuses.pending");
}

export function formatDateTime(value: string | undefined) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString();
}

export function Badge({ children, tone = "slate" }: { children: ReactNode; tone?: "red" | "blue" | "green" | "slate" }) {
  return (
    <span className={cn(
      "inline-flex rounded-full px-2 py-1 text-xs font-semibold",
      tone === "red" && "bg-red-100 text-red-700",
      tone === "blue" && "bg-blue-100 text-blue-700",
      tone === "green" && "bg-emerald-100 text-emerald-700",
      tone === "slate" && "bg-slate-100 text-slate-700",
    )}>
      {children}
    </span>
  );
}

export function InputField({ label, onChange, placeholder, type = "text", value }: { label: string; onChange: (value: string) => void; placeholder?: string; type?: string; value: string }) {
  return (
    <label className="grid min-w-0 gap-1.5">
      <span className="text-xs font-bold text-[#5f636d]">{label}</span>
      <input type={type} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#dc2626]" />
    </label>
  );
}

export function SelectField({ children, label, onChange, value }: { children: ReactNode; label: string; onChange: (value: string) => void; value: string }) {
  return (
    <label className="grid min-w-0 gap-1.5">
      <span className="text-xs font-bold text-[#5f636d]">{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)} className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#dc2626]">
        {children}
      </select>
    </label>
  );
}
