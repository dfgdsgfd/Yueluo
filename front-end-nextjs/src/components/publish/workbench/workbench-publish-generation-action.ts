import type { Dispatch, SetStateAction } from "react";
import type { useTranslations } from "next-intl";
import type { UploadAsset } from "@/lib/types";
import type {
  PublishGenerationApplyResult,
  PublishGenerationRunInput,
  PublishGenerationState,
} from "../shared/ai-publish-generation";
import { countPublishGenerationSelectableImages } from "../shared/publish-generation-image-count";
import {
  createPublishGenerationAction,
  splitPublishGenerationTags,
} from "../shared/publish-generation-action";
import type { UploadMode } from "./workbench-config";

export function createWorkbenchPublishGenerationAction({
  body,
  imageCount,
  isImageMode,
  publishGeneration,
  runPublishGenerationStream,
  setBody,
  setTitle,
  tags,
  title,
  t,
  uploadedAssets,
  uploadingMode,
}: {
  body: string;
  imageCount: number;
  isImageMode: boolean;
  publishGeneration: PublishGenerationState;
  runPublishGenerationStream: (input: PublishGenerationRunInput) => Promise<PublishGenerationApplyResult>;
  setBody: Dispatch<SetStateAction<string>>;
  setTitle: Dispatch<SetStateAction<string>>;
  tags: string;
  title: string;
  t: ReturnType<typeof useTranslations>;
  uploadedAssets: Record<UploadMode, UploadAsset[]>;
  uploadingMode: UploadMode | null;
}) {
  const generationImageCount = countPublishGenerationSelectableImages(
    uploadedAssets.image.length,
    publishGeneration.config?.maxImages,
  );
  return createPublishGenerationAction({
    body,
    enabledByAdmin: publishGeneration.config?.enabled !== false,
    imageCount: Math.max(imageCount, generationImageCount),
    images: uploadedAssets.image,
    isImageMode,
    isUploading: Boolean(uploadingMode),
    onBody: setBody,
    onTitle: setTitle,
    run: runPublishGenerationStream,
    state: publishGeneration,
    tags: splitPublishGenerationTags(tags),
    title,
    t,
  });
}
