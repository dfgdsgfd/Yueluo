"use client";

import { Loader2, MessageSquareText, RotateCcw, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AITemplateConfig } from "@/lib/types";
import { ToggleSwitch } from "./form-fields";
import { Panel } from "./layout-widgets";
import type { AIAgentPanelT, AIDraftSettings } from "./ai-agent-panel-model";
import { TemplateConcurrencyInput } from "./ai-template-concurrency-input";
import { TemplateModelInput } from "./ai-template-model-input";
import { TemplateRuntimeOverrides } from "./ai-template-runtime-overrides";

export function CommentReplyTab({
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
  const config = draft.commentReply;
  const updateConfig = (patch: Partial<AIDraftSettings["commentReply"]>) => {
    updateDraft({ commentReply: { ...config, ...patch } });
  };
  const templateKeys = Array.from(new Set([
    config.templateKey || "comment_reply",
    config.mentionTemplateKey || "comment_mention_reply",
    "comment_reply",
    "comment_mention_reply",
  ]));

  return (
    <Panel
      title={t("commentReply.title")}
      icon={MessageSquareText}
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
              onLabel={t("commentReply.enabled")}
              offLabel={t("commentReply.disabled")}
            />
          </div>
          <TemplateSelect
            label={t("commentReply.template")}
            templateKey={config.templateKey}
            templateOptions={Object.entries(draft.templates).filter(([, template]) => template.taskType === "comment_reply")}
            t={t}
            onTemplateChange={(templateKey) => updateConfig({ templateKey })}
          />
          <NumberInput
            label={t("commentReply.maxReplies")}
            value={config.maxRepliesPerAIComment}
            min={1}
            max={20}
            onChange={(maxRepliesPerAIComment) => updateConfig({ maxRepliesPerAIComment })}
          />
          <NumberInput
            label={t("commentReply.delaySeconds")}
            value={config.delaySeconds}
            min={0}
            max={3600}
            onChange={(delaySeconds) => updateConfig({ delaySeconds })}
          />
          <NumberInput
            label={t("commentReply.maxImages")}
            value={config.maxImages}
            min={0}
            max={12}
            onChange={(maxImages) => updateConfig({ maxImages })}
          />
          <label className="grid gap-1.5">
            <span className="text-xs font-semibold text-[#666c78]">{t("commentReply.imageSelectionMode")}</span>
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
          <label className="grid gap-1.5">
            <span className="text-xs font-semibold text-[#666c78]">{t("commentReply.style")}</span>
            <select
              value={config.style}
              onChange={(event) => updateConfig({ style: event.target.value })}
              className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
            >
              {["normal", "humorous", "bold"].map((style) => (
                <option key={style} value={style}>{t(`template.styles.${style}`)}</option>
              ))}
            </select>
          </label>
        </section>
        <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:grid-cols-2">
          <div className="md:col-span-2">
            <ToggleSwitch
              value={config.mentionEnabled}
              onChange={(mentionEnabled) => updateConfig({ mentionEnabled })}
              onLabel={t("commentReply.mentionEnabled")}
              offLabel={t("commentReply.mentionDisabled")}
            />
          </div>
          <TextInput
            label={t("commentReply.mentionName")}
            value={config.mentionName}
            onChange={(mentionName) => updateConfig({ mentionName: mentionName.trim().replace(/^@+/, "") })}
          />
          <TemplateSelect
            label={t("commentReply.mentionTemplate")}
            templateKey={config.mentionTemplateKey}
            templateOptions={Object.entries(draft.templates).filter(([, template]) => template.taskType === "comment_mention_reply")}
            t={t}
            onTemplateChange={(mentionTemplateKey) => updateConfig({ mentionTemplateKey })}
          />
          <NumberInput
            label={t("commentReply.mentionBotUserIdMin")}
            value={config.mentionBotUserIdMin}
            min={0}
            max={999999999}
            onChange={(mentionBotUserIdMin) => updateConfig({ mentionBotUserIdMin })}
          />
          <NumberInput
            label={t("commentReply.mentionBotUserIdMax")}
            value={config.mentionBotUserIdMax}
            min={0}
            max={999999999}
            onChange={(mentionBotUserIdMax) => updateConfig({ mentionBotUserIdMax })}
          />
          <NumberInput
            label={t("commentReply.maxMentionReplies")}
            value={config.maxMentionRepliesPerPost}
            min={1}
            max={20}
            onChange={(maxMentionRepliesPerPost) => updateConfig({ maxMentionRepliesPerPost })}
          />
        </section>
        {templateKeys.map((key) => {
          const template = draft.templates[key];
          if (!template) {
            return null;
          }
          return (
            <CommentReplyTemplateEditor
              key={key}
              templateKey={key}
              template={template}
              t={t}
              onTemplateChange={onTemplateChange}
              onTemplateReset={onTemplateReset}
            />
          );
        })}
      </div>
    </Panel>
  );
}

function CommentReplyTemplateEditor({
  templateKey,
  template,
  t,
  onTemplateChange,
  onTemplateReset,
}: {
  templateKey: string;
  template: AITemplateConfig;
  t: AIAgentPanelT;
  onTemplateChange: (key: string, patch: Partial<AITemplateConfig>) => void;
  onTemplateReset: (key: string) => void;
}) {
  return (
    <section className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <div className="mb-3 flex min-w-0 flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-semibold text-[#20232a]">
            {t.has(`templates.${templateKey}`) ? t(`templates.${templateKey}`) : templateKey}
          </h3>
          <p className="truncate text-xs text-[#7b8190]">{templateKey}</p>
        </div>
        <div className="w-40">
          <ToggleSwitch
            value={template.enabled}
            onChange={(enabled) => onTemplateChange(templateKey, { enabled })}
            onLabel={t("template.enabled")}
            offLabel={t("template.disabled")}
          />
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={() => onTemplateReset(templateKey)}
          className="h-9 rounded-lg border-black/[0.08] bg-white px-3"
        >
          <RotateCcw className="size-4" />
          <span>{t("template.resetDefault")}</span>
        </Button>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <TemplateModelInput template={template} t={t} onChange={(model) => onTemplateChange(templateKey, { model })} />
        <NumberInput label={t("template.temperature")} value={template.temperature} min={0} max={2} step={0.1} onChange={(temperature) => onTemplateChange(templateKey, { temperature })} />
        <NumberInput label={t("template.maxOutput")} value={template.maxOutputTokens} min={0} onChange={(maxOutputTokens) => onTemplateChange(templateKey, { maxOutputTokens })} />
        <TemplateConcurrencyInput template={template} t={t} onChange={(concurrency) => onTemplateChange(templateKey, { concurrency })} />
        <TemplateRuntimeOverrides
          template={template}
          t={t}
          onChange={(runtimeOverrides) => onTemplateChange(templateKey, { runtimeOverrides })}
        />
        <label className="grid gap-1.5 md:col-span-2">
          <span className="text-xs font-semibold text-[#666c78]">{t("template.systemPrompt")}</span>
          <textarea
            value={template.systemPrompt ?? ""}
            onChange={(event) => onTemplateChange(templateKey, { systemPrompt: event.target.value })}
            className="min-h-[130px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        </label>
        <label className="grid gap-1.5 md:col-span-2">
          <span className="text-xs font-semibold text-[#666c78]">{t("template.userPrompt")}</span>
          <textarea
            value={template.userPrompt ?? template.prompt}
            onChange={(event) => onTemplateChange(templateKey, { userPrompt: event.target.value, prompt: event.target.value })}
            className="min-h-[180px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        </label>
      </div>
    </section>
  );
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
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <select
        value={templateKey}
        onChange={(event) => onTemplateChange(event.target.value)}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      >
        {templateOptions.map(([key]) => (
          <option key={key} value={key}>{t.has(`templates.${key}`) ? t(`templates.${key}`) : key}</option>
        ))}
      </select>
    </label>
  );
}

function TextInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <input
        value={value}
        type="text"
        onChange={(event) => onChange(event.target.value)}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      />
    </label>
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
