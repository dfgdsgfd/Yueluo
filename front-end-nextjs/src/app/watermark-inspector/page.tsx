import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { WatermarkInspectorPage } from "@/components/watermark/watermark-inspector-page";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("watermarkInspector");
  return {
    title: t("title"),
  };
}

export default function WatermarkInspectorRoute() {
  return <WatermarkInspectorPage />;
}
