"use client";

import { useEffect, useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { X } from "lucide-react";
import { apiGet } from "@/lib/api";
import { cn } from "@/lib/utils";

type MaintenanceStatus = {
  enabled?: boolean;
  bypass?: boolean;
  message?: string;
  estimated_end_at?: string;
  now?: string;
  border_visible?: boolean;
  border_color?: string;
  border_opacity?: number;
  border_dismissible?: boolean;
};

export function MaintenanceGate() {
  const t = useTranslations("maintenance");
  const pathname = usePathname();
  const [status, setStatus] = useState<MaintenanceStatus | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [closedEdgeKey, setClosedEdgeKey] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const payload = await apiGet<MaintenanceStatus>("/api/maintenance/status", undefined, { auth: false });
        if (!cancelled) setStatus(payload);
      } catch {
        if (!cancelled) setStatus(null);
      }
    }
    void load();
    const timer = window.setInterval(() => void load(), 30000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const isInternalPath = pathname?.startsWith("/admin") || pathname?.startsWith("/service-mode");
  const countdown = useMemo(() => formatCountdown(status?.estimated_end_at, now), [now, status?.estimated_end_at]);
  const edgeKey = `${status?.border_visible}-${status?.border_color}-${status?.border_opacity}-${status?.border_dismissible}`;
  const edgesClosed = closedEdgeKey === edgeKey;

  if (!status?.enabled) return null;
  if (status.bypass) {
    if (status.border_visible === false || edgesClosed) return null;
    return <ServiceModeEdges color={status.border_color || "#dc2626"} opacity={Number(status.border_opacity ?? 1)} dismissible={Boolean(status.border_dismissible)} onClose={() => setClosedEdgeKey(edgeKey)} />;
  }
  if (isInternalPath) return null;

  return (
    <div className="fixed inset-0 z-[90] flex items-center justify-center bg-[#111217]/70 px-4 py-8 backdrop-blur-md">
      <div className="w-full max-w-[460px] overflow-hidden rounded-2xl border border-[#ef4444]/30 bg-white shadow-[0_24px_80px_rgba(0,0,0,0.30)]">
        <div className="border-b border-[#fee2e2] bg-[#fff1f2] px-5 py-4">
          <p className="text-xs font-black uppercase tracking-[0.18em] text-[#dc2626]">{t("serviceMode")}</p>
          <h2 className="mt-2 text-xl font-black text-[#17171d]">{t("title")}</h2>
          <p className="mt-2 text-sm leading-6 text-[#6b7280]">{status.message || t("defaultMessage")}</p>
        </div>
        <div className="grid gap-4 px-5 py-5">
          <div className="rounded-xl border border-black/[0.06] bg-[#f8fafc] px-4 py-3">
            <p className="text-xs font-semibold text-[#7a8495]">{t("estimatedRemaining")}</p>
            <p className="mt-1 text-3xl font-black tabular-nums text-[#dc2626]">{countdown || t("unknownTime")}</p>
          </div>
          <div className="grid grid-cols-3 gap-2">
            {["left", "center", "right"].map((item) => (
              <span key={item} className="h-2 rounded-full bg-[#ef4444]" />
            ))}
          </div>
          <p className="text-center text-xs leading-5 text-[#8a8f9d]">{t("entryHint")}</p>
        </div>
      </div>
    </div>
  );
}

function ServiceModeEdges({ color, opacity, dismissible, onClose }: { color: string; opacity: number; dismissible: boolean; onClose: () => void }) {
  const t = useTranslations("maintenance");
  return (
    <div className="pointer-events-none fixed inset-0 z-[80]">
      <ServiceModeStrip position="top" label={t("edgeLabel")} color={color} opacity={opacity} />
      <ServiceModeStrip position="bottom" label={t("edgeLabel")} color={color} opacity={opacity} />
      <ServiceModeStrip position="left" label={t("edgeLabel")} color={color} opacity={opacity} />
      <ServiceModeStrip position="right" label={t("edgeLabel")} color={color} opacity={opacity} />
      {dismissible ? (
        <button
          type="button"
          aria-label={t("closeEdge")}
          title={t("closeEdge")}
          onClick={onClose}
          className="pointer-events-auto fixed right-8 top-8 inline-flex size-8 items-center justify-center rounded-full bg-white/95 text-[#17171d] shadow-lg"
        >
          <X className="size-4" />
        </button>
      ) : null}
    </div>
  );
}

function ServiceModeStrip({ position, label, color, opacity }: { position: "top" | "bottom" | "left" | "right"; label: string; color: string; opacity: number }) {
  const vertical = position === "left" || position === "right";
  return (
    <div
      className={cn(
        "fixed flex items-center justify-center text-[11px] font-black uppercase tracking-[0.18em] text-white shadow-lg",
        position === "top" && "inset-x-0 top-0 h-6",
        position === "bottom" && "inset-x-0 bottom-0 h-6",
        position === "left" && "inset-y-0 left-0 w-6",
        position === "right" && "inset-y-0 right-0 w-6",
      )}
      style={{ backgroundColor: colorWithOpacity(color, opacity), boxShadow: `0 10px 24px ${colorWithOpacity(color, Math.min(opacity, 0.28))}` }}
    >
      <span className={vertical ? "rotate-90 whitespace-nowrap" : "whitespace-nowrap"}>{label}</span>
    </div>
  );
}

function colorWithOpacity(color: string, opacity: number) {
  const clamped = Math.max(0, Math.min(1, Number.isFinite(opacity) ? opacity : 1));
  const hex = color.trim();
  const short = /^#([0-9a-f]{3})$/i.exec(hex);
  if (short) {
    const [r, g, b] = short[1].split("").map((part) => parseInt(part + part, 16));
    return `rgba(${r}, ${g}, ${b}, ${clamped})`;
  }
  const long = /^#([0-9a-f]{6})$/i.exec(hex);
  if (long) {
    const raw = long[1];
    const r = parseInt(raw.slice(0, 2), 16);
    const g = parseInt(raw.slice(2, 4), 16);
    const b = parseInt(raw.slice(4, 6), 16);
    return `rgba(${r}, ${g}, ${b}, ${clamped})`;
  }
  return hex;
}

function formatCountdown(value: string | undefined, now: number) {
  if (!value) return "";
  const target = new Date(value).getTime();
  if (!Number.isFinite(target)) return "";
  const remaining = Math.max(0, target - now);
  const totalSeconds = Math.floor(remaining / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return [hours, minutes, seconds].map((part) => String(part).padStart(2, "0")).join(":");
}
