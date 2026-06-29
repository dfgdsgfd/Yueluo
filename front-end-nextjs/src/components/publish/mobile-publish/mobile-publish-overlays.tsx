"use client";

import type { Dispatch, RefObject, SetStateAction } from "react";
import Image from "next/image";
import { useTranslations } from "next-intl";
import { ImageIcon, RefreshCw, X } from "lucide-react";
import { getCategoryDisplayName } from "../mobile-drafts";
import type { Category, PostTag, UserSearchResult } from "@/lib/types";
import { cn } from "@/lib/utils";
import { type EmojiGroupId, type MobileMediaAsset, emojiGroups, formatMobileTopics, parseMobileTopics } from "./mobile-publish-config";
import { getDisplayName } from "./mobile-publish-utils";

type MobilePublishOverlaysProps = {
  activeEmojiGroupId: EmojiGroupId;
  categories: Category[];
  closeWithoutSaving: () => void;
  customTopicInput: string;
  handleReplaceVideo: () => void;
  insertEmoji: (emoji: string) => void;
  insertMention: (user: UserSearchResult) => void;
  isSearchingMentionUsers: boolean;
  lightboxAsset: MobileMediaAsset | null;
  mentionKeyword: string;
  mentionUsers: UserSearchResult[];
  replaceThumbnailInputRef: RefObject<HTMLInputElement | null>;
  saveAndClose: () => Promise<void>;
  selectedCategoryId: number | null;
  setActiveEmojiGroupId: Dispatch<SetStateAction<EmojiGroupId>>;
  setBoard: Dispatch<SetStateAction<string>>;
  setCustomTopicInput: Dispatch<SetStateAction<string>>;
  setIsSearchingMentionUsers: Dispatch<SetStateAction<boolean>>;
  setLightboxAsset: Dispatch<SetStateAction<MobileMediaAsset | null>>;
  setMentionKeyword: Dispatch<SetStateAction<string>>;
  setMentionUsers: Dispatch<SetStateAction<UserSearchResult[]>>;
  setReplacingThumbnailAssetId: Dispatch<SetStateAction<string | null>>;
  setSelectedCategoryId: Dispatch<SetStateAction<number | null>>;
  setShowBoardSheet: Dispatch<SetStateAction<boolean>>;
  setShowCloseConfirm: Dispatch<SetStateAction<boolean>>;
  setShowEmojiSheet: Dispatch<SetStateAction<boolean>>;
  setShowMentionSheet: Dispatch<SetStateAction<boolean>>;
  setShowTopicSheet: Dispatch<SetStateAction<boolean>>;
  setTopic: Dispatch<SetStateAction<string>>;
  showBoardSheet: boolean;
  showCloseConfirm: boolean;
  showEmojiSheet: boolean;
  showMentionSheet: boolean;
  showTopicSheet: boolean;
  submitCustomTopic: () => void;
  tags: PostTag[];
  topic: string;
};

export function MobilePublishOverlays({
  activeEmojiGroupId,
  categories,
  closeWithoutSaving,
  customTopicInput,
  handleReplaceVideo,
  insertEmoji,
  insertMention,
  isSearchingMentionUsers,
  lightboxAsset,
  mentionKeyword,
  mentionUsers,
  replaceThumbnailInputRef,
  saveAndClose,
  selectedCategoryId,
  setActiveEmojiGroupId,
  setBoard,
  setCustomTopicInput,
  setIsSearchingMentionUsers,
  setLightboxAsset,
  setMentionKeyword,
  setMentionUsers,
  setReplacingThumbnailAssetId,
  setSelectedCategoryId,
  setShowBoardSheet,
  setShowCloseConfirm,
  setShowEmojiSheet,
  setShowMentionSheet,
  setShowTopicSheet,
  setTopic,
  showBoardSheet,
  showCloseConfirm,
  showEmojiSheet,
  showMentionSheet,
  showTopicSheet,
  submitCustomTopic,
  tags,
  topic,
}: MobilePublishOverlaysProps) {
  const t = useTranslations("publish.mobile");
  const selectedTopics = parseMobileTopics(topic);
  return (
    <>
      {/* Board / Category selection sheet */}
      {showBoardSheet ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center"
          onClick={() => setShowBoardSheet(false)}
        >
          <div className="absolute inset-0 bg-black/40" />
          <div
            className="relative w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] pb-[env(safe-area-inset-bottom)] shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-5 pb-3 pt-5">
              <h2 className="text-[20px] font-bold text-[var(--mobile-publish-heading)]">{t("selectBoard")}</h2>
              <button
                type="button"
                onClick={() => setShowBoardSheet(false)}
                className="flex size-9 items-center justify-center rounded-full active:bg-[var(--mobile-publish-accent-soft)]"
              >
                <X className="size-6 text-[var(--mobile-publish-muted)]" />
              </button>
            </div>
            <div className="max-h-[60vh] overflow-y-auto px-5 pb-6">
              {categories.length === 0 ? (
                <p className="py-6 text-center text-[15px] text-[var(--mobile-publish-muted)]">{t("noBoards")}</p>
              ) : (
                <div className="grid grid-cols-2 gap-3">
                  {categories.map((cat) => {
                    const categoryDisplayName = getCategoryDisplayName(cat);
                    return (
                      <button
                        key={cat.id}
                        type="button"
                        onClick={() => {
                          setBoard(categoryDisplayName);
                          setSelectedCategoryId(cat.id);
                          setShowBoardSheet(false);
                        }}
                        className={cn(
                          "flex h-[52px] items-center justify-center rounded-[14px] border text-[17px] font-semibold transition-colors active:bg-[var(--mobile-publish-accent-soft)]",
                          selectedCategoryId === cat.id
                            ? "border-[var(--mobile-publish-accent)] bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]"
                            : "border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-text)]",
                        )}
                      >
                        {categoryDisplayName}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      ) : null}

      {/* Topic / Tag selection sheet */}
      {showTopicSheet ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center"
          onClick={() => setShowTopicSheet(false)}
        >
          <div className="absolute inset-0 bg-black/40" />
          <div
            className="relative w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] pb-[env(safe-area-inset-bottom)] shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-5 pb-3 pt-5">
              <h2 className="text-[20px] font-bold text-[var(--mobile-publish-heading)]">{t("addTopic")}</h2>
              <button
                type="button"
                onClick={() => setShowTopicSheet(false)}
                className="flex size-9 items-center justify-center rounded-full active:bg-[var(--mobile-publish-accent-soft)]"
              >
                <X className="size-6 text-[var(--mobile-publish-muted)]" />
              </button>
            </div>
            <div className="px-5 pb-4">
              <form
                className="flex gap-2"
                onSubmit={(event) => {
                  event.preventDefault();
                  submitCustomTopic();
                }}
              >
                <label htmlFor="mobile-custom-topic" className="sr-only">
                  {t("customTopic")}
                </label>
                <div className="flex h-11 min-w-0 flex-1 items-center rounded-full border border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] px-4">
                  <span className="mr-2 shrink-0 text-[17px] font-bold text-[var(--mobile-publish-accent-strong)]">#</span>
                  <input
                    id="mobile-custom-topic"
                    value={customTopicInput}
                    onChange={(event) => setCustomTopicInput(event.target.value.slice(0, 30))}
                    placeholder={t("customTopicPlaceholder")}
                    className="min-w-0 flex-1 bg-transparent text-[16px] font-medium text-[var(--mobile-publish-input)] outline-none placeholder:text-[var(--mobile-publish-subtle)]"
                  />
                </div>
                <button
                  type="submit"
                  disabled={!customTopicInput.trim()}
                  className="flex h-11 shrink-0 items-center rounded-full bg-[var(--mobile-publish-accent)] px-5 text-[15px] font-bold text-white active:opacity-85 disabled:opacity-50"
                >
                  {t("add")}
                </button>
              </form>
            </div>
            <div className="max-h-[60vh] overflow-y-auto px-5 pb-6">
              {topic ? (
                <button
                  type="button"
                  onClick={() => {
                    setTopic("");
                  }}
                  className="mb-3 flex h-10 items-center gap-2 rounded-full border border-[var(--mobile-publish-border)] px-4 text-[15px] text-[var(--mobile-publish-muted)] active:bg-[var(--mobile-publish-accent-soft)]"
                >
                  <X className="size-4" />
                  {t("clearTopic", { topic })}
                </button>
              ) : null}
              {tags.length === 0 ? (
                <p className="py-6 text-center text-[15px] text-[var(--mobile-publish-muted)]">{t("noTopics")}</p>
              ) : (
                <div className="flex flex-wrap gap-3">
                  {tags.map((tag) => (
                    <button
                      key={tag.id}
                      type="button"
                      onClick={() => {
                        setTopic((current) => {
                          const currentTopics = parseMobileTopics(current);
                          const nextTopics = currentTopics.includes(tag.name)
                            ? currentTopics.filter((item) => item !== tag.name)
                            : [...currentTopics, tag.name].slice(0, 12);
                          return formatMobileTopics(nextTopics);
                        });
                      }}
                      className={cn(
                        "flex h-10 items-center rounded-full border px-4 text-[16px] font-semibold transition-colors active:bg-[var(--mobile-publish-accent-soft)]",
                        selectedTopics.includes(tag.name)
                          ? "border-[var(--mobile-publish-accent)] bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]"
                          : "border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-text)]",
                      )}
                    >
                      # {tag.name}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      ) : null}

      {showEmojiSheet ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center"
          onClick={() => setShowEmojiSheet(false)}
        >
          <div className="absolute inset-0 bg-black/40" />
          <div
            className="relative w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] pb-[env(safe-area-inset-bottom)] shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-5 pb-3 pt-5">
              <h2 className="text-[20px] font-bold text-[var(--mobile-publish-heading)]">{t("addEmoji")}</h2>
              <button
                type="button"
                aria-label={t("closeEmoji")}
                onClick={() => setShowEmojiSheet(false)}
                className="flex size-9 items-center justify-center rounded-full active:bg-[var(--mobile-publish-accent-soft)]"
              >
                <X className="size-6 text-[var(--mobile-publish-muted)]" />
              </button>
            </div>

            <div className="px-5 pb-5">
              <div className="flex gap-2 overflow-x-auto pb-2">
                {emojiGroups.map((group) => (
                  <button
                    key={group.id}
                    type="button"
                    onClick={() => setActiveEmojiGroupId(group.id)}
                    className={cn(
                      "flex h-9 shrink-0 items-center rounded-full border px-4 text-[15px] font-semibold transition-colors active:bg-[var(--mobile-publish-accent-soft)]",
                      activeEmojiGroupId === group.id
                        ? "border-[var(--mobile-publish-accent)] bg-[var(--mobile-publish-accent-soft)] text-[var(--mobile-publish-accent-strong)]"
                        : "border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] text-[var(--mobile-publish-muted)]",
                    )}
                  >
                    {t(group.labelKey)}
                  </button>
                ))}
              </div>

              <div
                className="mt-3 grid max-h-[38vh] gap-2 overflow-y-auto"
                style={{ gridTemplateColumns: "repeat(auto-fit, minmax(40px, 1fr))" }}
              >
                {(emojiGroups.find((group) => group.id === activeEmojiGroupId) ?? emojiGroups[0]).emojis.map((emoji) => (
                  <button
                    key={emoji}
                    type="button"
                    aria-label={t("insertEmoji", { emoji })}
                    onClick={() => insertEmoji(emoji)}
                    className="flex aspect-square min-h-10 items-center justify-center rounded-[12px] bg-[var(--mobile-publish-card)] text-[26px] shadow-[var(--mobile-publish-shadow)] transition-transform active:scale-95 active:bg-[var(--mobile-publish-accent-soft)]"
                  >
                    <span aria-hidden="true">{emoji}</span>
                  </button>
                ))}
              </div>

              <div className="mt-4 flex items-center justify-between gap-3">
                <p className="min-w-0 text-[13px] font-medium text-[var(--mobile-publish-muted)]">
                  {t("emojiHint")}
                </p>
                <button
                  type="button"
                  onClick={() => setShowEmojiSheet(false)}
                  className="flex h-10 shrink-0 items-center rounded-full bg-[var(--mobile-publish-accent)] px-5 text-[15px] font-bold text-white active:opacity-85"
                >
                  {t("done")}
                </button>
              </div>
            </div>
          </div>
        </div>
      ) : null}

      {showMentionSheet ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center"
          onClick={() => setShowMentionSheet(false)}
        >
          <div className="absolute inset-0 bg-black/40" />
          <div
            className="relative w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] pb-[env(safe-area-inset-bottom)] shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-5 pb-3 pt-5">
              <h2 className="text-[20px] font-bold text-[var(--mobile-publish-heading)]">{t("mentionUser")}</h2>
              <button
                type="button"
                onClick={() => setShowMentionSheet(false)}
                className="flex size-9 items-center justify-center rounded-full active:bg-[var(--mobile-publish-accent-soft)]"
              >
                <X className="size-6 text-[var(--mobile-publish-muted)]" />
              </button>
            </div>
            <div className="px-5 pb-6">
              <input
                value={mentionKeyword}
                onChange={(event) => {
                  const nextValue = event.target.value;
                  setMentionKeyword(nextValue);
                  setMentionUsers([]);
                  setIsSearchingMentionUsers(true);
                }}
                autoFocus
                placeholder={t("mentionPlaceholder")}
                className="h-11 w-full rounded-full border border-[var(--mobile-publish-border)] bg-[var(--mobile-publish-card)] px-4 text-[16px] outline-none placeholder:text-[var(--mobile-publish-subtle)]"
              />
              <div className="mt-4 max-h-[46vh] overflow-y-auto">
                {isSearchingMentionUsers ? (
                  <p className="py-5 text-center text-[15px] text-[var(--mobile-publish-muted)]">Searching...</p>
                ) : mentionUsers.length > 0 ? (
                  <div className="space-y-2">
                    {mentionUsers.map((user) => (
                      <button
                        key={`${user.id}-${user.user_id ?? ""}`}
                        type="button"
                        onClick={() => insertMention(user)}
                        className="flex w-full items-center gap-3 rounded-[14px] px-2 py-2 text-left active:bg-[var(--mobile-publish-accent-soft)]"
                      >
                        <span className="relative flex size-11 shrink-0 overflow-hidden rounded-full bg-[var(--mobile-publish-card-strong)]">
                          {user.avatar ? (
                            <Image src={user.avatar} alt={getDisplayName(user)} fill unoptimized sizes="44px" className="object-cover" />
                          ) : (
                            <span className="flex size-full items-center justify-center text-base font-bold text-[var(--mobile-publish-accent-strong)]">
                              {getDisplayName(user).charAt(0).toUpperCase()}
                            </span>
                          )}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[16px] font-semibold text-[var(--mobile-publish-input)]">{getDisplayName(user)}</span>
                          <span className="block truncate text-xs text-[var(--mobile-publish-muted)]">{user.user_id ?? user.xise_id ?? user.id}</span>
                        </span>
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="py-5 text-center text-[15px] text-[var(--mobile-publish-muted)]">
                    {mentionKeyword.trim() ? t("mentionNotFound") : t("mentionEmpty")}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>
      ) : null}

      {showCloseConfirm ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center"
          onClick={() => setShowCloseConfirm(false)}
        >
          <div className="absolute inset-0 bg-black/45" />
          <div
            className="relative w-full max-w-[430px] rounded-t-[24px] bg-[var(--mobile-publish-surface)] px-5 pb-[calc(18px+env(safe-area-inset-bottom))] pt-5 shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <h2 className="text-[20px] font-bold text-[var(--mobile-publish-heading)]">{t("saveDraftPrompt")}</h2>
            <p className="mt-2 text-[14px] leading-6 text-[var(--mobile-publish-muted)]">
              {t("saveDraftDescription")}
            </p>
            <div className="mt-5 space-y-3">
              <button
                type="button"
                onClick={() => void saveAndClose()}
                className="flex h-12 w-full items-center justify-center rounded-full bg-[var(--mobile-publish-accent)] text-[16px] font-bold text-white active:opacity-85"
              >
                {t("saveDraft")}
              </button>
              <button
                type="button"
                onClick={closeWithoutSaving}
                className="flex h-12 w-full items-center justify-center rounded-full border border-[var(--mobile-publish-border)] text-[16px] font-bold text-[var(--mobile-publish-text)] active:bg-[var(--mobile-publish-accent-soft)]"
              >
                {t("discard")}
              </button>
              <button
                type="button"
                onClick={() => setShowCloseConfirm(false)}
                className="flex h-11 w-full items-center justify-center rounded-full text-[15px] font-semibold text-[var(--mobile-publish-muted)] active:bg-[var(--mobile-publish-accent-soft)]"
              >
                {t("cancel")}
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {/* Media lightbox */}
      {lightboxAsset ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/90"
          onClick={() => setLightboxAsset(null)}
        >
          <button
            type="button"
            aria-label={t("closePreview")}
            onClick={() => setLightboxAsset(null)}
            className="absolute right-4 top-4 flex size-10 items-center justify-center rounded-full bg-black/50 text-white"
          >
            <X className="size-6" />
          </button>
          {lightboxAsset.kind === "image" ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={lightboxAsset.previewUrl}
              alt={lightboxAsset.name}
              className="max-h-[90dvh] max-w-[90vw] rounded-xl object-contain"
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <div
              className="flex max-h-[90dvh] max-w-[90vw] flex-col items-center gap-4"
              onClick={(e) => e.stopPropagation()}
            >
              {lightboxAsset.previewUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={lightboxAsset.previewUrl}
                  alt={lightboxAsset.name}
                  className="max-h-[70dvh] max-w-[90vw] rounded-xl object-contain"
                />
              ) : null}
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => {
                    const assetId = lightboxAsset.id;
                    setLightboxAsset(null);
                    setReplacingThumbnailAssetId(assetId);
                    replaceThumbnailInputRef.current?.click();
                  }}
                  className="flex h-10 items-center gap-2 rounded-full bg-white/10 px-5 text-[15px] font-semibold text-white"
                >
                  <ImageIcon className="size-4" />
                  {t("replaceCover")}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setLightboxAsset(null);
                    handleReplaceVideo();
                  }}
                  className="flex h-10 items-center gap-2 rounded-full bg-white/10 px-5 text-[15px] font-semibold text-white"
                >
                  <RefreshCw className="size-4" />
                  {t("replaceVideo")}
                </button>
              </div>
            </div>
          )}
        </div>
      ) : null}

    </>
  );
}
