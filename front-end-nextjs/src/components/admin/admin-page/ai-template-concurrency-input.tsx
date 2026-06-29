"use client";

import type { AITemplateConfig } from "@/lib/types";
import type { AIAgentPanelT } from "./ai-agent-panel-model";

type TemplateConcurrencyInputProps = {
  template: AITemplateConfig;
  t: AIAgentPanelT;
  onChange: (concurrency: number) => void;
};

export function TemplateConcurrencyInput({
  template,
  t,
  onChange,
}: TemplateConcurrencyInputProps) {
  const value = normalizeTemplateConcurrency(template.concurrency);
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{t("template.concurrency")}</span>
      <input
        type="number"
        min={0}
        max={50}
        value={value}
        onChange={(event) => onChange(normalizeTemplateConcurrency(event.currentTarget.valueAsNumber))}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      />
      <span className="text-[11px] leading-4 text-[#8b93a2]">{t("template.concurrencyHint")}</span>
    </label>
  );
}

function normalizeTemplateConcurrency(value: number | undefined) {
  if (!Number.isFinite(value) || !value || value < 0) {
    return 0;
  }
  return Math.min(50, Math.floor(value));
}
