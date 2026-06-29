"use client";

import { useState } from "react";
import type { useTranslations } from "next-intl";
import { Loader2, UsersRound } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { batchGenerateAdminUsers } from "@/lib/api";
import { errorMessage } from "./helpers";

export function UserBatchGenerate({
  token,
  t,
  onGenerated,
}: {
  token: string;
  t: ReturnType<typeof useTranslations<"adminPortal.userManagement">>;
  onGenerated: () => Promise<void> | void;
}) {
  const [count, setCount] = useState(10);
  const [loading, setLoading] = useState(false);

  async function generate() {
    const nextCount = Math.max(1, Math.min(500, Math.floor(Number(count) || 1)));
    setLoading(true);
    try {
      const payload = await batchGenerateAdminUsers(nextCount, token);
      toast.success(t("batchGenerateSuccess", { count: payload.count }));
      await onGenerated();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 rounded-lg border border-[#1d4ed8]/10 bg-[#f5f8ff] px-2 py-2">
      <label className="flex min-w-0 items-center gap-2 text-xs font-semibold text-[#4f5f7a]">
        <UsersRound className="size-4 text-[#1d4ed8]" />
        <span>{t("batchGenerateCount")}</span>
        <input
          type="number"
          min={1}
          max={500}
          value={count}
          onChange={(event) => setCount(Number(event.target.value))}
          className="h-9 w-20 rounded-lg border border-black/[0.08] bg-white px-2 text-sm text-[#20232a] outline-none focus:border-[#1d4ed8]"
        />
      </label>
      <Button
        type="button"
        variant="outline"
        disabled={loading}
        onClick={() => void generate()}
        className="h-9 rounded-lg border-[#1d4ed8]/20 bg-white px-3 text-[#1d4ed8] hover:bg-[#eef4ff]"
      >
        {loading ? <Loader2 className="size-4 animate-spin" /> : <UsersRound className="size-4" />}
        <span>{loading ? t("batchGenerating") : t("batchGenerate")}</span>
      </Button>
    </div>
  );
}
