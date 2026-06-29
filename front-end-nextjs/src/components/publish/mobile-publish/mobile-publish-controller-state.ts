import { getStoredUser } from "@/lib/api";
import type {
  AuthUser,
  Category,
  PostAttachment,
  PostTag,
  UserSearchResult,
} from "@/lib/types";
import { useRef, useState } from "react";
import { getCachedMobilePublishData } from "./mobile-publish-bootstrap";
import {
  defaultImageLimit,
  defaultMobilePaymentMaxPrices,
  type EmojiGroupId,
  type MobileMarkdownEditorMode,
  type MobileMediaAsset,
  type MobilePaymentMethod,
  type MobileUploadProgressState,
  type Visibility,
} from "./mobile-publish-config";

export function useMobilePublishControllerState(initialUser?: AuthUser | null) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const attachmentInputRef = useRef<HTMLInputElement | null>(null);
  const replaceVideoInputRef = useRef<HTMLInputElement | null>(null);
  const replaceThumbnailInputRef = useRef<HTMLInputElement | null>(null);
  const uploadAbortControllerRef = useRef<AbortController | null>(null);
  const [initialBootstrap] = useState(() => getCachedMobilePublishData());
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(() => initialUser ?? getStoredUser());
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [aiFormatOpen, setAIFormatOpen] = useState(false);
  const [markdownEditorMode, setMarkdownEditorMode] = useState<MobileMarkdownEditorMode>("live");
  const [topic, setTopic] = useState("");
  const [board, setBoard] = useState("");
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null);
  const [visibility, setVisibility] = useState<Visibility>("public");
  const [currentDraftId, setCurrentDraftId] = useState<string | null>(null);
  const [editingPostId, setEditingPostId] = useState<string | null>(null);
  const [editingPostType, setEditingPostType] = useState<number | null>(null);
  const [mediaAssets, setMediaAssets] = useState<MobileMediaAsset[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<MobileUploadProgressState>(null);
  const [categories, setCategories] = useState<Category[]>(() => initialBootstrap?.categories ?? []);
  const [tags, setTagsList] = useState<PostTag[]>(() => initialBootstrap?.tags ?? []);
  const [showBoardSheet, setShowBoardSheet] = useState(false);
  const [showTopicSheet, setShowTopicSheet] = useState(false);
  const [showEmojiSheet, setShowEmojiSheet] = useState(false);
  const [showMentionSheet, setShowMentionSheet] = useState(false);
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);
  const [activeEmojiGroupId, setActiveEmojiGroupId] = useState<EmojiGroupId>("recent");
  const [customTopicInput, setCustomTopicInput] = useState("");
  const [mentionKeyword, setMentionKeyword] = useState("");
  const [mentionUsers, setMentionUsers] = useState<UserSearchResult[]>([]);
  const [isSearchingMentionUsers, setIsSearchingMentionUsers] = useState(false);
  const [attachmentFile, setAttachmentFile] = useState<File | null>(null);
  const [existingAttachment, setExistingAttachment] = useState<PostAttachment | null>(null);
  const [attachmentTouched, setAttachmentTouched] = useState(false);
  const [lightboxAsset, setLightboxAsset] = useState<MobileMediaAsset | null>(null);
  const [replacingThumbnailAssetId, setReplacingThumbnailAssetId] = useState<string | null>(null);
  const [selectedImageIds, setSelectedImageIds] = useState<string[]>([]);
  const [imageProtectionEnabled, setImageProtectionEnabled] = useState(false);
  const [imageLimit, setImageLimit] = useState(defaultImageLimit);
  const [imageProtectionNoticeEnabled, setImageProtectionNoticeEnabled] = useState(true);
  const [imageSelectAllEnabled, setImageSelectAllEnabled] = useState(true);
  const [paidContentPaymentMethods, setPaidContentPaymentMethods] = useState<Record<MobilePaymentMethod, boolean>>({
    balance: true,
    points: true,
  });
  const [paymentMaxPrices, setPaymentMaxPrices] = useState<Record<MobilePaymentMethod, number>>(
    defaultMobilePaymentMaxPrices,
  );
  const [imagePaymentMethod, setImagePaymentMethod] = useState<MobilePaymentMethod>("balance");
  const [imagePrice, setImagePrice] = useState("1");
  const [postContentLimit, setPostContentLimit] = useState(100000);

  return {
    activeEmojiGroupId,
    aiFormatOpen,
    attachmentFile,
    attachmentInputRef,
    attachmentTouched,
    board,
    body,
    categories,
    currentDraftId,
    currentUser,
    customTopicInput,
    editingPostId,
    editingPostType,
    existingAttachment,
    fileInputRef,
    imageLimit,
    imagePaymentMethod,
    imagePrice,
    imageProtectionEnabled,
    imageProtectionNoticeEnabled,
    imageSelectAllEnabled,
    isSearchingMentionUsers,
    isSubmitting,
    lightboxAsset,
    markdownEditorMode,
    mediaAssets,
    mentionKeyword,
    mentionUsers,
    paidContentPaymentMethods,
    paymentMaxPrices,
    postContentLimit,
    replaceThumbnailInputRef,
    replaceVideoInputRef,
    replacingThumbnailAssetId,
    selectedCategoryId,
    selectedImageIds,
    setAIFormatOpen,
    setActiveEmojiGroupId,
    setAttachmentFile,
    setAttachmentTouched,
    setBoard,
    setBody,
    setCategories,
    setCurrentDraftId,
    setCurrentUser,
    setCustomTopicInput,
    setEditingPostId,
    setEditingPostType,
    setExistingAttachment,
    setImageLimit,
    setImagePaymentMethod,
    setImagePrice,
    setImageProtectionEnabled,
    setImageProtectionNoticeEnabled,
    setImageSelectAllEnabled,
    setIsSearchingMentionUsers,
    setIsSubmitting,
    setLightboxAsset,
    setMarkdownEditorMode,
    setMediaAssets,
    setMentionKeyword,
    setMentionUsers,
    setPaidContentPaymentMethods,
    setPaymentMaxPrices,
    setPostContentLimit,
    setReplacingThumbnailAssetId,
    setSelectedCategoryId,
    setSelectedImageIds,
    setShowBoardSheet,
    setShowCloseConfirm,
    setShowEmojiSheet,
    setShowMentionSheet,
    setShowTopicSheet,
    setTagsList,
    setTitle,
    setTopic,
    setUploadProgress,
    setVisibility,
    showBoardSheet,
    showCloseConfirm,
    showEmojiSheet,
    showMentionSheet,
    showTopicSheet,
    tags,
    title,
    topic,
    uploadAbortControllerRef,
    uploadProgress,
    visibility,
  };
}

export type MobilePublishControllerState = ReturnType<typeof useMobilePublishControllerState>;
