import type { AIJobPayload } from "@/lib/types";

export type QueueState = {
  active?: {
    actorDisplayId?: string | null;
    actorId?: number | null;
    generatedTokens: number;
    jobId?: string;
    tokensPerSecond?: number;
  } | null;
  etaSeconds: number;
  jobId?: string;
  position: number;
  state?: string;
  total: number;
};

export function queueFromAIJob(job: AIJobPayload): QueueState | null {
  if (job.status !== "queued" && job.stage !== "queued") {
    return null;
  }
  const queueJob = job.queueJob;
  if (!queueJob || typeof queueJob !== "object") {
    return fallbackQueuedState(job);
  }
  const position = numberFromQueueValue(queueJob.queuePosition);
  const total = numberFromQueueValue(queueJob.queueCount ?? queueJob.queueTotal);
  if (position <= 0 && total <= 0) {
    return fallbackQueuedState(job);
  }
  return {
    active: activeQueueJobFromValue(queueJob.activeJob),
    etaSeconds: Math.max(0, numberFromQueueValue(queueJob.estimatedWaitSeconds ?? queueJob.etaSeconds)),
    jobId: typeof queueJob.jobId === "string" ? queueJob.jobId : job.jobId,
    position: Math.max(1, position || total || 1),
    state: typeof queueJob.state === "string" ? queueJob.state : job.status,
    total: Math.max(position || 1, total || position || 1),
  };
}

function fallbackQueuedState(job: AIJobPayload): QueueState {
  return {
    active: null,
    etaSeconds: 0,
    jobId: job.jobId,
    position: 1,
    state: job.status || "queued",
    total: 1,
  };
}

export function activeQueueJobFromProgress(event: {
  activeActorDisplayId?: string;
  activeActorId?: number;
  activeGeneratedTokens?: number;
  activeJobId?: string;
  activeTokensPerSecond?: number;
}) {
  const generatedTokens = numberFromQueueValue(event.activeGeneratedTokens);
  if (!event.activeJobId && !event.activeActorId && generatedTokens <= 0) {
    return null;
  }
  return {
    actorDisplayId: typeof event.activeActorDisplayId === "string" ? event.activeActorDisplayId : null,
    actorId: numberFromOptionalValue(event.activeActorId),
    generatedTokens: Math.max(0, generatedTokens),
    jobId: typeof event.activeJobId === "string" ? event.activeJobId : undefined,
    tokensPerSecond: numberFromOptionalValue(event.activeTokensPerSecond) ?? undefined,
  };
}

function activeQueueJobFromValue(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const active = value as Record<string, unknown>;
  const generatedTokens = numberFromQueueValue(active.generatedTokens);
  if (!active.jobId && !active.actorId && generatedTokens <= 0) {
    return null;
  }
  return {
    actorDisplayId: typeof active.actorDisplayId === "string" ? active.actorDisplayId : null,
    actorId: numberFromOptionalValue(active.actorId),
    generatedTokens: Math.max(0, generatedTokens),
    jobId: typeof active.jobId === "string" ? active.jobId : undefined,
    tokensPerSecond: numberFromOptionalValue(active.tokensPerSecond) ?? undefined,
  };
}

function numberFromOptionalValue(value: unknown) {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) && parsed > 0 ? Math.trunc(parsed) : null;
}

function numberFromQueueValue(value: unknown) {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? Math.trunc(parsed) : 0;
}
