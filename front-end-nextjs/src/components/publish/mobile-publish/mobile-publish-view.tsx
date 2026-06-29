import { ChevronDown,ChevronRight,Earth,Inbox,Paperclip,Play,Shield,Sparkles,Trash2,X } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import { toast } from "sonner";
import { AIFormatPanel } from "../shared/ai-format-panel";
import { PublishGenerationCard } from "../shared/publish-generation-card";
import { PublishGenerationPanel } from "../shared/publish-generation-panel";
import type { PublishGenerationRunOptions } from "../shared/publish-generation-action";
import { MobileMarkdownEditor } from "./mobile-markdown-editor";
import { MobileImageManager } from "./mobile-image-manager";
import { acceptedMediaTypes,parseMobileTopics,titleLimit } from "./mobile-publish-config";
import { MobilePublishOverlays } from "./mobile-publish-overlays";
import { formatFileSize,formatMobileUploadChunkLabel,mobileUploadProgressKey } from "./mobile-publish-utils";
import type { useMobilePublishController } from "./use-mobile-publish-controller";

export function MobilePublishView({ controller }: { controller: ReturnType<typeof useMobilePublishController> }) {
  const { activeEmojiGroupId, aiFormatOpen, attachmentFile, attachmentInputRef, avatarUrl, board, body, canPublish, cancelPublishGeneration, categories, closeWithoutSaving, customTopicInput, cycleVisibility, displayName, existingAttachment, fileInputRef, handleApplyPublishGeneration, handleAttachmentChange, handleBatchUpdateImages, handleCloseClick, handleGeneratePublishContent, handleMediaChange, handlePickAttachment, handlePickMedia, handleRemoveImages, handleReorderImages, handleReplaceThumbnailChange, handleReplaceVideo, handleReplaceVideoChange, handleRetryImage, handleSubmit, hasAttachment, imageLimit, imagePaymentMethod, imagePrice, imageProtectionEnabled, imageProtectionNoticeEnabled, imageSelectAllEnabled, insertEmoji, insertMention, isSearchingMentionUsers, isSubmitting, lightboxAsset, markdownEditorMode, managedImages, mediaAssets, mediaHelpText, mediaKind, mentionKeyword, mentionUsers, openDraftPage, openMediaPreview, paidContentEnabled, paidContentPaymentMethods, paidImageCount, paymentMaxPrices, postContentLimit, publishGeneration, publishGenerationCanRun, publishGenerationImageCount, removeMedia, replaceThumbnailInputRef, replaceVideoInputRef, saveAndClose, selectedCategoryId, setAIFormatOpen, setActiveEmojiGroupId, setAttachmentFile, setAttachmentTouched, setBoard, setBody, setCustomTopicInput, setExistingAttachment, setImagePaymentMethod, setImagePrice, setIsSearchingMentionUsers, setLightboxAsset, setMarkdownEditorMode, setMentionKeyword, setMentionUsers, setReplacingThumbnailAssetId, setSelectedCategoryId, setSelectedImageIds, setShowBoardSheet, setShowCloseConfirm, setShowEmojiSheet, setShowMentionSheet, setShowTopicSheet, setTitle, setTopic, showBoardSheet, showCloseConfirm, showEmojiSheet, showMentionSheet, showTopicSheet, submitCustomTopic, t, tags, title, topic, uploadAbortControllerRef, uploadProgress, validSelectedImageIds, visibility } = controller;
  const [publishGenerationOpen, setPublishGenerationOpen] = useState(false);
  const runPublishGenerationInPanel = (options?: PublishGenerationRunOptions) => {
    setPublishGenerationOpen(true);
    void handleGeneratePublishContent(options);
  };
  return (
    <main className="mobile-publish-page min-h-dvh bg-[var(--mobile-publish-bg)] text-[var(--mobile-publish-text)]">
      <div className="mx-auto flex min-h-dvh w-full max-w-[430px] flex-col bg-[var(--mobile-publish-surface)] max-[430px]:max-w-none">
        <header className="sticky top-0 z-20 grid h-18 shrink-0 grid-cols-[44px_minmax(0,1fr)_68px] items-center gap-3 border-b border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-4 pt-[env(safe-area-inset-top)]">
          <button
            type="button"
            aria-label={t("publish.mobile.close")}
            onClick={handleCloseClick}
            className="flex size-11 shrink-0 items-center justify-center rounded-full text-[var(--mobile-publish-text)] transition-colors active:bg-[var(--mobile-publish-accent-soft)]"
          >
            <X className="size-8" strokeWidth={2.2} />
          </button>

          <div className="flex min-w-0 flex-1 items-center justify-center gap-1.5">
            <h1 className="truncate text-[24px] font-black leading-none text-[var(--mobile-publish-heading)]">{t("publish.mobile.title")}</h1>
            <Sparkles className="size-5 shrink-0 fill-[var(--mobile-publish-accent)] text-[var(--mobile-publish-accent)]" />
          </div>

          <button
            type="button"
            disabled={!canPublish}
            onClick={() => void handleSubmit(false)}
            className="flex h-9 min-w-[64px] items-center justify-center rounded-full bg-[var(--mobile-publish-accent)] px-4 text-[15px] font-bold text-white transition-colors active:bg-[var(--mobile-publish-accent-strong)] disabled:bg-[var(--mobile-publish-border-soft)] disabled:text-[var(--mobile-publish-muted)]"
          >
            {t("publish.mobile.publish")}
          </button>
        </header>

        <section className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain bg-[var(--mobile-publish-surface)] px-4 pb-[calc(28px+env(safe-area-inset-bottom))] pt-4">
          <div className="flex items-center gap-4">
            <div className="relative size-[58px] shrink-0 overflow-hidden rounded-full bg-[var(--mobile-publish-card-strong)]">
              {avatarUrl ? (
                <Image
                  src={avatarUrl}
                  alt={t("publish.mobile.avatarAlt", { name: displayName })}
                  fill
                  unoptimized
                  sizes="58px"
                  className="object-cover"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center bg-[var(--mobile-publish-accent)] text-xl font-black text-white">
                  {displayName.charAt(0).toUpperCase()}
                </div>
              )}
            </div>

            <button
              type="button"
              onClick={() => setShowBoardSheet(true)}
              className="flex h-[42px] min-w-0 flex-1 items-center justify-center gap-2 rounded-full border border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-surface)] px-4 text-[17px] font-bold text-[var(--mobile-publish-accent-strong)] active:bg-[var(--mobile-publish-accent-soft)]"
            >
              <Shield className="size-5 shrink-0" />
              <span className="truncate">{board || t("publish.mobile.selectBoard")}</span>
              <ChevronDown className="size-5 shrink-0" />
            </button>

            <button
              type="button"
              onClick={openDraftPage}
              disabled={isSubmitting}
              className="flex h-[42px] shrink-0 items-center gap-1.5 rounded-full px-1 text-[16px] font-semibold text-[var(--mobile-publish-muted)] disabled:opacity-50"
            >
              <span>{t("publish.mobile.drafts")}</span>
              <Inbox className="size-5" />
            </button>
          </div>

          <div className="mt-7 flex h-[54px] items-center rounded-[14px] bg-[var(--mobile-publish-card)] px-4 shadow-[var(--mobile-publish-shadow)]">
            <input
              value={title}
              onChange={(event) => setTitle(event.target.value.slice(0, titleLimit))}
              placeholder={t("publish.mobile.titlePlaceholder", { count: titleLimit })}
              className="min-w-0 flex-1 bg-transparent text-[19px] font-semibold text-[var(--mobile-publish-input)] outline-none placeholder:text-[var(--mobile-publish-subtle)]"
            />
            <span className="ml-3 shrink-0 text-[17px] font-semibold text-[var(--mobile-publish-counter)]">
              {title.length}/{titleLimit}
            </span>
          </div>

          <MobileMarkdownEditor
            aiDisabled={false}
            limit={postContentLimit}
            mode={markdownEditorMode}
            onChange={setBody}
            onModeChange={setMarkdownEditorMode}
            onOpenAIFormat={() => setAIFormatOpen(true)}
            onOpenAttachment={handlePickAttachment}
            onOpenEmoji={() => setShowEmojiSheet(true)}
            onOpenMention={() => {
              setShowMentionSheet(true);
              setIsSearchingMentionUsers(true);
            }}
            placeholder={t("publish.mobile.bodyPlaceholder")}
            t={t}
            value={body}
          />

          <input
            ref={fileInputRef}
            type="file"
            accept={acceptedMediaTypes}
            multiple
            onChange={handleMediaChange}
            className="hidden"
          />

          <input
            ref={attachmentInputRef}
            type="file"
            onChange={handleAttachmentChange}
            className="hidden"
          />

          <input
            ref={replaceVideoInputRef}
            type="file"
            accept="video/*"
            onChange={handleReplaceVideoChange}
            className="hidden"
          />

          <input
            ref={replaceThumbnailInputRef}
            type="file"
            accept="image/*"
            onChange={handleReplaceThumbnailChange}
            className="hidden"
          />

          {hasAttachment ? (
            <div className="mt-4 flex min-h-12 items-center gap-3 rounded-[12px] bg-[var(--mobile-publish-card)] px-4 text-[15px] shadow-[var(--mobile-publish-shadow)]">
              <Paperclip className="size-5 shrink-0 text-[var(--mobile-publish-accent-strong)]" />
              <div className="min-w-0 flex-1">
                <p className="truncate font-semibold text-[var(--mobile-publish-input)]">
                  {attachmentFile?.name ?? existingAttachment?.filename ?? t("publish.mobile.attachment")}
                </p>
                <p className="text-xs text-[var(--mobile-publish-muted)]">
                  {formatFileSize(attachmentFile?.size ?? existingAttachment?.filesize ?? 0)}
                </p>
              </div>
              <button
                type="button"
                aria-label={t("publish.mobile.deleteAttachment")}
                onClick={() => {
                  setAttachmentFile(null);
                  setExistingAttachment(null);
                  setAttachmentTouched(true);
                }}
                className="flex size-8 shrink-0 items-center justify-center rounded-full text-[var(--mobile-publish-muted)] active:bg-[var(--mobile-publish-accent-soft)]"
              >
                <X className="size-5" />
              </button>
            </div>
          ) : null}

          {mediaKind === "video" ? (
            <div className="mt-10 grid grid-cols-3 gap-3">
              {mediaAssets.map((item) => (
                <div
                  key={item.id}
                  className="relative aspect-square overflow-hidden rounded-[14px] bg-[var(--mobile-publish-card-strong)]"
                >
                  <button
                    type="button"
                    aria-label={t("publish.mobile.previewVideo")}
                    onClick={() => openMediaPreview(item)}
                    className="group relative flex size-full items-center justify-center"
                  >
                    {item.previewUrl ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={item.previewUrl} alt={item.name} className="absolute inset-0 size-full object-cover" />
                    ) : (
                      <Play className="size-10 text-[var(--mobile-publish-muted)]" />
                    )}
                  </button>
                  <button
                    type="button"
                    aria-label={t("publish.mobile.deleteMedia")}
                    onClick={() => removeMedia(item.id)}
                    className="absolute right-1.5 top-1.5 z-10 flex size-7 items-center justify-center rounded-full bg-black/45 text-white"
                  >
                    <Trash2 className="size-4" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <MobileImageManager
              disabled={isSubmitting}
              imageLimit={imageLimit}
              imagePaymentMethod={imagePaymentMethod}
              paymentMethodsEnabled={paidContentPaymentMethods}
              paymentMaxPrices={paymentMaxPrices}
              imagePrice={imagePrice}
              items={managedImages}
              mediaCount={mediaAssets.length}
              mediaHelpText={mediaHelpText}
              paidImageCount={paidImageCount}
              paidContentEnabled={paidContentEnabled}
              protectionEnabled={imageProtectionEnabled}
              protectionNoticeEnabled={imageProtectionNoticeEnabled}
              selectAllEnabled={imageSelectAllEnabled}
              selectedIds={validSelectedImageIds}
              showAddButton
              onAddMedia={handlePickMedia}
              onBatchUpdate={handleBatchUpdateImages}
              onPaymentMethodChange={(method) => {
                if (!paidContentPaymentMethods[method]) {
                  toast.error(t("publish.protection.paymentMethodDisabledToast"));
                  return;
                }
                setImagePaymentMethod(method);
              }}
              onPriceChange={setImagePrice}
              onRemove={handleRemoveImages}
              onReorder={handleReorderImages}
              onRetry={handleRetryImage}
              onSelectionChange={setSelectedImageIds}
            />
          )}

          {mediaKind === "image" ? (
            <div className="mt-4">
              <PublishGenerationCard
                canRun={publishGenerationCanRun}
                imageCount={publishGenerationImageCount}
                onCancel={cancelPublishGeneration}
                onOpen={() => setPublishGenerationOpen(true)}
                onRun={runPublishGenerationInPanel}
                state={publishGeneration}
                t={t}
                variant="mobile"
              />
            </div>
          ) : null}

          <button
            type="button"
            onClick={() => {
              setCustomTopicInput(topic);
              setShowTopicSheet(true);
            }}
            className="mt-10 flex min-h-[58px] w-full items-center gap-3 rounded-[12px] bg-[var(--mobile-publish-card)] px-4 text-left shadow-[var(--mobile-publish-shadow)] active:bg-[var(--mobile-publish-accent-soft)]"
          >
            <span className="flex h-9 shrink-0 items-center gap-2 rounded-full border border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-4 text-[18px] font-bold text-[var(--mobile-publish-accent-strong)]">
              # {t("publish.mobile.addTopic")}
            </span>
            <span className="min-w-0 flex-1 truncate text-[16px] font-medium text-[var(--mobile-publish-muted)]">
              {topic
                ? parseMobileTopics(topic).map((item) => `#${item}`).join("  ")
                : t("publish.mobile.topicHint")}
            </span>
            <ChevronRight className="size-7 shrink-0 text-[var(--mobile-publish-muted)]" />
          </button>

          <button
            type="button"
            onClick={cycleVisibility}
            className="mt-8 flex min-h-[58px] w-full items-center gap-3 rounded-[12px] bg-[var(--mobile-publish-card)] px-4 text-left shadow-[var(--mobile-publish-shadow)] active:bg-[var(--mobile-publish-accent-soft)]"
          >
            <span className="flex h-9 shrink-0 items-center gap-2 rounded-full border border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-4 text-[18px] font-bold text-[var(--mobile-publish-accent-strong)]">
              <Earth className="size-5" />
              {t(`publish.mobile.visibility.${visibility}`)}
            </span>
            <ChevronDown className="ml-auto size-7 shrink-0 text-[var(--mobile-publish-muted)]" />
          </button>
        </section>
        {uploadProgress ? (
          <footer className="sticky bottom-0 z-20 shrink-0 border-t border-[var(--mobile-publish-border-soft)] bg-[var(--mobile-publish-surface)] px-4 pb-[calc(24px+env(safe-area-inset-bottom))] pt-4 shadow-[0_-12px_28px_rgba(0,0,0,0.18)]">
            <div className="rounded-[14px] bg-[var(--mobile-publish-card)] px-3 py-3 shadow-[var(--mobile-publish-shadow)]">
              <div className="flex items-center justify-between gap-3">
                <p className="min-w-0 truncate text-[13px] font-bold text-[var(--mobile-publish-accent-strong)]">
                  {t(`publish.mobile.upload.${mobileUploadProgressKey(uploadProgress)}`)}
                </p>
                <div className="flex shrink-0 items-center gap-2">
                  {uploadProgress.phase === "uploading" ? (
                    <button
                      type="button"
                      onClick={() => uploadAbortControllerRef.current?.abort()}
                      className="rounded-full px-2 py-1 text-[12px] font-bold text-[var(--mobile-publish-muted)] active:bg-[var(--mobile-publish-border-soft)]"
                    >
                      {t("drawer.cancel")}
                    </button>
                  ) : null}
                  <span className="text-[13px] font-bold text-[var(--mobile-publish-accent-strong)]">
                    {uploadProgress.percent}%
                  </span>
                </div>
              </div>
              <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-[var(--mobile-publish-border-soft)]">
                <div
                  className="h-full rounded-full bg-[var(--mobile-publish-accent)] transition-[width]"
                  style={{ width: `${uploadProgress.percent}%` }}
                />
              </div>
              {uploadProgress.detail?.totalChunks ? (
                <p className="mt-1 truncate text-[12px] font-medium text-[var(--mobile-publish-muted)]">
                  {t("publish.mobile.upload.chunk", { progress: formatMobileUploadChunkLabel(uploadProgress.detail) })}
                </p>
              ) : null}
            </div>
          </footer>
        ) : null}
      </div>

      <MobilePublishOverlays
        activeEmojiGroupId={activeEmojiGroupId}
        categories={categories}
        closeWithoutSaving={closeWithoutSaving}
        customTopicInput={customTopicInput}
        handleReplaceVideo={handleReplaceVideo}
        insertEmoji={insertEmoji}
        insertMention={insertMention}
        isSearchingMentionUsers={isSearchingMentionUsers}
        lightboxAsset={lightboxAsset}
        mentionKeyword={mentionKeyword}
        mentionUsers={mentionUsers}
        replaceThumbnailInputRef={replaceThumbnailInputRef}
        saveAndClose={saveAndClose}
        selectedCategoryId={selectedCategoryId}
        setActiveEmojiGroupId={setActiveEmojiGroupId}
        setBoard={setBoard}
        setCustomTopicInput={setCustomTopicInput}
        setIsSearchingMentionUsers={setIsSearchingMentionUsers}
        setLightboxAsset={setLightboxAsset}
        setMentionKeyword={setMentionKeyword}
        setMentionUsers={setMentionUsers}
        setReplacingThumbnailAssetId={setReplacingThumbnailAssetId}
        setSelectedCategoryId={setSelectedCategoryId}
        setShowBoardSheet={setShowBoardSheet}
        setShowCloseConfirm={setShowCloseConfirm}
        setShowEmojiSheet={setShowEmojiSheet}
        setShowMentionSheet={setShowMentionSheet}
        setShowTopicSheet={setShowTopicSheet}
        setTopic={setTopic}
        showBoardSheet={showBoardSheet}
        showCloseConfirm={showCloseConfirm}
        showEmojiSheet={showEmojiSheet}
        showMentionSheet={showMentionSheet}
        showTopicSheet={showTopicSheet}
        submitCustomTopic={submitCustomTopic}
        tags={tags}
        topic={topic}
      />
      <AIFormatPanel
        open={aiFormatOpen}
        value={body}
        variant="mobile"
        onApply={setBody}
        onClose={() => setAIFormatOpen(false)}
      />
      <PublishGenerationPanel
        canRun={publishGenerationCanRun}
        imageCount={publishGenerationImageCount}
        onApply={handleApplyPublishGeneration}
        onCancel={cancelPublishGeneration}
        onClose={() => setPublishGenerationOpen(false)}
        onRun={runPublishGenerationInPanel}
        open={publishGenerationOpen}
        state={publishGeneration}
        t={t}
        variant="mobile"
      />
    </main>
  );
}
