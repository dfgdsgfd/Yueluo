"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  Banknote,
  CheckCircle2,
  CircleDollarSign,
  CreditCard,
  Loader2,
  RefreshCw,
  Save,
  Send,
  Wallet,
  Upload,
} from "lucide-react";
import { toast } from "sonner";
import { useLocale, useTranslations } from "next-intl";
import {
  applyWithdraw,
  getCreatorConfig,
  getCreatorOverview,
  getStoredAccessToken,
  getWithdrawOrders,
  getWithdrawPaymentCode,
  getWithdrawWallet,
  saveWithdrawPaymentCode,
  uploadImage,
} from "@/lib/api";
import type {
  CreatorConfigPayload,
  CreatorOverviewPayload,
  WithdrawOrderItem,
  WithdrawPaymentCodePayload,
  WithdrawType,
  WithdrawWalletPayload,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const withdrawTypes = [
  { value: "cash", icon: Banknote },
  { value: "moon_coin", icon: CreditCard },
] as const satisfies ReadonlyArray<{ value: WithdrawType; icon: typeof Banknote }>;

export function MobileWithdrawPage() {
  const router = useRouter();
  const locale = useLocale();
  const t = useTranslations("publish.workbench");
  const [authToken, setAuthToken] = useState<string | null | undefined>(undefined);
  const [creatorConfig, setCreatorConfig] = useState<CreatorConfigPayload | null>(null);
  const [creatorOverview, setCreatorOverview] = useState<CreatorOverviewPayload | null>(null);
  const [withdrawWallet, setWithdrawWallet] = useState<WithdrawWalletPayload | null>(null);
  const [orders, setOrders] = useState<WithdrawOrderItem[]>([]);
  const [paymentDraft, setPaymentDraft] = useState<WithdrawPaymentCodePayload>({
    alipay_url: "",
    wechat_url: "",
  });
  const [withdrawAmount, setWithdrawAmount] = useState("");
  const [withdrawType, setWithdrawType] = useState<WithdrawType>("cash");
  const [isLoading, setIsLoading] = useState(true);
  const [isSavingCode, setIsSavingCode] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [uploadingCodeType, setUploadingCodeType] = useState<"wechat_url" | "alipay_url" | null>(null);

  const minimumAmount = creatorConfig?.minWithdrawAmount;
  const hasPaymentCode = Boolean(paymentDraft.wechat_url?.trim() || paymentDraft.alipay_url?.trim());
  const withdrawEnabled = creatorConfig?.withdrawEnabled !== false;
  const availableBalance = creatorOverview?.balance ?? 0;
  const canSubmit = withdrawEnabled && !isSubmitting;

  const loadWithdraw = useCallback(async (options: { silent?: boolean } = {}) => {
    const token = getStoredAccessToken();
    setAuthToken(token);

    if (!token) {
      setIsLoading(false);
      return;
    }

    if (!options.silent) {
      setIsLoading(true);
    }

    const [
      configResult,
      overviewResult,
      walletResult,
      paymentCodeResult,
      ordersResult,
    ] = await Promise.allSettled([
      getCreatorConfig(),
      getCreatorOverview(),
      getWithdrawWallet(),
      getWithdrawPaymentCode(),
      getWithdrawOrders({ limit: 6 }),
    ]);

    if (configResult.status === "fulfilled") {
      setCreatorConfig(configResult.value);
    }
    if (overviewResult.status === "fulfilled") {
      setCreatorOverview(overviewResult.value);
    }
    if (walletResult.status === "fulfilled") {
      setWithdrawWallet(walletResult.value);
    }
    if (paymentCodeResult.status === "fulfilled") {
      setPaymentDraft({
        alipay_url: paymentCodeResult.value.alipay_url ?? "",
        wechat_url: paymentCodeResult.value.wechat_url ?? "",
      });
    }
    if (ordersResult.status === "fulfilled") {
      setOrders(ordersResult.value.list);
    }

    const failed = [configResult, overviewResult, walletResult, paymentCodeResult, ordersResult].some(
      (result) => result.status === "rejected",
    );
    if (failed && options.silent) {
      toast.error(t("withdraw.refreshFailed"));
    }

    setIsLoading(false);
  }, [t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadWithdraw();
    });
  }, [loadWithdraw]);

  const summaryItems = useMemo(
    () => [
      { label: t("withdraw.monthEarnings"), value: formatMoney(creatorOverview?.month_earnings, locale), tone: "purple" },
      { label: t("withdraw.totalEarnings"), value: formatMoney(creatorOverview?.total_earnings, locale), tone: "blue" },
      { label: t("withdraw.frozen"), value: formatMoney(withdrawWallet?.frozen_amount, locale), tone: "amber" },
    ],
    [creatorOverview, locale, t, withdrawWallet],
  );

  const handleBack = useCallback(() => {
    if (window.history.length > 1) {
      router.back();
      return;
    }
    router.push("/creator-center");
  }, [router]);

  async function handleSavePaymentCode() {
    if (!hasPaymentCode) {
      toast.error(t("withdraw.paymentCodeRequired"));
      return;
    }

    setIsSavingCode(true);
    try {
      const nextCode = await saveWithdrawPaymentCode({
        alipay_url: paymentDraft.alipay_url?.trim() || null,
        wechat_url: paymentDraft.wechat_url?.trim() || null,
      });
      setPaymentDraft({
        alipay_url: nextCode.alipay_url ?? "",
        wechat_url: nextCode.wechat_url ?? "",
      });
      toast.success(t("withdraw.paymentCodeSaved"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("withdraw.saveFailed"));
    } finally {
      setIsSavingCode(false);
    }
  }

  async function handleUploadPaymentCode(file: File | undefined, key: "wechat_url" | "alipay_url") {
    if (!file) {
      return;
    }
    if (!file.type.startsWith("image/")) {
      toast.error(t("withdraw.imageRequired"));
      return;
    }

    setUploadingCodeType(key);
    try {
      const asset = await uploadImage(file);
      setPaymentDraft((current) => ({ ...current, [key]: asset.url }));
      toast.success(t("withdraw.uploadedSavePrompt"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("withdraw.uploadFailed"));
    } finally {
      setUploadingCodeType(null);
    }
  }

  async function handleApplyWithdraw() {
    const amount = Number(withdrawAmount);
    if (!Number.isFinite(amount) || amount <= 0) {
      toast.error(t("withdraw.invalidAmount"));
      return;
    }
    if (typeof minimumAmount === "number" && amount < minimumAmount) {
      toast.error(t("withdraw.minimumError", { amount: formatMoney(minimumAmount, locale) }));
      return;
    }
    if (withdrawType === "cash" && !hasPaymentCode) {
      toast.error(t("withdraw.savePaymentCodeFirst"));
      return;
    }

    setIsSubmitting(true);
    try {
      await applyWithdraw({ amount, type: withdrawType });
      toast.success(t("withdraw.submitted"));
      setWithdrawAmount("");
      await loadWithdraw({ silent: true });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("withdraw.submitFailed"));
    } finally {
      setIsSubmitting(false);
    }
  }

  if (authToken === null) {
    return <WithdrawLoginRequired />;
  }

  return (
    <main className="min-h-dvh bg-[#fbfbff] text-[#22222a]">
      <div className="mx-auto min-h-dvh w-full max-w-[430px] bg-[#fbfbff] pb-6 shadow-[0_0_38px_rgba(112,94,173,0.08)]">
        <header className="sticky top-0 z-30 bg-[#fbfbff]/92 px-4 pb-3 pt-[calc(0.75rem+env(safe-area-inset-top))] backdrop-blur">
          <div className="grid h-11 grid-cols-[40px_minmax(0,1fr)_40px] items-center min-[390px]:grid-cols-[44px_minmax(0,1fr)_44px]">
            <button
              type="button"
              aria-label={t("withdraw.back")}
              onClick={handleBack}
              className="flex size-10 items-center justify-center rounded-full text-[#15151b] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <ArrowLeft className="size-6" strokeWidth={2.4} />
            </button>
            <h1 className="min-w-0 truncate text-center text-[21px] font-black tracking-normal text-[#16161d]">
              {t("withdraw.title")}
            </h1>
            <button
              type="button"
              aria-label={t("refresh")}
              onClick={() => void loadWithdraw({ silent: true })}
              className="flex size-10 items-center justify-center justify-self-end rounded-full text-[#8b7adf] active:bg-[#f0eef8] min-[390px]:size-11"
            >
              <RefreshCw className={cn("size-5.5", isLoading && "animate-spin")} strokeWidth={2.4} />
            </button>
          </div>
        </header>

        {isLoading ? (
          <div className="flex min-h-[520px] items-center justify-center gap-2 text-sm font-bold text-[#9a96a5]">
            <Loader2 className="size-4 animate-spin" />
            {t("withdraw.loading")}
          </div>
        ) : (
          <div className="space-y-4 px-4">
            <section className="relative overflow-hidden rounded-[8px] bg-[#8170e8] px-4 py-5 text-white shadow-[0_16px_38px_rgba(110,88,210,0.24)]">
              <div className="absolute -right-12 -top-16 size-40 rounded-full bg-white/18" />
              <div className="absolute bottom-0 right-5 size-20 rounded-full bg-[#c9a7ff]/22" />
              <div className="relative z-10">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="flex items-center gap-1.5 text-[13px] font-bold text-white/76">
                      <CircleDollarSign className="size-4" />
                      {t("withdraw.available")}
                    </p>
                    <p className="mt-3 truncate text-[34px] font-black leading-none">¥{formatMoney(availableBalance, locale)}</p>
                  </div>
                  <span className={cn(
                    "shrink-0 rounded-full px-3 py-1 text-[12px] font-black",
                    withdrawEnabled ? "bg-white/20 text-white" : "bg-black/15 text-white/62",
                  )}>
                    {withdrawEnabled ? t("withdraw.open") : t("withdraw.closed")}
                  </span>
                </div>

                <div className="mt-5 grid grid-cols-3 gap-2 text-center">
                  {summaryItems.map((item) => (
                    <div key={item.label} className="min-w-0 border-l border-white/18 first:border-l-0 first:pl-0">
                      <p className="truncate text-[18px] font-black leading-none">{item.value}</p>
                      <p className="mt-2 truncate text-[12px] font-bold text-white/72">{item.label}</p>
                    </div>
                  ))}
                </div>
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("withdraw.application")}</h2>
                <span className="text-[12px] font-bold text-[#9a96a5]">
                  {t("withdraw.minimum", { amount: formatMoney(minimumAmount, locale) })}
                </span>
              </div>

              <label className="mt-4 block">
                <span className="text-[12px] font-bold text-[#777381]">{t("withdraw.amount")}</span>
                <input
                  type="number"
                  inputMode="decimal"
                  min="0"
                  step="0.01"
                  value={withdrawAmount}
                  onChange={(event) => setWithdrawAmount(event.target.value)}
                  placeholder={t("withdraw.amountPlaceholder")}
                  className="mt-2 h-12 w-full rounded-[8px] border border-[#ebe8f3] bg-[#fbfaff] px-3 text-[18px] font-black text-[#15151b] outline-none transition focus:border-[#8b7adf]"
                />
              </label>

              <div className="mt-4">
                <span className="text-[12px] font-bold text-[#777381]">{t("withdraw.method")}</span>
                <div className="mt-2 grid grid-cols-2 gap-2">
                  {withdrawTypes.map(({ icon: Icon, value }) => (
                    <button
                      key={value}
                      type="button"
                      onClick={() => setWithdrawType(value)}
                      className={cn(
                        "flex h-12 min-w-0 items-center justify-center gap-2 rounded-[8px] border text-[13px] font-black transition",
                        withdrawType === value
                          ? "border-[#8b7adf] bg-[#f4efff] text-[#765eda]"
                          : "border-[#ebe8f3] bg-[#fbfaff] text-[#777381]",
                      )}
                    >
                      <Icon className="size-4.5 shrink-0" />
                      <span className="truncate">{t(`withdrawType.${value === "moon_coin" ? "moonCoin" : "cash"}`)}</span>
                    </button>
                  ))}
                </div>
              </div>

              <div className="mt-4 grid gap-3">
                <PaymentCodeField
                  id="wechat-payment-code"
                  label={t("withdraw.wechatCode")}
                  uploading={uploadingCodeType === "wechat_url"}
                  value={paymentDraft.wechat_url ?? ""}
                  onUpload={(file) => void handleUploadPaymentCode(file, "wechat_url")}
                />
                <PaymentCodeField
                  id="alipay-payment-code"
                  label={t("withdraw.alipayCode")}
                  uploading={uploadingCodeType === "alipay_url"}
                  value={paymentDraft.alipay_url ?? ""}
                  onUpload={(file) => void handleUploadPaymentCode(file, "alipay_url")}
                />
              </div>

              <div className="mt-4 grid grid-cols-[minmax(0,0.88fr)_minmax(0,1.12fr)] gap-2">
                <button
                  type="button"
                  onClick={handleSavePaymentCode}
                  disabled={isSavingCode}
                  className="flex h-11 min-w-0 items-center justify-center gap-1.5 rounded-full bg-[#f3f0ff] px-3 text-[13px] font-black text-[#765eda] active:scale-[0.98] disabled:opacity-60"
                >
                  {isSavingCode ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
                  <span className="truncate">{t("withdraw.savePaymentCode")}</span>
                </button>
                <button
                  type="button"
                  onClick={handleApplyWithdraw}
                  disabled={!canSubmit}
                  className="flex h-11 min-w-0 items-center justify-center gap-1.5 rounded-full bg-[#8069dd] px-3 text-[13px] font-black text-white shadow-[0_8px_22px_rgba(128,105,221,0.24)] active:scale-[0.98] disabled:opacity-55"
                >
                  {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
                  <span className="truncate">{t("withdraw.submit")}</span>
                </button>
              </div>
            </section>

            <section className="rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-[18px] font-black tracking-normal text-[#1f1f27]">{t("withdraw.records")}</h2>
                <Wallet className="size-5 text-[#c9c5d3]" />
              </div>
              <div className="mt-3 divide-y divide-[#f0eef7]">
                {orders.length ? (
                  orders.map((order) => (
                    <article key={order.id} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 py-3">
                      <div className="min-w-0">
                        <p className="truncate text-[14px] font-black text-[#1d1d24]">
                          {t(`withdrawType.${order.type === "moon_coin" ? "moonCoin" : "cash"}`)} · ¥{formatMoney(order.amount, locale)}
                        </p>
                        <p className="mt-1 truncate text-[12px] font-bold text-[#9a96a5]">
                          {formatDate(order.created_at, locale, t("withdraw.justNow"))}
                          {order.remark ? ` · ${order.remark}` : ""}
                        </p>
                      </div>
                      <span className={cn("rounded-full px-2.5 py-1 text-[12px] font-black", withdrawStatusClassName(order.status))}>
                        {t(`withdrawStatus.${isKnownWithdrawStatus(order.status) ? order.status : "unknown"}`)}
                      </span>
                    </article>
                  ))
                ) : (
                  <div className="flex min-h-[112px] flex-col items-center justify-center gap-2 text-center text-[13px] font-bold text-[#9a96a5]">
                    <CheckCircle2 className="size-6 text-[#c9c5d3]" />
                    {t("withdraw.noRecords")}
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

function PaymentCodeField({
  id,
  label,
  onUpload,
  uploading,
  value,
}: {
  id: string;
  label: string;
  onUpload: (file: File | undefined) => void;
  uploading: boolean;
  value: string;
}) {
  const t = useTranslations("publish.workbench.withdraw");
  const imageUrl = value.trim();

  return (
    <div className="block">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-[12px] font-bold text-[#777381]">{label}</span>
        <label className="relative shrink-0" htmlFor={id}>
          <input
            id={id}
            type="file"
            accept="image/*"
            disabled={uploading}
            className="sr-only"
            onChange={(event) => {
              onUpload(event.target.files?.[0]);
              event.target.value = "";
            }}
          />
          <span
            className={cn(
              "flex h-8 cursor-pointer items-center justify-center gap-1.5 rounded-full bg-[#f3f0ff] px-3 text-[12px] font-black text-[#765eda]",
              uploading && "cursor-not-allowed opacity-60",
            )}
          >
            {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
            {imageUrl ? t("replace") : t("upload")}
          </span>
        </label>
      </div>
      <div className="flex min-h-[148px] items-center justify-center overflow-hidden rounded-[8px] border border-[#ebe8f3] bg-[#fbfaff]">
        {imageUrl ? (
          <div className="relative h-[220px] w-full">
            <Image
              src={imageUrl}
              alt={t("codeImageAlt", { label })}
              fill
              unoptimized
              sizes="398px"
              className="object-contain p-2"
            />
          </div>
        ) : (
          <label
            htmlFor={id}
            className={cn(
              "flex min-h-[148px] w-full cursor-pointer flex-col items-center justify-center gap-2 text-[12px] font-black text-[#a59fb5]",
              uploading && "cursor-not-allowed opacity-60",
            )}
          >
            {uploading ? <Loader2 className="size-5 animate-spin text-[#765eda]" /> : <Upload className="size-5 text-[#c6bddf]" />}
            {t("uploadCode")}
          </label>
        )}
      </div>
    </div>
  );
}

function WithdrawLoginRequired() {
  const t = useTranslations("publish.workbench.withdraw");
  return (
    <main className="flex min-h-dvh items-center justify-center bg-[#fbfbff] px-4 text-[#22222a]">
      <section className="w-full max-w-[360px] rounded-[8px] bg-white px-5 py-6 text-center shadow-[0_10px_28px_rgba(116,102,176,0.08)]">
        <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#f4efff] text-[#765eda]">
          <Wallet className="size-6" />
        </div>
        <h1 className="mt-4 text-[18px] font-black">{t("loginTitle")}</h1>
        <p className="mt-2 text-sm font-bold leading-6 text-[#9a96a5]">{t("loginDescription")}</p>
        <div className="mt-5 grid grid-cols-2 gap-2">
          <Link href="/login" className="flex h-10 items-center justify-center rounded-full bg-[#8069dd] text-sm font-black text-white">
            {t("login")}
          </Link>
          <Link href="/creator-center" className="flex h-10 items-center justify-center rounded-full bg-[#f3f0ff] text-sm font-black text-[#765eda]">
            {t("back")}
          </Link>
        </div>
      </section>
    </main>
  );
}

function formatMoney(value: number | undefined, locale: string) {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;
  return amount.toLocaleString(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  });
}

function formatDate(value: string | null | undefined, locale: string, fallback: string) {
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

function isKnownWithdrawStatus(value: string): value is "pending" | "approved" | "rejected" | "paid" {
  return value === "pending" || value === "approved" || value === "rejected" || value === "paid";
}

function withdrawStatusClassName(value: string) {
  switch (value) {
    case "approved":
    case "paid":
      return "bg-[#e9fbf4] text-[#42b389]";
    case "rejected":
      return "bg-[#fff0f5] text-[#f25a89]";
    case "pending":
      return "bg-[#fff7db] text-[#f0a000]";
    default:
      return "bg-[#f4efff] text-[#765eda]";
  }
}
