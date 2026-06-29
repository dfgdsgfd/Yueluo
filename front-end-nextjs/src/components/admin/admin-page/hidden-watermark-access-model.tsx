import {
ShieldCheck,
Trash2
} from "lucide-react";
import {
ToggleSwitch
} from "./form-fields";
import {
truthy
} from "./helpers";
import {
EmptyBlock
} from "./resource-editor";

export type SettingMeta = {
  value?: unknown;
};

export type WatermarkRuntime = {
  payloadBytes?: number;
  payloadBits?: number;
  payloadFormat?: string;
  tokenBytes?: number;
  engineMode?: string;
  remoteConfigured?: boolean;
  referenceRecovery?: boolean;
};

export type ManualType = "id" | "username";

export const allUsersKey = "hidden_watermark_extract_all_users";

export const userIDsKey = "hidden_watermark_extract_user_ids";

export const usernamesKey = "hidden_watermark_extract_usernames";

export const defaultGolaySeed = 1234567890;

export const officialRemotePassword = 1;

export const watermarkSettingDefaults: Record<string, boolean | number | string> = {
  hidden_watermark_enabled: true,
  hidden_watermark_protected_only: true,
  hidden_watermark_include_uid: true,
  hidden_watermark_include_user_id: true,
  hidden_watermark_include_username: false,
  hidden_watermark_include_time: true,
  hidden_watermark_include_file_hash: true,
  hidden_watermark_include_custom: false,
  hidden_watermark_custom_text: "",
  hidden_watermark_engine: "auto",
  hidden_watermark_profile: "current",
  hidden_watermark_block_width: 8,
  hidden_watermark_block_height: 6,
  hidden_watermark_coefficient_mode: "d1d2",
  hidden_watermark_d1: 21,
  hidden_watermark_d2: 9,
  hidden_watermark_ecc_mode: "golay",
  hidden_watermark_golay_seed: defaultGolaySeed,
  hidden_watermark_remote_password_wm: officialRemotePassword,
  hidden_watermark_remote_password_img: officialRemotePassword,
  hidden_watermark_remote_engine: "auto",
  hidden_watermark_remote_profile: "adaptive",
  hidden_watermark_remote_custom_d1: 18,
  hidden_watermark_remote_custom_d2: 8,
  hidden_watermark_remote_timeout_seconds: 50,
  hidden_watermark_remote_operation_timeout_seconds: 45,
  image_protection_max_dimension: 2048,
  image_protection_allowed_failure_percent: 20,
  image_protection_output_mode: "lossless_webp",
  image_protection_webp_quality: 95,
};

export const stableProtectionWatermarkSettings: Record<string, boolean | number | string> = {
  hidden_watermark_enabled: true,
  hidden_watermark_protected_only: true,
  hidden_watermark_engine: "auto",
  hidden_watermark_remote_engine: "auto",
  hidden_watermark_remote_profile: "adaptive",
  hidden_watermark_remote_password_wm: officialRemotePassword,
  hidden_watermark_remote_password_img: officialRemotePassword,
  hidden_watermark_remote_timeout_seconds: 50,
  hidden_watermark_remote_operation_timeout_seconds: 45,
  image_protection_max_dimension: 2048,
  image_protection_allowed_failure_percent: 20,
  image_protection_output_mode: "lossless_webp",
  image_protection_webp_quality: 95,
};

export const watermarkBooleanKeys = [
  "hidden_watermark_enabled",
  "hidden_watermark_protected_only",
  "hidden_watermark_include_uid",
  "hidden_watermark_include_user_id",
  "hidden_watermark_include_username",
  "hidden_watermark_include_time",
  "hidden_watermark_include_file_hash",
  "hidden_watermark_include_custom",
];

export const watermarkNumberKeys = [
  "hidden_watermark_block_width",
  "hidden_watermark_block_height",
  "hidden_watermark_d1",
  "hidden_watermark_d2",
  "hidden_watermark_golay_seed",
  "hidden_watermark_remote_password_wm",
  "hidden_watermark_remote_password_img",
  "hidden_watermark_remote_custom_d1",
  "hidden_watermark_remote_custom_d2",
  "hidden_watermark_remote_timeout_seconds",
  "hidden_watermark_remote_operation_timeout_seconds",
  "image_protection_max_dimension",
  "image_protection_allowed_failure_percent",
  "image_protection_webp_quality",
];

export const localWatermarkPresets = [
  {
    id: "current",
    settings: {
      hidden_watermark_block_width: 8,
      hidden_watermark_block_height: 6,
      hidden_watermark_coefficient_mode: "d1d2",
      hidden_watermark_d1: 21,
      hidden_watermark_d2: 9,
      hidden_watermark_ecc_mode: "golay",
      hidden_watermark_golay_seed: defaultGolaySeed,
    },
  },
  {
    id: "author_recommended",
    settings: {
      hidden_watermark_block_width: 8,
      hidden_watermark_block_height: 8,
      hidden_watermark_coefficient_mode: "d1d2",
      hidden_watermark_d1: 21,
      hidden_watermark_d2: 9,
      hidden_watermark_ecc_mode: "golay",
      hidden_watermark_golay_seed: defaultGolaySeed,
    },
  },
  {
    id: "fidelity",
    settings: {
      hidden_watermark_block_width: 12,
      hidden_watermark_block_height: 12,
      hidden_watermark_coefficient_mode: "d1d2",
      hidden_watermark_d1: 8,
      hidden_watermark_d2: 3,
      hidden_watermark_ecc_mode: "golay",
      hidden_watermark_golay_seed: defaultGolaySeed,
    },
  },
  {
    id: "robust",
    settings: {
      hidden_watermark_block_width: 6,
      hidden_watermark_block_height: 4,
      hidden_watermark_coefficient_mode: "d1d2",
      hidden_watermark_d1: 36,
      hidden_watermark_d2: 20,
      hidden_watermark_ecc_mode: "golay",
      hidden_watermark_golay_seed: defaultGolaySeed,
    },
  },
];

export function metadataRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};
}

export function metadataText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

export function formatUsageSize(value: unknown) {
  const size = Number(value ?? 0);
  if (!Number.isFinite(size) || size <= 0) return "-";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatUsageTime(value: unknown) {
  if (typeof value !== "string" || !value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function AccessList({
  emptyLabel,
  hint,
  items,
  onRemove,
  removeLabel,
  title,
}: {
  emptyLabel: string;
  hint: string;
  items: string[];
  onRemove: (value: string) => void;
  removeLabel: string;
  title: string;
}) {
  return (
    <section className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <div className="mb-3 flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-[#303642]">{title}</h3>
          <p className="mt-1 text-xs leading-5 text-[#737b88]">{hint}</p>
        </div>
        <span className="shrink-0 rounded-full bg-white px-2 py-1 text-xs font-semibold text-[#59606c]">{items.length}</span>
      </div>
      {items.length ? (
        <div className="flex max-h-[320px] flex-wrap gap-2 overflow-y-auto">
          {items.map((item) => (
            <span
              key={item}
              className="inline-flex max-w-full items-center gap-2 rounded-lg border border-[#1d4ed8]/15 bg-white px-2.5 py-1.5 text-xs font-semibold text-[#1e3a8a]"
            >
              <span className="min-w-0 break-all">{item}</span>
              <button
                type="button"
                className="shrink-0 rounded-md p-0.5 text-[#64748b] hover:bg-[#eff6ff] hover:text-[#1d4ed8]"
                aria-label={removeLabel}
                title={removeLabel}
                onClick={() => onRemove(item)}
              >
                <Trash2 className="size-3.5" />
              </button>
            </span>
          ))}
        </div>
      ) : (
        <EmptyBlock icon={ShieldCheck} label={emptyLabel} />
      )}
    </section>
  );
}

export function listFromSetting(value: unknown) {
  if (Array.isArray(value)) {
    return mergeList([], value.map((item) => String(item ?? "")));
  }
  if (typeof value !== "string") {
    return [];
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return [];
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (Array.isArray(parsed)) {
      return mergeList([], parsed.map((item) => String(item ?? "")));
    }
  } catch {
    // Plain comma/newline separated settings are expected here.
  }
  return mergeList([], trimmed.split(/[\s,，]+/));
}

export function mergeList(current: string[], additions: string[]) {
  const seen = new Set(current.map((item) => item.trim().toLowerCase()).filter(Boolean));
  const next = current.filter((item) => item.trim());
  additions.forEach((item) => {
    const text = item.trim();
    const key = text.toLowerCase();
    if (!text || seen.has(key)) {
      return;
    }
    seen.add(key);
    next.push(text);
  });
  return next;
}

export function hasValue(values: string[], value: string) {
  return values.some((item) => item.trim().toLowerCase() === value.trim().toLowerCase());
}

export function watermarkSettingsPayload(settings: Record<string, unknown>) {
  const out: Record<string, unknown> = {};
  Object.keys(watermarkSettingDefaults).forEach((key) => {
    const value = settings[key] ?? watermarkSettingDefaults[key];
    if (watermarkBooleanKeys.includes(key)) {
      out[key] = truthy(value);
      return;
    }
    if (watermarkNumberKeys.includes(key)) {
      const numeric = Number(value);
      out[key] = Number.isFinite(numeric) ? numeric : watermarkSettingDefaults[key];
      return;
    }
    out[key] = typeof value === "string" ? value.trim() : value;
  });
  return out;
}

export function watermarkSettingsFromServer(settingValue: (key: string) => unknown) {
  const next = Object.fromEntries(
    Object.entries(watermarkSettingDefaults).map(([key, fallback]) => [key, settingValue(key) ?? fallback]),
  );
  const localPreset = localWatermarkPresets.find((item) => item.id === next.hidden_watermark_profile);
  return {
    ...next,
    ...(localPreset?.settings ?? {}),
  };
}

export function WatermarkToggle({
  disabled,
  hint,
  label,
  offLabel,
  onChange,
  onLabel,
  value,
}: {
  disabled?: boolean;
  hint: string;
  label: string;
  offLabel: string;
  onChange: (value: boolean) => void;
  onLabel: string;
  value: boolean;
}) {
  return (
    <div className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{label}</span>
      <ToggleSwitch value={value} onChange={onChange} disabled={disabled} onLabel={onLabel} offLabel={offLabel} />
      <span className="mt-2 block text-xs leading-5 text-[#7b808c]">{hint}</span>
    </div>
  );
}

export function WatermarkNumber({
  disabled,
  hint,
  label,
  max,
  min,
  onChange,
  value,
}: {
  disabled?: boolean;
  hint: string;
  label: string;
  max: number;
  min: number;
  onChange: (value: number) => void;
  value: number;
}) {
  return (
    <label className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{label}</span>
      <input
        type="number"
        value={Number.isFinite(value) ? value : min}
        min={min}
        max={max}
        disabled={disabled}
        onChange={(event) => onChange(Number(event.target.value))}
        className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8] disabled:bg-[#f1f2f5] disabled:text-[#8a8f9d]"
      />
      <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{hint}</span>
    </label>
  );
}

export function WatermarkSelect({
  disabled,
  hint,
  label,
  onChange,
  options,
  value,
}: {
  disabled?: boolean;
  hint: string;
  label: string;
  onChange: (value: string) => void;
  options: Array<{ label: string; value: string }>;
  value: string;
}) {
  return (
    <label className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <span className="mb-2 block text-sm font-semibold text-[#343944]">{label}</span>
      <select
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8] disabled:bg-[#f1f2f5] disabled:text-[#8a8f9d]"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
      <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{hint}</span>
    </label>
  );
}
