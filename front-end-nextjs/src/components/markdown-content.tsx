"use client";

import dynamic from "next/dynamic";

export type MarkdownContentProps = {
  className?: string;
  content: string;
};

type PostMarkdownContentProps = MarkdownContentProps & {
  onNavigateAway?: () => void;
};

const RichMarkdownContent = dynamic(
  () => import("@/components/rich-markdown-content").then((module) => module.RichMarkdownContent),
  { ssr: false },
);

const LazyPostMarkdownContent = dynamic(
  () => import("@/components/feed/post-detail/post-markdown-content").then((module) => module.PostMarkdownContent),
  { ssr: false },
);

export function MarkdownContent({
  className,
  content,
}: MarkdownContentProps) {
  if (!content.trim()) {
    return null;
  }

  return <RichMarkdownContent className={className} content={content} />;
}

export function PostMarkdownContent(props: PostMarkdownContentProps) {
  if (!props.content.trim()) {
    return null;
  }

  return <LazyPostMarkdownContent {...props} />;
}
