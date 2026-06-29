import {
  detectCdn,
  pickNetworkResponseDiagnosticHeaders,
  type HeaderPair,
  type NetworkCdnSnapshot,
  type NetworkDiagnosticsServerPayload,
} from "@/lib/network-diagnostics";
import {
  fetchOriginServerSnapshot,
  type OriginServerSnapshot,
} from "./network-origin-probe";

export type BrowserNetworkSnapshot = {
  downlinkMbps: number | null;
  effectiveType: string | null;
  host: string | null;
  language: string | null;
  languages: string[];
  online: boolean;
  protocol: string | null;
  platform: string | null;
  rttMs: number | null;
  saveData: boolean | null;
  screen: string | null;
  timezone: string | null;
  type: string | null;
  userAgent: string | null;
  viewport: string | null;
};

export type PublicIpSnapshot = {
  fetchedAt: string;
  ip: string;
  provider: string;
};

export type GeoSnapshot = {
  asn: string | null;
  city: string | null;
  countryCode: string | null;
  countryName: string | null;
  ip: string | null;
  isp: string | null;
  latitude: number | null;
  longitude: number | null;
  provider: string;
  region: string | null;
  timezone: string | null;
};

export type CloudflareTraceSnapshot = {
  colo: string | null;
  gateway: string | null;
  http: string | null;
  ip: string | null;
  loc: string | null;
  tls: string | null;
  warp: string | null;
};

export type WebRtcAddressKind = "loopback" | "mdns" | "private" | "public" | "unknown";

export type WebRtcAddressExposure = {
  address: string;
  kind: WebRtcAddressKind;
  source: "candidate" | "sdp";
};

export type WebRtcSnapshot = {
  addresses: WebRtcAddressExposure[];
  error: string | null;
  supported: boolean;
};

export type NetworkProbeError = {
  message: string;
  source: "cloudflareTrace" | "geo" | "originServer" | "pageResponse" | "publicIp" | "server";
};

export type EdgeResponseSnapshot = {
  cdn: NetworkCdnSnapshot;
  headers: HeaderPair[];
};

export type NetworkDiagnosticsSnapshot = {
  browser: BrowserNetworkSnapshot;
  cloudflareTrace: CloudflareTraceSnapshot | null;
  durationMs: number;
  edgeResponse: EdgeResponseSnapshot | null;
  errors: NetworkProbeError[];
  generatedAt: string;
  geo: GeoSnapshot | null;
  originServer: OriginServerSnapshot | null;
  pageResponse: EdgeResponseSnapshot | null;
  publicIp: PublicIpSnapshot | null;
  server: NetworkDiagnosticsServerPayload | null;
  serverLatencyMs: number | null;
  webRtc: WebRtcSnapshot;
};

type BrowserNetworkConnection = {
  downlink?: number;
  effectiveType?: string;
  rtt?: number;
  saveData?: boolean;
  type?: string;
};

type NavigatorWithConnection = Navigator & {
  connection?: BrowserNetworkConnection;
  mozConnection?: BrowserNetworkConnection;
  webkitConnection?: BrowserNetworkConnection;
};

type IpWhoIsPayload = {
  city?: string;
  connection?: {
    asn?: number | string;
    isp?: string;
    org?: string;
  };
  country?: string;
  country_code?: string;
  ip?: string;
  latitude?: number;
  longitude?: number;
  region?: string;
  success?: boolean;
  timezone?: {
    id?: string;
  };
};

type IpApiPayload = {
  asn?: string;
  city?: string;
  country_code?: string;
  country_name?: string;
  ip?: string;
  latitude?: number;
  longitude?: number;
  org?: string;
  region?: string;
  timezone?: string;
};

const publicIpProviders = [
  {
    name: "ipify IPv6",
    url: "https://api64.ipify.org?format=json",
  },
  {
    name: "ipify IPv4",
    url: "https://api.ipify.org?format=json",
  },
] as const;

export async function runNetworkDiagnostics(): Promise<NetworkDiagnosticsSnapshot> {
  const startedAt = performance.now();
  const browser = collectBrowserNetworkSnapshot();
  const errors: NetworkProbeError[] = [];

  const [serverResult, originServerResult, pageResponseResult, publicIpResult, geoResult, traceResult, webRtcResult] =
    await Promise.allSettled([
      fetchServerSnapshot(),
      fetchOriginServerSnapshot(),
      fetchPageResponseSnapshot(),
      fetchPublicIp(),
      fetchGeoSnapshot(),
      fetchCloudflareTrace(),
      probeWebRtcCandidates(),
    ]);

  const serverTimed = collectSettledValue(serverResult, errors, "server");
  const originServer = collectSettledValue(originServerResult, errors, "originServer");
  const pageResponse = collectSettledValue(pageResponseResult, errors, "pageResponse");
  const publicIp = collectSettledValue(publicIpResult, errors, "publicIp");
  const geo = collectSettledValue(geoResult, errors, "geo");
  const cloudflareTrace = collectSettledValue(traceResult, errors, "cloudflareTrace");
  const webRtc = webRtcResult.status === "fulfilled"
    ? webRtcResult.value
    : {
        addresses: [],
        error: getErrorMessage(webRtcResult.reason),
        supported: false,
      };

  return {
    browser,
    cloudflareTrace,
    durationMs: Math.round(performance.now() - startedAt),
    edgeResponse: serverTimed?.edgeResponse ?? null,
    errors,
    generatedAt: new Date().toISOString(),
    geo,
    originServer,
    pageResponse,
    publicIp,
    server: serverTimed?.data ?? null,
    serverLatencyMs: serverTimed?.latencyMs ?? null,
    webRtc,
  };
}

function collectBrowserNetworkSnapshot(): BrowserNetworkSnapshot {
  const nav = navigator as NavigatorWithConnection;
  const connection = nav.connection ?? nav.mozConnection ?? nav.webkitConnection;

  return {
    downlinkMbps: normalizeNumber(connection?.downlink),
    effectiveType: connection?.effectiveType ?? null,
    host: window.location.host || null,
    language: navigator.language || null,
    languages: Array.from(navigator.languages ?? []),
    online: navigator.onLine,
    protocol: window.location.protocol.replace(/:$/, "") || null,
    platform: navigator.platform || null,
    rttMs: normalizeNumber(connection?.rtt),
    saveData: typeof connection?.saveData === "boolean" ? connection.saveData : null,
    screen: typeof window.screen?.width === "number" ? `${window.screen.width}x${window.screen.height}` : null,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone ?? null,
    type: connection?.type ?? null,
    userAgent: navigator.userAgent || null,
    viewport: `${window.innerWidth}x${window.innerHeight}`,
  };
}

async function fetchServerSnapshot() {
  const startedAt = performance.now();
  const response = await fetchWithTimeout("/api/network-diagnostics", 6_000, {
    headers: {
      Accept: "application/json",
    },
  });
  const edgeResponse = collectEdgeResponseSnapshot(response.headers);
  const data = (await response.json()) as NetworkDiagnosticsServerPayload;
  return {
    data,
    edgeResponse,
    latencyMs: Math.round(performance.now() - startedAt),
  };
}

function collectEdgeResponseSnapshot(headers: Headers): EdgeResponseSnapshot {
  return {
    cdn: detectCdn(headers),
    headers: pickNetworkResponseDiagnosticHeaders(headers),
  };
}

async function fetchPageResponseSnapshot(): Promise<EdgeResponseSnapshot> {
  const response = await fetchCurrentPageHeaders();
  return collectEdgeResponseSnapshot(response.headers);
}

async function fetchCurrentPageHeaders() {
  try {
    return await fetchWithTimeout(window.location.href, 6_000, {
      credentials: "same-origin",
      headers: {
        Accept: "text/html",
      },
      method: "HEAD",
    });
  } catch {
    return fetchWithTimeout(window.location.href, 6_000, {
      credentials: "same-origin",
      headers: {
        Accept: "text/html",
      },
      method: "GET",
    });
  }
}

async function fetchPublicIp(): Promise<PublicIpSnapshot> {
  let lastError: unknown = null;
  for (const provider of publicIpProviders) {
    try {
      const payload = await fetchJsonWithTimeout<{ ip?: string }>(provider.url, 5_000);
      if (payload.ip) {
        return {
          fetchedAt: new Date().toISOString(),
          ip: payload.ip,
          provider: provider.name,
        };
      }
      lastError = new Error("Missing IP in provider response.");
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError ?? new Error("Public IP providers failed.");
}

async function fetchGeoSnapshot(): Promise<GeoSnapshot> {
  try {
    const payload = await fetchJsonWithTimeout<IpWhoIsPayload>("https://ipwho.is/", 6_000);
    if (payload.success === false) {
      throw new Error("ipwho.is returned an unsuccessful response.");
    }
    return {
      asn: stringifyNullable(payload.connection?.asn),
      city: payload.city ?? null,
      countryCode: payload.country_code ?? null,
      countryName: payload.country ?? null,
      ip: payload.ip ?? null,
      isp: payload.connection?.isp ?? payload.connection?.org ?? null,
      latitude: normalizeNumber(payload.latitude),
      longitude: normalizeNumber(payload.longitude),
      provider: "ipwho.is",
      region: payload.region ?? null,
      timezone: payload.timezone?.id ?? null,
    };
  } catch {
    const payload = await fetchJsonWithTimeout<IpApiPayload>("https://ipapi.co/json/", 6_000);
    return {
      asn: payload.asn ?? null,
      city: payload.city ?? null,
      countryCode: payload.country_code ?? null,
      countryName: payload.country_name ?? null,
      ip: payload.ip ?? null,
      isp: payload.org ?? null,
      latitude: normalizeNumber(payload.latitude),
      longitude: normalizeNumber(payload.longitude),
      provider: "ipapi.co",
      region: payload.region ?? null,
      timezone: payload.timezone ?? null,
    };
  }
}

async function fetchCloudflareTrace(): Promise<CloudflareTraceSnapshot> {
  const text = await fetchTextWithTimeout("https://www.cloudflare.com/cdn-cgi/trace", 5_000);
  const values = Object.fromEntries(
    text
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const separatorIndex = line.indexOf("=");
        return separatorIndex > 0
          ? [line.slice(0, separatorIndex), line.slice(separatorIndex + 1)]
          : [line, ""];
      }),
  );

  return {
    colo: values.colo ?? null,
    gateway: values.gateway ?? null,
    http: values.http ?? null,
    ip: values.ip ?? null,
    loc: values.loc ?? null,
    tls: values.tls ?? null,
    warp: values.warp ?? null,
  };
}

async function probeWebRtcCandidates(timeoutMs = 4_500): Promise<WebRtcSnapshot> {
  if (typeof RTCPeerConnection === "undefined") {
    return {
      addresses: [],
      error: null,
      supported: false,
    };
  }

  const addresses = new Map<string, WebRtcAddressExposure>();
  const peer = new RTCPeerConnection({
    iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
  });

  return new Promise((resolve) => {
    let finished = false;
    const finish = (error: string | null = null) => {
      if (finished) {
        return;
      }
      finished = true;
      window.clearTimeout(timer);
      collectCandidateText(peer.localDescription?.sdp ?? "", "sdp", addresses);
      peer.close();
      resolve({
        addresses: Array.from(addresses.values()).sort((a, b) =>
          `${a.kind}:${a.address}`.localeCompare(`${b.kind}:${b.address}`),
        ),
        error,
        supported: true,
      });
    };

    const timer = window.setTimeout(() => finish(null), timeoutMs);
    peer.onicecandidate = (event) => {
      if (!event.candidate) {
        finish(null);
        return;
      }
      collectCandidateText(event.candidate.candidate, "candidate", addresses);
    };

    void (async () => {
      try {
        peer.createDataChannel("network-diagnostics");
        const offer = await peer.createOffer();
        await peer.setLocalDescription(offer);
      } catch (error) {
        finish(getErrorMessage(error));
      }
    })();
  });
}

function collectCandidateText(
  text: string,
  source: WebRtcAddressExposure["source"],
  addresses: Map<string, WebRtcAddressExposure>,
) {
  for (const address of extractAddresses(text)) {
    if (!addresses.has(address)) {
      addresses.set(address, {
        address,
        kind: classifyAddress(address),
        source,
      });
    }
  }
}

function extractAddresses(text: string) {
  const matches = [
    ...Array.from(text.matchAll(/\b(?:\d{1,3}\.){3}\d{1,3}\b/g), (match) => match[0]),
    ...Array.from(text.matchAll(/\b[a-z0-9-]+\.local\b/gi), (match) => match[0]),
    ...Array.from(text.matchAll(/\b(?:[a-f0-9]{1,4}:){2,}[a-f0-9:.]{1,}\b/gi), (match) => match[0]),
  ];

  return Array.from(new Set(matches));
}

function classifyAddress(address: string): WebRtcAddressKind {
  const lower = address.toLowerCase();
  if (lower.endsWith(".local")) {
    return "mdns";
  }
  if (lower === "127.0.0.1" || lower === "::1") {
    return "loopback";
  }

  const ipv4Parts = lower.split(".").map((part) => Number.parseInt(part, 10));
  if (ipv4Parts.length === 4 && ipv4Parts.every((part) => Number.isInteger(part) && part >= 0 && part <= 255)) {
    const [first, second] = ipv4Parts;
    if (
      first === 0 ||
      first === 10 ||
      first === 127 ||
      (first === 100 && second >= 64 && second <= 127) ||
      (first === 172 && second >= 16 && second <= 31) ||
      (first === 192 && second === 168) ||
      (first === 169 && second === 254) ||
      first >= 224
    ) {
      return first === 127 ? "loopback" : "private";
    }
    return "public";
  }

  if (lower.includes(":")) {
    if (
      lower.startsWith("fe80:") ||
      lower.startsWith("fc") ||
      lower.startsWith("fd") ||
      lower.startsWith("::ffff:10.") ||
      lower.startsWith("::ffff:192.168.")
    ) {
      return "private";
    }
    return "public";
  }

  return "unknown";
}

function collectSettledValue<T>(
  result: PromiseSettledResult<T>,
  errors: NetworkProbeError[],
  source: NetworkProbeError["source"],
) {
  if (result.status === "fulfilled") {
    return result.value;
  }
  errors.push({
    message: getErrorMessage(result.reason),
    source,
  });
  return null;
}

async function fetchJsonWithTimeout<T>(url: string, timeoutMs: number): Promise<T> {
  const response = await fetchWithTimeout(url, timeoutMs, {
    headers: {
      Accept: "application/json",
    },
  });
  return response.json() as Promise<T>;
}

async function fetchTextWithTimeout(url: string, timeoutMs: number): Promise<string> {
  const response = await fetchWithTimeout(url, timeoutMs, {
    headers: {
      Accept: "text/plain",
    },
  });
  return response.text();
}

async function fetchWithTimeout(
  url: string,
  timeoutMs: number,
  init: RequestInit = {},
) {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(url, {
      ...init,
      cache: "no-store",
      signal: controller.signal,
    });
    if (!response.ok) {
      throw new Error(`${response.status} ${response.statusText}`);
    }
    return response;
  } finally {
    window.clearTimeout(timeout);
  }
}

function normalizeNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function stringifyNullable(value: unknown) {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  return String(value);
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
