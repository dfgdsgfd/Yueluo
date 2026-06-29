package domain

import (
	"time"

	"gorm.io/datatypes"
)

type SystemUpdateConfig struct {
	ID                   int64      `gorm:"column:id;primaryKey"`
	FrontendRepoURL      string     `gorm:"column:frontend_repo_url"`
	BackendRepoURL       string     `gorm:"column:backend_repo_url"`
	GithubToken          string     `gorm:"column:github_token"`
	FrontendBranch       string     `gorm:"column:frontend_branch"`
	BackendBranch        string     `gorm:"column:backend_branch"`
	FrontendReleaseTag   string     `gorm:"column:frontend_release_tag"`
	BackendReleaseTag    string     `gorm:"column:backend_release_tag"`
	FrontendArtifactURL  string     `gorm:"column:frontend_artifact_url"`
	BackendArtifactURL   string     `gorm:"column:backend_artifact_url"`
	FrontendAssetPattern string     `gorm:"column:frontend_asset_pattern"`
	BackendAssetPattern  string     `gorm:"column:backend_asset_pattern"`
	FrontendInstallDir   string     `gorm:"column:frontend_install_dir"`
	BackendInstallPath   string     `gorm:"column:backend_install_path"`
	FrontendStartMode    string     `gorm:"column:frontend_start_mode"`
	FrontendSourceDir    string     `gorm:"column:frontend_source_dir"`
	BackendSourceDir     string     `gorm:"column:backend_source_dir"`
	ArtifactDir          string     `gorm:"column:artifact_dir"`
	FrontendBuildCommand string     `gorm:"column:frontend_build_command"`
	BackendBuildCommand  string     `gorm:"column:backend_build_command"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            *time.Time `gorm:"column:updated_at"`
}

func (SystemUpdateConfig) TableName() string { return "system_update_configs" }

type SystemUpdateJob struct {
	ID            int64          `gorm:"column:id;primaryKey"`
	Action        string         `gorm:"column:action;size:32;index:idx_system_update_jobs_created,priority:2"`
	Status        string         `gorm:"column:status;size:32;index:idx_system_update_jobs_created,priority:3"`
	FrontendState string         `gorm:"column:frontend_state;size:32"`
	BackendState  string         `gorm:"column:backend_state;size:32"`
	Progress      int            `gorm:"column:progress"`
	CurrentStep   string         `gorm:"column:current_step"`
	ArtifactPaths datatypes.JSON `gorm:"column:artifact_paths"`
	ArtifactMeta  datatypes.JSON `gorm:"column:artifact_meta"`
	Logs          string         `gorm:"column:logs"`
	ErrorMessage  *string        `gorm:"column:error_message"`
	StartedAt     *time.Time     `gorm:"column:started_at"`
	FinishedAt    *time.Time     `gorm:"column:finished_at"`
	CreatedAt     time.Time      `gorm:"column:created_at;index:idx_system_update_jobs_created,priority:1"`
	UpdatedAt     *time.Time     `gorm:"column:updated_at"`
}

func (SystemUpdateJob) TableName() string { return "system_update_jobs" }

type RecommendConfig struct {
	ID                 int        `gorm:"column:id;primaryKey"`
	UserID             *int64     `gorm:"column:user_id"`
	LikeWeight         float64    `gorm:"column:like_weight"`
	CollectWeight      float64    `gorm:"column:collect_weight"`
	ViewWeight         float64    `gorm:"column:view_weight"`
	CategoryWeight     float64    `gorm:"column:category_weight"`
	TagWeight          float64    `gorm:"column:tag_weight"`
	FollowingWeight    float64    `gorm:"column:following_weight"`
	MutualFollowWeight float64    `gorm:"column:mutual_follow_weight"`
	PopularityWeight   float64    `gorm:"column:popularity_weight"`
	InterestWeight     float64    `gorm:"column:interest_weight"`
	TimeDecayHalfLife  int        `gorm:"column:time_decay_half_life"`
	IsActive           bool       `gorm:"column:is_active"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          *time.Time `gorm:"column:updated_at"`
}

func (RecommendConfig) TableName() string { return "recommend_configs" }

type PostRecommendConfig struct {
	ID           int        `gorm:"column:id;primaryKey"`
	PostID       int64      `gorm:"column:post_id"`
	BoostScore   float64    `gorm:"column:boost_score"`
	IsPinned     bool       `gorm:"column:is_pinned"`
	IsSuppressed bool       `gorm:"column:is_suppressed"`
	TargetUserID *int64     `gorm:"column:target_user_id"`
	Reason       *string    `gorm:"column:reason"`
	IsActive     bool       `gorm:"column:is_active"`
	StartTime    *time.Time `gorm:"column:start_time"`
	EndTime      *time.Time `gorm:"column:end_time"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (PostRecommendConfig) TableName() string { return "post_recommend_configs" }

type PostQualityRewardSetting struct {
	ID           int        `gorm:"column:id;primaryKey"`
	QualityLevel string     `gorm:"column:quality_level"`
	RewardAmount float64    `gorm:"column:reward_amount"`
	Description  *string    `gorm:"column:description"`
	IsActive     bool       `gorm:"column:is_active"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (PostQualityRewardSetting) TableName() string { return "post_quality_reward_settings" }

type PostTag struct {
	ID     int64 `gorm:"column:id;primaryKey"`
	PostID int64 `gorm:"column:post_id"`
	TagID  int   `gorm:"column:tag_id"`
}

func (PostTag) TableName() string { return "post_tags" }

type Announcement struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	Title       string     `gorm:"column:title"`
	Content     string     `gorm:"column:content"`
	Type        string     `gorm:"column:type"`
	IsPublished bool       `gorm:"column:is_published"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	ExpiresAt   *time.Time `gorm:"column:expires_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Announcement) TableName() string { return "announcements" }

type License struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	LicenseKey     string     `gorm:"column:license_key"`
	MachineModel   *string    `gorm:"column:machine_model"`
	MachineID      *string    `gorm:"column:machine_id"`
	Remark         *string    `gorm:"column:remark"`
	IsActive       bool       `gorm:"column:is_active"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	LastVerifiedAt *time.Time `gorm:"column:last_verified_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (License) TableName() string { return "licenses" }

type OpenAPI struct {
	ID           int64          `gorm:"column:id;primaryKey"`
	Name         string         `gorm:"column:name"`
	APIKey       string         `gorm:"column:api_key;uniqueIndex:idx_open_apis_api_key"`
	APIKeyPrefix string         `gorm:"column:api_key_prefix"`
	Permissions  datatypes.JSON `gorm:"column:permissions"`
	IsActive     bool           `gorm:"column:is_active"`
	LastUsedAt   *time.Time     `gorm:"column:last_used_at"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
}

func (OpenAPI) TableName() string { return "open_apis" }

type AppVersion struct {
	ID          int        `gorm:"column:id;primaryKey"`
	AppName     string     `gorm:"column:app_name"`
	VersionCode int        `gorm:"column:version_code"`
	VersionName string     `gorm:"column:version_name"`
	Platform    string     `gorm:"column:platform"`
	DownloadURL string     `gorm:"column:download_url"`
	SizeBytes   int64      `gorm:"column:size_bytes;default:0"`
	UpdateLog   *string    `gorm:"column:update_log"`
	ForceUpdate bool       `gorm:"column:force_update"`
	IsActive    bool       `gorm:"column:is_active"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (AppVersion) TableName() string { return "app_versions" }

type AppUsageLog struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	DeviceID    string    `gorm:"column:device_id"`
	EventType   string    `gorm:"column:event_type"`
	VersionCode *int      `gorm:"column:version_code"`
	VersionID   *int      `gorm:"column:version_id"`
	Platform    string    `gorm:"column:platform"`
	Duration    *int      `gorm:"column:duration"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (AppUsageLog) TableName() string { return "app_usage_logs" }

// OAuthAppHandoff is a short-lived, one-time bridge between the system browser
// OAuth callback and a native app. Only hashes and the PKCE challenge are
// persisted, so neither the browser redirect nor the database contains a JWT.
type OAuthAppHandoff struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	CodeHash      string     `gorm:"column:code_hash;size:64;uniqueIndex:idx_oauth_app_handoffs_code"`
	AppStateHash  string     `gorm:"column:app_state_hash;size:64"`
	CodeChallenge string     `gorm:"column:code_challenge;size:128"`
	UserID        int64      `gorm:"column:user_id;index:idx_oauth_app_handoffs_user_created,priority:1"`
	IsNewUser     bool       `gorm:"column:is_new_user"`
	ExpiresAt     time.Time  `gorm:"column:expires_at;index:idx_oauth_app_handoffs_expires"`
	ConsumedAt    *time.Time `gorm:"column:consumed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;index:idx_oauth_app_handoffs_user_created,priority:2"`
}

func (OAuthAppHandoff) TableName() string { return "oauth_app_handoffs" }

type AccessLog struct {
	ID              int64          `gorm:"column:id;primaryKey"`
	UserID          *int64         `gorm:"column:user_id;index:idx_access_logs_user_created,priority:1"`
	UserDisplayID   *string        `gorm:"column:user_display_id;size:128"`
	PrincipalType   string         `gorm:"column:principal_type;size:32;index:idx_access_logs_principal_created,priority:1"`
	IP              string         `gorm:"column:ip;size:64;index:idx_access_logs_ip_created,priority:1"`
	UserAgent       string         `gorm:"column:user_agent;type:text"`
	BrowserLanguage string         `gorm:"column:browser_language;size:128"`
	Method          string         `gorm:"column:method;size:16"`
	Path            string         `gorm:"column:path;size:255;index:idx_access_logs_path_created,priority:1"`
	Status          int            `gorm:"column:status"`
	LatencyMS       int64          `gorm:"column:latency_ms"`
	Behavior        string         `gorm:"column:behavior;size:64;index:idx_access_logs_behavior_created,priority:1"`
	TargetType      *string        `gorm:"column:target_type;size:32;index:idx_access_logs_target_created,priority:1"`
	TargetID        *int64         `gorm:"column:target_id;index:idx_access_logs_target_created,priority:2"`
	RequestID       string         `gorm:"column:request_id;size:128"`
	Metadata        datatypes.JSON `gorm:"column:metadata"`
	CreatedAt       time.Time      `gorm:"column:created_at;index:idx_access_logs_created_at;index:idx_access_logs_behavior_created,priority:2;index:idx_access_logs_user_created,priority:2;index:idx_access_logs_principal_created,priority:2;index:idx_access_logs_target_created,priority:3;index:idx_access_logs_path_created,priority:2;index:idx_access_logs_ip_created,priority:2"`
}

func (AccessLog) TableName() string { return "access_logs" }

type SecurityAuditLog struct {
	ID              int64          `gorm:"column:id;primaryKey"`
	Category        string         `gorm:"column:category;size:64;index:idx_security_audit_category_created,priority:1"`
	Action          string         `gorm:"column:action;size:64;index:idx_security_audit_action_created,priority:1"`
	Outcome         string         `gorm:"column:outcome;size:32;index:idx_security_audit_outcome_created,priority:1"`
	ActorID         *int64         `gorm:"column:actor_id;index:idx_security_audit_actor_created,priority:2"`
	ActorType       string         `gorm:"column:actor_type;size:32;index:idx_security_audit_actor_created,priority:1"`
	ActorDisplayID  *string        `gorm:"column:actor_display_id;size:128"`
	IP              string         `gorm:"column:ip;size:64;index:idx_security_audit_ip_created,priority:1"`
	UserAgent       string         `gorm:"column:user_agent;type:text"`
	BrowserLanguage string         `gorm:"column:browser_language;size:128"`
	Method          string         `gorm:"column:method;size:16"`
	Path            string         `gorm:"column:path;size:255"`
	Status          int            `gorm:"column:status"`
	ReasonCode      string         `gorm:"column:reason_code;size:128"`
	RequestID       string         `gorm:"column:request_id;size:128"`
	Metadata        datatypes.JSON `gorm:"column:metadata"`
	CreatedAt       time.Time      `gorm:"column:created_at;index:idx_security_audit_created_at;index:idx_security_audit_category_created,priority:2;index:idx_security_audit_action_created,priority:2;index:idx_security_audit_actor_created,priority:3;index:idx_security_audit_outcome_created,priority:2;index:idx_security_audit_ip_created,priority:2"`
}

func (SecurityAuditLog) TableName() string { return "security_audit_logs" }

type AccessBlockRule struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	ImportSourceID int64      `gorm:"column:import_source_id;default:0;uniqueIndex:idx_access_block_rules_source_unique,priority:1;index:idx_access_block_rules_import_source"`
	Kind           string     `gorm:"column:kind;size:16;uniqueIndex:idx_access_block_rules_source_unique,priority:2"`
	MatchType      string     `gorm:"column:match_type;size:32;uniqueIndex:idx_access_block_rules_source_unique,priority:3"`
	Pattern        string     `gorm:"column:pattern;size:512;uniqueIndex:idx_access_block_rules_source_unique,priority:4"`
	Enabled        bool       `gorm:"column:enabled;default:true;index:idx_access_block_rules_enabled_priority_updated,priority:1"`
	Priority       int        `gorm:"column:priority;default:1000;index:idx_access_block_rules_enabled_priority_updated,priority:2"`
	Action         string     `gorm:"column:action;size:16;default:status"`
	StatusCode     int        `gorm:"column:status_code;default:444"`
	RedirectURL    string     `gorm:"column:redirect_url;type:text"`
	Note           string     `gorm:"column:note;type:text"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at;index:idx_access_block_rules_enabled_priority_updated,priority:3"`
}

func (AccessBlockRule) TableName() string { return "access_block_rules" }

type AccessBlockImportSource struct {
	ID                    int64      `gorm:"column:id;primaryKey"`
	URL                   string     `gorm:"column:url;size:768;uniqueIndex:idx_access_block_import_sources_url"`
	Enabled               bool       `gorm:"column:enabled;default:true;index:idx_access_block_import_sources_enabled_next_sync,priority:1"`
	Priority              int        `gorm:"column:priority;default:1000"`
	Action                string     `gorm:"column:action;size:16;default:status"`
	StatusCode            int        `gorm:"column:status_code;default:444"`
	RedirectURL           string     `gorm:"column:redirect_url;type:text"`
	Note                  string     `gorm:"column:note;type:text"`
	UpdateIntervalSeconds int        `gorm:"column:update_interval_seconds;default:3600"`
	LastSyncAt            *time.Time `gorm:"column:last_sync_at"`
	NextSyncAt            *time.Time `gorm:"column:next_sync_at;index:idx_access_block_import_sources_enabled_next_sync,priority:2"`
	LastStatus            string     `gorm:"column:last_status;size:32"`
	LastError             string     `gorm:"column:last_error;type:text"`
	LastCount             int        `gorm:"column:last_count;default:0"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedAt             *time.Time `gorm:"column:updated_at"`
}

func (AccessBlockImportSource) TableName() string { return "access_block_import_sources" }

type AIGenerationLog struct {
	ID               int64          `gorm:"column:id;primaryKey"`
	JobID            string         `gorm:"column:job_id;size:64;uniqueIndex:idx_ai_generation_logs_job_id"`
	TaskType         string         `gorm:"column:task_type;size:64;index:idx_ai_generation_logs_task_created,priority:1"`
	TemplateKey      string         `gorm:"column:template_key;size:128"`
	ActorType        string         `gorm:"column:actor_type;size:32;index:idx_ai_generation_logs_actor_created,priority:1"`
	ActorID          *int64         `gorm:"column:actor_id;index:idx_ai_generation_logs_actor_created,priority:2"`
	ActorDisplayID   *string        `gorm:"column:actor_display_id;size:128"`
	InputSummary     string         `gorm:"column:input_summary;type:text"`
	OutputSummary    string         `gorm:"column:output_summary;type:text"`
	Status           string         `gorm:"column:status;size:32"`
	Model            string         `gorm:"column:model;size:128"`
	BaseURL          string         `gorm:"column:base_url;size:255"`
	PromptTokens     int            `gorm:"column:prompt_tokens"`
	CompletionTokens int            `gorm:"column:completion_tokens"`
	TotalTokens      int            `gorm:"column:total_tokens"`
	EstimatedCost    float64        `gorm:"column:estimated_cost"`
	ErrorCode        string         `gorm:"column:error_code;size:128"`
	ErrorMessage     string         `gorm:"column:error_message;type:text"`
	DurationMS       int64          `gorm:"column:duration_ms"`
	TokensPerSecond  float64        `gorm:"column:tokens_per_second"`
	Metadata         datatypes.JSON `gorm:"column:metadata"`
	CreatedAt        time.Time      `gorm:"column:created_at;index:idx_ai_generation_logs_created;index:idx_ai_generation_logs_actor_created,priority:3;index:idx_ai_generation_logs_task_created,priority:2"`
	UpdatedAt        *time.Time     `gorm:"column:updated_at"`
}

func (AIGenerationLog) TableName() string { return "ai_generation_logs" }

type AIJob struct {
	ID               int64          `gorm:"column:id;primaryKey"`
	JobID            string         `gorm:"column:job_id;size:64;uniqueIndex:idx_ai_jobs_job_id"`
	RequestHash      string         `gorm:"column:request_hash;size:96;index:idx_ai_jobs_request_status_created,priority:1"`
	TaskType         string         `gorm:"column:task_type;size:64;index:idx_ai_jobs_actor_status_created,priority:1"`
	TemplateKey      string         `gorm:"column:template_key;size:128"`
	ActorType        string         `gorm:"column:actor_type;size:32;index:idx_ai_jobs_actor_status_created,priority:2"`
	ActorID          *int64         `gorm:"column:actor_id;index:idx_ai_jobs_actor_status_created,priority:3"`
	ActorDisplayID   *string        `gorm:"column:actor_display_id;size:128"`
	Status           string         `gorm:"column:status;size:32;index:idx_ai_jobs_request_status_created,priority:2;index:idx_ai_jobs_actor_status_created,priority:4"`
	Stage            string         `gorm:"column:stage;size:64"`
	Percent          int            `gorm:"column:percent"`
	CurrentChunk     int            `gorm:"column:current_chunk"`
	TotalChunks      int            `gorm:"column:total_chunks"`
	ProcessedChars   int            `gorm:"column:processed_chars"`
	TotalChars       int            `gorm:"column:total_chars"`
	InputSummary     string         `gorm:"column:input_summary;type:text"`
	Output           string         `gorm:"column:output;type:text"`
	Reasoning        string         `gorm:"column:reasoning;type:text"`
	ErrorCode        string         `gorm:"column:error_code;size:128"`
	ErrorMessage     string         `gorm:"column:error_message;type:text"`
	UpstreamStatus   int            `gorm:"column:upstream_status"`
	UpstreamDetail   string         `gorm:"column:upstream_detail;type:text"`
	PromptTokens     int            `gorm:"column:prompt_tokens"`
	CompletionTokens int            `gorm:"column:completion_tokens"`
	TotalTokens      int            `gorm:"column:total_tokens"`
	EstimatedTokens  int            `gorm:"column:estimated_tokens"`
	TokensPerSecond  float64        `gorm:"column:tokens_per_second"`
	Request          datatypes.JSON `gorm:"column:request"`
	Metadata         datatypes.JSON `gorm:"column:metadata"`
	StartedAt        *time.Time     `gorm:"column:started_at"`
	FinishedAt       *time.Time     `gorm:"column:finished_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;index:idx_ai_jobs_created;index:idx_ai_jobs_request_status_created,priority:3;index:idx_ai_jobs_actor_status_created,priority:5"`
	UpdatedAt        *time.Time     `gorm:"column:updated_at"`
}

func (AIJob) TableName() string { return "ai_jobs" }

type AIModerationLog struct {
	ID                 int64          `gorm:"column:id;primaryKey"`
	TargetType         string         `gorm:"column:target_type;size:32;index:idx_ai_moderation_target"`
	TargetID           int64          `gorm:"column:target_id;index:idx_ai_moderation_target"`
	UserID             int64          `gorm:"column:user_id;index:idx_ai_moderation_user_created,priority:1"`
	Status             string         `gorm:"column:status;size:32;index:idx_ai_moderation_status_created,priority:1"`
	Action             string         `gorm:"column:action;size:32"`
	TriggerReason      string         `gorm:"column:trigger_reason;type:text"`
	Categories         datatypes.JSON `gorm:"column:categories"`
	ModelResult        datatypes.JSON `gorm:"column:model_result"`
	UserViolationCount int            `gorm:"column:user_violation_count"`
	ErrorCode          string         `gorm:"column:error_code;size:128"`
	ErrorMessage       string         `gorm:"column:error_message;type:text"`
	Metadata           datatypes.JSON `gorm:"column:metadata"`
	CreatedAt          time.Time      `gorm:"column:created_at;index:idx_ai_moderation_user_created,priority:2;index:idx_ai_moderation_status_created,priority:2"`
	UpdatedAt          *time.Time     `gorm:"column:updated_at"`
}

func (AIModerationLog) TableName() string { return "ai_moderation_logs" }

type MediaLibrary struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	Title     *string    `gorm:"column:title"`
	Type      string     `gorm:"column:type"`
	URL       string     `gorm:"column:url"`
	Filename  *string    `gorm:"column:filename"`
	Filesize  int64      `gorm:"column:filesize"`
	MimeType  *string    `gorm:"column:mime_type"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (MediaLibrary) TableName() string { return "media_library" }

type NotificationTemplate struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	TemplateKey string     `gorm:"column:template_key"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	Subject     *string    `gorm:"column:subject"`
	Content     string     `gorm:"column:content"`
	Type        string     `gorm:"column:type"`
	IsActive    bool       `gorm:"column:is_active"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (NotificationTemplate) TableName() string { return "notification_templates" }

type Report struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	ReporterID  int64      `gorm:"column:reporter_id"`
	TargetType  string     `gorm:"column:target_type"`
	TargetID    int64      `gorm:"column:target_id"`
	Reason      string     `gorm:"column:reason"`
	Description *string    `gorm:"column:description"`
	Status      string     `gorm:"column:status"`
	AdminNote   *string    `gorm:"column:admin_note"`
	ReviewedBy  *int64     `gorm:"column:reviewed_by"`
	ReviewedAt  *time.Time `gorm:"column:reviewed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Report) TableName() string { return "reports" }

type SystemNotification struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	Title         string     `gorm:"column:title"`
	Content       string     `gorm:"column:content"`
	Type          string     `gorm:"column:type"`
	ContentFormat string     `gorm:"column:content_format"`
	ImageURL      *string    `gorm:"column:image_url"`
	LinkURL       *string    `gorm:"column:link_url"`
	ShowPopup     bool       `gorm:"column:show_popup"`
	IsActive      bool       `gorm:"column:is_active"`
	StartTime     *time.Time `gorm:"column:start_time"`
	EndTime       *time.Time `gorm:"column:end_time"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func (SystemNotification) TableName() string { return "system_notifications" }

type SystemNotificationConfirmation struct {
	ID             int64     `gorm:"column:id;primaryKey"`
	NotificationID int64     `gorm:"column:notification_id"`
	UserID         int64     `gorm:"column:user_id"`
	ConfirmedAt    time.Time `gorm:"column:confirmed_at"`
	IsDismissed    bool      `gorm:"column:is_dismissed"`
}

func (SystemNotificationConfirmation) TableName() string {
	return "system_notification_confirmations"
}

type Feedback struct {
	ID         int64          `gorm:"column:id;primaryKey"`
	UserID     int64          `gorm:"column:user_id"`
	Content    string         `gorm:"column:content"`
	Images     datatypes.JSON `gorm:"column:images"`
	VideoURL   *string        `gorm:"column:video_url"`
	Status     string         `gorm:"column:status"`
	AdminReply *string        `gorm:"column:admin_reply"`
	RepliedAt  *time.Time     `gorm:"column:replied_at"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  *time.Time     `gorm:"column:updated_at"`
}

func (Feedback) TableName() string { return "feedback" }

type IMConversation struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	ExternalID    *int64     `gorm:"column:external_id"`
	Type          string     `gorm:"column:type"`
	Name          *string    `gorm:"column:name"`
	CreatorID     int64      `gorm:"column:creator_id"`
	LastMessageID *int64     `gorm:"column:last_message_id"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func (IMConversation) TableName() string { return "im_conversations" }

type IMConversationMember struct {
	ID                int64     `gorm:"column:id;primaryKey"`
	ConversationID    int64     `gorm:"column:conversation_id"`
	UserID            int64     `gorm:"column:user_id"`
	JoinedAt          time.Time `gorm:"column:joined_at"`
	LastReadMessageID *int64    `gorm:"column:last_read_message_id"`
}

func (IMConversationMember) TableName() string { return "im_conversation_members" }

type IMMessage struct {
	ID             int64     `gorm:"column:id;primaryKey"`
	ConversationID int64     `gorm:"column:conversation_id"`
	SenderID       int64     `gorm:"column:sender_id"`
	Content        string    `gorm:"column:content"`
	ClientMsgID    *string   `gorm:"column:client_msg_id"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (IMMessage) TableName() string { return "im_messages" }

type IMMessageReceipt struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	MessageID   int64      `gorm:"column:message_id"`
	UserID      int64      `gorm:"column:user_id"`
	DeliveredAt *time.Time `gorm:"column:delivered_at"`
	ReadAt      *time.Time `gorm:"column:read_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (IMMessageReceipt) TableName() string { return "im_message_receipts" }
