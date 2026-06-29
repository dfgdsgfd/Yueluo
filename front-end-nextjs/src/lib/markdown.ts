import katex from "katex";
import MarkdownIt from "markdown-it";
import footnote from "markdown-it-footnote";
import taskLists from "markdown-it-task-lists";
import sanitizeHtml from "sanitize-html";
import type StateBlock from "markdown-it/lib/rules_block/state_block.mjs";
import type StateInline from "markdown-it/lib/rules_inline/state_inline.mjs";

const markdown = new MarkdownIt({
  breaks: true,
  html: false,
  linkify: true,
  typographer: true,
})
  .use(mathPlugin)
  .use(footnote)
  .use(taskLists, { enabled: false, label: true, labelAfter: true });

export type MarkdownHeading = {
  id: string;
  level: number;
  text: string;
};

type MarkdownRenderEnvironment = {
  headingCounts: Map<string, number>;
  headings: MarkdownHeading[];
};

const defaultFenceRenderer = markdown.renderer.rules.fence?.bind(markdown.renderer);

markdown.renderer.rules.fence = (tokens, index, options, environment, renderer) => {
  const token = tokens[index];
  const language = token.info.trim().split(/\s+/)[0]?.toLocaleLowerCase();
  if (language !== "mermaid") {
    return defaultFenceRenderer
      ? defaultFenceRenderer(tokens, index, options, environment, renderer)
      : renderer.renderToken(tokens, index, options);
  }

  const source = token.content.trim();
  return `<div class="mermaid-diagram" data-mermaid-source="${escapeAttribute(source)}" data-mermaid-status="idle">${escapeHtml(source)}</div>`;
};

markdown.renderer.rules.heading_open = (tokens, index, options, environment, renderer) => {
  const env = environment as MarkdownRenderEnvironment;
  const headingToken = tokens[index];
  const inlineToken = tokens[index + 1];
  const text = inlineToken?.children
    ?.map((child) => child.type === "image" ? child.content : child.content ?? "")
    .join("")
    .trim() || inlineToken?.content?.trim() || "";
  const id = uniqueHeadingId(text, env.headingCounts, env.headings.length);
  headingToken.attrSet("id", id);
  env.headings.push({ id, level: Number(headingToken.tag.slice(1)) || 1, text: text || id });
  return renderer.renderToken(tokens, index, options);
};

const allowedTags = Array.from(new Set([
  ...sanitizeHtml.defaults.allowedTags,
  "annotation",
  "dd",
  "del",
  "details",
  "dl",
  "dt",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "img",
  "input",
  "label",
  "math",
  "mark",
  "menclose",
  "mfrac",
  "mi",
  "mmultiscripts",
  "mn",
  "mo",
  "mover",
  "mpadded",
  "mprescripts",
  "mphantom",
  "mroot",
  "mrow",
  "mspace",
  "msqrt",
  "mstyle",
  "msub",
  "msubsup",
  "msup",
  "mtable",
  "mtd",
  "mtext",
  "mtr",
  "munder",
  "munderover",
  "none",
  "s",
  "section",
  "semantics",
  "span",
  "sup",
  "summary",
  "table",
  "tbody",
  "td",
  "tfoot",
  "th",
  "thead",
  "tr",
  "u",
]));

const sanitizeOptions: sanitizeHtml.IOptions = {
  allowedTags,
  allowedAttributes: {
    "*": ["class", "data-mermaid-source", "data-mermaid-status", "id", "style"],
    a: ["aria-label", "href", "name", "rel", "target", "title"],
    annotation: ["encoding"],
    code: ["class"],
    details: ["open"],
    img: ["alt", "height", "loading", "src", "title", "width"],
    input: ["checked", "disabled", "type"],
    label: ["for"],
    li: ["class", "id"],
    math: ["display", "xmlns"],
    mo: ["fence", "form", "lspace", "rspace", "separator", "stretchy"],
    mpadded: ["depth", "height", "lspace", "voffset", "width"],
    mspace: ["depth", "height", "width"],
    mstyle: ["displaystyle", "mathbackground", "mathcolor", "mathsize", "mathvariant", "scriptlevel"],
    mtable: ["columnalign", "columnspacing", "rowalign", "rowspacing"],
    mtd: ["columnalign", "columnspan", "rowalign", "rowspan"],
    ol: ["class", "start"],
    td: ["align", "colspan", "rowspan"],
    th: ["align", "colspan", "rowspan", "scope"],
  },
  allowedSchemes: ["http", "https", "mailto"],
  allowProtocolRelative: false,
  allowedStyles: {
    "*": {
      "background-color": [/^#[0-9a-f]{3,8}$/i, /^rgba?\([0-9.,% ]+\)$/i],
      color: [/^#[0-9a-f]{3,8}$/i, /^rgba?\([0-9.,% ]+\)$/i],
      "text-align": [/^(?:left|center|right|justify)$/],
    },
  },
  transformTags: {
    a: (tagName, attribs) => {
      const external = /^(?:https?:)?\/\//i.test(attribs.href ?? "");
      return {
        tagName,
        attribs: external
          ? { ...attribs, rel: "noopener noreferrer", target: "_blank" }
          : attribs,
      };
    },
    img: (tagName, attribs) => ({
      tagName,
      attribs: { ...attribs, loading: "lazy" },
    }),
    input: (tagName, attribs) => ({
      tagName,
      attribs: { ...attribs, disabled: "disabled", type: "checkbox" },
    }),
  },
};

function mathPlugin(md: InstanceType<typeof MarkdownIt>) {
  md.block.ruler.before("paragraph", "math_block", mathBlockRule, {
    alt: ["paragraph", "reference", "blockquote", "list"],
  });
  md.inline.ruler.after("escape", "math_inline", mathInlineRule);
  md.renderer.rules.math_inline = (tokens, index) =>
    `<span class="math-inline">${renderMath(tokens[index].content, false)}</span>`;
  md.renderer.rules.math_block = (tokens, index) =>
    `<div class="math-display">${renderMath(tokens[index].content, true)}</div>\n`;
}

function mathBlockRule(state: StateBlock, startLine: number, endLine: number, silent: boolean) {
  const lineStart = state.bMarks[startLine] + state.tShift[startLine];
  const lineEnd = state.eMarks[startLine];
  if (state.sCount[startLine] - state.blkIndent >= 4) {
    return false;
  }
  if (state.src.charCodeAt(lineStart) !== 0x24 || state.src.charCodeAt(lineStart + 1) !== 0x24) {
    return false;
  }

  const openingLine = state.src.slice(lineStart + 2, lineEnd);
  const sameLineClose = findUnescapedBlockMathDelimiter(openingLine, 0);
  if (sameLineClose >= 0 && !openingLine.slice(sameLineClose + 2).trim()) {
    if (!silent) {
      const token = state.push("math_block", "math", 0);
      token.block = true;
      token.content = openingLine.slice(0, sameLineClose).trim();
      token.markup = "$$";
      token.map = [startLine, startLine + 1];
    }
    state.line = startLine + 1;
    return true;
  }

  const contentLines = [openingLine.trimStart()];
  let nextLine = startLine + 1;
  let foundClose = false;
  for (; nextLine < endLine; nextLine += 1) {
    const currentStart = state.bMarks[nextLine] + state.tShift[nextLine];
    const currentEnd = state.eMarks[nextLine];
    const currentLine = state.src.slice(currentStart, currentEnd);
    const closeIndex = findUnescapedBlockMathDelimiter(currentLine, 0);
    if (closeIndex >= 0 && !currentLine.slice(closeIndex + 2).trim()) {
      contentLines.push(currentLine.slice(0, closeIndex));
      foundClose = true;
      nextLine += 1;
      break;
    }
    contentLines.push(currentLine);
  }
  if (!foundClose) {
    return false;
  }
  if (!silent) {
    const token = state.push("math_block", "math", 0);
    token.block = true;
    token.content = contentLines.join("\n").trim();
    token.markup = "$$";
    token.map = [startLine, nextLine];
  }
  state.line = nextLine;
  return true;
}

function findUnescapedBlockMathDelimiter(value: string, start: number) {
  for (let index = start; index < value.length - 1; index += 1) {
    if (
      value.charCodeAt(index) === 0x24
      && value.charCodeAt(index + 1) === 0x24
      && !isEscaped(value, index)
    ) {
      return index;
    }
  }
  return -1;
}

function mathInlineRule(state: StateInline, silent: boolean) {
  if (state.src.charCodeAt(state.pos) !== 0x24 || state.src.charCodeAt(state.pos + 1) === 0x24) {
    return false;
  }
  const contentStart = state.pos + 1;
  if (contentStart >= state.posMax || /\s/.test(state.src[contentStart])) {
    return false;
  }
  const closeIndex = findUnescapedMathDelimiter(state.src, contentStart);
  if (closeIndex < 0 || closeIndex >= state.posMax || closeIndex === contentStart) {
    return false;
  }
  const content = state.src.slice(contentStart, closeIndex);
  if (!content.trim() || /\s/.test(content.at(-1) ?? "")) {
    return false;
  }

  if (!silent) {
    const token = state.push("math_inline", "math", 0);
    token.content = content;
    token.markup = "$";
  }
  state.pos = closeIndex + 1;
  return true;
}

function findUnescapedMathDelimiter(value: string, start: number) {
  for (let index = start; index < value.length; index += 1) {
    if (value.charCodeAt(index) === 0x0A) {
      return -1;
    }
    if (value.charCodeAt(index) !== 0x24) {
      continue;
    }
    if (!isEscaped(value, index)) {
      return index;
    }
  }
  return -1;
}

function isEscaped(value: string, index: number) {
  let backslashes = 0;
  for (let cursor = index - 1; cursor >= 0 && value.charCodeAt(cursor) === 0x5C; cursor -= 1) {
    backslashes += 1;
  }
  return backslashes % 2 === 1;
}

function renderMath(source: string, displayMode: boolean) {
  return katex.renderToString(source, {
    displayMode,
    output: "mathml",
    strict: "ignore",
    throwOnError: false,
    trust: false,
  });
}

export function renderRichText(value: string | null | undefined) {
	return renderRichTextDocument(value).html;
}

export function renderRichTextDocument(value: string | null | undefined): {
  headings: MarkdownHeading[];
  html: string;
} {
  const content = value?.trim();
  if (!content) {
    return { headings: [], html: "" };
  }

  const environment: MarkdownRenderEnvironment = {
    headingCounts: new Map(),
    headings: [],
  };
  const rendered = markdown.render(content, environment);
  const sanitized = sanitizeHtml(rendered, sanitizeOptions);
  return { headings: environment.headings, html: sanitized };
}

function uniqueHeadingId(text: string, counts: Map<string, number>, index: number) {
  const base = slugifyHeading(text) || `section-${index + 1}`;
  const count = (counts.get(base) ?? 0) + 1;
  counts.set(base, count);
  return count === 1 ? base : `${base}-${count}`;
}

export function slugifyHeading(value: string) {
  return value
    .normalize("NFKC")
    .trim()
    .toLocaleLowerCase()
    .replace(/[\s_]+/g, "-")
    .replace(/[^\p{Letter}\p{Number}\p{Mark}-]+/gu, "")
    .replace(/-{2,}/g, "-")
    .replace(/^-|-$/g, "");
}

function escapeAttribute(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
