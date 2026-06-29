"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { AlertTriangle, CheckCircle2, Database, Loader2, RefreshCw, Wrench } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { adminRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

type DatabaseOverview = Record<string, unknown> & {
  configured?: boolean;
  driver?: string;
  database?: string;
  version?: string;
  total_bytes?: number;
  table_count?: number;
};

type TableItem = {
  name: string;
  rows?: number;
  table_bytes?: number;
  index_bytes?: number;
  total_bytes?: number;
};

type ColumnItem = {
  name: string;
  data_type?: string;
  nullable?: boolean;
  avg_width?: number;
  estimated_bytes?: number;
};

type AuditIssue = {
  kind?: string;
  table?: string;
  name?: string;
  columns?: string[];
  message?: string;
  repair?: string;
  repairable?: boolean;
  duplicateGroupCount?: number;
  duplicateRowCount?: number;
  duplicateSamples?: Array<{ values?: Record<string, unknown>; count?: number }>;
};

type VacuumConfig = {
  enabled?: boolean;
  tables?: string[];
  interval_hours?: number;
  next_run_at?: string;
  last_run_at?: string;
  last_result?: Record<string, unknown> | null;
};

type VacuumPayload = {
  configured?: boolean;
  supported?: boolean;
  available_tables?: string[];
  config?: VacuumConfig;
  message?: string;
};

export function DatabasePanel({ token }: { token: string }) {
  const t = useTranslations("adminDatabase");
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [tables, setTables] = useState<TableItem[]>([]);
  const [issues, setIssues] = useState<AuditIssue[]>([]);
  const [columns, setColumns] = useState<ColumnItem[]>([]);
  const [vacuum, setVacuum] = useState<VacuumPayload | null>(null);
  const [vacuumDraft, setVacuumDraft] = useState<VacuumConfig>({});
  const [selectedTable, setSelectedTable] = useState("");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [repairing, setRepairing] = useState(false);
  const [savingVacuum, setSavingVacuum] = useState(false);
  const [runningVacuum, setRunningVacuum] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [nextOverview, tablePayload, auditPayload, vacuumPayload] = await Promise.all([
        adminRequest<DatabaseOverview>("/api/admin/database/overview", { method: "GET", token }),
        adminRequest<{ items?: TableItem[] }>("/api/admin/database/tables", { method: "GET", token, query: { keyword } }),
        adminRequest<{ issues?: AuditIssue[] }>("/api/admin/database/index-audit", { method: "GET", token }),
        adminRequest<VacuumPayload>("/api/admin/database/vacuum-config", { method: "GET", token }),
      ]);
      const nextTables = tablePayload.items ?? [];
      setOverview(nextOverview);
      setTables(nextTables);
      setIssues(auditPayload.issues ?? []);
      setVacuum(vacuumPayload);
      setVacuumDraft(vacuumPayload.config ?? {});
      setSelectedTable((current) => nextTables.some((table) => table.name === current) ? current : nextTables[0]?.name || "");
      if (!nextTables.length) setColumns([]);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [keyword, t, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  useEffect(() => {
    if (!selectedTable) return;
    let cancelled = false;
    queueMicrotask(async () => {
      try {
        const payload = await adminRequest<{ items?: ColumnItem[] }>(`/api/admin/database/tables/${encodeURIComponent(selectedTable)}/columns`, { method: "GET", token });
        if (!cancelled) setColumns(payload.items ?? []);
      } catch {
        if (!cancelled) setColumns([]);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [selectedTable, token]);

  async function repair() {
    if (!window.confirm(t("repairConfirm"))) return;
    setRepairing(true);
    try {
      const payload = await adminRequest<{ issues?: AuditIssue[] }>("/api/admin/database/repair", { method: "POST", token });
      setIssues(payload.issues ?? []);
      const blockers = (payload.issues ?? []).filter((issue) => issue.repairable === false);
      if (blockers.length) {
        toast.warning(t("repairBlocked", { count: blockers.length }));
      } else {
        toast.success(t("repairDone"));
      }
      await load();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("repairFailed"));
    } finally {
      setRepairing(false);
    }
  }

  async function saveVacuumConfig() {
    setSavingVacuum(true);
    try {
      const payload = await adminRequest<VacuumPayload>("/api/admin/database/vacuum-config", {
        method: "PUT",
        token,
        body: JSON.stringify({
          enabled: Boolean(vacuumDraft.enabled),
          tables: vacuumDraft.tables ?? [],
          interval_hours: Number(vacuumDraft.interval_hours ?? 24),
          next_run_at: vacuumDraft.next_run_at ?? "",
        }),
      });
      setVacuum(payload);
      setVacuumDraft(payload.config ?? {});
      toast.success(t("vacuumSaved"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("vacuumSaveFailed"));
    } finally {
      setSavingVacuum(false);
    }
  }

  async function runVacuum(table: string) {
    if (!table || !window.confirm(t("vacuumRunConfirm", { table }))) return;
    setRunningVacuum(true);
    try {
      const payload = await adminRequest<{ result?: Record<string, unknown>; config?: VacuumConfig }>("/api/admin/database/vacuum-analyze", {
        method: "POST",
        token,
        body: JSON.stringify({ table }),
      });
      setVacuum((current) => current ? { ...current, config: payload.config ?? current.config } : current);
      setVacuumDraft(payload.config ?? vacuumDraft);
      toast.success(t("vacuumRunDone"));
      await load();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("vacuumRunFailed"));
    } finally {
      setRunningVacuum(false);
    }
  }

  function toggleVacuumTable(table: string) {
    setVacuumDraft((value) => {
      const selected = new Set(value.tables ?? []);
      if (selected.has(table)) selected.delete(table);
      else selected.add(table);
      return { ...value, tables: Array.from(selected) };
    });
  }

  if (loading) {
    return (
      <div className="flex h-40 items-center justify-center rounded-xl border border-dashed border-black/[0.08] bg-white text-sm text-[#7b808c]">
        <Loader2 className="mr-2 size-4 animate-spin" />
        {t("loading")}
      </div>
    );
  }

  return (
    <div className="grid min-w-0 gap-4">
      <section className="rounded-xl border border-black/[0.06] bg-white p-4">
        <div className="flex flex-wrap items-center gap-3">
          <span className="flex size-11 items-center justify-center rounded-xl bg-[#eef6ff] text-[#2f7df6]">
            <Database className="size-5" />
          </span>
          <div className="min-w-0 flex-1">
            <h1 className="text-lg font-black text-[#252932]">{t("title")}</h1>
            <p className="mt-1 text-sm text-[#7b808c]">{t("description")}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => void load()} className="h-9 rounded-lg border-black/[0.08] bg-white">
            <RefreshCw className="size-4" />
            {t("refresh")}
          </Button>
        </div>
      </section>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label={t("driver")} value={String(overview?.driver ?? "-")} />
        <Metric label={t("database")} value={String(overview?.database ?? "-")} />
        <Metric label={t("tables")} value={formatCompact(overview?.table_count)} />
        <Metric label={t("size")} value={formatBytes(overview?.total_bytes)} />
      </section>

      <section className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
        <div className="min-w-0 rounded-xl border border-black/[0.06] bg-white p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <h2 className="min-w-0 flex-1 text-base font-bold text-[#252932]">{t("tableSizes")}</h2>
            <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder={t("searchTables")} className="h-9 w-full rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#2f7df6] sm:w-64" />
          </div>
          <div className="grid gap-2">
            {tables.map((table) => (
              <button key={table.name} type="button" onClick={() => setSelectedTable(table.name)} className={cn("grid min-w-0 gap-2 rounded-lg border px-3 py-2 text-left sm:grid-cols-[minmax(0,1fr)_120px_120px_120px]", selectedTable === table.name ? "border-[#2f7df6]/30 bg-[#eef6ff]" : "border-black/[0.06] bg-[#f8fafc] hover:bg-white")}>
                <span className="min-w-0 truncate text-sm font-bold text-[#252932]">{table.name}</span>
                <span className="text-xs text-[#7b808c]">{formatCompact(table.rows)} {t("rows")}</span>
                <span className="text-xs text-[#7b808c]">{formatBytes(table.table_bytes)}</span>
                <span className="text-xs font-semibold text-[#252932]">{formatBytes(table.total_bytes)}</span>
              </button>
            ))}
          </div>
        </div>

        <aside className="min-w-0 rounded-xl border border-black/[0.06] bg-white p-4">
          <h2 className="text-base font-bold text-[#252932]">{selectedTable ? t("columnsFor", { table: selectedTable }) : t("columns")}</h2>
          <div className="mt-3 grid gap-2">
            {columns.map((column) => (
              <div key={column.name} className="rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-sm font-bold text-[#252932]">{column.name}</span>
                  <span className="rounded-full bg-white px-2 py-1 text-xs text-[#6b7280]">{column.nullable ? t("nullable") : t("notNullable")}</span>
                </div>
                <p className="mt-1 truncate text-xs text-[#7b808c]">{column.data_type}</p>
                <p className="mt-1 text-xs text-[#7b808c]">{t("estimatedColumnSize", { size: formatBytes(column.estimated_bytes) })}</p>
              </div>
            ))}
          </div>
        </aside>
      </section>

      <section className="grid min-w-0 gap-4 rounded-xl border border-black/[0.06] bg-white p-4 shadow-[0_10px_30px_rgba(17,24,39,0.04)] xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0">
          <div className="mb-3 flex flex-wrap items-center gap-3">
            <div className="min-w-0 flex-1">
              <h2 className="text-base font-bold text-[#252932]">{t("vacuumTitle")}</h2>
              <p className="mt-1 text-sm text-[#7b808c]">{t("vacuumDescription")}</p>
            </div>
            <Button type="button" variant="outline" disabled={runningVacuum || !selectedTable || vacuum?.supported === false} onClick={() => void runVacuum(selectedTable)} className="h-10 rounded-lg border-[#2f7df6]/20 bg-[#eef6ff] text-[#1d4ed8]">
              {runningVacuum ? <Loader2 className="size-4 animate-spin" /> : <Wrench className="size-4" />}
              {t("vacuumRunSelected")}
            </Button>
          </div>
          {vacuum?.supported === false ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">{vacuum.message ?? t("vacuumUnsupported")}</div>
          ) : (
            <div className="grid gap-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="flex items-center justify-between gap-3 rounded-lg border border-black/[0.06] bg-[#f8fafc] px-3 py-2">
                  <span className="text-sm font-semibold text-[#252932]">{t("vacuumEnabled")}</span>
                  <input type="checkbox" checked={Boolean(vacuumDraft.enabled)} onChange={(event) => setVacuumDraft((value) => ({ ...value, enabled: event.target.checked }))} className="size-5 accent-[#2f7df6]" />
                </label>
                <label className="grid gap-1.5">
                  <span className="text-xs font-bold text-[#5f636d]">{t("vacuumInterval")}</span>
                  <input type="number" min={1} max={720} value={vacuumDraft.interval_hours ?? 24} onChange={(event) => setVacuumDraft((value) => ({ ...value, interval_hours: Number(event.target.value) }))} className="h-10 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#2f7df6]" />
                </label>
              </div>
              <div className="grid max-h-64 gap-2 overflow-y-auto rounded-lg border border-black/[0.06] bg-[#f8fafc] p-3 sm:grid-cols-2 xl:grid-cols-3">
                {(vacuum?.available_tables ?? tables.map((table) => table.name)).map((table) => (
                  <label key={table} className="flex min-w-0 items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm">
                    <input type="checkbox" checked={Boolean(vacuumDraft.tables?.includes(table))} onChange={() => toggleVacuumTable(table)} className="size-4 accent-[#2f7df6]" />
                    <span className="min-w-0 truncate font-semibold text-[#252932]">{table}</span>
                  </label>
                ))}
              </div>
              <div className="flex justify-end">
                <Button type="button" disabled={savingVacuum} onClick={() => void saveVacuumConfig()} className="h-10 rounded-lg bg-[#2f7df6] px-4 hover:bg-[#1d4ed8]">
                  {savingVacuum ? <Loader2 className="size-4 animate-spin" /> : <Wrench className="size-4" />}
                  {t("vacuumSave")}
                </Button>
              </div>
            </div>
          )}
        </div>
        <aside className="min-w-0 rounded-xl border border-black/[0.06] bg-[#f8fafc] p-4">
          <p className="text-xs font-black uppercase tracking-[0.14em] text-[#2f7df6]">{t("vacuumLastResult")}</p>
          <div className="mt-3 grid gap-2">
            <Metric label={t("vacuumNextRun")} value={formatDate(vacuumDraft.next_run_at)} />
            <Metric label={t("vacuumLastRun")} value={formatDate(vacuumDraft.last_run_at)} />
            <Metric label={t("vacuumLastDuration")} value={formatMs(recordValue(vacuumDraft.last_result, "duration_ms"))} />
          </div>
          <VacuumResult result={vacuumDraft.last_result} />
        </aside>
      </section>

      <section className="rounded-xl border border-black/[0.06] bg-white p-4">
        <div className="mb-3 flex flex-wrap items-center gap-3">
          <div className="min-w-0 flex-1">
            <h2 className="text-base font-bold text-[#252932]">{t("indexAudit")}</h2>
            <p className="mt-1 text-sm text-[#7b808c]">{t("indexAuditDescription")}</p>
          </div>
          <Button type="button" disabled={repairing} onClick={() => void repair()} className="h-10 rounded-lg bg-[#2f7df6] px-4 hover:bg-[#1d4ed8]">
            {repairing ? <Loader2 className="size-4 animate-spin" /> : <Wrench className="size-4" />}
            {t("repair")}
          </Button>
        </div>
        {issues.length ? (
          <div className="grid gap-2">
            {issues.map((issue, index) => (
              <div key={`${issue.kind}-${issue.table}-${issue.name}-${index}`} className="rounded-lg border border-[#f59e0b]/20 bg-[#fff7e8] p-3">
                <div className="flex items-center gap-2 text-sm font-bold text-[#92400e]">
                  <AlertTriangle className="size-4" />
                  {issue.kind} · {issue.table} {issue.name ? `/ ${issue.name}` : ""}
                </div>
                <p className="mt-1 text-xs text-[#7b808c]">{issue.message}</p>
                {issue.duplicateGroupCount ? (
                  <p className="mt-2 text-xs font-semibold text-[#92400e]">
                    {t("duplicateSummary", {
                      groups: issue.duplicateGroupCount,
                      rows: issue.duplicateRowCount ?? 0,
                    })}
                  </p>
                ) : null}
                {issue.duplicateSamples?.length ? (
                  <pre className="mt-2 overflow-x-auto rounded-md bg-white/70 p-2 text-[11px] text-[#7c2d12]">
                    {JSON.stringify(issue.duplicateSamples, null, 2)}
                  </pre>
                ) : null}
              </div>
            ))}
          </div>
        ) : (
          <div className="flex min-h-24 items-center justify-center rounded-xl border border-dashed border-[#18a058]/30 bg-[#f0fff6] text-sm font-semibold text-[#16824a]">
            <CheckCircle2 className="mr-2 size-4" />
            {t("noIssues")}
          </div>
        )}
      </section>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-xl border border-black/[0.06] bg-white p-4">
      <p className="truncate text-xs font-semibold text-[#7b808c]">{label}</p>
      <p className="mt-2 truncate text-lg font-black text-[#252932]">{value}</p>
    </div>
  );
}

function VacuumResult({ result }: { result?: Record<string, unknown> | null }) {
  const t = useTranslations("adminDatabase");
  const tables = Array.isArray(result?.tables) ? result.tables.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === "object" && !Array.isArray(item)) : [];
  if (!tables.length) {
    return <p className="mt-3 rounded-lg border border-dashed border-black/[0.08] bg-white px-3 py-2 text-xs text-[#7b808c]">{t("vacuumNoResult")}</p>;
  }
  return (
    <div className="mt-3 grid gap-2">
      {tables.map((item, index) => (
        <div key={`${item.table ?? index}`} className="rounded-lg border border-black/[0.06] bg-white p-3">
          <div className="flex min-w-0 items-center gap-2">
            <span className="min-w-0 flex-1 truncate text-sm font-bold text-[#252932]">{String(item.table ?? "-")}</span>
            <span className="rounded-full bg-[#eef6ff] px-2 py-1 text-xs font-semibold text-[#1d4ed8]">{String(item.status ?? "-")}</span>
          </div>
          <p className="mt-1 text-xs text-[#7b808c]">{formatMs(item.duration_ms)} · {formatDate(item.finished_at)}</p>
          {item.message ? <p className="mt-1 break-all text-xs text-[#dc2626]">{String(item.message)}</p> : null}
        </div>
      ))}
    </div>
  );
}

function formatBytes(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  if (numeric < 1024) return `${numeric.toFixed(0)} B`;
  if (numeric < 1024 * 1024) return `${(numeric / 1024).toFixed(1)} KB`;
  if (numeric < 1024 * 1024 * 1024) return `${(numeric / 1024 / 1024).toFixed(1)} MB`;
  return `${(numeric / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

function formatCompact(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) return "-";
  return numeric.toLocaleString("zh-CN");
}

function formatMs(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return "-";
  return `${numeric.toFixed(numeric >= 10 ? 0 : 1)} ms`;
}

function formatDate(value: unknown) {
  if (!value || typeof value !== "string") return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(date);
}

function recordValue(value: unknown, key: string) {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>)[key] : undefined;
}
