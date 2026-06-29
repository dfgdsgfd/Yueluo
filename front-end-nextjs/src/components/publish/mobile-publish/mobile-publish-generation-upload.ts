import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type { useTranslations } from "next-intl";
import { toast } from "sonner";
import { uploadImage, type UploadProgress } from "@/lib/api";
import { mapWithConcurrency } from "@/lib/async-pool";
import type { UploadAsset } from "@/lib/types";
import { enforceImageCoverPolicy } from "../shared/image-access";
import type { MobileMediaAsset, MobileUploadProgressState } from "./mobile-publish-config";

export type MobileGenerationUploadResult = {
  assets: MobileMediaAsset[];
  failedCount: number;
  remoteImages: UploadAsset[];
};

export function mobileGenerationRemoteImages(assets: MobileMediaAsset[]) {
  return assets.flatMap((asset) =>
    asset.kind === "image" && asset.remoteAsset ? [asset.remoteAsset] : [],
  );
}

export async function uploadMobileImagesForGeneration({
  mediaAssets,
  onAssetProgress,
  onProgress,
  signal,
}: {
  mediaAssets: MobileMediaAsset[];
  onAssetProgress: (id: string, patch: Partial<MobileMediaAsset>) => void;
  onProgress: (progress: UploadProgress, percent: number) => void;
  signal?: AbortSignal;
}): Promise<MobileGenerationUploadResult> {
  const imageAssets = mediaAssets.filter((asset) => asset.kind === "image");
  const localImages = imageAssets.filter(
    (asset): asset is MobileMediaAsset & { file: File } => asset.file instanceof File && !asset.remoteAsset,
  );
  const progressById = new Map<string, number>();
  const uploadedById = new Map<string, UploadAsset>();
  let failedCount = 0;

  await mapWithConcurrency(localImages, 3, async (asset) => {
    onAssetProgress(asset.id, { uploadError: null, uploadProgress: 0, uploadStatus: "uploading" });
    try {
      const uploaded = await uploadImage(asset.file, {
        purpose: "ai_analysis",
        onProgress: (progress) => {
          progressById.set(asset.id, progress.percent ?? 0);
          const percent = Math.min(
            96,
            Math.round(Array.from(progressById.values()).reduce((sum, value) => sum + value, 0) / Math.max(localImages.length, 1)),
          );
          onAssetProgress(asset.id, {
            uploadError: null,
            uploadProgress: progress.percent ?? asset.uploadProgress ?? 0,
            uploadStatus: "uploading",
          });
          onProgress(progress, percent);
        },
        signal,
      });
      progressById.set(asset.id, 100);
      uploadedById.set(asset.id, uploaded);
      onAssetProgress(asset.id, { uploadError: null, uploadProgress: 100, uploadStatus: "succeeded" });
    } catch (error) {
      failedCount += 1;
      onAssetProgress(asset.id, {
        uploadError: error instanceof Error ? error.message : "error.upload_failed",
        uploadProgress: 0,
        uploadStatus: "failed",
      });
    }
  });

  const assets = enforceImageCoverPolicy(mediaAssets.map((asset) => {
    const uploaded = uploadedById.get(asset.id);
    return uploaded ? { ...asset, remoteAsset: uploaded } : asset;
  }));
  return {
    assets,
    failedCount,
    remoteImages: mobileGenerationRemoteImages(assets),
  };
}

type MobileGenerationPrepareOptions = {
  mediaAssets: MobileMediaAsset[];
  setMediaAssets: Dispatch<SetStateAction<MobileMediaAsset[]>>;
  setUploadProgress: Dispatch<SetStateAction<MobileUploadProgressState>>;
  t: ReturnType<typeof useTranslations>;
  uploadAbortControllerRef: MutableRefObject<AbortController | null>;
};

export async function prepareMobileGenerationImages({
  mediaAssets,
  setMediaAssets,
  setUploadProgress,
  t,
  uploadAbortControllerRef,
}: MobileGenerationPrepareOptions) {
  const uploadController = new AbortController();
  uploadAbortControllerRef.current = uploadController;
  setUploadProgress({ detail: null, percent: 0, phase: "uploading" });

  try {
    const result = await uploadMobileImagesForGeneration({
      mediaAssets,
      onAssetProgress: (id, patch) => {
        setMediaAssets((current) =>
          current.map((asset) => (asset.id === id ? { ...asset, ...patch } : asset)),
        );
      },
      onProgress: (detail, percent) => {
        setUploadProgress({ detail, percent, phase: "uploading" });
      },
      signal: uploadController.signal,
    });
    setMediaAssets(result.assets);
    if (result.failedCount > 0) {
      toast.error(t("publish.imageManager.someUploadsFailed", { count: result.failedCount }));
    }
    return result.remoteImages;
  } finally {
    if (uploadAbortControllerRef.current === uploadController) {
      uploadAbortControllerRef.current = null;
    }
    setUploadProgress(null);
  }
}
