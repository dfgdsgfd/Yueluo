"use client";

import { Activity, ChevronDown, ChevronUp, Loader2, RotateCcw, Terminal, TimerReset, XCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { cancelAdminAIJob, getAdminAIJobs, streamAdminAIJob } from "@/lib/api";
import type { AIJobListPayload, AIJobPayload, AIUpstreamEvent } from "@/lib/types";
import { cn } from "@/lib/utils";
import { errorMessage } from "./helpers";
import { Panel } from "./layout-widgets";
import {
  aiJobStatusLabel,
  aiJobTypeLabel,
  formatAdminQueueEta,
  formatAdminRelativeTime,
  type AIAgentPanelT,
} from "./ai-agent-panel-model";

const activeStatuses = new Set(["queued", "running"]);

export function AIJobQueuePanel({ token, t }: { token: string; t: AIAgentPanelT }) {
  const [payload, setPayload] = useState<AIJobListPayload | null>(null);
  const [loading, setLoading] = useState(false);
  const [cancelingJobId, setCancelingJobId] = useState("");
  const [expandedJobId, setExpandedJobId] = useState("");

  const loadJobs = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      setPayload(await getAdminAIJobs({ status: "active", limit: 20 }, token));
    } catch (error) {
      if (!silent) {
        toast.error(errorMessage(error));
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, [token]);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void loadJobs();
    });
    const timer = window.setInterval(() => {
      if (!cancelled) void loadJobs(true);
    }, 5000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [loadJobs]);

  async function cancelJob(job: AIJobPayload) {
    setCancelingJobId(job.jobId);
    try {
      await cancelAdminAIJob(job.jobId, token);
      toast.success(t("jobs.cancelSuccess"));
      await loadJobs(true);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setCancelingJobId("");
    }
  }

  const jobs = payload?.items ?? [];
  const stats = payload?.stats ?? {};
  const activeCount = Number(stats.active ?? jobs.length ?? 0);

  return (
    <Panel
      title={t("jobs.title")}
      icon={Activity}
      action={
        <Button type="button" variant="outline" onClick={() => void loadJobs()} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
          {loading ? <Loader2 className="size-4 animate-spin" /> : <RotateCcw className="size-4" />}
          <span>{t("jobs.refresh")}</span>
        </Button>
      }
    >
      <div className="mb-3 grid gap-2 sm:grid-cols-3">
        <AIJobMetric label={t("jobs.metrics.active")} value={activeCount} tone="blue" />
        <AIJobMetric label={t("jobs.metrics.queued")} value={Number(stats.queued ?? 0)} tone="amber" />
        <AIJobMetric label={t("jobs.metrics.running")} value={Number(stats.running ?? 0)} tone="green" />
      </div>

      {loading && jobs.length === 0 ? (
        <div className="flex min-h-28 items-center justify-center rounded-lg border border-dashed border-black/[0.08] bg-[#fafbfe] text-sm font-semibold text-[#6b7280]">
          <Loader2 className="mr-2 size-4 animate-spin" />
          {t("jobs.loading")}
        </div>
      ) : jobs.length === 0 ? (
        <div className="flex min-h-28 items-center justify-center rounded-lg border border-dashed border-black/[0.08] bg-[#fafbfe] text-sm font-semibold text-[#6b7280]">
          {t("jobs.empty")}
        </div>
      ) : (
        <div className="grid gap-2">
          {jobs.map((job) => (
            <AIJobQueueItem
              key={job.jobId}
              canceling={cancelingJobId === job.jobId}
              expanded={expandedJobId === job.jobId}
              job={job}
              t={t}
              token={token}
              onCancel={() => void cancelJob(job)}
              onToggle={() => setExpandedJobId((current) => current === job.jobId ? "" : job.jobId)}
            />
          ))}
        </div>
      )}
    </Panel>
  );
}

function AIJobQueueItem({
  canceling,
  expanded,
  job,
  t,
  token,
  onCancel,
  onToggle,
}: {
  canceling: boolean;
  expanded: boolean;
  job: AIJobPayload;
  t: AIAgentPanelT;
  token: string;
  onCancel: () => void;
  onToggle: () => void;
}) {
  const canCancel = activeStatuses.has(job.status);
  const running = job.status === "running";
  const userLabel = job.actorDisplayId || (job.actorId ? String(job.actorId) : "-");
  const numericId = job.actorId ? String(job.actorId) : "-";
  const queue = queueInfo(job);
  const activeJob = queue?.active;
  const waiting = job.status === "queued";
  const speed = running && job.tokensPerSecond && job.tokensPerSecond > 0
    ? t("jobs.tokenRateValue", { value: job.tokensPerSecond.toFixed(1) })
    : activeJob?.tokensPerSecond && activeJob.tokensPerSecond > 0
      ? t("jobs.tokenRateValue", { value: activeJob.tokensPerSecond.toFixed(1) })
    : running
      ? t("jobs.tokenRatePending")
      : activeJob
        ? t("jobs.tokenRatePending")
        : t("jobs.waiting");
  const lastTokenValue = waiting
    ? t("jobs.notAvailable")
    : formatAdminRelativeTime(t, job.updatedAt ?? job.createdAt);
  const estimatedTokensValue = running && job.estimatedTokens > 0
    ? String(job.estimatedTokens)
    : t("jobs.notAvailable");
  const generatedTokensValue = running
    ? String(aiJobGeneratedTokenCount(job))
    : activeJob
      ? String(activeJob.generatedTokens)
    : t("jobs.notAvailable");

  return (
    <div className="overflow-hidden rounded-lg border border-black/[0.06] bg-white shadow-sm">
      <div className="flex min-w-0 flex-col gap-3 p-3 lg:flex-row lg:items-center">
        <div className="flex min-w-0 flex-1 items-start gap-3">
          <span
            className={cn(
              "relative mt-1 flex size-3 shrink-0 rounded-full",
              job.status === "queued" ? "bg-amber-500" : job.status === "running" ? "bg-emerald-500" : "bg-slate-400",
            )}
            aria-hidden="true"
          >
            {running ? <span className="absolute inset-0 rounded-full bg-current opacity-35 motion-safe:animate-ping motion-reduce:hidden" /> : null}
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              <p className="truncate text-sm font-semibold text-[#20232a]">{aiJobTypeLabel(t, job.type)}</p>
              <span className="rounded-full bg-[#eef4ff] px-2 py-0.5 text-[11px] font-semibold text-[#1d4ed8]">
                {aiJobStatusLabel(t, job.status)}
              </span>
              {queue ? (
                <span className="rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700">
                  {t("jobs.queuePosition", { position: queue.position, total: queue.total })}
                </span>
              ) : null}
            </div>
            <p className="mt-1 truncate text-xs text-[#737987]">
              {t("jobs.userLine", { user: userLabel, id: numericId })}
            </p>
            <div className="mt-2 grid gap-2 text-xs text-[#5f6673] sm:grid-cols-4">
              <AIJobTinyMetric icon={TimerReset} label={t("jobs.lastToken")} value={lastTokenValue} />
              <AIJobTinyMetric label={t("jobs.tokenRate")} value={speed} />
              <AIJobTinyMetric label={t("jobs.generatedTokens")} value={generatedTokensValue} />
              <AIJobTinyMetric label={t("jobs.estimatedTokens")} value={estimatedTokensValue} />
            </div>
          </div>
        </div>
        <div className="flex shrink-0 items-center justify-between gap-2 lg:justify-end">
          {queue ? <span className="text-xs font-medium text-[#7b8190]">{formatAdminQueueEta(t, queue.etaSeconds)}</span> : null}
          <Button
            type="button"
            variant="outline"
            onClick={onToggle}
            className="h-9 rounded-lg border-black/[0.08] bg-white px-3 text-[#4b5563] hover:bg-[#f8fafc]"
          >
            <Terminal className="size-4" />
            <span>{expanded ? t("jobs.upstream.hide") : t("jobs.upstream.show")}</span>
            {expanded ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={!canCancel || canceling}
            onClick={onCancel}
            className="h-9 rounded-lg border-[#dc2626]/20 bg-white px-3 text-[#dc2626] hover:bg-[#fff1f2]"
          >
            {canceling ? <Loader2 className="size-4 animate-spin" /> : <XCircle className="size-4" />}
            <span>{t("jobs.cancel")}</span>
          </Button>
        </div>
      </div>
      <div className="h-1 overflow-hidden bg-[#edf2f7]">
        <div
          className={cn(
            "h-full transition-[width]",
            job.status === "queued"
              ? "bg-amber-500 motion-safe:animate-pulse"
              : "bg-emerald-500 motion-safe:animate-pulse",
          )}
          style={{ width: `${Math.max(4, Math.min(100, job.percent || (job.status === "queued" ? 8 : 16)))}%` }}
        />
      </div>
      {expanded ? <AIJobUpstreamPanel job={job} t={t} token={token} /> : null}
    </div>
  );
}

type AIUpstreamLine = {
  id: string;
  payload: Record<string, unknown>;
  phase: string;
  source: "history" | "live";
};

function AIJobUpstreamPanel({ job, t, token }: { job: AIJobPayload; t: AIAgentPanelT; token: string }) {
  const [lines, setLines] = useState<AIUpstreamLine[]>(() => aiJobUpstreamHistory(job));
  const [streaming, setStreaming] = useState(activeStatuses.has(job.status));
  const bottomRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: "end" });
  }, [lines.length]);

  useEffect(() => {
    if (!activeStatuses.has(job.status)) {
      return;
    }
    const controller = new AbortController();
    void streamAdminAIJob(job.jobId, {
      onUpstream: (event) => {
        setLines((current) => {
          const payload = recordFromUnknown(event.upstream) ?? {};
          const next: AIUpstreamLine = {
            id: `live-${Date.now()}-${current.length}`,
            payload,
            phase: upstreamPhase(event, payload),
            source: "live",
          };
          return [...current, next].slice(-200);
        });
      },
      onError: (event) => {
        const next: AIUpstreamLine = {
          id: `live-error-${Date.now()}`,
          payload: { code: event.code, detail: event.detail, message: event.message },
          phase: "error",
          source: "live",
        };
        setLines((current) => [...current, next].slice(-200));
      },
    }, { signal: controller.signal, token }).catch((error) => {
      if (controller.signal.aborted) {
        return;
      }
      const next: AIUpstreamLine = {
        id: `live-error-${Date.now()}`,
        payload: { message: errorMessage(error) },
        phase: "error",
        source: "live",
      };
      setLines((current) => [...current, next].slice(-200));
    }).finally(() => {
      if (!controller.signal.aborted) {
        setStreaming(false);
      }
    });
    return () => controller.abort();
  }, [job.jobId, job.status, token]);

  const listening = streaming && activeStatuses.has(job.status);

  return (
    <div className="border-t border-black/[0.06] bg-[#0f172a] p-3 text-[#dbeafe]">
      <div className="mb-2 flex min-w-0 flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2 text-xs font-semibold">
          <Terminal className="size-4 text-[#93c5fd]" />
          <span>{t("jobs.upstream.title")}</span>
        </div>
        <span className="rounded-full bg-white/10 px-2 py-0.5 text-[11px] font-semibold text-[#bfdbfe]">
          {listening ? t("jobs.upstream.streaming") : t("jobs.upstream.stopped")}
        </span>
      </div>
      <div className="max-h-[300px] overflow-y-auto rounded-lg border border-white/10 bg-black/20 p-2 font-mono text-[11px] leading-5">
        {lines.length === 0 ? (
          <p className="px-2 py-5 text-center font-sans text-sm font-semibold text-[#94a3b8]">{t("jobs.upstream.empty")}</p>
        ) : (
          lines.map((line) => (
            <div key={line.id} className="mb-2 rounded-md border border-white/10 bg-white/[0.03] p-2">
              <div className="mb-1 flex items-center justify-between gap-2 text-[#bfdbfe]">
                <span>{line.source === "live" ? t("jobs.upstream.live") : t("jobs.upstream.history")}</span>
                <span>{line.phase}</span>
              </div>
              <pre className="whitespace-pre-wrap break-words text-[#e0f2fe]">{JSON.stringify(line.payload, null, 2)}</pre>
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

function AIJobMetric({ label, value, tone }: { label: string; value: number; tone: "blue" | "amber" | "green" }) {
  return (
    <div
      className={cn(
        "rounded-lg border p-3",
        tone === "blue" && "border-[#1d4ed8]/15 bg-[#eff6ff] text-[#1d4ed8]",
        tone === "amber" && "border-amber-500/15 bg-amber-50 text-amber-700",
        tone === "green" && "border-emerald-500/15 bg-emerald-50 text-emerald-700",
      )}
    >
      <p className="text-xs font-semibold">{label}</p>
      <p className="mt-1 text-2xl font-bold leading-none">{value}</p>
    </div>
  );
}

function AIJobTinyMetric({ icon: Icon, label, value }: { icon?: typeof TimerReset; label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg bg-[#f7f9fc] px-2 py-1.5">
      <span className="flex min-w-0 items-center gap-1 text-[11px] font-semibold text-[#7b8190]">
        {Icon ? <Icon className="size-3" /> : null}
        <span className="truncate">{label}</span>
      </span>
      <span className="mt-0.5 block truncate font-semibold text-[#303743]">{value}</span>
    </div>
  );
}

function aiJobUpstreamHistory(job: AIJobPayload): AIUpstreamLine[] {
  const metadata = recordFromUnknown(job.metadata);
  const attempts = Array.isArray(metadata?.upstreamAttempts) ? metadata.upstreamAttempts : [];
  return attempts.slice(-20).map((item, index) => {
    const payload = recordFromUnknown(item) ?? {};
    return {
      id: `history-${job.jobId}-${index}`,
      payload,
      phase: upstreamPhase(null, payload),
      source: "history",
    };
  });
}

function upstreamPhase(event: AIUpstreamEvent | null, payload: Record<string, unknown>) {
  const phase = stringFromUnknown(payload.phase) || event?.stage || stringFromUnknown(payload.errorCode);
  return phase || "upstream";
}

function recordFromUnknown(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function stringFromUnknown(value: unknown) {
  return typeof value === "string" ? value : "";
}

function queueInfo(job: AIJobPayload) {
  const raw = job.queueJob;
  if (!raw || typeof raw !== "object") {
    return job.status === "queued" || job.stage === "queued"
      ? { position: 1, total: 1, etaSeconds: 0 }
      : null;
  }
  const position = numberFromQueue(raw.queuePosition);
  const total = numberFromQueue(raw.queueCount ?? raw.queueTotal);
  if (position <= 0 && total <= 0) {
    return job.status === "queued" || job.stage === "queued"
      ? { position: 1, total: 1, etaSeconds: 0 }
      : null;
  }
  return {
    active: activeJobInfo(raw.activeJob),
    position: Math.max(1, position || total || 1),
    total: Math.max(position || 1, total || position || 1),
    etaSeconds: Math.max(0, numberFromQueue(raw.estimatedWaitSeconds ?? raw.etaSeconds)),
  };
}

function activeJobInfo(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const raw = value as Record<string, unknown>;
  const generatedTokens = numberFromQueue(raw.generatedTokens);
  if (!raw.jobId && !raw.actorId && generatedTokens <= 0) {
    return null;
  }
  const speed = Number(raw.tokensPerSecond ?? 0);
  return {
    generatedTokens: Math.max(0, generatedTokens),
    tokensPerSecond: Number.isFinite(speed) && speed > 0 ? speed : 0,
  };
}

function numberFromQueue(value: unknown) {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? Math.trunc(parsed) : 0;
}

function aiJobGeneratedTokenCount(job: AIJobPayload) {
  const tokens = Number(job.completionTokens || job.totalTokens || 0);
  return Number.isFinite(tokens) && tokens > 0 ? Math.trunc(tokens) : 0;
}
