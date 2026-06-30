package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (state *demoSeedState) seedContentActivity(ctx context.Context, tx *gorm.DB, seeds []demoPostSeed, now time.Time) error {
	activityUsers := []string{"demo_alice", "demo_ben", "demo_cora", "demo_drew", "demo_elin", "demo_finn"}
	for index, seed := range seeds {
		post, ok := state.posts[seed.Key]
		if !ok {
			continue
		}
		authorID := post.UserID
		for offset, userKey := range activityUsers {
			user, ok := state.users[userKey]
			if !ok || user.ID == authorID {
				continue
			}
			if offset%2 == index%2 || offset < 3 {
				if err := state.ensureLike(ctx, tx, user.ID, 1, post.ID, now.Add(time.Duration(offset)*time.Minute)); err != nil {
					return err
				}
			}
			if offset%3 == index%3 {
				if err := state.ensureCollection(ctx, tx, user.ID, post.ID, now.Add(time.Duration(offset+10)*time.Minute)); err != nil {
					return err
				}
			}
			if offset < 2 {
				if err := state.ensureBrowsingHistory(ctx, tx, user.ID, post.ID, now.Add(time.Duration(offset+20)*time.Minute)); err != nil {
					return err
				}
			}
		}
		firstCommentUser := state.pickUser(activityUsers[(index+1)%len(activityUsers)], authorID)
		secondCommentUser := state.pickUser(activityUsers[(index+2)%len(activityUsers)], authorID)
		if firstCommentUser.ID != 0 {
			comment, err := state.ensureComment(ctx, tx, post.ID, firstCommentUser.ID, nil, "这条中文演示内容让信息流更接近真实使用场景。", now.Add(time.Duration(index)*time.Hour), "This demo post makes the feed feel alive.")
			if err != nil {
				return err
			}
			if secondCommentUser.ID != 0 {
				reply, err := state.ensureComment(ctx, tx, post.ID, secondCommentUser.ID, &comment.ID, "同意，这组数据很适合做中文界面和流程验收。", now.Add(time.Duration(index)*time.Hour+15*time.Minute), "Agreed. The sample data is useful for QA.")
				if err != nil {
					return err
				}
				if err := state.ensureLike(ctx, tx, firstCommentUser.ID, 2, reply.ID, now.Add(time.Duration(index)*time.Hour+20*time.Minute)); err != nil {
					return err
				}
			}
		}
		keyword := seed.Category
		if category, ok := state.categories[seed.Category]; ok {
			keyword = nonEmpty(deref(category.CategoryTitle), seed.Category)
		}
		if err := state.ensureSearchHistory(ctx, tx, authorID, keyword, now.Add(time.Duration(index)*time.Minute), seed.Category); err != nil {
			return err
		}
	}
	return nil
}

func (state *demoSeedState) pickUser(key string, notID int64) domain.User {
	user := state.users[key]
	if user.ID != notID {
		return user
	}
	for _, candidate := range state.users {
		if candidate.ID != notID {
			return candidate
		}
	}
	return domain.User{}
}

func (state *demoSeedState) ensureComment(ctx context.Context, tx *gorm.DB, postID int64, userID int64, parentID *int64, content string, createdAt time.Time, legacyContents ...string) (domain.Comment, error) {
	row := domain.Comment{PostID: postID, UserID: userID, ParentID: parentID, Content: content, AuditStatus: 1, IsPublic: true, AuditResult: jsonValue(map[string]any{"demo": true}), CreatedAt: createdAt}
	contents := []string{content}
	contents = append(contents, legacyContents...)
	query := tx.WithContext(ctx).Where("post_id = ? AND user_id = ? AND content IN ?", postID, userID, contents)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	err := query.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := tx.WithContext(ctx).Create(&row)
		if result.Error != nil {
			return domain.Comment{}, result.Error
		}
		state.result.CommentsCreated += result.RowsAffected
		return row, nil
	}
	if err != nil {
		return domain.Comment{}, err
	}
	if row.Content != content {
		if err := tx.WithContext(ctx).Model(&domain.Comment{}).Where("id = ?", row.ID).Updates(map[string]any{"content": content, "audit_result": jsonValue(map[string]any{"demo": true})}).Error; err != nil {
			return domain.Comment{}, err
		}
		row.Content = content
	}
	return row, nil
}

func (state *demoSeedState) ensureLike(ctx context.Context, tx *gorm.DB, userID int64, targetType int, targetID int64, createdAt time.Time) error {
	result := tx.WithContext(ctx).Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Attrs(domain.Like{UserID: userID, TargetType: targetType, TargetID: targetID, CreatedAt: createdAt}).
		FirstOrCreate(&domain.Like{})
	if result.Error != nil {
		return result.Error
	}
	state.result.LikesCreated += result.RowsAffected
	return nil
}

func (state *demoSeedState) ensureCollection(ctx context.Context, tx *gorm.DB, userID int64, postID int64, createdAt time.Time) error {
	result := tx.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).
		Attrs(domain.Collection{UserID: userID, PostID: postID, CreatedAt: createdAt}).
		FirstOrCreate(&domain.Collection{})
	if result.Error != nil {
		return result.Error
	}
	state.result.CollectionsCreated += result.RowsAffected
	return nil
}

func (state *demoSeedState) ensureBrowsingHistory(ctx context.Context, tx *gorm.DB, userID int64, postID int64, at time.Time) error {
	result := tx.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).
		Attrs(domain.BrowsingHistory{UserID: userID, PostID: postID, CreatedAt: at, UpdatedAt: &at}).
		FirstOrCreate(&domain.BrowsingHistory{})
	if result.Error != nil {
		return result.Error
	}
	state.result.HistoryRowsCreated += result.RowsAffected
	return nil
}

func (state *demoSeedState) ensureSearchHistory(ctx context.Context, tx *gorm.DB, userID int64, keyword string, at time.Time, legacyKeywords ...string) error {
	keywords := []string{keyword}
	keywords = append(keywords, legacyKeywords...)
	var row domain.UserSearchHistory
	err := tx.WithContext(ctx).Where("user_id = ? AND keyword IN ?", userID, keywords).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := tx.WithContext(ctx).Create(&domain.UserSearchHistory{UserID: userID, Keyword: keyword, CreatedAt: at})
		if result.Error != nil {
			return result.Error
		}
		state.result.HistoryRowsCreated += result.RowsAffected
		return nil
	}
	if err != nil {
		return err
	}
	if row.Keyword != keyword {
		return tx.WithContext(ctx).Model(&domain.UserSearchHistory{}).Where("id = ?", row.ID).Update("keyword", keyword).Error
	}
	return nil
}

func (state *demoSeedState) seedCommerce(ctx context.Context, tx *gorm.DB, now time.Time) error {
	products := []struct {
		Product    domain.GiftCardProduct
		LegacyName string
	}{
		{Product: domain.GiftCardProduct{Name: "演示咖啡兑换卡", Description: stringPtr("用于积分兑换流程测试的中文演示奖励。"), FaceValue: stringPtr("10元"), PointsRequired: 120, IsActive: true, SortOrder: 10, CreatedAt: now, UpdatedAt: &now}, LegacyName: "Demo Coffee Card"},
		{Product: domain.GiftCardProduct{Name: "演示书店礼品卡", Description: stringPtr("用于创作者礼品卡流程测试的中文演示奖励。"), FaceValue: stringPtr("25元"), PointsRequired: 260, IsActive: true, SortOrder: 20, CreatedAt: now, UpdatedAt: &now}, LegacyName: "Demo Book Card"},
	}
	for index, seed := range products {
		var row domain.GiftCardProduct
		err := tx.WithContext(ctx).Where("name IN ?", []string{seed.Product.Name, seed.LegacyName}).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result := tx.WithContext(ctx).Create(&seed.Product)
			if result.Error != nil {
				return result.Error
			}
			state.result.GiftCardRowsCreated += result.RowsAffected
			row = seed.Product
		} else if err != nil {
			return err
		} else {
			if err := tx.WithContext(ctx).Model(&domain.GiftCardProduct{}).Where("id = ?", row.ID).Updates(map[string]any{
				"name":            seed.Product.Name,
				"description":     seed.Product.Description,
				"face_value":      seed.Product.FaceValue,
				"points_required": seed.Product.PointsRequired,
				"is_active":       seed.Product.IsActive,
				"sort_order":      seed.Product.SortOrder,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}
		}
		for codeIndex := 1; codeIndex <= 3; codeIndex++ {
			code := fmt.Sprintf("DEMO-%02d-%02d", index+1, codeIndex)
			result := tx.WithContext(ctx).Where("product_id = ? AND code = ?", row.ID, code).
				Attrs(domain.GiftCardCode{ProductID: row.ID, Code: code, Status: demoGiftCardCodeStatusAvailable, ImportBatch: stringPtr("demo-seed"), CreatedAt: now, UpdatedAt: &now}).
				FirstOrCreate(&domain.GiftCardCode{})
			if result.Error != nil {
				return result.Error
			}
			state.result.GiftCardRowsCreated += result.RowsAffected
		}
	}
	return nil
}

func (state *demoSeedState) seedSystemRows(ctx context.Context, tx *gorm.DB, now time.Time) error {
	publishedAt := now.Add(-2 * time.Hour)
	expiresAt := now.AddDate(0, 1, 0)
	announcement := domain.Announcement{
		Title:       "中文演示数据已准备好",
		Content:     "系统已准备好来自中国场景的演示用户、帖子、评论、钱包余额、礼品卡和站内消息，适合本地中文验收。",
		Type:        "info",
		IsPublished: true,
		PublishedAt: &publishedAt,
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
		UpdatedAt:   &now,
	}
	var existingAnnouncement domain.Announcement
	result := tx.WithContext(ctx).Where("title IN ?", []string{announcement.Title, "Demo data is available"}).First(&existingAnnouncement)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = tx.WithContext(ctx).Create(&announcement)
		if result.Error != nil {
			return result.Error
		}
		state.result.SystemRowsCreated += result.RowsAffected
	} else if result.Error != nil {
		return result.Error
	} else {
		if err := tx.WithContext(ctx).Model(&domain.Announcement{}).Where("id = ?", existingAnnouncement.ID).Updates(announcement).Error; err != nil {
			return err
		}
	}

	systemNotification := domain.SystemNotification{
		Title:         "中文演示环境已就绪",
		Content:       "可使用 demo_alice 或 demo-alice@example.test 搭配配置的演示密码体验中文社区数据。",
		Type:          "info",
		ContentFormat: "markdown",
		ShowPopup:     true,
		IsActive:      true,
		StartTime:     &publishedAt,
		EndTime:       &expiresAt,
		CreatedAt:     now,
		UpdatedAt:     &now,
	}
	var existingNotification domain.SystemNotification
	result = tx.WithContext(ctx).Where("title IN ?", []string{systemNotification.Title, "Demo workspace ready"}).First(&existingNotification)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = tx.WithContext(ctx).Create(&systemNotification)
		if result.Error != nil {
			return result.Error
		}
		state.result.SystemRowsCreated += result.RowsAffected
	} else if result.Error != nil {
		return result.Error
	} else {
		if err := tx.WithContext(ctx).Model(&domain.SystemNotification{}).Where("id = ?", existingNotification.ID).Updates(systemNotification).Error; err != nil {
			return err
		}
	}
	return nil
}

func (state *demoSeedState) seedIM(ctx context.Context, tx *gorm.DB, now time.Time) error {
	alice, okA := state.users["demo_alice"]
	ben, okB := state.users["demo_ben"]
	if !okA || !okB {
		return nil
	}
	name := "中文演示会话"
	legacyName := "Demo chat"
	var conversation domain.IMConversation
	result := tx.WithContext(ctx).Where("type = ? AND creator_id = ? AND name IN ?", "direct", alice.ID, []string{name, legacyName}).First(&conversation)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = tx.WithContext(ctx).Create(&domain.IMConversation{Type: "direct", Name: &name, CreatorID: alice.ID, CreatedAt: now.Add(-6 * time.Hour), UpdatedAt: &now})
		if result.Error != nil {
			return result.Error
		}
		state.result.IMRowsCreated += result.RowsAffected
		if err := tx.WithContext(ctx).Where("type = ? AND creator_id = ? AND name = ?", "direct", alice.ID, name).First(&conversation).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else if deref(conversation.Name) != name {
		if err := tx.WithContext(ctx).Model(&domain.IMConversation{}).Where("id = ?", conversation.ID).Updates(map[string]any{"name": name, "updated_at": now}).Error; err != nil {
			return err
		}
		conversation.Name = &name
	}
	for _, user := range []domain.User{alice, ben} {
		result = tx.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conversation.ID, user.ID).
			Attrs(domain.IMConversationMember{ConversationID: conversation.ID, UserID: user.ID, JoinedAt: now.Add(-6 * time.Hour)}).
			FirstOrCreate(&domain.IMConversationMember{})
		if result.Error != nil {
			return result.Error
		}
		state.result.IMRowsCreated += result.RowsAffected
	}
	messages := []struct {
		Sender domain.User
		ID     string
		Text   string
		At     time.Time
	}{
		{alice, "demo-msg-001", "中文演示信息流已经准备好，可以开始验收。", now.Add(-5 * time.Hour)},
		{ben, "demo-msg-002", "收到，我会检查登录、评论、钱包和通知流程。", now.Add(-5*time.Hour + 8*time.Minute)},
		{alice, "demo-msg-003", "这条会话也可以用来测试消息未读角标。", now.Add(-5*time.Hour + 18*time.Minute)},
	}
	var lastMessageID *int64
	for _, message := range messages {
		row := domain.IMMessage{ConversationID: conversation.ID, SenderID: message.Sender.ID, Content: message.Text, ClientMsgID: &message.ID, CreatedAt: message.At}
		result = tx.WithContext(ctx).Where("conversation_id = ? AND client_msg_id = ?", conversation.ID, message.ID).First(&row)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			result = tx.WithContext(ctx).Create(&row)
			if result.Error != nil {
				return result.Error
			}
			state.result.IMRowsCreated += result.RowsAffected
		} else if result.Error != nil {
			return result.Error
		} else if row.Content != message.Text {
			if err := tx.WithContext(ctx).Model(&domain.IMMessage{}).Where("id = ?", row.ID).Update("content", message.Text).Error; err != nil {
				return err
			}
			row.Content = message.Text
		}
		lastMessageID = &row.ID
		for _, user := range []domain.User{alice, ben} {
			deliveredAt := message.At.Add(time.Minute)
			var readAt *time.Time
			if user.ID == message.Sender.ID {
				readAt = &deliveredAt
			}
			result = tx.WithContext(ctx).Where("message_id = ? AND user_id = ?", row.ID, user.ID).
				Attrs(domain.IMMessageReceipt{MessageID: row.ID, UserID: user.ID, DeliveredAt: &deliveredAt, ReadAt: readAt, UpdatedAt: &deliveredAt}).
				FirstOrCreate(&domain.IMMessageReceipt{})
			if result.Error != nil {
				return result.Error
			}
			state.result.IMRowsCreated += result.RowsAffected
		}
	}
	if lastMessageID != nil {
		if err := tx.WithContext(ctx).Model(&domain.IMConversation{}).Where("id = ?", conversation.ID).Updates(map[string]any{"last_message_id": *lastMessageID, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (state *demoSeedState) recountAffectedRows(ctx context.Context, tx *gorm.DB, now time.Time) error {
	for _, post := range state.posts {
		var likeCount, collectCount, commentCount int64
		if err := tx.WithContext(ctx).Model(&domain.Like{}).Where("target_type = ? AND target_id = ?", 1, post.ID).Count(&likeCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Collection{}).Where("post_id = ?", post.ID).Count(&collectCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Comment{}).Where("post_id = ? AND is_public = ?", post.ID, true).Count(&commentCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", post.ID).Updates(map[string]any{
			"like_count":    int(likeCount),
			"collect_count": int(collectCount),
			"comment_count": int(commentCount),
		}).Error; err != nil {
			return err
		}
		targetID := post.ID
		for _, likeUser := range state.users {
			if likeUser.ID == post.UserID {
				continue
			}
			var existing int64
			if err := tx.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ? AND type = ? AND target_id = ?", post.UserID, 1, post.ID).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				if err := tx.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ? AND type = ? AND target_id = ?", post.UserID, 1, post.ID).Update("title", "点赞了你的帖子").Error; err != nil {
					return err
				}
				break
			}
			var liked int64
			if err := tx.WithContext(ctx).Model(&domain.Like{}).Where("user_id = ? AND target_type = ? AND target_id = ?", likeUser.ID, 1, post.ID).Count(&liked).Error; err != nil {
				return err
			}
			if liked == 0 {
				continue
			}
			result := tx.WithContext(ctx).Create(&domain.Notification{UserID: post.UserID, SenderID: likeUser.ID, Type: 1, Title: "点赞了你的帖子", TargetID: &targetID, CreatedAt: now})
			if result.Error != nil {
				return result.Error
			}
			state.result.NotificationsCreated += result.RowsAffected
			break
		}
	}
	for _, user := range state.users {
		var followCount, fansCount, likeCount int64
		if err := tx.WithContext(ctx).Model(&domain.Follow{}).Where("follower_id = ?", user.ID).Count(&followCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Follow{}).Where("following_id = ?", user.ID).Count(&fansCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Post{}).Where("user_id = ?", user.ID).Select("COALESCE(SUM(like_count), 0)").Scan(&likeCount).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.User{}).Where("id = ?", user.ID).Updates(map[string]any{"follow_count": int(followCount), "fans_count": int(fansCount), "like_count": int(likeCount)}).Error; err != nil {
			return err
		}
	}
	for _, category := range state.categories {
		var count int64
		if err := tx.WithContext(ctx).Model(&domain.Post{}).Where("category_id = ? AND is_draft = ?", category.ID, false).Count(&count).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Category{}).Where("id = ?", category.ID).Update("use_count", int(count)).Error; err != nil {
			return err
		}
	}
	for _, tag := range state.tags {
		var count int64
		if err := tx.WithContext(ctx).Model(&domain.PostTag{}).Where("tag_id = ?", tag.ID).Count(&count).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&domain.Tag{}).Where("id = ?", tag.ID).Update("use_count", int(count)).Error; err != nil {
			return err
		}
	}
	return nil
}
