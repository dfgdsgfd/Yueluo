"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";

import { StatusPill } from "./resource-cells";

type TranslationFn = (key: string, values?: Record<string, number | string>) => string;

export function FileRecycleRemainingCell({
  purgeAfter,
  status,
}: {
  purgeAfter: unknown;
  status: unknown;
}) {
  const t = useTranslations("adminPortal.fileRecycleBin") as TranslationFn;
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  if (String(status ?? "") === "purged") {
    return <span className="text-xs font-semibold text-[#8b919e]">{t("countdown.purged")}</span>;
  }
  if (!purgeAfter || typeof purgeAfter !== "string") return <span className="text-[#8b919e]">-</span>;
  const target = new Date(purgeAfter).getTime();
  if (!Number.isFinite(target)) return <span className="text-[#8b919e]">-</span>;
  const diff = target - now;
  if (diff <= 0) {
    return <StatusPill value={t("countdown.due")} tone="red" />;
  }
  return <span className="text-sm font-semibold text-[#b45309]">{formatFileRecycleRemaining(diff, t)}</span>;
}

function formatFileRecycleRemaining(ms: number, t: TranslationFn) {
  const totalMinutes = Math.max(1, Math.ceil(ms / 60_000));
  const days = Math.floor(totalMinutes / 1440);
  const hours = Math.floor((totalMinutes % 1440) / 60);
  const minutes = totalMinutes % 60;
  const parts: string[] = [];
  if (days > 0) parts.push(t("countdown.day", { count: days }));
  if (hours > 0 && parts.length < 2) parts.push(t("countdown.hour", { count: hours }));
  if (parts.length === 0 || (minutes > 0 && parts.length < 2)) {
    parts.push(t("countdown.minute", { count: minutes }));
  }
  return t("countdown.remaining", { value: parts.join(t("countdown.separator")) });
}
