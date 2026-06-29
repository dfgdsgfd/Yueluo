"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  BadgeCheck,
  Check,
  ChevronRight,
  Eye,
  EyeOff,
  Loader2,
  MessageCircle,
  Save,
  Shield,
  SlidersHorizontal,
  Wifi,
  type LucideIcon,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { LanguageSwitcher } from "@/components/language-switcher";
import { Button } from "@/components/ui/button";
import {
  getStoredAccessToken,
  getUserPrivacySettings,
  updateUserPrivacySettings,
} from "@/lib/api";
import type { UserPrivacySettingsPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

type PrivacyBooleanKey = Exclude<keyof UserPrivacySettingsPayload, "privacy_custom_fields">;

const privacyItems = [
  {
    key: "privacy_birthday",
    translationKey: "birthday",
  },
  {
    key: "privacy_age",
    translationKey: "age",
  },
  {
    key: "privacy_zodiac",
    translationKey: "zodiac",
  },
  {
    key: "privacy_mbti",
    translationKey: "mbti",
  },
] as const satisfies ReadonlyArray<{
  key: PrivacyBooleanKey;
  translationKey: string;
}>;

const defaultPrivacySettings: UserPrivacySettingsPayload = {
  privacy_age: false,
  privacy_birthday: false,
  privacy_custom_fields: {},
  privacy_mbti: false,
  privacy_zodiac: false,
  ai_auto_comment_enabled: true,
};

export function SystemSettingsPage() {
  const t = useTranslations("settingsPage");
  const [authChecked, setAuthChecked] = useState(false);
  const [authToken, setAuthToken] = useState<string | null>(null);
  const [settings, setSettings] = useState<UserPrivacySettingsPayload>(defaultPrivacySettings);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const visibleCount = useMemo(
    () => privacyItems.filter((item) => settings[item.key]).length,
    [settings],
  );

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) {
        return;
      }

      const token = getStoredAccessToken();
      setAuthToken(token);
      setAuthChecked(true);
      if (!token) {
        return;
      }

      setIsLoading(true);
      getUserPrivacySettings()
        .then((payload) => {
          if (!cancelled) {
            setSettings(payload);
          }
        })
        .catch((error) => {
          if (!cancelled) {
            toast.error(error instanceof Error ? error.message : t("errors.load"));
          }
        })
        .finally(() => {
          if (!cancelled) {
            setIsLoading(false);
          }
        });
    });

    return () => {
      cancelled = true;
    };
  }, [t]);

  async function savePrivacySettings() {
    setIsSaving(true);
    try {
      const updated = await updateUserPrivacySettings(settings);
      setSettings(updated);
      toast.success(t("saved"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("errors.save"));
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <main className="theme-adaptive min-h-dvh bg-[#121212] text-white">
      <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col">
        <header className="sticky top-0 z-20 flex h-14 items-center gap-2 border-b border-white/[0.07] bg-[#121212]/95 px-3 backdrop-blur">
          <Button
            asChild
            variant="ghost"
            size="icon"
            className="size-10 text-white/78 hover:bg-white/[0.06] hover:text-white"
          >
            <Link href="/">
              <ArrowLeft className="size-5" />
              <span className="sr-only">{t("back")}</span>
            </Link>
          </Button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-lg font-bold">{t("title")}</h1>
          </div>
        </header>

        <section className="flex-1 space-y-4 overflow-y-auto px-4 pb-[calc(24px+env(safe-area-inset-bottom))] pt-4">
          <div className="rounded-[16px] border border-white/[0.08] bg-white/[0.06] p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <h2 className="text-base font-bold">{t("language.title")}</h2>
                <p className="mt-1 text-sm leading-5 text-white/52">{t("language.description")}</p>
              </div>
            </div>
            <LanguageSwitcher className="mt-4" fullWidth tone="dark" />
          </div>

          <div className="rounded-[16px] border border-white/[0.08] bg-white/[0.06] p-4">
            <div className="flex items-center gap-3">
              <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-white/[0.07] text-white/70">
                <Shield className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <h2 className="text-base font-bold">{t("account.title")}</h2>
                <p className="mt-1 text-sm leading-5 text-white/52">{t("account.description")}</p>
              </div>
            </div>
            <div className="mt-4 overflow-hidden rounded-[8px] bg-white/[0.04]">
              <SettingsLinkRow
                href="/verification"
                icon={BadgeCheck}
                title={t("account.verificationTitle")}
                description={t("account.verificationDescription")}
              />
              <SettingsLinkRow
                href="/network-diagnostics"
                icon={Wifi}
                title={t("account.networkDiagnosticsTitle")}
                description={t("account.networkDiagnosticsDescription")}
                separated
              />
            </div>
          </div>

          <div className="rounded-[16px] border border-white/[0.08] bg-white/[0.06] p-4">
            <div className="flex items-center gap-3">
              <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-primary/15 text-primary">
                <SlidersHorizontal className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <h2 className="text-base font-bold">{t("privacy.title")}</h2>
                <p className="mt-1 text-sm leading-5 text-white/52">
                  {t("privacy.description")}
                </p>
              </div>
            </div>
            <div className="mt-4 flex items-center gap-2 rounded-full bg-white/[0.06] px-3 py-2 text-xs font-semibold text-white/58">
              <Shield className="size-4" />
              {t("privacy.summary", { visible: visibleCount, total: privacyItems.length })}
            </div>
          </div>

          {!authChecked ? (
            <div className="flex min-h-[220px] items-center justify-center rounded-[16px] border border-white/[0.08] bg-white/[0.06] text-white/60">
              <Loader2 className="mr-2 size-5 animate-spin" />
              {t("checkingAuth")}
            </div>
          ) : !authToken ? (
            <div className="rounded-[16px] border border-white/[0.08] bg-white/[0.06] px-4 py-8 text-center">
              <p className="text-base font-bold">{t("loginRequired.title")}</p>
              <p className="mt-2 text-sm text-white/50">{t("loginRequired.description")}</p>
              <Button asChild className="mt-5 h-10 rounded-full bg-primary px-5 text-white">
                <Link href="/login">{t("loginRequired.login")}</Link>
              </Button>
            </div>
          ) : isLoading ? (
            <div className="flex min-h-[220px] items-center justify-center rounded-[16px] border border-white/[0.08] bg-white/[0.06] text-white/60">
              <Loader2 className="mr-2 size-5 animate-spin" />
              {t("loading")}
            </div>
          ) : (
            <>
              <div className="overflow-hidden rounded-[16px] border border-white/[0.08] bg-white/[0.06]">
                <PrivacySwitchRow
                  checked={settings.ai_auto_comment_enabled}
                  description={t("aiAutoComment.description")}
                  icon={MessageCircle}
                  title={t("aiAutoComment.title")}
                  onToggle={() =>
                    setSettings((current) => ({
                      ...current,
                      ai_auto_comment_enabled: !current.ai_auto_comment_enabled,
                    }))
                  }
                />
              </div>
              <div className="overflow-hidden rounded-[16px] border border-white/[0.08] bg-white/[0.06]">
                {privacyItems.map((item, index) => (
                  <PrivacySwitchRow
                    key={item.key}
                    checked={settings[item.key]}
                    description={t(`privacy.items.${item.translationKey}.description`)}
                    title={t(`privacy.items.${item.translationKey}.title`)}
                    separated={index > 0}
                    onToggle={() =>
                      setSettings((current) => ({
                        ...current,
                        [item.key]: !current[item.key],
                      }))
                    }
                  />
                ))}
              </div>
              <Button
                type="button"
                disabled={isSaving}
                onClick={() => void savePrivacySettings()}
                className="sticky bottom-4 h-12 w-full rounded-full bg-primary text-base font-bold text-white shadow-lg shadow-black/30 hover:bg-primary/90 disabled:opacity-60"
              >
                {isSaving ? <Loader2 className="size-5 animate-spin" /> : <Save className="size-5" />}
                {t("save")}
              </Button>
            </>
          )}
        </section>
      </div>
    </main>
  );
}

function SettingsLinkRow({
  description,
  href,
  icon: Icon,
  separated,
  title,
}: {
  description: string;
  href: string;
  icon: LucideIcon;
  separated?: boolean;
  title: string;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "flex min-w-0 items-center gap-3 px-4 py-3 text-left transition-colors active:bg-white/[0.08]",
        separated && "border-t border-white/[0.07]",
      )}
    >
      <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-white/[0.07] text-white/58">
        <Icon className="size-5" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-bold text-white">{title}</span>
        <span className="mt-1 block truncate text-xs text-white/42">{description}</span>
      </span>
      <ChevronRight className="size-4 shrink-0 text-white/38" />
    </Link>
  );
}

function PrivacySwitchRow({
  checked,
  description,
  icon: Icon,
  onToggle,
  separated,
  title,
}: {
  checked: boolean;
  description: string;
  icon?: LucideIcon;
  onToggle: () => void;
  separated?: boolean;
  title: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={onToggle}
      className={cn(
        "flex w-full items-center gap-3 px-4 py-4 text-left active:bg-white/[0.08]",
        separated && "border-t border-white/[0.07]",
      )}
    >
      <span
        className={cn(
          "flex size-10 shrink-0 items-center justify-center rounded-full",
          checked ? "bg-primary/15 text-primary" : "bg-white/[0.07] text-white/45",
        )}
      >
        {Icon ? <Icon className="size-5" /> : checked ? <Eye className="size-5" /> : <EyeOff className="size-5" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-bold text-white">{title}</span>
        <span className="mt-1 block text-xs leading-5 text-white/45">{description}</span>
      </span>
      <span
        className={cn(
          "flex h-7 w-12 shrink-0 items-center rounded-full border p-0.5 transition-colors",
          checked ? "border-primary/30 bg-primary" : "border-white/[0.08] bg-white/[0.12]",
        )}
      >
        <span
          className={cn(
            "flex size-6 items-center justify-center rounded-full bg-white text-[10px] text-primary shadow-sm transition-transform",
            checked ? "translate-x-5" : "translate-x-0",
          )}
        >
          {checked ? <Check className="size-3.5" /> : null}
        </span>
      </span>
    </button>
  );
}
