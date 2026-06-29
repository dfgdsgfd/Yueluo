"use client";

import dynamic from "next/dynamic";

type LazyPostMarkdownContentProps = {
  className?: string;
  content: string;
  onNavigateAway?: () => void;
};

const PostMarkdownContentInner = dynamic(
  () => import("./post-markdown-content").then((module) => module.PostMarkdownContent),
  { ssr: false },
);

export function LazyPostMarkdownContent(props: LazyPostMarkdownContentProps) {
  if (!props.content.trim()) {
    return null;
  }

  return <PostMarkdownContentInner {...props} />;
}
