import type { HiddenWatermarkResult } from "../types";
import {
  ApiError,
  applyAuthorizationHeader,
  buildApiUrl,
  getRequestAccessToken,
} from "./core";

export type WatermarkExtractionStage =
  | "uploading"
  | "queued"
  | "decoding"
  | "recovering"
  | "extracting"
  | "verifying"
  | "complete";

export type WatermarkExtractionProgress = {
  stage: WatermarkExtractionStage;
  percent: number;
  completed?: number;
  total?: number;
  elapsedMs: number;
  heartbeat?: boolean;
  source?: "engine" | "gateway";
};

type WatermarkStreamEvent = WatermarkExtractionProgress & {
  type: "progress" | "heartbeat" | "result" | "error";
  result?: HiddenWatermarkResult;
  error?: string;
  retryable?: boolean;
  status?: number;
};

export type ExtractHiddenWatermarkOptions = {
  path?: string;
  token?: string | null;
  timeoutMs?: number;
  onProgress?: (progress: WatermarkExtractionProgress) => void;
};

export function extractHiddenWatermark(
  file: File,
  referenceFile?: File | null,
  options: ExtractHiddenWatermarkOptions = {},
) {
  const {
    path = "/api/image-watermark/extract",
    token = getRequestAccessToken(),
    timeoutMs = 60_000,
    onProgress,
  } = options;
  const body = new FormData();
  body.append("file", file);
  if (referenceFile) {
    body.append("reference_file", referenceFile);
  }

  return new Promise<HiddenWatermarkResult>((resolve, reject) => {
    const request = new XMLHttpRequest();
    const headers = new Headers({ Accept: "application/x-ndjson" });
    applyAuthorizationHeader(headers, token);
    request.open("POST", buildApiUrl(path), true);
    request.withCredentials = true;
    request.timeout = timeoutMs;
    headers.forEach((value, name) => request.setRequestHeader(name, value));

    const startedAt = Date.now();
    let consumed = 0;
    let pending = "";
    let settled = false;

    const finish = (callback: () => void) => {
      if (settled) return;
      settled = true;
      callback();
    };

    const processLine = (line: string) => {
      const trimmed = line.trim();
      if (!trimmed) return;
      let event: WatermarkStreamEvent;
      try {
        event = JSON.parse(trimmed) as WatermarkStreamEvent;
      } catch {
        return;
      }
      if (event.type === "progress" || event.type === "heartbeat") {
        onProgress?.({
          stage: event.stage,
          percent: clampPercent(event.percent),
          completed: event.completed,
          total: event.total,
          elapsedMs: Number(event.elapsedMs ?? Date.now() - startedAt),
          heartbeat: event.type === "heartbeat" || event.heartbeat,
          source: event.source,
        });
        return;
      }
      if (event.type === "result" && event.result) {
        finish(() => resolve(event.result as HiddenWatermarkResult));
        return;
      }
      if (event.type === "error") {
        finish(() =>
          reject(
            new ApiError(event.error || "error.watermark_extract_failed", {
              status: Number(event.status || 500),
              details: { retryable: Boolean(event.retryable) },
            }),
          ),
        );
      }
    };

    const consumeResponse = (final = false) => {
      const chunk = request.responseText.slice(consumed);
      consumed = request.responseText.length;
      pending += chunk;
      const lines = pending.split(/\r?\n/);
      pending = lines.pop() ?? "";
      for (const line of lines) processLine(line);
      if (final && pending.trim()) {
        processLine(pending);
        pending = "";
      }
    };

    request.upload.onprogress = (event) => {
      const percent = event.lengthComputable
        ? Math.max(1, Math.min(8, Math.round((event.loaded / event.total) * 8)))
        : 2;
      onProgress?.({
        stage: "uploading",
        percent,
        completed: event.loaded,
        total: event.lengthComputable ? event.total : undefined,
        elapsedMs: Date.now() - startedAt,
      });
    };
    request.onprogress = () => consumeResponse();
    request.onload = () => {
      consumeResponse(true);
      if (settled) return;
      const contentType = request.getResponseHeader("content-type") ?? "";
      if (contentType.includes("application/json")) {
        try {
          const payload = JSON.parse(request.responseText) as {
            data?: HiddenWatermarkResult;
            message?: string;
          };
          if (request.status >= 200 && request.status < 300 && payload.data) {
            finish(() => resolve(payload.data as HiddenWatermarkResult));
            return;
          }
          finish(() =>
            reject(
              new ApiError(payload.message || `API request failed with status ${request.status}.`, {
                status: request.status,
              }),
            ),
          );
          return;
        } catch {
          // Fall through to the generic transport error.
        }
      }
      finish(() =>
        reject(
          new ApiError("error.watermark_stream_incomplete", {
            status: request.status || 502,
          }),
        ),
      );
    };
    request.onerror = () =>
      finish(() => reject(new ApiError("error.watermark_stream_disconnected", { status: 502 })));
    request.ontimeout = () =>
      finish(() =>
        reject(new ApiError("error.hidden_watermark_remote_timeout", { status: 504 })),
      );
    request.onabort = () =>
      finish(() => reject(new DOMException("The operation was aborted.", "AbortError")));
    request.send(body);
  });
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Number.isFinite(value) ? Math.round(value) : 0));
}
