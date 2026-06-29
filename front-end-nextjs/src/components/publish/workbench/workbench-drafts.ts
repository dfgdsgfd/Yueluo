import { getDraftPosts } from "@/lib/api";

export async function loadWorkbenchDrafts() {
  const payload = await getDraftPosts({ limit: 20 });
  return payload.posts;
}
