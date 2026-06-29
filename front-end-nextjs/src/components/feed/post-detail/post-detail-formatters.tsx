"use client";
import type {
  AuthUser,
  FeedPost,
  PostAttachment
} from "@/lib/types";

export function isCurrentUserPostOwner(post: FeedPost, user: AuthUser | null) {
  if (!user) {
    return false;
  }
  const currentUserIds = uniqueDefinedStrings([user.id, user.user_id, user.xise_id]);
  const postOwnerIds = uniqueDefinedStrings([post.user_id, post.author_auto_id, post.author_account]);
  return postOwnerIds.some((ownerId) => currentUserIds.includes(ownerId));
}


export async function downloadImage(url: string, title: string) {
  const response = await fetch(url, { credentials: "include" });
  if (!response.ok) {
    throw new Error("download failed");
  }
  const blob = await response.blob();
  const extension = imageExtensionFromType(blob.type, url);
  const filename = `${safeDownloadName(title)}${extension}`;
  const file = new File([blob], filename, { type: blob.type || "image/jpeg" });
  if (navigator.canShare?.({ files: [file] })) {
    await navigator.share({ files: [file], title: filename });
    return;
  }
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
}


export function safeDownloadName(value: string) {
  const normalized = value.trim().replace(/[\\/:*?"<>|]+/g, "-").slice(0, 48);
  return normalized || "yuem-image";
}


export function imageExtensionFromType(contentType: string, url: string) {
  if (contentType.includes("png")) return ".png";
  if (contentType.includes("webp")) return ".webp";
  if (contentType.includes("gif")) return ".gif";
  const cleanPath = url.split("?")[0] ?? "";
  const match = cleanPath.match(/\.(jpe?g|png|webp|gif)$/i);
  return match ? `.${match[1].toLowerCase()}` : ".jpg";
}


export function uniqueDefinedStrings(values: Array<number | string | null | undefined>) {
  return Array.from(
    new Set(
      values
        .map((value) => (value === null || value === undefined ? "" : String(value).trim()))
        .filter(Boolean),
    ),
  );
}


export function buildTags(post: FeedPost) {
  const sourceTags = post.tags?.map((tag) => tag.name) ?? [];
  const categoryTag = post.category ? [post.category] : [];
  const tags = [...sourceTags, ...categoryTag];

  return Array.from(new Set(tags)).slice(0, 5);
}


export function normalizeLineBreaks(value: string) {
  return value.replace(/<br\s*\/?>/gi, "\n");
}


export function formatDetailDate(date: string | undefined, fallback: string, locale: string) {
  if (!date) {
    return fallback;
  }

  const parsedDate = new Date(date);

  if (Number.isNaN(parsedDate.getTime())) {
    return fallback;
  }

  const diffMs = parsedDate.getTime() - Date.now();

  const absDiffMs = Math.abs(diffMs);
  const minuteMs = 60 * 1000;
  const hourMs = 60 * minuteMs;
  const dayMs = 24 * hourMs;
  const yearMs = 365 * dayMs;

  if (absDiffMs < minuteMs) {
    return fallback;
  }

  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "always" });

  if (absDiffMs < hourMs) {
    return formatter.format(-Math.trunc(absDiffMs / minuteMs), "minute");
  }

  if (absDiffMs < dayMs) {
    return formatter.format(-Math.trunc(absDiffMs / hourMs), "hour");
  }

  if (absDiffMs < yearMs) {
    return formatter.format(-Math.trunc(absDiffMs / dayMs), "day");
  }

  return formatter.format(-Math.trunc(absDiffMs / yearMs), "year");
}


export function formatCommentDate(date: string | undefined, fallback: string, locale?: string) {
  if (!date) {
    return fallback;
  }

  const parsedDate = new Date(date);

  if (Number.isNaN(parsedDate.getTime())) {
    return fallback;
  }

  return new Intl.DateTimeFormat(locale || undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(parsedDate);
}


export function isAudioAttachment(attachment: PostAttachment) {
  const value = `${attachment.filename ?? ""} ${attachment.url}`.toLowerCase();
  return /\.(mp3|m4a|wav|flac|aac|ogg|opus)(\?|#|$)/.test(value);
}


export function formatAttachmentSize(size?: number | null) {
  if (!size || size <= 0) {
    return null;
  }

  const units = ["B", "KB", "MB", "GB"];
  let value = size;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const precision = unitIndex === 0 || value >= 10 ? 0 : 1;
  return `${value.toFixed(precision)} ${units[unitIndex]}`;
}


export function formatReplyPlaceholder(author: string, locale: string) {
  return locale.startsWith("zh") ? `回复 ${author}：` : `Reply to ${author}...`;
}


export function formatReplyStatus(author: string, locale: string) {
  return locale.startsWith("zh") ? `回复 ${author}` : `Replying to ${author}`;
}


export function formatReplyAction(locale: string) {
  return locale.startsWith("zh") ? "回复" : "Reply";
}


export function formatLikeComment(locale: string) {
  return locale.startsWith("zh") ? "点赞评论" : "Like comment";
}


export function formatUnlikeComment(locale: string) {
  return locale.startsWith("zh") ? "取消点赞评论" : "Unlike comment";
}


export function formatDeleteCommentAction(locale: string) {
  return locale.startsWith("zh") ? "删除" : "Delete";
}


export function formatDeleteCommentSuccess(locale: string) {
  return locale.startsWith("zh") ? "评论已删除" : "Comment deleted";
}


export function formatRepliesLoading(locale: string) {
  return locale.startsWith("zh") ? "回复加载中..." : "Loading replies...";
}


export function formatCollapseReplies(locale: string) {
  return locale.startsWith("zh") ? "收起回复" : "Collapse replies";
}


export function formatViewReplies(count: number, locale: string) {
  const safeCount = Math.max(count, 0);
  return locale.startsWith("zh")
    ? `查看 ${safeCount} 条回复`
    : `View ${safeCount} ${safeCount === 1 ? "reply" : "replies"}`;
}


export function formatCount(count: number) {
  if (count >= 10000) {
    return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`;
  }

  if (count >= 1000) {
    return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`;
  }

  return String(count);
}
