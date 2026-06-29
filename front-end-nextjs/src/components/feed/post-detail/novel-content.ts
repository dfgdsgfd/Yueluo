import { isRichTextHtml, richTextToPlainText } from "@/lib/rich-text";

const novelReadingCharactersPerMinute = 300;

export type NovelDetailContent =
  | { kind: "empty" }
  | { kind: "markdown"; content: string }
  | { kind: "plain"; content: string };

export function getNovelDetailContent(value: string | null | undefined): NovelDetailContent {
  const content = value?.trim();
  if (!content) {
    return { kind: "empty" };
  }

  if (isRichTextHtml(content)) {
    const text = richTextToPlainText(content);
    return text ? { kind: "plain", content: text } : { kind: "empty" };
  }

  return { kind: "markdown", content };
}

export function getNovelReadingMinutes(value: string | null | undefined) {
  const visibleText = richTextToPlainText(value);
  return Math.max(1, Math.ceil(Array.from(visibleText).length / novelReadingCharactersPerMinute));
}
