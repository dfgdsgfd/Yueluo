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
  Plus,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Sparkles,
  Trash2
} from "lucide-react";
import {
  toast
} from "sonner";
import {
  Button
} from "@/components/ui/button";
import { LOCALES } from "@/i18n/locales";
import {
  adminRequest
} from "@/lib/api";
import {
  errorMessage,
  truthy
} from "./helpers";
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
  ToggleSwitch
} from "./form-fields";
import {
  isOnboardingSettingKey,
  LocalizedSettingEditor,
  SettingMeta,
  settingsDateTimeLocalValue,
  settingsDraftPayload,
  settingsDraftText,
  settingsValueLooksStructured
} from "./settings-panel";

type OnboardingSettingsTranslator = ReturnType<typeof useTranslations>;

export function OnboardingSettingsPanel({ token }: { token: string }) {
  const t = useTranslations("adminPortal.onboardingSettingsPanel");
  const settingsT = useTranslations("adminPortal.settingsPanel");
  const [settings, setSettings] = useState<Record<string, SettingMeta>>({});
  const [draft, setDraft] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [resetResult, setResetResult] = useState<Record<string, unknown> | null>(null);
  const [singleUserId, setSingleUserId] = useState("");
  const [resettingSingle, setResettingSingle] = useState(false);
  const [singleResetResult, setSingleResetResult] = useState<Record<string, unknown> | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await adminRequest<{ settings?: Record<string, SettingMeta> }>("/api/admin/system-settings", { method: "GET", token });
      const nextSettings = Object.fromEntries(
        Object.entries(data.settings ?? {}).filter(([key]) => isOnboardingSettingKey(key)),
      );
      setSettings(nextSettings);
      setDraft(Object.fromEntries(Object.entries(nextSettings).map(([key, value]) => [key, value.value])));
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

  async function save() {
    setSaving(true);
    try {
      const payload = settingsDraftPayload(settings, draft);
      payload.onboarding_interest_options = normalizedLocalizedInterestOptions(draft.onboarding_interest_options);
      await adminRequest("/api/admin/system-settings", { method: "PUT", token, body: JSON.stringify(payload) });
      toast.success(t("saved"));
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function resetOnboarding() {
    if (!window.confirm(t("resetConfirm"))) return;
    setResetting(true);
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/reset-all-onboarding", { method: "POST", token });
      setResetResult(result);
      toast.success(t("resetDone"));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setResetting(false);
    }
  }

  async function resetSingleOnboarding() {
    const userId = singleUserId.trim();
    if (!userId) {
      toast.error(t("resetSingleMissing"));
      return;
    }
    if (!window.confirm(t("resetSingleConfirm", { userId }))) return;
    setResettingSingle(true);
    try {
      const result = await adminRequest<Record<string, unknown>>(`/api/admin/users/${encodeURIComponent(userId)}/reset-onboarding`, { method: "POST", token });
      setSingleResetResult(result);
      toast.success(t("resetSingleDone"));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setResettingSingle(false);
    }
  }

  const entries = Object.entries(settings);
  const sampleInterestOptions = [t("sampleInterests.bondage"), t("sampleInterests.handcuffs"), t("sampleInterests.legIrons")];
  const renderSetting = ([key, meta]: [string, SettingMeta]) => {
    if (key === "onboarding_interest_options") {
      if (meta.localized) {
        return (
          <LocalizedInterestOptionsEditor
            key={key}
            hint={settingHint(t, key, meta)}
            label={settingLabel(t, key, meta)}
            sampleOptions={sampleInterestOptions}
            settingKey={key}
            t={t}
            value={draft[key]}
            onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
          />
        );
      }
      return (
        <InterestOptionsEditor
          key={key}
          hint={settingHint(t, key, meta)}
          label={settingLabel(t, key, meta)}
          sampleOptions={sampleInterestOptions}
          settingKey={key}
          t={t}
          value={draft[key]}
          onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
        />
      );
    }
    if (meta.localized) {
      return (
        <LocalizedSettingEditor
          key={key}
          label={settingLabel(t, key, meta)}
          structured={key === "onboarding_interest_options"}
          value={draft[key]}
          onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
        />
      );
    }
    const multiline = meta.type === "textarea" || settingsValueLooksStructured(draft[key]);
    return (
      <label key={key} className={multiline ? "min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:col-span-2" : "min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3"}>
        <span className="mb-2 block text-sm font-semibold text-[#343944]">{settingLabel(t, key, meta)}</span>
        {meta.type === "boolean" ? (
          <ToggleSwitch
            value={truthy(draft[key])}
            onChange={(value) => setDraft((current) => ({ ...current, [key]: value }))}
            onLabel={settingsT("on")}
            offLabel={settingsT("off")}
          />
        ) : meta.type === "datetime" ? (
          <input
            value={settingsDateTimeLocalValue(draft[key])}
            onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
            className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
            type="datetime-local"
          />
        ) : multiline ? (
          <textarea
            value={settingsDraftText(draft[key])}
            onChange={(event) => setDraft((current) => ({ ...current, [key]: event.target.value }))}
            className="min-h-[128px] w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        ) : (
          <input
            value={String(draft[key] ?? "")}
            onChange={(event) => setDraft((current) => ({ ...current, [key]: meta.type === "number" ? Number(event.target.value) : event.target.value }))}
            className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
            type={meta.type === "number" ? "number" : "text"}
          />
        )}
        <span className="mt-1 block break-all text-xs text-[#8b919e]">{key}</span>
        {settingHint(t, key, meta) ? (
          <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{settingHint(t, key, meta)}</span>
        ) : null}
      </label>
    );
  };

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Sparkles} title={t("title")} description={t("description")} tone="blue" />
      <Panel
        title={t("configTitle")}
        icon={SlidersHorizontal}
        action={
          <Button type="button" disabled={saving || loading} onClick={() => void save()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            <Save className="size-4" />
            <span>{t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loadingSettings")} />
        ) : entries.length ? (
          <div className="grid gap-3 md:grid-cols-2">
            {entries.map(renderSetting)}
          </div>
        ) : (
          <EmptyBlock icon={Sparkles} label={t("emptySettings")} />
        )}
      </Panel>
      <Panel
        title={t("resetTitle")}
        icon={RotateCcw}
        action={
          <Button type="button" variant="outline" disabled={resetting} onClick={() => void resetOnboarding()} className="h-9 rounded-lg border-black/[0.08] bg-white px-3 hover:bg-[#f6f7fb]">
            <RotateCcw className="size-4" />
            <span>{resetting ? t("resetting") : t("resetAction")}</span>
          </Button>
        }
      >
        <p className="text-sm leading-6 text-[#6b7280]">{t("resetDescription")}</p>
        {resetResult ? (
          <div className="mt-3">
            <KeyValueGrid entries={Object.entries(resetResult)} />
          </div>
        ) : null}
        <div className="mt-4 grid gap-3 border-t border-black/[0.06] pt-4">
          <div>
            <h3 className="text-sm font-semibold text-[#343944]">{t("resetSingleTitle")}</h3>
            <p className="mt-1 text-sm leading-6 text-[#6b7280]">{t("resetSingleDescription")}</p>
          </div>
          <div className="flex min-w-0 flex-col gap-2 sm:flex-row">
            <input
              value={singleUserId}
              onChange={(event) => setSingleUserId(event.target.value)}
              aria-label={t("resetSingleLabel")}
              placeholder={t("resetSinglePlaceholder")}
              className="h-10 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
            />
            <Button
              type="button"
              variant="outline"
              disabled={resettingSingle || !singleUserId.trim()}
              onClick={() => void resetSingleOnboarding()}
              className="h-10 shrink-0 rounded-lg border-black/[0.08] bg-white px-3 hover:bg-[#f6f7fb]"
            >
              <RotateCcw className="size-4" />
              <span>{resettingSingle ? t("resetting") : t("resetSingleAction")}</span>
            </Button>
          </div>
          {singleResetResult ? (
            <KeyValueGrid entries={Object.entries(singleResetResult)} />
          ) : null}
        </div>
      </Panel>
    </div>
  );
}

function settingLabel(t: OnboardingSettingsTranslator, key: string, meta: SettingMeta) {
  switch (key) {
    case "onboarding_enabled":
      return t("fields.enabled");
    case "onboarding_allow_skip":
      return t("fields.allowSkip");
    case "onboarding_interest_options":
      return t("fields.interestOptions");
    case "onboarding_custom_fields":
      return t("fields.customFields");
    case "onboarding_avatar_enabled":
      return t("fields.avatarEnabled");
    case "onboarding_avatar_required":
      return t("fields.avatarRequired");
    case "onboarding_background_enabled":
      return t("fields.backgroundEnabled");
    case "onboarding_background_required":
      return t("fields.backgroundRequired");
    case "onboarding_name_enabled":
      return t("fields.nameEnabled");
    case "onboarding_name_required":
      return t("fields.nameRequired");
    case "onboarding_signature_enabled":
      return t("fields.signatureEnabled");
    case "onboarding_signature_required":
      return t("fields.signatureRequired");
    case "onboarding_interests_enabled":
      return t("fields.interestsEnabled");
    case "onboarding_interests_required":
      return t("fields.interestsRequired");
    case "onboarding_min_interests":
      return t("fields.minInterests");
    case "onboarding_points_intro_title":
      return t("fields.pointsIntroTitle");
    case "onboarding_points_intro_summary":
      return t("fields.pointsIntroSummary");
    case "onboarding_points_intro_detail":
      return t("fields.pointsIntroDetail");
    case "onboarding_result_title":
      return t("fields.resultTitle");
    case "onboarding_result_saved_text":
      return t("fields.resultSavedText");
    case "onboarding_points_wallet_label":
      return t("fields.walletLabel");
    case "onboarding_points_wallet_url":
      return t("fields.walletUrl");
    default:
      return meta.label ?? key;
  }
}

function settingHint(t: OnboardingSettingsTranslator, key: string, meta: SettingMeta) {
  switch (key) {
    case "onboarding_enabled":
      return t("hints.enabled");
    case "onboarding_allow_skip":
      return t("hints.allowSkip");
    case "onboarding_interest_options":
      return t("hints.interestOptions");
    case "onboarding_custom_fields":
      return t("hints.customFields");
    case "onboarding_min_interests":
      return t("hints.minInterests");
    case "onboarding_points_intro_detail":
      return t("hints.pointsIntroDetail");
    case "onboarding_points_wallet_url":
      return t("hints.walletUrl");
    default:
      return meta.hint ?? "";
  }
}

function LocalizedInterestOptionsEditor({
  hint,
  label,
  onChange,
  sampleOptions,
  settingKey,
  t,
  value,
}: {
  hint: string;
  label: string;
  onChange: (value: Record<string, string[]>) => void;
  sampleOptions: string[];
  settingKey: string;
  t: OnboardingSettingsTranslator;
  value: unknown;
}) {
  const localized = value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};
  const updateLocale = (locale: string, options: string[]) => {
    onChange({ ...localized, [locale]: options } as Record<string, string[]>);
  };
  return (
    <fieldset className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:col-span-2">
      <legend className="px-1 text-sm font-semibold text-[#343944]">{label}</legend>
      <div className="mt-2 grid gap-3">
        {LOCALES.map((locale) => (
          <InterestOptionsEditor
            key={locale}
            hint=""
            label={locale}
            sampleOptions={sampleOptions}
            settingKey={`${settingKey}.${locale}`}
            t={t}
            value={localized[locale]}
            onChange={(options) => updateLocale(locale, options)}
          />
        ))}
      </div>
      <span className="mt-2 block break-all text-xs text-[#8b919e]">{settingKey}</span>
      {hint ? <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{hint}</span> : null}
    </fieldset>
  );
}

function InterestOptionsEditor({
  hint,
  label,
  onChange,
  sampleOptions,
  settingKey,
  t,
  value,
}: {
  hint: string;
  label: string;
  onChange: (value: string[]) => void;
  sampleOptions: string[];
  settingKey: string;
  t: OnboardingSettingsTranslator;
  value: unknown;
}) {
  const options = interestOptionsForEditor(value);
  const updateOption = (index: number, nextValue: string) => {
    onChange(options.map((option, currentIndex) => (currentIndex === index ? nextValue : option)));
  };
  const removeOption = (index: number) => {
    onChange(options.filter((_, currentIndex) => currentIndex !== index));
  };
  const addSampleOptions = () => {
    const existing = new Set(options.map((option) => option.trim()).filter(Boolean));
    const nextOptions = [...options];
    for (const option of sampleOptions) {
      if (!existing.has(option)) {
        existing.add(option);
        nextOptions.push(option);
      }
    }
    onChange(nextOptions);
  };

  return (
    <div className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:col-span-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="block text-sm font-semibold text-[#343944]">{label}</span>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" onClick={addSampleOptions} className="h-8 rounded-lg border-black/[0.08] bg-white px-2 text-xs hover:bg-[#f6f7fb]">
            <Sparkles className="size-3.5" />
            <span>{t("addSampleInterests")}</span>
          </Button>
          <Button type="button" variant="outline" onClick={() => onChange([...options, ""])} className="h-8 rounded-lg border-black/[0.08] bg-white px-2 text-xs hover:bg-[#f6f7fb]">
            <Plus className="size-3.5" />
            <span>{t("addInterest")}</span>
          </Button>
        </div>
      </div>
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        {options.length ? (
          options.map((option, index) => (
            <div key={`interest-${index}`} className="flex min-w-0 items-center gap-2 rounded-lg border border-black/[0.06] bg-white p-2">
              <input
                value={option}
                onChange={(event) => updateOption(index, event.target.value)}
                className="h-9 min-w-0 flex-1 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
                placeholder={t("interestPlaceholder")}
              />
              <Button type="button" variant="ghost" size="icon" onClick={() => removeOption(index)} aria-label={t("removeInterest")} className="size-9 shrink-0 rounded-lg text-[#b91c1c] hover:bg-[#fef2f2] hover:text-[#991b1b]">
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))
        ) : (
          <div className="rounded-lg border border-dashed border-black/[0.08] bg-white px-3 py-4 text-sm font-medium text-[#7b808c] sm:col-span-2">
            {t("emptyInterestOptions")}
          </div>
        )}
      </div>
      <span className="mt-2 block break-all text-xs text-[#8b919e]">{settingKey}</span>
      {hint ? <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{hint}</span> : null}
    </div>
  );
}

function interestOptionsForEditor(value: unknown) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item ?? ""));
  }
  if (typeof value === "string") {
    const text = value.trim();
    if (!text) {
      return [];
    }
    try {
      const parsed = JSON.parse(text);
      if (Array.isArray(parsed)) {
        return parsed.map((item) => String(item ?? ""));
      }
    } catch {
      // Keep legacy plain text editable instead of discarding it.
    }
    return [text];
  }
  return [];
}

function normalizedInterestOptions(value: unknown) {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const option of interestOptionsForEditor(value)) {
    const text = option.trim();
    if (!text || seen.has(text)) {
      continue;
    }
    seen.add(text);
    out.push(text);
  }
  return out;
}

function normalizedLocalizedInterestOptions(value: unknown) {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([locale, localeValue]) => [
        locale,
        normalizedInterestOptions(localeValue),
      ]),
    );
  }
  return normalizedInterestOptions(value);
}
