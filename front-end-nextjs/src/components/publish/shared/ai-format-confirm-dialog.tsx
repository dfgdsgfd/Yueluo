"use client";

import { Loader2, Sparkles, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type AIFormatConfirmDialogProps = {
  cancelLabel: string;
  confirmLabel: string;
  description: string;
  loading?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  open: boolean;
  title: string;
  variant?: "desktop" | "mobile";
};

export function AIFormatConfirmDialog({
  cancelLabel,
  confirmLabel,
  description,
  loading = false,
  onCancel,
  onConfirm,
  open,
  title,
  variant = "desktop",
}: AIFormatConfirmDialogProps) {
  if (!open) {
    return null;
  }
  return (
    <div className="absolute inset-0 z-20 flex items-end justify-center bg-[#111827]/35 p-3 backdrop-blur-[2px] sm:items-center">
      <button type="button" aria-label={cancelLabel} className="absolute inset-0" onClick={onCancel} />
      <div
        className={cn(
          "relative z-10 w-full border border-white/70 bg-white p-4 text-[#20232a] shadow-2xl",
          variant === "mobile" ? "max-w-[430px] rounded-2xl" : "max-w-[420px] rounded-2xl",
        )}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-[#e8f1ff] text-[#1d4ed8]">
              <Sparkles className="size-5" />
            </span>
            <div className="min-w-0">
              <h3 className="text-base font-semibold leading-6 text-[#17171d]">{title}</h3>
              <p className="mt-1 text-sm leading-6 text-[#687081]">{description}</p>
            </div>
          </div>
          <button
            type="button"
            aria-label={cancelLabel}
            disabled={loading}
            onClick={onCancel}
            className="flex size-8 shrink-0 items-center justify-center rounded-lg text-[#6b7280] hover:bg-[#f1f4f8] disabled:opacity-50"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-2">
          <Button type="button" variant="outline" disabled={loading} onClick={onCancel} className="h-11 rounded-xl border-black/[0.08]">
            {cancelLabel}
          </Button>
          <Button type="button" disabled={loading} onClick={onConfirm} className="h-11 rounded-xl bg-[#1d4ed8] hover:bg-[#1e40af]">
            {loading ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
            <span>{confirmLabel}</span>
          </Button>
        </div>
      </div>
    </div>
  );
}
