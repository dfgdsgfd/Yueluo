import type { PostDetailDrawerProps } from "./post-detail-drawer-types";
import { usePostDetailBase } from "./use-post-detail-base";
import { usePostDetailCommentActions } from "./use-post-detail-comment-actions";
import { usePostDetailCommentLifecycle } from "./use-post-detail-comment-lifecycle";
import { usePostDetailPostActions } from "./use-post-detail-post-actions";

export function usePostDetailDrawerController(props: PostDetailDrawerProps) {
  const base = usePostDetailBase(props);
  const commentLifecycle = usePostDetailCommentLifecycle(base);
  const commentActions = usePostDetailCommentActions({ ...base, ...commentLifecycle });
  const postActions = usePostDetailPostActions({ ...base, ...commentLifecycle, ...commentActions });
  return { ...base, ...commentLifecycle, ...commentActions, ...postActions };
}
