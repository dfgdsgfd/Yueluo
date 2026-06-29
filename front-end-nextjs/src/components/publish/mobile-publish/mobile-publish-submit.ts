import type { PublishPostInput, UploadAsset } from "@/lib/types";
import {
  MobileMediaAsset,
  MobilePaymentMethod,
  Visibility,
  parseMobileTopics,
} from "./mobile-publish-config";

export type UploadedMobileMedia = MobileMediaAsset & {
  asset: UploadAsset;
};

export function hasMobilePublishContent(
  title: string,
  body: string,
  mediaCount: number,
  attachmentFile: File | null,
  extra = "",
) {
  return Boolean(title.trim() || body.trim() || extra.trim() || mediaCount > 0 || attachmentFile);
}

export function nextMobileVisibility(visibility: Visibility): Visibility {
  if (visibility === "public") return "followers";
  if (visibility === "followers") return "private";
  return "public";
}

export function buildMobilePublishPayload({
  attachmentFile,
  body,
  editingPostType,
  isEditing,
  imagePaymentMethod,
  imagePrice,
  isDraft,
  paidImageCount,
  paymentMaxPrices,
  selectedCategoryId,
  title,
  topic,
  uploadedAttachment,
  uploadedMedia,
  visibility,
  attachmentValue,
}: {
  attachmentFile: File | null;
  body: string;
  editingPostType: number | null;
  isEditing: boolean;
  imagePaymentMethod: "balance" | "points";
  imagePrice: string;
  isDraft: boolean;
  paidImageCount: number;
  paymentMaxPrices?: Record<MobilePaymentMethod, number>;
  selectedCategoryId: number | null;
  title: string;
  topic: string;
  uploadedAttachment: UploadAsset | null;
  uploadedMedia: UploadedMobileMedia[];
  visibility: Visibility;
  attachmentValue?: { url: string; filename?: string | null; filesize?: number | null } | null;
}) {
  const firstMedia = uploadedMedia[0];
  const payload: PublishPostInput = {
    title: title.trim(),
    content: body.trim(),
    type: firstMedia ? (firstMedia.kind === "video" ? 2 : 1) : editingPostType ?? 1,
    category_id: selectedCategoryId ?? (isEditing ? null : undefined),
    tags: topic.trim() ? parseMobileTopics(topic) : (isEditing ? [] : undefined),
    is_draft: isDraft,
    visibility: visibility === "followers" ? "friends_only" : visibility,
  };

  if (firstMedia?.kind === "video") {
    payload.video = {
      url: firstMedia.asset.url,
      coverUrl: firstMedia.asset.coverUrl ?? null,
    };
  } else if (uploadedMedia.length > 0 || (isEditing && editingPostType === 1)) {
    payload.images = uploadedMedia.map((item, index) => ({
      url: item.asset.url,
      watermarkTraceToken: item.asset.watermarkTraceToken,
      isFreePreview: item.isFreePreview,
      isProtected: item.isProtected,
      sortOrder: index + 1,
    }));
    const price = Math.max(1, Number(imagePrice) || 0);
    const maxPrice = paymentMaxPrices?.[imagePaymentMethod];
    payload.paymentSettings =
      paidImageCount > 0
        ? {
            enabled: true,
            paymentType: "single",
            paymentMethod: imagePaymentMethod,
            price: maxPrice && maxPrice > 0 ? Math.min(price, maxPrice) : price,
            freePreviewCount: uploadedMedia.filter((item) => item.isFreePreview).length,
            previewDuration: 0,
            hideAll: false,
          }
        : { enabled: false };
  }

  if (isEditing && editingPostType === 2 && uploadedMedia.length === 0) {
    payload.video = null;
  }

  if (uploadedAttachment) {
    payload.attachment = {
      url: uploadedAttachment.url,
      filename: uploadedAttachment.originalname ?? attachmentFile?.name,
      filesize: uploadedAttachment.size ?? attachmentFile?.size,
    };
  } else if (attachmentValue !== undefined) {
    payload.attachment = attachmentValue
      ? {
          url: attachmentValue.url,
          filename: attachmentValue.filename ?? undefined,
          filesize: attachmentValue.filesize ?? undefined,
        }
      : null;
  }

  return payload;
}

export function isMobilePublishAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}
