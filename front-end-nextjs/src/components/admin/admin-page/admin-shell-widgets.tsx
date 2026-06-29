"use client";

import Link from "next/link";
import { Home, LogOut, Menu, RefreshCw, Search, ShieldCheck, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import type { AdminUser } from "@/lib/types";
import { cn } from "@/lib/utils";
import type { AdminSection, AdminView } from "./types";

export function AdminSidebar({
  activeView,
  getViewHref,
  mobileOpen,
  onClose,
  onNavigate,
  sections,
}: {
  activeView: AdminView;
  getViewHref: (view: AdminView) => string;
  mobileOpen: boolean;
  onClose: () => void;
  onNavigate: () => void;
  sections: AdminSection[];
}) {
  const t = useTranslations("adminPortal");
  const sidebar = (
    <aside className="flex h-full min-h-0 w-[280px] flex-col border-r border-black/[0.06] bg-white/86 backdrop-blur-xl">
      <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
        <span className="flex size-10 items-center justify-center rounded-lg bg-[#1d4ed8] text-white shadow-lg shadow-[#1d4ed8]/20">
          <ShieldCheck className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-sm font-semibold text-[#17171d]">{t("brand.name")}</h2>
          <p className="truncate text-xs text-[#8a8f9d]">{t("brand.subtitle")}</p>
        </div>
        <button type="button" className="lg:hidden" onClick={onClose} aria-label={t("layout.closeMenu")}>
          <X className="size-5" />
        </button>
      </div>
      <nav className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
        {sections.map((section) => (
          <div key={section.id} className="mb-5">
            <p className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#a0a5b1]">
              {section.label}
            </p>
            <div className="grid gap-1">
              {section.items.map((item) => {
                const active = item.view ? sameView(activeView, item.view) : false;
                const Icon = item.icon;
                const itemClassName = cn(
                  "flex h-10 min-w-0 items-center gap-3 rounded-lg px-3 text-left text-sm font-medium transition",
                  active
                    ? "bg-[#1d4ed8] text-white shadow-md shadow-[#1d4ed8]/20"
                    : "text-[#555b66] hover:bg-[#f4f5f8] hover:text-[#17171d]",
                );
                if (typeof item.href === "string") {
                  return (
                    <Link key={item.id} href={item.href} className={itemClassName} onClick={onNavigate}>
                      <Icon className="size-4 shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </Link>
                  );
                }
                if (!item.view) {
                  return null;
                }
                const nextView = item.view;
                return (
                  <Link
                    key={item.id}
                    href={getViewHref(nextView)}
                    scroll={false}
                    onClick={onNavigate}
                    className={itemClassName}
                  >
                    <Icon className="size-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>
      <div className="shrink-0 border-t border-black/[0.06] p-3">
        <Button asChild variant="outline" className="h-10 w-full rounded-lg border-black/[0.08] bg-[#fafbfe] text-[#4b515c] hover:bg-white">
          <Link href="/">
            <Home className="size-4" />
            {t("layout.backToSite")}
          </Link>
        </Button>
      </div>
    </aside>
  );

  return (
    <>
      <div className="fixed inset-y-0 left-0 z-40 hidden lg:block">{sidebar}</div>
      {mobileOpen ? (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button
            type="button"
            aria-label={t("layout.closeMenuOverlay")}
            className="absolute inset-0 bg-black/25"
            onClick={onClose}
          />
          <div className="absolute inset-y-0 left-0">{sidebar}</div>
        </div>
      ) : null}
    </>
  );
}

export function AdminTopbar({
  admin,
  title,
  onMenu,
  onRefresh,
  onLogout,
}: {
  admin: AdminUser;
  title: string;
  onMenu: () => void;
  onRefresh: () => void;
  onLogout: () => void;
}) {
  const t = useTranslations("adminPortal");
  return (
    <header className="sticky top-0 z-30 border-b border-black/[0.06] bg-white/82 backdrop-blur-xl">
      <div className="flex h-16 min-w-0 items-center gap-3 px-3 sm:px-5 lg:px-6 xl:px-8">
        <Button type="button" variant="ghost" size="icon" onClick={onMenu} className="size-10 rounded-lg text-[#30333b] hover:bg-[#f2f3f6] lg:hidden" aria-label={t("layout.openMenu")}>
          <Menu className="size-5" />
        </Button>
        <div className="min-w-0 flex-1">
          <p className="truncate text-xs text-[#8a8f9d]">{t("layout.breadcrumb", { title })}</p>
          <h1 className="truncate text-base font-semibold text-[#17171d] sm:text-lg">{title}</h1>
        </div>
        <label className="relative hidden min-w-0 flex-1 md:block md:max-w-[420px]">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#9aa0ad]" />
          <input
            className="h-10 w-full rounded-lg border border-black/[0.06] bg-[#f6f7fb] pl-9 pr-3 text-sm outline-none transition focus:border-[#1d4ed8] focus:bg-white focus:ring-4 focus:ring-[#1d4ed8]/10"
            placeholder={t("layout.searchPlaceholder")}
          />
        </label>
        <Button type="button" variant="outline" size="icon" onClick={onRefresh} className="size-10 rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]" aria-label={t("layout.refresh")}>
          <RefreshCw className="size-4" />
        </Button>
        <div className="hidden items-center gap-2 rounded-lg border border-black/[0.06] bg-[#fafbfe] px-2 py-1.5 sm:flex">
          <span className="flex size-8 items-center justify-center rounded-full bg-[#1d4ed8]/10 text-xs font-semibold text-[#1d4ed8]">
            {admin.username.charAt(0).toUpperCase()}
          </span>
          <div className="min-w-0">
            <p className="max-w-[110px] truncate text-xs font-semibold text-[#17171d]">{admin.username}</p>
            <p className="text-[11px] text-[#8a8f9d]">{t("layout.adminRole")}</p>
          </div>
        </div>
        <Button type="button" variant="outline" size="icon" onClick={onLogout} className="size-10 rounded-lg border-black/[0.08] bg-white hover:bg-[#fff5f6]" aria-label={t("layout.logout")}>
          <LogOut className="size-4" />
        </Button>
      </div>
    </header>
  );
}

function sameView(a: AdminView, b: AdminView) {
  if (a.kind !== b.kind) {
    return false;
  }
  return a.kind !== "resource" || (b.kind === "resource" && a.resource === b.resource);
}
