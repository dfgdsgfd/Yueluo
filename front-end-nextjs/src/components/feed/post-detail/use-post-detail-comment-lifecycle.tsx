/* eslint-disable react-hooks/exhaustive-deps -- Stable refs and state setters arrive through the controller context. */
import {
getCommentReplies,
getOptionalDislikeStatus,
getOptionalFollowStatus,
getOptionalReportStatus,
getPostComments
} from "@/lib/api";
import {
useCallback,
useEffect
} from "react";
import {
toast
} from "sonner";
import {
findCommentPath,
getCommentElementId,
mapDetailComment,
mergeCommentReplies,
updateCommentInTree
} from "./comments";
import {
getRequestedCommentId,
normalizeTargetCommentId,
} from "./post-detail-target";
import {
DetailComment
} from "./post-detail-types";
import type { usePostDetailBase } from "./use-post-detail-base";

const COMMENT_REPLIES_PAGE_LIMIT = 20;

export function usePostDetailCommentLifecycle(context: ReturnType<typeof usePostDetailBase>) {
  const { authorUserId, commentInputExpanded, commentInputRef, commentState, commentText, comments, drawerContentRef, emblaApi, expandedForTargetRef, highlightedCommentTimerRef, imageKey, isCommentInputFocused, keyboardOffset, locale, mode, open, postId, replyTarget, scrolledCommentRef, setActionState, setActionsOpen, setCommentDraft, setCommentState, setFollowState, setHighlightedCommentId, setIsCommentInputFocused, setReportDescription, setReportReason, setSelectedImageIndex, setShareOpen, t, targetCommentId, targetCommentParentId } = context;
const loadCommentRepliesForTarget = useCallback(async (comment: DetailComment) => {
    if (!postId || comment.repliesStatus === "loading") {
      return;
    }

    const loadedReplyCount = comment.replies?.length ?? 0;
    const hasMoreReplies = comment.replyCount > loadedReplyCount;
    if (comment.repliesStatus === "loaded" && !hasMoreReplies) {
      setCommentState((currentState) => {
        if (currentState.postId !== postId) {
          return currentState;
        }

        return {
          ...currentState,
          comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => ({
            ...currentComment,
            repliesExpanded: !currentComment.repliesExpanded,
          })),
        };
      });
      return;
    }

    const nextPage = (comment.repliesPage ?? 0) + 1;
    setCommentState((currentState) => {
      if (currentState.postId !== postId) {
        return currentState;
      }

      return {
        ...currentState,
        comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => ({
          ...currentComment,
          repliesExpanded: true,
          repliesStatus: "loading",
        })),
      };
    });

    try {
      const payload = await getCommentReplies(comment.id, nextPage, COMMENT_REPLIES_PAGE_LIMIT);
      const replies = (payload.comments ?? []).map((reply) =>
        mapDetailComment(
          reply,
          t("card.unknownAuthor"),
          t("drawer.recentDate"),
          locale,
        ),
      );
      const totalReplyCount =
        typeof payload.pagination?.total === "number"
          ? payload.pagination.total
          : undefined;

      setCommentState((currentState) => {
        if (currentState.postId !== postId) {
          return currentState;
        }

        return {
          ...currentState,
          comments: updateCommentInTree(currentState.comments, comment.id, (currentComment) => {
            const mergedReplies = mergeCommentReplies(
              currentComment.replies ?? [],
              replies,
              nextPage > 1,
            );

            return {
              ...currentComment,
              replies: mergedReplies,
              repliesExpanded: true,
              repliesPage: Math.max(currentComment.repliesPage ?? 0, nextPage),
              repliesStatus: "loaded",
              replyCount: Math.max(
                totalReplyCount ?? (replies.length === 0 ? 0 : currentComment.replyCount),
                mergedReplies.length,
              ),
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
            repliesExpanded: (currentComment.replies?.length ?? 0) > 0
              ? currentComment.repliesExpanded
              : false,
            repliesStatus: "error",
          })),
        };
      });
      toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
    }
  }, [locale, postId, t]);

const ensureCommentInputVisible = useCallback(() => {
    const input = commentInputRef.current;
    if (!input) {
      return;
    }

    const viewport = window.visualViewport;
    const viewportTop = viewport?.offsetTop ?? 0;
    const viewportBottom = viewport ? viewport.offsetTop + viewport.height : window.innerHeight;
    const rect = input.getBoundingClientRect();
    const bottomGap = viewportBottom - rect.bottom;
    const topGap = rect.top - viewportTop;

    if (bottomGap < 12 || topGap < 0) {
      input.scrollIntoView({ block: "nearest", inline: "nearest" });
    }
  }, []);

useEffect(() => {
    const input = commentInputRef.current;
    if (!input) {
      return;
    }

    input.style.height = "44px";
    input.style.height = `${Math.min(Math.max(input.scrollHeight, 44), 80)}px`;
  }, [commentInputExpanded, commentText]);

useEffect(() => {
    if (!isCommentInputFocused) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      ensureCommentInputVisible();
    });

    return () => window.cancelAnimationFrame(frame);
  }, [commentInputExpanded, ensureCommentInputVisible, isCommentInputFocused, keyboardOffset, replyTarget?.id]);

useEffect(() => {
    if (!open || mode === "page") {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      drawerContentRef.current?.focus({ preventScroll: true });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [mode, open, postId]);

// Sync browser URL with the open post so users can share / bookmark it.
  // In page mode the URL is already correct, so skip this effect.
  useEffect(() => {
    if (mode !== "drawer" || !open || !postId) {
      return;
    }

    const prevUrl = window.location.href;
    const url = new URL(window.location.href);
    url.pathname = "/post";
    url.searchParams.set("id", String(postId));
    window.history.replaceState(window.history.state, "", url.toString());

    return () => {
      // Restore the previous URL when the drawer closes.
      window.history.replaceState(window.history.state, "", prevUrl);
    };
  }, [mode, open, postId]);

useEffect(() => {
    if (!emblaApi) {
      return;
    }

    const updateSelectedImage = () => {
      setSelectedImageIndex(emblaApi.selectedScrollSnap());
    };

    updateSelectedImage();
    emblaApi.on("select", updateSelectedImage);
    emblaApi.on("reInit", updateSelectedImage);

    return () => {
      emblaApi.off("select", updateSelectedImage);
      emblaApi.off("reInit", updateSelectedImage);
    };
  }, [emblaApi]);

useEffect(() => {
    if (!open || !emblaApi) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      emblaApi.reInit();
      emblaApi.scrollTo(0, true);
      setSelectedImageIndex(0);
    });

    return () => window.cancelAnimationFrame(frame);
  }, [emblaApi, imageKey, open, postId]);

useEffect(() => {
    if (!open || !postId) {
      return;
    }

    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) {
        return;
      }

      setCommentDraft({ postId, replyTarget: null, text: "" });
      setIsCommentInputFocused(false);
      setCommentState({
        comments: [],
        countDelta: 0,
        postId,
        status: "loading",
      });
    });

    getPostComments(postId)
      .then((payload) => {
        if (cancelled) {
          return;
        }

        setCommentState({
          comments: (payload.comments ?? []).map((comment) =>
            mapDetailComment(
              comment,
              t("card.unknownAuthor"),
              t("drawer.recentDate"),
              locale,
            ),
          ),
          countDelta: 0,
          postId,
          status: "loaded",
        });
      })
      .catch((error) => {
        if (!cancelled) {
          setCommentState({
            comments: [],
            countDelta: 0,
            postId,
            status: "error",
          });
          toast.error(error instanceof Error ? error.message : t("feed.loadFailed"));
        }
      });

    return () => {
      cancelled = true;
    };
  }, [locale, open, postId, t]);

useEffect(() => {
    if (!open || !postId || commentState.status !== "loaded") {
      return;
    }

    const requestedCommentId = getRequestedCommentId(targetCommentId);
    if (!requestedCommentId) {
      return;
    }

    const requestedParentCommentId = normalizeTargetCommentId(targetCommentParentId);
    const scrollKey = `${postId}:${requestedCommentId}`;
    const targetPath = findCommentPath(comments, requestedCommentId);

    if (targetPath.length > 0) {
      const collapsedParent = targetPath
        .slice(0, -1)
        .find((comment) => !comment.repliesExpanded);

      if (collapsedParent) {
        expandCommentForTarget(collapsedParent, scrollKey);
        return;
      }

      if (scrolledCommentRef.current === scrollKey) {
        return;
      }

      const element = document.getElementById(getCommentElementId(requestedCommentId));
      if (!element) {
        return;
      }

      scrolledCommentRef.current = scrollKey;
      window.requestAnimationFrame(() => {
        setHighlightedCommentId(requestedCommentId);
        element.scrollIntoView({ behavior: "smooth", block: "center", inline: "nearest" });

        if (highlightedCommentTimerRef.current) {
          window.clearTimeout(highlightedCommentTimerRef.current);
        }
        highlightedCommentTimerRef.current = window.setTimeout(() => {
          setHighlightedCommentId((currentId) =>
            currentId === requestedCommentId ? null : currentId,
          );
        }, 2200);
      });
      return;
    }

    if (!requestedParentCommentId) {
      return;
    }

    const parentPath = findCommentPath(comments, requestedParentCommentId);
    const parentComment = parentPath[parentPath.length - 1];
    if (!parentComment || parentComment.repliesStatus === "loading") {
      return;
    }

    if (parentComment.repliesStatus !== "loaded" || !parentComment.repliesExpanded) {
      expandCommentForTarget(parentComment, scrollKey);
    }

    function expandCommentForTarget(comment: DetailComment, key: string) {
      const commentId = String(comment.id);
      if (expandedForTargetRef.current === `${key}:${commentId}`) {
        return;
      }

      expandedForTargetRef.current = `${key}:${commentId}`;
      void Promise.resolve(loadCommentRepliesForTarget(comment)).finally(() => {
        if (expandedForTargetRef.current === `${key}:${commentId}`) {
          expandedForTargetRef.current = null;
        }
      });
    }
  }, [
    commentState.status,
    comments,
    loadCommentRepliesForTarget,
    open,
    postId,
    targetCommentId,
    targetCommentParentId,
  ]);

useEffect(() => {
    return () => {
      if (highlightedCommentTimerRef.current) {
        window.clearTimeout(highlightedCommentTimerRef.current);
      }
    };
  }, []);

useEffect(() => {
    let cancelled = false;

    if (!open || !authorUserId) {
      queueMicrotask(() => {
        if (!cancelled) {
          setFollowState({ isFollowing: false, status: "idle" });
        }
      });
      return () => {
        cancelled = true;
      };
    }

    queueMicrotask(() => {
      if (!cancelled) {
        setFollowState({ isFollowing: false, status: "loading", userId: authorUserId });
      }
    });

    getOptionalFollowStatus(authorUserId)
      .then((payload) => {
        if (cancelled) {
          return;
        }

        setFollowState({
          buttonType: payload?.buttonType,
          isFollowing: payload?.isFollowing ?? payload?.followed ?? false,
          status: "ready",
          userId: authorUserId,
        });
      })
      .catch(() => {
        if (!cancelled) {
          setFollowState({
            isFollowing: false,
            status: "error",
            userId: authorUserId,
          });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [authorUserId, open]);

useEffect(() => {
    let cancelled = false;

    if (!open || !postId) {
      queueMicrotask(() => {
        if (!cancelled) {
          setActionsOpen(false);
          setActionState({
            disliked: false,
            dislikeStatus: "idle",
            reported: false,
            reportStatus: "idle",
          });
          setReportReason("spam");
          setReportDescription("");
          setShareOpen(false);
        }
      });
      return () => {
        cancelled = true;
      };
    }

    queueMicrotask(() => {
      if (!cancelled) {
        setActionsOpen(false);
        setActionState({
          disliked: false,
          dislikeStatus: "loading",
          postId,
          reported: false,
          reportStatus: "loading",
        });
        setReportReason("spam");
        setReportDescription("");
        setShareOpen(false);
      }
    });

    Promise.allSettled([
      getOptionalDislikeStatus(postId),
      getOptionalReportStatus({ targetType: "post", targetId: postId }),
    ]).then(([dislikeResult, reportResult]) => {
      if (cancelled) {
        return;
      }

      setActionState({
        disliked:
          dislikeResult.status === "fulfilled"
            ? dislikeResult.value?.disliked ?? false
            : false,
        dislikeStatus: dislikeResult.status === "rejected" ? "error" : "ready",
        postId,
        reported:
          reportResult.status === "fulfilled"
            ? reportResult.value?.reported ?? false
            : false,
        reportStatus: reportResult.status === "rejected" ? "error" : "ready",
      });
    });

    return () => {
      cancelled = true;
    };
  }, [open, postId]);

  return { ensureCommentInputVisible, loadCommentRepliesForTarget };
}
