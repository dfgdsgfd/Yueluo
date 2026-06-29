import type { useTranslations } from "next-intl";
import type { PublishGenerationState } from "./ai-publish-generation";

export function publishGenerationStatusLabel(
  t: ReturnType<typeof useTranslations>,
  state: PublishGenerationState,
) {
  if (state.queue) {
    return t("publish.aiGenerate.queue", {
      position: state.queue.position,
      total: state.queue.total,
    });
  }
  if (state.phase === "done") return t("publish.aiGenerate.done");
  if (state.phase === "error") return t("publish.aiGenerate.failed");
  if (state.activeField === "detail") return t("publish.aiGenerate.generatingDetail");
  if (state.activeField === "title") return t("publish.aiGenerate.generatingTitle");
  if (state.stage === "connected") return t("publish.aiGenerate.connected");
  return t("publish.aiGenerate.generating");
}
