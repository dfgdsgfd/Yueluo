"use client";

import { useCallback, useEffect, useState } from "react";
import { ClipboardList, Eye, Gauge, Loader2, Radio, RefreshCw, Trash2, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { adminRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

import { RedisStatus } from "./observability-panel";
import { RedisMaintenancePanel } from "./redis-maintenance-panel";
import {
  EmptyBlock,
  HeaderCard,
  IconButton,
  InfoTile,
  KeyValueGrid,
  LoadingBlock,
  Panel,
  StatusPill,
  errorMessage,
  fieldText,
  formatCompact,
  formatMs,
  formatQueueTime,
  isHiddenDetailKey,
  readableValue,
  renderReadableValue,
  toneTextClass,
  type Tone,
} from "./shared";

type QueueStatsPayload = {
  enabled?: boolean;
  status?: Record<string, unknown>;
  runtimeStatus?: Record<string, unknown>;
  queues?: Record<string, QueueStat>;
  queueList?: QueueStat[];
  events?: Array<Record<string, unknown>>;
};

type QueueStat = Record<string, unknown> & {
  name?: string;
  waiting?: number;
  active?: number;
  completed?: number;
  failed?: number;
  delayed?: number;
  paused?: number;
};

type QueueJob = Record<string, unknown> & {
  id?: string | number;
  name?: string;
  data?: unknown;
  progress?: unknown;
  attemptsMade?: number;
  failedReason?: string;
  timestamp?: string | number;
  processedOn?: string | number;
  finishedOn?: string | number;
};

export function QueuesPanel({ token }: { token: string }) {
  const t = useTranslations("adminPortal.queuesPanel");
  const [stats, setStats] = useState<QueueStatsPayload | null>(null);
  const [names, setNames] = useState<string[]>([]);
  const [selected, setSelected] = useState("");
  const [status, setStatus] = useState("all");
  const [jobs, setJobs] = useState<unknown[]>([]);
  const [jobDetail, setJobDetail] = useState<QueueJob | null>(null);
  const [loading, setLoading] = useState(true);
  const [actingJob, setActingJob] = useState<string | number | null>(null);
  const [dangerConfirm, setDangerConfirm] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [queueStats, queueNames] = await Promise.all([
        adminRequest<QueueStatsPayload>("/api/admin/queues", { method: "GET", token }),
        adminRequest<{ names?: string[] }>("/api/admin/queue-names", { method: "GET", token }),
      ]);
      setStats(queueStats);
      const nextNames = queueNames.names ?? [];
      setNames(nextNames);
      setSelected((current) => current || nextNames[0] || "");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [token]);

  const loadJobs = useCallback(async () => {
    if (!selected) return;
    try {
      const data = await adminRequest<{ jobs?: unknown[] }>(`/api/admin/queues/${encodeURIComponent(selected)}/jobs`, {
        method: "GET",
        query: { status, start: 0, end: 30 },
        token,
      });
      setJobs(data.jobs ?? []);
    } catch {
      setJobs([]);
    }
  }, [selected, status, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  useEffect(() => {
    queueMicrotask(() => void loadJobs());
  }, [loadJobs]);

  async function cleanQueue() {
    if (!selected || dangerConfirm !== selected) return;
    try {
      await adminRequest(`/api/admin/queues/${encodeURIComponent(selected)}`, { method: "DELETE", token });
      toast.success(t("danger.cleared"));
      setDangerConfirm("");
      await Promise.all([load(), loadJobs()]);
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }

  async function inspectJob(job: QueueJob) {
    const jobID = queueJobID(job);
    if (!selected || !jobID) return;
    setActingJob(jobID);
    try {
      const data = await adminRequest<{ job?: QueueJob }>(`/api/admin/queues/${encodeURIComponent(selected)}/jobs/${encodeURIComponent(String(jobID))}`, { method: "GET", token });
      setJobDetail(data.job ?? job);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActingJob(null);
    }
  }

  async function retryJob(job: QueueJob) {
    const jobID = queueJobID(job);
    if (!selected || !jobID || !window.confirm(t("job.retryConfirm", { id: String(jobID) }))) return;
    setActingJob(jobID);
    try {
      await adminRequest(`/api/admin/queues/${encodeURIComponent(selected)}/jobs/${encodeURIComponent(String(jobID))}/retry`, { method: "POST", token });
      toast.success(t("job.retrySuccess"));
      setJobDetail(null);
      await loadJobs();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActingJob(null);
    }
  }

  return (
    <div className="grid gap-4">
      <HeaderCard icon={Gauge} title={t("title")} description={t("description")} tone="blue" />
      <RedisMaintenancePanel token={token} queueNames={names} />
      {loading ? (
        <LoadingBlock label={t("loading")} />
      ) : (
        <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(260px,320px)_minmax(0,1fr)]">
          <Panel title={t("queues")} icon={Radio}>
            <RedisStatus status={(stats?.runtimeStatus?.redis as Record<string, unknown> | undefined) ?? (stats?.status?.redis as Record<string, unknown> | undefined)} />
            <div className="mt-3 grid gap-2">
              {names.map((name) => (
                <button
                  key={name}
                  type="button"
                  onClick={() => {
                    setSelected(name);
                    setDangerConfirm("");
                  }}
                  className={cn("min-w-0 rounded-lg border px-3 py-2 text-left text-sm transition", selected === name ? "border-[#1d4ed8]/30 bg-[#eff6ff] text-[#1e3a8a]" : "border-black/[0.06] bg-[#fafbfe] text-[#3f444e] hover:bg-white")}
                >
                  <span className="block truncate font-semibold">{queueDisplayName(stats, name)}</span>
                  <span className="mt-0.5 block truncate text-xs text-[#7b808c]">{name}</span>
                </button>
              ))}
            </div>
          </Panel>

          <Panel
            title={selected ? t("taskTitle", { queue: selected }) : t("tasks")}
            icon={ClipboardList}
            action={
              <select value={status} onChange={(event) => setStatus(event.target.value)} className="h-9 max-w-full rounded-lg border border-black/[0.08] bg-white px-2 text-sm">
                {["all", "waiting", "active", "completed", "failed", "retry", "scheduled"].map((item) => (
                  <option key={item} value={item}>{t(`status.${item}`)}</option>
                ))}
              </select>
            }
          >
            <QueueStatsGrid stats={stats} selected={selected} />
            <QueueEvents events={stats?.events ?? []} selected={selected} />
            {jobs.length ? (
              <div className="grid gap-2">
                {jobs.map((job, index) => (
                  <QueueJobCard
                    key={queueJobKey(job, index)}
                    job={job as QueueJob}
                    acting={actingJob === queueJobID(job as QueueJob)}
                    onInspect={() => void inspectJob(job as QueueJob)}
                    onRetry={() => void retryJob(job as QueueJob)}
                  />
                ))}
              </div>
            ) : (
              <EmptyBlock icon={ClipboardList} label={t("emptyJobs")} />
            )}

            <div className="mt-4 rounded-xl border border-red-200 bg-red-50 p-4">
              <h3 className="text-sm font-semibold text-red-900">{t("danger.title")}</h3>
              <p className="mt-1 text-xs leading-5 text-red-800">{t("danger.description")}</p>
              <div className="mt-3 flex min-w-0 flex-col gap-2 sm:flex-row">
                <input
                  value={dangerConfirm}
                  onChange={(event) => setDangerConfirm(event.target.value)}
                  placeholder={t("danger.placeholder", { queue: selected || "-" })}
                  className="h-9 min-w-0 flex-1 rounded-lg border border-red-200 bg-white px-3 text-sm outline-none focus:border-red-500"
                />
                <Button type="button" variant="outline" disabled={!selected || dangerConfirm !== selected} onClick={() => void cleanQueue()} className="h-9 shrink-0 rounded-lg border-red-300 bg-white px-3 text-red-800 hover:bg-red-100">
                  <Trash2 className="size-4" />
                  {t("danger.clear")}
                </Button>
              </div>
            </div>
          </Panel>

          <QueueJobDetailDrawer job={jobDetail} onClose={() => setJobDetail(null)} onRetry={() => jobDetail ? void retryJob(jobDetail) : undefined} acting={actingJob === queueJobID(jobDetail ?? {})} />
        </section>
      )}
    </div>
  );
}

function QueueStatsGrid({ stats, selected }: { stats: QueueStatsPayload | null; selected: string }) {
  const t = useTranslations("adminPortal.queuesPanel");
  const stat = selected ? stats?.queues?.[selected] : null;
  const entries: Array<[string, unknown, Tone, boolean?]> = [
    [t("stats.waiting"), stat?.waiting ?? stat?.pending ?? 0, "amber"],
    [t("stats.active"), stat?.active ?? 0, "blue"],
    [t("stats.completed"), stat?.completed ?? 0, "green"],
    [t("stats.failed"), stat?.failed ?? 0, "red"],
    [t("stats.delayed"), stat?.delayed ?? 0, "purple"],
    [t("stats.latency"), stat?.latencyMs ?? 0, "slate", true],
  ];
  return (
    <div className="mb-3 grid gap-2 sm:grid-cols-2 xl:grid-cols-6">
      {entries.map(([label, value, tone, isDuration]) => (
        <div key={label} className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
          <p className="truncate text-xs text-[#7b808c]">{label}</p>
          <p className={cn("mt-1 truncate text-lg font-semibold", toneTextClass(tone))}>{isDuration ? formatMs(value) : formatCompact(value)}</p>
        </div>
      ))}
      {stats?.enabled === false ? <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 sm:col-span-2 xl:col-span-6">{t("serviceDisabled")}</div> : null}
    </div>
  );
}

function QueueEvents({ events, selected }: { events: Array<Record<string, unknown>>; selected: string }) {
  const t = useTranslations("adminPortal.queuesPanel");
  const scoped = selected ? events.filter((item) => item.queue === selected).slice(0, 6) : events.slice(0, 6);
  if (!scoped.length) return null;
  return (
    <div className="mb-3 rounded-lg border border-black/[0.06] bg-white p-3">
      <p className="mb-2 text-sm font-semibold text-[#252932]">{t("recentEvents")}</p>
      <div className="grid gap-2 md:grid-cols-2">
        {scoped.map((event, index) => (
          <div key={`${event.id ?? event.task_id ?? index}`} className="min-w-0 rounded-lg bg-[#f8fafc] px-3 py-2 text-xs">
            <div className="flex min-w-0 items-center gap-2">
              <StatusPill value={String(event.event ?? "-")} tone={queueEventTone(event.event)} />
              <span className="truncate text-[#252932]">{String(event.task_id ?? event.id ?? "-")}</span>
            </div>
            <p className="mt-1 text-[#7b808c]">{t("eventTiming", { time: formatQueueTime(event.at), wait: formatMs(event.latencyMs), duration: formatMs(event.durationMs) })}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

function QueueJobCard({ job, acting, onInspect, onRetry }: { job: QueueJob; acting?: boolean; onInspect?: () => void; onRetry?: () => void }) {
  const t = useTranslations("adminPortal.queuesPanel");
  return (
    <article className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
      <div className="mb-2 flex min-w-0 items-center justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-semibold text-[#252932]">{fieldText(job, "name")}</h3>
          <p className="truncate text-xs text-[#8b919e]">{t("job.id", { id: fieldText(job, "id") })}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <StatusPill value={String(job.status ?? job.state ?? "unknown")} tone={job.failedReason || job.error ? "red" : "slate"} />
          {onInspect ? <IconButton label={t("job.inspect")} icon={Eye} onClick={onInspect} /> : null}
          {onRetry ? (
            <button type="button" aria-label={t("job.retry")} title={t("job.retry")} disabled={acting} onClick={onRetry} className="inline-flex size-8 items-center justify-center rounded-lg text-[#59606c] transition hover:bg-[#edf0f5] disabled:opacity-50">
              {acting ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
            </button>
          ) : null}
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-4">
        <InfoTile label={t("job.enqueued")} value={formatQueueTime(job.enqueuedAt ?? job.queuedAt ?? job.timestamp)} />
        <InfoTile label={t("job.started")} value={formatQueueTime(job.startedAt ?? job.processedOn)} />
        <InfoTile label={t("job.wait")} value={formatMs(job.waitMs)} />
        <InfoTile label={t("job.duration")} value={formatMs(job.processMs ?? job.durationMs)} />
      </div>
      {job.failedReason || job.error ? <p className="mt-2 break-words rounded-lg bg-[#fff0f2] px-3 py-2 text-xs text-[#b0122a]">{String(job.failedReason ?? job.error)}</p> : null}
      {job.data !== undefined ? <div className="mt-2 min-w-0 overflow-x-auto text-xs text-[#6f7582]">{renderReadableValue(job.data)}</div> : null}
    </article>
  );
}

function QueueJobDetailDrawer({ job, acting, onClose, onRetry }: { job: QueueJob | null; acting?: boolean; onClose: () => void; onRetry: () => void }) {
  const t = useTranslations("adminPortal.queuesPanel");
  if (!job) return null;
  const entries = Object.entries(job).filter(([key, value]) => !isHiddenDetailKey(key, value));
  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label={t("job.close")} className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[560px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">{t("job.details")}</h2>
          <Button type="button" variant="outline" size="sm" disabled={acting} onClick={onRetry} className="rounded-lg border-black/[0.08] bg-white">
            {acting ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
            {t("job.retry")}
          </Button>
          <Button type="button" size="icon" variant="ghost" aria-label={t("job.close")} onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">
          <div className="mb-3 grid gap-2 sm:grid-cols-3">
            <InfoTile label={t("job.idLabel")} value={readableValue(job.id)} />
            <InfoTile label={t("job.name")} value={readableValue(job.name)} />
            <InfoTile label={t("job.attempts")} value={formatCompact(job.attemptsMade ?? job.attempts ?? 0)} />
          </div>
          <KeyValueGrid entries={entries.filter(([key]) => !["id", "name", "attemptsMade"].includes(key))} />
        </div>
      </aside>
    </div>
  );
}

function queueDisplayName(stats: QueueStatsPayload | null, name: string) {
  const stat = stats?.queues?.[name];
  return String(stat?.label ?? stat?.kind ?? name);
}

function queueJobKey(job: unknown, index: number) {
  const id = queueJobID((job ?? {}) as QueueJob);
  return id ? String(id) : String(index);
}

function queueJobID(job: QueueJob) {
  const value = job.id ?? job.task_id ?? job.taskId;
  return typeof value === "string" || typeof value === "number" ? value : "";
}

function queueEventTone(value: unknown): Tone {
  switch (String(value)) {
    case "completed":
      return "green";
    case "failed":
      return "red";
    case "started":
      return "blue";
    case "duplicate":
      return "amber";
    default:
      return "slate";
  }
}
