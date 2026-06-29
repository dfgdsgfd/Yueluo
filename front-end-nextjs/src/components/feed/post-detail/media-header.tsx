"use client";
import {
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent
} from "react";
import Link from "next/link";
import {
  Bookmark,
  ChevronLeft,
  ChevronRight,
  Copy,
  Heart,
  Loader2,
  Pencil,
  RefreshCw,
  Share2,
  Trash2,
  UserPlus,
  X
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  toast
} from "sonner";
import {
  Avatar,
  AvatarFallback,
  AvatarImage
} from "@/components/ui/avatar";
import {
  Button
} from "@/components/ui/button";
import { MarkdownContent } from "@/components/markdown-content";
import {
  useLocaleContext
} from "@/components/providers/locale-provider";
import {
  cn
} from "@/lib/utils";
import type {
  AuthUser
} from "@/lib/types";
import {
  DetailComment
} from "./post-detail-types";
import {
  getCommentElementId,
  isCurrentUserCommentOwner
} from "./comments";
import {
  formatCollapseReplies,
  formatCount,
  formatDeleteCommentAction,
  formatLikeComment,
  formatRepliesLoading,
  formatReplyAction,
  formatUnlikeComment,
  formatViewReplies
} from "./post-detail-formatters";

export function ImageViewerOverlay({
  closeLabel,
  counterLabel,
  failedLabel,
  images,
  index,
  nextLabel,
  onClose,
  onIndexChange,
  previousLabel,
  title,
}: {
  closeLabel: string;
  counterLabel: string;
  failedLabel: string;
  images: string[];
  index: number;
  nextLabel: string;
  onClose: () => void;
  onIndexChange: (index: number) => void;
  previousLabel: string;
  title: string;
}) {
  const [failedImages, setFailedImages] = useState<Record<string, true>>({});
  const pointerStartXRef = useRef<number | null>(null);
  const pointerMovedRef = useRef(false);
  const total = images.length;
  const canGoPrevious = index > 0;
  const canGoNext = index < total - 1;

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
        return;
      }

      if (event.key === "ArrowLeft" && canGoPrevious) {
        onIndexChange(index - 1);
        return;
      }

      if (event.key === "ArrowRight" && canGoNext) {
        onIndexChange(index + 1);
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [canGoNext, canGoPrevious, index, onClose, onIndexChange]);

  function handlePointerDown(event: ReactPointerEvent<HTMLDivElement>) {
    pointerStartXRef.current = event.clientX;
    pointerMovedRef.current = false;
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    const startX = pointerStartXRef.current;
    if (startX === null) {
      return;
    }

    if (Math.abs(event.clientX - startX) > 8) {
      pointerMovedRef.current = true;
    }
  }

  function handlePointerUp(event: ReactPointerEvent<HTMLDivElement>) {
    const startX = pointerStartXRef.current;
    pointerStartXRef.current = null;

    if (startX === null) {
      return;
    }

    const distance = event.clientX - startX;
    if (Math.abs(distance) < 52) {
      return;
    }

    if (distance > 0 && canGoPrevious) {
      onIndexChange(index - 1);
    } else if (distance < 0 && canGoNext) {
      onIndexChange(index + 1);
    }
  }

  if (total === 0) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-[9999] flex overflow-hidden bg-black/70 text-white backdrop-blur-[14px]"
      role="dialog"
      aria-modal="true"
      aria-label={title}
      onClick={(event) => event.stopPropagation()}
      onPointerDown={(event) => event.stopPropagation()}
    >
      <button
        type="button"
        aria-label={closeLabel}
        onClick={onClose}
        className="absolute left-2 top-2 z-[3001] flex size-10 items-center justify-center rounded-full bg-[#989898]/50 text-white backdrop-blur transition-colors hover:bg-[#989898]/65"
      >
        <X className="size-5" />
      </button>

      <div className="absolute right-4 top-4 z-[3001] rounded-full bg-[#989898]/50 px-3 py-1.5 text-sm font-medium leading-none text-white backdrop-blur">
        {counterLabel}
      </div>

      {total > 1 ? (
        <>
          <button
            type="button"
            aria-label={previousLabel}
            disabled={!canGoPrevious}
            onClick={() => canGoPrevious && onIndexChange(index - 1)}
            className={cn(
              "absolute left-2.5 top-1/2 z-[3001] flex size-10 -translate-y-1/2 items-center justify-center rounded-full bg-[#989898]/50 text-white backdrop-blur transition-opacity",
              canGoPrevious ? "opacity-80 hover:opacity-100" : "cursor-default opacity-30",
            )}
          >
            <ChevronLeft className="size-6" />
          </button>
          <button
            type="button"
            aria-label={nextLabel}
            disabled={!canGoNext}
            onClick={() => canGoNext && onIndexChange(index + 1)}
            className={cn(
              "absolute right-2.5 top-1/2 z-[3001] flex size-10 -translate-y-1/2 items-center justify-center rounded-full bg-[#989898]/50 text-white backdrop-blur transition-opacity",
              canGoNext ? "opacity-80 hover:opacity-100" : "cursor-default opacity-30",
            )}
          >
            <ChevronRight className="size-6" />
          </button>
        </>
      ) : null}

      <div
        className="flex h-full w-full touch-pan-y transition-transform duration-300 ease-out"
        style={{ transform: `translateX(-${index * 100}%)` }}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={() => {
          pointerStartXRef.current = null;
        }}
      >
        {images.map((image, imageIndex) => {
          const imageKey = `${image}-${imageIndex}`;
          const failed = Boolean(failedImages[imageKey]);

          return (
            <div key={imageKey} className="relative h-full min-w-full">
              {failed ? (
                <button
                  type="button"
                  onClick={onClose}
                  className="absolute inset-0 flex cursor-zoom-out items-center justify-center px-8 text-center text-sm text-white/70"
                >
                  {failedLabel}
                </button>
              ) : null}
              {/* eslint-disable-next-line @next/next/no-img-element -- Full-screen viewer needs native image sizing and direct click-to-close behavior. */}
              <img
                src={image}
                alt={`${title} ${imageIndex + 1}`}
                draggable={false}
                loading={imageIndex === index ? "eager" : "lazy"}
                fetchPriority={imageIndex === index ? "high" : "low"}
                decoding={imageIndex === index ? "sync" : "async"}
                onClick={() => {
                  if (!pointerMovedRef.current) {
                    onClose();
                  }
                }}
                onError={() => {
                  setFailedImages((current) => ({ ...current, [imageKey]: true }));
                }}
                className={cn(
                  "h-full w-full cursor-zoom-out object-contain",
                  failed && "opacity-0",
                )}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}


export function ShareSheet({
  onClose,
  onCopy,
  onSave,
}: {
  copyLabel: string;
  onClose: () => void;
  onCopy: () => void;
  onSave: () => void;
}) {
  const actions = [
    { label: "邀请好友", icon: <UserPlus className="size-7" />, onClick: noopShareAction },
    { label: "复制链接", icon: <Copy className="size-7" />, onClick: onCopy },
    { label: "保存", icon: <Bookmark className="size-7" />, onClick: onSave },
  ];

  return (
    <div
      className="fixed inset-0 z-[4000] flex items-end justify-center bg-black/45 text-white md:items-center"
      role="dialog"
      aria-modal="true"
      aria-label="分享至"
      onClick={onClose}
    >
      <div
        className="flex max-h-[min(72dvh,520px)] w-full max-w-[430px] flex-col overflow-hidden rounded-t-[18px] bg-[#101014] pb-[calc(0.75rem+env(safe-area-inset-bottom))] shadow-[0_-20px_70px_rgba(0,0,0,0.55)] md:rounded-[18px]"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="grid h-12 shrink-0 grid-cols-[2.25rem_minmax(0,1fr)_2.25rem] items-center gap-2 border-b border-white/[0.07] px-3">
          <span aria-hidden="true" />
          <h2 className="min-w-0 truncate text-center text-[17px] font-semibold">分享至</h2>
          <button
            type="button"
            aria-label="关闭"
            onClick={onClose}
            className="flex size-9 items-center justify-center rounded-full text-white/75 active:bg-white/[0.08]"
          >
            <X className="size-5" />
          </button>
        </div>

        <div className="min-h-0 overflow-y-auto">
          <div className="overflow-x-auto overscroll-x-contain px-4 py-3 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            <div className="flex min-w-max gap-4">
            {actions.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={item.onClick}
                className="flex w-[58px] shrink-0 flex-col items-center gap-2 text-white/68"
              >
                <span className="flex size-12 items-center justify-center rounded-full bg-white/[0.06] text-white/85">
                  {item.icon}
                </span>
                <span className="max-w-full truncate text-center text-xs leading-4">{item.label}</span>
              </button>
            ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}


export function noopShareAction() {
  toast.info("Coming soon.");
}


export function VideoPlayerSkeleton() {
  return (
    <div
      className="flex aspect-video h-full min-h-full items-center justify-center rounded-none bg-black text-sm font-medium text-white/62 md:aspect-auto md:rounded-l-[20px] md:rounded-r-none"
      aria-busy="true"
    >
      <span className="inline-flex items-center rounded-full border border-white/10 bg-white/[0.08] px-4 py-2">
        Loading video...
      </span>
    </div>
  );
}


export function MobileDetailHeader({
  author,
  avatar,
  deleting,
  disabled,
  fallback,
  followLabel,
  following,
  href,
  owner,
  onClose,
  onDelete,
  onEdit,
  onFollowToggle,
  onNavigate,
  onShare,
}: {
  author: string;
  avatar?: string;
  deleting?: boolean;
  disabled?: boolean;
  fallback: string;
  followLabel: string;
  following: boolean;
  href: string;
  owner?: boolean;
  onClose: () => void;
  onDelete?: () => void;
  onEdit?: () => void;
  onFollowToggle: () => void;
  onNavigate: () => void;
  onShare: () => void;
}) {
  const t = useTranslations();

  return (
    <div className="sticky top-0 z-30 flex h-[72px] shrink-0 items-center gap-3 border-b border-white/[0.08] bg-[#121212] px-4 md:hidden">
      <Button
        size="icon"
        variant="ghost"
        aria-label={t("drawer.close")}
        onClick={onClose}
        className="size-10 text-white/82 hover:bg-white/[0.06]"
      >
        <X className="size-5" />
      </Button>
      <Link
        href={href}
        onClick={onNavigate}
        aria-label={t("card.openAuthor", { name: author })}
        className="flex min-w-0 flex-1 items-center gap-3 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary"
      >
        <Avatar className="size-9">
          <AvatarImage src={avatar} alt="" />
          <AvatarFallback>{fallback}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-white">{author}</p>
          <p className="truncate text-xs text-white/68">{t("drawer.authorSubtitle")}</p>
        </div>
      </Link>
      {owner ? (
        <div className="flex shrink-0 items-center gap-1">
          <Button type="button" size="icon" variant="ghost" aria-label="编辑" onClick={onEdit} className="size-9 text-white/72 hover:bg-white/[0.06]">
            <Pencil className="size-4" />
          </Button>
          <Button type="button" size="icon" variant="ghost" aria-label="删除" disabled={deleting} onClick={onDelete} className="size-9 text-white/72 hover:bg-white/[0.06] hover:text-[#ef4444] disabled:opacity-50">
            <Trash2 className="size-4" />
          </Button>
        </div>
      ) : (
        <FollowButton
          disabled={disabled}
          following={following}
          label={followLabel}
          onClick={onFollowToggle}
          className="h-8 px-4 text-sm font-semibold"
        />
      )}
      <Button
        type="button"
        size="icon"
        variant="ghost"
        aria-label={t("drawer.share")}
        onClick={onShare}
        className="size-10 shrink-0 text-white/72 hover:bg-white/[0.06]"
      >
        <Share2 className="size-6" />
      </Button>
    </div>
  );
}


export function FollowButton({
  className,
  disabled,
  following,
  label,
  onClick,
}: {
  className?: string;
  disabled?: boolean;
  following: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      onClick={onClick}
      disabled={disabled}
      variant={following ? "outline" : "default"}
      className={cn(
        "bg-primary text-white hover:bg-primary/90",
        following && "border-white/10 bg-white/[0.06] text-white hover:bg-white/[0.1]",
        className ?? "h-9 px-5 text-sm font-semibold",
      )}
    >
      {label}
    </Button>
  );
}


export function AuthorBlock({
  author,
  avatar,
  fallback,
  href,
  onNavigate,
  subtitle,
}: {
  author: string;
  avatar?: string;
  fallback: string;
  href: string;
  onNavigate: () => void;
  subtitle?: string;
}) {
  const t = useTranslations();

  return (
    <Link
      href={href}
      onClick={onNavigate}
      aria-label={t("card.openAuthor", { name: author })}
      className="flex min-w-0 items-center gap-3 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      <Avatar className="size-10">
        <AvatarImage src={avatar} alt="" />
        <AvatarFallback className="bg-[#29292e] text-white/75">{fallback}</AvatarFallback>
      </Avatar>
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold text-white">{author}</p>
        {subtitle ? <p className="truncate text-xs text-white/68">{subtitle}</p> : null}
      </div>
    </Link>
  );
}


export function CommentThread({
  comment,
  currentUser,
  deletingCommentIds,
  depth = 0,
  highlightedCommentId,
  likingCommentIds,
  onDelete,
  onLoadReplies,
  onLike,
  onNavigate,
  onReply,
  onRetry,
}: {
  comment: DetailComment;
  currentUser: AuthUser | null;
  deletingCommentIds: Set<string>;
  depth?: number;
  highlightedCommentId?: string | null;
  likingCommentIds: Set<string>;
  onDelete: (comment: DetailComment) => void;
  onLoadReplies: (comment: DetailComment) => void;
  onLike: (comment: DetailComment) => void;
  onNavigate: () => void;
  onReply: (comment: DetailComment) => void;
  onRetry: (comment: DetailComment) => void;
}) {
  const t = useTranslations();
  const { locale } = useLocaleContext();
  const replies = comment.replies ?? [];
  const repliesExpanded = comment.repliesExpanded ?? false;
  const hasVisibleReplies = replies.length > 0;
  const hasMoreReplies = comment.replyCount > replies.length;
  const canLoadReplies = comment.replyCount > 0 && !hasVisibleReplies;
  const loadingReplies = comment.repliesStatus === "loading";
  const commentId = String(comment.id);
  const highlighted = highlightedCommentId === commentId;
  const canDelete = currentUser ? isCurrentUserCommentOwner(comment, currentUser) : false;
  const isDeleting = deletingCommentIds.has(commentId);
  const isLiking = likingCommentIds.has(commentId);
  const isPending = comment.status === "pending";
  const isFailed = comment.status === "failed";

  return (
    <div
      id={getCommentElementId(comment.id)}
      data-comment-id={commentId}
      className={cn(
        "scroll-mt-24 rounded-lg transition-[background-color,box-shadow] duration-500",
        depth > 0 && "pl-5",
        highlighted && "bg-[#ff7ba6]/10 shadow-[0_0_0_1px_rgba(255,123,166,0.35)]",
      )}
    >
      <div className="flex gap-3">
        <Link
          href={`/user/${encodeURIComponent(comment.userId)}`}
          onClick={onNavigate}
          aria-label={t("card.openAuthor", { name: comment.author })}
          className="size-9 shrink-0 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-primary"
        >
          <Avatar className="size-9">
            <AvatarImage src={comment.avatar ?? undefined} alt="" />
            <AvatarFallback className="bg-[#29292e] text-xs text-white/72">
              {comment.author.charAt(0)}
            </AvatarFallback>
          </Avatar>
        </Link>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <Link
                href={`/user/${encodeURIComponent(comment.userId)}`}
                onClick={onNavigate}
                className="block truncate text-sm font-medium text-white/66 outline-none hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
              >
                {comment.author}
              </Link>
              <MarkdownContent
                className="markdown-content-compact mt-1 text-sm leading-6 text-white/88"
                content={comment.body}
              />
            </div>
            <button
              type="button"
              disabled={isLiking || isPending || isFailed}
              onClick={() => onLike(comment)}
              className={cn(
                "flex shrink-0 items-center gap-1 rounded-full px-1 text-xs text-white/68 transition-colors hover:text-primary disabled:cursor-wait disabled:opacity-55",
                comment.liked && "text-primary",
              )}
              aria-label={comment.liked ? formatUnlikeComment(locale) : formatLikeComment(locale)}
            >
              <Heart className={cn("size-3.5", comment.liked && "fill-primary")} />
              {formatCount(comment.likes)}
            </button>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-white/68">
            <span>{comment.meta}</span>
            {isPending ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
            ) : isFailed ? (
              <button
                type="button"
                onClick={() => onRetry(comment)}
                className="inline-flex items-center gap-1 font-medium text-red-300 hover:text-red-200"
              >
                <RefreshCw className="size-3.5" />
                {t("feed.retry")}
              </button>
            ) : (
              <button
                type="button"
                onClick={() => onReply(comment)}
                className="font-medium text-white/68 hover:text-primary"
              >
                {formatReplyAction(locale)}
              </button>
            )}
            {canDelete && !isPending ? (
              <button
                type="button"
                disabled={isDeleting}
                onClick={() => onDelete(comment)}
                className="inline-flex items-center gap-1 font-medium text-white/68 hover:text-red-300 disabled:cursor-wait disabled:text-white/40"
              >
                <Trash2 className="size-3.5" />
                {formatDeleteCommentAction(locale)}
              </button>
            ) : null}
          </div>

          {canLoadReplies || hasMoreReplies || loadingReplies ? (
            <button
              type="button"
              disabled={loadingReplies}
              onClick={() => onLoadReplies(comment)}
              className="mt-2 text-xs font-medium text-white/68 hover:text-primary disabled:cursor-wait disabled:text-white/40"
            >
              {loadingReplies
                ? formatRepliesLoading(locale)
                : formatViewReplies(Math.max(comment.replyCount - replies.length, 0), locale)}
            </button>
          ) : hasVisibleReplies ? (
            <button
              type="button"
              onClick={() => onLoadReplies(comment)}
              className="mt-2 text-xs font-medium text-white/68 hover:text-primary"
            >
              {repliesExpanded
                ? formatCollapseReplies(locale)
                : formatViewReplies(replies.length, locale)}
            </button>
          ) : null}

          {hasVisibleReplies && repliesExpanded ? (
            <div className="mt-4 space-y-4 border-l border-white/[0.08]">
              {replies.map((reply) => (
                <CommentThread
                  key={reply.id}
                  comment={reply}
                  currentUser={currentUser}
                  deletingCommentIds={deletingCommentIds}
                  depth={depth + 1}
                  highlightedCommentId={highlightedCommentId}
                  likingCommentIds={likingCommentIds}
                  onDelete={onDelete}
                  onLoadReplies={onLoadReplies}
                  onLike={onLike}
                  onNavigate={onNavigate}
                  onReply={onReply}
                  onRetry={onRetry}
                />
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
