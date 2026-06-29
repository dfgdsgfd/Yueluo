import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { renderRichText, renderRichTextDocument } from "../src/lib/markdown";

const fullMarkdown = readFileSync(
  resolve(process.cwd(), "../testdata/Full-Markdown.md"),
  "utf8",
);
const postMarkdownComponent = readFileSync(
  resolve(process.cwd(), "src/components/feed/post-detail/post-markdown-content.tsx"),
  "utf8",
);
const globalStylesEntry = resolve(process.cwd(), "src/app/globals.css");
const globalStylesSource = readFileSync(globalStylesEntry, "utf8");
const globalStyles = [
  globalStylesSource,
  ...Array.from(globalStylesSource.matchAll(/@import\s+"([^"]+\.css)";/g),
    ([, importedPath]) =>
      readFileSync(resolve(dirname(globalStylesEntry), importedPath), "utf8"),
  ),
].join("\n");

describe("renderRichText", () => {
  it("renders the complete Markdown fixture", () => {
    const html = renderRichText(fullMarkdown);

    expect(html).toContain('<h1 id="markdown-演示文档完整版">✨ Markdown 演示文档（完整版）</h1>');
    expect(html).toContain('<h6 id="h6-标题">H6 标题</h6>');
    expect(html).toContain("<strong>加粗文本</strong>");
    expect(html).toContain("<em>斜体文本</em>");
    expect(html).toContain("<s>删除线</s>");
    expect(html).toContain("<ol>");
    expect(html).toContain("<ul>");
    expect(html).toContain("task-list-item");
    expect(html).toContain("contains-task-list");
    expect(html).toContain('type="checkbox"');
    expect(html).toMatch(/<input[^>]+\schecked(?:\s|=)/);
    expect(html).toContain('disabled="disabled"');
    expect(html).toContain('class="task-list-item-label"');
    expect(html).toMatch(/<label[^>]+for="task-item-\d+"/);
    expect(html).toContain("<table>");
    expect(html).toContain('class="language-javascript"');
    expect(html).toContain("<blockquote>");
    expect(html).toContain("<img");
    expect(html).toContain('loading="lazy"');
    expect(html).toContain('rel="noopener noreferrer"');
    expect(html).toContain('target="_blank"');
    expect(html).toContain('<span class="math-inline"><span class="katex"><math');
    expect(html).toContain("<annotation encoding=\"application/x-tex\">E = mc^2</annotation>");
    expect(html).toContain('<div class="math-display"><span class="katex"><math');
    expect(html).toContain('display="block"');
    expect(html).toContain("&lt;details&gt;");
    expect(html).toContain("&lt;summary&gt;点击展开更多内容&lt;/summary&gt;");
    expect(html).toContain("这是折叠区域中的代码");
    expect(html.match(/class="mermaid-diagram"/g)).toHaveLength(7);
    expect(html).toContain('data-mermaid-source="pie title Markdown 文档元素占比');
    expect(html).toContain('data-mermaid-source="flowchart TD');
    expect(html).toContain('data-mermaid-source="sequenceDiagram');
  });

  it("keeps nested task lists readable and non-interactive", () => {
    const html = renderRichText(`
- [x] Parent task
  - [ ] Nested task with **formatting**
  - [x] Nested completed task
- [ ] A very long task item that must remain available for responsive wrapping
`);

    expect(html.match(/class="task-list-item"/g)).toHaveLength(4);
    expect(html.match(/class="contains-task-list"/g)).toHaveLength(2);
    expect(html.match(/disabled="disabled"/g)).toHaveLength(4);
    expect(html.match(/\schecked(?:\s|=)/g)).toHaveLength(2);
    expect(html).toContain("<strong>formatting</strong>");
  });

  it("uses a semantic article without card styling for post content", () => {
    expect(postMarkdownComponent).toContain('as="article"');
    expect(postMarkdownComponent).toContain(
      '"markdown-content post-markdown-content"',
    );

    const rootRule = globalStyles.match(
      /\.post-rich-text-content,\s*\.post-markdown-content\s*\{([^}]+)\}/,
    )?.[1];
    expect(rootRule).toContain("width: 100%");
    expect(rootRule).toContain("max-width: 100%");
    expect(rootRule).toContain("font-size: clamp(");
    expect(rootRule).not.toMatch(/\b(?:background|border|padding)\s*:/);
    expect(globalStyles).toMatch(
      /\.post-markdown-content pre\s*\{[\s\S]*?overflow-x: auto;/,
    );
    expect(globalStyles).toMatch(
      /\.post-markdown-content table\s*\{[\s\S]*?overflow-x: auto;/,
    );
    expect(globalStyles).toMatch(
      /\.markdown-content \.task-list-item\s*\{[\s\S]*?position: relative;/,
    );
  });

  it("renders Markdown only and escapes legacy HTML", () => {
    const html = renderRichText(`
# Safe

<script>alert(1)</script>

[bad](javascript:alert(1))

![safe](https://example.com/image.png)
`);

    expect(html).toContain('<h1 id="safe">Safe</h1>');
    expect(html).not.toContain("<script");
    expect(html).not.toContain('href="javascript:');
    expect(html).not.toContain("onerror");
    expect(html).toContain("&lt;script&gt;alert(1)&lt;/script&gt;");
    expect(html).toContain('loading="lazy"');
  });

  it("renders Markdown math and treats foldable HTML as text", () => {
    const html = renderRichText(`
Inline $E = mc^2$ and escaped \\$not math$.

$$
S_n = \\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}
$$

<details onclick="alert(1)" open>
<summary>More</summary>

Hidden **Markdown** content.

</details>
`);

    expect(html).toContain('<span class="math-inline"><span class="katex"><math');
    expect(html).toContain("<annotation encoding=\"application/x-tex\">E = mc^2</annotation>");
    expect(html).toContain('<div class="math-display"><span class="katex"><math');
    expect(html).toContain("<munderover>");
    expect(html).toContain("<mfrac>");
    expect(html).toContain("escaped $not math$");
    expect(html).toContain("&lt;details");
    expect(html).toContain("alert(1)");
    expect(html).toContain("&lt;summary&gt;More&lt;/summary&gt;");
    expect(html).toContain("Hidden <strong>Markdown</strong> content.");
    expect(html).not.toContain("<details");
    expect(html).not.toContain('onclick="alert(1)"');
  });

  it("creates stable Unicode heading destinations and deduplicates them", () => {
    const document = renderRichTextDocument(`
# 第一章

[继续阅读](#第二章)

## 第二章
## 第二章
`);

    expect(document.html).toContain('<h1 id="第一章">第一章</h1>');
    expect(document.html).toContain('<h2 id="第二章">第二章</h2>');
    expect(document.html).toContain('<h2 id="第二章-2">第二章</h2>');
    expect(document.html).toContain('href="#%E7%AC%AC%E4%BA%8C%E7%AB%A0"');
    expect(document.headings.map((heading) => heading.id)).toEqual(["第一章", "第二章", "第二章-2"]);
  });

  it("marks Mermaid fences for client-side diagram rendering", () => {
    const html = renderRichText([
      "```mermaid",
      "flowchart TD",
      "  A[Start] --> B{Ready?}",
      "  B -->|Yes| C[Ship]",
      "```",
    ].join("\n"));

    expect(html).toContain('class="mermaid-diagram"');
    expect(html).toContain('data-mermaid-source="flowchart TD');
    expect(html).toContain("data-mermaid-status=\"idle\"");
    expect(html).toContain("flowchart TD");
    expect(html).not.toContain("language-mermaid");
  });
});
