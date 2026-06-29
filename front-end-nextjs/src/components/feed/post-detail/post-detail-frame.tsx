"use client";

import type { ReactNode, RefObject } from "react";
import { Drawer } from "vaul";
import { cn } from "@/lib/utils";

export function PostDetailFrame({
  children,
  description,
  drawerContentRef,
  fullScreen,
  mode,
  onOpenChange,
  open,
}: {
  children: ReactNode;
  description: string;
  drawerContentRef: RefObject<HTMLDivElement | null>;
  fullScreen?: boolean;
  mode: "drawer" | "page" | "modal";
  onOpenChange: (open: boolean) => void;
  open: boolean;
}) {
  if (mode === "page") {
    return (
      <main
        className={cn(
          "theme-adaptive flex h-dvh flex-col items-center bg-[#121212] text-white",
          fullScreen ? "" : "md:justify-center md:p-8",
        )}
      >
        {children}
      </main>
    );
  }

  return (
    <Drawer.Root
      open={open}
      onOpenChange={onOpenChange}
      preventScrollRestoration
      repositionInputs={false}
    >
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 z-40 bg-black/55 transition-opacity duration-200 data-[state=closed]:opacity-0 data-[state=open]:opacity-100" />
        <Drawer.Content
          ref={drawerContentRef}
          tabIndex={-1}
          onOpenAutoFocus={(event) => {
            event.preventDefault();
            drawerContentRef.current?.focus({ preventScroll: true });
          }}
          className={cn(
            "post-detail-content theme-adaptive fixed inset-0 z-50 flex flex-col overflow-hidden bg-[#121212] text-white outline-none",
            fullScreen ? "" : "md:items-center md:justify-center md:bg-transparent md:p-8",
          )}
        >
          <Drawer.Description className="sr-only">{description}</Drawer.Description>
          {children}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  );
}
