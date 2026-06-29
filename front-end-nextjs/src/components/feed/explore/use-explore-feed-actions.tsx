"use client";

import { ApiUnauthorizedError, toggleLike } from "@/lib/api";
import type { FeedPost } from "@/lib/types";
import type { QueryClient } from "@tanstack/react-query";
import type { useTranslations } from "next-intl";
import { useCallback } from "react";
import { toast } from "sonner";
import { updateFeedPostState } from "../post-feed-state";

type ExploreFeedActionsInput = {
  openLoginUnlock: () => void;
  queryClient: QueryClient;
  requireLogin: () => boolean;
  t: ReturnType<typeof useTranslations>;
};

export function useExploreFeedActions({
  openLoginUnlock,
  queryClient,
  requireLogin,
  t,
}: ExploreFeedActionsInput) {
  const updatePostState = useCallback(
    (postId: FeedPost["id"], updater: (post: FeedPost) => FeedPost) => {
      updateFeedPostState(queryClient, postId, updater);
    },
    [queryClient],
  );

  const handleLike = useCallback(
    async (post: FeedPost) => {
      if (requireLogin()) {
        return;
      }
      const nextLiked = !post.liked;
      updatePostState(post.id, (item) => ({
        ...item,
        liked: nextLiked,
        like_count: Math.max(0, item.like_count + (nextLiked ? 1 : -1)),
      }));
      try {
        const result = await toggleLike(post.id, { redirectOnUnauthorized: false });
        updatePostState(post.id, (item) => ({
          ...item,
          liked: result.liked,
          like_count: Math.max(
            0,
            item.like_count +
              (result.liked === nextLiked ? 0 : result.liked ? 1 : -1),
          ),
        }));
        toast.success(result.liked ? t("status.liked") : t("status.unliked"));
      } catch (error) {
        updatePostState(post.id, (item) => ({
          ...item,
          liked: post.liked,
          like_count: post.like_count,
        }));
        if (error instanceof ApiUnauthorizedError) {
          openLoginUnlock();
          return;
        }
        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
      }
    },
    [openLoginUnlock, requireLogin, t, updatePostState],
  );

  return { handleLike };
}
