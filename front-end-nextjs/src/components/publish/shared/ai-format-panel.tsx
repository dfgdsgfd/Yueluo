"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Loader2, Sparkles, X } from "lucide-react";
import { toast } from "sonner";
import { MarkdownContent } from "@/components/markdown-content";
import { Button } from "@/components/ui/button";
import { AUTH_USER_EVENT, ApiError, cancelAIJob, createAIJob, getAIJob, getActiveAIJob, getStoredUser, streamAIJob } from "@/lib/api";
import type { AuthUser } from "@/lib/types";
import type { AIJobPayload } from "@/lib/types";
import { cn } from "@/lib/utils";
import { AIFormatConfirmDialog } from "./ai-format-confirm-dialog";
import { aiJobGeneratedTokenCount, formatJobStoragePrefix, formatSessions, modeControls, phaseFromJobStatus, runningJobStatuses, type AIFormatMode, type AIFormatSession } from "./ai-format-panel-state";
import { AIFormatPromptSamples } from "./ai-format-prompt-samples";
import { AIFormatQueueStatus, formatAIFormatRelativeTime } from "./ai-format-queue-status";
import { queueFromAIJob, type QueueState } from "./ai-job-queue";
import { ThinkingTicker, type AIFormatPhase } from "./ai-format-thinking-ticker";
import {
  ProgressBar,
  aiErrorLabel,
  aiFormatLiveDisplay,
  aiStageLabel,
  clearStoredJobId,
  formatErrorDetail,
  formatQueueEta,
  hashAIFormatSource,
  isAbortError,
  readStoredJobId,
  storeJobId,
} from "./ai-format-panel-utils";
import { useAIFormatLiveStream } from "./use-ai-format-live-stream";

type AIFormatPanelProps = {
  open: boolean;
  value: string;
  onApply: (value: string) => void;
  onClose: () => void;
  variant?: "desktop" | "mobile";
};

type Phase = AIFormatPhase;

export function AIFormatPanel({
  open,
  value,
  onApply,
  onClose,
  variant = "desktop",
}: AIFormatPanelProps) {
  const t = useTranslations("publish.aiFormat");
  const locale = useLocale();
  const [mode, setMode] = useState<AIFormatMode>("format");
  const [customPrompt, setCustomPrompt] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [percent, setPercent] = useState(0);
  const [currentChunk, setCurrentChunk] = useState(0);
  const [totalChunks, setTotalChunks] = useState(0);
  const [processedChars, setProcessedChars] = useState(0);
  const [totalChars, setTotalChars] = useState(0);
  const [stage, setStage] = useState("");
  const [queue, setQueue] = useState<QueueState | null>(null);
  const [result, setResult] = useState("");
  const [reasoning, setReasoning] = useState("");
  const [error, setError] = useState("");
  const [errorDetail, setErrorDetail] = useState("");
  const [generatedTokens, setGeneratedTokens] = useState(0);
  const [jobId, setJobId] = useState("");
  const [jobActorId, setJobActorId] = useState<number | null>(null);
  const [jobActorDisplayId, setJobActorDisplayId] = useState<string | null>(null);
  const [jobUpdatedAt, setJobUpdatedAt] = useState<string | null>(null);
  const [viewerUser, setViewerUser] = useState<AuthUser | null>(() => getStoredUser());
  const [tokensPerSecond, setTokensPerSecond] = useState(0);
  const [confirmStartOpen, setConfirmStartOpen] = useState(false);
  const [switchingMode, setSwitchingMode] = useState(false);
  const resultRef = useRef<HTMLDivElement | null>(null);
  const pollTimerRef = useRef<number | null>(null);
  const pollGenerationRef = useRef(0);
  const streamAbortRef = useRef<AbortController | null>(null);
  const original = useMemo(() => value.trim(), [value]);
  const prompt = customPrompt.trim();
  const sessionKey = useMemo(
    () => `${locale}:${mode}:${hashAIFormatSource(original)}:${mode === "custom" ? hashAIFormatSource(prompt) : ""}`,
    [locale, mode, original, prompt],
  );
  const selectedMode = modeControls.find((item) => item.key === mode) ?? modeControls[0];
  const sourceInput = original;
  const customPromptRequired = mode === "custom" && !prompt;
  const canGenerate = mode === "custom" ? !customPromptRequired : Boolean(sourceInput);
  const liveStream = useAIFormatLiveStream({
    generationRef: pollGenerationRef,
    patchSession,
    setCurrentChunk,
    setGeneratedTokens,
    setPercent,
    setProcessedChars,
    setQueue,
    setReasoning,
    setResult,
    setStage,
    setTokensPerSecond,
    setTotalChars,
    setTotalChunks,
  });

  useEffect(() => {
    let cancelled = false;
    if (!open) {
      return;
    }
    const cached = formatSessions.get(sessionKey);
    if (cached && cached.original === original && cached.mode === mode) {
      restoreSession(cached);
    } else {
      resetView();
    }
    queueMicrotask(() => {
      void resumePersistedJob(() => cancelled);
    });
    return () => {
      cancelled = true;
    };
    // Persisted jobs can resume, but new AI work only starts after explicit confirmation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, mode, original, canGenerate, sessionKey]);

  useEffect(() => {
    if (!open) {
      stopLiveUpdates();
    }
    return () => {
      stopLiveUpdates();
    };
    // Live update cleanup follows the panel lifecycle; helpers only touch refs and timers.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    const handleAuthUser = (event: Event) => {
      const customEvent = event as CustomEvent<AuthUser | null>;
      setViewerUser(customEvent.detail ?? getStoredUser());
    };
    handleAuthUser(new CustomEvent(AUTH_USER_EVENT, { detail: getStoredUser() }));
    window.addEventListener(AUTH_USER_EVENT, handleAuthUser as EventListener);
    return () => {
      window.removeEventListener(AUTH_USER_EVENT, handleAuthUser as EventListener);
    };
  }, []);

  useEffect(() => {
    if (resultRef.current) {
      resultRef.current.scrollTop = resultRef.current.scrollHeight;
    }
  }, [result]);

  function resetView() {
    setPhase("idle");
    setError("");
    setResult("");
    setReasoning("");
    setPercent(0);
    setCurrentChunk(0);
    setTotalChunks(0);
    setProcessedChars(0);
    setTotalChars(0);
    setStage("");
    setQueue(null);
    setErrorDetail("");
    setJobId("");
    setJobActorId(null);
    setJobActorDisplayId(null);
    setJobUpdatedAt(null);
    setGeneratedTokens(0);
    setTokensPerSecond(0);
    liveStream.resetLivePreview();
  }

  function restoreSession(session: AIFormatSession) {
    setPhase(session.phase);
    setError(session.error);
    setErrorDetail(session.errorDetail);
    setGeneratedTokens(session.generatedTokens ?? 0);
    setJobId(session.jobId);
    setJobActorId(session.actorId ?? null);
    setJobActorDisplayId(session.actorDisplayId ?? null);
    setJobUpdatedAt(session.updatedAt ?? null);
    setResult(session.result);
    setReasoning(session.reasoning);
    setPercent(session.percent);
    setCurrentChunk(session.currentChunk);
    setTotalChunks(session.totalChunks);
    setProcessedChars(session.processedChars);
    setTotalChars(session.totalChars);
    setStage(session.stage);
    setQueue(session.queue);
    setTokensPerSecond(session.tokensPerSecond);
    if (session.phase === "running" && session.jobId) {
      startJobStream(session.jobId);
    }
  }

  function patchSession(patch: Partial<AIFormatSession>) {
    const current = formatSessions.get(sessionKey) ?? {
      actorDisplayId: null,
      actorId: null,
      currentChunk: 0,
      error: "",
      errorDetail: "",
      generatedTokens: 0,
      jobId: "",
      mode,
      original,
      percent: 0,
      phase: "idle",
      processedChars: 0,
      queue: null,
      reasoning: "",
      result: "",
      stage: "",
      tokensPerSecond: 0,
      totalChars: 0,
      totalChunks: 0,
      updatedAt: null,
    };
    formatSessions.set(sessionKey, { ...current, ...patch, mode, original });
  }

  async function resumePersistedJob(isCancelled: () => boolean) {
    if (!canGenerate) {
      return false;
    }
    const persistedJobId = readStoredJobId(formatJobStoragePrefix, sessionKey);
    try {
      let job: AIJobPayload | null = null;
      if (persistedJobId) {
        job = await getAIJob(persistedJobId);
      }
      if (!job || !runningJobStatuses.has(job.status)) {
        try {
          job = await getActiveAIJob(sessionKey);
        } catch (error) {
          if (!(error instanceof ApiError) || (error.status !== 400 && error.status !== 404)) {
            throw error;
          }
        }
      }
      if (isCancelled() || !job) {
        return false;
      }
      applyJob(job);
      if (runningJobStatuses.has(job.status)) {
        storeJobId(formatJobStoragePrefix, sessionKey, job.jobId);
        startJobStream(job.jobId);
      }
      return true;
    } catch {
      clearStoredJobId(formatJobStoragePrefix, sessionKey);
      return false;
    }
  }

  async function runFormat(runMode: "resume" | "regenerate" = "regenerate") {
    if (!canGenerate) {
      toast.error(t("empty"));
      return;
    }
    if (customPromptRequired) {
      toast.error(t("customPromptRequired"));
      return;
    }
    const existing = formatSessions.get(sessionKey);
    if (runMode === "resume" && existing?.phase === "running") {
      restoreSession(existing);
      return;
    }
    stopLiveUpdates();
    resetView();
    setPhase("running");
    patchSession({
      error: "",
      errorDetail: "",
      jobId: "",
      percent: 0,
      phase: "running",
      queue: null,
      reasoning: "",
      result: "",
      stage: "",
      generatedTokens: 0,
      tokensPerSecond: 0,
    });
    try {
      const job = await createAIJob({
        type: selectedMode.taskType,
        locale,
        input: sourceInput,
        templateKey: selectedMode.templateKey,
        requestHash: sessionKey,
        variables: {
          customPrompt: prompt,
          mode,
        },
      });
      storeJobId(formatJobStoragePrefix, sessionKey, job.jobId);
      applyJob(job);
      if (runningJobStatuses.has(job.status)) {
        startJobStream(job.jobId);
      }
    } catch (nextError) {
      const message = nextError instanceof Error ? nextError.message : "error.ai_request_failed";
      setError(message);
      setPhase("error");
      patchSession({ error: message, phase: "error" });
    }
  }

  async function cancelFormat() {
    await abandonActiveJob(false);
  }

  async function abandonActiveJob(notify: boolean) {
    const activeJobId = jobId || formatSessions.get(sessionKey)?.jobId || readStoredJobId(formatJobStoragePrefix, sessionKey);
    if (!activeJobId) {
      setPhase("idle");
      patchSession({ phase: "idle" });
      return true;
    }
    stopLiveUpdates();
    try {
      const job = await cancelAIJob(activeJobId);
      applyJob(job);
      clearStoredJobId(formatJobStoragePrefix, sessionKey);
      if (notify) {
        toast.info(t("taskAbandoned"));
      }
      return true;
    } catch (nextError) {
      const message = nextError instanceof Error ? nextError.message : "error.ai_request_canceled";
      setError(message);
      setPhase("error");
      patchSession({ error: message, phase: "error" });
      return false;
    }
  }

  function requestStart() {
    if (!canGenerate) {
      toast.error(t("empty"));
      return;
    }
    if (customPromptRequired) {
      toast.error(t("customPromptRequired"));
      return;
    }
    setConfirmStartOpen(true);
  }

  async function confirmStart() {
    setConfirmStartOpen(false);
    await runFormat("regenerate");
  }

  async function handleModeSelect(nextMode: AIFormatMode) {
    if (nextMode === mode || switchingMode) {
      return;
    }
    if (phase === "running" && (jobId || formatSessions.get(sessionKey)?.jobId || readStoredJobId(formatJobStoragePrefix, sessionKey))) {
      setSwitchingMode(true);
      const abandoned = await abandonActiveJob(true);
      setSwitchingMode(false);
      if (!abandoned) {
        return;
      }
    }
    setConfirmStartOpen(false);
    setMode(nextMode);
  }

  function startPolling(nextJobId: string) {
    stopPolling();
    const generation = pollGenerationRef.current + 1;
    pollGenerationRef.current = generation;
    const poll = async () => {
      try {
        const job = await getAIJob(nextJobId);
        if (pollGenerationRef.current !== generation) {
          return;
        }
        applyJob(job);
        if (runningJobStatuses.has(job.status) && open) {
          pollTimerRef.current = window.setTimeout(poll, 1000);
        }
      } catch (nextError) {
        if (pollGenerationRef.current !== generation) {
          return;
        }
        const message = nextError instanceof Error ? nextError.message : "error.ai_request_failed";
        setError(message);
        setPhase("error");
        patchSession({ error: message, phase: "error" });
      }
    };
    pollTimerRef.current = window.setTimeout(poll, 400);
  }

  function stopPolling() {
    pollGenerationRef.current += 1;
    if (pollTimerRef.current !== null) {
      window.clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }

  function startJobStream(nextJobId: string) {
    stopLiveUpdates();
    const generation = pollGenerationRef.current + 1;
    pollGenerationRef.current = generation;
    const controller = new AbortController();
    streamAbortRef.current = controller;
    setStage("connecting");
    patchSession({ stage: "connecting" });
    const liveHandlers = liveStream.createLiveHandlers(generation);
    void streamAIJob(nextJobId, {
      onConnected: () => {
        if (pollGenerationRef.current !== generation) return;
        setStage("connected");
        patchSession({ stage: "connected" });
      },
      onJob: (job) => {
        if (pollGenerationRef.current !== generation) return;
        applyJob(job);
      },
      ...liveHandlers,
      onError: (event) => {
        if (pollGenerationRef.current !== generation) return;
        const nextError = event.message || event.code || "error.ai_request_failed";
        setError(nextError);
        setErrorDetail(event.detail ?? "");
        patchSession({ error: nextError, errorDetail: event.detail ?? "" });
      },
    }, { signal: controller.signal }).catch((nextError: unknown) => {
      if (pollGenerationRef.current !== generation || isAbortError(nextError)) {
        return;
      }
      const message = nextError instanceof Error ? nextError.message : "error.ai_stream_unavailable";
      const detail = nextError instanceof ApiError ? formatErrorDetail(nextError.details) : "";
      setError(message);
      setErrorDetail(detail);
      setStage("stream_fallback");
      patchSession({ error: message, errorDetail: detail, stage: "stream_fallback" });
      if (open) {
        startPolling(nextJobId);
      }
    }).finally(() => {
      if (streamAbortRef.current === controller) {
        streamAbortRef.current = null;
      }
    });
  }

  function stopJobStream() {
    if (streamAbortRef.current) {
      streamAbortRef.current.abort();
      streamAbortRef.current = null;
    }
  }

  function stopLiveUpdates() {
    stopJobStream();
    stopPolling();
  }

  function applyJob(job: AIJobPayload) {
    const nextPhase = phaseFromJobStatus(job.status);
    const nextQueue = queueFromAIJob(job);
    const nextError = job.errorCode || (nextPhase === "error" ? "error.ai_request_failed" : "");
    const nextErrorDetail = job.upstreamDetail || job.errorMessage || "";
    const nextGeneratedTokens = aiJobGeneratedTokenCount(job);
    setJobId(job.jobId);
    setJobActorId(job.actorId ?? null);
    setJobActorDisplayId(job.actorDisplayId ?? null);
    setJobUpdatedAt(job.updatedAt ?? job.createdAt ?? null);
    setPhase(nextPhase);
    setPercent(job.percent ?? 0);
    setCurrentChunk(job.currentChunk ?? 0);
    setTotalChunks(job.totalChunks ?? 0);
    setProcessedChars(job.processedChars ?? 0);
    setTotalChars(job.totalChars ?? 0);
    setStage(job.stage ?? "");
    setQueue(nextQueue);
    if (!liveStream.liveOutputActive()) {
      setResult(job.output ?? "");
    }
    if (!liveStream.liveReasoningActive()) {
      setReasoning(job.reasoning ?? "");
    }
    setError(nextError);
    setErrorDetail(nextErrorDetail);
    setGeneratedTokens(nextGeneratedTokens);
    setTokensPerSecond(job.tokensPerSecond ?? 0);
    patchSession({
      currentChunk: job.currentChunk ?? 0,
      error: nextError,
      errorDetail: nextErrorDetail,
      generatedTokens: nextGeneratedTokens,
      jobId: job.jobId,
      actorId: job.actorId ?? null,
      actorDisplayId: job.actorDisplayId ?? null,
      percent: job.percent ?? 0,
      phase: nextPhase,
      processedChars: job.processedChars ?? 0,
      queue: nextQueue,
      reasoning: liveStream.liveReasoningActive() ? liveStream.liveReasoningText() : job.reasoning ?? "",
      result: liveStream.liveOutputActive() ? liveStream.liveResultText() : job.output ?? "",
      stage: job.stage ?? "",
      tokensPerSecond: job.tokensPerSecond ?? 0,
      totalChars: job.totalChars ?? 0,
      totalChunks: job.totalChunks ?? 0,
      updatedAt: job.updatedAt ?? job.createdAt ?? null,
    });
    if (nextPhase === "done" || nextPhase === "error" || job.status === "canceled") {
      clearStoredJobId(formatJobStoragePrefix, sessionKey);
    }
  }

  if (!open) {
    return null;
  }

  const running = phase === "running";
  const canApply = result.trim().length > 0;
  const chunkLabel = totalChunks > 0
    ? t("chunkProgress", { current: Math.max(1, currentChunk), total: totalChunks })
    : t("preparing");
  const chunkDetail = totalChunks > 1 && totalChars > 0
    ? t("chunkDetail", {
        current: Math.max(1, currentChunk),
        total: totalChunks,
        processed: Math.min(processedChars, totalChars),
        totalChars,
      })
    : "";
  const longChunkNotice = totalChunks > 1 ? t("chunkNotice", { total: totalChunks }) : "";
  const activeModeLabel = t(`modes.${mode}`);
  const startLabel = t("startAction", { mode: activeModeLabel });
  const statusLabel = queue ? t("queueStatus") : t(`status.${phase}`);
  const previewText = result || (running ? t("streaming") : t("emptyResult"));
  const queueStatusLabel = stage === "queued" || queue ? t("queueStatus") : t("status.running");
  const queueUpdatedLabel = formatAIFormatRelativeTime(t, jobUpdatedAt);
  const queueAhead = queue ? Math.max(0, queue.position - 1) : 0;
  const queuePositionLabel = queue
    ? queueAhead > 0
      ? t("queuePositionCurrent", { position: queueAhead })
      : t("queuePositionCurrentImmediate")
    : "";
  const queueTotalLabel = queue ? t("queuePositionTotal", { total: queue.total }) : "";
  const queueDetailLabel = queue
    ? queueAhead > 0
      ? t("queueDetail", { position: queueAhead, total: queue.total, eta: formatQueueEta(t, queue.etaSeconds) })
      : t("queueDetailImmediate", { total: queue.total, eta: formatQueueEta(t, queue.etaSeconds) })
    : "";
  const { activeLiveLabel, generatedTokenLabel, speedLabel } = aiFormatLiveDisplay(t, {
    generatedTokens,
    jobActorDisplayId,
    jobActorId,
    queue,
    viewerUser,
    result,
    running,
    tokensPerSecond,
  });

  return (
    <div
      className={cn(
        "fixed inset-0 z-[70] flex bg-[#111827]/55 p-3 backdrop-blur-[3px] sm:p-5",
        variant === "mobile" ? "items-end justify-center" : "items-center justify-center",
      )}
      role="dialog"
      aria-modal="true"
    >
      <button type="button" aria-label={t("close")} className="absolute inset-0" onClick={onClose} />
      <section
        className={cn(
          "relative z-10 flex min-h-0 w-full flex-col overflow-hidden border border-white/70 bg-white text-[#20232a] shadow-2xl",
          variant === "mobile"
            ? "max-h-[90dvh] max-w-[430px] rounded-t-[22px]"
            : "h-[min(880px,calc(100dvh-40px))] max-w-[1040px] rounded-[18px]",
        )}
      >
        <header className="shrink-0 border-b border-black/[0.06] bg-[#fbfcff] px-4 py-3 sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-[#e8f1ff] text-[#1d4ed8]">
              <Sparkles className="size-5" />
            </span>
            <div className="min-w-0 flex-1">
              <h2 className="truncate text-base font-semibold text-[#17171d]">{t("title")}</h2>
              <p className={cn("text-xs text-[#7b8190]", queue ? "whitespace-normal break-words leading-5" : "truncate")}>
                {[activeLiveLabel || statusLabel, speedLabel].filter(Boolean).join(" · ")}
              </p>
            </div>
            <button
              type="button"
              aria-label={t("close")}
              onClick={onClose}
              className="flex size-9 shrink-0 items-center justify-center rounded-lg text-[#5f6673] hover:bg-[#eef1f6]"
            >
              <X className="size-4" />
            </button>
          </div>
          <div className={cn("mt-3 grid gap-3", queue ? "lg:grid-cols-1" : "lg:grid-cols-[minmax(0,1fr)_220px]")}>
            <div className="flex min-w-0 flex-nowrap gap-1.5 overflow-x-auto pb-0.5">
              {modeControls.map(({ key, icon: Icon }) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => void handleModeSelect(key)}
                  disabled={switchingMode}
                  className={cn(
                    "flex h-9 shrink-0 items-center gap-1.5 rounded-full border px-2.5 text-xs font-semibold shadow-sm transition",
                    mode === key
                      ? "border-[#1d4ed8] bg-[#1d4ed8] text-white shadow-[0_10px_22px_rgba(29,78,216,0.18)]"
                      : "border-black/[0.08] bg-white text-[#5f6673] hover:border-[#1d4ed8]/20 hover:bg-[#f4f7fb] hover:text-[#1d4ed8]",
                  )}
                >
                  <Icon className="size-3.5" />
                  <span>{t(`modes.${key}`)}</span>
                </button>
              ))}
            </div>
            <ProgressBar percent={percent} label={activeLiveLabel || (queue ? t("queued") : chunkLabel)} wrapLabel={Boolean(activeLiveLabel)} />
          </div>
          {activeLiveLabel ? (
            <div className="mt-3 flex min-w-0 items-center gap-2 rounded-xl border border-[#1d4ed8]/15 bg-[#eef4ff] px-3 py-2 text-xs font-semibold leading-5 text-[#1d4ed8]">
              <span className="relative flex size-2.5 shrink-0 rounded-full bg-[#1d4ed8]" aria-hidden="true">
                <span className="absolute inset-0 rounded-full bg-[#1d4ed8] opacity-35 motion-safe:animate-ping motion-reduce:hidden" />
              </span>
              <span className="min-w-0 break-words">{activeLiveLabel}</span>
            </div>
          ) : null}
          <ThinkingTicker
            doneText={t("thinkingTickerDone")}
            errorText={t("thinkingTickerError")}
            idleText={t("thinkingTickerIdle")}
            label={t("thinking")}
            pendingText={t("thinkingTickerPending")}
            phase={phase}
            running={running}
            text={reasoning}
          />
        </header>

        <div className="grid min-h-0 flex-1 gap-3 overflow-y-auto bg-[#f5f7fb] p-4 sm:p-5">
          {running ? (
            <AIFormatQueueStatus
              generatedTokens={generatedTokenLabel}
              jobId={jobId}
              queue={queue}
              queueDetail={queueDetailLabel}
              queuePosition={queuePositionLabel}
              queueTotal={queueTotalLabel}
              status={queueStatusLabel}
              updated={queueUpdatedLabel}
            />
          ) : null}

          {mode === "custom" ? (
            <label className="grid gap-1.5">
              <span className="text-xs font-semibold text-[#606773]">{t("customPrompt")}</span>
              <textarea
                value={customPrompt}
                onChange={(event) => setCustomPrompt(event.target.value)}
                placeholder={t("customPromptPlaceholder")}
                className="min-h-[92px] resize-none rounded-xl border border-black/[0.08] bg-white px-3 py-2 text-sm leading-6 outline-none focus:border-[#1d4ed8]"
              />
              <AIFormatPromptSamples
                disabled={running}
                t={t}
                onPick={(sample) => setCustomPrompt(sample)}
              />
            </label>
          ) : null}

          <div className="flex min-h-[min(56dvh,560px)] min-w-0 flex-col overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm">
            <div className="flex shrink-0 items-center justify-between gap-3 border-b border-black/[0.06] px-4 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-[#20232a]">{t("preview")}</p>
                <p className="truncate text-xs text-[#7b8190]">{mode === "custom" ? t("modeHints.custom") : t(`modeHints.${mode}`)}</p>
              </div>
              {stage && !queue && running ? (
                <span className="shrink-0 rounded-full bg-[#eef4ff] px-3 py-1 text-xs font-semibold text-[#1d4ed8]">
                  {aiStageLabel(t, stage)}
                </span>
              ) : null}
            </div>
            <div
              ref={resultRef}
              className={cn(
                "min-h-0 flex-1 overflow-auto px-4 py-4 text-sm leading-7 sm:px-6",
                result ? "text-[#20232a]" : "text-[#9aa1ad]",
              )}
            >
              {result ? <MarkdownContent content={result} /> : <p className="whitespace-pre-wrap break-words">{previewText}</p>}
            </div>
          </div>

          <details className="rounded-xl border border-black/[0.06] bg-white px-3 py-2 text-sm text-[#5f6673]">
            <summary className="cursor-pointer select-none text-xs font-semibold text-[#606773]">{t("sourceDetails")}</summary>
            <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words text-xs leading-5">{original || t("emptySource")}</pre>
          </details>

          {longChunkNotice ? <p className="text-xs font-medium text-[#1d4ed8]">{longChunkNotice}</p> : null}
          {speedLabel ? <p className="text-xs font-medium text-[#536070]">{speedLabel}</p> : null}
          {chunkDetail ? <p className="text-xs font-medium text-[#1d4ed8]">{chunkDetail}</p> : null}
          {error ? (
            <p className="text-xs font-medium text-[#dc2626]">
              {aiErrorLabel(t, error)}
              {errorDetail ? <span className="mt-1 block break-words text-[#991b1b]">{t("apiErrorDetail", { detail: errorDetail })}</span> : null}
            </p>
          ) : null}
        </div>

        <footer className="flex shrink-0 flex-col-reverse gap-2 border-t border-black/[0.06] bg-white px-4 py-3 sm:flex-row sm:items-center sm:justify-end">
          {running ? (
            <Button type="button" variant="outline" onClick={() => void cancelFormat()} className="h-10 rounded-xl border-black/[0.08] bg-white px-3">
              <X className="size-4" />
              <span>{t("cancel")}</span>
            </Button>
          ) : (
            <Button
              type="button"
              disabled={!canGenerate}
              onClick={requestStart}
              className="h-12 rounded-xl bg-[#1d4ed8] px-5 text-base font-semibold shadow-[0_14px_32px_rgba(29,78,216,0.22)] hover:bg-[#1e40af] sm:h-10 sm:text-sm"
            >
              <Sparkles className="size-4" />
              <span>{result ? t("restartAction", { mode: activeModeLabel }) : startLabel}</span>
            </Button>
          )}
          <Button
            type="button"
            disabled={!canApply}
            onClick={() => {
              onApply(result);
              onClose();
            }}
            className="h-10 rounded-xl bg-[#111827] px-4 hover:bg-[#0f172a]"
          >
            {running ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
            <span>{t("apply")}</span>
          </Button>
        </footer>
        <AIFormatConfirmDialog
          cancelLabel={t("confirmCancel")}
          confirmLabel={startLabel}
          description={t("confirmDescription", { mode: activeModeLabel })}
          onCancel={() => setConfirmStartOpen(false)}
          onConfirm={() => void confirmStart()}
          open={confirmStartOpen}
          title={t("confirmTitle", { mode: activeModeLabel })}
          variant={variant}
        />
      </section>
    </div>
  );
}
