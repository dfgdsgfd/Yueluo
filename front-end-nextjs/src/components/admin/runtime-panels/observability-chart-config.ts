import type { AdminChartSeries } from "@/components/admin/admin-time-series-chart";

import { formatBytes, formatMegabytes, formatMs, formatNetworkRate } from "./shared";

type Translate = (key: string) => string;

export function requestChartSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartQps"), key: "qps", color: "#2f7df6" },
    { name: "P50", key: "p50_latency_ms", color: "#18a058", unit: "ms" },
    { name: "P95", key: "p95_latency_ms", color: "#c97900", unit: "ms" },
    { name: "P99", key: "p99_latency_ms", color: "#dc2626", unit: "ms" },
  ];
}

export function requestHealthSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartErrorRate"), key: "error_rate", color: "#dc2626", unit: "%" },
    { name: t("chartSlowCount"), key: "slow_count", color: "#c97900" },
    { name: "4xx", key: "status_4xx", color: "#7357d6" },
    { name: "5xx", key: "status_5xx", color: "#ef4444" },
  ];
}

export function runtimeChartSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartHeap"), key: "heap_alloc", color: "#18a058", unit: "megabytes" },
    { name: t("chartSys"), key: "sys_bytes", color: "#2f7df6", unit: "megabytes" },
    { name: t("chartGoroutines"), key: "goroutines", color: "#7357d6" },
  ];
}

export function networkChartSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartNetworkRx"), key: "net_rx_bps", color: "#18a058", unit: "bytes_per_second" },
    { name: t("chartNetworkTx"), key: "net_tx_bps", color: "#2f7df6", unit: "bytes_per_second" },
  ];
}

export function systemIoLatencySeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartRead"), key: "disk_read_latency_ms", color: "#0f766e", unit: "ms" },
    { name: t("chartWrite"), key: "disk_write_latency_ms", color: "#c97900", unit: "ms" },
  ];
}

export function systemIoOpsSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("chartReadOps"), key: "disk_read_ops_per_sec", color: "#2f7df6", unit: "ops_per_second" },
    { name: t("chartWriteOps"), key: "disk_write_ops_per_sec", color: "#7357d6", unit: "ops_per_second" },
  ];
}

export function postgresChartSeries(t: Translate): AdminChartSeries[] {
  return [
    { name: t("pgInUse"), key: "in_use", color: "#2f7df6" },
    { name: t("pgOpen"), key: "open_connections", color: "#7357d6" },
    { name: t("pgCacheHit"), key: "cache_hit_ratio", color: "#18a058", unit: "%" },
    { name: t("pgActiveConnections"), key: "active_connections", color: "#c97900" },
    { name: t("pgIdleTransactionShort"), key: "idle_in_transaction", color: "#dc2626" },
    { name: t("pgWaitingLocks"), key: "waiting_locks", color: "#0f766e" },
  ];
}

export function chartValueFormatter(value: number, item: AdminChartSeries) {
  if (item.unit === "ms") return formatMs(value);
  if (item.unit === "bytes") return formatBytes(value);
  if (item.unit === "megabytes") return formatMegabytes(value);
  if (item.unit === "bytes_per_second") return formatNetworkRate(value);
  if (item.unit === "ops_per_second") return `${value.toFixed(value >= 10 ? 0 : 1)}/s`;
  if (item.unit === "percent") return `${value.toFixed(1)}%`;
  if (item.unit === "%") return `${(value * 100).toFixed(1)}%`;
  return Number.isInteger(value) ? value.toLocaleString("zh-CN") : value.toFixed(3);
}

export function cpuCoreSeries(rows: Array<Record<string, unknown>>, runtime: Record<string, unknown>): AdminChartSeries[] {
  const current = Array.isArray(runtime.cpu_cores_percent) ? runtime.cpu_cores_percent.length : 0;
  const fromRows = rows.reduce((count, row) => {
    const coreKeys = Object.keys(row).filter((key) => key.startsWith("cpu_core_"));
    return Math.max(count, coreKeys.length);
  }, 0);
  const count = Math.max(current, fromRows);
  const colors = ["#2f7df6", "#18a058", "#c97900", "#7357d6", "#dc2626", "#0f766e", "#a855f7", "#64748b"];
  return Array.from({ length: count }, (_, index) => ({
    name: `CPU ${index}`,
    key: `cpu_core_${index}`,
    color: colors[index % colors.length],
    unit: "percent",
  }));
}
