import type { ApiEnvelope } from "../../types/auth";
import { ApiError } from "./contracts";

export async function parseResponse(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as unknown;
  }

  const text = await response.text();
  return text ? { message: text } : null;
}

export function filenameFromContentDisposition(value: string | null) {
  if (!value) {
    return null;
  }

  const utf8Match = value.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    return decodeURIComponent(utf8Match[1].replace(/^"|"$/g, ""));
  }

  const asciiMatch = value.match(/filename="?([^";]+)"?/i);
  return asciiMatch?.[1] ?? null;
}

export function extractEnvelope<T>(payload: unknown, status: number, requestId?: string | null): T {
  const envelope = payload as Partial<ApiEnvelope<T>> | null;

  if (!envelope || typeof envelope !== "object") {
    return payload as T;
  }

  if ("code" in envelope && envelope.code !== 200) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      code: envelope.code,
      status,
      details: responseErrorDetails(payload, requestId),
    });
  }

  if ("success" in envelope && envelope.success === false) {
    throw new ApiError(envelope.message ?? "API request failed.", {
      status,
      details: responseErrorDetails(payload, requestId),
    });
  }

  if ("data" in envelope) {
    return envelope.data as T;
  }

  if ("code" in envelope || "success" in envelope) {
    return {} as T;
  }

  return payload as T;
}

export function responseErrorDetails(payload: unknown, requestId?: string | null) {
  const normalizedRequestID = requestId?.trim();
  if (!normalizedRequestID) return payload;
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return { ...(payload as Record<string, unknown>), requestId: normalizedRequestID };
  }
  return { payload, requestId: normalizedRequestID };
}

export function responseErrorCode(payload: unknown) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  const code = Number((payload as { code?: unknown }).code);
  return Number.isFinite(code) ? code : undefined;
}
