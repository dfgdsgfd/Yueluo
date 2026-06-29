import type { Dispatch, SetStateAction } from "react";
import type { useTranslations } from "next-intl";
import type {
  PublishGenerationApplyResult,
  PublishGenerationRunInput,
  PublishGenerationState,
} from "../shared/ai-publish-generation";
import { countPublishGenerationSelectableImages } from "../shared/publish-generation-image-count";
import { createPublishGenerationAction } from "../shared/publish-generation-action";
import type { MobileMediaAsset } from "./mobile-publish-config";
import { parseMobileTopics } from "./mobile-publish-config";
import { mobileGenerationRemoteImages } from "./mobile-publish-generation-upload";

export function createMobilePublishGenerationAction({
  body,
  imageAssets,
  mediaKind,
  publishGeneration,
  runPublishGenerationStream,
  selectedImageCount,
  setBody,
  setTitle,
  title,
  topic,
  t,
  uploadProgress,
}: {
  body: string;
  imageAssets: MobileMediaAsset[];
  mediaKind: MobileMediaAsset["kind"] | null;
  publishGeneration: PublishGenerationState;
  runPublishGenerationStream: (input: PublishGenerationRunInput) => Promise<PublishGenerationApplyResult>;
  selectedImageCount: (images: ReturnType<typeof mobileGenerationRemoteImages>) => number;
  setBody: Dispatch<SetStateAction<string>>;
  setTitle: Dispatch<SetStateAction<string>>;
  title: string;
  topic: string;
  t: ReturnType<typeof useTranslations>;
  uploadProgress: unknown;
}) {
  const generationImages = mobileGenerationRemoteImages(imageAssets);
  const generationImageCount = countPublishGenerationSelectableImages(
    imageAssets.length,
    publishGeneration.config?.maxImages,
  );
  return createPublishGenerationAction({
    body,
    enabledByAdmin: publishGeneration.config?.enabled !== false,
    imageCount: Math.max(selectedImageCount(generationImages), generationImageCount),
    images: generationImages,
    isImageMode: mediaKind === "image",
    isUploading: Boolean(uploadProgress),
    onBody: setBody,
    onTitle: setTitle,
    run: runPublishGenerationStream,
    state: publishGeneration,
    tags: parseMobileTopics(topic),
    title,
    t,
  });
}
