"use client";

import { useMemo, useState } from "react";
import type { AITemplateConfig, AIRuntimeOverrides } from "@/lib/types";
import { ToggleSwitch } from "./form-fields";
import type { AIAgentPanelT } from "./ai-agent-panel-model";

type RuntimeOverridesProps = {
  template: AITemplateConfig;
  t: AIAgentPanelT;
  onChange: (runtimeOverrides: AIRuntimeOverrides) => void;
};

const emptyOverrides: AIRuntimeOverrides = {
  enabled: false,
  modelParameters: {},
  reasoningEffort: "",
};

export function TemplateRuntimeOverrides({
  template,
  t,
  onChange,
}: RuntimeOverridesProps) {
  const overrides = useMemo(
    () => normalizeRuntimeOverrides(template.runtimeOverrides),
    [template.runtimeOverrides],
  );
  const [modelParametersText, setModelParametersText] = useState(() =>
    formatRuntimeModelParameters(overrides.modelParameters),
  );
  const modelParametersValid = useMemo(
    () => parseRuntimeModelParameters(modelParametersText) !== null,
    [modelParametersText],
  );

  const patchOverrides = (patch: Partial<AIRuntimeOverrides>) => {
    const next = { ...overrides, ...patch };
    onChange(next);
  };

  const commitModelParameters = (value: string) => {
    setModelParametersText(value);
    const parsed = parseRuntimeModelParameters(value);
    if (parsed) {
      patchOverrides({ modelParameters: parsed });
    }
  };

  return (
    <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-white p-3 md:col-span-full">
      <div className="flex min-w-0 flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <h4 className="truncate text-sm font-semibold text-[#20232a]">
            {t("template.runtimeOverrides")}
          </h4>
          <p className="mt-1 text-xs leading-5 text-[#7b8190]">
            {t("template.runtimeOverridesHint")}
          </p>
        </div>
        <div className="w-44">
          <ToggleSwitch
            value={overrides.enabled}
            onChange={(enabled) => patchOverrides({ enabled })}
            onLabel={t("template.runtimeOn")}
            offLabel={t("template.runtimeOff")}
          />
        </div>
      </div>
      {overrides.enabled ? (
        <div className="grid gap-3 md:grid-cols-2">
          <ToggleSwitch
            value={Boolean(overrides.showReasoning)}
            onChange={(showReasoning) => patchOverrides({ showReasoning })}
            onLabel={t("base.showReasoningOn")}
            offLabel={t("base.showReasoningOff")}
          />
          <ToggleSwitch
            value={Boolean(overrides.thinkingParameterEnabled)}
            onChange={(thinkingParameterEnabled) => patchOverrides({ thinkingParameterEnabled })}
            onLabel={t("base.thinkingParameterOn")}
            offLabel={t("base.thinkingParameterOff")}
          />
          <ToggleSwitch
            value={Boolean(overrides.thinkingEnabled)}
            onChange={(thinkingEnabled) => patchOverrides({ thinkingEnabled })}
            onLabel={t("base.thinkingOn")}
            offLabel={t("base.thinkingOff")}
          />
          <label className="grid gap-1.5">
            <span className="text-xs font-semibold text-[#666c78]">{t("base.reasoningEffort")}</span>
            <select
              value={overrides.reasoningEffort ?? ""}
              onChange={(event) => patchOverrides({ reasoningEffort: event.target.value })}
              className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
            >
              {["", "minimal", "low", "medium", "high"].map((value) => (
                <option key={value || "default"} value={value}>
                  {t(`base.reasoningEfforts.${value || "default"}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1.5 md:col-span-2">
            <span className="text-xs font-semibold text-[#666c78]">{t("template.runtimeModelParameters")}</span>
            <textarea
              value={modelParametersText}
              onChange={(event) => commitModelParameters(event.target.value)}
              placeholder={t("base.modelParametersPlaceholder")}
              className="min-h-[90px] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
              spellCheck={false}
            />
            {!modelParametersValid ? (
              <span className="text-xs font-semibold text-[#dc2626]">
                {t("base.modelParametersInvalid")}
              </span>
            ) : null}
          </label>
        </div>
      ) : null}
    </section>
  );
}

function normalizeRuntimeOverrides(value?: AIRuntimeOverrides): AIRuntimeOverrides {
  return {
    ...emptyOverrides,
    ...value,
    enabled: Boolean(value?.enabled),
    reasoningEffort: value?.reasoningEffort ?? "",
    modelParameters: value?.modelParameters ?? {},
  };
}

function formatRuntimeModelParameters(params?: Record<string, unknown>) {
  if (!params || Object.keys(params).length === 0) {
    return "";
  }
  return JSON.stringify(params, null, 2);
}

function parseRuntimeModelParameters(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? parsed as Record<string, unknown>
      : null;
  } catch {
    return null;
  }
}
