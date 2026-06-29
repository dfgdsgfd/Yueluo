import { describe, expect, it } from "vitest";
import { parsePublishGenerationPreview } from "../src/components/publish/shared/ai-publish-generation-parser";
import { publishGenerationRunInput } from "../src/components/publish/shared/publish-generation-run-input";
import { countPublishGenerationSelectableImages } from "../src/components/publish/shared/publish-generation-image-count";

describe("publish generation action", () => {
  it("counts local selected images before temporary upload", () => {
    expect(countPublishGenerationSelectableImages(5, 3)).toBe(3);
    expect(countPublishGenerationSelectableImages(2, undefined)).toBe(2);
    expect(countPublishGenerationSelectableImages(2, 0)).toBe(0);
  });

  it("extracts two-step JSON output for detail and title", () => {
    expect(parsePublishGenerationPreview('{"detail":"窗边的花和木桌都很清楚。"}', "detail")).toBe("窗边的花和木桌都很清楚。");
    expect(parsePublishGenerationPreview('{"title":"窗边这束花很亮"}', "title")).toBe("窗边这束花很亮");
  });

  it("supports streaming partial JSON without applying the raw object", () => {
    expect(parsePublishGenerationPreview('{"detail":"白色花瓶', "detail")).toBe("白色花瓶");
    expect(parsePublishGenerationPreview('{"title":"周末窗边', "title")).toBe("周末窗边");
    expect(parsePublishGenerationPreview('{"detail":""}', "detail")).toBe("");
  });

  it("clears previous title and detail when regenerating from images", () => {
    const input = {
      body: "上一次 AI 生成的详情",
      images: [{ url: "https://cdn.example.test/image.jpg" }],
      tags: ["日常"],
      title: "上一次标题",
    };
    expect(publishGenerationRunInput(input, { fresh: true })).toEqual({
      ...input,
      body: "",
      title: "",
    });
    expect(publishGenerationRunInput(input)).toEqual(input);
  });
});
