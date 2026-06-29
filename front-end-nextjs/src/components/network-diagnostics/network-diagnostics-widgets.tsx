"use client";

import Link from "next/link";
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  Copy,
  Globe2,
  Info,
  Network,
  RadioTower,
  RefreshCw,
  Server,
  ShieldAlert,
  Wifi,
  WifiOff,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import type {
  HeaderPair,
  NetworkCdnProvider,
  NetworkCdnSnapshot,
} from "@/lib/network-diagnostics";
import { cn } from "@/lib/utils";
import type {
  GeoSnapshot,
  NetworkDiagnosticsSnapshot,
  NetworkProbeError,
} from "./network-probes";
import {
  CandidateList,
  DetailGrid,
  HeaderList,
  LoadingBlock,
  MetricCard,
  SectionCard,
} from "./network-diagnostics-primitives";

type DiagnosticsHeaderProps = {
  checking: boolean;
  copied: boolean;
  hasSnapshot: boolean;
  online: boolean | null;
  onCopy: () => void;
  onRefresh: () => void;
};

type SnapshotSectionProps = {
  checking: boolean;
  snapshot: NetworkDiagnosticsSnapshot | null;
};

export function DiagnosticsHeader({
  checking,
  copied,
  hasSnapshot,
  online,
  onCopy,
  onRefresh,
}: DiagnosticsHeaderProps) {
  const t = useTranslations("networkDiagnostics");

  return (
    <header className="sticky top-0 z-30 border-b border-white/[0.08] bg-[#121212]/95 backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-[1180px] items-center gap-2 px-3">
        <Button
          asChild
          variant="ghost"
          size="icon"
          className="size-10 text-white/78 hover:bg-white/[0.06] hover:text-white"
        >
          <Link href="/settings">
            <ArrowLeft className="size-5" />
            <span className="sr-only">{t("back")}</span>
          </Link>
        </Button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-lg font-bold">{t("title")}</h1>
        </div>
        <span
          className={cn(
            "hidden h-8 items-center gap-2 rounded-full border px-3 text-xs font-bold sm:inline-flex",
            online === false
              ? "border-rose-300/25 bg-rose-300/10 text-rose-100"
              : "border-emerald-300/20 bg-emerald-300/10 text-emerald-100",
          )}
        >
          {online === false ? <WifiOff className="size-4" /> : <Wifi className="size-4" />}
          {online === false ? t("offline") : t("online")}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          disabled={checking}
          aria-label={t("refresh")}
          onClick={onRefresh}
          className="size-10 text-white/70 hover:bg-white/[0.06] hover:text-white"
        >
          <RefreshCw className={cn("size-5", checking && "animate-spin")} />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          disabled={!hasSnapshot}
          aria-label={t("copy")}
          onClick={onCopy}
          className="size-10 text-white/70 hover:bg-white/[0.06] hover:text-white"
        >
          {copied ? <Check className="size-5" /> : <Copy className="size-5" />}
        </Button>
      </div>
    </header>
  );
}

export function SummaryGrid({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const publicIp = getBestPublicIp(snapshot);
  const geo = getGeoLabel(snapshot?.geo);
  const cdn = getCdnSummary(snapshot, t);
  const privacy = getPrivacySummary(snapshot, t);

  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <MetricCard
        icon={Globe2}
        label={t("summary.publicIp")}
        loading={checking && !snapshot}
        tone="sky"
        value={publicIp ?? t("unknown")}
        detail={snapshot?.publicIp?.provider ?? snapshot?.geo?.provider ?? t("summary.serverSeen")}
      />
      <MetricCard
        icon={Network}
        label={t("summary.geo")}
        loading={checking && !snapshot}
        tone="emerald"
        value={geo.primary}
        detail={geo.secondary}
      />
      <MetricCard
        icon={Server}
        label={t("summary.cdn")}
        loading={checking && !snapshot}
        tone="violet"
        value={cdn.primary}
        detail={cdn.secondary}
      />
      <MetricCard
        icon={ShieldAlert}
        label={t("summary.privacy")}
        loading={checking && !snapshot}
        tone={privacy.tone}
        value={privacy.primary}
        detail={privacy.secondary}
      />
    </section>
  );
}

export function BrowserNetworkSection({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const browser = snapshot?.browser;

  return (
    <SectionCard icon={Wifi} title={t("sections.browser")}>
      {browser ? (
        <DetailGrid
          rows={[
            [t("labels.onlineStatus"), browser.online ? t("online") : t("offline")],
            [t("labels.host"), browser.host],
            [t("labels.protocol"), browser.protocol],
            [t("labels.effectiveType"), browser.effectiveType],
            [t("labels.connectionType"), browser.type],
            [t("labels.rtt"), formatUnit(browser.rttMs, "ms")],
            [t("labels.downlink"), formatUnit(browser.downlinkMbps, "Mbps")],
            [t("labels.saveData"), formatBoolean(browser.saveData, t)],
            [t("labels.timezone"), browser.timezone],
            [t("labels.language"), browser.language],
            [t("labels.viewport"), browser.viewport],
            [t("labels.screen"), browser.screen],
            [t("labels.platform"), browser.platform],
          ]}
        />
      ) : (
        <LoadingBlock checking={checking} />
      )}
    </SectionCard>
  );
}

export function GeoSection({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const geo = snapshot?.geo;
  const publicIp = snapshot?.publicIp;

  return (
    <SectionCard icon={Globe2} title={t("sections.geo")}>
      {geo || publicIp ? (
        <DetailGrid
          rows={[
            [t("labels.ip"), geo?.ip ?? publicIp?.ip],
            [t("labels.country"), countryLine(geo)],
            [t("labels.region"), geo?.region],
            [t("labels.city"), geo?.city],
            [t("labels.isp"), geo?.isp],
            [t("labels.asn"), geo?.asn],
            [t("labels.coordinates"), formatCoordinates(geo)],
            [t("labels.timezone"), geo?.timezone],
            [t("labels.provider"), geo?.provider ?? publicIp?.provider],
            [t("labels.fetchedAt"), publicIp?.fetchedAt],
          ]}
        />
      ) : (
        <LoadingBlock checking={checking} />
      )}
    </SectionCard>
  );
}

export function ServerSection({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const server = snapshot?.server;
  const bestCdn = getBestCdnResult(snapshot);

  return (
    <SectionCard icon={Server} title={t("sections.server")}>
      {server ? (
        <div className="space-y-5">
          <DetailGrid
            rows={[
              [t("labels.serverIp"), server.observedIp],
              [t("labels.host"), server.host],
              [t("labels.protocol"), server.protocol],
              [t("labels.latency"), formatUnit(snapshot?.serverLatencyMs ?? null, "ms")],
              [t("labels.cdnProvider"), getCdnProviderLabel(bestCdn?.cdn.provider ?? null, t)],
              [t("labels.cdnSource"), bestCdn ? t(`cdnSources.${bestCdn.source}`) : null],
              [t("labels.cdnCountry"), bestCdn?.cdn.countryCode],
              [t("labels.cacheStatus"), bestCdn?.cdn.cacheStatus],
              [t("labels.cdnRay"), bestCdn?.cdn.rayId],
              [t("labels.requestCdnProvider"), getCdnProviderLabel(server.cdn.provider, t)],
              [t("labels.timestamp"), server.timestamp],
            ]}
          />
          <HeaderList title={t("labels.forwardedIps")} headers={toValuePairs(server.forwardedIps)} />
          <HeaderList title={t("labels.pageResponseHeaders")} headers={snapshot?.pageResponse?.headers ?? []} />
          <HeaderList title={t("labels.apiResponseHeaders")} headers={snapshot?.edgeResponse?.headers ?? []} />
          <HeaderList title={t("labels.cdnEvidence")} headers={bestCdn?.cdn.evidence ?? []} />
          <HeaderList title={t("labels.requestCdnEvidence")} headers={server.cdn.evidence} />
          <HeaderList title={t("labels.selectedHeaders")} headers={server.headers} />
        </div>
      ) : (
        <LoadingBlock checking={checking} />
      )}
    </SectionCard>
  );
}

export function PrivacySection({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const summary = getPrivacySummary(snapshot, t);
  const addresses = snapshot?.webRtc.addresses ?? [];

  return (
    <SectionCard icon={ShieldAlert} title={t("sections.privacy")}>
      {snapshot ? (
        <div className="space-y-4">
          <div
            className={cn(
              "flex items-start gap-3 rounded-[8px] border px-3 py-3",
              summary.tone === "rose"
                ? "border-rose-300/20 bg-rose-300/10 text-rose-50"
                : summary.tone === "amber"
                  ? "border-amber-300/20 bg-amber-300/10 text-amber-50"
                  : "border-emerald-300/20 bg-emerald-300/10 text-emerald-50",
            )}
          >
            {summary.tone === "rose" || summary.tone === "amber" ? (
              <AlertTriangle className="mt-0.5 size-5 shrink-0" />
            ) : (
              <Check className="mt-0.5 size-5 shrink-0" />
            )}
            <div className="min-w-0">
              <p className="text-sm font-bold">{summary.primary}</p>
              <p className="mt-1 text-xs leading-5 opacity-75">{summary.secondary}</p>
            </div>
          </div>
          <p className="flex gap-2 text-xs leading-5 text-white/45">
            <Info className="mt-0.5 size-4 shrink-0 text-sky-200/70" />
            <span>{t("privacy.preciseDns")}</span>
          </p>
          <CandidateList addresses={addresses} supported={snapshot.webRtc.supported} />
        </div>
      ) : (
        <LoadingBlock checking={checking} />
      )}
    </SectionCard>
  );
}

export function TraceSection({ checking, snapshot }: SnapshotSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const trace = snapshot?.cloudflareTrace;

  return (
    <SectionCard icon={RadioTower} title={t("sections.trace")}>
      {trace ? (
        <DetailGrid
          rows={[
            [t("labels.ip"), trace.ip],
            [t("labels.cloudflareColo"), trace.colo],
            [t("labels.country"), trace.loc],
            [t("labels.http"), trace.http],
            [t("labels.tls"), trace.tls],
            [t("labels.warp"), trace.warp],
            [t("labels.gateway"), trace.gateway],
          ]}
        />
      ) : (
        <LoadingBlock checking={checking} unavailableLabel={t("traceUnavailable")} />
      )}
    </SectionCard>
  );
}

export function ProbeErrorsSection({ errors }: { errors: NetworkProbeError[] }) {
  const t = useTranslations("networkDiagnostics");

  if (errors.length === 0) {
    return null;
  }

  return (
    <SectionCard icon={AlertTriangle} title={t("errors.title")}>
      <p className="text-sm leading-6 text-white/50">{t("errors.description")}</p>
      <div className="mt-3 space-y-2">
        {errors.map((error) => (
          <div
            key={`${error.source}-${error.message}`}
            className="rounded-[8px] border border-amber-300/15 bg-amber-300/8 px-3 py-2 text-xs text-amber-50/80"
          >
            <span className="font-bold">{t(`errorSources.${error.source}`)}</span>
            <span className="text-amber-50/45"> - </span>
            <span>{error.message}</span>
          </div>
        ))}
      </div>
    </SectionCard>
  );
}

function getBestPublicIp(snapshot: NetworkDiagnosticsSnapshot | null) {
  return (
    snapshot?.geo?.ip ??
    snapshot?.publicIp?.ip ??
    snapshot?.cloudflareTrace?.ip ??
    snapshot?.originServer?.clientIp ??
    snapshot?.server?.observedIp ??
    null
  );
}

function getGeoLabel(geo: GeoSnapshot | null | undefined) {
  const primary = countryLine(geo) ?? "-";
  const secondary = [geo?.city, geo?.region].filter(Boolean).join(" / ");
  return {
    primary,
    secondary: secondary || geo?.timezone || null,
  };
}

function getCdnSummary(
  snapshot: NetworkDiagnosticsSnapshot | null,
  t: ReturnType<typeof useTranslations>,
) {
  const result = getBestCdnResult(snapshot);
  if (!result) {
    return {
      primary: t("unknown"),
      secondary: null,
    };
  }
  return {
    primary: result.cdn.detected ? getCdnProviderLabel(result.cdn.provider, t) : t("notDetected"),
    secondary: result.cdn.cacheStatus ?? result.cdn.rayId ?? result.cdn.countryCode ?? null,
  };
}

function getBestCdnResult(snapshot: NetworkDiagnosticsSnapshot | null): {
  cdn: NetworkCdnSnapshot;
  source: "apiResponse" | "originServerRequest" | "pageResponse" | "serverRequest";
} | null {
  const pageCdn = snapshot?.pageResponse?.cdn;
  if (pageCdn && (pageCdn.detected || pageCdn.evidence.length > 0)) {
    return {
      cdn: pageCdn,
      source: "pageResponse",
    };
  }

  const apiCdn = snapshot?.edgeResponse?.cdn;
  if (apiCdn && (apiCdn.detected || apiCdn.evidence.length > 0)) {
    return {
      cdn: apiCdn,
      source: "apiResponse",
    };
  }

  const originCdn = snapshot?.originServer?.cdn;
  if (originCdn && (originCdn.detected || originCdn.evidence.length > 0)) {
    return {
      cdn: originCdn,
      source: "originServerRequest",
    };
  }

  const serverCdn = snapshot?.server?.cdn;
  if (serverCdn) {
    return {
      cdn: serverCdn,
      source: "serverRequest",
    };
  }
  return null;
}

function getPrivacySummary(
  snapshot: NetworkDiagnosticsSnapshot | null,
  t: ReturnType<typeof useTranslations>,
) {
  if (!snapshot) {
    return {
      primary: t("unknown"),
      secondary: t("privacy.dnsLimited"),
      tone: "amber" as const,
    };
  }

  if (!snapshot.webRtc.supported) {
    return {
      primary: t("privacy.webrtcUnsupportedShort"),
      secondary: t("privacy.dnsLimited"),
      tone: "amber" as const,
    };
  }

  const knownPublicIps = new Set(
    [snapshot.publicIp?.ip, snapshot.geo?.ip, snapshot.cloudflareTrace?.ip]
      .filter((value): value is string => Boolean(value)),
  );
  const exposedPublicIps = snapshot.webRtc.addresses
    .filter((item) => item.kind === "public")
    .map((item) => item.address);
  const extraPublicIps = exposedPublicIps.filter((ip) => !knownPublicIps.has(ip));

  if (extraPublicIps.length > 0) {
    return {
      primary: t("privacy.webrtcExtraPublic"),
      secondary: extraPublicIps.join(", "),
      tone: "rose" as const,
    };
  }

  if (exposedPublicIps.length > 0) {
    return {
      primary: t("privacy.webrtcSamePublic"),
      secondary: t("privacy.dnsLimited"),
      tone: "amber" as const,
    };
  }

  return {
    primary: t("privacy.webrtcNoPublic"),
    secondary: t("privacy.dnsLimited"),
    tone: "emerald" as const,
  };
}

function getCdnProviderLabel(
  provider: NetworkCdnProvider | null,
  t: ReturnType<typeof useTranslations>,
) {
  return provider ? t(`cdnProviders.${provider}`) : t("notDetected");
}

function countryLine(geo: GeoSnapshot | null | undefined) {
  return [geo?.countryName, geo?.countryCode].filter(Boolean).join(" / ") || null;
}

function formatCoordinates(geo: GeoSnapshot | null | undefined) {
  if (typeof geo?.latitude !== "number" || typeof geo.longitude !== "number") {
    return null;
  }
  return `${geo.latitude.toFixed(4)}, ${geo.longitude.toFixed(4)}`;
}

function formatUnit(value: number | null | undefined, unit: string) {
  return typeof value === "number" ? `${value} ${unit}` : null;
}

function formatBoolean(
  value: boolean | null | undefined,
  t: ReturnType<typeof useTranslations>,
) {
  if (typeof value !== "boolean") {
    return null;
  }
  return value ? t("yes") : t("no");
}

function toValuePairs(values: string[]): HeaderPair[] {
  return values.map((value, index) => ({
    name: `#${index + 1}`,
    value,
  }));
}
