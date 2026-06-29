import { emitPointsAwardFromResponsePayload } from "../../points-award-events";
import type { UploadAsset } from "../../types/content";
import type { ApiUploadOptions,UploadChunkConfigPayload,UploadChunkPayload,UploadChunkVerifyPayload,UploadProgress } from "./contracts";
import { ApiError,ApiUnauthorizedError,DEFAULT_UPLOAD_TIMEOUT_MS } from "./contracts";
import { buildApiUrl,createAbortError,createRequestSignal,normalizeAuthToken } from "./http";
import { apiGet,apiPost,apiRequest,getUnauthorizedRetryToken,numberFromUnknown,redirectToLogin,resolveMutationTokenOrRedirect } from "./request";
import { extractEnvelope } from "./response";
import { clearSession,getRequestAuthorizationToken } from "./session";

export function parseXhrPayload(xhr: XMLHttpRequest) {
  const contentType = xhr.getResponseHeader("content-type") ?? "";
  const text = xhr.responseText;

  if (contentType.includes("application/json") && text) {
    return JSON.parse(text) as unknown;
  }

  return text ? { message: text } : null;
}

export function apiUploadWithProgress<T>(
  path: string,
  formData: FormData,
  options: ApiUploadOptions,
  retryOnUnauthorized = true,
): Promise<T> {
  const {
    auth = true,
    context,
    onProgress,
    query,
    signal,
    timeoutMs = DEFAULT_UPLOAD_TIMEOUT_MS,
  } = options;

  return new Promise(async (resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", buildApiUrl(path, query));
    xhr.withCredentials = true;
    xhr.timeout = timeoutMs;

    const requestSignal = createRequestSignal(
      0,
      signal,
      context?.signal,
    );
    const abortUpload = () => xhr.abort();
    const cleanup = () => {
      requestSignal?.removeEventListener("abort", abortUpload);
    };
    requestSignal?.addEventListener("abort", abortUpload, { once: true });
    if (requestSignal?.aborted) {
      cleanup();
      reject(createAbortError());
      return;
    }

    let token: string | null;
    try {
      token = await resolveMutationTokenOrRedirect(
        auth,
        getRequestAuthorizationToken(context, auth),
        "POST",
        retryOnUnauthorized,
      );
    } catch (error) {
      cleanup();
      reject(error);
      return;
    }
    const normalizedToken = normalizeAuthToken(token);
    if (normalizedToken) {
      xhr.setRequestHeader("authorization", `Bearer ${normalizedToken}`);
    }

    let lastUploadLoaded = 0;
    let lastUploadTotal: number | undefined;

    xhr.upload.onprogress = (event) => {
      const total = event.lengthComputable ? event.total : undefined;
      const percent = total ? Math.min(100, Math.round((event.loaded / total) * 100)) : undefined;
      lastUploadLoaded = event.loaded;
      lastUploadTotal = total;
      onProgress?.({ loaded: event.loaded, percent, stage: "uploading", total });
    };

    xhr.upload.onload = () => {
      onProgress?.({
        loaded: lastUploadTotal ?? lastUploadLoaded,
        message: "Processing upload",
        percent: 99,
        stage: "processing",
        total: lastUploadTotal,
      });
    };

    xhr.onerror = () => {
      cleanup();
      reject(new ApiError("Upload failed. Please check your network and try again."));
    };
    xhr.onabort = () => {
      cleanup();
      reject(createAbortError());
    };
    xhr.ontimeout = () => {
      cleanup();
      reject(new ApiError("Upload timed out. Please try again."));
    };

    xhr.onload = async () => {
      cleanup();
      if (xhr.status === 401) {
        if (auth && typeof window !== "undefined") {
          const nextToken = await getUnauthorizedRetryToken(normalizedToken, retryOnUnauthorized);
          if (nextToken) {
            try {
              resolve(
                await apiUploadWithProgress<T>(
                  path,
                  formData,
                  { ...options, context: { ...context, token: nextToken } },
                  false,
                ),
              );
            } catch (error) {
              reject(error);
            }
            return;
          }
        }

        if (auth) {
          clearSession();
          redirectToLogin();
        }
        reject(new ApiUnauthorizedError());
        return;
      }

      let payload: unknown;
      try {
        payload = parseXhrPayload(xhr);
      } catch (error) {
        reject(new ApiError("Upload response could not be parsed.", { status: xhr.status, details: error }));
        return;
      }

      if (xhr.status < 200 || xhr.status >= 300) {
        const message =
          payload && typeof payload === "object" && "message" in payload
            ? String((payload as { message?: unknown }).message)
            : `Upload failed with status ${xhr.status}.`;
        reject(new ApiError(message, { status: xhr.status, details: payload }));
        return;
      }

      try {
        emitPointsAwardFromResponsePayload(payload);
        resolve(extractEnvelope<T>(payload, xhr.status));
      } catch (error) {
        reject(error);
      }
    };

    xhr.send(formData);
  });
}

export function apiUpload<T>(path: string, formData: FormData, options: ApiUploadOptions = {}) {
  if (options.onProgress && typeof window !== "undefined" && typeof XMLHttpRequest !== "undefined") {
    return apiUploadWithProgress<T>(path, formData, options);
  }

  return apiRequest<T>(path, {
    method: "POST",
    body: formData,
    auth: options.auth,
    context: options.context,
    query: options.query,
    signal: options.signal,
    timeoutMs: options.timeoutMs ?? DEFAULT_UPLOAD_TIMEOUT_MS,
  });
}

export function emitUploadProgress(
  options: ApiUploadOptions | undefined,
  progress: UploadProgress,
) {
  options?.onProgress?.(progress);
}

export function uploadProgressWithStage(
  progress: UploadProgress,
  stage: NonNullable<UploadProgress["stage"]>,
  message?: string,
) {
  return {
    ...progress,
    message,
    stage,
  };
}

export async function getUploadChunkConfig(options?: ApiUploadOptions) {
  try {
    const config = await apiGet<UploadChunkConfigPayload>(
      "/api/upload/chunk/config",
      undefined,
      { auth: options?.auth, context: options?.context },
    );

    return {
      chunkSize: Math.max(256 * 1024, numberFromUnknown(config.chunkSize, 3 * 1024 * 1024)),
      imageChunkThreshold: Math.max(1, numberFromUnknown(config.imageChunkThreshold, 2 * 1024 * 1024)),
      imageMaxSize: Math.max(1, numberFromUnknown(config.imageMaxSize, 100 * 1024 * 1024)),
      maxFileSize: Math.max(1, numberFromUnknown(config.maxFileSize, 100 * 1024 * 1024)),
    };
  } catch {
    return {
      chunkSize: 3 * 1024 * 1024,
      imageChunkThreshold: 2 * 1024 * 1024,
      imageMaxSize: 100 * 1024 * 1024,
      maxFileSize: 100 * 1024 * 1024,
    };
  }
}

export async function uploadFileInChunks(
  file: File,
  mergePath: "/api/upload/chunk/merge" | "/api/upload/chunk/merge/image" | "/api/upload/chunk/merge/apk",
  options?: ApiUploadOptions,
) {
  const config = await getUploadChunkConfig(options);
  const chunkSize = Math.max(1, config.chunkSize);
  const totalSize = Math.max(1, file.size);
  const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));
  const identifier = await buildUploadIdentifier(file);
  let completedChunks = 0;

  const emitChunkProgress = (
    stage: NonNullable<UploadProgress["stage"]>,
    chunkNumber: number,
    loaded: number,
    message?: string,
  ) => {
    const chunkTotal = Math.max(1, Math.min(chunkSize, file.size - (chunkNumber - 1) * chunkSize));
    const totalLoaded = Math.min(file.size, completedChunks * chunkSize + loaded);
    const percent = Math.min(99, Math.round((totalLoaded / totalSize) * 96));
    const uploadedChunks = completedChunks + (loaded >= chunkTotal ? 1 : 0);
    emitUploadProgress(options, {
      chunkNumber,
      chunkPercent: Math.min(100, Math.round((loaded / chunkTotal) * 100)),
      fileName: file.name,
      loaded: totalLoaded,
      message,
      percent,
      stage,
      total: file.size,
      totalChunks,
      uploadedChunks: Math.min(totalChunks, uploadedChunks),
    });
  };

  emitUploadProgress(options, {
    fileName: file.name,
    loaded: 0,
    message: "Preparing chunks",
    percent: 0,
    stage: "preparing",
    total: file.size,
    totalChunks,
    uploadedChunks: 0,
  });

  for (let index = 0; index < totalChunks; index += 1) {
    const chunkNumber = index + 1;
    const start = index * chunkSize;
    const chunk = file.slice(start, Math.min(file.size, start + chunkSize));
    emitChunkProgress("verifying", chunkNumber, 0, `Checking chunk ${chunkNumber}/${totalChunks}`);
    const verify = await apiGet<UploadChunkVerifyPayload>(
      "/api/upload/chunk/verify",
      {
        chunkNumber,
        identifier,
      },
      { auth: options?.auth, context: options?.context },
    );

    if (verify.exists && verify.valid) {
      emitChunkProgress("chunking", chunkNumber, chunk.size, `Chunk ${chunkNumber}/${totalChunks} already uploaded`);
      completedChunks += 1;
      continue;
    }

    const formData = new FormData();
    formData.set("file", chunk, `${file.name}.part${chunkNumber}`);
    formData.set("identifier", identifier);
    formData.set("chunkNumber", String(chunkNumber));
    formData.set("totalChunks", String(totalChunks));
    formData.set("filename", file.name);
    formData.set("totalSize", String(file.size));

    const chunkResult = await apiUpload<UploadChunkPayload>("/api/upload/chunk", formData, {
      auth: options?.auth,
      context: options?.context,
      onProgress: (progress) => {
        emitChunkProgress("chunking", chunkNumber, progress.loaded, `Uploading chunk ${chunkNumber}/${totalChunks}`);
      },
    });

    emitChunkProgress("chunking", chunkNumber, chunk.size, `Chunk ${chunkNumber}/${totalChunks} uploaded`);
    completedChunks = Math.max(
      completedChunks + 1,
      numberFromUnknown(chunkResult.uploaded, completedChunks + 1),
    );
  }

  emitUploadProgress(options, {
    fileName: file.name,
    loaded: file.size,
    message: "Merging chunks",
    percent: 98,
    stage: "merging",
    total: file.size,
    totalChunks,
    uploadedChunks: totalChunks,
  });

  const asset = await apiPost<UploadAsset>(
    mergePath,
    {
      filename: file.name,
      identifier,
      purpose: mergePath === "/api/upload/chunk/merge/image" ? options?.purpose : undefined,
      totalChunks,
    },
    { auth: options?.auth, context: options?.context },
  );

  emitUploadProgress(options, {
    fileName: file.name,
    loaded: file.size,
    message: "Upload complete",
    percent: 100,
    stage: "complete",
    total: file.size,
    totalChunks,
    uploadedChunks: totalChunks,
  });

  return asset;
}

export async function shouldUseChunkUpload(file: File, kind: "image" | "video" | "apk", options?: ApiUploadOptions) {
  if (typeof window === "undefined" || typeof crypto === "undefined" || !crypto.subtle) {
    return false;
  }

  const config = await getUploadChunkConfig(options);
  if (kind === "image") {
    return file.size >= config.imageChunkThreshold;
  }

  return file.size > config.chunkSize;
}

export async function buildUploadIdentifier(file: File) {
  const fingerprintSize = Math.min(file.size, 2 * 1024 * 1024);
  const head = file.slice(0, fingerprintSize);
  const tail = file.size > fingerprintSize
    ? file.slice(Math.max(0, file.size - fingerprintSize))
    : new Blob([]);
  const joined = new Blob([
    file.name,
    "\n",
    String(file.size),
    "\n",
    String(file.lastModified),
    "\n",
    head,
    "\n",
    tail,
  ]);
  const hash = await blobSHA256(joined);
  return hash.slice(0, 48);
}

export async function blobSHA256(blob: Blob) {
  const buffer = await blob.arrayBuffer();
  const digest = await crypto.subtle.digest("SHA-256", buffer);
  return hexFromArrayBuffer(digest);
}

export function hexFromArrayBuffer(buffer: ArrayBuffer) {
  return Array.from(new Uint8Array(buffer))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}
