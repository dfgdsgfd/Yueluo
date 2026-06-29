import type { FeedPost, UploadAsset } from "@/lib/types";
import { enforceImageCoverPolicy } from "../shared/image-access";
import type {
  ImagePaymentSettings,
  PublishMode,
  UploadMode,
  Visibility,
} from "./workbench-config";

export type WorkbenchPostEditState = {
  assets: Record<UploadMode, UploadAsset[]>;
  body: string;
  categoryId: number | null;
  mode: PublishMode;
  paymentSettings: ImagePaymentSettings;
  tags: string;
  title: string;
  visibility: Visibility;
};

export function workbenchPostEditState(post: FeedPost): WorkbenchPostEditState {
  return {
    assets: workbenchPostAssets(post),
    body: post.content ?? "",
    categoryId: typeof post.category_id === "number" ? post.category_id : null,
    mode: workbenchPostMode(post),
    paymentSettings: {
      enabled: Boolean(post.paymentSettings?.enabled),
      paymentMethod: post.paymentSettings?.paymentMethod ?? "balance",
      price: post.paymentSettings?.price ? String(post.paymentSettings.price) : "",
    },
    tags: (post.tags ?? []).map((tag) => tag.name).filter(Boolean).join(", "),
    title: post.title ?? "",
    visibility: mapBackendWorkbenchVisibility(post.visibility),
  };
}

export function workbenchPostMode(post: FeedPost): PublishMode {
  if (post.type === 2) {
    return "video";
  }

  if (post.attachment?.url) {
    return "podcast";
  }

  return "image";
}

export function workbenchPostAssets(post: FeedPost): Record<UploadMode, UploadAsset[]> {
  const assets: Record<UploadMode, UploadAsset[]> = {
    image: [],
    podcast: [],
    video: [],
  };

  assets.image = enforceImageCoverPolicy((post.images ?? []).flatMap((image) => {
    const url = typeof image === "string" ? image : image.url;
    return url
      ? [{
          isFreePreview: typeof image === "string" ? true : (image.isFreePreview ?? true),
          isProtected: typeof image === "string" ? false : (image.isProtected ?? false),
          originalname: fileNameFromUrl(url),
          uploadProgress: 100,
          uploadStatus: "succeeded" as const,
          url,
          watermarkTraceToken: typeof image === "string" ? undefined : image.watermarkTraceToken,
        }]
      : [];
  }));

  const video =
    post.videos?.[0] ??
    (post.video_url
      ? {
          cover_url: post.cover_url,
          video_url: post.video_url,
        }
      : null);
  if (video?.video_url) {
    assets.video = [{
      coverUrl: video.cover_url ?? post.cover_url ?? null,
      originalname: fileNameFromUrl(video.video_url),
      url: video.video_url,
    }];
  }

  if (post.attachment?.url) {
    assets.podcast = [{
      originalname: post.attachment.filename ?? fileNameFromUrl(post.attachment.url),
      size: post.attachment.filesize ?? undefined,
      url: post.attachment.url,
    }];
  }

  return assets;
}

export function mapBackendWorkbenchVisibility(value: string | undefined): Visibility {
  if (value === "friends_only" || value === "followers") {
    return "followers";
  }

  if (value === "private") {
    return "private";
  }

  return "public";
}

function fileNameFromUrl(url: string) {
  try {
    const pathname = new URL(url, "http://localhost").pathname;
    const name = decodeURIComponent(pathname.split("/").filter(Boolean).pop() ?? "");
    return name || url;
  } catch {
    return url;
  }
}
