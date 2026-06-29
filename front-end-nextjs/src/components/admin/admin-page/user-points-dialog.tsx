"use client";

import { useMemo, useState, type FormEvent } from "react";
import { Coins, Loader2, Minus, PencilLine, Plus, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { updateAdminUserPoints } from "@/lib/api";
import type { AdminListRow } from "@/lib/types";
import { cn } from "@/lib/utils";
import { errorMessage, formatMoney } from "./helpers";

type PointsOperation = "add" | "deduct" | "set";

const operations: Array<{ value: PointsOperation; icon: typeof Plus }> = [
  { value: "add", icon: Plus },
  { value: "deduct", icon: Minus },
  { value: "set", icon: PencilLine },
];

export function UserPointsDialog({
  row,
  token,
  onClose,
  onSaved,
}: {
  row: AdminListRow | null;
  token: string;
  onClose: () => void;
  onSaved: () => void | Promise<void>;
}) {
  const t = useTranslations("adminPortal.userManagement");
  const [operation, setOperation] = useState<PointsOperation>("add");
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);
  const currentPoints = useMemo(() => numberValue(row?.points), [row?.points]);

  if (!row || row.id === undefined) return null;
  const rowID = row.id;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const numericAmount = Number(amount);
    const valid = Number.isFinite(numericAmount) && (operation === "set" ? numericAmount >= 0 : numericAmount > 0);
    if (!valid) {
      toast.error(t("validation.invalidAmount"));
      return;
    }
    setSaving(true);
    try {
      const result = await updateAdminUserPoints(rowID, {
        operation,
        amount: numericAmount,
        reason: reason.trim(),
      }, token);
      toast.success(t("messages.saved", { points: formatMoney(result.balance_after) }));
      await onSaved();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-[80] flex items-end justify-center bg-[#111827]/45 p-0 backdrop-blur-[2px] sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="admin-user-points-title"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !saving) onClose();
      }}
    >
      <div className="max-h-[92dvh] w-full overflow-y-auto rounded-t-2xl bg-white shadow-2xl sm:max-w-lg sm:rounded-2xl">
        <div className="flex items-start justify-between gap-3 border-b border-black/[0.06] px-4 py-4 sm:px-5">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-700">
              <Coins className="size-5" />
            </span>
            <div className="min-w-0">
              <h2 id="admin-user-points-title" className="text-lg font-semibold text-[#17171d]">{t("title")}</h2>
              <p className="mt-0.5 truncate text-sm text-[#737987]">{String(row.nickname || row.user_id || row.id)}</p>
            </div>
          </div>
          <button
            type="button"
            disabled={saving}
            onClick={onClose}
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-lg text-[#717784] transition hover:bg-[#f1f3f7] disabled:opacity-50"
            aria-label={t("actions.close")}
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="grid grid-cols-3 gap-2 border-b border-black/[0.06] bg-[#fafbfe] px-4 py-3 text-sm sm:px-5">
          <UserMetric label={t("fields.uid")} value={row.uid ?? row.id} />
          <UserMetric label={t("fields.oauth2Id")} value={row.oauth2_id ?? "-"} />
          <UserMetric label={t("fields.points")} value={formatMoney(currentPoints)} emphasis />
        </div>

        <form onSubmit={submit} className="grid gap-4 px-4 py-5 sm:px-5">
          <fieldset className="grid gap-2">
            <legend className="mb-1 text-sm font-semibold text-[#30343b]">{t("operation.label")}</legend>
            <div className="grid grid-cols-3 gap-2">
              {operations.map((item) => {
                const Icon = item.icon;
                return (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => {
                      setOperation(item.value);
                      if (item.value === "set" && amount === "") setAmount(String(currentPoints));
                    }}
                    className={cn(
                      "flex min-w-0 flex-col items-center justify-center gap-1 rounded-xl border px-2 py-3 text-sm font-semibold transition",
                      operation === item.value
                        ? "border-[#1d4ed8]/35 bg-[#eff6ff] text-[#1d4ed8]"
                        : "border-black/[0.08] bg-white text-[#59606c] hover:bg-[#f8fafc]",
                    )}
                  >
                    <Icon className="size-4" />
                    <span>{t(`operation.${item.value}`)}</span>
                  </button>
                );
              })}
            </div>
          </fieldset>

          <label className="grid gap-1.5">
            <span className="text-sm font-semibold text-[#30343b]">
              {operation === "set" ? t("amount.target") : t("amount.change")}
            </span>
            <input
              type="number"
              min={operation === "set" ? 0 : 0.01}
              max={1_000_000_000}
              step="0.01"
              inputMode="decimal"
              required
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              className="h-11 rounded-xl border border-black/[0.09] bg-[#fafbfe] px-3 text-base outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10"
              placeholder={t("amount.placeholder")}
            />
          </label>

          <label className="grid gap-1.5">
            <span className="text-sm font-semibold text-[#30343b]">{t("reason.label")}</span>
            <textarea
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              maxLength={500}
              rows={3}
              className="resize-none rounded-xl border border-black/[0.09] bg-[#fafbfe] px-3 py-2.5 text-sm outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10"
              placeholder={t("reason.placeholder")}
            />
          </label>

          <div className="flex flex-col-reverse gap-2 pt-1 sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" disabled={saving} onClick={onClose} className="h-10 rounded-xl border-black/[0.09]">
              {t("actions.cancel")}
            </Button>
            <Button type="submit" disabled={saving} className="h-10 rounded-xl bg-[#1d4ed8] px-5 hover:bg-[#1e40af]">
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Coins className="size-4" />}
              {saving ? t("actions.saving") : t("actions.save")}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function UserMetric({ label, value, emphasis = false }: { label: string; value: unknown; emphasis?: boolean }) {
  return (
    <div className="min-w-0">
      <span className="block truncate text-[11px] font-semibold uppercase tracking-wide text-[#8a90a0]">{label}</span>
      <span className={cn("mt-1 block truncate", emphasis ? "text-base font-bold text-amber-700" : "font-semibold text-[#30343b]")} title={String(value ?? "-")}>
        {String(value ?? "-")}
      </span>
    </div>
  );
}

function numberValue(value: unknown) {
  const numeric = Number(value ?? 0);
  return Number.isFinite(numeric) ? numeric : 0;
}
