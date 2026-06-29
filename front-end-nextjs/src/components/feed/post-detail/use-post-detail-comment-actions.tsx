import {
ApiUnauthorizedError,
createComment,
deleteComment,
toggleCommentLike
} from "@/lib/api";
import {
type FormEvent
} from "react";
import {
toast
} from "sonner";
import {
updateFeedPostState
} from "../post-feed-state";
import {
commentTreeContainsId,
insertReplyIntoComments,
mapDetailComment,
removeCommentFromTree,
updateCommentInTree
} from "./comments";
import {
formatDeleteCommentSuccess
} from "./post-detail-formatters";
import {
DetailComment
} from "./post-detail-types";
import type { usePostDetailBase } from "./use-post-detail-base";
import type { usePostDetailCommentLifecycle } from "./use-post-detail-comment-lifecycle";

type CommentActionContext = ReturnType<typeof usePostDetailBase> & ReturnType<typeof usePostDetailCommentLifecycle>;

export function usePostDetailCommentActions(context: CommentActionContext) {
  const { commentInputRef, commentText, currentUser, isSubmittingComment, locale, mutatingCommentDeleteIds, mutatingCommentLikeIds, openLoginUnlock, post, postId, queryClient, replyTarget, setCommentDraft, setCommentState, setIsCommentInputFocused, setIsSubmittingComment, setMutatingCommentDeleteIds, setMutatingCommentLikeIds, t, ensureCommentInputVisible, loadCommentRepliesForTarget } = context;
async function handleSubmitComment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!post || isSubmittingComment) {
      return;
    }
    if (!currentUser) {
      openLoginUnlock();
      return;
    }

    const content = commentText.trim();
    if (!content) {
      return;
    }

    const currentReplyTarget = replyTarget;
    const optimisticId = `pending-${crypto.randomUUID()}`;
    const optimisticComment: DetailComment = {
      id: optimisticId,
      author: currentUser?.nickname?.trim() || t("card.unknownAuthor"),
      avatar: currentUser?.avatar ?? null,
      body: content,
      likes: 0,
      meta: t("drawer.recentDate"),
      ownerIds: [
        currentUser?.xise_id,
        currentUser?.user_id,
        currentUser?.id,
      ].filter((value): value is string | number => value !== null && value !== undefined).map(String),
      parentId: currentReplyTarget?.id ?? null,
      replies: [],
      repliesExpanded: false,
      repliesStatus: "idle",
      replyCount: 0,
      status: "pending",
      userId: String(currentUser?.xise_id ?? currentUser?.user_id ?? currentUser?.id ?? "current-user"),
    };
    setCommentState((currentState) => {
      const currentComments =
        currentState.postId === post.id ? currentState.comments : [];
      const currentDelta =
        currentState.postId === post.id ? currentState.countDelta : 0;
      return {
        comments: currentReplyTarget
          ? insertReplyIntoComments(currentComments, currentReplyTarget.id, optimisticComment)
          : [optimisticComment, ...currentComments],
        countDelta: currentDelta + 1,
        postId: post.id,
        status: "loaded",
      };
    });
    setCommentDraft({ postId: post.id, replyTarget: null, text: "" });
    setIsCommentInputFocused(false);
    commentInputRef.current?.blur();
    updateFeedPostState(queryClient, post.id, (currentPost) => ({
      ...currentPost,
      comment_count: Math.max(0, (currentPost.comment_count ?? 0) + 1),
    }));
    setIsSubmittingComment(true);
    try {
      const comment = await createComment(post.id, content, currentReplyTarget?.id, {
        redirectOnUnauthorized: false,
      });
      const mappedComment = mapDetailComment(
        comment,
        t("card.unknownAuthor"),
        t("drawer.recentDate"),
        locale,
      );
      setCommentState((currentState) => {
        return {
          ...currentState,
          comments: updateCommentInTree(
            currentState.postId === post.id ? currentState.comments : [],
            optimisticId,
            () => mappedComment,
          ),
          postId: post.id,
          status: "loaded",
        };
      });
    } catch (error) {
      setCommentState((currentState) => ({
        ...currentState,
        comments: updateCommentInTree(currentState.comments, optimisticId, (comment) => ({
          ...comment,
          status: "failed",
        })),
      }));
      if (error instanceof ApiUnauthorizedError) {
        openLoginUnlock();
      } else {
        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
      }
      updateFeedPostState(queryClient, post.id, (currentPost) => ({
        ...currentPost,
        comment_count: Math.max(0, (currentPost.comment_count ?? 0) - 1),
      }));
    } finally {
      setIsSubmittingComment(false);
    }
  }

async function handleRetryComment(comment: DetailComment) {
    if (!post || isSubmittingComment || comment.status !== "failed") {
      return;
    }

    setIsSubmittingComment(true);
    setCommentState((currentState) => ({
      ...currentState,
      comments: updateCommentInTree(currentState.comments, comment.id, (item) => ({
        ...item,
        status: "pending",
      })),
    }));
    try {
      const createdComment = await createComment(
        post.id,
        comment.body,
        comment.parentId ?? undefined,
        { redirectOnUnauthorized: false },
      );
      const mappedComment = mapDetailComment(
        createdComment,
        t("card.unknownAuthor"),
        t("drawer.recentDate"),
        locale,
      );
      setCommentState((currentState) => ({
        ...currentState,
        comments: updateCommentInTree(currentState.comments, comment.id, () => mappedComment),
      }));
    } catch (error) {
      setCommentState((currentState) => ({
        ...currentState,
        comments: updateCommentInTree(currentState.comments, comment.id, (item) => ({
          ...item,
          status: "failed",
        })),
      }));
      if (error instanceof ApiUnauthorizedError) {
        openLoginUnlock();
      } else {
        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
      }
    } finally {
      setIsSubmittingComment(false);
    }
  }

function handleCancelCommentInput() {
    if (post) {
      setCommentDraft({ postId: post.id, replyTarget: null, text: "" });
    }
    setIsCommentInputFocused(false);
    commentInputRef.current?.blur();
  }

function focusCommentInput({ immediate = false }: { immediate?: boolean } = {}) {
    const focusInput = () => {
      const input = commentInputRef.current;
      if (!input) {
        return;
      }

      input.focus({ preventScroll: true });
      ensureCommentInputVisible();
      window.setTimeout(ensureCommentInputVisible, 180);
    };

    if (immediate) {
      focusInput();
      return;
    }

    requestAnimationFrame(focusInput);
  }

function handleOpenCommentInput(
    reply?: DetailComment,
    options: { immediateFocus?: boolean } = {},
  ) {
    if (!currentUser) {
      openLoginUnlock();
      return;
    }
    if (post && reply) {
      setCommentDraft((currentDraft) => ({
        postId: post.id,
        replyTarget: { author: reply.author, id: reply.id },
        text: currentDraft.postId === post.id ? currentDraft.text : "",
      }));
    }
    setIsCommentInputFocused(true);
    focusCommentInput({ immediate: options.immediateFocus });
  }

async function handleLoadCommentReplies(comment: DetailComment) {
    await loadCommentRepliesForTarget(comment);
  }

async function handleToggleCommentLike(comment: DetailComment) {
    if (!currentUser) {
      openLoginUnlock();
      return;
    }
    const commentId = String(comment.id);
    if (mutatingCommentLikeIds.has(commentId)) {
      return;
    }

    const nextLiked = !comment.liked;
    setMutatingCommentLikeIds((currentIds) => new Set(currentIds).add(commentId));
    setCommentState((currentState) => {
      if (currentState.postId !== postId) {
        return currentState;
      }

      return {
        ...currentState,
        comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => ({
          ...currentComment,
          liked: nextLiked,
          likes: Math.max(0, currentComment.likes + (nextLiked ? 1 : -1)),
        })),
      };
    });

    try {
      const result = await toggleCommentLike(comment.id, {
        redirectOnUnauthorized: false,
      });
      setCommentState((currentState) => {
        if (currentState.postId !== postId) {
          return currentState;
        }

        return {
          ...currentState,
          comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => {
            const correctedLiked = result.liked;
            const correction =
              correctedLiked === nextLiked ? 0 : correctedLiked ? 1 : -1;

            return {
              ...currentComment,
              liked: correctedLiked,
              likes: Math.max(0, currentComment.likes + correction),
            };
          }),
        };
      });
    } catch (error) {
      setCommentState((currentState) => {
        if (currentState.postId !== postId) {
          return currentState;
        }

        return {
          ...currentState,
          comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => ({
            ...currentComment,
            liked: comment.liked,
            likes: comment.likes,
          })),
        };
      });
      if (error instanceof ApiUnauthorizedError) {
        openLoginUnlock();
      } else {
        toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
      }
    } finally {
      setMutatingCommentLikeIds((currentIds) => {
        const nextIds = new Set(currentIds);
        nextIds.delete(commentId);
        return nextIds;
      });
    }
  }

async function handleDeleteComment(comment: DetailComment) {
    const currentPostId = postId;
    if (currentPostId === undefined) {
      return;
    }

    const commentId = String(comment.id);
    if (mutatingCommentDeleteIds.has(commentId)) {
      return;
    }

    setMutatingCommentDeleteIds((currentIds) => new Set(currentIds).add(commentId));
    try {
      const result = await deleteComment(comment.id);
      const deletedCount = Math.max(1, Number(result.deletedCount ?? 1));
      const shouldClearReplyTarget = replyTarget
        ? commentTreeContainsId(comment, replyTarget.id)
        : false;
      setCommentState((currentState) => {
        if (currentState.postId !== currentPostId) {
          return currentState;
        }

        const removal = removeCommentFromTree(currentState.comments, comment.id);
        return {
          ...currentState,
          comments: removal.comments,
          countDelta: currentState.countDelta - deletedCount,
        };
      });
      updateFeedPostState(queryClient, currentPostId, (currentPost) => ({
        ...currentPost,
        comment_count: Math.max(0, (currentPost.comment_count ?? 0) - deletedCount),
      }));
      if (shouldClearReplyTarget) {
        setCommentDraft((currentDraft) => ({
          postId: currentDraft.postId,
          replyTarget: null,
          text: currentDraft.text,
        }));
      }
      toast.success(formatDeleteCommentSuccess(locale));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    } finally {
      setMutatingCommentDeleteIds((currentIds) => {
        const nextIds = new Set(currentIds);
        nextIds.delete(commentId);
        return nextIds;
      });
    }
  }

  return { focusCommentInput, handleCancelCommentInput, handleDeleteComment, handleLoadCommentReplies, handleOpenCommentInput, handleRetryComment, handleSubmitComment, handleToggleCommentLike };
}
