package domain

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Post struct {
	ID                 int64          `gorm:"column:id;primaryKey"`
	UserID             int64          `gorm:"column:user_id"`
	Title              string         `gorm:"column:title"`
	Content            string         `gorm:"column:content"`
	CategoryID         *int           `gorm:"column:category_id"`
	Type               int            `gorm:"column:type"`
	ViewCount          int64          `gorm:"column:view_count"`
	LikeCount          int            `gorm:"column:like_count"`
	CollectCount       int            `gorm:"column:collect_count"`
	CommentCount       int            `gorm:"column:comment_count"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
	IsDraft            bool           `gorm:"column:is_draft"`
	Visibility         string         `gorm:"column:visibility"`
	PublicAccessExempt bool           `gorm:"column:public_access_exempt;default:false;index:idx_posts_public_access_exempt"`
	AuditStatus        int            `gorm:"column:audit_status;default:1;index:idx_posts_audit_status"`
	AuditResult        datatypes.JSON `gorm:"column:audit_result"`
	QualityLevel       string         `gorm:"column:quality_level;default:none"`
	QualityMarkedAt    *time.Time     `gorm:"column:quality_marked_at"`
	QualityReward      *float64       `gorm:"column:quality_reward"`
}

func (Post) TableName() string { return "posts" }

func (p *Post) BeforeCreate(_ *gorm.DB) error {
	if strings.TrimSpace(p.QualityLevel) == "" {
		p.QualityLevel = "none"
	}
	return nil
}

type Comment struct {
	ID          int64          `gorm:"column:id;primaryKey"`
	PostID      int64          `gorm:"column:post_id"`
	UserID      int64          `gorm:"column:user_id"`
	ParentID    *int64         `gorm:"column:parent_id"`
	Content     string         `gorm:"column:content"`
	LikeCount   int            `gorm:"column:like_count"`
	AuditStatus int            `gorm:"column:audit_status"`
	IsPublic    bool           `gorm:"column:is_public"`
	AuditResult datatypes.JSON `gorm:"column:audit_result"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
}

func (Comment) TableName() string { return "comments" }

type Like struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	UserID     int64     `gorm:"column:user_id"`
	TargetType int       `gorm:"column:target_type"`
	TargetID   int64     `gorm:"column:target_id"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (Like) TableName() string { return "likes" }

type Collection struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	PostID    int64     `gorm:"column:post_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Collection) TableName() string { return "collections" }

type Follow struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	FollowerID  int64     `gorm:"column:follower_id"`
	FollowingID int64     `gorm:"column:following_id"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (Follow) TableName() string { return "follows" }

type Dislike struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	PostID    int64     `gorm:"column:post_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Dislike) TableName() string { return "dislikes" }

type Notification struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	SenderID  int64     `gorm:"column:sender_id"`
	Type      int       `gorm:"column:type"`
	Title     string    `gorm:"column:title"`
	TargetID  *int64    `gorm:"column:target_id"`
	CommentID *int64    `gorm:"column:comment_id"`
	IsRead    bool      `gorm:"column:is_read"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Notification) TableName() string { return "notifications" }

type Blacklist struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	BlockerID int64     `gorm:"column:blocker_id"`
	BlockedID int64     `gorm:"column:blocked_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Blacklist) TableName() string { return "blacklist" }

type Tag struct {
	ID        int       `gorm:"column:id;primaryKey" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	UseCount  int       `gorm:"column:use_count" json:"use_count"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Tag) TableName() string { return "tags" }

type Category struct {
	ID            int            `gorm:"column:id;primaryKey" json:"id"`
	Name          string         `gorm:"column:name" json:"name"`
	CategoryTitle *string        `gorm:"column:category_title" json:"category_title"`
	Translations  datatypes.JSON `gorm:"column:translations;type:jsonb" json:"translations,omitempty"`
	UseCount      int            `gorm:"column:use_count" json:"use_count"`
	CreatedAt     time.Time      `gorm:"column:created_at" json:"created_at"`
}

func (Category) TableName() string { return "categories" }

type PostImage struct {
	ID                  int64  `gorm:"column:id;primaryKey"`
	PostID              int64  `gorm:"column:post_id;index:idx_post_images_post_sort,priority:1"`
	ImageURL            string `gorm:"column:image_url;index:idx_post_images_image_url"`
	WatermarkTraceToken string `gorm:"column:watermark_trace_token;size:16;index:idx_post_images_watermark_trace"`
	IsFreePreview       bool   `gorm:"column:is_free_preview"`
	IsProtected         bool   `gorm:"column:is_protected"`
	SortOrder           int    `gorm:"column:sort_order;index:idx_post_images_post_sort,priority:2"`
}

func (PostImage) TableName() string { return "post_images" }

type PostAttachment struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	PostID        int64     `gorm:"column:post_id"`
	AttachmentURL string    `gorm:"column:attachment_url"`
	Filename      string    `gorm:"column:filename"`
	Filesize      int64     `gorm:"column:filesize"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (PostAttachment) TableName() string { return "post_attachments" }

type UploadAsset struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	UserID       int64      `gorm:"column:user_id;index:idx_upload_assets_user_status_created,priority:1"`
	Purpose      string     `gorm:"column:purpose;size:32"`
	Kind         string     `gorm:"column:kind;size:32"`
	URL          string     `gorm:"column:url;type:text;uniqueIndex:idx_upload_assets_url"`
	Storage      string     `gorm:"column:storage;size:32"`
	ObjectKey    string     `gorm:"column:object_key;type:text"`
	LocalPath    string     `gorm:"column:local_path;type:text"`
	OriginalName string     `gorm:"column:original_name;size:255"`
	Size         int64      `gorm:"column:size"`
	MimeType     string     `gorm:"column:mime_type;size:128"`
	Status       string     `gorm:"column:status;size:32;index:idx_upload_assets_status_expires,priority:1;index:idx_upload_assets_user_status_created,priority:2"`
	BoundPostID  *int64     `gorm:"column:bound_post_id;index:idx_upload_assets_bound_post"`
	ExpiresAt    *time.Time `gorm:"column:expires_at;index:idx_upload_assets_status_expires,priority:2"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
	CleanupError string     `gorm:"column:cleanup_error;type:text"`
	CreatedAt    time.Time  `gorm:"column:created_at;index:idx_upload_assets_user_status_created,priority:3"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (UploadAsset) TableName() string { return "upload_assets" }

type PostVideo struct {
	ID              int64   `gorm:"column:id;primaryKey"`
	PostID          int64   `gorm:"column:post_id"`
	CoverURL        *string `gorm:"column:cover_url;index:idx_post_videos_cover_url"`
	VideoURL        string  `gorm:"column:video_url;index:idx_post_videos_video_url"`
	DashURL         *string `gorm:"column:dash_url;index:idx_post_videos_dash_url"`
	PreviewVideoURL *string `gorm:"column:preview_video_url;index:idx_post_videos_preview_video_url"`
}

func (PostVideo) TableName() string { return "post_videos" }

type FileRecycleItem struct {
	ID           int64          `gorm:"column:id;primaryKey" json:"id"`
	GroupID      string         `gorm:"column:group_id;size:64;index:idx_file_recycle_group" json:"group_id"`
	ResourceType string         `gorm:"column:resource_type;size:32;index:idx_file_recycle_resource,priority:1" json:"resource_type"`
	ResourceID   int64          `gorm:"column:resource_id;index:idx_file_recycle_resource,priority:2" json:"resource_id"`
	PostID       *int64         `gorm:"column:post_id;index:idx_file_recycle_post" json:"post_id,omitempty"`
	UserID       *int64         `gorm:"column:user_id;index:idx_file_recycle_user" json:"user_id,omitempty"`
	Kind         string         `gorm:"column:kind;size:32;index:idx_file_recycle_kind" json:"kind"`
	Storage      string         `gorm:"column:storage;size:32" json:"storage"`
	OriginalURL  string         `gorm:"column:original_url;type:text" json:"original_url"`
	OriginalPath string         `gorm:"column:original_path;type:text" json:"original_path"`
	RecycledPath string         `gorm:"column:recycled_path;type:text" json:"recycled_path"`
	IsDir        bool           `gorm:"column:is_dir" json:"is_dir"`
	FileCount    int64          `gorm:"column:file_count" json:"file_count"`
	SizeBytes    int64          `gorm:"column:size_bytes" json:"size_bytes"`
	Status       string         `gorm:"column:status;size:32;index:idx_file_recycle_status_purge,priority:1" json:"status"`
	DeletedAt    time.Time      `gorm:"column:deleted_at;index:idx_file_recycle_deleted" json:"deleted_at"`
	PurgeAfter   time.Time      `gorm:"column:purge_after;index:idx_file_recycle_status_purge,priority:2" json:"purge_after"`
	PurgedAt     *time.Time     `gorm:"column:purged_at" json:"purged_at,omitempty"`
	Error        string         `gorm:"column:error;type:text" json:"error,omitempty"`
	Metadata     datatypes.JSON `gorm:"column:metadata" json:"metadata,omitempty"`
	CreatedAt    time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    *time.Time     `gorm:"column:updated_at" json:"updated_at,omitempty"`
}

func (FileRecycleItem) TableName() string { return "file_recycle_items" }

type PostPaymentSetting struct {
	ID               int64      `gorm:"column:id;primaryKey"`
	PostID           int64      `gorm:"column:post_id"`
	Enabled          bool       `gorm:"column:enabled"`
	PaymentType      string     `gorm:"column:payment_type"`
	PaymentMethod    string     `gorm:"column:payment_method"`
	Price            float64    `gorm:"column:price"`
	FreePreviewCount int        `gorm:"column:free_preview_count"`
	PreviewDuration  int        `gorm:"column:preview_duration"`
	HideAll          bool       `gorm:"column:hide_all"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        *time.Time `gorm:"column:updated_at"`
}

func (PostPaymentSetting) TableName() string { return "post_payment_settings" }

type UserPurchasedContent struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	UserID        int64     `gorm:"column:user_id;uniqueIndex:idx_user_purchased_content_user_post,priority:1"`
	PostID        int64     `gorm:"column:post_id;uniqueIndex:idx_user_purchased_content_user_post,priority:2"`
	AuthorID      int64     `gorm:"column:author_id"`
	Price         float64   `gorm:"column:price"`
	PaidAmount    float64   `gorm:"column:paid_amount"`
	DiscountRate  float64   `gorm:"column:discount_rate"`
	PurchaseType  string    `gorm:"column:purchase_type"`
	PaymentMethod string    `gorm:"column:payment_method"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	PurchasedAt   time.Time `gorm:"column:purchased_at"`
}

func (UserPurchasedContent) TableName() string { return "user_purchased_content" }

type ContentPurchaseIntent struct {
	ID                   int64      `gorm:"column:id;primaryKey"`
	UserID               int64      `gorm:"column:user_id;uniqueIndex:idx_content_purchase_intents_user_post,priority:1"`
	PostID               int64      `gorm:"column:post_id;uniqueIndex:idx_content_purchase_intents_user_post,priority:2"`
	PaymentMethod        string     `gorm:"column:payment_method;size:32"`
	Status               string     `gorm:"column:status;size:32;index:idx_content_purchase_intents_status_updated,priority:1"`
	Price                float64    `gorm:"column:price"`
	PaidAmount           float64    `gorm:"column:paid_amount"`
	ExternalDebitApplied bool       `gorm:"column:external_debit_applied"`
	ErrorCode            *string    `gorm:"column:error_code;size:128"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            *time.Time `gorm:"column:updated_at;index:idx_content_purchase_intents_status_updated,priority:2"`
}

func (ContentPurchaseIntent) TableName() string { return "content_purchase_intents" }

type ImageProtectionJob struct {
	ID                        int64      `gorm:"column:id;primaryKey"`
	JobID                     string     `gorm:"column:job_id;size:64;uniqueIndex:idx_image_protection_jobs_job_id"`
	PostID                    int64      `gorm:"column:post_id;index:idx_image_protection_jobs_post_user_status,priority:1;index:idx_image_protection_jobs_post_kind_status,priority:1"`
	UserID                    int64      `gorm:"column:user_id;index:idx_image_protection_jobs_user_status_created,priority:1;index:idx_image_protection_jobs_post_user_status,priority:2"`
	AuthorID                  int64      `gorm:"column:author_id"`
	PackageKind               string     `gorm:"column:package_kind;size:32;index:idx_image_protection_jobs_post_kind_status,priority:2"`
	SourceSignature           string     `gorm:"column:source_signature;size:64"`
	Status                    string     `gorm:"column:status;size:32;index:idx_image_protection_jobs_user_status_created,priority:2;index:idx_image_protection_jobs_post_user_status,priority:3;index:idx_image_protection_jobs_post_kind_status,priority:3"`
	Progress                  int        `gorm:"column:progress"`
	QueuePosition             int        `gorm:"column:queue_position"`
	EstimatedWaitSeconds      int        `gorm:"column:estimated_wait_seconds"`
	EstimatedRemainingSeconds int        `gorm:"column:estimated_remaining_seconds"`
	ProtectedImageCount       int        `gorm:"column:protected_image_count"`
	ProcessedImageCount       int        `gorm:"column:processed_image_count"`
	CurrentStep               string     `gorm:"column:current_step"`
	ActiveProfile             string     `gorm:"column:active_profile;size:32"`
	HeartbeatAt               *time.Time `gorm:"column:heartbeat_at"`
	PackagePath               string     `gorm:"column:package_path"`
	ErrorMessage              *string    `gorm:"column:error_message"`
	ErrorCode                 *string    `gorm:"column:error_code;size:128"`
	Retryable                 bool       `gorm:"column:retryable"`
	StartedAt                 *time.Time `gorm:"column:started_at"`
	FinishedAt                *time.Time `gorm:"column:finished_at"`
	ExpiresAt                 *time.Time `gorm:"column:expires_at;index:idx_image_protection_jobs_expires"`
	CreatedAt                 time.Time  `gorm:"column:created_at;index:idx_image_protection_jobs_user_status_created,priority:3"`
	UpdatedAt                 *time.Time `gorm:"column:updated_at"`
}

func (ImageProtectionJob) TableName() string { return "image_protection_jobs" }

type UserSearchHistory struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	Keyword   string    `gorm:"column:keyword"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (UserSearchHistory) TableName() string { return "user_search_history" }

type BrowsingHistory struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	UserID    int64      `gorm:"column:user_id"`
	PostID    int64      `gorm:"column:post_id"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (BrowsingHistory) TableName() string { return "browsing_history" }

type UserAPIKey struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	UserID       int64      `gorm:"column:user_id"`
	Name         string     `gorm:"column:name"`
	APIKey       string     `gorm:"column:api_key;uniqueIndex:idx_user_api_keys_api_key"`
	APIKeyPrefix string     `gorm:"column:api_key_prefix"`
	IsActive     bool       `gorm:"column:is_active"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (UserAPIKey) TableName() string { return "user_api_keys" }

type UserToolbar struct {
	ID        int        `gorm:"column:id;primaryKey"`
	Name      string     `gorm:"column:name"`
	Icon      string     `gorm:"column:icon"`
	URL       string     `gorm:"column:url"`
	SortOrder int        `gorm:"column:sort_order"`
	IsActive  bool       `gorm:"column:is_active"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (UserToolbar) TableName() string { return "user_toolbar" }

type SystemSetting struct {
	ID           int        `gorm:"column:id;primaryKey"`
	SettingKey   string     `gorm:"column:setting_key"`
	SettingValue string     `gorm:"column:setting_value"`
	SettingGroup string     `gorm:"column:setting_group"`
	Description  *string    `gorm:"column:description"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (SystemSetting) TableName() string { return "system_settings" }
