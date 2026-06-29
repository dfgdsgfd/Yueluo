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
  Edit3,
  Eye,
  Megaphone,
  Plus,
  Search,
  Tags,
  Trash2
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
  AdminListPayload,
  AdminListRow
} from "@/lib/types";
import {
  CouponAdminItem,
  CouponStatsPayload,
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
  CouponEditorDrawer,
  CouponIssueDrawer,
  CouponUsageDrawer
} from "./coupon-drawers";
import {
  pickerIDs
} from "./object-picker";
import {
  StatusPill
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  couponDefaultDraft,
  couponDraftPayload,
  couponValueLabel,
  errorMessage,
  fallbackPagination,
  formatCompact,
  formatDateTime,
  formatMoney,
  paginationHasNext,
  truthy
} from "./helpers";

export function CouponPanel({ token }: { token: string }) {
  const [stats, setStats] = useState<CouponStatsPayload | null>(null);
  const [payload, setPayload] = useState<AdminListPayload<CouponAdminItem> | null>(null);
  const [keyword, setKeyword] = useState("");
  const [keywordDraft, setKeywordDraft] = useState("");
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState<CouponAdminItem | null>(null);
  const [draft, setDraft] = useState<Record<string, unknown>>(() => couponDefaultDraft(null));
  const [issueCoupon, setIssueCoupon] = useState<CouponAdminItem | null>(null);
  const [issueTarget, setIssueTarget] = useState("users");
  const [issueUsers, setIssueUsers] = useState<PickerSelection[]>([]);
  const [usageCoupon, setUsageCoupon] = useState<CouponAdminItem | null>(null);
  const [usagePayload, setUsagePayload] = useState<AdminListPayload<AdminListRow> | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [acting, setActing] = useState<string | number | null>(null);

  const load = useCallback(async (next: { page?: number; keyword?: string } = {}) => {
    setLoading(true);
    try {
      const nextPage = next.page ?? page;
      const nextKeyword = next.keyword ?? keyword;
      const [statsData, listData] = await Promise.all([
        adminRequest<CouponStatsPayload>("/api/coupon/admin/stats", { method: "GET", token }),
        adminRequest<{ list?: CouponAdminItem[]; pagination?: AdminListPayload["pagination"] }>("/api/coupon/admin/list", {
          method: "GET",
          token,
          query: { page: nextPage, limit: 12, keyword: nextKeyword },
        }),
      ]);
      setStats(statsData);
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

  async function goCouponPage(nextPage: number) {
    if (nextPage < 1) return;
    setPage(nextPage);
    await load({ page: nextPage });
  }

  function openCouponEditor(row: CouponAdminItem | null) {
    setEditing(row ?? {});
    setDraft(couponDefaultDraft(row));
  }

  async function saveCoupon(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      const body = couponDraftPayload(draft);
      if (editing?.id) {
        await adminRequest(`/api/coupon/admin/${encodeURIComponent(String(editing.id))}`, { method: "PUT", token, body: JSON.stringify(body) });
        toast.success("优惠券已更新");
      } else {
        await adminRequest("/api/coupon/admin/create", { method: "POST", token, body: JSON.stringify(body) });
        toast.success("优惠券已创建");
      }
      setEditing(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteCoupon(row: CouponAdminItem) {
    if (!row.id || !window.confirm(`确认删除优惠券 ${row.name ?? row.id}？`)) return;
    setActing(row.id);
    try {
      await adminRequest(`/api/coupon/admin/${encodeURIComponent(String(row.id))}`, { method: "DELETE", token });
      toast.success("优惠券已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function issueCouponToUsers(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!issueCoupon?.id) return;
    setActing("issue");
    try {
      const ids = pickerIDs(issueUsers);
      if (issueTarget === "users" && !ids.length) {
        toast.error("请选择要发放的用户");
        return;
      }
      const body: Record<string, unknown> = { target_type: issueTarget };
      if (issueTarget === "users") body.user_ids = ids;
      await adminRequest(`/api/coupon/admin/${encodeURIComponent(String(issueCoupon.id))}/issue`, { method: "POST", token, body: JSON.stringify(body) });
      setIssueCoupon(null);
      setIssueUsers([]);
      toast.success("优惠券已发放");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function openCouponUsages(row: CouponAdminItem) {
    if (!row.id) return;
    setUsageCoupon(row);
    setActing(`usage-${row.id}`);
    try {
      const data = await adminRequest<{ list?: AdminListRow[]; pagination?: AdminListPayload["pagination"] }>(`/api/coupon/admin/${encodeURIComponent(String(row.id))}/usages`, {
        method: "GET",
        token,
        query: { page: 1, limit: 20 },
      });
      setUsagePayload({ items: data.list ?? [], pagination: data.pagination ?? fallbackPagination(1, 20, data.list?.length ?? 0) });
    } catch (error) {
      toast.error(errorMessage(error));
      setUsagePayload(null);
    } finally {
      setActing(null);
    }
  }

  const rows = payload?.items ?? [];

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Tags} title="优惠券运营" description="优惠券创建、编辑、发放与领取使用记录" tone="blue" />
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile label="总优惠券" value={stats?.totalCoupons ?? 0} tone="red" />
        <MetricTile label="启用中" value={stats?.activeCoupons ?? 0} tone="green" />
        <MetricTile label="已发放" value={stats?.totalIssued ?? 0} tone="blue" />
        <MetricTile label="已使用" value={stats?.totalUsed ?? 0} tone="amber" />
      </section>

      <Panel title="优惠券列表" icon={Tags} action={<Button type="button" onClick={() => openCouponEditor(null)} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><Plus className="size-4" />新建</Button>}>
        <form onSubmit={submitSearch} className="mb-3 flex min-w-0 gap-2">
          <input value={keywordDraft} onChange={(event) => setKeywordDraft(event.target.value)} className="h-10 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" placeholder="搜索优惠券名称、券码" />
          <Button type="submit" className="h-10 rounded-lg bg-[#17171d] px-4 hover:bg-[#2a2b32]"><Search className="size-4" />查询</Button>
        </form>
        {loading ? (
          <LoadingBlock label="正在加载优惠券" />
        ) : rows.length ? (
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {rows.map((row) => (
              <article key={String(row.id)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-4">
                <div className="mb-3 flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="line-clamp-2 break-all text-sm font-semibold text-[#17171d]">{row.name || `优惠券 ${row.id}`}</h3>
                    <p className="mt-1 break-all text-xs text-[#8b919e]">券码 {row.code || "自动生成"}</p>
                  </div>
                  <StatusPill value={truthy(row.is_active) ? "启用" : "停用"} tone={truthy(row.is_active) ? "green" : "slate"} />
                </div>
                <div className="mb-3 grid grid-cols-2 gap-2">
                  <InfoTile label="优惠" value={couponValueLabel(row)} />
                  <InfoTile label="门槛" value={formatMoney(row.min_order ?? 0)} />
                  <InfoTile label="已发放" value={formatCompact(row.issued_count ?? 0)} />
                  <InfoTile label="已使用" value={formatCompact(row.used_count ?? 0)} />
                </div>
                <p className="mb-3 line-clamp-2 min-h-10 text-xs leading-5 text-[#6f7582]">{row.description || "无描述"}</p>
                <div className="mb-3 rounded-lg bg-white px-3 py-2 text-xs text-[#6f7582]">
                  {formatDateTime(row.start_time)} - {formatDateTime(row.end_time)}
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" variant="outline" onClick={() => openCouponEditor(row)} className="rounded-lg border-black/[0.08] bg-white"><Edit3 className="size-4" />编辑</Button>
                  <Button type="button" size="sm" variant="outline" onClick={() => setIssueCoupon(row)} className="rounded-lg border-black/[0.08] bg-white"><Megaphone className="size-4" />发放</Button>
                  <Button type="button" size="sm" variant="outline" disabled={acting === `usage-${row.id}`} onClick={() => void openCouponUsages(row)} className="rounded-lg border-black/[0.08] bg-white"><Eye className="size-4" />记录</Button>
                  <Button type="button" size="sm" variant="outline" disabled={acting === row.id} onClick={() => void deleteCoupon(row)} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]"><Trash2 className="size-4" />删除</Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Tags} label="暂无优惠券" />
        )}
        <PaginationBar
          page={payload?.pagination.page ?? page}
          total={payload?.pagination.total}
          hasNext={paginationHasNext(payload?.pagination)}
          disabled={loading}
          onPrev={() => void goCouponPage(page - 1)}
          onNext={() => void goCouponPage(page + 1)}
        />
      </Panel>

      <CouponEditorDrawer
        row={editing}
        draft={draft}
        saving={saving}
        onDraftChange={(key, value) => setDraft((current) => ({ ...current, [key]: value }))}
        onClose={() => setEditing(null)}
        onSubmit={(event) => void saveCoupon(event)}
      />
      <CouponIssueDrawer
        token={token}
        coupon={issueCoupon}
        target={issueTarget}
        users={issueUsers}
        acting={acting === "issue"}
        onTargetChange={setIssueTarget}
        onUsersChange={setIssueUsers}
        onClose={() => setIssueCoupon(null)}
        onSubmit={(event) => void issueCouponToUsers(event)}
      />
      <CouponUsageDrawer coupon={usageCoupon} payload={usagePayload} onClose={() => setUsageCoupon(null)} />
    </div>
  );
}
