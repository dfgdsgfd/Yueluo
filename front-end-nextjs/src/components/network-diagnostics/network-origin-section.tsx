"use client";

import { Server } from "lucide-react";
import { useTranslations } from "next-intl";
import type {
  HeaderPair,
  NetworkCdnProvider,
} from "@/lib/network-diagnostics";
import type { NetworkDiagnosticsSnapshot } from "./network-probes";
import {
  DetailGrid,
  HeaderList,
  LoadingBlock,
  SectionCard,
} from "./network-diagnostics-primitives";

type OriginServerSectionProps = {
  checking: boolean;
  snapshot: NetworkDiagnosticsSnapshot | null;
};

export function OriginServerSection({ checking, snapshot }: OriginServerSectionProps) {
  const t = useTranslations("networkDiagnostics");
  const origin = snapshot?.originServer;

  return (
    <SectionCard icon={Server} title={t("sections.originServer")}>
      {origin ? (
        <div className="space-y-5">
          <DetailGrid
            rows={[
              [t("labels.originLocalAddress"), origin.connection.local.address],
              [t("labels.originLocalIp"), origin.connection.local.ip],
              [t("labels.originLocalPort"), origin.connection.local.port],
              [t("labels.originRemoteAddress"), origin.connection.remote.address],
              [t("labels.originRemoteIp"), origin.connection.remote.ip],
              [t("labels.originRemotePort"), origin.connection.remote.port],
              [t("labels.originClientIp"), origin.clientIp],
              [t("labels.originMethod"), origin.request.method],
              [t("labels.originForwardedMethod"), origin.forwarded.method],
              [t("labels.originHost"), origin.request.host],
              [t("labels.originForwardedHost"), origin.forwarded.host],
              [t("labels.originPath"), origin.request.path],
              [t("labels.originRequestUri"), origin.request.requestUri],
              [t("labels.originForwardedUri"), origin.forwarded.uri],
              [t("labels.originScheme"), origin.request.scheme],
              [t("labels.originForwardedProtocol"), origin.forwarded.protocol],
              [t("labels.originForwardedPort"), origin.forwarded.port],
              [t("labels.originHttpProtocol"), origin.request.protocol],
              [t("labels.originTls"), formatBoolean(origin.request.tls, t)],
              [t("labels.originTlsVersion"), origin.request.tlsVersion],
              [t("labels.originTlsCipher"), origin.request.tlsCipher],
              [t("labels.originHostname"), origin.server.hostname],
              [t("labels.originCdnProvider"), cdnProviderLabel(origin.cdn.provider, t)],
              [t("labels.cdnCountry"), origin.cdn.countryCode],
              [t("labels.cdnRay"), origin.cdn.rayId],
              [t("labels.timestamp"), origin.timestamp],
            ]}
          />
          <HeaderList
            title={t("labels.originForwardedChain")}
            headers={toValuePairs(origin.forwarded.chain)}
          />
          <HeaderList title={t("labels.originForwardedHeaders")} headers={origin.headers.forwarded} />
          <HeaderList title={t("labels.originCdnHeaders")} headers={origin.headers.cdn} />
          <HeaderList title={t("labels.originDiagnosticHeaders")} headers={origin.headers.diagnostic} />
        </div>
      ) : (
        <LoadingBlock checking={checking} unavailableLabel={t("originUnavailable")} />
      )}
    </SectionCard>
  );
}

function cdnProviderLabel(
  provider: NetworkCdnProvider | null | undefined,
  t: ReturnType<typeof useTranslations>,
) {
  return provider ? t(`cdnProviders.${provider}`) : t("notDetected");
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
