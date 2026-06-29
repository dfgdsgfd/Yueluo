"use client";

import { FileText, Loader2, RotateCcw, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AIContentFormatConfig, AITemplateConfig } from "@/lib/types";
import { ToggleSwitch } from "./form-fields";
import { Panel } from "./layout-widgets";
import type { AIAgentPanelT, AIDraftSettings } from "./ai-agent-panel-model";
import { TemplateConcurrencyInput } from "./ai-template-concurrency-input";
import { TemplateModelInput } from "./ai-template-model-input";
import { TemplateRuntimeOverrides } from "./ai-template-runtime-overrides";

const contentFormatModes = [
  { key: "format", templateTask: "format_markdown" },
  { key: "polish", templateTask: "post_polish" },
  { key: "custom", templateTask: "post_custom_generate" },
] as const;

const contentFormatTemplateKeys = ["markdown_format", "post_polish", "post_custom_generate"] as const;

const defaultCustomContinuation = {
  enabled: true,
  triggerChars: 6000,
  maxRounds: 2,
  contextChars: 2400,
};

export function ContentFormatTab({
  draft,
  saving,
  t,
  updateDraft,
  onTemplateChange,
  onTemplateReset,
  onSave,
}: {
  draft: AIDraftSettings;
  saving: boolean;
  t: AIAgentPanelT;
  updateDraft: (patch: Partial<AIDraftSettings>) => void;
  onTemplateChange: (key: string, patch: Partial<AITemplateConfig>) => void;
  onTemplateReset: (key: string) => void;
  onSave: () => void;
}) {
  const config = draft.contentFormat;
  const updateConfig = (patch: Partial<AIContentFormatConfig>) => {
    updateDraft({ contentFormat: { ...config, ...patch } });
  };
  const updateMode = (mode: keyof Pick<AIContentFormatConfig, "format" | "polish" | "custom">, patch: Partial<AIContentFormatConfig[typeof mode]>) => {
    updateConfig({ [mode]: { ...config[mode], ...patch } });
  };
  const customContinuation = config.custom.continuation ?? defaultCustomContinuation;
  const updateCustomContinuation = (patch: Partial<typeof defaultCustomContinuation>) => {
    updateMode("custom", { continuation: { ...defaultCustomContinuation, ...customContinuation, ...patch } });
  };
  return (
    <Panel
      title={t("contentFormat.title")}
      icon={FileText}
      action={
        <Button type="button" disabled={saving} onClick={onSave} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
          {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          <span>{saving ? t("saving") : t("save")}</span>
        </Button>
      }
    >
      <div className="grid gap-4">
        <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:grid-cols-3">
          <div className="md:col-span-3">
            <ToggleSwitch
              value={config.enabled}
              onChange={(enabled) => updateConfig({ enabled })}
              onLabel={t("contentFormat.enabled")}
              offLabel={t("contentFormat.disabled")}
            />
          </div>
          {contentFormatModes.map(({ key, templateTask }) => (
            <section key={key} className="grid gap-3 rounded-lg border border-black/[0.06] bg-white p-3">
              <ToggleSwitch
                value={config[key].enabled}
                onChange={(enabled) => updateMode(key, { enabled })}
                onLabel={t(`contentFormat.modes.${key}.enabled`)}
                offLabel={t(`contentFormat.modes.${key}.disabled`)}
              />
              <label className="grid gap-1.5">
                <span className="text-xs font-semibold text-[#666c78]">{t("contentFormat.template")}</span>
                <select
                  value={config[key].templateKey}
                  onChange={(event) => updateMode(key, { templateKey: event.target.value })}
                  className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
                >
                  {Object.entries(draft.templates)
                    .filter(([, template]) => template.taskType === templateTask)
                    .map(([templateKey]) => (
                      <option key={templateKey} value={templateKey}>
                        {t.has(`templates.${templateKey}`) ? t(`templates.${templateKey}`) : templateKey}
                  </option>
                    ))}
                </select>
              </label>
              {key === "custom" ? (
                <div className="grid gap-3 border-t border-black/[0.06] pt-3">
                  <ToggleSwitch
                    value={customContinuation.enabled}
                    onChange={(enabled) => updateCustomContinuation({ enabled })}
                    onLabel={t("contentFormat.continuation.enabled")}
                    offLabel={t("contentFormat.continuation.disabled")}
                  />
                  <div className="grid gap-3">
                    <NumberInput
                      label={t("contentFormat.continuation.triggerChars")}
                      value={customContinuation.triggerChars}
                      min={1000}
                      max={100000}
                      onChange={(triggerChars) => updateCustomContinuation({ triggerChars })}
                    />
                    <NumberInput
                      label={t("contentFormat.continuation.maxRounds")}
                      value={customContinuation.maxRounds}
                      min={1}
                      max={8}
                      onChange={(maxRounds) => updateCustomContinuation({ maxRounds })}
                    />
                    <NumberInput
                      label={t("contentFormat.continuation.contextChars")}
                      value={customContinuation.contextChars}
                      min={600}
                      max={20000}
                      onChange={(contextChars) => updateCustomContinuation({ contextChars })}
                    />
                  </div>
                </div>
              ) : null}
            </section>
          ))}
        </section>

        {contentFormatTemplateKeys.map((key) => {
          const template = draft.templates[key];
          if (!template) return null;
          return (
            <section key={key} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
              <div className="mb-3 flex min-w-0 flex-wrap items-center gap-3">
                <div className="min-w-0 flex-1">
                  <h3 className="truncate text-sm font-semibold text-[#20232a]">{t(`templates.${key}`)}</h3>
                  <p className="truncate text-xs text-[#7b8190]">{key}</p>
                </div>
                <div className="w-40">
                  <ToggleSwitch
                    value={template.enabled}
                    onChange={(enabled) => onTemplateChange(key, { enabled })}
                    onLabel={t("template.enabled")}
                    offLabel={t("template.disabled")}
                  />
                </div>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => onTemplateReset(key)}
                  className="h-9 rounded-lg border-black/[0.08] bg-white px-3"
                >
                  <RotateCcw className="size-4" />
                  <span>{t("template.resetDefault")}</span>
                </Button>
              </div>
              <div className="grid gap-3 md:grid-cols-3">
                <TemplateModelInput template={template} t={t} onChange={(model) => onTemplateChange(key, { model })} />
                <NumberInput label={t("template.temperature")} value={template.temperature} min={0} max={2} step={0.1} onChange={(temperature) => onTemplateChange(key, { temperature })} />
                <NumberInput label={t("template.maxOutput")} value={template.maxOutputTokens} min={0} onChange={(maxOutputTokens) => onTemplateChange(key, { maxOutputTokens })} />
                <TemplateConcurrencyInput template={template} t={t} onChange={(concurrency) => onTemplateChange(key, { concurrency })} />
                <TemplateRuntimeOverrides
                  template={template}
                  t={t}
                  onChange={(runtimeOverrides) => onTemplateChange(key, { runtimeOverrides })}
                />
                <label className="grid gap-1.5 md:col-span-3">
                  <span className="text-xs font-semibold text-[#666c78]">{t("template.systemPrompt")}</span>
                  <textarea
                    value={template.systemPrompt ?? ""}
                    onChange={(event) => onTemplateChange(key, { systemPrompt: event.target.value })}
                    className="min-h-[130px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
                    spellCheck={false}
                  />
                </label>
                <label className="grid gap-1.5 md:col-span-3">
                  <span className="text-xs font-semibold text-[#666c78]">{t("template.userPrompt")}</span>
                  <textarea
                    value={template.userPrompt ?? template.prompt}
                    onChange={(event) => onTemplateChange(key, { userPrompt: event.target.value, prompt: event.target.value })}
                    className="min-h-[180px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
                    spellCheck={false}
                  />
                </label>
              </div>
            </section>
          );
        })}
      </div>
    </Panel>
  );
}

function NumberInput({
  label,
  value,
  min,
  max,
  step,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max?: number;
  step?: number;
  onChange: (value: number) => void;
}) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <input
        value={value}
        min={min}
        max={max}
        step={step}
        type="number"
        onChange={(event) => onChange(Number(event.target.value))}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      />
    </label>
  );
}
