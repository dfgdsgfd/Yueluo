import {
  createPost,
  updatePost,
  uploadAttachment,
  uploadImage,
  uploadVideo,
  type UploadProgress,
} from "@/lib/api";
import { mapWithConcurrency } from "@/lib/async-pool";
import { postContentLength } from "@/lib/post-content";
import { dataURLtoFile } from "@/lib/utils";
import type { useTranslations } from "next-intl";
import { toast } from "sonner";
import { enforceImageCoverPolicy } from "../shared/image-access";
import type { MobilePublishControllerState } from "./mobile-publish-controller-state";
import {
  defaultMobilePaymentMaxPrices,
  type MobileMediaAsset,
} from "./mobile-publish-config";
import {
  buildMobilePublishPayload,
  isMobilePublishAbortError,
  type UploadedMobileMedia,
} from "./mobile-publish-submit";

type MobilePublishSubmitActionOptions = {
  paidContentEnabled: boolean;
  paidImageCount: number;
  publishableContent: boolean;
  router: { push(href: string): void };
  state: MobilePublishControllerState;
  t: ReturnType<typeof useTranslations>;
};

export function createMobilePublishSubmitAction({
  paidContentEnabled,
  paidImageCount,
  publishableContent,
  router,
  state,
  t,
}: MobilePublishSubmitActionOptions) {
  const {
    attachmentFile,
    attachmentTouched,
    body,
    editingPostId,
    editingPostType,
    existingAttachment,
    imagePaymentMethod,
    imagePrice,
    mediaAssets,
    paidContentPaymentMethods,
    paymentMaxPrices,
    postContentLimit,
    selectedCategoryId,
    setAttachmentFile,
    setAttachmentTouched,
    setBoard,
    setBody,
    setCurrentDraftId,
    setEditingPostId,
    setEditingPostType,
    setExistingAttachment,
    setImagePaymentMethod,
    setImagePrice,
    setIsSubmitting,
    setMediaAssets,
    setSelectedCategoryId,
    setSelectedImageIds,
    setTitle,
    setTopic,
    setUploadProgress,
    setVisibility,
    title,
    topic,
    uploadAbortControllerRef,
    visibility,
  } = state;

  async function handleSubmit(isDraft = false) {
    if (!isDraft && !publishableContent) {
      toast.error(t("publish.mobile.contentRequired"));
      return;
    }
    if (postContentLength(body) > postContentLimit) {
      toast.error(t("publish.mobile.contentLimit", { count: postContentLimit }));
      return;
    }
    if (paidImageCount > 0 && (!paidContentEnabled || !paidContentPaymentMethods[imagePaymentMethod])) {
      toast.error(t("publish.protection.paymentDisabledToast"));
      return;
    }
    if (paidImageCount > 0) {
      const price = Number(imagePrice) || 0;
      const maxPrice = paymentMaxPrices[imagePaymentMethod] ?? defaultMobilePaymentMaxPrices[imagePaymentMethod];
      if (price > maxPrice) {
        toast.error(t("publish.protection.priceLimitToast", {
          method: t(`publish.protection.${imagePaymentMethod}`),
          max: maxPrice,
        }));
        return;
      }
    }
    const uploadController = new AbortController();
    uploadAbortControllerRef.current = uploadController;
    setIsSubmitting(true);
    setUploadProgress({ detail: null, percent: 0, phase: "uploading" });
    try {
      const localMediaAssets = mediaAssets.filter(
        (item): item is MobileMediaAsset & { file: File } => item.file instanceof File,
      );
      const uploadTasks = localMediaAssets.length + (attachmentFile ? 1 : 0);
      const progressByTask = new Map<string, number>();
      const updateFileProgress = (taskId: string, progress: UploadProgress) => {
        progressByTask.set(taskId, progress.percent ?? 0);
        const percent = uploadTasks > 0
          ? Math.min(
              96,
              Math.round(
                (Array.from(progressByTask.values()).reduce((sum, value) => sum + value, 0)
                  / uploadTasks)
                  * 0.96,
              ),
            )
          : 0;
        setUploadProgress({ detail: progress, percent, phase: "uploading" });
      };
      const uploadItems: Array<
        | { id: string; item: MobileMediaAsset & { file: File }; type: "media" }
        | { file: File; id: string; type: "attachment" }
      > = [
        ...localMediaAssets.map((item) => ({ id: `media-${item.id}`, item, type: "media" as const })),
        ...(attachmentFile
          ? [{ file: attachmentFile, id: "attachment", type: "attachment" as const }]
          : []),
      ];
      const uploadedItems = await mapWithConcurrency(uploadItems, 3, async (task) => {
        if (task.type === "media") {
          setMediaAssets((current) => current.map((asset) =>
            asset.id === task.item.id
              ? { ...asset, uploadError: null, uploadProgress: 0, uploadStatus: "uploading" }
              : asset,
          ));
        }
        const onTaskProgress = (progress: UploadProgress) => {
          updateFileProgress(task.id, progress);
          if (task.type === "media") {
            setMediaAssets((current) => current.map((asset) =>
              asset.id === task.item.id
                ? {
                    ...asset,
                    uploadError: null,
                    uploadProgress: progress.percent ?? asset.uploadProgress ?? 0,
                    uploadStatus: "uploading",
                  }
                : asset,
            ));
          }
        };
        try {
          const asset =
            task.type === "attachment"
              ? await uploadAttachment(task.file, {
                  onProgress: onTaskProgress,
                  signal: uploadController.signal,
                })
              : task.item.kind === "video"
                ? await uploadVideo(
                    task.item.file,
                    task.item.previewUrl
                      ? dataURLtoFile(task.item.previewUrl, "thumbnail.jpg")
                      : undefined,
                    {
                      onProgress: onTaskProgress,
                      signal: uploadController.signal,
                    },
                  )
                : await uploadImage(task.item.file, {
                    onProgress: onTaskProgress,
                    signal: uploadController.signal,
                  });
          if (task.type === "media") {
            setMediaAssets((current) => current.map((currentAsset) =>
              currentAsset.id === task.item.id
                ? { ...currentAsset, uploadError: null, uploadProgress: 100, uploadStatus: "succeeded" }
                : currentAsset,
            ));
          }
          progressByTask.set(task.id, 100);
          const percent = uploadTasks > 0
            ? Math.round(
                (Array.from(progressByTask.values()).reduce((sum, value) => sum + value, 0)
                  / uploadTasks)
                  * 0.96,
              )
            : 96;
          setUploadProgress({
            detail: {
              fileName: task.type === "attachment" ? task.file.name : task.item.name,
              loaded: task.type === "attachment" ? task.file.size : task.item.file.size,
              percent,
              stage: "complete",
              total: task.type === "attachment" ? task.file.size : task.item.file.size,
            },
            percent,
            phase: "uploading",
          });
          return { asset, error: null, task };
        } catch (error) {
          const message = error instanceof Error ? error.message : t("publish.imageManager.uploadFailed");
          if (task.type === "media") {
            setMediaAssets((current) => current.map((currentAsset) =>
              currentAsset.id === task.item.id
                ? { ...currentAsset, uploadError: message, uploadProgress: 0, uploadStatus: "failed" }
                : currentAsset,
            ));
          }
          return { asset: null, error: message, task };
        }
      });
      const failedUploads = uploadedItems.filter((result) => result.error);
      if (failedUploads.length > 0) {
        throw new Error(t("publish.imageManager.someUploadsFailed", { count: failedUploads.length }));
      }
      const uploadedLocalById = new Map(
        uploadedItems.flatMap((result) =>
          result.task.type === "media" && result.asset
            ? [[result.task.item.id, result.asset] as const]
            : [],
        ),
      );
      const uploadedMedia: UploadedMobileMedia[] = enforceImageCoverPolicy(
        mediaAssets.flatMap((item) => {
          const asset = item.remoteAsset ?? uploadedLocalById.get(item.id);
          return asset ? [{ ...item, asset }] : [];
        }),
      );
      const uploadedAttachment =
        uploadedItems.find((result) => result.task.type === "attachment" && result.asset)?.asset ?? null;
      uploadAbortControllerRef.current = null;
      setUploadProgress({
        detail: { loaded: 0, percent: 98, stage: "publishing", total: 0 },
        percent: 98,
        phase: "publishing",
      });
      const payload = buildMobilePublishPayload({
        attachmentFile,
        attachmentValue: existingAttachment
          ? {
              url: existingAttachment.url,
              filename: existingAttachment.filename,
              filesize: existingAttachment.filesize,
            }
          : editingPostId && attachmentTouched
            ? null
            : undefined,
        body,
        editingPostType,
        imagePaymentMethod,
        imagePrice,
        isDraft,
        isEditing: Boolean(editingPostId),
        paidImageCount,
        paymentMaxPrices,
        selectedCategoryId,
        title,
        topic,
        uploadedAttachment,
        uploadedMedia,
        visibility,
      });
      const result = editingPostId && !isDraft
        ? await updatePost(editingPostId, payload)
        : await createPost(payload);
      toast.success(editingPostId && !isDraft ? t("publish.mobile.updated") : isDraft ? t("publish.mobile.draftSaved") : t("publish.mobile.published"));
      setTitle("");
      setBody("");
      setTopic("");
      setBoard("");
      setSelectedCategoryId(null);
      setVisibility("public");
      setCurrentDraftId(null);
      setEditingPostId(null);
      setEditingPostType(null);
      setMediaAssets([]);
      setAttachmentFile(null);
      setExistingAttachment(null);
      setAttachmentTouched(false);
      setImagePaymentMethod("balance");
      setImagePrice("1");
      setSelectedImageIds([]);
      const resultId = result.id ?? editingPostId;
      if (!isDraft && resultId) {
        router.push(`/post?id=${encodeURIComponent(String(resultId))}`);
      }
    } catch (error) {
      if (!isMobilePublishAbortError(error)) {
        toast.error(
          error instanceof Error && error.message === "error.post_content_limit"
            ? t("publish.mobile.contentLimit", { count: postContentLimit })
            : error instanceof Error
              ? error.message
              : t("publish.mobile.publishFailed"),
        );
      }
    } finally {
      if (uploadAbortControllerRef.current === uploadController) {
        uploadAbortControllerRef.current = null;
      }
      setUploadProgress(null);
      setIsSubmitting(false);
    }
  }

  return { handleSubmit };
}
