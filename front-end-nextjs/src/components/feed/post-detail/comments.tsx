"use client";
import type {
  AuthUser,
  BackendComment
} from "@/lib/types";
import {
  DetailComment
} from "./post-detail-types";
import {
  formatCommentDate,
  uniqueDefinedStrings
} from "./post-detail-formatters";

export function CommentSkeletonList() {
  return (
    <>
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="flex animate-pulse gap-3">
          <div className="size-9 shrink-0 rounded-full bg-white/[0.08]" />
          <div className="min-w-0 flex-1 space-y-2">
            <div className="h-3 w-24 rounded bg-white/[0.08]" />
            <div className="h-3 w-full rounded bg-white/[0.08]" />
            <div className="h-3 w-2/3 rounded bg-white/[0.08]" />
          </div>
        </div>
      ))}
    </>
  );
}


export function mapDetailComment(
  comment: BackendComment,
  authorFallback: string,
  dateFallback: string,
  locale?: string,
): DetailComment {
  const author = comment.nickname?.trim() || authorFallback;
  const userId = String(
    comment.user_display_id ??
      comment.user_auto_id ??
      comment.user_id ??
      "yuem-user",
  );

  return {
    id: comment.id,
    author,
    body: comment.content,
    meta: formatCommentDate(comment.created_at, dateFallback, locale),
    likes: comment.like_count ?? 0,
    liked: Boolean(comment.liked),
    parentId: comment.parent_id ?? null,
    ownerIds: buildCommentOwnerIds(comment, userId),
    replies: [],
    repliesExpanded: false,
    repliesPage: 0,
    repliesStatus: "idle",
    replyCount: comment.reply_count ?? 0,
    userId,
    avatar: comment.user_avatar ?? null,
  };
}


export function mergeCommentReplies(
  currentReplies: DetailComment[],
  incomingReplies: DetailComment[],
  append: boolean,
): DetailComment[] {
  const nextReplies = append ? [...currentReplies] : [...incomingReplies];
  const seen = new Set(nextReplies.map((reply) => String(reply.id)));
  const remainingReplies = append ? incomingReplies : currentReplies;

  for (const reply of remainingReplies) {
    const replyId = String(reply.id);
    if (seen.has(replyId)) {
      continue;
    }

    nextReplies.push(reply);
    seen.add(replyId);
  }

  return nextReplies;
}


export function updateCommentInTree(
  comments: DetailComment[],
  targetId: DetailComment["id"],
  update: (comment: DetailComment) => DetailComment,
): DetailComment[] {
  return comments.map((comment) => {
    if (idsMatch(comment.id, targetId)) {
      return update(comment);
    }

    if (!comment.replies?.length) {
      return comment;
    }

    return {
      ...comment,
      replies: updateCommentInTree(comment.replies, targetId, update),
    };
  });
}


export function insertReplyIntoComments(
  comments: DetailComment[],
  targetId: DetailComment["id"],
  reply: DetailComment,
): DetailComment[] {
  return updateCommentInTree(comments, targetId, (comment) => ({
    ...comment,
    replies: [reply, ...(comment.replies ?? [])],
    repliesExpanded: true,
    repliesPage: comment.repliesPage ?? 0,
    repliesStatus: "loaded",
    replyCount: Math.max(comment.replyCount + 1, (comment.replies?.length ?? 0) + 1),
  }));
}


export function removeCommentFromTree(
  comments: DetailComment[],
  targetId: DetailComment["id"],
): { comments: DetailComment[]; removedIds: Set<string> } {
  const removedIds = new Set<string>();

  function removeFromList(items: DetailComment[]): DetailComment[] {
    return items.flatMap((comment) => {
      if (idsMatch(comment.id, targetId)) {
        collectCommentIds(comment, removedIds);
        return [];
      }

      if (!comment.replies?.length) {
        return [comment];
      }

      const previousReplyCount = removedIds.size;
      const replies = removeFromList(comment.replies);
      const removedReplyCount = removedIds.size - previousReplyCount;

      return [
        {
          ...comment,
          replies,
          repliesExpanded: replies.length > 0 ? comment.repliesExpanded : false,
          replyCount: Math.max(0, comment.replyCount - removedReplyCount),
        },
      ];
    });
  }

  return { comments: removeFromList(comments), removedIds };
}


export function collectCommentIds(comment: DetailComment, ids: Set<string>) {
  ids.add(String(comment.id));
  for (const reply of comment.replies ?? []) {
    collectCommentIds(reply, ids);
  }
}


export function commentTreeContainsId(
  comment: DetailComment,
  targetId: DetailComment["id"],
): boolean {
  if (idsMatch(comment.id, targetId)) {
    return true;
  }

  return (comment.replies ?? []).some((reply) => commentTreeContainsId(reply, targetId));
}

export function findCommentPath(
  comments: DetailComment[],
  targetId: DetailComment["id"],
): DetailComment[] {
  for (const comment of comments) {
    if (idsMatch(comment.id, targetId)) {
      return [comment];
    }

    const replyPath = findCommentPath(comment.replies ?? [], targetId);
    if (replyPath.length > 0) {
      return [comment, ...replyPath];
    }
  }

  return [];
}

export function getCommentElementId(commentId: DetailComment["id"]) {
  return `comment-${String(commentId).replace(/[^a-zA-Z0-9_-]/g, "_")}`;
}


export function idsMatch(left: DetailComment["id"], right: DetailComment["id"]) {
  return String(left) === String(right);
}


export function buildCommentOwnerIds(comment: BackendComment, userId: string) {
  return uniqueDefinedStrings([
    comment.user_id,
    comment.user_auto_id,
    comment.user_display_id,
    userId,
  ]);
}


export function isCurrentUserCommentOwner(comment: DetailComment, user: AuthUser) {
  const currentUserIds = uniqueDefinedStrings([
    user.id,
    user.user_id,
    user.xise_id,
  ]);

  return comment.ownerIds.some((ownerId) => currentUserIds.includes(ownerId));
}
