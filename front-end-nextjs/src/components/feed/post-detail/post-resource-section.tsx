"use client";

import type { FeedPost,PostImageArchiveJob,ProtectedPackageJob } from "@/lib/types";
import type { useTranslations } from "next-intl";
import { PostAttachmentBlock } from "./actions-panel";
import { AuthorPaidContentPanel,ImageArchivePanel,ProtectedPackagePanel,PurchaseUnlockPanel } from "./post-resource-panels";

export function PostResourceSection({
  attachment,
  handleCreateProtectedPackage,
  handleDownloadImageArchive,
  handleDownloadProtectedPackage,
  handleNavigateAway,
  handlePurchaseContent,
  handleRefreshImageArchive,
  imageArchiveJob,
  isCreatingProtectedPackage,
  isDownloadingImageArchive,
  isDownloadingProtectedPackage,
  isPurchasingContent,
  isRefreshingImageArchive,
  post,
  protectedPackageJob,
  t,
}: {
  attachment: FeedPost["attachment"] | null;
  handleCreateProtectedPackage: () => Promise<void>;
  handleDownloadImageArchive: () => Promise<void>;
  handleDownloadProtectedPackage: () => Promise<void>;
  handleNavigateAway: () => void;
  handlePurchaseContent: () => Promise<boolean>;
  handleRefreshImageArchive: () => Promise<void>;
  imageArchiveJob: PostImageArchiveJob | null;
  isCreatingProtectedPackage: boolean;
  isDownloadingImageArchive: boolean;
  isDownloadingProtectedPackage: boolean;
  isPurchasingContent: boolean;
  isRefreshingImageArchive: boolean;
  post: FeedPost;
  protectedPackageJob: ProtectedPackageJob | null;
  t: ReturnType<typeof useTranslations>;
}) {
  return (
    <>
      {attachment ? <PostAttachmentBlock attachment={attachment} /> : null}
      <PurchaseUnlockPanel isPurchasing={isPurchasingContent} onPurchase={handlePurchaseContent} post={post} t={t} />
      <AuthorPaidContentPanel onNavigate={handleNavigateAway} post={post} t={t} />
      {post.imageArchiveEligible && !post.protectedPackageRequired ? (
        <ImageArchivePanel
          isDownloading={isDownloadingImageArchive}
          isRefreshing={isRefreshingImageArchive}
          job={imageArchiveJob}
          onDownload={handleDownloadImageArchive}
          onRefresh={handleRefreshImageArchive}
          post={post}
          t={t}
        />
      ) : null}
      {post.protectedPackageRequired ? (
        <ProtectedPackagePanel
          isCreating={isCreatingProtectedPackage}
          isDownloading={isDownloadingProtectedPackage}
          job={protectedPackageJob}
          onCreate={handleCreateProtectedPackage}
          onDownload={handleDownloadProtectedPackage}
          post={post}
          t={t}
        />
      ) : null}
    </>
  );
}
