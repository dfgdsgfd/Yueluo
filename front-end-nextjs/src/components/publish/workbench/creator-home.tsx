"use client";
import {
  useEffect,
  useMemo,
  useState
} from "react";
import Image from "next/image";
import {
  ArrowRight,
  FileText
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  toast
} from "sonner";
import {
  getCreatorEarningsLog,
  getCreatorOverview,
  getCreatorPaidContent,
  getCreatorQualityRewards,
  getCreatorStats,
  getCreatorTrends,
  getStoredAccessToken
} from "@/lib/api";
import type {
  CreatorEarningsLogPayload,
  CreatorOverviewPayload,
  CreatorPaidContentPayload,
  CreatorQualityRewardsPayload,
  CreatorStatsPayload,
  CreatorTrendsPayload
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  CreatorListPanel,
  LiveRange,
  MetricsView,
  NoteRange,
  PublishMode,
  creatorEarningsLogLimit,
  creatorPaidContentLimit,
  creatorQualityRewardsLimit,
  creatorTrendSeries,
  homeCreateActions,
  liveMetricKeys,
  livePeriodKeys,
  liveRanges,
  metricsViews,
  noteMetricKeys,
  notePeriodKeys,
  noteRanges
} from "./workbench-config";
import {
  buildCreatorMetricValues,
  creatorContentTypeKey,
  creatorEarningsTypeKey,
  formatCreatorCurrency,
  formatCreatorDate,
  formatCreatorMetricValue,
  getCreatorMetricRangeKey,
  getCreatorTrendMax,
  getCreatorTrendTotal,
  getCreatorTrendValue,
  hasCreatorListNextPage,
  mergeCreatorItems,
  nextCreatorListPage
} from "./creator-formatters";
import {
  CreatorEmptyState,
  CreatorLoadMoreButton,
  SectionHeader
} from "./shared-ui";

export function CreatorHome({ onCreate }: { onCreate: (mode: PublishMode) => void }) {
  const t = useTranslations();
  const [metricsView, setMetricsView] = useState<MetricsView>("note");
  const [noteRange, setNoteRange] = useState<NoteRange>("recent7");
  const [liveRange, setLiveRange] = useState<LiveRange>("today");
  const [creatorOverview, setCreatorOverview] = useState<CreatorOverviewPayload | null>(null);
  const [creatorStats, setCreatorStats] = useState<CreatorStatsPayload | null>(null);
  const [creatorTrends, setCreatorTrends] = useState<CreatorTrendsPayload | null>(null);
  const [creatorEarningsLog, setCreatorEarningsLog] =
    useState<CreatorEarningsLogPayload | null>(null);
  const [creatorPaidContent, setCreatorPaidContent] =
    useState<CreatorPaidContentPayload | null>(null);
  const [creatorQualityRewards, setCreatorQualityRewards] =
    useState<CreatorQualityRewardsPayload | null>(null);
  const [loadingCreatorPanel, setLoadingCreatorPanel] = useState<CreatorListPanel | null>(null);

  const metricKeys = metricsView === "note" ? noteMetricKeys : liveMetricKeys;
  const activeRange = metricsView === "note" ? noteRange : liveRange;
  const activePeriodKey =
    metricsView === "note" ? notePeriodKeys[noteRange] : livePeriodKeys[liveRange];
  const metricRangeKey = getCreatorMetricRangeKey(metricsView, noteRange, liveRange, creatorStats);
  const metricValues = useMemo(
    () => buildCreatorMetricValues(creatorStats, creatorOverview, metricRangeKey),
    [creatorOverview, creatorStats, metricRangeKey],
  );
  const creatorFans = creatorStats?.fans.total ?? 0;
  const creatorLikesAndCollections =
    (creatorStats?.post_totals.like_count ?? 0) +
    (creatorStats?.post_totals.collect_count ?? 0);
  const creatorTrendMax = useMemo(() => getCreatorTrendMax(creatorTrends), [creatorTrends]);
  const creatorTrendLabels = creatorTrends?.labels ?? [];
  const hasMoreCreatorEarnings = hasCreatorListNextPage(creatorEarningsLog);
  const hasMoreCreatorPaidContent = hasCreatorListNextPage(creatorPaidContent);
  const hasMoreCreatorQualityRewards = hasCreatorListNextPage(creatorQualityRewards);

  useEffect(() => {
    if (!getStoredAccessToken()) {
      return;
    }

    let mounted = true;

    Promise.allSettled([
      getCreatorOverview(),
      getCreatorStats(30),
      getCreatorTrends(14),
      getCreatorEarningsLog({ limit: creatorEarningsLogLimit }),
      getCreatorPaidContent({ limit: creatorPaidContentLimit }),
      getCreatorQualityRewards({ limit: creatorQualityRewardsLimit }),
    ]).then(
      ([
        overviewResult,
        statsResult,
        trendsResult,
        earningsLogResult,
        paidContentResult,
        qualityRewardsResult,
      ]) => {
        if (!mounted) {
          return;
        }

        if (overviewResult.status === "fulfilled") {
          setCreatorOverview(overviewResult.value);
        }

        if (statsResult.status === "fulfilled") {
          setCreatorStats(statsResult.value);
        }

        if (trendsResult.status === "fulfilled") {
          setCreatorTrends(trendsResult.value);
        }

        if (earningsLogResult.status === "fulfilled") {
          setCreatorEarningsLog(earningsLogResult.value);
        }

        if (paidContentResult.status === "fulfilled") {
          setCreatorPaidContent(paidContentResult.value);
        }

        if (qualityRewardsResult.status === "fulfilled") {
          setCreatorQualityRewards(qualityRewardsResult.value);
        }
      },
    );

    return () => {
      mounted = false;
    };
  }, []);

  function handleCreate(actionMode: PublishMode) {
    onCreate(actionMode);
  }

  async function handleLoadMoreCreatorPanel(panel: CreatorListPanel) {
    if (loadingCreatorPanel) {
      return;
    }

    setLoadingCreatorPanel(panel);
    try {
      if (panel === "earnings") {
        const payload = await getCreatorEarningsLog({
          limit: creatorEarningsLogLimit,
          page: nextCreatorListPage(creatorEarningsLog),
        });
        setCreatorEarningsLog((current) =>
          current ? { ...payload, list: mergeCreatorItems(current.list, payload.list) } : payload,
        );
        return;
      }

      if (panel === "paid") {
        const payload = await getCreatorPaidContent({
          limit: creatorPaidContentLimit,
          page: nextCreatorListPage(creatorPaidContent),
        });
        setCreatorPaidContent((current) =>
          current ? { ...payload, list: mergeCreatorItems(current.list, payload.list) } : payload,
        );
        return;
      }

      const payload = await getCreatorQualityRewards({
        limit: creatorQualityRewardsLimit,
        page: nextCreatorListPage(creatorQualityRewards),
      });
      setCreatorQualityRewards((current) =>
        current ? { ...payload, list: mergeCreatorItems(current.list, payload.list) } : payload,
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("publish.workbench.creator.detailsLoadFailed"));
    } finally {
      setLoadingCreatorPanel(null);
    }
  }

  return (
    <div className="mx-auto min-h-[calc(100vh-64px)] w-full max-w-[1440px] px-4 py-6 sm:px-5 lg:px-6 2xl:px-8">
      <div className="min-w-0 space-y-5">
        <section className="grid gap-5">
          <div className="rounded-2xl bg-white p-6 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
            <div className="flex flex-wrap items-center gap-5">
              <div className="relative flex size-24 shrink-0 items-center justify-center rounded-full bg-[#141420] text-white">
                <span className="text-2xl font-semibold">Y</span>
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h1 className="truncate text-xl font-semibold text-[#25252b]">
                    {t("publish.home.profile.name")}
                  </h1>
                </div>
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-[#67676f]">
                  <span>{t("publish.home.profile.following", { count: 0 })}</span>
                  <span>{t("publish.home.profile.followers", { count: creatorFans })}</span>
                  <span>{t("publish.home.profile.likes", { count: creatorLikesAndCollections })}</span>
                </div>
                <p className="mt-3 truncate text-sm text-[#8a8a91]">
                  {t("publish.home.profile.id")}
                </p>
              </div>
            </div>
          </div>
        </section>

        <section className="rounded-2xl bg-white p-6 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
          <SectionHeader
            title={t("publish.home.create.title")}
            description={t("publish.home.create.description")}
          />
          <div className="mt-5 grid grid-cols-[repeat(auto-fit,minmax(min(100%,220px),1fr))] gap-4">
            {homeCreateActions.map(({ key, icon: Icon, mode: actionMode, className }) => (
              <button
                key={key}
                type="button"
                onClick={() => handleCreate(actionMode)}
                className="flex min-h-[118px] min-w-0 items-center gap-4 rounded-2xl bg-[#fafafa] p-5 text-left transition-[background-color,transform] duration-200 ease-out hover:-translate-y-0.5 hover:bg-[#f6f6f7] active:scale-[0.98]"
              >
                <span className={cn("flex size-14 shrink-0 items-center justify-center rounded-2xl", className)}>
                  <Icon className="size-7" />
                </span>
                <span className="min-w-0">
                  <span className="block text-base font-semibold text-[#25252b]">
                    {t(`publish.home.create.${key}Title`)}
                  </span>
                  <span className="mt-1 block text-sm leading-5 text-[#777780]">
                    {t(`publish.home.create.${key}Body`)}
                  </span>
                </span>
              </button>
            ))}
          </div>
        </section>

        <section className="rounded-2xl bg-white p-6 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
          <div className="flex flex-wrap items-center gap-4 border-b border-[#ededf0] pb-3">
            {metricsViews.map((view) => {
              const active = metricsView === view;

              return (
                <button
                  key={view}
                  type="button"
                  onClick={() => setMetricsView(view)}
                  className={cn(
                    "relative h-8 text-base font-semibold transition-colors",
                    active ? "text-[#25252b]" : "text-[#777780] hover:text-[#25252b]",
                  )}
                >
                  {t(`publish.home.metrics.${view}Overview`)}
                  {active ? (
                    <span
                      className="absolute inset-x-0 -bottom-3 h-0.5 rounded-full bg-primary"
                    />
                  ) : null}
                </button>
              );
            })}
            <button type="button" className="ml-auto hidden items-center gap-1 text-sm font-medium text-[#777780] hover:text-primary sm:flex">
              {t("publish.home.metrics.details")}
              <ArrowRight className="size-4" />
            </button>
          </div>

          <div className="mt-5 flex flex-wrap items-center justify-between gap-3 text-sm text-[#8a8a91]">
            <span>{t(`publish.home.metrics.${activePeriodKey}`)}</span>
            <div className="flex flex-wrap rounded-xl bg-[#f6f6f7] p-1 text-xs font-semibold">
              {(metricsView === "note" ? noteRanges : liveRanges).map((range) => {
                const active = activeRange === range;

                return (
                  <button
                    key={range}
                    type="button"
                    onClick={() => {
                      if (metricsView === "note") {
                        setNoteRange(range as NoteRange);
                      } else {
                        setLiveRange(range as LiveRange);
                      }
                    }}
                    className={cn(
                      "min-h-8 rounded-lg px-3 py-1.5 transition-colors",
                      active ? "bg-white text-[#25252b] shadow-sm" : "text-[#777780] hover:text-[#25252b]",
                    )}
                  >
                    {t(`publish.home.metrics.${range}`)}
                  </button>
                );
              })}
            </div>
          </div>

          <div
            className={cn(
              "mt-6 grid grid-cols-2 gap-x-0 gap-y-8 md:grid-cols-3",
              metricsView === "note" ? "xl:grid-cols-6" : "xl:grid-cols-4",
            )}
          >
            {metricKeys.map((key, index) => {
              const isRateMetric = key === "coverClicks" || key === "completionRate";
              const xlColumnCount = metricsView === "note" ? 6 : 4;

              return (
                <div
                  key={key}
                  className={cn(
                    "min-h-[76px] border-[#eeeeef] px-4",
                    index % 3 !== 0 && "md:border-l",
                    index % xlColumnCount !== 0 && "xl:border-l",
                  )}
                >
                  <p className="text-sm text-[#777780]">{t(`publish.home.metrics.${key}`)}</p>
                  <p className="mt-2 text-2xl font-semibold text-[#1f1f24]">
                    {isRateMetric ? "0%" : formatCreatorMetricValue(metricValues[key] ?? 0)}
                  </p>
                  {metricsView === "note" ? (
                    <p className="mt-1 text-xs text-[#9a9aa1]">
                      {t("publish.home.metrics.change")}
                    </p>
                  ) : null}
                </div>
              );
            })}
          </div>
        </section>

        <section className="rounded-2xl bg-white p-6 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
          <SectionHeader title={t("publish.workbench.creator.title")} description={t("publish.workbench.creator.description")} />

          <div className="mt-5 grid grid-cols-[repeat(auto-fit,minmax(min(100%,360px),1fr))] gap-5">
            <div className="min-w-0 rounded-2xl bg-[#fafafa] p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold text-[#25252b]">{t("publish.workbench.creator.trendTitle")}</h3>
                  <p className="mt-1 text-xs text-[#8a8a91]">{t("publish.workbench.creator.trendHint")}</p>
                </div>
                <div className="flex flex-wrap gap-2 text-xs text-[#777780]">
                  {creatorTrendSeries.map((series) => (
                    <span key={series.key} className="inline-flex items-center gap-1.5">
                      <span className={cn("size-2 rounded-full", series.className)} />
                      {t(`publish.workbench.creator.${series.labelKey}`)} {formatCreatorMetricValue(getCreatorTrendTotal(creatorTrends, series.key))}
                    </span>
                  ))}
                </div>
              </div>

              {creatorTrendLabels.length > 0 ? (
                <div className="mt-5 overflow-hidden">
                  <div className="flex h-[188px] items-end gap-2 border-b border-[#e7e7ea] px-1">
                    {creatorTrendLabels.map((label, index) => (
                      <div key={`${label}-${index}`} className="flex min-w-0 flex-1 flex-col items-center gap-1">
                        <div className="flex h-[160px] w-full items-end justify-center gap-0.5">
                          {creatorTrendSeries.map((series) => {
                            const value = getCreatorTrendValue(creatorTrends, series.key, index);
                            const height = Math.max(4, Math.round((value / creatorTrendMax) * 154));

                            return (
                              <span
                                key={series.key}
                                title={`${label} ${t(`publish.workbench.creator.${series.labelKey}`)}: ${value}`}
                                className={cn("w-full max-w-[9px] rounded-t-sm", series.className)}
                                style={{ height }}
                              />
                            );
                          })}
                        </div>
                        <span className="w-full truncate text-center text-[11px] text-[#9a9aa1]">
                          {label}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <CreatorEmptyState label={t("publish.workbench.creator.trendEmpty")} />
              )}
            </div>

            <div className="grid min-w-0 grid-cols-[repeat(auto-fit,minmax(min(100%,220px),1fr))] gap-5">
              <div className="min-w-0 rounded-2xl bg-[#fafafa] p-4">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-semibold text-[#25252b]">{t("publish.workbench.creator.earningsLog")}</h3>
                  <span className="text-xs text-[#8a8a91]">
                    {t("publish.workbench.creator.itemCount", { count: creatorEarningsLog?.pagination.total ?? 0 })}
                  </span>
                </div>
                <div className="mt-4 rounded-xl bg-white p-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="text-xs font-semibold text-[#55555d]">{t("publish.workbench.creator.qualityReward")}</p>
                    <p className="shrink-0 text-sm font-semibold text-primary">
                      {t("publish.workbench.creator.moonCoinAmount", { amount: formatCreatorCurrency(creatorQualityRewards?.total_earnings ?? 0) })}
                    </p>
                  </div>
                  {creatorQualityRewards?.stats?.length ? (
                    <div className="mt-3 grid grid-cols-[repeat(auto-fit,minmax(min(100%,72px),1fr))] gap-2">
                      {creatorQualityRewards.stats.map((stat) => (
                        <div key={stat.quality_label} className="min-w-0 rounded-lg bg-[#fafafa] p-2">
                          <p className="truncate text-[11px] text-[#8a8a91]">{stat.quality_label}</p>
                          <p className="mt-1 text-sm font-semibold text-[#25252b]">{stat.count}</p>
                        </div>
                      ))}
                    </div>
                  ) : null}
                  <div className="mt-3 space-y-3">
                    {(creatorQualityRewards?.list ?? []).length > 0 ? (
                      creatorQualityRewards?.list.map((item) => (
                        <div key={item.id} className="min-w-0 border-t border-[#ededf0] pt-3 first:border-t-0 first:pt-0">
                          <div className="flex items-center justify-between gap-3">
                            <p className="truncate text-sm font-medium text-[#25252b]">
                              {item.post?.title || t("publish.workbench.creator.qualityContentReward")}
                            </p>
                            <p className="shrink-0 text-sm font-semibold text-primary">
                              +{formatCreatorCurrency(item.amount)}
                            </p>
                          </div>
                          <p className="mt-1 truncate text-xs text-[#8a8a91]">
                            {item.reason || item.post?.quality_level || formatCreatorDate(item.created_at) || t("publish.workbench.creator.qualityRatingReward")}
                          </p>
                        </div>
                      ))
                    ) : (
                      <p className="text-xs text-[#9a9aa1]">{t("publish.workbench.creator.noQualityRewards")}</p>
                    )}
                  </div>
                  {hasMoreCreatorQualityRewards ? (
                    <CreatorLoadMoreButton
                      label={t("publish.workbench.creator.loadMore")}
                      loading={loadingCreatorPanel === "rewards"}
                      onClick={() => void handleLoadMoreCreatorPanel("rewards")}
                    />
                  ) : null}
                </div>
                <div className="mt-4 space-y-3">
                  {(creatorEarningsLog?.list ?? []).length > 0 ? (
                    creatorEarningsLog?.list.map((item) => (
                      <div key={item.id} className="min-w-0 border-b border-[#ededf0] pb-3 last:border-b-0 last:pb-0">
                        <div className="flex items-center justify-between gap-3">
                          <p className="truncate text-sm font-medium text-[#25252b]">
                            {t(`publish.workbench.creator.${creatorEarningsTypeKey(item.type)}`)}
                          </p>
                          <p className={cn("shrink-0 text-sm font-semibold", item.amount < 0 ? "text-[#777780]" : "text-primary")}>
                            {item.amount < 0 ? "-" : "+"}
                            {formatCreatorCurrency(Math.abs(item.amount))}
                          </p>
                        </div>
                        <p className="mt-1 truncate text-xs text-[#8a8a91]">
                          {item.reason || item.source?.title || item.buyer?.nickname || formatCreatorDate(item.created_at) || t("publish.workbench.creator.noRemark")}
                        </p>
                      </div>
                    ))
                  ) : (
                    <CreatorEmptyState label={t("publish.workbench.creator.noEarnings")} compact />
                  )}
                </div>
                {hasMoreCreatorEarnings ? (
                  <CreatorLoadMoreButton
                    label={t("publish.workbench.creator.loadMore")}
                    loading={loadingCreatorPanel === "earnings"}
                    onClick={() => void handleLoadMoreCreatorPanel("earnings")}
                  />
                ) : null}
              </div>

              <div className="min-w-0 rounded-2xl bg-[#fafafa] p-4">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-semibold text-[#25252b]">{t("publish.workbench.creator.paidContent")}</h3>
                  <span className="text-xs text-[#8a8a91]">
                    {t("publish.workbench.creator.postCount", { count: creatorPaidContent?.pagination.total ?? 0 })}
                  </span>
                </div>
                <div className="mt-4 space-y-3">
                  {(creatorPaidContent?.list ?? []).length > 0 ? (
                    creatorPaidContent?.list.map((item) => (
                      <div key={item.id} className="grid min-w-0 grid-cols-[48px_minmax(0,1fr)] gap-3">
                        <div className="relative aspect-square overflow-hidden rounded-xl bg-[#eeeeef]">
                          {item.cover ? (
                            <Image
                              src={item.cover}
                              alt={item.title}
                              fill
                              unoptimized
                              sizes="48px"
                              className="object-cover"
                            />
                          ) : (
                            <div className="flex size-full items-center justify-center text-[#b0b0b8]">
                              <FileText className="size-5" />
                            </div>
                          )}
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <p className="truncate text-sm font-medium text-[#25252b]">{item.title}</p>
                            <span className="shrink-0 rounded-full bg-white px-2 py-0.5 text-[11px] text-[#777780]">
                              {t(`publish.workbench.creator.${creatorContentTypeKey(item.type)}`)}
                            </span>
                          </div>
                          <p className="mt-1 truncate text-xs text-[#8a8a91]">
                            {t("publish.workbench.creator.salesRevenue", { count: item.sales_count, amount: formatCreatorCurrency(item.total_revenue) })}
                          </p>
                        </div>
                      </div>
                    ))
                  ) : (
                    <CreatorEmptyState label={t("publish.workbench.creator.noPaidContent")} compact />
                  )}
                </div>
                {hasMoreCreatorPaidContent ? (
                  <CreatorLoadMoreButton
                    label={t("publish.workbench.creator.loadMore")}
                    loading={loadingCreatorPanel === "paid"}
                    onClick={() => void handleLoadMoreCreatorPanel("paid")}
                  />
                ) : null}
              </div>
            </div>
          </div>
        </section>

      </div>
    </div>
  );
}
