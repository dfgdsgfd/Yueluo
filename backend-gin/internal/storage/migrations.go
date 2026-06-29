package storage

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.AutoMigrate(AutoMigrateModels()...); err != nil {
		return err
	}
	if err := migrateAccessBlockRuleIndexes(db); err != nil {
		return err
	}
	return seedAutoMigrateDefaults(context.Background(), db)
}

func AutoMigrateModels() []any {
	return []any{
		&domain.User{},
		&domain.InviteCode{},
		&domain.InviteClick{},
		&domain.InviteEarningsLog{},
		&domain.Coupon{},
		&domain.UserCoupon{},
		&domain.UserPoints{},
		&domain.PointsLog{},
		&domain.PointsTaskConfig{},
		&domain.PointsTaskEvent{},
		&domain.PointsDailyStat{},
		&domain.PointsAchievementRule{},
		&domain.UserAchievementReward{},
		&domain.UserCreatorBonus{},
		&domain.GiftCardProduct{},
		&domain.GiftCardCode{},
		&domain.GiftCardRedemption{},
		&domain.UserWallet{},
		&domain.ExternalBalanceAccount{},
		&domain.ExternalBalanceTransaction{},
		&domain.UserPaymentCode{},
		&domain.WithdrawOrder{},
		&domain.CreatorEarnings{},
		&domain.CreatorEarningsLog{},
		&domain.Admin{},
		&domain.Audit{},
		&domain.BannedWord{},
		&domain.BannedWordCategory{},
		&domain.Post{},
		&domain.Comment{},
		&domain.Like{},
		&domain.Collection{},
		&domain.Follow{},
		&domain.Dislike{},
		&domain.Notification{},
		&domain.Blacklist{},
		&domain.Tag{},
		&domain.Category{},
		&domain.PostImage{},
		&domain.PostAttachment{},
		&domain.UploadAsset{},
		&domain.PostVideo{},
		&domain.FileRecycleItem{},
		&domain.PostPaymentSetting{},
		&domain.UserPurchasedContent{},
		&domain.ContentPurchaseIntent{},
		&domain.ImageProtectionJob{},
		&domain.ImageWatermarkTrace{},
		&domain.UserSearchHistory{},
		&domain.BrowsingHistory{},
		&domain.UserAPIKey{},
		&domain.UserToolbar{},
		&domain.SystemSetting{},
		&domain.SystemUpdateConfig{},
		&domain.SystemUpdateJob{},
		&domain.RecommendConfig{},
		&domain.PostRecommendConfig{},
		&domain.PostQualityRewardSetting{},
		&domain.PostTag{},
		&domain.Announcement{},
		&domain.License{},
		&domain.OpenAPI{},
		&domain.AppVersion{},
		&domain.AppUsageLog{},
		&domain.OAuthAppHandoff{},
		&domain.AccessLog{},
		&domain.SecurityAuditLog{},
		&domain.AccessBlockImportSource{},
		&domain.AccessBlockRule{},
		&domain.AIGenerationLog{},
		&domain.AIJob{},
		&domain.AIModerationLog{},
		&domain.MediaLibrary{},
		&domain.NotificationTemplate{},
		&domain.Report{},
		&domain.SystemNotification{},
		&domain.SystemNotificationConfirmation{},
		&domain.Feedback{},
		&domain.IMConversation{},
		&domain.IMConversationMember{},
		&domain.IMMessage{},
		&domain.IMMessageReceipt{},
	}
}

func migrateAccessBlockRuleIndexes(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&domain.AccessBlockRule{}) {
		return nil
	}
	if db.Migrator().HasIndex(&domain.AccessBlockRule{}, "idx_access_block_rules_unique") {
		if err := db.Migrator().DropIndex(&domain.AccessBlockRule{}, "idx_access_block_rules_unique"); err != nil {
			return err
		}
	}
	if !db.Migrator().HasIndex(&domain.AccessBlockRule{}, "idx_access_block_rules_source_unique") {
		return db.Migrator().CreateIndex(&domain.AccessBlockRule{}, "idx_access_block_rules_source_unique")
	}
	return nil
}

func seedAutoMigrateDefaults(ctx context.Context, db *gorm.DB) error {
	now := time.Now()
	for _, row := range defaultPointsTaskConfigs(now) {
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_type"}},
			DoNothing: true,
		}).Create(&row).Error; err != nil {
			return err
		}
	}

	description := "积分每日获取上限"
	setting := domain.SystemSetting{
		SettingKey:   "points_daily_cap",
		SettingValue: "50",
		SettingGroup: "points",
		Description:  &description,
		CreatedAt:    now,
		UpdatedAt:    &now,
	}
	if err := db.WithContext(ctx).Where(domain.SystemSetting{SettingKey: "points_daily_cap"}).
		Attrs(setting).
		FirstOrCreate(&domain.SystemSetting{}).Error; err != nil {
		return err
	}
	if _, err := EnsureInitialAdmin(ctx, db, DefaultAdminUsername, DefaultAdminInitialPassword); err != nil {
		return err
	}
	_, err := RandomizeLegacyUserPasswords(ctx, db)
	return err
}

func defaultPointsTaskConfigs(now time.Time) []domain.PointsTaskConfig {
	description := func(value string) *string { return &value }
	rows := []domain.PointsTaskConfig{
		{TaskType: "comment", Name: "评论", Description: description("发布评论自动获得积分"), Points: 2, DailyLimit: 10, IsDailyTask: true, IsActive: true, SortOrder: 10, CreatedAt: now},
		{TaskType: "click", Name: "点击", Description: description("点击进入内容详情自动获得积分"), Points: 1, DailyLimit: 30, IsDailyTask: true, IsActive: true, SortOrder: 20, CreatedAt: now},
		{TaskType: "like", Name: "点赞", Description: description("点赞内容自动获得积分"), Points: 1, DailyLimit: 20, IsDailyTask: true, IsActive: true, SortOrder: 30, CreatedAt: now},
		{TaskType: "collect", Name: "收藏", Description: description("收藏内容自动获得积分"), Points: 2, DailyLimit: 10, IsDailyTask: true, IsActive: true, SortOrder: 40, CreatedAt: now},
		{TaskType: "view", Name: "浏览", Description: description("浏览内容自动获得积分"), Points: 1, DailyLimit: 30, IsDailyTask: true, IsActive: true, SortOrder: 50, CreatedAt: now},
		{TaskType: "post", Name: "发帖", Description: description("发布公开内容自动获得积分"), Points: 5, DailyLimit: 5, IsDailyTask: true, IsActive: true, SortOrder: 60, CreatedAt: now},
		{TaskType: "set_avatar", Name: "设置头像", Description: description("设置头像获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 70, CreatedAt: now},
		{TaskType: "set_background", Name: "设置背景", Description: description("设置个人背景获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 80, CreatedAt: now},
		{TaskType: "set_signature", Name: "设置签名", Description: description("设置个人签名获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 90, CreatedAt: now},
		{TaskType: "set_name", Name: "设置名称", Description: description("设置名称获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 100, CreatedAt: now},
	}
	for i := range rows {
		rows[i].UpdatedAt = &now
	}
	return rows
}
