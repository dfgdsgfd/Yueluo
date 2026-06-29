"use client";

import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import type { FeedPayload, FeedPost } from "@/lib/types";

export const FEED_QUERY_KEY = "explore-feed";

export function findFeedPostInQueryCache(
  queryClient: QueryClient,
  postId: FeedPost["id"],
) {
  const targetId = String(postId);
  const cachedFeeds = queryClient.getQueriesData<InfiniteData<FeedPayload, number>>({
    queryKey: [FEED_QUERY_KEY],
  });

  for (const [, data] of cachedFeeds) {
    for (const page of data?.pages ?? []) {
      const post = page.posts.find((item) => String(item.id) === targetId);
      if (post) {
        return post;
      }
    }
  }

  return null;
}

export function updateFeedPostState(
  queryClient: QueryClient,
  postId: FeedPost["id"],
  updater: (post: FeedPost) => FeedPost | null,
) {
  const targetId = String(postId);

  queryClient.setQueriesData<InfiniteData<FeedPayload, number>>(
    { queryKey: [FEED_QUERY_KEY] },
    (current) =>
      current
        ? {
            ...current,
            pages: current.pages.map((page) => ({
              ...page,
              posts: page.posts.flatMap((item) => {
                if (String(item.id) !== targetId) {
                  return [item];
                }

                const nextPost = updater(item);
                return nextPost ? [nextPost] : [];
              }),
            })),
          }
        : current,
  );
}
