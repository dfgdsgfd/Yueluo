"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { Loader2, ShieldCheck, TriangleAlert } from "lucide-react";
import { apiPost, storeOAuthCallbackTokens } from "@/lib/api";
import type { AuthUser } from "@/lib/types";

type MaintenanceEnterPayload = {
  enabled?: boolean;
  bypass?: boolean;
  user?: AuthUser;
  tokens?: {
    access_token?: string;
    refresh_token?: string;
  };
};

export function ServiceModeEntry({ code }: { code: string }) {
  const t = useTranslations("maintenance.entry");
  const router = useRouter();
  const [state, setState] = useState<"loading" | "success" | "error">("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function enter() {
      try {
        const payload = await apiPost<MaintenanceEnterPayload>("/api/maintenance/enter", { code }, { auth: false });
        if (payload.tokens?.access_token && payload.tokens.refresh_token) {
          storeOAuthCallbackTokens({
            accessToken: payload.tokens.access_token,
            refreshToken: payload.tokens.refresh_token,
            user: payload.user,
          });
        }
        if (cancelled) return;
        setState("success");
        setMessage(t("success"));
        window.setTimeout(() => router.replace("/explore"), 900);
      } catch (error) {
        if (cancelled) return;
        setState("error");
        setMessage(error instanceof Error ? error.message : t("error"));
      }
    }
    void enter();
    return () => {
      cancelled = true;
    };
  }, [code, router, t]);

  const Icon = state === "error" ? TriangleAlert : state === "success" ? ShieldCheck : Loader2;

  return (
    <main className="flex min-h-dvh items-center justify-center bg-[#111217] px-4 text-white">
      <section className="w-full max-w-[420px] rounded-2xl border border-[#ef4444]/40 bg-[#181a21] p-6 shadow-[0_24px_80px_rgba(220,38,38,0.20)]">
        <div className="flex items-center gap-3">
          <span className="flex size-12 items-center justify-center rounded-xl bg-[#dc2626] text-white">
            <Icon className={state === "loading" ? "size-5 animate-spin" : "size-5"} />
          </span>
          <div className="min-w-0">
            <p className="text-xs font-black uppercase tracking-[0.2em] text-[#fca5a5]">{t("eyebrow")}</p>
            <h1 className="mt-1 text-xl font-black">{t("title")}</h1>
          </div>
        </div>
        <p className="mt-5 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm leading-6 text-[#d1d5db]">{message || t("checking")}</p>
      </section>
    </main>
  );
}
