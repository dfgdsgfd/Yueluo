"use client";

import { useRef, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type {
  AIChunkDeltaEvent,
  AIChunkDoneEvent,
  AIFinalEvent,
  AIProgressEvent,
  AIReasoningDeltaEvent,
  AIReasoningDoneEvent,
} from "@/lib/types";
import { activeQueueJobFromProgress, type QueueState } from "./ai-job-queue";

type LiveSessionPatch = {
  currentChunk?: number;
  generatedTokens?: number;
  percent?: number;
  processedChars?: number;
  queue?: QueueState | null;
  reasoning?: string;
  result?: string;
  stage?: string;
  tokensPerSecond?: number;
  totalChars?: number;
  totalChunks?: number;
};

type UseAIFormatLiveStreamOptions = {
  generationRef: MutableRefObject<number>;
  patchSession: (patch: LiveSessionPatch) => void;
  setCurrentChunk: Dispatch<SetStateAction<number>>;
  setGeneratedTokens: Dispatch<SetStateAction<number>>;
  setPercent: Dispatch<SetStateAction<number>>;
  setProcessedChars: Dispatch<SetStateAction<number>>;
  setQueue: Dispatch<SetStateAction<QueueState | null>>;
  setReasoning: Dispatch<SetStateAction<string>>;
  setResult: Dispatch<SetStateAction<string>>;
  setStage: Dispatch<SetStateAction<string>>;
  setTokensPerSecond: Dispatch<SetStateAction<number>>;
  setTotalChars: Dispatch<SetStateAction<number>>;
  setTotalChunks: Dispatch<SetStateAction<number>>;
};

export function useAIFormatLiveStream({
  generationRef,
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
}: UseAIFormatLiveStreamOptions) {
  const liveChunkPartsRef = useRef<Map<number, string>>(new Map());
  const liveReasoningRef = useRef("");
  const liveOutputActiveRef = useRef(false);
  const liveReasoningActiveRef = useRef(false);
  const continuationPreviewChunksRef = useRef<Set<number>>(new Set());

  function resetLivePreview() {
    liveChunkPartsRef.current.clear();
    liveReasoningRef.current = "";
    liveOutputActiveRef.current = false;
    liveReasoningActiveRef.current = false;
    continuationPreviewChunksRef.current.clear();
  }

  function liveOutputActive() {
    return liveOutputActiveRef.current;
  }

  function liveReasoningActive() {
    return liveReasoningActiveRef.current;
  }

  function liveReasoningText() {
    return liveReasoningRef.current;
  }

  function liveResultText() {
    return Array.from(liveChunkPartsRef.current.entries())
      .sort(([left], [right]) => left - right)
      .map(([, text]) => text.trim())
      .filter(Boolean)
      .join("\n\n")
      .trim();
  }

  function resetLiveChunk(chunkIndex: number) {
    if (!liveOutputActiveRef.current) {
      return;
    }
    liveChunkPartsRef.current.delete(chunkIndex);
    syncLiveResult();
  }

  function appendLiveChunk(chunkIndex: number, delta: string) {
    if (!delta) {
      return;
    }
    liveOutputActiveRef.current = true;
    const current = liveChunkPartsRef.current.get(chunkIndex) ?? "";
    const separator = current.trim() && continuationPreviewChunksRef.current.has(chunkIndex) ? "\n\n" : "";
    continuationPreviewChunksRef.current.delete(chunkIndex);
    liveChunkPartsRef.current.set(chunkIndex, `${current}${separator}${delta}`);
    syncLiveResult();
  }

  function setLiveChunk(chunkIndex: number, text: string) {
    liveOutputActiveRef.current = true;
    continuationPreviewChunksRef.current.delete(chunkIndex);
    liveChunkPartsRef.current.set(chunkIndex, text);
    syncLiveResult();
  }

  function syncLiveResult() {
    const next = liveResultText();
    const nextGeneratedTokens = estimateAITextTokens(next);
    setResult(next);
    setGeneratedTokens(nextGeneratedTokens);
    patchSession({ generatedTokens: nextGeneratedTokens, result: next });
  }

  function createLiveHandlers(generation: number) {
    const isCurrentGeneration = () => generationRef.current === generation;
    return {
      onProgress: (event: AIProgressEvent) => {
        if (!isCurrentGeneration()) return;
        setPercent(event.percent ?? 0);
        setCurrentChunk(event.currentChunk ?? 0);
        setTotalChunks(event.totalChunks ?? 0);
        setProcessedChars(event.processedChars ?? 0);
        setTotalChars(event.totalChars ?? 0);
        setStage(event.stage ?? "");
        setTokensPerSecond(event.tokensPerSecond ?? 0);
        let nextQueue: QueueState | null = null;
        if (event.stage === "connecting" || event.stage === "retrying" || event.stage === "chunk_start") {
          resetLiveChunk(event.currentChunk ? event.currentChunk - 1 : 0);
        }
        if (event.stage === "continuation_start" && event.currentChunk) {
          continuationPreviewChunksRef.current.add(event.currentChunk - 1);
        }
        if (event.stage === "queued") {
          const position = Number(event.queuePosition ?? 0);
          const total = Number(event.queueTotal ?? event.queuePosition ?? 0);
          nextQueue = {
            active: activeQueueJobFromProgress(event),
            etaSeconds: Number(event.etaSeconds ?? 0),
            jobId: event.jobId,
            position: Math.max(1, position || total || 1),
            state: "queued",
            total: Math.max(position || 1, total || position || 1),
          };
        }
        setQueue(nextQueue);
        patchSession({
          currentChunk: event.currentChunk ?? 0,
          percent: event.percent ?? 0,
          processedChars: event.processedChars ?? 0,
          queue: nextQueue,
          stage: event.stage ?? "",
          tokensPerSecond: event.tokensPerSecond ?? 0,
          totalChars: event.totalChars ?? 0,
          totalChunks: event.totalChunks ?? 0,
        });
      },
      onChunkDelta: (event: AIChunkDeltaEvent) => {
        if (!isCurrentGeneration()) return;
        appendLiveChunk(event.chunkIndex, event.delta);
      },
      onChunkDone: (event: AIChunkDoneEvent) => {
        if (!isCurrentGeneration()) return;
        setLiveChunk(event.chunkIndex, event.text);
      },
      onReasoningDelta: (event: AIReasoningDeltaEvent) => {
        if (!isCurrentGeneration() || !event.reasoningDelta) return;
        liveReasoningActiveRef.current = true;
        liveReasoningRef.current += event.reasoningDelta;
        setReasoning(liveReasoningRef.current);
        patchSession({ reasoning: liveReasoningRef.current });
      },
      onReasoningDone: (event: AIReasoningDoneEvent) => {
        if (!isCurrentGeneration() || !event.reasoning) return;
        liveReasoningActiveRef.current = true;
        liveReasoningRef.current = event.reasoning;
        setReasoning(event.reasoning);
        patchSession({ reasoning: event.reasoning });
      },
      onFinal: (event: AIFinalEvent) => {
        if (!isCurrentGeneration()) return;
        const nextGeneratedTokens = aiGeneratedTokenCount(event.usage, event.text);
        liveOutputActiveRef.current = false;
        liveChunkPartsRef.current.clear();
        setResult(event.text);
        setPercent(100);
        setQueue(null);
        setGeneratedTokens(nextGeneratedTokens);
        setTokensPerSecond(event.tokensPerSecond ?? 0);
        patchSession({ generatedTokens: nextGeneratedTokens, percent: 100, queue: null, result: event.text, tokensPerSecond: event.tokensPerSecond ?? 0 });
      },
    };
  }

  return {
    createLiveHandlers,
    liveOutputActive,
    liveReasoningActive,
    liveReasoningText,
    liveResultText,
    resetLivePreview,
  };
}

function aiGeneratedTokenCount(usage: AIFinalEvent["usage"], text: string) {
  const tokens = usage?.completionTokens || usage?.totalTokens || 0;
  if (tokens > 0) {
    return tokens;
  }
  return estimateAITextTokens(text);
}

function estimateAITextTokens(text: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    return 0;
  }
  return Math.floor(Array.from(trimmed).length / 4) + 1;
}
