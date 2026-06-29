import type { FeedPost } from "./types";

export function getPostAuthorUserId(post: FeedPost) {
  const authorAccount = normalizeIdentifier(post.author_account);

  if (authorAccount) {
    return authorAccount;
  }

  if (post.user_id !== undefined && post.user_id !== 0) {
    return String(post.user_id);
  }

  return slugifyUserId(getPostAuthorName(post));
}

export function getUserHrefFromPost(post: FeedPost) {
  return getUserHref(getPostAuthorUserId(post));
}

export function getUserHref(userId: string) {
  return `/user/${encodeURIComponent(userId)}`;
}

function slugifyUserId(value: string) {
  const normalized = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

  return normalized || "yuem-user";
}

function getPostAuthorName(post: FeedPost) {
  return post.author ?? post.nickname ?? post.author_account ?? "Yuem User";
}

function normalizeIdentifier(value: string | null | undefined) {
  const normalized = value?.trim();
  return normalized || null;
}
