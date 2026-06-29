"use client";

import { Ban, Loader2, Plus, RefreshCw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  AccessBlockImportSource,
  actionLabel,
  Badge,
  formatDateTime,
  statusLabel,
} from "./access-block-shared";

type Translator = (key: string, values?: Record<string, string | number>) => string;

type Props = {
  activeSourceCount: number;
  importedRuleCounts: Record<string, number>;
  onDeleteSource: (source: AccessBlockImportSource) => void;
  onEditSource: (source: AccessBlockImportSource) => void;
  onNewSource: () => void;
  onSyncSource: (source: AccessBlockImportSource) => void;
  onToggleSource: (source: AccessBlockImportSource) => void;
  sources: AccessBlockImportSource[];
  syncingSourceId: string | null;
  t: Translator;
};

export function AccessBlockImportsSection({
  activeSourceCount,
  importedRuleCounts,
  onDeleteSource,
  onEditSource,
  onNewSource,
  onSyncSource,
  onToggleSource,
  sources,
  syncingSourceId,
  t,
}: Props) {
  return (
    <section className="min-w-0 overflow-hidden rounded-xl border border-black/[0.06] bg-white">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-black/[0.06] px-4 py-3">
        <div className="min-w-0">
          <h2 className="text-sm font-black text-[#252932]">{t("importsTitle")}</h2>
          <p className="mt-1 text-xs text-[#7b808c]">{t("importsSummary", { active: activeSourceCount, total: sources.length })}</p>
        </div>
        <Button type="button" variant="outline" onClick={onNewSource} className="h-9 rounded-lg border-black/[0.08] bg-white">
          <Plus className="size-4" />
          {t("newImport")}
        </Button>
      </div>
      {sources.length === 0 ? (
        <p className="px-4 py-10 text-center text-sm text-[#7b808c]">{t("importsEmpty")}</p>
      ) : (
        <div className="grid gap-3 p-3 sm:grid-cols-2 xl:grid-cols-3">
          {sources.map((source) => (
            <ImportSourceCard
              importedCount={importedRuleCounts[String(source.id)] ?? source.last_count ?? 0}
              key={String(source.id)}
              onDeleteSource={onDeleteSource}
              onEditSource={onEditSource}
              onSyncSource={onSyncSource}
              onToggleSource={onToggleSource}
              source={source}
              syncing={syncingSourceId === String(source.id)}
              t={t}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function ImportSourceCard({
  importedCount,
  onDeleteSource,
  onEditSource,
  onSyncSource,
  onToggleSource,
  source,
  syncing,
  t,
}: {
  importedCount: number;
  onDeleteSource: (source: AccessBlockImportSource) => void;
  onEditSource: (source: AccessBlockImportSource) => void;
  onSyncSource: (source: AccessBlockImportSource) => void;
  onToggleSource: (source: AccessBlockImportSource) => void;
  source: AccessBlockImportSource;
  syncing: boolean;
  t: Translator;
}) {
  return (
    <article className="grid min-w-0 gap-3 rounded-lg border border-black/[0.06] bg-[#fbfcfe] p-3">
      <button type="button" onClick={() => onEditSource(source)} className="grid min-w-0 gap-2 text-left">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={source.last_status === "failed" ? "red" : source.last_status === "success" ? "green" : "slate"}>{statusLabel(source.last_status, t)}</Badge>
          <Badge tone={source.enabled ? "green" : "slate"}>{source.enabled ? t("enabled") : t("disabled")}</Badge>
          <span className="text-xs font-semibold text-[#7b808c]">{t("importCount", { count: importedCount })}</span>
        </div>
        <span className="truncate font-mono text-xs font-semibold text-[#252932]">{source.url}</span>
        {source.last_error ? (
          <span className="truncate text-xs text-red-600" title={source.last_error}>{source.last_error}</span>
        ) : null}
      </button>
      <div className="grid gap-1.5 text-xs text-[#59606c]">
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("importColumns.interval")}</span>
          <span className="truncate font-semibold">{t("intervalMinutes", { minutes: Math.round(Number(source.update_interval_seconds || 3600) / 60) })}</span>
        </div>
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("columns.action")}</span>
          <span className="truncate font-semibold">{actionLabel(source, t)}</span>
        </div>
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("importColumns.lastSync")}</span>
          <span className="truncate font-semibold">{formatDateTime(source.last_sync_at)}</span>
        </div>
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("importColumns.nextSync")}</span>
          <span className="truncate font-semibold">{formatDateTime(source.next_sync_at)}</span>
        </div>
      </div>
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" variant="outline" disabled={syncing} onClick={() => onSyncSource(source)} className="h-8 rounded-lg border-black/[0.08] bg-white px-3 text-xs">
          {syncing ? <Loader2 className="size-3.5 animate-spin" /> : <RefreshCw className="size-3.5" />}
          {t("syncNow")}
        </Button>
        <Button type="button" variant="outline" onClick={() => onToggleSource(source)} className="h-8 rounded-lg border-black/[0.08] bg-white px-3 text-xs">
          <Ban className="size-3.5" />
          {source.enabled ? t("disable") : t("enable")}
        </Button>
        <Button type="button" variant="outline" onClick={() => onDeleteSource(source)} className="h-8 rounded-lg border-red-200 bg-white px-3 text-xs text-red-700">
          <Trash2 className="size-3.5" />
          {t("delete")}
        </Button>
      </div>
    </article>
  );
}
