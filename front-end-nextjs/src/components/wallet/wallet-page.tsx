"use client";

import {
GiftCardRedemptionHistory,
GiftCardRedemptionSuccess,
GiftCardRow,
WalletEmptyState,
WalletInfoMini,
useGiftCardRedemptionHistory,
} from "@/components/wallet/gift-card-redemptions";
import {
getBalanceConfig,
getBalanceLocalPoints,
getBalanceOrders,
getBalanceRechargeConfig,
getBalanceUserBalance,
getPointsLogs,
getPointsOverview,
getStoredAccessToken,
getWithdrawOrders,
redeemPointsGiftCard
} from "@/lib/api";
import type { WalletInitialData } from "@/lib/server/wallet-page-data";
import type {
BalanceConfigPayload,
BalanceLocalPointsPayload,
BalanceOrdersPayload,
BalanceRechargeConfigPayload,
BalanceUserBalancePayload,
PointsGiftCardProduct,
PointsGiftCardRedemption,
PointsLogsPayload,
PointsOverviewPayload,
WithdrawOrdersPayload
} from "@/lib/types";
import { cn } from "@/lib/utils";
import {
ArrowLeft,
ClipboardList,
Coins,
ExternalLink,
Gift,
Loader2,
ReceiptText,
RefreshCw,
ShoppingBag
} from "lucide-react";
import { useLocale,useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { useCallback,useEffect,useMemo,useState } from "react";
import { hiddenPointLogTypes,loadLabelKeys,type WalletBalanceStatus,type WalletError } from "./wallet-page-model";
import { BillItem,PointsLogRow,PointsTaskCard,PurchaseOrderItem,RechargeOptionGrid,WalletDesktopView,WalletLoginRequired,detailedWalletError,formatDate,formatMoney,formatRechargeAmountRange,normalizeRechargeOptions } from "./wallet-page-views";

export function WalletPage({
  initialData,
}: {
  initialData?: WalletInitialData | null;
} = {}) {
  const t = useTranslations("walletPage");
  const locale = useLocale();
  const router = useRouter();
  const [authToken, setAuthToken] = useState<string | null | undefined>(
    () => initialData ? (initialData.authenticated ? "server-authenticated" : null) : undefined,
  );
  const [balanceConfig, setBalanceConfig] = useState<BalanceConfigPayload | null>(
    () => initialData?.balanceConfig ?? null,
  );
  const [rechargeConfig, setRechargeConfig] = useState<BalanceRechargeConfigPayload | null>(
    () => initialData?.rechargeConfig ?? null,
  );
  const [localPoints, setLocalPoints] = useState<BalanceLocalPointsPayload | null>(
    () => initialData?.localPoints ?? null,
  );
  const [userBalance, setUserBalance] = useState<BalanceUserBalancePayload | null>(
    () => initialData?.userBalance ?? null,
  );
  const [balanceStatus, setBalanceStatus] = useState<WalletBalanceStatus>(() => {
    if (!initialData?.authenticated) return "idle";
    return initialData.userBalance ? "ready" : "loading";
  });
  const [balanceError, setBalanceError] = useState<string | null>(null);
  const [withdrawOrders, setWithdrawOrders] = useState<WithdrawOrdersPayload | null>(
    () => initialData?.withdrawOrders ?? null,
  );
  const [balanceOrders, setBalanceOrders] = useState<BalanceOrdersPayload | null>(
    () => initialData?.balanceOrders ?? null,
  );
  const [pointsOverview, setPointsOverview] = useState<PointsOverviewPayload | null>(
    () => initialData?.pointsOverview ?? null,
  );
  const [pointsLogs, setPointsLogs] = useState<PointsLogsPayload | null>(
    () => initialData?.pointsLogs ?? null,
  );
  const [errors, setErrors] = useState<WalletError[]>([]);
  const [isLoading, setIsLoading] = useState(() => !initialData);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [redeemingId, setRedeemingId] = useState<string | number | null>(null);
  const [redeemMessage, setRedeemMessage] = useState<{ tone: "success" | "error"; text: string } | null>(null);
  const [latestRedemption, setLatestRedemption] = useState<PointsGiftCardRedemption | null>(null);
  const {
    error: redemptionsError,
    loadPage: loadRedemptionsPage,
    loading: redemptionsLoading,
    payload: pointsRedemptions,
    refresh: refreshRedemptions,
  } = useGiftCardRedemptionHistory(initialData?.pointsRedemptions ?? null);

  const rechargeUrl = useMemo(() => {
    const value = rechargeConfig?.recharge_url;
    return typeof value === "string" && value.trim() ? value : null;
  }, [rechargeConfig]);
  const rechargeOptions = useMemo(
    () => normalizeRechargeOptions(rechargeConfig?.options, t, locale),
    [locale, rechargeConfig?.options, t],
  );
  const rechargeAmountRange = useMemo(
    () => formatRechargeAmountRange(rechargeConfig?.min_amount, rechargeConfig?.max_amount, t, locale),
    [locale, rechargeConfig?.max_amount, rechargeConfig?.min_amount, t],
  );
  const money = useCallback((value?: number) => formatMoney(value, locale), [locale]);
  const date = useCallback((value?: string | null) => formatDate(value, locale, t("time.justNow")), [locale, t]);

  const moonCoinBalance = userBalance?.balance ?? null;
  const accountName = userBalance?.username?.trim() || t("balance.defaultAccount");
  const purchaseOrders = balanceOrders?.list ?? [];
  const bills = withdrawOrders?.list ?? [];
  const pointsBalance = pointsOverview?.points ?? localPoints?.points ?? 0;
  const pointLogRows = (pointsLogs?.list ?? []).filter((item) => !hiddenPointLogTypes.has(item.type));

  const loadWallet = useCallback(async (options: { silent?: boolean } = {}) => {
    const token = getStoredAccessToken();
    setAuthToken(token);

    if (!token) {
      setBalanceStatus("idle");
      setBalanceError(null);
      setIsLoading(false);
      setIsRefreshing(false);
      return;
    }
    setBalanceStatus("loading");
    setBalanceError(null);

    if (options.silent) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setErrors([]);

    const requests = [
      ["balanceConfig", getBalanceConfig()],
      ["rechargeConfig", getBalanceRechargeConfig()],
      ["localPoints", getBalanceLocalPoints()],
      ["userBalance", getBalanceUserBalance()],
      ["withdrawOrders", getWithdrawOrders({ limit: 8 })],
      ["balanceOrders", getBalanceOrders({ limit: 5 })],
      ["pointsOverview", getPointsOverview()],
      ["pointsLogs", getPointsLogs({ limit: 8 })],
    ] as const;

    const results = await Promise.allSettled(requests.map(([, request]) => request));
    const nextErrors: WalletError[] = [];

    results.forEach((result, index) => {
      const key = requests[index][0];
      if (result.status === "rejected") {
        const message = detailedWalletError(result.reason, t);
        nextErrors.push({
          label: t(loadLabelKeys[key]),
          message,
        });
        if (key === "userBalance") {
          setUserBalance(null);
          setBalanceStatus("error");
          setBalanceError(message);
        }
        return;
      }

      switch (key) {
        case "balanceConfig":
          setBalanceConfig(result.value as BalanceConfigPayload);
          break;
        case "rechargeConfig":
          setRechargeConfig(result.value as BalanceRechargeConfigPayload);
          break;
        case "localPoints":
          setLocalPoints(result.value as BalanceLocalPointsPayload);
          break;
        case "userBalance":
          setUserBalance(result.value as BalanceUserBalancePayload);
          setBalanceStatus("ready");
          setBalanceError(null);
          break;
        case "withdrawOrders":
          setWithdrawOrders(result.value as WithdrawOrdersPayload);
          break;
        case "balanceOrders":
          setBalanceOrders(result.value as BalanceOrdersPayload);
          break;
        case "pointsOverview":
          setPointsOverview(result.value as PointsOverviewPayload);
          break;
        case "pointsLogs":
          setPointsLogs(result.value as PointsLogsPayload);
          break;
      }
    });

    setErrors(nextErrors);
    setIsLoading(false);
    setIsRefreshing(false);
  }, [t]);

  useEffect(() => {
    if (initialData) {
      if (initialData.authenticated) {
        queueMicrotask(() => {
          void getBalanceUserBalance()
            .then((payload) => {
              setUserBalance(payload);
              setBalanceStatus("ready");
              setBalanceError(null);
            })
            .catch((error) => {
              setUserBalance(null);
              setBalanceStatus("error");
              setBalanceError(detailedWalletError(error, t));
            });
        });
      }
      return;
    }
    queueMicrotask(() => {
      void loadWallet();
    });
  }, [initialData, loadWallet, t]);

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

  const handleRedeemGiftCard = useCallback(async (product: PointsGiftCardProduct) => {
    if (!product.id || redeemingId) {
      return;
    }
    setRedeemMessage(null);
    setRedeemingId(product.id);
    try {
      const redemption = await redeemPointsGiftCard(product.id);
      setLatestRedemption(redemption);
      await Promise.all([loadWallet({ silent: true }), refreshRedemptions()]);
    } catch (error) {
      setRedeemMessage({ tone: "error", text: error instanceof Error ? error.message : t("giftCards.redeemFailed") });
    } finally {
      setRedeemingId(null);
    }
  }, [loadWallet, redeemingId, refreshRedemptions, t]);

  const handleRefresh = useCallback(async () => {
    await Promise.all([loadWallet({ silent: true }), refreshRedemptions()]);
  }, [loadWallet, refreshRedemptions]);

  if (authToken === null) {
    return <WalletLoginRequired />;
  }

  return (
    <main className="wallet-page theme-adaptive min-h-dvh bg-[#fbfbff] text-[#22222a]">
      <WalletDesktopView
        accountName={accountName}
        balanceError={balanceError}
        balanceStatus={balanceStatus}
        bills={bills}
        date={date}
        errors={errors}
        isLoading={isLoading}
        isRefreshing={isRefreshing}
        moonCoinBalance={moonCoinBalance}
        money={money}
        onBack={handleBack}
        onRedeemGiftCard={handleRedeemGiftCard}
        onRefresh={() => void handleRefresh()}
        pointLogRows={pointLogRows}
        pointsBalance={pointsBalance}
        pointsOverview={pointsOverview}
        purchaseOrders={purchaseOrders}
        rechargeAmountRange={rechargeAmountRange}
        rechargeConfig={rechargeConfig}
        rechargeOptions={rechargeOptions}
        rechargeUrl={rechargeUrl}
        redeemingId={redeemingId}
        redeemMessage={redeemMessage}
        latestRedemption={latestRedemption}
        loadRedemptionsPage={loadRedemptionsPage}
        pointsRedemptions={pointsRedemptions}
        redemptionsError={redemptionsError}
        redemptionsLoading={redemptionsLoading}
      />
      <div className="mx-auto min-h-dvh w-full max-w-[430px] bg-[#fbfbff] pb-6 shadow-[0_0_38px_rgba(112,94,173,0.08)] lg:hidden">
        <header className="sticky top-0 z-30 bg-[#fbfbff]/92 px-4 pb-3 pt-[calc(0.75rem+env(safe-area-inset-top))] backdrop-blur">
          <div className="grid h-11 grid-cols-[40px_minmax(0,1fr)_40px] items-center min-[390px]:grid-cols-[44px_minmax(0,1fr)_44px]">
            <button
              type="button"
              aria-label={t("actions.back")}
              onClick={handleBack}
              className="flex size-10 items-center justify-center rounded-full text-[#15151b] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <ArrowLeft className="size-6" strokeWidth={2.4} />
            </button>
            <h1 className="min-w-0 truncate text-center text-[21px] font-black tracking-normal text-[#16161d]">
              {t("title")}
            </h1>
            <button
              type="button"
              aria-label={t("actions.refresh")}
              onClick={() => void handleRefresh()}
              className="flex size-10 items-center justify-center justify-self-end rounded-full text-[#8b7adf] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <RefreshCw className={cn("size-5.5", isRefreshing && "animate-spin")} strokeWidth={2.4} />
            </button>
          </div>
        </header>

        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <Loader2 className="size-4 animate-spin" />
            {t("loading")}
          </div>
        ) : (
          <div className="space-y-4 px-4">
            {errors.length ? (
              <section className="rounded-[8px] border border-[#ffe1a8] bg-[#fff8e7] px-4 py-3 text-[12px] font-bold text-[#b67b10]">
                <p>{t("errors.partial")}</p>
                <ul className="mt-2 grid gap-1 font-semibold">
                  {errors.map((error) => (
                    <li key={error.label} className="break-words">{error.label}: {error.message}</li>
                  ))}
                </ul>
              </section>
            ) : null}

            <section className="relative overflow-hidden rounded-[8px] bg-[#8170e8] px-4 py-5 text-white shadow-[0_16px_38px_rgba(110,88,210,0.24)]">
              <div className="absolute -right-12 -top-16 size-40 rounded-full bg-white/18" />
              <div className="absolute bottom-0 right-5 size-20 rounded-full bg-[#c9a7ff]/22" />
              <div className="relative z-10">
                <p className="flex items-center gap-1.5 text-[13px] font-bold text-white/76">
                  <Coins className="size-4" />
                  {t("balance.title")}
                </p>
                <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
                  <div className="min-w-0">
                    {balanceStatus === "loading" ? (
                      <p className="flex items-center gap-2 text-[24px] font-black leading-none">
                        <Loader2 className="size-5 animate-spin" />
                        {t("balance.loading")}
                      </p>
                    ) : balanceStatus === "error" ? (
                      <>
                        <p className="text-[18px] font-black leading-tight">{t("balance.errorTitle")}</p>
                        <p className="mt-2 max-w-[300px] break-words text-[12px] font-semibold leading-5 text-white/78">{balanceError}</p>
                      </>
                    ) : (
                      <>
                        <p className="truncate text-[34px] font-black leading-none">{money(moonCoinBalance ?? 0)} {t("balance.unit")}</p>
                        <p className="mt-2 truncate text-[13px] font-bold text-white/72">{t("balance.account", { account: accountName })}</p>
                      </>
                    )}
                  </div>
                  {rechargeUrl ? (
                    <a
                      href={rechargeUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="flex h-9 shrink-0 items-center justify-center gap-1.5 rounded-full bg-[#c9a7ff]/55 px-4 text-[14px] font-black text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.18)] active:scale-[0.98]"
                    >
                      {t("actions.recharge")}
                    </a>
                  ) : null}
                </div>

              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("points.title")}</h2>
                  <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">
                    {t("points.todaySummary", { earned: money(pointsOverview?.today_earned ?? 0), cap: money(pointsOverview?.daily_cap ?? 0) })}
                  </p>
                </div>
                <span className="flex h-9 shrink-0 items-center gap-1.5 rounded-full bg-[#f0f7ff] px-3 text-[13px] font-black text-[#3479c8]">
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
                <div className="mt-3 rounded-[8px] bg-[#fff7e8] px-3 py-2 text-[12px] font-bold text-[#a96b00]">
                  {t("points.withdrawBonus", { percent: money(pointsOverview.active_bonus.bonus_percent) })}
                  {pointsOverview.active_bonus.expires_at ? ` · ${t("points.expiresAt", { date: date(pointsOverview.active_bonus.expires_at) })}` : ""}
                </div>
              ) : null}
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("tasks.title")}</h2>
                <ClipboardList className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 -mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
                {pointsOverview?.tasks?.length ? (
                  pointsOverview.tasks.map((task) => <PointsTaskCard key={task.task_type} money={money} task={task} t={t} />)
                ) : (
                  <div className="w-full shrink-0">
                    <WalletEmptyState text={t("tasks.empty")} />
                  </div>
                )}
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("giftCards.title")}</h2>
                <Gift className="size-5 text-[#c9c5d3]" />
              </div>
              {redeemMessage ? (
                <div className={cn(
                  "mt-3 rounded-[8px] px-3 py-2 text-[12px] font-bold",
                  redeemMessage.tone === "success" ? "bg-[#ecfdf3] text-[#16824a]" : "bg-[#fff1f1] text-[#c83232]",
                )}>
                  {redeemMessage.text}
                </div>
              ) : null}
              {latestRedemption ? <GiftCardRedemptionSuccess redemption={latestRedemption} /> : null}
              <div className="mt-3 grid gap-2">
                {pointsOverview?.gift_cards?.length ? (
                  pointsOverview.gift_cards.map((card) => (
                    <GiftCardRow
                      key={String(card.id)}
                      card={card}
                      loading={redeemingId === card.id}
                      money={money}
                      points={pointsBalance}
                      onRedeem={() => void handleRedeemGiftCard(card)}
                    />
                  ))
                ) : (
                  <WalletEmptyState text={t("giftCards.empty")} />
                )}
              </div>
            </section>

            <GiftCardRedemptionHistory
              error={redemptionsError}
              loading={redemptionsLoading}
              onPage={loadRedemptionsPage}
              payload={pointsRedemptions}
              variant="mobile"
            />

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("pointsLogs.title")}</h2>
                <ReceiptText className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 divide-y divide-[#f0eef7]">
                {pointLogRows.length ? (
                  pointLogRows.map((item) => <PointsLogRow key={String(item.id)} date={date} item={item} money={money} t={t} />)
                ) : (
                  <WalletEmptyState text={t("pointsLogs.empty")} />
                )}
              </div>
            </section>

            <section id="recharge" className="scroll-mt-20 rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("recharge.title")}</h2>
                  <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">
                    {rechargeAmountRange ?? (rechargeConfig?.custom_amount_enable ? t("recharge.customAmount") : t("recharge.fixedOptions"))}
                  </p>
                </div>
                {rechargeUrl ? (
                  <a
                    href={rechargeUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="flex h-9 shrink-0 items-center justify-center gap-1 rounded-full bg-[#f3f0ff] px-3 text-[12px] font-black text-[#765eda]"
                  >
                    {t("actions.recharge")}
                    <ExternalLink className="size-3.5" />
                  </a>
                ) : null}
              </div>

              <RechargeOptionGrid
                empty={balanceConfig?.enabled === false ? t("recharge.balanceDisabled") : t("recharge.empty")}
                href={rechargeUrl}
                money={money}
                options={rechargeOptions}
                t={t}
              />
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("bills.title")}</h2>
                <ReceiptText className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 divide-y divide-[#f0eef7]">
                {bills.length ? (
                  bills.map((bill) => <BillItem key={bill.id} date={date} item={bill} money={money} t={t} />)
                ) : (
                  <WalletEmptyState text={t("bills.empty")} />
                )}
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("purchases.title")}</h2>
                <ShoppingBag className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 divide-y divide-[#f0eef7]">
                {purchaseOrders.length ? (
                  purchaseOrders.map((order) => <PurchaseOrderItem key={order.id} date={date} item={order} money={money} t={t} />)
                ) : (
                  <WalletEmptyState text={t("purchases.empty")} />
                )}
              </div>
            </section>
          </div>
        )}
      </div>
    </main>
  );
}
