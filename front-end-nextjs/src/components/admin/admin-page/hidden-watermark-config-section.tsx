"use client";

import { Loader2, Save, ShieldCheck, SlidersHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { truthy } from "./helpers";
import {
  defaultGolaySeed,
  localWatermarkPresets,
  officialRemotePassword,
  WatermarkNumber,
  WatermarkSelect,
  WatermarkToggle,
} from "./hidden-watermark-access-model";
import { Panel } from "./layout-widgets";
import { KeyValueGrid, LoadingBlock } from "./resource-editor";
import type { HiddenWatermarkAccessController } from "./hidden-watermark-access-view-types";

export function WatermarkConfigSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { applyLocalPreset, applyStableProtectionPreset, loading, save, saving, t, updateRemoteWatermarkSetting, updateWatermarkSetting, usingRemoteWatermark, watermarkEngine, watermarkRuntime, watermarkSettings } = controller;
  return (
      <Panel
        title={t("watermarkConfig.title")}
        icon={SlidersHorizontal}
        action={
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              type="button"
              disabled={saving || loading}
              onClick={() => void applyStableProtectionPreset()}
              className="h-9 rounded-lg bg-[#0f766e] px-3 hover:bg-[#0f5f59]"
            >
              {saving ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
              <span>{t("watermarkConfig.stablePreset.action")}</span>
            </Button>
            <Button
              type="button"
              disabled={saving || loading}
              onClick={() => void save()}
              className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"
            >
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              <span>{saving ? t("saving") : t("save")}</span>
            </Button>
          </div>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loading")} />
        ) : (
          <div className="grid gap-4">
            <KeyValueGrid
              entries={[
                [t("watermarkConfig.runtime.payload"), t("watermarkConfig.runtime.payloadValue", {
                  bytes: watermarkRuntime?.payloadBytes ?? 8,
                  bits: watermarkRuntime?.payloadBits ?? 64,
                })],
                [t("watermarkConfig.runtime.format"), watermarkRuntime?.payloadFormat ?? "short_code_v1"],
                [t("watermarkConfig.runtime.engine"), watermarkRuntime?.engineMode ?? watermarkEngine],
                [t("watermarkConfig.runtime.referenceRecovery"), Boolean(watermarkRuntime?.referenceRecovery)],
              ]}
            />
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <WatermarkSelect
                label={t("watermarkConfig.fields.engine")}
                hint={t("watermarkConfig.hints.engine")}
                value={String(watermarkSettings.hidden_watermark_engine ?? "auto")}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_engine", value, false)}
                options={[
                  { value: "auto", label: t("watermarkConfig.options.engine.auto") },
                  { value: "local", label: t("watermarkConfig.options.engine.local") },
                  { value: "remote", label: t("watermarkConfig.options.engine.remote") },
                ]}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.enabled")}
                hint={t("watermarkConfig.hints.enabled")}
                value={truthy(watermarkSettings.hidden_watermark_enabled)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_enabled", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.protectedOnly")}
                hint={t("watermarkConfig.hints.protectedOnly")}
                value={truthy(watermarkSettings.hidden_watermark_protected_only)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_protected_only", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeUid")}
                hint={t("watermarkConfig.hints.includeUid")}
                value={truthy(watermarkSettings.hidden_watermark_include_uid)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_uid", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeUserId")}
                hint={t("watermarkConfig.hints.includeUserId")}
                value={truthy(watermarkSettings.hidden_watermark_include_user_id)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_user_id", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeUsername")}
                hint={t("watermarkConfig.hints.includeUsername")}
                value={truthy(watermarkSettings.hidden_watermark_include_username)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_username", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeTime")}
                hint={t("watermarkConfig.hints.includeTime")}
                value={truthy(watermarkSettings.hidden_watermark_include_time)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_time", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeFileHash")}
                hint={t("watermarkConfig.hints.includeFileHash")}
                value={truthy(watermarkSettings.hidden_watermark_include_file_hash)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_file_hash", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <WatermarkToggle
                label={t("watermarkConfig.fields.includeCustom")}
                hint={t("watermarkConfig.hints.includeCustom")}
                value={truthy(watermarkSettings.hidden_watermark_include_custom)}
                disabled={saving}
                onChange={(value) => updateWatermarkSetting("hidden_watermark_include_custom", value, false)}
                onLabel={t("on")}
                offLabel={t("off")}
              />
              <label className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
                <span className="mb-2 block text-sm font-semibold text-[#343944]">{t("watermarkConfig.fields.customText")}</span>
                <input
                  value={String(watermarkSettings.hidden_watermark_custom_text ?? "")}
                  onChange={(event) => updateWatermarkSetting("hidden_watermark_custom_text", event.target.value, false)}
                  maxLength={255}
                  className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
                />
                <span className="mt-1 block text-xs leading-5 text-[#7b808c]">{t("watermarkConfig.hints.customText")}</span>
              </label>
            </div>
            {usingRemoteWatermark ? (
              <>
                <div className="min-w-0 rounded-lg border border-[#1d4ed8]/20 bg-[#eff6ff] p-4">
                  <span className="block text-sm font-semibold text-[#1e3a8a]">
                    {t("watermarkConfig.remoteOfficial.title")}
                  </span>
                  <span className="mt-1 block text-xs leading-5 text-[#475569]">
                    {t("watermarkConfig.remoteOfficial.description")}
                  </span>
                </div>
                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <WatermarkSelect
                    label={t("watermarkConfig.fields.remoteEngine")}
                    hint={t("watermarkConfig.hints.remoteEngine")}
                    value={String(watermarkSettings.hidden_watermark_remote_engine ?? "auto")}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_engine", value)}
                    options={["blind_watermark", "dwt_dct_svd", "auto"].map((value) => ({
                      value,
                      label: t(`watermarkConfig.options.remoteEngine.${value}`),
                    }))}
                  />
                  <WatermarkSelect
                    label={t("watermarkConfig.fields.remoteProfile")}
                    hint={t("watermarkConfig.hints.remoteProfile")}
                    value={String(watermarkSettings.hidden_watermark_remote_profile ?? "adaptive")}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_profile", value)}
                    options={["adaptive", "fidelity", "balanced", "strong", "official", "custom"].map((value) => ({
                      value,
                      label: t(`watermarkConfig.options.remoteProfile.${value}`),
                    }))}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.remotePasswordWm")}
                    hint={t("watermarkConfig.hints.remotePasswordWm")}
                    value={Number(watermarkSettings.hidden_watermark_remote_password_wm ?? officialRemotePassword)}
                    min={0}
                    max={2147483647}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_password_wm", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.remotePasswordImg")}
                    hint={t("watermarkConfig.hints.remotePasswordImg")}
                    value={Number(watermarkSettings.hidden_watermark_remote_password_img ?? officialRemotePassword)}
                    min={0}
                    max={2147483647}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_password_img", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.remoteTimeout")}
                    hint={t("watermarkConfig.hints.remoteTimeout")}
                    value={Number(watermarkSettings.hidden_watermark_remote_timeout_seconds ?? 50)}
                    min={10}
                    max={300}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_timeout_seconds", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.remoteOperationTimeout")}
                    hint={t("watermarkConfig.hints.remoteOperationTimeout")}
                    value={Number(watermarkSettings.hidden_watermark_remote_operation_timeout_seconds ?? 45)}
                    min={10}
                    max={300}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_operation_timeout_seconds", value)}
                  />
                  {watermarkSettings.hidden_watermark_remote_profile === "custom" ? (
                    <>
                      <WatermarkNumber
                        label={t("watermarkConfig.fields.remoteD1")}
                        hint={t("watermarkConfig.hints.remoteD1D2")}
                        value={Number(watermarkSettings.hidden_watermark_remote_custom_d1 ?? 18)}
                        min={1}
                        max={64}
                        disabled={saving}
                        onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_custom_d1", value)}
                      />
                      <WatermarkNumber
                        label={t("watermarkConfig.fields.remoteD2")}
                        hint={t("watermarkConfig.hints.remoteD1D2")}
                        value={Number(watermarkSettings.hidden_watermark_remote_custom_d2 ?? 8)}
                        min={0}
                        max={64}
                        disabled={saving}
                        onChange={(value) => updateRemoteWatermarkSetting("hidden_watermark_remote_custom_d2", value)}
                      />
                    </>
                  ) : null}
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.protectionMaxDimension")}
                    hint={t("watermarkConfig.hints.protectionMaxDimension")}
                    value={Number(watermarkSettings.image_protection_max_dimension ?? 2048)}
                    min={0}
                    max={16384}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("image_protection_max_dimension", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.protectionAllowedFailurePercent")}
                    hint={t("watermarkConfig.hints.protectionAllowedFailurePercent")}
                    value={Number(watermarkSettings.image_protection_allowed_failure_percent ?? 20)}
                    min={0}
                    max={100}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("image_protection_allowed_failure_percent", value)}
                  />
                  <WatermarkSelect
                    label={t("watermarkConfig.fields.protectionOutputMode")}
                    hint={t("watermarkConfig.hints.protectionOutputMode")}
                    value={String(watermarkSettings.image_protection_output_mode ?? "lossless_webp")}
                    disabled={saving}
                    onChange={(value) => updateRemoteWatermarkSetting("image_protection_output_mode", value)}
                    options={[
                      { value: "lossless_webp", label: t("watermarkConfig.options.protectionOutputMode.lossless_webp") },
                      { value: "quality_webp", label: t("watermarkConfig.options.protectionOutputMode.quality_webp") },
                    ]}
                  />
                  {watermarkSettings.image_protection_output_mode === "quality_webp" ? (
                    <WatermarkNumber
                      label={t("watermarkConfig.fields.protectionQuality")}
                      hint={t("watermarkConfig.hints.protectionQuality")}
                      value={Number(watermarkSettings.image_protection_webp_quality ?? 95)}
                      min={1}
                      max={100}
                      disabled={saving}
                      onChange={(value) => updateRemoteWatermarkSetting("image_protection_webp_quality", value)}
                    />
                  ) : null}
                </div>
              </>
            ) : (
              <>
                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                  {localWatermarkPresets.map((preset) => {
                    const active = watermarkSettings.hidden_watermark_profile === preset.id;
                    return (
                      <button
                        key={preset.id}
                        type="button"
                        onClick={() => applyLocalPreset(preset.id)}
                        className={active
                          ? "min-w-0 rounded-lg border border-[#1d4ed8]/35 bg-[#eff6ff] p-3 text-left shadow-sm"
                          : "min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 text-left hover:border-[#1d4ed8]/25 hover:bg-white"}
                      >
                        <span className="block text-sm font-semibold text-[#303642]">{t(`watermarkConfig.presets.${preset.id}.title`)}</span>
                        <span className="mt-1 block text-xs leading-5 text-[#667085]">{t(`watermarkConfig.presets.${preset.id}.description`)}</span>
                      </button>
                    );
                  })}
                </div>
                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <WatermarkSelect
                    label={t("watermarkConfig.fields.coefficientMode")}
                    hint={t("watermarkConfig.hints.coefficientMode")}
                    value={String(watermarkSettings.hidden_watermark_coefficient_mode ?? "d1d2")}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_coefficient_mode", value)}
                    options={[
                      { value: "d1d2", label: t("watermarkConfig.options.coefficientMode.d1d2") },
                      { value: "d1", label: t("watermarkConfig.options.coefficientMode.d1") },
                    ]}
                  />
                  <WatermarkSelect
                    label={t("watermarkConfig.fields.eccMode")}
                    hint={t("watermarkConfig.hints.eccMode")}
                    value={String(watermarkSettings.hidden_watermark_ecc_mode ?? "golay")}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_ecc_mode", value)}
                    options={[
                      { value: "golay", label: t("watermarkConfig.options.eccMode.golay") },
                      { value: "none", label: t("watermarkConfig.options.eccMode.none") },
                    ]}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.golaySeed")}
                    hint={t("watermarkConfig.hints.golaySeed")}
                    value={Number(watermarkSettings.hidden_watermark_golay_seed ?? defaultGolaySeed)}
                    min={0}
                    max={2147483647}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_golay_seed", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.blockWidth")}
                    hint={t("watermarkConfig.hints.blockShape")}
                    value={Number(watermarkSettings.hidden_watermark_block_width ?? 8)}
                    min={4}
                    max={64}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_block_width", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.blockHeight")}
                    hint={t("watermarkConfig.hints.blockShape")}
                    value={Number(watermarkSettings.hidden_watermark_block_height ?? 6)}
                    min={4}
                    max={64}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_block_height", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.d1")}
                    hint={t("watermarkConfig.hints.d1d2")}
                    value={Number(watermarkSettings.hidden_watermark_d1 ?? 21)}
                    min={1}
                    max={64}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_d1", value)}
                  />
                  <WatermarkNumber
                    label={t("watermarkConfig.fields.d2")}
                    hint={t("watermarkConfig.hints.d1d2")}
                    value={Number(watermarkSettings.hidden_watermark_d2 ?? 9)}
                    min={0}
                    max={64}
                    disabled={saving}
                    onChange={(value) => updateWatermarkSetting("hidden_watermark_d2", value)}
                  />
                </div>
              </>
            )}
            <div className="rounded-lg border border-[#1d4ed8]/12 bg-[#f8fbff] p-3 text-xs leading-5 text-[#596579]">
              <span className="block">{usingRemoteWatermark ? t("watermarkConfig.remoteReference") : t("watermarkConfig.reference")}</span>
            </div>
          </div>
        )}
      </Panel>
  );
}

