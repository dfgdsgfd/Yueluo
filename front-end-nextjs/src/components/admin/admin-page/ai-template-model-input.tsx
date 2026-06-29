import type { AITemplateConfig } from "@/lib/types";
import type { AIAgentPanelT } from "./ai-agent-panel-model";

export function TemplateModelInput({
  template,
  t,
  onChange,
}: {
  template: AITemplateConfig;
  t: AIAgentPanelT;
  onChange: (model: string) => void;
}) {
  return (
    <label className="grid gap-1.5">
      <span className="text-xs font-semibold text-[#666c78]">{t("template.model")}</span>
      <input
        value={template.model ?? ""}
        onChange={(event) => onChange(event.target.value)}
        placeholder={t("template.modelPlaceholder")}
        className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-white px-3 text-sm outline-none focus:border-[#1d4ed8]"
      />
    </label>
  );
}
