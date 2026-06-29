import { FileSearch } from "lucide-react";
import { HeaderCard } from "./layout-widgets";
import { WatermarkAccessListsSection, WatermarkGlobalAccessSection, WatermarkPickerManualSection } from "./hidden-watermark-access-list-sections";
import { WatermarkConfigSection } from "./hidden-watermark-config-section";
import { WatermarkExtractSection } from "./hidden-watermark-extract-section";
import { WatermarkUsageSection } from "./hidden-watermark-usage-section";
import type { HiddenWatermarkAccessController } from "./hidden-watermark-access-view-types";

export function HiddenWatermarkAccessView({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { t } = controller;

  return (
    <div className="grid gap-4">
      <HeaderCard icon={FileSearch} title={t("title")} description={t("description")} tone="blue" />
      <WatermarkConfigSection controller={controller} />
      <WatermarkExtractSection controller={controller} />
      <WatermarkGlobalAccessSection controller={controller} />
      <WatermarkPickerManualSection controller={controller} />
      <WatermarkAccessListsSection controller={controller} />
      <WatermarkUsageSection controller={controller} />
    </div>
  );
}
