"use client";

import { Sparkles } from "lucide-react";
import type { FeedPost } from "@/lib/types";
import { cn } from "@/lib/utils";

type OriginalIncentivePost = Pick<FeedPost, "original_incentive" | "quality_reward">;

export function getOriginalIncentiveAmount(post: OriginalIncentivePost) {
  const amount = Number(post.quality_reward ?? 0);
  return Number.isFinite(amount) && amount > 0 ? amount : 0;
}

export function hasOriginalIncentive(post: OriginalIncentivePost) {
  return Boolean(post.original_incentive) || getOriginalIncentiveAmount(post) > 0;
}

export function formatOriginalIncentiveAmount(post: OriginalIncentivePost, locale: string) {
  const amount = getOriginalIncentiveAmount(post);
  return amount.toLocaleString(locale || "zh-CN", {
    maximumFractionDigits: 2,
    minimumFractionDigits: amount % 1 === 0 ? 0 : 2,
  });
}

export function OriginalIncentiveBadge({
  post,
  label,
  className,
}: {
  post: OriginalIncentivePost;
  label: string;
  className?: string;
}) {
  if (!hasOriginalIncentive(post)) {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex h-6 items-center gap-1 rounded-full border border-amber-200/40 bg-amber-100/45 px-2 text-[11px] font-medium text-[#5c4114] shadow-[0_6px_16px_rgba(0,0,0,0.10)] backdrop-blur-md",
        className,
      )}
    >
      <Sparkles className="size-3 text-amber-500/80" />
      {label}
    </span>
  );
}

export function OriginalIncentiveReward({
  post,
  title,
  amountLabel,
  className,
}: {
  post: OriginalIncentivePost;
  title: string;
  amountLabel: string;
  className?: string;
}) {
  if (getOriginalIncentiveAmount(post) <= 0) {
    return null;
  }

  return (
    <div className={cn("rounded-lg border border-white/12 bg-white/[0.055] px-3 py-2.5 backdrop-blur-sm", className)}>
      <p className="inline-flex items-center gap-1.5 text-xs font-medium text-white/64">
        <Sparkles className="size-3.5 text-amber-200/70" />
        {title}
      </p>
      <p className="mt-1 text-sm font-medium text-white/82">{amountLabel}</p>
    </div>
  );
}
