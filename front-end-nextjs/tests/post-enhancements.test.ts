import { describe, expect, it } from "vitest";
import { getPostSlideshowMaxImages } from "../src/components/feed/post-detail/post-detail-config";
import { mobilePostEditState } from "../src/components/publish/mobile-publish/mobile-publish-edit";
import { buildMobilePublishPayload } from "../src/components/publish/mobile-publish/mobile-publish-submit";
import { workbenchPostEditState } from "../src/components/publish/workbench/post-edit-state";
import { postContentLength } from "../src/lib/post-content";
import type { FeedPost } from "../src/lib/types";

describe("post enhancement configuration", () => {
  it("bounds the configured slideshow image count", () => {
    expect(getPostSlideshowMaxImages()).toBe(25);
    expect(getPostSlideshowMaxImages("20")).toBe(20);
    expect(getPostSlideshowMaxImages("0")).toBe(25);
    expect(getPostSlideshowMaxImages("900")).toBe(500);
  });

  it("counts Markdown source and rich-text visible Unicode consistently", () => {
    expect(postContentLength("# 标题\n\n😀正文")).toBe(9);
    expect(postContentLength("<h2>标题</h2><p>😀正文 <strong>加粗</strong></p>")).toBe(8);
  });
});

describe("mobile post editing", () => {
  const post: FeedPost = {
    id: 8,
    title: "Novel",
    content: "Body",
    type: 1,
    category_id: 3,
    category: "fiction",
    visibility: "friends_only",
    like_count: 0,
    liked: false,
    collected: false,
    tags: [{ id: 1, name: "one" }, { id: 2, name: "two" }],
    images: [
      { url: "/one.webp", isFreePreview: true, isProtected: false, sortOrder: 1 },
      { url: "/two.webp", isFreePreview: false, isProtected: true, sortOrder: 2 },
    ],
    attachment: { url: "/book.zip", filename: "book.zip", filesize: 12 },
    paymentSettings: {
      enabled: true,
      paymentMethod: "points",
      paymentType: "single",
      freePreviewCount: 1,
      previewDuration: 0,
      price: 9,
      hideAll: false,
    },
  };

  it("restores remote media, payment, tags, category, attachment and visibility", () => {
    const restored = mobilePostEditState(post);
    expect(restored.visibility).toBe("followers");
    expect(restored.topicInput).toBe("one, two");
    expect(restored.categoryId).toBe(3);
    expect(restored.attachment?.filename).toBe("book.zip");
    expect(restored.paymentMethod).toBe("points");
    expect(restored.price).toBe("9");
    expect(restored.media).toHaveLength(2);
    expect(restored.media[1]).toMatchObject({ file: null, isFreePreview: false, isProtected: true });
  });

  it("maps follower visibility and serializes all restored remote assets", () => {
    const restored = mobilePostEditState(post);
    const payload = buildMobilePublishPayload({
      attachmentFile: null,
      attachmentValue: restored.attachment,
      body: restored.body,
      editingPostType: restored.type,
      imagePaymentMethod: restored.paymentMethod,
      imagePrice: restored.price,
      isDraft: false,
      isEditing: true,
      paidImageCount: 1,
      selectedCategoryId: restored.categoryId,
      title: restored.title,
      topic: restored.topicInput,
      uploadedAttachment: null,
      uploadedMedia: restored.media.map((item) => ({ ...item, asset: item.remoteAsset! })),
      visibility: restored.visibility,
    });
    expect(payload.visibility).toBe("friends_only");
    expect(payload.tags).toEqual(["one", "two"]);
    expect(payload.images).toHaveLength(2);
    expect(payload.paymentSettings).toMatchObject({ enabled: true, paymentMethod: "points", price: 9 });
    expect(payload.attachment).toMatchObject({ url: "/book.zip" });
  });

  it("emits explicit empty values when editable resources are removed", () => {
    const payload = buildMobilePublishPayload({
      attachmentFile: null,
      attachmentValue: null,
      body: "Body",
      editingPostType: 1,
      imagePaymentMethod: "balance",
      imagePrice: "1",
      isDraft: false,
      isEditing: true,
      paidImageCount: 0,
      selectedCategoryId: null,
      title: "Title",
      topic: "",
      uploadedAttachment: null,
      uploadedMedia: [],
      visibility: "public",
    });
    expect(payload.category_id).toBeNull();
    expect(payload.tags).toEqual([]);
    expect(payload.images).toEqual([]);
    expect(payload.attachment).toBeNull();
    expect(payload.paymentSettings).toEqual({ enabled: false });
  });
});

describe("desktop workbench post editing", () => {
  it("restores a published image post into desktop workbench state", () => {
    const post: FeedPost = {
      id: 18,
      title: "Desktop note",
      content: "Body",
      type: 1,
      category_id: 7,
      visibility: "friends_only",
      like_count: 0,
      liked: false,
      collected: false,
      tags: [{ id: 1, name: "desk" }, { id: 2, name: "edit" }],
      images: [
        { url: "/cover.webp", isFreePreview: true, isProtected: false, sortOrder: 1 },
        { url: "/paid.webp", isFreePreview: false, isProtected: true, sortOrder: 2 },
      ],
      paymentSettings: {
        enabled: true,
        paymentMethod: "points",
        paymentType: "single",
        freePreviewCount: 1,
        previewDuration: 0,
        price: 12,
        hideAll: false,
      },
    };

    const restored = workbenchPostEditState(post);

    expect(restored.mode).toBe("image");
    expect(restored.title).toBe("Desktop note");
    expect(restored.body).toBe("Body");
    expect(restored.tags).toBe("desk, edit");
    expect(restored.visibility).toBe("followers");
    expect(restored.categoryId).toBe(7);
    expect(restored.assets.image).toHaveLength(2);
    expect(restored.assets.image[1]).toMatchObject({ isFreePreview: false, isProtected: true });
    expect(restored.paymentSettings).toEqual({ enabled: true, paymentMethod: "points", price: "12" });
  });
});
