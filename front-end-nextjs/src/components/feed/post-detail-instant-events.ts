"use client";

import type { FeedPost } from "@/lib/types";

export const POST_DETAIL_INSTANT_OPEN_EVENT = "yuem:post-detail-instant-open";

export type PostDetailInstantOpenDetail = {
  post: FeedPost;
};

export function openPostDetailInstantly(post: FeedPost) {
  window.dispatchEvent(
    new CustomEvent<PostDetailInstantOpenDetail>(POST_DETAIL_INSTANT_OPEN_EVENT, {
      detail: { post },
    }),
  );
}
