package domain

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID                   int64          `gorm:"column:id;primaryKey"`
	XiseID               *string        `gorm:"column:xise_id"`
	Password             *string        `gorm:"column:password"`
	UserID               string         `gorm:"column:user_id"`
	OAuth2ID             *int64         `gorm:"column:oauth2_id"`
	Nickname             string         `gorm:"column:nickname"`
	Email                *string        `gorm:"column:email"`
	Avatar               *string        `gorm:"column:avatar"`
	AvatarXXH3           *string        `gorm:"column:avatar_xxh3;size:32"`
	Background           *string        `gorm:"column:background"`
	BackgroundXXH3       *string        `gorm:"column:background_xxh3;size:32"`
	Bio                  *string        `gorm:"column:bio"`
	BioAuditStatus       int            `gorm:"column:bio_audit_status"`
	Location             *string        `gorm:"column:location"`
	FollowCount          int            `gorm:"column:follow_count"`
	FansCount            int            `gorm:"column:fans_count"`
	LikeCount            int            `gorm:"column:like_count"`
	IsActive             bool           `gorm:"column:is_active"`
	LastLoginAt          *time.Time     `gorm:"column:last_login_at"`
	CreatedAt            time.Time      `gorm:"column:created_at"`
	UpdatedAt            *time.Time     `gorm:"column:updated_at"`
	Gender               *string        `gorm:"column:gender"`
	ZodiacSign           *string        `gorm:"column:zodiac_sign"`
	MBTI                 *string        `gorm:"column:mbti"`
	Education            *string        `gorm:"column:education"`
	Major                *string        `gorm:"column:major"`
	Interests            datatypes.JSON `gorm:"column:interests"`
	Birthday             *time.Time     `gorm:"column:birthday"`
	CustomFields         datatypes.JSON `gorm:"column:custom_fields"`
	ProfileCompleted     bool           `gorm:"column:profile_completed"`
	PrivacyBirthday      bool           `gorm:"column:privacy_birthday"`
	PrivacyAge           bool           `gorm:"column:privacy_age"`
	PrivacyZodiac        bool           `gorm:"column:privacy_zodiac"`
	PrivacyMBTI          bool           `gorm:"column:privacy_mbti"`
	PrivacyCustomFields  datatypes.JSON `gorm:"column:privacy_custom_fields"`
	AIAutoCommentEnabled bool           `gorm:"column:ai_auto_comment_enabled;default:true"`
	Verified             int            `gorm:"column:verified"`
	VerifiedName         *string        `gorm:"column:verified_name"`
}

func (User) TableName() string { return "users" }

type InviteCode struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	UserID        int64      `gorm:"column:user_id"`
	Code          string     `gorm:"column:code"`
	InvitedByID   *int64     `gorm:"column:invited_by_id"`
	ClickCount    int        `gorm:"column:click_count"`
	RegisterCount int        `gorm:"column:register_count"`
	TotalEarnings float64    `gorm:"column:total_earnings"`
	IsActive      bool       `gorm:"column:is_active"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func (InviteCode) TableName() string { return "invite_codes" }

type InviteClick struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Code      string    `gorm:"column:code"`
	IPHash    string    `gorm:"column:ip_hash"`
	UserAgent *string   `gorm:"column:user_agent"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (InviteClick) TableName() string { return "invite_clicks" }

type InviteEarningsLog struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	InviteeID *int64    `gorm:"column:invitee_id"`
	Amount    float64   `gorm:"column:amount"`
	Type      string    `gorm:"column:type"`
	Reason    *string   `gorm:"column:reason"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (InviteEarningsLog) TableName() string { return "invite_earnings_log" }

type Coupon struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	Code        *string    `gorm:"column:code"`
	Type        string     `gorm:"column:type"`
	Value       float64    `gorm:"column:value"`
	MinOrder    float64    `gorm:"column:min_order"`
	MaxDiscount *float64   `gorm:"column:max_discount"`
	StartTime   time.Time  `gorm:"column:start_time"`
	EndTime     time.Time  `gorm:"column:end_time"`
	TotalCount  int        `gorm:"column:total_count"`
	UsedCount   int        `gorm:"column:used_count"`
	IsActive    bool       `gorm:"column:is_active"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Coupon) TableName() string { return "coupons" }

type UserCoupon struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	UserID    int64      `gorm:"column:user_id"`
	CouponID  int64      `gorm:"column:coupon_id"`
	Status    string     `gorm:"column:status"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	IssuedBy  *int64     `gorm:"column:issued_by"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (UserCoupon) TableName() string { return "user_coupons" }

type UserPoints struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	UserID    int64      `gorm:"column:user_id"`
	Points    float64    `gorm:"column:points"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (UserPoints) TableName() string { return "user_points" }

type PointsLog struct {
	ID                 int64     `gorm:"column:id;primaryKey"`
	UserID             int64     `gorm:"column:user_id;index:idx_points_log_user_created,priority:1"`
	Amount             float64   `gorm:"column:amount"`
	BalanceAfter       float64   `gorm:"column:balance_after"`
	Type               string    `gorm:"column:type;index:idx_points_log_type_created,priority:1"`
	Reason             *string   `gorm:"column:reason"`
	PostID             *int64    `gorm:"column:post_id;index:idx_points_log_post_created,priority:1"`
	PurchaseID         *int64    `gorm:"column:purchase_id;index:idx_points_log_purchase_role,priority:1"`
	EntryRole          string    `gorm:"column:entry_role;size:32;index:idx_points_log_purchase_role,priority:2"`
	CounterpartyUserID *int64    `gorm:"column:counterparty_user_id"`
	PaymentMethod      string    `gorm:"column:payment_method;size:32"`
	CreatedAt          time.Time `gorm:"column:created_at;index:idx_points_log_created_at;index:idx_points_log_user_created,priority:2;index:idx_points_log_type_created,priority:2;index:idx_points_log_post_created,priority:2"`
}

func (PointsLog) TableName() string { return "points_log" }

type PointsTaskConfig struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	TaskType    string     `gorm:"column:task_type;size:64;uniqueIndex:idx_points_task_configs_task_type"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	Points      float64    `gorm:"column:points"`
	DailyLimit  int        `gorm:"column:daily_limit"`
	IsDailyTask bool       `gorm:"column:is_daily_task"`
	IsActive    bool       `gorm:"column:is_active"`
	SortOrder   int        `gorm:"column:sort_order"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (PointsTaskConfig) TableName() string { return "points_task_configs" }

type PointsTaskEvent struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	UserID    int64     `gorm:"column:user_id;uniqueIndex:idx_points_task_events_daily_target,priority:1"`
	TaskType  string    `gorm:"column:task_type;size:64;uniqueIndex:idx_points_task_events_daily_target,priority:2"`
	TargetKey string    `gorm:"column:target_key;size:191;uniqueIndex:idx_points_task_events_daily_target,priority:3"`
	EventDate time.Time `gorm:"column:event_date;type:date;index:idx_points_task_events_date;uniqueIndex:idx_points_task_events_daily_target,priority:4"`
	Points    float64   `gorm:"column:points"`
	Reason    *string   `gorm:"column:reason"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (PointsTaskEvent) TableName() string { return "points_task_events" }

type PointsDailyStat struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	UserID         int64      `gorm:"column:user_id;index:idx_points_daily_stats_user_date,priority:1;uniqueIndex:idx_points_daily_stats_user_task_date,priority:1"`
	TaskType       string     `gorm:"column:task_type;size:64;uniqueIndex:idx_points_daily_stats_user_task_date,priority:2"`
	StatDate       time.Time  `gorm:"column:stat_date;type:date;index:idx_points_daily_stats_user_date,priority:2;uniqueIndex:idx_points_daily_stats_user_task_date,priority:3"`
	CompletedCount int        `gorm:"column:completed_count"`
	AwardedPoints  float64    `gorm:"column:awarded_points"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at"`
}

func (PointsDailyStat) TableName() string { return "points_daily_stats" }

type PointsAchievementRule struct {
	ID                  int64      `gorm:"column:id;primaryKey"`
	Name                string     `gorm:"column:name"`
	TriggerType         string     `gorm:"column:trigger_type"`
	ThresholdValue      int        `gorm:"column:threshold_value"`
	PointsReward        float64    `gorm:"column:points_reward"`
	CreatorBonusPercent float64    `gorm:"column:creator_bonus_percent"`
	BonusDays           int        `gorm:"column:bonus_days"`
	Description         *string    `gorm:"column:description"`
	IsActive            bool       `gorm:"column:is_active;index:idx_points_achievement_rules_active"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           *time.Time `gorm:"column:updated_at"`
}

func (PointsAchievementRule) TableName() string { return "points_achievement_rules" }

type UserAchievementReward struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	UserID              int64     `gorm:"column:user_id;uniqueIndex:idx_user_achievement_rewards_user_rule,priority:1"`
	RuleID              int64     `gorm:"column:rule_id;uniqueIndex:idx_user_achievement_rewards_user_rule,priority:2"`
	PointsAwarded       float64   `gorm:"column:points_awarded"`
	CreatorBonusPercent float64   `gorm:"column:creator_bonus_percent"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

func (UserAchievementReward) TableName() string { return "user_achievement_rewards" }

type UserCreatorBonus struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	UserID       int64      `gorm:"column:user_id;index:idx_user_creator_bonus_active,priority:1;uniqueIndex:idx_user_creator_bonus_user_rule,priority:1"`
	RuleID       *int64     `gorm:"column:rule_id;uniqueIndex:idx_user_creator_bonus_user_rule,priority:2"`
	BonusPercent float64    `gorm:"column:bonus_percent"`
	IsActive     bool       `gorm:"column:is_active;index:idx_user_creator_bonus_active,priority:2"`
	StartsAt     time.Time  `gorm:"column:starts_at;index:idx_user_creator_bonus_active,priority:3"`
	ExpiresAt    *time.Time `gorm:"column:expires_at;index:idx_user_creator_bonus_active,priority:4"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (UserCreatorBonus) TableName() string { return "user_creator_bonus" }

type GiftCardProduct struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	Name           string     `gorm:"column:name"`
	Description    *string    `gorm:"column:description"`
	FaceValue      *string    `gorm:"column:face_value"`
	PointsRequired float64    `gorm:"column:points_required"`
	IsActive       bool       `gorm:"column:is_active;index:idx_gift_card_products_active_sort,priority:1"`
	SortOrder      int        `gorm:"column:sort_order;index:idx_gift_card_products_active_sort,priority:2"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at"`
}

func (GiftCardProduct) TableName() string { return "gift_card_products" }

type GiftCardCode struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	ProductID    int64      `gorm:"column:product_id;index:idx_gift_card_codes_stock,priority:1;uniqueIndex:idx_gift_card_codes_product_code,priority:1"`
	Code         string     `gorm:"column:code;uniqueIndex:idx_gift_card_codes_product_code,priority:2"`
	Status       string     `gorm:"column:status;size:32;index:idx_gift_card_codes_stock,priority:2"`
	ImportBatch  *string    `gorm:"column:import_batch"`
	RedemptionID *int64     `gorm:"column:redemption_id"`
	UserID       *int64     `gorm:"column:user_id"`
	RedeemedAt   *time.Time `gorm:"column:redeemed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (GiftCardCode) TableName() string { return "gift_card_codes" }

type GiftCardRedemption struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	UserID       int64     `gorm:"column:user_id;index:idx_gift_card_redemptions_user_created,priority:1"`
	ProductID    int64     `gorm:"column:product_id;index:idx_gift_card_redemptions_product_created,priority:1"`
	CodeID       int64     `gorm:"column:code_id"`
	CodeSnapshot string    `gorm:"column:code_snapshot"`
	PointsSpent  float64   `gorm:"column:points_spent"`
	BalanceAfter float64   `gorm:"column:balance_after"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at;index:idx_gift_card_redemptions_user_created,priority:2;index:idx_gift_card_redemptions_product_created,priority:2"`
}

func (GiftCardRedemption) TableName() string { return "gift_card_redemptions" }

type UserWallet struct {
	UserID       int64      `gorm:"column:user_id;primaryKey"`
	CashBalance  float64    `gorm:"column:cash_balance"`
	TotalIncome  float64    `gorm:"column:total_income"`
	FrozenAmount float64    `gorm:"column:frozen_amount"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
}

func (UserWallet) TableName() string { return "user_wallet" }

type ExternalBalanceAccount struct {
	OAuth2ID          int64      `gorm:"column:oauth2_id;primaryKey"`
	UserID            int64      `gorm:"column:user_id;uniqueIndex:idx_external_balance_accounts_user"`
	ActiveOperationID *int64     `gorm:"column:active_operation_id"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at"`
}

func (ExternalBalanceAccount) TableName() string { return "external_balance_accounts" }

type ExternalBalanceTransaction struct {
	ID                 int64      `json:"id" gorm:"column:id;primaryKey"`
	OperationKey       string     `json:"operation_key" gorm:"column:operation_key;size:160;uniqueIndex:idx_external_balance_transactions_operation"`
	UserID             int64      `json:"user_id" gorm:"column:user_id;index:idx_external_balance_transactions_user_created,priority:1"`
	OAuth2ID           int64      `json:"oauth2_id" gorm:"column:oauth2_id;index:idx_external_balance_transactions_oauth_created,priority:1"`
	Amount             float64    `json:"amount" gorm:"column:amount"`
	Reason             string     `json:"reason" gorm:"column:reason;size:500"`
	PostID             *int64     `json:"post_id,omitempty" gorm:"column:post_id;index:idx_external_balance_transactions_post_created,priority:1"`
	PurchaseID         *int64     `json:"purchase_id,omitempty" gorm:"column:purchase_id"`
	CounterpartyUserID *int64     `json:"counterparty_user_id,omitempty" gorm:"column:counterparty_user_id"`
	EntryRole          string     `json:"entry_role,omitempty" gorm:"column:entry_role;size:32"`
	PaymentMethod      string     `json:"payment_method,omitempty" gorm:"column:payment_method;size:32"`
	PlatformFee        float64    `json:"platform_fee" gorm:"column:platform_fee"`
	Status             string     `json:"status" gorm:"column:status;size:32;index:idx_external_balance_transactions_status_updated,priority:1"`
	RemoteBalanceAfter *float64   `json:"remote_balance_after,omitempty" gorm:"column:remote_balance_after"`
	CompensationAmount *float64   `json:"compensation_amount,omitempty" gorm:"column:compensation_amount"`
	Attempts           int        `json:"attempts" gorm:"column:attempts"`
	LastError          *string    `json:"last_error,omitempty" gorm:"column:last_error;size:1000"`
	CreatedAt          time.Time  `json:"created_at" gorm:"column:created_at;index:idx_external_balance_transactions_user_created,priority:2;index:idx_external_balance_transactions_oauth_created,priority:2;index:idx_external_balance_transactions_post_created,priority:2"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty" gorm:"column:updated_at;index:idx_external_balance_transactions_status_updated,priority:2"`
	AppliedAt          *time.Time `json:"applied_at,omitempty" gorm:"column:applied_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
}

func (ExternalBalanceTransaction) TableName() string { return "external_balance_transactions" }

type UserPaymentCode struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	UserID    int64      `gorm:"column:user_id"`
	WechatURL *string    `gorm:"column:wechat_url"`
	AlipayURL *string    `gorm:"column:alipay_url"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (UserPaymentCode) TableName() string { return "user_payment_codes" }

type WithdrawOrder struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	UserID    int64      `gorm:"column:user_id"`
	Amount    float64    `gorm:"column:amount"`
	Type      string     `gorm:"column:type"`
	Status    string     `gorm:"column:status"`
	Remark    *string    `gorm:"column:remark"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (WithdrawOrder) TableName() string { return "withdraw_orders" }

type CreatorEarnings struct {
	ID              int64      `gorm:"column:id;primaryKey"`
	UserID          int64      `gorm:"column:user_id"`
	Balance         float64    `gorm:"column:balance"`
	TotalEarnings   float64    `gorm:"column:total_earnings"`
	WithdrawnAmount float64    `gorm:"column:withdrawn_amount"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (CreatorEarnings) TableName() string { return "creator_earnings" }

type CreatorEarningsLog struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	UserID       int64     `gorm:"column:user_id"`
	EarningsID   int64     `gorm:"column:earnings_id"`
	Amount       float64   `gorm:"column:amount"`
	BalanceAfter float64   `gorm:"column:balance_after"`
	Type         string    `gorm:"column:type"`
	SourceID     *int64    `gorm:"column:source_id"`
	SourceType   *string   `gorm:"column:source_type"`
	BuyerID      *int64    `gorm:"column:buyer_id"`
	Reason       *string   `gorm:"column:reason"`
	PlatformFee  float64   `gorm:"column:platform_fee"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (CreatorEarningsLog) TableName() string { return "creator_earnings_log" }

type Admin struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Username  string    `gorm:"column:username"`
	Password  string    `gorm:"column:password"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Admin) TableName() string { return "admin" }

type Audit struct {
	ID          int64          `gorm:"column:id;primaryKey"`
	UserID      int64          `gorm:"column:user_id;index:idx_audit_user_type_created,priority:1"`
	Type        int            `gorm:"column:type;index:idx_audit_type_created,priority:1;index:idx_audit_type_status_created,priority:1;index:idx_audit_user_type_created,priority:2"`
	TargetID    *int64         `gorm:"column:target_id"`
	Content     string         `gorm:"column:content"`
	AuditResult datatypes.JSON `gorm:"column:audit_result"`
	RiskLevel   *string        `gorm:"column:risk_level"`
	Categories  datatypes.JSON `gorm:"column:categories"`
	Reason      *string        `gorm:"column:reason"`
	RetryCount  int            `gorm:"column:retry_count"`
	CreatedAt   time.Time      `gorm:"column:created_at;index:idx_audit_type_created,priority:2;index:idx_audit_type_status_created,priority:3;index:idx_audit_user_type_created,priority:3"`
	AuditTime   *time.Time     `gorm:"column:audit_time"`
	Status      *int           `gorm:"column:status;index:idx_audit_type_status_created,priority:2"`
}

func (Audit) TableName() string { return "audit" }

type BannedWord struct {
	ID         int        `gorm:"column:id;primaryKey"`
	Word       string     `gorm:"column:word"`
	CategoryID *int       `gorm:"column:category_id"`
	IsRegex    bool       `gorm:"column:is_regex"`
	Enabled    bool       `gorm:"column:enabled"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (BannedWord) TableName() string { return "banned_words" }

type BannedWordCategory struct {
	ID          int        `gorm:"column:id;primaryKey"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (BannedWordCategory) TableName() string { return "banned_word_categories" }
