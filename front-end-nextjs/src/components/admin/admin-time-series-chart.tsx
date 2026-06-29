"use client";

import { useEffect, useRef } from "react";
import type { ECharts, EChartsOption } from "echarts";

export type AdminChartSeries = {
  name: string;
  key: string;
  color?: string;
  unit?: string;
};

export function AdminTimeSeriesChart({
  data,
  series,
  height = 260,
  valueFormatter,
}: {
  data: Array<Record<string, unknown>>;
  series: AdminChartSeries[];
  height?: number;
  valueFormatter?: (value: number, item: AdminChartSeries) => string;
}) {
  const nodeRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<ECharts | null>(null);

  useEffect(() => {
    let disposed = false;
    let resizeObserver: ResizeObserver | null = null;

    async function mount() {
      const echarts = await import("echarts");
      if (disposed || !nodeRef.current) return;
      const chart = echarts.init(nodeRef.current, undefined, { renderer: "canvas" });
      chartRef.current = chart;
      resizeObserver = new ResizeObserver(() => chart.resize());
      resizeObserver.observe(nodeRef.current);
    }

    void mount();

    return () => {
      disposed = true;
      resizeObserver?.disconnect();
      chartRef.current?.dispose();
      chartRef.current = null;
    };
  }, []);

  useEffect(() => {
    chartRef.current?.setOption(chartOptions(data, series, valueFormatter), true);
  }, [data, series, valueFormatter]);

  return <div ref={nodeRef} style={{ height }} className="w-full min-w-0" />;
}

function chartOptions(
  data: Array<Record<string, unknown>>,
  series: AdminChartSeries[],
  valueFormatter?: (value: number, item: AdminChartSeries) => string,
): EChartsOption {
  const yAxisFormatter = valueFormatter && series.length
    ? (value: number) => {
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) return "-";
        return valueFormatter(numeric, series[0]);
      }
    : undefined;
  return {
    animation: false,
    color: series.map((item) => item.color).filter(Boolean) as string[],
    grid: { left: 58, right: 18, top: 28, bottom: 42 },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "line" },
      valueFormatter: (value, index) => {
        const item = series[index ?? 0] ?? series[0];
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) return "-";
        return valueFormatter ? valueFormatter(numeric, item) : `${numeric.toFixed(2)}${item?.unit ?? ""}`;
      },
    },
    legend: { top: 0, left: 0, textStyle: { color: "#4b5563", fontSize: 12 } },
    xAxis: {
      type: "time",
      axisLabel: { color: "#7b808c", hideOverlap: true },
      axisLine: { lineStyle: { color: "#e5e7eb" } },
      axisTick: { show: false },
    },
    yAxis: {
      type: "value",
      axisLabel: { color: "#7b808c", formatter: yAxisFormatter },
      splitLine: { lineStyle: { color: "#eef0f4" } },
    },
    series: series.map((item) => ({
      name: item.name,
      type: "line",
      smooth: true,
      showSymbol: false,
      emphasis: { focus: "series" },
      data: data.map((row) => [timestamp(row.ts ?? row.created_at), numeric(row[item.key])]),
    })),
  };
}

function timestamp(value: unknown) {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const numeric = Number(value);
    if (Number.isFinite(numeric)) return numeric;
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return Date.now();
}

function numeric(value: unknown) {
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}
