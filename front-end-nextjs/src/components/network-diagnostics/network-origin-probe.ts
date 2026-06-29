import type {
  HeaderPair,
  NetworkCdnSnapshot,
} from "@/lib/network-diagnostics";

export type OriginAddressSnapshot = {
  address: string;
  ip: string;
  port: string;
};

export type OriginServerSnapshot = {
  cdn: NetworkCdnSnapshot;
  clientIp: string;
  connection: {
    local: OriginAddressSnapshot;
    remote: OriginAddressSnapshot;
  };
  forwarded: {
    chain: string[];
    host: string;
    method: string;
    port: string;
    protocol: string;
    uri: string;
  };
  headers: {
    cdn: HeaderPair[];
    diagnostic: HeaderPair[];
    forwarded: HeaderPair[];
  };
  request: {
    host: string;
    method: string;
    path: string;
    protocol: string;
    protocolMajor: number;
    protocolMinor: number;
    query: string;
    requestUri: string;
    scheme: string;
    tls: boolean;
    tlsCipher: string;
    tlsVersion: string;
    userAgent: string;
  };
  server: {
    hostname: string;
  };
  timestamp: string;
};

type BackendEnvelope<T> = {
  code?: number;
  data?: T;
  message?: string;
};

export async function fetchOriginServerSnapshot(): Promise<OriginServerSnapshot> {
  const payload = await fetchJsonWithTimeout<BackendEnvelope<OriginServerSnapshot>>(
    "/api/diagnostics/network",
    6_000,
  );
  if (!payload.data) {
    throw new Error(payload.message ?? "Missing backend diagnostics payload.");
  }
  return payload.data;
}

async function fetchJsonWithTimeout<T>(url: string, timeoutMs: number): Promise<T> {
  const response = await fetchWithTimeout(url, timeoutMs, {
    headers: {
      Accept: "application/json",
    },
  });
  return response.json() as Promise<T>;
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
