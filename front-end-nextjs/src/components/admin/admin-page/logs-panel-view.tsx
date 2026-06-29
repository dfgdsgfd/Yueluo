"use client";

import { Activity, BarChart3, CircleDollarSign, ClipboardList, Coins, RefreshCw, Search, ShieldCheck, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AdminPieChart } from "@/components/admin/admin-pie-chart";
import { AdminTimeSeriesChart } from "@/components/admin/admin-time-series-chart";
import { HeaderCard, Panel } from "./layout-widgets";
import { LoadingBlock } from "./resource-editor";
import { formatCompact } from "./helpers";
import { AccessLogTable, BalanceAuditTable, PointsAuditTable, SecurityLogTable } from "./logs-panel-tables";
import { LogMetric, LogPagination, RankingList, VisitorToggle, logButtonClass } from "./logs-panel-widgets";
import { rangeOptions } from "./logs-panel-types";
import type { LogTab } from "./logs-panel-types";
import type { LogsPanelController } from "./use-logs-panel-controller";

export function LogsPanelView({ controller }: { controller: LogsPanelController }) {
  const {
    t,
    range,
    setRange,
    visitor,
    setVisitor,
    tab,
    setTab,
    keywordDraft,
    setKeywordDraft,
    anomaliesOnly,
    setAnomaliesOnly,
    pageByTab,
    setPageByTab,
    pageSize,
    setPageSize,
    analytics,
    accessLogs,
    operationLogs,
    securityLogs,
    systemLogs,
    pointsLogs,
    balanceLogs,
    loading,
    load,
    totals,
    status,
    chartSeries,
    activeLogs,
    resetPages,
    applySearch,
  } = controller;

    return (
    <div className="grid min-w-0 gap-4">
      <HeaderCard icon={ShieldCheck} title={t("title")} description={t("description")} tone="blue" />

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <LogMetric label={t("pv")} value={totals.pv} />
        <LogMetric label={t("activeUsers")} value={totals.active_users} />
        <LogMetric label={t("uniqueIps")} value={totals.unique_ips} />
        <LogMetric label={t("postViews")} value={totals.post_views} />
        <LogMetric label={t("securityEvents")} value={totals.security_events} />
      </section>

      <Panel
        title={t("activityTrend")}
        icon={Activity}
        action={
          <div className="flex flex-wrap items-center gap-2">
            {rangeOptions.map((item) => (
              <Button
                key={item}
                type="button"
                variant="outline"
                onClick={() => {
                  setRange(item);
                  resetPages();
                }}
                className={logButtonClass(range === item, "h-8 px-3 text-xs")}
              >
                {t(`ranges.${item}`)}
              </Button>
            ))}
            <span className="mx-1 h-5 w-px bg-black/[0.08]" />
            <VisitorToggle
              value={visitor}
              onChange={(nextVisitor) => {
                setVisitor(nextVisitor);
                resetPages();
              }}
            />
            <Button type="button" variant="outline" disabled={loading} onClick={() => void load()} className={logButtonClass(false, "h-8 px-3 text-xs")}>
              <RefreshCw className={loading ? "size-3.5 animate-spin" : "size-3.5"} />
              {t("refresh")}
            </Button>
          </div>
        }
      >
        <AdminTimeSeriesChart data={analytics?.series ?? []} series={chartSeries} height={320} valueFormatter={(value) => formatCompact(value)} />
      </Panel>

      <section className="grid min-w-0 gap-4 lg:grid-cols-2">
        <Panel title={t("countryDistribution")} icon={BarChart3}>
          <AdminPieChart data={analytics?.rankings?.countries ?? []} />
        </Panel>
        <Panel title={t("languageDistribution")} icon={BarChart3}>
          <AdminPieChart data={analytics?.rankings?.languages ?? []} />
        </Panel>
      </section>

      <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Panel title={t("rankings")} icon={BarChart3}>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <RankingList title={t("rankingPaths")} items={analytics?.rankings?.paths ?? []} />
            <RankingList title={t("rankingPosts")} items={analytics?.rankings?.posts ?? []} />
            <RankingList title={t("rankingUsers")} items={analytics?.rankings?.users ?? []} />
            <RankingList title={t("rankingIps")} items={analytics?.rankings?.ips ?? []} />
            <RankingList title={t("rankingCountries")} items={analytics?.rankings?.countries ?? []} />
            <RankingList title={t("rankingLanguages")} items={analytics?.rankings?.languages ?? []} />
            <RankingList title={t("rankingBehaviors")} items={analytics?.rankings?.behaviors ?? []} />
            <RankingList title={t("rankingDeviceUas")} items={analytics?.rankings?.device_uas ?? []} />
            <RankingList title={t("rankingBrowsers")} items={analytics?.rankings?.browsers ?? []} />
            <RankingList title={t("rankingOperatingSystems")} items={analytics?.rankings?.operating_systems ?? []} />
          </div>
        </Panel>
        <Panel title={t("pipelineStatus")} icon={ClipboardList}>
          <div className="grid gap-2">
            <LogMetric label={t("queueAvailable")} value={status.available ? t("yes") : t("no")} compact />
            <LogMetric label={t("accessBuffered")} value={status.access_buffered ?? 0} compact />
            <LogMetric label={t("securityBuffered")} value={status.security_buffered ?? 0} compact />
            <LogMetric label={t("accessDropped")} value={status.access_dropped ?? 0} compact />
            <LogMetric label={t("securityDropped")} value={status.security_dropped ?? 0} compact />
            <LogMetric label={t("enqueueFailures")} value={status.enqueue_failures ?? 0} compact />
          </div>
        </Panel>
      </section>

      <Panel
        title={t("details")}
        icon={Search}
        action={
          <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto">
            <div className="flex h-9 min-w-[220px] flex-1 items-center gap-2 rounded-lg border border-black/[0.08] bg-white px-3 sm:w-72 sm:flex-none">
              <Search className="size-4 shrink-0 text-[#8a8f9d]" />
              <input
                value={keywordDraft}
                onChange={(event) => setKeywordDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") applySearch();
                }}
                className="min-w-0 flex-1 bg-transparent text-sm outline-none"
                placeholder={t("searchPlaceholder")}
              />
            </div>
            <Button type="button" variant="outline" onClick={applySearch} className={logButtonClass(false, "h-9 px-3")}>
              {t("query")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setAnomaliesOnly((value) => !value);
                resetPages();
              }}
              className={logButtonClass(anomaliesOnly, "h-9 px-3")}
            >
              <TriangleAlert className="size-4" />
              {t("anomaliesOnly")}
            </Button>
          </div>
        }
      >
        <div className="mb-3 flex flex-wrap gap-2">
          {(["access", "operation", "security", "system", "points", "balance"] as LogTab[]).map((item) => (
            <Button key={item} type="button" variant="outline" onClick={() => setTab(item)} className={logButtonClass(tab === item, "h-9 px-3")}>
              {item === "points" ? <Coins className="size-4" /> : null}
              {item === "balance" ? <CircleDollarSign className="size-4" /> : null}
              {t(`tabs.${item}`)}
            </Button>
          ))}
        </div>
        {loading ? (
          <LoadingBlock label={t("loading")} />
        ) : tab === "access" ? (
          <AccessLogTable items={accessLogs?.items ?? []} />
        ) : tab === "operation" ? (
          <AccessLogTable items={operationLogs?.items ?? []} />
        ) : tab === "security" ? (
          <SecurityLogTable items={securityLogs?.items ?? []} />
        ) : tab === "system" ? (
          <SecurityLogTable items={systemLogs?.items ?? []} />
        ) : tab === "points" ? (
          <PointsAuditTable items={pointsLogs?.items ?? []} />
        ) : (
          <BalanceAuditTable items={balanceLogs?.items ?? []} />
        )}
        <LogPagination
          pagination={activeLogs?.pagination}
          page={pageByTab[tab]}
          pageSize={pageSize}
          loading={loading}
          t={t}
          onPageSizeChange={(nextPageSize) => {
            setPageSize(nextPageSize);
            resetPages();
          }}
          onPageChange={(nextPage) => {
            setPageByTab((current) => ({ ...current, [tab]: nextPage }));
          }}
        />
      </Panel>
    </div>
  );
}
