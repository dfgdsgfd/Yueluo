"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { PostDetailDrawer } from "@/components/feed/post-detail-drawer";
import { ApiUnauthorizedError, getPostDetail } from "@/lib/api";
import { isAccessBlockApiError } from "@/lib/api/core/access-block";
import { emitPointsAwardFromResponsePayload } from "@/lib/points-award-events";
import type { FeedPost } from "@/lib/types";
import { LoginUnlockDialog } from "./login-unlock-dialog";
import { findFeedPostInQueryCache, updateFeedPostState } from "./post-feed-state";

type PostDetailPageClientProps = {
  postId: FeedPost["id"];
  initialPost?: FeedPost | null;
  initialAccessBlocked?: boolean;
  initialPostComplete?: boolean;
  mode?: "page" | "modal" | "drawer";
  onClose?: () => void;
};

export function PostDetailPageClient(props: PostDetailPageClientProps) {
  return <PostDetailPageClientInner key={String(props.postId)} {...props} />;
}

function PostDetailPageClientInner({
  postId,
  initialAccessBlocked = false,
  initialPost = null,
  initialPostComplete = false,
  mode = "page",
  onClose,
}: PostDetailPageClientProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const t = useTranslations();
  const normalizedPostId = String(postId);
  const [post, setPost] = useState<FeedPost | null>(
    () => initialAccessBlocked ? null : initialPost ?? findFeedPostInQueryCache(queryClient, postId),
  );
  const [loginUnlockOpen, setLoginUnlockOpen] = useState(false);
  const targetComment = useMemo(
    () => ({
      id: getSearchParamValue(searchParams, "comment") ??
        getSearchParamValue(searchParams, "commentId") ??
        getSearchParamValue(searchParams, "comment_id"),
      parentId: getSearchParamValue(searchParams, "parentComment") ??
        getSearchParamValue(searchParams, "parentCommentId") ??
        getSearchParamValue(searchParams, "parent_comment_id"),
    }),
    [searchParams],
  );
  const closeBlockedPostDetail = useCallback(() => {
    setPost(null);
    updateFeedPostState(queryClient, normalizedPostId, () => null);
    if (mode === "drawer") {
      onClose?.();
      return;
    }
    router.replace("/");
  }, [mode, normalizedPostId, onClose, queryClient, router]);

  useEffect(() => {
    if (initialPost) {
      emitPointsAwardFromResponsePayload(initialPost);
    }
  }, [initialPost]);

  useEffect(() => {
    if (!initialAccessBlocked) {
      return;
    }
    updateFeedPostState(queryClient, normalizedPostId, () => null);
    if (mode === "drawer") {
      onClose?.();
      return;
    }
    router.replace("/");
  }, [initialAccessBlocked, mode, normalizedPostId, onClose, queryClient, router]);

  useEffect(() => {
    if (initialAccessBlocked || initialPostComplete) {
      return;
    }
    const hasFallbackContent = Boolean(
      initialPost ?? findFeedPostInQueryCache(queryClient, normalizedPostId),
    );

    let cancelled = false;
    const controller = new AbortController();

    getPostDetail(normalizedPostId, { signal: controller.signal })
      .then((nextPost) => {
        if (cancelled) {
          return;
        }

        setPost(nextPost);
        updateFeedPostState(queryClient, nextPost.id, () => nextPost);
      })
      .catch((error) => {
        if (cancelled || isAbortError(error)) {
          return;
        }

        if (error instanceof ApiUnauthorizedError) {
          setLoginUnlockOpen(true);
          return;
        }

        if (isAccessBlockApiError(error)) {
          closeBlockedPostDetail();
          return;
        }

        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
        if (!hasFallbackContent) {
          router.replace("/");
        }
      });

    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [closeBlockedPostDetail, initialAccessBlocked, initialPost, initialPostComplete, normalizedPostId, queryClient, router, t]);

  return (
    <>
      <PostDetailDrawer
        post={post}
        open={true}
        mode={mode}
        onOpenChange={(open) => {
          if (!open) {
            if (mode === "drawer") {
              onClose?.();
              return;
            }

            if (mode === "modal") {
              router.back();
              return;
            }

            router.push("/");
          }
        }}
        targetCommentId={targetComment.id}
        targetCommentParentId={targetComment.parentId}
      />
      <LoginUnlockDialog
        open={loginUnlockOpen}
        onOpenChange={(open) => {
          setLoginUnlockOpen(open);
          if (!open && !post) {
            router.push("/");
          }
        }}
      />
    </>
  );
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}

function getSearchParamValue(params: Pick<URLSearchParams, "get">, key: string) {
  const value = params.get(key)?.trim();
  return value || null;
}
