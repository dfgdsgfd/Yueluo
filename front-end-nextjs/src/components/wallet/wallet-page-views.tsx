"use client";

import {
GiftCardRedemptionHistory,
GiftCardRedemptionSuccess,
GiftCardRow,
WalletEmptyState,
WalletInfoMini
} from "@/components/wallet/gift-card-redemptions";
import {
ApiError
} from "@/lib/api";
import type {
BalancePurchaseOrderItem,
BalanceRechargeConfigPayload,
BalanceRechargeOption,
PointsGiftCardProduct,
PointsGiftCardRedemption,
PointsLogItem,
PointsOverviewPayload,
PointsTaskConfig,
WithdrawOrderItem
} from "@/lib/types";
import { cn } from "@/lib/utils";
import {
ArrowLeft,
ChevronRight,
ClipboardList,
Coins,
CreditCard,
ExternalLink,
Gift,
Loader2,
ReceiptText,
RefreshCw,
ShoppingBag,
Wallet,
} from "lucide-react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import type { ReactNode } from "react";
import type { RechargeDisplayOption,WalletBalanceStatus,WalletError } from "./wallet-page-model";

export function WalletLoginRequired() {
  const t = useTranslations("walletPage");
  return (
    <main className="wallet-page theme-adaptive flex min-h-dvh items-center justify-center bg-[var(--wallet-bg)] px-4 text-[var(--wallet-text)]">
      <section className="w-full max-w-[360px] rounded-[8px] bg-[var(--wallet-card)] px-5 py-6 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
        <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#f4efff] text-[#765eda]">
          <Wallet className="size-6" />
        </div>
        <h1 className="mt-4 text-[18px] font-black">{t("login.title")}</h1>
        <p className="mt-2 text-sm font-bold leading-6 text-[var(--wallet-muted)]">{t("login.description")}</p>
        <div className="mt-5 grid grid-cols-2 gap-2">
          <Link href="/login" className="flex h-10 items-center justify-center rounded-full bg-[#8069dd] text-sm font-black text-white">
            {t("login.login")}
          </Link>
          <Link href="/" className="flex h-10 items-center justify-center rounded-full bg-[#f3f0ff] text-sm font-black text-[#765eda]">
            {t("actions.backHome")}
          </Link>
        </div>
      </section>
    </main>
  );
}

export function WalletDesktopView({
  accountName,
  balanceError,
  balanceStatus,
  bills,
  date,
  errors,
  isLoading,
  isRefreshing,
  moonCoinBalance,
  money,
  onBack,
  onRedeemGiftCard,
  onRefresh,
  pointLogRows,
  pointsBalance,
  pointsOverview,
  purchaseOrders,
  rechargeAmountRange,
  rechargeConfig,
  rechargeOptions,
  rechargeUrl,
  redeemingId,
  redeemMessage,
  latestRedemption,
  loadRedemptionsPage,
  pointsRedemptions,
  redemptionsError,
  redemptionsLoading,
}: {
  accountName: string;
  balanceError: string | null;
  balanceStatus: WalletBalanceStatus;
  bills: WithdrawOrderItem[];
  date: (value?: string | null) => string;
  errors: WalletError[];
  isLoading: boolean;
  isRefreshing: boolean;
  moonCoinBalance: number | null;
  money: (value?: number) => string;
  onBack: () => void;
  onRedeemGiftCard: (product: PointsGiftCardProduct) => void | Promise<void>;
  onRefresh: () => void;
  pointLogRows: PointsLogItem[];
  pointsBalance: number;
  pointsOverview: PointsOverviewPayload | null;
  purchaseOrders: BalancePurchaseOrderItem[];
  rechargeAmountRange: string | null;
  rechargeConfig: BalanceRechargeConfigPayload | null;
  rechargeOptions: RechargeDisplayOption[];
  rechargeUrl: string | null;
  redeemingId: string | number | null;
  redeemMessage: { tone: "success" | "error"; text: string } | null;
  latestRedemption: PointsGiftCardRedemption | null;
  loadRedemptionsPage: (page: number) => void | Promise<unknown>;
  pointsRedemptions: import("@/lib/types").PointsRedemptionsPayload | null;
  redemptionsError: string | null;
  redemptionsLoading: boolean;
}) {
  const t = useTranslations("walletPage");
  return (
    <div className="hidden min-h-dvh bg-[var(--wallet-bg)] lg:block">
      <header className="sticky top-0 z-30 border-b border-[var(--wallet-border)] bg-[var(--wallet-header)] backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-[1180px] items-center gap-3 px-6">
          <button type="button" onClick={onBack} className="inline-flex size-10 items-center justify-center rounded-lg text-[var(--wallet-text)] transition hover:bg-[var(--wallet-soft)]" aria-label={t("actions.back")}>
            <ArrowLeft className="size-5" />
          </button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-xl font-black tracking-normal text-[var(--wallet-text)]">{t("title")}</h1>
            <p className="mt-0.5 text-xs font-semibold text-[var(--wallet-muted)]">{t("subtitle")}</p>
          </div>
          <button type="button" onClick={onRefresh} className="inline-flex h-10 items-center gap-2 rounded-lg border border-[var(--wallet-border)] bg-[var(--wallet-card)] px-3 text-sm font-bold text-[var(--wallet-muted)] transition hover:bg-[var(--wallet-soft)]">
            <RefreshCw className={cn("size-4", isRefreshing && "animate-spin")} />
            {t("actions.refresh")}
          </button>
        </div>
      </header>

      <div className="mx-auto grid w-full max-w-[1180px] gap-5 px-6 py-6">
        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 rounded-lg bg-[var(--wallet-card)] text-sm font-bold text-[var(--wallet-muted)]">
            <Loader2 className="size-4 animate-spin" />
            {t("loading")}
          </div>
        ) : (
          <>
            {errors.length ? (
              <section className="rounded-lg border border-[#ffe1a8] bg-[#fff8e7] px-4 py-3 text-sm font-bold text-[#a96b00]">
                <p>{t("errors.partial")}</p>
                <ul className="mt-2 grid gap-1 font-semibold">
                  {errors.map((error) => (
                    <li key={error.label} className="break-words">{error.label}: {error.message}</li>
                  ))}
                </ul>
              </section>
            ) : null}

            <section className="grid gap-4 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.75fr)]">
              <div className="relative min-w-0 overflow-hidden rounded-lg bg-[#7065d8] p-6 text-white shadow-[0_18px_44px_rgba(95,86,190,0.22)]">
                <div className="absolute right-8 top-8 size-28 rounded-full bg-white/10" />
                <p className="flex items-center gap-2 text-sm font-bold text-white/70">
                  <Coins className="size-4" />
                  {t("balance.title")}
                </p>
                <div className="relative mt-6 flex flex-wrap items-end justify-between gap-5">
                  <div className="min-w-0">
                    {balanceStatus === "loading" ? (
                      <p className="flex items-center gap-3 text-3xl font-black leading-none">
                        <Loader2 className="size-6 animate-spin" />
                        {t("balance.loading")}
                      </p>
                    ) : balanceStatus === "error" ? (
                      <>
                        <p className="text-2xl font-black leading-tight">{t("balance.errorTitle")}</p>
                        <p className="mt-3 max-w-[620px] break-words text-sm font-semibold leading-6 text-white/78">{balanceError}</p>
                      </>
                    ) : (
                      <>
                        <p className="truncate text-5xl font-black leading-none">{money(moonCoinBalance ?? 0)} {t("balance.unit")}</p>
                        <p className="mt-3 truncate text-sm font-bold text-white/68">{t("balance.account", { account: accountName })}</p>
                      </>
                    )}
                  </div>
                  {rechargeUrl ? (
                    <a href={rechargeUrl} target="_blank" rel="noreferrer" className="inline-flex h-11 items-center gap-2 rounded-lg bg-white px-4 text-sm font-black text-[#5f54c8] transition hover:bg-[#f5f3ff]">
                      {t("actions.recharge")}
                      <ExternalLink className="size-4" />
                    </a>
                  ) : null}
                </div>
              </div>

              <div className="rounded-lg bg-[var(--wallet-card)] p-5 shadow-[0_12px_30px_rgba(31,41,55,0.06)]">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="text-lg font-black tracking-normal text-[var(--wallet-text)]">{t("points.title")}</h2>
                    <p className="mt-1 text-sm font-semibold text-[var(--wallet-muted)]">{t("points.todaySummary", { earned: money(pointsOverview?.today_earned ?? 0), cap: money(pointsOverview?.daily_cap ?? 0) })}</p>
                  </div>
                  <span className="inline-flex h-10 items-center gap-2 rounded-lg bg-[#eff6ff] px-3 text-sm font-black text-[#2563eb]">
                    <Coins className="size-4" />
                    {money(pointsBalance)}
                  </span>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-2">
                  <WalletInfoMini label={t("points.available")} value={money(pointsBalance)} />
                  <WalletInfoMini label={t("points.todayEarned")} value={money(pointsOverview?.today_earned ?? 0)} />
                  <WalletInfoMini label={t("points.dailyCap")} value={money(pointsOverview?.daily_cap ?? 0)} />
                </div>
                {pointsOverview?.active_bonus ? (
                  <div className="mt-3 rounded-lg bg-[#fff7e8] px-3 py-2 text-xs font-bold text-[#a96b00]">
                    {t("points.withdrawBonus", { percent: money(pointsOverview.active_bonus.bonus_percent) })}
                    {pointsOverview.active_bonus.expires_at ? ` · ${t("points.expiresAt", { date: date(pointsOverview.active_bonus.expires_at) })}` : ""}
                  </div>
                ) : null}
              </div>
            </section>

            <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.7fr)]">
              <DesktopPanel title={t("tasks.title")} icon={ClipboardList}>
                <div className="-mx-1 flex snap-x gap-3 overflow-x-auto px-1 pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
                  {pointsOverview?.tasks?.length ? (
                    pointsOverview.tasks.map((task) => <PointsTaskCard key={task.task_type} money={money} task={task} t={t} />)
                  ) : (
                    <DesktopEmpty text={t("tasks.empty")} />
                  )}
                </div>
              </DesktopPanel>

              <DesktopPanel title={t("giftCards.title")} icon={Gift}>
                {redeemMessage ? (
                  <div className={cn(
                    "mb-3 rounded-lg px-3 py-2 text-xs font-bold",
                    redeemMessage.tone === "success" ? "bg-[#ecfdf3] text-[#16824a]" : "bg-[#fff1f1] text-[#c83232]",
                  )}>
                    {redeemMessage.text}
                  </div>
                ) : null}
                {latestRedemption ? <GiftCardRedemptionSuccess redemption={latestRedemption} /> : null}
                <div className="grid gap-2">
                  {pointsOverview?.gift_cards?.length ? (
                    pointsOverview.gift_cards.map((card) => (
                      <GiftCardRow
                        key={String(card.id)}
                        card={card}
                        loading={redeemingId === card.id}
                        money={money}
                        points={pointsBalance}
                        onRedeem={() => void onRedeemGiftCard(card)}
                      />
                    ))
                  ) : (
                    <WalletEmptyState text={t("giftCards.empty")} />
                  )}
                </div>
              </DesktopPanel>
            </section>

            <GiftCardRedemptionHistory
              error={redemptionsError}
              loading={redemptionsLoading}
              onPage={loadRedemptionsPage}
              payload={pointsRedemptions}
              variant="desktop"
            />

            <section className="grid gap-5 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
              <DesktopPanel title={t("recharge.title")} icon={CreditCard}>
                <div className="flex items-center justify-between gap-3">
                  <p className="truncate text-sm font-bold text-[var(--wallet-muted)]">{rechargeAmountRange ?? (rechargeConfig?.custom_amount_enable ? t("recharge.customAmount") : t("recharge.fixedOptions"))}</p>
                  {rechargeUrl ? (
                    <a href={rechargeUrl} target="_blank" rel="noreferrer" className="inline-flex h-9 shrink-0 items-center gap-1 rounded-lg bg-[#f3f0ff] px-3 text-xs font-black text-[#765eda]">
                      {t("actions.open")}
                      <ExternalLink className="size-3.5" />
                    </a>
                  ) : null}
                </div>
                <RechargeOptionGrid empty={t("recharge.empty")} href={rechargeUrl} money={money} options={rechargeOptions} t={t} className="grid-cols-3" />
              </DesktopPanel>

              <DesktopPanel title={t("pointsLogs.title")} icon={ReceiptText}>
                <div className="divide-y divide-[#edf0f5]">
                  {pointLogRows.length ? (
                    pointLogRows.map((item) => <PointsLogRow key={String(item.id)} date={date} item={item} money={money} t={t} />)
                  ) : (
                    <WalletEmptyState text={t("pointsLogs.empty")} />
                  )}
                </div>
              </DesktopPanel>
            </section>

            <section className="grid gap-5 xl:grid-cols-2">
              <DesktopPanel title={t("bills.title")} icon={ReceiptText}>
                <div className="divide-y divide-[#edf0f5]">
                  {bills.length ? bills.map((bill) => <BillItem key={bill.id} date={date} item={bill} money={money} t={t} />) : <WalletEmptyState text={t("bills.empty")} />}
                </div>
              </DesktopPanel>
              <DesktopPanel title={t("purchases.title")} icon={ShoppingBag}>
                <div className="divide-y divide-[#edf0f5]">
                  {purchaseOrders.length ? purchaseOrders.map((order) => <PurchaseOrderItem key={order.id} date={date} item={order} money={money} t={t} />) : <WalletEmptyState text={t("purchases.empty")} />}
                </div>
              </DesktopPanel>
            </section>
          </>
        )}
      </div>
    </div>
  );
}

export function DesktopPanel({
  children,
  icon: Icon,
  title,
}: {
  children: ReactNode;
  icon: typeof ClipboardList;
  title: string;
}) {
  return (
    <section className="min-w-0 rounded-lg bg-[var(--wallet-card)] p-5 shadow-[0_12px_30px_rgba(31,41,55,0.06)]">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-lg font-black tracking-normal text-[var(--wallet-text)]">{title}</h2>
        <Icon className="size-5 text-[var(--wallet-faint)]" />
      </div>
      {children}
    </section>
  );
}

export function DesktopEmpty({ text }: { text: string }) {
  return (
    <div className="flex min-h-[150px] w-full items-center justify-center rounded-lg bg-[var(--wallet-soft)] text-sm font-bold text-[var(--wallet-muted)]">
      {text}
    </div>
  );
}

export function RechargeOptionGrid({
  className,
  empty,
  href,
  icon: Icon = CreditCard,
  money,
  options,
  t,
}: {
  className?: string;
  empty: string;
  href: string | null;
  icon?: typeof CreditCard;
  money: (value?: number) => string;
  options: RechargeDisplayOption[];
  t: ReturnType<typeof useTranslations<"walletPage">>;
}) {
  return (
    <div className={cn("mt-4 grid grid-cols-2 gap-2", className)}>
      {options.length ? (
        options.slice(0, 6).map((option) => {
          const content = (
            <>
              <span className="flex size-9 shrink-0 items-center justify-center rounded-[10px] bg-[#f3f0ff] text-[#765eda]">
                <Icon className="size-4.5" />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-[14px] font-black text-[var(--wallet-text)]">{t("money.moonCoinAmount", { amount: money(option.amount) })}</span>
                <span className="mt-1 block truncate text-[11px] font-bold text-[#9a96a5]">{option.detail}</span>
              </span>
            </>
          );

          return href ? (
            <a
              key={option.key}
              href={href}
              target="_blank"
              rel="noreferrer"
              className="flex h-[68px] min-w-0 items-center gap-2 rounded-[8px] bg-[#fbfaff] px-3 active:scale-[0.98]"
            >
              {content}
            </a>
          ) : (
            <div key={option.key} className="flex h-[68px] min-w-0 items-center gap-2 rounded-[8px] bg-[#fbfaff] px-3">
              {content}
            </div>
          );
        })
      ) : (
        <div className="col-span-2 flex h-[68px] items-center justify-center rounded-[8px] bg-[#fbfaff] text-[12px] font-bold text-[#9a96a5]">
          {empty}
        </div>
      )}
    </div>
  );
}

export function BillItem({
  date,
  item,
  money,
  t,
}: {
  date: (value?: string | null) => string;
  item: WithdrawOrderItem;
  money: (value?: number) => string;
  t: ReturnType<typeof useTranslations<"walletPage">>;
}) {
  return (
    <article className="grid grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-3 py-3">
      <span className="flex size-11 items-center justify-center rounded-[10px] bg-[#f4efff] text-[#765eda]">
        <ReceiptText className="size-5" />
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-[14px] font-black text-[var(--wallet-text)]">{withdrawTypeLabel(item.type, t)}</h3>
        <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">
          {date(item.created_at)}
          {item.remark ? ` · ${item.remark}` : ""}
        </p>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="text-[13px] font-black text-[#42b389]">¥{money(item.amount)}</span>
        <ChevronRight className="size-4 text-[#c9c5d3]" />
      </div>
    </article>
  );
}

export function PurchaseOrderItem({
  date,
  item,
  money,
  t,
}: {
  date: (value?: string | null) => string;
  item: BalancePurchaseOrderItem;
  money: (value?: number) => string;
  t: ReturnType<typeof useTranslations<"walletPage">>;
}) {
  return (
    <article className="grid grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-3 py-3">
      <span className="flex size-11 items-center justify-center rounded-[10px] bg-[#eaf2ff] text-[#5688e8]">
        <ShoppingBag className="size-5" />
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-[14px] font-black text-[var(--wallet-text)]">{item.post_title || t("purchases.fallbackTitle", { id: String(item.post_id ?? item.id) })}</h3>
        <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">
          {item.purchase_type || t("purchases.defaultType")}
          {item.purchased_at ? ` · ${date(item.purchased_at)}` : ""}
        </p>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="text-[13px] font-black text-[#f0a000]">{t("money.moonCoinAmount", { amount: money(item.paid_amount) })}</span>
        <ChevronRight className="size-4 text-[#c9c5d3]" />
      </div>
    </article>
  );
}

export function PointsTaskCard({
  money,
  task,
  t,
}: {
  money: (value?: number) => string;
  task: PointsTaskConfig;
  t: ReturnType<typeof useTranslations<"walletPage">>;
}) {
  const completed = task.completed ?? 0;
  const limit = task.daily_limit ?? 0;
  const progress = limit > 0 ? `${completed}/${limit}` : String(completed);
  const done = Boolean(task.reached_limit);
  return (
    <article className={cn(
      "flex min-h-[150px] w-[min(74vw,245px)] shrink-0 snap-start flex-col justify-between rounded-[8px] px-3 py-3",
      done ? "bg-[#f3f4f7]" : "bg-[#fbfaff]",
    )}>
      <div className="flex items-start justify-between gap-2">
        <span className={cn(
          "flex size-10 shrink-0 items-center justify-center rounded-[10px]",
          done ? "bg-white text-[#8a8f9d]" : "bg-[#f0f7ff] text-[#3479c8]",
        )}>
          <ClipboardList className="size-4.5" />
        </span>
        <span className={cn(
          "shrink-0 rounded-full px-2.5 py-1 text-[11px] font-black",
          done ? "bg-white text-[#16824a]" : "bg-[#ecfdf3] text-[#16824a]",
        )}>
          {done ? t("tasks.done") : t("tasks.inProgress")}
        </span>
      </div>
      <div className="min-w-0">
        <h3 className="line-clamp-2 text-[15px] font-black leading-5 text-[var(--wallet-text)]">{task.name}</h3>
        <p className="mt-2 line-clamp-2 min-h-8 text-[11px] font-bold leading-4 text-[#9a96a5]">
          {task.description || (task.is_daily_task ? t("tasks.daily") : t("tasks.once"))}
        </p>
      </div>
      <div className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-bold text-[#9a96a5]">{t("tasks.progress")}</p>
          <p className="mt-0.5 text-[16px] font-black text-[var(--wallet-text)]">{progress}</p>
        </div>
        <span className={cn(
          "rounded-full px-2.5 py-1 text-[12px] font-black",
          done ? "bg-white text-[#8a8f9d]" : "bg-[#fff3e2] text-[#d98218]",
        )}>
          {t("money.pointsPlus", { amount: money(task.points) })}
        </span>
      </div>
    </article>
  );
}

export function PointsLogRow({
  date,
  item,
  money,
  t,
}: {
  date: (value?: string | null) => string;
  item: PointsLogItem;
  money: (value?: number) => string;
  t: ReturnType<typeof useTranslations<"walletPage">>;
}) {
  const positive = item.amount >= 0;
  return (
    <article className="grid grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-3 py-3">
      <span className={cn("flex size-11 items-center justify-center rounded-[10px]", positive ? "bg-[#ecfdf3] text-[#16824a]" : "bg-[#fff3e2] text-[#d98218]")}>
        <Coins className="size-5" />
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-[14px] font-black text-[var(--wallet-text)]">{item.reason || pointLogTypeLabel(item.type, t)}</h3>
        <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">{date(item.created_at)} · {t("pointsLogs.balanceAfter", { points: money(item.balance_after) })}</p>
      </div>
      <span className={cn("text-[13px] font-black", positive ? "text-[#16824a]" : "text-[#d98218]")}>
        {positive ? "+" : ""}{money(item.amount)}
      </span>
    </article>
  );
}

export function normalizeRechargeOptions(
  options: BalanceRechargeOption[] | undefined,
  t: ReturnType<typeof useTranslations<"walletPage">>,
  locale: string,
) {
  if (!Array.isArray(options)) {
    return [];
  }

  return options
    .filter((option) => Number.isFinite(option.amount))
    .map((option) => ({
      amount: option.amount,
      detail:
        typeof option.bonus === "number" && Number.isFinite(option.bonus) && option.bonus > 0
          ? t("recharge.bonus", { amount: formatMoney(option.bonus, locale) })
          : t("recharge.noBonus"),
      key: `recharge-${option.amount}-${option.bonus ?? 0}`,
    }))
    .sort((left, right) => left.amount - right.amount);
}

export function formatRechargeAmountRange(
  min: number | undefined,
  max: number | undefined,
  t: ReturnType<typeof useTranslations<"walletPage">>,
  locale: string,
) {
  const hasMin = typeof min === "number" && Number.isFinite(min);
  const hasMax = typeof max === "number" && Number.isFinite(max);

  if (hasMin && hasMax) {
    return t("recharge.range", { min: formatMoney(min, locale), max: formatMoney(max, locale) });
  }
  if (hasMin) {
    return t("recharge.min", { amount: formatMoney(min, locale) });
  }
  if (hasMax) {
    return t("recharge.max", { amount: formatMoney(max, locale) });
  }
  return null;
}

export function withdrawTypeLabel(value: string, t: ReturnType<typeof useTranslations<"walletPage">>) {
  if (value === "moon_coin") {
    return t("bills.types.moonCoin");
  }
  if (value === "cash") {
    return t("bills.types.cash");
  }
  return value || t("bills.defaultType");
}

export function pointLogTypeLabel(value: string, t: ReturnType<typeof useTranslations<"walletPage">>) {
  if (value === "gift_card_redeem") {
    return t("pointsLogs.types.giftCardRedeem");
  }
  if (value === "achievement") {
    return t("pointsLogs.types.achievement");
  }
  if (value.startsWith("task_")) {
    return t("pointsLogs.types.task");
  }
  return value || t("pointsLogs.defaultType");
}

export function detailedWalletError(error: unknown, t: ReturnType<typeof useTranslations>) {
  if (!(error instanceof ApiError)) {
    return error instanceof Error && error.message ? error.message : t("errors.loadFailed");
  }
  const details = error.details && typeof error.details === "object"
    ? error.details as Record<string, unknown>
    : {};
  const requestID = String(details.requestId ?? details.request_id ?? "").trim();
  const parts = [
    error.message === "error.balance_center_not_configured"
      ? t("errors.balanceCenterNotConfigured")
      : error.message || t("errors.loadFailed"),
    error.status ? t("errors.httpStatus", { status: error.status }) : "",
    error.code ? t("errors.businessCode", { code: error.code }) : "",
    requestID ? t("errors.requestId", { requestId: requestID }) : "",
  ].filter(Boolean);
  return parts.join(" · ");
}

export function formatMoney(value?: number, locale = "zh-CN") {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;
  return amount.toLocaleString(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: amount % 1 === 0 ? 0 : 2,
  });
}

export function formatDate(value?: string | null, locale = "zh-CN", fallback = "Just now") {
  if (!value) {
    return fallback;
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
