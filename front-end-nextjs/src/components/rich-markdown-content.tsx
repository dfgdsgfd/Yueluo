"use client";

import { MermaidContent } from "@/components/mermaid-content";
import { renderRichText } from "@/lib/markdown";
import { cn } from "@/lib/utils";

type RichMarkdownContentProps = {
  className?: string;
  content: string;
};

export function RichMarkdownContent({
  className,
  content,
}: RichMarkdownContentProps) {
  const html = renderRichText(content);
  if (!html) {
    return null;
  }

  return (
    <MermaidContent
      as="div"
      className={cn("markdown-content", className)}
      html={html}
    />
  );
}
