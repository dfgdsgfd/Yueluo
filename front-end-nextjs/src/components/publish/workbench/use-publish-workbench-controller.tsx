import {
deletePost,
getPostDetail,
getPostProtectionConfig,
logout,
} from "@/lib/api";
import { postContentLength } from "@/lib/post-content";
import {
richTextToPlainText
} from "@/lib/rich-text";
import type {
Category,
FeedPost,
UploadAsset
} from "@/lib/types";
import {
generateVideoThumbnail
} from "@/lib/utils";
import {
useTranslations
} from "next-intl";
import {
useRouter,
useSearchParams,
} from "next/navigation";
import {
useEffect,
useMemo,
useRef,
useState
} from "react";
import {
toast
} from "sonner";
import {
	enforceImageCoverPolicy,
	type ImageAccessPatch,
} from "../shared/image-access";
import {
	usePublishGeneration,
} from "../shared/ai-publish-generation";
import type { PublishGenerationRunOptions } from "../shared/publish-generation-action";
import {
buildUploadFailure,
isLocalAssetUrl,
isValidUploadFile,
movePendingUploadFile,
moveUploadAsset,
revokePendingObjectUrls,
} from "./post-builders";
import { getCachedPublishWorkbenchData } from "./publish-workbench-bootstrap";
import { submitWorkbenchPost } from "./submit-post";
import { useWorkbenchAccountMenuDismiss } from "./workbench-account-menu";
import { loadWorkbenchCategories } from "./workbench-categories";
import { loadWorkbenchDrafts } from "./workbench-drafts";
import { prepareWorkbenchGenerationImages } from "./workbench-generation-upload";
import { createWorkbenchPublishGenerationAction } from "./workbench-publish-generation-action";
import {
  restoreWorkbenchDraftState,
  restoreWorkbenchEditingState,
} from "./workbench-state-restore";
import { replaceWorkbenchThumbnail } from "./workbench-thumbnail";
import {
defaultImagePostLimit,
defaultPaymentMaxPrices,
ImagePaymentSettings,
PaymentMethod,
PendingUploadFile,
PublishMode,
UploadFailure,
UploadMode,
UploadProgressDetailState,
UploadProgressState,
	Visibility,
	WorkspaceSection
} from "./workbench-config";
export function usePublishWorkbenchController() {
const t = useTranslations();
  const {
    cancelPublishGeneration,
    publishGeneration,
    runPublishGenerationStream,
    selectedImageCount: selectedPublishGenerationImageCount,
  } = usePublishGeneration();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [initialBootstrap] = useState(() => getCachedPublishWorkbenchData());
  const [activeSection, setActiveSection] = useState<WorkspaceSection>("home");
  const [mode, setMode] = useState<PublishMode>("video");
  const [visibility, setVisibility] = useState<Visibility>("public");
  const [categories, setCategories] = useState<Category[]>(
    () => initialBootstrap?.categories ?? [],
  );
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [tags, setTags] = useState("");
  const [uploadedAssets, setUploadedAssets] = useState<Record<UploadMode, UploadAsset[]>>({
    image: [],
    video: [],
    podcast: [],
  });
  const [imageProtectionEnabled, setImageProtectionEnabled] = useState(false);
  const [imagePostLimit, setImagePostLimit] = useState(defaultImagePostLimit);
  const [postContentLimit, setPostContentLimit] = useState(100000);
  const [imageProtectionNoticeEnabled, setImageProtectionNoticeEnabled] = useState(true);
  const [imageSelectAllEnabled, setImageSelectAllEnabled] = useState(true);
  const [paidContentPaymentMethods, setPaidContentPaymentMethods] = useState<Record<PaymentMethod, boolean>>({
    balance: true,
    points: true,
  });
  const [paymentMaxPrices, setPaymentMaxPrices] = useState<Record<PaymentMethod, number>>(defaultPaymentMaxPrices);
  const [imagePaymentSettings, setImagePaymentSettings] = useState<ImagePaymentSettings>({
    enabled: false,
    paymentMethod: "balance",
    price: "",
  });
  const [pendingFiles, setPendingFiles] = useState<Record<UploadMode, PendingUploadFile[]>>({
    image: [],
    video: [],
    podcast: [],
  });
  const [uploadFailures, setUploadFailures] = useState<Record<UploadMode, UploadFailure[]>>({
    image: [],
    video: [],
    podcast: [],
  });
  const [uploadProgress, setUploadProgress] = useState<UploadProgressState>({
    image: null,
    video: null,
    podcast: null,
  });
  const [uploadProgressDetails, setUploadProgressDetails] = useState<UploadProgressDetailState>({
    image: null,
    video: null,
    podcast: null,
  });
  const [uploadingMode, setUploadingMode] = useState<UploadMode | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [articleComposerOpen, setArticleComposerOpen] = useState(false);
  const [aiFormatOpen, setAIFormatOpen] = useState(false);
  const [drafts, setDrafts] = useState<FeedPost[]>(
    () => initialBootstrap?.drafts ?? [],
  );
  const [draftsOpen, setDraftsOpen] = useState(false);
  const [draftsLoading, setDraftsLoading] = useState(false);
  const [draftActionId, setDraftActionId] = useState<string | number | null>(null);
  const [currentDraftId, setCurrentDraftId] = useState<string | number | null>(null);
  const [editingPostId, setEditingPostId] = useState<string | number | null>(null);
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const accountMenuRef = useRef<HTMLDivElement | null>(null);
  const replaceThumbnailInputRef = useRef<HTMLInputElement | null>(null);
  const uploadAbortControllerRef = useRef<AbortController | null>(null);

  const titleLimit = mode === "article" ? 80 : 40;
  const bodyLimit = postContentLimit;
  const bodyText = useMemo(() => richTextToPlainText(body), [body]);
  const bodyContentLength = useMemo(() => postContentLength(body), [body]);
  const paidImageCount = uploadedAssets.image.filter((asset) => asset.isFreePreview === false).length;
  const paidContentEnabled = paidContentPaymentMethods.balance || paidContentPaymentMethods.points;
  const completion = useMemo(() => {
    let score = 34;

    if (title.trim()) {
      score += 30;
    }

    if (bodyText.trim()) {
      score += 24;
    }

    if (tags.trim()) {
      score += 12;
    }

    return Math.min(score, 100);
  }, [bodyText, tags, title]);
  const {
    handleApplyPublishGeneration,
    handleGeneratePublishContent: runGeneratePublishContent,
    publishGenerationCanRun,
    publishGenerationImageCount,
  } = createWorkbenchPublishGenerationAction({
    body,
    imageCount: selectedPublishGenerationImageCount(uploadedAssets.image),
    isImageMode: mode === "image",
    publishGeneration, runPublishGenerationStream, setBody, setTitle, tags, title, t,
    uploadedAssets, uploadingMode,
  });
  const handleGeneratePublishContent = (options?: PublishGenerationRunOptions) => runGeneratePublishContent(() =>
    prepareWorkbenchGenerationImages({
      messages: {
        someUploadsFailed: (count) => t("publish.imageManager.someUploadsFailed", { count }),
        uploadFailed: t("publish.imageManager.uploadFailed"),
      },
      pendingFiles,
      setPendingFiles,
      setUploadedAssets,
      setUploadFailures,
      setUploadingMode,
      setUploadProgress,
      setUploadProgressDetails,
      uploadAbortControllerRef,
      uploadedAssets,
    }), options);

  useEffect(() => {
    return () => {
      // eslint-disable-next-line react-hooks/exhaustive-deps -- unmount must abort the latest active upload
      uploadAbortControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (mode === "podcast") {
      const frameId = window.requestAnimationFrame(() => setMode("video"));
      return () => window.cancelAnimationFrame(frameId);
    }
  }, [mode]);

  useEffect(() => {
    void refreshDrafts({ silent: true });
    // Initial draft loading intentionally runs once; explicit refreshes use the same helper.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    let cancelled = false;
    getPostProtectionConfig()
      .then((config) => {
        if (!cancelled) {
          setImageProtectionEnabled(Boolean(config.enabled));
          setImagePostLimit(Math.max(1, Number(config.maxImages) || defaultImagePostLimit));
          setPostContentLimit(Math.max(1, Number(config.maxContentLength) || 100000));
          setImageProtectionNoticeEnabled(config.noticeEnabled !== false);
          setImageSelectAllEnabled(config.selectAllEnabled !== false);
          const methods = {
            balance: config.paymentMethods?.balance !== false,
            points: config.paymentMethods?.points !== false,
          };
          setPaidContentPaymentMethods(methods);
          setPaymentMaxPrices({
            balance: Number(config.paymentMaxPrices?.balance) > 0 ? Number(config.paymentMaxPrices?.balance) : defaultPaymentMaxPrices.balance,
            points: Number(config.paymentMaxPrices?.points) > 0 ? Number(config.paymentMaxPrices?.points) : defaultPaymentMaxPrices.points,
          });
          setImagePaymentSettings((current) => ({
            ...current,
            paymentMethod: methods[current.paymentMethod]
              ? current.paymentMethod
              : methods.balance
                ? "balance"
                : "points",
          }));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setImageProtectionEnabled(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const editId = searchParams.get("edit");
    if (!editId) {
      return;
    }

    let cancelled = false;
    getPostDetail(editId)
      .then((post) => {
        if (cancelled) {
          return;
        }
        restorePostForEditing(post, { notify: true });
      })
      .catch((error) => {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("publish.workbench.loadPostFailed"));
        }
      });

    return () => {
      cancelled = true;
    };
    // The edit query should be consumed when the route changes; restorePostForEditing only writes state.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, t]);

  useWorkbenchAccountMenuDismiss(accountMenuOpen, accountMenuRef, setAccountMenuOpen);


  useEffect(() => {
    let cancelled = false;

    void loadWorkbenchCategories()
      .then((items) => {
        if (!cancelled) {
          setCategories(items);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  function openComposer(nextMode: PublishMode = mode) {
    setMode(nextMode);
    setArticleComposerOpen(false);
    setCurrentDraftId(null);
    setEditingPostId(null);
    setSelectedCategoryId(null);
    setActiveSection("publish");
  }

  async function refreshDrafts(options: { silent?: boolean } = {}) {
    if (!options.silent) {
      setDraftsLoading(true);
    }

    try {
      setDrafts(await loadWorkbenchDrafts());
    } catch (error) {
      if (!options.silent) {
        toast.error(error instanceof Error ? error.message : t("publish.workbench.draftsLoadFailed"));
      }
    } finally {
      if (!options.silent) {
        setDraftsLoading(false);
      }
    }
  }

  function handleOpenDrafts() {
    setDraftsOpen(true);
    void refreshDrafts();
  }

  function resetUploadState() {
    revokePendingObjectUrls(pendingFiles);
    setUploadedAssets({ image: [], video: [], podcast: [] });
    setImagePaymentSettings({ enabled: false, paymentMethod: "balance", price: "" });
    setPendingFiles({ image: [], video: [], podcast: [] });
    setUploadFailures({ image: [], video: [], podcast: [] });
    setUploadProgress({ image: null, video: null, podcast: null });
    setUploadProgressDetails({ image: null, video: null, podcast: null });
    setUploadingMode(null);
  }

  function restoreDraft(draft: FeedPost) {
    restoreWorkbenchDraftState(workbenchRestoreOptions(draft, draft.id));
    toast.success(t("publish.workbench.draftRestored"));
  }

  function restorePostForEditing(post: FeedPost, options: { notify?: boolean } = {}) {
    restoreWorkbenchEditingState(workbenchRestoreOptions(post, null));
    if (options.notify) {
      toast.success(t("publish.workbench.editMode"));
    }
  }

  function workbenchRestoreOptions(post: FeedPost, currentDraftId: string | number | null) {
    return {
      currentDraftId,
      pendingFiles,
      post,
      setActiveSection,
      setArticleComposerOpen,
      setBody,
      setCurrentDraftId,
      setDraftsOpen,
      setEditingPostId,
      setImagePaymentSettings,
      setMode,
      setPendingFiles,
      setSelectedCategoryId,
      setTags,
      setTitle,
      setUploadedAssets,
      setUploadFailures,
      setUploadingMode,
      setUploadProgress,
      setUploadProgressDetails,
      setVisibility,
    };
  }

  async function handleDeleteDraft(draftId: string | number) {
    setDraftActionId(draftId);
    try {
      await deletePost(draftId);
      setDrafts((currentDrafts) => currentDrafts.filter((draft) => draft.id !== draftId));
      setCurrentDraftId((currentId) => currentId === draftId ? null : currentId);
      toast.success(t("publish.workbench.draftDeleted"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("publish.workbench.draftDeleteFailed"));
    } finally {
      setDraftActionId(null);
    }
  }

  function handleUploadFiles(files: FileList | File[], uploadMode: UploadMode) {
    if (uploadingMode) {
      toast.error(t("publish.imageManager.waitForUpload"));
      return;
    }

    if (paidImageCount > 0 && (!paidContentEnabled || !paidContentPaymentMethods[imagePaymentSettings.paymentMethod])) {
      toast.error(t("publish.protection.paymentDisabledToast"));
      return;
    }

    const selectedFiles = Array.from(files);
    if (selectedFiles.length === 0) {
      return;
    }

    setUploadFailures((currentFailures) => ({ ...currentFailures, [uploadMode]: [] }));

    const uploadableFiles =
      uploadMode === "video" || uploadMode === "podcast"
        ? selectedFiles.slice(0, 1)
        : selectedFiles;
    const availableImageSlots = Math.max(0, imagePostLimit - uploadedAssets.image.length);
    const boundedFiles =
      uploadMode === "image" && uploadableFiles.length > availableImageSlots
        ? uploadableFiles.slice(0, availableImageSlots)
        : uploadableFiles;
    if (uploadMode === "image" && uploadableFiles.length > availableImageSlots) {
      toast.error(t("publish.protection.limitToast", { count: imagePostLimit }));
    }
    if (uploadMode === "image" && boundedFiles.length === 0) {
      return;
    }
    const validFiles = boundedFiles.filter((file) => isValidUploadFile(file, uploadMode));
    const invalidFailures = boundedFiles
      .filter((file) => !isValidUploadFile(file, uploadMode))
      .map((file) =>
        buildUploadFailure(file, `${file.name} has an unsupported type or size.`),
      );

    if (invalidFailures.length > 0) {
      setUploadFailures((currentFailures) => ({
        ...currentFailures,
        [uploadMode]: [...currentFailures[uploadMode], ...invalidFailures],
      }));
      toast.error(t("publish.workbench.filesRejected", { count: invalidFailures.length }));
    }

    if (validFiles.length === 0) {
      return;
    }

    const localItems = validFiles.map((file) => {
      const blobUrl = URL.createObjectURL(file);
      return {
        asset: {
          url: blobUrl,
          originalname: file.name,
          size: file.size,
          isFreePreview: true,
          isProtected: false,
          uploadProgress: 0,
          uploadStatus: "queued",
        } satisfies UploadAsset,
        pendingFile: {
          file,
          blobUrl,
        } satisfies PendingUploadFile,
      };
    });
    const localAssets: UploadAsset[] = localItems.map((item) => item.asset);
    const localPendingFiles: PendingUploadFile[] = localItems.map((item) => item.pendingFile);

    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      [uploadMode]:
        uploadMode === "image"
          ? enforceImageCoverPolicy([...currentAssets.image, ...localAssets])
          : localAssets,
    }));
    setPendingFiles((currentPending) => ({
      ...currentPending,
      [uploadMode]:
        uploadMode === "image"
          ? [...currentPending.image, ...localPendingFiles]
          : localPendingFiles,
    }));

    if (uploadMode === "video") {
      for (const { file, blobUrl } of localPendingFiles) {
        void generateVideoThumbnail(file).then((thumbnailDataUrl) => {
          if (!thumbnailDataUrl) {
            return;
          }
          setPendingFiles((prev) => ({
            ...prev,
            video: prev.video.map((item) =>
              item.blobUrl === blobUrl ? { ...item, thumbnailDataUrl } : item,
            ),
          }));
        });
      }
    }

    toast.success(t(`publish.workbench.${uploadMode}Added`));
  }

  function handleRemoveAsset(uploadMode: UploadMode, assetUrl: string) {
    if (uploadingMode === uploadMode) {
      toast.error(t("publish.imageManager.waitForUpload"));
      return;
    }

    if (isLocalAssetUrl(assetUrl)) {
      URL.revokeObjectURL(assetUrl);
    }

    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      [uploadMode]: uploadMode === "image"
        ? enforceImageCoverPolicy(currentAssets.image.filter((asset) => asset.url !== assetUrl))
        : currentAssets[uploadMode].filter((asset) => asset.url !== assetUrl),
    }));
    setPendingFiles((currentPending) => ({
      ...currentPending,
      [uploadMode]: currentPending[uploadMode].filter((item) => item.blobUrl !== assetUrl),
    }));
  }

  function handleRemoveUploadFailure(uploadMode: UploadMode, failureId: string) {
    setUploadFailures((currentFailures) => ({
      ...currentFailures,
      [uploadMode]: currentFailures[uploadMode].filter((failure) => failure.id !== failureId),
    }));
  }

  function handleMoveAsset(uploadMode: UploadMode, assetUrl: string, direction: -1 | 1) {
    if (uploadMode !== "image") {
      return;
    }

    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      image: enforceImageCoverPolicy(moveUploadAsset(currentAssets.image, assetUrl, direction)),
    }));
    setPendingFiles((currentPending) => ({
      ...currentPending,
      image: movePendingUploadFile(currentPending.image, assetUrl, direction),
    }));
  }

  function handleReorderImageAssets(assetUrls: string[]) {
    const order = new Map(assetUrls.map((url, index) => [url, index]));
    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      image: enforceImageCoverPolicy([...currentAssets.image].sort(
        (a, b) => (order.get(a.url) ?? Number.MAX_SAFE_INTEGER) - (order.get(b.url) ?? Number.MAX_SAFE_INTEGER),
      )),
    }));
    setPendingFiles((currentPending) => ({
      ...currentPending,
      image: [...currentPending.image].sort(
        (a, b) => (order.get(a.blobUrl) ?? Number.MAX_SAFE_INTEGER) - (order.get(b.blobUrl) ?? Number.MAX_SAFE_INTEGER),
      ),
    }));
  }

  function handleRemoveImageAssets(assetUrls: string[]) {
    if (uploadingMode === "image") {
      toast.error(t("publish.imageManager.waitForUpload"));
      return;
    }
    const removed = new Set(assetUrls);
    for (const url of removed) {
      if (isLocalAssetUrl(url)) {
        URL.revokeObjectURL(url);
      }
    }
    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      image: enforceImageCoverPolicy(currentAssets.image.filter((asset) => !removed.has(asset.url))),
    }));
    setPendingFiles((currentPending) => ({
      ...currentPending,
      image: currentPending.image.filter((item) => !removed.has(item.blobUrl)),
    }));
  }

  function handleBatchUpdateImages(
    assetUrls: string[],
    flags: ImageAccessPatch,
  ) {
    if (flags.isProtected && !imageProtectionEnabled) {
      toast.error(t("publish.protection.disabledToast"));
      return;
    }
    if (flags.isFreePreview === false && !paidContentEnabled) {
      toast.error(t("publish.protection.paymentDisabledToast"));
      return;
    }
    if (flags.isFreePreview === false) {
      ensureImagePaymentPrice();
    }
    const selected = new Set(assetUrls);
    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      image: enforceImageCoverPolicy(currentAssets.image.map((asset) =>
        selected.has(asset.url) ? { ...asset, ...flags } : asset,
      )),
    }));
  }

  function updateImageAsset(assetUrl: string, updater: (asset: UploadAsset) => UploadAsset) {
    setUploadedAssets((currentAssets) => ({
      ...currentAssets,
      image: enforceImageCoverPolicy(currentAssets.image.map((asset) => asset.url === assetUrl ? updater(asset) : asset)),
    }));
  }

  function ensureImagePaymentPrice() {
    setImagePaymentSettings((current) =>
      current.price.trim() ? current : { ...current, price: "1" },
    );
  }

  function paymentPriceOverLimit(settings = imagePaymentSettings) {
    const price = Number(settings.price) || 0;
    const maxPrice = paymentMaxPrices[settings.paymentMethod] ?? defaultPaymentMaxPrices[settings.paymentMethod];
    return price > maxPrice ? { maxPrice, method: settings.paymentMethod } : null;
  }

  function handleToggleImageFree(assetUrl: string) {
    const currentAsset = uploadedAssets.image.find((asset) => asset.url === assetUrl);
    const nextIsFree = !(currentAsset?.isFreePreview ?? true);
    if (!nextIsFree) {
      if (!paidContentEnabled) {
        toast.error(t("publish.protection.paymentDisabledToast"));
        return;
      }
      ensureImagePaymentPrice();
    }
    updateImageAsset(assetUrl, (asset) => ({
      ...asset,
      isFreePreview: !(asset.isFreePreview ?? true),
    }));
  }

  function handleToggleImageProtection(assetUrl: string) {
    if (!imageProtectionEnabled) {
      toast.error(t("publish.protection.disabledToast"));
      return;
    }
    updateImageAsset(assetUrl, (asset) => ({
      ...asset,
      isProtected: !(asset.isProtected ?? false),
    }));
  }

  function handlePaymentMethodChange(paymentMethod: PaymentMethod) {
    if (!paidContentPaymentMethods[paymentMethod]) {
      toast.error(t("publish.protection.paymentMethodDisabledToast"));
      return;
    }
    setImagePaymentSettings((current) => ({ ...current, paymentMethod }));
  }

  function handleReplaceThumbnail(dataUrl: string) {
    replaceWorkbenchThumbnail(setPendingFiles, dataUrl);
  }

  function handleRetryUpload(uploadMode: UploadMode, failure: UploadFailure) {
    if (uploadingMode) {
      toast.error(t("publish.imageManager.waitForUpload"));
      return;
    }

    if (!isValidUploadFile(failure.file, uploadMode)) {
      toast.error(failure.message);
      return;
    }

    if (
      uploadMode === "image" &&
      uploadedAssets.image.some((asset) => asset.url === failure.id && isLocalAssetUrl(asset.url))
    ) {
      updateImageAsset(failure.id, (asset) => ({
        ...asset,
        uploadError: null,
        uploadProgress: 0,
        uploadStatus: "queued",
      }));
      handleRemoveUploadFailure(uploadMode, failure.id);
      return;
    }

    handleRemoveUploadFailure(uploadMode, failure.id);
    void handleUploadFiles([failure.file], uploadMode);
  }

  async function handleSubmitPost(isDraft: boolean) {
    if (bodyContentLength > bodyLimit) {
      toast.error(t("publish.mobile.contentLimit", { count: bodyLimit }));
      return;
    }
    if (paidImageCount > 0) {
      const overLimit = paymentPriceOverLimit();
      if (overLimit) {
        toast.error(t("publish.protection.priceLimitToast", {
          method: t(`publish.protection.${overLimit.method}`),
          max: overLimit.maxPrice,
        }));
        return;
      }
    }
      await submitWorkbenchPost({
      body,
      currentDraftId,
      editingPostId,
      imagePaymentSettings,
      paymentMaxPrices,
      isDraft,
      isSubmitting,
      mode,
      pendingFiles,
      refreshDrafts,
      resetUploadState,
      router,
      selectedCategoryId,
      setBody,
      setCurrentDraftId,
      setDrafts,
      setEditingPostId,
      setIsSubmitting,
      setPendingFiles,
      setSelectedCategoryId,
      setTags,
      setTitle,
      setUploadedAssets,
      setUploadFailures,
      setUploadingMode,
      setUploadProgress,
      setUploadProgressDetails,
      tags,
      title,
      messages: {
        contentLimit: t("publish.mobile.contentLimit", { count: bodyLimit }),
        draftSaved: t("publish.workbench.draftSaved"),
        updated: t("publish.workbench.updated"),
        publishFailed: t("publish.workbench.publishFailed"),
        published: t("publish.workbench.published"),
        uploadFailed: t("publish.imageManager.uploadFailed"),
        uploadMediaFirst: t("publish.workbench.uploadMediaFirst"),
        waitForUpload: t("publish.imageManager.waitForUpload"),
      },
      translateUploadFailures: (count) =>
        t("publish.imageManager.someUploadsFailed", { count }),
      uploadAbortControllerRef,
      uploadedAssets,
      uploadingMode,
      visibility,
    });
  }

  async function handleLogout() {
    if (isLoggingOut) {
      return;
    }

    setIsLoggingOut(true);
    try {
      await logout();
    } catch (error) {
      console.warn("Logout request failed after local session cleanup.", error);
    } finally {
      setIsLoggingOut(false);
      setAccountMenuOpen(false);
      toast.success(t("publish.header.logoutSuccess"));
      router.replace("/login");
    }
  }

  return { accountMenuOpen, accountMenuRef, activeSection, aiFormatOpen, articleComposerOpen, body, bodyContentLength, bodyLimit, cancelPublishGeneration, categories, completion, draftActionId, drafts, draftsLoading, draftsOpen, handleApplyPublishGeneration, handleBatchUpdateImages, handleDeleteDraft, handleGeneratePublishContent, handleLogout, handleMoveAsset, handleOpenDrafts, handlePaymentMethodChange, handleRemoveAsset, handleRemoveImageAssets, handleRemoveUploadFailure, handleReorderImageAssets, handleReplaceThumbnail, handleRetryUpload, handleSubmitPost, handleToggleImageFree, handleToggleImageProtection, handleUploadFiles, imagePaymentSettings, imageProtectionEnabled, imageProtectionNoticeEnabled, imageSelectAllEnabled, isLoggingOut, isSubmitting, mode, openComposer, paidContentEnabled, paidContentPaymentMethods, paidImageCount, paymentMaxPrices, pendingFiles, publishGeneration, publishGenerationCanRun, publishGenerationImageCount, refreshDrafts, replaceThumbnailInputRef, restoreDraft, selectedCategoryId, setAIFormatOpen, setAccountMenuOpen, setActiveSection, setArticleComposerOpen, setBody, setDraftsOpen, setImagePaymentSettings, setMode, setSelectedCategoryId, setTags, setTitle, setVisibility, t, tags, title, titleLimit, uploadAbortControllerRef, uploadFailures, uploadProgress, uploadProgressDetails, uploadedAssets, uploadingMode, visibility };
}
