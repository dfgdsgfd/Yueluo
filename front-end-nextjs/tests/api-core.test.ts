import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ACCESS_TOKEN_KEY,
  ADMIN_ACCESS_TOKEN_KEY,
  ApiError,
  apiAdminRequest,
  apiRequestContextFromHeaders,
  apiRequest,
  buildApiUrl,
  extractEnvelope,
  filenameFromContentDisposition,
  getCookieValue,
  normalizeAdminPagination,
  normalizeAuthToken,
  REFRESH_TOKEN_KEY,
  uploadProgressWithStage,
} from "../src/lib/api/core";
import { isAccessBlockApiError } from "../src/lib/api/core/access-block";
import { createRequestSignal } from "../src/lib/api/core/http";

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

function fetchUrl(input: Parameters<typeof fetch>[0]) {
  if (typeof input === "string") {
    return input;
  }
  if (input instanceof URL) {
    return input.toString();
  }
  return input.url;
}

function restoreEnv(name: string, value: string | undefined) {
  if (value === undefined) {
    delete process.env[name];
    return;
  }
  process.env[name] = value;
}

describe("API core compatibility", () => {
  const originalApiBase = process.env.API_BASE_URL;
  const originalClientIPHeaders = process.env.CLIENT_IP_HEADERS;

  beforeEach(() => {
    delete process.env.CLIENT_IP_HEADERS;
    const localStorage = memoryStorage();
    vi.stubGlobal("window", {
      localStorage,
      dispatchEvent: vi.fn(),
      location: {
        assign: vi.fn(),
        origin: "https://frontend.example",
        protocol: "https:",
      },
    });
    vi.stubGlobal("document", { cookie: "" });
  });

  afterEach(() => {
    restoreEnv("API_BASE_URL", originalApiBase);
    restoreEnv("CLIENT_IP_HEADERS", originalClientIPHeaders);
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("preserves URL, token and cookie normalization", () => {
    expect(normalizeAuthToken(" Bearer abc ")).toBe("abc");
    expect(getCookieValue("a=1; encoded=hello%20world", "encoded")).toBe(
      "hello world",
    );
    expect(
      buildApiUrl("/api/posts", { page: 2, empty: "", enabled: true }),
    ).toBe("/api/posts?page=2&enabled=true");
  });

  it("uses CLIENT_IP_HEADERS to order forwarded client IP headers", () => {
    process.env.CLIENT_IP_HEADERS = ",X-Real-IP,X-Forwarded-For,CF-Connecting-IP";
    const context = apiRequestContextFromHeaders(
      new Headers({
        "accept-language": "zh-CN",
        "cf-connecting-ip": "198.51.100.10",
        "true-client-ip": "192.0.2.2",
        "user-agent": "vitest",
        "x-forwarded-for": "203.0.113.7, 10.0.0.4",
        "x-real-ip": "198.51.100.9",
      }),
    );

    expect(Object.keys(context.forwardedHeaders ?? {})).toEqual([
      "user-agent",
      "accept-language",
      "x-real-ip",
      "x-forwarded-for",
      "cf-connecting-ip",
    ]);
    expect(context.forwardedHeaders).toMatchObject({
      "cf-connecting-ip": "198.51.100.10",
      "x-forwarded-for": "203.0.113.7, 10.0.0.4",
      "x-real-ip": "198.51.100.9",
    });
    expect(context.forwardedHeaders?.["true-client-ip"]).toBeUndefined();
  });

  it("preserves envelope and response metadata behavior", () => {
    expect(
      extractEnvelope<{ id: number }>({ code: 200, data: { id: 7 } }, 200),
    ).toEqual({ id: 7 });
    expect(() =>
      extractEnvelope({ code: 422, message: "invalid" }, 422, "request-1"),
    ).toThrowError(ApiError);
    try {
      extractEnvelope({ code: 422, message: "invalid" }, 422, "request-1");
    } catch (error) {
      expect((error as ApiError).details).toEqual({
        code: 422,
        message: "invalid",
        requestId: "request-1",
      });
    }
    expect(
      filenameFromContentDisposition(
        "attachment; filename*=UTF-8''hello%20world.zip",
      ),
    ).toBe("hello world.zip");
  });

  it("retries once with a refreshed access token after 401", async () => {
    window.localStorage.setItem(REFRESH_TOKEN_KEY, "refresh-old");
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ message: "expired" }), { status: 401 }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 200,
            data: { access_token: "access-new", refresh_token: "refresh-new" },
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ code: 200, data: { ok: true } }), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest<{ ok: boolean }>("/api/private")).resolves.toEqual({
      ok: true,
    });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("access-new");
    expect(window.localStorage.getItem(REFRESH_TOKEN_KEY)).toBe("refresh-new");
    const retryHeaders = new Headers(fetchMock.mock.calls[2]?.[1]?.headers);
    expect(retryHeaders.get("authorization")).toBe("Bearer access-new");
  });

  it("coalesces parallel 401 refreshes into a single token refresh", async () => {
    window.localStorage.setItem(ACCESS_TOKEN_KEY, "access-old");
    window.localStorage.setItem(REFRESH_TOKEN_KEY, "refresh-old");
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(
      async (input, init) => {
        const url = fetchUrl(input);
        const headers = new Headers(init?.headers);
        if (url === "/api/auth/refresh") {
          return new Response(
            JSON.stringify({
              code: 200,
              data: { access_token: "access-new", refresh_token: "refresh-new" },
            }),
            { status: 200, headers: { "content-type": "application/json" } },
          );
        }
        if (headers.get("authorization") === "Bearer access-new") {
          return new Response(JSON.stringify({ code: 200, data: { ok: true } }), {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        }
        return new Response(JSON.stringify({ message: "expired" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        });
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      Promise.all([
        apiRequest<{ ok: boolean }>("/api/private-a"),
        apiRequest<{ ok: boolean }>("/api/private-b"),
      ]),
    ).resolves.toEqual([{ ok: true }, { ok: true }]);
    expect(
      fetchMock.mock.calls.filter(([input]) => fetchUrl(input) === "/api/auth/refresh"),
    ).toHaveLength(1);
    expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("access-new");
    expect(window.localStorage.getItem(REFRESH_TOKEN_KEY)).toBe("refresh-new");
  });

  it("retries with the latest stored access token before clearing the session", async () => {
    window.localStorage.setItem(ACCESS_TOKEN_KEY, "access-old");
    window.localStorage.setItem(REFRESH_TOKEN_KEY, "refresh-old");
    const assign = vi.mocked(window.location.assign);
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(
      async (input, init) => {
        const url = fetchUrl(input);
        const headers = new Headers(init?.headers);
        const authorization = headers.get("authorization");
        if (url === "/api/auth/refresh") {
          return new Response(
            JSON.stringify({
              code: 200,
              data: { access_token: "access-one", refresh_token: "refresh-old" },
            }),
            { status: 200, headers: { "content-type": "application/json" } },
          );
        }
        if (authorization === "Bearer access-one") {
          window.localStorage.setItem(ACCESS_TOKEN_KEY, "access-two");
          return new Response(JSON.stringify({ message: "rotated" }), {
            status: 401,
            headers: { "content-type": "application/json" },
          });
        }
        if (authorization === "Bearer access-two") {
          return new Response(JSON.stringify({ code: 200, data: { ok: true } }), {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        }
        return new Response(JSON.stringify({ message: "expired" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        });
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest<{ ok: boolean }>("/api/private")).resolves.toEqual({
      ok: true,
    });
    expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("access-two");
    expect(window.localStorage.getItem(REFRESH_TOKEN_KEY)).toBe("refresh-old");
    expect(assign).not.toHaveBeenCalled();
  });

  it("can keep background 401 failures from clearing the active session", async () => {
    window.localStorage.setItem(ACCESS_TOKEN_KEY, "access-old");
    const assign = vi.mocked(window.location.assign);
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(
        new Response(JSON.stringify({ message: "expired" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    await expect(
      apiRequest("/api/im/conversations", {
        redirectOnUnauthorized: false,
        retryOnUnauthorized: false,
        timeoutMs: 1,
      }),
    ).rejects.toMatchObject({ status: 401 });
    expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBe("access-old");
    expect(assign).not.toHaveBeenCalled();
  });

  it("throws access-block details for 444 responses before auth handling", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(null, {
        status: 444,
        headers: {
          "X-Access-Block": "1",
          "X-Access-Block-Action": "status",
          "X-Access-Block-Rule-ID": "17",
          "X-Access-Block-Status-Code": "444",
          "X-Request-ID": "request-blocked",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    let thrown: unknown;
    try {
      await apiRequest("/api/posts/blocked", {
        retryOnUnauthorized: false,
        redirectOnUnauthorized: false,
      });
    } catch (error) {
      thrown = error;
    }

    expect(fetchMock.mock.calls[0]?.[1]?.redirect).toBe("manual");
    expect(isAccessBlockApiError(thrown)).toBe(true);
    expect(thrown).toBeInstanceOf(ApiError);
    expect((thrown as ApiError).details).toMatchObject({
      accessBlocked: true,
      action: "status",
      requestId: "request-blocked",
      ruleId: "17",
      statusCode: 444,
    });
  });

  it("handles manual redirect access-block responses without following content", async () => {
    const assign = vi.mocked(window.location.assign);
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(null, {
        status: 302,
        headers: {
          "Location": "https://blocked.example/landing",
          "X-Access-Block": "1",
          "X-Access-Block-Action": "redirect",
          "X-Access-Block-Rule-ID": "18",
          "X-Access-Block-Status-Code": "302",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      apiRequest("/api/posts/redirected", {
        retryOnUnauthorized: false,
        redirectOnUnauthorized: false,
      }),
    ).rejects.toMatchObject({ message: "error.access_blocked", status: 302 });
    expect(fetchMock.mock.calls[0]?.[1]?.redirect).toBe("manual");
    expect(assign).toHaveBeenCalledWith("https://blocked.example/landing");
  });

  it("keeps admin data 401 failures from clearing the admin session by default", async () => {
    window.localStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, "admin-old");
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(
        new Response(JSON.stringify({ message: "expired" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    await expect(
      apiAdminRequest("/api/admin/dashboard/overview"),
    ).rejects.toMatchObject({ status: 401 });
    expect(window.localStorage.getItem(ADMIN_ACCESS_TOKEN_KEY)).toBe("admin-old");
  });

  it("clears the admin session when the authoritative admin check returns 401", async () => {
    window.localStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, "admin-old");
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(
        new Response(JSON.stringify({ message: "expired" }), {
          status: 401,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    await expect(
      apiAdminRequest("/api/auth/admin/me", {
        clearSessionOnUnauthorized: true,
      }),
    ).rejects.toMatchObject({ status: 401 });
    expect(window.localStorage.getItem(ADMIN_ACCESS_TOKEN_KEY)).toBeNull();
  });

  it("keeps abort, pagination and upload progress helpers stable", () => {
    const controller = new AbortController();
    const signal = createRequestSignal(0, controller.signal);
    controller.abort("cancelled");
    expect(signal?.aborted).toBe(true);
    expect(
      normalizeAdminPagination(
        { page: 2, pageSize: 10, total: 25 },
        { page: 1, limit: 20, itemCount: 0 },
      ),
    ).toEqual({
      page: 2,
      limit: 10,
      pageSize: 10,
      total: 25,
      pages: 3,
      totalPages: 3,
      hasNextPage: true,
    });
    expect(
      uploadProgressWithStage(
        { loaded: 5, total: 10 },
        "processing",
        "Processing",
      ),
    ).toEqual({
      loaded: 5,
      total: 10,
      stage: "processing",
      message: "Processing",
    });
  });
});
