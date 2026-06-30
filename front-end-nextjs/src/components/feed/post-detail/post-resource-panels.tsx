"use client";

import { Button } from "@/components/ui/button";
import { buildApiUrl,getBalanceLocalPoints,getPostPurchaseUsers,getWithdrawWallet } from "@/lib/api";
import type { FeedPost,PostImageArchiveJob,PostPurchaseUser,ProtectedPackageJob } from "@/lib/types";
import * as Dialog from "@radix-ui/react-dialog";
import { Copy,CreditCard,Download,Loader2,LockKeyhole,ShieldCheck,Users,X } from "lucide-react";
import type { useTranslations } from "next-intl";
import Link from "next/link";
import { useEffect,useRef,useState } from "react";
import { toast } from "sonner";
import { getUnlockImageCounts } from "./paid-image-access-utils";

export function PurchaseUnlockPanel({
  isPurchasing,
  onPurchase,
  post,
  t,
}: {
  isPurchasing: boolean;
  onPurchase: () => Promise<boolean>;
  post: FeedPost;
  t: ReturnType<typeof useTranslations>;
}) {
  const [open, setOpen] = useState(false);
  const [availableFunds, setAvailableFunds] = useState<number | null>(null);
  const [fundsStatus, setFundsStatus] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const payment = post.paymentSettings;
  const method = payment?.paymentMethod ?? "balance";
  const { directCount, protectedCount } = getUnlockImageCounts(post);

  useEffect(() => {
    if (!open || !payment?.enabled) {
      return;
    }
    let cancelled = false;
    const request = method === "points"
      ? getBalanceLocalPoints().then((payload) => payload.points)
      : getWithdrawWallet().then((payload) => payload.cash_balance);
    Promise.resolve()
      .then(() => {
        if (!cancelled) {
          setFundsStatus("loading");
        }
        return request;
      })
      .then((funds) => {
        if (!cancelled) {
          setAvailableFunds(Number(funds));
          setFundsStatus("ready");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setAvailableFunds(null);
          setFundsStatus("error");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [method, open, payment?.enabled]);

  if (!payment?.enabled || post.hasPurchased || post.isAuthor) {
    return null;
  }

  async function confirmPurchase() {
    if (await onPurchase()) {
      setOpen(false);
    }
  }

  return (
    <section className="mt-5 overflow-hidden rounded-2xl border border-primary/25 bg-primary/[0.08] p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/18 text-primary">
          <LockKeyhole className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-white">{t("drawer.purchaseTitle")}</h3>
          <p className="mt-1 text-xs leading-5 text-white/58">
            {t("drawer.purchaseDescription", { direct: directCount, protected: protectedCount })}
          </p>
        </div>
        <strong className="shrink-0 text-base text-primary">
          {t("drawer.purchasePrice", { price: payment.price })}
        </strong>
      </div>
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog.Trigger asChild>
          <Button type="button" className="mt-4 h-10 w-full">
            <CreditCard className="size-4" />
            {t("drawer.purchaseWith", { method: t(`publish.protection.${method}`) })}
          </Button>
        </Dialog.Trigger>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-[80] bg-black/65 backdrop-blur-sm" />
          <Dialog.Content className="fixed inset-x-0 bottom-0 z-[81] rounded-t-[22px] border border-white/10 bg-[#18181c] p-5 text-white shadow-2xl outline-none md:left-1/2 md:top-1/2 md:bottom-auto md:w-[min(420px,calc(100vw-2rem))] md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-[22px]">
            <Dialog.Title className="text-lg font-black">{t("drawer.purchaseConfirmTitle")}</Dialog.Title>
            <Dialog.Description className="mt-2 text-sm leading-6 text-white/58">
              {t("drawer.purchaseConfirmDescription", {
                price: payment.price,
                method: t(`publish.protection.${method}`),
              })}
            </Dialog.Description>
            <div className="mt-4 grid grid-cols-2 gap-2 rounded-2xl bg-white/[0.05] p-3 text-sm">
              <div className="min-w-0">
                <span className="block text-xs text-white/42">{t("drawer.purchaseAvailable")}</span>
                <strong className="mt-1 block truncate text-white">
                  {fundsStatus === "loading"
                    ? t("drawer.purchaseBalanceLoading")
                    : fundsStatus === "error"
                      ? t("drawer.purchaseBalanceError")
                      : formatPurchaseAmount(availableFunds)}
                </strong>
              </div>
              <div className="min-w-0 text-right">
                <span className="block text-xs text-white/42">{t("drawer.purchaseRequired")}</span>
                <strong className="mt-1 block truncate text-primary">{formatPurchaseAmount(payment.price)}</strong>
              </div>
            </div>
            <div className="mt-5 grid grid-cols-2 gap-3 pb-[env(safe-area-inset-bottom)]">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" className="h-11 border-white/10 bg-white/[0.04] text-white">
                  {t("drawer.cancel")}
                </Button>
              </Dialog.Close>
              <Button type="button" disabled={isPurchasing} onClick={() => void confirmPurchase()} className="h-11">
                {isPurchasing ? <Loader2 className="size-4 animate-spin" /> : <CreditCard className="size-4" />}
                {t("drawer.confirmPurchase")}
              </Button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </section>
  );
}

export function AuthorPaidContentPanel({
  onNavigate,
  post,
  t,
}: {
  onNavigate: () => void;
  post: FeedPost;
  t: ReturnType<typeof useTranslations>;
}) {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<PostPurchaseUser[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const payment = post.paymentSettings;
  const method = payment?.paymentMethod ?? "balance";

  useEffect(() => {
    if (!open || !post.id) {
      return;
    }
    let cancelled = false;
    getPostPurchaseUsers(post.id, 1, 50)
      .then((payload) => {
        if (cancelled) {
          return;
        }
        setItems(payload.list ?? []);
        setTotal(Number(payload.pagination?.total ?? payload.list?.length ?? 0));
        setStatus("ready");
      })
      .catch(() => {
        if (!cancelled) {
          setStatus("error");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, post.id]);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (nextOpen) {
      setStatus("loading");
    }
  }

  if (!post.isAuthor || !post.isPaidContent || !payment?.enabled) {
    return null;
  }

  return (
    <section className="mt-5 overflow-hidden rounded-2xl border border-white/[0.08] bg-white/[0.045] p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary">
          <Users className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-white">{t("drawer.authorPaidNoticeTitle")}</h3>
          <p className="mt-1 text-xs leading-5 text-white/58">
            {t("drawer.authorPaidNoticeDescription", {
              method: t(`publish.protection.${method}`),
              price: payment.price,
            })}
          </p>
        </div>
      </div>
      <Dialog.Root open={open} onOpenChange={handleOpenChange}>
        <Dialog.Trigger asChild>
          <Button type="button" variant="outline" className="mt-4 h-10 w-full border-white/10 bg-white/[0.04] text-white hover:bg-white/[0.08]">
            <Users className="size-4" />
            {t("drawer.viewPurchasers")}
          </Button>
        </Dialog.Trigger>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-[80] bg-black/65 backdrop-blur-sm" />
          <Dialog.Content className="fixed inset-x-0 bottom-0 z-[81] max-h-[78dvh] overflow-hidden rounded-t-[22px] border border-white/10 bg-[#18181c] p-5 text-white shadow-2xl outline-none md:left-1/2 md:top-1/2 md:bottom-auto md:w-[min(480px,calc(100vw-2rem))] md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-[22px]">
            <div className="flex items-start gap-3">
              <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary">
                <Users className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <Dialog.Title className="text-lg font-black">
                  {t("drawer.purchasersTitle", { count: total })}
                </Dialog.Title>
                <Dialog.Description className="mt-1 text-sm leading-6 text-white/58">
                  {t("drawer.purchasersDescription")}
                </Dialog.Description>
              </div>
              <Dialog.Close asChild>
                <button
                  type="button"
                  className="flex size-9 shrink-0 items-center justify-center rounded-full text-white/56 transition-colors hover:bg-white/10 hover:text-white"
                  aria-label={t("drawer.cancel")}
                >
                  <X className="size-5" />
                </button>
              </Dialog.Close>
            </div>
            <div className="mt-4 max-h-[52dvh] space-y-2 overflow-y-auto pr-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
              {status === "loading" ? (
                <div className="flex h-24 items-center justify-center text-sm text-white/58">
                  <Loader2 className="mr-2 size-4 animate-spin" />
                  {t("drawer.purchasersLoading")}
                </div>
              ) : status === "error" ? (
                <div className="rounded-2xl bg-white/[0.05] p-4 text-sm text-white/58">
                  {t("drawer.purchasersError")}
                </div>
              ) : items.length === 0 ? (
                <div className="rounded-2xl bg-white/[0.05] p-4 text-sm text-white/58">
                  {t("drawer.purchasersEmpty")}
                </div>
              ) : (
                items.map((item) => {
                  const buyer = item.buyer;
                  const buyerName = buyer?.nickname || buyer?.user_id || t("drawer.unknownBuyer");
                  const buyerHref = buyer?.user_id ? `/user/${encodeURIComponent(buyer.user_id)}` : undefined;
                  const row = (
                    <div className="flex min-w-0 items-center gap-3 rounded-2xl bg-white/[0.05] p-3 transition-colors hover:bg-white/[0.08]">
                      {buyer?.avatar ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={buyer.avatar} alt="" className="size-10 shrink-0 rounded-full object-cover" />
                      ) : (
                        <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary/15 text-sm font-bold text-primary">
                          {buyerName.slice(0, 1).toUpperCase()}
                        </span>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-white">{buyerName}</p>
                        <p className="mt-0.5 truncate text-xs text-white/45">
                          {formatPurchaseDate(item.purchased_at)}
                        </p>
                      </div>
                      <div className="shrink-0 text-right">
                        <strong className="block text-sm text-primary">{formatPurchaseAmount(item.paid_amount)}</strong>
                        <span className="mt-0.5 block text-xs text-white/45">
                          {t(`publish.protection.${item.payment_method}`)}
                        </span>
                      </div>
                    </div>
                  );
                  return buyerHref ? (
                    <Link key={String(item.id)} href={buyerHref} onClick={onNavigate} className="block">
                      {row}
                    </Link>
                  ) : (
                    <div key={String(item.id)}>{row}</div>
                  );
                })
              )}
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </section>
  );
}

export function ImageArchivePanel({
  isDownloading,
  isRefreshing,
  job,
  onDownload,
  onRefresh,
  post,
  t,
}: {
  isDownloading: boolean;
  isRefreshing: boolean;
  job: PostImageArchiveJob | null;
  onDownload: () => Promise<void>;
  onRefresh: () => Promise<void>;
  post: FeedPost;
  t: ReturnType<typeof useTranslations>;
}) {
  const requiresPurchase = Boolean(post.imageArchiveRequiresPurchase);
  const status = job?.status ?? (requiresPurchase ? "locked" : "queued");
  const inProgress = status === "queued" || status === "processing";
  const completed = status === "completed";
  const failed = status === "failed" || status === "expired";
  const progress = Math.max(0, Math.min(100, job?.progress ?? 0));
  const total = Math.max(0, job?.imageCount ?? post.totalImagesCount ?? 0);
  const processed = Math.max(0, Math.min(total, job?.processedImageCount ?? 0));

  return (
    <section className="mt-5 overflow-hidden rounded-2xl border border-white/[0.08] bg-white/[0.045] p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary">
          <Download className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-white">{t("drawer.imageArchiveTitle")}</h3>
          <p className="mt-1 text-xs leading-5 text-white/58">
            {t("drawer.imageArchiveDescription", {
              count: total,
              threshold: post.imageArchiveThreshold ?? 25,
            })}
          </p>
        </div>
      </div>

      {requiresPurchase ? (
        <div className="mt-4 rounded-xl bg-black/18 px-3 py-3 text-xs leading-5 text-white/58">
          <LockKeyhole className="mr-2 inline size-4 text-primary" />
          {t("drawer.imageArchivePurchaseRequired")}
        </div>
      ) : (
        <div className="mt-4 rounded-xl bg-black/18 p-3">
          <div className="flex items-center justify-between gap-3 text-xs text-white/58">
            <span>
              {completed
                ? t("drawer.imageArchiveReady")
                : failed
                  ? t("drawer.imageArchiveFailed")
                  : t("drawer.imageArchiveProcessing")}
            </span>
            <span>{progress}%</span>
          </div>
          <div className="mt-2 h-2 overflow-hidden rounded-full bg-white/10">
            <div className="h-full rounded-full bg-primary transition-[width]" style={{ width: `${progress}%` }} />
          </div>
          {inProgress ? (
            <p className="mt-2 text-xs text-white/50">
              {t("drawer.imageArchiveWorking", { current: processed, total })}
            </p>
          ) : null}
        </div>
      )}

      <div className="mt-4">
        {completed ? (
          <Button type="button" onClick={() => void onDownload()} disabled={isDownloading} className="h-10 w-full">
            {isDownloading ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
            {t("drawer.imageArchiveDownload")}
          </Button>
        ) : failed ? (
          <Button type="button" onClick={() => void onRefresh()} disabled={isRefreshing} className="h-10 w-full">
            {isRefreshing ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
            {t("drawer.imageArchiveRetry")}
          </Button>
        ) : (
          <Button type="button" disabled className="h-10 w-full">
            <Loader2 className="size-4 animate-spin" />
            {requiresPurchase ? t("drawer.imageArchivePurchaseRequired") : t("drawer.imageArchiveProcessing")}
          </Button>
        )}
      </div>
    </section>
  );
}

export function ProtectedPackagePanel({
  isCreating,
  isDownloading,
  job,
  onCreate,
  onDownload,
  post,
  t,
}: {
  isCreating: boolean;
  isDownloading: boolean;
  job: ProtectedPackageJob | null;
  onCreate: () => Promise<void>;
  onDownload: () => Promise<void>;
  post: FeedPost;
  t: ReturnType<typeof useTranslations>;
}) {
  const availableCount = post.protectedPackageImageCount ?? 0;
  const lockedCount = post.lockedProtectedImagesCount ?? 0;
  const hasAvailablePackage = Boolean(post.protectedPackageAvailable);
  const status = job?.status ?? "idle";
  const inProgress = status === "queued" || status === "processing";
  const completed = status === "completed";
  const failed = status === "failed" || status === "expired";
  const progress = Math.max(0, Math.min(100, job?.progress ?? 0));
  const totalImages = Math.max(0, job?.protectedImageCount ?? availableCount);
  const processedImages = Math.max(0, Math.min(totalImages, job?.processedImageCount ?? 0));
  const currentImageIndex = Math.max(0, Math.min(totalImages, job?.currentImageIndex ?? processedImages));
  const showCurrentImage = inProgress && currentImageIndex > processedImages;
  const partialSkipped = completed && job?.errorCode === "partial_images_skipped";
  const remainingSeconds = job
    ? Math.max(
        0,
        status === "queued"
          ? (job.estimatedWaitSeconds ?? job.estimatedRemainingSeconds ?? 0)
          : (job.estimatedRemainingSeconds ?? job.estimatedWaitSeconds ?? 0),
      )
    : 0;
  const [downloadPromptOpen, setDownloadPromptOpen] = useState(false);
  const promptedJobIdRef = useRef<string | null>(null);
  const downloadUrl = job?.downloadUrl ? absoluteProtectedPackageUrl(job.downloadUrl) : "";

  useEffect(() => {
    if (!completed || !job?.jobId || !downloadUrl || promptedJobIdRef.current === job.jobId) {
      return;
    }
    promptedJobIdRef.current = job.jobId;
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) {
        setDownloadPromptOpen(true);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [completed, downloadUrl, job?.jobId]);

  async function copyDownloadUrl() {
    if (!downloadUrl) {
      return;
    }
    try {
      await navigator.clipboard.writeText(downloadUrl);
      toast.success(t("drawer.protectedPackageLinkCopied"));
    } catch {
      toast.error(t("drawer.protectedPackageLinkCopyFailed"));
    }
  }

  return (
    <section className="mt-5 overflow-hidden rounded-2xl border border-white/[0.08] bg-white/[0.045] p-4">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary">
          <ShieldCheck className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-white">
            {t("drawer.protectedPackageTitle")}
          </h3>
          <p className="mt-1 text-xs leading-5 text-white/58">
            {post.imageArchiveEligible
              ? t("drawer.protectedPackageAllImagesDescription", {
                  count: post.totalImagesCount ?? 0,
                })
              : t("drawer.protectedPackageDescription", {
                  count: post.protectedImagesCount ?? 0,
                })}
          </p>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
        <div className="rounded-xl bg-black/18 px-3 py-2">
          <span className="block text-white/42">{t("drawer.protectedAvailable")}</span>
          <strong className="mt-1 block text-sm text-white">{availableCount}</strong>
        </div>
        <div className="rounded-xl bg-black/18 px-3 py-2">
          <span className="block text-white/42">{t("drawer.protectedLocked")}</span>
          <strong className="mt-1 block text-sm text-white">{lockedCount}</strong>
        </div>
      </div>

      {job ? (
        <div className="mt-4 rounded-xl bg-black/18 p-3">
          <div className="flex items-center justify-between gap-3 text-xs text-white/58">
            <span>{t(protectedStatusLabelKey(status))}</span>
            <span>{progress}%</span>
          </div>
          <div className="mt-2 h-2 overflow-hidden rounded-full bg-white/10">
            <div
              className="h-full rounded-full bg-primary transition-[width]"
              style={{ width: `${progress}%` }}
            />
          </div>
          {inProgress ? (
            <div className="mt-2 space-y-1 text-xs text-white/50">
              {status === "queued" ? (
                <p>{t("drawer.protectedQueueInfo", {
                  count: Math.max(1, job.queueCount ?? 1),
                  position: Math.max(1, job.queuePosition ?? 1),
                  seconds: Math.max(0, job.estimatedWaitSeconds ?? 0),
                })}</p>
              ) : null}
              {showCurrentImage && !post.imageArchiveEligible ? (
                <p>{t("drawer.protectedCurrentImage", {
                  current: currentImageIndex,
                  total: totalImages,
                })}</p>
              ) : null}
              <p>
                {t(post.imageArchiveEligible ? "drawer.imageArchiveWorking" : "drawer.protectedWorking", {
                  current: processedImages,
                  total: totalImages,
                })}
              </p>
              <p>{t("drawer.protectedStage", {
                stage: t(`drawer.protectedStages.${protectedStageKey(job.currentStep)}`),
              })}</p>
              <p>{t("drawer.protectedTiming", {
                elapsed: Math.max(0, job.elapsedSeconds ?? 0),
                remaining: remainingSeconds,
              })}</p>
              {job.activeProfile ? <p>{t("drawer.protectedProfile", { profile: job.activeProfile })}</p> : null}
            </div>
          ) : null}
          {failed ? (
            <div className="mt-2 space-y-1 text-xs text-primary">
              <p>{t(`drawer.protectedErrors.${protectedErrorKey(job.errorCode)}`)}</p>
              {job.errorMessage ? (
                <p className="break-words text-white/45">
                  {t("drawer.protectedErrorDetail", { detail: job.errorMessage })}
                </p>
              ) : null}
            </div>
          ) : null}
          {partialSkipped ? (
            <div className="mt-2 space-y-1 text-xs text-primary">
              <p>{t("drawer.protectedErrors.partialSkipped")}</p>
              {job.errorMessage ? (
                <p className="break-words text-white/45">
                  {t("drawer.protectedErrorDetail", { detail: job.errorMessage })}
                </p>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="mt-4">
        {completed ? (
          <Button
            type="button"
            onClick={() => setDownloadPromptOpen(true)}
            disabled={!downloadUrl}
            className="h-10 w-full"
          >
            <Download className="size-4" />
            {t("drawer.protectedDownload")}
          </Button>
        ) : (
          <Button
            type="button"
            onClick={() => void onCreate()}
            disabled={isCreating || inProgress || !hasAvailablePackage}
            className="h-10 w-full"
          >
            {isCreating || inProgress ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
            {hasAvailablePackage
              ? t(inProgress
                  ? "drawer.protectedGenerating"
                  : failed && job?.retryable
                    ? "drawer.protectedRetry"
                    : "drawer.protectedCreate")
              : t("drawer.protectedPurchaseRequired")}
          </Button>
        )}
      </div>
      <Dialog.Root open={downloadPromptOpen} onOpenChange={setDownloadPromptOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-[80] bg-black/65 backdrop-blur-sm" />
          <Dialog.Content className="fixed inset-x-0 bottom-0 z-[81] rounded-t-[22px] border border-white/10 bg-[#18181c] p-5 text-white shadow-2xl outline-none md:left-1/2 md:top-1/2 md:bottom-auto md:w-[min(480px,calc(100vw-2rem))] md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-[22px]">
            <div className="flex items-start gap-3">
              <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary">
                <Download className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <Dialog.Title className="text-lg font-black">
                  {t("drawer.protectedPackageReadyTitle")}
                </Dialog.Title>
                <Dialog.Description className="mt-1 text-sm leading-6 text-white/58">
                  {t("drawer.protectedPackageReadyDescription")}
                </Dialog.Description>
              </div>
              <Dialog.Close asChild>
                <button
                  type="button"
                  className="flex size-9 shrink-0 items-center justify-center rounded-full text-white/56 transition-colors hover:bg-white/10 hover:text-white"
                  aria-label={t("drawer.cancel")}
                >
                  <X className="size-5" />
                </button>
              </Dialog.Close>
            </div>
            <div className="mt-4 rounded-2xl border border-white/10 bg-black/20 p-3">
              <label className="block text-xs font-semibold text-white/52" htmlFor={`protected-package-url-${job?.jobId ?? "ready"}`}>
                {t("drawer.protectedPackageLinkLabel")}
              </label>
              <div className="mt-2 flex min-w-0 items-center gap-2">
                <input
                  id={`protected-package-url-${job?.jobId ?? "ready"}`}
                  readOnly
                  value={downloadUrl}
                  className="h-10 min-w-0 flex-1 rounded-xl border border-white/10 bg-white/[0.05] px-3 text-xs text-white/78 outline-none selection:bg-primary/30"
                  onFocus={(event) => event.currentTarget.select()}
                />
                <Button
                  type="button"
                  variant="outline"
                  className="h-10 shrink-0 border-white/10 bg-white/[0.04] px-3 text-white"
                  onClick={() => void copyDownloadUrl()}
                >
                  <Copy className="size-4" />
                  {t("drawer.protectedPackageCopyLink")}
                </Button>
              </div>
            </div>
            <div className="mt-5 grid grid-cols-2 gap-3 pb-[env(safe-area-inset-bottom)]">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" className="h-11 border-white/10 bg-white/[0.04] text-white">
                  {t("drawer.cancel")}
                </Button>
              </Dialog.Close>
              <Button
                type="button"
                disabled={isDownloading}
                onClick={() => void onDownload()}
                className="h-11"
              >
                {isDownloading ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                {t("drawer.protectedDownload")}
              </Button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </section>
  );
}

export function protectedStageKey(stage: string | null | undefined) {
  switch (stage) {
    case "preparing":
    case "queued":
    case "reading_source":
    case "preparing_watermark_trace":
    case "updating_watermark_trace":
    case "connecting_watermark_server":
    case "watermark_server_connected":
    case "watermark_stream_unavailable":
    case "decoding":
    case "embedding":
    case "encoding":
    case "direct_verification":
    case "transport_verification":
    case "verifying":
    case "image_completed":
    case "image_skipped":
    case "writing_archive":
    case "finalizing":
    case "completed":
      return stage;
    default:
      return "processing";
  }
}

export function absoluteProtectedPackageUrl(path: string) {
  const built = buildApiUrl(path);
  if (typeof window === "undefined" || !built.startsWith("/")) {
    return built;
  }
  return new URL(built, window.location.origin).toString();
}

export function protectedStatusLabelKey(status: string) {
  switch (status) {
    case "queued":
      return "drawer.protectedStatus.queued";
    case "processing":
      return "drawer.protectedStatus.processing";
    case "completed":
      return "drawer.protectedStatus.completed";
    case "failed":
      return "drawer.protectedStatus.failed";
    case "expired":
      return "drawer.protectedStatus.expired";
    default:
      return "drawer.protectedStatus.idle";
  }
}

export function protectedErrorKey(errorCode?: string | null) {
  switch (errorCode) {
    case "watermark_verification_failed":
      return "watermark";
    case "watermark_server_unavailable":
      return "watermarkServerUnavailable";
    case "watermark_server_timeout":
      return "watermarkServerTimeout";
    case "watermark_server_rejected":
      return "watermarkServerRejected";
    case "image_source_unavailable":
      return "source";
    case "no_eligible_images":
      return "empty";
    case "selection_changed":
      return "selectionChanged";
    case "partial_images_skipped":
      return "partialSkipped";
    default:
      return "default";
  }
}

export function formatPurchaseAmount(value: number | null | undefined) {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) {
    return "—";
  }
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(Number(value));
}

export function formatPurchaseDate(value: string | undefined) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
