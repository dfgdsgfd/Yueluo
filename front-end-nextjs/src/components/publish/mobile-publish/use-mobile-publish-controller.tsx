import type { AuthUser } from "@/lib/types";
import { useTranslations } from "next-intl";
import { useRouter, useSearchParams } from "next/navigation";
import { useMemo } from "react";
import { usePublishGeneration } from "../shared/ai-publish-generation";
import type { PublishGenerationRunOptions } from "../shared/publish-generation-action";
import { useMobilePublishControllerState } from "./mobile-publish-controller-state";
import { createMobilePublishDraftActions } from "./mobile-publish-draft-actions";
import { createMobilePublishGenerationAction } from "./mobile-publish-generation-action";
import { prepareMobileGenerationImages } from "./mobile-publish-generation-upload";
import { useMobilePublishLifecycle } from "./mobile-publish-lifecycle";
import { useMobilePublishMediaActions } from "./mobile-publish-media-actions";
import { createMobilePublishSubmitAction } from "./mobile-publish-submit-action";
import { hasMobilePublishContent } from "./mobile-publish-submit";
import { getDisplayName, toManagedImageItem } from "./mobile-publish-utils";

export function useMobilePublishController({ initialUser }: { initialUser?: AuthUser | null }) {
  const t = useTranslations();
  const {
    cancelPublishGeneration,
    publishGeneration,
    runPublishGenerationStream,
    selectedImageCount: selectedPublishGenerationImageCount,
  } = usePublishGeneration();
  const router = useRouter();
  const searchParams = useSearchParams();
  const state = useMobilePublishControllerState(initialUser);

  useMobilePublishLifecycle({ searchParams, state, t });

  const mediaKind = state.mediaAssets[0]?.kind ?? null;
  const imageAssets = state.mediaAssets.filter((asset) => asset.kind === "image");
  const paidImageCount = imageAssets.filter((asset) => !asset.isFreePreview).length;
  const paidContentEnabled = state.paidContentPaymentMethods.balance || state.paidContentPaymentMethods.points;
  const managedImages = imageAssets.map(toManagedImageItem);
  const {
    handleApplyPublishGeneration,
    handleGeneratePublishContent: runGeneratePublishContent,
    publishGenerationCanRun,
    publishGenerationImageCount,
  } = createMobilePublishGenerationAction({
    body: state.body,
    imageAssets,
    mediaKind,
    publishGeneration,
    runPublishGenerationStream,
    selectedImageCount: selectedPublishGenerationImageCount,
    setBody: state.setBody,
    setTitle: state.setTitle,
    title: state.title,
    topic: state.topic,
    t,
    uploadProgress: state.uploadProgress,
  });
  const handleGeneratePublishContent = (options?: PublishGenerationRunOptions) => runGeneratePublishContent(() =>
    prepareMobileGenerationImages({
      mediaAssets: state.mediaAssets,
      setMediaAssets: state.setMediaAssets,
      setUploadProgress: state.setUploadProgress,
      t,
      uploadAbortControllerRef: state.uploadAbortControllerRef,
    }), options);
  const displayName = getDisplayName(state.currentUser);
  const avatarUrl = state.currentUser?.avatar?.trim() || null;
  const hasAttachment = Boolean(state.attachmentFile || state.existingAttachment);
  const draftableContent = hasMobilePublishContent(
    state.title,
    state.body,
    state.mediaAssets.length,
    state.attachmentFile,
    `${state.topic}${state.board}${state.existingAttachment?.url ?? ""}`,
  );
  const publishableContent = hasMobilePublishContent(
    state.title,
    state.body,
    state.mediaAssets.length,
    state.attachmentFile,
    state.existingAttachment?.url ?? "",
  );
  const canPublish = !state.isSubmitting && publishableContent;
  const mediaHelpText = useMemo(() => {
    if (mediaKind === "video") {
      return t("publish.protection.mobileVideoAdded");
    }
    return state.mediaAssets.length > 0
      ? t("publish.protection.mobileImagesAdded", { count: state.mediaAssets.length, max: state.imageLimit })
      : t("publish.protection.mobileImageLimit", { count: state.imageLimit });
  }, [state.imageLimit, state.mediaAssets.length, mediaKind, t]);
  const availableImageIds = new Set(imageAssets.map((asset) => asset.id));
  const validSelectedImageIds = state.selectedImageIds.filter((id) => availableImageIds.has(id));
  const mediaActions = useMobilePublishMediaActions({
    mediaKind,
    paidContentEnabled,
    state,
    t,
  });
  const draftActions = createMobilePublishDraftActions({
    draftableContent,
    router,
    state,
    t,
  });
  const { handleSubmit } = createMobilePublishSubmitAction({
    paidContentEnabled,
    paidImageCount,
    publishableContent,
    router,
    state,
    t,
  });

  return {
    activeEmojiGroupId: state.activeEmojiGroupId,
    aiFormatOpen: state.aiFormatOpen,
    attachmentFile: state.attachmentFile,
    attachmentInputRef: state.attachmentInputRef,
    avatarUrl,
    board: state.board,
    body: state.body,
    canPublish,
    cancelPublishGeneration,
    categories: state.categories,
    closeWithoutSaving: draftActions.closeWithoutSaving,
    customTopicInput: state.customTopicInput,
    cycleVisibility: draftActions.cycleVisibility,
    displayName,
    existingAttachment: state.existingAttachment,
    fileInputRef: state.fileInputRef,
    handleApplyPublishGeneration,
    handleAttachmentChange: mediaActions.handleAttachmentChange,
    handleBatchUpdateImages: mediaActions.handleBatchUpdateImages,
    handleCloseClick: draftActions.handleCloseClick,
    handleGeneratePublishContent,
    handleMediaChange: mediaActions.handleMediaChange,
    handlePickAttachment: mediaActions.handlePickAttachment,
    handlePickMedia: mediaActions.handlePickMedia,
    handleRemoveImages: mediaActions.handleRemoveImages,
    handleReorderImages: mediaActions.handleReorderImages,
    handleReplaceThumbnailChange: mediaActions.handleReplaceThumbnailChange,
    handleReplaceVideo: mediaActions.handleReplaceVideo,
    handleReplaceVideoChange: mediaActions.handleReplaceVideoChange,
    handleRetryImage: mediaActions.handleRetryImage,
    handleSubmit,
    hasAttachment,
    imageLimit: state.imageLimit,
    imagePaymentMethod: state.imagePaymentMethod,
    imagePrice: state.imagePrice,
    imageProtectionEnabled: state.imageProtectionEnabled,
    imageProtectionNoticeEnabled: state.imageProtectionNoticeEnabled,
    imageSelectAllEnabled: state.imageSelectAllEnabled,
    insertEmoji: draftActions.insertEmoji,
    insertMention: draftActions.insertMention,
    isSearchingMentionUsers: state.isSearchingMentionUsers,
    isSubmitting: state.isSubmitting,
    lightboxAsset: state.lightboxAsset,
    markdownEditorMode: state.markdownEditorMode,
    managedImages,
    mediaAssets: state.mediaAssets,
    mediaHelpText,
    mediaKind,
    mentionKeyword: state.mentionKeyword,
    mentionUsers: state.mentionUsers,
    openDraftPage: draftActions.openDraftPage,
    openMediaPreview: mediaActions.openMediaPreview,
    paidContentEnabled,
    paidContentPaymentMethods: state.paidContentPaymentMethods,
    paidImageCount,
    paymentMaxPrices: state.paymentMaxPrices,
    postContentLimit: state.postContentLimit,
    publishGeneration,
    publishGenerationCanRun,
    publishGenerationImageCount,
    removeMedia: mediaActions.removeMedia,
    replaceThumbnailInputRef: state.replaceThumbnailInputRef,
    replaceVideoInputRef: state.replaceVideoInputRef,
    saveAndClose: draftActions.saveAndClose,
    selectedCategoryId: state.selectedCategoryId,
    setAIFormatOpen: state.setAIFormatOpen,
    setActiveEmojiGroupId: state.setActiveEmojiGroupId,
    setAttachmentFile: state.setAttachmentFile,
    setAttachmentTouched: state.setAttachmentTouched,
    setBoard: state.setBoard,
    setBody: state.setBody,
    setCustomTopicInput: state.setCustomTopicInput,
    setExistingAttachment: state.setExistingAttachment,
    setImagePaymentMethod: state.setImagePaymentMethod,
    setImagePrice: state.setImagePrice,
    setIsSearchingMentionUsers: state.setIsSearchingMentionUsers,
    setLightboxAsset: state.setLightboxAsset,
    setMarkdownEditorMode: state.setMarkdownEditorMode,
    setMentionKeyword: state.setMentionKeyword,
    setMentionUsers: state.setMentionUsers,
    setReplacingThumbnailAssetId: state.setReplacingThumbnailAssetId,
    setSelectedCategoryId: state.setSelectedCategoryId,
    setSelectedImageIds: state.setSelectedImageIds,
    setShowBoardSheet: state.setShowBoardSheet,
    setShowCloseConfirm: state.setShowCloseConfirm,
    setShowEmojiSheet: state.setShowEmojiSheet,
    setShowMentionSheet: state.setShowMentionSheet,
    setShowTopicSheet: state.setShowTopicSheet,
    setTitle: state.setTitle,
    setTopic: state.setTopic,
    showBoardSheet: state.showBoardSheet,
    showCloseConfirm: state.showCloseConfirm,
    showEmojiSheet: state.showEmojiSheet,
    showMentionSheet: state.showMentionSheet,
    showTopicSheet: state.showTopicSheet,
    submitCustomTopic: draftActions.submitCustomTopic,
    t,
    tags: state.tags,
    title: state.title,
    topic: state.topic,
    uploadAbortControllerRef: state.uploadAbortControllerRef,
    uploadProgress: state.uploadProgress,
    validSelectedImageIds,
    visibility: state.visibility,
  };
}
