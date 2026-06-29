"use client";

import { ChevronDown, ChevronUp, ClipboardList, FileText, Loader2, RotateCcw, Save, Settings2 } from "lucide-react";
import { Fragment, useState } from "react";
import { Button } from "@/components/ui/button";
import type { AIGenerationLogItem, AITemplateConfig } from "@/lib/types";
import { ToggleSwitch } from "./form-fields";
import { Panel } from "./layout-widgets";
import { EmptyBlock, LoadingBlock } from "./resource-editor";
import type { AIAgentPanelT, AIDraftSettings } from "./ai-agent-panel-model";
import { aiLogActorLabel, aiLogStatusLabel, aiLogTypeLabel } from "./ai-agent-panel-model";
import { TemplateConcurrencyInput } from "./ai-template-concurrency-input";
import { TemplateModelInput } from "./ai-template-model-input";
import { TemplateRuntimeOverrides } from "./ai-template-runtime-overrides";

export function BaseSettingsTab({
  draft,
  extraHeadersText,
  modelParametersText,
  saving,
  t,
  updateExtraHeadersText,
  updateModelParametersText,
  updateDraft,
  onSave,
}: {
  draft: AIDraftSettings;
  extraHeadersText: string;
  modelParametersText: string;
  saving: boolean;
  t: AIAgentPanelT;
  updateExtraHeadersText: (value: string) => void;
  updateModelParametersText: (value: string) => void;
  updateDraft: (patch: Partial<AIDraftSettings>) => void;
  onSave: () => void;
}) {
  return (
    <Panel
      title={t("base.title")}
      icon={Settings2}
      action={<SaveButton saving={saving} label={t("save")} savingLabel={t("saving")} onClick={onSave} />}
    >
      <div className="grid gap-3 md:grid-cols-2">
        <div className="md:col-span-2">
          <ToggleSwitch
            value={draft.enabled}
            onChange={(enabled) => updateDraft({ enabled })}
            onLabel={t("base.enabled")}
            offLabel={t("base.disabled")}
          />
        </div>
        <TextInput label={t("base.baseUrl")} value={draft.baseUrl} onChange={(baseUrl) => updateDraft({ baseUrl })} />
        <TextInput label={t("base.model")} value={draft.model} onChange={(model) => updateDraft({ model })} />
        <TextInput label={t("base.apiKey")} value={draft.apiKey ?? ""} placeholder={draft.apiKeySet ? draft.apiKeyMasked : t("base.apiKeyPlaceholder")} onChange={(apiKey) => updateDraft({ apiKey })} />
        <NumberInput label={t("base.timeout")} value={draft.timeoutSeconds} min={5} max={300} onChange={(timeoutSeconds) => updateDraft({ timeoutSeconds })} />
        <NumberInput label={t("base.maxRun")} value={draft.maxRunSeconds} min={0} max={86400} onChange={(maxRunSeconds) => updateDraft({ maxRunSeconds })} />
        <NumberInput label={t("base.chunkSize")} value={draft.chunkMaxChars} min={0} max={20000} onChange={(chunkMaxChars) => updateDraft({ chunkMaxChars })} />
        <NumberInput label={t("base.concurrency")} value={draft.concurrency} min={1} max={50} onChange={(concurrency) => updateDraft({ concurrency })} />
        <NumberInput label={t("base.temperature")} value={draft.temperature} min={0} max={2} step={0.1} onChange={(temperature) => updateDraft({ temperature })} />
        <NumberInput label={t("base.maxOutput")} value={draft.maxOutputTokens} min={0} onChange={(maxOutputTokens) => updateDraft({ maxOutputTokens })} />
        <ToggleSwitch
          value={draft.showReasoning}
          onChange={(showReasoning) => updateDraft({ showReasoning })}
          onLabel={t("base.showReasoningOn")}
          offLabel={t("base.showReasoningOff")}
        />
        <ToggleSwitch
          value={draft.thinkingParameterEnabled}
          onChange={(thinkingParameterEnabled) => updateDraft({ thinkingParameterEnabled })}
          onLabel={t("base.thinkingParameterOn")}
          offLabel={t("base.thinkingParameterOff")}
        />
        <ToggleSwitch
          value={draft.thinkingEnabled}
          onChange={(thinkingEnabled) => updateDraft({ thinkingEnabled })}
          onLabel={t("base.thinkingOn")}
          offLabel={t("base.thinkingOff")}
        />
        <ToggleSwitch
          value={draft.logHttpDetails}
          onChange={(logHttpDetails) => updateDraft({ logHttpDetails })}
          onLabel={t("base.logHttpDetailsOn")}
          offLabel={t("base.logHttpDetailsOff")}
        />
        <label className="grid gap-1.5">
          <span className="text-xs font-semibold text-[#666c78]">{t("base.reasoningEffort")}</span>
          <select
            value={draft.reasoningEffort}
            onChange={(event) => updateDraft({ reasoningEffort: event.target.value })}
            className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
          >
            {["", "minimal", "low", "medium", "high"].map((value) => (
              <option key={value || "default"} value={value}>
                {t(`base.reasoningEfforts.${value || "default"}`)}
              </option>
            ))}
          </select>
        </label>
        <div className="md:col-span-2">
          <ToggleSwitch
            value={draft.autoComment.enabled}
            onChange={(enabled) => updateDraft({ autoComment: { ...draft.autoComment, enabled } })}
            onLabel={t("autoComment.enabled")}
            offLabel={t("autoComment.disabled")}
          />
        </div>
        <NumberInput
          label={t("autoComment.botUserId")}
          value={draft.autoComment.botUserId}
          min={0}
          max={999999999}
          onChange={(botUserId) => updateDraft({ autoComment: { ...draft.autoComment, botUserId } })}
        />
        <NumberInput
          label={t("autoComment.botUserIdMin")}
          value={draft.autoComment.botUserIdMin}
          min={0}
          max={999999999}
          onChange={(botUserIdMin) => updateDraft({ autoComment: { ...draft.autoComment, botUserIdMin } })}
        />
        <NumberInput
          label={t("autoComment.botUserIdMax")}
          value={draft.autoComment.botUserIdMax}
          min={0}
          max={999999999}
          onChange={(botUserIdMax) => updateDraft({ autoComment: { ...draft.autoComment, botUserIdMax } })}
        />
        <label className="grid gap-1.5">
          <span className="text-xs font-semibold text-[#666c78]">{t("autoComment.template")}</span>
          <select
            value={draft.autoComment.templateKey}
            onChange={(event) => updateDraft({ autoComment: { ...draft.autoComment, templateKey: event.target.value } })}
            className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
          >
            {Object.entries(draft.templates)
              .filter(([, template]) => template.taskType === "post_auto_comment")
              .map(([key]) => (
                <option key={key} value={key}>{t.has(`templates.${key}`) ? t(`templates.${key}`) : key}</option>
              ))}
          </select>
        </label>
        <NumberInput
          label={t("autoComment.delaySeconds")}
          value={draft.autoComment.delaySeconds}
          min={0}
          max={3600}
          onChange={(delaySeconds) => updateDraft({ autoComment: { ...draft.autoComment, delaySeconds } })}
        />
        <NumberInput
          label={t("autoComment.maxImages")}
          value={draft.autoComment.maxImages}
          min={0}
          max={12}
          onChange={(maxImages) => updateDraft({ autoComment: { ...draft.autoComment, maxImages } })}
        />
        <label className="grid gap-1.5">
          <span className="text-xs font-semibold text-[#666c78]">{t("autoComment.imageSelectionMode")}</span>
          <select
            value={draft.autoComment.imageSelectionMode ?? "ordered"}
            onChange={(event) => updateDraft({ autoComment: { ...draft.autoComment, imageSelectionMode: event.target.value } })}
            className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
          >
            {["ordered", "random"].map((mode) => (
              <option key={mode} value={mode}>{t(`imageSelectionModes.${mode}`)}</option>
            ))}
          </select>
        </label>
        <label className="grid gap-1.5">
          <span className="text-xs font-semibold text-[#666c78]">{t("autoComment.style")}</span>
          <select
            value={draft.autoComment.style}
            onChange={(event) => updateDraft({ autoComment: { ...draft.autoComment, style: event.target.value } })}
            className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
          >
            {["normal", "humorous", "bold"].map((style) => (
              <option key={style} value={style}>{t(`autoComment.styles.${style}`)}</option>
            ))}
          </select>
        </label>
        <label className="grid gap-1.5 md:col-span-2">
          <span className="text-xs font-semibold text-[#666c78]">{t("base.extraHeaders")}</span>
          <textarea
            value={extraHeadersText}
            onChange={(event) => updateExtraHeadersText(event.target.value)}
            placeholder={t("base.extraHeadersPlaceholder")}
            className="min-h-[110px] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        </label>
        <label className="grid gap-1.5 md:col-span-2">
          <span className="text-xs font-semibold text-[#666c78]">{t("base.modelParameters")}</span>
          <textarea
            value={modelParametersText}
            onChange={(event) => updateModelParametersText(event.target.value)}
            placeholder={t("base.modelParametersPlaceholder")}
            className="min-h-[110px] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 py-2 font-mono text-sm leading-6 outline-none focus:border-[#1d4ed8]"
            spellCheck={false}
          />
        </label>
      </div>
    </Panel>
  );
}

export function TemplateTab({
  keys,
  settings,
  t,
  onTemplateChange,
  onTemplateReset,
  onSave,
  saving,
}: {
  keys: string[];
  settings: AIDraftSettings;
  t: AIAgentPanelT;
  onTemplateChange: (key: string, patch: Partial<AITemplateConfig>) => void;
  onTemplateReset: (key: string) => void;
  onSave: () => void;
  saving: boolean;
}) {
  return (
    <Panel
      title={t("templatesTitle")}
      icon={FileText}
      action={<SaveButton saving={saving} label={t("save")} savingLabel={t("saving")} onClick={onSave} />}
    >
      <div className="grid gap-3">
        {keys.map((key) => {
          const template = settings.templates[key];
          if (!template) {
            return null;
          }
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
                <label className="grid gap-1.5">
                  <span className="text-xs font-semibold text-[#666c78]">{t("template.style")}</span>
                  <select
                    value={template.style ?? "normal"}
                    onChange={(event) => onTemplateChange(key, { style: event.target.value })}
                    className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
                  >
                    {["normal", "humorous", "bold"].map((style) => (
                      <option key={style} value={style}>{t(`template.styles.${style}`)}</option>
                    ))}
                  </select>
                </label>
                <ToggleSwitch
                  value={Boolean(template.structuredJson)}
                  onChange={(structuredJson) => onTemplateChange(key, { structuredJson })}
                  onLabel={t("template.jsonOn")}
                  offLabel={t("template.jsonOff")}
                />
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

export function LogsTab({
  logs,
  loading,
  t,
  onRefresh,
}: {
  logs: AIGenerationLogItem[];
  loading: boolean;
  t: AIAgentPanelT;
  onRefresh: () => void;
}) {
  const [expandedLogId, setExpandedLogId] = useState<number | null>(null);
  return (
    <Panel
      title={t("logs.title")}
      icon={ClipboardList}
      action={
        <Button type="button" variant="outline" onClick={onRefresh} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
          <RotateCcw className="size-4" />
          <span>{t("logs.refresh")}</span>
        </Button>
      }
    >
      {loading ? (
        <LoadingBlock label={t("logs.loading")} />
      ) : logs.length === 0 ? (
        <EmptyBlock icon={ClipboardList} label={t("logs.empty")} />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1040px] text-left text-sm">
            <thead className="text-xs uppercase text-[#7b8190]">
              <tr>
                <th className="px-3 py-2">{t("logs.columns.type")}</th>
                <th className="px-3 py-2">{t("logs.columns.status")}</th>
                <th className="px-3 py-2">{t("logs.columns.actor")}</th>
                <th className="px-3 py-2">{t("logs.columns.model")}</th>
                <th className="px-3 py-2">{t("logs.columns.imagesSent")}</th>
                <th className="px-3 py-2">{t("logs.columns.tokens")}</th>
                <th className="px-3 py-2">{t("logs.columns.tokenSpeed")}</th>
                <th className="px-3 py-2">{t("logs.columns.duration")}</th>
                <th className="px-3 py-2">{t("logs.columns.error")}</th>
                <th className="px-3 py-2">{t("logs.columns.details")}</th>
                <th className="px-3 py-2">{t("logs.columns.created")}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((row) => {
                const expanded = expandedLogId === row.id;
                return (
                  <Fragment key={row.id}>
                    <tr key={row.id} className="border-t border-black/[0.06]">
                      <td className="px-3 py-2 font-medium text-[#20232a]">{aiLogTypeLabel(t, row.type)}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{aiLogStatusLabel(t, row.status)}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{row.actorDisplayId ?? aiLogActorLabel(t, row.actorType)}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{row.model}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{aiLogImageSendCount(row)}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{row.totalTokens}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{formatTokenSpeed(row.tokensPerSecond)}</td>
                      <td className="px-3 py-2 text-[#5f6673]">{t("logs.durationMs", { value: row.durationMs })}</td>
                      <td className="max-w-[260px] px-3 py-2 text-[#991b1b]">
                        <span className="line-clamp-2 break-words">{row.errorMessage || row.errorCode || "-"}</span>
                      </td>
                      <td className="px-3 py-2">
                        <Button
                          type="button"
                          variant="outline"
                          aria-expanded={expanded}
                          onClick={() => setExpandedLogId(expanded ? null : row.id)}
                          className="h-8 rounded-lg border-black/[0.08] bg-white px-2 text-xs"
                        >
                          {expanded ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
                          <span>{expanded ? t("logs.details.hide") : t("logs.details.show")}</span>
                        </Button>
                      </td>
                      <td className="px-3 py-2 text-[#5f6673]">{new Date(row.createdAt).toLocaleString()}</td>
                    </tr>
                    {expanded ? (
                      <tr key={`${row.id}-details`} className="border-t border-black/[0.06] bg-[#f8fbff]">
                        <td colSpan={11} className="px-3 py-3">
                          <AILogDetails row={row} t={t} />
                        </td>
                      </tr>
                    ) : null}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}

function AILogDetails({ row, t }: { row: AIGenerationLogItem; t: AIAgentPanelT }) {
  const attempts = aiLogUpstreamAttempts(row);
  const latestAttempt = attempts.at(-1);
  const upstreamUsage = aiLogRecord(row.metadata?.upstreamUsage) ?? aiLogRecord(latestAttempt?.usage);
  const upstreamStatus = aiLogValueText(row.metadata?.upstreamStatus ?? latestAttempt?.status);
  const contentType = aiLogValueText(row.metadata?.upstreamContentType ?? latestAttempt?.contentType);
  const responseSummary = aiLogValueText(row.metadata?.upstreamResponseSummary ?? latestAttempt?.responseSummary);
  const upstreamOutput = aiLogValueText(row.metadata?.upstreamOutputSummary ?? latestAttempt?.outputSummary);

  return (
    <div className="grid gap-3 text-xs text-[#4b5563]">
      <div className="grid gap-2 md:grid-cols-4">
        <AILogDetailMetric label={t("logs.details.httpStatus")} value={upstreamStatus} />
        <AILogDetailMetric label={t("logs.details.contentType")} value={contentType} />
        <AILogDetailMetric label={t("logs.details.usage")} value={formatAIUsage(t, upstreamUsage, row.totalTokens)} />
        <AILogDetailMetric label={t("logs.details.attempts")} value={String(attempts.length)} />
      </div>
      <AILogDetailBlock label={t("logs.details.outputSummary")} value={row.outputSummary || upstreamOutput} />
      <AILogDetailBlock label={t("logs.details.responseSummary")} value={responseSummary} />
      {attempts.length > 0 ? (
        <div className="grid gap-2">
          <p className="font-semibold text-[#20232a]">{t("logs.details.upstreamAttempts")}</p>
          <div className="grid gap-2">
            {attempts.map((attempt, index) => (
              <pre key={index} className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-black/[0.06] bg-white p-3 font-mono text-[11px] leading-5 text-[#334155]">
                {formatAILogJSON(attempt)}
              </pre>
            ))}
          </div>
        </div>
      ) : null}
      <AILogDetailBlock label={t("logs.details.rawMetadata")} value={formatAILogJSON(row.metadata ?? {})} />
    </div>
  );
}

function AILogDetailMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg border border-black/[0.06] bg-white p-3">
      <p className="truncate font-semibold text-[#20232a]">{label}</p>
      <p className="mt-1 break-words font-mono text-[11px] leading-5 text-[#475569]">{value || "-"}</p>
    </div>
  );
}

function AILogDetailBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1.5">
      <p className="font-semibold text-[#20232a]">{label}</p>
      <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-black/[0.06] bg-white p-3 font-mono text-[11px] leading-5 text-[#334155]">
        {value || "-"}
      </pre>
    </div>
  );
}

function aiLogImageSendCount(row: AIGenerationLogItem) {
  const value = row.metadata?.imageSendSuccessCount;
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function aiLogUpstreamAttempts(row: AIGenerationLogItem) {
  const attempts = row.metadata?.upstreamAttempts;
  if (!Array.isArray(attempts)) {
    return [];
  }
  return attempts.map(aiLogRecord).filter((item): item is Record<string, unknown> => Boolean(item));
}

function aiLogRecord(value: unknown) {
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null;
}

function aiLogValueText(value: unknown) {
  if (value === null || value === undefined || value === "") {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return formatAILogJSON(value);
}

function formatAIUsage(t: AIAgentPanelT, usage: Record<string, unknown> | null, fallbackTotal: number) {
  const prompt = Number(usage?.promptTokens ?? 0);
  const completion = Number(usage?.completionTokens ?? 0);
  const total = Number(usage?.totalTokens ?? fallbackTotal ?? 0);
  return t("logs.details.usageValue", {
    prompt: Number.isFinite(prompt) ? prompt : 0,
    completion: Number.isFinite(completion) ? completion : 0,
    total: Number.isFinite(total) ? total : 0,
  });
}

function formatAILogJSON(value: unknown) {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function formatTokenSpeed(value?: number) {
  if (!value || !Number.isFinite(value) || value <= 0) {
    return "-";
  }
  return value.toFixed(1);
}

export function ProgressBar({ percent, label }: { percent: number; label: string }) {
  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-xs font-semibold text-[#666c78]">
        <span>{label}</span>
        <span>{percent}%</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-[#e8ecf3]">
        <div className="h-full rounded-full bg-[#1d4ed8]" style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} />
      </div>
    </div>
  );
}

function SaveButton({ saving, label, savingLabel, onClick }: { saving: boolean; label: string; savingLabel: string; onClick: () => void }) {
  return (
    <Button type="button" disabled={saving} onClick={onClick} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
      {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
      <span>{saving ? savingLabel : label}</span>
    </Button>
  );
}

function TextInput({ label, value, placeholder, onChange }: { label: string; value: string; placeholder?: string; onChange: (value: string) => void }) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{label}</span>
      <input
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
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
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
      />
    </label>
  );
}
