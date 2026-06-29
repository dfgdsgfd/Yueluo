import {
ApiUnauthorizedError,
createReport,
deletePost,
followUser,
toggleCollect,
toggleDislike,
toggleLike,
unfollowUser
} from "@/lib/api";
import type {
FeedPost
} from "@/lib/types";
import { shareNativeUrl } from "@/lib/native-app";
import {
type FormEvent
} from "react";
import {
toast
} from "sonner";
import {
getPostCover
} from "../feed-utils";
import {
updateFeedPostState
} from "../post-feed-state";
import {
downloadImage
} from "./post-detail-formatters";
import { postEditRoute } from "./post-edit-route";
import type { usePostDetailBase } from "./use-post-detail-base";
import type { usePostDetailCommentActions } from "./use-post-detail-comment-actions";
import type { usePostDetailCommentLifecycle } from "./use-post-detail-comment-lifecycle";

type PostActionContext = ReturnType<typeof usePostDetailBase> & ReturnType<typeof usePostDetailCommentLifecycle> & ReturnType<typeof usePostDetailCommentActions>;

export function usePostDetailPostActions(context: PostActionContext) {
  const { actionState, authorUserId, currentUser, followState, images, isDeletingPost, isMutatingDislike, isMutatingFollow, isMutatingPostCollect, isMutatingPostLike, isSubmittingReport, mode, onOpenChange, openLoginUnlock, post, queryClient, reportDescription, reportReason, router, selectedImageIndex, setActionState, setFollowState, setIsDeletingPost, setIsMutatingDislike, setIsMutatingFollow, setIsMutatingPostCollect, setIsMutatingPostLike, setIsSubmittingReport, setReportDescription, setShareOpen, slideshowImages, t, updateLocalPost } = context;
  const safeImageIndex = slideshowImages.length > 0
    ? Math.min(selectedImageIndex, slideshowImages.length - 1)
    : 0;
  const cover = post ? getPostCover(post) : null;
async function handleFollowToggle() {
    if (!authorUserId || isMutatingFollow) {
      return;
    }

    const wasFollowing =
      followState.userId === authorUserId ? followState.isFollowing : false;
    const nextFollowing = !wasFollowing;

    setIsMutatingFollow(true);
    setFollowState({
      buttonType: nextFollowing ? "unfollow" : "follow",
      isFollowing: nextFollowing,
      status: "ready",
      userId: authorUserId,
    });

    try {
      const result = nextFollowing
        ? await followUser(authorUserId)
        : await unfollowUser(authorUserId);
      const isFollowing = result.isFollowing ?? result.followed ?? nextFollowing;

      setFollowState({
        buttonType: result.buttonType ?? (isFollowing ? "unfollow" : "follow"),
        isFollowing,
        status: "ready",
        userId: authorUserId,
      });
    } catch (error) {
      setFollowState({
        buttonType: wasFollowing ? "unfollow" : "follow",
        isFollowing: wasFollowing,
        status: "error",
        userId: authorUserId,
      });
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setIsMutatingFollow(false);
    }
  }

async function handleDislikeToggle() {
    if (!post || isMutatingDislike) {
      return;
    }

    const wasDisliked =
      actionState.postId === post.id ? actionState.disliked : false;
    const nextDisliked = !wasDisliked;

    setIsMutatingDislike(true);
    setActionState((currentState) => ({
      ...currentState,
      disliked: nextDisliked,
      dislikeStatus: "ready",
      postId: post.id,
    }));

    try {
      const result = await toggleDislike(post.id);
      setActionState((currentState) => ({
        ...currentState,
        disliked: result.disliked,
        dislikeStatus: "ready",
        postId: post.id,
      }));
      toast.success(
        result.disliked ? t("drawer.dislikeMarked") : t("drawer.dislikeRemoved"),
      );
    } catch (error) {
      setActionState((currentState) => ({
        ...currentState,
        disliked: wasDisliked,
        dislikeStatus: "error",
        postId: post.id,
      }));
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setIsMutatingDislike(false);
    }
  }

async function handlePostLike(targetPost: FeedPost) {
    if (!post || isMutatingPostLike || String(targetPost.id) !== String(post.id)) {
      return;
    }
    if (!currentUser) {
      openLoginUnlock();
      return;
    }

    const previousPost = post;
    const nextLiked = !previousPost.liked;
    const optimisticPost = {
      ...previousPost,
      liked: nextLiked,
      like_count: Math.max(0, (previousPost.like_count ?? 0) + (nextLiked ? 1 : -1)),
    };

    setIsMutatingPostLike(true);
    updateLocalPost(() => optimisticPost);
    updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
      ...currentPost,
      liked: nextLiked,
      like_count: Math.max(0, (currentPost.like_count ?? 0) + (nextLiked ? 1 : -1)),
    }));

    try {
      const result = await toggleLike(previousPost.id, {
        redirectOnUnauthorized: false,
      });
      updateLocalPost((currentPost) => ({
        ...currentPost,
        liked: result.liked,
        like_count: Math.max(
          0,
          (currentPost.like_count ?? 0) + (result.liked === nextLiked ? 0 : result.liked ? 1 : -1),
        ),
      }));
      updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
        ...currentPost,
        liked: result.liked,
        like_count: Math.max(
          0,
          (currentPost.like_count ?? 0) + (result.liked === nextLiked ? 0 : result.liked ? 1 : -1),
        ),
      }));
      toast.success(result.liked ? t("status.liked") : t("status.unliked"));
    } catch (error) {
      updateLocalPost(() => previousPost);
      updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
        ...currentPost,
        liked: previousPost.liked,
        like_count: previousPost.like_count,
      }));
      if (error instanceof ApiUnauthorizedError) {
        openLoginUnlock();
        return;
      }
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setIsMutatingPostLike(false);
    }
  }

async function handlePostCollect(targetPost: FeedPost) {
    if (!post || isMutatingPostCollect || String(targetPost.id) !== String(post.id)) {
      return;
    }
    if (!currentUser) {
      openLoginUnlock();
      return;
    }

    const previousPost = post;
    const nextCollected = !previousPost.collected;
    const optimisticPost = {
      ...previousPost,
      collected: nextCollected,
      collect_count: Math.max(0, (previousPost.collect_count ?? 0) + (nextCollected ? 1 : -1)),
    };

    setIsMutatingPostCollect(true);
    updateLocalPost(() => optimisticPost);
    updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
      ...currentPost,
      collected: nextCollected,
      collect_count: Math.max(0, (currentPost.collect_count ?? 0) + (nextCollected ? 1 : -1)),
    }));

    try {
      const result = await toggleCollect(previousPost.id, {
        redirectOnUnauthorized: false,
      });
      updateLocalPost((currentPost) => ({
        ...currentPost,
        collected: result.collected,
        collect_count: Math.max(
          0,
          (currentPost.collect_count ?? 0) +
            (result.collected === nextCollected ? 0 : result.collected ? 1 : -1),
        ),
      }));
      updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
        ...currentPost,
        collected: result.collected,
        collect_count: Math.max(
          0,
          (currentPost.collect_count ?? 0) +
            (result.collected === nextCollected ? 0 : result.collected ? 1 : -1),
        ),
      }));
    } catch (error) {
      updateLocalPost(() => previousPost);
      updateFeedPostState(queryClient, previousPost.id, (currentPost) => ({
        ...currentPost,
        collected: previousPost.collected,
        collect_count: previousPost.collect_count,
      }));
      if (error instanceof ApiUnauthorizedError) {
        openLoginUnlock();
        return;
      }
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setIsMutatingPostCollect(false);
    }
  }

async function handleSubmitReport(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!post || isSubmittingReport) {
      return;
    }

    const alreadyReported =
      actionState.postId === post.id ? actionState.reported : false;
    if (alreadyReported) {
      return;
    }

    setIsSubmittingReport(true);
    try {
      await createReport({
        targetType: "post",
        targetId: post.id,
        reason: reportReason,
        description: reportDescription,
      });
      setActionState((currentState) => ({
        ...currentState,
        postId: post.id,
        reported: true,
        reportStatus: "ready",
      }));
      setReportDescription("");
      toast.success(t("drawer.reportSubmitted"));
    } catch (error) {
      setActionState((currentState) => ({
        ...currentState,
        postId: post.id,
        reportStatus: "error",
      }));
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setIsSubmittingReport(false);
    }
  }

async function handleSharePost() {
    if (!post || typeof window === "undefined") {
      return;
    }

    try {
      if (await shareNativeUrl({ title: post.title, url: window.location.href })) {
        return;
      }
    } catch {
      // Fall back to the existing in-app share sheet.
    }
    setShareOpen(true);
  }

async function handleCopyPostLink() {
    if (!post || typeof window === "undefined") {
      return;
    }

    const shareUrl = window.location.href;
    try {
      await navigator.clipboard.writeText(shareUrl);
      toast.success(t("drawer.shareCopied"));
      setShareOpen(false);
    } catch {
      toast.error(t("drawer.shareFailed"));
    }
  }

function handleEditPost() {
    if (!post) {
      return;
    }
    if (mode === "drawer") {
      onOpenChange(false);
    }
    router.push(postEditRoute(post.id));
  }

async function handleDeletePost() {
    if (!post || isDeletingPost) {
      return;
    }
    if (!window.confirm("确认删除这篇笔记？")) {
      return;
    }
    if (!window.confirm("删除后不可恢复，是否继续删除？")) {
      return;
    }
    setIsDeletingPost(true);
    try {
      await deletePost(post.id);
      toast.success("帖子已删除");
      setShareOpen(false);
      updateFeedPostState(queryClient, post.id, () => null);
      onOpenChange(false);
      if (mode === "page") {
        router.push("/");
      } else if (mode === "drawer") {
        router.refresh();
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setIsDeletingPost(false);
    }
  }

async function handleSaveCurrentImage() {
    if (!post) {
      return;
    }
    const imageURL = images[safeImageIndex] ?? cover;
    if (!imageURL) {
      toast.error("暂无可保存的图片");
      return;
    }
    try {
      await downloadImage(imageURL, post.title || `post-${post.id}`);
      toast.success("图片保存已发起");
      setShareOpen(false);
    } catch {
      toast.error("保存失败，请长按图片保存");
    }
  }

  return { handleCopyPostLink, handleDeletePost, handleDislikeToggle, handleEditPost, handleFollowToggle, handlePostCollect, handlePostLike, handleSaveCurrentImage, handleSharePost, handleSubmitReport };
}
