"use client";
import type {
  FormEvent
} from "react";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  CircleDollarSign,
  Megaphone,
  Search,
  SlidersHorizontal,
  Users
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  adminRequest
} from "@/lib/api";
import type {
  AdminListPayload
} from "@/lib/types";
import {
  InviteAdminItem,
  InviteOverviewPayload,
  PickerSelection
} from "./types";
import {
  HeaderCard,
  MetricTile,
  Panel
} from "./layout-widgets";
import {
  EmptyBlock,
  LoadingBlock,
  PaginationBar
} from "./resource-editor";
import {
  AdminObjectPicker,
  firstPickerID
} from "./object-picker";
import {
  Avatar,
  StatusPill
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  errorMessage,
  fallbackPagination,
  formatCompact,
  formatDateTime,
  formatMoney,
  paginationHasNext,
  truthy
} from "./helpers";

export function InvitePanel({ token }: { token: string }) {
  const [overview, setOverview] = useState<InviteOverviewPayload | null>(null);
  const [payload, setPayload] = useState<AdminListPayload<InviteAdminItem> | null>(null);
  const [keyword, setKeyword] = useState("");
  const [keywordDraft, setKeywordDraft] = useState("");
  const [page, setPage] = useState(1);
  const [rewardUsers, setRewardUsers] = useState<PickerSelection[]>([]);
  const [rewardAmount, setRewardAmount] = useState("");
  const [rewardReason, setRewardReason] = useState("后台手动奖励");
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | number | null>(null);

  const load = useCallback(async (next: { page?: number; keyword?: string } = {}) => {
    setLoading(true);
    try {
      const nextPage = next.page ?? page;
      const nextKeyword = next.keyword ?? keyword;
      const [overviewData, listData] = await Promise.all([
        adminRequest<InviteOverviewPayload>("/api/invite/admin/overview", { method: "GET", token }),
        adminRequest<{ list?: InviteAdminItem[]; pagination?: AdminListPayload["pagination"] }>("/api/invite/admin/list", {
          method: "GET",
          token,
          query: { page: nextPage, limit: 12, keyword: nextKeyword },
        }),
      ]);
      setOverview(overviewData);
      setPayload({
        items: listData.list ?? [],
        pagination: listData.pagination ?? fallbackPagination(nextPage, 12, listData.list?.length ?? 0),
      });
    } catch (error) {
      toast.error(errorMessage(error));
      setPayload(null);
    } finally {
      setLoading(false);
    }
  }, [keyword, page, token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  async function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextKeyword = keywordDraft.trim();
    setKeyword(nextKeyword);
    setPage(1);
    await load({ page: 1, keyword: nextKeyword });
  }

  async function goInvitePage(nextPage: number) {
    if (nextPage < 1) return;
    setPage(nextPage);
    await load({ page: nextPage });
  }

  async function toggleInvite(row: InviteAdminItem) {
    if (row.id === undefined) return;
    setActing(row.id);
    try {
      await adminRequest(`/api/invite/admin/${encodeURIComponent(String(row.id))}/toggle`, { method: "PATCH", token });
      toast.success("邀请码状态已更新");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function grantInviteReward(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const userID = firstPickerID(rewardUsers);
    const amount = Number(rewardAmount);
    if (!Number.isFinite(userID) || userID <= 0 || !Number.isFinite(amount) || amount === 0) {
      toast.error("请选择用户并输入有效奖励金额");
      return;
    }
    setActing("reward");
    try {
      await adminRequest("/api/invite/admin/reward", {
        method: "POST",
        token,
        body: JSON.stringify({ user_id: userID, amount, type: "manual_reward", reason: rewardReason }),
      });
      setRewardAmount("");
      setRewardUsers([]);
      toast.success("邀请奖励已发放");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  const rows = payload?.items ?? [];

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Megaphone} title="邀请增长" description="邀请码、点击注册转化与手动奖励" tone="blue" />
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile label="邀请码" value={overview?.total_codes ?? 0} tone="purple" />
        <MetricTile label="总点击" value={overview?.total_clicks ?? 0} tone="blue" />
        <MetricTile label="总注册" value={overview?.total_registers ?? 0} tone="green" />
        <MetricTile label="累计奖励" value={formatMoney(overview?.total_earnings ?? 0)} tone="amber" />
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Panel title="邀请码列表" icon={Users}>
          <form onSubmit={submitSearch} className="mb-3 flex min-w-0 gap-2">
            <input value={keywordDraft} onChange={(event) => setKeywordDraft(event.target.value)} className="h-10 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="搜索邀请码、昵称、账号" />
            <Button type="submit" className="h-10 rounded-lg bg-[#17171d] px-4 hover:bg-[#2a2b32]"><Search className="size-4" />查询</Button>
          </form>
          {loading ? (
            <LoadingBlock label="正在加载邀请数据" />
          ) : rows.length ? (
            <div className="grid gap-2">
              {rows.map((row) => (
                <article key={String(row.id ?? row.code)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
                  <div className="mb-3 flex min-w-0 items-start gap-3">
                    <Avatar src={row.avatar} label={row.nickname || row.user_id || row.code || "-"} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="break-all text-sm font-semibold text-[#17171d]">{row.nickname || row.user_id || "未命名用户"}</h3>
                        <StatusPill value={truthy(row.is_active) ? "启用" : "停用"} tone={truthy(row.is_active) ? "green" : "slate"} />
                      </div>
                      <p className="mt-1 break-all text-xs text-[#8b919e]">邀请码 {row.code || "-"}</p>
                      <p className="mt-1 break-all text-xs text-[#8b919e]">账号 {row.user_id || "-"} · 邀请人 {row.invited_by_nickname || "-"}</p>
                    </div>
                    <Button type="button" variant="outline" size="sm" disabled={acting === row.id} onClick={() => void toggleInvite(row)} className="rounded-lg border-black/[0.08] bg-white">
                      <SlidersHorizontal className="size-4" />
                      启停
                    </Button>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-4">
                    <InfoTile label="点击" value={formatCompact(row.click_count ?? 0)} />
                    <InfoTile label="注册" value={formatCompact(row.register_count ?? 0)} />
                    <InfoTile label="收益" value={formatMoney(row.total_earnings ?? 0)} />
                    <InfoTile label="创建" value={formatDateTime(row.created_at)} />
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <EmptyBlock icon={Users} label="暂无邀请记录" />
          )}
          <PaginationBar
            page={payload?.pagination.page ?? page}
            total={payload?.pagination.total}
            hasNext={paginationHasNext(payload?.pagination)}
            disabled={loading}
            onPrev={() => void goInvitePage(page - 1)}
            onNext={() => void goInvitePage(page + 1)}
          />
        </Panel>
        <Panel title="手动奖励" icon={CircleDollarSign}>
          <form onSubmit={grantInviteReward} className="grid gap-3">
            <AdminObjectPicker
              token={token}
              resource="users"
              label="奖励对象"
              value={rewardUsers}
              onChange={setRewardUsers}
              placeholder="搜索昵称、账号或邮箱"
              emptyLabel="未找到用户"
            />
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">奖励金额</span>
              <input value={rewardAmount} onChange={(event) => setRewardAmount(event.target.value)} type="number" className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" />
            </label>
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">原因</span>
              <textarea value={rewardReason} onChange={(event) => setRewardReason(event.target.value)} className="min-h-[90px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 text-sm outline-none focus:border-[#1d4ed8]" />
            </label>
            <Button type="submit" disabled={acting === "reward"} className="h-10 rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]"><CircleDollarSign className="size-4" />发放奖励</Button>
          </form>
        </Panel>
      </section>
    </div>
  );
}
