"use client";
import {
  useState,
  type FormEvent,
  type ReactNode,
  type SyntheticEvent
} from "react";
import Link from "next/link";
import useEmblaCarousel from "embla-carousel-react";
import {
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Flag,
  ThumbsDown
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  Button
} from "@/components/ui/button";
import {
  cn
} from "@/lib/utils";
import type {
  PostAttachment,
  ReportReason
} from "@/lib/types";
import {
  reportReasons
} from "./post-detail-types";
import {
  formatAttachmentSize,
  formatCount,
  isAudioAttachment
} from "./post-detail-formatters";

export function DetailActionsPanel({
  description,
  disliked,
  isMutatingDislike,
  isSubmittingReport,
  onDescriptionChange,
  onDislikeToggle,
  onReasonChange,
  onReportSubmit,
  reason,
  reported,
}: {
  description: string;
  disliked: boolean;
  isMutatingDislike: boolean;
  isSubmittingReport: boolean;
  onDescriptionChange: (value: string) => void;
  onDislikeToggle: () => void;
  onReasonChange: (value: ReportReason) => void;
  onReportSubmit: (event: FormEvent<HTMLFormElement>) => void;
  reason: ReportReason;
  reported: boolean;
}) {
  const t = useTranslations();

  return (
    <div className="mb-5 rounded-[10px] border border-white/[0.08] bg-white/[0.04] p-3">
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant={disliked ? "default" : "outline"}
          onClick={onDislikeToggle}
          disabled={isMutatingDislike}
          className={cn(
            "h-9 rounded-full px-3 text-xs",
            disliked
              ? "bg-primary text-white hover:bg-primary/90"
              : "border-white/[0.1] bg-white/[0.04] text-white/72 hover:bg-white/[0.08]",
          )}
        >
          <ThumbsDown className={cn("size-4", disliked && "fill-current")} />
          {disliked ? t("drawer.dislikeRemove") : t("drawer.dislike")}
        </Button>
      </div>

      <form onSubmit={onReportSubmit} className="mt-3 space-y-2">
        <div className="flex items-center gap-2">
          <Flag className="size-4 shrink-0 text-white/45" />
          <select
            value={reason}
            onChange={(event) => onReasonChange(event.target.value as ReportReason)}
            disabled={reported || isSubmittingReport}
            className="h-9 min-w-0 flex-1 rounded-md border border-white/[0.1] bg-[#1b1b1f] px-2 text-xs text-white/72 outline-none focus:ring-2 focus:ring-primary/30"
            aria-label={t("drawer.reportReason")}
          >
            {reportReasons.map((item) => (
              <option key={item} value={item}>
                {t(`drawer.reportReasons.${item}`)}
              </option>
            ))}
          </select>
        </div>
        <textarea
          value={description}
          onChange={(event) => onDescriptionChange(event.target.value)}
          disabled={reported || isSubmittingReport}
          maxLength={300}
          aria-label={t("drawer.reportDescription")}
          placeholder={t("drawer.reportDescription")}
          className="min-h-16 w-full resize-none rounded-md border border-white/[0.1] bg-white/[0.04] px-3 py-2 text-xs leading-5 text-white/72 outline-none placeholder:text-white/35 focus:ring-2 focus:ring-primary/30"
        />
        <Button
          type="submit"
          size="sm"
          disabled={reported || isSubmittingReport}
          className="h-8 rounded-full px-3 text-xs"
        >
          <Flag className="size-3.5" />
          {reported ? t("drawer.reported") : t("drawer.report")}
        </Button>
      </form>
    </div>
  );
}


export function PostAttachmentBlock({ attachment }: { attachment: PostAttachment }) {
  const filename = attachment.filename?.trim() || "Attachment";
  const filesize = formatAttachmentSize(attachment.filesize);
  const audio = isAudioAttachment(attachment);

  return (
    <div className="mt-4 rounded-[10px] border border-white/[0.08] bg-white/[0.04] p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-white">
            {filename}
          </p>
          {filesize ? (
            <p className="mt-1 text-xs text-white/45">
              {filesize}
            </p>
          ) : null}
        </div>
        <Button
          asChild
          variant="ghost"
          size="icon"
          className="size-9 shrink-0 text-white/60"
        >
          <Link href={attachment.url} target="_blank" aria-label={filename}>
            <ExternalLink className="size-4" />
          </Link>
        </Button>
      </div>
      {audio ? <audio controls src={attachment.url} className="mt-3 w-full" /> : null}
    </div>
  );
}


export function FooterAction({
  active,
  ariaLabel,
  count,
  icon,
  onClick,
}: {
  active?: boolean;
  ariaLabel: string;
  count: number;
  icon: ReactNode;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex h-10 w-[50px] items-center justify-center gap-0.5 rounded-full text-sm hover:text-primary",
        active && "text-primary",
      )}
      aria-label={ariaLabel}
    >
      {icon}
      <span className="min-w-2 text-xs">{formatCount(count)}</span>
    </button>
  );
}


export function CommentToolButton({
  ariaLabel,
  icon,
}: {
  ariaLabel: string;
  icon: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={ariaLabel}
      title={ariaLabel}
      onMouseDown={(event) => event.preventDefault()}
      className="flex size-8 items-center justify-center rounded-full text-[var(--post-comment-footer-muted)] transition-colors hover:bg-[var(--post-comment-control-hover)] hover:text-[var(--post-comment-footer-text)]"
    >
      {icon}
    </button>
  );
}


export function ImageCarousel({
  images,
  title,
  emblaRef,
  emblaApi,
  previousLabel,
  nextLabel,
  failedLabel,
  openLabel,
  onImageOpen,
  counterLabel,
  overflowCount = 0,
}: {
  images: string[];
  title: string;
  emblaRef: (node: HTMLElement | null) => void;
  emblaApi: ReturnType<typeof useEmblaCarousel>[1];
  previousLabel: string;
  nextLabel: string;
  failedLabel: string;
  openLabel: string;
  onImageOpen: (index: number) => void;
  counterLabel: string;
  overflowCount?: number;
}) {
  const [loadedImages, setLoadedImages] = useState<Record<string, true>>({});
  const [failedImages, setFailedImages] = useState<Record<string, true>>({});

  function handleImageLoad(
    slideKey: string,
    event: SyntheticEvent<HTMLImageElement>,
  ) {
    const image = event.currentTarget;
    const markLoaded = () => {
      setLoadedImages((current) => ({ ...current, [slideKey]: true }));
      emblaApi?.reInit();
    };

    if ("decode" in image) {
      void image.decode().catch(() => undefined).finally(markLoaded);
      return;
    }

    markLoaded();
  }

  return (
    <div className="relative size-full overflow-hidden md:rounded-l-[20px]">
      <div ref={emblaRef} className="size-full overflow-hidden">
        <div className="flex h-full">
          {images.map((image, index) => {
            const slideKey = `${image}-${index}`;
            const loaded = Boolean(loadedImages[slideKey]);
            const failed = Boolean(failedImages[slideKey]);
            const loading = !loaded && !failed;

            return (
              <div key={slideKey} className="relative h-full min-w-0 flex-[0_0_100%]">
                <button
                  type="button"
                  aria-label={`${openLabel} ${index + 1}`}
                  onClick={() => onImageOpen(index)}
                  className="absolute inset-0 cursor-zoom-in overflow-hidden text-left"
                >
                  {failed ? (
                    <span className="absolute inset-0 flex items-center justify-center bg-[#151515] px-6 text-center text-sm text-white/60">
                      {failedLabel}
                    </span>
                  ) : null}
                  {loading ? (
                    <span
                      aria-hidden
                      className="absolute inset-0 overflow-hidden bg-[#101010]"
                    >
                      <span className="absolute inset-0 bg-[linear-gradient(135deg,rgba(255,255,255,0.04)_0%,rgba(255,255,255,0.11)_45%,rgba(255,255,255,0.035)_100%)]" />
                      <span className="absolute inset-x-8 top-1/2 h-px bg-gradient-to-r from-transparent via-white/18 to-transparent" />
                      <span className="absolute inset-y-0 -left-1/2 w-1/2 bg-gradient-to-r from-transparent via-white/18 to-transparent animate-[feed-image-shimmer_1.35s_ease-in-out_infinite] motion-reduce:animate-none" />
                    </span>
                  ) : null}
                  {/* eslint-disable-next-line @next/next/no-img-element -- Embla transform slides need native eager image loading. */}
                  <img
                    src={image}
                    alt={`${title} ${index + 1}`}
                    draggable={false}
                    loading={index === 0 ? "eager" : "lazy"}
                    fetchPriority={index === 0 ? "high" : "low"}
                    decoding={index === 0 ? "sync" : "async"}
                    onLoad={(event) => handleImageLoad(slideKey, event)}
                    onError={() => {
                      emblaApi?.reInit();
                      setFailedImages((current) => ({ ...current, [slideKey]: true }));
                    }}
                    className={cn(
                      "absolute inset-0 h-full w-full object-contain transition-[opacity,filter,transform] duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]",
                      loaded
                        ? "scale-100 opacity-100 blur-0 motion-safe:animate-[post-detail-image-in_520ms_cubic-bezier(0.16,1,0.3,1)]"
                        : "scale-[1.015] opacity-0 blur-md",
                      failed && "opacity-0",
                    )}
                  />
                  {overflowCount > 0 && index === images.length - 1 ? (
                    <span className="absolute inset-0 flex items-center justify-center bg-black/42 text-2xl font-black text-white backdrop-blur-[1px]">
                      +{overflowCount}
                    </span>
                  ) : null}
                </button>
              </div>
            );
          })}
        </div>
      </div>

      {images.length > 1 ? (
        <>
          <button
            type="button"
            aria-label={previousLabel}
            onClick={() => emblaApi?.scrollPrev()}
            className="absolute left-3 top-1/2 hidden size-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/45 text-white backdrop-blur hover:bg-black/60 md:flex"
          >
            <ChevronLeft className="size-5" />
          </button>
          <button
            type="button"
            aria-label={nextLabel}
            onClick={() => emblaApi?.scrollNext()}
            className="absolute right-3 top-1/2 hidden size-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/45 text-white backdrop-blur hover:bg-black/60 md:flex"
          >
            <ChevronRight className="size-5" />
          </button>
          <div className="absolute bottom-4 left-1/2 rounded-full bg-black/45 px-3 py-1 text-xs font-medium text-white backdrop-blur -translate-x-1/2">
            {counterLabel}
          </div>
        </>
      ) : null}

      <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-black/25 to-transparent md:hidden" />
    </div>
  );
}
