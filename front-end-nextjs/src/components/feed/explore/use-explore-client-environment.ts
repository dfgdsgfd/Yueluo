"use client";

import { useEffect, type Dispatch, type SetStateAction } from "react";
import { getAuthConfig, getCurrentUser, getStoredAccessToken } from "@/lib/api";
import type { VideoCenterVisibilityConfig } from "@/lib/types";
import {
  normalizeVideoCenterConfig,
  videoCenterConfigFromAuthConfig,
} from "@/lib/video-center";
import {
  EXPLORE_THEME_STORAGE_KEY,
  type ExploreTheme,
  type ExploreThemePreference,
} from "./explore-config";

export function useExploreClientEnvironment({
  exploreTheme,
  exploreThemePreference,
  hasClientAccessToken,
  hasLoadedExploreTheme,
  setExploreTheme,
  setExploreThemePreference,
  setHasClientAccessToken,
  setHasLoadedExploreTheme,
  setVideoCenterConfig,
  setViewerCreatedAt,
}: {
  exploreTheme: ExploreTheme;
  exploreThemePreference: ExploreThemePreference;
  hasClientAccessToken: boolean;
  hasLoadedExploreTheme: boolean;
  setExploreTheme: Dispatch<SetStateAction<ExploreTheme>>;
  setExploreThemePreference: Dispatch<SetStateAction<ExploreThemePreference>>;
  setHasClientAccessToken: Dispatch<SetStateAction<boolean>>;
  setHasLoadedExploreTheme: Dispatch<SetStateAction<boolean>>;
  setVideoCenterConfig: Dispatch<SetStateAction<VideoCenterVisibilityConfig>>;
  setViewerCreatedAt: Dispatch<SetStateAction<string | null>>;
}) {
  useEffect(() => {
    let cancelled = false;
    let checks = 0;
    function syncAccessTokenState() {
      if (cancelled) {
        return;
      }
      checks += 1;
      const hasToken = Boolean(getStoredAccessToken());
      setHasClientAccessToken(hasToken);
      if (hasToken || checks >= 20) {
        window.clearInterval(intervalId);
      }
    }
    const intervalId = window.setInterval(syncAccessTokenState, 250);
    syncAccessTokenState();
    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, [setHasClientAccessToken]);

  useEffect(() => {
    let cancelled = false;
    getAuthConfig()
      .then((config) => {
        if (!cancelled) {
          setVideoCenterConfig(videoCenterConfigFromAuthConfig(config));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setVideoCenterConfig((current) =>
            normalizeVideoCenterConfig(current),
          );
        }
      });
    if (!hasClientAccessToken) {
      setViewerCreatedAt(null);
      return () => {
        cancelled = true;
      };
    }
    getCurrentUser()
      .then((user) => {
        if (!cancelled) {
          setViewerCreatedAt(user.created_at ?? null);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setViewerCreatedAt(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [hasClientAccessToken, setVideoCenterConfig, setViewerCreatedAt]);

  useEffect(() => {
    const storedTheme = window.localStorage.getItem(EXPLORE_THEME_STORAGE_KEY);
    const prefersLight = window.matchMedia?.(
      "(prefers-color-scheme: light)",
    ).matches;
    const systemTheme = prefersLight ? "light" : "dark";
    const nextPreference: ExploreThemePreference =
      storedTheme === "light" ||
      storedTheme === "dark" ||
      storedTheme === "system"
        ? storedTheme
        : "system";
    const nextTheme =
      nextPreference === "system" ? systemTheme : nextPreference;
    setExploreThemePreference(nextPreference);
    setExploreTheme(nextTheme);
    document.documentElement.dataset.yuemTheme = nextTheme;
    setHasLoadedExploreTheme(true);
  }, [setExploreTheme, setExploreThemePreference, setHasLoadedExploreTheme]);

  useEffect(() => {
    if (!hasLoadedExploreTheme) {
      return;
    }
    document.documentElement.dataset.yuemTheme = exploreTheme;
    window.localStorage.setItem(
      EXPLORE_THEME_STORAGE_KEY,
      exploreThemePreference,
    );
  }, [exploreTheme, exploreThemePreference, hasLoadedExploreTheme]);

  useEffect(() => {
    if (!hasLoadedExploreTheme || exploreThemePreference !== "system") {
      return;
    }
    const mediaQuery = window.matchMedia?.("(prefers-color-scheme: light)");
    if (!mediaQuery) {
      return;
    }
    function syncSystemTheme(event: MediaQueryListEvent | MediaQueryList) {
      setExploreTheme(event.matches ? "light" : "dark");
    }
    syncSystemTheme(mediaQuery);
    mediaQuery.addEventListener("change", syncSystemTheme);
    return () => mediaQuery.removeEventListener("change", syncSystemTheme);
  }, [exploreThemePreference, hasLoadedExploreTheme, setExploreTheme]);
}
