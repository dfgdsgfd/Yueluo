import type {
  AIGenerationLogsPayload,
  AIJobInput,
  AIJobListPayload,
  AIJobPayload,
  AIJobStreamEvent,
  AIJobStreamHandlers,
  AIModerationDebugInput,
  AIModerationDebugResult,
  AIPublishGenerationConfig,
  AIPublishGenerationInput,
  AIPublishGenerationResult,
  AIRequestInput,
  AISettingsUpdate,
  AIStreamEvent,
  AIStreamHandlers,
  AIPublicSettings,
} from "../types";
import {
  ApiError,
  apiRequest,
  applyAuthorizationHeader,
  buildApiUrl,
  getRequestAccessToken,
  getStoredAdminAccessToken,
  parseResponse,
  apiAdminRequest,
} from "./core";
import { createRequestSignal } from "./core/http";

type StreamOptions = {
  signal?: AbortSignal;
  timeoutMs?: number;
  token?: string | null;
};

export function streamAIFormatMarkdown(
  input: AIRequestInput,
  handlers: AIStreamHandlers,
  options: StreamOptions = {},
) {
  return streamAI("/api/ai/format-markdown/stream", input, handlers, {
    ...options,
    token: options.token ?? getRequestAccessToken(),
  });
}

export function streamAdminAIGenerate(
  input: AIRequestInput,
  handlers: AIStreamHandlers,
  options: StreamOptions = {},
) {
  return streamAI("/api/admin/ai/generate/stream", input, handlers, {
    ...options,
    token: options.token ?? getStoredAdminAccessToken(),
  });
}

export function createAIJob(input: AIJobInput, token?: string | null) {
  return apiRequest<AIJobPayload>("/api/ai/jobs", {
    method: "POST",
    context: token ? { token } : undefined,
    body: JSON.stringify(input),
  });
}

export function getAIJob(jobId: string, token?: string | null) {
  return apiRequest<AIJobPayload>(`/api/ai/jobs/${encodeURIComponent(jobId)}`, {
    method: "GET",
    context: token ? { token } : undefined,
  });
}

export function streamAIJob(
  jobId: string,
  handlers: AIJobStreamHandlers,
  options: StreamOptions = {},
) {
  return streamAIJobPath(`/api/ai/jobs/${encodeURIComponent(jobId)}/stream`, handlers, {
    ...options,
    token: options.token ?? getRequestAccessToken(),
  });
}

export function getActiveAIJob(requestHash: string, token?: string | null) {
  return apiRequest<AIJobPayload>("/api/ai/jobs/active", {
    method: "GET",
    context: token ? { token } : undefined,
    query: { requestHash },
  });
}

export function cancelAIJob(jobId: string, token?: string | null) {
  return apiRequest<AIJobPayload>(`/api/ai/jobs/${encodeURIComponent(jobId)}/cancel`, {
    method: "POST",
    context: token ? { token } : undefined,
  });
}

export function createAdminAIJob(input: AIJobInput, token?: string | null) {
  return apiAdminRequest<AIJobPayload>("/api/admin/ai/jobs", {
    method: "POST",
    token,
    body: JSON.stringify(input),
  });
}

export function getAdminAIJobs(
  query: Record<string, string | number | boolean | null | undefined> = {},
  token?: string | null,
) {
  return apiAdminRequest<AIJobListPayload>("/api/admin/ai/jobs", {
    method: "GET",
    token,
    query,
  });
}

export function getAdminAIJob(jobId: string, token?: string | null) {
  return apiAdminRequest<AIJobPayload>(`/api/admin/ai/jobs/${encodeURIComponent(jobId)}`, {
    method: "GET",
    token,
  });
}

export function cancelAdminAIJob(jobId: string, token?: string | null) {
  return apiAdminRequest<AIJobPayload>(`/api/admin/ai/jobs/${encodeURIComponent(jobId)}/cancel`, {
    method: "POST",
    token,
  });
}

export function streamAdminAIJob(
  jobId: string,
  handlers: AIJobStreamHandlers,
  options: StreamOptions = {},
) {
  return streamAIJobPath(`/api/admin/ai/jobs/${encodeURIComponent(jobId)}/stream`, handlers, {
    ...options,
    token: options.token ?? getStoredAdminAccessToken(),
  });
}

export function getAdminAISettings(token?: string | null) {
  return apiAdminRequest<AIPublicSettings>("/api/admin/ai/settings", {
    method: "GET",
    token,
  });
}

export function updateAdminAISettings(input: AISettingsUpdate, token?: string | null) {
  return apiAdminRequest<AIPublicSettings>("/api/admin/ai/settings", {
    method: "PUT",
    token,
    body: JSON.stringify(input),
  });
}

export function getAIPublishGenerationSettings(token?: string | null) {
  return apiRequest<AIPublishGenerationConfig>("/api/ai/publish-generation/settings", {
    method: "GET",
    context: token ? { token } : undefined,
  });
}

export function generatePublishContent(input: AIPublishGenerationInput, token?: string | null) {
  return apiRequest<AIPublishGenerationResult>("/api/ai/publish-generation", {
    method: "POST",
    context: token ? { token } : undefined,
    body: JSON.stringify(input),
  });
}

export function debugAdminAIModeration(input: AIModerationDebugInput, token?: string | null) {
  return apiAdminRequest<AIModerationDebugResult>("/api/admin/ai/moderation/debug", {
    method: "POST",
    token,
    body: JSON.stringify(input),
  });
}

export function getAdminAILogs(
  query: Record<string, string | number | boolean | null | undefined> = {},
  token?: string | null,
) {
  return apiAdminRequest<AIGenerationLogsPayload>("/api/admin/ai/logs", {
    method: "GET",
    token,
    query,
  });
}

export async function streamAI(
  path: string,
  input: AIRequestInput,
  handlers: AIStreamHandlers,
  options: StreamOptions = {},
) {
  const headers = new Headers({
    accept: "text/event-stream",
    "content-type": "application/json",
  });
  applyAuthorizationHeader(headers, options.token ?? null);
  const response = await fetch(buildApiUrl(path), {
    method: "POST",
    credentials: "include",
    cache: "no-store",
    headers,
    body: JSON.stringify(input),
    signal: createRequestSignal(options.timeoutMs ?? 0, options.signal),
  });

  if (!response.ok) {
    const payload = await parseResponse(response);
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : "error.ai_request_failed";
    throw new ApiError(message, { status: response.status, details: payload });
  }
  if (!response.body) {
    throw new ApiError("error.ai_stream_unavailable", { status: response.status });
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let finalEvent: Extract<AIStreamEvent, { type: "final" }> | null = null;

  const processBlock = (block: string) => {
    const event = parseSSEBlock(block);
    if (!event) {
      return;
    }
    handlers.onEvent?.(event);
    switch (event.type) {
      case "progress":
        handlers.onProgress?.(event);
        break;
      case "chunk_delta":
        handlers.onChunkDelta?.(event);
        break;
      case "chunk_done":
        handlers.onChunkDone?.(event);
        break;
      case "reasoning_delta":
        handlers.onReasoningDelta?.(event);
        break;
      case "reasoning_done":
        handlers.onReasoningDone?.(event);
        break;
      case "final":
        finalEvent = event;
        handlers.onFinal?.(event);
        break;
      case "upstream_event":
        handlers.onUpstream?.(event);
        break;
      case "error":
        handlers.onError?.(event);
        throw new ApiError(event.message || event.code || "error.ai_request_failed", {
          details: event,
        });
    }
  };

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const blocks = buffer.split(/\n\n|\r\n\r\n/);
    buffer = blocks.pop() ?? "";
    for (const block of blocks) {
      processBlock(block);
    }
    if (done) {
      break;
    }
  }
  if (buffer.trim()) {
    processBlock(buffer);
  }
  if (!finalEvent) {
    throw new ApiError("error.ai_stream_incomplete", { status: 502 });
  }
  return finalEvent;
}

async function streamAIJobPath(
  path: string,
  handlers: AIJobStreamHandlers,
  options: StreamOptions = {},
) {
  const headers = new Headers({ accept: "text/event-stream" });
  applyAuthorizationHeader(headers, options.token ?? null);
  const response = await fetch(buildApiUrl(path), {
    method: "GET",
    credentials: "include",
    cache: "no-store",
    headers,
    signal: createRequestSignal(options.timeoutMs ?? 0, options.signal),
  });

  if (!response.ok) {
    const payload = await parseResponse(response);
    const message =
      payload && typeof payload === "object" && "message" in payload
        ? String((payload as { message?: unknown }).message)
        : "error.ai_request_failed";
    throw new ApiError(message, { status: response.status, details: payload });
  }
  if (!response.body) {
    throw new ApiError("error.ai_stream_unavailable", { status: response.status });
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let lastJob: AIJobPayload | null = null;

  const processBlock = (block: string) => {
    const event = parseSSEBlock(block) as AIJobStreamEvent | null;
    if (!event) return;
    handlers.onEvent?.(event);
    switch (event.type) {
      case "connected":
        handlers.onConnected?.(event);
        break;
      case "heartbeat":
        break;
      case "job":
        lastJob = event.job;
        handlers.onJob?.(event.job);
        break;
      case "progress":
        handlers.onProgress?.(event);
        break;
      case "chunk_delta":
        handlers.onChunkDelta?.(event);
        break;
      case "chunk_done":
        handlers.onChunkDone?.(event);
        break;
      case "reasoning_delta":
        handlers.onReasoningDelta?.(event);
        break;
      case "reasoning_done":
        handlers.onReasoningDone?.(event);
        break;
      case "final":
        handlers.onFinal?.(event);
        break;
      case "upstream_event":
        handlers.onUpstream?.(event);
        break;
      case "error":
        handlers.onError?.(event);
        throw new ApiError(event.message || event.code || "error.ai_request_failed", {
          details: event,
        });
    }
  };

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const blocks = buffer.split(/\n\n|\r\n\r\n/);
    buffer = blocks.pop() ?? "";
    for (const block of blocks) {
      processBlock(block);
    }
    if (done) break;
  }
  if (buffer.trim()) {
    processBlock(buffer);
  }
  return lastJob;
}

function parseSSEBlock(block: string): AIStreamEvent | null {
  const lines = block.split(/\r?\n/);
  let eventType = "";
  const dataLines: string[] = [];
  for (const line of lines) {
    if (line.startsWith("event:")) {
      eventType = line.slice(6).trim();
      continue;
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }
  if (!eventType || dataLines.length === 0) {
    return null;
  }
  try {
    const payload = JSON.parse(dataLines.join("\n")) as Record<string, unknown>;
    return { ...(payload as object), type: eventType } as AIStreamEvent;
  } catch (error) {
    throw new ApiError("error.ai_stream_decode_failed", { details: error });
  }
}
