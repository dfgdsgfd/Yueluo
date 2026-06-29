"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  CheckCircle2,
  Copy,
  Gift,
  Link2,
  Loader2,
  RefreshCw,
  Sparkles,
  UserPlus,
  Users,
  Wallet,
} from "lucide-react";
import { toast } from "sonner";
import { getInviteStats, getStoredAccessToken } from "@/lib/api";
import type { InviteStatsPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

export function MobileInvitePage() {
  const router = useRouter();
  const [authToken, setAuthToken] = useState<string | null | undefined>(undefined);
  const [stats, setStats] = useState<InviteStatsPayload | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const rawInviteUrl = stats?.invite_url;
  const inviteUrl = useMemo(() => {
    if (!rawInviteUrl) {
      return "";
    }
    if (/^[a-z][a-z\d+\-.]*:/i.test(rawInviteUrl)) {
      return rawInviteUrl;
    }
    if (typeof window === "undefined") {
      return rawInviteUrl;
    }
    return new URL(rawInviteUrl, window.location.origin).toString();
  }, [rawInviteUrl]);

  const loadInvite = useCallback(async (options: { silent?: boolean } = {}) => {
    const token = getStoredAccessToken();
    setAuthToken(token);
    if (!token) {
      setIsLoading(false);
      return;
    }

    if (options.silent) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }

    try {
      setStats(await getInviteStats({ limit: 6 }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "邀请数据加载失败");
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadInvite();
    });
  }, [loadInvite]);

  const handleBack = useCallback(() => {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/");
  }, [router]);

  async function copyText(value: string, label: string) {
    if (!value) {
      toast.error(`${label}暂不可用`);
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      toast.success(`${label}已复制`);
    } catch {
      toast.error("复制失败，请手动复制");
    }
  }

  if (authToken === null) {
    return <InviteLoginRequired />;
  }

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#22222a]">
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
              邀请注册
            </h1>
            <button
              type="button"
              aria-label="刷新邀请数据"
              onClick={() => void loadInvite({ silent: true })}
              className="flex size-10 items-center justify-center justify-self-end rounded-full text-[#8b7adf] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <RefreshCw className={cn("size-5.5", isRefreshing && "animate-spin")} strokeWidth={2.4} />
            </button>
          </div>
        </header>

        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <Loader2 className="size-4 animate-spin" />
            正在加载邀请数据
          </div>
        ) : (
          <div className="space-y-4 px-4">
            <section className="relative overflow-hidden rounded-[8px] bg-[#7569df] px-4 py-5 text-white shadow-[0_16px_38px_rgba(110,88,210,0.24)]">
              <div className="absolute -right-8 -top-16 size-40 rounded-full bg-white/16" />
              <div className="absolute bottom-0 right-10 size-20 rounded-full bg-[#c9a7ff]/24" />
              <div className="relative z-10">
                <p className="flex items-center gap-1.5 text-[13px] font-bold text-white/76">
                  <Gift className="size-4" />
                  我的邀请码
                </p>
                <div className="mt-3 flex items-end justify-between gap-3">
                  <p className="min-w-0 truncate text-[34px] font-black leading-none tracking-[0.08em]">
                    {stats?.invite_code || "----"}
                  </p>
                  <span className={cn(
                    "shrink-0 rounded-full px-3 py-1 text-[12px] font-black",
                    stats?.is_active === false ? "bg-black/15 text-white/62" : "bg-white/20 text-white",
                  )}>
                    {stats?.is_active === false ? "已停用" : "可邀请"}
                  </span>
                </div>
                <div className="mt-5 grid grid-cols-3 gap-2 text-center">
                  <HeroStat label="点击" value={formatNumber(stats?.click_count)} />
                  <HeroStat label="注册" value={formatNumber(stats?.register_count)} />
                  <HeroStat label="收益" value={formatMoney(stats?.total_earnings)} />
                </div>
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">邀请链接</h2>
              <div className="mt-3 rounded-[8px] bg-[#fbfaff] px-3 py-3 text-[13px] font-bold leading-5 text-[#777381]">
                {inviteUrl || "邀请链接生成中"}
              </div>
              <div className="mt-3 grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={() => void copyText(stats?.invite_code ?? "", "邀请码")}
                  className="flex h-11 min-w-0 items-center justify-center gap-1.5 rounded-full bg-[#f3f0ff] px-3 text-[13px] font-black text-[#765eda] active:scale-[0.98]"
                >
                  <Copy className="size-4" />
                  复制邀请码
                </button>
                <button
                  type="button"
                  onClick={() => void copyText(inviteUrl, "邀请链接")}
                  className="flex h-11 min-w-0 items-center justify-center gap-1.5 rounded-full bg-[#8069dd] px-3 text-[13px] font-black text-white active:scale-[0.98]"
                >
                  <Link2 className="size-4" />
                  复制链接
                </button>
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">邀请明细</h2>
                <Users className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 grid grid-cols-3 gap-2">
                <MiniStat icon={UserPlus} label="已邀请" value={formatNumber(stats?.invitees_total)} />
                <MiniStat icon={Wallet} label="累计奖励" value={formatMoney(stats?.total_earnings)} />
                <MiniStat icon={Sparkles} label="有效状态" value={stats?.is_active === false ? "关闭" : "开启"} />
              </div>
              <div className="mt-3 divide-y divide-[#f0eef7]">
                {stats?.invitees?.length ? (
                  stats.invitees.map((item) => (
                    <article key={`${item.user_id}-${item.joined_at}`} className="flex min-w-0 items-center gap-3 py-3">
                      <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-[#f4efff] text-[14px] font-black text-[#765eda]">
                        {(item.nickname || item.user_id).charAt(0).toUpperCase()}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-[14px] font-black text-[#1d1d24]">{item.nickname || item.user_id}</p>
                        <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">{formatDate(item.joined_at)}</p>
                      </div>
                    </article>
                  ))
                ) : (
                  <div className="flex min-h-[112px] flex-col items-center justify-center gap-2 text-center text-[13px] font-bold text-[#9a96a5]">
                    <CheckCircle2 className="size-6 text-[#c9c5d3]" />
                    暂无受邀用户
                  </div>
                )}
              </div>
            </section>
          </div>
        )}
      </div>
    </main>
  );
}

function InviteLoginRequired() {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-[#fbfbff] px-4 text-[#22222a]">
      <section className="w-full max-w-[360px] rounded-[8px] bg-white px-5 py-6 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
        <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#f4efff] text-[#765eda]">
          <Gift className="size-6" />
        </div>
        <h1 className="mt-4 text-[18px] font-black">登录后邀请注册</h1>
        <p className="mt-2 text-sm font-bold leading-6 text-[#9a96a5]">登录后可生成专属邀请码并查看邀请收益。</p>
        <div className="mt-5 grid grid-cols-2 gap-2">
          <Link href="/login" className="flex h-10 items-center justify-center rounded-full bg-[#8069dd] text-sm font-black text-white">
            去登录
          </Link>
          <Link href="/" className="flex h-10 items-center justify-center rounded-full bg-[#f3f0ff] text-sm font-black text-[#765eda]">
            返回首页
          </Link>
        </div>
      </section>
    </main>
  );
}

function HeroStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 border-l border-white/18 first:border-l-0 first:pl-0">
      <p className="truncate text-[20px] font-black leading-none">{value}</p>
      <p className="mt-2 truncate text-[12px] font-bold text-white/78">{label}</p>
    </div>
  );
}

function MiniStat({ icon: Icon, label, value }: { icon: typeof UserPlus; label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-[8px] bg-[#fbfaff] px-2 py-3">
      <Icon className="size-4 text-[#8b7adf]" />
      <p className="mt-2 truncate text-[11px] font-bold text-[#777381]">{label}</p>
      <p className="mt-1 truncate text-[16px] font-black text-[#111116]">{value}</p>
    </div>
  );
}

function formatMoney(value?: number) {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;
  return amount.toLocaleString("zh-CN", {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  });
}

function formatNumber(value?: number) {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;
  return amount.toLocaleString("zh-CN");
}

function formatDate(value?: string) {
  if (!value) {
    return "刚刚加入";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("zh-CN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(date);
}
