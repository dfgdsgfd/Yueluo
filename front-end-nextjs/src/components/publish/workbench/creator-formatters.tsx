"use client";
import type {
  CreatorOverviewPayload,
  CreatorStatsPayload,
  CreatorTrendsPayload
} from "@/lib/types";
import {
  CreatorTrendSeriesKey,
  LiveRange,
  MetricsView,
  NoteRange,
  creatorTrendSeries,
  liveMetricKeys,
  noteMetricKeys
} from "./workbench-config";

export function getCreatorMetricRangeKey(
  metricsView: MetricsView,
  noteRange: NoteRange,
  liveRange: LiveRange,
  stats: CreatorStatsPayload | null,
) {
  if (metricsView === "note") {
    return noteRange === "recent7" ? "last_7_days" : "last_30_days";
  }

  if (liveRange === "today") {
    return "today";
  }

  if (liveRange === "yesterday") {
    return "yesterday";
  }

  if (liveRange === "recent7") {
    return "last_7_days";
  }

  return stats ? `last_${stats.window_days}_days` : "last_30_days";
}


export function buildCreatorMetricValues(
  stats: CreatorStatsPayload | null,
  overview: CreatorOverviewPayload | null,
  rangeKey: string,
) {
  const postTotals = stats?.post_totals ?? {};
  const interactions = stats?.interactions ?? {};
  const fans = stats?.fans ?? {};

  return {
    averageInteractions: 0,
    averageViewers: 0,
    collections: interactions.collects?.[rangeKey] ?? postTotals.collect_count ?? 0,
    comments: interactions.comments?.[rangeKey] ?? postTotals.comment_count ?? 0,
    completionRate: 0,
    coverClicks: 0,
    diamonds: overview?.balance ?? 0,
    exposure: interactions.views?.[rangeKey] ?? postTotals.view_count ?? 0,
    likes: interactions.likes?.[rangeKey] ?? postTotals.like_count ?? 0,
    liveDuration: 0,
    liveFollowers: fans[rangeKey] ?? 0,
    liveSessions: 0,
    merchantRevenue: overview?.month_earnings ?? 0,
    netFollowers: fans[rangeKey] ?? 0,
    newFollows: fans[rangeKey] ?? 0,
    paidOrders: 0,
    profileVisitors: fans.total ?? 0,
    shares: 0,
    unfollows: 0,
    views: interactions.views?.[rangeKey] ?? postTotals.view_count ?? 0,
  } satisfies Record<(typeof noteMetricKeys)[number] | (typeof liveMetricKeys)[number], number>;
}


export function formatCreatorMetricValue(value: number) {
  if (!Number.isFinite(value)) {
    return "0";
  }

  if (value >= 10000) {
    return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}w`;
  }

  if (value >= 1000) {
    return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}k`;
  }

  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}


export function formatCreatorCurrency(value: number | undefined) {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;

  return amount.toLocaleString("zh-CN", {
    maximumFractionDigits: 2,
    minimumFractionDigits: amount % 1 === 0 ? 0 : 2,
  });
}


export function formatCreatorDate(value?: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}


export function getCreatorTrendValue(
  trends: CreatorTrendsPayload | null,
  key: CreatorTrendSeriesKey,
  index: number,
) {
  return trends?.[key]?.[index] ?? 0;
}


export function getCreatorTrendTotal(trends: CreatorTrendsPayload | null, key: CreatorTrendSeriesKey) {
  return (trends?.[key] ?? []).reduce((total, value) => total + value, 0);
}


export function getCreatorTrendMax(trends: CreatorTrendsPayload | null) {
  const values = creatorTrendSeries.flatMap(({ key }) => trends?.[key] ?? []);
  return Math.max(1, ...values);
}


export function creatorContentTypeKey(type?: number) {
  if (type === 2 || type === 3) {
    return "contentType.video";
  }

  return "contentType.article";
}


export function creatorEarningsTypeKey(type?: string | null) {
  switch (type) {
    case "purchase":
      return "earningsType.purchase";
    case "withdraw":
      return "earningsType.withdraw";
    case "quality_reward":
      return "earningsType.qualityReward";
    case "extended_daily":
      return "earningsType.extendedDaily";
    case "transfer_from_wallet":
      return "earningsType.walletTransfer";
    default:
      return "earningsType.earnings";
  }
}


export function withdrawTypeKey(value: string) {
  return value === "cash" ? "withdrawType.cash" : "withdrawType.moonCoin";
}


export function withdrawStatusKey(value: string) {
  switch (value) {
    case "pending":
      return "withdrawStatus.pending";
    case "approved":
      return "withdrawStatus.approved";
    case "rejected":
      return "withdrawStatus.rejected";
    case "paid":
      return "withdrawStatus.paid";
    default:
      return "withdrawStatus.unknown";
  }
}


export function withdrawStatusClassName(value: string) {
  switch (value) {
    case "approved":
    case "paid":
      return "bg-[#e9fbf4] text-[#42b389]";
    case "rejected":
      return "bg-[#fff0f5] text-[#f25a89]";
    case "pending":
      return "bg-[#fff7db] text-[#f0a000]";
    default:
      return "bg-[#f4f4f6] text-[#777780]";
  }
}


export function formatWithdrawDate(value?: string | null, locale = "en") {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale, {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
  }).format(date);
}


export function hasCreatorListNextPage(
  payload:
    | {
        list: unknown[];
        pagination: {
          limit?: number;
          page?: number;
          pages?: number;
          pageSize?: number;
          total?: number;
          totalPages?: number;
        };
      }
    | null,
) {
  if (!payload) {
    return false;
  }

  const page = payload.pagination.page ?? 1;
  const totalPages = payload.pagination.totalPages ?? payload.pagination.pages;
  if (typeof totalPages === "number") {
    return page < totalPages;
  }

  const total = payload.pagination.total;
  const limit = payload.pagination.limit ?? payload.pagination.pageSize ?? payload.list.length;
  return typeof total === "number" && limit > 0 ? page * limit < total : false;
}


export function nextCreatorListPage(payload: { pagination: { page?: number } } | null) {
  return (payload?.pagination.page ?? 1) + 1;
}


export function mergeCreatorItems<T extends { id: number | string }>(current: T[], next: T[]) {
  const seen = new Set<string>();
  const merged: T[] = [];

  for (const item of [...current, ...next]) {
    const key = String(item.id);
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    merged.push(item);
  }

  return merged;
}
