"use client";

import Link from "next/link";
import { Menu } from "lucide-react";
import { useTranslations } from "next-intl";
import type { FeedMode } from "@/lib/types";
import type { UserToolbarItem } from "@/lib/types";
import type { SiteProfile } from "@/lib/types/site";
import { cn } from "@/lib/utils";
import {
  bottomNavItems,
  desktopNavItems,
  toolbarIconMap,
  type FollowingSort,
  type MobileMainView,
} from "./explore-config";
import { NavIconWithBadge } from "./explore-widgets";

type WarmNavigationTarget = (key: string, href: string | null) => void;

export function ExploreDesktopSidebar({
  messageBadgeCount,
  mobileMoreOpen,
  onOpenMore,
  siteProfile,
  toolbarItems,
  visibleItems,
  warmNavigationTarget,
}: {
  messageBadgeCount: number;
  mobileMoreOpen: boolean;
  onOpenMore: () => void;
  siteProfile: SiteProfile;
  toolbarItems: UserToolbarItem[];
  visibleItems: Array<(typeof desktopNavItems)[number]>;
  warmNavigationTarget: WarmNavigationTarget;
}) {
  const t = useTranslations();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-[164px] bg-[var(--explore-sidebar)] px-4 py-8 lg:block">
      <div className="mb-10 flex min-h-[52px] items-center pl-1">
        <ExploreSiteBrand siteProfile={siteProfile} />
      </div>

      <nav aria-label={t("nav.home")} className="space-y-2">
        {visibleItems.map(({ key, icon: Icon, href }, index) => (
          <div key={key}>
            <Link
              href={href}
              transitionTypes={["nav-forward"]}
              aria-current={index === 0 ? "page" : undefined}
              onFocus={() => warmNavigationTarget(key, href)}
              onMouseEnter={() => warmNavigationTarget(key, href)}
              onTouchStart={() => warmNavigationTarget(key, href)}
              className={cn(
                "flex h-12 w-[132px] items-center gap-3 rounded-full px-4 text-base font-semibold text-[var(--explore-muted)] transition-[background-color,color,transform] duration-200 ease-out hover:bg-[var(--explore-surface-soft)] hover:text-[var(--explore-strong)] active:scale-[0.98]",
                index === 0 && "bg-[var(--explore-surface-soft)] text-[var(--explore-strong)]",
              )}
            >
              <NavIconWithBadge
                Icon={Icon}
                count={key === "messages" ? messageBadgeCount : 0}
                className="size-6"
                strokeWidth={2.1}
              />
              <span className="truncate">
                {key === "videoCenter" ? t("tabs.videoCenter") : t(`nav.${key}`)}
              </span>
            </Link>
          </div>
        ))}
      </nav>

      {toolbarItems.length > 0 ? (
        <nav aria-label={t("nav.more")} className="mt-6 space-y-2 border-t border-[var(--explore-border)] pt-4">
          {toolbarItems.map((item) => {
            const href = item.url || "#";
            const external = /^[a-z][a-z\d+\-.]*:/i.test(href);
            const Icon = toolbarIconMap[(item.icon ?? "link") as keyof typeof toolbarIconMap] ?? toolbarIconMap.link;
            const className =
              "flex h-10 w-[132px] items-center gap-3 rounded-full px-4 text-sm font-semibold text-[var(--explore-muted)] transition-[background-color,color,transform] duration-200 ease-out hover:bg-[var(--explore-surface-soft)] hover:text-[var(--explore-strong)] active:scale-[0.98]";
            const content = (
              <>
                <Icon className="size-5 shrink-0" />
                <span className="truncate">{item.name}</span>
              </>
            );
            return external ? (
              <a key={item.id} href={href} target="_blank" rel="noreferrer" className={className}>
                {content}
              </a>
            ) : (
              <Link
                key={item.id}
                href={href}
                onFocus={() => warmNavigationTarget(`toolbar-${item.id}`, href)}
                onMouseEnter={() => warmNavigationTarget(`toolbar-${item.id}`, href)}
                onTouchStart={() => warmNavigationTarget(`toolbar-${item.id}`, href)}
                className={className}
              >
                {content}
              </Link>
            );
          })}
        </nav>
      ) : null}

      <div className="absolute bottom-7 left-4 right-4 space-y-2 text-[var(--explore-subtle)]">
        <button
          aria-expanded={mobileMoreOpen}
          className="flex h-10 w-full items-center gap-3 rounded-full px-4 text-xs font-semibold transition-[background-color,color,transform] duration-200 ease-out hover:bg-[var(--explore-surface-soft)] active:scale-[0.98]"
          type="button"
          onClick={onOpenMore}
        >
          <Menu className="size-5" />
          {t("nav.more")}
        </button>
      </div>
    </aside>
  );
}

export function ExploreSiteBrand({
  compact = false,
  siteProfile,
}: {
  compact?: boolean;
  siteProfile: SiteProfile;
}) {
  const initial = Array.from(siteProfile.title.trim())[0]?.toUpperCase() ?? "Y";
  const avatar = siteProfile.avatarUrl?.trim();

  return (
    <div className={cn("flex min-w-0 items-center gap-3", compact && "flex-1")}>
      <div
        className={cn(
          "flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-[var(--explore-surface-soft)] text-sm font-black text-[var(--explore-strong)] ring-1 ring-[var(--explore-border)]",
          compact ? "size-9" : "size-10",
        )}
      >
        {avatar ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={avatar} alt={siteProfile.title} className="size-full object-cover" />
        ) : (
          <span>{initial}</span>
        )}
      </div>
      <div className="min-w-0">
        <p className={cn("truncate font-black tracking-normal text-[var(--explore-strong)]", compact ? "text-[15px]" : "text-[17px]")}>
          {siteProfile.title}
        </p>
        <p className="mt-0.5 truncate text-[11px] font-medium text-[var(--explore-subtle)]">
          {siteProfile.description}
        </p>
      </div>
    </div>
  );
}

export function ExploreMobileBottomNav({
  activeMobileView,
  activeMode,
  loadMode,
  messageBadgeCount,
  openProfileView,
  warmNavigationTarget,
}: {
  activeMobileView: MobileMainView;
  activeMode: FeedMode;
  loadMode: (mode: FeedMode, categoryId?: number | "all", sort?: FollowingSort) => void;
  messageBadgeCount: number;
  openProfileView: () => void;
  warmNavigationTarget: WarmNavigationTarget;
}) {
  const t = useTranslations();

  return (
    <nav
      aria-label={t("nav.home")}
      className="fixed inset-x-0 bottom-0 z-40 flex min-h-14 items-center justify-around border-t border-[var(--explore-border)] bg-[var(--explore-bottom)] px-3 pb-[env(safe-area-inset-bottom)] backdrop-blur lg:hidden"
    >
      {bottomNavItems.map(({ key, icon: Icon, href }) => {
        const active =
          (key === "profile" && activeMobileView === "profile") ||
          (activeMobileView === "feed" &&
            ((key === "home" && activeMode !== "following") ||
              (key === "friends" && activeMode === "following")));
        const itemClassName = cn(
          "flex min-w-0 flex-1 flex-col items-center gap-0.5 text-[11px] font-medium text-[var(--explore-subtle)]",
          active && "text-[var(--explore-strong)]",
          key === "create" && "text-white",
        );
        const itemContent = (
          <>
            <span
              className={cn(
                "relative flex size-7 items-center justify-center",
                key === "create" && "rounded-lg bg-primary text-white",
              )}
            >
              <NavIconWithBadge
                Icon={Icon}
                count={key === "messages" ? messageBadgeCount : 0}
                className="size-[19px]"
              />
            </span>
            <span className="max-w-full truncate">{t(`nav.${key}`)}</span>
          </>
        );

        return href ? (
          <Link
            key={key}
            href={href}
            aria-current={active ? "page" : undefined}
            onFocus={() => warmNavigationTarget(key, href)}
            onMouseEnter={() => warmNavigationTarget(key, href)}
            onTouchStart={() => warmNavigationTarget(key, href)}
            className={itemClassName}
          >
            {itemContent}
          </Link>
        ) : (
          <button
            key={key}
            type="button"
            aria-pressed={active}
            className={itemClassName}
            onFocus={() => warmNavigationTarget(key, key === "profile" ? "/profile" : null)}
            onMouseEnter={() => warmNavigationTarget(key, key === "profile" ? "/profile" : null)}
            onTouchStart={() => warmNavigationTarget(key, key === "profile" ? "/profile" : null)}
            onClick={() => {
              if (key === "home") {
                loadMode("recommended", "all");
                return;
              }

              if (key === "friends") {
                loadMode("following", "all");
                return;
              }

              if (key === "profile") {
                warmNavigationTarget(key, "/profile");
                openProfileView();
              }
            }}
          >
            {itemContent}
          </button>
        );
      })}
    </nav>
  );
}
