import { cn } from "@/lib/utils";

export type AIFormatPhase = "idle" | "running" | "done" | "error";

type ThinkingTickerProps = {
  doneText: string;
  errorText: string;
  fallbackText?: string;
  idleText: string;
  label: string;
  pendingText: string;
  phase: AIFormatPhase;
  running: boolean;
  text: string;
};

export function ThinkingTicker({
  doneText,
  errorText,
  fallbackText,
  idleText,
  label,
  pendingText,
  phase,
  running,
  text,
}: ThinkingTickerProps) {
  const snippet = latestThinkingSnippet(text);
  const displayText = snippet || (running ? fallbackText || pendingText : phase === "done" ? doneText : phase === "error" ? errorText : idleText);
  const shouldScroll = snippet.length > 44;

  return (
    <div className="mt-3 overflow-hidden rounded-xl border border-[#1d4ed8]/15 bg-white/80 px-3 py-2 shadow-[0_12px_30px_rgba(29,78,216,0.08)] backdrop-blur">
      <div className="flex min-w-0 items-center gap-2">
        <span
          className={cn(
            "relative flex size-2.5 shrink-0 rounded-full",
            running ? "bg-[#1d4ed8]" : phase === "error" ? "bg-[#dc2626]" : "bg-[#8aa0bd]",
          )}
          aria-hidden="true"
        >
          {running ? <span className="absolute inset-0 rounded-full bg-[#1d4ed8] opacity-40 motion-safe:animate-ping motion-reduce:hidden" /> : null}
        </span>
        <span className="shrink-0 text-xs font-semibold text-[#1d4ed8]">{label}</span>
        <span className="h-3 w-px shrink-0 bg-[#d7e4ff]" aria-hidden="true" />
        <div className="yuem-thinking-ticker-mask relative min-w-0 flex-1 overflow-hidden">
          <div
            aria-live="polite"
            className={cn(
              "flex min-w-full items-center gap-8 whitespace-nowrap text-xs leading-5 text-[#4d5562]",
              shouldScroll ? "w-max motion-safe:animate-[yuem-thinking-ticker_24s_linear_infinite] motion-reduce:animate-none" : "",
            )}
          >
            <span className={cn("min-w-0", shouldScroll ? "shrink-0" : "truncate")}>{displayText}</span>
            {shouldScroll ? <span className="shrink-0" aria-hidden="true">{displayText}</span> : null}
          </div>
        </div>
      </div>
    </div>
  );
}

function latestThinkingSnippet(value: string) {
  return value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .slice(-3)
    .join(" / ")
    .replace(/\s+/g, " ")
    .trim();
}
