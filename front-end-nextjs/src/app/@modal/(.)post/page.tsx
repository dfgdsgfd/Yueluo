import { headers } from "next/headers";
import { PostDetailPageClient } from "@/components/feed/post-detail-page-client";
import { loadPostIdFromSearchParams } from "@/app/post/post-detail-loader";
import { apiRequestContextFromHeaders, getPostDetail } from "@/lib/api";
import { isAccessBlockApiError } from "@/lib/api/core/access-block";

type InterceptedPostPageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function InterceptedPostPage({
  searchParams,
}: InterceptedPostPageProps) {
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

  return (
    <PostDetailPageClient
      postId={postId}
      initialPost={initialResult.initialPost}
      initialAccessBlocked={initialResult.initialAccessBlocked}
      initialPostComplete={Boolean(initialResult.initialPost)}
      mode="modal"
    />
  );
}
