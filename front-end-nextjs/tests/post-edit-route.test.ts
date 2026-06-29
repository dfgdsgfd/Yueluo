import { describe, expect, it } from "vitest";
import { postEditRouteForViewport } from "../src/components/feed/post-detail/post-edit-route";

describe("postEditRouteForViewport", () => {
  it("routes desktop editing to the desktop publish workbench", () => {
    expect(postEditRouteForViewport(1024, 88)).toBe("/publish?edit=88");
    expect(postEditRouteForViewport(768, "abc 123")).toBe("/publish?edit=abc%20123");
  });

  it("keeps mobile editing on the mobile publish page", () => {
    expect(postEditRouteForViewport(767, 88)).toBe("/publish/mobile?edit=88");
    expect(postEditRouteForViewport(390, "abc 123")).toBe("/publish/mobile?edit=abc%20123");
  });
});
