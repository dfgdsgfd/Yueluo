import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Loader2, RotateCcw, Sparkles, Wand2, X } from "lucide-react";
import type { useTranslations } from "next-intl";
import type { PublishGenerationState } from "./ai-publish-generation";
import type { PublishGenerationRunOptions } from "./publish-generation-action";
import { publishGenerationStatusLabel } from "./publish-generation-status";

type PublishGenerationCardProps = {
  canRun: boolean;
  imageCount: number;
  onCancel: () => void;
  onOpen: () => void;
  onRun: (options?: PublishGenerationRunOptions) => void;
  state: PublishGenerationState;
  t: ReturnType<typeof useTranslations>;
  variant?: "desktop" | "mobile";
};

export function PublishGenerationCard({
  canRun,
  imageCount,
  onCancel,
  onOpen,
  onRun,
  state,
  t,
  variant = "desktop",
}: PublishGenerationCardProps) {
  const enabledByAdmin = state.config?.enabled !== false;
  if (!state.loading && !enabledByAdmin) {
    return null;
  }
  const running = state.phase === "queued" || state.phase === "running";
  const done = state.phase === "done";
  const error = state.phase === "error";
  const status = publishGenerationStatusLabel(t, state);
  const disabled = !canRun || running || state.loading || !enabledByAdmin;
  const compact = variant === "mobile";

  return (
    <section
      className={cn(
        "overflow-hidden border bg-white shadow-[0_10px_30px_rgba(15,23,42,0.06)]",
        compact
          ? "rounded-[14px] border-[var(--mobile-publish-border-soft)]"
          : "rounded-2xl border-[#e7edf6]",
      )}
    >
      <div
        className={cn(
          "flex items-center gap-3",
          compact ? "px-4 py-3" : "px-4 py-4 sm:px-5",
        )}
      >
        <span
          className={cn(
            "flex shrink-0 items-center justify-center rounded-xl bg-[#eaf5ff] text-[#0f68b8]",
            compact ? "size-10" : "size-11",
          )}
        >
          {running ? <Loader2 className="size-5 animate-spin" /> : <Sparkles className="size-5" />}
        </span>
        <div className="min-w-0 flex-1">
          <p className={cn("font-semibold text-[#1f2937]", compact ? "text-[15px]" : "text-sm")}>
            {t("publish.aiGenerate.cardTitle")}
          </p>
          <p className={cn("mt-0.5 truncate text-[#64748b]", compact ? "text-[12px]" : "text-xs")}>
            {imageCount > 0
              ? t("publish.aiGenerate.cardBody", { count: imageCount })
              : t("publish.aiGenerate.noImages")}
          </p>
        </div>
        {running ? (
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={onOpen}
              className={cn(
                "gap-2 rounded-full border-[#cbd5e1] bg-white text-[#315a7f] hover:bg-[#f8fbff]",
                compact ? "h-9 px-3 text-xs" : "h-10 px-4 text-sm",
              )}
            >
              <Sparkles className="size-4" />
              {t("publish.aiGenerate.view")}
            </Button>
            <button
              type="button"
              aria-label={t("publish.aiGenerate.cancel")}
              onClick={onCancel}
              className="flex size-9 shrink-0 items-center justify-center rounded-full text-[#64748b] transition hover:bg-[#f1f5f9]"
            >
              <X className="size-4" />
            </button>
          </div>
        ) : done ? (
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              onClick={onOpen}
              className={cn(
                "gap-2 bg-[#0f68b8] text-white hover:bg-[#0b579b]",
                compact ? "h-9 rounded-full px-3 text-xs" : "h-10 rounded-full px-4 text-sm",
              )}
            >
              <Sparkles className="size-4" />
              {t("publish.aiGenerate.review")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => onRun({ fresh: true })}
              disabled={!canRun}
              className={cn(
                "gap-2 rounded-full border-[#cbd5e1] bg-white text-[#315a7f] hover:bg-[#f8fbff]",
                compact ? "h-9 px-3 text-xs" : "h-10 px-4 text-sm",
              )}
            >
              <RotateCcw className="size-4" />
              {t("publish.aiGenerate.regenerate")}
            </Button>
          </div>
        ) : (
          <Button
            type="button"
            onClick={() => onRun(error ? { fresh: true } : undefined)}
            disabled={disabled}
            className={cn(
              "shrink-0 gap-2 bg-[#0f68b8] text-white hover:bg-[#0b579b]",
              compact ? "h-9 rounded-full px-3 text-xs" : "h-10 rounded-full px-4 text-sm",
            )}
          >
            <Wand2 className="size-4" />
            {error ? t("publish.aiGenerate.regenerate") : t("publish.aiGenerate.button")}
          </Button>
        )}
      </div>

      {running || done || error ? (
        <div className={cn("border-t border-[#e7edf6] bg-[#f8fbff]", compact ? "px-4 py-3" : "px-5 py-3")}>
          <div className="flex items-center justify-between gap-3">
            <p className="min-w-0 truncate text-xs font-semibold text-[#315a7f]">{status}</p>
            <span className="shrink-0 text-xs font-semibold text-[#315a7f]">{Math.round(state.percent)}%</span>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-[#dbeafe]">
            <div
              className="h-full rounded-full bg-[#0f68b8] transition-[width]"
              style={{ width: `${Math.max(4, Math.min(100, state.percent))}%` }}
            />
          </div>
          {state.error ? <p className="mt-2 line-clamp-2 text-xs text-[#dc2626]">{state.error}</p> : null}
        </div>
      ) : null}
    </section>
  );
}
