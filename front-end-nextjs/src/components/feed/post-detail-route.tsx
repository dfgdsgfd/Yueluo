import { loadPostIdFromSearchParams } from "@/app/post/post-detail-loader";
import { PostDetailPageClient } from "@/components/feed/post-detail-page-client";

type PostDetailSearchParams = Promise<Record<string, string | string[] | undefined>> | undefined;

type PostDetailRouteProps = {
  presentation?: "page" | "modal";
  searchParams?: PostDetailSearchParams;
};

export async function PostDetailRoute({
  presentation = "page",
  searchParams,
}: PostDetailRouteProps) {
  const postId = await loadPostIdFromSearchParams(searchParams);

  return (
    <PostDetailPageClient
      postId={postId}
      mode={presentation}
    />
  );
}
