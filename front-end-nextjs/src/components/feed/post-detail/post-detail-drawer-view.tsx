import {
Button
} from "@/components/ui/button";
import { richTextToPlainText } from "@/lib/rich-text";
import {
getUserHrefFromPost
} from "@/lib/users";
import {
cn
} from "@/lib/utils";
import {
X
} from "lucide-react";
import { Drawer } from "vaul";
import {
getAuthorName,
getPlayableVideoUrl,
getPostCover,
isNovelPost,
isVideoPost
} from "../feed-utils";
import {
buildTags,
formatDetailDate,
isCurrentUserPostOwner
} from "./post-detail-formatters";
import { PostDetailFrame } from "./post-detail-frame";
import { PostDetailSectionContent } from "./post-detail-section-content";
import type { usePostDetailDrawerController } from "./use-post-detail-drawer-controller";

export function PostDetailDrawerView({ controller }: { controller: ReturnType<typeof usePostDetailDrawerController> }) {
  const { actionState, actionsOpen, authorUserId, commentCountDelta, commentFooterStyle, commentInputExpanded, commentInputRef, commentPlaceholder, commentState, commentText, comments, currentUser, drawerContentRef, emblaApi, emblaRef, expandedCommentInputHeight, followState, handleCancelCommentInput, handleCopyPostLink, handleCreateProtectedPackage, handleDeleteComment, handleDeletePost, handleDislikeToggle, handleDownloadImageArchive, handleDownloadProtectedPackage, handleEditPost, handleFollowToggle, handleLoadCommentReplies, handleOpenCommentInput, handlePostCollect, handlePostLike, handlePurchaseContent, handleRefreshImageArchive, handleRetryComment, handleSaveCurrentImage, handleSharePost, handleSubmitComment, handleSubmitReport, handleToggleCommentLike, highlightedCommentId, imageArchiveJob, imageKey, imageViewerIndex, images, isCreatingProtectedPackage, isDeletingPost, isDownloadingImageArchive, isDownloadingProtectedPackage, isLoadingComments, isMutatingDislike, isMutatingFollow, isPurchasingContent, isRefreshingImageArchive, isSubmittingComment, isSubmittingReport, locale, loginUnlockOpen, mode, mutatingCommentDeleteIds, mutatingCommentLikeIds, onOpenChange, open, post, protectedPackageJob, replyTarget, reportDescription, reportReason, selectedImageIndex, setActionsOpen, setCommentDraft, setImageViewerState, setIsCommentInputFocused, setLoginUnlockOpen, setReportDescription, setReportReason, setShareOpen, shareOpen, slideshowImages, t } = controller;
if (!post) {
    const loadingLabel = t("drawer.loadingPost");
    const loadingContent = (
      <section
        aria-busy="true"
        aria-label={loadingLabel}
        className="relative flex w-full flex-1 flex-col overflow-hidden bg-[#121212] md:h-[min(780px,calc(100vh-64px))] md:max-w-[1028px] md:flex-initial md:flex-row md:rounded-[20px] md:shadow-[0_24px_80px_rgba(0,0,0,0.42)]"
      >
        {mode === "page" ? (
          <h1 className="sr-only">{loadingLabel}</h1>
        ) : (
          <Drawer.Title className="sr-only">{loadingLabel}</Drawer.Title>
        )}

        <div className="relative flex h-[min(64vh,565px)] w-full shrink-0 items-center justify-center bg-black md:h-full md:flex-1 md:rounded-l-[20px]">
          <div className="absolute inset-x-0 top-0 z-20 flex items-center justify-between p-4 md:hidden">
            <Button
              size="icon"
              variant="ghost"
              aria-label={t("drawer.close")}
              onClick={() => onOpenChange(false)}
              className="post-media-close-button size-10 rounded-full"
            >
              <X className="size-5" />
            </Button>
          </div>
          <Button
            size="icon"
            variant="ghost"
            aria-label={t("drawer.close")}
            onClick={() => onOpenChange(false)}
            className="post-media-close-button absolute left-4 top-4 z-20 hidden size-10 md:inline-flex"
          >
            <X className="size-5" />
          </Button>
          <div className="size-12 animate-spin rounded-full border-2 border-white/15 border-t-primary motion-reduce:animate-none" />
        </div>

        <aside className="flex min-h-[75dvh] flex-col bg-[#121212] md:min-h-0 md:w-[440px] md:flex-none md:rounded-r-[20px]">
          <div className="hidden h-[88px] shrink-0 items-center justify-between border-b border-white/[0.08] px-6 md:flex">
            <div className="flex min-w-0 items-center gap-3">
              <div className="size-10 animate-pulse rounded-full bg-white/[0.08] motion-reduce:animate-none" />
              <div className="space-y-2">
                <div className="h-3 w-28 animate-pulse rounded-full bg-white/[0.10] motion-reduce:animate-none" />
                <div className="h-2.5 w-20 animate-pulse rounded-full bg-white/[0.06] motion-reduce:animate-none" />
              </div>
            </div>
            <div className="h-9 w-20 animate-pulse rounded-full bg-white/[0.08] motion-reduce:animate-none" />
          </div>

          <div className="min-h-0 flex-1 px-5 pb-[calc(69px+env(safe-area-inset-bottom)+1.25rem)] pt-5 md:overflow-hidden md:px-6 md:pb-6 md:pt-5">
            <div className="space-y-3">
              <div className="h-5 w-4/5 animate-pulse rounded-full bg-white/[0.10] motion-reduce:animate-none" />
              <div className="h-5 w-2/3 animate-pulse rounded-full bg-white/[0.08] motion-reduce:animate-none" />
              <div className="pt-2">
                <div className="mb-2 h-3 w-full animate-pulse rounded-full bg-white/[0.07] motion-reduce:animate-none" />
                <div className="mb-2 h-3 w-11/12 animate-pulse rounded-full bg-white/[0.07] motion-reduce:animate-none" />
                <div className="h-3 w-7/12 animate-pulse rounded-full bg-white/[0.07] motion-reduce:animate-none" />
              </div>
            </div>

            <div className="mt-6 border-t border-white/[0.08] pt-5">
              <div className="mb-5 h-4 w-24 animate-pulse rounded-full bg-white/[0.09] motion-reduce:animate-none" />
              <div className="space-y-5">
                {[0, 1, 2].map((item) => (
                  <div key={item} className="flex gap-3">
                    <div className="size-8 shrink-0 animate-pulse rounded-full bg-white/[0.08] motion-reduce:animate-none" />
                    <div className="min-w-0 flex-1 space-y-2">
                      <div className="h-3 w-28 animate-pulse rounded-full bg-white/[0.08] motion-reduce:animate-none" />
                      <div className="h-3 w-full animate-pulse rounded-full bg-white/[0.06] motion-reduce:animate-none" />
                      <div className="h-3 w-7/12 animate-pulse rounded-full bg-white/[0.06] motion-reduce:animate-none" />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <footer
            className={cn(
              "post-comment-footer bottom-0 z-30 h-[calc(69px+env(safe-area-inset-bottom))] shrink-0 border-t border-[var(--post-comment-footer-border)] bg-[var(--post-comment-footer-bg)] px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 md:static md:z-auto md:px-6 md:pb-4 md:pt-3",
              mode === "page" ? "sticky" : "fixed inset-x-0",
            )}
          >
            <div className="h-11 animate-pulse rounded-2xl bg-[var(--post-comment-control-bg)] ring-1 ring-inset ring-[var(--post-comment-control-border)] motion-reduce:animate-none md:h-10" />
          </footer>
        </aside>
      </section>
    );

    return (
      <PostDetailFrame
        description={loadingLabel}
        drawerContentRef={drawerContentRef}
        mode={mode}
        open={open}
        onOpenChange={onOpenChange}
      >
        {loadingContent}
      </PostDetailFrame>
    );
  }

const attachment = post.attachment?.url ? post.attachment : null;

const cover = getPostCover(post);

const playableVideo = getPlayableVideoUrl(post);

const video = isVideoPost(post);

const safeImageIndex = slideshowImages.length > 0 ? Math.min(selectedImageIndex, slideshowImages.length - 1) : 0;

const viewerImageIndex =
    imageViewerIndex === null || images.length === 0
      ? 0
      : Math.min(Math.max(imageViewerIndex, 0), images.length - 1);

const author = getAuthorName(post, t("card.unknownAuthor"));

const authorHref = getUserHrefFromPost(post);

const isFollowingAuthor =
    followState.userId === authorUserId ? followState.isFollowing : false;

const followButtonDisabled = followState.buttonType === "self" || isMutatingFollow;

const followButtonLabel = isFollowingAuthor
    ? t("profile.followingAction")
    : t("drawer.follow");

const tags = buildTags(post);

const rawDetailContent = post.content?.trim() ? post.content : null;

const detailContent = rawDetailContent ? richTextToPlainText(rawDetailContent) : null;

const detailDate = formatDetailDate(post.created_at, t("drawer.recentDate"), locale);

const commentCount = Math.max(
    (post.comment_count ?? 0) + commentCountDelta,
    comments.length,
  );

const isPostDisliked =
    actionState.postId === post.id ? actionState.disliked : false;

const isPostReported =
    actionState.postId === post.id ? actionState.reported : false;

const isPostOwner = isCurrentUserPostOwner(post, currentUser);

const handleNavigateAway = () => {
    if (mode === "drawer") {
      onOpenChange(false);
    }
  };

const sectionContent = (
    <PostDetailSectionContent
      actionsOpen={actionsOpen}
      attachment={attachment}
      author={author}
      authorHref={authorHref}
      commentCount={commentCount}
      commentFooterStyle={commentFooterStyle}
      commentInputExpanded={commentInputExpanded}
      commentInputRef={commentInputRef}
      commentPlaceholder={commentPlaceholder}
      commentState={commentState}
      commentText={commentText}
      comments={comments}
      cover={cover}
      currentUser={currentUser}
      detailDate={detailDate}
      emblaApi={emblaApi}
      emblaRef={emblaRef}
      expandedCommentInputHeight={expandedCommentInputHeight}
      followButtonDisabled={followButtonDisabled}
      followButtonLabel={followButtonLabel}
      handleCancelCommentInput={handleCancelCommentInput}
      handleCopyPostLink={handleCopyPostLink}
      handleDeleteComment={handleDeleteComment}
      handleDeletePost={handleDeletePost}
      handleDislikeToggle={handleDislikeToggle}
      handleEditPost={handleEditPost}
      handleFollowToggle={handleFollowToggle}
      handleLoadCommentReplies={handleLoadCommentReplies}
      handleNavigateAway={handleNavigateAway}
      handleOpenCommentInput={handleOpenCommentInput}
      handleCreateProtectedPackage={handleCreateProtectedPackage}
      handleDownloadImageArchive={handleDownloadImageArchive}
      handleDownloadProtectedPackage={handleDownloadProtectedPackage}
      handlePurchaseContent={handlePurchaseContent}
      handleRefreshImageArchive={handleRefreshImageArchive}
      handleSaveCurrentImage={handleSaveCurrentImage}
      handleSharePost={handleSharePost}
      handleSubmitComment={handleSubmitComment}
      handleRetryComment={handleRetryComment}
      handleSubmitReport={handleSubmitReport}
      handleToggleCommentLike={handleToggleCommentLike}
      highlightedCommentId={highlightedCommentId}
      imageKey={imageKey}
      imageViewerIndex={imageViewerIndex}
      images={images}
      slideshowImages={slideshowImages}
      isDeletingPost={isDeletingPost}
      isFollowingAuthor={isFollowingAuthor}
      isLoadingComments={isLoadingComments}
      isMutatingDislike={isMutatingDislike}
      isPostDisliked={isPostDisliked}
      isPostOwner={isPostOwner}
      isPostReported={isPostReported}
      isCreatingProtectedPackage={isCreatingProtectedPackage}
      isDownloadingImageArchive={isDownloadingImageArchive}
      isDownloadingProtectedPackage={isDownloadingProtectedPackage}
      isPurchasingContent={isPurchasingContent}
      isRefreshingImageArchive={isRefreshingImageArchive}
      isSubmittingComment={isSubmittingComment}
      isSubmittingReport={isSubmittingReport}
      locale={locale}
      loginUnlockOpen={loginUnlockOpen}
      mode={mode}
      mutatingCommentDeleteIds={mutatingCommentDeleteIds}
      mutatingCommentLikeIds={mutatingCommentLikeIds}
      onCollect={handlePostCollect}
      onLike={handlePostLike}
      onOpenChange={onOpenChange}
      playableVideo={playableVideo}
      post={post}
      imageArchiveJob={imageArchiveJob}
      protectedPackageJob={protectedPackageJob}
      rawDetailContent={rawDetailContent}
      replyTarget={replyTarget}
      reportDescription={reportDescription}
      reportReason={reportReason}
      safeImageIndex={safeImageIndex}
      setActionsOpen={setActionsOpen}
      setCommentDraft={setCommentDraft}
      setImageViewerState={setImageViewerState}
      setIsCommentInputFocused={setIsCommentInputFocused}
      setLoginUnlockOpen={setLoginUnlockOpen}
      setReportDescription={setReportDescription}
      setReportReason={setReportReason}
      setShareOpen={setShareOpen}
      shareOpen={shareOpen}
      t={t}
      tags={tags}
      video={video}
      viewerImageIndex={viewerImageIndex}
    />
  );

return (
    <PostDetailFrame
      description={detailContent ?? post.title}
      drawerContentRef={drawerContentRef}
      fullScreen={isNovelPost(post)}
      mode={mode}
      open={open}
      onOpenChange={onOpenChange}
    >
      {sectionContent}
    </PostDetailFrame>
  );
}
