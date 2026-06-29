import { FileText, PencilLine, WandSparkles } from "lucide-react";
import type { AIJobPayload } from "@/lib/types";
import type { AIFormatPhase } from "./ai-format-thinking-ticker";
import type { QueueState } from "./ai-job-queue";

export type AIFormatMode = "format" | "polish" | "custom";

export type AIFormatSession = {
  actorDisplayId?: string | null;
  actorId?: number | null;
  currentChunk: number;
  error: string;
  errorDetail: string;
  generatedTokens: number;
  jobId: string;
  mode: AIFormatMode;
  original: string;
  percent: number;
  phase: AIFormatPhase;
  processedChars: number;
  queue: QueueState | null;
  reasoning: string;
  result: string;
  stage: string;
  tokensPerSecond: number;
  totalChars: number;
  totalChunks: number;
  updatedAt?: string | null;
};

export const formatSessions = new Map<string, AIFormatSession>();
export const formatJobStoragePrefix = "yuem:ai-format-job:";
export const runningJobStatuses = new Set(["queued", "running"]);

export const modeControls = [
  { key: "format" as const, icon: FileText, taskType: "format_markdown", templateKey: "markdown_format" },
  { key: "polish" as const, icon: WandSparkles, taskType: "post_polish", templateKey: "post_polish" },
  { key: "custom" as const, icon: PencilLine, taskType: "post_custom_generate", templateKey: "post_custom_generate" },
] as const;

export function phaseFromJobStatus(status: string): AIFormatPhase {
  switch (status) {
    case "queued":
    case "running":
      return "running";
    case "completed":
      return "done";
    case "failed":
    case "canceled":
      return "error";
    default:
      return "running";
  }
}

export function aiJobGeneratedTokenCount(job: AIJobPayload) {
  const tokens = job.completionTokens || job.totalTokens || 0;
  if (tokens > 0) {
    return tokens;
  }
  return estimateAIFormatTextTokens(job.output ?? "");
}

function estimateAIFormatTextTokens(text: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    return 0;
  }
  return Math.floor(Array.from(trimmed).length / 4) + 1;
}
