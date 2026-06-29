"use client";
import { useState } from "react";
import {
  Headphones,
  ImageIcon,
  ImagePlus,
  Link2,
  Upload
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  Button
} from "@/components/ui/button";
import {
  type UploadProgress
} from "@/lib/api";
import type {
  UploadAsset
} from "@/lib/types";
import {
  UploadFailure,
  UploadMode,
  uploadAccept,
  uploadGuides
} from "./workbench-config";
import {
  UploadStatusList
} from "./article-composer";
import {
  ImageManager,
  type ManagedImageItem
} from "../shared/image-manager";
import type { ImageAccessPatch } from "../shared/image-access";

export function UploadPanel({
  assets,
  failures,
  imageProtectionEnabled,
  imageProtectionNoticeEnabled = true,
  imageSelectAllEnabled = true,
  paidContentEnabled = true,
  mode,
  onCancelUpload,
  onMoveAsset,
  onBatchUpdateImages,
  onRemoveAssets,
  onReorderAssets,
  onRemoveAsset,
  onRemoveFailure,
  onReplaceThumbnail,
  onRetryFailure,
  onToggleImageFree,
  onToggleImageProtection,
  onUploadFiles,
  progress,
  progressDetail,
  thumbnailDataUrl,
  uploading,
}: {
  assets: UploadAsset[];
  failures: UploadFailure[];
  imageProtectionEnabled?: boolean;
  imageProtectionNoticeEnabled?: boolean;
  imageSelectAllEnabled?: boolean;
  paidContentEnabled?: boolean;
  mode: UploadMode;
  onCancelUpload: () => void;
  onMoveAsset?: (assetUrl: string, direction: -1 | 1) => void;
  onBatchUpdateImages?: (
    assetUrls: string[],
    flags: ImageAccessPatch,
  ) => void;
  onRemoveAssets?: (assetUrls: string[]) => void;
  onReorderAssets?: (assetUrls: string[]) => void;
  onRemoveAsset: (assetUrl: string) => void;
  onRemoveFailure: (failureId: string) => void;
  onReplaceThumbnail?: () => void;
  onRetryFailure: (failure: UploadFailure) => void;
  onToggleImageFree?: (assetUrl: string) => void;
  onToggleImageProtection?: (assetUrl: string) => void;
  onUploadFiles: (files: FileList) => void;
  progress: number | null;
  progressDetail: UploadProgress | null;
  thumbnailDataUrl?: string | null;
  uploading: boolean;
}) {
  const t = useTranslations();
  const inputId = `publish-upload-${mode}`;
  const [selectedImageIds, setSelectedImageIds] = useState<string[]>([]);

  const availableImageIds = new Set(assets.map((asset) => asset.url));
  const validSelectedImageIds = selectedImageIds.filter((id) => availableImageIds.has(id));

  const managedImages: ManagedImageItem[] = mode === "image"
    ? assets.map((asset) => ({
        id: asset.url,
        previewUrl: getUploadAssetPreviewUrl(asset),
        name: asset.originalname ?? asset.url,
        size: asset.size,
        isFreePreview: asset.isFreePreview ?? true,
        isProtected: asset.isProtected ?? false,
        status: asset.uploadStatus ?? (asset.url.startsWith("blob:") ? "queued" : "succeeded"),
        progress: asset.uploadProgress,
        error: asset.uploadError,
      }))
    : [];

  if (mode === "podcast") {
    return (
      <div>
        <div className="flex min-h-[438px] flex-col items-center justify-center rounded-2xl bg-[#f7f7f8] px-6 text-center">
          <div className="flex size-24 items-center justify-center rounded-[28px] bg-white text-primary shadow-sm">
            <Headphones className="size-12" />
          </div>
          <p className="mt-5 text-[15px] font-semibold text-[#34343a]">
            {t("publish.upload.podcastTitle")}
          </p>
          <p className="mt-2 max-w-[520px] text-sm leading-6 text-[#777780]">
            {t("publish.upload.podcastDescription")}
          </p>
          <div className="mt-6 flex gap-3">
            <Button asChild className="h-10 cursor-pointer px-6">
              <label htmlFor={inputId}>
                <Upload className="size-4" />
                {t("publish.upload.audioButton")}
              </label>
            </Button>
            <input
              id={inputId}
              type="file"
              accept={uploadAccept[mode]}
              disabled={uploading}
              onChange={(event) => {
                if (event.target.files) {
                  onUploadFiles(event.target.files);
                  event.target.value = "";
                }
              }}
              className="hidden"
            />
            <Button variant="outline" className="h-10 border-[#e3e3e6] bg-white px-5 text-[#55555d]">
              <Link2 className="size-4" />
              {t("publish.upload.rssButton")}
            </Button>
          </div>
          <UploadStatusList
            assets={assets}
            failures={failures}
            mode={mode}
            onMoveAsset={onMoveAsset}
            onToggleImageFree={onToggleImageFree}
            onToggleImageProtection={onToggleImageProtection}
            onRemoveAsset={onRemoveAsset}
            onRemoveFailure={onRemoveFailure}
            onRetryFailure={onRetryFailure}
            progress={progress}
            progressDetail={progressDetail}
            uploading={uploading}
          />
          {uploading ? (
            <Button type="button" variant="outline" onClick={onCancelUpload} className="mt-3">
              {t("drawer.cancel")}
            </Button>
          ) : null}
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex min-h-[438px] flex-col rounded-2xl bg-[#f7f7f8] px-6 py-7 text-center">
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center">
          {mode === "video" && assets.length > 0 ? (
            <div className="w-full max-w-[640px] overflow-hidden rounded-2xl bg-black shadow-sm">
              <video
                src={getUploadAssetPreviewUrl(assets[0])}
                controls
                poster={thumbnailDataUrl ?? getUploadAssetCoverPreviewUrl(assets[0]) ?? undefined}
                className="aspect-video w-full bg-black object-contain"
              />
            </div>
          ) : mode === "image" && assets.length > 0 ? (
            <ImageManager
              disabled={uploading}
              items={managedImages}
              protectionEnabled={Boolean(imageProtectionEnabled)}
              protectionNoticeEnabled={imageProtectionNoticeEnabled}
              selectAllEnabled={imageSelectAllEnabled}
              paidContentEnabled={paidContentEnabled}
              selectedIds={validSelectedImageIds}
              onSelectionChange={setSelectedImageIds}
              onBatchUpdate={(ids, flags) => onBatchUpdateImages?.(ids, flags)}
              onRemove={(ids) => (onRemoveAssets ?? ((urls) => urls.forEach(onRemoveAsset)))(ids)}
              onReorder={(ids) => onReorderAssets?.(ids)}
              onRetry={(id) => {
                const failure = failures.find((item) => item.id === id || item.name === assets.find((asset) => asset.url === id)?.originalname);
                if (failure) {
                  onRetryFailure(failure);
                }
              }}
            />
          ) : (
            <>
              <div className="flex size-24 items-center justify-center rounded-[28px] bg-white text-primary shadow-sm">
                {mode === "video" ? <Upload className="size-12" /> : <ImagePlus className="size-12" />}
              </div>
              <p className="mt-5 text-[15px] font-semibold text-[#34343a]">
                {t(`publish.upload.${mode}Title`)}
              </p>
            </>
          )}
        </div>

        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <Button asChild className="h-10 cursor-pointer px-6">
            <label htmlFor={inputId}>
              <Upload className="size-4" />
              {t(`publish.upload.${mode}Primary`)}
            </label>
          </Button>
          <input
            id={inputId}
            type="file"
            accept={uploadAccept[mode]}
            multiple={mode === "image"}
            disabled={uploading}
            onChange={(event) => {
              if (event.target.files) {
                onUploadFiles(event.target.files);
                event.target.value = "";
              }
            }}
            className="hidden"
          />
          {mode === "video" && assets.length > 0 ? (
            <Button
              type="button"
              variant="outline"
              onClick={onReplaceThumbnail}
              className="h-10 border-[#e3e3e6] bg-white px-5 text-[#55555d]"
            >
              <ImageIcon className="size-4" />
              {thumbnailDataUrl ? t("publish.workbench.replaceCover") : t("publish.workbench.selectCover")}
            </Button>
          ) : null}
        </div>

        {mode === "video" || assets.length === 0 ? (
          <UploadStatusList
            assets={assets}
            failures={failures}
            imageProtectionEnabled={imageProtectionEnabled}
            mode={mode}
            onMoveAsset={onMoveAsset}
            onToggleImageFree={onToggleImageFree}
            onToggleImageProtection={onToggleImageProtection}
            onRemoveAsset={onRemoveAsset}
            onRemoveFailure={onRemoveFailure}
            onRetryFailure={onRetryFailure}
            progress={progress}
            progressDetail={progressDetail}
            uploading={uploading}
          />
        ) : failures.length > 0 || uploading ? (
          <UploadStatusList
            assets={[]}
            failures={failures}
            imageProtectionEnabled={imageProtectionEnabled}
            mode={mode}
            onMoveAsset={onMoveAsset}
            onToggleImageFree={onToggleImageFree}
            onToggleImageProtection={onToggleImageProtection}
            onRemoveAsset={onRemoveAsset}
            onRemoveFailure={onRemoveFailure}
            onRetryFailure={onRetryFailure}
            progress={progress}
            progressDetail={progressDetail}
            uploading={uploading}
          />
        ) : null}
        {uploading ? (
          <Button type="button" variant="outline" onClick={onCancelUpload} className="mx-auto mt-3">
            {t("drawer.cancel")}
          </Button>
        ) : null}
      </div>

      <div className="mt-6 grid gap-4 md:grid-cols-3">
        {uploadGuides[mode].map((guideKey) => (
          <div key={guideKey} className="rounded-2xl bg-[#fafafa] p-4">
            <p className="text-sm font-semibold text-[#3c3c43]">
              {t(`publish.guide.${guideKey}Title`)}
            </p>
            <p className="mt-2 text-xs leading-5 text-[#777780]">
              {t(`publish.guide.${guideKey}Body`)}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}


export function getUploadAssetPreviewUrl(asset: UploadAsset) {
  return asset.signedUrl || asset.url;
}


export function getUploadAssetCoverPreviewUrl(asset: UploadAsset) {
  return asset.coverSignedUrl || asset.coverUrl || null;
}
