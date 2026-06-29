"use client";

import dynamic from "next/dynamic";
import { useEffect, useState } from "react";
import type { FeedPost } from "@/lib/types";
import {
  POST_DETAIL_INSTANT_OPEN_EVENT,
  type PostDetailInstantOpenDetail,
} from "./post-detail-instant-events";

const InstantPostDetailPageClient = dynamic(
  () =>
    import("./post-detail-page-client").then(
      (module) => module.PostDetailPageClient,
    ),
  { ssr: false },
);

export function PostDetailInstantHost() {
  const [post, setPost] = useState<FeedPost | null>(null);

  useEffect(() => {
    function handleOpen(event: Event) {
      const detail = (event as CustomEvent<PostDetailInstantOpenDetail>).detail;
      if (detail?.post) {
        setPost(detail.post);
      }
    }

    window.addEventListener(POST_DETAIL_INSTANT_OPEN_EVENT, handleOpen);
    return () => window.removeEventListener(POST_DETAIL_INSTANT_OPEN_EVENT, handleOpen);
  }, []);

  if (!post) {
    return null;
  }

  return (
    <InstantPostDetailPageClient
      postId={post.id}
      initialPost={post}
      mode="drawer"
      onClose={() => setPost(null)}
    />
  );
}
