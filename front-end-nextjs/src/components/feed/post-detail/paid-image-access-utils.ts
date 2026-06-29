import type { FeedPost } from "@/lib/types";

export function getUnlockImageCounts(post: FeedPost) {
  const protectedCount = post.lockedProtectedImagesCount ?? post.protectedPaidImagesCount ?? 0;
  const fallbackHiddenCount = Math.max(
    0,
    (post.totalImagesCount ?? 0) - (post.images?.length ?? 0),
  );
  const hiddenCount = post.hiddenPaidImagesCount ?? fallbackHiddenCount;
  return {
    directCount: Math.max(0, hiddenCount - protectedCount),
    protectedCount,
  };
}
