"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState, type CSSProperties, type MouseEvent, type ReactNode } from "react";
import { Heart, LockKeyhole, Play } from "lucide-react";
import { useTranslations } from "next-intl";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { FeedPost } from "@/lib/types";
import { cn } from "@/lib/utils";
import {
  getAuthorInitial,
  getAuthorName,
  getPostImages,
  getPostFeedCover,
  getPostFeedCoverImage,
  isNovelPost,
  isVideoPost,
} from "./feed-utils";
import { getUserHrefFromPost } from "@/lib/users";
import { openPostDetailInstantly } from "./post-detail-instant-events";
import { OriginalIncentiveBadge } from "./original-incentive";

const deferredCoverCallbacks = new Map<Element, () => void>();
let deferredCoverObserver: IntersectionObserver | null = null;

function releaseDeferredCoverObserverIfIdle(observer: IntersectionObserver) {
  if (deferredCoverCallbacks.size === 0 && deferredCoverObserver === observer) {
    observer.disconnect();
    deferredCoverObserver = null;
  }
}

function observeDeferredCover(element: Element, onVisible: () => void) {
  if (!deferredCoverObserver) {
    deferredCoverObserver = new IntersectionObserver(
      (entries, observer) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) {
            continue;
          }

          const callback = deferredCoverCallbacks.get(entry.target);
          if (!callback) {
            continue;
          }

          deferredCoverCallbacks.delete(entry.target);
          observer.unobserve(entry.target);
          callback();
        }

        releaseDeferredCoverObserverIfIdle(observer);
      },
      { rootMargin: "600px 0px" },
    );
  }

  const observer = deferredCoverObserver;
  deferredCoverCallbacks.set(element, onVisible);
  observer.observe(element);

  return () => {
    if (deferredCoverCallbacks.delete(element)) {
      observer.unobserve(element);
    }
    releaseDeferredCoverObserverIfIdle(observer);
  };
}

export function PostCard({
  post,
  index,
  imagePriority = false,
  onLike,
  detailNavigation = "modal",
  theme = "dark",
}: {
  post: FeedPost;
  index: number;
  imagePriority?: boolean;
  transitionScope: string;
  onLike: (post: FeedPost) => void;
  detailNavigation?: "modal" | "page";
  theme?: "dark" | "light";
}) {
  const t = useTranslations("card");
  const cover = getPostFeedCover(post);
  const coverImage = getPostFeedCoverImage(post);
  const author = getAuthorName(post, t("unknownAuthor"));
  const isVideo = isVideoPost(post);
  const [aspectRatio] = useState(
    () => getCoverAspectRatio(coverImage) ?? getFallbackAspectRatio(index),
  );
  const [loadedCover, setLoadedCover] = useState<string | null>(null);
  const [failedCover, setFailedCover] = useState<string | null>(null);
  const [renderCover, setRenderCover] = useState(imagePriority);
  const mediaRef = useRef<HTMLDivElement>(null);
  const authorHref = getUserHrefFromPost(post);
  const postHref = `/post?id=${post.id}`;
  const coverLoaded = Boolean(cover && loadedCover === cover);
  const coverFailed = Boolean(cover && failedCover === cover);
  const isLightTheme = theme === "light";
  const showCoverImage = Boolean(cover && !coverFailed);
  const novelCoverImages = getNovelCoverImages(post, cover);
  const isNovel = showCoverImage && novelCoverImages.length > 0 && isNovelPost(post);

  useEffect(() => {
    if (!cover || renderCover) {
      return;
    }

    const element = mediaRef.current;
    if (!element) {
      return;
    }

    let timerId: number | null = null;
    const stopObserving = observeDeferredCover(element, () => {
      timerId = window.setTimeout(() => {
        setRenderCover(true);
      }, (index % 8) * 32);
    });

    return () => {
      stopObserving();
      if (timerId !== null) {
        window.clearTimeout(timerId);
      }
    };
  }, [cover, index, renderCover]);

  function handleCoverLoad() {
    setLoadedCover(cover);
  }

  function handlePostDetailClick(event: MouseEvent<HTMLAnchorElement>) {
    event.currentTarget.blur();

    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.altKey ||
      event.ctrlKey ||
      event.shiftKey
    ) {
      return;
    }

    if (detailNavigation === "page") {
      event.preventDefault();
      window.location.assign(postHref);
      return;
    }

    event.preventDefault();
    openPostDetailInstantly(post);
  }

  if (isNovel) {
    return (
      <NovelPostCard
        post={post}
        author={author}
        cover={cover}
        coverLoaded={coverLoaded}
        coverImages={novelCoverImages}
        imagePriority={imagePriority}
        onCoverLoad={handleCoverLoad}
        onCoverError={() => setFailedCover(cover)}
        onPostDetailClick={handlePostDetailClick}
        postHref={postHref}
      />
    );
  }

  return (
    <article
      data-feed-post-id={post.id}
      className="break-inside-avoid transition-transform duration-300 ease-out hover:-translate-y-0.5"
    >
      <div
        className={cn(
          "group overflow-hidden rounded-[10px] transition-[background-color,box-shadow,transform] duration-300 ease-out",
          isLightTheme
            ? "bg-white hover:bg-white hover:shadow-[0_12px_28px_rgba(20,20,24,0.12)]"
            : "bg-[#121212] hover:bg-[#171717] hover:shadow-[0_14px_34px_rgba(0,0,0,0.24)]",
        )}
      >
        {showCoverImage ? (
          <Link
            href={postHref}
            prefetch={false}
            scroll={false}
            onClick={handlePostDetailClick}
            aria-label={t("openDetails")}
            className={cn(
              "relative block w-full overflow-hidden rounded-[10px] text-left outline-none transition-[box-shadow,transform] duration-300 ease-out active:scale-[0.985] focus-visible:ring-2 focus-visible:ring-primary",
              isLightTheme ? "bg-[#eeeeef]" : "bg-[#1e1e1e]",
            )}
            style={{ aspectRatio }}
          >
            <div ref={mediaRef} className="absolute inset-0">
              <>
                <div
                  className={cn(
                    "absolute inset-0 overflow-hidden transition-opacity duration-500",
                    isLightTheme ? "bg-[#e6e6e8]" : "bg-[#202024]",
                    coverLoaded ? "opacity-0" : "opacity-100",
                  )}
                >
                  <span
                    className={cn(
                      "absolute inset-y-0 -left-1/2 w-1/2 bg-gradient-to-r from-transparent via-white/10 to-transparent",
                      renderCover &&
                        !coverLoaded &&
                        "animate-[feed-image-shimmer_1.4s_ease-in-out_infinite] motion-reduce:animate-none",
                    )}
                  />
                </div>
                {renderCover ? (
                  <Image
                    src={cover}
                    alt={post.title}
                    fill
                    sizes="(max-width: 640px) calc((100vw - 24px) / 2), (max-width: 960px) calc((100vw - 84px) / 3), (max-width: 1023px) calc((100vw - 120px) / 4), (max-width: 1280px) calc((100vw - 300px) / 4), (max-width: 1600px) calc((100vw - 356px) / 5), 360px"
                    quality={70}
                    loading={imagePriority ? "eager" : "lazy"}
                    fetchPriority={imagePriority ? "high" : "auto"}
                    decoding="async"
                    onError={() => setFailedCover(cover)}
                    onLoad={handleCoverLoad}
                    className={cn(
                      "object-cover transition-[opacity,transform,filter] duration-700 ease-[cubic-bezier(0.16,1,0.3,1)] group-hover:scale-[1.025]",
                      coverLoaded
                        ? "scale-100 opacity-100 blur-0"
                        : "scale-[1.035] opacity-0 blur-md",
                    )}
                  />
                ) : null}
              </>
            </div>

            <div className="absolute inset-x-0 top-0 flex items-start justify-between p-2">
              <div className="flex flex-col items-start gap-1.5">
                <OriginalIncentiveBadge post={post} label={t("originalIncentive")} />
                {post.isPaidContent ? (
                  <span className="inline-flex h-6 items-center gap-1 rounded-full bg-black/55 px-2 text-[11px] font-medium text-white backdrop-blur">
                    <LockKeyhole className="size-3" />
                    {t("paid")}
                  </span>
                ) : null}
              </div>
              {isVideo && (
                <span className="inline-flex size-7 items-center justify-center rounded-full bg-black/50 text-white shadow-[0_8px_18px_rgba(0,0,0,0.24)] backdrop-blur transition-transform duration-300 group-hover:scale-105">
                  <Play className="size-3.5 fill-current" />
                </span>
              )}
            </div>
          </Link>
        ) : null}

        <div className={cn("px-1.5 pb-3", showCoverImage ? "pt-2" : "pt-3")}>
          {!showCoverImage ? (
            <OriginalIncentiveBadge post={post} label={t("originalIncentive")} className="mb-2" />
          ) : null}
          <Link
            href={postHref}
            prefetch={false}
            onClick={handlePostDetailClick}
            className={cn(
              "w-full text-left",
              showCoverImage
                ? "line-clamp-2 min-h-[34px] text-[14px] font-normal leading-[18px]"
                : "line-clamp-4 text-[15px] font-semibold leading-5",
              isLightTheme ? "text-[#25252b]" : "text-[#e0e0e0]",
            )}
          >
            {post.title}
          </Link>
          <div className="mt-2 flex h-6 items-center gap-1.5">
            <Link
              href={authorHref}
              prefetch={false}
              transitionTypes={["nav-forward"]}
              aria-label={t("openAuthor", { name: author })}
              className={cn(
                "flex min-w-0 flex-1 items-center gap-1.5 rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-primary",
                isLightTheme ? "hover:text-[#25252b]" : "hover:text-white",
              )}
            >
              <Avatar className={cn("size-5 border", isLightTheme ? "border-black/10" : "border-white/10")}>
                <AvatarImage
                  src={post.avatar ?? post.user_avatar ?? undefined}
                  alt=""
                  loading="lazy"
                  decoding="async"
                />
                <AvatarFallback
                  className={cn(
                    "text-[10px]",
                    isLightTheme ? "bg-[#eeeeef] text-[#64646d]" : "bg-[#29292e] text-white/70",
                  )}
                >
                  {getAuthorInitial(post, "Y")}
                </AvatarFallback>
              </Avatar>
              <span
                className={cn(
                  "min-w-0 flex-1 truncate text-xs",
                  isLightTheme ? "text-[#64646d]" : "text-[#b0b0b0]",
                )}
              >
                {author}
              </span>
            </Link>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={post.liked ? t("unlike") : t("like")}
                  aria-pressed={post.liked}
                  onClick={() => onLike(post)}
                  className={cn(
                    "size-6 hover:bg-transparent hover:text-primary",
                    isLightTheme ? "text-[#64646d]" : "text-[#b0b0b0]",
                  )}
                >
                  <Heart
                    className={cn(
                      "size-4 transition-[fill,transform,color] duration-200",
                      post.liked && "scale-110 fill-primary text-primary",
                    )}
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("likes", { count: post.like_count })}</TooltipContent>
            </Tooltip>
            <span
              className={cn(
                "min-w-4 text-right text-xs",
                isLightTheme ? "text-[#64646d]" : "text-[#b0b0b0]",
              )}
            >
              {formatCount(post.like_count)}
            </span>
          </div>
        </div>
      </div>
    </article>
  );
}

function NovelPostCard({
  author,
  cover,
  coverImages,
  coverLoaded,
  imagePriority,
  onCoverError,
  onCoverLoad,
  onPostDetailClick,
  post,
  postHref,
}: {
  author: string;
  cover: string | null;
  coverImages: string[];
  coverLoaded: boolean;
  imagePriority: boolean;
  onCoverError: () => void;
  onCoverLoad: () => void;
  onPostDetailClick: (event: MouseEvent<HTMLAnchorElement>) => void;
  post: FeedPost;
  postHref: string;
}) {
  const t = useTranslations("card");
  const excerpt = getNovelExcerpt(post, t("novel.fallbackSummary"));
  const slideCount = Math.max(coverImages.length, 1);
  const coverTrackStyle = {
    "--novel-cover-count": String(slideCount),
    "--novel-cover-duration": Math.max(14, slideCount * 7) + "s",
    "--novel-cover-travel": "-" + (((slideCount - 1) / slideCount) * 100) + "%",
  } as CSSProperties;

  return (
    <article
      data-feed-post-id={post.id}
      className="break-inside-avoid transition-transform duration-300 ease-out hover:-translate-y-0.5"
    >
      <Link
        href={postHref}
        prefetch={false}
        scroll={false}
        onClick={onPostDetailClick}
        aria-label={t("openDetails")}
        className="novel-card-link group relative block overflow-hidden rounded-[18px] bg-[#17120f] text-left shadow-[0_16px_42px_rgba(0,0,0,0.28)] outline-none transition-[box-shadow,transform] duration-300 ease-out active:scale-[0.985] focus-visible:ring-2 focus-visible:ring-primary"
      >
        <div className="absolute inset-0 overflow-hidden">
          <div
            className="novel-cover-track absolute inset-y-0 left-0 flex"
            data-scroll={slideCount > 1 ? "true" : "false"}
            style={coverTrackStyle}
          >
            {coverImages.map((image, imageIndex) => (
              <div key={image + "-" + imageIndex} className="novel-cover-slide relative h-full shrink-0">
                <Image
                  src={image}
                  alt=""
                  fill
                  sizes="(max-width: 640px) calc((100vw - 24px) / 2), (max-width: 960px) calc((100vw - 84px) / 3), (max-width: 1280px) calc((100vw - 300px) / 4), 320px"
                  quality={76}
                  loading={imagePriority && imageIndex === 0 ? "eager" : "lazy"}
                  fetchPriority={imagePriority && imageIndex === 0 ? "high" : "auto"}
                  decoding="async"
                  onError={image === cover ? onCoverError : undefined}
                  onLoad={image === cover ? onCoverLoad : undefined}
                  className={cn(
                    "object-cover transition-[opacity,filter] duration-700",
                    image === cover && !coverLoaded ? "opacity-0 blur-md" : "opacity-100 blur-0",
                  )}
                />
              </div>
            ))}
          </div>

          <div
            className={cn(
              "absolute inset-0 bg-[#211b18] transition-opacity duration-500",
              coverLoaded ? "opacity-0" : "opacity-100",
            )}
          >
            <span
              className={cn(
                "absolute inset-y-0 -left-1/2 w-1/2 bg-gradient-to-r from-transparent via-white/10 to-transparent",
                !coverLoaded && "animate-[feed-image-shimmer_1.4s_ease-in-out_infinite] motion-reduce:animate-none",
              )}
            />
          </div>
        </div>

        <div className="absolute inset-0 bg-gradient-to-b from-black/12 via-[#1a1410]/54 to-[#241b15]/96" />
        <OriginalIncentiveBadge post={post} label={t("originalIncentive")} className="absolute left-4 top-4 z-10" />

        <div className="absolute inset-0 flex flex-col justify-end overflow-hidden px-[clamp(14px,5vw,22px)] py-[clamp(14px,5vw,24px)] text-white md:px-7 md:py-7">
          <NovelAutoScrollContent>
            <div className="novel-content-body">
              {post.isPaidContent ? (
                <span className="mb-3 inline-flex w-fit items-center gap-1 rounded-full bg-black/45 px-2.5 py-1 text-[11px] font-semibold text-white/90 backdrop-blur">
                  <LockKeyhole className="size-3" />
                  {t("paid")}
                </span>
              ) : null}
              <h2 className="line-clamp-2 text-[clamp(18px,4.5vw,24px)] font-black leading-[1.16] tracking-[-0.03em] text-white drop-shadow-[0_2px_8px_rgba(0,0,0,0.45)] md:text-[22px]">
                {post.title}
              </h2>
              <p className="mt-2 line-clamp-1 text-[clamp(11px,3.2vw,14px)] text-white/74">{author}</p>
              <blockquote className="novel-excerpt-block mt-[clamp(12px,4vw,20px)] border-l border-white/24 pl-3 text-[clamp(13px,3.6vw,16px)] leading-[1.65] text-white/72 md:text-[15px]">
                <p>{excerpt}</p>
              </blockquote>
            </div>
          </NovelAutoScrollContent>
          <span className="relative z-10 mx-auto mt-[clamp(14px,5vw,24px)] inline-flex h-[clamp(38px,10vw,48px)] shrink-0 min-w-[clamp(118px,38vw,160px)] items-center justify-center rounded-full border border-white/28 bg-white/[0.03] px-5 text-[clamp(14px,4vw,17px)] font-bold text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.12)] backdrop-blur transition-[background-color,border-color,transform] duration-300 group-hover:border-white/44 group-hover:bg-white/[0.08] group-hover:scale-[1.02] md:h-11 md:min-w-[140px] md:text-[15px]">
            {t("novel.readArticle")}
          </span>
        </div>
      </Link>
    </article>
  );
}

function NovelAutoScrollContent({ children }: { children: ReactNode }) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const [scrollStyle, setScrollStyle] = useState<CSSProperties>({});

  useEffect(() => {
    const viewport = viewportRef.current;
    const body = bodyRef.current;
    if (!viewport || !body) {
      return;
    }

    const viewportElement = viewport;
    const bodyElement = body;

    function updateScroll() {
      const distance = Math.max(0, bodyElement.scrollHeight - viewportElement.clientHeight);
      if (distance < 8) {
        setScrollStyle({});
        return;
      }

      setScrollStyle({
        "--novel-content-travel": `-${distance}px`,
        "--novel-content-duration": `${Math.min(Math.max(distance / 7, 18), 60)}s`,
      } as CSSProperties);
    }

    updateScroll();
    const observer = new ResizeObserver(updateScroll);
    observer.observe(viewportElement);
    observer.observe(bodyElement);

    return () => observer.disconnect();
  }, [children]);

  return (
    <div ref={viewportRef} className="novel-content-scroll">
      <div ref={bodyRef} className="novel-content-track" style={scrollStyle}>
        {children}
      </div>
    </div>
  );
}

function getCoverAspectRatio(image: ReturnType<typeof getPostFeedCoverImage>) {
  if (!image || typeof image === "string") {
    return null;
  }

  return getAspectRatioFromSize(image.width, image.height);
}

function getFallbackAspectRatio(index: number) {
  const ratio = index % 7;

  if (ratio === 0 || ratio === 4) {
    return 3 / 4;
  }

  if (ratio === 1 || ratio === 5) {
    return 4 / 5;
  }

  if (ratio === 2) {
    return 1 / 1.18;
  }

  return 4 / 3;
}

function clampAspectRatio(ratio: number) {
  return Math.min(Math.max(ratio, 3 / 4), 4 / 3);
}

function getAspectRatioFromSize(width: number | undefined, height: number | undefined) {
  if (!width || !height || width <= 0 || height <= 0) {
    return null;
  }

  return clampAspectRatio(width / height);
}

function getNovelCoverImages(post: FeedPost, cover: string | null) {
  return Array.from(new Set([cover, ...getPostImages(post)].filter((image): image is string => Boolean(image)))).slice(0, 4);
}

const NOVEL_COVER_EXCERPT_MAX_LENGTH = 512;

function getNovelExcerpt(post: FeedPost, fallback: string) {
  const content = post.content?.trim();
  if (!content) {
    return fallback;
  }

  const text = content
    .replace(/<[^>]*>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/\s+/g, " ")
    .trim();

  if (!text || text === post.title.trim()) {
    return fallback;
  }

  if (text.length <= NOVEL_COVER_EXCERPT_MAX_LENGTH) {
    return text;
  }

  return text.slice(0, NOVEL_COVER_EXCERPT_MAX_LENGTH).trimEnd() + "...";
}

function formatCount(count: number) {
  if (count >= 10000) {
    return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`;
  }

  if (count >= 1000) {
    return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`;
  }

  return String(count);
}
