"use client";

import { useEffect, useRef } from "react";
import type { ECharts, EChartsOption } from "echarts";

export type AdminPieChartItem = {
  key?: string;
  label?: string;
  count: number;
};

export function AdminPieChart({
  data,
  height = 240,
}: {
  data: AdminPieChartItem[];
  height?: number;
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
    chartRef.current?.setOption(pieOptions(data), true);
  }, [data]);

  return <div ref={nodeRef} style={{ height }} className="w-full min-w-0" />;
}

function pieOptions(data: AdminPieChartItem[]): EChartsOption {
  const rows = data
    .map((item) => ({
      name: item.label || item.key || "-",
      value: Number(item.count ?? 0),
    }))
    .filter((item) => item.value > 0);
  return {
    animation: false,
    color: ["#2563eb", "#16a34a", "#f59e0b", "#dc2626", "#9333ea", "#0891b2", "#64748b", "#ea580c"],
    tooltip: { trigger: "item" },
    legend: {
      orient: "vertical",
      right: 0,
      top: "middle",
      type: "scroll",
      textStyle: { color: "#4b5563", fontSize: 12 },
    },
    series: [
      {
        type: "pie",
        radius: ["42%", "70%"],
        center: ["36%", "50%"],
        avoidLabelOverlap: true,
        label: { color: "#4b5563", formatter: "{b}\n{d}%" },
        labelLine: { lineStyle: { color: "#cbd5e1" } },
        data: rows,
      },
    ],
  };
}
