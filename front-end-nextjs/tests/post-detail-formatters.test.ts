import { describe, expect, it, vi } from "vitest";
import { formatCommentDate } from "../src/components/feed/post-detail/post-detail-formatters";

describe("post detail formatters", () => {
  it("formats comment dates with caller locale and browser timezone defaults", () => {
    const originalDateTimeFormat = Intl.DateTimeFormat;
    let capturedLocale: Intl.LocalesArgument;
    let capturedOptions: Intl.DateTimeFormatOptions | undefined;

    vi.spyOn(Intl, "DateTimeFormat").mockImplementation(function DateTimeFormat(
      locale,
      options,
    ) {
      capturedLocale = locale;
      capturedOptions = options;
      return new originalDateTimeFormat(locale, options);
    } as typeof Intl.DateTimeFormat);

    const value = formatCommentDate("2026-06-25T12:00:00Z", "recent", "zh-CN");

    expect(value).not.toBe("recent");
    expect(capturedLocale).toBe("zh-CN");
    expect(capturedOptions?.timeZone).toBeUndefined();
  });
});
