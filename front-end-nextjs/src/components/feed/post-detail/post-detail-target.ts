export function getRequestedCommentId(value: string | number | null | undefined) {
  const normalized = normalizeTargetCommentId(value);
  if (normalized) {
    return normalized;
  }

  if (typeof window === "undefined") {
    return null;
  }

  const params = new URLSearchParams(window.location.search);
  const queryComment =
    params.get("comment") ?? params.get("commentId") ?? params.get("comment_id");
  const normalizedQueryComment = normalizeTargetCommentId(queryComment);
  if (normalizedQueryComment) {
    return normalizedQueryComment;
  }

  const hashComment = window.location.hash.replace(/^#comment-?/, "");
  return normalizeTargetCommentId(hashComment ? decodeURIComponent(hashComment) : null);
}

export function normalizeTargetCommentId(value: string | number | null | undefined) {
  if (value === undefined || value === null) {
    return null;
  }

  const normalized = String(value).trim();
  return normalized || null;
}
