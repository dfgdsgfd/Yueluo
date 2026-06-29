"use client";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  Activity,
  ClipboardList,
  FileText,
  Lock,
  Radio,
  ShieldCheck
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  getAdminAiReviewStatus,
  getAdminDashboardHotContent,
  getAdminDashboardOverview,
  getAdminDashboardTrends,
  getAdminGuestAccessStatus,
  getAdminStatsOverview,
  toggleAdminAiReview,
  toggleAdminGuestAccess
} from "@/lib/api";
import type {
  AdminAiReviewStatusPayload,
  AdminDashboardHotContentPayload,
  AdminDashboardOverviewPayload,
  AdminDashboardTrendsPayload,
  AdminGuestAccessStatusPayload
} from "@/lib/types";
import {
  HeroPill,
  HotContentList,
  MetricCard,
  Panel,
  ToggleCard,
  TrendChart
} from "./layout-widgets";
import {
  LoadingBlock
} from "./resource-editor";
import {
  StatusPill
} from "./resource-cells";
import {
  errorMessage,
  formatCompact,
  formatDateTime,
  formatMoney,
  pendingLabel,
  statsToOverview
} from "./helpers";

export function DashboardPanel({ token }: { token: string }) {
  const [overview, setOverview] = useState<AdminDashboardOverviewPayload | null>(null);
  const [trends, setTrends] = useState<AdminDashboardTrendsPayload | null>(null);
  const [hot, setHot] = useState<AdminDashboardHotContentPayload | null>(null);
  const [aiReview, setAiReview] = useState<AdminAiReviewStatusPayload | null>(null);
  const [guestAccess, setGuestAccess] = useState<AdminGuestAccessStatusPayload | null>(null);
  const [errors, setErrors] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [actingToggle, setActingToggle] = useState<"ai" | "guest" | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    const results = await Promise.allSettled([
      getAdminDashboardOverview(token),
      getAdminDashboardTrends(7, token),
      getAdminDashboardHotContent(token),
      getAdminAiReviewStatus(token),
      getAdminGuestAccessStatus(token),
    ]);
    const nextErrors: string[] = [];

    if (results[0].status === "fulfilled") {
      setOverview(results[0].value);
    } else {
      nextErrors.push(`运营概览：${errorMessage(results[0].reason)}`);
      try {
        const fallback = await getAdminStatsOverview(token);
        setOverview(statsToOverview(fallback));
      } catch (error) {
        nextErrors.push(`基础统计：${errorMessage(error)}`);
      }
    }
    if (results[1].status === "fulfilled") setTrends(results[1].value);
    else nextErrors.push(`趋势：${errorMessage(results[1].reason)}`);
    if (results[2].status === "fulfilled") setHot(results[2].value);
    else nextErrors.push(`热门内容：${errorMessage(results[2].reason)}`);
    if (results[3].status === "fulfilled") setAiReview(results[3].value);
    else nextErrors.push(`自动审核：${errorMessage(results[3].reason)}`);
    if (results[4].status === "fulfilled") setGuestAccess(results[4].value);
    else nextErrors.push(`访客访问：${errorMessage(results[4].reason)}`);

    setErrors(nextErrors);
    setLoading(false);
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  async function handleToggle(kind: "ai" | "guest") {
    setActingToggle(kind);
    try {
      if (kind === "ai") {
        const nextEnabled = !(aiReview?.enabled ?? false);
        await toggleAdminAiReview(nextEnabled, token);
        setAiReview({ enabled: nextEnabled, username_enabled: nextEnabled, content_enabled: nextEnabled });
      } else {
        const nextRestricted = !(guestAccess?.restricted ?? false);
        await toggleAdminGuestAccess(nextRestricted, token);
        setGuestAccess({ restricted: nextRestricted, note_restricted: nextRestricted, video_restricted: nextRestricted, admin_restricted: true });
      }
      toast.success("状态已更新");
      void load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActingToggle(null);
    }
  }

  if (loading && !overview) {
    return <LoadingBlock label="正在加载运营看板" />;
  }

  const metrics = overview?.metrics ?? [];
  const pending = overview?.pending ?? {};
  const finance = overview?.finance ?? {};

  return (
    <div className="grid gap-4">
      <section className="overflow-hidden rounded-lg border border-black/[0.06] bg-white shadow-[0_14px_45px_rgba(20,20,35,0.06)]">
        <div className="grid gap-4 bg-[linear-gradient(135deg,#f8fafc_0%,#ffffff_46%,#eff6ff_100%)] px-4 py-5 sm:px-6 lg:grid-cols-[minmax(0,1fr)_360px] lg:px-7">
          <div className="min-w-0">
            <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-[#1d4ed8]/15 bg-white/70 px-3 py-1 text-xs font-semibold text-[#1d4ed8]">
              <ShieldCheck className="size-3.5" />
              管理控制台
            </div>
            <h1 className="text-[clamp(1.45rem,2vw,2.1rem)] font-semibold leading-tight tracking-normal text-[#16161d]">
              运营总览
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-[#6a6f7c]">
              重点关注审核、提现、队列与系统开关，常用操作已收敛为可搜索、可确认的业务入口。
            </p>
            <div className="mt-4 flex flex-wrap gap-2 text-xs text-[#6a6f7c]">
              <span className="rounded-full bg-white/80 px-3 py-1">更新时间 {formatDateTime(overview?.generated_at)}</span>
              <span className="rounded-full bg-white/80 px-3 py-1">对象选择操作</span>
              <span className="rounded-full bg-white/80 px-3 py-1">后端能力对齐</span>
            </div>
          </div>
          <div className="grid min-w-0 grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-2">
            <HeroPill label="待审核" value={pending.content_review ?? 0} tone="amber" />
            <HeroPill label="举报待处理" value={pending.reports ?? 0} tone="red" />
            <HeroPill label="提现待审" value={pending.withdraw ?? 0} tone="green" />
            <HeroPill label="预计收入" value={formatMoney(finance.creator_total_earnings)} tone="blue" />
          </div>
        </div>
      </section>

      {errors.length ? (
        <div className="grid gap-2">
          {errors.map((error) => (
            <div key={error} className="rounded-lg border border-amber-300/60 bg-amber-50 px-3 py-2 text-sm text-amber-900">
              {error}
            </div>
          ))}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        {metrics.map((metric) => (
          <MetricCard key={metric.key} metric={metric} />
        ))}
      </section>

      <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(340px,0.65fr)]">
        <Panel title="社区数据趋势" icon={Activity} action={<span className="text-xs text-[#8a8f9d]">近 7 日</span>}>
          <div className="min-h-[320px] min-w-0">
            <TrendChart trends={trends} />
          </div>
        </Panel>
        <Panel title="实时状态" icon={Radio}>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            {(overview?.statuses ?? []).map((status) => (
              <div key={status.key} className="rounded-lg border border-black/[0.05] bg-[#fafbfe] p-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm font-semibold text-[#30333b]">{status.label}</span>
                  <StatusPill value={String(status.value)} tone={status.tone ?? "slate"} />
                </div>
                {status.description ? <p className="mt-1 text-xs text-[#7b808c]">{status.description}</p> : null}
              </div>
            ))}
            <ToggleCard
              icon={ShieldCheck}
              label="自动审核"
              description={`用户名 ${aiReview?.username_enabled ? "开启" : "关闭"} / 内容 ${aiReview?.content_enabled ? "开启" : "关闭"}`}
              active={Boolean(aiReview?.enabled)}
              loading={actingToggle === "ai"}
              onClick={() => void handleToggle("ai")}
            />
            <ToggleCard
              icon={Lock}
              label="访客访问限制"
              description={`笔记 ${guestAccess?.note_restricted ? "限制" : "开放"} / 视频 ${guestAccess?.video_restricted ? "限制" : "开放"}`}
              active={Boolean(guestAccess?.restricted)}
              loading={actingToggle === "guest"}
              onClick={() => void handleToggle("guest")}
            />
          </div>
        </Panel>
      </section>

      <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.75fr)]">
        <Panel title="热门内容" icon={FileText}>
          <HotContentList items={hot?.items ?? []} />
        </Panel>
        <Panel title="待处理事项" icon={ClipboardList}>
          <div className="grid gap-2">
            {Object.entries(pending).map(([key, value]) => (
              <div key={key} className="flex items-center justify-between rounded-lg bg-[#fafbfe] px-3 py-2 text-sm">
                <span className="text-[#555b66]">{pendingLabel(key)}</span>
                <span className="font-semibold text-[#17171d]">{formatCompact(value)}</span>
              </div>
            ))}
          </div>
        </Panel>
      </section>
    </div>
  );
}
