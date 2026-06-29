"use client";
import {
  ArrowRight,
  Loader2
} from "lucide-react";
import {
  cn
} from "@/lib/utils";

export function SectionHeader({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="flex min-h-7 items-center justify-between gap-4">
      <h2 className="text-base font-semibold text-[#25252b]">{title}</h2>
      {description ? (
        <p className="truncate text-sm text-[#8a8a91]">{description}</p>
      ) : null}
    </div>
  );
}


export function CreatorEmptyState({ compact, label }: { compact?: boolean; label: string }) {
  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-xl border border-dashed border-[#dedee3] bg-white px-4 text-center text-xs leading-5 text-[#9a9aa1]",
        compact ? "min-h-[72px] py-4" : "min-h-[188px] py-8",
      )}
    >
      {label}
    </div>
  );
}


export function CreatorLoadMoreButton({
  label,
  loading,
  onClick,
}: {
  label: string;
  loading: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      className="mx-auto mt-4 flex h-8 items-center gap-1 rounded-full bg-white px-3 text-xs font-medium text-[#777780] shadow-sm transition-colors hover:text-primary disabled:cursor-not-allowed disabled:text-[#b0b0b8]"
    >
      {loading ? <Loader2 className="size-3.5 animate-spin" /> : <ArrowRight className="size-3.5" />}
      {label}
    </button>
  );
}
