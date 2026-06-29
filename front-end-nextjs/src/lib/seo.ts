import { richTextToPlainText } from "./rich-text";
import type { FeedPost } from "./types";
import type { SiteProfile } from "./types/site";
import { getUserHrefFromPost } from "./users";

const DEFAULT_SITE_URL = "https://xse.yuelk.com";
export const SITE_NAME = "Yuem";
export const DEFAULT_DESCRIPTION = "Discover, create, and share moments with Yuem.";
export const DEFAULT_SITE_PROFILE: SiteProfile = {
  title: SITE_NAME,
  description: DEFAULT_DESCRIPTION,
  avatarUrl: null,
};

export function normalizeSiteProfile(value?: Partial<SiteProfile> | null): SiteProfile {
  return {
    title: nonEmptyText(value?.title, DEFAULT_SITE_PROFILE.title),
    description: nonEmptyText(value?.description, DEFAULT_SITE_PROFILE.description),
    avatarUrl: nonEmptyText(value?.avatarUrl, "") || null,
  };
}

export function getSiteUrl() {
  const raw =
    process.env.NEXT_PUBLIC_SITE_URL ??
    process.env.SITE_URL ??
    process.env.NEXT_PUBLIC_APP_URL ??
    process.env.VERCEL_PROJECT_PRODUCTION_URL ??
    process.env.VERCEL_URL ??
    DEFAULT_SITE_URL;
  const trimmed = raw.trim().replace(/\/$/, "");
  if (!trimmed) {
    return DEFAULT_SITE_URL;
  }
  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed;
  }
  return `https://${trimmed}`;
}

export function absoluteSiteUrl(path: string) {
  return new URL(path, getSiteUrl()).toString();
}

export function absoluteMediaUrl(value: string | null | undefined) {
  const trimmed = value?.trim();
  if (!trimmed) {
    return null;
  }
  try {
    const url = new URL(trimmed, getSiteUrl());
    if (url.pathname.startsWith("/api/file/")) {
      url.searchParams.delete("pvimg_exp");
      url.searchParams.delete("sign");
    }
    return url.toString();
  } catch {
    return null;
  }
}

export function postSeoTitle(post: FeedPost | null | undefined) {
  return post?.title?.trim() || SITE_NAME;
}

export function postSeoDescription(post: FeedPost | null | undefined) {
  const content = richTextToPlainText(post?.content).replace(/\s+/g, " ").trim();
  if (content) {
    return truncateText(content, 150);
  }

  const title = post?.title?.trim();
  const tags = postSeoTags(post);
  const fallback = [title, tags.map((tag) => `#${tag}`).join(" ")]
    .filter(Boolean)
    .join(" ")
    .trim();

  return fallback ? truncateText(fallback, 150) : DEFAULT_DESCRIPTION;
}

export function postSeoImage(post: FeedPost | null | undefined) {
  const firstImage = post?.images?.[0];
  const candidate =
    typeof firstImage === "string"
      ? firstImage
      : firstImage?.url ??
        firstImage?.preview_url ??
        firstImage?.thumbnail_url ??
        firstImage?.thumb_url ??
        post?.image ??
        post?.cover_url ??
        null;
  return absoluteMediaUrl(candidate);
}

export function postSeoAuthorName(post: FeedPost | null | undefined) {
  return (
    post?.author?.trim() ||
    post?.nickname?.trim() ||
    post?.author_account?.trim() ||
    SITE_NAME
  );
}

export function postSeoAuthorUrl(post: FeedPost | null | undefined) {
  return post ? absoluteSiteUrl(getUserHrefFromPost(post)) : getSiteUrl();
}

export function postSeoTags(post: FeedPost | null | undefined) {
  const tags = post?.tags
    ?.map((tag) => tag.name.trim())
    .filter(Boolean) ?? [];
  return uniqueTextValues(tags);
}

export function postSeoKeywords(post: FeedPost | null | undefined) {
  return uniqueTextValues([
    ...postSeoTags(post),
    post?.category?.trim() ?? "",
    post?.title?.trim() ?? "",
    SITE_NAME,
  ]).slice(0, 12);
}

export function postSeoJsonLd(
  post: FeedPost,
  {
    description,
    image,
    title,
    url,
  }: {
    description: string;
    image: string | null;
    title: string;
    url: string;
  },
) {
  const authorName = postSeoAuthorName(post);
  const authorUrl = postSeoAuthorUrl(post);
  const keywords = postSeoKeywords(post);
  const articleBody = richTextToPlainText(post.content).replace(/\s+/g, " ").trim();
  const interactionStatistic = [
    interactionCounter("https://schema.org/LikeAction", post.like_count),
    interactionCounter("https://schema.org/CommentAction", post.comment_count),
  ].filter(Boolean);

  return {
    "@context": "https://schema.org",
    "@type": "Article",
    mainEntityOfPage: {
      "@type": "WebPage",
      "@id": url,
    },
    headline: title,
    description,
    url,
    image: image ? [image] : undefined,
    thumbnailUrl: image ?? undefined,
    datePublished: post.created_at,
    author: {
      "@type": "Person",
      name: authorName,
      url: authorUrl,
    },
    publisher: {
      "@type": "Organization",
      name: SITE_NAME,
      url: getSiteUrl(),
    },
    articleSection: post.category?.trim() || undefined,
    keywords: keywords.length > 0 ? keywords.join(", ") : undefined,
    articleBody: articleBody ? truncateText(articleBody, 5_000) : undefined,
    interactionStatistic:
      interactionStatistic.length > 0 ? interactionStatistic : undefined,
  };
}

export function jsonLdScriptContent(value: unknown) {
  return JSON.stringify(value).replace(/</g, "\\u003c");
}

function interactionCounter(actionUrl: string, count: number | null | undefined) {
  if (typeof count !== "number" || count < 0) {
    return null;
  }

  return {
    "@type": "InteractionCounter",
    interactionType: actionUrl,
    userInteractionCount: count,
  };
}

function uniqueTextValues(values: string[]) {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = value.trim();
    const key = normalized.toLocaleLowerCase();
    if (!normalized || seen.has(key)) {
      continue;
    }
    seen.add(key);
    result.push(normalized);
  }

  return result;
}

function truncateText(value: string, maxLength: number) {
  const chars = Array.from(value);
  if (chars.length <= maxLength) {
    return value;
  }
  return `${chars.slice(0, maxLength - 1).join("")}...`;
}

function nonEmptyText(value: string | null | undefined, fallback: string) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : fallback;
}
