import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { NetworkDiagnosticsPage } from "@/components/network-diagnostics/network-diagnostics-page";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("networkDiagnostics");
  return {
    title: t("metadataTitle"),
  };
}

export default function NetworkDiagnosticsRoute() {
  return <NetworkDiagnosticsPage />;
}
