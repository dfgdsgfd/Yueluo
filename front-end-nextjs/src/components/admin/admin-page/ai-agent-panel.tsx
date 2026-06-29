"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import {
  Bot,
  ClipboardList,
  FileText,
  Image as ImageIcon,
  Loader2,
  MessageSquareText,
  Send,
  Settings2,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  getAdminAILogs,
  getAdminAISettings,
  streamAdminAIGenerate,
  updateAdminAISettings,
} from "@/lib/api";
import type {
  AIGenerationLogsPayload,
  AIPublicSettings,
  AITemplateConfig,
} from "@/lib/types";
import { cn } from "@/lib/utils";
import { errorMessage } from "./helpers";
import { HeaderCard, Panel } from "./layout-widgets";
import { LoadingBlock } from "./resource-editor";
import {
  BaseSettingsTab,
  LogsTab,
  ProgressBar,
  TemplateTab,
} from "./ai-agent-panel-sections";
import { CommentReplyTab } from "./ai-agent-comment-reply-tab";
import { AIJobQueuePanel } from "./ai-agent-job-queue-panel";
import { ModerationTab } from "./ai-agent-moderation-tab";
import { ContentFormatTab } from "./ai-agent-content-format-tab";
import { PublishGenerationTab } from "./ai-agent-publish-generation-tab";
import {
  formatAdminQueueEta,
  formatExtraHeaders,
  formatModelParameters,
  parseExtraHeaders,
  parseModelParameters,
  toDraftSettings,
  toSettingsUpdate,
} from "./ai-agent-panel-model";
import type { AIDraftSettings } from "./ai-agent-panel-model";

const tabs = [
  { key: "base", icon: Settings2 },
  { key: "contentFormat", icon: FileText },
  { key: "review", icon: MessageSquareText },
  { key: "comments", icon: Bot },
  { key: "commentReply", icon: MessageSquareText },
  { key: "moderation", icon: ShieldCheck },
  { key: "publishGeneration", icon: Sparkles },
  { key: "vision", icon: ImageIcon },
  { key: "copy", icon: Sparkles },
  { key: "logs", icon: ClipboardList },
] as const;

type TabKey = (typeof tabs)[number]["key"];

type QueueState = {
  position: number;
  total: number;
  etaSeconds: number;
};

const templateGroups: Record<Exclude<TabKey, "base" | "logs" | "moderation" | "publishGeneration" | "commentReply" | "contentFormat">, string[]> = {
  review: ["post_review_reply"],
  comments: ["post_auto_comment"],
  vision: ["image_analysis"],
  copy: [
    "announcement",
    "system_notification",
    "popup",
    "activity_description",
    "post_title",
    "post_summary",
    "tag_suggestions",
  ],
};

export function AIAgentPanel({ token }: { token: string }) {
  const t = useTranslations("adminPortal.aiAgentPanel");
  const locale = useLocale();
  const [activeTab, setActiveTab] = useState<TabKey>("base");
  const [settings, setSettings] = useState<AIPublicSettings | null>(null);
  const [draft, setDraft] = useState<AIDraftSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [generatorTemplate, setGeneratorTemplate] = useState("announcement");
  const [generatorInput, setGeneratorInput] = useState("");
  const [generatorOutput, setGeneratorOutput] = useState("");
  const [generatorReasoning, setGeneratorReasoning] = useState("");
  const [generatorProgress, setGeneratorProgress] = useState(0);
  const [generatorQueue, setGeneratorQueue] = useState<QueueState | null>(null);
  const [logs, setLogs] = useState<AIGenerationLogsPayload | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);
  const [extraHeadersText, setExtraHeadersText] = useState("");
  const [modelParametersText, setModelParametersText] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const next = await getAdminAISettings(token);
      setSettings(next);
      setDraft(toDraftSettings(next));
      setExtraHeadersText(formatExtraHeaders(next.extraHeaders));
      setModelParametersText(formatModelParameters(next.modelParameters));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [token]);

  const loadLogs = useCallback(async () => {
    setLogsLoading(true);
    try {
      setLogs(await getAdminAILogs({ limit: 20 }, token));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLogsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) {
        void load();
      }
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  useEffect(() => {
    let cancelled = false;
    if (activeTab === "logs") {
      queueMicrotask(() => {
        if (!cancelled) {
          void loadLogs();
        }
      });
    }
    return () => {
      cancelled = true;
    };
  }, [activeTab, loadLogs]);

  const generatorOptions = useMemo(() => {
    const templates = draft?.templates ?? settings?.templates ?? {};
    return Object.keys(templates).filter((key) => templates[key]?.taskType === "admin_copy");
  }, [draft?.templates, settings?.templates]);

  async function save() {
    if (!draft) {
      return;
    }
    const extraHeaders = parseExtraHeaders(extraHeadersText);
    if (!extraHeaders) {
      toast.error(t("base.extraHeadersInvalid"));
      return;
    }
    const modelParameters = parseModelParameters(modelParametersText);
    if (!modelParameters) {
      toast.error(t("base.modelParametersInvalid"));
      return;
    }
    setSaving(true);
    try {
      const next = await updateAdminAISettings(toSettingsUpdate({ ...draft, extraHeaders, modelParameters }), token);
      setSettings(next);
      setDraft(toDraftSettings(next));
      setExtraHeadersText(formatExtraHeaders(next.extraHeaders));
      setModelParametersText(formatModelParameters(next.modelParameters));
      toast.success(t("saved"));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function generate() {
    if (!generatorInput.trim()) {
      toast.error(t("generator.emptyInput"));
      return;
    }
    setGenerating(true);
    setGeneratorOutput("");
    setGeneratorReasoning("");
    setGeneratorProgress(0);
    setGeneratorQueue(null);
    try {
      await streamAdminAIGenerate(
        {
          type: "admin_copy",
          locale,
          input: generatorInput,
          templateKey: generatorTemplate,
        },
        {
          onProgress: (event) => {
            setGeneratorProgress(event.percent);
            if (event.stage === "connecting" || event.stage === "retrying" || event.stage === "chunk_start") {
              setGeneratorOutput("");
            }
            if (event.stage === "queued") {
              const position = Number(event.queuePosition ?? 0);
              const total = Number(event.queueTotal ?? event.queuePosition ?? 0);
              setGeneratorQueue(position > 0 || total > 0
                ? {
                    position: Math.max(1, position || total || 1),
                    total: Math.max(position || 1, total || position || 1),
                    etaSeconds: Number(event.etaSeconds ?? 0),
                  }
                : null);
            } else {
              setGeneratorQueue(null);
            }
          },
          onChunkDelta: (event) => {
            setGeneratorQueue(null);
            setGeneratorOutput((current) => `${current}${event.delta}`);
          },
          onReasoningDelta: (event) => {
            setGeneratorReasoning((current) => `${current}${event.reasoningDelta}`);
          },
          onFinal: (event) => {
            setGeneratorOutput(event.text);
            setGeneratorProgress(100);
            setGeneratorQueue(null);
          },
        },
        { token },
      );
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setGenerating(false);
    }
  }

  const updateDraft = (patch: Partial<AIDraftSettings>) => {
    setDraft((current) => current ? { ...current, ...patch } : current);
  };

  const updateTemplate = (key: string, patch: Partial<AITemplateConfig>) => {
    setDraft((current) => {
      if (!current) return current;
      const previous = current.templates[key];
      if (!previous) return current;
      return {
        ...current,
        templates: {
          ...current.templates,
          [key]: { ...previous, ...patch },
        },
      };
    });
  };

  const resetTemplate = (key: string) => {
    setDraft((current) => {
      if (!current) return current;
      const nextTemplate = current.defaultTemplates?.[key] ?? settings?.defaultTemplates?.[key];
      if (!nextTemplate) return current;
      return {
        ...current,
        templates: {
          ...current.templates,
          [key]: structuredClone(nextTemplate),
        },
      };
    });
  };

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Sparkles} title={t("title")} description={t("description")} tone="blue" />
      <AIJobQueuePanel token={token} t={t} />
      <div className="flex min-w-0 gap-2 overflow-x-auto rounded-lg border border-black/[0.06] bg-white p-2">
        {tabs.map(({ key, icon: Icon }) => (
          <button
            key={key}
            type="button"
            onClick={() => setActiveTab(key)}
            className={cn(
              "flex h-9 shrink-0 items-center gap-2 rounded-lg px-3 text-sm font-semibold transition",
              activeTab === key ? "bg-[#1d4ed8] text-white" : "text-[#5f6673] hover:bg-[#f4f6f8]",
            )}
          >
            <Icon className="size-4" />
            <span>{t(`tabs.${key}`)}</span>
          </button>
        ))}
      </div>

      {loading || !draft ? (
        <LoadingBlock label={t("loading")} />
      ) : activeTab === "base" ? (
        <BaseSettingsTab
          draft={draft}
          extraHeadersText={extraHeadersText}
          modelParametersText={modelParametersText}
          saving={saving}
          t={t}
          updateExtraHeadersText={setExtraHeadersText}
          updateModelParametersText={setModelParametersText}
          updateDraft={updateDraft}
          onSave={() => void save()}
        />
      ) : activeTab === "logs" ? (
        <LogsTab logs={logs?.items ?? []} loading={logsLoading} t={t} onRefresh={() => void loadLogs()} />
      ) : activeTab === "moderation" ? (
        <ModerationTab
          draft={draft}
          saving={saving}
          t={t}
          updateDraft={updateDraft}
          onSave={() => void save()}
          token={token}
          onTemplateChange={updateTemplate}
          onTemplateReset={resetTemplate}
        />
      ) : activeTab === "commentReply" ? (
        <CommentReplyTab
          draft={draft}
          saving={saving}
          t={t}
          updateDraft={updateDraft}
          onTemplateChange={updateTemplate}
          onTemplateReset={resetTemplate}
          onSave={() => void save()}
        />
      ) : activeTab === "contentFormat" ? (
        <ContentFormatTab
          draft={draft}
          saving={saving}
          t={t}
          updateDraft={updateDraft}
          onTemplateChange={updateTemplate}
          onTemplateReset={resetTemplate}
          onSave={() => void save()}
        />
      ) : activeTab === "publishGeneration" ? (
        <PublishGenerationTab
          draft={draft}
          saving={saving}
          t={t}
          updateDraft={updateDraft}
          onTemplateChange={updateTemplate}
          onTemplateReset={resetTemplate}
          onSave={() => void save()}
        />
      ) : (
        <TemplateTab
          keys={templateGroups[activeTab]}
          settings={draft}
          t={t}
          onTemplateChange={updateTemplate}
          onTemplateReset={resetTemplate}
          onSave={() => void save()}
          saving={saving}
        />
      )}

      <Panel
        title={t("generator.title")}
        icon={Send}
        action={
          <Button type="button" disabled={generating} onClick={() => void generate()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            {generating ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
            <span>{generating ? t("generator.generating") : t("generator.run")}</span>
          </Button>
        }
      >
        <div className="grid gap-3 lg:grid-cols-[minmax(0,0.42fr)_minmax(0,0.58fr)]">
          <div className="grid gap-3">
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">{t("generator.template")}</span>
              <select
                value={generatorTemplate}
                onChange={(event) => setGeneratorTemplate(event.target.value)}
                className="h-10 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
              >
                {generatorOptions.map((key) => (
                  <option key={key} value={key}>{t(`templates.${key}`)}</option>
                ))}
              </select>
            </label>
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#666c78]">{t("generator.input")}</span>
              <textarea
                value={generatorInput}
                onChange={(event) => setGeneratorInput(event.target.value)}
                placeholder={t("generator.inputPlaceholder")}
                className="min-h-[180px] rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 py-2 text-sm outline-none focus:border-[#1d4ed8]"
              />
            </label>
            <ProgressBar
              percent={generatorProgress}
              label={
                generatorQueue
                  ? t("generator.queue", {
                      position: generatorQueue.position,
                      total: generatorQueue.total,
                      eta: formatAdminQueueEta(t, generatorQueue.etaSeconds),
                    })
                  : t("generator.progress", { percent: generatorProgress })
              }
            />
          </div>
          <div className="grid min-h-[260px] gap-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            {generatorReasoning ? (
              <div className="rounded-lg border border-[#1d4ed8]/15 bg-white p-3">
                <p className="mb-2 text-xs font-semibold text-[#1d4ed8]">{t("generator.reasoning")}</p>
                <pre className="max-h-[180px] overflow-auto whitespace-pre-wrap break-words text-xs leading-5 text-[#4d5562]">
                  {generatorReasoning}
                </pre>
              </div>
            ) : null}
            <div>
              <p className="mb-2 text-xs font-semibold text-[#666c78]">{t("generator.output")}</p>
              <pre className="max-h-[360px] overflow-auto whitespace-pre-wrap break-words text-sm leading-6 text-[#20232a]">
                {generatorOutput || t("generator.emptyOutput")}
              </pre>
            </div>
          </div>
        </div>
      </Panel>
    </div>
  );
}
