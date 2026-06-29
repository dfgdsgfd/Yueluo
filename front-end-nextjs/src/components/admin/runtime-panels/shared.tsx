"use client";

import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export type Tone = "red" | "green" | "blue" | "purple" | "amber" | "slate";

export function Panel({ title, icon: Icon, action, children }: { title: string; icon: LucideIcon; action?: ReactNode; children: ReactNode }) {
  return (
    <section className="min-w-0 rounded-xl border border-black/[0.06] bg-white p-4 shadow-[0_10px_30px_rgba(17,24,39,0.04)]">
      <div className="mb-3 flex min-w-0 flex-wrap items-center gap-3">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <Icon className="size-5 shrink-0 text-[#2f7df6]" />
          <h2 className="truncate text-base font-semibold text-[#252932]">{title}</h2>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function HeaderCard({ icon: Icon, title, description, tone }: { icon: LucideIcon; title: string; description: string; tone: Tone }) {
  return (
    <div className="rounded-xl border border-black/[0.06] bg-white p-4">
      <div className="flex min-w-0 items-center gap-3">
        <span className={cn("flex size-11 shrink-0 items-center justify-center rounded-xl", toneSoftClass(tone))}>
          <Icon className="size-5" />
        </span>
        <div className="min-w-0">
          <h1 className="truncate text-lg font-bold text-[#252932]">{title}</h1>
          <p className="mt-1 text-sm text-[#7b808c]">{description}</p>
        </div>
      </div>
    </div>
  );
}

export function InfoTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg bg-[#f8fafc] px-3 py-2">
      <p className="truncate text-xs text-[#7b808c]">{label}</p>
      <p className="mt-1 truncate text-sm font-semibold text-[#252932]">{value}</p>
    </div>
  );
}

export function LoadingBlock({ label }: { label: string }) {
  return (
    <div className="flex h-40 items-center justify-center rounded-xl border border-dashed border-black/[0.08] bg-white text-sm text-[#7b808c]">
      <Loader2 className="mr-2 size-4 animate-spin" />
      {label}
    </div>
  );
}

export function EmptyBlock({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <div className="flex min-h-28 flex-col items-center justify-center rounded-xl border border-dashed border-black/[0.08] bg-[#f8fafc] text-sm text-[#7b808c]">
      <Icon className="mb-2 size-5" />
      {label}
    </div>
  );
}

export function KeyValueGrid({ entries }: { entries: Array<[string, unknown]> }) {
  return (
    <div className="grid gap-2">
      {entries.map(([key, value]) => (
        <div key={key} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
          <p className="text-xs font-semibold text-[#7b808c]">{key}</p>
          <div className="mt-1 break-all text-sm text-[#252932]">{renderReadableValue(value)}</div>
        </div>
      ))}
    </div>
  );
}

export function StatusPill({ value, tone }: { value: string; tone: Tone }) {
  return <span className={cn("inline-flex shrink-0 items-center rounded-full px-2 py-1 text-xs font-semibold", tonePillClass(tone))}>{value}</span>;
}

export function IconButton({ label, icon: Icon, onClick, danger }: { label: string; icon: LucideIcon; onClick: () => void; danger?: boolean }) {
  return (
    <button type="button" aria-label={label} title={label} onClick={onClick} className={cn("inline-flex size-8 items-center justify-center rounded-lg transition hover:bg-[#edf0f5]", danger ? "text-[#dc2626]" : "text-[#59606c]")}>
      <Icon className="size-4" />
    </button>
  );
}

export function fieldText(row: Record<string, unknown>, key: string) {
  return readableValue(row[key]);
}

export function readableValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function renderReadableValue(value: unknown): ReactNode {
  if (value === null || value === undefined || value === "") return <span className="text-[#9aa0aa]">-</span>;
  if (typeof value === "object") {
    return <pre className="max-h-80 overflow-auto whitespace-pre-wrap rounded-lg bg-[#f0f2f6] p-3 text-xs leading-5 text-[#252932]">{JSON.stringify(value, null, 2)}</pre>;
  }
  return <span>{String(value)}</span>;
}

export function isHiddenDetailKey(key: string, value: unknown) {
  return value === undefined || key.startsWith("_");
}

export function formatQueueTime(value: unknown) {
  if (value === null || value === undefined || value === "") return "-";
  const numeric = Number(value);
  const date = Number.isFinite(numeric) ? new Date(numeric < 1_000_000_000_000 ? numeric * 1000 : numeric) : new Date(String(value));
  if (Number.isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(date);
}

export function formatBytes(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  if (numeric < 1024) return `${numeric.toFixed(0)} B`;
  if (numeric < 1024 * 1024) return `${(numeric / 1024).toFixed(1)} KB`;
  if (numeric < 1024 * 1024 * 1024) return `${(numeric / 1024 / 1024).toFixed(1)} MB`;
  return `${(numeric / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

export function formatMegabytes(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return `${(numeric / 1024 / 1024).toFixed(1)} MB`;
}

export function formatNetworkRate(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  if (numeric < 1024 * 1024) return `${(numeric / 1024).toFixed(1)} KB/s`;
  return `${(numeric / 1024 / 1024).toFixed(1)} MB/s`;
}

export function formatCompact(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return numeric.toLocaleString("zh-CN");
}

export function formatDecimal(value: unknown, digits = 1) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return numeric.toFixed(digits);
}

export function formatMs(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return "-";
  return `${numeric.toFixed(numeric >= 10 ? 0 : 1)} ms`;
}

export function formatPercent(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return `${(numeric * 100).toFixed(1)}%`;
}

export function formatCpuPercent(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return `${numeric.toFixed(1)}%`;
}

export function recordArray(value: unknown): Array<Record<string, unknown>> {
  return Array.isArray(value) ? value.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === "object" && !Array.isArray(item)) : [];
}

export function recordObject(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
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

export function toneSoftClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "bg-[#fef2f2] text-[#dc2626]",
    green: "bg-[#eafff3] text-[#18a058]",
    blue: "bg-[#eef6ff] text-[#2f7df6]",
    purple: "bg-[#f3efff] text-[#7357d6]",
    amber: "bg-[#fff7e8] text-[#c97900]",
    slate: "bg-[#f0f2f6] text-[#5f6674]",
  };
  return classes[tone];
}

export function tonePillClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "bg-[#fff0f2] text-[#d71935]",
    green: "bg-[#eafff3] text-[#16824a]",
    blue: "bg-[#eef6ff] text-[#1e62ca]",
    purple: "bg-[#f3efff] text-[#6446c5]",
    amber: "bg-[#fff7e8] text-[#a96300]",
    slate: "bg-[#eef0f4] text-[#626977]",
  };
  return classes[tone];
}

export function toneTextClass(tone: Tone) {
  const classes: Record<Tone, string> = {
    red: "text-[#dc2626]",
    green: "text-[#18a058]",
    blue: "text-[#2f7df6]",
    purple: "text-[#7357d6]",
    amber: "text-[#c97900]",
    slate: "text-[#414856]",
  };
  return classes[tone];
}
