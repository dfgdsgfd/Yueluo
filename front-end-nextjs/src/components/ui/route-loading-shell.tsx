import { cn } from "@/lib/utils";

type RouteLoadingShellProps = {
  tone?: "dark" | "light";
};

export function RouteLoadingShell({ tone = "light" }: RouteLoadingShellProps) {
  const dark = tone === "dark";

  return (
    <main
      aria-busy="true"
      className={cn(
        "min-h-dvh px-4 py-4",
        dark ? "bg-[#121212] text-white" : "bg-[#fbfbff] text-[#24242c]",
      )}
    >
      <div className="mx-auto flex min-h-[calc(100dvh-2rem)] w-full max-w-[1180px] flex-col gap-4">
        <div
          className={cn(
            "h-14 rounded-[8px] motion-reduce:animate-none",
            dark ? "animate-pulse bg-white/[0.08]" : "animate-pulse bg-black/[0.06]",
          )}
        />
        <div className="grid min-h-0 flex-1 gap-4 md:grid-cols-[minmax(220px,320px)_minmax(0,1fr)]">
          <div
            className={cn(
              "hidden rounded-[8px] md:block motion-reduce:animate-none",
              dark ? "animate-pulse bg-white/[0.07]" : "animate-pulse bg-black/[0.05]",
            )}
          />
          <div className="grid content-start gap-3">
            {[0, 1, 2, 3, 4].map((item) => (
              <div
                key={item}
                className={cn(
                  "h-20 rounded-[8px] motion-reduce:animate-none",
                  dark ? "animate-pulse bg-white/[0.07]" : "animate-pulse bg-black/[0.05]",
                )}
              />
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}
