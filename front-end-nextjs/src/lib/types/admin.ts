import type { AuthTokens } from "./auth";
import type { CreatorListPagination } from "./creator";
import type { WithdrawOrderItem } from "./wallet";

export type AdminUser = {
  id: number | string;
  username: string;
  created_at?: string;
};

export type AdminSession = {
  admin: AdminUser;
  tokens: AuthTokens;
};

export type AdminStatsOverviewPayload = {
  users: number;
  posts: number;
  comments: number;
  reports: number;
  feedback: number;
  announcements: number;
};

export type AdminAiReviewStatusPayload = {
  enabled: boolean;
  username_enabled?: boolean;
  content_enabled?: boolean;
};

export type AdminGuestAccessStatusPayload = {
  restricted: boolean;
  note_restricted?: boolean;
  video_restricted?: boolean;
  admin_restricted?: boolean;
};

export type SystemUpdateConfigPayload = {
  id?: number | string;
  frontend_repo_url: string;
  backend_repo_url: string;
  github_token_set?: boolean;
  github_token_masked?: string;
  frontend_branch: string;
  backend_branch: string;
  frontend_release_tag?: string;
  backend_release_tag?: string;
  frontend_artifact_url?: string;
  backend_artifact_url?: string;
  frontend_asset_pattern?: string;
  backend_asset_pattern?: string;
  frontend_install_dir?: string;
  backend_install_path?: string;
  frontend_start_mode?: string;
  frontend_start_command?: string;
  frontend_source_dir: string;
  backend_source_dir: string;
  artifact_dir: string;
  frontend_build_command: string;
  backend_build_command: string;
  updated_at?: string;
};

export type SystemUpdateToolStatus = {
  name: string;
  ok: boolean;
  path?: string;
  version?: string;
  message?: string;
};

export type SystemUpdateCheck = {
  key: string;
  label: string;
  status: string;
  message: string;
  path?: string;
};

export type SystemUpdateJobPayload = {
  id: number | string;
  action: string;
  status: string;
  frontend_state?: string;
  backend_state?: string;
  progress?: number;
  current_step?: string;
  artifact_paths?: string[];
  artifacts?: Array<{
    kind: string;
    name: string;
    source_url?: string;
    cache_path?: string;
    install_path?: string;
    size_bytes?: number;
    sha256?: string;
    release_tag?: string;
    github_updated_at?: string;
    downloaded_at?: string;
    installed_at?: string;
  }>;
  logs?: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
  updated_at?: string;
};

export type SystemUpdateReleaseAssetPayload = {
  name: string;
  size_bytes?: number;
  download_url?: string;
  updated_at?: string;
  sha256?: string;
  matched?: boolean;
};

export type SystemUpdateReleaseOptionPayload = {
  tag_name: string;
  name?: string;
  html_url?: string;
  target_commit?: string;
  commit_hash?: string;
  created_at?: string;
  published_at?: string;
  matching_assets?: SystemUpdateReleaseAssetPayload[];
  assets?: SystemUpdateReleaseAssetPayload[];
};

export type SystemUpdateReleaseOptionsPayload = {
  frontend?: SystemUpdateReleaseOptionPayload[];
  backend?: SystemUpdateReleaseOptionPayload[];
  frontend_error?: string;
  backend_error?: string;
};

export type SystemUpdateStatusPayload = {
  config: SystemUpdateConfigPayload;
  environment: {
    os: string;
    arch: string;
    cwd?: string;
    tools?: SystemUpdateToolStatus[];
  };
  current?: {
    backend?: SystemUpdateVersionPayload;
    frontend?: SystemUpdateVersionPayload | null;
  };
  checks?: SystemUpdateCheck[];
  component_checks?: SystemUpdateCheck[];
  last_job?: SystemUpdateJobPayload | null;
};

export type SystemUpdateVersionPayload = {
  version_tag?: string;
  commit_hash?: string;
  build_time?: string;
  github_run_id?: string;
  source?: string;
  path?: string;
};

export type AdminListResource =
  | "admins"
  | "ai-moderation-logs"
  | "announcements"
  | "app-versions"
  | "audit"
  | "banned-word-categories"
  | "banned-words"
  | "categories"
  | "collections"
  | "comments"
  | "users"
  | "feedback"
  | "file-recycle-bin"
  | "follows"
  | "licenses"
  | "likes"
  | "media-library"
  | "notification-templates"
  | "open-apis"
  | "posts"
  | "posts-quality"
  | "post-configs"
  | "quality-reward-settings"
  | "content-review"
  | "reports"
  | "sessions"
  | "system-notifications"
  | "tags"
  | "user-configs"
  | "user-toolbar";

export type AdminListRow = Record<string, unknown> & {
  id?: number | string;
  created_at?: string;
  updated_at?: string | null;
};

export type AdminListPayload<T extends AdminListRow = AdminListRow> = {
  items: T[];
  pagination: CreatorListPagination;
};

export type FileRecyclePathState = {
  configured: boolean;
  exists: boolean;
  is_dir: boolean;
  size_bytes: number;
  file_count: number;
  unsafe: boolean;
  path?: string;
};

export type FileRecycleDirEntry = {
  name: string;
  is_dir: boolean;
  size_bytes: number;
};

export type FileRecycleInspectPayload = {
  item: AdminListRow;
  original: FileRecyclePathState;
  recycled: FileRecyclePathState;
  files?: FileRecycleDirEntry[];
  previewable: boolean;
  downloadable: boolean;
  mime_type?: string;
  filename?: string;
  preview_kind?: "image" | "video" | "audio" | "text" | "pdf" | "other" | string;
};

export type AdminBatchGenerateUsersPayload = {
  count: number;
  items: AdminListRow[];
};

export type AdminWithdrawOrderItem = WithdrawOrderItem & {
  user_id?: number | string | null;
  user_uid?: string | null;
  nickname?: string | null;
  avatar?: string | null;
  wechat_url?: string | null;
  alipay_url?: string | null;
};

export type AdminWithdrawOrdersPayload = {
  list: AdminWithdrawOrderItem[];
  pagination: CreatorListPagination;
};

export type AdminDashboardMetric = {
  key: string;
  label: string;
  value: number;
  delta?: number;
  tone?: "red" | "green" | "blue" | "purple" | "amber" | "slate";
};

export type AdminDashboardStatus = {
  key: string;
  label: string;
  value: number | string;
  description?: string;
  tone?: "red" | "green" | "blue" | "purple" | "amber" | "slate";
};

export type AdminDashboardOverviewPayload = {
  generated_at?: string;
  metrics: AdminDashboardMetric[];
  pending: Record<string, number>;
  finance: Record<string, number>;
  statuses: AdminDashboardStatus[];
};

export type AdminDashboardTrendsPayload = {
  labels: string[];
  users: number[];
  posts: number[];
  comments: number[];
  reports: number[];
  income: number[];
};

export type AdminDashboardHotContentItem = {
  id: number | string;
  title?: string | null;
  nickname?: string | null;
  user_display_id?: string | null;
  cover_url?: string | null;
  type?: number | string | null;
  view_count?: number;
  like_count?: number;
  collect_count?: number;
  comment_count?: number;
  heat?: number;
  created_at?: string;
};

export type AdminDashboardHotContentPayload = {
  items: AdminDashboardHotContentItem[];
};

export type AdminLogRuntimeStatus = {
  access_enabled?: boolean;
  security_enabled?: boolean;
  queue_enabled?: boolean;
  available?: boolean;
  access_buffered?: number;
  security_buffered?: number;
  access_dropped?: number;
  security_dropped?: number;
  access_enqueued?: number;
  security_enqueued?: number;
  enqueue_failures?: number;
};

export type AdminAccessLogItem = {
  id: number | string;
  user_id?: number | string | null;
  user_display_id?: string | null;
  principal_type?: string | null;
  ip?: string | null;
  country_code?: string | null;
  country_name?: string | null;
  country_flag?: string | null;
  user_agent?: string | null;
  browser_language?: string | null;
  method?: string | null;
  path?: string | null;
  status?: number;
  latency_ms?: number;
  behavior?: string | null;
  target_type?: string | null;
  target_id?: number | string | null;
  request_id?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at?: string;
};

export type AdminSecurityAuditLogItem = {
  id: number | string;
  category?: string | null;
  action?: string | null;
  outcome?: string | null;
  actor_id?: number | string | null;
  actor_type?: string | null;
  actor_display_id?: string | null;
  ip?: string | null;
  country_code?: string | null;
  country_name?: string | null;
  country_flag?: string | null;
  user_agent?: string | null;
  browser_language?: string | null;
  method?: string | null;
  path?: string | null;
  status?: number;
  reason_code?: string | null;
  request_id?: string | null;
  metadata?: Record<string, unknown> | null;
  created_at?: string;
};

export type AdminPointsAuditLogItem = {
  id: number | string;
  user_id?: number | string | null;
  user_display_id?: string | null;
  nickname?: string | null;
  amount?: number;
  balance_after?: number;
  type?: string | null;
  reason?: string | null;
  post_id?: number | string | null;
  post_title?: string | null;
  purchase_id?: number | string | null;
  entry_role?: string | null;
  payment_method?: string | null;
  counterparty?: {
    user_id?: number | string | null;
    user_display_id?: string | null;
    nickname?: string | null;
  } | null;
  is_anomaly?: boolean;
  created_at?: string;
};

export type AdminBalanceAuditLogItem = {
  id: number | string;
  operation_key?: string | null;
  ledger_source?: string | null;
  user_id?: number | string | null;
  user_display_id?: string | null;
  nickname?: string | null;
  oauth2_id?: number | string | null;
  amount?: number;
  reason?: string | null;
  status?: string | null;
  remote_balance_after?: number | null;
  compensation_amount?: number | null;
  attempts?: number;
  last_error?: string | null;
  post_id?: number | string | null;
  post_title?: string | null;
  purchase_id?: number | string | null;
  entry_role?: string | null;
  payment_method?: string | null;
  platform_fee?: number;
  counterparty?: {
    user_id?: number | string | null;
    user_display_id?: string | null;
    nickname?: string | null;
  } | null;
  is_anomaly?: boolean;
  created_at?: string;
  updated_at?: string | null;
  applied_at?: string | null;
  completed_at?: string | null;
};

export type AdminLogListPayload<T> = {
  items: T[];
  pagination: CreatorListPagination;
  status?: AdminLogRuntimeStatus;
};

export type AdminUserPointsUpdatePayload = {
  uid: number | string;
  operation: "add" | "deduct" | "set";
  previous_balance: number;
  amount: number;
  balance_after: number;
};

export type AdminAccessLogRankItem = {
  key?: string;
  label?: string;
  title?: string;
  post_title?: string;
  count: number;
};

export type AdminAccessLogAnalyticsPayload = {
  range?: {
    start?: string;
    end?: string;
    bucket?: "hour" | "day" | "month";
  };
  totals?: Record<string, number>;
  series?: Array<{
    ts: string;
    pv?: number;
    active_users?: number;
    unique_ips?: number;
    post_views?: number;
    security_events?: number;
  }>;
  rankings?: Record<string, AdminAccessLogRankItem[]>;
  status?: AdminLogRuntimeStatus;
};
