import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { toast } from "sonner";
import { createPost, updatePost } from "@/lib/api";
import type { FeedPost, UploadAsset } from "@/lib/types";
import {
  type ImagePaymentSettings,
  type PaymentMethod,
  type PendingUploadFile,
  type PublishMode,
  type UploadFailure,
  type UploadMode,
  type UploadProgressDetailState,
  type UploadProgressState,
  type Visibility,
} from "./workbench-config";
import {
  buildPostInput,
  isLocalAssetUrl,
  revokePendingObjectUrls,
  uploadPendingAssets,
} from "./post-builders";

type UploadAssetState = Record<UploadMode, UploadAsset[]>;
type PendingFileState = Record<UploadMode, PendingUploadFile[]>;
type UploadFailureState = Record<UploadMode, UploadFailure[]>;

export async function submitWorkbenchPost({
  body,
  currentDraftId,
  editingPostId,
  imagePaymentSettings,
  paymentMaxPrices,
  isDraft,
  isSubmitting,
  mode,
  pendingFiles,
  refreshDrafts,
  resetUploadState,
  router,
  selectedCategoryId,
  setBody,
  setCurrentDraftId,
  setDrafts,
  setEditingPostId,
  setIsSubmitting,
  setPendingFiles,
  setSelectedCategoryId,
  setTags,
  setTitle,
  setUploadedAssets,
  setUploadFailures,
  setUploadingMode,
  setUploadProgress,
  setUploadProgressDetails,
  tags,
  title,
  messages,
  translateUploadFailures,
  uploadAbortControllerRef,
  uploadedAssets,
  uploadingMode,
  visibility,
}: {
  body: string;
  currentDraftId: string | number | null;
  editingPostId: string | number | null;
  imagePaymentSettings: ImagePaymentSettings;
  paymentMaxPrices: Record<PaymentMethod, number>;
  isDraft: boolean;
  isSubmitting: boolean;
  mode: PublishMode;
  pendingFiles: PendingFileState;
  refreshDrafts: (options: { silent?: boolean }) => Promise<void>;
  resetUploadState: () => void;
  router: { push: (href: string) => void };
  selectedCategoryId: number | null;
  setBody: Dispatch<SetStateAction<string>>;
  setCurrentDraftId: Dispatch<SetStateAction<string | number | null>>;
  setDrafts: Dispatch<SetStateAction<FeedPost[]>>;
  setEditingPostId: Dispatch<SetStateAction<string | number | null>>;
  setIsSubmitting: Dispatch<SetStateAction<boolean>>;
  setPendingFiles: Dispatch<SetStateAction<PendingFileState>>;
  setSelectedCategoryId: Dispatch<SetStateAction<number | null>>;
  setTags: Dispatch<SetStateAction<string>>;
  setTitle: Dispatch<SetStateAction<string>>;
  setUploadedAssets: Dispatch<SetStateAction<UploadAssetState>>;
  setUploadFailures: Dispatch<SetStateAction<UploadFailureState>>;
  setUploadingMode: Dispatch<SetStateAction<UploadMode | null>>;
  setUploadProgress: Dispatch<SetStateAction<UploadProgressState>>;
  setUploadProgressDetails: Dispatch<SetStateAction<UploadProgressDetailState>>;
  tags: string;
  title: string;
  messages: {
    draftSaved: string;
    contentLimit: string;
    publishFailed: string;
    published: string;
    updated: string;
    uploadFailed: string;
    uploadMediaFirst: string;
    waitForUpload: string;
  };
  translateUploadFailures: (count: number) => string;
  uploadAbortControllerRef: MutableRefObject<AbortController | null>;
  uploadedAssets: UploadAssetState;
  uploadingMode: UploadMode | null;
  visibility: Visibility;
}) {
  if (isSubmitting) {
    return;
  }

  if (uploadingMode) {
    toast.error(messages.waitForUpload);
    return;
  }

  const uploadController = new AbortController();
  uploadAbortControllerRef.current = uploadController;
  setIsSubmitting(true);
  try {
    let realAssets: UploadAssetState = {
      image: uploadedAssets.image.filter((asset) => !isLocalAssetUrl(asset.url)),
      video: uploadedAssets.video.filter((asset) => !isLocalAssetUrl(asset.url)),
      podcast: uploadedAssets.podcast.filter((asset) => !isLocalAssetUrl(asset.url)),
    };

    if (mode !== "article") {
      const uploadMode = mode as UploadMode;
      const selectedAssets = uploadedAssets[uploadMode];

      if (pendingFiles[uploadMode].length > 0) {
        setUploadingMode(uploadMode);
        setUploadProgress((current) => ({ ...current, [uploadMode]: 0 }));
        setUploadProgressDetails((current) => ({
          ...current,
          [uploadMode]: {
            fileName: pendingFiles[uploadMode][0]?.file.name,
            loaded: 0,
            percent: 0,
            stage: "preparing",
            total: pendingFiles[uploadMode].reduce((sum, item) => sum + item.file.size, 0),
          },
        }));
        const uploadResult = await uploadPendingAssets(
          uploadMode,
          selectedAssets,
          pendingFiles[uploadMode],
          (progress) => {
            setUploadProgress((current) => ({
              ...current,
              [uploadMode]: progress.percent ?? current[uploadMode] ?? 0,
            }));
            setUploadProgressDetails((current) => ({ ...current, [uploadMode]: progress }));
          },
          messages.uploadFailed,
          (assetUrl, state) => {
            setUploadedAssets((current) => ({
              ...current,
              [uploadMode]: current[uploadMode].map((asset) =>
                asset.url === assetUrl ? { ...asset, ...state } : asset,
              ),
            }));
          },
          uploadController.signal,
        );
        const uploadedLocalAssets = uploadResult.assets;
        realAssets = {
          ...realAssets,
          [uploadMode]: uploadedLocalAssets.filter((asset) => !isLocalAssetUrl(asset.url)),
        };
        setUploadedAssets((current) => ({ ...current, [uploadMode]: uploadedLocalAssets }));
        setPendingFiles((current) => ({
          ...current,
          [uploadMode]: current[uploadMode].filter((item) =>
            uploadResult.failures.some((failure) => failure.assetUrl === item.blobUrl),
          ),
        }));
        setUploadFailures((current) => ({
          ...current,
          [uploadMode]: uploadResult.failures.map((failure) => ({
            file: failure.file,
            id: failure.assetUrl,
            message: failure.error,
            name: failure.file.name,
            size: failure.file.size,
          })),
        }));
        setUploadProgress((current) => ({ ...current, [uploadMode]: null }));
        setUploadProgressDetails((current) => ({ ...current, [uploadMode]: null }));
        setUploadingMode(null);
        if (uploadResult.failures.length > 0) {
          toast.error(translateUploadFailures(uploadResult.failures.length));
          return;
        }
      }

      if (!isDraft && realAssets[uploadMode].length === 0) {
        toast.error(messages.uploadMediaFirst);
        return;
      }
    }

    const payload = buildPostInput({
      body,
      categoryId: selectedCategoryId,
      isDraft,
      mode,
      tags,
      title,
      uploadedAssets: realAssets,
      visibility,
      paymentSettings: imagePaymentSettings,
      paymentMaxPrices,
    });
    const shouldUpdateDraft = isDraft && currentDraftId !== null;
    const shouldUpdatePublishedPost = !isDraft && editingPostId !== null;
    uploadAbortControllerRef.current = null;
    const result = shouldUpdatePublishedPost
      ? await updatePost(editingPostId, payload)
      : shouldUpdateDraft
      ? await updatePost(currentDraftId, payload)
      : await createPost(payload);
    const resultId = result.id ?? editingPostId ?? currentDraftId;
    toast.success(isDraft ? messages.draftSaved : shouldUpdatePublishedPost ? messages.updated : messages.published);

    if (isDraft) {
      if (!currentDraftId && resultId) {
        setCurrentDraftId(resultId);
      }
      revokePendingObjectUrls(pendingFiles);
      setUploadedAssets(realAssets);
      setPendingFiles({ image: [], video: [], podcast: [] });
      setUploadFailures({ image: [], video: [], podcast: [] });
      setUploadProgress({ image: null, video: null, podcast: null });
      setUploadProgressDetails({ image: null, video: null, podcast: null });
      void refreshDrafts({ silent: true });
    }

    if (!isDraft) {
      setCurrentDraftId(null);
      setEditingPostId(null);
      setTitle("");
      setBody("");
      setTags("");
      setSelectedCategoryId(null);
      resetUploadState();
    }

    if (!isDraft && resultId) {
      setDrafts((current) => current.filter((draft) => draft.id !== resultId));
      router.push(`/post?id=${encodeURIComponent(String(resultId))}`);
    }
  } catch (error) {
    if (!(error instanceof Error && error.name === "AbortError")) {
      toast.error(
        error instanceof Error && error.message === "error.post_content_limit"
          ? messages.contentLimit
          : error instanceof Error
            ? error.message
            : messages.publishFailed,
      );
    }
  } finally {
    if (uploadAbortControllerRef.current === uploadController) {
      uploadAbortControllerRef.current = null;
    }
    setUploadingMode(null);
    setUploadProgressDetails({ image: null, video: null, podcast: null });
    setIsSubmitting(false);
  }
}
