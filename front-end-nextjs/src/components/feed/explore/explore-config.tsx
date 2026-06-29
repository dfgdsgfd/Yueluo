"use client";
import {
  type CSSProperties
} from "react";
import dynamic from "next/dynamic";
import {
  Bell,
  CirclePlus,
  ExternalLink,
  Home,
  LinkIcon,
  MessageCircle,
  Plus,
  User,
  Video,
  Users
} from "lucide-react";
import {
  ProfileTabSkeleton
} from "./explore-widgets";

export const Lightbox = dynamic(() => import("@/components/feed/lightbox-client"), {
  ssr: false,
});

export const ViewerProfileDataPage = dynamic(
  () => import("@/components/profile/profile-data-page").then((mod) => mod.ViewerProfileDataPage),
  {
    loading: () => <ProfileTabSkeleton />,
    ssr: false,
  },
);


export const desktopNavItems = [
  { key: "home", icon: Home, href: "/" },
  { key: "videoCenter", icon: Video, href: "https://v2.yuelk.com" },
  { key: "publish", icon: CirclePlus, href: "/publish" },
  { key: "messages", icon: Bell, href: "/messages" },
  { key: "profile", icon: User, href: "/profile" },
] as const;

export const toolbarIconMap = {
  external: ExternalLink,
  link: LinkIcon,
} as const;


export const bottomNavItems = [
  { key: "home", icon: Home, href: null },
  { key: "friends", icon: Users, href: null },
  { key: "create", icon: Plus, href: "/publish/mobile" },
  { key: "messages", icon: MessageCircle, href: "/messages" },
  { key: "profile", icon: User, href: null },
] as const;


export const followingSortTabs = [
  { labelKey: "latest", sort: "time" },
  { labelKey: "hot", sort: "hot" },
] as const;


export type FollowingSort = (typeof followingSortTabs)[number]["sort"];

export type MobileMainView = "feed" | "profile";

export type ExploreTheme = "dark" | "light";

export type ExploreThemePreference = "system" | ExploreTheme;

export type ExploreThemeVars = CSSProperties & Record<`--${string}`, string>;


export const EXPLORE_THEME_STORAGE_KEY = "yuem_explore_theme";

const DEFAULT_FEED_MAX_PAGES = 11;
const MIN_FEED_MAX_PAGES = 3;
const MAX_FEED_MAX_PAGES = 40;
const DEFAULT_FEED_SENTINEL_ROOT_MARGIN_PX = 900;
const MIN_FEED_SENTINEL_ROOT_MARGIN_PX = 0;
const MAX_FEED_SENTINEL_ROOT_MARGIN_PX = 4_000;

function getBoundedIntegerEnv(
  value: string | undefined,
  fallback: number,
  min: number,
  max: number,
) {
  const parsed = Number.parseInt(value?.trim() ?? "", 10);
  if (!Number.isFinite(parsed)) {
    return fallback;
  }

  return Math.min(max, Math.max(min, parsed));
}

export const FEED_MAX_PAGES = getBoundedIntegerEnv(
  process.env.NEXT_PUBLIC_EXPLORE_FEED_RETAIN_PAGES,
  DEFAULT_FEED_MAX_PAGES,
  MIN_FEED_MAX_PAGES,
  MAX_FEED_MAX_PAGES,
);

export const FEED_SENTINEL_ROOT_MARGIN_PX = getBoundedIntegerEnv(
  process.env.NEXT_PUBLIC_EXPLORE_FEED_AUTOLOAD_MARGIN_PX,
  DEFAULT_FEED_SENTINEL_ROOT_MARGIN_PX,
  MIN_FEED_SENTINEL_ROOT_MARGIN_PX,
  MAX_FEED_SENTINEL_ROOT_MARGIN_PX,
);

export const FEED_TOP_SENTINEL_ROOT_MARGIN = `${FEED_SENTINEL_ROOT_MARGIN_PX}px 0px 0px 0px`;

export const FEED_BOTTOM_SENTINEL_ROOT_MARGIN = `0px 0px ${FEED_SENTINEL_ROOT_MARGIN_PX}px 0px`;

export const FEED_SENTINEL_DEBOUNCE_MS = 280;

export const FEED_LOADING_MIN_VISIBLE_MS = 240;

export const exploreThemeOptions = [
  { value: "system" },
  { value: "light" },
  { value: "dark" },
] as const satisfies Array<{ value: ExploreThemePreference }>;


export function estimateCategoryItemSize(name: string) {
  const visualWidth = Array.from(name).reduce((total, char) => {
    return total + (char.charCodeAt(0) > 255 ? 15 : 8);
  }, 0);

  return Math.max(72, visualWidth + 44);
}


export function formatWalletAmount(value?: number) {
  const amount = Number.isFinite(value) ? value ?? 0 : 0;

  return amount.toLocaleString("zh-CN", {
    maximumFractionDigits: 2,
    minimumFractionDigits: amount % 1 === 0 ? 0 : 2,
  });
}


export function getExploreThemeVars(theme: ExploreTheme): ExploreThemeVars {
  const light = theme === "light";

  return {
    "--explore-bg": light ? "#f6f6f7" : "#121212",
    "--explore-sidebar": light ? "#ffffff" : "#141418",
    "--explore-surface": light ? "#ffffff" : "#181818",
    "--explore-surface-strong": light ? "#f1f1f3" : "#1e1e1e",
    "--explore-surface-soft": light ? "rgba(28, 28, 32, 0.06)" : "rgba(255, 255, 255, 0.06)",
    "--explore-control": light ? "#f1f1f3" : "#1e1e1e",
    "--explore-control-hover": light ? "#e8e8eb" : "#29292e",
    "--explore-text": light ? "#25252b" : "#e0e0e0",
    "--explore-strong": light ? "#17171c" : "#ffffff",
    "--explore-muted": light ? "#64646d" : "#b0b0b0",
    "--explore-subtle": light ? "#8a8a91" : "rgba(255, 255, 255, 0.45)",
    "--explore-border": light ? "rgba(28, 28, 32, 0.1)" : "rgba(255, 255, 255, 0.08)",
    "--explore-bottom": light ? "rgba(255, 255, 255, 0.96)" : "rgba(18, 18, 18, 0.96)",
    "--explore-badge-ring": light ? "#ffffff" : "#121212",
  };
}
