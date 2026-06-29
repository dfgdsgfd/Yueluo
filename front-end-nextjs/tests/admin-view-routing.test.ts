import { describe, expect, it } from "vitest";
import {
  adminViewSearchSuffix,
  adminViewToSlug,
  parseAdminViewSearchParams,
  parseAdminViewSlug,
} from "../src/components/admin/admin-page/admin-view-routing";

describe("admin view routing", () => {
  it("keeps dashboard as the default view", () => {
    expect(parseAdminViewSlug(null)).toEqual({ kind: "dashboard" });
    expect(parseAdminViewSlug("")).toEqual({ kind: "dashboard" });
    expect(parseAdminViewSlug("unknown")).toEqual({ kind: "dashboard" });
    expect(parseAdminViewSearchParams(new URLSearchParams())).toEqual({ kind: "dashboard" });
  });

  it("parses console views and admin resources from the URL", () => {
    expect(parseAdminViewSlug("logs")).toEqual({ kind: "logs" });
    expect(parseAdminViewSlug("users")).toEqual({ kind: "resource", resource: "users" });
    expect(parseAdminViewSlug("media-library")).toEqual({
      kind: "resource",
      resource: "media-library",
    });
    expect(parseAdminViewSearchParams(new URLSearchParams("view=component-check"))).toEqual({
      kind: "component-check",
    });
  });

  it("serializes dashboard without a view query and preserves unrelated params", () => {
    expect(adminViewToSlug({ kind: "resource", resource: "posts" })).toBe("posts");
    expect(adminViewSearchSuffix({ kind: "logs" }, new URLSearchParams())).toBe("?view=logs");
    expect(adminViewSearchSuffix({ kind: "resource", resource: "users" }, new URLSearchParams("modal=1&view=logs"))).toBe(
      "?modal=1&view=users",
    );
    expect(adminViewSearchSuffix({ kind: "dashboard" }, new URLSearchParams("modal=1&view=logs"))).toBe("?modal=1");
  });
});
