import { generateVideoThumbnail, isVideoFile } from "@/lib/utils";
import type { useTranslations } from "next-intl";
import type { ChangeEvent } from "react";
import { useCallback } from "react";
import { toast } from "sonner";
import { enforceImageCoverPolicy, type ImageAccessPatch } from "../shared/image-access";
import type { MobilePublishControllerState } from "./mobile-publish-controller-state";
import type { MobileMediaAsset } from "./mobile-publish-config";

type MobilePublishMediaActionsOptions = {
  mediaKind: MobileMediaAsset["kind"] | null;
  paidContentEnabled: boolean;
  state: MobilePublishControllerState;
  t: ReturnType<typeof useTranslations>;
};

export function useMobilePublishMediaActions({
  mediaKind,
  paidContentEnabled,
  state,
  t,
}: MobilePublishMediaActionsOptions) {
  const {
    attachmentInputRef,
    fileInputRef,
    imageLimit,
    imagePrice,
    imageProtectionEnabled,
    mediaAssets,
    replaceVideoInputRef,
    replacingThumbnailAssetId,
    setAttachmentFile,
    setAttachmentTouched,
    setExistingAttachment,
    setImagePrice,
    setLightboxAsset,
    setMediaAssets,
    setReplacingThumbnailAssetId,
  } = state;

  function handleReorderImages(ids: string[]) {
    const order = new Map(ids.map((id, index) => [id, index]));
    setMediaAssets((current) => enforceImageCoverPolicy([...current].sort((a, b) => {
      if (a.kind !== "image" || b.kind !== "image") {
        return 0;
      }
      return (order.get(a.id) ?? Number.MAX_SAFE_INTEGER) - (order.get(b.id) ?? Number.MAX_SAFE_INTEGER);
    })));
  }

  function handleBatchUpdateImages(
    ids: string[],
    flags: ImageAccessPatch,
  ) {
    if (flags.isProtected && !imageProtectionEnabled) {
      toast.error(t("publish.protection.disabledToast"));
      return;
    }
    if (flags.isFreePreview === false && !paidContentEnabled) {
      toast.error(t("publish.protection.paymentDisabledToast"));
      return;
    }
    const selected = new Set(ids);
    setMediaAssets((current) => enforceImageCoverPolicy(current.map((asset) =>
      selected.has(asset.id) ? { ...asset, ...flags } : asset,
    )));
    if (flags.isFreePreview === false && !imagePrice.trim()) {
      setImagePrice("1");
    }
  }

  function handleRemoveImages(ids: string[]) {
    const selected = new Set(ids);
    for (const asset of mediaAssets) {
      if (selected.has(asset.id) && asset.previewUrl.startsWith("blob:")) {
        URL.revokeObjectURL(asset.previewUrl);
      }
    }
    setMediaAssets((current) => enforceImageCoverPolicy(current.filter((asset) => !selected.has(asset.id))));
  }

  function handleRetryImage(id: string) {
    setMediaAssets((current) => current.map((asset) =>
      asset.id === id
        ? { ...asset, uploadError: null, uploadProgress: 0, uploadStatus: "queued" }
        : asset,
    ));
  }

  function handlePickMedia() {
    fileInputRef.current?.click();
  }

  function handlePickAttachment() {
    attachmentInputRef.current?.click();
  }

  function handleReplaceVideo() {
    replaceVideoInputRef.current?.click();
  }

  const processMediaFiles = useCallback((
    selectedFiles: File[],
    options?: { replaceVideo?: boolean },
  ) => {
    if (selectedFiles.length === 0) {
      return;
    }
    const firstKind = isVideoFile(selectedFiles[0]) ? "video" : "image";
    const normalizedFiles =
      firstKind === "video"
        ? selectedFiles.slice(0, 1)
        : selectedFiles.filter((file) => file.type.startsWith("image/"));
    if (firstKind === "video" && selectedFiles.length > 1) {
      toast.info(t("publish.mobile.singleVideo"));
    }
    if (!options?.replaceVideo) {
      if (mediaKind && mediaKind !== firstKind) {
        toast.error(t("publish.mobile.noMixedMedia"));
        return;
      }
      if (firstKind === "image" && mediaAssets.length + normalizedFiles.length > imageLimit) {
        toast.error(t("publish.protection.limitToast", { count: imageLimit }));
        return;
      }
      if (firstKind === "video" && mediaAssets.length > 0) {
        toast.error(t("publish.mobile.singleVideo"));
        return;
      }
    }
    const localAssets: MobileMediaAsset[] = normalizedFiles.map((file) => ({
      file,
      id: `${file.name}-${file.lastModified}-${crypto.randomUUID()}`,
      isFreePreview: true,
      isProtected: false,
      kind: firstKind,
      name: file.name,
      previewUrl: firstKind === "video" ? "" : URL.createObjectURL(file),
      uploadProgress: 0,
      uploadStatus: "queued",
    }));
    if (options?.replaceVideo) {
      setMediaAssets(localAssets);
    } else {
      setMediaAssets((currentAssets) => firstKind === "image"
        ? enforceImageCoverPolicy([...currentAssets, ...localAssets])
        : [...currentAssets, ...localAssets]);
    }
    toast.success(firstKind === "video" ? t("publish.mobile.videoAdded") : t("publish.mobile.imageAdded"));
    if (firstKind === "video") {
      for (const asset of localAssets) {
        const assetId = asset.id;
        if (!asset.file) continue;
        void generateVideoThumbnail(asset.file).then((dataUrl) => {
          if (dataUrl) {
            setMediaAssets((prev) =>
              prev.map((a) => (a.id === assetId ? { ...a, previewUrl: dataUrl } : a)),
            );
          }
        });
      }
    }
  }, [imageLimit, mediaKind, mediaAssets.length, setMediaAssets, t]);

  function handleMediaChange(event: ChangeEvent<HTMLInputElement>) {
    const selectedFiles = Array.from(event.target.files ?? []);
    event.target.value = "";
    processMediaFiles(selectedFiles);
  }

  function handleAttachmentChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null;
    event.target.value = "";
    if (!file) {
      return;
    }
    setAttachmentFile(file);
    setExistingAttachment(null);
    setAttachmentTouched(true);
    toast.success(t("publish.mobile.attachmentAdded"));
  }

  function handleReplaceVideoChange(event: ChangeEvent<HTMLInputElement>) {
    const selectedFiles = Array.from(event.target.files ?? []);
    event.target.value = "";
    if (selectedFiles.length === 0) {
      return;
    }
    const file = selectedFiles[0];
    if (!isVideoFile(file)) {
      toast.error(t("publish.mobile.selectReplacementVideo"));
      return;
    }
    processMediaFiles([file], { replaceVideo: true });
  }

  function handleReplaceThumbnailChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file || !file.type.startsWith("image/") || !replacingThumbnailAssetId) {
      setReplacingThumbnailAssetId(null);
      return;
    }
    const url = URL.createObjectURL(file);
    const targetId = replacingThumbnailAssetId;
    setReplacingThumbnailAssetId(null);
    setMediaAssets((prev) =>
      prev.map((a) => (a.id === targetId ? { ...a, previewUrl: url } : a)),
    );
  }

  function removeMedia(id: string) {
    setMediaAssets((currentAssets) => {
      const asset = currentAssets.find((a) => a.id === id);
      const next = currentAssets.filter((a) => a.id !== id);
      if (asset) {
        toast.success(asset.kind === "video" ? t("publish.mobile.videoDeleted") : t("publish.mobile.imageDeleted"));
      }
      return next[0]?.kind === "image" ? enforceImageCoverPolicy(next) : next;
    });
  }

  function openMediaPreview(item: MobileMediaAsset) {
    setLightboxAsset(item);
  }

  return {
    handleAttachmentChange,
    handleBatchUpdateImages,
    handleMediaChange,
    handlePickAttachment,
    handlePickMedia,
    handleRemoveImages,
    handleReorderImages,
    handleReplaceThumbnailChange,
    handleReplaceVideo,
    handleReplaceVideoChange,
    handleRetryImage,
    openMediaPreview,
    removeMedia,
  };
}
