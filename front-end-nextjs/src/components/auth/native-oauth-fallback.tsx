"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { selectNativeOAuthAutoOpenLink } from "@/lib/native-oauth-links";

export function NativeOAuthFallback({
  deepLink,
  intentLink,
  openLabel,
}: {
  deepLink: string;
  intentLink?: string;
  openLabel: string;
}) {
  useEffect(() => {
    const timer = window.setTimeout(() => {
      window.location.assign(selectNativeOAuthAutoOpenLink({ deepLink, intentLink }));
    }, 250);
    return () => window.clearTimeout(timer);
  }, [deepLink, intentLink]);

  return (
    <Button asChild className="mt-7 h-11 w-full">
      <a href={intentLink ?? deepLink}>{openLabel}</a>
    </Button>
  );
}
