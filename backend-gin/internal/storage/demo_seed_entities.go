package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (state *demoSeedState) seedCategories(ctx context.Context, tx *gorm.DB, now time.Time) error {
	for _, seed := range demoCategorySeeds {
		category := domain.Category{
			Name:          seed.Name,
			CategoryTitle: stringPtr(seed.Title),
			Translations:  demoTranslations(seed.Title),
			CreatedAt:     now,
		}
		var row domain.Category
		result := tx.WithContext(ctx).Where("name = ?", seed.Name).Attrs(category).FirstOrCreate(&row)
		if result.Error != nil {
			return result.Error
		}
		state.result.CategoriesCreated += result.RowsAffected
		if err := tx.WithContext(ctx).Model(&domain.Category{}).Where("id = ?", row.ID).Updates(map[string]any{
			"category_title": seed.Title,
			"translations":   demoTranslations(seed.Title),
		}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("id = ?", row.ID).First(&row).Error; err != nil {
			return err
		}
		state.categories[seed.Name] = row
	}
	return nil
}

func (state *demoSeedState) seedTags(ctx context.Context, tx *gorm.DB, now time.Time) error {
	for _, seed := range demoTagSeeds {
		var row domain.Tag
		err := tx.WithContext(ctx).Where("name = ?", seed.Name).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			for _, legacyName := range seed.LegacyNames {
				err = tx.WithContext(ctx).Where("name = ?", legacyName).First(&row).Error
				if err == nil {
					break
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result := tx.WithContext(ctx).Create(&domain.Tag{Name: seed.Name, CreatedAt: now})
			if result.Error != nil {
				return result.Error
			}
			state.result.TagsCreated += result.RowsAffected
			if err := tx.WithContext(ctx).Where("name = ?", seed.Name).First(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if row.Name != seed.Name {
			if err := tx.WithContext(ctx).Model(&domain.Tag{}).Where("id = ?", row.ID).Update("name", seed.Name).Error; err != nil {
				return err
			}
			row.Name = seed.Name
		}
		state.tags[seed.Name] = row
	}
	return nil
}

func (state *demoSeedState) seedUsers(ctx context.Context, tx *gorm.DB, seeds []demoUserSeed, now time.Time) error {
	for _, seed := range seeds {
		createdAt := now.AddDate(0, 0, -seed.DaysAgo)
		updatedAt := now
		email := seed.Email
		user := domain.User{
			Password:         &state.passwordHash,
			UserID:           seed.UserID,
			Nickname:         seed.Nickname,
			Email:            &email,
			Avatar:           stringPtr(seed.Avatar),
			Bio:              stringPtr(seed.Bio),
			BioAuditStatus:   1,
			Location:         stringPtr(seed.Location),
			IsActive:         true,
			CreatedAt:        createdAt,
			UpdatedAt:        &updatedAt,
			Gender:           stringPtr(seed.Gender),
			Education:        stringPtr(seed.Education),
			Major:            stringPtr(seed.Major),
			MBTI:             stringPtr(seed.MBTI),
			Interests:        jsonValue(seed.Interests),
			CustomFields:     jsonValue(map[string]any{"demo": true}),
			ProfileCompleted: true,
		}
		var row domain.User
		result := tx.WithContext(ctx).Where("user_id = ?", seed.UserID).Attrs(user).FirstOrCreate(&row)
		if result.Error != nil {
			return result.Error
		}
		state.result.UsersCreated += result.RowsAffected
		updates := map[string]any{
			"password":          state.passwordHash,
			"nickname":          seed.Nickname,
			"avatar":            seed.Avatar,
			"bio":               seed.Bio,
			"bio_audit_status":  1,
			"location":          seed.Location,
			"is_active":         true,
			"updated_at":        now,
			"gender":            seed.Gender,
			"education":         seed.Education,
			"major":             seed.Major,
			"mbti":              seed.MBTI,
			"interests":         jsonValue(seed.Interests),
			"custom_fields":     jsonValue(map[string]any{"demo": true, "region": "中国"}),
			"profile_completed": true,
		}
		if state.emailAvailable(ctx, tx, seed.Email, row.ID) && (row.Email == nil || !strings.EqualFold(strings.TrimSpace(*row.Email), seed.Email)) {
			updates["email"] = seed.Email
		}
		if err := tx.WithContext(ctx).Model(&domain.User{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("id = ?", row.ID).First(&row).Error; err != nil {
			return err
		}
		state.users[seed.UserID] = row
		state.result.LoginAccounts = append(state.result.LoginAccounts, DemoLoginAccount{Account: seed.UserID, Email: seed.Email})
		if err := state.seedUserBalances(ctx, tx, row.ID, seed, now); err != nil {
			return err
		}
	}
	return nil
}

func (state *demoSeedState) emailAvailable(ctx context.Context, tx *gorm.DB, email string, excludeID int64) bool {
	var count int64
	err := tx.WithContext(ctx).Model(&domain.User{}).
		Where("LOWER(COALESCE(email, '')) = LOWER(?) AND id <> ?", strings.TrimSpace(email), excludeID).
		Count(&count).Error
	return err == nil && count == 0
}

func (state *demoSeedState) seedUserBalances(ctx context.Context, tx *gorm.DB, userID int64, seed demoUserSeed, now time.Time) error {
	updatedAt := now
	pointsSeed := domain.UserPoints{UserID: userID, Points: seed.Points, CreatedAt: now, UpdatedAt: &updatedAt}
	var points domain.UserPoints
	result := tx.WithContext(ctx).Where("user_id = ?", userID).Attrs(pointsSeed).FirstOrCreate(&points)
	if result.Error != nil {
		return result.Error
	}
	state.result.PointRowsCreated += result.RowsAffected
	if result.RowsAffected == 0 {
		if err := tx.WithContext(ctx).Model(&domain.UserPoints{}).Where("id = ?", points.ID).Updates(map[string]any{"points": seed.Points, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	pointReason := "演示初始积分"
	if err := state.ensurePointsLog(ctx, tx, userID, seed.Points, seed.Points, "demo_seed", &pointReason, now, "Demo starting balance"); err != nil {
		return err
	}

	walletSeed := domain.UserWallet{UserID: userID, CashBalance: seed.Wallet, TotalIncome: seed.Wallet, CreatedAt: now, UpdatedAt: &updatedAt}
	var wallet domain.UserWallet
	result = tx.WithContext(ctx).Where("user_id = ?", userID).Attrs(walletSeed).FirstOrCreate(&wallet)
	if result.Error != nil {
		return result.Error
	}
	state.result.WalletRowsCreated += result.RowsAffected
	if result.RowsAffected == 0 {
		if err := tx.WithContext(ctx).Model(&domain.UserWallet{}).Where("user_id = ?", userID).Updates(map[string]any{"cash_balance": seed.Wallet, "total_income": seed.Wallet, "updated_at": now}).Error; err != nil {
			return err
		}
	}

	var earnings domain.CreatorEarnings
	result = tx.WithContext(ctx).Where("user_id = ?", userID).Attrs(domain.CreatorEarnings{
		UserID:        userID,
		Balance:       seed.Earnings,
		TotalEarnings: seed.Earnings,
		CreatedAt:     now,
		UpdatedAt:     &updatedAt,
	}).FirstOrCreate(&earnings)
	if result.Error != nil {
		return result.Error
	}
	state.result.CreatorRowsCreated += result.RowsAffected
	if result.RowsAffected == 0 {
		if err := tx.WithContext(ctx).Model(&domain.CreatorEarnings{}).Where("id = ?", earnings.ID).Updates(map[string]any{"balance": seed.Earnings, "total_earnings": seed.Earnings, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	if seed.Earnings > 0 {
		reason := "演示创作者收益"
		reasons := []string{reason, "Demo creator earnings"}
		var row domain.CreatorEarningsLog
		err := tx.WithContext(ctx).Where("user_id = ? AND earnings_id = ? AND type = ? AND reason IN ?", userID, earnings.ID, "demo_seed", reasons).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result = tx.WithContext(ctx).Create(&domain.CreatorEarningsLog{UserID: userID, EarningsID: earnings.ID, Amount: seed.Earnings, BalanceAfter: seed.Earnings, Type: "demo_seed", Reason: &reason, CreatedAt: now})
			if result.Error != nil {
				return result.Error
			}
			state.result.CreatorRowsCreated += result.RowsAffected
		} else if err != nil {
			return err
		} else {
			if err := tx.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Where("id = ?", row.ID).Updates(map[string]any{"amount": seed.Earnings, "balance_after": seed.Earnings, "reason": reason}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (state *demoSeedState) ensurePointsLog(ctx context.Context, tx *gorm.DB, userID int64, amount float64, balance float64, logType string, reason *string, now time.Time, legacyReasons ...string) error {
	reasons := []string{deref(reason)}
	reasons = append(reasons, legacyReasons...)
	var row domain.PointsLog
	err := tx.WithContext(ctx).Where("user_id = ? AND type = ? AND reason IN ?", userID, logType, reasons).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := tx.WithContext(ctx).Create(&domain.PointsLog{UserID: userID, Amount: amount, BalanceAfter: balance, Type: logType, Reason: reason, PaymentMethod: "points", CreatedAt: now})
		if result.Error != nil {
			return result.Error
		}
		state.result.PointRowsCreated += result.RowsAffected
		return nil
	}
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&domain.PointsLog{}).Where("id = ?", row.ID).Updates(map[string]any{
		"amount":         amount,
		"balance_after":  balance,
		"reason":         reason,
		"payment_method": "points",
	}).Error
}

func (state *demoSeedState) seedRelationships(ctx context.Context, tx *gorm.DB, now time.Time) error {
	pairs := [][2]string{
		{"demo_alice", "demo_ben"},
		{"demo_ben", "demo_alice"},
		{"demo_alice", "demo_cora"},
		{"demo_cora", "demo_alice"},
		{"demo_drew", "demo_alice"},
		{"demo_elin", "demo_ben"},
		{"demo_finn", "demo_cora"},
		{"demo_ben", "demo_drew"},
		{"demo_cora", "demo_elin"},
	}
	for _, pair := range pairs {
		follower, okA := state.users[pair[0]]
		following, okB := state.users[pair[1]]
		if !okA || !okB {
			continue
		}
		result := tx.WithContext(ctx).Where("follower_id = ? AND following_id = ?", follower.ID, following.ID).
			Attrs(domain.Follow{FollowerID: follower.ID, FollowingID: following.ID, CreatedAt: now.Add(-48 * time.Hour)}).
			FirstOrCreate(&domain.Follow{})
		if result.Error != nil {
			return result.Error
		}
		state.result.FollowsCreated += result.RowsAffected
	}
	return nil
}

func (state *demoSeedState) seedPosts(ctx context.Context, tx *gorm.DB, seeds []demoPostSeed, now time.Time) error {
	for _, seed := range seeds {
		author, ok := state.users[seed.Author]
		if !ok {
			continue
		}
		category := state.categories[seed.Category]
		createdAt := now.AddDate(0, 0, -seed.DaysAgo).Add(time.Duration(seed.HoursOffset) * time.Hour)
		quality := seed.QualityLevel
		if strings.TrimSpace(quality) == "" {
			quality = "none"
		}
		post := domain.Post{
			UserID:             author.ID,
			Title:              seed.Title,
			Content:            seed.Content,
			CategoryID:         &category.ID,
			Type:               seed.Type,
			ViewCount:          seed.ViewCount,
			CreatedAt:          createdAt,
			IsDraft:            false,
			Visibility:         demoVisibilityPublic,
			PublicAccessExempt: true,
			AuditStatus:        1,
			AuditResult:        jsonValue(map[string]any{"demo": true, "status": "approved"}),
			QualityLevel:       quality,
		}
		var row domain.Post
		err := tx.WithContext(ctx).Where("user_id = ? AND title IN ?", author.ID, seed.postTitles()).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result := tx.WithContext(ctx).Create(&post)
			if result.Error != nil {
				return result.Error
			}
			state.result.PostsCreated += result.RowsAffected
			row = post
		} else if err != nil {
			return err
		} else {
			if err := tx.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", row.ID).Updates(map[string]any{
				"title":                seed.Title,
				"content":              seed.Content,
				"category_id":          category.ID,
				"type":                 seed.Type,
				"view_count":           seed.ViewCount,
				"is_draft":             false,
				"visibility":           demoVisibilityPublic,
				"public_access_exempt": true,
				"audit_status":         1,
				"audit_result":         jsonValue(map[string]any{"demo": true, "status": "approved"}),
				"quality_level":        quality,
			}).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Where("id = ?", row.ID).First(&row).Error; err != nil {
				return err
			}
		}
		state.posts[seed.Key] = row
		if err := state.seedPostChildren(ctx, tx, row, seed, now); err != nil {
			return err
		}
	}
	return nil
}

func (state *demoSeedState) seedPostChildren(ctx context.Context, tx *gorm.DB, post domain.Post, seed demoPostSeed, now time.Time) error {
	for index, imageURL := range seed.Images {
		row := domain.PostImage{
			PostID:        post.ID,
			ImageURL:      imageURL,
			IsFreePreview: index == 0,
			IsProtected:   index > 0 && seed.Payment != nil,
			SortOrder:     index + 1,
		}
		result := tx.WithContext(ctx).Where("post_id = ? AND image_url = ?", post.ID, imageURL).Attrs(row).FirstOrCreate(&domain.PostImage{})
		if result.Error != nil {
			return result.Error
		}
		state.result.PostImagesCreated += result.RowsAffected
	}
	if strings.TrimSpace(seed.VideoURL) != "" {
		video := domain.PostVideo{PostID: post.ID, VideoURL: seed.VideoURL}
		if strings.TrimSpace(seed.VideoCover) != "" {
			video.CoverURL = stringPtr(seed.VideoCover)
		}
		result := tx.WithContext(ctx).Where("post_id = ? AND video_url = ?", post.ID, seed.VideoURL).Attrs(video).FirstOrCreate(&domain.PostVideo{})
		if result.Error != nil {
			return result.Error
		}
		state.result.PostVideosCreated += result.RowsAffected
	}
	if seed.Attachment != nil {
		attachment := domain.PostAttachment{
			PostID:        post.ID,
			AttachmentURL: seed.Attachment.URL,
			Filename:      seed.Attachment.Filename,
			Filesize:      seed.Attachment.Filesize,
			CreatedAt:     now,
		}
		var row domain.PostAttachment
		result := tx.WithContext(ctx).Where("post_id = ? AND attachment_url = ?", post.ID, seed.Attachment.URL).Attrs(attachment).FirstOrCreate(&row)
		if result.Error != nil {
			return result.Error
		}
		state.result.PostAttachmentsCreated += result.RowsAffected
		if result.RowsAffected == 0 {
			if err := tx.WithContext(ctx).Model(&domain.PostAttachment{}).Where("id = ?", row.ID).Updates(map[string]any{"filename": seed.Attachment.Filename, "filesize": seed.Attachment.Filesize}).Error; err != nil {
				return err
			}
		}
	}
	if seed.Payment != nil {
		payment := domain.PostPaymentSetting{
			PostID:           post.ID,
			Enabled:          true,
			PaymentType:      nonEmpty(seed.Payment.PaymentType, "content"),
			PaymentMethod:    nonEmpty(seed.Payment.PaymentMethod, "points"),
			Price:            seed.Payment.Price,
			FreePreviewCount: seed.Payment.FreePreviewCount,
			PreviewDuration:  seed.Payment.PreviewDuration,
			HideAll:          seed.Payment.HideAll,
			CreatedAt:        now,
			UpdatedAt:        &now,
		}
		var row domain.PostPaymentSetting
		result := tx.WithContext(ctx).Where("post_id = ?", post.ID).Attrs(payment).FirstOrCreate(&row)
		if result.Error != nil {
			return result.Error
		}
		state.result.PostPaymentSettingsCreated += result.RowsAffected
		if result.RowsAffected == 0 {
			if err := tx.WithContext(ctx).Model(&domain.PostPaymentSetting{}).Where("id = ?", row.ID).Updates(map[string]any{
				"enabled":            true,
				"payment_type":       payment.PaymentType,
				"payment_method":     payment.PaymentMethod,
				"price":              payment.Price,
				"free_preview_count": payment.FreePreviewCount,
				"preview_duration":   payment.PreviewDuration,
				"hide_all":           payment.HideAll,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
		}
	}
	for _, tagName := range seed.Tags {
		tag, ok := state.tags[tagName]
		if !ok {
			continue
		}
		result := tx.WithContext(ctx).Where("post_id = ? AND tag_id = ?", post.ID, tag.ID).Attrs(domain.PostTag{PostID: post.ID, TagID: tag.ID}).FirstOrCreate(&domain.PostTag{})
		if result.Error != nil {
			return result.Error
		}
		state.result.PostTagsCreated += result.RowsAffected
	}
	return nil
}
