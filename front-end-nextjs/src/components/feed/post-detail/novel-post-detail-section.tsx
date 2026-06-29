"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Bookmark,BookOpen,ChevronLeft,Heart,MessageCircle,Pencil,Search,Share2,Trash2 } from "lucide-react";
import { useRef,useState,type ReactNode } from "react";
import { Drawer } from "vaul";
import { getAuthorInitial } from "../feed-utils";
import { formatOriginalIncentiveAmount,OriginalIncentiveReward } from "../original-incentive";
import { CommentSkeletonList } from "./comments";
import { AuthorBlock,CommentThread,FollowButton } from "./media-header";
import { LazyPostMarkdownContent as PostMarkdownContent } from "./lazy-post-markdown-content";
import { getNovelDetailContent,getNovelReadingMinutes } from "./novel-content";
import type { PostDetailSectionContentProps } from "./post-detail-section-content-types";
import { PostReaderToolbar,usePostReader } from "./post-reader";

type NovelPostDetailSectionProps = Pick<
  PostDetailSectionContentProps,
  | "author"
  | "authorHref"
  | "commentCount"
  | "commentInputExpanded"
  | "commentInputRef"
  | "commentPlaceholder"
  | "commentState"
  | "commentText"
  | "comments"
  | "currentUser"
  | "detailDate"
  | "followButtonDisabled"
  | "followButtonLabel"
  | "handleCancelCommentInput"
  | "handleCopyPostLink"
  | "handleDeletePost"
  | "handleDeleteComment"
  | "handleEditPost"
  | "handleFollowToggle"
  | "handleLoadCommentReplies"
  | "handleNavigateAway"
  | "handleOpenCommentInput"
  | "handleSubmitComment"
  | "handleRetryComment"
  | "handleToggleCommentLike"
  | "highlightedCommentId"
  | "isDeletingPost"
  | "isFollowingAuthor"
  | "isLoadingComments"
  | "isPostOwner"
  | "isSubmittingComment"
  | "locale"
  | "mode"
  | "mutatingCommentDeleteIds"
  | "mutatingCommentLikeIds"
  | "onCollect"
  | "onLike"
  | "onOpenChange"
  | "post"
  | "rawDetailContent"
  | "setCommentDraft"
  | "setIsCommentInputFocused"
  | "t"
>;

function NovelFooterAction({
  active,
  ariaLabel,
  count,
  icon,
  onClick,
}: {
  active?: boolean;
  ariaLabel: string;
  count?: number;
  icon: ReactNode;
  onClick?: () => void;
}) {
  const [popped, setPopped] = useState(false);

  function handleClick() {
    setPopped(true);
    setTimeout(() => setPopped(false), 300);
    onClick?.();
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      aria-label={ariaLabel}
      className={cn(
        "flex min-w-[44px] flex-col items-center justify-center gap-0.5 rounded-full text-white/92 transition-transform duration-150",
        active && "text-primary",
        popped && "scale-125",
      )}
    >
      {icon}
      {count !== undefined ? (
        <span className="text-sm font-semibold leading-none">{formatCompactCount(count)}</span>
      ) : null}
    </button>
  );
}
function formatCompactCount(count: number) {
  if (count >= 10000) {
    return (count / 10000).toFixed(count >= 100000 ? 0 : 1) + "w";
  }

  if (count >= 1000) {
    return (count / 1000).toFixed(count >= 10000 ? 0 : 1) + "k";
  }

  return String(count);
}

export function NovelPostDetailSection({
  author,
  authorHref,
  commentCount,
  commentInputExpanded,
  commentInputRef,
  commentPlaceholder,
  commentState,
  commentText,
  comments,
  currentUser,
  detailDate,
  followButtonDisabled,
  followButtonLabel,
  handleCancelCommentInput,
  handleCopyPostLink,
  handleDeleteComment,
  handleDeletePost,
  handleEditPost,
  handleFollowToggle,
  handleLoadCommentReplies,
  handleNavigateAway,
  handleOpenCommentInput,
  handleRetryComment,
  handleSubmitComment,
  handleToggleCommentLike,
  highlightedCommentId,
  isDeletingPost,
  isFollowingAuthor,
  isLoadingComments,
  isPostOwner,
  isSubmittingComment,
  locale,
  mode,
  mutatingCommentDeleteIds,
  mutatingCommentLikeIds,
  onCollect,
  onLike,
  onOpenChange,
  post,
  rawDetailContent,
  setCommentDraft,
  setIsCommentInputFocused,
  t,
}: NovelPostDetailSectionProps) {
  const novelContent = getNovelDetailContent(rawDetailContent);
  const readingMinutes = getNovelReadingMinutes(rawDetailContent);
  const readTime = t("drawer.novel.readTime", { minutes: readingMinutes });

  const [searchOpen, setSearchOpen] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState("");
  const contentRef = useRef<HTMLElement | null>(null);
  const readerScrollRef = useRef<HTMLDivElement | null>(null);
  const reader = usePostReader({
    content: rawDetailContent,
    postId: String(post.id),
    scrollRef: readerScrollRef,
    autoEnter: false,
  });

  function handleLike() {
    onLike?.(post);
  }

  function handleCollect() {
    onCollect?.(post);
  }

  function runSearch(keyword = searchKeyword) {
    const query = keyword.trim();
    if (!query) {
      setSearchOpen(true);
      return;
    }

    const selection = window.getSelection();
    selection?.removeAllRanges();
    contentRef.current?.focus({ preventScroll: true });
    const findInPage = "find" in window ? window.find : undefined;
    if (typeof findInPage === "function") {
      findInPage.call(window, query, false, false, true, false, false, false);
    }
  }

  return (
    <section className="relative flex h-dvh w-full min-w-0 flex-1 flex-col overflow-hidden bg-[#2b2119] text-[#f2eee8]">
      <header className="sticky top-0 z-20 flex h-[88px] shrink-0 items-center gap-3 bg-[#2b2119]/96 px-5 pt-[env(safe-area-inset-top)] backdrop-blur md:h-[82px] md:px-7 md:pt-0">
        <Button
          type="button"
          size="icon"
          variant="ghost"
          aria-label={t("drawer.close")}
          onClick={() => onOpenChange(false)}
          className="-ml-2 size-10 rounded-full text-white hover:bg-white/[0.08]"
        >
          <ChevronLeft className="size-8" />
        </Button>
        <AuthorBlock
          author={author}
          avatar={post.avatar ?? post.user_avatar ?? undefined}
          fallback={getAuthorInitial(post, "Y")}
          href={authorHref}
          onNavigate={handleNavigateAway}
        />
        <div className="ml-auto flex shrink-0 items-center gap-2">
          {isPostOwner ? (
            <>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={t("drawer.edit")}
                onClick={handleEditPost}
                className="size-10 rounded-full text-white/84 hover:bg-white/[0.08] hover:text-primary"
              >
                <Pencil className="size-4" />
              </Button>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={t("drawer.delete")}
                disabled={isDeletingPost}
                onClick={() => void handleDeletePost()}
                className="size-10 rounded-full text-white/84 hover:bg-white/[0.08] hover:text-[#ef4444] disabled:opacity-50"
              >
                <Trash2 className="size-4" />
              </Button>
            </>
          ) : (
            <FollowButton
              disabled={followButtonDisabled}
              following={isFollowingAuthor}
              label={followButtonLabel}
              onClick={handleFollowToggle}
              className="h-10 rounded-[9px] border border-white/72 bg-transparent px-5 text-base font-semibold text-white hover:bg-white/[0.08]"
            />
          )}
          <form
            onSubmit={(event) => {
              event.preventDefault();
              runSearch();
            }}
            className={cn(
              "flex min-w-0 items-center gap-2 overflow-hidden transition-[width,opacity] duration-200",
              searchOpen ? "w-[min(220px,34vw)] opacity-100" : "w-10 opacity-100",
            )}
          >
            {searchOpen ? (
              <input
                autoFocus
                value={searchKeyword}
                onChange={(event) => setSearchKeyword(event.target.value)}
                placeholder={t("drawer.search")}
                className="h-10 min-w-0 flex-1 rounded-full border border-white/12 bg-white/[0.08] px-3 text-sm text-white outline-none placeholder:text-white/64"
              />
            ) : null}
            <Button
              type={searchOpen ? "submit" : "button"}
              size="icon"
              variant="ghost"
              aria-label={t("drawer.search")}
              onClick={() => {
                if (!searchOpen) {
                  setSearchOpen(true);
                  return;
                }
                runSearch();
              }}
              className="size-10 shrink-0 rounded-full text-white hover:bg-white/[0.08]"
            >
              <Search className="size-8" />
            </Button>
          </form>
        </div>
      </header>

      <div
        ref={readerScrollRef}
        className={cn(
          "min-h-0 flex-1 overflow-y-auto overscroll-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
          !reader.reading && "md:grid md:grid-cols-[minmax(0,1fr)_minmax(300px,360px)] md:overflow-hidden",
        )}
      >
        <main
          ref={contentRef}
          tabIndex={-1}
          style={reader.reading ? reader.readerStyle : undefined}
          className={cn(
            "min-h-0 w-full px-[clamp(28px,7vw,52px)] pb-6 pt-6 outline-none md:px-[clamp(42px,5vw,68px)] md:pb-8 md:[scrollbar-width:none] md:[&::-webkit-scrollbar]:hidden",
            reader.reading
              ? "mx-auto max-w-[880px] pb-[calc(2rem+env(safe-area-inset-bottom))] md:overflow-visible md:px-10 md:pb-12"
              : "md:overflow-y-auto",
            reader.reading && "post-reader-content",
          )}
        >
          {reader.reading ? (
            <PostReaderToolbar
              headings={reader.headings}
              onExit={() => reader.setReading(false)}
              onFontIndexChange={reader.setFontIndex}
              onLineIndexChange={reader.setLineIndex}
              preferences={reader.preferences}
              progress={reader.progress}
            />
          ) : null}
          <p className="flex items-center gap-2 text-[clamp(15px,4vw,18px)] text-white/76">
            <span>{readTime}</span>
            <span className="text-white/68">·</span>
            <span>{detailDate}</span>
          </p>
          {!reader.reading && reader.canRead ? (
            <Button
              type="button"
              onClick={() => reader.setReading(true)}
              className="mt-6 h-10 rounded-full border border-white/12 bg-white/[0.08] px-4 text-sm font-semibold text-white hover:bg-white/[0.12]"
            >
              <BookOpen className="size-4" />
              <span>{t("drawer.reader.enter")}</span>
            </Button>
          ) : null}
          <OriginalIncentiveReward
            post={post}
            title={t("drawer.originalIncentiveTitle")}
            amountLabel={t("drawer.originalIncentiveAmount", { amount: formatOriginalIncentiveAmount(post, locale) })}
            className="mt-6"
          />
          {mode === "page" ? (
            <h1 className="mt-8 text-[clamp(32px,6vw,48px)] font-black leading-[1.36] tracking-[-0.035em] text-white">
              {post.title}
            </h1>
          ) : (
            <Drawer.Title className="mt-8 text-[clamp(32px,6vw,48px)] font-black leading-[1.36] tracking-[-0.035em] text-white">
              {post.title}
            </Drawer.Title>
          )}
          <p className="mt-7 text-[clamp(20px,4vw,28px)] leading-relaxed text-white/74">
            {t("drawer.novel.subtitle")}
          </p>
          <div
            className={cn(
              reader.reading
                ? "novel-reader-body mt-8"
                : "mt-9 border-l border-white/12 pl-5 text-[clamp(22px,4.8vw,34px)] leading-[2.18] text-white/74",
            )}
          >
            {novelContent.kind === "plain" ? (
              <p className="whitespace-pre-wrap break-words">{novelContent.content}</p>
            ) : novelContent.kind === "markdown" ? (
              <PostMarkdownContent
                className="novel-reading-content"
                content={novelContent.content}
                onNavigateAway={handleNavigateAway}
              />
            ) : (
              <p>{t("card.novel.fallbackSummary")}</p>
            )}
          </div>
        </main>

        {!reader.reading ? (
          <>
            <aside className="hidden min-h-0 overflow-y-auto border-l border-white/[0.08] px-5 pb-[104px] pt-6 [scrollbar-width:none] md:block [&::-webkit-scrollbar]:hidden">
              <section>
                <h2 className="text-base font-semibold text-white/88">
                  {t("drawer.commentCount", { count: commentCount })}
                </h2>
                <div className="mt-5 space-y-5">
                  {isLoadingComments ? (
                    <CommentSkeletonList />
                  ) : comments.length > 0 ? (
                    <>
                      {comments.map((comment) => (
                        <CommentThread
                          key={comment.id}
                          comment={comment}
                          currentUser={currentUser}
                          deletingCommentIds={mutatingCommentDeleteIds}
                          highlightedCommentId={highlightedCommentId}
                          likingCommentIds={mutatingCommentLikeIds}
                          onDelete={handleDeleteComment}
                          onLoadReplies={handleLoadCommentReplies}
                          onLike={handleToggleCommentLike}
                          onNavigate={handleNavigateAway}
                          onReply={handleOpenCommentInput}
                          onRetry={handleRetryComment}
                        />
                      ))}
                      {commentState.status === "loaded" ? (
                        <p className="py-2 text-center text-sm text-white/68">
                          {t("drawer.noMoreComments")}
                        </p>
                      ) : null}
                    </>
                  ) : (
                    <p className="py-2 text-center text-sm text-white/68">
                      {t("drawer.noComments")}
                    </p>
                  )}
                </div>
              </section>
            </aside>

            <section className="border-t border-white/[0.08] px-[clamp(28px,7vw,52px)] pb-[calc(112px+env(safe-area-inset-bottom))] pt-6 md:hidden">
              <h2 className="text-base font-semibold text-white/88">
                {t("drawer.commentCount", { count: commentCount })}
              </h2>
              <div className="mt-5 space-y-5">
                {isLoadingComments ? (
                  <CommentSkeletonList />
                ) : comments.length > 0 ? (
                  <>
                    {comments.map((comment) => (
                      <CommentThread
                        key={comment.id}
                        comment={comment}
                        currentUser={currentUser}
                        deletingCommentIds={mutatingCommentDeleteIds}
                        highlightedCommentId={highlightedCommentId}
                        likingCommentIds={mutatingCommentLikeIds}
                        onDelete={handleDeleteComment}
                        onLoadReplies={handleLoadCommentReplies}
                        onLike={handleToggleCommentLike}
                        onNavigate={handleNavigateAway}
                        onReply={handleOpenCommentInput}
                        onRetry={handleRetryComment}
                      />
                    ))}
                    {commentState.status === "loaded" ? (
                      <p className="py-2 text-center text-sm text-white/68">
                        {t("drawer.noMoreComments")}
                      </p>
                    ) : null}
                  </>
                ) : (
                  <p className="py-2 text-center text-sm text-white/68">
                    {t("drawer.noComments")}
                  </p>
                )}
              </div>
            </section>
          </>
        ) : null}
      </div>

      {!reader.reading ? (
        <footer
          className={cn(
            "post-comment-footer bottom-0 z-30 shrink-0 border-t border-white/[0.04] bg-[#2b2119]/96 px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 text-white backdrop-blur md:px-5",
            mode === "page" ? "sticky" : "fixed inset-x-0 md:absolute md:inset-x-0 md:bottom-0",
          )}
        >
          <div className="relative mx-auto flex h-12 max-w-[760px] items-center gap-3">
            <form
              onSubmit={handleSubmitComment}
              className={cn(
                "min-w-0 flex-1 transition-[max-width,opacity] duration-300",
                commentInputExpanded ? "max-w-full" : "max-w-[calc(100%_-_180px)]",
              )}
            >
              <textarea
                ref={commentInputRef}
                value={commentText}
                onChange={(event) =>
                  setCommentDraft((currentDraft) => ({
                    postId: post.id,
                    replyTarget: currentDraft.postId === post.id ? (currentDraft.replyTarget ?? null) : null,
                    text: event.target.value,
                  }))
                }
                onPointerDown={(event) => {
                  if (!commentInputExpanded) {
                    event.preventDefault();
                    handleOpenCommentInput(undefined, { immediateFocus: true });
                  }
                }}
                onClick={(event) => {
                  if (!commentInputExpanded) {
                    event.preventDefault();
                    handleOpenCommentInput(undefined, { immediateFocus: true });
                  }
                }}
                onFocus={() => setIsCommentInputFocused(true)}
                onBlur={() => {
                  if (!commentText.trim()) {
                    setIsCommentInputFocused(false);
                  }
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
                    event.preventDefault();
                    event.currentTarget.form?.requestSubmit();
                  }
                }}
                disabled={isSubmittingComment}
                aria-label={t("drawer.saySomething")}
                placeholder={commentPlaceholder}
                rows={1}
                className="h-12 max-h-24 min-h-10 w-full resize-none overflow-y-auto rounded-full border border-white/[0.04] bg-white/[0.08] px-5 py-3 text-base leading-6 text-white outline-none placeholder:text-white/64 focus:border-white/22 focus:ring-2 focus:ring-white/10"
              />
            </form>
            <div
              className={cn(
                "flex h-12 shrink-0 items-center gap-3 transition-[opacity,transform] duration-300",
                commentInputExpanded
                  ? "pointer-events-none absolute right-0 translate-x-10 opacity-0"
                  : "relative opacity-100",
              )}
            >
              <NovelFooterAction
                active={post.liked}
                ariaLabel={post.liked ? t("card.unlike") : t("card.like")}
                count={post.like_count}
                icon={<Heart className={cn("size-8", post.liked && "fill-primary text-primary")} />}
                onClick={handleLike}
              />
              <NovelFooterAction
                ariaLabel={t("drawer.comments", { count: commentCount })}
                count={commentCount}
                icon={<MessageCircle className="size-8" />}
                onClick={() => handleOpenCommentInput()}
              />
              <NovelFooterAction
                active={post.collected}
                ariaLabel={t("drawer.collections", { count: post.collect_count ?? 0 })}
                count={post.collect_count ?? 0}
                icon={<Bookmark className={cn("size-8", post.collected && "fill-primary text-primary")} />}
                onClick={handleCollect}
              />
              <NovelFooterAction
                ariaLabel={t("drawer.share")}
                icon={<Share2 className="size-8" />}
                onClick={() => void handleCopyPostLink()}
              />
            </div>
          </div>
          {commentInputExpanded ? (
            <div className="mx-auto mt-3 flex max-w-[760px] items-center justify-end gap-3">
              <button
                type="button"
                onMouseDown={(event) => event.preventDefault()}
                onClick={handleCancelCommentInput}
                className="h-10 rounded-full border border-white/12 px-5 text-sm font-semibold text-white/68"
              >
                {t("drawer.cancel")}
              </button>
              <button
                type="button"
                disabled={!commentText.trim() || isSubmittingComment}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => commentInputRef.current?.form?.requestSubmit()}
                className="h-10 rounded-full bg-white px-5 text-sm font-semibold text-[#2b2119] disabled:opacity-50"
              >
                {t("drawer.send")}
              </button>
            </div>
          ) : null}
        </footer>
      ) : null}
    </section>
  );
}
