import { headers } from "next/headers";
import type { Metadata } from "next";
import { PostDetailPageClient } from "@/components/feed/post-detail-page-client";
import { apiRequestContextFromHeaders, getPostDetail } from "@/lib/api";
import { isAccessBlockApiError } from "@/lib/api/core/access-block";
import {
  absoluteSiteUrl,
  jsonLdScriptContent,
  postSeoAuthorName,
  postSeoAuthorUrl,
  postSeoDescription,
  postSeoImage,
  postSeoJsonLd,
  postSeoKeywords,
  postSeoTags,
  postSeoTitle,
} from "@/lib/seo";
import { loadPostIdFromSearchParams } from "./post-detail-loader";

type PostPageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export async function generateMetadata({ searchParams }: PostPageProps): Promise<Metadata> {
  const postId = await loadPostIdFromSearchParams(searchParams);
  const headerStore = await headers();
  const post = await getPostDetail(
    postId,
    apiRequestContextFromHeaders(headerStore),
  ).catch(() => null);

  if (!post) {
    return {
      title: "Post",
      robots: { index: false, follow: true },
    };
  }

  const title = postSeoTitle(post);
  const description = postSeoDescription(post);
  const url = absoluteSiteUrl(`/post?id=${encodeURIComponent(String(postId))}`);
  const image = postSeoImage(post);
  const authorName = postSeoAuthorName(post);
  const authorUrl = postSeoAuthorUrl(post);
  const keywords = postSeoKeywords(post);
  const tags = postSeoTags(post);

  return {
    title,
    description,
    authors: [{ name: authorName, url: authorUrl }],
    keywords: keywords.length > 0 ? keywords : undefined,
    alternates: { canonical: url },
    robots: {
      index: true,
      follow: true,
      googleBot: {
        index: true,
        follow: true,
        "max-image-preview": "large",
        "max-video-preview": -1,
        "max-snippet": -1,
      },
    },
    openGraph: {
      type: "article",
      siteName: "Yuem",
      title,
      description,
      url,
      publishedTime: post.created_at,
      authors: [authorUrl],
      tags: tags.length > 0 ? tags : undefined,
      images: image ? [{ url: image, alt: title }] : undefined,
    },
    twitter: {
      card: image ? "summary_large_image" : "summary",
      title,
      description,
      images: image ? [image] : undefined,
    },
  };
}

export default async function PostPage({ searchParams }: PostPageProps) {
  const postId = await loadPostIdFromSearchParams(searchParams);
  const headerStore = await headers();
  const initialResult = await getPostDetail(
    postId,
    apiRequestContextFromHeaders(headerStore),
  ).then(
    (post) => ({ initialAccessBlocked: false, initialPost: post }),
    (error) => ({
      initialAccessBlocked: isAccessBlockApiError(error),
      initialPost: null,
    }),
  );
  const initialPost = initialResult.initialPost;

  const url = absoluteSiteUrl(`/post?id=${encodeURIComponent(String(postId))}`);
  const title = postSeoTitle(initialPost);
  const description = postSeoDescription(initialPost);
  const image = postSeoImage(initialPost);
  const jsonLd = initialPost
    ? postSeoJsonLd(initialPost, { description, image, title, url })
    : null;

  return (
    <>
      {jsonLd ? (
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: jsonLdScriptContent(jsonLd) }}
        />
      ) : null}
      <PostDetailPageClient
        postId={postId}
        initialAccessBlocked={initialResult.initialAccessBlocked}
        initialPost={initialPost}
        initialPostComplete={Boolean(initialPost)}
      />
    </>
  );
}
