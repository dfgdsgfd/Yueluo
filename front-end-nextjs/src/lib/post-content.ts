const richTextHTMLPattern = /^\s*<\/?(?:p|h[1-6]|ul|ol|li|blockquote|pre|code|strong|em|a|br|hr|table|thead|tbody|tr|td|th|span|mark|u|s|img|div|dl|dt|dd|section|sup)\b/i;

export function countUnicodeCharacters(value: string) {
  return Array.from(value).length;
}

export function postContentLength(value: string) {
  const content = value.trim();
  if (!content) return 0;
  if (!richTextHTMLPattern.test(content)) return countUnicodeCharacters(content);

  const visibleText = decodeHtmlEntities(content.replace(/<[^>]*>/g, ""));
  return countUnicodeCharacters(visibleText);
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
    const radix = lower.startsWith("#x") ? 16 : 10;
    const raw = lower.startsWith("#x") ? lower.slice(2) : lower.slice(1);
    const codePoint = Number.parseInt(raw, radix);
    return Number.isFinite(codePoint) ? String.fromCodePoint(codePoint) : match;
  });
}
