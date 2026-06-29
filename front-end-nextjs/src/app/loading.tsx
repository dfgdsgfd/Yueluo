export default function Loading() {
  return (
    <main
      aria-busy="true"
      aria-label="Loading page"
      className="flex min-h-dvh items-center justify-center bg-background px-6 text-foreground"
    >
      <div className="w-full max-w-sm rounded-2xl border border-border bg-card p-8 shadow-sm">
        <div className="mx-auto size-12 animate-pulse rounded-full bg-primary/70 motion-reduce:animate-none" />
        <div className="mx-auto mt-7 h-5 w-32 animate-pulse rounded-full bg-foreground/15 motion-reduce:animate-none" />
        <div className="mx-auto mt-4 h-3 w-52 max-w-full animate-pulse rounded-full bg-foreground/10 motion-reduce:animate-none" />
        <span className="sr-only">Loading page</span>
      </div>
    </main>
  );
}
