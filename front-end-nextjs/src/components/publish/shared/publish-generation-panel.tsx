import { MarkdownContent } from "@/components/markdown-content";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2, RotateCcw, Sparkles, Wand2, X } from "lucide-react";
import type { useTranslations } from "next-intl";
import { useEffect, useRef } from "react";
import type { PublishGenerationState } from "./ai-publish-generation";
import { ThinkingTicker } from "./ai-format-thinking-ticker";
import type { PublishGenerationRunOptions } from "./publish-generation-action";
import { publishGenerationStatusLabel } from "./publish-generation-status";

type PublishGenerationPanelProps = {
  canRun: boolean;
  imageCount: number;
  onApply: () => void;
  onCancel: () => void;
  onClose: () => void;
  onRun: (options?: PublishGenerationRunOptions) => void;
  open: boolean;
  state: PublishGenerationState;
  t: ReturnType<typeof useTranslations>;
  variant?: "desktop" | "mobile";
};

export function PublishGenerationPanel({
  canRun,
  imageCount,
  onApply,
  onCancel,
  onClose,
  onRun,
  open,
  state,
  t,
  variant = "desktop",
}: PublishGenerationPanelProps) {
  const detailRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (detailRef.current) {
      detailRef.current.scrollTop = detailRef.current.scrollHeight;
    }
  }, [state.generatedDetail]);

  if (!open) {
    return null;
  }

  const running = state.phase === "queued" || state.phase === "running";
  const error = state.phase === "error";
  const canApply = Boolean(state.generatedTitle.trim() || state.generatedDetail.trim());
  const regenerate = Boolean(state.generatedTitle || state.generatedDetail || error);
  const status = publishGenerationStatusLabel(t, state);
  const previewPhase = state.phase === "done"
    ? "done"
    : state.phase === "error"
      ? "error"
      : state.phase === "idle"
        ? "idle"
        : "running";
  const imageSummary = imageCount > 0
    ? t("publish.aiGenerate.cardBody", { count: imageCount })
    : t("publish.aiGenerate.noImages");

  return (
    <div
      className={cn(
        "fixed inset-0 z-[70] flex bg-[#111827]/55 p-3 backdrop-blur-[3px] sm:p-5",
        variant === "mobile" ? "items-end justify-center" : "items-center justify-center",
      )}
      role="dialog"
      aria-modal="true"
    >
      <button type="button" aria-label={t("publish.aiGenerate.close")} className="absolute inset-0" onClick={onClose} />
      <section
        className={cn(
          "relative z-10 flex min-h-0 w-full flex-col overflow-hidden border border-white/70 bg-white text-[#20232a] shadow-2xl",
          variant === "mobile"
            ? "max-h-[90dvh] max-w-[430px] rounded-t-[22px]"
            : "h-[min(820px,calc(100dvh-40px))] max-w-[920px] rounded-[18px]",
        )}
      >
        <header className="shrink-0 border-b border-black/[0.06] bg-[#fbfcff] px-4 py-3 sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-[#e8f1ff] text-[#1d4ed8]">
              {running ? <Loader2 className="size-5 animate-spin" /> : <Sparkles className="size-5" />}
            </span>
            <div className="min-w-0 flex-1">
              <h2 className="truncate text-base font-semibold text-[#17171d]">{t("publish.aiGenerate.panelTitle")}</h2>
              <p className="truncate text-xs text-[#7b8190]">{[status, imageSummary].filter(Boolean).join(" · ")}</p>
            </div>
            <button
              type="button"
              aria-label={t("publish.aiGenerate.close")}
              onClick={onClose}
              className="flex size-9 shrink-0 items-center justify-center rounded-lg text-[#5f6673] hover:bg-[#eef1f6]"
            >
              <X className="size-4" />
            </button>
          </div>
          <div className="mt-3">
            <div className="mb-1 flex items-center justify-between gap-3 text-xs font-semibold text-[#666c78]">
              <span className="truncate">{status}</span>
              <span>{Math.round(state.percent)}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-[#e8ecf3]">
              <div
                className="h-full rounded-full bg-[#1d4ed8] transition-[width]"
                style={{ width: `${Math.max(0, Math.min(100, state.percent))}%` }}
              />
            </div>
          </div>
          <ThinkingTicker
            doneText={t("publish.aiGenerate.thinkingDone")}
            errorText={t("publish.aiGenerate.thinkingError")}
            fallbackText={publishGenerationThinkingFallback(t, state)}
            idleText={t("publish.aiGenerate.thinkingIdle")}
            label={t("publish.aiGenerate.thinking")}
            pendingText={t("publish.aiGenerate.thinkingPending")}
            phase={previewPhase}
            running={running}
            text={state.reasoning}
          />
        </header>

        <div className="grid min-h-0 flex-1 gap-3 overflow-y-auto bg-[#f5f7fb] p-4 sm:p-5">
          <section className="overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm">
            <div className="border-b border-black/[0.06] px-4 py-3">
              <p className="text-sm font-semibold text-[#20232a]">{t("publish.aiGenerate.titlePreview")}</p>
            </div>
            <div className="min-h-16 px-4 py-3 text-sm leading-6">
              {state.generatedTitle ? (
                <p className="break-words font-semibold text-[#1f3654]">{state.generatedTitle}</p>
              ) : (
                <p className="text-[#9aa1ad]">{running ? t("publish.aiGenerate.titlePending") : t("publish.aiGenerate.emptyTitle")}</p>
              )}
            </div>
          </section>

          <section className="flex min-h-[min(46dvh,430px)] min-w-0 flex-col overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-sm">
            <div className="flex shrink-0 items-center justify-between gap-3 border-b border-black/[0.06] px-4 py-3">
              <p className="truncate text-sm font-semibold text-[#20232a]">{t("publish.aiGenerate.detailPreview")}</p>
              {state.stage && running ? (
                <span className="shrink-0 rounded-full bg-[#eef4ff] px-3 py-1 text-xs font-semibold text-[#1d4ed8]">
                  {status}
                </span>
              ) : null}
            </div>
            <div
              ref={detailRef}
              className={cn(
                "min-h-0 flex-1 overflow-auto px-4 py-4 text-sm leading-7 sm:px-6",
                state.generatedDetail ? "text-[#20232a]" : "text-[#9aa1ad]",
              )}
            >
              {state.generatedDetail ? (
                <MarkdownContent content={state.generatedDetail} />
              ) : (
                <p className="whitespace-pre-wrap break-words">
                  {running ? t("publish.aiGenerate.detailPending") : t("publish.aiGenerate.emptyDetail")}
                </p>
              )}
            </div>
          </section>

          {state.queue ? (
            <p className="text-xs font-medium text-[#1d4ed8]">
              {t("publish.aiGenerate.queue", {
                position: state.queue.position,
                total: state.queue.total,
              })}
            </p>
          ) : null}
          {state.error ? (
            <p className="text-xs font-medium text-[#dc2626]">{state.error}</p>
          ) : null}
        </div>

        <footer className="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-black/[0.06] bg-white px-4 py-3">
          {running ? (
            <Button type="button" variant="outline" onClick={() => void onCancel()} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
              <X className="size-4" />
              <span>{t("publish.aiGenerate.cancel")}</span>
            </Button>
          ) : (
            <Button
              type="button"
              variant="outline"
              disabled={!canRun}
              onClick={() => onRun(regenerate ? { fresh: true } : undefined)}
              className="h-9 rounded-lg border-black/[0.08] bg-white px-3"
            >
              {error ? <RotateCcw className="size-4" /> : <Wand2 className="size-4" />}
              <span>{regenerate ? t("publish.aiGenerate.regenerate") : t("publish.aiGenerate.button")}</span>
            </Button>
          )}
          <Button
            type="button"
            disabled={!canApply}
            onClick={() => {
              onApply();
              onClose();
            }}
            className="h-9 rounded-lg bg-[#1d4ed8] px-4 hover:bg-[#1e40af]"
          >
            {running ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
            <span>{t("publish.aiGenerate.apply")}</span>
          </Button>
        </footer>
      </section>
    </div>
  );
}

function publishGenerationThinkingFallback(
  t: ReturnType<typeof useTranslations>,
  state: PublishGenerationState,
) {
  if (state.queue) return t("publish.aiGenerate.thinkingQueued");
  if (state.activeField === "detail") return t("publish.aiGenerate.thinkingDetail");
  if (state.activeField === "title") return t("publish.aiGenerate.thinkingTitle");
  if (state.stage === "connected") return t("publish.aiGenerate.thinkingConnected");
  return t("publish.aiGenerate.thinkingPending");
}
