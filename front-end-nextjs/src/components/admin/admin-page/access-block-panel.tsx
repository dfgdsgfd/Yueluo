"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { AlertTriangle, Ban, ExternalLink, Loader2, Plus, RefreshCw, ShieldX, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { adminRequest } from "@/lib/api";
import { AccessBlockRuleDialog, AccessBlockImportDialog } from "./access-block-dialogs";
import { AccessBlockImportsSection } from "./access-block-imports-section";
import {
  AccessBlockDraft,
  AccessBlockImportDraft,
  AccessBlockImportSource,
  AccessBlockRule,
  accessBlockMatchType,
  actionLabel,
  Badge,
  defaultDraft,
  defaultImportDraft,
  importSourceBody,
} from "./access-block-shared";

export function AccessBlockPanel({ token }: { token: string }) {
  const t = useTranslations("adminPortal.accessBlockPanel");
  const [rules, setRules] = useState<AccessBlockRule[]>([]);
  const [sources, setSources] = useState<AccessBlockImportSource[]>([]);
  const [disabled, setDisabled] = useState(false);
  const [draft, setDraft] = useState<AccessBlockDraft>({ ...defaultDraft });
  const [importDraft, setImportDraft] = useState<AccessBlockImportDraft>({ ...defaultImportDraft });
  const [editing, setEditing] = useState<AccessBlockRule | null>(null);
  const [editingSource, setEditingSource] = useState<AccessBlockImportSource | null>(null);
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [savingSource, setSavingSource] = useState(false);
  const [syncingSourceId, setSyncingSourceId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await adminRequest<{ rules?: AccessBlockRule[]; disabled?: boolean }>("/api/admin/access-block/rules", {
        method: "GET",
        token,
      });
      const importPayload = await adminRequest<{ sources?: AccessBlockImportSource[] }>("/api/admin/access-block/import-sources", {
        method: "GET",
        token,
      });
      setRules(payload.rules ?? []);
      setSources(importPayload.sources ?? []);
      setDisabled(Boolean(payload.disabled));
    } catch (error) {
      toastError(error, t, "toasts.loadFailed");
    } finally {
      setLoading(false);
    }
  }, [t, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  const manualRules = useMemo(() => rules.filter((rule) => !rule.import_source_id), [rules]);
  const activeCount = useMemo(() => manualRules.filter((rule) => rule.enabled).length, [manualRules]);
  const activeSourceCount = useMemo(() => sources.filter((source) => source.enabled).length, [sources]);
  const importedRuleCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const rule of rules) {
      if (!rule.import_source_id) {
        continue;
      }
      const id = String(rule.import_source_id);
      counts[id] = (counts[id] ?? 0) + 1;
    }
    return counts;
  }, [rules]);

  function patchDraft(patch: Partial<AccessBlockDraft>) {
    setDraft((current) => {
      const next = { ...current, ...patch };
      if (patch.kind === "ip" && current.kind !== "ip") next.match_type = "cidr";
      if (patch.kind === "ua" && current.kind !== "ua") next.match_type = "ua_contains";
      return next;
    });
  }

  function newRule() {
    setEditing(null);
    setDraft({ ...defaultDraft });
    setRuleDialogOpen(true);
  }

  function editRule(rule: AccessBlockRule) {
    setEditing(rule);
    setDraft({
      kind: rule.kind === "ua" ? "ua" : "ip",
      match_type: accessBlockMatchType(rule),
      pattern: rule.pattern ?? "",
      batch: "",
      enabled: Boolean(rule.enabled),
      priority: Number(rule.priority ?? 1000),
      action: rule.action === "redirect" ? "redirect" : "status",
      status_code: Number(rule.status_code || 444),
      redirect_url: rule.redirect_url ?? "",
      note: rule.note ?? "",
      force: false,
    });
    setRuleDialogOpen(true);
  }

  function resetForm() {
    setEditing(null);
    setDraft({ ...defaultDraft });
    setRuleDialogOpen(false);
  }

  function patchImportDraft(patch: Partial<AccessBlockImportDraft>) {
    setImportDraft((current) => ({ ...current, ...patch }));
  }

  function newImportSource() {
    setEditingSource(null);
    setImportDraft({ ...defaultImportDraft });
    setImportDialogOpen(true);
  }

  function editSource(source: AccessBlockImportSource) {
    setEditingSource(source);
    setImportDraft({
      url: source.url ?? "",
      enabled: Boolean(source.enabled),
      priority: Number(source.priority ?? 1000),
      action: source.action === "redirect" ? "redirect" : "status",
      status_code: Number(source.status_code || 444),
      redirect_url: source.redirect_url ?? "",
      note: source.note ?? "",
      update_interval_minutes: Math.max(5, Math.round(Number(source.update_interval_seconds || 3600) / 60)),
      force: false,
    });
    setImportDialogOpen(true);
  }

  function resetImportForm() {
    setEditingSource(null);
    setImportDraft({ ...defaultImportDraft });
    setImportDialogOpen(false);
  }

  async function saveRule() {
    const isBatch = draft.batch.trim().length > 0;
    setSaving(true);
    try {
      if (isBatch && !editing) {
        const payload = await adminRequest<{ created?: number; failed?: number }>("/api/admin/access-block/rules/batch", {
          method: "POST",
          token,
          body: JSON.stringify({ ...ruleBody(draft), patterns: draft.batch, force: draft.force }),
        });
        toast.success(t("toasts.batchSaved", { created: payload.created ?? 0, failed: payload.failed ?? 0 }));
      } else if (editing) {
        await adminRequest(`/api/admin/access-block/rules/${encodeURIComponent(String(editing.id))}`, {
          method: "PUT",
          token,
          body: JSON.stringify(ruleBody(draft)),
        });
        toast.success(t("toasts.saved"));
      } else {
        await adminRequest("/api/admin/access-block/rules", {
          method: "POST",
          token,
          body: JSON.stringify(ruleBody(draft)),
        });
        toast.success(t("toasts.saved"));
      }
      resetForm();
      await load();
    } catch (error) {
      toastError(error, t, "toasts.saveFailed");
    } finally {
      setSaving(false);
    }
  }

  async function toggleRule(rule: AccessBlockRule) {
    try {
      await adminRequest(`/api/admin/access-block/rules/${encodeURIComponent(String(rule.id))}`, {
        method: "PUT",
        token,
        body: JSON.stringify({ ...rule, enabled: !rule.enabled, force: true }),
      });
      await load();
    } catch (error) {
      toastError(error, t, "toasts.saveFailed");
    }
  }

  async function deleteRule(rule: AccessBlockRule) {
    if (!window.confirm(t("deleteConfirm"))) return;
    try {
      await adminRequest(`/api/admin/access-block/rules/${encodeURIComponent(String(rule.id))}`, {
        method: "DELETE",
        token,
      });
      toast.success(t("toasts.deleted"));
      await load();
    } catch (error) {
      toastError(error, t, "toasts.deleteFailed");
    }
  }

  async function saveImportSource() {
    setSavingSource(true);
    try {
      const body = importSourceBody(importDraft);
      if (editingSource) {
        await adminRequest(`/api/admin/access-block/import-sources/${encodeURIComponent(String(editingSource.id))}`, {
          method: "PUT",
          token,
          body: JSON.stringify(body),
        });
      } else {
        await adminRequest("/api/admin/access-block/import-sources", {
          method: "POST",
          token,
          body: JSON.stringify(body),
        });
      }
      toast.success(t("toasts.importSaved"));
      resetImportForm();
      await load();
    } catch (error) {
      toastError(error, t, "toasts.importSaveFailed");
    } finally {
      setSavingSource(false);
    }
  }

  async function toggleSource(source: AccessBlockImportSource) {
    try {
      await adminRequest(`/api/admin/access-block/import-sources/${encodeURIComponent(String(source.id))}`, {
        method: "PUT",
        token,
        body: JSON.stringify({ ...source, enabled: !source.enabled }),
      });
      await load();
    } catch (error) {
      toastError(error, t, "toasts.importSaveFailed");
    }
  }

  async function syncSource(source: AccessBlockImportSource) {
    const id = String(source.id);
    setSyncingSourceId(id);
    try {
      const payload = await adminRequest<{ count?: number }>(`/api/admin/access-block/import-sources/${encodeURIComponent(id)}/sync`, {
        method: "POST",
        token,
        body: JSON.stringify({ force: importDraft.force }),
      });
      toast.success(t("toasts.importSynced", { count: payload.count ?? 0 }));
      await load();
    } catch (error) {
      toastError(error, t, "toasts.importSyncFailed");
    } finally {
      setSyncingSourceId(null);
    }
  }

  async function deleteSource(source: AccessBlockImportSource) {
    if (!window.confirm(t("importDeleteConfirm"))) return;
    try {
      await adminRequest(`/api/admin/access-block/import-sources/${encodeURIComponent(String(source.id))}`, {
        method: "DELETE",
        token,
      });
      toast.success(t("toasts.importDeleted"));
      await load();
    } catch (error) {
      toastError(error, t, "toasts.importDeleteFailed");
    }
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
      <section className="rounded-xl border border-[#dc2626]/20 bg-white p-4">
        <div className="flex flex-wrap items-center gap-3">
          <span className="flex size-11 items-center justify-center rounded-xl bg-[#fff0f2] text-[#dc2626]">
            <ShieldX className="size-5" />
          </span>
          <div className="min-w-0 flex-1">
            <h1 className="text-lg font-black text-[#252932]">{t("title")}</h1>
            <p className="mt-1 text-sm text-[#7b808c]">{t("description")}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => void load()} className="h-9 rounded-lg border-black/[0.08] bg-white">
            <RefreshCw className="size-4" />
            {t("refresh")}
          </Button>
          <Button asChild variant="outline" className="h-9 rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c]">
            <Link href="/admin/console?view=logs&category=access_block">
              <ExternalLink className="size-4" />
              {t("viewLogs")}
            </Link>
          </Button>
        </div>
      </section>

      {disabled ? (
        <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <span>{t("disabledHint")}</span>
        </div>
      ) : null}

      <section className="min-w-0 overflow-hidden rounded-xl border border-black/[0.06] bg-white">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-black/[0.06] px-4 py-3">
          <div className="min-w-0">
            <h2 className="text-sm font-black text-[#252932]">{t("rulesTitle")}</h2>
            <p className="mt-1 text-xs text-[#7b808c]">{t("rulesSummary", { active: activeCount, total: manualRules.length })}</p>
          </div>
          <Button type="button" variant="outline" onClick={newRule} className="h-9 rounded-lg border-black/[0.08] bg-white">
            <Plus className="size-4" />
            {t("newRule")}
          </Button>
        </div>
        {manualRules.length === 0 ? (
          <p className="px-4 py-10 text-center text-sm text-[#7b808c]">{t("empty")}</p>
        ) : (
          <div className="grid gap-3 p-3 sm:grid-cols-2 xl:grid-cols-3">
            {manualRules.map((rule) => (
              <AccessBlockRuleCard
                key={String(rule.id)}
                onDeleteRule={(target) => void deleteRule(target)}
                onEditRule={editRule}
                onToggleRule={(target) => void toggleRule(target)}
                rule={rule}
                t={t}
              />
            ))}
          </div>
        )}
      </section>

      <AccessBlockImportsSection
        activeSourceCount={activeSourceCount}
        importedRuleCounts={importedRuleCounts}
        onDeleteSource={(source) => void deleteSource(source)}
        onEditSource={editSource}
        onNewSource={newImportSource}
        onSyncSource={(source) => void syncSource(source)}
        onToggleSource={(source) => void toggleSource(source)}
        sources={sources}
        syncingSourceId={syncingSourceId}
        t={t}
      />

      <AccessBlockRuleDialog
        draft={draft}
        editing={editing}
        onClose={resetForm}
        onPatchDraft={patchDraft}
        onSave={() => void saveRule()}
        open={ruleDialogOpen}
        saving={saving}
        t={t}
      />
      <AccessBlockImportDialog
        draft={importDraft}
        editingSource={editingSource}
        onClose={resetImportForm}
        onPatchDraft={patchImportDraft}
        onSave={() => void saveImportSource()}
        open={importDialogOpen}
        saving={savingSource}
        t={t}
      />
    </div>
  );
}

function AccessBlockRuleCard({
  onDeleteRule,
  onEditRule,
  onToggleRule,
  rule,
  t,
}: {
  onDeleteRule: (rule: AccessBlockRule) => void;
  onEditRule: (rule: AccessBlockRule) => void;
  onToggleRule: (rule: AccessBlockRule) => void;
  rule: AccessBlockRule;
  t: (key: string, values?: Record<string, string | number>) => string;
}) {
  return (
    <article className="grid min-w-0 gap-3 rounded-lg border border-black/[0.06] bg-[#fbfcfe] p-3">
      <button type="button" onClick={() => onEditRule(rule)} className="grid min-w-0 gap-2 text-left">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={rule.kind === "ua" ? "blue" : "red"}>{t(`kinds.${rule.kind === "ua" ? "ua" : "ip"}`)}</Badge>
          <Badge>{t(`matchTypes.${accessBlockMatchType(rule)}`)}</Badge>
          <Badge tone={rule.enabled ? "green" : "slate"}>{rule.enabled ? t("enabled") : t("disabled")}</Badge>
        </div>
        <span className="truncate font-mono text-xs font-semibold text-[#252932]">{rule.pattern}</span>
      </button>
      <div className="grid gap-1.5 text-xs text-[#59606c]">
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("columns.action")}</span>
          <span className="truncate font-semibold">{actionLabel(rule, t)}</span>
        </div>
        <div className="flex min-w-0 justify-between gap-2">
          <span className="text-[#7b808c]">{t("columns.priority")}</span>
          <span className="truncate font-semibold">{rule.priority ?? 1000}</span>
        </div>
        {rule.note ? (
          <div className="flex min-w-0 justify-between gap-2">
            <span className="text-[#7b808c]">{t("columns.note")}</span>
            <span className="truncate font-semibold" title={rule.note}>{rule.note}</span>
          </div>
        ) : null}
      </div>
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" variant="outline" onClick={() => onToggleRule(rule)} className="h-8 rounded-lg border-black/[0.08] bg-white px-3 text-xs">
          <Ban className="size-3.5" />
          {rule.enabled ? t("disable") : t("enable")}
        </Button>
        <Button type="button" variant="outline" onClick={() => onDeleteRule(rule)} className="h-8 rounded-lg border-red-200 bg-white px-3 text-xs text-red-700">
          <Trash2 className="size-3.5" />
          {t("delete")}
        </Button>
      </div>
    </article>
  );
}

function ruleBody(draft: AccessBlockDraft) {
  return {
    kind: draft.kind,
    match_type: draft.match_type,
    pattern: draft.pattern,
    enabled: draft.enabled,
    priority: draft.priority,
    action: draft.action,
    status_code: draft.status_code,
    redirect_url: draft.redirect_url,
    note: draft.note,
    force: draft.force,
  };
}

function toastError(error: unknown, t: (key: string, values?: Record<string, string | number>) => string, fallbackKey: string) {
  const message = error instanceof Error ? error.message : "";
  const knownKey = accessBlockErrorKey(message);
  toast.error(knownKey ? t(knownKey) : message || t(fallbackKey));
}

function accessBlockErrorKey(message: string) {
  switch (message) {
    case "error.access_block_self_lock":
      return "errors.selfLock";
    case "error.access_block_import_disabled":
      return "errors.importDisabled";
    case "error.access_block_import_fetch_failed":
      return "errors.importFetchFailed";
    case "error.access_block_invalid_rule":
      return "errors.invalidRule";
    case "error.access_block_save_failed":
      return "errors.saveFailed";
    case "error.access_block_unavailable":
      return "errors.unavailable";
    default:
      return "";
  }
}
