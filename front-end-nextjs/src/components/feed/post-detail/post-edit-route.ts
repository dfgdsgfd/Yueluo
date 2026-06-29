export const POST_EDIT_MOBILE_BREAKPOINT = 768;

export function postEditRouteForViewport(
  viewportWidth: number | null | undefined,
  postId: string | number,
) {
  const encodedPostId = encodeURIComponent(String(postId));
  const route = typeof viewportWidth === "number" && viewportWidth < POST_EDIT_MOBILE_BREAKPOINT
    ? "/publish/mobile"
    : "/publish";

  return `${route}?edit=${encodedPostId}`;
}

export function postEditRoute(postId: string | number) {
  const viewportWidth = typeof window === "undefined" ? undefined : window.innerWidth;
  return postEditRouteForViewport(viewportWidth, postId);
}
