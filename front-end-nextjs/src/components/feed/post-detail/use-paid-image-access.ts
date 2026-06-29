"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import type { QueryClient } from "@tanstack/react-query";
import type { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  ApiError,
  createProtectedPackage,
  downloadPostImageArchive,
  downloadProtectedPackage,
  getPostDetail,
  getPostImageArchive,
  getProtectedPackageStatus,
  getStoredAccessToken,
  purchaseContent,
  streamProtectedPackageEvents,
} from "@/lib/api";
import type { FeedPost, PostImageArchiveJob, ProtectedPackageJob } from "@/lib/types";
import { updateFeedPostState } from "../post-feed-state";
import { purchaseShortageFromError } from "./purchase-shortage";

export function usePaidImageAccess({
  inputPost,
  open,
  queryClient,
  t,
}: {
  inputPost: FeedPost | null;
  open: boolean;
  queryClient: QueryClient;
  t: ReturnType<typeof useTranslations>;
}) {
  const router = useRouter();
  const [protectedPackageJob, setProtectedPackageJob] = useState<ProtectedPackageJob | null>(null);
  const [imageArchiveJob, setImageArchiveJob] = useState<PostImageArchiveJob | null>(null);
  const [isCreatingProtectedPackage, setIsCreatingProtectedPackage] = useState(false);
  const [isDownloadingImageArchive, setIsDownloadingImageArchive] = useState(false);
  const [isDownloadingProtectedPackage, setIsDownloadingProtectedPackage] = useState(false);
  const [isRefreshingImageArchive, setIsRefreshingImageArchive] = useState(false);
  const [postOverride, setPostOverride] = useState<FeedPost | null>(null);
  const [isPurchasingContent, setIsPurchasingContent] = useState(false);
  const post = postOverride && String(postOverride.id) === String(inputPost?.id)
    ? postOverride
    : inputPost;
  const postId = post?.id;
  const autoProtectedArchivePostRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) {
        setProtectedPackageJob(null);
        setImageArchiveJob(null);
        setIsCreatingProtectedPackage(false);
        setIsDownloadingImageArchive(false);
        setIsDownloadingProtectedPackage(false);
        setIsRefreshingImageArchive(false);
        autoProtectedArchivePostRef.current = null;
      }
    });
    return () => {
      cancelled = true;
    };
  }, [postId]);

  useEffect(() => {
    if (!open || !postId) {
      return;
    }
    let cancelled = false;
    const controller = new AbortController();
    getPostDetail(postId, { signal: controller.signal })
      .then((refreshed) => {
        if (!cancelled) {
          setPostOverride(refreshed);
          updateFeedPostState(queryClient, refreshed.id, () => refreshed);
        }
      })
      .catch(() => {
        // The list item remains usable if the detail refresh is interrupted.
      });
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [open, postId, queryClient]);

  useEffect(() => {
    if (
      !open ||
      !post?.imageArchiveEligible ||
      post.imageArchiveMode !== "shared" ||
      post.imageArchiveRequiresPurchase
    ) {
      return;
    }
    let cancelled = false;
    let timer: number | undefined;
    const poll = () => {
      getPostImageArchive(post.id)
        .then((job) => {
          if (cancelled) return;
          setImageArchiveJob(job);
          if (job.status === "queued" || job.status === "processing") {
            timer = window.setTimeout(poll, 2000);
          }
        })
        .catch(() => {
          if (!cancelled) {
            setImageArchiveJob((current) => current
              ? { ...current, status: "failed", retryable: true }
              : failedPostImageArchiveJob(post));
          }
        });
    };
    poll();
    return () => {
      cancelled = true;
      if (timer) window.clearTimeout(timer);
    };
  }, [open, post]);

  useEffect(() => {
    if (
      !open ||
      !post?.imageArchiveEligible ||
      post.imageArchiveMode !== "protected" ||
      post.imageArchiveRequiresPurchase ||
      !post.protectedPackageAvailable ||
      protectedPackageJob ||
      isCreatingProtectedPackage ||
      !getStoredAccessToken()
    ) {
      return;
    }
    const key = String(post.id);
    if (autoProtectedArchivePostRef.current === key) {
      return;
    }
    autoProtectedArchivePostRef.current = key;
    setIsCreatingProtectedPackage(true);
    createProtectedPackage(post.id)
      .then((job) => setProtectedPackageJob((current) => mergeProtectedPackageJob(current, job)))
      .catch(() => {
        autoProtectedArchivePostRef.current = null;
      })
      .finally(() => setIsCreatingProtectedPackage(false));
  }, [
    isCreatingProtectedPackage,
    open,
    post,
    protectedPackageJob,
  ]);

  useEffect(() => {
    if (!open || !protectedPackageJob?.jobId) {
      return;
    }
    if (!["queued", "processing"].includes(protectedPackageJob.status)) {
      return;
    }
    let cancelled = false;
    let timer: number | undefined;
    const controller = new AbortController();
    const poll = () => {
      getProtectedPackageStatus(protectedPackageJob.jobId)
        .then((job) => {
          if (!cancelled) {
            setProtectedPackageJob((current) => mergeProtectedPackageJob(current, job));
          }
        })
        .catch(() => {
          if (!cancelled) {
            if (timer) window.clearInterval(timer);
          }
        });
    };
    streamProtectedPackageEvents(
      protectedPackageJob.jobId,
      (event) => {
        if (!cancelled) {
          setProtectedPackageJob((current) => mergeProtectedPackageJob(current, event));
        }
      },
      { signal: controller.signal },
    ).catch(() => {
      if (!cancelled) {
        poll();
        timer = window.setInterval(poll, 2000);
      }
    });
    return () => {
      cancelled = true;
      controller.abort();
      if (timer) window.clearInterval(timer);
    };
  }, [open, protectedPackageJob?.jobId, protectedPackageJob?.status]);

  async function handleCreateProtectedPackage() {
    if (!post || isCreatingProtectedPackage) {
      return;
    }
    setIsCreatingProtectedPackage(true);
    try {
      const job = await createProtectedPackage(post.id);
      setProtectedPackageJob((current) => mergeProtectedPackageJob(current, job));
      if (job.status !== "completed") {
        toast.success(t("drawer.protectedPackageQueued"));
      }
    } catch (error) {
      if (
        post &&
        error instanceof ApiError &&
        ["error.protected_package_missing", "error.protected_package_expired"].includes(error.message)
      ) {
        try {
          const replacement = await createProtectedPackage(post.id);
          setProtectedPackageJob((current) => mergeProtectedPackageJob(current, replacement));
          toast.success(t("drawer.protectedPackageRegenerating"));
          return;
        } catch (regenerationError) {
          toast.error(regenerationError instanceof Error ? regenerationError.message : t("drawer.protectedPackageFailed"));
          return;
        }
      }
      setProtectedPackageJob((current) => current ? {
        ...current,
        status: "failed",
        errorCode: "package_generation_failed",
        retryable: true,
      } : current);
      toast.error(error instanceof Error ? error.message : t("drawer.protectedPackageFailed"));
    } finally {
      setIsCreatingProtectedPackage(false);
    }
  }

  async function handlePurchaseContent() {
    if (!post || isPurchasingContent || !post.paymentSettings?.enabled) {
      return false;
    }
    setIsPurchasingContent(true);
    try {
      const method = post.paymentSettings.paymentMethod ?? "balance";
      await purchaseContent(post.id, method);
      const refreshed = await getPostDetail(post.id);
      setPostOverride(refreshed);
      updateFeedPostState(queryClient, refreshed.id, () => refreshed);
      setProtectedPackageJob(null);
      setImageArchiveJob(null);
      autoProtectedArchivePostRef.current = null;
      toast.success(t("drawer.purchaseSuccess"));
      return true;
    } catch (error) {
      const shortage = purchaseShortageFromError(error);
      if (shortage) {
        toast.error(t(`drawer.purchaseShortage.${shortage}.title`), {
          action: {
            label: t(`drawer.purchaseShortage.${shortage}.action`),
            onClick: () => router.push("/wallet"),
          },
          description: t(`drawer.purchaseShortage.${shortage}.description`),
          duration: 9000,
        });
      } else {
        toast.error(error instanceof Error ? error.message : t("drawer.purchaseFailed"));
      }
      return false;
    } finally {
      setIsPurchasingContent(false);
    }
  }

  async function handleDownloadProtectedPackage() {
    if (!protectedPackageJob?.jobId || isDownloadingProtectedPackage) {
      return;
    }
    setIsDownloadingProtectedPackage(true);
    try {
      const payload = await downloadProtectedPackage(protectedPackageJob.jobId);
      triggerBlobDownload(
        payload.blob,
        payload.filename ?? `protected-post-${post?.id ?? "images"}.zip`,
      );
      toast.success(t("drawer.protectedPackageDownloadStarted"));
    } catch (error) {
      if (
        post &&
        error instanceof ApiError &&
        ["error.protected_package_missing", "error.protected_package_expired"].includes(error.message)
      ) {
        try {
          const replacement = await createProtectedPackage(post.id);
          setProtectedPackageJob((current) => mergeProtectedPackageJob(current, replacement));
          toast.success(t("drawer.protectedPackageRegenerating"));
          return;
        } catch (regenerationError) {
          toast.error(regenerationError instanceof Error ? regenerationError.message : t("drawer.protectedPackageFailed"));
          return;
        }
      }
      setProtectedPackageJob((current) => current ? {
        ...current,
        status: "failed",
        errorCode: "package_generation_failed",
        retryable: true,
      } : current);
      toast.error(error instanceof Error ? error.message : t("drawer.protectedPackageFailed"));
    } finally {
      setIsDownloadingProtectedPackage(false);
    }
  }

  async function handleRefreshImageArchive() {
    if (!post || isRefreshingImageArchive) {
      return;
    }
    setIsRefreshingImageArchive(true);
    try {
      const job = await getPostImageArchive(post.id);
      setImageArchiveJob(job);
    } catch {
      setImageArchiveJob((current) => current
        ? { ...current, status: "failed", retryable: true }
        : failedPostImageArchiveJob(post));
    } finally {
      setIsRefreshingImageArchive(false);
    }
  }

  async function handleDownloadImageArchive() {
    if (!imageArchiveJob?.jobId || isDownloadingImageArchive) {
      return;
    }
    setIsDownloadingImageArchive(true);
    try {
      const payload = await downloadPostImageArchive(imageArchiveJob.jobId);
      triggerBlobDownload(payload.blob, payload.filename ?? `post-${post?.id ?? "images"}-images.zip`);
      toast.success(t("drawer.imageArchiveDownloadStarted"));
    } catch {
      toast.error(t("drawer.imageArchiveFailed"));
    } finally {
      setIsDownloadingImageArchive(false);
    }
  }

  return {
    handleCreateProtectedPackage,
    handleDownloadImageArchive,
    handleDownloadProtectedPackage,
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
    updateLocalPost,
  };

  function updateLocalPost(updater: (post: FeedPost) => FeedPost | null) {
    setPostOverride((currentOverride) => {
      const basePost = currentOverride && String(currentOverride.id) === String(inputPost?.id)
        ? currentOverride
        : inputPost;

      return basePost ? updater(basePost) : null;
    });
  }
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
}

function failedPostImageArchiveJob(post: FeedPost): PostImageArchiveJob {
  return {
    eligible: true,
    enabled: true,
    imageCount: post.totalImagesCount ?? 0,
    jobId: "",
    mode: "shared",
    postId: post.id,
    progress: 0,
    requiresPurchase: false,
    retryable: true,
    status: "failed",
    threshold: post.imageArchiveThreshold ?? 25,
  };
}

function mergeProtectedPackageJob(current: ProtectedPackageJob | null, next: ProtectedPackageJob): ProtectedPackageJob {
  if (!current || current.jobId !== next.jobId) {
    return normalizeProtectedPackageJob(next);
  }
  const currentRank = protectedPackageStatusRank(current.status);
  const nextRank = protectedPackageStatusRank(next.status);
  const terminal = protectedPackageTerminal(next.status);
  const nextIsFresh = terminal || protectedPackageJobIsFresh(current, next);
  const newerTimestamps = nextIsFresh
    ? { heartbeatAt: next.heartbeatAt, updatedAt: next.updatedAt }
    : { heartbeatAt: current.heartbeatAt, updatedAt: current.updatedAt };
  const status = terminal || nextRank >= currentRank ? next.status : current.status;
  const progress = terminal
    ? Math.max(0, Math.min(100, next.progress ?? current.progress ?? 0))
    : Math.max(current.progress ?? 0, next.progress ?? 0);
  const protectedImageCount = Math.max(current.protectedImageCount ?? 0, next.protectedImageCount ?? 0) || (next.protectedImageCount ?? current.protectedImageCount);
  return normalizeProtectedPackageJob({
    ...current,
    ...next,
    status,
    progress,
    processedImageCount: Math.max(current.processedImageCount ?? 0, next.processedImageCount ?? 0),
    currentImageIndex: Math.max(current.currentImageIndex ?? 0, next.currentImageIndex ?? 0),
    protectedImageCount,
    queuePosition: next.queuePosition ?? current.queuePosition,
    estimatedWaitSeconds: next.estimatedWaitSeconds ?? current.estimatedWaitSeconds,
    estimatedRemainingSeconds: next.estimatedRemainingSeconds ?? current.estimatedRemainingSeconds,
    elapsedSeconds: Math.max(current.elapsedSeconds ?? 0, next.elapsedSeconds ?? 0),
    currentStep: nextIsFresh ? next.currentStep : current.currentStep,
    activeProfile: nextIsFresh ? next.activeProfile : current.activeProfile,
    heartbeatAt: newerTimestamps.heartbeatAt,
    updatedAt: newerTimestamps.updatedAt,
    downloadUrl: next.downloadUrl ?? current.downloadUrl,
    errorCode: nextIsFresh ? next.errorCode ?? current.errorCode : current.errorCode,
    errorMessage: nextIsFresh ? next.errorMessage ?? current.errorMessage : current.errorMessage,
    retryable: next.retryable ?? current.retryable,
  });
}

function normalizeProtectedPackageJob(job: ProtectedPackageJob): ProtectedPackageJob {
  return {
    ...job,
    progress: Math.max(0, Math.min(100, job.progress ?? 0)),
    processedImageCount: Math.max(0, job.processedImageCount ?? 0),
    currentImageIndex: Math.max(0, job.currentImageIndex ?? 0),
  };
}

function protectedPackageTerminal(status: string) {
  return status === "completed" || status === "failed" || status === "expired";
}

function protectedPackageJobIsFresh(current: ProtectedPackageJob, next: ProtectedPackageJob) {
  const currentTime = Date.parse(String(current.updatedAt ?? current.heartbeatAt ?? current.createdAt ?? ""));
  const nextTime = Date.parse(String(next.updatedAt ?? next.heartbeatAt ?? next.createdAt ?? ""));
  if (!Number.isFinite(currentTime) || !Number.isFinite(nextTime)) {
    return true;
  }
  return nextTime >= currentTime;
}

function protectedPackageStatusRank(status: string) {
  switch (status) {
    case "queued":
      return 1;
    case "processing":
      return 2;
    case "completed":
    case "failed":
    case "expired":
      return 3;
    default:
      return 0;
  }
}
