import { expect, test } from "@playwright/test";
import { getUnlockImageCounts } from "../../src/components/feed/post-detail/paid-image-access-utils";
import {
  applyImageAccessPatch,
  enforceImageCoverPolicy,
} from "../../src/components/publish/shared/image-access";

test("cover policy survives paid and protection batch updates", () => {
  const images = enforceImageCoverPolicy([
    { id: "cover", isFreePreview: false, isProtected: true },
    { id: "two", isFreePreview: true, isProtected: false },
    { id: "three", isFreePreview: true, isProtected: false },
  ]);
  const updated = applyImageAccessPatch(images, images.map((image) => image.id), {
    isFreePreview: false,
    isProtected: true,
  });

  expect(updated[0]).toMatchObject({ id: "cover", isFreePreview: true, isProtected: false });
  expect(updated.slice(1).every((image) => !image.isFreePreview && image.isProtected)).toBe(true);
});

test("unlock summary separates direct and protected paid images", () => {
  const counts = getUnlockImageCounts({
    id: 8,
    title: "Protected post",
    type: 1,
    like_count: 0,
    liked: false,
    collected: false,
    hiddenPaidImagesCount: 3,
    lockedProtectedImagesCount: 2,
  });
  expect(counts).toEqual({ directCount: 1, protectedCount: 2 });
});
