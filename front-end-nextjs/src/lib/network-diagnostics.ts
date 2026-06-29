export type HeaderPair = {
  name: string;
  value: string;
};

export type NetworkCdnProvider =
  | "alibaba"
  | "cloudflare"
  | "cloudfront"
  | "fastly"
  | "netlify"
  | "tencent"
  | "vercel"
  | "unknown";

export type NetworkCdnSnapshot = {
  cacheStatus: string | null;
  countryCode: string | null;
  detected: boolean;
  evidence: HeaderPair[];
  provider: NetworkCdnProvider | null;
  rayId: string | null;
};

export type NetworkDiagnosticsServerPayload = {
  cdn: NetworkCdnSnapshot;
  forwardedIps: string[];
  headers: HeaderPair[];
  host: string | null;
  observedIp: string | null;
  protocol: string | null;
  requestUrl: string;
  timestamp: string;
  userAgent: string | null;
};

const clientIpHeaders = [
  "cf-connecting-ip",
  "true-client-ip",
  "x-real-ip",
  "x-client-ip",
  "x-forwarded-for",
  "fastly-client-ip",
  "x-vercel-forwarded-for",
];

const protocolHeaders = [
  "x-forwarded-proto",
  "cloudfront-forwarded-proto",
  "x-forwarded-protocol",
  "x-url-scheme",
  "x-scheme",
];

const diagnosticHeaderNames = [
  "accept-language",
  "cdn-loop",
  "cf-cache-status",
  "cf-connecting-ip",
  "cf-ipcountry",
  "cf-ray",
  "cloudfront-forwarded-proto",
  "cloudfront-viewer-country",
  "fastly-client-ip",
  "forwarded",
  "host",
  "true-client-ip",
  "user-agent",
  "via",
  "x-amz-cf-id",
  "x-cache",
  "x-cdn",
  "x-client-ip",
  "x-forwarded-for",
  "x-forwarded-host",
  "x-forwarded-proto",
  "x-nf-request-id",
  "x-nws-log-uuid",
  "x-real-ip",
  "x-request-id",
  "x-served-by",
  "x-tencent-ua",
  "x-vercel-cache",
  "x-vercel-id",
  "x-vercel-ip-country",
];

const responseDiagnosticHeaderNames = [
  "age",
  "alt-svc",
  "cache-control",
  "cdn-cache-control",
  "cdn-loop",
  "cf-apo-via",
  "cf-cache-status",
  "cf-ray",
  "content-encoding",
  "server",
  "vary",
  "via",
  "x-amz-cf-id",
  "x-cache",
  "x-cdn",
  "x-nf-request-id",
  "x-nws-log-uuid",
  "x-served-by",
  "x-vercel-cache",
  "x-vercel-id",
];

const cdnEvidenceHeaders = [
  "cdn-loop",
  "cf-cache-status",
  "cf-connecting-ip",
  "cf-ipcountry",
  "cf-ray",
  "cloudfront-viewer-country",
  "fastly-client-ip",
  "via",
  "x-amz-cf-id",
  "x-cache",
  "x-cdn",
  "x-nf-request-id",
  "x-nws-log-uuid",
  "x-served-by",
  "x-vercel-cache",
  "x-vercel-id",
  "x-vercel-ip-country",
];

export function createNetworkServerSnapshot(
  headers: Headers,
  requestUrl: string,
): NetworkDiagnosticsServerPayload {
  const requestProtocol = getProtocolFromUrl(requestUrl);

  return {
    cdn: detectCdn(headers),
    forwardedIps: getForwardedIps(headers),
    headers: pickHeaders(headers, diagnosticHeaderNames),
    host: getFirstHeader(headers, ["x-forwarded-host", "host"]),
    observedIp: getObservedIp(headers),
    protocol: getObservedProtocol(headers, requestProtocol),
    requestUrl,
    timestamp: new Date().toISOString(),
    userAgent: headers.get("user-agent"),
  };
}

export function detectCdn(headers: Headers): NetworkCdnSnapshot {
  const provider = detectCdnProvider(headers);
  const evidence = pickHeaders(headers, cdnEvidenceHeaders);

  return {
    cacheStatus: getFirstHeader(headers, [
      "cf-cache-status",
      "x-cache",
      "x-vercel-cache",
    ]),
    countryCode: getFirstHeader(headers, [
      "cf-ipcountry",
      "cloudfront-viewer-country",
      "x-vercel-ip-country",
    ]),
    detected: provider !== null,
    evidence,
    provider,
    rayId: getFirstHeader(headers, [
      "cf-ray",
      "x-amz-cf-id",
      "x-nf-request-id",
      "x-vercel-id",
      "x-nws-log-uuid",
      "x-request-id",
    ]),
  };
}

export function pickNetworkDiagnosticHeaders(headers: Headers) {
  return pickHeaders(headers, diagnosticHeaderNames);
}

export function pickNetworkResponseDiagnosticHeaders(headers: Headers) {
  return pickHeaders(headers, responseDiagnosticHeaderNames);
}

function detectCdnProvider(headers: Headers): NetworkCdnProvider | null {
  const via = headers.get("via")?.toLowerCase() ?? "";
  const xCache = headers.get("x-cache")?.toLowerCase() ?? "";
  const xCdn = headers.get("x-cdn")?.toLowerCase() ?? "";
  const cdnLoop = headers.get("cdn-loop")?.toLowerCase() ?? "";

  if (hasAnyHeader(headers, ["cf-ray", "cf-cache-status", "cf-connecting-ip"]) || cdnLoop.includes("cloudflare")) {
    return "cloudflare";
  }

  if (hasAnyHeader(headers, ["x-amz-cf-id", "cloudfront-viewer-country"]) || via.includes("cloudfront")) {
    return "cloudfront";
  }

  if (hasAnyHeader(headers, ["fastly-client-ip", "x-served-by"]) || xCache.includes("fastly")) {
    return "fastly";
  }

  if (hasAnyHeader(headers, ["x-vercel-id", "x-vercel-cache", "x-vercel-ip-country"])) {
    return "vercel";
  }

  if (hasAnyHeader(headers, ["x-nf-request-id"])) {
    return "netlify";
  }

  if (hasAnyHeader(headers, ["x-nws-log-uuid", "x-tencent-ua"]) || xCdn.includes("tencent")) {
    return "tencent";
  }

  if (via.includes("kunlun") || via.includes("alicdn") || xCdn.includes("alibaba")) {
    return "alibaba";
  }

  return evidenceSuggestsCdn(headers) ? "unknown" : null;
}

function getObservedIp(headers: Headers) {
  for (const name of clientIpHeaders) {
    const value = headers.get(name);
    if (!value) {
      continue;
    }
    return name === "x-forwarded-for" ? splitForwardedList(value)[0] ?? value : value.trim();
  }
  return null;
}

function getForwardedIps(headers: Headers) {
  const values = [
    ...splitForwardedList(headers.get("x-forwarded-for")),
    ...extractForwardedFor(headers.get("forwarded")),
  ];
  return Array.from(new Set(values));
}

function getObservedProtocol(headers: Headers, fallback: string | null) {
  const direct = getFirstHeader(headers, protocolHeaders);
  if (direct) {
    return direct.replace(/:$/, "").toLowerCase();
  }

  const forwardedProto = extractForwardedProto(headers.get("forwarded"));
  return forwardedProto ?? fallback;
}

function pickHeaders(headers: Headers, names: readonly string[]) {
  return names.flatMap((name) => {
    const value = headers.get(name);
    return value ? [{ name, value }] : [];
  });
}

function getFirstHeader(headers: Headers, names: readonly string[]) {
  for (const name of names) {
    const value = headers.get(name)?.trim();
    if (value) {
      return value;
    }
  }
  return null;
}

function hasAnyHeader(headers: Headers, names: readonly string[]) {
  return names.some((name) => Boolean(headers.get(name)));
}

function evidenceSuggestsCdn(headers: Headers) {
  return pickHeaders(headers, cdnEvidenceHeaders).length > 0;
}

function splitForwardedList(value: string | null) {
  return (value ?? "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function extractForwardedFor(value: string | null) {
  return splitForwardedList(value).flatMap((entry) => {
    const match = entry.match(/(?:^|;)for="?([^";,]+)"?/i);
    return match?.[1] ? [match[1].trim()] : [];
  });
}

function extractForwardedProto(value: string | null) {
  for (const entry of splitForwardedList(value)) {
    const match = entry.match(/(?:^|;)proto="?([^";,]+)"?/i);
    if (match?.[1]) {
      return match[1].replace(/:$/, "").toLowerCase();
    }
  }
  return null;
}

function getProtocolFromUrl(value: string) {
  try {
    return new URL(value).protocol.replace(/:$/, "");
  } catch {
    return null;
  }
}
