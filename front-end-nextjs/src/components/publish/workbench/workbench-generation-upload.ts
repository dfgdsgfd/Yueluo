import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { toast } from "sonner";
import type { UploadAsset } from "@/lib/types";
import { enforceImageCoverPolicy } from "../shared/image-access";
import { isLocalAssetUrl, uploadPendingAssets } from "./post-builders";
import type {
  PendingUploadFile,
  UploadFailure,
  UploadMode,
  UploadProgressDetailState,
  UploadProgressState,
} from "./workbench-config";

type UploadAssetState = Record<UploadMode, UploadAsset[]>;
type PendingFileState = Record<UploadMode, PendingUploadFile[]>;
type UploadFailureState = Record<UploadMode, UploadFailure[]>;

export async function prepareWorkbenchGenerationImages({
  messages,
  pendingFiles,
  setPendingFiles,
  setUploadedAssets,
  setUploadFailures,
  setUploadingMode,
  setUploadProgress,
  setUploadProgressDetails,
  uploadAbortControllerRef,
  uploadedAssets,
}: {
  messages: {
    someUploadsFailed: (count: number) => string;
    uploadFailed: string;
  };
  pendingFiles: PendingFileState;
  setPendingFiles: Dispatch<SetStateAction<PendingFileState>>;
  setUploadedAssets: Dispatch<SetStateAction<UploadAssetState>>;
  setUploadFailures: Dispatch<SetStateAction<UploadFailureState>>;
  setUploadingMode: Dispatch<SetStateAction<UploadMode | null>>;
  setUploadProgress: Dispatch<SetStateAction<UploadProgressState>>;
  setUploadProgressDetails: Dispatch<SetStateAction<UploadProgressDetailState>>;
  uploadAbortControllerRef: MutableRefObject<AbortController | null>;
  uploadedAssets: UploadAssetState;
}) {
  if (pendingFiles.image.length === 0) {
    return uploadedAssets.image;
  }

  const uploadController = new AbortController();
  uploadAbortControllerRef.current = uploadController;
  setUploadingMode("image");
  setUploadProgress((current) => ({ ...current, image: 0 }));
  setUploadProgressDetails((current) => ({
    ...current,
    image: {
      fileName: pendingFiles.image[0]?.file.name,
      loaded: 0,
      percent: 0,
      stage: "preparing",
      total: pendingFiles.image.reduce((sum, item) => sum + item.file.size, 0),
    },
  }));

  try {
    const uploadResult = await uploadPendingAssets(
      "image",
      uploadedAssets.image,
      pendingFiles.image,
      (progress) => {
        setUploadProgress((current) => ({
          ...current,
          image: progress.percent ?? current.image ?? 0,
        }));
        setUploadProgressDetails((current) => ({ ...current, image: progress }));
      },
      messages.uploadFailed,
      (assetUrl, state) => {
        setUploadedAssets((current) => ({
          ...current,
          image: current.image.map((asset) =>
            asset.url === assetUrl ? { ...asset, ...state } : asset,
          ),
        }));
      },
      uploadController.signal,
      "ai_analysis",
    );
    const nextAssets = enforceImageCoverPolicy(uploadResult.assets);
    setUploadedAssets((current) => ({ ...current, image: nextAssets }));
    setPendingFiles((current) => ({
      ...current,
      image: current.image.filter((item) =>
        uploadResult.failures.some((failure) => failure.assetUrl === item.blobUrl),
      ),
    }));
    setUploadFailures((current) => ({
      ...current,
      image: uploadResult.failures.map((failure) => ({
        file: failure.file,
        id: failure.assetUrl,
        message: failure.error,
        name: failure.file.name,
        size: failure.file.size,
      })),
    }));
    if (uploadResult.failures.length > 0) {
      toast.error(messages.someUploadsFailed(uploadResult.failures.length));
    }
    return nextAssets.filter((asset) => !isLocalAssetUrl(asset.url));
  } finally {
    if (uploadAbortControllerRef.current === uploadController) {
      uploadAbortControllerRef.current = null;
    }
    setUploadingMode(null);
    setUploadProgress((current) => ({ ...current, image: null }));
    setUploadProgressDetails((current) => ({ ...current, image: null }));
  }
}
