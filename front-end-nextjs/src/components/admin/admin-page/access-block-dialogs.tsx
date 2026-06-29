"use client";

import type { ReactNode } from "react";
import { Loader2, Save, UploadCloud, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type {
  AccessBlockDraft,
  AccessBlockImportDraft,
  AccessBlockImportSource,
  AccessBlockRule,
} from "./access-block-shared";
import { InputField, SelectField } from "./access-block-shared";

type Translator = (key: string, values?: Record<string, string | number>) => string;

type DialogShellProps = {
  children: ReactNode;
  description: string;
  onClose: () => void;
  open: boolean;
  t: Translator;
  title: string;
};

export function AccessBlockRuleDialog({
  draft,
  editing,
  onClose,
  onPatchDraft,
  onSave,
  open,
  saving,
  t,
}: {
  draft: AccessBlockDraft;
  editing: AccessBlockRule | null;
  onClose: () => void;
  onPatchDraft: (patch: Partial<AccessBlockDraft>) => void;
  onSave: () => void;
  open: boolean;
  saving: boolean;
  t: Translator;
}) {
  const isBatch = draft.batch.trim().length > 0;

  return (
    <AccessBlockDialogShell
      description={t("formHint")}
      onClose={onClose}
      open={open}
      t={t}
      title={editing ? t("editTitle") : t("createTitle")}
    >
      <div className="grid gap-3">
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <SelectField label={t("kind")} value={draft.kind} onChange={(value) => onPatchDraft({ kind: value as AccessBlockDraft["kind"] })}>
            <option value="ip">{t("kinds.ip")}</option>
            <option value="ua">{t("kinds.ua")}</option>
          </SelectField>
          <SelectField label={t("matchType")} value={draft.match_type} onChange={(value) => onPatchDraft({ match_type: value as AccessBlockDraft["match_type"] })}>
            {draft.kind === "ip" ? (
              <>
                <option value="ip">{t("matchTypes.ip")}</option>
                <option value="cidr">{t("matchTypes.cidr")}</option>
              </>
            ) : (
              <>
                <option value="ua_contains">{t("matchTypes.ua_contains")}</option>
                <option value="ua_regex">{t("matchTypes.ua_regex")}</option>
              </>
            )}
          </SelectField>
        </div>
        <InputField label={t("pattern")} value={draft.pattern} onChange={(value) => onPatchDraft({ pattern: value })} />
        {!editing ? (
          <label className="grid gap-1.5">
            <span className="text-xs font-bold text-[#5f636d]">{t("batch")}</span>
            <textarea
              value={draft.batch}
              onChange={(event) => onPatchDraft({ batch: event.target.value })}
              placeholder={t("batchPlaceholder")}
              className="min-h-28 rounded-lg border border-black/[0.08] bg-white px-3 py-2 font-mono text-xs outline-none focus:border-[#dc2626]"
            />
          </label>
        ) : null}
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <SelectField label={t("action")} value={draft.action} onChange={(value) => onPatchDraft({ action: value as AccessBlockDraft["action"] })}>
            <option value="status">{t("actions.status")}</option>
            <option value="redirect">{t("actions.redirect")}</option>
          </SelectField>
          {draft.action === "status" ? (
            <SelectField label={t("statusCode")} value={String(draft.status_code)} onChange={(value) => onPatchDraft({ status_code: Number(value) })}>
              {[444, 403, 404, 410, 429].map((code) => <option key={code} value={code}>{code}</option>)}
            </SelectField>
          ) : (
            <InputField label={t("redirectUrl")} value={draft.redirect_url} onChange={(value) => onPatchDraft({ redirect_url: value })} />
          )}
        </div>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <InputField type="number" label={t("priority")} value={String(draft.priority)} onChange={(value) => onPatchDraft({ priority: Number(value) || 1000 })} />
          <InputField label={t("note")} value={draft.note} onChange={(value) => onPatchDraft({ note: value })} />
        </div>
        <RuleSwitch checked={draft.enabled} label={t("enabled")} onChange={(enabled) => onPatchDraft({ enabled })} />
        <ForceSwitch checked={draft.force} label={t("force")} onChange={(force) => onPatchDraft({ force })} />
        <DialogActions
          loading={saving}
          onClose={onClose}
          onSave={onSave}
          saveIcon={<Save className="size-4" />}
          saveLabel={isBatch && !editing ? t("batchSave") : t("save")}
          t={t}
        />
      </div>
    </AccessBlockDialogShell>
  );
}

export function AccessBlockImportDialog({
  draft,
  editingSource,
  onClose,
  onPatchDraft,
  onSave,
  open,
  saving,
  t,
}: {
  draft: AccessBlockImportDraft;
  editingSource: AccessBlockImportSource | null;
  onClose: () => void;
  onPatchDraft: (patch: Partial<AccessBlockImportDraft>) => void;
  onSave: () => void;
  open: boolean;
  saving: boolean;
  t: Translator;
}) {
  return (
    <AccessBlockDialogShell
      description={t("importFormHint")}
      onClose={onClose}
      open={open}
      t={t}
      title={editingSource ? t("editImportTitle") : t("createImportTitle")}
    >
      <div className="grid gap-3">
        <InputField label={t("importUrl")} value={draft.url} onChange={(value) => onPatchDraft({ url: value })} />
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <InputField type="number" label={t("updateIntervalMinutes")} value={String(draft.update_interval_minutes)} onChange={(value) => onPatchDraft({ update_interval_minutes: Math.max(5, Number(value) || 60) })} />
          <InputField type="number" label={t("priority")} value={String(draft.priority)} onChange={(value) => onPatchDraft({ priority: Number(value) || 1000 })} />
        </div>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <SelectField label={t("action")} value={draft.action} onChange={(value) => onPatchDraft({ action: value as AccessBlockImportDraft["action"] })}>
            <option value="status">{t("actions.status")}</option>
            <option value="redirect">{t("actions.redirect")}</option>
          </SelectField>
          {draft.action === "status" ? (
            <SelectField label={t("statusCode")} value={String(draft.status_code)} onChange={(value) => onPatchDraft({ status_code: Number(value) })}>
              {[444, 403, 404, 410, 429].map((code) => <option key={code} value={code}>{code}</option>)}
            </SelectField>
          ) : (
            <InputField label={t("redirectUrl")} value={draft.redirect_url} onChange={(value) => onPatchDraft({ redirect_url: value })} />
          )}
        </div>
        <InputField label={t("note")} value={draft.note} onChange={(value) => onPatchDraft({ note: value })} />
        <RuleSwitch checked={draft.enabled} label={t("enabled")} onChange={(enabled) => onPatchDraft({ enabled })} />
        <ForceSwitch checked={draft.force} label={t("importForce")} onChange={(force) => onPatchDraft({ force })} />
        <DialogActions
          loading={saving}
          onClose={onClose}
          onSave={onSave}
          saveIcon={<UploadCloud className="size-4" />}
          saveLabel={t("saveImport")}
          t={t}
        />
      </div>
    </AccessBlockDialogShell>
  );
}

function AccessBlockDialogShell({
  children,
  description,
  onClose,
  open,
  t,
  title,
}: DialogShellProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/45 px-2 py-3 sm:items-center sm:px-4">
      <button type="button" aria-label={t("cancel")} className="absolute inset-0 cursor-default" onClick={onClose} />
      <section
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="relative flex max-h-[min(92dvh,46rem)] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-black/[0.08] bg-white shadow-2xl"
      >
        <header className="flex shrink-0 items-start gap-3 border-b border-black/[0.06] px-4 py-3">
          <div className="min-w-0 flex-1">
            <h2 className="text-sm font-black text-[#252932]">{title}</h2>
            <p className="mt-1 text-xs text-[#7b808c]">{description}</p>
          </div>
          <Button type="button" variant="ghost" size="icon" aria-label={t("cancel")} onClick={onClose} className="size-8 rounded-lg">
            <X className="size-4" />
          </Button>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          {children}
        </div>
      </section>
    </div>
  );
}

function DialogActions({
  loading,
  onClose,
  onSave,
  saveIcon,
  saveLabel,
  t,
}: {
  loading: boolean;
  onClose: () => void;
  onSave: () => void;
  saveIcon: ReactNode;
  saveLabel: string;
  t: Translator;
}) {
  return (
    <div className="flex flex-wrap justify-end gap-2 pt-1">
      <Button type="button" variant="outline" onClick={onClose} className="h-10 rounded-lg border-black/[0.08] bg-white">
        {t("cancel")}
      </Button>
      <Button type="button" disabled={loading} onClick={onSave} className="h-10 rounded-lg bg-[#dc2626] px-4 hover:bg-[#b91c1c]">
        {loading ? <Loader2 className="size-4 animate-spin" /> : saveIcon}
        {saveLabel}
      </Button>
    </div>
  );
}

function RuleSwitch({ checked, label, onChange }: { checked: boolean; label: string; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-lg border border-black/[0.06] bg-[#f8fafc] px-3 py-2 text-sm font-semibold text-[#343944]">
      <span>{label}</span>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} className="size-4 accent-[#dc2626]" />
    </label>
  );
}

function ForceSwitch({ checked, label, onChange }: { checked: boolean; label: string; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} className="mt-0.5 size-4 accent-[#dc2626]" />
      <span>{label}</span>
    </label>
  );
}
