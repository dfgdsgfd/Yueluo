import type {
WatermarkExtractionProgress
} from "@/lib/api";
import {
adminRequest,
extractHiddenWatermark,
getAdminSecurityAuditLogs
} from "@/lib/api";
import type {
AdminSecurityAuditLogItem,
HiddenWatermarkResult
} from "@/lib/types";
import {
useTranslations
} from "next-intl";
import type {
FormEvent
} from "react";
import {
useCallback,
useEffect,
useMemo,
useState
} from "react";
import {
toast
} from "sonner";
import {
errorMessage,
truthy
} from "./helpers";
import { allUsersKey,hasValue,listFromSetting,localWatermarkPresets,ManualType,mergeList,SettingMeta,stableProtectionWatermarkSettings,userIDsKey,usernamesKey,WatermarkRuntime,watermarkSettingDefaults,watermarkSettingsFromServer,watermarkSettingsPayload } from "./hidden-watermark-access-model";
import type {
PickerSelection
} from "./types";

export function useHiddenWatermarkAccessController({ token }: { token: string }) {
const t = useTranslations("adminPortal.hiddenWatermarkAccessPanel");
  const [allUsers, setAllUsers] = useState(false);
  const [userIDs, setUserIDs] = useState<string[]>([]);
  const [usernames, setUsernames] = useState<string[]>([]);
  const [selectedUsers, setSelectedUsers] = useState<PickerSelection[]>([]);
  const [manualType, setManualType] = useState<ManualType>("id");
  const [manualValue, setManualValue] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [usageRecords, setUsageRecords] = useState<AdminSecurityAuditLogItem[]>([]);
  const [watermarkSettings, setWatermarkSettings] = useState<Record<string, unknown>>(watermarkSettingDefaults);
  const [watermarkFile, setWatermarkFile] = useState<File | null>(null);
  const [watermarkReferenceFile, setWatermarkReferenceFile] = useState<File | null>(null);
  const [watermarkResult, setWatermarkResult] = useState<HiddenWatermarkResult | null>(null);
  const [watermarkRuntime, setWatermarkRuntime] = useState<WatermarkRuntime | null>(null);
  const [extracting, setExtracting] = useState(false);
  const [extractionProgress, setExtractionProgress] = useState<WatermarkExtractionProgress>({
    stage: "uploading",
    percent: 0,
    elapsedMs: 0,
  });
  const watermarkEngine = String(watermarkSettings.hidden_watermark_engine ?? "auto");
  const usingRemoteWatermark = watermarkEngine !== "local";

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [settingsResult, usageResult] = await Promise.allSettled([
        adminRequest<{ raw?: Record<string, unknown>; settings?: Record<string, SettingMeta>; watermarkRuntime?: WatermarkRuntime }>(
          "/api/admin/system-settings",
          { method: "GET", token },
        ),
        getAdminSecurityAuditLogs({ category: "hidden_watermark", limit: 100, range: "all" }, token),
      ]);
      if (settingsResult.status === "rejected") {
        throw settingsResult.reason;
      }
      const data = settingsResult.value;
      const settingValue = (key: string) => data.settings?.[key]?.value ?? data.raw?.[key];
      setAllUsers(truthy(settingValue(allUsersKey)));
      setUserIDs(listFromSetting(settingValue(userIDsKey)));
      setUsernames(listFromSetting(settingValue(usernamesKey)));
      setWatermarkSettings(watermarkSettingsFromServer(settingValue));
      setWatermarkRuntime(data.watermarkRuntime ?? null);
      setUsageRecords(usageResult.status === "fulfilled" ? usageResult.value.items ?? [] : []);
      if (usageResult.status === "rejected") {
        toast.error(t("usage.loadFailed"));
      }
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [t, token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  const totalAccessCount = useMemo(
    () => userIDs.length + usernames.length,
    [userIDs.length, usernames.length],
  );

  async function saveWatermarkSettings(nextSettings: Record<string, unknown>, successMessage = t("saved")) {
    setSaving(true);
    try {
      await adminRequest("/api/admin/system-settings", {
        method: "PUT",
        token,
        body: JSON.stringify({
          ...watermarkSettingsPayload(nextSettings),
          [allUsersKey]: allUsers,
          [userIDsKey]: userIDs,
          [usernamesKey]: usernames,
        }),
      });
      toast.success(successMessage);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function save() {
    await saveWatermarkSettings(watermarkSettings);
  }

  async function extractWatermark() {
    if (!watermarkFile) return;
    setExtracting(true);
    setExtractionProgress({ stage: "uploading", percent: 0, elapsedMs: 0 });
    try {
      const result = await extractHiddenWatermark(watermarkFile, watermarkReferenceFile, {
        path: "/api/admin/image-watermark/extract",
        token,
        timeoutMs: 60_000,
        onProgress: (progress) => {
          setExtractionProgress((current) => ({
            ...progress,
            percent: Math.max(current.percent, progress.percent),
          }));
        },
      });
      setWatermarkResult(result);
      toast.success(t("extract.completed"));
    } catch (error) {
      setWatermarkResult(null);
      toast.error(
        error instanceof Error && error.message === "error.screenshot_recovery_failed"
          ? t("extract.recoveryFailed")
          : error instanceof Error && error.message === "error.hidden_watermark_remote_timeout"
            ? t("extract.remoteTimeout")
          : error instanceof Error &&
              (error.message === "error.watermark_stream_incomplete" ||
                error.message === "error.watermark_stream_disconnected")
            ? t("extract.remoteDisconnected")
          : errorMessage(error),
      );
    } finally {
      setExtracting(false);
    }
  }

  function updateWatermarkSetting(key: string, value: unknown, custom = true) {
    setWatermarkSettings((current) => ({
      ...current,
      [key]: value,
      ...(custom ? { hidden_watermark_profile: "custom" } : {}),
    }));
  }

  function updateRemoteWatermarkSetting(key: string, value: unknown) {
    setWatermarkSettings((current) => ({
      ...current,
      [key]: value,
    }));
  }

  function applyLocalPreset(id: string) {
    const preset = localWatermarkPresets.find((item) => item.id === id);
    if (!preset) return;
    setWatermarkSettings((current) => ({
      ...current,
      ...preset.settings,
      hidden_watermark_profile: id,
    }));
  }

  async function applyStableProtectionPreset() {
    const nextSettings = {
      ...watermarkSettings,
      ...stableProtectionWatermarkSettings,
    };
    setWatermarkSettings(nextSettings);
    await saveWatermarkSettings(nextSettings, t("watermarkConfig.stablePreset.saved"));
  }

  function addSelectedUsers() {
    if (!selectedUsers.length) {
      toast(t("picker.noSelected"));
      return;
    }
    const nextIDs: string[] = [];
    const nextUsernames: string[] = [];
    selectedUsers.forEach((item) => {
      nextIDs.push(String(item.id));
      if (item.displayId) {
        nextIDs.push(item.displayId);
      }
      const label = item.label.trim();
      if (label && label !== item.displayId && label !== String(item.id)) {
        nextUsernames.push(label);
      }
    });
    setUserIDs((current) => mergeList(current, nextIDs));
    setUsernames((current) => mergeList(current, nextUsernames));
    setSelectedUsers([]);
    toast.success(t("picker.added", { count: selectedUsers.length }));
  }

  function submitManual(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = manualValue.trim();
    if (!value) {
      toast(t("manual.emptyManual"));
      return;
    }
    if (manualType === "id") {
      setUserIDs((current) => {
        if (hasValue(current, value)) {
          toast(t("manual.duplicate"));
          return current;
        }
        return [...current, value];
      });
    } else {
      setUsernames((current) => {
        if (hasValue(current, value)) {
          toast(t("manual.duplicate"));
          return current;
        }
        return [...current, value];
      });
    }
    setManualValue("");
  }

  function removeValue(kind: ManualType, value: string) {
    if (kind === "id") {
      setUserIDs((current) => current.filter((item) => item !== value));
      return;
    }
    setUsernames((current) => current.filter((item) => item !== value));
  }

  return { addSelectedUsers, allUsers, applyLocalPreset, applyStableProtectionPreset, extractWatermark, extracting, extractionProgress, loading, manualType, manualValue, removeValue, save, saving, selectedUsers, setAllUsers, setManualType, setManualValue, setSelectedUsers, setWatermarkFile, setWatermarkReferenceFile, setWatermarkResult, submitManual, t, token, totalAccessCount, updateRemoteWatermarkSetting, updateWatermarkSetting, usageRecords, userIDs, usernames, usingRemoteWatermark, watermarkEngine, watermarkFile, watermarkResult, watermarkRuntime, watermarkSettings };
}
