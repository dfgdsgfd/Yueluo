import type { AuthUser } from "./auth";
import type { SiteProfile } from "./site";

export type FeedMode = "recommended" | "hot" | "videoCenter" | "following";

export type BackendImage =
  | string
  | {
      url?: string | null;
      isFreePreview?: boolean;
      isProtected?: boolean;
      watermarkTraceToken?: string;
      sortOrder?: number;
      width?: number;
      height?: number;
      preview_url?: string | null;
      thumb_url?: string | null;
      thumbnail_url?: string | null;
    };

export type PaymentSettings = {
  enabled: boolean;
  paymentType?: string;
  paymentMethod?: "balance" | "points";
  freePreviewCount: number;
  previewDuration: number;
  price: number;
  hideAll: boolean;
};

export type PostPurchaseUser = {
  id: number | string;
  buyer?: {
    id?: number | string;
    user_id?: string;
    nickname?: string | null;
    avatar?: string | null;
    verified?: number | boolean | null;
  } | null;
  price: number;
  paid_amount: number;
  discount_rate?: number;
  purchase_type?: string | null;
  payment_method: "balance" | "points";
  purchased_at?: string;
};

export type PostPurchaseUsersPayload = {
  list: PostPurchaseUser[];
  pagination?: FeedPagination;
};

export type PostAttachment = {
  url: string;
  filename?: string | null;
  filesize?: number | null;
};

export type PostVideo = {
  id?: number | string;
  video_url?: string | null;
  cover_url?: string | null;
  dash_url?: string | null;
  preview_video_url?: string | null;
};

export type FeedPost = {
  id: number | string;
  user_id?: number;
  title: string;
  content?: string;
  category_id?: number | null;
  category?: string | null;
  type: number;
  view_count?: number;
  like_count: number;
  collect_count?: number;
  comment_count?: number;
  share_count?: number;
  created_at?: string;
  is_draft?: boolean;
  visibility?: string;
  public_access_exempt?: boolean;
  quality_level?: string | null;
  quality_reward?: number | null;
  quality_marked_at?: string | null;
  original_incentive?: boolean;
  nickname?: string | null;
  user_avatar?: string | null;
  author_account?: string | null;
  author_auto_id?: number | null;
  location?: string | null;
  verified?: number | boolean | null;
  avatar?: string | null;
  author?: string | null;
  tags?: { id: number | string; name: string }[];
  liked: boolean;
  collected: boolean;
  images?: BackendImage[];
  image?: string | null;
  cover_url?: string | null;
  preview_url?: string | null;
  thumb_url?: string | null;
  thumbnail_url?: string | null;
  video_url?: string | null;
  dash_url?: string | null;
  preview_video_url?: string | null;
  videos?: PostVideo[];
  duration?: number;
  friendy_date?: string;
  video_center_url?: string;
  isPaidContent?: boolean;
  hasPurchased?: boolean;
  isAuthor?: boolean;
  totalImagesCount?: number;
  hiddenPaidImagesCount?: number;
  protectedImagesCount?: number;
  protectedFreeImagesCount?: number;
  protectedPaidImagesCount?: number;
  protectedPackageImageCount?: number;
  lockedProtectedImagesCount?: number;
  protectedPackageAvailable?: boolean;
  protectedPackageRequired?: boolean;
  imageArchiveEnabled?: boolean;
  imageArchiveEligible?: boolean;
  imageArchiveThreshold?: number;
  imageArchiveMode?: "shared" | "protected" | string;
  imageArchiveRequiresPurchase?: boolean;
  attachment?: PostAttachment | null;
  paymentSettings?: PaymentSettings | null;
  resourceSectionPosition?: "before_content" | "after_content";
  _recommendationScore?: number;
  points_award?: unknown;
  points_awards?: unknown;
};

export type FeedPagination = {
  page: number;
  limit?: number;
  pageSize?: number;
  total?: number;
  pages?: number;
  hasNextPage?: boolean;
  cursor?: string | null;
};

export type FeedPayload = {
  posts: FeedPost[];
  pagination: FeedPagination;
};

export type SearchType = "all" | "posts" | "videos" | "users";

export type SearchTagStat = {
  id: number | string;
  label: string;
  count: number;
};

export type SearchPostGroup = {
  data: FeedPost[];
  tagStats?: SearchTagStat[];
  pagination: FeedPagination;
};

export type SearchPayload = {
  keyword: string;
  tag?: string;
  type: SearchType;
  data?: FeedPost[];
  tagStats?: SearchTagStat[];
  pagination?: FeedPagination;
  posts?: SearchPostGroup;
  users?: {
    data: AuthUser[];
    pagination: FeedPagination;
  };
};

export type SearchFeedQuery = {
  keyword?: string;
  tag?: string;
  type?: SearchType;
  page?: number;
  limit?: number;
};

export type DownloadPayload = {
  blob: Blob;
  contentDisposition?: string | null;
  contentType?: string | null;
  filename?: string | null;
};

export type FeedQuery = {
  page?: number;
  limit?: number;
  category_id?: number | string | null;
  sort?: string;
};

export type BackendComment = {
  id: number | string;
  post_id?: number | string;
  parent_id?: number | string | null;
  user_id?: number | string;
  content: string;
  created_at?: string;
  like_count?: number;
  liked?: boolean;
  reply_count?: number;
  nickname?: string | null;
  user_avatar?: string | null;
  user_auto_id?: number | string | null;
  user_display_id?: string | null;
  user_location?: string | null;
  verified?: number | boolean | null;
};

export type CommentsPayload = {
  comments: BackendComment[];
  pagination: FeedPagination;
};

export type FollowStatusPayload = {
  followed?: boolean;
  isFollowing?: boolean;
  isMutual?: boolean;
  buttonType?: string;
};

export type DislikeStatusPayload = {
  disliked: boolean;
};

export type ReportStatusPayload = {
  reported: boolean;
};

export type CreateReportPayload = {
  id: number | string;
};

export type ReportReason =
  | "spam"
  | "porn"
  | "violence"
  | "fake"
  | "harassment"
  | "copyright"
  | "other";

export type UploadAsset = {
  batchIndex?: number;
  clientId?: string;
  originalname?: string;
  size?: number;
  url: string;
  signedUrl?: string;
  format?: string;
  contentType?: string;
  processedSize?: number;
  width?: number;
  height?: number;
  uploadAssetId?: number;
  uploadAssetPurpose?: string;
  uploadAssetExpiresAt?: string | null;
  watermarkApplied?: boolean;
  watermarkTraceToken?: string;
  watermarkPayloadBytes?: number;
  watermarkEngine?: string;
  filePath?: string;
  coverUrl?: string | null;
  coverSignedUrl?: string | null;
  transcoding?: boolean;
  isFreePreview?: boolean;
  isProtected?: boolean;
  uploadError?: string | null;
  uploadProgress?: number;
  uploadStatus?: "queued" | "uploading" | "succeeded" | "failed";
};

export type HiddenWatermarkResult = {
  customText?: string;
  found?: boolean;
  includedFields?: string[];
  jobId?: string;
  postId?: number;
  traceToken?: string;
  traceType?: string;
  traceResolved?: boolean;
  payloadBytes?: number;
  payloadBits?: number;
  payloadFormat?: string;
  watermarkEngine?: string;
  imageId?: number;
  sourceHash?: string;
  uid?: number;
  uploadedAt?: string;
  userId?: string;
  username?: string;
  valid?: boolean;
  version?: number;
};

export type PublishPostInput = {
  title: string;
  content: string;
  type: number;
  category_id?: number | null;
  tags?: string[];
  is_draft?: boolean;
  visibility?: string;
  images?: { url: string; watermarkTraceToken?: string; isFreePreview?: boolean; isProtected?: boolean; sortOrder?: number }[];
  video?: { url: string; coverUrl?: string | null } | null;
  attachment?: { url: string; filename?: string; filesize?: number } | null;
  paymentSettings?: {
    enabled: boolean;
    paymentType?: string;
    paymentMethod?: "balance" | "points";
    price?: number;
    freePreviewCount?: number;
    previewDuration?: number;
    hideAll?: boolean;
  };
};

export type ProtectedPackageJob = {
  jobId: string;
  postId: number | string;
  status: "queued" | "processing" | "completed" | "failed" | "expired" | string;
  progress: number;
  queuePosition?: number | null;
  queueCount?: number;
  estimatedWaitSeconds?: number | null;
  estimatedRemainingSeconds?: number | null;
  elapsedSeconds?: number;
  protectedImageCount?: number;
  processedImageCount?: number;
  currentImageIndex?: number;
  currentStep?: string;
  activeProfile?: string | null;
  heartbeatAt?: string | null;
  downloadUrl?: string | null;
  errorMessage?: string | null;
  errorCode?: string | null;
  retryable?: boolean;
  expiresAt?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type ProtectedPackageEvent = ProtectedPackageJob & {
  type: "progress" | "heartbeat" | "result" | "error" | string;
};

export type PostImageArchiveJob = ProtectedPackageJob & {
  eligible: boolean;
  enabled: boolean;
  imageCount: number;
  mode: "shared" | "protected" | string;
  requiresPurchase: boolean;
  threshold: number;
};

export type PurchaseContentResult = {
  alreadyPurchased?: boolean;
  authorEarnings?: number;
  balanceAfter?: number;
  couponDiscount?: number;
  discountRate?: number;
  messageKey?: string;
  orderId?: number | string;
  paidAmount?: number;
  paymentMethod: "balance" | "points";
  platformFee?: number;
  postId?: number | string;
  price?: number;
  status: "completed" | "processing" | "failed" | string;
};

export type Category = {
  id: number;
  name: string;
  category_title?: string | null;
  display_name?: string;
  translations?: Partial<Record<"en" | "zh-CN" | "zh-TW" | "vi" | "ja" | "ko", string>>;
  use_count?: number;
};

export type PostTag = {
  id: number;
  name: string;
  use_count?: number;
};

export type InitialFeedData = {
  posts: FeedPost[];
  categories: Category[];
  pagination: FeedPagination;
  source: "backend" | "fixture";
  backendNotice?: string;
  toolbarItems?: UserToolbarItem[];
  viewer?: {
    createdAt?: string | null;
  };
  siteProfile?: SiteProfile;
  videoCenter?: VideoCenterVisibilityConfig;
};

export type UserToolbarItem = {
  icon?: string | null;
  id: number | string;
  name: string;
  sort_order?: number;
  url: string;
};

export type AppDownloadPlatform = {
  bundle_id?: string;
  download_url?: string;
  enabled?: boolean;
  name?: string;
  package_name?: string;
  release_notes?: string;
  size_bytes?: number | string;
  size_label?: string;
  version_code?: number | string;
  version_name?: string;
};

export type AppDownloadConfig = {
  android?: AppDownloadPlatform;
  android_fast?: AppDownloadPlatform;
  ios?: AppDownloadPlatform;
};

export type VideoCenterVisibilityConfig = {
  enabled: boolean;
  accountCutoff?: string | null;
  guestRestricted?: boolean;
  requestCountryCode?: string | null;
};
