"use client";

import { CircleDot, ListOrdered, TimerReset } from "lucide-react";
import type { useTranslations } from "next-intl";
import type { QueueState } from "./ai-job-queue";

type AIFormatQueueStatusProps = {
  generatedTokens: string;
  jobId?: string;
  queue: QueueState | null;
  queueDetail?: string;
  queuePosition: string;
  queueTotal: string;
  status: string;
  updated: string;
};

export function AIFormatQueueStatus({
  generatedTokens,
  jobId,
  queue,
  queueDetail,
  queuePosition,
  queueTotal,
  status,
  updated,
}: AIFormatQueueStatusProps) {
  const detailItems = queue
    ? [queuePosition, queueTotal].filter(Boolean)
    : [generatedTokens || updated].filter(Boolean);
  const queueTokenLine = queue ? generatedTokens : "";
  return (
    <div className="overflow-hidden rounded-xl border border-[#1d4ed8]/15 bg-white shadow-sm">
      <div className="flex min-w-0 flex-col gap-3 p-3 sm:flex-row sm:items-center">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <span className="relative flex size-3 shrink-0 rounded-full bg-[#1d4ed8]" aria-hidden="true">
            <span className="absolute inset-0 rounded-full bg-[#1d4ed8] opacity-35 motion-safe:animate-ping motion-reduce:hidden" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <span className="text-sm font-semibold text-[#20232a]">{status}</span>
              {queue ? (
                <span className="rounded-full bg-[#eef4ff] px-2 py-0.5 text-[11px] font-semibold text-[#1d4ed8]">
                  {queue.position}/{queue.total}
                </span>
              ) : null}
              {jobId ? <span className="truncate text-[11px] font-medium text-[#8b93a2]">{jobId}</span> : null}
            </div>
            {queueDetail ? <p className="mt-1 break-words text-xs leading-5 text-[#737987]">{queueDetail}</p> : null}
            {queueTokenLine ? <p className="mt-1 break-words text-xs font-semibold leading-5 text-[#1d4ed8]">{queueTokenLine}</p> : null}
            {detailItems.length > 0 ? (
              <div className="mt-2 flex min-w-0 flex-wrap gap-1.5">
                {detailItems.map((item) => (
                  <QueueDetailPill key={item} value={item} />
                ))}
              </div>
            ) : null}
          </div>
        </div>
        <div className="grid min-w-0 grid-cols-1 gap-2 text-xs sm:w-[360px] sm:grid-cols-3">
          <QueueMetric icon={ListOrdered} value={queue ? queuePosition : "-"} />
          <QueueMetric icon={CircleDot} value={queue ? queueTotal : "-"} />
          <QueueMetric icon={TimerReset} value={queue ? updated : generatedTokens || updated} />
        </div>
      </div>
      <div className="h-1 overflow-hidden bg-[#edf2f7]">
        <div className="h-full w-2/3 bg-[#1d4ed8] motion-safe:animate-pulse" />
      </div>
    </div>
  );
}

function QueueDetailPill({ value }: { value: string }) {
  return (
    <span className="min-w-0 rounded-full bg-[#eef4ff] px-2 py-1 text-xs font-semibold text-[#1d4ed8]">
      {value}
    </span>
  );
}

function QueueMetric({ icon: Icon, value }: { icon: typeof TimerReset; value: string }) {
  return (
    <span className="flex min-w-0 items-center gap-1 rounded-lg bg-[#f7f9fc] px-2 py-1.5 font-semibold text-[#536070]">
      <Icon className="size-3 shrink-0 text-[#1d4ed8]" />
      <span className="truncate">{value}</span>
    </span>
  );
}

export function formatAIFormatRelativeTime(t: ReturnType<typeof useTranslations<"publish.aiFormat">>, value?: string | null) {
  if (!value) {
    return t("time.unknown");
  }
  const date = new Date(value);
  const timestamp = date.getTime();
  if (!Number.isFinite(timestamp)) {
    return t("time.unknown");
  }
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (seconds < 60) {
    return t("time.secondsAgo", { count: seconds });
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return t("time.minutesAgo", { count: minutes });
  }
  return t("time.hoursAgo", { count: Math.floor(minutes / 60) });
}
