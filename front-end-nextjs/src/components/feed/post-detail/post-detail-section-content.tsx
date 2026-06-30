"use client";

import { Button } from "@/components/ui/button";
import { isRichTextHtml, shouldInsertAsMarkdown } from "@/lib/rich-text";
import { cn } from "@/lib/utils";
import { AtSign,BookOpen,Bookmark,Heart,ImageIcon,MessageCircle,MoreHorizontal,Pencil,Share2,Smile,Trash2,X } from "lucide-react";
import { useRef } from "react";
import { Drawer } from "vaul";
import { LoginUnlockDialog } from "../login-unlock-dialog";
import { getAuthorInitial,isNovelPost } from "../feed-utils";
import { formatOriginalIncentiveAmount,OriginalIncentiveReward } from "../original-incentive";
import { CommentToolButton,DetailActionsPanel,FooterAction,ImageCarousel } from "./actions-panel";
import { CommentSkeletonList } from "./comments";
import { AuthorBlock,CommentThread,FollowButton,ImageViewerOverlay,MobileDetailHeader,ShareSheet } from "./media-header";
import { NovelPostDetailSection } from "./novel-post-detail-section";
import { formatReplyStatus } from "./post-detail-formatters";
import { ShakaVideoPlayer } from "./post-detail-types";
import type { PostDetailSectionContentProps } from "./post-detail-section-content-types";
import { PostReaderToolbar,usePostReader } from "./post-reader";
import { LazyPostMarkdownContent as PostMarkdownContent } from "./lazy-post-markdown-content";
import { PostResourceSection } from "./post-resource-section";

export type { PostDetailSectionContentProps } from "./post-detail-section-content-types";

export function PostDetailSectionContent({
  actionsOpen,
  attachment,
  author,
  authorHref,
  commentCount,
  commentFooterStyle,
  commentInputExpanded,
  commentInputRef,
  commentPlaceholder,
  commentState,
  commentText,
  comments,
  cover,
  currentUser,
  detailDate,
  emblaApi,
  emblaRef,
  expandedCommentInputHeight,
  followButtonDisabled,
  followButtonLabel,
  handleCancelCommentInput,
  handleCopyPostLink,
  handleDeleteComment,
  handleDeletePost,
  handleDislikeToggle,
  handleEditPost,
  handleFollowToggle,
  handleLoadCommentReplies,
  handleNavigateAway,
  handleOpenCommentInput,
  handleCreateProtectedPackage,
  handleDownloadImageArchive,
  handleDownloadProtectedPackage,
  handlePurchaseContent,
  handleRefreshImageArchive,
  handleSaveCurrentImage,
  handleSharePost,
  handleRetryComment,
  handleSubmitComment,
  handleSubmitReport,
  handleToggleCommentLike,
  highlightedCommentId,
  imageKey,
  imageViewerIndex,
  images,
  slideshowImages,
  isDeletingPost,
  isFollowingAuthor,
  isLoadingComments,
  isMutatingDislike,
  isPostDisliked,
  isPostOwner,
  isPostReported,
  isCreatingProtectedPackage,
  isDownloadingImageArchive,
  isDownloadingProtectedPackage,
  isPurchasingContent,
  isRefreshingImageArchive,
  isSubmittingComment,
  isSubmittingReport,
  locale,
  loginUnlockOpen,
  mode,
  mutatingCommentDeleteIds,
  mutatingCommentLikeIds,
  onCollect,
  onLike,
  onOpenChange,
  playableVideo,
  post,
  imageArchiveJob,
  protectedPackageJob,
  rawDetailContent,
  replyTarget,
  reportDescription,
  reportReason,
  safeImageIndex,
  setActionsOpen,
  setCommentDraft,
  setImageViewerState,
  setIsCommentInputFocused,
  setLoginUnlockOpen,
  setReportDescription,
  setReportReason,
  setShareOpen,
  shareOpen,
  t,
  tags,
  video,
  viewerImageIndex,
}: PostDetailSectionContentProps) {
  const readerScrollRef = useRef<HTMLDivElement>(null);
  const reader = usePostReader({
    content: rawDetailContent,
    postId: String(post.id),
    scrollRef: readerScrollRef,
  });
  const plainDetailContent = isPlainPostText(rawDetailContent)
    ? rawDetailContent?.trim()
    : null;
  const resourceSection = (
    <PostResourceSection
      attachment={attachment}
      handleCreateProtectedPackage={handleCreateProtectedPackage}
      handleDownloadImageArchive={handleDownloadImageArchive}
      handleDownloadProtectedPackage={handleDownloadProtectedPackage}
      handleNavigateAway={handleNavigateAway}
      handlePurchaseContent={handlePurchaseContent}
      handleRefreshImageArchive={handleRefreshImageArchive}
      imageArchiveJob={imageArchiveJob}
      isCreatingProtectedPackage={isCreatingProtectedPackage}
      isDownloadingImageArchive={isDownloadingImageArchive}
      isDownloadingProtectedPackage={isDownloadingProtectedPackage}
      isPurchasingContent={isPurchasingContent}
      isRefreshingImageArchive={isRefreshingImageArchive}
      post={post}
      protectedPackageJob={protectedPackageJob}
      t={t}
    />
  );

  if (isNovelPost(post)) {
    return (
      <NovelPostDetailSection
        author={author}
        authorHref={authorHref}
        commentCount={commentCount}
        commentInputExpanded={commentInputExpanded}
        commentInputRef={commentInputRef}
        commentPlaceholder={commentPlaceholder}
        commentState={commentState}
        commentText={commentText}
        comments={comments}
        currentUser={currentUser}
        detailDate={detailDate}
        followButtonDisabled={followButtonDisabled}
        followButtonLabel={followButtonLabel}
        handleCancelCommentInput={handleCancelCommentInput}
        handleCopyPostLink={handleCopyPostLink}
        handleDeleteComment={handleDeleteComment}
        handleDeletePost={handleDeletePost}
        handleEditPost={handleEditPost}
        handleFollowToggle={handleFollowToggle}
        handleLoadCommentReplies={handleLoadCommentReplies}
        handleNavigateAway={handleNavigateAway}
        handleOpenCommentInput={handleOpenCommentInput}
        handleRetryComment={handleRetryComment}
        handleSubmitComment={handleSubmitComment}
        handleToggleCommentLike={handleToggleCommentLike}
        highlightedCommentId={highlightedCommentId}
        isDeletingPost={isDeletingPost}
        isFollowingAuthor={isFollowingAuthor}
        isLoadingComments={isLoadingComments}
        isPostOwner={isPostOwner}
        isSubmittingComment={isSubmittingComment}
        locale={locale}
        mode={mode}
        mutatingCommentDeleteIds={mutatingCommentDeleteIds}
        mutatingCommentLikeIds={mutatingCommentLikeIds}
        onCollect={onCollect}
        onLike={onLike}
        onOpenChange={onOpenChange}
        post={post}
        rawDetailContent={rawDetailContent}
        setCommentDraft={setCommentDraft}
        setIsCommentInputFocused={setIsCommentInputFocused}
        t={t}
      />
    );
  }

  return (
    <>
          <section
            className={cn(
              "relative flex w-full flex-1 flex-col overflow-hidden bg-[#121212] md:flex-initial md:flex-row md:rounded-[20px] md:shadow-[0_24px_80px_rgba(0,0,0,0.42)]",
              reader.reading
                ? "md:h-[min(900px,calc(100vh-40px))] md:max-w-[1180px]"
                : video
                  ? "md:h-[min(836px,calc(100vh-64px))] md:max-w-[1232px]"
                  : "md:h-[min(780px,calc(100vh-64px))] md:max-w-[1028px]",
            )}
          >
              <MobileDetailHeader
              author={author}
              avatar={post.avatar ?? post.user_avatar ?? undefined}
              fallback={getAuthorInitial(post, "Y")}
              href={authorHref}
              onClose={() => onOpenChange(false)}
              disabled={followButtonDisabled}
              followLabel={followButtonLabel}
              following={isFollowingAuthor}
              onFollowToggle={handleFollowToggle}
              owner={isPostOwner}
              deleting={isDeletingPost}
              onEdit={handleEditPost}
              onDelete={() => void handleDeletePost()}
              onNavigate={handleNavigateAway}
              onShare={handleSharePost}
            />

            <div
              ref={readerScrollRef}
              className={cn(
                "flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
                reader.reading ? "md:flex md:items-center md:overflow-y-auto" : "md:contents md:overflow-visible",
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
              {!reader.reading || video || slideshowImages.length > 0 ? (
              <div
                className={cn(
                  "relative w-full shrink-0 bg-black md:h-full md:flex-1 md:rounded-l-[20px]",
                  reader.reading
                    ? "mx-auto h-[min(46vh,420px)] max-w-[760px] md:h-[360px] md:flex-none md:rounded-2xl"
                    : video
                      ? "aspect-video h-auto md:aspect-auto"
                      : "h-[min(64vh,565px)]",
                )}
              >
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={t("drawer.close")}
                  onClick={() => onOpenChange(false)}
                  className="post-media-close-button absolute left-4 top-4 z-20 hidden size-10 md:inline-flex"
                >
                  <X className="size-5" />
                </Button>

                {video ? (
                  <ShakaVideoPlayer
                    src={playableVideo}
                    poster={cover}
                    className="aspect-video h-full min-h-full rounded-none md:aspect-auto md:rounded-l-[20px] md:rounded-r-none"
                  />
                ) : slideshowImages.length > 0 ? (
                  <ImageCarousel
                    key={imageKey}
                    images={slideshowImages}
                    title={post.title}
                    emblaRef={emblaRef}
                    emblaApi={emblaApi}
                    previousLabel={t("drawer.previousImage")}
                    nextLabel={t("drawer.nextImage")}
                    failedLabel={t("drawer.noMedia")}
                    openLabel={t("card.openImage")}
                    onImageOpen={(index) => setImageViewerState({ index, postId: post.id })}
                    counterLabel={t("drawer.imageCounter", {
                      current: safeImageIndex + 1,
                      total: slideshowImages.length,
                    })}
                    overflowCount={Math.max(0, images.length - slideshowImages.length)}
                  />
                ) : (
                  <div className="flex size-full items-center justify-center px-6 text-center text-sm text-white/60">
                    {t("drawer.noMedia")}
                  </div>
                )}
              </div>
              ) : null}

              <aside className={cn(
                "flex min-h-[75dvh] flex-col bg-[#121212] md:min-h-0 md:flex-1 md:overflow-hidden md:w-[440px] md:flex-none md:rounded-r-[20px]",
                reader.reading && "mx-auto min-h-0 w-full max-w-[760px] flex-none md:w-full md:max-w-[760px] md:overflow-visible md:rounded-2xl",
              )}>
                <div className="hidden h-[88px] shrink-0 items-center justify-between border-b border-white/[0.08] px-6 md:flex">
                  <AuthorBlock
                    author={author}
                    avatar={post.avatar ?? post.user_avatar ?? undefined}
                    fallback={getAuthorInitial(post, "Y")}
                    href={authorHref}
                    onNavigate={handleNavigateAway}
                    subtitle={t("drawer.authorSubtitle")}
                  />
                  <div className="flex shrink-0 items-center gap-2">
                    {isPostOwner ? (
                      <>
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          aria-label={t("drawer.edit")}
                          onClick={handleEditPost}
                          className="size-9 shrink-0 rounded-full text-white/72 hover:bg-white/[0.06] hover:text-primary"
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
                          className="size-9 shrink-0 rounded-full text-white/72 hover:bg-white/[0.06] hover:text-[#ef4444] disabled:opacity-50"
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
                      />
                    )}
                    <Button
                      type="button"
                      size="icon"
                      variant="ghost"
                      aria-label={t("drawer.share")}
                      onClick={handleSharePost}
                      className="size-9 shrink-0 rounded-full text-white/72 hover:bg-white/[0.06] hover:text-primary"
                    >
                      <Share2 className="size-5" />
                    </Button>
                  </div>
                </div>

                <div
                  style={reader.readerStyle}
                  className={cn(
                    "px-5 pb-[calc(69px+env(safe-area-inset-bottom)+1.25rem)] pt-5 [scrollbar-width:none] md:min-h-0 md:flex-1 md:overflow-y-auto md:px-6 md:pb-6 md:pt-5 [&::-webkit-scrollbar]:hidden",
                    reader.reading && "post-reader-content md:overflow-visible md:px-10 md:pb-10",
                  )}
                >
                  <article>
                    {!reader.reading && reader.canRead ? (
                      <button
                        type="button"
                        onClick={() => reader.setReading(true)}
                        className="mb-3 inline-flex h-8 items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 text-xs font-semibold text-white/68 hover:bg-white/10 hover:text-white"
                      >
                        <BookOpen className="size-3.5" />
                        {t("drawer.reader.enter")}
                      </button>
                    ) : null}
                    {mode === "page" ? (
                      <h1 className="text-[18px] font-semibold leading-snug text-white">
                        {post.title}
                      </h1>
                    ) : (
                      <Drawer.Title className="text-[18px] font-semibold leading-snug text-white">
                        {post.title}
                      </Drawer.Title>
                    )}
                    <OriginalIncentiveReward
                      post={post}
                      title={t("drawer.originalIncentiveTitle")}
                      amountLabel={t("drawer.originalIncentiveAmount", { amount: formatOriginalIncentiveAmount(post, locale) })}
                      className="mt-3"
                    />
                    {post.resourceSectionPosition !== "after_content" ? resourceSection : null}

                    {plainDetailContent ? (
                      <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-white/90">
                        {plainDetailContent}
                      </p>
                    ) : rawDetailContent ? (
                      <PostMarkdownContent
                        className="mt-3"
                        content={rawDetailContent}
                        onNavigateAway={handleNavigateAway}
                      />
                    ) : null}
                    {post.resourceSectionPosition === "after_content" ? resourceSection : null}

                    {tags.length > 0 ? (
                      <div className="mt-4 flex flex-wrap gap-2">
                        {tags.map((tag) => (
                          <span
                            key={tag}
                            className="rounded-full bg-white/[0.06] px-2.5 py-1 text-[13px] text-white/78"
                          >
                            #{tag}
                          </span>
                        ))}
                      </div>
                    ) : null}

                    <p className="mt-4 text-xs text-white/68">
                      {detailDate}
                    </p>
                  </article>

                  <div className="mt-6 border-t border-white/[0.08] pt-5">
                    <div className="mb-4 flex items-center justify-between">
                      <h2 className="text-sm font-semibold text-white">
                        {t("drawer.commentCount", { count: commentCount })}
                      </h2>
                      <button
                        type="button"
                        onClick={() => setActionsOpen((currentOpen) => !currentOpen)}
                        className="rounded-full p-1 text-white/55 hover:bg-white/[0.06]"
                        aria-label={t("drawer.moreActions")}
                        aria-expanded={actionsOpen}
                      >
                        <MoreHorizontal className="size-5" />
                      </button>
                    </div>

                    {actionsOpen ? (
                      <DetailActionsPanel
                        description={reportDescription}
                        disliked={isPostDisliked}
                        isMutatingDislike={isMutatingDislike}
                        isSubmittingReport={isSubmittingReport}
                        onDescriptionChange={setReportDescription}
                        onDislikeToggle={handleDislikeToggle}
                        onReasonChange={setReportReason}
                        onReportSubmit={handleSubmitReport}
                        reason={reportReason}
                        reported={isPostReported}
                      />
                    ) : null}

                    <div className="space-y-5">
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
                  </div>
                </div>

                <footer
                  style={commentFooterStyle}
                  className={cn(
                    "post-comment-footer bottom-0 z-30 shrink-0 border-t border-[var(--post-comment-footer-border)] bg-[var(--post-comment-footer-bg)] px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 text-[var(--post-comment-footer-text)] transition-[height,transform] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)] will-change-transform md:static md:z-auto md:h-auto md:px-6 md:pb-4 md:pt-3",
                    mode === "page" ? "sticky" : "fixed inset-x-0",
                    commentInputExpanded
                      ? expandedCommentInputHeight
                      : "h-[calc(69px+env(safe-area-inset-bottom))]",
                  )}
                >
                  <div className="relative flex h-11 items-center md:h-10">
                    <form
                      onSubmit={handleSubmitComment}
                      className={cn(
                        "flex h-11 min-w-0 flex-1 items-center transition-[max-width,margin] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)] md:h-10",
                        commentInputExpanded
                          ? "mr-0 max-w-full"
                          : "mr-0 max-w-[calc(100%_-_200px)]",
                      )}
                    >
                      <textarea
                        ref={commentInputRef}
                        value={commentText}
                        onChange={(event) =>
                          setCommentDraft((currentDraft) => ({
                            postId: post.id,
                            replyTarget:
                              currentDraft.postId === post.id
                                ? (currentDraft.replyTarget ?? null)
                                : null,
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
                        onFocus={() => {
                          setIsCommentInputFocused(true);
                        }}
                        onBlur={() => {
                          if (!commentText.trim()) {
                            setIsCommentInputFocused(false);
                          }
                        }}
                        onKeyDown={(event) => {
                          if (
                            event.key === "Enter" &&
                            !event.shiftKey &&
                            !event.nativeEvent.isComposing
                          ) {
                            event.preventDefault();
                            event.currentTarget.form?.requestSubmit();
                          }
                        }}
                        disabled={isSubmittingComment}
                        aria-label={t("drawer.saySomething")}
                        placeholder={commentPlaceholder}
                        rows={1}
                        className={cn(
                          "h-11 max-h-20 min-h-8 min-w-0 flex-1 resize-none overflow-y-auto rounded-2xl border border-[var(--post-comment-control-border)] bg-[var(--post-comment-control-bg)] px-4 py-3 text-base leading-5 text-[var(--post-comment-input-text)] caret-primary outline-none transition-[border-radius,padding,font-size,background-color,border-color,box-shadow] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)] placeholder:text-[var(--post-comment-placeholder)] focus:border-primary/45 focus:ring-2 focus:ring-primary/15 md:h-10 md:py-2.5",
                          commentInputExpanded && "rounded-[20px]",
                        )}
                      />
                    </form>

                    <div
                      className={cn(
                        "flex h-11 shrink-0 items-center gap-1 text-[var(--post-comment-footer-muted)] transition-[opacity,transform] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)] md:h-10",
                        commentInputExpanded
                          ? "pointer-events-none absolute right-0 translate-x-[50px] opacity-0"
                          : "relative translate-x-0 opacity-100",
                      )}
                    >
                      <FooterAction
                        active={post.liked}
                        ariaLabel={post.liked ? t("card.unlike") : t("card.like")}
                        count={post.like_count}
                        icon={<Heart className={cn("size-6", post.liked && "fill-primary text-primary")} />}
                        onClick={() => onLike?.(post)}
                      />
                      <FooterAction
                        active={post.collected}
                        ariaLabel={t("drawer.collections", { count: post.collect_count ?? 0 })}
                        count={post.collect_count ?? 0}
                        icon={<Bookmark className={cn("size-6", post.collected && "fill-primary text-primary")} />}
                        onClick={() => onCollect?.(post)}
                      />
                      <FooterAction
                        ariaLabel={t("drawer.comments", { count: commentCount })}
                        count={commentCount}
                        icon={<MessageCircle className="size-6" />}
                        onClick={() => handleOpenCommentInput()}
                      />
                    </div>
                  </div>

                  <div
                    className={cn(
                      "overflow-hidden transition-[height,opacity,padding] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)]",
                      commentInputExpanded && replyTarget
                        ? "h-8 pb-2 opacity-100"
                        : "h-0 pb-0 opacity-0",
                    )}
                  >
                    {replyTarget ? (
                      <div className="flex h-6 items-center justify-between gap-3 rounded-full bg-[var(--post-comment-control-bg)] px-3 text-xs text-[var(--post-comment-footer-subtle)] ring-1 ring-inset ring-[var(--post-comment-control-border)]">
                        <span className="min-w-0 truncate">
                          {formatReplyStatus(replyTarget.author, locale)}
                        </span>
                        <button
                          type="button"
                          onMouseDown={(event) => event.preventDefault()}
                          onClick={() => {
                            setCommentDraft((currentDraft) => ({
                              postId: post.id,
                              replyTarget: null,
                              text: currentDraft.postId === post.id ? currentDraft.text : "",
                            }));
                          }}
                          className="shrink-0 rounded-full p-0.5 text-[var(--post-comment-footer-faint)] hover:text-[var(--post-comment-footer-text)]"
                          aria-label={t("drawer.cancel")}
                        >
                          <X className="size-3.5" />
                        </button>
                      </div>
                    ) : null}
                  </div>

                  <div
                    className={cn(
                      "flex items-center justify-between gap-3 overflow-hidden transition-[height,padding,opacity] duration-[400ms] ease-[cubic-bezier(0.4,0,0.2,1)] md:hidden",
                      commentInputExpanded
                        ? "h-[66px] py-3 opacity-100"
                        : "h-0 py-0 opacity-0",
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <CommentToolButton
                        ariaLabel={t("drawer.mention")}
                        icon={<AtSign className="size-5" />}
                      />
                      <CommentToolButton
                        ariaLabel={t("drawer.emoji")}
                        icon={<Smile className="size-5" />}
                      />
                      <CommentToolButton
                        ariaLabel={t("drawer.attachImage")}
                        icon={<ImageIcon className="size-5" />}
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <button
                        type="button"
                        disabled={!commentText.trim() || isSubmittingComment}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => {
                          commentInputRef.current?.form?.requestSubmit();
                        }}
                        className="h-10 rounded-full bg-primary px-5 text-base font-semibold text-white transition-opacity disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {t("drawer.send")}
                      </button>
                      <button
                        type="button"
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={handleCancelCommentInput}
                        className="h-10 rounded-full border border-[var(--post-comment-control-border)] bg-transparent px-5 text-base font-semibold text-[var(--post-comment-footer-muted)] transition-colors hover:bg-[var(--post-comment-control-hover)] hover:text-[var(--post-comment-footer-text)]"
                      >
                        {t("drawer.cancel")}
                      </button>
                    </div>
                  </div>

                </footer>
              </aside>
            </div>
          </section>
          {imageViewerIndex !== null && images.length > 0 ? (
            <ImageViewerOverlay
              closeLabel={t("drawer.close")}
              counterLabel={`${viewerImageIndex + 1} / ${images.length}`}
              failedLabel={t("drawer.noMedia")}
              images={images}
              index={viewerImageIndex}
              nextLabel={t("drawer.nextImage")}
              onClose={() => setImageViewerState(null)}
              onIndexChange={(index) => setImageViewerState({ index, postId: post.id })}
              previousLabel={t("drawer.previousImage")}
              title={post.title}
            />
          ) : null}
          {shareOpen ? (
            <ShareSheet
              copyLabel={t("drawer.shareCopied")}
              onClose={() => setShareOpen(false)}
              onCopy={() => void handleCopyPostLink()}
              onSave={() => void handleSaveCurrentImage()}
            />
          ) : null}
          <LoginUnlockDialog
            open={loginUnlockOpen}
            onOpenChange={setLoginUnlockOpen}
          />
        </>

  );
}

function isPlainPostText(content: string | null | undefined) {
  return Boolean(
    content?.trim() &&
      !isRichTextHtml(content) &&
      !shouldInsertAsMarkdown(content),
  );
}
