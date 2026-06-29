"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, Copy, Gift, Loader2, ReceiptText } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { getPointsRedemptions } from "@/lib/api";
import type { PointsGiftCardProduct, PointsGiftCardRedemption, PointsRedemptionsPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

const redemptionPageSize = 8;

export function useGiftCardRedemptionHistory(initialData: PointsRedemptionsPayload | null) {
  const t = useTranslations("walletPage");
  const [payload, setPayload] = useState<PointsRedemptionsPayload | null>(initialData);
  const [loading, setLoading] = useState(!initialData);
  const [error, setError] = useState<string | null>(null);

  const loadPage = useCallback(async (page: number) => {
    setLoading(true);
    setError(null);
    try {
      const nextPayload = await getPointsRedemptions({ page, limit: redemptionPageSize });
      setPayload(nextPayload);
      return nextPayload;
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : t("redemptions.loadFailed");
      setError(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    if (initialData) {
      return;
    }
    queueMicrotask(() => {
      void loadPage(1);
    });
  }, [initialData, loadPage]);

  const refresh = useCallback(() => loadPage(1), [loadPage]);

  return { error, loadPage, loading, payload, refresh };
}

export function GiftCardRedemptionSuccess({ redemption }: { redemption: PointsGiftCardRedemption }) {
  const t = useTranslations("walletPage");

  return (
    <div className="mt-3 rounded-[8px] border border-[#b9ecd1] bg-[#ecfdf3] p-3 text-[#126b3d]" data-testid="gift-card-redeem-success">
      <p className="text-xs font-black">{t("giftCards.redeemSuccess", { name: redemption.product?.name ?? t("giftCards.defaultFaceValue") })}</p>
      <GiftCardCode code={redemption.code ?? ""} compact />
    </div>
  );
}

export function GiftCardRow({
  card,
  loading,
  money,
  points,
  onRedeem,
}: {
  card: PointsGiftCardProduct;
  loading: boolean;
  money: (value?: number) => string;
  points: number;
  onRedeem: () => void;
}) {
  const t = useTranslations("walletPage");
  const disabled = loading || card.available_stock <= 0 || points < card.points_required;
  return (
    <article className="grid min-h-[76px] grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-3 rounded-[8px] bg-[#fbfaff] px-3">
      <span className="flex size-10 items-center justify-center rounded-[10px] bg-[#fff3e2] text-[#d98218]">
        <Gift className="size-4.5" />
      </span>
      <div className="min-w-0">
        <h3 className="truncate text-[14px] font-black text-[var(--wallet-text)]">{card.name}</h3>
        <p className="mt-1 truncate text-[11px] font-bold text-[#9a96a5]">
          {t("giftCards.meta", { value: card.face_value || t("giftCards.defaultFaceValue"), stock: card.available_stock })}
        </p>
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={onRedeem}
        className="flex h-9 min-w-[76px] items-center justify-center rounded-full bg-[#8069dd] px-3 text-[12px] font-black text-white disabled:bg-[#d9d6e5] disabled:text-white"
      >
        {loading ? <Loader2 className="size-4 animate-spin" /> : t("money.points", { amount: money(card.points_required) })}
      </button>
    </article>
  );
}

export function GiftCardRedemptionHistory({
  error,
  loading,
  onPage,
  payload,
  variant,
}: {
  error: string | null;
  loading: boolean;
  onPage: (page: number) => void | Promise<unknown>;
  payload: PointsRedemptionsPayload | null;
  variant: "desktop" | "mobile";
}) {
  const t = useTranslations("walletPage");
  const locale = useLocale();
  const page = Math.max(1, Number(payload?.pagination.page ?? 1));
  const total = Math.max(0, Number(payload?.pagination.total ?? payload?.list.length ?? 0));
  const pages = Math.max(1, Number(payload?.pagination.totalPages ?? (Math.ceil(total / redemptionPageSize) || 1)));
  const rows = payload?.list ?? [];
  const number = useMemo(() => new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }), [locale]);

  return (
    <section
      id="gift-card-redemptions"
      className={cn(
        "min-w-0 scroll-mt-20 bg-[var(--wallet-card)]",
        variant === "mobile"
          ? "rounded-[8px] bg-white px-4 py-4 shadow-[0_10px_28px_rgba(116,102,176,0.08)]"
          : "rounded-lg p-5 shadow-[0_12px_30px_rgba(31,41,55,0.06)]",
      )}
      data-testid={`gift-card-redemption-history-${variant}`}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h2 className={cn("font-black tracking-normal text-[var(--wallet-text)]", variant === "mobile" ? "text-[18px]" : "text-lg")}>
            {t("redemptions.title")}
          </h2>
          <p className="mt-1 truncate text-xs font-bold text-[var(--wallet-muted)]">{t("redemptions.total", { total })}</p>
        </div>
        <ReceiptText className="size-5 shrink-0 text-[var(--wallet-faint)]" />
      </div>

      {error ? (
        <div className="mt-4 rounded-[8px] bg-[#fff1f1] px-3 py-3 text-xs font-bold text-[#c83232]">
          {t("redemptions.loadFailed")}: {error}
        </div>
      ) : loading && !payload ? (
        <div className="flex min-h-32 items-center justify-center gap-2 text-sm font-bold text-[var(--wallet-muted)]">
          <Loader2 className="size-4 animate-spin" />
          {t("redemptions.loading")}
        </div>
      ) : rows.length ? (
        <div className="mt-4 grid gap-3">
          {rows.map((redemption) => (
            <article key={String(redemption.id)} className="min-w-0 rounded-[8px] border border-[var(--wallet-border)] bg-[var(--wallet-soft)] p-3">
              <div className="flex min-w-0 items-start gap-3">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-[#fff3e2] text-[#d98218]">
                  <Gift className="size-5" />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-3 gap-y-1">
                    <h3 className="min-w-0 truncate text-sm font-black text-[var(--wallet-text)]">
                      {redemption.product?.name ?? t("giftCards.defaultFaceValue")}
                    </h3>
                    <span className="shrink-0 text-[11px] font-bold text-[var(--wallet-muted)]">{formatDate(redemption.created_at, locale, t("time.justNow"))}</span>
                  </div>
                  <p className="mt-1 text-xs font-bold text-[var(--wallet-muted)]">
                    {t("redemptions.meta", {
                      points: number.format(redemption.points_spent),
                      status: t(`redemptions.status.${redemption.status === "completed" ? "completed" : "other"}`),
                      value: redemption.product?.face_value ?? t("giftCards.defaultFaceValue"),
                    })}
                  </p>
                </div>
              </div>
              <GiftCardCode code={redemption.code ?? ""} />
            </article>
          ))}
        </div>
      ) : (
        <div className="mt-4 flex min-h-28 items-center justify-center rounded-[8px] bg-[var(--wallet-soft)] text-sm font-bold text-[var(--wallet-muted)]">
          {t("redemptions.empty")}
        </div>
      )}

      <div className="mt-4 flex min-w-0 items-center justify-between gap-3 border-t border-[var(--wallet-border)] pt-3">
        <span className="min-w-0 truncate text-xs font-bold text-[var(--wallet-muted)]">{t("redemptions.pageSummary", { page, pages, total })}</span>
        <div className="flex shrink-0 gap-2">
          <Button type="button" variant="outline" size="icon" aria-label={t("redemptions.previous")} disabled={loading || page <= 1} onClick={() => void onPage(page - 1)} className="size-9 border-[var(--wallet-border)] bg-[var(--wallet-card)]">
            <ChevronLeft className="size-4" />
          </Button>
          <Button type="button" variant="outline" size="icon" aria-label={t("redemptions.next")} disabled={loading || page >= pages} onClick={() => void onPage(page + 1)} className="size-9 border-[var(--wallet-border)] bg-[var(--wallet-card)]">
            <ChevronRight className="size-4" />
          </Button>
        </div>
      </div>
    </section>
  );
}

export function WalletInfoMini({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-[8px] bg-[var(--wallet-soft)] px-2 py-3 text-center">
      <p className="truncate text-[11px] font-bold text-[var(--wallet-muted)]">{label}</p>
      <p className="mt-1 truncate text-[14px] font-black text-[var(--wallet-text)]">{value}</p>
    </div>
  );
}

export function WalletEmptyState({ text }: { text: string }) {
  return (
    <div className="flex min-h-[112px] items-center justify-center text-[13px] font-bold text-[var(--wallet-muted)]">
      {text}
    </div>
  );
}

function GiftCardCode({ code, compact = false }: { code: string; compact?: boolean }) {
  const t = useTranslations("walletPage");

  async function copyCode() {
    try {
      await navigator.clipboard.writeText(code);
      toast.success(t("redemptions.copySuccess"));
    } catch {
      toast.error(t("redemptions.copyFailed"));
    }
  }

  return (
    <div className={cn("mt-3 flex min-w-0 items-center gap-2 rounded-[8px] bg-white/75", compact ? "p-2" : "p-3")}>
      <code className="min-w-0 flex-1 break-all text-xs font-black leading-5 text-[var(--wallet-text)]" data-testid="gift-card-code">{code || t("redemptions.codeUnavailable")}</code>
      <Button type="button" variant="ghost" size="icon" aria-label={t("redemptions.copy")} disabled={!code} onClick={() => void copyCode()} className="size-9 shrink-0 text-[#765eda] hover:bg-[#f3f0ff]">
        <Copy className="size-4" />
      </Button>
    </div>
  );
}

function formatDate(value: string | undefined, locale: string, fallback: string) {
  if (!value) return fallback;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}
