"use client";

import type { Dispatch, SetStateAction } from "react";
import Link from "next/link";
import { BadgeCheck, Check, ChevronDown, ChevronRight, Coins, Download, LoaderCircle, LogOut, Menu, Monitor, Moon, Palette, Settings, Sun, UserPlus } from "lucide-react";
import type { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import type { WithdrawWalletPayload } from "@/lib/types";
import { cn } from "@/lib/utils";
import { type ExploreTheme, type ExploreThemePreference, exploreThemeOptions, formatWalletAmount } from "./explore-config";

type ExploreMoreSheetProps = {
  exploreThemePreference: ExploreThemePreference;
  hasClientAccessToken: boolean;
  isLightTheme: boolean;
  isLoggingOut: boolean;
  isWalletBalanceLoading: boolean;
  mobileMoreOpen: boolean;
  onLogout: () => void;
  setExploreTheme: Dispatch<SetStateAction<ExploreTheme>>;
  setExploreThemePreference: Dispatch<SetStateAction<ExploreThemePreference>>;
  setMobileMoreOpen: Dispatch<SetStateAction<boolean>>;
  setThemeSettingsOpen: Dispatch<SetStateAction<boolean>>;
  t: ReturnType<typeof useTranslations>;
  themeSettingsOpen: boolean;
  walletBalance: WithdrawWalletPayload | null;
  walletBalanceError: string | null;
};

export function ExploreMoreSheet({
  exploreThemePreference,
  hasClientAccessToken,
  isLightTheme,
  isLoggingOut,
  isWalletBalanceLoading,
  mobileMoreOpen,
  onLogout,
  setExploreTheme,
  setExploreThemePreference,
  setMobileMoreOpen,
  setThemeSettingsOpen,
  t,
  themeSettingsOpen,
  walletBalance,
  walletBalanceError,
}: ExploreMoreSheetProps) {
  return (
    <>
        {mobileMoreOpen ? (
          <div
            className={cn(
              "fixed inset-0 z-50",
              isLightTheme ? "bg-black/25" : "bg-black/55",
            )}
            onClick={() => setMobileMoreOpen(false)}
          >
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="explore-more-title"
              className="absolute inset-y-0 right-0 flex min-h-0 w-[68vw] min-w-[216px] max-w-[300px] flex-col border-l border-[var(--explore-border)] bg-[var(--explore-surface)] px-4 pb-[calc(1rem+env(safe-area-inset-bottom))] pt-[calc(1rem+env(safe-area-inset-top))] shadow-2xl"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex h-11 items-center justify-between gap-2">
                <h2 id="explore-more-title" className="min-w-0 truncate text-lg font-bold text-[var(--explore-strong)]">{t("nav.more")}</h2>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={t("profile.edit.cancel")}
                  className="size-9 text-[var(--explore-muted)] hover:bg-[var(--explore-surface-soft)] hover:text-[var(--explore-strong)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <Menu className="size-5" />
                </Button>
              </div>
              <div className="mt-5 flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto overscroll-contain pr-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
                <Link
                  href="/wallet"
                  className="grid min-w-0 gap-1 rounded-2xl border border-[var(--explore-border)] bg-[var(--explore-surface-soft)] px-4 py-3 text-left active:bg-[var(--explore-control-hover)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <span className="flex items-center gap-2 text-xs font-semibold text-[var(--explore-subtle)]">
                    <Coins className="size-4 text-[#f7d36b]" />
                    {t("moreMenu.wallet.title")}
                  </span>
                  <span className="min-w-0 truncate text-2xl font-black leading-tight text-[var(--explore-strong)]">
                    {hasClientAccessToken
                      ? isWalletBalanceLoading
                        ? t("moreMenu.wallet.loading")
                        : walletBalanceError
                          ? t("moreMenu.wallet.error")
                          : walletBalance
                            ? t("moreMenu.wallet.amount", { amount: formatWalletAmount(walletBalance.cash_balance) })
                            : t("moreMenu.wallet.loading")
                      : t("moreMenu.wallet.login")}
                  </span>
                  <span className="min-w-0 truncate text-xs font-medium text-[var(--explore-subtle)]">
                    {walletBalanceError
                      ? walletBalanceError
                      : t("moreMenu.wallet.detail")}
                  </span>
                </Link>
                <Link
                  href="/invite"
                  className="flex h-12 min-w-0 items-center gap-3 rounded-2xl bg-[var(--explore-surface-soft)] px-4 text-sm font-semibold text-[var(--explore-text)] transition-colors active:bg-[var(--explore-control-hover)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <UserPlus className="size-5 shrink-0 text-[var(--explore-muted)]" />
                  <span className="min-w-0 truncate">{t("moreMenu.invite")}</span>
                </Link>
                <Link
                  href="/verification"
                  className="flex h-12 min-w-0 items-center gap-3 rounded-2xl bg-[var(--explore-surface-soft)] px-4 text-sm font-semibold text-[var(--explore-text)] transition-colors active:bg-[var(--explore-control-hover)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <BadgeCheck className="size-5 shrink-0 text-[var(--explore-muted)]" />
                  <span className="min-w-0 truncate">{t("moreMenu.verification")}</span>
                </Link>
                <Link
                  href="/download"
                  className="flex h-12 min-w-0 items-center gap-3 rounded-2xl bg-[var(--explore-surface-soft)] px-4 text-sm font-semibold text-[var(--explore-text)] transition-colors active:bg-[var(--explore-control-hover)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <Download className="size-5 shrink-0 text-[var(--explore-muted)]" />
                  <span className="min-w-0 truncate">{t("moreMenu.appDownload")}</span>
                </Link>
                <div className="overflow-hidden rounded-2xl bg-[var(--explore-surface-soft)]">
                  <button
                    type="button"
                    aria-expanded={themeSettingsOpen}
                    className="flex h-12 w-full min-w-0 items-center gap-3 px-4 text-left text-sm font-semibold text-[var(--explore-text)] transition-colors active:bg-[var(--explore-control-hover)]"
                    onClick={() => setThemeSettingsOpen((open) => !open)}
                  >
                    <Palette className="size-5 shrink-0 text-[var(--explore-muted)]" />
                    <span className="min-w-0 flex-1 truncate">{t("moreMenu.theme.title")}</span>
                    <span className="shrink-0 text-xs font-semibold text-[var(--explore-subtle)]">
                      {t(`moreMenu.theme.options.${exploreThemePreference}`)}
                    </span>
                    <ChevronDown
                      className={cn(
                        "size-4 shrink-0 text-[var(--explore-muted)] transition-transform",
                        themeSettingsOpen && "rotate-180",
                      )}
                    />
                  </button>
                  {themeSettingsOpen ? (
                    <div className="grid gap-1 border-t border-[var(--explore-border)] px-2 py-2">
                      {exploreThemeOptions.map((option) => {
                        const selected = exploreThemePreference === option.value;
                        const Icon =
                          option.value === "system" ? Monitor : option.value === "light" ? Sun : Moon;

                        return (
                          <button
                            key={option.value}
                            type="button"
                            aria-pressed={selected}
                            className={cn(
                              "flex h-10 w-full min-w-0 items-center gap-3 rounded-xl px-3 text-left text-sm font-semibold transition-colors active:bg-[var(--explore-control-hover)]",
                              selected
                                ? "bg-[var(--explore-control)] text-[var(--explore-strong)]"
                                : "text-[var(--explore-muted)]",
                            )}
                            onClick={() => {
                              setExploreThemePreference(option.value);
                              if (option.value === "system") {
                                const prefersLight = window.matchMedia?.("(prefers-color-scheme: light)").matches;
                                setExploreTheme(prefersLight ? "light" : "dark");
                              } else {
                                setExploreTheme(option.value);
                              }
                            }}
                          >
                            <Icon
                              className={cn(
                                "size-4 shrink-0",
                                option.value === "light" && "text-[#e2a500]",
                                option.value === "dark" && "text-[#9eb6ff]",
                                option.value === "system" && "text-[var(--explore-muted)]",
                              )}
                            />
                            <span className="min-w-0 flex-1 truncate">{t(`moreMenu.theme.options.${option.value}`)}</span>
                            {selected ? <Check className="size-4 shrink-0 text-primary" /> : null}
                          </button>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
                <Link
                  href="/settings"
                  className="flex h-12 min-w-0 items-center gap-3 rounded-2xl bg-[var(--explore-surface-soft)] px-4 text-sm font-semibold text-[var(--explore-text)] transition-colors active:bg-[var(--explore-control-hover)]"
                  onClick={() => setMobileMoreOpen(false)}
                >
                  <Settings className="size-5 shrink-0 text-[var(--explore-muted)]" />
                  <span className="min-w-0 flex-1 truncate">{t("moreMenu.systemSettings")}</span>
                  <ChevronRight className="size-4 shrink-0 text-[var(--explore-muted)]" />
                </Link>
                {hasClientAccessToken ? (
                  <button
                    type="button"
                    aria-busy={isLoggingOut}
                    disabled={isLoggingOut}
                    className="flex h-12 min-w-0 items-center gap-3 rounded-2xl bg-[var(--explore-surface-soft)] px-4 text-left text-sm font-semibold text-primary transition-colors active:bg-[var(--explore-control-hover)] disabled:cursor-wait disabled:opacity-60"
                    onClick={onLogout}
                  >
                    {isLoggingOut ? (
                      <LoaderCircle className="size-5 shrink-0 motion-safe:animate-spin" />
                    ) : (
                      <LogOut className="size-5 shrink-0" />
                    )}
                    <span className="min-w-0 truncate">
                      {t(isLoggingOut ? "nav.loggingOut" : "nav.logout")}
                    </span>
                  </button>
                ) : null}
              </div>
            </div>
          </div>
        ) : null}


    </>
  );
}
