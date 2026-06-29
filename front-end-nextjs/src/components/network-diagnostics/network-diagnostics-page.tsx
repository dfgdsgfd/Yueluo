"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  BrowserNetworkSection,
  DiagnosticsHeader,
  GeoSection,
  PrivacySection,
  ProbeErrorsSection,
  ServerSection,
  SummaryGrid,
  TraceSection,
} from "./network-diagnostics-widgets";
import {
  runNetworkDiagnostics,
  type NetworkDiagnosticsSnapshot,
} from "./network-probes";
import { OriginServerSection } from "./network-origin-section";

export function NetworkDiagnosticsPage() {
  const t = useTranslations("networkDiagnostics");
  const [checking, setChecking] = useState(true);
  const [copied, setCopied] = useState(false);
  const [snapshot, setSnapshot] = useState<NetworkDiagnosticsSnapshot | null>(null);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const nextSnapshot = await runNetworkDiagnostics();
        if (!cancelled) {
          setSnapshot(nextSnapshot);
        }
      } catch (error) {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("errors.unexpected"));
        }
      } finally {
        if (!cancelled) {
          setChecking(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [t]);

  const refresh = useCallback(async () => {
    setChecking(true);
    setCopied(false);
    try {
      setSnapshot(await runNetworkDiagnostics());
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("errors.unexpected"));
    } finally {
      setChecking(false);
    }
  }, [t]);

  const copySnapshot = useCallback(async () => {
    if (!snapshot) {
      return;
    }
    try {
      await navigator.clipboard.writeText(JSON.stringify(snapshot, null, 2));
      setCopied(true);
      toast.success(t("copied"));
      window.setTimeout(() => {
        setCopied(false);
      }, 1600);
    } catch {
      toast.error(t("copyFailed"));
    }
  }, [snapshot, t]);

  return (
    <main className="theme-adaptive min-h-dvh bg-[#121212] text-white">
      <DiagnosticsHeader
        checking={checking}
        copied={copied}
        hasSnapshot={Boolean(snapshot)}
        online={snapshot?.browser.online ?? null}
        onCopy={copySnapshot}
        onRefresh={() => void refresh()}
      />

      <div className="mx-auto w-full max-w-[1180px] px-4 pb-[calc(28px+env(safe-area-inset-bottom))] pt-4">
        <section className="mb-4">
          <p className="text-sm font-bold uppercase tracking-normal text-sky-200/70">
            {t("eyebrow")}
          </p>
          <h2 className="mt-2 max-w-3xl text-2xl font-black leading-tight text-white sm:text-3xl">
            {t("headline")}
          </h2>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-white/52">
            {t("subtitle")}
          </p>
        </section>

        <div className="space-y-4">
          <SummaryGrid checking={checking} snapshot={snapshot} />

          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
            <div className="min-w-0 space-y-4">
              <GeoSection checking={checking} snapshot={snapshot} />
              <ServerSection checking={checking} snapshot={snapshot} />
              <OriginServerSection checking={checking} snapshot={snapshot} />
              <PrivacySection checking={checking} snapshot={snapshot} />
            </div>
            <aside className="min-w-0 space-y-4">
              <BrowserNetworkSection checking={checking} snapshot={snapshot} />
              <TraceSection checking={checking} snapshot={snapshot} />
              <ProbeErrorsSection errors={snapshot?.errors ?? []} />
            </aside>
          </div>
        </div>
      </div>
    </main>
  );
}
