"use client";

import { useEffect, useId, useRef, useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { Check, ChevronDown, Languages } from "lucide-react";
import { useTranslations } from "next-intl";
import { useLocaleContext } from "@/components/providers/locale-provider";
import type { AppLocale } from "@/i18n/locales";
import { cn } from "@/lib/utils";

const localeBadges: Record<AppLocale, string> = {
  en: "EN",
  "zh-CN": "简",
  "zh-TW": "繁",
  vi: "VI",
  ja: "日",
  ko: "한",
};

type LanguageSwitcherProps = {
  className?: string;
  compact?: boolean;
  fullWidth?: boolean;
  tone?: "dark" | "light";
};

export function LanguageSwitcher({
  className,
  compact = false,
  fullWidth = false,
  tone = "dark",
}: LanguageSwitcherProps) {
  const router = useRouter();
  const t = useTranslations("locale");
  const { locale, locales, setLocale } = useLocaleContext();
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();
  const rootRef = useRef<HTMLDivElement>(null);
  const optionsId = useId();

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, []);

  function selectLocale(nextLocale: AppLocale) {
    setOpen(false);

    if (nextLocale === locale) {
      return;
    }

    void setLocale(nextLocale).then(() => {
      startTransition(() => {
        router.refresh();
      });
    });
  }

  const currentLabel = t(`options.${locale}`);

  return (
    <div
      ref={rootRef}
      className={cn("relative inline-flex", fullWidth && "w-full", className)}
    >
      <button
        type="button"
        aria-label={t("trigger")}
        aria-controls={optionsId}
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        disabled={isPending}
        className={cn(
          "inline-flex h-10 items-center justify-center gap-2 rounded-full border px-3 text-sm font-semibold outline-none transition-colors focus-visible:ring-2 focus-visible:ring-primary disabled:opacity-60",
          tone === "dark"
            ? "border-white/10 bg-white/[0.06] text-white/75 hover:bg-white/[0.1] hover:text-white"
            : "border-[#e3e3e6] bg-white text-[#4c4c54] hover:bg-[#f5f5f6]",
          compact && "size-10 px-0",
          fullWidth && "w-full justify-start",
        )}
      >
        <Languages className="size-4 shrink-0" />
        {compact ? (
          <span className="sr-only">{currentLabel}</span>
        ) : (
          <span className="min-w-0 truncate">{currentLabel}</span>
        )}
        {!compact ? <ChevronDown className="ml-auto size-4 shrink-0" /> : null}
      </button>

      {open ? (
        <div
          id={optionsId}
          className={cn(
            "absolute right-0 top-full z-50 mt-2 min-w-[184px] rounded-2xl border p-1 shadow-xl",
            tone === "dark"
              ? "border-white/10 bg-[#202024] text-white shadow-black/30"
              : "border-[#eeeeef] bg-white text-[#25252b] shadow-black/10",
            fullWidth && "left-0 right-auto w-full min-w-0",
          )}
        >
          <p
            className={cn(
              "px-3 py-2 text-xs font-medium",
              tone === "dark" ? "text-white/45" : "text-[#8a8a91]",
            )}
          >
            {t("label")}
          </p>
          {locales.map((option) => {
            const active = option === locale;

            return (
              <button
                key={option}
                type="button"
                onClick={() => selectLocale(option)}
                className={cn(
                  "flex h-10 w-full items-center gap-3 rounded-xl px-3 text-left text-sm font-medium transition-colors",
                  tone === "dark"
                    ? "text-white/72 hover:bg-white/[0.07] hover:text-white"
                    : "text-[#4c4c54] hover:bg-[#f5f5f6] hover:text-[#1f1f24]",
                  active &&
                    (tone === "dark"
                      ? "bg-white/[0.08] text-white"
                      : "bg-[#f5f5f6] text-[#1f1f24]"),
                )}
              >
                <span
                  className={cn(
                    "flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold",
                    tone === "dark"
                      ? "bg-white/[0.08] text-white"
                      : "bg-[#eeeeef] text-[#33333a]",
                  )}
                >
                  {localeBadges[option]}
                </span>
                <span className="min-w-0 flex-1 truncate">
                  {t(`options.${option}`)}
                </span>
                {active ? <Check className="size-4 shrink-0 text-primary" /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
