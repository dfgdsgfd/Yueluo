"use client";

import type { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

type AIFormatPromptSamplesProps = {
  disabled?: boolean;
  onPick: (value: string) => void;
  t: ReturnType<typeof useTranslations<"publish.aiFormat">>;
};

export function AIFormatPromptSamples({ disabled, onPick, t }: AIFormatPromptSamplesProps) {
  const samples = [
    {
      key: "shortPost",
      label: t("customPromptSampleLabels.shortPost"),
      prompt: t("customPromptSamples.shortPost"),
    },
    {
      key: "novelMarkdown",
      label: t("customPromptSampleLabels.novelMarkdown"),
      prompt: t("customPromptSamples.novelMarkdown"),
    },
  ];
  return (
    <div className="flex min-w-0 flex-nowrap gap-1.5 overflow-x-auto pb-0.5">
      {samples.map((sample) => (
        <button
          key={sample.key}
          type="button"
          disabled={disabled}
          title={sample.prompt}
          aria-label={sample.prompt}
          onClick={() => onPick(sample.prompt)}
          className={cn(
            "h-8 shrink-0 rounded-full border border-[#1d4ed8]/15 bg-[#eef4ff] px-2.5 text-xs font-semibold text-[#1d4ed8] shadow-sm transition hover:border-[#1d4ed8]/25 hover:bg-[#dbeafe]",
            disabled && "cursor-not-allowed opacity-60",
          )}
        >
          {sample.label}
        </button>
      ))}
    </div>
  );
}
