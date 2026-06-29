"use client";
import {
  type ApiUploadOptions,
  uploadAttachment,
  uploadImage,
  uploadVideo,
  type UploadProgress
} from "@/lib/api";
import type {
  FeedPost,
  PublishPostInput,
  UploadAsset
} from "@/lib/types";
import {
  dataURLtoFile,
  generateVideoThumbnail,
  isVideoFile
} from "@/lib/utils";
import { mapWithConcurrency } from "@/lib/async-pool";
import {
  PendingUploadFile,
  ImagePaymentSettings,
  PaymentMethod,
  PublishMode,
  UploadFailure,
  UploadMode,
  Visibility,
  uploadLimits
} from "./workbench-config";
import { enforceImageCoverPolicy } from "../shared/image-access";

export function buildPostInput({
  body,
  categoryId,
  isDraft,
  mode,
  tags,
  title,
  uploadedAssets,
  visibility,
  paymentSettings,
  paymentMaxPrices,
}: {
  body: string;
  categoryId: number | null;
  isDraft: boolean;
  mode: PublishMode;
  tags: string;
  title: string;
  uploadedAssets: Record<UploadMode, UploadAsset[]>;
  visibility: Visibility;
  paymentSettings?: ImagePaymentSettings;
  paymentMaxPrices?: Record<PaymentMethod, number>;
}): PublishPostInput {
  const input: PublishPostInput = {
    title: title.trim(),
    content: body.trim(),
    type: mode === "video" ? 2 : 1,
    tags: parseTags(tags),
    is_draft: isDraft,
    visibility: mapVisibility(visibility),
  };

  if (categoryId !== null) {
    input.category_id = categoryId;
  }

  if (mode === "image") {
    input.images = enforceImageCoverPolicy(uploadedAssets.image).map((asset, index) => ({
      url: asset.url,
      watermarkTraceToken: asset.watermarkTraceToken,
      isFreePreview: asset.isFreePreview ?? true,
      isProtected: asset.isProtected ?? false,
      sortOrder: index + 1,
    }));
    const hasPaidImages = input.images.some((image) => image.isFreePreview === false);
    if (hasPaidImages && paymentSettings) {
      const price = Math.max(1, Number(paymentSettings.price) || 0);
      const maxPrice = paymentMaxPrices?.[paymentSettings.paymentMethod];
      input.paymentSettings = {
        enabled: true,
        paymentType: "single",
        paymentMethod: paymentSettings.paymentMethod,
        price: maxPrice && maxPrice > 0 ? Math.min(price, maxPrice) : price,
        freePreviewCount: input.images.filter((image) => image.isFreePreview !== false).length,
        previewDuration: 0,
        hideAll: false,
      };
    } else {
      input.paymentSettings = { enabled: false };
    }
  }

  if (mode === "video") {
    const video = uploadedAssets.video[0];
    if (video) {
      input.video = {
        url: video.url,
        coverUrl: video.coverUrl ?? null,
      };
    }
  }

  if (mode === "podcast") {
    const audio = uploadedAssets.podcast[0];
    if (audio) {
      input.attachment = {
        url: audio.url,
        filename: audio.originalname ?? "audio",
        filesize: audio.size ?? 0,
      };
    }
  }

  return input;
}

export function parseTags(value: string) {
  return Array.from(
    new Set(
      value
        .split(/[,，]/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ).slice(0, 12);
}


export function mapVisibility(value: Visibility) {
  if (value === "followers") {
    return "friends_only";
  }

  return value;
}


export async function uploadPublishFile(
  file: File,
  uploadMode: UploadMode,
  onProgress?: (progress: UploadProgress) => void,
  prebuiltThumbnailDataUrl?: string | null,
  signal?: AbortSignal,
) {
  if (uploadMode === "image") {
    return uploadImage(file, { onProgress, signal });
  }

  if (uploadMode === "video") {
    // Use the pre-generated thumbnail if available; otherwise generate on the fly.
    const thumbnailDataURL = prebuiltThumbnailDataUrl !== undefined
      ? prebuiltThumbnailDataUrl
      : await generateVideoThumbnail(file);
    const thumbnail = thumbnailDataURL
      ? dataURLtoFile(thumbnailDataURL, "thumbnail.jpg")
      : undefined;
    return uploadVideo(file, thumbnail, { onProgress, signal });
  }

  return uploadAttachment(file, { onProgress, signal });
}


export async function uploadPendingAssets(
  uploadMode: UploadMode,
  orderedAssets: UploadAsset[],
  files: PendingUploadFile[],
  onProgress: (progress: UploadProgress) => void,
  fallbackError: string,
  onAssetState?: (
    assetUrl: string,
    state: Pick<UploadAsset, "uploadError" | "uploadProgress" | "uploadStatus">,
  ) => void,
  signal?: AbortSignal,
  purpose?: ApiUploadOptions["purpose"],
) {
  const pendingByUrl = new Map(files.map((item) => [item.blobUrl, item]));
  const uploadableAssets = orderedAssets.filter((asset) => isLocalAssetUrl(asset.url));
  const progressByUrl = new Map<string, number>();
  const uploadedByUrl = new Map<string, UploadAsset>();
  const failures: Array<{ assetUrl: string; error: string; file: File }> = [];

  if (uploadMode === "image") {
    const uploadedEntries = await mapWithConcurrency(uploadableAssets, 3, async (asset) => {
      const pendingFile = pendingByUrl.get(asset.url);
      if (!pendingFile) {
        return [asset.url, null] as const;
      }
      onAssetState?.(asset.url, { uploadError: null, uploadProgress: 0, uploadStatus: "uploading" });
      try {
        const uploadedAsset = await uploadImage(pendingFile.file, {
          purpose,
          onProgress: (progress) => {
            const itemPercent = progress.percent ?? 0;
            progressByUrl.set(asset.url, itemPercent);
            onAssetState?.(asset.url, {
              uploadError: null,
              uploadProgress: itemPercent,
              uploadStatus: "uploading",
            });
            const totalPercent = Math.min(
              99,
              Math.round(
                Array.from(progressByUrl.values()).reduce((sum, value) => sum + value, 0)
                  / Math.max(uploadableAssets.length, 1),
              ),
            );
            onProgress({ ...progress, percent: totalPercent });
          },
          signal,
        });
        onAssetState?.(asset.url, {
          uploadError: null,
          uploadProgress: 100,
          uploadStatus: "succeeded" as const,
        });
        progressByUrl.set(asset.url, 100);
        return [asset.url, uploadedAsset] as const;
      } catch (error) {
        const message = error instanceof Error ? error.message : fallbackError;
        failures.push({ assetUrl: asset.url, error: message, file: pendingFile.file });
        onAssetState?.(asset.url, {
          uploadError: message,
          uploadProgress: 0,
          uploadStatus: "failed",
        });
        return [asset.url, null] as const;
      }
    });
    for (const [url, asset] of uploadedEntries) {
      if (asset) {
        const source = orderedAssets.find((item) => item.url === url);
        uploadedByUrl.set(url, {
          ...asset,
          isFreePreview: source?.isFreePreview ?? asset.isFreePreview,
          isProtected: source?.isProtected ?? asset.isProtected,
          uploadError: null,
          uploadProgress: 100,
          uploadStatus: "succeeded",
        });
      }
    }
  } else {
    const uploadedEntries = await mapWithConcurrency(uploadableAssets, 1, async (asset) => {
      const pendingFile = pendingByUrl.get(asset.url);
      if (!pendingFile) {
        return [asset.url, null] as const;
      }
      onAssetState?.(asset.url, { uploadError: null, uploadProgress: 0, uploadStatus: "uploading" });

      try {
        const uploadedAsset = await uploadPublishFile(
          pendingFile.file,
          uploadMode,
          (progress) => {
            progressByUrl.set(asset.url, progress.percent ?? 0);
            onAssetState?.(asset.url, {
              uploadError: null,
              uploadProgress: progress.percent ?? 0,
              uploadStatus: "uploading",
            });
            onProgress(progress);
          },
          pendingFile.thumbnailDataUrl,
          signal,
        );
        onAssetState?.(asset.url, {
          uploadError: null,
          uploadProgress: 100,
          uploadStatus: "succeeded" as const,
        });
        return [asset.url, uploadedAsset] as const;
      } catch (error) {
        const message = error instanceof Error ? error.message : fallbackError;
        failures.push({ assetUrl: asset.url, error: message, file: pendingFile.file });
        onAssetState?.(asset.url, {
          uploadError: message,
          uploadProgress: 0,
          uploadStatus: "failed",
        });
        return [asset.url, null] as const;
      }
    });
    for (const [url, asset] of uploadedEntries) {
      if (asset) {
        uploadedByUrl.set(url, {
          ...asset,
          uploadError: null,
          uploadProgress: 100,
          uploadStatus: "succeeded",
        });
      }
    }
  }

  const assets = orderedAssets.map((asset) => {
    if (!isLocalAssetUrl(asset.url)) {
      return asset;
    }
    const uploadedAsset = uploadedByUrl.get(asset.url);
    return uploadedAsset ?? asset;
  });
  return { assets, failures };
}


export function isLocalAssetUrl(url: string) {
  return url.startsWith("blob:") || url.startsWith("data:");
}


export function moveUploadAsset(items: UploadAsset[], assetUrl: string, direction: -1 | 1) {
  const currentIndex = items.findIndex((asset) => asset.url === assetUrl);
  if (currentIndex < 0) {
    return items;
  }

  const nextIndex = currentIndex + direction;
  if (nextIndex < 0 || nextIndex >= items.length) {
    return items;
  }

  const nextItems = [...items];
  const [item] = nextItems.splice(currentIndex, 1);
  nextItems.splice(nextIndex, 0, item);
  return nextItems;
}


export function movePendingUploadFile(items: PendingUploadFile[], assetUrl: string, direction: -1 | 1) {
  const currentIndex = items.findIndex((item) => item.blobUrl === assetUrl);
  if (currentIndex < 0) {
    return items;
  }

  const nextIndex = currentIndex + direction;
  if (nextIndex < 0 || nextIndex >= items.length) {
    return items;
  }

  const nextItems = [...items];
  const [item] = nextItems.splice(currentIndex, 1);
  nextItems.splice(nextIndex, 0, item);
  return nextItems;
}


export function revokePendingObjectUrls(pending: Record<UploadMode, PendingUploadFile[]>) {
  for (const files of Object.values(pending)) {
    for (const item of files) {
      if (item.blobUrl.startsWith("blob:")) {
        URL.revokeObjectURL(item.blobUrl);
      }
    }
  }
}


export function getDraftMode(draft: FeedPost): PublishMode {
  if (draft.type === 2) {
    return "video";
  }

  if (draft.attachment?.url) {
    return "podcast";
  }

  return "image";
}


export function getDraftAssets(draft: FeedPost): Record<UploadMode, UploadAsset[]> {
  const assets: Record<UploadMode, UploadAsset[]> = {
    image: [],
    video: [],
    podcast: [],
  };

  assets.image = enforceImageCoverPolicy((draft.images ?? []).flatMap((image) => {
    const url = typeof image === "string" ? image : image.url;
    return url
      ? [{
          url,
          originalname: fileNameFromUrl(url),
          watermarkTraceToken: typeof image === "string" ? undefined : image.watermarkTraceToken,
          isFreePreview: typeof image === "string" ? true : (image.isFreePreview ?? true),
          isProtected: typeof image === "string" ? false : (image.isProtected ?? false),
          uploadProgress: 100,
          uploadStatus: "succeeded",
        }]
      : [];
  }));

  const video =
    draft.videos?.[0] ??
    (draft.video_url
      ? {
          video_url: draft.video_url,
          cover_url: draft.cover_url,
        }
      : null);
  if (video?.video_url) {
    assets.video = [{
      url: video.video_url,
      coverUrl: video.cover_url ?? draft.cover_url ?? null,
      originalname: fileNameFromUrl(video.video_url),
    }];
  }

  if (draft.attachment?.url) {
    assets.podcast = [{
      url: draft.attachment.url,
      originalname: draft.attachment.filename ?? fileNameFromUrl(draft.attachment.url),
      size: draft.attachment.filesize ?? undefined,
    }];
  }

  return assets;
}


export function getDraftTagInput(draft: FeedPost) {
  return (draft.tags ?? [])
    .map((tag) => tag.name)
    .filter(Boolean)
    .join(", ");
}


export function mapBackendVisibility(value: string | undefined): Visibility {
  if (value === "friends_only" || value === "followers") {
    return "followers";
  }

  if (value === "private") {
    return "private";
  }

  return "public";
}


export function fileNameFromUrl(url: string) {
  try {
    const pathname = new URL(url, "http://localhost").pathname;
    const name = decodeURIComponent(pathname.split("/").filter(Boolean).pop() ?? "");
    return name || url;
  } catch {
    return url;
  }
}


export function formatDraftLabel(draft: FeedPost) {
  if (draft.type === 2) {
    return "draftType.video";
  }

  if (draft.attachment?.url) {
    return "draftType.audio";
  }

  return "draftType.article";
}


export function formatDraftDate(value: string | undefined, locale = "en") {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString(locale);
}


export function buildUploadFailure(file: File, message: string, thumbnailDataUrl?: string | null): UploadFailure {
  return {
    file,
    id: `${file.name}-${file.size}-${file.lastModified}-${crypto.randomUUID()}`,
    message,
    name: file.name,
    size: file.size,
    thumbnailDataUrl,
  };
}


export function isValidUploadFile(file: File, uploadMode: UploadMode) {
  const withinSize = file.size <= uploadLimits[uploadMode];
  if (uploadMode === "video") {
    return isVideoFile(file) && withinSize;
  }
  const typePrefix = uploadMode === "image" ? "image/" : "audio/";
  return file.type.startsWith(typePrefix) && withinSize;
}


export function formatFileSize(bytes: number | undefined) {
  if (!bytes || bytes <= 0) {
    return "0 KB";
  }

  if (bytes >= 1024 * 1024) {
    return `${(bytes / 1024 / 1024).toFixed(bytes >= 10 * 1024 * 1024 ? 0 : 1)} MB`;
  }

  return `${Math.ceil(bytes / 1024)} KB`;
}


export function formatUploadProgressLabel(progressDetail: UploadProgress | null, fallback: number | null) {
  const percent = progressDetail?.percent ?? fallback ?? 0;
  const stage = progressDetail?.stage;
  const key =
    stage === "verifying"
      ? "verifying"
      : stage === "chunking"
        ? "uploadingChunks"
        : stage === "merging"
          ? "merging"
          : stage === "thumbnail"
            ? "thumbnail"
            : stage === "publishing"
              ? "publishing"
              : stage === "processing"
                ? "processing"
                : "uploading";

  return { key, percent };
}


export function formatUploadChunkLabel(progressDetail: UploadProgress) {
  if (!progressDetail.totalChunks) {
    return progressDetail.message ?? "";
  }

  const current = progressDetail.chunkNumber ?? progressDetail.uploadedChunks ?? 0;
  const chunkPercent = progressDetail.chunkPercent;
  const chunkText = chunkPercent === undefined ? "" : ` · ${chunkPercent}%`;
  return `${current}/${progressDetail.totalChunks}${chunkText}`;
}
