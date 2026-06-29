export type PublishGenerationField = "title" | "detail";

type PublishGenerationParseResult = {
  title: string;
  detail: string;
};

type PublishGenerationJSONParseResult = PublishGenerationParseResult & {
  parsed: boolean;
};

export function parsePublishGenerationTitleDetail(raw: string): PublishGenerationParseResult {
  const text = normalizePublishGenerationOutput(raw);
  if (!text) return { title: "", detail: "" };
  const json = parsePublishGenerationJSONTitleDetail(text);
  if (json.parsed) return { title: json.title, detail: json.detail };
  return parsePublishGenerationTextTitleDetail(text);
}

export function parsePublishGenerationPreview(raw: string, field: PublishGenerationField) {
  const text = normalizePublishGenerationOutput(raw);
  if (!text || isPublishGenerationPromptEcho(text)) return "";
  const jsonValue = parsePublishGenerationJSONTitleDetail(text);
  if (jsonValue[field]) return jsonValue[field];
  if (jsonValue.parsed) return "";
  const partial = extractPublishGenerationPartialJSONField(text, field);
  if (partial) return partial;
  const labeled = parsePublishGenerationLabeledField(text, field);
  if (labeled) return labeled;
  if (hasPublishGenerationLabeledOutput(text)) return "";
  const plain = parsePublishGenerationPlainTitleDetail(text);
  return plain[field];
}

function parsePublishGenerationTextTitleDetail(text: string): PublishGenerationParseResult {
  const normalized = normalizePublishGenerationOutput(text);
  if (!normalized) return { title: "", detail: "" };
  const title = parsePublishGenerationLabeledField(normalized, "title");
  const detail = parsePublishGenerationLabeledField(normalized, "detail");
  if (title || detail) return { title, detail };
  return parsePublishGenerationPlainTitleDetail(normalized);
}

function normalizePublishGenerationOutput(raw: string) {
  const text = extractJSONText(raw).trim().replace(/^\uFEFF/, "").trim();
  if (!text || isPublishGenerationPromptEcho(text)) return "";
  return text;
}

function parsePublishGenerationJSONTitleDetail(text: string): PublishGenerationJSONParseResult {
  const empty = { title: "", detail: "", parsed: false };
  try {
    const parsed = JSON.parse(extractJSONText(text)) as unknown;
    const title = firstPublishGenerationJSONField(parsed, publishGenerationFieldKeys("title"));
    let detail = firstPublishGenerationJSONField(parsed, publishGenerationFieldKeys("detail"));
    if (!title && !detail) {
      const payload = firstPublishGenerationJSONTextPayload(parsed);
      if (payload) {
        const parsedPayload = parsePublishGenerationTextTitleDetail(payload);
        return { ...parsedPayload, parsed: true };
      }
    } else if (!detail) {
      const payload = firstPublishGenerationJSONTextPayload(parsed);
      if (payload) {
        const parsedPayload = parsePublishGenerationTextTitleDetail(payload);
        if (parsedPayload.detail) {
          detail = parsedPayload.detail;
        } else if (parsedPayload.title && parsedPayload.title !== title) {
          detail = parsedPayload.title;
        }
      }
    }
    return { title, detail, parsed: true };
  } catch {
    return empty;
  }
}

function firstPublishGenerationJSONField(value: unknown, keys: string[]): string {
  if (Array.isArray(value)) {
    for (const item of value) {
      const nested = firstPublishGenerationJSONField(item, keys);
      if (nested) return nested;
    }
    return "";
  }
  if (!value || typeof value !== "object") return "";
  const record = value as Record<string, unknown>;
  for (const key of keys) {
    const text = typeof record[key] === "string" ? normalizePublishGenerationPlainText(record[key]) : "";
    if (text) return text;
  }
  for (const wrapper of ["data", "result", "output", "message"]) {
    const nested = firstPublishGenerationJSONField(record[wrapper], keys);
    if (nested) return nested;
  }
  return "";
}

function firstPublishGenerationJSONTextPayload(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (Array.isArray(value)) {
    for (const item of value) {
      const nested = firstPublishGenerationJSONTextPayload(item);
      if (nested) return nested;
    }
    return "";
  }
  if (!value || typeof value !== "object") return "";
  const record = value as Record<string, unknown>;
  for (const wrapper of ["data", "result", "output", "message", "response", "payload", "content", "text"]) {
    const direct = record[wrapper];
    if (typeof direct === "string" && direct.trim()) return direct.trim();
    const nested = firstPublishGenerationJSONTextPayload(direct);
    if (nested) return nested;
  }
  return "";
}

function publishGenerationFieldKeys(field: PublishGenerationField) {
  return field === "title"
    ? ["title", "postTitle", "post_title", "headline", "name"]
    : ["detail", "details", "body", "description", "postDetail", "post_detail"];
}

function parsePublishGenerationLabeledField(text: string, field: PublishGenerationField) {
  const normalized = text.trim().replace(/\r\n/g, "\n");
  const labels = publishGenerationFieldLabels(field);
  let bestStart = -1;
  let bestEnd = -1;
  for (const label of labels) {
    for (const marker of [`${label}：`, `${label}:`]) {
      const index = normalized.indexOf(marker);
      if (index >= 0 && (bestStart < 0 || index < bestStart)) {
        bestStart = index;
        bestEnd = index + marker.length;
      }
    }
  }
  if (bestStart < 0) return "";
  let value = normalized.slice(bestEnd).trim();
  for (const label of publishGenerationOtherFieldLabels(field)) {
    for (const marker of [`\n${label}：`, `\n${label}:`]) {
      const index = value.indexOf(marker);
      if (index >= 0) value = value.slice(0, index);
    }
  }
  return normalizePublishGenerationPlainText(value.replace(/^["'`]+|["'`]+$/g, ""));
}

function publishGenerationFieldLabels(field: PublishGenerationField) {
  if (field === "title") return ["标题", "Title", "title"];
  return ["详情", "正文", "Detail", "Details", "Body", "detail", "details", "body"];
}

function publishGenerationOtherFieldLabels(field: PublishGenerationField) {
  return field === "title" ? publishGenerationFieldLabels("detail") : publishGenerationFieldLabels("title");
}

function hasPublishGenerationLabeledOutput(text: string) {
  const normalized = text.trim().replace(/\r\n/g, "\n");
  return [...publishGenerationFieldLabels("title"), ...publishGenerationFieldLabels("detail")].some(
    (label) => normalized.includes(`${label}：`) || normalized.includes(`${label}:`),
  );
}

function parsePublishGenerationPlainTitleDetail(text: string): PublishGenerationParseResult {
  const lines = text
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => line.trim().replace(/^["'`]+|["'`]+$/g, ""))
    .filter(Boolean);
  if (lines.length === 0) return { title: "", detail: "" };
  return {
    title: normalizePublishGenerationPlainText(lines[0]),
    detail: normalizePublishGenerationPlainText(lines.slice(1).join("\n")),
  };
}

function extractPublishGenerationPartialJSONField(text: string, field: PublishGenerationField) {
  for (const key of publishGenerationFieldKeys(field)) {
    const pattern = new RegExp(`"${escapeRegExp(key)}"\\s*:\\s*"((?:\\\\.|[^"\\\\])*)`, "i");
    const match = text.match(pattern);
    const value = match?.[1] ? decodeJSONStringFragment(match[1]) : "";
    const normalized = normalizePublishGenerationPlainText(value);
    if (normalized) return normalized;
  }
  return "";
}

function normalizePublishGenerationPlainText(value: string) {
  const text = value.trim();
  if (!text || isPublishGenerationPromptEcho(text)) return "";
  if (hasPublishGenerationLabeledOutput(text)) return "";
  if (text.startsWith("{") || text.startsWith("[") || text.startsWith("```")) return "";
  return text;
}

function decodeJSONStringFragment(value: string) {
  const fragment = value.replace(/\\u[0-9a-fA-F]{0,3}$/, "").replace(/\\$/, "");
  try {
    return JSON.parse(`"${fragment}"`) as string;
  } catch {
    return fragment
      .replace(/\\n/g, "\n")
      .replace(/\\r/g, "\r")
      .replace(/\\t/g, "\t")
      .replace(/\\"/g, '"')
      .replace(/\\\\/g, "\\");
  }
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function isPublishGenerationPromptEcho(value: string) {
  const normalized = value.trim().replace(/\r\n/g, "\n");
  return (
    normalized.includes("已有标题") &&
    normalized.includes("已有详情正文") &&
    normalized.includes("可用于分析的图片数量")
  );
}

function extractJSONText(text: string) {
  const trimmed = text.trim();
  const fenced = trimmed.match(/```(?:json)?\s*([\s\S]*?)```/i);
  if (fenced?.[1]) return fenced[1].trim();
  const start = trimmed.indexOf("{");
  const end = trimmed.lastIndexOf("}");
  if (start >= 0 && end > start) {
    return trimmed.slice(start, end + 1);
  }
  return trimmed;
}
