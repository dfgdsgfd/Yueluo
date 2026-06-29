import type { BackendImage, FeedPost } from "@/lib/types";

export function getImageUrl(image: BackendImage | undefined) {
  if (!image) {
    return null;
  }

  return typeof image === "string" ? image : image.url;
}

export function getImagePreviewUrl(image: BackendImage | undefined) {
  if (!image) {
    return null;
  }

  return typeof image === "string"
    ? image
    : image.preview_url ?? image.thumbnail_url ?? image.thumb_url ?? image.url;
}

export function getPostImages(post: FeedPost) {
  const imageUrls = (post.images ?? [])
    .map((image) => getImageUrl(image))
    .filter((url): url is string => Boolean(url));

  const singleCover = post.image ?? post.cover_url ?? null;

  if (imageUrls.length > 0) {
    return imageUrls;
  }

  return singleCover ? [singleCover] : [];
}

export function getPostPreviewImages(post: FeedPost) {
  const imageUrls = (post.images ?? [])
    .map((image) => getImagePreviewUrl(image))
    .filter((url): url is string => Boolean(url));

  const singleCover =
    post.preview_url ??
    post.thumbnail_url ??
    post.thumb_url ??
    post.image ??
    post.cover_url ??
    null;

  if (imageUrls.length > 0) {
    return imageUrls;
  }

  return singleCover ? [singleCover] : [];
}

export function getPostCover(post: FeedPost) {
  return getPostImages(post)[0] ?? null;
}

export function getPostFeedCover(post: FeedPost) {
  const preferredCandidates = [
    post.thumbnail_url,
    post.thumb_url,
    post.preview_url,
    post.cover_url,
    ...(post.images ?? []).flatMap((image) =>
      typeof image === "string"
        ? [image]
        : [image.thumbnail_url, image.thumb_url, image.preview_url, image.url],
    ),
  ];

  return (
    preferredCandidates.find((url): url is string => isFeedThumbnailUrl(url)) ??
    getPostCover(post)
  );
}

export function getPostFeedCoverImage(post: FeedPost) {
  const cover = getPostFeedCover(post);
  if (!cover) {
    return undefined;
  }

  return (post.images ?? []).find(
    (image) =>
      typeof image !== "string" &&
      [image.thumbnail_url, image.thumb_url, image.preview_url, image.url].includes(cover),
  );
}

export function getPostCoverImage(post: FeedPost) {
  return (post.images ?? []).find((image) => Boolean(getImageUrl(image)));
}

function isFeedThumbnailUrl(url: string | null | undefined): url is string {
  if (!url) {
    return false;
  }

  try {
    const pathname = new URL(url, "https://feed.local").pathname;
    return /\/api\/file\/(?:covers|thumbnails)\//i.test(pathname);
  } catch {
    return false;
  }
}

export function getAuthorName(post: FeedPost, fallback: string) {
  return post.author ?? post.nickname ?? post.author_account ?? fallback;
}

export function getAuthorInitial(post: FeedPost, fallback: string) {
  return getAuthorName(post, fallback).trim().charAt(0).toUpperCase() || "Y";
}

export function isVideoPost(post: FeedPost) {
  const firstVideo = post.videos?.[0];

  return (
    post.type === 2 ||
    post.type === 3 ||
    Boolean(
      post.preview_video_url ||
        post.dash_url ||
        post.video_url ||
        firstVideo?.preview_video_url ||
        firstVideo?.dash_url ||
        firstVideo?.video_url,
    )
  );
}

export function isNovelPost(post: FeedPost) {
  const normalized = post.category?.trim().toLocaleLowerCase().replace(/[\s_-]+/g, "");
  return ["novel", "novels", "小说", "小說", "小説"].includes(normalized ?? "");
}

export function getPlayableVideoUrl(post: FeedPost) {
  const firstVideo = post.videos?.[0];

  return (
    post.dash_url ??
    firstVideo?.dash_url ??
    post.preview_video_url ??
    firstVideo?.preview_video_url ??
    post.video_url ??
    firstVideo?.video_url ??
    null
  );
}
