"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Copy, Loader2, RefreshCw, Save, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { adminRequest } from "@/lib/api";

type MaintenancePayload = {
  enabled?: boolean;
  entry_code?: string;
  entry_url?: string;
  auto_login_uid?: number;
  started_at?: string;
  estimated_end_at?: string;
  message?: string;
  border_visible?: boolean;
  border_color?: string;
  border_opacity?: number;
  border_dismissible?: boolean;
};

export function MaintenancePanel({ token }: { token: string }) {
  const t = useTranslations("adminMaintenance");
  const [draft, setDraft] = useState<MaintenancePayload>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setDraft(await adminRequest<MaintenancePayload>("/api/admin/maintenance", { method: "GET", token }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [t, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  async function save() {
    setSaving(true);
    try {
      const payload = await adminRequest<MaintenancePayload>("/api/admin/maintenance", {
        method: "PUT",
        token,
        body: JSON.stringify({
          enabled: Boolean(draft.enabled),
          entry_code: draft.entry_code,
          auto_login_uid: Number(draft.auto_login_uid ?? 0),
          started_at: draft.started_at ?? "",
          estimated_end_at: draft.estimated_end_at ?? "",
          message: draft.message ?? "",
          border_visible: draft.border_visible ?? true,
          border_color: draft.border_color ?? "#dc2626",
          border_opacity: Number(draft.border_opacity ?? 1),
          border_dismissible: Boolean(draft.border_dismissible),
        }),
      });
      setDraft(payload);
      toast.success(t("saved"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("saveFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function rotateEntry() {
    setSaving(true);
    try {
      setDraft(await adminRequest<MaintenancePayload>("/api/admin/maintenance/rotate-entry", { method: "POST", token }));
      toast.success(t("rotated"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("rotateFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function copyEntry() {
    const path = draft.entry_url || (draft.entry_code ? `/service-mode/${draft.entry_code}` : "");
    if (!path) return;
    const url = new URL(path, window.location.origin).toString();
    await navigator.clipboard.writeText(url);
    toast.success(t("copied"));
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
            <ShieldAlert className="size-5" />
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

      <section className="grid gap-4 rounded-xl border border-black/[0.06] bg-white p-4 shadow-[0_10px_30px_rgba(17,24,39,0.04)] lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="grid min-w-0 gap-4">
          <label className="flex items-center justify-between gap-4 rounded-xl border border-black/[0.06] bg-[#f8fafc] px-4 py-3">
            <span>
              <span className="block text-sm font-bold text-[#252932]">{t("enabled")}</span>
              <span className="mt-1 block text-xs text-[#7b808c]">{t("enabledHint")}</span>
            </span>
            <input type="checkbox" checked={Boolean(draft.enabled)} onChange={(event) => setDraft((value) => ({ ...value, enabled: event.target.checked }))} className="size-5 accent-[#dc2626]" />
          </label>

          <label className="grid gap-1.5">
            <span className="text-xs font-bold text-[#5f636d]">{t("message")}</span>
            <textarea value={draft.message ?? ""} onChange={(event) => setDraft((value) => ({ ...value, message: event.target.value }))} className="min-h-24 rounded-lg border border-black/[0.08] bg-white px-3 py-2 text-sm outline-none focus:border-[#dc2626]" />
          </label>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="grid gap-1.5">
              <span className="text-xs font-bold text-[#5f636d]">{t("autoLoginUid")}</span>
              <input type="number" min={0} value={draft.auto_login_uid ?? 0} onChange={(event) => setDraft((value) => ({ ...value, auto_login_uid: Number(event.target.value) }))} className="h-10 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#dc2626]" />
            </label>
            <label className="grid gap-1.5">
              <span className="text-xs font-bold text-[#5f636d]">{t("estimatedEnd")}</span>
              <input type="datetime-local" value={toDatetimeLocal(draft.estimated_end_at)} onChange={(event) => setDraft((value) => ({ ...value, estimated_end_at: fromDatetimeLocal(event.target.value) }))} className="h-10 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#dc2626]" />
            </label>
          </div>

          <div className="grid gap-3 rounded-xl border border-black/[0.06] bg-[#f8fafc] p-4">
            <div className="flex min-w-0 flex-wrap items-center gap-3">
              <div className="min-w-0 flex-1">
                <p className="text-sm font-bold text-[#252932]">{t("borderSettings")}</p>
                <p className="mt-1 text-xs text-[#7b808c]">{t("borderSettingsHint")}</p>
              </div>
              <label className="inline-flex items-center gap-2 text-sm font-semibold text-[#343944]">
                <input type="checkbox" checked={draft.border_visible ?? true} onChange={(event) => setDraft((value) => ({ ...value, border_visible: event.target.checked }))} className="size-4 accent-[#dc2626]" />
                {t("borderVisible")}
              </label>
              <label className="inline-flex items-center gap-2 text-sm font-semibold text-[#343944]">
                <input type="checkbox" checked={Boolean(draft.border_dismissible)} onChange={(event) => setDraft((value) => ({ ...value, border_dismissible: event.target.checked }))} className="size-4 accent-[#dc2626]" />
                {t("borderDismissible")}
              </label>
            </div>
            <div className="grid gap-3 sm:grid-cols-[180px_minmax(0,1fr)]">
              <label className="grid gap-1.5">
                <span className="text-xs font-bold text-[#5f636d]">{t("borderColor")}</span>
                <div className="flex h-10 items-center gap-2 rounded-lg border border-black/[0.08] bg-white px-2">
                  <input type="color" value={draft.border_color || "#dc2626"} onChange={(event) => setDraft((value) => ({ ...value, border_color: event.target.value }))} className="size-7 rounded border-0 bg-transparent p-0" />
                  <input value={draft.border_color || "#dc2626"} onChange={(event) => setDraft((value) => ({ ...value, border_color: event.target.value }))} className="min-w-0 flex-1 text-sm outline-none" />
                </div>
              </label>
              <label className="grid gap-1.5">
                <span className="text-xs font-bold text-[#5f636d]">{t("borderOpacity", { value: Math.round(Number(draft.border_opacity ?? 1) * 100) })}</span>
                <input type="range" min={0} max={1} step={0.05} value={draft.border_opacity ?? 1} onChange={(event) => setDraft((value) => ({ ...value, border_opacity: Number(event.target.value) }))} className="h-10 accent-[#dc2626]" />
              </label>
            </div>
          </div>
        </div>

        <aside className="min-w-0 rounded-xl border border-[#fee2e2] bg-[#fff7f7] p-4">
          <p className="text-xs font-black uppercase tracking-[0.16em] text-[#dc2626]">{t("serviceEntry")}</p>
          <p className="mt-2 break-all text-sm font-semibold text-[#252932]">{draft.entry_url || "-"}</p>
          <div className="mt-4 grid gap-2">
            <Button type="button" variant="outline" onClick={() => void copyEntry()} className="h-10 rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c]">
              <Copy className="size-4" />
              {t("copyEntry")}
            </Button>
            <Button type="button" variant="outline" disabled={saving} onClick={() => void rotateEntry()} className="h-10 rounded-lg border-black/[0.08] bg-white">
              {saving ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
              {t("rotateEntry")}
            </Button>
          </div>
        </aside>
      </section>

      <div className="flex justify-end">
        <Button type="button" disabled={saving} onClick={() => void save()} className="h-10 rounded-lg bg-[#dc2626] px-4 hover:bg-[#b91c1c]">
          {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          {t("save")}
        </Button>
      </div>
    </div>
  );
}

function toDatetimeLocal(value: string | undefined) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function fromDatetimeLocal(value: string) {
  if (!value) return "";
  return new Date(value).toISOString();
}
