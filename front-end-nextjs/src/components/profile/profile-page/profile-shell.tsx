"use client";
import Link from "next/link";
import {
  ChevronRight
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  Button
} from "@/components/ui/button";

export function ProfileNotFound({
  onBack,
  userId,
}: {
  onBack?: () => void;
  userId: string;
}) {
  const t = useTranslations();

  return (
    <div className="theme-adaptive flex min-h-screen items-center justify-center bg-[#121212] px-6 text-white">
      <div className="w-full max-w-[420px] text-center">
        <h1 className="text-2xl font-bold">{t("profile.notFoundTitle")}</h1>
        <p className="mt-3 text-sm leading-6 text-white/52">
          {t("profile.notFoundDescription", { id: userId })}
        </p>
        {onBack ? (
          <Button type="button" onClick={onBack} className="mt-6 h-10 px-5">
            {t("profile.backHome")}
            <ChevronRight className="size-4" />
          </Button>
        ) : (
          <Button asChild className="mt-6 h-10 px-5">
            <Link href="/">
              {t("profile.backHome")}
              <ChevronRight className="size-4" />
            </Link>
          </Button>
        )}
      </div>
    </div>
  );
}


export function formatCompactCount(count: number) {
  if (count >= 10000) {
    return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`;
  }

  if (count >= 1000) {
    return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`;
  }

  return String(count);
}
