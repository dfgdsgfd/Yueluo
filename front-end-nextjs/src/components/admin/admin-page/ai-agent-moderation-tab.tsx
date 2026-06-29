"use client";

import { useState } from "react";
import { Loader2, Play, RotateCcw, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { debugAdminAIModeration } from "@/lib/api";
import type { AIModerationConfig, AIModerationDebugResult, AIModerationRuleConfig, AITemplateConfig } from "@/lib/types";
import { errorMessage } from "./helpers";
import { ToggleSwitch } from "./form-fields";
import { Panel } from "./layout-widgets";
import type { AIAgentPanelT, AIDraftSettings } from "./ai-agent-panel-model";
import { TemplateConcurrencyInput } from "./ai-template-concurrency-input";
import { TemplateModelInput } from "./ai-template-model-input";
import { TemplateRuntimeOverrides } from "./ai-template-runtime-overrides";
import { toast } from "sonner";

const moderationRules = ["spam", "porn", "political_sensitive"] as const;

export function ModerationTab({
  draft,
  saving,
  t,
  updateDraft,
  onSave,
  token,
  onTemplateChange,
  onTemplateReset,
}: {
  draft: AIDraftSettings;
  saving: boolean;
  t: AIAgentPanelT;
  updateDraft: (patch: Partial<AIDraftSettings>) => void;
  onSave: () => void;
  token: string;
  onTemplateChange: (key: string, patch: Partial<AITemplateConfig>) => void;
  onTemplateReset: (key: string) => void;
}) {
  const moderation = draft.moderation;
  const templateKeys = Array.from(new Set([
    moderation.comment.templateKey || "comment_moderation",
    moderation.post.templateKey || "post_moderation",
  ]));
  const [debugTarget, setDebugTarget] = useState<keyof AIModerationConfig>("comment");
  const [debugInput, setDebugInput] = useState("");
  const [debugLoading, setDebugLoading] = useState(false);
  const [debugResult, setDebugResult] = useState<AIModerationDebugResult | null>(null);
  const updateTarget = (
    target: keyof AIModerationConfig,
    patch: Partial<AIModerationConfig[keyof AIModerationConfig]>,
  ) => {
    updateDraft({
      moderation: {
        ...moderation,
        [target]: {
          ...moderation[target],
          ...patch,
        },
      },
    });
  };
  const updateRule = (
    target: keyof AIModerationConfig,
    ruleKey: string,
    patch: Partial<AIModerationRuleConfig>,
  ) => {
    updateTarget(target, {
      rules: {
        ...moderation[target].rules,
        [ruleKey]: {
          ...moderation[target].rules[ruleKey],
          ...patch,
        },
      },
    });
  };
  const runDebug = async () => {
    if (!debugInput.trim()) {
      toast.error(t("moderation.debugEmpty"));
      return;
    }
    const config = moderation[debugTarget];
    const template = draft.templates[config.templateKey];
    setDebugLoading(true);
    setDebugResult(null);
    try {
      const result = await debugAdminAIModeration({
        targetType: debugTarget,
        content: debugInput,
        templateKey: config.templateKey,
        systemPrompt: template?.systemPrompt ?? "",
        userPrompt: template?.userPrompt ?? template?.prompt ?? "",
        prompt: config.prompt ?? "",
        config,
      }, token);
      setDebugResult(result);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setDebugLoading(false);
    }
  };

  return (
    <div className="grid gap-4">
      <Panel
        title={t("moderation.title")}
        icon={ShieldCheck}
        action={
          <Button type="button" disabled={saving} onClick={onSave} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            {saving ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
            <span>{saving ? t("saving") : t("save")}</span>
          </Button>
        }
      >
        <div className="grid gap-4 xl:grid-cols-2">
          <ModerationTargetCard
            target="comment"
            config={moderation.comment}
            t={t}
            onTargetChange={(patch) => updateTarget("comment", patch)}
            onRuleChange={(rule, patch) => updateRule("comment", rule, patch)}
          />
          <ModerationTargetCard
            target="post"
            config={moderation.post}
            t={t}
            onTargetChange={(patch) => updateTarget("post", patch)}
            onRuleChange={(rule, patch) => updateRule("post", rule, patch)}
          />
        </div>
      </Panel>
      <Panel title={t("templatesTitle")} icon={ShieldCheck}>
        <div className="grid gap-3">
          {templateKeys.map((key) => {
            const template = draft.templates[key];
            if (!template) {
              return null;
            }
            return (
              <ModerationTemplateEditor
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
      <Panel
        title={t("moderation.debugTitle")}
        icon={Play}
        action={
          <Button type="button" disabled={debugLoading} onClick={() => void runDebug()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            {debugLoading ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
            <span>{debugLoading ? t("moderation.debugRunning") : t("moderation.debugRun")}</span>
          </Button>
        }
      >
        <div className="grid gap-3 lg:grid-cols-[minmax(0,0.42fr)_minmax(0,0.58fr)]">
          <div className="grid gap-3">
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">{t("moderation.debugTarget")}</span>
              <select
                value={debugTarget}
                onChange={(event) => setDebugTarget(event.target.value as keyof AIModerationConfig)}
                className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
              >
                <option value="comment">{t("moderation.targets.comment.title")}</option>
                <option value="post">{t("moderation.targets.post.title")}</option>
              </select>
            </label>
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">{t("moderation.debugInput")}</span>
              <textarea
                value={debugInput}
                onChange={(event) => setDebugInput(event.target.value)}
                placeholder={t("moderation.debugPlaceholder")}
                className="min-h-[180px] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 py-2 text-sm leading-6 outline-none focus:border-[#1d4ed8]"
              />
            </label>
          </div>
          <div className="grid min-h-[260px] gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            {debugResult ? (
              <>
                <div className="grid gap-2 rounded-lg border border-black/[0.06] bg-white p-3 text-sm text-[#20232a] sm:grid-cols-3">
                  <p><span className="font-semibold">{t("moderation.debugStatus")}</span> {debugResult.status}</p>
                  <p><span className="font-semibold">{t("moderation.debugAction")}</span> {debugResult.action}</p>
                  <p><span className="font-semibold">{t("moderation.debugTemplate")}</span> {debugResult.templateKey}</p>
                </div>
                <DebugBlock label={t("moderation.debugDecision")} value={JSON.stringify(debugResult.decision, null, 2)} />
                <DebugBlock label={t("moderation.debugRaw")} value={debugResult.rawOutput} />
              </>
            ) : (
              <p className="text-sm text-[#7b8190]">{t("moderation.debugEmptyOutput")}</p>
            )}
          </div>
        </div>
      </Panel>
    </div>
  );
}

function ModerationTemplateEditor({
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
      <div className="grid gap-3">
        <TemplateModelInput template={template} t={t} onChange={(model) => onTemplateChange(templateKey, { model })} />
        <TemplateConcurrencyInput template={template} t={t} onChange={(concurrency) => onTemplateChange(templateKey, { concurrency })} />
        <TemplateRuntimeOverrides
          template={template}
          t={t}
          onChange={(runtimeOverrides) => onTemplateChange(templateKey, { runtimeOverrides })}
        />
        <label className="grid gap-1.5">
          <span className="text-xs font-semibold text-[#666c78]">{t("template.systemPrompt")}</span>
          <textarea
            value={template.systemPrompt ?? ""}
            onChange={(event) => onTemplateChange(templateKey, { systemPrompt: event.target.value })}
            className="min-h-[130px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        </label>
        <label className="grid gap-1.5">
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

function DebugBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-black/[0.06] bg-white p-3">
      <p className="mb-2 text-xs font-semibold text-[#666c78]">{label}</p>
      <pre className="max-h-[220px] overflow-auto whitespace-pre-wrap break-words text-xs leading-5 text-[#20232a]">
        {value}
      </pre>
    </div>
  );
}

function ModerationTargetCard({
  target,
  config,
  t,
  onTargetChange,
  onRuleChange,
}: {
  target: keyof AIModerationConfig;
  config: AIModerationConfig[keyof AIModerationConfig];
  t: AIAgentPanelT;
  onTargetChange: (patch: Partial<AIModerationConfig[keyof AIModerationConfig]>) => void;
  onRuleChange: (rule: string, patch: Partial<AIModerationRuleConfig>) => void;
}) {
  const actions = target === "post" ? ["observe", "delete", "private"] : ["observe", "delete"];
  return (
    <section className="grid gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <div className="min-w-0">
        <h3 className="text-sm font-semibold text-[#20232a]">{t(`moderation.targets.${target}.title`)}</h3>
        <p className="mt-1 text-xs leading-5 text-[#7b8190]">{t(`moderation.targets.${target}.description`)}</p>
      </div>
      <ToggleSwitch
        value={config.enabled}
        onChange={(enabled) => onTargetChange({ enabled })}
        onLabel={t("moderation.enabled")}
        offLabel={t("moderation.disabled")}
      />
      <label className="grid gap-1.5">
        <span className="text-xs font-semibold text-[#666c78]">{t("moderation.templateKey")}</span>
        <input
          value={config.templateKey}
          onChange={(event) => onTargetChange({ templateKey: event.target.value })}
          className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
        />
      </label>
      <label className="grid gap-1.5">
        <span className="text-xs font-semibold text-[#666c78]">{t("moderation.prompt")}</span>
        <textarea
          value={config.prompt ?? ""}
          onChange={(event) => onTargetChange({ prompt: event.target.value })}
          placeholder={t("moderation.promptPlaceholder")}
          className="min-h-[120px] rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm leading-6 outline-none focus:border-[#1d4ed8]"
        />
      </label>
      <div className="grid gap-2">
        {moderationRules.map((rule) => {
          const current = config.rules[rule] ?? { enabled: true, action: actions[0] };
          return (
            <div key={rule} className="grid gap-2 rounded-lg border border-black/[0.06] bg-white p-3 sm:grid-cols-[minmax(0,1fr)_140px_120px_150px] sm:items-center">
              <div className="min-w-0">
                <p className="text-sm font-semibold text-[#20232a]">{t(`moderation.rules.${rule}`)}</p>
                <p className="mt-1 text-xs text-[#7b8190]">{rule}</p>
              </div>
              <ToggleSwitch
                value={current.enabled}
                onChange={(enabled) => onRuleChange(rule, { enabled })}
                onLabel={t("moderation.ruleEnabled")}
                offLabel={t("moderation.ruleDisabled")}
              />
              <label className="grid gap-1">
                <span className="text-[11px] font-semibold text-[#7b8190]">{t("moderation.sensitivity")}</span>
                <input
                  type="number"
                  min={0}
                  max={1}
                  step={0.05}
                  value={current.sensitivity ?? 0.65}
                  onChange={(event) => onRuleChange(rule, { sensitivity: clampRuleSensitivity(event.target.value) })}
                  className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
                />
              </label>
              <select
                value={current.action}
                onChange={(event) => onRuleChange(rule, { action: event.target.value })}
                className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
              >
                {actions.map((action) => (
                  <option key={action} value={action}>
                    {t(`moderation.actions.${action}`)}
                  </option>
                ))}
              </select>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function clampRuleSensitivity(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return 0.65;
  return Math.max(0, Math.min(1, parsed));
}
