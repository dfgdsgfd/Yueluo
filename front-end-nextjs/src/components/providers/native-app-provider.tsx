"use client";

import { App } from "@capacitor/app";
import { Browser } from "@capacitor/browser";
import { Haptics, ImpactStyle } from "@capacitor/haptics";
import { Network } from "@capacitor/network";
import { StatusBar, Style } from "@capacitor/status-bar";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { exchangeOAuthMobileSession, getStoredAccessToken, storeOAuthCallbackTokens } from "@/lib/api";
import {
  claimNativeOAuthCallback,
  describeNativeOAuthError,
  getNativeLaunchUrl,
  isNativeAndroidApp,
  openNativeBrowser,
  parseNativeOAuthCallback,
  recordNativeOAuthStatus,
  type NativeOAuthStatusStep,
} from "@/lib/native-app";

type ListenerHandle = { remove: () => Promise<void> };

export function NativeAppProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const t = useTranslations("login");
  const [offline, setOffline] = useState(false);
  const exchangingRef = useRef(false);
  const lastBackRef = useRef(0);

  useEffect(() => {
    if (!isNativeAndroidApp()) {
      return;
    }

    document.documentElement.dataset.nativePlatform = "android";
    const handles: ListenerHandle[] = [];

    async function handleOAuthUrl(rawUrl: string) {
      const callback = parseNativeOAuthCallback(rawUrl);
      if (!callback) {
        return false;
      }
      if (!claimNativeOAuthCallback(callback)) {
        return true;
      }
      void Browser.close().catch(() => {});
      recordNativeOAuthStatus({
        ok: true,
        step: "callback_received",
      });
      if (exchangingRef.current) {
        recordNativeOAuthStatus({
          detail: "another_exchange_in_progress",
          ok: false,
          step: "token_exchange_failed",
        });
        return true;
      }
      if (callback.error) {
        recordNativeOAuthStatus({
          detail: callback.error,
          ok: false,
          step: "oauth_error",
        });
        toast.error(t("nativeCancelledWithDetail", { detail: callback.error }));
        router.replace("/login");
        return true;
      }
      if (!callback.ticket) {
        recordNativeOAuthStatus({
          detail: "missing_ticket",
          ok: false,
          step: "missing_code",
        });
        toast.error(t("nativeExchangeFailed"));
        router.replace("/login");
        return true;
      }

      exchangingRef.current = true;
      let failureStep: NativeOAuthStatusStep = "token_exchange_failed";
      try {
        const payload = await exchangeOAuthMobileSession(callback.ticket);
        storeOAuthCallbackTokens({
          accessToken: payload.access_token,
          refreshToken: payload.refresh_token,
          user: payload.user,
        });
        if (getStoredAccessToken() !== payload.access_token) {
          failureStep = "token_storage_failed";
          recordNativeOAuthStatus({
            detail: "native_token_storage_failed",
            ok: false,
            step: "token_storage_failed",
          });
          throw new Error("native_token_storage_failed");
        }
        recordNativeOAuthStatus({
          ok: true,
          step: "signed_in",
        });
        toast.success(t("oauthSuccess"));
        await waitForNativeCookieFlush();
        window.location.replace(nativeOAuthSuccessReloadUrl(callback.returnUrl));
      } catch (error) {
        const detail = describeNativeOAuthError(error);
        recordNativeOAuthStatus({
          detail,
          ok: false,
          step: failureStep,
        });
        console.error("[native-oauth] token exchange failed", error);
        toast.error(t("nativeExchangeFailedWithDetail", { detail }));
        router.replace("/login");
      } finally {
        exchangingRef.current = false;
      }
      return true;
    }

    function syncStatusBar() {
      const dark = document.documentElement.dataset.yuemTheme !== "light";
      void StatusBar.setStyle({ style: dark ? Style.Dark : Style.Light });
    }

    function onDocumentClick(event: MouseEvent) {
      const target = event.target instanceof Element ? event.target : null;
      const interactive = target?.closest<HTMLElement>(
        "button:not(:disabled), a[href], [role='button']:not([aria-disabled='true'])",
      );
      if (interactive) {
        void Haptics.impact({ style: ImpactStyle.Light });
      }
      const anchor = target?.closest<HTMLAnchorElement>("a[href]");
      if (!anchor || anchor.hasAttribute("download")) {
        return;
      }
      let destination: URL;
      try {
        destination = new URL(anchor.href, window.location.href);
      } catch {
        return;
      }
      if (destination.origin === window.location.origin) {
        return;
      }
      event.preventDefault();
      void openNativeBrowser(destination.toString());
    }

    const themeObserver = new MutationObserver(syncStatusBar);
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-yuem-theme"],
    });
    syncStatusBar();
    document.addEventListener("click", onDocumentClick, true);

    void Network.getStatus().then((status) => setOffline(!status.connected));
    void Network.addListener("networkStatusChange", (status) => {
      setOffline(!status.connected);
      if (status.connected) {
        router.refresh();
      }
    }).then((handle) => handles.push(handle));
    void App.addListener("appUrlOpen", ({ url }) => {
      void handleOAuthUrl(url);
    }).then((handle) => handles.push(handle));
    void App.addListener("backButton", ({ canGoBack }) => {
      const path = window.location.pathname;
      const isRoot = path === "/" || path === "/explore" || path === "/login";
      if (canGoBack && !isRoot) {
        window.history.back();
        return;
      }
      const now = Date.now();
      if (now - lastBackRef.current < 1800) {
        void App.exitApp();
        return;
      }
      lastBackRef.current = now;
      toast(t("nativeBackAgain"));
    }).then((handle) => handles.push(handle));

    void getNativeLaunchUrl().then((url) => {
      if (url) {
        void handleOAuthUrl(url);
      }
    });

    return () => {
      themeObserver.disconnect();
      document.removeEventListener("click", onDocumentClick, true);
      delete document.documentElement.dataset.nativePlatform;
      for (const handle of handles) {
        void handle.remove();
      }
    };
  }, [router, t]);

  return (
    <>
      {children}
      {offline ? (
        <div
          role="status"
          className="native-offline-banner fixed inset-x-3 top-[calc(var(--native-safe-top)+0.5rem)] z-[10050] mx-auto max-w-sm rounded-full bg-amber-500 px-4 py-2 text-center text-sm font-medium text-black shadow-lg"
        >
          {t("nativeOffline")}
        </div>
      ) : null}
    </>
  );
}

function nativeOAuthSuccessReloadUrl(rawUrl: string) {
  const url = new URL(rawUrl, window.location.origin);
  if (url.origin === window.location.origin && isLoginPath(url.pathname)) {
    url.pathname = "/explore";
    url.search = "";
    url.hash = "";
  }
  if (url.origin === window.location.origin) {
    url.searchParams.set("_native_oauth_reload", Date.now().toString(36));
  }
  return url.toString();
}

function isLoginPath(pathname: string) {
  return (pathname.replace(/\/+$/u, "") || "/") === "/login";
}

function waitForNativeCookieFlush() {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, 250);
  });
}
