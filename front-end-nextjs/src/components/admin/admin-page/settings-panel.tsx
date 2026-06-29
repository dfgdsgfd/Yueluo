"use client";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  useTranslations
} from "next-intl";
import {
  Database,
  ImageIcon,
  Save,
  Settings,
  SlidersHorizontal,
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
import {
  achievementTriggerOptions
} from "./types";
import {
  HeaderCard,
  Panel
} from "./layout-widgets";
import {
  EmptyBlock,
  KeyValueGrid,
  LoadingBlock
} from "./resource-editor";
import {
  errorMessage,
  parseJsonLike,
  selectOptionLabel,
  truthy
} from "./helpers";
import {
  ToggleSwitch
} from "./form-fields";
import { LOCALES } from "@/i18n/locales";

export type SettingMeta = {
  hint?: string;
  hintKey?: string;
  label?: string;
  labelKey?: string;
  localized?: boolean;
  type?: string;
  value?: unknown;
};

const imageSettingKeys = new Set([
  "image_webp_enabled",
  "image_webp_quality",
  "image_avatar_webp_quality",
  "image_webp_method",
  "image_webp_alpha_quality",
  "image_webp_lossless",
  "image_libvips_enabled",
  "image_max_width",
  "image_max_height",
  "image_processing_concurrency",
  "image_post_max_count",
  "image_archive_enabled",
  "image_archive_threshold",
  "image_protection_enabled",
  "image_protection_notice_enabled",
  "image_select_all_enabled",
  "paid_content_balance_enabled",
  "paid_content_points_enabled",
  "paid_content_balance_max_price",
  "paid_content_points_max_price",
]);

const appDownloadSettingKeys = new Set([
  "app_download_android_enabled",
  "app_download_android_name",
  "app_download_android_version_name",
  "app_download_android_version_code",
  "app_download_android_download_url",
  "app_download_android_size_label",
  "app_download_android_size_bytes",
  "app_download_android_package_name",
  "app_download_android_release_notes",
  "app_download_android_fast_enabled",
  "app_download_android_fast_name",
  "app_download_android_fast_version_name",
  "app_download_android_fast_version_code",
  "app_download_android_fast_download_url",
  "app_download_android_fast_size_label",
  "app_download_android_fast_size_bytes",
  "app_download_android_fast_package_name",
  "app_download_android_fast_release_notes",
  "app_download_ios_enabled",
  "app_download_ios_name",
  "app_download_ios_version_name",
  "app_download_ios_version_code",
  "app_download_ios_download_url",
  "app_download_ios_size_label",
  "app_download_ios_size_bytes",
  "app_download_ios_bundle_id",
  "app_download_ios_release_notes",
]);

const oauthSettingKeys = new Set([
  "oauth2_app_callback_urls",
]);

const fileRecycleSettingKeys = new Set([
  "file_recycle_retention_days",
  "file_recycle_cleanup_interval_hours",
]);

function appDownloadSizeBytesSetting(key: string) {
  return key.startsWith("app_download_") && key.endsWith("_size_bytes");
}

function sizeBytesToMBInput(value: unknown) {
  const bytes = Number(value ?? 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return "0";
  const mb = bytes / (1024 * 1024);
  return Number(mb.toFixed(2)).toString();
}

function sizeMBInputToBytes(value: string) {
  const mb = Number(value.trim());
  if (!Number.isFinite(mb) || mb <= 0) return 0;
  return Math.round(mb * 1024 * 1024);
}

const hiddenWatermarkSettingKeys = new Set([
  "hidden_watermark_enabled",
  "hidden_watermark_protected_only",
  "hidden_watermark_engine",
  "hidden_watermark_include_uid",
  "hidden_watermark_include_user_id",
  "hidden_watermark_include_username",
  "hidden_watermark_include_time",
  "hidden_watermark_include_file_hash",
  "hidden_watermark_include_custom",
  "hidden_watermark_custom_text",
  "hidden_watermark_profile",
  "hidden_watermark_block_width",
  "hidden_watermark_block_height",
  "hidden_watermark_coefficient_mode",
  "hidden_watermark_d1",
  "hidden_watermark_d2",
  "hidden_watermark_ecc_mode",
  "hidden_watermark_golay_seed",
  "hidden_watermark_remote_password_wm",
  "hidden_watermark_remote_password_img",
  "hidden_watermark_remote_engine",
  "hidden_watermark_remote_profile",
  "hidden_watermark_remote_custom_d1",
  "hidden_watermark_remote_custom_d2",
  "hidden_watermark_remote_timeout_seconds",
  "hidden_watermark_remote_operation_timeout_seconds",
  "image_protection_max_dimension",
  "image_protection_output_mode",
  "image_protection_webp_quality",
  "hidden_watermark_extract_all_users",
  "hidden_watermark_extract_user_ids",
  "hidden_watermark_extract_usernames",
]);

export function isOnboardingSettingKey(key: string) {
  return key.startsWith("onboarding_");
}

const imageNumberBounds: Record<string, { max: number; min: number }> = {
  post_content_max_length: { min: 1, max: 1000000 },
  image_webp_quality: { min: 1, max: 100 },
  image_avatar_webp_quality: { min: 1, max: 100 },
  image_webp_method: { min: 0, max: 6 },
  image_webp_alpha_quality: { min: 0, max: 100 },
  image_max_width: { min: 0, max: 16384 },
  image_max_height: { min: 0, max: 16384 },
  image_processing_concurrency: { min: 1, max: 8 },
  image_post_max_count: { min: 1, max: 500 },
  image_archive_threshold: { min: 1, max: 500 },
  paid_content_points_max_price: { min: 1, max: 1000000 },
  paid_content_balance_max_price: { min: 1, max: 1000000 },
  file_recycle_retention_days: { min: 1, max: 3650 },
  file_recycle_cleanup_interval_hours: { min: 1, max: 720 },
};

export function settingsValueLooksStructured(value: unknown) {
  if (Array.isArray(value)) return true;
  if (value && typeof value === "object") return true;
  if (typeof value !== "string") return false;
  const trimmed = value.trim();
  return (trimmed.startsWith("{") && trimmed.endsWith("}")) || (trimmed.startsWith("[") && trimmed.endsWith("]"));
}


export function settingsDraftText(value: unknown) {
  if (typeof value === "string") return value;
  if (value && typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value ?? "");
}


export function settingsDateTimeLocalValue(value: unknown) {
  const text = String(value ?? "").trim().replace(" ", "T");
  if (!text) return "";
  const match = text.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2})/);
  return match?.[1] ?? "";
}


export function settingsDateTimePayload(value: unknown) {
  const text = String(value ?? "").trim();
  if (!text) return "";
  if (/[zZ]$|[+-]\d{2}:?\d{2}$/.test(text)) return text;
  return text.length === 16 ? `${text}:00Z` : text;
}


export function settingsDraftPayload(settings: Record<string, SettingMeta>, draft: Record<string, unknown>) {
  return Object.fromEntries(Object.entries(draft).map(([key, value]) => {
    if (appDownloadSizeBytesSetting(key)) {
      return [key, sizeMBInputToBytes(String(value ?? ""))];
    }
    if (settings[key]?.type === "number") {
      const numeric = Number(value);
      return [key, Number.isFinite(numeric) ? numeric : value];
    }
    if (settings[key]?.type === "datetime") {
      return [key, settingsDateTimePayload(value)];
    }
    if (typeof value === "string" && settingsValueLooksStructured(value)) {
      const parsed = parseJsonLike(value);
      return [key, parsed ?? value];
    }
    return [key, value];
  }));
}

export function LocalizedSettingEditor({
  label,
  onChange,
  structured = false,
  value,
}: {
  label: string;
  onChange: (value: Record<string, unknown>) => void;
  structured?: boolean;
  value: unknown;
}) {
  const localized = value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};
  return (
    <fieldset className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:col-span-2">
      <legend className="px-1 text-sm font-semibold text-[#343944]">{label}</legend>
      <div className="mt-2 grid gap-3 md:grid-cols-2">
        {LOCALES.map((locale) => (
          <label key={locale} className="min-w-0">
            <span className="mb-1 block text-xs font-semibold text-[#737986]">{locale}</span>
            {structured ? (
              <textarea
                value={settingsDraftText(localized[locale])}
                onChange={(event) => onChange({ ...localized, [locale]: event.target.value })}
                className="min-h-[104px] w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
                spellCheck={false}
              />
            ) : (
              <textarea
                value={String(localized[locale] ?? "")}
                onChange={(event) => onChange({ ...localized, [locale]: event.target.value })}
                className="min-h-20 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
              />
            )}
          </label>
        ))}
      </div>
    </fieldset>
  );
}


export function achievementTriggerLabel(value: string) {
  return selectOptionLabel(achievementTriggerOptions, value) ?? (value || "成就");
}


export function SettingsPanel({ token }: { token: string }) {
  const t = useTranslations("adminPortal.settingsPanel");
  const [settings, setSettings] = useState<Record<string, SettingMeta>>({});
  const [rawSettings, setRawSettings] = useState<Record<string, unknown>>({});
  const [draft, setDraft] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [data, rawData] = await Promise.all([
        adminRequest<{ settings?: Record<string, SettingMeta>; raw?: Record<string, unknown> }>("/api/admin/system-settings", { method: "GET", token }),
        adminRequest<Record<string, unknown>>("/api/admin/settings", { method: "GET", token }),
      ]);
      const nextSettings = Object.fromEntries(
        Object.entries(data.settings ?? {}).filter(([key]) => !isOnboardingSettingKey(key)),
      );
      const nextRawSettings = rawData && typeof rawData === "object" ? rawData : data.raw ?? {};
      setSettings(nextSettings);
      setRawSettings(Object.fromEntries(Object.entries(nextRawSettings).filter(([key]) => !isOnboardingSettingKey(key))));
      setDraft(Object.fromEntries(Object.entries(nextSettings).map(([key, value]) => [
        key,
        appDownloadSizeBytesSetting(key) ? sizeBytesToMBInput(value.value) : value.value,
      ])));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  async function save(keys: string[]) {
    setSaving(true);
    try {
      const selectedKeys = new Set(keys);
      const selectedDraft = Object.fromEntries(
        Object.entries(draft).filter(([key]) => selectedKeys.has(key)),
      );
      await adminRequest("/api/admin/system-settings", {
        method: "PUT",
        token,
        body: JSON.stringify(settingsDraftPayload(settings, selectedDraft)),
      });
      toast.success(t("saved"));
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  const entries = Object.entries(settings).filter(([key]) => !hiddenWatermarkSettingKeys.has(key));
  const imageEntries = entries.filter(([key]) => imageSettingKeys.has(key));
  const appDownloadEntries = entries.filter(([key]) => appDownloadSettingKeys.has(key));
  const oauthEntries = entries.filter(([key]) => oauthSettingKeys.has(key));
  const fileRecycleEntries = entries.filter(([key]) => fileRecycleSettingKeys.has(key));
  const generalEntries = entries.filter(
    ([key]) =>
      !imageSettingKeys.has(key) &&
      !appDownloadSettingKeys.has(key) &&
      !oauthSettingKeys.has(key) &&
      !fileRecycleSettingKeys.has(key) &&
      !key.startsWith("image_") &&
      !key.startsWith("hidden_watermark_") &&
      !key.startsWith("oauth2_") &&
      !key.startsWith("file_recycle_") &&
      !key.startsWith("redis_"),
  );
  const rawEntries = Object.entries(rawSettings).filter(
    ([key, value]) =>
      !hiddenWatermarkSettingKeys.has(key) &&
      !key.startsWith("hidden_watermark_") &&
      !key.startsWith("redis_") &&
      value !== undefined &&
      value !== null &&
      value !== "",
  );
  const settingLabel = (key: string, meta: SettingMeta) => {
    if (meta.labelKey && t.has(meta.labelKey)) return t(meta.labelKey);
    switch (key) {
      case "image_webp_enabled":
        return t("fields.imageWebpEnabled");
      case "image_webp_quality":
        return t("fields.imageWebpQuality");
      case "image_avatar_webp_quality":
        return t("fields.imageAvatarWebpQuality");
      case "image_webp_method":
        return t("fields.imageWebpMethod");
      case "image_webp_alpha_quality":
        return t("fields.imageWebpAlphaQuality");
      case "image_webp_lossless":
        return t("fields.imageWebpLossless");
      case "image_libvips_enabled":
        return t("fields.imageLibvipsEnabled");
      case "image_max_width":
        return t("fields.imageMaxWidth");
      case "image_max_height":
        return t("fields.imageMaxHeight");
      case "image_processing_concurrency":
        return t("fields.imageProcessingConcurrency");
      case "image_post_max_count":
        return t("fields.imagePostMaxCount");
      case "image_archive_enabled":
        return t("fields.imageArchiveEnabled");
      case "image_archive_threshold":
        return t("fields.imageArchiveThreshold");
      case "image_protection_enabled":
        return t("fields.imageProtectionEnabled");
      case "image_protection_notice_enabled":
        return t("fields.imageProtectionNoticeEnabled");
      case "image_select_all_enabled":
        return t("fields.imageSelectAllEnabled");
      case "paid_content_balance_enabled":
        return t("fields.paidContentBalanceEnabled");
      case "paid_content_points_enabled":
        return t("fields.paidContentPointsEnabled");
      case "paid_content_balance_max_price":
        return t("fields.paidContentBalanceMaxPrice");
      case "paid_content_points_max_price":
        return t("fields.paidContentPointsMaxPrice");
      case "video_center_enabled":
        return t("fields.videoCenterEnabled");
      case "video_center_homepage_enabled":
        return t("fields.videoCenterHomepageEnabled");
      case "video_center_recommend_limit":
        return t("fields.videoCenterRecommendLimit");
      case "video_center_account_cutoff":
        return t("fields.videoCenterAccountCutoff");
      default:
        return meta.label ?? key;
    }
  };
  const settingHint = (key: string, meta: SettingMeta) => {
    if (meta.hintKey && t.has(meta.hintKey)) return t(meta.hintKey);
    switch (key) {
      case "image_max_width":
      case "image_max_height":
        return t("hints.imageMaxDimension");
      case "image_processing_concurrency":
        return t("hints.imageProcessingConcurrency");
      case "image_post_max_count":
        return t("hints.imagePostMaxCount");
      case "image_archive_enabled":
        return t("hints.imageArchiveEnabled");
      case "image_archive_threshold":
        return t("hints.imageArchiveThreshold");
      case "image_libvips_enabled":
        return t("hints.imageLibvipsEnabled");
      case "image_protection_enabled":
        return t("hints.imageProtectionEnabled");
      case "image_protection_notice_enabled":
        return t("hints.imageProtectionNoticeEnabled");
      case "image_select_all_enabled":
        return t("hints.imageSelectAllEnabled");
      case "paid_content_balance_enabled":
      case "paid_content_points_enabled":
        return t("hints.paidContentPaymentMethod");
      case "paid_content_balance_max_price":
      case "paid_content_points_max_price":
        return t("hints.paidContentPriceLimit");
      case "video_center_account_cutoff":
        return t("hints.videoCenterAccountCutoff");
      default:
        return meta.hint ?? "";
    }
  };
  const renderSetting = ([key, meta]: [string, SettingMeta]) => (
    meta.localized ? (
      <LocalizedSettingEditor
        key={key}
        label={settingLabel(key, meta)}
        structured={key === "onboarding_interest_options"}
        value={draft[key]}
        onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
      />
    ) : (
    <label key={key} className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{settingLabel(key, meta)}</span>
      {key === "post_resource_section_position" ? (
        <select
          value={String(draft[key] ?? "before_content")}
          onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
          className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
        >
          <option value="before_content">{t("systemSettings.post_resource_section_position.before")}</option>
          <option value="after_content">{t("systemSettings.post_resource_section_position.after")}</option>
        </select>
      ) : meta.type === "boolean" ? (
        <ToggleSwitch
          value={truthy(draft[key])}
          onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
          onLabel={t("on")}
          offLabel={t("off")}
        />
      ) : meta.type === "datetime" ? (
        <input
          value={settingsDateTimeLocalValue(draft[key])}
          onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
          className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
          type="datetime-local"
        />
      ) : meta.type === "textarea" ? (
        <textarea
          value={settingsDraftText(draft[key])}
          onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
          className="min-h-[110px] w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
          spellCheck={false}
        />
      ) : settingsValueLooksStructured(draft[key]) ? (
        <textarea
          value={settingsDraftText(draft[key])}
          onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
          className="min-h-[110px] w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
          spellCheck={false}
        />
      ) : (
        <input
          value={String(draft[key] ?? "")}
          onChange={(event) => setDraft((current) => ({
            ...current,
            [key]: appDownloadSizeBytesSetting(key)
              ? event.target.value
              : meta.type === "number" ? Number(event.target.value) : event.target.value,
          }))}
          className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
          type={meta.type === "number" ? "number" : "text"}
          step={appDownloadSizeBytesSetting(key) ? "0.01" : undefined}
          min={appDownloadSizeBytesSetting(key) ? 0 : imageNumberBounds[key]?.min}
          max={imageNumberBounds[key]?.max}
        />
      )}
      <span className="mt-1 block break-all text-xs text-[#8b919e]">{key}</span>
      {settingHint(key, meta) ? (
        <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{settingHint(key, meta)}</span>
      ) : null}
    </label>
    )
  );

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Settings} title={t("title")} description={t("description")} tone="slate" />
      <Panel
        title={t("configTitle")}
        icon={SlidersHorizontal}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save(generalEntries.map(([key]) => key))} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : generalEntries.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {generalEntries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={Settings} label={t("emptySettings")} />
        )}
      </Panel>
      <Panel
        title={t("fileRecycleTitle")}
        icon={Trash2}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save(fileRecycleEntries.map(([key]) => key))} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : fileRecycleEntries.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {fileRecycleEntries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={Trash2} label={t("emptySettings")} />
        )}
      </Panel>
      <Panel
        title={t("authTitle")}
        icon={SlidersHorizontal}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save(oauthEntries.map(([key]) => key))} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : oauthEntries.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {oauthEntries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={Settings} label={t("emptySettings")} />
        )}
      </Panel>
      <Panel
        title={t("appDownloadTitle")}
        icon={SlidersHorizontal}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save(appDownloadEntries.map(([key]) => key))} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : appDownloadEntries.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {appDownloadEntries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={Settings} label={t("emptySettings")} />
        )}
      </Panel>
      <Panel
        title={t("imageTitle")}
        icon={ImageIcon}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save(imageEntries.map(([key]) => key))} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : imageEntries.length ? (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {imageEntries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={ImageIcon} label={t("emptyImageSettings")} />
        )}
      </Panel>
      <Panel title={t("rawTitle")} icon={Database}>
        {loading ? (
          <LoadingBlock label={t("loadingRaw")} />
        ) : rawEntries.length ? (
          <KeyValueGrid entries={rawEntries} />
        ) : (
          <EmptyBlock icon={Database} label={t("emptyRaw")} />
        )}
      </Panel>
    </div>
  );
}
