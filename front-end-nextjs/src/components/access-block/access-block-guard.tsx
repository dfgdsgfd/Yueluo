"use client";

import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { buildApiUrl } from "@/lib/api";
import {
  isAccessBlockApiError,
  throwAccessBlockResponse,
} from "@/lib/api/core/access-block";

const ACCESS_BLOCK_PROBE_PATH = "/api/health";
const MIN_PROBE_INTERVAL_MS = 30_000;
const PROBE_TIMEOUT_MS = 2_500;

export function AccessBlockGuard() {
  const pathname = usePathname();
  const [blocked, setBlocked] = useState(false);
  const lastProbeAtRef = useRef(0);

  useEffect(() => {
    if (blocked) {
      return;
    }

    let cancelled = false;
    let timeoutId: number | null = null;
    const runProbe = async () => {
      const now = Date.now();
      if (now - lastProbeAtRef.current < MIN_PROBE_INTERVAL_MS) {
        return;
      }
      lastProbeAtRef.current = now;

      const controller = new AbortController();
      timeoutId = window.setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
      const url = buildApiUrl(ACCESS_BLOCK_PROBE_PATH);

      try {
        const response = await fetch(url, {
          method: "GET",
          credentials: "include",
          cache: "no-store",
          redirect: "manual",
          headers: { accept: "application/json" },
          signal: controller.signal,
        });
        throwAccessBlockResponse(response, url);
      } catch (error) {
        if (cancelled || isAbortError(error)) {
          return;
        }
        if (isAccessBlockApiError(error)) {
          setBlocked(true);
        }
      } finally {
        if (timeoutId !== null) {
          window.clearTimeout(timeoutId);
        }
      }
    };

    const scheduleIdle: typeof window.requestIdleCallback =
      window.requestIdleCallback ??
      ((callback) =>
        window.setTimeout(
          () => callback({ didTimeout: false, timeRemaining: () => 0 }),
          900,
        ));
    const cancelIdle: typeof window.cancelIdleCallback =
      window.cancelIdleCallback ?? ((id) => window.clearTimeout(id));
    const idleId = scheduleIdle(() => {
      void runProbe();
    }, { timeout: 2_000 });

    return () => {
      cancelled = true;
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
      }
      cancelIdle(idleId);
    };
  }, [blocked, pathname]);

  if (!blocked) {
    return null;
  }

  return (
    <div
      aria-hidden="true"
      className="fixed inset-0 z-[2147483647] bg-background"
    />
  );
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}
