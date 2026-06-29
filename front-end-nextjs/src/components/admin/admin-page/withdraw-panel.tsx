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
  Check,
  CircleDollarSign,
  RefreshCw,
  Search,
  Wallet,
  X
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import {
  adminRequest,
  getAdminWithdrawOrders,
  updateAdminWithdrawOrder
} from "@/lib/api";
import { useTranslations } from "next-intl";
import type {
  AdminWithdrawOrderItem,
  AdminWithdrawOrdersPayload
} from "@/lib/types";
import {
  HeaderCard,
  Panel
} from "./layout-widgets";
import {
  EmptyBlock,
  LoadingBlock,
  PaginationBar
} from "./resource-editor";
import {
  Avatar,
  StatusPill,
  Thumbnail
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  errorMessage,
  formatDateTime,
  formatMoney,
  withdrawTone
} from "./helpers";
import {
  ChoiceSelect
} from "./form-fields";
import { cn } from "@/lib/utils";

export function WithdrawPanel({ token }: { token: string }) {
  const t = useTranslations("adminWithdraw");
  const balanceT = useTranslations("adminBalanceTransactions");
  const [payload, setPayload] = useState<AdminWithdrawOrdersPayload | null>(null);
  const [status, setStatus] = useState("");
  const [keyword, setKeyword] = useState("");
  const [keywordDraft, setKeywordDraft] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [actingId, setActingId] = useState<string | number | null>(null);
  const [balanceRows, setBalanceRows] = useState<ExternalBalanceTransaction[]>([]);
  const [balanceLoading, setBalanceLoading] = useState(true);
  const [balanceActingId, setBalanceActingId] = useState<number | null>(null);

  const loadBalanceTransactions = useCallback(async () => {
    setBalanceLoading(true);
    try {
      const data = await adminRequest<{ list: ExternalBalanceTransaction[] }>("/api/admin/balance-transactions", {
        method: "GET",
        token,
        query: { limit: 20 },
      });
      setBalanceRows(data.list ?? []);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBalanceLoading(false);
    }
  }, [token]);

  const load = useCallback(async (next: { page?: number; status?: string; keyword?: string } = {}) => {
    setLoading(true);
    try {
      const data = await getAdminWithdrawOrders({
        page: next.page ?? page,
        limit: 10,
        status: next.status ?? status,
        keyword: next.keyword ?? keyword,
      }, token);
      setPayload(data);
    } catch (error) {
      toast.error(errorMessage(error));
      setPayload(null);
    } finally {
      setLoading(false);
    }
  }, [keyword, page, status, token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  useEffect(() => {
    queueMicrotask(() => void loadBalanceTransactions());
  }, [loadBalanceTransactions]);

  async function compensateBalanceTransaction(row: ExternalBalanceTransaction) {
    if (!window.confirm(balanceT("confirmCompensation"))) return;
    setBalanceActingId(row.id);
    try {
      await adminRequest(`/api/admin/balance-transactions/${row.id}/compensate`, { method: "POST", token });
      toast.success(balanceT("compensated"));
      await loadBalanceTransactions();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBalanceActingId(null);
    }
  }

  async function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextKeyword = keywordDraft.trim();
    setKeyword(nextKeyword);
    setPage(1);
    await load({ page: 1, keyword: nextKeyword });
  }

  async function changeStatus(value: string) {
    setStatus(value);
    setPage(1);
    await load({ page: 1, status: value });
  }

  async function action(orderId: string | number, kind: "approve" | "reject" | "payout") {
    if (!window.confirm(t("confirmAction", { action: t(`actions.${kind}`) }))) return;
    setActingId(orderId);
    try {
      await updateAdminWithdrawOrder(orderId, kind, {}, token);
      toast.success(t("actionCompleted", { action: t(`actions.${kind}`) }));
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActingId(null);
    }
  }

  const rows = payload?.list ?? [];

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Wallet} title={t("title")} description={t("description")} tone="green" />
      <Panel title={t("orders")} icon={CircleDollarSign}>
        <div className="mb-4 grid gap-2 md:grid-cols-[minmax(240px,1fr)_220px]">
          <form onSubmit={submitSearch} className="flex min-w-0 gap-2">
            <input
              value={keywordDraft}
              onChange={(event) => setKeywordDraft(event.target.value)}
              className="h-10 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8] focus:ring-4 focus:ring-[#1d4ed8]/10"
              placeholder={t("searchPlaceholder")}
            />
            <Button type="submit" className="h-10 rounded-lg bg-[#17171d] px-4 hover:bg-[#2a2b32]">
              <Search className="size-4" />
              {t("search")}
            </Button>
          </form>
          <ChoiceSelect
            value={status}
            onChange={(value) => void changeStatus(value)}
            options={["pending", "approved", "rejected", "paid"].map((value) => ({ value, label: t(`statuses.${value}`) }))}
            placeholder={t("allStatuses")}
          />
        </div>
        {loading ? (
          <LoadingBlock label={t("loading")} />
        ) : rows.length ? (
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {rows.map((order) => (
              <article key={order.id} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-4">
                <div className="mb-3 flex items-start gap-3">
                  <Avatar src={order.avatar} label={order.nickname || order.user_uid || "-"} />
                  <div className="min-w-0 flex-1">
                    <h3 className="truncate text-sm font-semibold text-[#17171d]">{order.nickname || order.user_uid || t("userFallback", { id: order.user_id ?? "-" })}</h3>
                    <p className="text-xs text-[#8b919e]">{formatDateTime(order.created_at)}</p>
                  </div>
                  <StatusPill value={t(`statuses.${knownWithdrawStatus(order.status)}`)} tone={withdrawTone(order.status)} />
                </div>
                <div className="mb-4 grid grid-cols-2 gap-2 text-sm">
                  <InfoTile label={t("amount")} value={formatMoney(order.amount)} />
                  <InfoTile label={t("type")} value={order.type === "moon_coin" ? t("types.moonCoin") : t("types.cash")} />
                </div>
                <WithdrawPaymentCodeGrid order={order} />
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" disabled={actingId === order.id || order.status !== "pending"} onClick={() => void action(order.id, "approve")} className="rounded-lg bg-[#18a058] hover:bg-[#138a4a]">
                    <Check className="size-4" />
                    {t("actions.approve")}
                  </Button>
                  <Button type="button" size="sm" variant="outline" disabled={actingId === order.id || order.status !== "pending"} onClick={() => void action(order.id, "reject")} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]">
                    <X className="size-4" />
                    {t("actions.reject")}
                  </Button>
                  <Button type="button" size="sm" variant="outline" disabled={actingId === order.id || order.status !== "approved"} onClick={() => void action(order.id, "payout")} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">
                    <CircleDollarSign className="size-4" />
                    {t("actions.payout")}
                  </Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Wallet} label={t("empty")} />
        )}
        <PaginationBar page={payload?.pagination.page ?? page} total={payload?.pagination.total} hasNext={Boolean(payload?.pagination.hasNextPage)} disabled={loading} onPrev={() => { setPage((value) => Math.max(1, value - 1)); void load({ page: Math.max(1, page - 1) }); }} onNext={() => { setPage((value) => value + 1); void load({ page: page + 1 }); }} />
      </Panel>
      <Panel title={balanceT("title")} icon={RefreshCw}>
        <div className="mb-4 flex items-center justify-between gap-3">
          <p className="text-sm text-[#6f7683]">{balanceT("description")}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void loadBalanceTransactions()} disabled={balanceLoading}>
            <RefreshCw className={cn("size-4", balanceLoading && "animate-spin")} />
            {balanceT("refresh")}
          </Button>
        </div>
        {balanceLoading ? (
          <LoadingBlock label={balanceT("loading")} />
        ) : balanceRows.length ? (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[860px] text-left text-sm">
              <thead className="text-xs text-[#8b919e]"><tr><th className="p-2">{balanceT("operation")}</th><th className="p-2">{balanceT("oauth2Id")}</th><th className="p-2">{balanceT("amount")}</th><th className="p-2">{balanceT("status")}</th><th className="p-2">{balanceT("created")}</th><th className="p-2">{balanceT("action")}</th></tr></thead>
              <tbody>
                {balanceRows.map((row) => (
                  <tr key={row.id} className="border-t border-black/[0.06]">
                    <td className="max-w-[280px] truncate p-2 font-mono text-xs">{row.operation_key}</td>
                    <td className="p-2">{row.oauth2_id}</td>
                    <td className="p-2 font-semibold">{formatMoney(row.amount)}</td>
                    <td className="p-2"><StatusPill value={balanceT(`statuses.${knownBalanceStatus(row.status)}`)} tone={row.status === "unknown" ? "red" : row.status === "local_committed" ? "green" : "amber"} /></td>
                    <td className="p-2 text-xs text-[#8b919e]">{formatDateTime(row.created_at)}</td>
                    <td className="p-2">
                      <Button type="button" size="sm" variant="outline" disabled={row.status !== "applied" || balanceActingId === row.id} onClick={() => void compensateBalanceTransaction(row)}>
                        {balanceT("compensate")}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyBlock icon={Wallet} label={balanceT("empty")} />
        )}
      </Panel>
    </div>
  );
}

type ExternalBalanceTransaction = {
  id: number;
  operation_key: string;
  oauth2_id: number;
  amount: number;
  status: string;
  created_at: string;
};


export function WithdrawPaymentCodeGrid({ order }: { order: AdminWithdrawOrderItem }) {
  const t = useTranslations("adminWithdraw");
  const codes = [
    { label: t("wechatCode"), url: order.wechat_url },
    { label: t("alipayCode"), url: order.alipay_url },
  ].filter((item) => typeof item.url === "string" && item.url.trim());

  if (!codes.length) {
    return (
      <div className="mb-4 rounded-lg border border-dashed border-black/[0.08] bg-white px-3 py-2 text-xs font-medium text-[#8b919e]">
        {t("noPaymentCode")}
      </div>
    );
  }

  return (
    <div className="mb-4 grid gap-2 sm:grid-cols-2">
      {codes.map((item) => (
        <div key={item.label} className="flex min-w-0 items-center gap-2 rounded-lg border border-black/[0.06] bg-white p-2">
          <Thumbnail url={item.url} />
          <div className="min-w-0">
            <p className="text-xs font-semibold text-[#4f5562]">{item.label}</p>
            <p className="text-xs text-[#8b919e]">{t("previewHint")}</p>
          </div>
        </div>
      ))}
    </div>
  );
}

function knownWithdrawStatus(value: string) {
  return value === "pending" || value === "approved" || value === "rejected" || value === "paid"
    ? value
    : "unknown";
}

function knownBalanceStatus(value: string) {
  return value === "prepared" || value === "requesting" || value === "applied" || value === "local_committed"
    || value === "compensating" || value === "compensated" || value === "failed"
    ? value
    : "unknown";
}
