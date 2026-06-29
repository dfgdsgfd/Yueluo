import { redirect } from "next/navigation";

export async function loadPostIdFromSearchParams(
  searchParams?: Promise<Record<string, string | string[] | undefined>>,
) {
  const resolvedSearchParams = await searchParams;
  const rawId = resolvedSearchParams?.id;
  const postId = Array.isArray(rawId) ? rawId[0] : rawId;

  if (!postId) {
    redirect("/");
  }

  return postId;
}
