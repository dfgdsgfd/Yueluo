import Link from "next/link";
import { Compass } from "lucide-react";
import { Button } from "@/components/ui/button";

export const metadata = {
  title: "Page not found",
};

export default function NotFound() {
  return (
    <main className="flex min-h-dvh items-center justify-center bg-background px-6 text-foreground">
      <section
        aria-labelledby="not-found-title"
        className="w-full max-w-sm rounded-2xl border border-border bg-card p-8 text-center shadow-sm"
      >
        <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-primary/15 text-primary">
          <Compass className="size-6" aria-hidden="true" />
        </div>
        <p className="mt-6 text-sm font-semibold text-primary">404</p>
        <h1 id="not-found-title" className="mt-2 text-2xl font-semibold">
          Page not found
        </h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          The page may have moved or is no longer available.
        </p>
        <Button asChild className="mt-7 w-full">
          <Link href="/">Return home</Link>
        </Button>
      </section>
    </main>
  );
}
