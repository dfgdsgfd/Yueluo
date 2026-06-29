"use client";

import { useEffect } from "react";
import {
  emitMessageBadgeCount,
  getMessageBadgeCount,
  subscribeMessageBadgeCount,
} from "@/lib/im-unread";

export function useMessageBadgeSync({
  enabled,
  setMessageBadgeCount,
}: {
  enabled: boolean;
  setMessageBadgeCount: (count: number) => void;
}) {
  useEffect(() => {
    if (!enabled) {
      setMessageBadgeCount(0);
      emitMessageBadgeCount(0);
      return;
    }

    let cancelled = false;
    let idleLoadId: ReturnType<typeof window.requestIdleCallback> | null = null;
    const requestIdleCallback =
      window.requestIdleCallback ??
      ((callback: IdleRequestCallback) =>
        window.setTimeout(
          () => callback({ didTimeout: false, timeRemaining: () => 0 }),
          750,
        ));
    const cancelIdleCallback =
      window.cancelIdleCallback ??
      ((id: ReturnType<typeof window.setTimeout>) => window.clearTimeout(id));
    async function loadMessageBadgeCount() {
      const nextCount = await getMessageBadgeCount({ background: true });
      if (cancelled) {
        return;
      }
      setMessageBadgeCount(nextCount);
      emitMessageBadgeCount(nextCount);
    }
    function scheduleMessageBadgeCount() {
      if (idleLoadId !== null) {
        return;
      }
      idleLoadId = requestIdleCallback(
        () => {
          idleLoadId = null;
          void loadMessageBadgeCount();
        },
        { timeout: 2500 },
      );
    }
    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        scheduleMessageBadgeCount();
      }
    }
    function handleWindowFocus() {
      scheduleMessageBadgeCount();
    }
    const unsubscribe = subscribeMessageBadgeCount((count) => {
      if (!cancelled) {
        setMessageBadgeCount(count);
      }
    });
    const intervalId = window.setInterval(() => {
      if (document.visibilityState !== "hidden") {
        scheduleMessageBadgeCount();
      }
    }, 45000);
    window.addEventListener("focus", handleWindowFocus);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    if (document.visibilityState !== "hidden") {
      scheduleMessageBadgeCount();
    }
    return () => {
      cancelled = true;
      if (idleLoadId !== null) {
        cancelIdleCallback(idleLoadId);
      }
      unsubscribe();
      window.clearInterval(intervalId);
      window.removeEventListener("focus", handleWindowFocus);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [enabled, setMessageBadgeCount]);
}
