"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { DatabaseZap, Eraser, Loader2, Save, ShieldCheck } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { adminRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

import { formatBytes, formatDateTime, Panel, errorMessage } from "./shared";

type QueuePolicy = {
  completed_retention_hours: number;
  archived_retention_hours: number;
  archived_max_tasks: number;
};

type RefreshTokenMode = "redis_opaque" | "jwt_legacy";

type MaintenanceConfig = {
  enabled: boolean;
  interval_minutes: number;
  access_log_retention_hours: number;
  access_log_max_entries: number;
  system_log_retention_hours: number;
  system_log_max_entries: number;
  metrics_retention_hours: number;
  metrics_max_entries_per_key: number;
  queue_event_retention_hours: number;
  queue_event_max_entries: number;
  completed_retention_hours: number;
  archived_retention_hours: number;
  archived_max_tasks_per_queue: number;
  memory_warning_percent: number;
  memory_critical_percent: number;
  access_token_ttl_seconds: number;
  refresh_token_active_ttl_seconds: number;
  refresh_token_renewal_interval_seconds: number;
  refresh_token_mode: RefreshTokenMode;
  refresh_token_auto_renew_enabled: boolean;
  session_inactive_ttl_seconds: number;
  user_active_session_limit: number;
  queue_overrides: Record<string, QueuePolicy>;
  next_run_at?: string;
  last_run_at?: string;
  last_result?: Record<string, unknown>;
};

type InventoryCategory = {
  keys?: number;
  memory_bytes?: number;
  no_ttl_keys?: number;
  expected_permanent_keys?: number;
};

type MaintenanceStatus = {
  configured?: boolean;
  available?: boolean;
  running?: boolean;
  config?: MaintenanceConfig;
  redis?: {
    info?: Record<string, unknown>;
  };
  pressure?: {
    used_bytes?: number;
    max_bytes?: number;
    percent?: number;
    level?: string;
    warning?: number;
    critical?: number;
  };
  inventory?: {
    database_keys?: number;
    scanned_keys?: number;
    truncated?: boolean;
    no_ttl_keys?: number;
    expected_permanent_keys?: number;
    categories?: Record<string, InventoryCategory>;
  };
};

const defaultConfig: MaintenanceConfig = {
  enabled: true,
  interval_minutes: 60,
  access_log_retention_hours: 24,
  access_log_max_entries: 50000,
  system_log_retention_hours: 168,
  system_log_max_entries: 20000,
  metrics_retention_hours: 24,
  metrics_max_entries_per_key: 50000,
  queue_event_retention_hours: 24,
  queue_event_max_entries: 20000,
  completed_retention_hours: 24,
  archived_retention_hours: 720,
  archived_max_tasks_per_queue: 1000,
  memory_warning_percent: 75,
  memory_critical_percent: 90,
  access_token_ttl_seconds: 3600,
  refresh_token_active_ttl_seconds: 7776000,
  refresh_token_renewal_interval_seconds: 86400,
  refresh_token_mode: "redis_opaque",
  refresh_token_auto_renew_enabled: true,
  session_inactive_ttl_seconds: 604800,
  user_active_session_limit: 5,
  queue_overrides: {},
};

export function RedisMaintenancePanel({ token, queueNames }: { token: string; queueNames: string[] }) {
  const t = useTranslations("adminPortal.queuesPanel");
  const [status, setStatus] = useState<MaintenanceStatus | null>(null);
  const [draft, setDraft] = useState<MaintenanceConfig>(defaultConfig);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState("");

  const load = useCallback(async (quiet = false) => {
    if (!quiet) setLoading(true);
    try {
      const next = await adminRequest<MaintenanceStatus>("/api/admin/redis-maintenance", { method: "GET", token });
      setStatus(next);
      if (!quiet) setDraft(normalizeConfig(next.config));
    } catch (error) {
      if (!quiet) toast.error(errorMessage(error));
    } finally {
      if (!quiet) setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => void load());
    const timer = window.setInterval(() => void load(true), 60_000);
    return () => window.clearInterval(timer);
  }, [load]);

  const inventory = useMemo(
    () => Object.entries(status?.inventory?.categories ?? {}).sort((a, b) => Number(b[1].memory_bytes ?? 0) - Number(a[1].memory_bytes ?? 0)),
    [status?.inventory?.categories],
  );

  async function save() {
    setActing("save");
    try {
      const next = await adminRequest<MaintenanceStatus>("/api/admin/redis-maintenance", {
        method: "PUT",
        token,
        body: JSON.stringify(draft),
      });
      setStatus(next);
      setDraft(normalizeConfig(next.config));
      toast.success(t("maintenance.saved"));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing("");
    }
  }

  async function run(categories: string[]) {
    if (categories.includes("cache") && !window.confirm(t("maintenance.cacheConfirm"))) return;
    setActing(categories.includes("cache") ? "cache" : "safe");
    try {
      const payload = await adminRequest<{ status?: MaintenanceStatus }>("/api/admin/redis-maintenance/run", {
        method: "POST",
        token,
        body: JSON.stringify({ categories }),
      });
      if (payload.status) {
        setStatus(payload.status);
        setDraft(normalizeConfig(payload.status.config));
      } else {
        await load();
      }
      toast.success(t("maintenance.runComplete"));
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing("");
    }
  }

  function setNumber(key: keyof MaintenanceConfig, value: string) {
    setDraft((current) => ({ ...current, [key]: Number(value) }));
  }

  function setRefreshTokenMode(value: RefreshTokenMode) {
    setDraft((current) => ({ ...current, refresh_token_mode: value }));
  }

  function toggleOverride(queue: string) {
    setDraft((current) => {
      const overrides = { ...current.queue_overrides };
      if (overrides[queue]) {
        delete overrides[queue];
      } else {
        overrides[queue] = {
          completed_retention_hours: current.completed_retention_hours,
          archived_retention_hours: current.archived_retention_hours,
          archived_max_tasks: current.archived_max_tasks_per_queue,
        };
      }
      return { ...current, queue_overrides: overrides };
    });
  }

  function setOverride(queue: string, key: keyof QueuePolicy, value: string) {
    setDraft((current) => ({
      ...current,
      queue_overrides: {
        ...current.queue_overrides,
        [queue]: {
          ...(current.queue_overrides[queue] ?? {
            completed_retention_hours: current.completed_retention_hours,
            archived_retention_hours: current.archived_retention_hours,
            archived_max_tasks: current.archived_max_tasks_per_queue,
          }),
          [key]: Number(value),
        },
      },
    }));
  }

  const pressure = status?.pressure;
  const percent = Math.max(0, Math.min(100, Number(pressure?.percent ?? 0)));
  const info = status?.redis?.info ?? {};

  return (
    <Panel title={t("maintenance.title")} icon={DatabaseZap}>
      {loading ? (
        <div className="flex min-h-32 items-center justify-center text-sm text-[#7b808c]">
          <Loader2 className="mr-2 size-4 animate-spin" />
          {t("maintenance.loading")}
        </div>
      ) : (
        <div className="grid min-w-0 gap-4">
          <div className="grid gap-3 lg:grid-cols-[minmax(0,1.2fr)_minmax(280px,0.8fr)]">
            <section className="min-w-0 rounded-xl border border-black/[0.06] bg-[#fafbfe] p-4">
              <div className="flex min-w-0 items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="font-semibold text-[#252932]">{t("memory.title")}</h3>
                  <p className="mt-1 text-xs text-[#7b808c]">{t("maintenance.description")}</p>
                </div>
                <span className={cn("shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold", pressureTone(pressure?.level))}>
                  {t(`memory.level.${pressure?.level ?? "unbounded"}`)}
                </span>
              </div>
              <div className="mt-4 h-2 overflow-hidden rounded-full bg-[#e7eaf0]">
                <div className={cn("h-full rounded-full transition-all", pressureBar(pressure?.level))} style={{ width: `${pressure?.max_bytes ? percent : 0}%` }} />
              </div>
              <div className="mt-3 grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
                <Metric label={t("memory.used")} value={formatBytes(pressure?.used_bytes)} />
                <Metric label={t("memory.max")} value={pressure?.max_bytes ? formatBytes(pressure.max_bytes) : t("memory.unbounded")} />
                <Metric label={t("memory.policy")} value={String(info.maxmemory_policy ?? "-")} />
                <Metric label={t("memory.fragmentation")} value={String(info.mem_fragmentation_ratio ?? "-")} />
              </div>
              {!pressure?.max_bytes ? (
                <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-900">
                  {t("memory.noLimitHint")}
                </p>
              ) : null}
            </section>

            <section className="min-w-0 rounded-xl border border-black/[0.06] bg-white p-4">
              <h3 className="font-semibold text-[#252932]">{t("schedule.title")}</h3>
              <div className="mt-3 grid gap-2 sm:grid-cols-3 lg:grid-cols-1">
                <Metric label={t("schedule.next")} value={formatDateTime(draft.next_run_at)} />
                <Metric label={t("schedule.last")} value={formatDateTime(draft.last_run_at)} />
                <Metric label={t("schedule.freed")} value={formatBytes(recordNumber(draft.last_result, "memory_freed_bytes"))} />
              </div>
            </section>
          </div>

          <section className="rounded-xl border border-black/[0.06] bg-white p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="font-semibold text-[#252932]">{t("inventory.title")}</h3>
                <p className="mt-1 text-xs text-[#7b808c]">
                  {t("inventory.summary", {
                    total: status?.inventory?.database_keys ?? 0,
                    scanned: status?.inventory?.scanned_keys ?? 0,
                  })}
                </p>
                {(status?.inventory?.no_ttl_keys ?? 0) > 0 ? (
                  <p className="mt-1 text-xs text-amber-700">{t("inventory.noTtlSummary", { count: status?.inventory?.no_ttl_keys ?? 0 })}</p>
                ) : null}
                {(status?.inventory?.expected_permanent_keys ?? 0) > 0 ? (
                  <p className="mt-1 text-xs text-[#7b808c]">{t("inventory.expectedPermanentSummary", { count: status?.inventory?.expected_permanent_keys ?? 0 })}</p>
                ) : null}
              </div>
              {status?.inventory?.truncated ? <span className="text-xs text-amber-700">{t("inventory.sampled")}</span> : null}
            </div>
            <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
              {inventory.length ? inventory.map(([category, item]) => (
                <div key={category} className="min-w-0 rounded-lg bg-[#f8fafc] px-3 py-2">
                  <p className="truncate text-xs text-[#7b808c]">{t(`inventory.categories.${category}`)}</p>
                  <p className="mt-1 font-semibold text-[#252932]">{formatBytes(item.memory_bytes)}</p>
                  <p className="text-xs text-[#8b919e]">{t("inventory.keys", { count: item.keys ?? 0 })}</p>
                  {(item.no_ttl_keys ?? 0) > 0 ? <p className="text-xs text-amber-700">{t("inventory.noTtlKeys", { count: item.no_ttl_keys ?? 0 })}</p> : null}
                  {(item.expected_permanent_keys ?? 0) > 0 ? <p className="text-xs text-[#8b919e]">{t("inventory.expectedPermanentKeys", { count: item.expected_permanent_keys ?? 0 })}</p> : null}
                </div>
              )) : <p className="text-sm text-[#7b808c]">{t("inventory.empty")}</p>}
            </div>
          </section>

          <section className="rounded-xl border border-black/[0.06] bg-[#fafbfe] p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="font-semibold text-[#252932]">{t("maintenance.policyTitle")}</h3>
                <p className="mt-1 text-xs text-[#7b808c]">{t("maintenance.safeBoundary")}</p>
              </div>
              <label className="inline-flex items-center gap-2 text-sm font-medium text-[#343944]">
                <input type="checkbox" checked={draft.enabled} onChange={(event) => setDraft((current) => ({ ...current, enabled: event.target.checked }))} className="size-4 accent-[#1d4ed8]" />
                {t("fields.enabled")}
              </label>
            </div>
            <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
              <NumberField label={t("fields.interval")} value={draft.interval_minutes} min={5} max={1440} onChange={(value) => setNumber("interval_minutes", value)} />
              <NumberField label={t("fields.accessLogs")} value={draft.access_log_retention_hours} min={1} max={720} onChange={(value) => setNumber("access_log_retention_hours", value)} />
              <NumberField label={t("fields.accessLogMax")} value={draft.access_log_max_entries} min={1000} max={1000000} onChange={(value) => setNumber("access_log_max_entries", value)} />
              <NumberField label={t("fields.systemLogs")} value={draft.system_log_retention_hours} min={1} max={2160} onChange={(value) => setNumber("system_log_retention_hours", value)} />
              <NumberField label={t("fields.systemLogMax")} value={draft.system_log_max_entries} min={1000} max={500000} onChange={(value) => setNumber("system_log_max_entries", value)} />
              <NumberField label={t("fields.metrics")} value={draft.metrics_retention_hours} min={1} max={720} onChange={(value) => setNumber("metrics_retention_hours", value)} />
              <NumberField label={t("fields.metricsMax")} value={draft.metrics_max_entries_per_key} min={1000} max={1000000} onChange={(value) => setNumber("metrics_max_entries_per_key", value)} />
              <NumberField label={t("fields.queueEvents")} value={draft.queue_event_retention_hours} min={1} max={720} onChange={(value) => setNumber("queue_event_retention_hours", value)} />
              <NumberField label={t("fields.queueEventMax")} value={draft.queue_event_max_entries} min={1000} max={500000} onChange={(value) => setNumber("queue_event_max_entries", value)} />
              <NumberField label={t("fields.completed")} value={draft.completed_retention_hours} min={0} max={720} onChange={(value) => setNumber("completed_retention_hours", value)} />
              <NumberField label={t("fields.archived")} value={draft.archived_retention_hours} min={1} max={2160} onChange={(value) => setNumber("archived_retention_hours", value)} />
              <NumberField label={t("fields.archivedMax")} value={draft.archived_max_tasks_per_queue} min={1} max={10000} onChange={(value) => setNumber("archived_max_tasks_per_queue", value)} />
              <NumberField label={t("fields.warning")} value={draft.memory_warning_percent} min={10} max={95} onChange={(value) => setNumber("memory_warning_percent", value)} />
              <NumberField label={t("fields.critical")} value={draft.memory_critical_percent} min={11} max={100} onChange={(value) => setNumber("memory_critical_percent", value)} />
              <NumberField label={t("fields.accessTokenTTLSeconds")} value={draft.access_token_ttl_seconds} min={60} max={86400} onChange={(value) => setNumber("access_token_ttl_seconds", value)} />
              <NumberField label={t("fields.refreshTokenActiveTTLSeconds")} value={draft.refresh_token_active_ttl_seconds} min={3600} max={31536000} onChange={(value) => setNumber("refresh_token_active_ttl_seconds", value)} />
              <NumberField label={t("fields.refreshTokenRenewalIntervalSeconds")} value={draft.refresh_token_renewal_interval_seconds} min={3600} max={2592000} onChange={(value) => setNumber("refresh_token_renewal_interval_seconds", value)} />
              <SelectField
                label={t("fields.refreshTokenMode")}
                value={draft.refresh_token_mode}
                onChange={setRefreshTokenMode}
                options={[
                  { value: "redis_opaque", label: t("fields.refreshTokenModeRedisOpaque") },
                  { value: "jwt_legacy", label: t("fields.refreshTokenModeJWTLegacy") },
                ]}
              />
              <label className="flex min-w-0 items-center gap-2 pt-5 text-sm font-medium text-[#343944]">
                <input type="checkbox" checked={draft.refresh_token_auto_renew_enabled} onChange={(event) => setDraft((current) => ({ ...current, refresh_token_auto_renew_enabled: event.target.checked }))} className="size-4 accent-[#1d4ed8]" />
                <span className="min-w-0 truncate">{t("fields.refreshTokenAutoRenewEnabled")}</span>
              </label>
              <NumberField label={t("fields.sessionInactiveTTLSeconds")} value={draft.session_inactive_ttl_seconds} min={3600} max={31536000} onChange={(value) => setNumber("session_inactive_ttl_seconds", value)} />
              <NumberField label={t("fields.userActiveSessionLimit")} value={draft.user_active_session_limit} min={1} max={100} onChange={(value) => setNumber("user_active_session_limit", value)} />
            </div>
          </section>

          <section className="rounded-xl border border-black/[0.06] bg-white p-4">
            <h3 className="font-semibold text-[#252932]">{t("overrides.title")}</h3>
            <p className="mt-1 text-xs text-[#7b808c]">{t("overrides.description")}</p>
            <div className="mt-3 grid gap-2">
              {queueNames.map((queue) => {
                const policy = draft.queue_overrides[queue];
                return (
                  <div key={queue} className="grid min-w-0 gap-2 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 md:grid-cols-[minmax(180px,1fr)_auto]">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-[#252932]">{queue}</p>
                      <label className="mt-1 inline-flex items-center gap-2 text-xs text-[#6f7582]">
                        <input type="checkbox" checked={Boolean(policy)} onChange={() => toggleOverride(queue)} className="size-4 accent-[#1d4ed8]" />
                        {policy ? t("overrides.custom") : t("overrides.default")}
                      </label>
                    </div>
                    {policy ? (
                      <div className="grid gap-2 sm:grid-cols-3">
                        <CompactNumber label={t("fields.completed")} value={policy.completed_retention_hours} min={0} max={720} onChange={(value) => setOverride(queue, "completed_retention_hours", value)} />
                        <CompactNumber label={t("fields.archived")} value={policy.archived_retention_hours} min={1} max={2160} onChange={(value) => setOverride(queue, "archived_retention_hours", value)} />
                        <CompactNumber label={t("fields.archivedMax")} value={policy.archived_max_tasks} min={1} max={10000} onChange={(value) => setOverride(queue, "archived_max_tasks", value)} />
                      </div>
                    ) : null}
                  </div>
                );
              })}
            </div>
          </section>

          <div className="flex flex-wrap items-center gap-2">
            <Button type="button" disabled={Boolean(acting)} onClick={() => void save()} className="rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
              {acting === "save" ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              {t("maintenance.save")}
            </Button>
            <Button type="button" variant="outline" disabled={Boolean(acting) || status?.available === false} onClick={() => void run(["observability", "queue_history", "sessions"])} className="rounded-lg border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-100">
              {acting === "safe" ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
              {t("maintenance.runSafe")}
            </Button>
            <Button type="button" variant="outline" disabled={Boolean(acting) || status?.available === false} onClick={() => void run(["cache"])} className="rounded-lg border-amber-200 bg-amber-50 text-amber-900 hover:bg-amber-100">
              {acting === "cache" ? <Loader2 className="size-4 animate-spin" /> : <Eraser className="size-4" />}
              {t("maintenance.runCache")}
            </Button>
          </div>
        </div>
      )}
    </Panel>
  );
}

function normalizeConfig(value?: MaintenanceConfig): MaintenanceConfig {
  return {
    ...defaultConfig,
    ...(value ?? {}),
    queue_overrides: { ...(value?.queue_overrides ?? {}) },
  };
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg border border-black/[0.05] bg-white px-3 py-2">
      <p className="truncate text-xs text-[#7b808c]">{label}</p>
      <p className="mt-1 truncate text-sm font-semibold text-[#252932]">{value}</p>
    </div>
  );
}

function NumberField({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (value: string) => void }) {
  return (
    <label className="min-w-0">
      <span className="mb-1 block truncate text-xs font-semibold text-[#5f6570]">{label}</span>
      <input type="number" min={min} max={max} value={value} onChange={(event) => onChange(event.target.value)} className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]" />
    </label>
  );
}

function SelectField({ label, value, options, onChange }: { label: string; value: RefreshTokenMode; options: Array<{ value: RefreshTokenMode; label: string }>; onChange: (value: RefreshTokenMode) => void }) {
  return (
    <label className="min-w-0">
      <span className="mb-1 block truncate text-xs font-semibold text-[#5f6570]">{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value as RefreshTokenMode)} className="h-10 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]">
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function CompactNumber({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (value: string) => void }) {
  return (
    <label className="min-w-0">
      <span className="mb-1 block truncate text-[11px] text-[#7b808c]">{label}</span>
      <input type="number" min={min} max={max} value={value} onChange={(event) => onChange(event.target.value)} className="h-9 w-full min-w-0 rounded-lg border border-black/[0.08] bg-white px-2 text-sm outline-none focus:border-[#1d4ed8]" />
    </label>
  );
}

function pressureTone(level?: string) {
  switch (level) {
    case "critical":
      return "bg-red-100 text-red-800";
    case "warning":
      return "bg-amber-100 text-amber-800";
    case "normal":
      return "bg-emerald-100 text-emerald-800";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

function pressureBar(level?: string) {
  switch (level) {
    case "critical":
      return "bg-red-500";
    case "warning":
      return "bg-amber-500";
    default:
      return "bg-emerald-500";
  }
}

function recordNumber(value: unknown, key: string) {
  if (!value || typeof value !== "object") return 0;
  const raw = (value as Record<string, unknown>)[key];
  return typeof raw === "number" ? raw : Number(raw ?? 0);
}
