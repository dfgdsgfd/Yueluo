"use client";

import { useRouter } from "next/navigation";
import type { MouseEvent } from "react";
import { MermaidContent } from "@/components/mermaid-content";
import { renderRichTextDocument } from "@/lib/markdown";
import { cn } from "@/lib/utils";

export function PostMarkdownContent({
  className,
  content,
  onNavigateAway,
}: {
  className?: string;
  content: string;
  onNavigateAway?: () => void;
}) {
  const router = useRouter();
  const { html } = renderRichTextDocument(content);
  if (!html) {
    return null;
  }

  function handleClick(event: MouseEvent<HTMLElement>) {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const anchor = target.closest<HTMLAnchorElement>("a[href]");
    if (!anchor) {
      return;
    }
    const href = anchor.getAttribute("href")?.trim();
    if (!href) {
      return;
    }
    if (href.startsWith("#")) {
      const id = decodeFragment(href.slice(1));
      const destination = document.getElementById(id);
      if (!destination) {
        return;
      }
      event.preventDefault();
      destination.scrollIntoView({
        behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
        block: "start",
      });
      window.history.replaceState(window.history.state, "", `${window.location.pathname}${window.location.search}#${encodeURIComponent(id)}`);
      return;
    }

    let destination: URL;
    try {
      destination = new URL(href, window.location.href);
    } catch {
      return;
    }
    if (destination.origin !== window.location.origin || !["http:", "https:"].includes(destination.protocol)) {
      return;
    }
    event.preventDefault();
    if (destination.pathname !== window.location.pathname || destination.search !== window.location.search) {
      onNavigateAway?.();
    }
    router.push(`${destination.pathname}${destination.search}${destination.hash}`);
  }

  return (
    <MermaidContent
      as="article"
      data-post-markdown
      className={cn("markdown-content post-markdown-content", className)}
      onClick={handleClick}
      html={html}
    />
  );
}

function decodeFragment(value: string) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
