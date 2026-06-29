import {
useLocaleContext
} from "@/components/providers/locale-provider";
import {
getStoredAccessToken,
getStoredUser
} from "@/lib/api";
import type {
AuthUser,
FeedPost,
ReportReason
} from "@/lib/types";
import {
useKeyboardOffset
} from "@/lib/use-keyboard-offset";
import {
getPostAuthorUserId
} from "@/lib/users";
import {
useQueryClient
} from "@tanstack/react-query";
import useEmblaCarousel from "embla-carousel-react";
import {
useTranslations
} from "next-intl";
import {
useRouter
} from "next/navigation";
import {
useMemo,
useCallback,
useRef,
useState
} from "react";
import {
getPostImages,
getPostPreviewImages
} from "../feed-utils";
import { getPostSlideshowMaxImages } from "./post-detail-config";
import type { PostDetailDrawerProps } from "./post-detail-drawer-types";
import {
formatReplyPlaceholder
} from "./post-detail-formatters";
import {
CommentDraftState,
CommentState,
DetailActionState,
FollowState
} from "./post-detail-types";
import { usePaidImageAccess } from "./use-paid-image-access";

export function usePostDetailBase(props: PostDetailDrawerProps) {
  const { post: inputPost, open, onOpenChange, mode = "drawer", targetCommentId, targetCommentParentId } = props;
const router = useRouter();

const queryClient = useQueryClient();

const t = useTranslations();

const {
    handleCreateProtectedPackage,
    handleDownloadImageArchive,
    handleDownloadProtectedPackage,
    handlePurchaseContent,
    handleRefreshImageArchive,
    imageArchiveJob,
    isCreatingProtectedPackage,
    isDownloadingImageArchive,
    isDownloadingProtectedPackage,
    isPurchasingContent,
    isRefreshingImageArchive,
    post,
    protectedPackageJob,
    updateLocalPost,
  } = usePaidImageAccess({ inputPost, open, queryClient, t });

const { locale } = useLocaleContext();

const [isCommentInputFocused, setIsCommentInputFocused] = useState(false);

const keyboardOffset = useKeyboardOffset(isCommentInputFocused);

const [emblaRef, emblaApi] = useEmblaCarousel({ align: "center", loop: true });

const [selectedImageIndex, setSelectedImageIndex] = useState(0);

const [imageViewerState, setImageViewerState] = useState<{
    index: number;
    postId: FeedPost["id"];
  } | null>(null);

const [commentState, setCommentState] = useState<CommentState>({
    comments: [],
    countDelta: 0,
    status: "idle",
  });

const [commentDraft, setCommentDraft] = useState<CommentDraftState>({ text: "" });

const [currentUser] = useState<AuthUser | null>(() =>
    getStoredAccessToken() ? getStoredUser() : null,
  );

const [isSubmittingComment, setIsSubmittingComment] = useState(false);

const [mutatingCommentDeleteIds, setMutatingCommentDeleteIds] = useState<Set<string>>(
    () => new Set(),
  );

const [mutatingCommentLikeIds, setMutatingCommentLikeIds] = useState<Set<string>>(
    () => new Set(),
  );

const commentInputRef = useRef<HTMLTextAreaElement>(null);

const drawerContentRef = useRef<HTMLDivElement>(null);

const [followState, setFollowState] = useState<FollowState>({
    isFollowing: false,
    status: "idle",
  });

const [isMutatingFollow, setIsMutatingFollow] = useState(false);

const [actionsOpen, setActionsOpen] = useState(false);

const [actionState, setActionState] = useState<DetailActionState>({
    disliked: false,
    dislikeStatus: "idle",
    reported: false,
    reportStatus: "idle",
  });

const [isMutatingDislike, setIsMutatingDislike] = useState(false);

const [isMutatingPostLike, setIsMutatingPostLike] = useState(false);

const [isMutatingPostCollect, setIsMutatingPostCollect] = useState(false);

const [isSubmittingReport, setIsSubmittingReport] = useState(false);

const [reportReason, setReportReason] = useState<ReportReason>("spam");

const [reportDescription, setReportDescription] = useState("");

const [shareOpen, setShareOpen] = useState(false);

const [loginUnlockOpen, setLoginUnlockOpen] = useState(false);

const openLoginUnlock = useCallback(() => setLoginUnlockOpen(true), []);

const [isDeletingPost, setIsDeletingPost] = useState(false);

const [highlightedCommentId, setHighlightedCommentId] = useState<string | null>(null);

const expandedForTargetRef = useRef<string | null>(null);

const highlightedCommentTimerRef = useRef<number | null>(null);

const scrolledCommentRef = useRef<string | null>(null);

const postId = post?.id;

const images = post ? getPostImages(post) : [];

const slideshowImages = (post ? getPostPreviewImages(post) : []).slice(0, getPostSlideshowMaxImages());

const imageKey = slideshowImages.join("\n");

const authorUserId = post ? getPostAuthorUserId(post) : undefined;

const comments = useMemo(
    () => (commentState.postId === postId ? commentState.comments : []),
    [commentState.comments, commentState.postId, postId],
  );

const commentText = commentDraft.postId === postId ? commentDraft.text : "";

const replyTarget =
    commentDraft.postId === postId ? (commentDraft.replyTarget ?? null) : null;

const commentInputExpanded = isCommentInputFocused || Boolean(commentText.trim());

const expandedCommentInputHeight = replyTarget
    ? "h-[calc(166px+env(safe-area-inset-bottom))]"
    : "h-[calc(134px+env(safe-area-inset-bottom))]";

const commentPlaceholder = replyTarget
    ? formatReplyPlaceholder(replyTarget.author, locale)
    : t("drawer.saySomething");

const imageViewerIndex =
    imageViewerState && imageViewerState.postId === postId ? imageViewerState.index : null;

const commentCountDelta = commentState.postId === postId ? commentState.countDelta : 0;

const isLoadingComments =
    Boolean(open && postId) &&
    (commentState.postId !== postId || commentState.status === "loading");

const commentFooterStyle =
    keyboardOffset > 0
      ? { transform: `translate3d(0, -${keyboardOffset}px, 0)` }
      : undefined;

  return { actionState, actionsOpen, authorUserId, commentCountDelta, commentDraft, commentFooterStyle, commentInputExpanded, commentInputRef, commentPlaceholder, commentState, commentText, comments, currentUser, drawerContentRef, emblaApi, emblaRef, expandedCommentInputHeight, expandedForTargetRef, followState, handleCreateProtectedPackage, handleDownloadImageArchive, handleDownloadProtectedPackage, handlePurchaseContent, handleRefreshImageArchive, highlightedCommentId, highlightedCommentTimerRef, imageArchiveJob, imageKey, imageViewerIndex, imageViewerState, images, inputPost, isCommentInputFocused, isCreatingProtectedPackage, isDeletingPost, isDownloadingImageArchive, isDownloadingProtectedPackage, isLoadingComments, isMutatingDislike, isMutatingFollow, isMutatingPostCollect, isMutatingPostLike, isPurchasingContent, isRefreshingImageArchive, isSubmittingComment, isSubmittingReport, keyboardOffset, locale, loginUnlockOpen, mode, mutatingCommentDeleteIds, mutatingCommentLikeIds, onOpenChange, open, openLoginUnlock, post, postId, protectedPackageJob, queryClient, replyTarget, reportDescription, reportReason, router, scrolledCommentRef, selectedImageIndex, setActionState, setActionsOpen, setCommentDraft, setCommentState, setFollowState, setHighlightedCommentId, setImageViewerState, setIsCommentInputFocused, setIsDeletingPost, setIsMutatingDislike, setIsMutatingFollow, setIsMutatingPostCollect, setIsMutatingPostLike, setIsSubmittingComment, setIsSubmittingReport, setLoginUnlockOpen, setMutatingCommentDeleteIds, setMutatingCommentLikeIds, setReportDescription, setReportReason, setSelectedImageIndex, setShareOpen, shareOpen, slideshowImages, t, targetCommentId, targetCommentParentId, updateLocalPost };
}
