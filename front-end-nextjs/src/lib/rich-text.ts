import type { Editor } from "@tiptap/core";

export type RichTextContentType = "html" | "markdown";

const richTextHtmlPattern = /^\s*<\/?(?:p|h[1-6]|ul|ol|li|blockquote|pre|code|strong|em|a|br|hr|table|thead|tbody|tr|td|th|span|mark|u|s|img|div|dl|dt|dd|details|summary)\b/i;
const unsafeHtmlPattern = /<\s*script\b|<\s*iframe\b|\son\w+\s*=|javascript:|data:text/i;
const markdownPattern = /(^|\n)\s{0,3}(?:#{1,6}\s|[-*+]\s+|\d+\.\s+|>\s+|```|---\s*$)|(?:\*\*|__)[\s\S]+?(?:\*\*|__)|`[^`]+`|\[[^\]]+\]\([^)]+\)/;

export function isRichTextHtml(value: string | null | undefined) {
  return Boolean(value && richTextHtmlPattern.test(value));
}

export function isRenderableRichTextHtml(value: string | null | undefined) {
  return isRichTextHtml(value) && !unsafeHtmlPattern.test(value ?? "");
}

export function inferRichTextContentType(value: string | null | undefined): RichTextContentType {
  return isRichTextHtml(value) ? "html" : "markdown";
}

export function shouldInsertAsMarkdown(value: string | null | undefined) {
  const text = value?.trim();
  if (!text || isRichTextHtml(text)) {
    return false;
  }
  return markdownPattern.test(text) || text.includes("\n\n");
}

export function insertMarkdownFromPaste(editor: Editor, event: ClipboardEvent) {
  const clipboard = event.clipboardData;
  if (!clipboard || clipboard.getData("text/html").trim()) {
    return false;
  }

  const markdown = clipboard.getData("text/plain");
  if (!shouldInsertAsMarkdown(markdown)) {
    return false;
  }

  event.preventDefault();
  editor.commands.insertContent(markdown, { contentType: "markdown" });
  return true;
}

export function normalizePlainTextContent(value: string | null | undefined) {
  return (value ?? "").replace(/<br\s*\/?>/gi, "\n").trim();
}

export function richTextToPlainText(value: string | null | undefined) {
  const text = value ?? "";
  if (!isRichTextHtml(text)) {
    return markdownToPlainText(text);
  }

  return decodeHtmlEntities(
    text
      .replace(/<br\s*\/?>/gi, "\n")
      .replace(/<li\b[^>]*>/gi, "- ")
      .replace(/<\/(?:p|h[1-6]|blockquote|pre|div|li|tr)>/gi, "\n")
      .replace(/<[^>]*>/g, ""),
  )
    .replace(/\u00a0/g, " ")
    .replace(/[ \t]+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

export function markdownToPlainText(value: string | null | undefined) {
  return normalizePlainTextContent(value)
    .replace(/```[^\n]*\n([\s\S]*?)```/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/^\s{0,3}(?:#{1,6}|>|[-+*]|\d+\.)\s+/gm, "")
    .replace(/^\s*\[[ xX]\]\s+/gm, "")
    .replace(/^\[\^[^\]]+\]:\s*/gm, "")
    .replace(/\[\^[^\]]+\]/g, "")
    .replace(/<[^>]+>/g, "")
    .replace(/(?:\*\*|__|~~|`)([\s\S]*?)(?:\*\*|__|~~|`)/g, "$1")
    .replace(/(^|[^\\])[*_](?=\S)|(?<=\S)[*_](?![*_])/g, "$1")
    .replace(/[ \t]+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function decodeHtmlEntities(value: string) {
  return value.replace(/&(#x?[0-9a-f]+|amp|lt|gt|quot|apos|nbsp|#039);/gi, (match, entity: string) => {
    const lower = entity.toLowerCase();
    if (lower === "amp") return "&";
    if (lower === "lt") return "<";
    if (lower === "gt") return ">";
    if (lower === "quot") return "\"";
    if (lower === "apos" || lower === "#039") return "'";
    if (lower === "nbsp") return " ";
    if (lower.startsWith("#x")) {
      return codePointToString(Number.parseInt(lower.slice(2), 16), match);
    }
    if (lower.startsWith("#")) {
      return codePointToString(Number.parseInt(lower.slice(1), 10), match);
    }
    return match;
  });
}

function codePointToString(value: number, fallback: string) {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  try {
    return String.fromCodePoint(value);
  } catch {
    return fallback;
  }
}
