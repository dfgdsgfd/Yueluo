"use client";

import { useReportWebVitals } from "next/web-vitals";

const WEB_VITALS_ENDPOINT = process.env.NEXT_PUBLIC_WEB_VITALS_ENDPOINT;

export function WebVitalsReporter() {
  useReportWebVitals((metric) => {
    if (WEB_VITALS_ENDPOINT) {
      const body = JSON.stringify(metric);
      if (navigator.sendBeacon) {
        navigator.sendBeacon(WEB_VITALS_ENDPOINT, body);
        return;
      }

      void fetch(WEB_VITALS_ENDPOINT, {
        body,
        cache: "no-store",
        headers: { "content-type": "application/json" },
        keepalive: true,
        method: "POST",
      }).catch(() => undefined);
      return;
    }

    if (process.env.NODE_ENV !== "production" && ["LCP", "INP", "CLS"].includes(metric.name)) {
      console.debug("[web-vitals]", metric.name, metric.value, metric);
    }
  });

  return null;
}
