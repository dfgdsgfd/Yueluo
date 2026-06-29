"use client";

import type { PostDetailDrawerProps } from "./post-detail-drawer-types";
import { PostDetailDrawerView } from "./post-detail-drawer-view";
import { usePostDetailDrawerController } from "./use-post-detail-drawer-controller";

export function PostDetailDrawer(props: PostDetailDrawerProps) {
  const controller = usePostDetailDrawerController(props);
  return <PostDetailDrawerView controller={controller} />;
}
