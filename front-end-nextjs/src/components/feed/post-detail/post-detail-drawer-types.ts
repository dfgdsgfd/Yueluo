import type { FeedPost } from "@/lib/types/content";

export type PostDetailDrawerProps = {
  post: FeedPost | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode?: "drawer" | "page" | "modal";
  targetCommentId?: string | number | null;
  targetCommentParentId?: string | number | null;
};
