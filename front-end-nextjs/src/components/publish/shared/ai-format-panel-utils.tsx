"use client";

import type { AuthUser } from "@/lib/types";
import type { useTranslations } from "next-intl";
import type { QueueState } from "./ai-job-queue";

export function ProgressBar({ percent, label, wrapLabel = false }: { percent: number; label: string; wrapLabel?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="mb-1 flex items-start justify-between gap-3 text-xs font-semibold text-[#666c78]">
        <span className={wrapLabel ? "min-w-0 flex-1 break-words leading-5" : "truncate"}>{label}</span>
        <span>{percent}%</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-[#e8ecf3]">
        <div className="h-full rounded-full bg-[#1d4ed8] transition-[width]" style={{ width: `${Math.max(0, Math.min(100, percent))}%` }} />
      </div>
    </div>
  );
}

export function aiErrorLabel(t: ReturnType<typeof useTranslations<"publish.aiFormat">>, key: string) {
  const normalized = key.replace(/^error\./, "");
  const messageKey = `errors.${normalized}`;
  return t.has(messageKey) ? t(messageKey) : key;
}

export function aiStageLabel(t: ReturnType<typeof useTranslations<"publish.aiFormat">>, key: string) {
  const messageKey = `stages.${key}`;
  return t.has(messageKey) ? t(messageKey) : t("status.running");
}

export function formatQueueEta(t: ReturnType<typeof useTranslations<"publish.aiFormat">>, seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 60) {
    return t("eta.lessThanMinute");
  }
  return t("eta.minutes", { count: Math.ceil(seconds / 60) });
}

export function hashAIFormatSource(value: string) {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}

export function formatErrorDetail(value: unknown) {
  if (!value) return "";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function estimateAIFormatTextTokens(text: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    return 0;
  }
  return Math.floor(Array.from(trimmed).length / 4) + 1;
}

export function aiFormatLiveDisplay(
  t: ReturnType<typeof useTranslations<"publish.aiFormat">>,
  {
    generatedTokens,
    jobActorDisplayId,
    jobActorId,
    queue,
    viewerUser,
    result,
    running,
    tokensPerSecond,
  }: {
    generatedTokens: number;
    jobActorDisplayId?: string | null;
    jobActorId?: number | null;
    queue: QueueState | null;
    viewerUser?: AuthUser | null;
    result: string;
    running: boolean;
    tokensPerSecond: number;
  },
) {
  const activeActorId = queue?.active?.actorId ?? jobActorId ?? null;
  const activeActorDisplayId = queue?.active?.actorDisplayId ?? jobActorDisplayId ?? null;
  const actorUID = activeActorId ? String(activeActorId) : activeActorDisplayId?.trim() || "-";
  const localGeneratedTokenCount = Math.max(generatedTokens, estimateAIFormatTextTokens(result));
  const generatedTokenCount = queue?.active ? Math.max(0, queue.active.generatedTokens) : localGeneratedTokenCount;
  const speed = queue?.active?.tokensPerSecond && queue.active.tokensPerSecond > 0 ? queue.active.tokensPerSecond : !queue ? tokensPerSecond : 0;
  const queueAhead = queue ? Math.max(0, queue.position - 1) : 0;
  const isSelf = matchesUserIdentity(viewerUser, activeActorId, activeActorDisplayId);
  const queueLiveLabel = queue
    ? queueAhead > 0
      ? t("queueLiveStatus", { position: queueAhead, total: queue.total })
      : t("queueLiveStatusImmediate", { total: queue.total })
    : "";
  const runningLiveLabel = running && !queue ? t("runningLiveStatus") : "";
  const generatedTokenLabel = isSelf
    ? t("generatedTokensSelf", { count: generatedTokenCount })
    : t("generatedTokensOther", { uid: actorUID, count: generatedTokenCount });
  return {
    activeLiveLabel: queueLiveLabel ? [queueLiveLabel, generatedTokenLabel].join(" · ") : runningLiveLabel,
    generatedTokenCount,
    generatedTokenLabel,
    speedLabel: speed > 0 ? t("tokenSpeed", { value: speed.toFixed(1) }) : "",
  };
}

function matchesUserIdentity(user: AuthUser | null | undefined, actorId: number | null, actorDisplayId: string | null) {
  if (!user) {
    return false;
  }
  const currentUserIds = uniqueDefinedStrings([user.id, user.user_id, user.xise_id]);
  const actorIds = uniqueDefinedStrings([actorId, actorDisplayId]);
  return actorIds.some((id) => currentUserIds.includes(id));
}

function uniqueDefinedStrings(values: Array<number | string | null | undefined>) {
  return Array.from(
    new Set(
      values
        .map((value) => (value === null || value === undefined ? "" : String(value).trim()))
        .filter(Boolean),
    ),
  );
}

export function readStoredJobId(storagePrefix: string, sessionKey: string) {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(storageKey(storagePrefix, sessionKey)) ?? "";
}

export function storeJobId(storagePrefix: string, sessionKey: string, jobId: string) {
  if (typeof window !== "undefined" && jobId) {
    window.localStorage.setItem(storageKey(storagePrefix, sessionKey), jobId);
  }
}

export function clearStoredJobId(storagePrefix: string, sessionKey: string) {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem(storageKey(storagePrefix, sessionKey));
  }
}

function storageKey(storagePrefix: string, sessionKey: string) {
  return `${storagePrefix}${sessionKey}`;
}
