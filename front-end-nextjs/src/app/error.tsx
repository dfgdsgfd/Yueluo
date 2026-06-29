"use client";

import { useEffect } from "react";
import Link from "next/link";
import { AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function ErrorPage({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string };
  unstable_retry: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="flex min-h-dvh items-center justify-center bg-background px-6 text-foreground">
      <section
        aria-labelledby="error-title"
        className="w-full max-w-sm rounded-2xl border border-border bg-card p-8 text-center shadow-sm"
      >
        <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-destructive/15 text-destructive">
          <AlertCircle className="size-6" aria-hidden="true" />
        </div>
        <h1 id="error-title" className="mt-6 text-2xl font-semibold">
          Something went wrong
        </h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          This page could not be loaded. Your data has not been changed.
        </p>
        <div className="mt-7 grid gap-3">
          <Button type="button" onClick={unstable_retry}>
            <RefreshCw className="size-4" aria-hidden="true" />
            Try again
          </Button>
          <Button asChild variant="outline">
            <Link href="/">Return home</Link>
          </Button>
        </div>
      </section>
    </main>
  );
}
