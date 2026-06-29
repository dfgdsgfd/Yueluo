"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  BarChart3,
  Loader2,
  MessageCircle,
  RefreshCw,
  Sparkles,
  Star,
  User,
  UserPlus,
} from "lucide-react";
import { toast } from "sonner";
import {
  getCreatorMetric,
  getCreatorMetricCardClassName,
  getCreatorRangeKey,
  formatCompactCount,
  creatorDataRanges,
  CreatorTrendChart,
  creatorTrendSeries,
  fallbackCreatorStats,
  fallbackCreatorTrends,
  normalizeCreatorTrends,
  type ActivityTone,
  type RangeDays,
} from "@/components/creator-center/mobile-creator-center-page";
import { getCreatorStats, getCreatorTrends, getStoredAccessToken } from "@/lib/api";
import type { CreatorStatsPayload, CreatorTrendsPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

export function MobileCreatorDataPage() {
  const router = useRouter();
  const [authToken, setAuthToken] = useState<string | null | undefined>(undefined);
  const [activeRange, setActiveRange] = useState<RangeDays>(30);
  const [stats, setStats] = useState<CreatorStatsPayload | null>(null);
  const [trends, setTrends] = useState<CreatorTrendsPayload | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const safeStats = stats ?? fallbackCreatorStats;
  const safeTrends = normalizeCreatorTrends(trends ?? fallbackCreatorTrends, activeRange);
  const rangeKey = getCreatorRangeKey(activeRange);
  const fanCount = getCreatorMetric(safeStats.fans, rangeKey, fallbackCreatorStats.fans.recent30 ?? 14);
  const views = getCreatorMetric(safeStats.interactions.views, rangeKey, safeStats.post_totals.view_count ?? 232);
  const likes = getCreatorMetric(safeStats.interactions.likes, rangeKey, safeStats.post_totals.like_count ?? 3);
  const collects = getCreatorMetric(safeStats.interactions.collects, rangeKey, safeStats.post_totals.collect_count ?? 1);
  const comments = getCreatorMetric(safeStats.interactions.comments, rangeKey, safeStats.post_totals.comment_count ?? 0);
  const totalFans = safeStats.fans.total ?? views;

  const metricCards = [
    { icon: UserPlus, label: "新增粉丝", value: fanCount, delta: 0, tone: "amber" as ActivityTone },
    { icon: User, label: "主页访问", value: totalFans, delta: 6, tone: "blue" as ActivityTone },
    { icon: Star, label: "点赞数", value: likes, delta: 0, tone: "pink" as ActivityTone },
    { icon: Sparkles, label: "收藏数", value: collects, delta: 0, tone: "green" as ActivityTone },
    { icon: MessageCircle, label: "评论数", value: comments, delta: 0, tone: "purple" as ActivityTone },
  ];

  const trendMax = useMemo(() => {
    const values = creatorTrendSeries.flatMap((series) => safeTrends[series.key]);
    return Math.max(1, ...values);
  }, [safeTrends]);

  const loadData = useCallback(async (days: RangeDays, options: { silent?: boolean } = {}) => {
    const token = getStoredAccessToken();
    setAuthToken(token);
    if (!token) {
      setIsLoading(false);
      setIsRefreshing(false);
      return;
    }

    if (options.silent) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }

    const [statsResult, trendsResult] = await Promise.allSettled([
      getCreatorStats(days),
      getCreatorTrends(days),
    ]);

    if (statsResult.status === "fulfilled") {
      setStats(statsResult.value);
    }
    if (trendsResult.status === "fulfilled") {
      setTrends(trendsResult.value);
    }
    if ((statsResult.status === "rejected" || trendsResult.status === "rejected") && options.silent) {
      toast.error("数据中心刷新失败，已保留当前展示。");
    }

    setIsLoading(false);
    setIsRefreshing(false);
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData(activeRange);
    });
  }, [activeRange, loadData]);

  const handleBack = useCallback(() => {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/creator-center");
  }, [router]);

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#24242c]">
      <div className="mx-auto min-h-dvh w-full max-w-[430px] bg-[#fbfbff] pb-6 shadow-[0_0_38px_rgba(112,94,173,0.08)]">
        <header className="sticky top-0 z-30 bg-[#fbfbff]/92 px-4 pb-3 pt-[calc(0.75rem+env(safe-area-inset-top))] backdrop-blur">
          <div className="grid h-11 grid-cols-[40px_minmax(0,1fr)_40px] items-center min-[390px]:grid-cols-[44px_minmax(0,1fr)_44px]">
            <button
              type="button"
              aria-label="返回上一页"
              onClick={handleBack}
              className="flex size-10 items-center justify-center rounded-full text-[#15151b] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <ArrowLeft className="size-6" strokeWidth={2.4} />
            </button>
            <h1 className="min-w-0 truncate text-center text-[21px] font-black tracking-normal text-[#16161d]">
              数据中心
            </h1>
            <button
              type="button"
              aria-label="刷新数据中心"
              onClick={() => void loadData(activeRange, { silent: true })}
              className="flex size-10 items-center justify-center justify-self-end rounded-full text-[#8b7adf] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <RefreshCw className={cn("size-5.5", isRefreshing && "animate-spin")} strokeWidth={2.4} />
            </button>
          </div>
        </header>

        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <Loader2 className="size-4 animate-spin" />
            正在同步创作数据
          </div>
        ) : (
          <div className="space-y-4 px-4">
            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="flex min-w-max items-center gap-2 whitespace-nowrap text-[18px] font-black tracking-normal min-[390px]:text-[19px]">
                  粉丝与互动统计
                  <BarChart3 className="size-4 text-[#8b7adf]" />
                </h2>
                <div className="ml-auto flex h-8 shrink-0 rounded-full bg-[#faf9ff] p-0.5 shadow-[inset_0_0_0_1px_rgba(128,113,196,0.12)]">
                  {creatorDataRanges.map((range) => (
                    <button
                      key={range.days}
                      type="button"
                      onClick={() => setActiveRange(range.days)}
                      className={cn(
                        "h-7 rounded-full px-2.5 text-[11px] font-black text-[#7d7a88] transition min-[390px]:px-3 min-[390px]:text-[12px]",
                        activeRange === range.days && "bg-white text-[#765eda] shadow-[0_2px_9px_rgba(115,96,194,0.18)]",
                      )}
                    >
                      {range.label}
                    </button>
                  ))}
                </div>
              </div>

              <div className="mt-4 grid grid-cols-2 gap-2 min-[370px]:grid-cols-3 min-[420px]:grid-cols-5">
                {metricCards.map(({ icon: Icon, label, value, delta, tone }) => (
                  <article key={label} className={cn("min-w-0 rounded-[8px] px-2 py-3", getCreatorMetricCardClassName(tone))}>
                    <p className="flex min-w-0 flex-col items-start gap-1 text-[11px] font-bold text-[#777381]">
                      <Icon className="size-3.5 shrink-0" />
                      <span className="whitespace-nowrap">{label}</span>
                    </p>
                    <p className="mt-2 truncate text-[clamp(21px,6vw,25px)] font-black leading-none text-[#111116]">
                      {formatCompactCount(value)}
                    </p>
                    <p className="mt-3 truncate text-[12px] font-bold text-[#8d8998]">
                      较昨日 {delta >= 0 ? "+" : ""}{delta}
                    </p>
                  </article>
                ))}
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-2">
                <h2 className="whitespace-nowrap text-[18px] font-black tracking-normal min-[430px]:text-[20px]">
                  近7天数据趋势
                </h2>
                <span className="flex h-8 shrink-0 items-center rounded-full bg-[#faf9ff] px-2.5 text-[12px] font-black text-[#8b8796]">
                  总览
                </span>
              </div>
              <CreatorTrendChart labels={safeTrends.labels} max={trendMax} series={safeTrends} />
            </section>

            {authToken === null ? (
              <section className="rounded-[8px] border border-[#eee8ff] bg-white px-4 py-4 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
                <p className="text-[15px] font-black text-[#23232a]">登录后可查看真实数据中心</p>
              </section>
            ) : null}
          </div>
        )}
      </div>
    </main>
  );
}
