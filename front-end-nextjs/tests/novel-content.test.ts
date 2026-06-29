import { describe, expect, it } from "vitest";
import { renderRichTextDocument } from "../src/lib/markdown";
import { getNovelDetailContent, getNovelReadingMinutes } from "../src/components/feed/post-detail/novel-content";

describe("novel detail content", () => {
  it("keeps Markdown novel bodies renderable as Markdown", () => {
    const content = getNovelDetailContent("# 第一章\n\n- 张无忌\n- 赵敏");

    expect(content).toEqual({
      kind: "markdown",
      content: "# 第一章\n\n- 张无忌\n- 赵敏",
    });

    const html = renderRichTextDocument(content.kind === "markdown" ? content.content : "").html;
    expect(html).toContain('<h1 id="第一章">第一章</h1>');
    expect(html).toContain("<ul>");
  });

  it("lets plain novel text render through the Markdown renderer", () => {
    const content = getNovelDetailContent("第一行\n第二行");

    expect(content.kind).toBe("markdown");
    expect(renderRichTextDocument(content.kind === "markdown" ? content.content : "").html)
      .toMatch(/第一行<br\s*\/?>/);
  });

  it("turns legacy rich-text HTML into readable plain text", () => {
    expect(getNovelDetailContent("<h2>标题</h2><p>正文 <strong>加粗</strong></p>"))
      .toEqual({ kind: "plain", content: "标题\n正文 加粗" });
  });

  it("calculates reading minutes from visible text", () => {
    expect(getNovelReadingMinutes("# 标题\n\n" + "字".repeat(300))).toBe(2);
    expect(getNovelReadingMinutes("<p>" + "字".repeat(300) + "</p>")).toBe(1);
  });
});
