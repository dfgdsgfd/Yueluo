"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Activity, ClipboardList, Database, Eye, Gauge, Loader2, Radio, RefreshCw, Search, Server } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { AdminTimeSeriesChart } from "@/components/admin/admin-time-series-chart";
import { adminRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

import {
  chartValueFormatter,
  cpuCoreSeries,
  networkChartSeries,
  postgresChartSeries,
  requestChartSeries,
  requestHealthSeries,
  runtimeChartSeries,
  systemIoLatencySeries,
  systemIoOpsSeries,
} from "./observability-chart-config";
import { EndpointRankingPanel, SlowQueryGroupsPanel } from "./observability-diagnostics-panels";
import {
  AccessLogList,
  ObservabilityEventsTable,
  RedisStatus,
  SlowList,
  observabilityEventTypeLabel,
  slowQuerySecondary,
  slowRequestSecondary,
} from "./observability-lists";
import {
  type AccessLogPayload,
  type ObservabilityEventsPayload,
  type ObservabilityEventType,
  type ObservabilityRange,
  type PerformancePayload,
  type SystemLogsPayload,
  observabilityBuckets,
  observabilityRanges,
} from "./observability-types";
import { PostgresAdvancedPanel, PostgresDiagnosticsPanel } from "./postgres-observability-panels";
import {
  EmptyBlock,
  HeaderCard,
  InfoTile,
  LoadingBlock,
  Panel,
  errorMessage,
  formatCompact,
  formatCpuPercent,
  formatDateTime,
  formatDecimal,
  formatMegabytes,
  formatMs,
  formatPercent,
  readableValue,
  recordArray,
  recordObject,
} from "./shared";

export { RedisStatus } from "./observability-lists";

export function ObservabilityPanel({ token }: { token: string }) {
  const t = useTranslations("adminObservability");
  const [range, setRange] = useState<ObservabilityRange>("24h");
  const [logs, setLogs] = useState<SystemLogsPayload | null>(null);
  const [performance, setPerformance] = useState<PerformancePayload | null>(null);
  const [lastRefreshedAt, setLastRefreshedAt] = useState<string>("");
  const [eventType, setEventType] = useState<ObservabilityEventType>("errors");
  const [eventKeyword, setEventKeyword] = useState("");
  const [events, setEvents] = useState<ObservabilityEventsPayload | null>(null);
  const [accessOpen, setAccessOpen] = useState(false);
  const [accessLogs, setAccessLogs] = useState<Array<Record<string, unknown>>>([]);
  const [loading, setLoading] = useState(true);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [loadingAccess, setLoadingAccess] = useState(false);
  const [loadingMoreLogs, setLoadingMoreLogs] = useState(false);

  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const [logData, performanceData] = await Promise.all([
        adminRequest<SystemLogsPayload>("/api/admin/system-logs", { method: "GET", token, query: { limit: 30 } }),
        adminRequest<PerformancePayload>("/api/admin/performance", { method: "GET", token, query: { range, bucket: observabilityBuckets[range], slowLimit: 50 } }),
      ]);
      setLogs(logData);
      setPerformance(performanceData);
      setLastRefreshedAt(new Date().toISOString());
    } catch (error) {
      if (!silent) toast.error(errorMessage(error));
    } finally {
      if (!silent) setLoading(false);
    }
  }, [range, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  useEffect(() => {
    const timer = window.setInterval(() => void load(true), 10000);
    return () => window.clearInterval(timer);
  }, [load]);

  const loadAccessLogs = useCallback(async () => {
    setLoadingAccess(true);
    try {
      const payload = await adminRequest<AccessLogPayload>("/api/admin/observability/access-log", { method: "GET", token, query: { limit: 100 } });
      setAccessLogs(payload.items ?? []);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoadingAccess(false);
    }
  }, [token]);

  useEffect(() => {
    if (!accessOpen) return;
    queueMicrotask(() => void loadAccessLogs());
    const timer = window.setInterval(() => void loadAccessLogs(), 2000);
    return () => window.clearInterval(timer);
  }, [accessOpen, loadAccessLogs]);

  const loadEvents = useCallback(async (page = 1) => {
    setLoadingEvents(true);
    try {
      const payload = await adminRequest<ObservabilityEventsPayload>("/api/admin/observability/events", {
        method: "GET",
        token,
        query: {
          type: eventType,
          page,
          limit: 20,
          keyword: eventKeyword,
          range,
        },
      });
      setEvents(payload);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoadingEvents(false);
    }
  }, [eventKeyword, eventType, range, token]);

  useEffect(() => {
    queueMicrotask(() => void loadEvents(1));
  }, [eventType, loadEvents]);

  const loadMoreLogs = useCallback(async () => {
    if (!logs?.hasMore || !logs.nextCursor || loadingMoreLogs) return;
    setLoadingMoreLogs(true);
    try {
      const next = await adminRequest<SystemLogsPayload>("/api/admin/system-logs", { method: "GET", token, query: { limit: 30, cursor: logs.nextCursor } });
      setLogs((previous) => ({
        ...next,
        items: [...(previous?.items ?? []), ...(next.items ?? [])],
      }));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoadingMoreLogs(false);
    }
  }, [loadingMoreLogs, logs, token]);

  const runtime = performance?.runtime ?? {};
  const requests = performance?.requests ?? {};
  const postgres = performance?.postgresql ?? {};
  const requestSeries = recordArray(requests.series);
  const runtimeSeries = recordArray(performance?.runtime_series);
  const postgresSeries = recordArray(performance?.postgresql_series);
  const endpointRankings = recordArray(requests.endpoints);
  const pgActivity = recordObject(postgres.activity);
  const pgLocks = recordObject(postgres.locks);
  const pgTableHealth = recordObject(postgres.table_health);
  const pgWal = recordObject(postgres.wal);
  const pgIO = recordObject(postgres.io);
  const pgCheckpointer = recordObject(postgres.checkpointer);
  const pgDiagnostics = recordObject(postgres.diagnostics);
  const pgTopSQL = recordArray(recordObject(postgres.top_sql).items);
  const pgTopDead = recordArray(pgTableHealth.top_dead);
  const slowRequests = recordArray(performance?.slow_requests?.items);
  const slowQueries = recordArray(performance?.slow_queries?.items);
  const slowQueryGroups = recordArray(performance?.slow_queries?.groups);
  const versions = performance?.versions ?? {};
  const items = logs?.items ?? [];
  const rangeLabel = t(`ranges.${range}`);
  const cpuCoreChartSeries = cpuCoreSeries(runtimeSeries, runtime);

  return (
    <div className="grid min-w-0 gap-4">
      <HeaderCard icon={Gauge} title={t("title")} description={t("description")} tone="blue" />
      <Panel title={t("rangeTitle")} icon={Activity}>
        <div className="flex flex-wrap gap-2">
          {observabilityRanges.map((item) => (
            <Button
              key={item}
              type="button"
              variant="outline"
              onClick={() => setRange(item)}
              className={cn(
                "h-9 rounded-lg border px-3 text-sm",
                range === item
                  ? "border-[#2f7df6]/30 bg-[#eef6ff] text-[#1d4ed8] hover:bg-[#dbeafe]"
                  : "border-black/[0.08] bg-white text-[#59606c] hover:bg-[#f8fafc]",
              )}
            >
              {t(`ranges.${item}`)}
            </Button>
          ))}
        </div>
      </Panel>
      <div className="grid gap-4 xl:grid-cols-5">
        <Panel title={t("refreshTitle")} icon={RefreshCw}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("autoRefresh")} value={t("everyTenSeconds")} />
            <InfoTile label={t("lastRefresh")} value={formatDateTime(lastRefreshedAt)} />
          </div>
        </Panel>
        <Panel title={t("runtimeTitle")} icon={Gauge}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("goroutines")} value={formatCompact(Number(runtime.goroutines ?? 0))} />
            <InfoTile label={t("gcCount")} value={formatCompact(Number(runtime.gc_count ?? 0))} />
            <InfoTile label={t("heapMemory")} value={formatMegabytes(Number(runtime.heap_alloc ?? 0))} />
            <InfoTile label={t("systemMemory")} value={formatMegabytes(Number(runtime.sys_bytes ?? 0))} />
          </div>
        </Panel>
        <Panel title={t("connectionTitle")} icon={Radio}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("estimatedConnections")} value={formatCompact(runtime.current_estimated_connections)} />
            <InfoTile label={t("activeRequests")} value={formatCompact(runtime.active_requests)} />
            <InfoTile label={t("uniqueClients")} value={formatCompact(runtime.unique_clients_60s)} />
            <InfoTile label={t("processCpu")} value={formatCpuPercent(runtime.process_cpu_percent)} />
          </div>
        </Panel>
        <Panel title={t("systemIoTitle")} icon={Database}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("systemIoReadLatency")} value={formatMs(runtime.disk_read_latency_ms)} />
            <InfoTile label={t("systemIoWriteLatency")} value={formatMs(runtime.disk_write_latency_ms)} />
            <InfoTile label={t("systemIoReads")} value={formatCompact(runtime.disk_read_count)} />
            <InfoTile label={t("systemIoWrites")} value={formatCompact(runtime.disk_write_count)} />
          </div>
          {runtime.disk_io_message ? <p className="mt-3 break-all text-xs text-amber-800">{readableValue(runtime.disk_io_message)}</p> : null}
        </Panel>
        <Panel title={t("requestTitle")} icon={Activity}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("windowQps", { range: rangeLabel })} value={formatDecimal(requests.qps, 3)} />
            <InfoTile label={t("windowRequests", { range: rangeLabel })} value={formatCompact(Number(requests.count ?? 0))} />
            <InfoTile label="P50" value={formatMs(requests.p50_latency_ms)} />
            <InfoTile label="P95" value={formatMs(requests.p95_latency_ms)} />
            <InfoTile label="P99" value={formatMs(requests.p99_latency_ms)} />
            <InfoTile label={t("maxLatency")} value={formatMs(requests.max_latency_ms)} />
            <InfoTile label="4xx" value={formatCompact(Number(requests.status_4xx ?? 0))} />
            <InfoTile label="5xx" value={formatCompact(Number(requests.status_5xx ?? 0))} />
            <InfoTile label={t("errorRate")} value={formatPercent(requests.error_rate)} />
            <InfoTile label={t("slow")} value={formatCompact(requests.slow_count)} />
          </div>
        </Panel>
        <Panel title={t("postgresTitle")} icon={Database}>
          <div className="grid grid-cols-2 gap-3">
            <InfoTile label={t("pgOpen")} value={formatCompact(Number(postgres.open_connections ?? 0))} />
            <InfoTile label={t("pgInUse")} value={formatCompact(Number(postgres.in_use ?? 0))} />
            <InfoTile label={t("pgCacheHit")} value={formatPercent(postgres.cache_hit_ratio)} />
            <InfoTile label={t("pgWaitingLocks")} value={formatCompact(pgLocks.waiting_locks)} />
          </div>
        </Panel>
        <Panel title={t("redisTitle")} icon={Server}>
          <RedisStatus status={performance?.redis} />
        </Panel>
      </div>
      <Panel title={t("versionsTitle")} icon={Server}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <InfoTile label="Go" value={readableValue(versions.go)} />
          <InfoTile label="Gin" value={readableValue(versions.gin)} />
          <InfoTile label="PostgreSQL" value={readableValue(versions.postgresql)} />
          <InfoTile label="Redis" value={readableValue(versions.redis)} />
        </div>
      </Panel>
      <div className="grid min-w-0 gap-4 xl:grid-cols-2">
        <Panel title={t("requestChartTitle")} icon={Activity}>
          <AdminTimeSeriesChart data={requestSeries} series={requestChartSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("requestHealthChartTitle")} icon={Activity}>
          <AdminTimeSeriesChart data={requestSeries} series={requestHealthSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("runtimeChartTitle")} icon={Gauge}>
          <AdminTimeSeriesChart data={runtimeSeries} series={runtimeChartSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("cpuCoreChart")} icon={Gauge}>
          <AdminTimeSeriesChart data={runtimeSeries} series={cpuCoreChartSeries} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("networkChart")} icon={Radio}>
          <AdminTimeSeriesChart data={runtimeSeries} series={networkChartSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("systemIoLatencyChart")} icon={Database}>
          <AdminTimeSeriesChart data={runtimeSeries} series={systemIoLatencySeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("systemIoOpsChart")} icon={Database}>
          <AdminTimeSeriesChart data={runtimeSeries} series={systemIoOpsSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
        <Panel title={t("postgresChartTitle")} icon={Database}>
          <AdminTimeSeriesChart data={postgresSeries} series={postgresChartSeries(t)} valueFormatter={chartValueFormatter} />
        </Panel>
      </div>
      <div className="grid min-w-0 gap-4 xl:grid-cols-2">
        <EndpointRankingPanel items={endpointRankings} />
        <SlowQueryGroupsPanel items={slowQueryGroups} />
      </div>
      <Panel title={t("slowOverviewTitle")} icon={ClipboardList}>
        <div className="grid gap-3 md:grid-cols-2">
          <SlowList title={t("slowRequests")} items={slowRequests} primary={(item) => `${item.method ?? ""} ${item.route ?? item.path ?? ""}`} secondary={slowRequestSecondary} />
          <SlowList title={t("slowQueries")} items={slowQueries} primary={(item) => String(item.sql ?? "-")} secondary={(item) => slowQuerySecondary(item, t("rows"))} />
        </div>
      </Panel>
      <PostgresDiagnosticsPanel diagnostics={pgDiagnostics} />
      <PostgresAdvancedPanel
        activity={pgActivity}
        locks={pgLocks}
        tableHealth={pgTableHealth}
        wal={pgWal}
        io={pgIO}
        checkpointer={pgCheckpointer}
        deadlocks={postgres.deadlocks}
        topSQL={pgTopSQL}
        topDead={pgTopDead}
      />
      <Panel
        title={t("accessLogTitle")}
        icon={Eye}
        action={
          <Button type="button" variant={accessOpen ? "default" : "outline"} onClick={() => setAccessOpen((value) => !value)} className="h-9 rounded-lg px-3">
            {accessOpen ? t("accessLogDisable") : t("accessLogEnable")}
          </Button>
        }
      >
        {accessOpen ? (
          <AccessLogList items={accessLogs} loading={loadingAccess} />
        ) : (
          <EmptyBlock icon={Eye} label={t("accessLogDisabled")} />
        )}
      </Panel>
      <Panel
        title={t("eventsTitle")}
        icon={Search}
        action={
          <Button type="button" variant="outline" disabled={loadingEvents} onClick={() => void loadEvents(1)} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
            <RefreshCw className={cn("size-4", loadingEvents && "animate-spin")} />
            {t("query")}
          </Button>
        }
      >
        <div className="mb-3 flex flex-wrap items-center gap-2">
          {(["errors", "slow_requests", "slow_queries"] as ObservabilityEventType[]).map((type) => (
            <button
              key={type}
              type="button"
              onClick={() => setEventType(type)}
              className={cn("h-9 rounded-lg border px-3 text-sm font-semibold transition", eventType === type ? "border-[#2f7df6]/30 bg-[#eef6ff] text-[#1d4ed8]" : "border-black/[0.06] bg-white text-[#59606c] hover:bg-[#f8fafc]")}
            >
              {observabilityEventTypeLabel(type, t)}
            </button>
          ))}
          <div className="ml-auto flex w-full min-w-0 items-center gap-2 sm:w-80">
            <Search className="size-4 shrink-0 text-[#8a8f9d]" />
            <input
              value={eventKeyword}
              onChange={(event) => setEventKeyword(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") void loadEvents(1);
              }}
              className="h-9 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#2f7df6]"
              placeholder={t("searchPlaceholder")}
            />
          </div>
        </div>
        <ObservabilityEventsTable type={eventType} payload={events} loading={loadingEvents} onPage={(page) => void loadEvents(page)} />
      </Panel>
      <Panel
        title={t("systemLogsTitle")}
        icon={ClipboardList}
        action={
          <Button type="button" variant="outline" disabled={loading} onClick={() => void load()} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
            <RefreshCw className={cn("size-4", loading && "animate-spin")} />
            {t("refresh")}
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingLogs")} />
        ) : items.length ? (
          <div className="grid gap-2">
            {items.map((item, index) => (
              <div key={String(item.id ?? index)} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
                <div className="flex flex-wrap items-center gap-2 text-xs text-[#7a8495]">
                  <span className="font-bold text-[#252932]">{String(item.type ?? "-")}</span>
                  <span>{String(item.level ?? "info")}</span>
                  <span>{String(item.status ?? "-")}</span>
                  <span>{formatDateTime(item.created_at)}</span>
                  <span className="ml-auto">{formatMs(item.latency_ms)}</span>
                </div>
                <p className="mt-1 line-clamp-2 break-all text-sm font-semibold text-[#252932]">{String(item.message ?? `${item.method ?? ""} ${item.path ?? ""}`)}</p>
                <p className="mt-1 line-clamp-1 break-all text-xs text-[#7a8495]">{String(item.path ?? "")}</p>
              </div>
            ))}
            {logs?.hasMore ? (
              <Button type="button" variant="outline" disabled={loadingMoreLogs} onClick={() => void loadMoreLogs()} className="h-10 rounded-lg border-black/[0.08] bg-white">
                <Loader2 className={cn("size-4", loadingMoreLogs && "animate-spin")} />
                {t("loadMoreLogs")}
              </Button>
            ) : null}
          </div>
        ) : (
          <EmptyBlock icon={ClipboardList} label={t("emptyLogs")} />
        )}
      </Panel>
    </div>
  );
}
