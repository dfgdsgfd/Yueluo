"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { useTranslations } from "next-intl";

export function AdminPublicShell({ children }: { children: ReactNode }) {
  const t = useTranslations("adminPortal");
  return (
    <div
      className="min-h-dvh bg-[linear-gradient(135deg,#f8fafc_0%,#ffffff_48%,#eef7ff_100%)] text-[#17171d]"
      translate="no"
    >
      <header className="mx-auto flex h-16 w-full max-w-[1180px] items-center justify-between px-4">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-semibold text-[#17171d]">
          <ArrowLeft className="size-4" />
          {t("layout.backHome")}
        </Link>
        <span className="rounded-full border border-black/[0.06] bg-white/70 px-3 py-1 text-xs text-[#777d89]">
          Admin
        </span>
      </header>
      {children}
    </div>
  );
}
