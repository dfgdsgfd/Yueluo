import type { NextRequest } from "next/server";

const DEFAULT_BACKEND_ORIGIN = "https://xse.yuelk.com";

type ApiProxyContext = {
  params: Promise<{
    path?: string[];
  }>;
};

const hopByHopHeaders = [
  "connection",
  "content-length",
  "host",
  "keep-alive",
  "origin",
  "proxy-authenticate",
  "proxy-authorization",
  "referer",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
];

const defaultClientIPHeaderNames = [
  "x-real-ip",
  "x-forwarded-for",
  "cf-connecting-ip",
  "x-custom-real-ip",
  "true-client-ip",
  "x-client-ip",
  "forwarded",
] as const;

const clientIPForwardHeaderNames = [
  "x-real-ip",
  "x-forwarded-for",
  "cf-connecting-ip",
  "x-custom-real-ip",
] as const;

const headerNamePattern = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;

type HeadersWithSetCookie = Headers & {
  getSetCookie?: () => string[];
};

function getBackendOrigin() {
  return (
    process.env.API_BASE_URL ??
    process.env.BACKEND_ORIGIN ??
    process.env.NEXT_PUBLIC_BACKEND_ORIGIN ??
    DEFAULT_BACKEND_ORIGIN
  ).replace(/\/$/, "");
}

async function proxyApiRequest(request: NextRequest, context: ApiProxyContext) {
  const { path = [] } = await context.params;
  const targetUrl = new URL(
    `/api/${path.map((segment) => encodeURIComponent(segment)).join("/")}`,
    getBackendOrigin(),
  );
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers(request.headers);
  for (const header of hopByHopHeaders) {
    headers.delete(header);
  }
  preserveForwardingHeaders(headers, request);

  const method = request.method.toUpperCase();
  const fetchOptions: RequestInit & { duplex?: "half" } = {
    cache: "no-store",
    headers,
    method,
    redirect: "manual",
    signal: AbortSignal.any([
      request.signal,
      AbortSignal.timeout(30_000),
    ]),
  };

  if (method !== "GET" && method !== "HEAD") {
    fetchOptions.body = request.body;
    fetchOptions.duplex = "half";
  }

  const diagnosticsRequest = isOriginDiagnosticsPath(targetUrl.pathname);
  let upstream: Response;
  try {
    upstream = await fetch(targetUrl, fetchOptions);
  } catch (error) {
    if (diagnosticsRequest) {
      return optionalDiagnosticsFailureResponse(502, getErrorMessage(error));
    }
    throw error;
  }
  if (diagnosticsRequest && !upstream.ok) {
    return optionalDiagnosticsFailureResponse(upstream.status, `${upstream.status} ${upstream.statusText}`);
  }
  const responseHeaders = copyResponseHeaders(upstream.headers);
  responseHeaders.delete("content-encoding");
  responseHeaders.delete("content-length");
  responseHeaders.delete("transfer-encoding");

  return new Response(upstream.body, {
    headers: responseHeaders,
    status: upstream.status,
    statusText: upstream.statusText,
  });
}

function isOriginDiagnosticsPath(pathname: string) {
  return pathname === "/api/diagnostics/network";
}

function optionalDiagnosticsFailureResponse(code: number, message: string) {
  return Response.json(
    {
      code,
      data: null,
      message,
    },
    {
      headers: {
        "Cache-Control": "no-store",
      },
      status: 200,
    },
  );
}

function preserveForwardingHeaders(headers: Headers, request: NextRequest) {
  const host = request.headers.get("host");
  const protocol = firstHeaderValue(request.headers, "x-forwarded-proto")
    ?? request.nextUrl.protocol.replace(/:$/, "");
  const port = firstHeaderValue(request.headers, "x-forwarded-port")
    ?? portFromHost(host)
    ?? defaultPortForProtocol(protocol);
  const clientIp = firstHeaderValue(
    request.headers,
    ...clientIPHeaderNamesFromEnv(process.env.CLIENT_IP_HEADERS),
  );

  setHeaderIfMissing(headers, "x-forwarded-host", host);
  setHeaderIfMissing(headers, "x-forwarded-proto", protocol);
  setHeaderIfMissing(headers, "x-forwarded-port", port);
  setHeaderIfMissing(headers, "x-forwarded-method", request.method.toUpperCase());
  setHeaderIfMissing(headers, "x-forwarded-uri", `${request.nextUrl.pathname}${request.nextUrl.search}`);
  setHeaderIfMissing(headers, "x-original-host", host);
  setHeaderIfMissing(headers, "x-original-uri", `${request.nextUrl.pathname}${request.nextUrl.search}`);
  if (clientIp) {
    for (const header of clientIPForwardHeaderNames) {
      setHeaderIfMissing(headers, header, clientIp);
    }
  }
}

function clientIPHeaderNamesFromEnv(value: string | undefined) {
  const configured = uniqueHeaderNames(
    (value ?? "")
      .split(",")
      .map((name) => name.trim().toLowerCase())
      .filter((name) => name && headerNamePattern.test(name)),
  );
  return configured.length > 0 ? configured : [...defaultClientIPHeaderNames];
}

function uniqueHeaderNames(names: readonly string[]) {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const name of names) {
    if (!seen.has(name)) {
      seen.add(name);
      out.push(name);
    }
  }
  return out;
}

function setHeaderIfMissing(headers: Headers, name: string, value: string | null | undefined) {
  if (!value || headers.get(name)?.trim()) {
    return;
  }
  headers.set(name, value);
}

function firstHeaderValue(headers: Headers, ...names: string[]) {
  for (const name of names) {
    const value = headers.get(name)?.trim();
    if (value) {
      return value;
    }
  }
  return null;
}

function portFromHost(host: string | null) {
  if (!host) {
    return null;
  }
  const match = host.match(/:(\d+)$/);
  return match?.[1] ?? null;
}

function defaultPortForProtocol(protocol: string | null) {
  if (protocol === "https") {
    return "443";
  }
  if (protocol === "http") {
    return "80";
  }
  return null;
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function copyResponseHeaders(headers: Headers) {
  const responseHeaders = new Headers(headers);
  const setCookies = (headers as HeadersWithSetCookie).getSetCookie?.() ?? [];
  if (setCookies.length > 0) {
    responseHeaders.delete("set-cookie");
    for (const cookie of setCookies) {
      responseHeaders.append("set-cookie", cookie);
    }
  }
  return responseHeaders;
}

export {
  proxyApiRequest as DELETE,
  proxyApiRequest as GET,
  proxyApiRequest as HEAD,
  proxyApiRequest as OPTIONS,
  proxyApiRequest as PATCH,
  proxyApiRequest as POST,
  proxyApiRequest as PUT,
};
