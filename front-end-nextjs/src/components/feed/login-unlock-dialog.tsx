"use client";

import { Button } from "@/components/ui/button";
import * as Dialog from "@radix-ui/react-dialog";
import { LogIn, X } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";

type LoginUnlockDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function LoginUnlockDialog({ open, onOpenChange }: LoginUnlockDialogProps) {
  const t = useTranslations();
  const loginHref = currentLoginHref();

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-[90] bg-black/60 backdrop-blur-sm" />
        <Dialog.Content className="fixed inset-x-0 bottom-0 z-[91] rounded-t-[22px] border border-white/10 bg-[#18181c] p-5 text-white shadow-2xl outline-none md:left-1/2 md:top-1/2 md:bottom-auto md:w-[min(420px,calc(100vw-2rem))] md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-[20px]">
          <div className="flex items-start justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
              <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-primary/18 text-primary">
                <LogIn className="size-5" />
              </span>
              <div className="min-w-0">
                <Dialog.Title className="text-lg font-black leading-tight">
                  {t("loginUnlock.title")}
                </Dialog.Title>
                <Dialog.Description className="mt-1 text-sm leading-6 text-white/58">
                  {t("loginUnlock.description")}
                </Dialog.Description>
              </div>
            </div>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label={t("loginUnlock.later")}
                className="rounded-full p-1.5 text-white/52 hover:bg-white/10 hover:text-white"
              >
                <X className="size-4" />
              </button>
            </Dialog.Close>
          </div>

          <div className="mt-5 flex gap-3">
            <Button
              asChild
              className="h-11 flex-1 rounded-full"
            >
              <Link href={loginHref}>
                <LogIn className="size-4" aria-hidden="true" />
                {t("loginUnlock.login")}
              </Link>
            </Button>
            <Dialog.Close asChild>
              <Button type="button" variant="outline" className="h-11 flex-1 rounded-full border-white/12 bg-white/5 text-white hover:bg-white/10">
                {t("loginUnlock.later")}
              </Button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function currentLoginHref() {
  if (typeof window === "undefined") {
    return "/login";
  }
  const returnPath = `${window.location.pathname}${window.location.search}`;
  if (!returnPath || returnPath === "/login" || returnPath.startsWith("/login?")) {
    return "/login";
  }
  return `/login?next=${encodeURIComponent(returnPath)}`;
}
