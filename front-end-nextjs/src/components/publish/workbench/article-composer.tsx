"use client";
import {
  ArrowLeft,
  ArrowRight,
  Heading1,
  Heading2,
  Highlighter,
  ImageIcon,
  List,
  ListOrdered,
  Plus,
  Quote,
  Redo2,
  RotateCcw,
  Smile,
  Undo2,
  X,
  type LucideIcon
} from "lucide-react";
import {
  useTranslations
} from "next-intl";
import {
  Button
} from "@/components/ui/button";
import {
  type UploadProgress
} from "@/lib/api";
import type {
  Category,
  UploadAsset
} from "@/lib/types";
import {
  getCategoryDisplayName
} from "@/components/publish/mobile-drafts";
import {
  cn
} from "@/lib/utils";
import {
  RichTextEditor,
  UploadFailure,
  UploadMode,
  Visibility,
  visibilityOptions
} from "./workbench-config";
import {
  formatFileSize,
  formatUploadChunkLabel,
  formatUploadProgressLabel
} from "./post-builders";
import {
  getUploadAssetPreviewUrl
} from "./upload-panel";

export function UploadStatusList({
  assets,
  failures,
  imageProtectionEnabled,
  mode,
  onMoveAsset,
  onRemoveAsset,
  onRemoveFailure,
  onRetryFailure,
  onToggleImageFree,
  onToggleImageProtection,
  progress,
  progressDetail,
  uploading,
}: {
  assets: UploadAsset[];
  failures: UploadFailure[];
  imageProtectionEnabled?: boolean;
  mode: UploadMode;
  onMoveAsset?: (assetUrl: string, direction: -1 | 1) => void;
  onRemoveAsset: (assetUrl: string) => void;
  onRemoveFailure: (failureId: string) => void;
  onRetryFailure: (failure: UploadFailure) => void;
  onToggleImageFree?: (assetUrl: string) => void;
  onToggleImageProtection?: (assetUrl: string) => void;
  progress: number | null;
  progressDetail: UploadProgress | null;
  uploading: boolean;
}) {
  const t = useTranslations();

  return (
    <>
      {uploading ? (
        <div className="mt-4 w-full max-w-[360px] text-center">
          <p className="text-sm font-medium text-primary">
            {t(`publish.workbench.upload.${formatUploadProgressLabel(progressDetail, progress).key}`, {
              percent: formatUploadProgressLabel(progressDetail, progress).percent,
            })}
          </p>
          {progressDetail?.totalChunks ? (
            <p className="mt-1 text-xs text-[#777780]">
              {t("publish.workbench.upload.chunk", { progress: formatUploadChunkLabel(progressDetail) })}
            </p>
          ) : null}
          <div className="mt-2 h-2 overflow-hidden rounded-full bg-white">
            <div
              className="h-full rounded-full bg-primary transition-[width]"
              style={{ width: `${progress ?? 0}%` }}
            />
          </div>
        </div>
      ) : null}
      {assets.length > 0 ? (
        mode === "image" ? (
          <div className="mt-5 grid w-full grid-cols-[repeat(auto-fit,minmax(120px,1fr))] gap-3 text-left">
            {assets.map((asset, index) => (
              <div
                key={asset.url}
                className="group min-w-0 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-[#eeeeef]"
              >
                <div className="relative aspect-[3/4] bg-[#efeff2]">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={getUploadAssetPreviewUrl(asset)}
                    alt={asset.originalname ?? `Image ${index + 1}`}
                    className="h-full w-full object-cover"
                  />
                  <span className="absolute left-2 top-2 flex h-6 min-w-6 items-center justify-center rounded-full bg-black/55 px-2 text-xs font-semibold text-white">
                    {index + 1}
                  </span>
                  <div className="absolute bottom-2 left-2 flex flex-wrap gap-1">
                    <span className={cn(
                      "rounded-full px-2 py-1 text-[11px] font-semibold text-white shadow-sm",
                      asset.isFreePreview === false ? "bg-[#1f1f24]/80" : "bg-primary/90",
                    )}>
                      {asset.isFreePreview === false
                        ? t("publish.protection.badgePaid")
                        : t("publish.protection.badgeFree")}
                    </span>
                    {asset.isProtected ? (
                      <span className="rounded-full bg-[#2563eb]/90 px-2 py-1 text-[11px] font-semibold text-white shadow-sm">
                        {t("publish.protection.badgeProtected")}
                      </span>
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label={t("publish.protection.removeImage")}
                    onClick={() => onRemoveAsset(asset.url)}
                    className="absolute right-2 top-2 size-7 bg-black/45 text-white hover:bg-black/65 hover:text-white"
                  >
                    <X className="size-4" />
                  </Button>
                </div>
                <div className="flex min-h-12 items-center gap-1 px-2 py-2">
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label={t("publish.protection.moveLeft")}
                    disabled={!onMoveAsset || index === 0 || uploading}
                    onClick={() => onMoveAsset?.(asset.url, -1)}
                    className="size-8 shrink-0 text-[#777780]"
                  >
                    <ArrowLeft className="size-4" />
                  </Button>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs font-medium text-[#34343a]">
                      {asset.originalname ?? asset.url}
                    </span>
                    <span className="block text-[11px] text-[#85858c]">
                      {formatFileSize(asset.size)}
                    </span>
                  </span>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label={t("publish.protection.moveRight")}
                    disabled={!onMoveAsset || index === assets.length - 1 || uploading}
                    onClick={() => onMoveAsset?.(asset.url, 1)}
                    className="size-8 shrink-0 text-[#777780]"
                  >
                    <ArrowRight className="size-4" />
                  </Button>
                </div>
                <div className="grid grid-cols-2 gap-1 border-t border-[#f1f1f2] px-2 pb-2">
                  <button
                    type="button"
                    disabled={uploading}
                    onClick={() => onToggleImageFree?.(asset.url)}
                    className="h-8 rounded-lg bg-[#f6f6f7] text-[11px] font-semibold text-[#55555d] hover:bg-[#eeeeef] disabled:opacity-50"
                  >
                    {asset.isFreePreview === false
                      ? t("publish.protection.setFree")
                      : t("publish.protection.setPaid")}
                  </button>
                  <button
                    type="button"
                    disabled={uploading || !imageProtectionEnabled}
                    onClick={() => onToggleImageProtection?.(asset.url)}
                    className="h-8 rounded-lg bg-[#f6f6f7] text-[11px] font-semibold text-[#55555d] hover:bg-[#eeeeef] disabled:opacity-50"
                  >
                    {asset.isProtected
                      ? t("publish.protection.setDirect")
                      : t("publish.protection.setProtected")}
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="mt-5 w-full max-w-[560px] space-y-2">
            {assets.map((asset) => (
              <div
                key={asset.url}
                className="flex min-h-11 items-center gap-3 rounded-xl bg-white px-3 text-left shadow-sm"
              >
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium text-[#34343a]">
                    {asset.originalname ?? asset.url}
                  </span>
                  <span className="block text-xs text-[#85858c]">
                    {formatFileSize(asset.size)}
                  </span>
                </span>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  aria-label={t("publish.workbench.removeUploadedFile")}
                  onClick={() => onRemoveAsset(asset.url)}
                  className="size-8 text-[#777780]"
                >
                  <X className="size-4" />
                </Button>
              </div>
            ))}
          </div>
        )
      ) : null}
      {failures.length > 0 ? (
        <div className="mt-3 w-full max-w-[560px] space-y-2">
          {failures.map((failure) => (
            <div
              key={failure.id}
              className="flex min-h-11 items-center gap-3 rounded-xl border border-primary/20 bg-white px-3 text-left shadow-sm"
            >
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-[#34343a]">
                  {failure.name}
                </span>
                <span className="block truncate text-xs text-primary">
                  {failure.message}
                </span>
                <span className="block text-xs text-[#85858c]">
                  {formatFileSize(failure.size)}
                </span>
              </span>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={t("publish.workbench.retryUpload")}
                onClick={() => onRetryFailure(failure)}
                disabled={uploading}
                className="size-8 text-[#777780]"
              >
                <RotateCcw className="size-4" />
              </Button>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={t("publish.workbench.removeFailedUpload")}
                onClick={() => onRemoveFailure(failure.id)}
                disabled={uploading}
                className="size-8 text-[#777780]"
              >
                <X className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      ) : null}
    </>
  );
}


export function ArticleComposer({
  body,
  bodyContentLength,
  bodyLimit,
  categories,
  categoryId,
  completion,
  composerOpen,
  onBodyChange,
  onCategoryChange,
  onOpenComposer,
  onTagsChange,
  onTitleChange,
  onVisibilityChange,
  tags,
  title,
  titleLimit,
  visibility,
}: {
  body: string;
  bodyContentLength: number;
  bodyLimit: number;
  categories: Category[];
  categoryId: number | null;
  completion: number;
  composerOpen: boolean;
  onBodyChange: (value: string) => void;
  onCategoryChange: (value: number | null) => void;
  onOpenComposer: () => void;
  onTagsChange: (value: string) => void;
  onTitleChange: (value: string) => void;
  onVisibilityChange: (value: Visibility) => void;
  tags: string;
  title: string;
  titleLimit: number;
  visibility: Visibility;
}) {
  const t = useTranslations();
  const selectedCategory = categories.find((category) => category.id === categoryId);

  if (composerOpen) {
    return (
      <div className="flex min-h-[calc(100vh-252px)] flex-col overflow-hidden rounded-2xl bg-[#f7f7f8]">
        <div className="flex min-h-12 items-center gap-2 border-b border-[#ececf0] bg-white px-3">
          <button
            type="button"
            onClick={onOpenComposer}
            className="flex size-9 shrink-0 items-center justify-center rounded-full text-[#5f5f67] hover:bg-[#f6f6f7] hover:text-[#25252b]"
            aria-label={t("publish.workbench.backToArticleStart")}
          >
            <ArrowLeft className="size-4" />
          </button>
          <div className="hidden h-6 w-px bg-[#ececf0] sm:block" />
          <ArticleToolbarIcon icon={Undo2} label={t("publish.editor.richText.actions.undo")} />
          <ArticleToolbarIcon icon={Redo2} label={t("publish.editor.richText.actions.redo")} />
          <div className="hidden h-6 w-px bg-[#ececf0] sm:block" />
          <ArticleToolbarIcon icon={Heading1} label={t("publish.editor.richText.actions.heading1")} />
          <ArticleToolbarIcon icon={Heading2} label={t("publish.editor.richText.actions.heading2")} />
          <ArticleToolbarIcon icon={List} label={t("publish.editor.richText.actions.bulletList")} />
          <ArticleToolbarIcon icon={ListOrdered} label={t("publish.editor.richText.actions.orderedList")} />
          <ArticleToolbarIcon icon={Quote} label={t("publish.editor.richText.actions.blockquote")} />
          <ArticleToolbarIcon icon={Highlighter} label={t("publish.workbench.mark")} />
          <ArticleToolbarIcon icon={ImageIcon} label={t("publish.workbench.image")} />
          <ArticleToolbarIcon icon={Smile} label={t("publish.workbench.emoji")} />
          <span className="ml-auto hidden min-w-0 truncate text-xs text-[#8a8a91] md:block">
            {t("publish.editor.completion", { percent: completion })}
          </span>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto bg-[#f7f7f8] px-3 py-4 sm:px-5">
          <div className="mx-auto flex min-h-[calc(100vh-360px)] w-full max-w-[920px] flex-col bg-white shadow-[0_8px_28px_rgba(0,0,0,0.04)]">
            <RichTextEditor
              className="flex min-h-0 flex-1 flex-col rounded-none border-0 bg-white focus-within:border-transparent [&_.publish-rich-text-content]:flex-1 [&_.publish-rich-text-content_.ProseMirror]:min-h-[clamp(360px,52vh,760px)] [&_.publish-rich-text-content_.ProseMirror]:px-[clamp(24px,5vw,72px)] [&_.publish-rich-text-content_.ProseMirror]:pb-12 [&_.publish-rich-text-content_.ProseMirror]:pt-4 [&_.publish-rich-text-content_.ProseMirror]:text-base [&_.publish-rich-text-content_.ProseMirror]:leading-8"
              contentBefore={(
                <div className="px-[clamp(24px,5vw,72px)] pb-2 pt-[clamp(28px,6vh,64px)]">
                  <input
                    value={title}
                    onChange={(event) => onTitleChange(event.target.value.slice(0, titleLimit))}
                    placeholder={t("publish.editor.titlePlaceholder")}
                    className="w-full border-0 bg-transparent text-[clamp(28px,4vw,42px)] font-semibold leading-tight tracking-normal text-[#25252b] outline-none placeholder:text-[#c7c7cc]"
                  />
                  <p className="mt-4 text-sm text-[#b0b0b7]">{t("publish.workbench.autosaveHint")}</p>
                </div>
              )}
              hideFooter
              limit={bodyLimit}
              onChange={onBodyChange}
              placeholder={t("publish.editor.articlePlaceholder")}
              value={body}
            />
          </div>
        </div>

        <div className="border-t border-[#ececf0] bg-white px-4 py-3">
          <div className="flex flex-wrap items-center gap-3">
            <span className={cn("text-sm text-[#8a8a91]", bodyContentLength > bodyLimit && "text-primary")}>
              {t("publish.workbench.characterCount", { current: bodyContentLength, limit: bodyLimit })}
            </span>
            <span className="text-xs text-[#b0b0b7]">{title.length}/{titleLimit}</span>

            <label className="flex min-w-[180px] flex-1 items-center gap-2 rounded-xl border border-[#ececf0] bg-[#fafafa] px-3 py-2 text-sm text-[#55555d] md:flex-none">
              <span className="shrink-0 font-semibold">{t("publish.workbench.category")}</span>
              <select
                value={categoryId ?? ""}
                onChange={(event) => {
                  const value = event.target.value;
                  onCategoryChange(value ? Number(value) : null);
                }}
                className="min-w-0 flex-1 bg-transparent text-sm outline-none"
              >
                <option value="">{t("publish.mobile.selectBoard")}</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {getCategoryDisplayName(category)}
                  </option>
                ))}
              </select>
            </label>

            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <span className="text-sm font-semibold text-[#55555d]">{t("publish.settings.visibility")}</span>
              {visibilityOptions.map((option) => (
                <button
                  key={option}
                  type="button"
                  onClick={() => onVisibilityChange(option)}
                  className={cn(
                    "h-9 rounded-full border border-[#e8e8eb] px-3 text-xs font-semibold text-[#66666f] transition-colors hover:border-primary/40",
                    visibility === option && "border-primary bg-[#fff1f3] text-primary",
                  )}
                >
                  {t(`publish.settings.${option}`)}
                </button>
              ))}
            </div>

            <input
              value={tags}
              onChange={(event) => onTagsChange(event.target.value)}
              placeholder={t("publish.editor.tagsPlaceholder")}
              className="h-9 min-w-[180px] flex-1 rounded-full border border-[#e8e8eb] bg-white px-4 text-sm outline-none transition focus:border-primary md:max-w-[260px]"
            />

            <Button
              type="button"
              variant="outline"
              className="ml-auto h-9 border-[#e3e3e6] bg-white px-4 text-[#55555d] hover:bg-[#f7f7f8]"
            >
              {t("publish.workbench.saveAndLeave")}
            </Button>
            <Button type="button" className="h-9 px-4">
              {t("publish.workbench.autoFormat")}
            </Button>
          </div>
          <p className="mt-2 truncate text-xs text-[#a0a0a7]">
            {selectedCategory
              ? t("publish.workbench.selectedCategory", { name: getCategoryDisplayName(selectedCategory) })
              : t("publish.workbench.categoryRequired")}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-[438px] flex-col justify-between rounded-2xl bg-[#f7f7f8] p-6">
      <div>
        <p className="text-lg font-semibold text-[#25252b]">
          {t("publish.article.title")}
        </p>
        <p className="mt-2 max-w-[560px] text-sm leading-6 text-[#777780]">
          {t("publish.article.description")}
        </p>
      </div>

      <div className="grid gap-3 md:max-w-[360px]">
        <button
          className="flex min-h-[150px] flex-col justify-between rounded-2xl bg-white p-5 text-left shadow-sm transition-colors hover:bg-[#fff8fa]"
          type="button"
          onClick={onOpenComposer}
        >
          <span className="flex size-11 items-center justify-center rounded-full bg-[#fff1f3] text-primary">
            <Plus className="size-5" />
          </span>
          <span>
            <span className="block text-base font-semibold text-[#25252b]">
              {t("publish.article.newCreation")}
            </span>
            <span className="mt-1 block text-sm leading-5 text-[#777780]">
              {t("publish.article.newCreationHint")}
            </span>
          </span>
        </button>
      </div>
    </div>
  );
}


export function ArticleToolbarIcon({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className="flex size-9 shrink-0 items-center justify-center rounded-full text-[#62626a] transition-colors hover:bg-[#f6f6f7] hover:text-[#25252b]"
    >
      <Icon className="size-4" />
    </button>
  );
}


export function RichTextEditorSkeleton() {
  return (
    <div
      className="flex min-h-[232px] flex-col rounded-xl border border-[#e8e8eb] bg-white"
      aria-busy="true"
    >
      <div className="flex h-11 items-center gap-2 border-b border-[#eeeeef] px-3">
        {[0, 1, 2, 3, 4].map((item) => (
          <span key={item} className="size-8 animate-pulse rounded-lg bg-[#f0f0f2]" />
        ))}
      </div>
      <div className="flex-1 space-y-3 p-4">
        <div className="h-4 w-4/5 animate-pulse rounded bg-[#ededf0]" />
        <div className="h-4 w-3/5 animate-pulse rounded bg-[#f1f1f3]" />
        <div className="h-4 w-2/3 animate-pulse rounded bg-[#f1f1f3]" />
      </div>
      <div className="h-10 border-t border-[#eeeeef] px-4 py-3">
        <div className="h-3 w-24 animate-pulse rounded bg-[#ededf0]" />
      </div>
    </div>
  );
}
