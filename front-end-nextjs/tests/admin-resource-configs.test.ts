import { describe, expect, it } from "vitest";
import { resourceConfigMap, resourceConfigs } from "../src/components/admin/admin-page/resource-configs";

const expectedResourceOrder = [
  "users",
  "posts",
  "content-review",
  "reports",
  "audit",
  "comments",
  "ai-moderation-logs",
  "categories",
  "tags",
  "media-library",
  "file-recycle-bin",
  "announcements",
  "system-notifications",
  "notification-templates",
  "feedback",
  "open-apis",
  "admins",
  "sessions",
  "banned-word-categories",
  "banned-words",
  "collections",
  "follows",
  "likes",
  "licenses",
  "app-versions",
  "post-configs",
  "user-configs",
  "posts-quality",
  "quality-reward-settings",
  "user-toolbar",
];

describe("admin resource configs", () => {
  it("preserves resource order after grouped config splits", () => {
    expect(resourceConfigs.map((config) => config.resource)).toEqual(expectedResourceOrder);
  });

  it("keeps resourceConfigMap aligned with the resource list", () => {
    expect([...resourceConfigMap.keys()]).toEqual(expectedResourceOrder);
    for (const config of resourceConfigs) {
      expect(resourceConfigMap.get(config.resource)).toBe(config);
    }
  });
});
