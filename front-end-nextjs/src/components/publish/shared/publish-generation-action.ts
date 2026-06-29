import type { useTranslations } from "next-intl";
import { toast } from "sonner";
import type { UploadAsset } from "@/lib/types";
import {
  publishGenerationHasRemoteImages,
  type PublishGenerationApplyResult,
  type PublishGenerationRunInput,
  type PublishGenerationState,
} from "./ai-publish-generation";
import {
  publishGenerationRunInput,
  type PublishGenerationRunOptions,
} from "./publish-generation-run-input";
export type { PublishGenerationRunOptions } from "./publish-generation-run-input";

type PublishGenerationActionOptions = {
  body: string;
  enabledByAdmin: boolean;
  imageCount?: number;
  images: UploadAsset[];
  isImageMode: boolean;
  isUploading: boolean;
  onBody: (body: string) => void;
  onTitle: (title: string) => void;
  prepareImages?: () => Promise<UploadAsset[]>;
  run: (input: PublishGenerationRunInput) => Promise<PublishGenerationApplyResult>;
  state: PublishGenerationState;
  tags: string[];
  title: string;
  t: ReturnType<typeof useTranslations>;
};

type PreparePublishGenerationImages = () => Promise<UploadAsset[]>;

export function splitPublishGenerationTags(value: string) {
  return value.split(/[,\s#]+/).map((item) => item.trim()).filter(Boolean);
}

export function createPublishGenerationAction({
  body,
  enabledByAdmin,
  imageCount,
  images,
  isImageMode,
  isUploading,
  onBody,
  onTitle,
  prepareImages,
  run,
  state,
  tags,
  title,
  t,
}: PublishGenerationActionOptions) {
  const hasImages = (imageCount ?? images.length) > 0;
  return {
    handleGeneratePublishContent: async (
      runtimePrepareImages?: PreparePublishGenerationImages,
      options?: PublishGenerationRunOptions,
    ) => {
      if (!isImageMode || !enabledByAdmin) {
        return;
      }
      if (isUploading) {
        toast.error(t("publish.imageManager.waitForUpload"));
        return;
      }
      let readyImages = images;
      const imagePreparer = runtimePrepareImages ?? prepareImages;
      if (imagePreparer) {
        readyImages = await imagePreparer();
      }
      if (!publishGenerationHasRemoteImages(readyImages)) {
        toast.error(t("publish.aiGenerate.uploadFirst"));
        return;
      }
      try {
        await run(publishGenerationRunInput({
          body,
          images: readyImages,
          tags,
          title,
        }, options));
      } catch (error) {
        toast.error(publishGenerationErrorMessage(error, t));
      }
    },
    handleApplyPublishGeneration: () => {
      const generatedDetail = state.generatedDetail.trim();
      const generatedTitle = state.generatedTitle.trim();
      if (!generatedDetail && !generatedTitle) {
        return;
      }
      if (generatedDetail) {
        onBody(generatedDetail);
      }
      if (generatedTitle) {
        onTitle(generatedTitle);
      }
      toast.success(t("publish.aiGenerate.applied"));
    },
    publishGenerationCanRun: isImageMode && !isUploading && enabledByAdmin && hasImages,
    publishGenerationImageCount: imageCount ?? images.length,
  };
}

function publishGenerationErrorMessage(error: unknown, t: ReturnType<typeof useTranslations>) {
  if (!(error instanceof Error)) {
    return t("publish.aiGenerate.failed");
  }
  const [code, ...detailParts] = error.message.split(":");
  const normalized = code.trim().replace(/^error\./, "");
  const errorKey = `publish.aiFormat.errors.${normalized}`;
  const label = t.has(errorKey) ? t(errorKey) : t("publish.aiGenerate.failed");
  const detail = detailParts.join(":").trim();
  return detail ? `${label} ${t("publish.aiFormat.apiErrorDetail", { detail })}` : label;
}
