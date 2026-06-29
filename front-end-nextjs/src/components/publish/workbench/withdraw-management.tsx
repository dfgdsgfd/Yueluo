"use client";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import Image from "next/image";
import {
  Loader2,
  RefreshCw,
  Save,
  Send,
  Upload
} from "lucide-react";
import {
  toast
} from "sonner";
import { useLocale, useTranslations } from "next-intl";
import {
  Button
} from "@/components/ui/button";
import {
  applyWithdraw,
  getCreatorConfig,
  getCreatorOverview,
  getStoredAccessToken,
  getWithdrawOrders,
  getWithdrawPaymentCode,
  getWithdrawWallet,
  saveWithdrawPaymentCode,
  uploadImage
} from "@/lib/api";
import type {
  CreatorConfigPayload,
  CreatorOverviewPayload,
  WithdrawOrderItem,
  WithdrawPaymentCodePayload,
  WithdrawType,
  WithdrawWalletPayload
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  withdrawOrderLimit,
  withdrawTypes
} from "./workbench-config";
import {
  formatCreatorCurrency,
  formatWithdrawDate,
  withdrawStatusClassName,
  withdrawStatusKey,
  withdrawTypeKey
} from "./creator-formatters";
import {
  CreatorEmptyState,
  SectionHeader
} from "./shared-ui";

export function CreatorWithdrawManagement() {
  const t = useTranslations("publish.workbench");
  const locale = useLocale();
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
    if (!getStoredAccessToken()) {
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
      getWithdrawOrders({ limit: withdrawOrderLimit }),
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
    const frameId = window.requestAnimationFrame(() => {
      void loadWithdraw();
    });

    return () => window.cancelAnimationFrame(frameId);
  }, [loadWithdraw]);

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
      toast.error(t("withdraw.minimumError", { amount: formatCreatorCurrency(minimumAmount) }));
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

  const summaryItems = [
    { label: t("withdraw.monthEarnings"), value: formatCreatorCurrency(creatorOverview?.month_earnings) },
    { label: t("withdraw.totalEarnings"), value: formatCreatorCurrency(creatorOverview?.total_earnings) },
    { label: t("withdraw.frozen"), value: formatCreatorCurrency(withdrawWallet?.frozen_amount) },
  ];

  return (
    <div className="mx-auto min-h-[calc(100vh-64px)] w-full max-w-[1440px] px-4 py-6 sm:px-5 lg:px-6 2xl:px-8">
      <div className="space-y-5">
        <section className="rounded-2xl bg-white p-6 shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <SectionHeader title={t("withdraw.title")} description={t("withdraw.description")} />
            <Button
              type="button"
              variant="outline"
              onClick={() => void loadWithdraw({ silent: true })}
              className="h-10 border-[#e3e3e6] bg-white px-4 text-[#55555d]"
            >
              <RefreshCw className={cn("size-4", isLoading && "animate-spin")} />
              {t("refresh")}
            </Button>
          </div>

          {isLoading ? (
            <div className="mt-6 flex min-h-[360px] items-center justify-center gap-2 text-sm font-semibold text-[#8a8a91]">
              <Loader2 className="size-4 animate-spin" />
              {t("withdraw.loading")}
            </div>
          ) : (
            <div className="mt-6 grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,0.9fr)_minmax(320px,0.52fr)]">
              <div className="min-w-0 space-y-5">
                <section className="rounded-2xl bg-[#fafafa] p-5">
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-[#777780]">{t("withdraw.available")}</p>
                      <p className="mt-2 truncate text-4xl font-semibold tracking-normal text-[#25252b]">
                        ¥{formatCreatorCurrency(availableBalance)}
                      </p>
                    </div>
                    <span
                      className={cn(
                        "rounded-full px-3 py-1 text-xs font-semibold",
                        withdrawEnabled ? "bg-[#e9fbf4] text-[#42b389]" : "bg-[#f1f1f3] text-[#8a8a91]",
                      )}
                    >
                      {withdrawEnabled ? t("withdraw.open") : t("withdraw.closed")}
                    </span>
                  </div>
                  <div className="mt-5 grid grid-cols-[repeat(auto-fit,minmax(min(100%,160px),1fr))] gap-3">
                    {summaryItems.map((item) => (
                      <div key={item.label} className="min-w-0 rounded-xl bg-white p-4">
                        <p className="truncate text-xs text-[#8a8a91]">{item.label}</p>
                        <p className="mt-2 truncate text-xl font-semibold text-[#25252b]">{item.value}</p>
                      </div>
                    ))}
                  </div>
                </section>

                <section className="rounded-2xl bg-[#fafafa] p-5">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <h2 className="text-base font-semibold text-[#25252b]">{t("withdraw.application")}</h2>
                    <span className="text-xs text-[#8a8a91]">
                      {t("withdraw.minimum", { amount: formatCreatorCurrency(minimumAmount) })}
                    </span>
                  </div>
                  <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(220px,0.65fr)]">
                    <label className="block min-w-0">
                      <span className="mb-2 block text-sm font-medium text-[#4f4f57]">{t("withdraw.amount")}</span>
                      <input
                        type="number"
                        inputMode="decimal"
                        min="0"
                        step="0.01"
                        value={withdrawAmount}
                        onChange={(event) => setWithdrawAmount(event.target.value)}
                        placeholder={t("withdraw.amountPlaceholder")}
                        className="h-12 w-full rounded-xl border border-[#e8e8eb] bg-white px-4 text-lg font-semibold outline-none transition focus:border-primary"
                      />
                    </label>
                    <div className="min-w-0">
                      <span className="mb-2 block text-sm font-medium text-[#4f4f57]">{t("withdraw.method")}</span>
                      <div className="grid grid-cols-2 gap-2">
                        {withdrawTypes.map(({ icon: Icon, labelKey, value }) => (
                          <button
                            key={value}
                            type="button"
                            onClick={() => setWithdrawType(value)}
                            className={cn(
                              "flex h-12 min-w-0 items-center justify-center gap-2 rounded-xl border text-sm font-semibold transition-colors",
                              withdrawType === value
                                ? "border-primary bg-[#fff1f3] text-primary"
                                : "border-[#e8e8eb] bg-white text-[#67676f] hover:border-primary/40",
                            )}
                          >
                            <Icon className="size-4 shrink-0" />
                            <span className="truncate">{t(labelKey)}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="mt-4 grid gap-4 md:grid-cols-2">
                    <CreatorPaymentCodeField
                      id="creator-wechat-payment-code"
                      label={t("withdraw.wechatCode")}
                      uploading={uploadingCodeType === "wechat_url"}
                      value={paymentDraft.wechat_url ?? ""}
                      onUpload={(file) => void handleUploadPaymentCode(file, "wechat_url")}
                    />
                    <CreatorPaymentCodeField
                      id="creator-alipay-payment-code"
                      label={t("withdraw.alipayCode")}
                      uploading={uploadingCodeType === "alipay_url"}
                      value={paymentDraft.alipay_url ?? ""}
                      onUpload={(file) => void handleUploadPaymentCode(file, "alipay_url")}
                    />
                  </div>

                  <div className="mt-5 flex flex-wrap justify-end gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleSavePaymentCode}
                      disabled={isSavingCode}
                      className="h-10 border-[#e3e3e6] bg-white px-5 text-[#55555d]"
                    >
                      {isSavingCode ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
                      {t("withdraw.savePaymentCode")}
                    </Button>
                    <Button
                      type="button"
                      onClick={handleApplyWithdraw}
                      disabled={!canSubmit}
                      className="h-10 px-6"
                    >
                      {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
                      {t("withdraw.submit")}
                    </Button>
                  </div>
                </section>
              </div>

              <section className="min-w-0 rounded-2xl bg-[#fafafa] p-5">
                <div className="flex items-center justify-between gap-3">
                  <h2 className="text-base font-semibold text-[#25252b]">{t("withdraw.records")}</h2>
                  <span className="text-xs text-[#8a8a91]">{t("withdraw.recordCount", { count: orders.length })}</span>
                </div>
                <div className="mt-4 space-y-3">
                  {orders.length ? (
                    orders.map((order) => (
                      <article key={order.id} className="rounded-xl bg-white p-4">
                        <div className="flex items-center justify-between gap-3">
                          <p className="truncate text-sm font-semibold text-[#25252b]">
                            {t(withdrawTypeKey(order.type))} · ¥{formatCreatorCurrency(order.amount)}
                          </p>
                          <span className={cn("shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold", withdrawStatusClassName(order.status))}>
                            {t(withdrawStatusKey(order.status))}
                          </span>
                        </div>
                        <p className="mt-2 truncate text-xs text-[#8a8a91]">
                          {formatWithdrawDate(order.created_at, locale) || t("withdraw.justNow")}
                          {order.remark ? ` · ${order.remark}` : ""}
                        </p>
                      </article>
                    ))
                  ) : (
                    <CreatorEmptyState label={t("withdraw.noRecords")} />
                  )}
                </div>
              </section>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}


export function CreatorPaymentCodeField({
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
    <div className="min-w-0">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-sm font-medium text-[#4f4f57]">{label}</span>
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
              "inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-full bg-white px-3 text-xs font-semibold text-primary shadow-sm",
              uploading && "cursor-not-allowed opacity-60",
            )}
          >
            {uploading ? <Loader2 className="size-3.5 animate-spin" /> : <Upload className="size-3.5" />}
            {imageUrl ? t("replace") : t("upload")}
          </span>
        </label>
      </div>
      <div className="flex min-h-[180px] items-center justify-center overflow-hidden rounded-xl border border-dashed border-[#dedee3] bg-white">
        {imageUrl ? (
          <div className="relative h-[220px] w-full">
            <Image
              src={imageUrl}
              alt={t("codeImageAlt", { label })}
              fill
              unoptimized
              sizes="(min-width: 1024px) 320px, 90vw"
              className="object-contain p-3"
            />
          </div>
        ) : (
          <label
            htmlFor={id}
            className={cn(
              "flex min-h-[180px] w-full cursor-pointer flex-col items-center justify-center gap-2 text-xs font-semibold text-[#9a9aa1]",
              uploading && "cursor-not-allowed opacity-60",
            )}
          >
            {uploading ? <Loader2 className="size-5 animate-spin text-primary" /> : <Upload className="size-5 text-[#cfcfd5]" />}
            {t("uploadCode")}
          </label>
        )}
      </div>
    </div>
  );
}
