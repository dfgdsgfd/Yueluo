"use client";

import type { ReactNode } from "react";
import { Loader2, type LucideIcon } from "lucide-react";
import { useTranslations } from "next-intl";
import type { HeaderPair } from "@/lib/network-diagnostics";
import { cn } from "@/lib/utils";
import type {
  WebRtcAddressExposure,
  WebRtcAddressKind,
} from "./network-probes";

export type MetricTone = "amber" | "emerald" | "rose" | "sky" | "violet";

export function MetricCard({
  detail,
  icon: Icon,
  label,
  loading,
  tone,
  value,
}: {
  detail?: string | null;
  icon: LucideIcon;
  label: string;
  loading: boolean;
  tone: MetricTone;
  value: string;
}) {
  return (
    <article className="min-w-0 rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
      <div className="flex items-center justify-between gap-3">
        <p className="truncate text-xs font-bold uppercase tracking-normal text-white/42">{label}</p>
        <span className={cn("flex size-9 shrink-0 items-center justify-center rounded-full", toneClassName(tone))}>
          <Icon className="size-4" />
        </span>
      </div>
      <div className="mt-4 min-h-12">
        {loading ? (
          <Loader2 className="size-5 animate-spin text-white/45" />
        ) : (
          <>
            <p className="break-words text-xl font-black leading-tight text-white">{value}</p>
            <p className="mt-2 truncate text-xs text-white/42">{detail}</p>
          </>
        )}
      </div>
    </article>
  );
}

export function SectionCard({
  children,
  icon: Icon,
  title,
}: {
  children: ReactNode;
  icon: LucideIcon;
  title: string;
}) {
  return (
    <section className="rounded-[8px] border border-white/[0.08] bg-white/[0.06] p-4">
      <div className="mb-4 flex items-center gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-white/[0.07] text-white/65">
          <Icon className="size-5" />
        </span>
        <h2 className="min-w-0 truncate text-base font-bold text-white">{title}</h2>
      </div>
      {children}
    </section>
  );
}

export function DetailGrid({
  rows,
}: {
  rows: Array<[string, string | number | null | undefined]>;
}) {
  return (
    <dl className="grid gap-2 sm:grid-cols-2">
      {rows.map(([label, value]) => (
        <div key={label} className="min-w-0 rounded-[8px] bg-black/20 px-3 py-2">
          <dt className="text-[11px] font-bold uppercase tracking-normal text-white/35">{label}</dt>
          <dd className="mt-1 break-words text-sm font-semibold text-white/82">{formatNullable(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

export function HeaderList({
  headers,
  title,
}: {
  headers: HeaderPair[];
  title: string;
}) {
  const t = useTranslations("networkDiagnostics");

  return (
    <div>
      <h3 className="mb-2 text-xs font-bold uppercase tracking-normal text-white/38">{title}</h3>
      {headers.length > 0 ? (
        <div className="overflow-hidden rounded-[8px] border border-white/[0.07]">
          {headers.map((header, index) => (
            <div
              key={`${header.name}-${index}`}
              className={cn(
                "grid gap-1 px-3 py-2 text-xs sm:grid-cols-[180px_minmax(0,1fr)]",
                index > 0 && "border-t border-white/[0.06]",
              )}
            >
              <code className="min-w-0 truncate text-white/45">{header.name}</code>
              <code className="min-w-0 break-words text-white/78">{header.value}</code>
            </div>
          ))}
        </div>
      ) : (
        <p className="rounded-[8px] border border-dashed border-white/[0.1] px-3 py-3 text-sm text-white/38">
          {t("empty")}
        </p>
      )}
    </div>
  );
}

export function CandidateList({
  addresses,
  supported,
}: {
  addresses: WebRtcAddressExposure[];
  supported: boolean;
}) {
  const t = useTranslations("networkDiagnostics");

  if (!supported) {
    return (
      <p className="rounded-[8px] border border-dashed border-white/[0.1] px-3 py-3 text-sm text-white/42">
        {t("privacy.webrtcUnsupported")}
      </p>
    );
  }

  if (addresses.length === 0) {
    return (
      <p className="rounded-[8px] border border-dashed border-white/[0.1] px-3 py-3 text-sm text-white/42">
        {t("privacy.webrtcNoCandidates")}
      </p>
    );
  }

  return (
    <div className="overflow-hidden rounded-[8px] border border-white/[0.07]">
      {addresses.map((item, index) => (
        <div
          key={`${item.address}-${index}`}
          className={cn(
            "grid gap-2 px-3 py-2 text-xs sm:grid-cols-[minmax(0,1fr)_110px_90px]",
            index > 0 && "border-t border-white/[0.06]",
          )}
        >
          <code className="break-words text-white/82">{item.address}</code>
          <span className={cn("font-bold", addressKindClassName(item.kind))}>
            {t(`webrtcKinds.${item.kind}`)}
          </span>
          <span className="text-white/42">{t(`candidateSources.${item.source}`)}</span>
        </div>
      ))}
    </div>
  );
}

export function LoadingBlock({
  checking,
  unavailableLabel,
}: {
  checking: boolean;
  unavailableLabel?: string;
}) {
  const t = useTranslations("networkDiagnostics");

  return (
    <div className="flex min-h-24 items-center justify-center rounded-[8px] border border-dashed border-white/[0.1] text-sm text-white/45">
      {checking ? (
        <>
          <Loader2 className="mr-2 size-4 animate-spin" />
          {t("checking")}
        </>
      ) : (
        unavailableLabel ?? t("unknown")
      )}
    </div>
  );
}

function toneClassName(tone: MetricTone) {
  switch (tone) {
    case "amber":
      return "bg-amber-300/12 text-amber-100";
    case "emerald":
      return "bg-emerald-300/12 text-emerald-100";
    case "rose":
      return "bg-rose-300/12 text-rose-100";
    case "violet":
      return "bg-violet-300/12 text-violet-100";
    case "sky":
      return "bg-sky-300/12 text-sky-100";
  }
}

function addressKindClassName(kind: WebRtcAddressKind) {
  switch (kind) {
    case "public":
      return "text-amber-100";
    case "private":
      return "text-sky-100";
    case "mdns":
      return "text-emerald-100";
    case "loopback":
      return "text-white/45";
    case "unknown":
      return "text-white/60";
  }
}

function formatNullable(value: string | number | null | undefined) {
  return value === null || value === undefined || value === "" ? "-" : String(value);
}
