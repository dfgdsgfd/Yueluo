"use client";

import { useTranslations } from "next-intl";
import { Activity, Database } from "lucide-react";

import { EmptyBlock, Panel, formatCompact, formatDateTime, formatMs, formatPercent, readableValue } from "./shared";

export function EndpointRankingPanel({ items }: { items: Array<Record<string, unknown>> }) {
  const t = useTranslations("adminObservability");
  return (
    <Panel title={t("endpointRankingTitle")} icon={Activity}>
      {items.length ? (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[920px] border-separate border-spacing-0 text-left text-sm">
            <thead>
              <tr className="text-xs text-[#7b808c]">
                <th className="border-b border-black/[0.06] px-3 py-2">{t("endpoint")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("count")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">P95</th>
                <th className="border-b border-black/[0.06] px-3 py-2">P99</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("maxLatency")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("errorRate")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">4xx</th>
                <th className="border-b border-black/[0.06] px-3 py-2">5xx</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("slow")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("lastSeen")}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, index) => (
                <tr key={String(item.method ?? "") + String(item.route ?? item.path ?? index)} className="align-top">
                  <td className="max-w-[320px] border-b border-black/[0.04] px-3 py-2">
                    <p className="line-clamp-1 break-all font-semibold text-[#252932]">{readableValue(item.method)} {readableValue(item.route ?? item.path)}</p>
                    <p className="mt-1 line-clamp-1 break-all text-xs text-[#7b808c]">{readableValue(item.sample_path)}</p>
                  </td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.count)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.p95_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.p99_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.max_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatPercent(item.error_rate)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.status_4xx)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.status_5xx)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.slow_count)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2 text-xs text-[#7b808c]">{formatDateTime(item.last_seen)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyBlock icon={Activity} label={t("emptyEndpointRanking")} />
      )}
    </Panel>
  );
}

export function SlowQueryGroupsPanel({ items }: { items: Array<Record<string, unknown>> }) {
  const t = useTranslations("adminObservability");
  return (
    <Panel title={t("slowSqlGroupsTitle")} icon={Database}>
      {items.length ? (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px] border-separate border-spacing-0 text-left text-sm">
            <thead>
              <tr className="text-xs text-[#7b808c]">
                <th className="border-b border-black/[0.06] px-3 py-2">{t("sqlPattern")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("count")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("avgLatency")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">P95</th>
                <th className="border-b border-black/[0.06] px-3 py-2">P99</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("maxLatency")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("rows")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("errorCount")}</th>
                <th className="border-b border-black/[0.06] px-3 py-2">{t("lastSeen")}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, index) => (
                <tr key={String(item.fingerprint ?? index)} className="align-top">
                  <td className="max-w-[360px] border-b border-black/[0.04] px-3 py-2">
                    <p className="line-clamp-2 break-all font-semibold text-[#252932]">{readableValue(item.sample_sql)}</p>
                    <p className="mt-1 line-clamp-1 break-all text-xs text-[#7b808c]">{readableValue(item.fingerprint)}</p>
                  </td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.count)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.avg_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.p95_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.p99_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatMs(item.max_latency_ms)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.rows)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2">{formatCompact(item.error_count)}</td>
                  <td className="border-b border-black/[0.04] px-3 py-2 text-xs text-[#7b808c]">{formatDateTime(item.last_seen)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyBlock icon={Database} label={t("emptySlowSqlGroups")} />
      )}
    </Panel>
  );
}
