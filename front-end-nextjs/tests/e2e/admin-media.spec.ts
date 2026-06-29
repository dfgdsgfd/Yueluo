import { expect, test } from "@playwright/test";
import { uploadAccept } from "../../src/components/admin/admin-page/resource-editor";

test("admin media library does not restrict file suffixes", () => {
  expect(uploadAccept("media")).toBeUndefined();
});
