import type { FeedPost, PostAttachment, UploadAsset } from "@/lib/types";
import { enforceImageCoverPolicy } from "../shared/image-access";
import {
  formatMobileTopics,
  mapBackendMobileVisibility,
  type MobileMediaAsset,
} from "./mobile-publish-config";

export type MobilePostEditState = {
  attachment: PostAttachment | null;
  body: string;
  categoryId: number | null;
  categoryName: string;
  media: MobileMediaAsset[];
  paymentMethod: "balance" | "points";
  price: string;
  title: string;
  topicInput: string;
  type: number;
  visibility: "public" | "followers" | "private";
};

export function mobilePostEditState(post: FeedPost): MobilePostEditState {
  const images = (post.images ?? [])
    .flatMap((image, index): MobileMediaAsset[] => {
      const url = typeof image === "string" ? image : image.url;
      if (!url) return [];
      const remoteAsset: UploadAsset = {
        url,
        originalname: fileNameFromUrl(url),
        watermarkTraceToken: typeof image === "string" ? undefined : image.watermarkTraceToken,
        isFreePreview: typeof image === "string" ? true : image.isFreePreview ?? true,
        isProtected: typeof image === "string" ? false : image.isProtected ?? false,
      };
      return [{
        file: null,
        id: `remote-image-${index}-${url}`,
        isFreePreview: remoteAsset.isFreePreview ?? true,
        isProtected: remoteAsset.isProtected ?? false,
        kind: "image",
        name: remoteAsset.originalname ?? fileNameFromUrl(url),
        previewUrl: url,
        remoteAsset,
        uploadProgress: 100,
        uploadStatus: "succeeded",
      }];
    });
  const video = post.videos?.[0] ?? (post.video_url ? {
    video_url: post.video_url,
    cover_url: post.cover_url,
  } : null);
  const media = images.length > 0
    ? enforceImageCoverPolicy(images)
    : video?.video_url
      ? [{
          file: null,
          id: `remote-video-${video.video_url}`,
          isFreePreview: true,
          isProtected: false,
          kind: "video" as const,
          name: fileNameFromUrl(video.video_url),
          previewUrl: video.cover_url ?? post.cover_url ?? "",
          remoteAsset: {
            url: video.video_url,
            coverUrl: video.cover_url ?? post.cover_url ?? null,
            originalname: fileNameFromUrl(video.video_url),
          },
          uploadProgress: 100,
          uploadStatus: "succeeded" as const,
        }]
      : [];

  return {
    attachment: post.attachment ?? null,
    body: post.content ?? "",
    categoryId: typeof post.category_id === "number" ? post.category_id : null,
    categoryName: post.category ?? "",
    media,
    paymentMethod: post.paymentSettings?.paymentMethod ?? "balance",
    price: post.paymentSettings?.price ? String(post.paymentSettings.price) : "1",
    title: post.title ?? "",
    topicInput: formatMobileTopics((post.tags ?? []).map((tag) => tag.name).filter(Boolean)),
    type: post.type,
    visibility: mapBackendMobileVisibility(post.visibility),
  };
}

function fileNameFromUrl(url: string) {
  try {
    const name = decodeURIComponent(new URL(url, "http://localhost").pathname.split("/").filter(Boolean).pop() ?? "");
    return name || "media";
  } catch {
    return "media";
  }
}
