"use client";

import { useCallback, useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  BarChart3,
  Bell,
  ChevronRight,
  CircleDollarSign,
  FileText,
  RefreshCw,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";
import {
  getCreatorEarningsLog,
  getCreatorOverview,
  getCurrentUser,
  getNotificationUnreadCount,
  getStoredAccessToken,
  getStoredUser,
} from "@/lib/api";
import type {
  AuthUser,
  CreatorEarningsLogItem,
  CreatorOverviewPayload,
  CreatorStatsPayload,
  CreatorTrendsPayload,
} from "@/lib/types";
import type { CreatorCenterInitialData } from "@/lib/server/creator-center-page-data";
import { cn } from "@/lib/utils";

export type RangeDays = 7 | 30 | 90;
export type TrendKey = "followers" | "views" | "likes" | "collects" | "comments";
export type ActivityTone = "amber" | "blue" | "green" | "pink" | "purple";

export type ActivityItem = {
  amount?: number;
  description: string;
  id: string;
  time: string;
  title: string;
  tone: ActivityTone;
};

export const creatorDataRanges: Array<{ days: RangeDays; label: string }> = [
  { days: 7, label: "近7天" },
  { days: 30, label: "近30天" },
  { days: 90, label: "近90天" },
];

const shortcuts = [
  { href: "/profile", icon: FileText, label: "作品管理", tone: "pink" },
  { href: "/creator-center/data", icon: BarChart3, label: "数据中心", tone: "blue" },
] as const;

const shortcutToneClassNames = {
  blue: "bg-[#eaf2ff] text-[#5688e8]",
  pink: "bg-[#ffeaf0] text-[#f26289]",
} satisfies Record<(typeof shortcuts)[number]["tone"], string>;

export const creatorTrendSeries: Array<{ color: string; key: TrendKey; label: string }> = [
  { key: "followers", label: "关注", color: "#f6a417" },
  { key: "views", label: "浏览", color: "#668eea" },
  { key: "likes", label: "点赞", color: "#f35f89" },
  { key: "collects", label: "收藏", color: "#5ac0a2" },
  { key: "comments", label: "评论", color: "#8a61e8" },
];

const fallbackOverview: CreatorOverviewPayload = {
  balance: 4.35,
  total_earnings: 36.03,
  withdrawn_amount: 0,
  today_earnings: 0,
  month_earnings: 0.02,
};

export const fallbackCreatorStats: CreatorStatsPayload = {
  fans: { recent7: 8, recent30: 14, recent90: 31, total: 232 },
  generated_at: "2026-06-12T00:00:00.000Z",
  interactions: {
    collects: { recent7: 1, recent30: 1, recent90: 3 },
    comments: { recent7: 0, recent30: 0, recent90: 1 },
    likes: { recent7: 3, recent30: 3, recent90: 9 },
    views: { recent7: 232, recent30: 232, recent90: 640 },
  },
  post_totals: { collect_count: 1, comment_count: 0, like_count: 3, view_count: 232 },
  window_days: 30,
};

export const fallbackCreatorTrends: CreatorTrendsPayload = {
  collects: [0, 0, 1, 0, 0, 0, 0],
  comments: [0, 0, 0, 0, 0, 0, 0],
  days: 7,
  followers: [0, 0, 3, 2, 5, 3, 0],
  labels: ["06/06", "06/07", "06/08", "06/09", "06/10", "06/11", "06/12"],
  likes: [0, 0, 2, 1, 4, 2, 0],
  views: [0, 0, 140, 92, 232, 128, 0],
};

type CreatorCenterData = {
  activities: ActivityItem[];
  authToken: string | null;
  currentUser: AuthUser | null;
  overview: CreatorOverviewPayload | null;
  overviewFailed: boolean;
  unreadCount: number;
};

type CreatorCenterCacheEntry = CreatorCenterData & {
  updatedAt: number;
};

const CREATOR_CENTER_CACHE_TTL_MS = 60 * 1000;
let creatorCenterCache: CreatorCenterCacheEntry | null = null;
let creatorCenterRequest: Promise<CreatorCenterData> | null = null;

function getCachedCreatorCenterData(): CreatorCenterData | null {
  if (!creatorCenterCache) {
    return null;
  }

  if (Date.now() - creatorCenterCache.updatedAt > CREATOR_CENTER_CACHE_TTL_MS) {
    creatorCenterCache = null;
    return null;
  }

  return {
    activities: creatorCenterCache.activities,
    authToken: creatorCenterCache.authToken,
    currentUser: creatorCenterCache.currentUser,
    overview: creatorCenterCache.overview,
    overviewFailed: creatorCenterCache.overviewFailed,
    unreadCount: creatorCenterCache.unreadCount,
  };
}

async function loadCreatorCenterData(options: { refresh?: boolean } = {}) {
  if (!options.refresh) {
    const cached = getCachedCreatorCenterData();
    if (cached) {
      return cached;
    }
  }

  if (creatorCenterRequest) {
    return creatorCenterRequest;
  }

  creatorCenterRequest = Promise.resolve()
    .then(async () => {
      const authToken = getStoredAccessToken();
      if (!authToken) {
        return {
          activities: [],
          authToken,
          currentUser: getStoredUser(),
          overview: null,
          overviewFailed: false,
          unreadCount: 0,
        } satisfies CreatorCenterData;
      }

      const [userResult, overviewResult, earningsResult, unreadResult] = await Promise.allSettled([
        getCurrentUser(),
        getCreatorOverview(),
        getCreatorEarningsLog({ limit: 5 }),
        getNotificationUnreadCount(),
      ]);

      const data: CreatorCenterData = {
        activities:
          earningsResult.status === "fulfilled"
            ? buildCreatorActivities(earningsResult.value.list, 5)
            : [],
        authToken,
        currentUser: userResult.status === "fulfilled" ? userResult.value : getStoredUser(),
        overview: overviewResult.status === "fulfilled" ? overviewResult.value : null,
        overviewFailed: overviewResult.status === "rejected",
        unreadCount: unreadResult.status === "fulfilled" ? unreadResult.value.total ?? 0 : 0,
      };

      creatorCenterCache = {
        ...data,
        updatedAt: Date.now(),
      };
      return data;
    })
    .finally(() => {
      creatorCenterRequest = null;
    });

  return creatorCenterRequest;
}

export function prefetchCreatorCenterData() {
  return loadCreatorCenterData();
}

export function MobileCreatorCenterPage({
  initialData,
}: {
  initialData?: CreatorCenterInitialData | null;
} = {}) {
  const router = useRouter();
  const [initialCreatorCenterData] = useState<CreatorCenterData | null>(() => {
    if (initialData) {
      return {
        activities: buildCreatorActivities(initialData.earnings, 5),
        authToken: initialData.authenticated ? "server-authenticated" : null,
        currentUser: initialData.currentUser,
        overview: initialData.overview,
        overviewFailed: initialData.overviewFailed,
        unreadCount: initialData.unreadCount,
      };
    }
    return getCachedCreatorCenterData();
  });
  const [authToken, setAuthToken] = useState<string | null | undefined>(
    () => initialCreatorCenterData?.authToken ?? undefined,
  );
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(
    () => initialCreatorCenterData?.currentUser ?? getStoredUser(),
  );
  const [overview, setOverview] = useState<CreatorOverviewPayload | null>(
    () => initialCreatorCenterData?.overview ?? null,
  );
  const [activities, setActivities] = useState<ActivityItem[]>(
    () => initialCreatorCenterData?.activities ?? [],
  );
  const [unreadCount, setUnreadCount] = useState(
    () => initialCreatorCenterData?.unreadCount ?? 0,
  );
  const [loading, setLoading] = useState(() => !initialCreatorCenterData);

  const safeOverview = overview ?? fallbackOverview;
  const displayName = getDisplayName(currentUser);
  const avatar = currentUser?.avatar?.trim() || "";

  const handleBack = useCallback(() => {
    const currentPath = window.location.pathname;
    if (window.history.length > 1) {
      router.back();
      window.setTimeout(() => {
        if (window.location.pathname === currentPath) {
          router.push("/");
        }
      }, 280);
      return;
    }
    router.push("/");
  }, [router]);

  const applyCreatorCenterData = useCallback((data: CreatorCenterData) => {
    setAuthToken(data.authToken);
    setCurrentUser(data.currentUser);
    setOverview(data.overview);
    setActivities(data.activities);
    setUnreadCount(data.unreadCount);
  }, []);

  const loadCreatorCenter = useCallback(async (options?: { silent?: boolean }) => {
    const cached = getCachedCreatorCenterData();
    if (cached && !options?.silent) {
      applyCreatorCenterData(cached);
      setLoading(false);
    } else if (!options?.silent) {
      setLoading(true);
    }

    try {
      const data = await loadCreatorCenterData({ refresh: true });
      applyCreatorCenterData(data);
      if (data.overviewFailed && options?.silent) {
        toast.error("创作者数据刷新失败，已保留当前展示。");
      }
    } finally {
      setLoading(false);
    }
  }, [applyCreatorCenterData]);

  useEffect(() => {
    if (initialData) {
      return;
    }
    queueMicrotask(() => {
      if (!initialCreatorCenterData) {
        setCurrentUser(getStoredUser());
      }
      void loadCreatorCenter({ silent: Boolean(initialCreatorCenterData) });
    });
  }, [initialCreatorCenterData, initialData, loadCreatorCenter]);

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#24242c]">
      <div className="mx-auto min-h-dvh w-full max-w-[430px] bg-[#fbfbff] pb-8 shadow-[0_0_38px_rgba(112,94,173,0.08)]">
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
            <h1 className="flex min-w-0 items-center justify-center gap-1.5 text-[20px] font-black tracking-normal min-[390px]:gap-2 min-[390px]:text-[22px]">
              <Sparkles className="size-3.5 shrink-0 text-[#c6b5f3] min-[390px]:size-4" />
              <span className="truncate">创作者中心</span>
              <Sparkles className="size-3.5 shrink-0 text-[#c6b5f3] min-[390px]:size-4" />
            </h1>
            <Link
              href="/notifications"
              aria-label="通知"
              className="relative flex size-10 items-center justify-center justify-self-end rounded-full text-[#16161d] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <Bell className="size-6 min-[390px]:size-7" strokeWidth={2.3} />
              {unreadCount > 0 ? (
                <span className="absolute right-1 top-1 flex min-w-5 items-center justify-center rounded-full bg-[#ff284b] px-1 text-[10px] font-black leading-5 text-white ring-2 ring-[#fbfbff]">
                  {unreadCount > 99 ? "99+" : unreadCount}
                </span>
              ) : null}
            </Link>
          </div>
        </header>

        <section className="px-4">
          <div className="relative overflow-hidden rounded-[8px] bg-[#8f88ef] px-4 pb-4 pt-5 text-white shadow-[0_16px_38px_rgba(110,88,210,0.24)]">
            <Image
              src="/creator-center/creator-banner.png"
              alt=""
              fill
              priority
              sizes="430px"
              className="object-cover"
            />
            <div className="absolute inset-0 bg-gradient-to-r from-[#7164d7]/86 via-[#8b82ee]/56 to-transparent" />
            <div className="relative z-10 grid min-h-[178px] grid-rows-[auto_1fr_auto]">
              <div className="flex min-w-0 items-center gap-3">
                <span className="relative flex size-14 shrink-0 overflow-hidden rounded-full bg-white/22 ring-2 ring-white/45 min-[390px]:size-16">
                  {avatar ? (
                    <Image src={avatar} alt={displayName} fill unoptimized sizes="64px" className="object-cover" />
                  ) : (
                    <span className="flex size-full items-center justify-center bg-[#f1eaff] text-2xl font-black text-[#8070de]">
                      {displayName.charAt(0).toUpperCase()}
                    </span>
                  )}
                </span>
                <div className="min-w-0">
                  <div className="flex min-w-0 items-center gap-2">
                    <h2 className="truncate text-[20px] font-black leading-tight min-[390px]:text-[22px]">{displayName}</h2>
                  </div>
                </div>
              </div>

              <div className="mt-6">
                <p className="flex items-center gap-2 text-[13px] font-bold text-white/78">
                  可提现金额（元）
                  <CircleDollarSign className="size-4 text-white/68" />
                </p>
                <div className="mt-2 flex flex-wrap items-center gap-3 min-[390px]:gap-4">
                  <p className="min-w-0 text-[28px] font-black leading-none min-[390px]:text-[30px]">¥{formatCreatorMoney(safeOverview.balance)}</p>
                  <Link
                    href="/wallet/withdraw"
                    className="flex h-9 min-w-20 items-center justify-center rounded-full bg-[#c9a7ff]/55 px-4 text-[14px] font-black text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.18)] active:scale-[0.98] min-[390px]:min-w-24 min-[390px]:px-5"
                  >
                    提现
                  </Link>
                </div>
              </div>

              <div className="mt-5 grid grid-cols-3 gap-2 text-center">
                <HeroMetric label="今日收益（元）" value={`+${formatCreatorMoney(safeOverview.today_earnings)}`} />
                <HeroMetric label="本月收益（元）" value={`+${formatCreatorMoney(safeOverview.month_earnings)}`} />
                <HeroMetric label="累计收益（元）" value={formatCreatorMoney(safeOverview.total_earnings)} />
              </div>
            </div>
          </div>
        </section>

        <section className="mx-4 mt-4 rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
          <div className="grid grid-cols-2 gap-3">
            {shortcuts.map(({ href, icon: Icon, label, tone }) => (
              <Link key={label} href={href} className="flex min-w-0 flex-col items-center gap-2 active:scale-[0.98]">
                <span
                  className={cn(
                    "flex size-11 items-center justify-center rounded-[12px] shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]",
                    shortcutToneClassNames[tone],
                  )}
                >
                  <Icon className="size-5.5 min-[390px]:size-6" strokeWidth={2.4} />
                </span>
                <span className="max-w-full truncate text-[12px] font-bold text-[#6d6b78]">{label}</span>
              </Link>
            ))}
          </div>
        </section>

        <section className="mx-4 mt-4 rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-[20px] font-black tracking-normal">近期收益</h2>
            <Link
              href="/creator-center/earnings"
              className="flex h-9 items-center gap-1 text-[13px] font-black text-[#8b8796]"
            >
              查看更多
              <ChevronRight className="size-4" />
            </Link>
          </div>

          {activities.length ? (
            <div className="mt-3 divide-y divide-[#f0eef7]">
              {activities.map((item) => (
                <CreatorEarningListItem key={item.id} item={item} />
              ))}
            </div>
          ) : (
            <div className="mt-4 rounded-[8px] bg-[#fbfaff] px-4 py-6 text-center text-[13px] font-bold text-[#9a96a5]">
              暂无真实收益记录
            </div>
          )}
        </section>

        {authToken === null ? (
          <section className="mx-4 mt-4 rounded-[8px] border border-[#eee8ff] bg-white px-4 py-4 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
            <p className="text-[15px] font-black text-[#23232a]">登录后可查看真实创作者数据</p>
            <Link href="/login" className="mt-3 inline-flex h-9 items-center justify-center rounded-full bg-[#8069dd] px-5 text-[13px] font-black text-white">
              去登录
            </Link>
          </section>
        ) : loading ? (
          <div className="mx-4 mt-4 flex h-10 items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <RefreshCw className="size-4 animate-spin" />
            正在同步创作者数据
          </div>
        ) : null}
      </div>

    </main>
  );
}

function HeroMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 border-l border-white/18 first:border-l-0 first:pl-0">
      <p className="truncate text-[20px] font-black leading-none">{value}</p>
      <p className="mt-2 truncate text-[12px] font-bold text-white/78">{label}</p>
    </div>
  );
}

export function CreatorTrendChart({
  labels,
  max,
  series,
}: {
  labels: string[];
  max: number;
  series: CreatorTrendsPayload;
}) {
  const chartHeight = 116;
  const chartWidth = 232;
  const points = series.followers.map((value, index) => {
    const x = labels.length <= 1 ? 0 : (index / (labels.length - 1)) * chartWidth;
    const y = chartHeight - (value / max) * (chartHeight - 8);
    return `${x},${y}`;
  });
  const areaPoints = [`0,${chartHeight}`, ...points, `${chartWidth},${chartHeight}`].join(" ");

  return (
    <div className="mt-3">
      <svg viewBox={`0 0 ${chartWidth} ${chartHeight}`} className="h-[132px] w-full overflow-visible" role="img" aria-label="近7天粉丝趋势图">
        {[0, 1, 2, 3, 4].map((lineIndex) => {
          const y = (lineIndex / 4) * chartHeight;
          return <line key={lineIndex} x1="0" x2={chartWidth} y1={y} y2={y} stroke="#ebe8f3" strokeWidth="1" />;
        })}
        <defs>
          <linearGradient id="creatorTrendArea" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="#8a61e8" stopOpacity="0.28" />
            <stop offset="100%" stopColor="#8a61e8" stopOpacity="0" />
          </linearGradient>
        </defs>
        <polygon points={areaPoints} fill="url(#creatorTrendArea)" />
        <polyline points={points.join(" ")} fill="none" stroke="#7059dc" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.2" />
        {points.map((point, index) => {
          const [x, y] = point.split(",").map(Number);
          return <circle key={`${point}-${index}`} cx={x} cy={y} r="3.6" fill="#7059dc" stroke="#fff" strokeWidth="2" />;
        })}
      </svg>

      <div className="mt-1 grid grid-cols-7 gap-1 text-center text-[11px] font-bold text-[#8d8998]">
        {labels.slice(-7).map((label) => (
          <span key={label} className="truncate">{label}</span>
        ))}
      </div>
      <div className="mt-3 flex flex-wrap justify-center gap-x-3 gap-y-1">
        {creatorTrendSeries.map((item) => (
          <span key={item.key} className="inline-flex items-center gap-1 text-[12px] font-bold text-[#777381]">
            <span className="size-2 rounded-full" style={{ backgroundColor: item.color }} />
            {item.label}
          </span>
        ))}
      </div>
    </div>
  );
}

export function buildCreatorActivities(items: CreatorEarningsLogItem[], limit?: number): ActivityItem[] {
  const source = typeof limit === "number" ? items.slice(0, limit) : items;
  return source.map((item) => ({
    amount: item.amount,
    description: getActivityDescription(item),
    id: String(item.id),
    time: formatActivityTime(item.created_at),
    title: getActivityTitle(item),
    tone: "amber",
  }));
}

export function CreatorEarningListItem({ item }: { item: ActivityItem }) {
  return (
    <article className="grid grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-3 py-3">
      <span className="flex size-11 items-center justify-center rounded-[10px] bg-[#fff7db] text-[#f0a000]">
        <CircleDollarSign className="size-5" />
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-[15px] font-black text-[#1d1d24]">{item.title}</h3>
        <p className="mt-1 truncate text-[13px] font-bold text-[#9a96a5]">{item.description}</p>
      </div>
      <div className="flex items-center gap-2">
        <span className="hidden text-[13px] font-bold text-[#9a96a5] min-[390px]:inline">{item.time}</span>
        {item.amount !== undefined ? (
          <span className={cn("text-[14px] font-black", item.amount >= 10 ? "text-[#f0a000]" : "text-[#42b389]")}>
            {item.amount >= 0 ? "+" : ""}
            {formatCreatorMoney(item.amount)}
          </span>
        ) : null}
        <ChevronRight className="size-4 text-[#c9c5d3]" />
      </div>
    </article>
  );
}

function getActivityDescription(item: CreatorEarningsLogItem) {
  const reason = item.reason?.trim();
  if (reason) {
    return reason;
  }

  const sourceTitle = item.source?.title?.trim();
  if (sourceTitle) {
    return `关联作品：${sourceTitle}`;
  }

  const buyerName = item.buyer?.nickname?.trim() || item.buyer?.user_id?.trim();
  if (buyerName) {
    return `来自 ${buyerName}`;
  }

  return "后端收益日志";
}

function getActivityTitle(item: CreatorEarningsLogItem) {
  const type = item.type?.trim();

  if (type === "content_sale") {
    return "内容销售";
  }
  if (type === "extended_daily") {
    return "流量激励";
  }
  if (type === "quality_reward") {
    return "优质奖励";
  }
  if (type === "transfer_from_wallet") {
    return "钱包转入";
  }
  if (type === "withdraw") {
    return "收益提现";
  }
  if (type === "withdraw_rejected") {
    return "提现退回";
  }
  if (type?.includes("reward")) {
    return "收益到账";
  }
  if (type?.includes("tip")) {
    return "打赏收入";
  }
  return type || "创作动态";
}

function getDisplayName(user: AuthUser | null) {
  return user?.nickname?.trim() || user?.user_id?.trim() || user?.xise_id?.trim() || "YueM";
}

export function getCreatorMetric(values: Record<string, number> | undefined, key: string, fallback: number): number {
  const value = values?.[key];
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

export function getCreatorRangeKey(days: RangeDays) {
  return `recent${days}`;
}

export function normalizeCreatorTrends(trends: CreatorTrendsPayload, activeRange: RangeDays): CreatorTrendsPayload {
  const count = Math.min(7, Math.max(1, trends.labels.length || 7));
  const fallbackLabels = fallbackCreatorTrends.labels.slice(-count);
  return {
    collects: normalizeSeries(trends.collects, count),
    comments: normalizeSeries(trends.comments, count),
    days: activeRange,
    followers: normalizeSeries(trends.followers, count),
    labels: trends.labels.length ? trends.labels.slice(-count) : fallbackLabels,
    likes: normalizeSeries(trends.likes, count),
    views: normalizeSeries(trends.views, count),
  };
}

function normalizeSeries(values: number[], count: number) {
  const source = values.length ? values : new Array(count).fill(0);
  return source.slice(-count).map((value) => (Number.isFinite(value) ? value : 0));
}

export function getCreatorMetricCardClassName(tone: ActivityTone) {
  if (tone === "amber") {
    return "bg-[#fffaf0]";
  }
  if (tone === "blue") {
    return "bg-[#f4f7ff]";
  }
  if (tone === "green") {
    return "bg-[#f3fbf7]";
  }
  if (tone === "pink") {
    return "bg-[#fff5f8]";
  }
  return "bg-[#f8f5ff]";
}

export function formatCreatorMoney(value: number) {
  const amount = Number.isFinite(value) ? value : 0;
  return amount.toLocaleString("zh-CN", {
    maximumFractionDigits: 2,
    minimumFractionDigits: amount % 1 === 0 ? 2 : 2,
  });
}

export function formatCompactCount(value: number) {
  if (!Number.isFinite(value)) {
    return "0";
  }
  if (value >= 10000) {
    return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}w`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}k`;
  }
  return String(Math.round(value));
}

function formatActivityTime(value?: string) {
  if (!value) {
    return "刚刚";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
  }).format(date);
}
