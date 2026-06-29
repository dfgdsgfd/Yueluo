"use client";
import {
  type MobilePublishDraft
} from "../mobile-drafts";
import {
  type UploadProgress
} from "@/lib/api";
import type {
  AuthUser
} from "@/lib/types";
import {
  MobileMediaAsset,
  MobileUploadProgressState
} from "./mobile-publish-config";
import type { ManagedImageItem } from "../shared/image-manager";

export function getDisplayName(user: AuthUser | null) {
  return user?.nickname?.trim() || user?.user_id?.trim() || user?.xise_id?.trim() || "Y";
}

export function toManagedImageItem(asset: MobileMediaAsset): ManagedImageItem {
  return {
    id: asset.id,
    previewUrl: asset.previewUrl,
    name: asset.name,
    size: asset.file?.size ?? asset.remoteAsset?.size ?? 0,
    isFreePreview: asset.isFreePreview,
    isProtected: asset.isProtected,
    status: asset.uploadStatus ?? "queued",
    progress: asset.uploadProgress,
    error: asset.uploadError,
  };
}


export function assetToDraftMedia(asset: MobileMediaAsset) {
  if (!asset.file) {
    return null;
  }
  return {
    file: asset.file,
    id: asset.id,
    isFreePreview: asset.isFreePreview,
    isProtected: asset.isProtected,
    kind: asset.kind,
    name: asset.name,
    previewDataUrl: asset.previewUrl.startsWith("data:") ? asset.previewUrl : undefined,
  };
}


export function draftMediaToAsset(media: MobilePublishDraft["mediaAssets"][number]): MobileMediaAsset {
  return {
    file: media.file,
    id: `${media.id}-${crypto.randomUUID()}`,
    isFreePreview: media.isFreePreview ?? true,
    isProtected: media.isProtected ?? false,
    kind: media.kind,
    name: media.name,
    previewUrl: media.previewDataUrl || (media.kind === "image" ? URL.createObjectURL(media.file) : ""),
    uploadProgress: 0,
    uploadStatus: "queued",
  };
}


export function fileToDraftAttachment(file: File) {
  return {
    file,
    name: file.name,
    size: file.size,
    type: file.type,
  };
}


export function formatFileSize(size: number) {
  if (size < 1024) {
    return `${size} B`;
  }

  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }

  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}


export function mobileUploadProgressKey(progress: NonNullable<MobileUploadProgressState>) {
  if (progress.phase === "publishing") {
    return "publishing";
  }

  const detail = progress.detail;
  if (!detail) {
    return "preparing";
  }

  const stage = detail.stage;
  return stage === "verifying"
      ? "verifying"
      : stage === "chunking"
        ? "uploading"
        : stage === "merging"
          ? "merging"
          : stage === "thumbnail"
            ? "thumbnail"
            : "working";
}


export function formatMobileUploadChunkLabel(detail: UploadProgress) {
  if (!detail.totalChunks) {
    return detail.message ?? "";
  }

  const current = detail.chunkNumber ?? detail.uploadedChunks ?? 0;
  const chunkPercent = detail.chunkPercent === undefined ? "" : ` · ${detail.chunkPercent}%`;
  return `${current}/${detail.totalChunks}${chunkPercent}`;
}


export function reorderMediaAssets(items: MobileMediaAsset[], activeId: string, targetId: string) {
  const activeIndex = items.findIndex((item) => item.id === activeId);
  const targetIndex = items.findIndex((item) => item.id === targetId);

  if (activeIndex < 0 || targetIndex < 0 || activeIndex === targetIndex) {
    return items;
  }

  const nextItems = [...items];
  const [activeItem] = nextItems.splice(activeIndex, 1);
  nextItems.splice(targetIndex, 0, activeItem);
  return nextItems;
}
