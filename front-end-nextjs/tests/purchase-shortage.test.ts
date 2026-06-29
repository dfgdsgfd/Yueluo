import { describe, expect, it } from "vitest";
import { ApiError } from "../src/lib/api";
import { purchaseShortageFromError } from "../src/components/feed/post-detail/purchase-shortage";

describe("purchaseShortageFromError", () => {
  it("detects points shortage from API error keys and legacy Chinese messages", () => {
    expect(purchaseShortageFromError(new ApiError("error.insufficient_points"))).toBe("points");
    expect(purchaseShortageFromError(new ApiError("积分不足，需要 9.00"))).toBe("points");
    expect(purchaseShortageFromError(new ApiError("purchase failed", {
      details: { errorCode: "insufficient_points" },
    }))).toBe("points");
  });

  it("detects moon coin shortage from API error keys and legacy Chinese messages", () => {
    expect(purchaseShortageFromError(new ApiError("error.insufficient_balance"))).toBe("balance");
    expect(purchaseShortageFromError(new ApiError("月币不足，需要 9.00"))).toBe("balance");
    expect(purchaseShortageFromError(new ApiError("purchase failed", {
      details: { data: { reason: "insufficient_balance" } },
    }))).toBe("balance");
  });

  it("ignores unrelated purchase errors", () => {
    expect(purchaseShortageFromError(new ApiError("error.purchase_in_progress"))).toBeNull();
    expect(purchaseShortageFromError(new Error("network failed"))).toBeNull();
  });
});
