import Link from "next/link";
import type { ElementType, SVGProps } from "react";
import { ArrowLeft, BadgeCheck, CheckCircle2, Download, ExternalLink, MonitorSmartphone, Zap } from "lucide-react";
import { getTranslations } from "next-intl/server";
import { Button } from "@/components/ui/button";
import type { AppDownloadConfig, AppDownloadPlatform } from "@/lib/types";

type AppDownloadPageProps = {
  config: AppDownloadConfig;
};

type DownloadTranslations = Awaited<ReturnType<typeof getTranslations>>;

export async function AppDownloadPage({ config }: AppDownloadPageProps) {
  const t = await getTranslations("appDownload");
  const androidFastFeatures = [
    t("featureTinySize"),
    t("featureSameWebStyle"),
    t("featureThirdPartyLogin"),
    t("featureNativeBridge"),
  ];
  const platforms = [
    { key: "android", label: "Android", icon: AndroidIcon, data: config.android, features: [], accentIcon: undefined },
    { key: "android_fast", label: t("androidFastLabel"), icon: AndroidIcon, data: config.android_fast, features: androidFastFeatures, accentIcon: Zap },
    { key: "ios", label: "iOS", icon: MonitorSmartphone, data: config.ios, features: [], accentIcon: undefined },
  ] as const;

  return (
    <main className="min-h-dvh bg-[#f6f7f9] text-[#17171c]">
      <div className="mx-auto flex min-h-dvh w-full max-w-5xl flex-col px-4 py-5 sm:px-6 lg:px-8">
        <header className="flex items-center justify-between gap-3">
          <Button asChild variant="ghost" className="h-10 px-2 text-[#4b5563] hover:bg-black/[0.04]">
            <Link href="/">
              <ArrowLeft className="size-4" />
              <span>{t("backHome")}</span>
            </Link>
          </Button>
          <span className="rounded-full bg-white px-3 py-1 text-xs font-semibold text-[#667085] shadow-sm ring-1 ring-black/[0.06]">
            {t("badge")}
          </span>
        </header>

        <section className="grid flex-1 content-center gap-8 py-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center">
          <div className="min-w-0">
            <div className="inline-flex items-center gap-2 rounded-full bg-[#e8f0ff] px-3 py-1 text-sm font-semibold text-[#1d4ed8]">
              <BadgeCheck className="size-4" />
              {t("official")}
            </div>
            <h1 className="mt-4 text-4xl font-black leading-tight text-[#111827] sm:text-5xl">
              {t("title")}
            </h1>
            <p className="mt-4 max-w-xl text-base leading-7 text-[#667085]">
              {t("description")}
            </p>
          </div>

          <div className="grid gap-4">
            {platforms.map((platform) => (
              <DownloadPanel
                key={platform.key}
                fallbackName={`Yuem ${platform.label}`}
                features={platform.features}
                icon={platform.icon}
                accentIcon={platform.accentIcon}
                label={platform.label}
                platform={platform.data}
                t={t}
              />
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}

function DownloadPanel({
  accentIcon: AccentIcon,
  fallbackName,
  features,
  icon: Icon,
  label,
  platform,
  t,
}: {
  accentIcon?: ElementType<{ className?: string }>;
  fallbackName: string;
  features?: readonly string[];
  icon: ElementType<{ className?: string }>;
  label: string;
  platform?: AppDownloadPlatform;
  t: DownloadTranslations;
}) {
  const enabled = platform?.enabled !== false;
  const downloadUrl = String(platform?.download_url ?? "").trim();
  const available = enabled && downloadUrl.length > 0;
  const versionName = String(platform?.version_name ?? "").trim();
  const versionCode = String(platform?.version_code ?? "").trim();
  const releaseNotes = String(platform?.release_notes ?? "").trim();
  const identifier = String(platform?.package_name ?? platform?.bundle_id ?? "").trim();
  const sizeText = appSizeText(platform);

  return (
    <article className="rounded-lg border border-black/[0.08] bg-white p-5 shadow-sm">
      <div className="flex min-w-0 items-start gap-4">
        <span className="relative flex size-12 shrink-0 items-center justify-center rounded-lg bg-[#f0f4ff] text-[#1d4ed8]">
          <Icon className="size-6" />
          {AccentIcon ? (
            <span className="absolute -right-1 -top-1 flex size-5 items-center justify-center rounded-full bg-[#16a34a] text-white ring-2 ring-white">
              <AccentIcon className="size-3" />
            </span>
          ) : null}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="min-w-0 text-xl font-bold text-[#111827]">
              {platform?.name || fallbackName}
            </h2>
            <span className="rounded-full bg-[#f3f4f6] px-2 py-0.5 text-xs font-semibold text-[#667085]">
              {label}
            </span>
          </div>
          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-[#667085]">
            {versionName ? <span>{t("version", { version: versionName })}</span> : null}
            {versionCode && versionCode !== "0" ? <span>{t("build", { build: versionCode })}</span> : null}
            {sizeText ? <span>{t("size", { size: sizeText })}</span> : null}
            {identifier ? <span className="break-all">{identifier}</span> : null}
          </div>
          {releaseNotes ? (
            <p className="mt-3 whitespace-pre-line text-sm leading-6 text-[#4b5563]">{releaseNotes}</p>
          ) : null}
          {features?.length ? (
            <div className="mt-3 rounded-lg bg-[#f7fbff] p-3 ring-1 ring-[#dbeafe]">
              <p className="text-xs font-bold uppercase tracking-[0.18em] text-[#1d4ed8]">{t("featuresTitle")}</p>
              <ul className="mt-2 grid gap-2 text-sm leading-5 text-[#3f4754]">
                {features.map((feature) => (
                  <li key={feature} className="flex gap-2">
                    <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-[#16a34a]" />
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
          <div className="mt-4">
            {available ? (
              <Button asChild className="h-10 rounded-lg bg-[#1d4ed8] px-4 hover:bg-[#1e40af]">
                <a href={downloadUrl}>
                  <Download className="size-4" />
                  <span>{t("download")}</span>
                  <ExternalLink className="size-4" />
                </a>
              </Button>
            ) : (
              <Button type="button" disabled className="h-10 rounded-lg px-4">
                {t("unavailable")}
              </Button>
            )}
          </div>
        </div>
      </div>
    </article>
  );
}

function appSizeText(platform?: AppDownloadPlatform) {
  const sizeLabel = String(platform?.size_label ?? "").trim();
  if (sizeLabel) return sizeLabel;
  const rawBytes = Number(platform?.size_bytes ?? 0);
  if (!Number.isFinite(rawBytes) || rawBytes <= 0) return "";
  if (rawBytes < 1024) return `${Math.round(rawBytes)} B`;
  if (rawBytes < 1024 * 1024) return `${(rawBytes / 1024).toFixed(1)} KB`;
  return `${(rawBytes / (1024 * 1024)).toFixed(1)} MB`;
}

function AndroidIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" {...props}>
      <path d="M8 8 6.5 5.5" />
      <path d="M16 8 17.5 5.5" />
      <path d="M5 11v5a3 3 0 0 0 3 3h8a3 3 0 0 0 3-3v-5Z" />
      <path d="M5 11a7 7 0 0 1 14 0" />
      <path d="M3 12v4" />
      <path d="M21 12v4" />
      <path d="M9 14h.01" />
      <path d="M15 14h.01" />
    </svg>
  );
}
