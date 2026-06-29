"use client";

import { useTranslations } from "next-intl";
import { ClipboardList, Eye } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import type { ObservabilityEventsPayload, ObservabilityEventType } from "./observability-types";
import { EmptyBlock, InfoTile, LoadingBlock, StatusPill, formatCompact, formatDateTime, formatMs, readableValue } from "./shared";

export function RedisStatus({ status }: { status?: Record<string, unknown> | null }) {
  const t = useTranslations("adminObservability");
  const available = Boolean(status?.available);
  const configured = Boolean(status?.configured);
  const info = (status?.info && typeof status.info === "object" && !Array.isArray(status.info) ? status.info : {}) as Record<string, unknown>;
  return (
    <div className={cn("rounded-lg border p-3 text-sm", available ? "border-[#18a058]/20 bg-[#f0fff6]" : "border-amber-200 bg-amber-50")}>
      <div className="flex items-center justify-between gap-3">
        <span className="font-semibold text-[#252932]">Redis</span>
        <StatusPill value={available ? t("statusAvailable") : configured ? t("statusUnavailable") : t("statusUnconfigured")} tone={available ? "green" : "amber"} />
      </div>
      <div className="mt-2 grid grid-cols-2 gap-2">
        <InfoTile label="PING" value={status?.pingMs === undefined ? "-" : formatMs(status.pingMs)} />
        <InfoTile label="DB" value={readableValue(status?.db ?? "-")} />
        <InfoTile label={t("memory")} value={readableValue(info.used_memory_human ?? "-")} />
        <InfoTile label={t("clients")} value={readableValue(info.connected_clients ?? "-")} />
      </div>
      {status?.message ? <p className="mt-2 break-all text-xs text-amber-900">{String(status.message)}</p> : null}
    </div>
  );
}

export function SlowList({
  title,
  items,
  primary,
  secondary,
}: {
  title: string;
  items: Array<Record<string, unknown>>;
  primary: (item: Record<string, unknown>) => string;
  secondary: (item: Record<string, unknown>) => string;
}) {
  const t = useTranslations("adminObservability");
  return (
    <div className="min-w-0 rounded-lg border border-black/[0.06] bg-white p-3">
      <p className="mb-2 text-sm font-semibold text-[#252932]">{title}</p>
      {items.length ? (
        <div className="grid gap-2">
          {items.slice(0, 8).map((item, index) => (
            <div key={String(item.id ?? item.request_id ?? item.fingerprint ?? index)} className="min-w-0 rounded-lg bg-[#f8fafc] px-3 py-2">
              <p className="line-clamp-2 break-all text-xs font-semibold text-[#252932]">{primary(item)}</p>
              <p className="mt-1 text-xs text-[#7b808c]">{secondary(item)}</p>
            </div>
          ))}
        </div>
      ) : (
        <EmptyBlock icon={ClipboardList} label={t("emptyData")} />
      )}
    </div>
  );
}

export function AccessLogList({ items, loading }: { items: Array<Record<string, unknown>>; loading: boolean }) {
  const t = useTranslations("adminObservability");
  if (loading && !items.length) return <LoadingBlock label={t("loadingAccessLogs")} />;
  if (!items.length) return <EmptyBlock icon={Eye} label={t("emptyAccessLogs")} />;
  return (
    <div className="max-h-[520px] overflow-y-auto pr-1">
      <div className="grid gap-2">
        {items.slice(0, 100).map((item, index) => (
          <article key={String(item.id ?? item.request_id ?? index)} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
            <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-[#7b808c]">
              <StatusPill value={String(item.status ?? "-")} tone={Number(item.status) >= 500 ? "red" : Number(item.status) >= 400 ? "amber" : "green"} />
              <span className="font-semibold text-[#252932]">{readableValue(item.method)}</span>
              <span>{formatMs(item.latency_ms)}</span>
              <span>{formatDateTime(item.created_at)}</span>
            </div>
            <p className="mt-1 line-clamp-1 break-all text-sm font-semibold text-[#252932]">{readableValue(item.path)}</p>
            <div className="mt-2 grid gap-1 text-xs text-[#7b808c] sm:grid-cols-2 xl:grid-cols-4">
              <span className="truncate">{t("accessIp")} {readableValue(item.ip)}</span>
              <span className="truncate">{t("accessUserId")} {readableValue(item.user_id)}</span>
              <span className="truncate">{t("accessUsername")} {readableValue(item.nickname ?? item.username)}</span>
              <span className="truncate">{t("accessUserType")} {readableValue(item.user_type)}</span>
            </div>
            <p className="mt-1 line-clamp-1 break-all text-xs text-[#9aa0aa]">{readableValue(item.user_agent)}</p>
          </article>
        ))}
      </div>
    </div>
  );
}

export function ObservabilityEventsTable({
  type,
  payload,
  loading,
  onPage,
}: {
  type: ObservabilityEventType;
  payload: ObservabilityEventsPayload | null;
  loading: boolean;
  onPage: (page: number) => void;
}) {
  const t = useTranslations("adminObservability");
  const items = payload?.items ?? [];
  const pagination = payload?.pagination ?? {};
  const page = Number(pagination.page ?? 1);
  const pages = Number(pagination.pages ?? pagination.totalPages ?? 1);
  if (loading && !items.length) return <LoadingBlock label={t("loadingEvents")} />;
  if (!items.length) return <EmptyBlock icon={ClipboardList} label={t("emptyEvents")} />;
  return (
    <div className="grid gap-3">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[760px] border-separate border-spacing-0 text-left text-sm">
          <thead>
            <tr className="text-xs text-[#7b808c]">
              <th className="border-b border-black/[0.06] px-3 py-2">{t("time")}</th>
              <th className="border-b border-black/[0.06] px-3 py-2">{type === "slow_queries" ? "SQL" : t("path")}</th>
              <th className="border-b border-black/[0.06] px-3 py-2">{t("status")}</th>
              <th className="border-b border-black/[0.06] px-3 py-2">{t("latency")}</th>
              <th className="border-b border-black/[0.06] px-3 py-2">{t("identity")}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item, index) => (
              <tr key={String(item.id ?? item.request_id ?? item.fingerprint ?? index)} className="align-top">
                <td className="border-b border-black/[0.04] px-3 py-2 text-xs text-[#7b808c]">{formatDateTime(item.created_at)}</td>
                <td className="max-w-[420px] border-b border-black/[0.04] px-3 py-2">
                  <p className="line-clamp-2 break-all font-semibold text-[#252932]">{type === "slow_queries" ? readableValue(item.sql) : `${readableValue(item.method)} ${readableValue(item.route ?? item.path)}`}</p>
                  {item.error ? <p className="mt-1 line-clamp-2 break-all text-xs text-[#dc2626]">{readableValue(item.error)}</p> : null}
                </td>
                <td className="border-b border-black/[0.04] px-3 py-2 text-xs text-[#7b808c]">{readableValue(item.status ?? item.rows)}</td>
                <td className="border-b border-black/[0.04] px-3 py-2 font-semibold text-[#dc2626]">{formatMs(item.latency_ms)}</td>
                <td className="max-w-[180px] border-b border-black/[0.04] px-3 py-2 text-xs text-[#7b808c]">
                  <span className="line-clamp-2 break-all">{readableValue(item.request_id ?? item.fingerprint ?? item.id)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-[#7b808c]">
        <span>{t("pageSummary", { total: pagination.total ?? items.length, page, pages: Math.max(1, pages) })}</span>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" disabled={loading || page <= 1} onClick={() => onPage(page - 1)} className="h-8 rounded-lg border-black/[0.08] bg-white px-3">{t("previousPage")}</Button>
          <Button type="button" variant="outline" disabled={loading || page >= pages} onClick={() => onPage(page + 1)} className="h-8 rounded-lg border-black/[0.08] bg-white px-3">{t("nextPage")}</Button>
        </div>
      </div>
    </div>
  );
}

export function observabilityEventTypeLabel(type: ObservabilityEventType, t: (key: string) => string) {
  if (type === "errors") return t("errors");
  if (type === "slow_requests") return t("slowRequests");
  return t("slowQueries");
}

export function slowRequestSecondary(item: Record<string, unknown>) {
  return `${formatMs(item.latency_ms)} · ${item.status ?? "-"} · ${formatDateTime(item.created_at)}`;
}

export function slowQuerySecondary(item: Record<string, unknown>, rowsLabel: string) {
  return `${formatMs(item.latency_ms)} · ${rowsLabel} ${formatCompact(item.rows ?? "-")} · ${formatDateTime(item.created_at)}`;
}
