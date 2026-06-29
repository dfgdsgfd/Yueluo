"use client";

import { Loader2, RotateCcw, Save, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AITemplateConfig } from "@/lib/types";
import { ToggleSwitch } from "./form-fields";
import { Panel } from "./layout-widgets";
import type { AIAgentPanelT, AIDraftSettings } from "./ai-agent-panel-model";
import { TemplateConcurrencyInput } from "./ai-template-concurrency-input";
import { TemplateModelInput } from "./ai-template-model-input";
import { TemplateRuntimeOverrides } from "./ai-template-runtime-overrides";

const publishGenerationTemplateKeys = ["publish_detail_generate", "publish_title_generate"] as const;

export function PublishGenerationTab({
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
  const config = draft.publishGeneration;
  const updateConfig = (patch: Partial<AIDraftSettings["publishGeneration"]>) => {
    updateDraft({ publishGeneration: { ...config, ...patch } });
  };
  return (
    <Panel
      title={t("publishGeneration.title")}
      icon={Sparkles}
      action={
        <Button type="button" disabled={saving} onClick={onSave} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
          {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          <span>{saving ? t("saving") : t("save")}</span>
        </Button>
      }
    >
      <div className="grid gap-4">
        <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:grid-cols-2">
          <div className="md:col-span-2">
            <ToggleSwitch
              value={config.enabled}
              onChange={(enabled) => updateConfig({ enabled })}
              onLabel={t("publishGeneration.enabled")}
              offLabel={t("publishGeneration.disabled")}
            />
          </div>
          <NumberInput
            label={t("publishGeneration.maxImages")}
            value={config.maxImages}
            min={0}
            max={12}
            onChange={(maxImages) => updateConfig({ maxImages })}
          />
          <NumberInput
            label={t("publishGeneration.titleMaxChars")}
            value={config.titleMaxChars}
            min={8}
            max={80}
            onChange={(titleMaxChars) => updateConfig({ titleMaxChars })}
          />
          <label className="grid gap-1.5">
            <span className="text-xs font-semibold text-[#666c78]">{t("publishGeneration.imageSelectionMode")}</span>
            <select
              value={config.imageSelectionMode ?? "ordered"}
              onChange={(event) => updateConfig({ imageSelectionMode: event.target.value })}
              className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
            >
              {["ordered", "random"].map((mode) => (
                <option key={mode} value={mode}>{t(`imageSelectionModes.${mode}`)}</option>
              ))}
            </select>
          </label>
          <TemplateSelect
            label={t("publishGeneration.detailTemplate")}
            templateKey={config.detail.templateKey}
            templateOptions={Object.entries(draft.templates).filter(([, template]) => template.taskType === "publish_detail_generate")}
            t={t}
            onTemplateChange={(templateKey) => updateConfig({ detail: { ...config.detail, enabled: true, templateKey } })}
          />
          <TemplateSelect
            label={t("publishGeneration.titleTemplate")}
            templateKey={config.title.templateKey}
            templateOptions={Object.entries(draft.templates).filter(([, template]) => template.taskType === "publish_title_generate")}
            t={t}
            onTemplateChange={(templateKey) => updateConfig({ title: { ...config.title, enabled: true, templateKey } })}
          />
        </section>
        {publishGenerationTemplateKeys.map((key) => {
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
                  onClick={() => {
                    const patch = publishGenerationTemplateResetPatch(draft, key, Boolean(template.structuredJson));
                    if (patch) {
                      onTemplateChange(key, patch);
                      return;
                    }
                    onTemplateReset(key);
                  }}
                  className="h-9 rounded-lg border-black/[0.08] bg-white px-3"
                >
                  <RotateCcw className="size-4" />
                  <span>{t("template.resetDefault")}</span>
                </Button>
              </div>
              <div className="grid gap-3 md:grid-cols-2">
                <TemplateModelInput template={template} t={t} onChange={(model) => onTemplateChange(key, { model })} />
                <NumberInput label={t("template.temperature")} value={template.temperature} min={0} max={2} step={0.1} onChange={(temperature) => onTemplateChange(key, { temperature })} />
                <NumberInput label={t("template.maxOutput")} value={template.maxOutputTokens} min={0} onChange={(maxOutputTokens) => onTemplateChange(key, { maxOutputTokens })} />
                <TemplateConcurrencyInput template={template} t={t} onChange={(concurrency) => onTemplateChange(key, { concurrency })} />
                <ToggleSwitch
                  value={Boolean(template.structuredJson)}
                  onChange={(structuredJson) => onTemplateChange(key, publishGenerationTemplateModePatch(draft, key, structuredJson))}
                  onLabel={t("template.jsonOn")}
                  offLabel={t("template.jsonOff")}
                />
                <TemplateRuntimeOverrides
                  template={template}
                  t={t}
                  onChange={(runtimeOverrides) => onTemplateChange(key, { runtimeOverrides })}
                />
                <label className="grid gap-1.5 md:col-span-2">
                  <span className="text-xs font-semibold text-[#666c78]">{t("template.systemPrompt")}</span>
                  <textarea
                    value={template.systemPrompt ?? ""}
                    onChange={(event) => onTemplateChange(key, { systemPrompt: event.target.value })}
                    className="min-h-[130px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
                    spellCheck={false}
                  />
                </label>
                <label className="grid gap-1.5 md:col-span-2">
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

function publishGenerationTemplateModePatch(draft: AIDraftSettings, key: string, structuredJson: boolean): Partial<AITemplateConfig> {
  const suffix = structuredJson ? `${key}_json` : key;
  const modeDefault = draft.defaultTemplates?.[suffix];
  if (!modeDefault) {
    return { structuredJson };
  }
  return {
    structuredJson,
    systemPrompt: modeDefault.systemPrompt,
    userPrompt: modeDefault.userPrompt ?? modeDefault.prompt,
    prompt: modeDefault.userPrompt ?? modeDefault.prompt,
  };
}

function publishGenerationTemplateResetPatch(draft: AIDraftSettings, key: string, structuredJson: boolean): Partial<AITemplateConfig> | null {
  const suffix = structuredJson ? `${key}_json` : key;
  const modeDefault = draft.defaultTemplates?.[suffix];
  return modeDefault ? structuredClone(modeDefault) : null;
}

function TemplateSelect({
  label,
  templateKey,
  templateOptions,
  t,
  onTemplateChange,
}: {
  label: string;
  templateKey: string;
  templateOptions: Array<[string, AITemplateConfig]>;
  t: AIAgentPanelT;
  onTemplateChange: (templateKey: string) => void;
}) {
  return (
    <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-white p-3">
      <h3 className="text-sm font-semibold text-[#20232a]">{label}</h3>
      <label className="grid gap-1.5">
        <span className="text-xs font-semibold text-[#666c78]">{t("publishGeneration.template")}</span>
        <select
          value={templateKey}
          onChange={(event) => onTemplateChange(event.target.value)}
          className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
        >
          {templateOptions.map(([key]) => (
            <option key={key} value={key}>{t.has(`templates.${key}`) ? t(`templates.${key}`) : key}</option>
          ))}
        </select>
      </label>
    </section>
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
