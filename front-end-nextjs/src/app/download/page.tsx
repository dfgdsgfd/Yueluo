import { headers } from "next/headers";
import { getTranslations } from "next-intl/server";
import { AppDownloadPage } from "@/components/download/app-download-page";
import { apiRequestContextFromHeaders, getAppDownloadConfig } from "@/lib/api";
import type { AppDownloadConfig } from "@/lib/types";

export async function generateMetadata() {
  const t = await getTranslations("appDownload");
  return {
    title: t("metadataTitle"),
  };
}

export default async function DownloadRoute() {
  const headerStore = await headers();
  const config = await getAppDownloadConfig(apiRequestContextFromHeaders(headerStore)).catch(
    (): AppDownloadConfig => ({}),
  );

  return <AppDownloadPage config={config} />;
}
