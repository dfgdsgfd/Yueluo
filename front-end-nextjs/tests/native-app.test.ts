import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@capacitor/app", () => ({ App: {} }));
const browserOpen = vi.hoisted(() => vi.fn());
vi.mock("@capacitor/browser", () => ({ Browser: { open: browserOpen } }));
vi.mock("@capacitor/core", () => ({
  Capacitor: {
    getPlatform: () => "android",
    isNativePlatform: () => true,
  },
}));
vi.mock("@capacitor/share", () => ({ Share: {} }));

function memoryStorage() {
  const values = new Map<string, string>();
  return {
    clear: () => values.clear(),
    getItem: (key: string) => values.get(key) ?? null,
    key: (index: number) => [...values.keys()][index] ?? null,
    get length() {
      return values.size;
    },
    removeItem: (key: string) => values.delete(key),
    setItem: (key: string, value: string) => values.set(key, value),
  } satisfies Storage;
}

describe("native app OAuth callback parsing", () => {
  beforeEach(() => {
    browserOpen.mockReset();
  });

  it("accepts only the native ticket callback", async () => {
    const { parseNativeOAuthCallback } = await import("../src/lib/native-app");

    expect(
      parseNativeOAuthCallback(
        "xsewebfast://auth-return?ticket=ticket-value&url=https%3A%2F%2Fxse.yuelk.com%2Fmessages",
      ),
    ).toEqual({
      error: "",
      returnUrl: "https://xse.yuelk.com/messages",
      ticket: "ticket-value",
    });

    expect(
      parseNativeOAuthCallback(
        "https://xse.yuelk.com/app/oauth/callback?code=legacy-code",
      ),
    ).toBeNull();
  });

  it("accepts the configured native scheme without requiring a fixed host", async () => {
    const { parseNativeOAuthCallback } = await import("../src/lib/native-app");

    expect(
      parseNativeOAuthCallback(
        "xsewebfast://callback-anywhere?ticket=ticket-value&url=https%3A%2F%2Fxse.yuelk.com%2Fmessages",
      ),
    ).toEqual({
      error: "",
      returnUrl: "https://xse.yuelk.com/messages",
      ticket: "ticket-value",
    });

    expect(
      parseNativeOAuthCallback(
        "xsewebfast://messages?ticket=ticket-value",
      ),
    ).toEqual({
      error: "",
      returnUrl: "https://xse.yuelk.com/messages?ticket=ticket-value",
      ticket: "ticket-value",
    });
  });

  it("opens the browser login bridge with callback and safe return URL", async () => {
    vi.stubGlobal("window", {
      location: { href: "https://xse.yuelk.com/messages?tab=unread" },
    });
    const { startNativeOAuth } = await import("../src/lib/native-app");

    await expect(startNativeOAuth("/api/auth/oauth2/login")).resolves.toBe(true);
    expect(browserOpen).toHaveBeenCalledOnce();
    const opened = new URL(browserOpen.mock.calls[0][0].url);
    expect(opened.pathname).toBe("/api/auth/oauth2/login");
    const callbackUrl = new URL(opened.searchParams.get("app_callback") ?? "");
    expect(callbackUrl.toString()).toBe(
      "xsewebfast://auth-return?url=https%3A%2F%2Fxse.yuelk.com%2Fmessages%3Ftab%3Dunread",
    );
    expect(opened.searchParams.get("app_return_url")).toBe(
      "https://xse.yuelk.com/messages?tab=unread",
    );
    expect(opened.searchParams.has("app_state")).toBe(false);
    expect(opened.searchParams.has("code_challenge")).toBe(false);
    vi.unstubAllGlobals();
  });

  it("uses explore as the native login fallback when launched from the login page", async () => {
    vi.stubGlobal("window", {
      location: { href: "https://xse.yuelk.com/login" },
    });
    const { startNativeOAuth } = await import("../src/lib/native-app");

    await expect(startNativeOAuth("/api/auth/oauth2/login")).resolves.toBe(true);
    const opened = new URL(browserOpen.mock.calls[0][0].url);
    const callbackUrl = new URL(opened.searchParams.get("app_callback") ?? "");
    expect(callbackUrl.searchParams.get("url")).toBe("https://xse.yuelk.com/explore");
    expect(opened.searchParams.get("app_return_url")).toBe(
      "https://xse.yuelk.com/explore",
    );
    vi.unstubAllGlobals();
  });

  it("keeps a safe next URL when native login is launched from the login page", async () => {
    vi.stubGlobal("window", {
      location: { href: "https://xse.yuelk.com/login?next=%2Fmessages" },
    });
    const { startNativeOAuth } = await import("../src/lib/native-app");

    await expect(startNativeOAuth("/api/auth/oauth2/login")).resolves.toBe(true);
    const opened = new URL(browserOpen.mock.calls[0][0].url);
    expect(opened.searchParams.get("app_return_url")).toBe(
      "https://xse.yuelk.com/messages",
    );
    vi.unstubAllGlobals();
  });

  it("rejects an external callback return URL", async () => {
    const { parseNativeOAuthCallback } = await import("../src/lib/native-app");

    expect(
      parseNativeOAuthCallback(
        "xsewebfast://auth-return?ticket=ticket-value&url=https%3A%2F%2Fevil.example%2Fsteal",
      ),
    ).toEqual({
      error: "",
      returnUrl: "https://xse.yuelk.com/explore",
      ticket: "ticket-value",
    });
  });

  it("describes native OAuth failures with useful diagnostics", async () => {
    const { ApiError } = await import("../src/lib/api/core/contracts");
    const { describeNativeOAuthError } = await import("../src/lib/native-app");

    expect(
      describeNativeOAuthError(
        new ApiError("error.oauth_app_ticket_invalid", {
          code: 401,
          status: 401,
        }),
      ),
    ).toBe("error.oauth_app_ticket_invalid (HTTP 401, code 401)");
    expect(describeNativeOAuthError(new Error("Network request failed"))).toBe(
      "Network request failed",
    );
  });

  it("claims the same OAuth callback only once across launch-url replays", async () => {
    vi.stubGlobal("window", { localStorage: memoryStorage() });
    const { claimNativeOAuthCallback, parseNativeOAuthCallback } = await import("../src/lib/native-app");
    const callback = parseNativeOAuthCallback(
      "xsewebfast://auth-return?ticket=ticket-replay&url=https%3A%2F%2Fxse.yuelk.com%2F",
    );

    expect(callback).not.toBeNull();
    expect(claimNativeOAuthCallback(callback!, 1_000)).toBe(true);
    expect(claimNativeOAuthCallback(callback!, 1_200)).toBe(false);
    expect(
      claimNativeOAuthCallback(
        { ...callback!, ticket: "ticket-next-login" },
        1_300,
      ),
    ).toBe(true);
    vi.unstubAllGlobals();
  });

  it("persists the last native OAuth failure for the login page", async () => {
    vi.stubGlobal("window", { localStorage: memoryStorage() });
    const {
      clearNativeOAuthStatus,
      readNativeOAuthStatus,
      recordNativeOAuthStatus,
    } = await import("../src/lib/native-app");

    recordNativeOAuthStatus(
      {
        detail: "native_token_storage_failed",
        ok: false,
        step: "token_storage_failed",
      },
      2_000,
    );

    expect(readNativeOAuthStatus(2_500)).toMatchObject({
      detail: "native_token_storage_failed",
      ok: false,
      step: "token_storage_failed",
    });
    expect(readNativeOAuthStatus(15 * 60 * 1_000)).toBeNull();
    clearNativeOAuthStatus();
    expect(readNativeOAuthStatus(2_600)).toBeNull();
    vi.unstubAllGlobals();
  });
});
