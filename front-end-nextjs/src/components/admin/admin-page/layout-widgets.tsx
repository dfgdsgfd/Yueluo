"use client";
import Link from "next/link";
import {
  useTranslations
} from "next-intl";
import type {
  ReactNode
} from "react";
import type {
  LucideIcon
} from "lucide-react";
import {
  Activity,
  FileText,
  Home,
  Loader2,
  LogOut,
  Menu,
  RefreshCw,
  Search,
  ShieldCheck,
  X
} from "lucide-react";
import {
  Button
} from "@/components/ui/button";
import type {
  AdminDashboardHotContentItem,
  AdminDashboardTrendsPayload,
  AdminUser
} from "@/lib/types";
import {
  cn
} from "@/lib/utils";
import {
  AdminSection,
  AdminView,
  Tone
} from "./types";
import {
  EmptyBlock
} from "./resource-editor";
import {
  Avatar,
  Thumbnail
} from "./resource-cells";
import {
  ArrowLeftIcon,
  formatCompact,
  sameView,
  toneDotClass,
  toneSoftClass,
  toneTextClass
} from "./helpers";

export function AdminSidebar({
  activeView,
  onChange,
  mobileOpen,
  onClose,
  sections,
}: {
  activeView: AdminView;
  onChange: (view: AdminView) => void;
  mobileOpen: boolean;
  onClose: () => void;
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
            <p className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#a0a5b1]">{section.label}</p>
            <div className="grid gap-1">
              {section.items.map((item) => {
                const active = item.view ? sameView(activeView, item.view) : false;
                const Icon = item.icon;
                const itemClassName = cn(
                  "flex h-10 min-w-0 items-center gap-3 rounded-lg px-3 text-left text-sm font-medium transition",
                  active ? "bg-[#1d4ed8] text-white shadow-md shadow-[#1d4ed8]/20" : "text-[#555b66] hover:bg-[#f4f5f8] hover:text-[#17171d]",
                );
                if (typeof item.href === "string") {
                  return (
                    <Link key={item.id} href={item.href} className={itemClassName} onClick={onClose}>
                      <Icon className="size-4 shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </Link>
                  );
                }
                if (!item.view) {
                  return null;
                }
                const view = item.view;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => onChange(view)}
                    className={itemClassName}
                  >
                    <Icon className="size-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </button>
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
          <button type="button" aria-label={t("layout.closeMenuOverlay")} className="absolute inset-0 bg-black/25" onClick={onClose} />
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
          <Avatar label={admin.username} />
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


export function AdminPublicShell({ children }: { children: ReactNode }) {
  const t = useTranslations("adminPortal");
  return (
    <div className="min-h-dvh bg-[linear-gradient(135deg,#f8fafc_0%,#ffffff_48%,#eef7ff_100%)] text-[#17171d]" translate="no">
      <header className="mx-auto flex h-16 w-full max-w-[1180px] items-center justify-between px-4">
        <Link href="/" className="inline-flex items-center gap-2 text-sm font-semibold text-[#17171d]">
          <ArrowLeftIcon />
          {t("layout.backHome")}
        </Link>
        <span className="rounded-full border border-black/[0.06] bg-white/70 px-3 py-1 text-xs text-[#777d89]">Admin</span>
      </header>
      {children}
    </div>
  );
}


export function Panel({ title, icon: Icon, action, children }: { title: string; icon: LucideIcon; action?: ReactNode; children: ReactNode }) {
  return (
    <section className="min-w-0 rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_34px_rgba(20,20,35,0.05)]">
      <div className="mb-4 flex min-w-0 items-center gap-3">
        <span className="flex size-9 items-center justify-center rounded-lg bg-[#f6f7fb] text-[#30333b]">
          <Icon className="size-4" />
        </span>
        <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-[#17171d]">{title}</h2>
        {action}
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
}


export function HeaderCard({ icon: Icon, title, description, tone }: { icon: LucideIcon; title: string; description: string; tone: Tone }) {
  return (
    <section className="rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_34px_rgba(20,20,35,0.05)] sm:p-5">
      <div className="flex min-w-0 items-center gap-3">
        <span className={cn("flex size-11 items-center justify-center rounded-lg", toneSoftClass(tone))}>
          <Icon className="size-5" />
        </span>
        <div className="min-w-0">
          <h1 className="truncate text-xl font-semibold text-[#17171d]">{title}</h1>
          <p className="truncate text-sm text-[#777d89]">{description}</p>
        </div>
      </div>
    </section>
  );
}


export function MetricCard({ metric }: { metric: { label: string; value: number; delta?: number; tone?: Tone } }) {
  return (
    <article className="rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_34px_rgba(20,20,35,0.05)]">
      <div className="mb-3 flex items-center justify-between gap-2">
        <span className="truncate text-sm font-semibold text-[#666c78]">{metric.label}</span>
        <span className={cn("size-2.5 rounded-full", toneDotClass(metric.tone ?? "red"))} />
      </div>
      <p className="text-xl font-semibold text-[#17171d] sm:text-2xl">{formatCompact(metric.value)}</p>
      <p className={cn("mt-1 text-xs", (metric.delta ?? 0) >= 0 ? "text-[#18a058]" : "text-[#d71935]")}>
        较昨日 {(metric.delta ?? 0) >= 0 ? "+" : ""}{metric.delta ?? 0}%
      </p>
    </article>
  );
}


export function MetricTile({ label, value, tone }: { label: string; value: ReactNode; tone: Tone }) {
  return (
    <article className="rounded-lg border border-black/[0.06] bg-white p-4 shadow-[0_12px_34px_rgba(20,20,35,0.05)]">
      <p className="text-sm font-semibold text-[#666c78]">{label}</p>
      <p className={cn("mt-2 text-xl font-semibold sm:text-2xl", toneTextClass(tone))}>{typeof value === "number" ? formatCompact(value) : value}</p>
    </article>
  );
}


export function HeroPill({ label, value, tone }: { label: string; value: ReactNode; tone: Tone }) {
  return (
    <div className="rounded-lg border border-white/70 bg-white/78 p-3 shadow-sm">
      <p className="text-xs text-[#7b808c]">{label}</p>
      <p className={cn("mt-1 truncate text-lg font-semibold", toneTextClass(tone))}>{value}</p>
    </div>
  );
}


export function ToggleCard({ icon: Icon, label, description, active, loading, onClick }: { icon: LucideIcon; label: string; description: string; active: boolean; loading: boolean; onClick: () => void }) {
  return (
    <div className="rounded-lg border border-black/[0.05] bg-[#fafbfe] p-3">
      <div className="flex items-center gap-3">
        <span className={cn("flex size-9 items-center justify-center rounded-lg", active ? "bg-[#e8fff2] text-[#16824a]" : "bg-[#f0f2f6] text-[#747b87]")}>
          <Icon className="size-4" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-[#30333b]">{label}</p>
          <p className="truncate text-xs text-[#7b808c]">{description}</p>
        </div>
        <Button type="button" variant="outline" onClick={onClick} disabled={loading} className="h-9 rounded-lg border-black/[0.08] bg-white px-3">
          <span className="inline-flex min-w-8 items-center justify-center">
            {loading ? <Loader2 className="size-4 animate-spin" /> : active ? "开启" : "关闭"}
          </span>
        </Button>
      </div>
    </div>
  );
}


export function TrendChart({ trends }: { trends: AdminDashboardTrendsPayload | null }) {
  const labels = trends?.labels ?? [];
  const series = [
    { key: "users", label: "用户", color: "#1d4ed8", values: trends?.users ?? [] },
    { key: "posts", label: "内容", color: "#2f7df6", values: trends?.posts ?? [] },
    { key: "comments", label: "评论", color: "#18a058", values: trends?.comments ?? [] },
    { key: "reports", label: "举报", color: "#f59e0b", values: trends?.reports ?? [] },
  ];
  const max = Math.max(1, ...series.flatMap((item) => item.values));

  if (!labels.length) {
    return <EmptyBlock icon={Activity} label="暂无趋势数据" />;
  }

  return (
    <div className="min-w-0">
      <div className="mb-3 flex flex-wrap gap-3">
        {series.map((item) => (
          <span key={item.key} className="inline-flex items-center gap-1 text-xs text-[#6b7180]">
            <span className="size-2 rounded-full" style={{ backgroundColor: item.color }} />
            {item.label}
          </span>
        ))}
      </div>
      <svg viewBox="0 0 720 260" className="h-[260px] w-full overflow-visible">
        {[0, 1, 2, 3, 4].map((line) => (
          <line key={line} x1="36" x2="704" y1={24 + line * 48} y2={24 + line * 48} stroke="rgba(20,20,30,0.07)" />
        ))}
        {series.map((item) => (
          <polyline
            key={item.key}
            fill="none"
            stroke={item.color}
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="3"
            points={item.values.map((value, index) => `${36 + (index * 668) / Math.max(1, item.values.length - 1)},${232 - (Number(value) / max) * 192}`).join(" ")}
          />
        ))}
        {labels.map((label, index) => (
          <text key={label} x={36 + (index * 668) / Math.max(1, labels.length - 1)} y="254" textAnchor="middle" className="fill-[#8b919e] text-[11px]">
            {label.slice(5)}
          </text>
        ))}
      </svg>
    </div>
  );
}


export function HotContentList({ items }: { items: AdminDashboardHotContentItem[] }) {
  if (!items.length) {
    return <EmptyBlock icon={FileText} label="暂无热门内容" />;
  }
  return (
    <div className="grid gap-2">
      {items.map((item, index) => (
        <article key={item.id} className="flex min-w-0 items-center gap-3 rounded-lg border border-black/[0.05] bg-[#fafbfe] p-3">
          <span className={cn("flex size-7 shrink-0 items-center justify-center rounded-lg text-xs font-semibold", index < 3 ? "bg-[#1d4ed8] text-white" : "bg-white text-[#6b7180]")}>
            {index + 1}
          </span>
          <Thumbnail url={item.cover_url} />
          <div className="min-w-0 flex-1">
            <h3 className="truncate text-sm font-semibold text-[#252932]">{item.title || `内容 ${item.id}`}</h3>
            <p className="truncate text-xs text-[#7b808c]">{item.nickname || item.user_display_id || "未知作者"}</p>
          </div>
          <span className="text-xs font-semibold text-[#1d4ed8]">{formatCompact(item.heat ?? item.like_count ?? 0)}</span>
        </article>
      ))}
    </div>
  );
}
