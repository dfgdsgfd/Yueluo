"use client";

import { ChevronLeft, ChevronRight, Eye, EyeOff, Users } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { AdminAccessLogRankItem, AdminLogListPayload } from "@/lib/types";
import { formatCompact } from "./helpers";
import { pageSizeOptions } from "./logs-panel-types";
import type { AdminLogsTranslator, VisitorMode } from "./logs-panel-types";

export function LogPagination({
  pagination,
  page,
  pageSize,
  loading,
  t,
  onPageChange,
  onPageSizeChange,
}: {
  pagination?: AdminLogListPayload<unknown>["pagination"];
  page: number;
  pageSize: number;
  loading: boolean;
  t: AdminLogsTranslator;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const total = Number(pagination?.total ?? 0);
  const declaredPages = pagination?.totalPages ?? pagination?.pages;
  const totalPages = Math.max(1, Number(declaredPages ?? (Math.ceil(total / pageSize) || 1)));
  const currentPage = Math.min(Math.max(1, Number(pagination?.page ?? page)), totalPages);
  return (
    <div className="mt-3 flex flex-col gap-3 border-t border-black/[0.06] pt-3 text-sm text-[#687080] sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-wrap items-center gap-3">
        <span>{t("pageSummary", { page: currentPage, pages: totalPages, total })}</span>
        <label className="flex items-center gap-2">
          <span>{t("perPage")}</span>
          <select
            value={pageSize}
            disabled={loading}
            onChange={(event) => onPageSizeChange(Number(event.target.value))}
            className="h-9 rounded-lg border border-black/[0.08] bg-white px-2 text-sm text-[#30343b] outline-none focus:border-[#1d4ed8]"
            aria-label={t("perPage")}
          >
            {pageSizeOptions.map((option) => <option key={option} value={option}>{option}</option>)}
          </select>
        </label>
      </div>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={loading || currentPage <= 1}
          onClick={() => onPageChange(currentPage - 1)}
          className="rounded-lg border-black/[0.08] bg-white"
        >
          <ChevronLeft className="size-4" />
          {t("previousPage")}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={loading || currentPage >= totalPages}
          onClick={() => onPageChange(currentPage + 1)}
          className="rounded-lg border-black/[0.08] bg-white"
        >
          {t("nextPage")}
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}

const visitorIcons: Record<VisitorMode, typeof Eye> = { all: Users, exclude: EyeOff, only: Eye };
const visitorLabels: Record<VisitorMode, string> = { all: "visitorAll", exclude: "visitorExclude", only: "visitorOnly" };

export function VisitorToggle({ value, onChange }: { value: VisitorMode; onChange: (value: VisitorMode) => void }) {
  const t = useTranslations("adminLogs");
  const modes: VisitorMode[] = ["exclude", "only", "all"];
  return (
    <div className="flex items-center gap-0.5 rounded-lg border border-black/[0.08] bg-white p-0.5">
      {modes.map((mode) => {
        const Icon = visitorIcons[mode];
        return (
          <button
            key={mode}
            type="button"
            onClick={() => onChange(mode)}
            className={cn(
              "flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors",
              value === mode
                ? "bg-[#eff6ff] text-[#1e3a8a]"
                : "text-[#7a8190] hover:text-[#3f444e]",
            )}
            title={t(visitorLabels[mode])}
          >
            <Icon className="size-3.5" />
            <span className="hidden sm:inline">{t(visitorLabels[mode])}</span>
          </button>
        );
      })}
    </div>
  );
}

export function logButtonClass(active: boolean, extra?: string) {
  return cn(
    "rounded-lg border px-3 text-sm",
    active
      ? "border-[#1d4ed8]/30 bg-[#eff6ff] text-[#1e3a8a] hover:bg-[#dbeafe]"
      : "border-black/[0.08] bg-white text-[#3f444e] hover:bg-[#f8fafc]",
    extra,
  );
}

export function LogMetric({ label, value, compact = false }: { label: string; value: unknown; compact?: boolean }) {
  const display = typeof value === "number" ? formatCompact(value) : String(value ?? "-");
  return (
    <div className={compact ? "flex items-center justify-between rounded-lg bg-[#f8fafc] px-3 py-2 text-sm" : "rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_35px_rgba(20,20,30,0.05)]"}>
      <span className={compact ? "text-[#606775]" : "text-xs font-semibold uppercase tracking-normal text-[#7a8190]"}>{label}</span>
      <span className={compact ? "font-semibold text-[#17171d]" : "mt-2 block text-2xl font-semibold text-[#17171d]"}>{display}</span>
    </div>
  );
}

export function RankingList({ title, items }: { title: string; items: AdminAccessLogRankItem[] }) {
  const t = useTranslations("adminLogs");
  const maxCount = Math.max(...items.map((item) => Number(item.count ?? 0)), 1);
  return (
    <div className="rounded-lg border border-black/[0.05] bg-[#fafbfe] p-3">
      <h3 className="text-sm font-semibold text-[#252932]">{title}</h3>
      <div className="mt-3 grid gap-2">
        {items.length ? items.map((item) => {
          const count = Number(item.count ?? 0);
          const titleText = item.title ?? item.post_title ?? "";
          const idText = String(item.label || item.key || "-");
          return (
            <div key={`${title}-${item.key ?? item.label}`} className="min-w-0">
              <div className="mb-1 flex items-start justify-between gap-2 text-xs">
                <span className="min-w-0 text-[#5d6472]">
                  {titleText ? (
                    <>
                      <span className="line-clamp-2 break-all font-semibold text-[#252932]" title={titleText}>{titleText}</span>
                      <span className="mt-0.5 block truncate text-[#8a90a0]">{idText}</span>
                    </>
                  ) : (
                    <span className="block truncate">{idText}</span>
                  )}
                </span>
                <span className="shrink-0 font-semibold text-[#252932]">{formatCompact(count)}</span>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-[#e8edf5]">
                <div className="h-full rounded-full bg-[#2563eb]" style={{ width: `${Math.max(4, Math.round((count / maxCount) * 100))}%` }} />
              </div>
            </div>
          );
        }) : <p className="text-sm text-[#7a8190]">{t("emptyRanking")}</p>}
      </div>
    </div>
  );
}
