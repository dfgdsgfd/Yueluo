package repositories

import (
	"context"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

func deleteCommentRecursive(ctx context.Context, tx *gorm.DB, commentID int64) (int64, error) {
	var children []domain.Comment
	if err := tx.WithContext(ctx).Where("parent_id = ?", commentID).Select("id").Find(&children).Error; err != nil {
		return 0, err
	}
	deleted := int64(0)
	for _, child := range children {
		count, err := deleteCommentRecursive(ctx, tx, child.ID)
		if err != nil {
			return 0, err
		}
		deleted += count
	}
	if err := tx.WithContext(ctx).Where("target_type = ? AND target_id = ?", 2, commentID).Delete(&domain.Like{}).Error; err != nil {
		return 0, err
	}
	if err := tx.WithContext(ctx).Where("id = ?", commentID).Delete(&domain.Comment{}).Error; err != nil {
		return 0, err
	}
	return deleted + 1, nil
}

func countVisibleCommentRecursive(ctx context.Context, tx *gorm.DB, commentID int64) (int64, error) {
	var comment domain.Comment
	if err := tx.WithContext(ctx).Where("id = ?", commentID).Select("id", "is_public").First(&comment).Error; err != nil {
		return 0, err
	}
	count := int64(0)
	if comment.IsPublic {
		count = 1
	}
	var children []domain.Comment
	if err := tx.WithContext(ctx).Where("parent_id = ?", commentID).Select("id").Find(&children).Error; err != nil {
		return 0, err
	}
	for _, child := range children {
		childCount, err := countVisibleCommentRecursive(ctx, tx, child.ID)
		if err != nil {
			return 0, err
		}
		count += childCount
	}
	return count, nil
}

func createPostChildren(ctx context.Context, tx *gorm.DB, postID, userID int64, images []PostImageInput, video *PostVideoInput, attachment *PostAttachmentInput, payment *PaymentSettingsInput) error {
	images = NormalizePostImageInputs(images)
	if err := createImages(ctx, tx, postID, userID, images); err != nil {
		return err
	}
	if video != nil && video.URL != "" {
		if err := tx.WithContext(ctx).Create(&domain.PostVideo{PostID: postID, VideoURL: video.URL, CoverURL: video.CoverURL}).Error; err != nil {
			return err
		}
	}
	if attachment != nil && attachment.URL != "" {
		if err := createAttachment(ctx, tx, postID, *attachment); err != nil {
			return err
		}
	}
	if payment != nil && payment.Enabled {
		if err := createPayment(ctx, tx, postID, *payment); err != nil {
			return err
		}
	}
	return bindUploadAssetsToPost(ctx, tx, postID, userID, images, video, attachment)
}

func createImages(ctx context.Context, tx *gorm.DB, postID, userID int64, images []PostImageInput) error {
	images = NormalizePostImageInputs(images)
	if len(images) == 0 {
		return nil
	}
	rows := make([]domain.PostImage, 0, len(images))
	for idx, image := range images {
		sortOrder := image.SortOrder
		if sortOrder <= 0 {
			sortOrder = idx + 1
		}
		rows = append(rows, domain.PostImage{PostID: postID, ImageURL: image.URL, WatermarkTraceToken: image.WatermarkTraceToken, IsFreePreview: image.IsFreePreview, IsProtected: image.IsProtected, SortOrder: sortOrder})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		return err
	}
	return BindPostImageWatermarkTraces(ctx, tx, postID, userID, rows)
}

func createAttachment(ctx context.Context, tx *gorm.DB, postID int64, input PostAttachmentInput) error {
	filename := input.Filename
	if filename == "" {
		filename = "attachment"
	}
	return tx.WithContext(ctx).Create(&domain.PostAttachment{PostID: postID, AttachmentURL: input.URL, Filename: filename, Filesize: input.Filesize}).Error
}

func bindUploadAssetsToPost(ctx context.Context, tx *gorm.DB, postID, userID int64, images []PostImageInput, video *PostVideoInput, attachment *PostAttachmentInput) error {
	if tx == nil || postID <= 0 || userID <= 0 {
		return nil
	}
	urls := make([]string, 0, len(images)+3)
	for _, image := range images {
		if url := strings.TrimSpace(image.URL); url != "" {
			urls = append(urls, url)
		}
	}
	if video != nil {
		if url := strings.TrimSpace(video.URL); url != "" {
			urls = append(urls, url)
		}
		if video.CoverURL != nil {
			if url := strings.TrimSpace(*video.CoverURL); url != "" {
				urls = append(urls, url)
			}
		}
	}
	if attachment != nil {
		if url := strings.TrimSpace(attachment.URL); url != "" {
			urls = append(urls, url)
		}
	}
	urls = uniqueStrings(urls)
	if len(urls) == 0 {
		return nil
	}
	now := time.Now()
	return tx.WithContext(ctx).Model(&domain.UploadAsset{}).
		Where("user_id = ? AND url IN ?", userID, urls).
		Updates(map[string]any{
			"status":        "bound",
			"bound_post_id": postID,
			"expires_at":    nil,
			"last_used_at":  &now,
			"cleanup_error": "",
			"updated_at":    &now,
		}).Error
}

func createPayment(ctx context.Context, tx *gorm.DB, postID int64, input PaymentSettingsInput) error {
	paymentType := strings.TrimSpace(input.PaymentType)
	if paymentType == "" {
		paymentType = "single"
	}
	paymentMethod := normalizePaymentMethod(input.PaymentMethod)
	return tx.WithContext(ctx).Create(&domain.PostPaymentSetting{
		PostID:           postID,
		Enabled:          true,
		PaymentType:      paymentType,
		PaymentMethod:    paymentMethod,
		Price:            input.Price,
		FreePreviewCount: input.FreePreviewCount,
		PreviewDuration:  input.PreviewDuration,
		HideAll:          input.HideAll,
	}).Error
}

func normalizePaymentMethod(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "points":
		return "points"
	default:
		return "balance"
	}
}

func replacePostTags(ctx context.Context, tx *gorm.DB, postID int64, oldPostID *int64, tagNames []string) error {
	if oldPostID != nil {
		var oldTags []domain.PostTag
		if err := tx.WithContext(ctx).Where("post_id = ?", *oldPostID).Select("tag_id").Find(&oldTags).Error; err != nil {
			return err
		}
		oldIDs := make([]int, 0, len(oldTags))
		for _, tag := range oldTags {
			oldIDs = append(oldIDs, tag.TagID)
		}
		if len(oldIDs) > 0 {
			if err := tx.WithContext(ctx).Model(&domain.Tag{}).Where("id IN ?", uniqueInt(oldIDs)).UpdateColumn("use_count", gorm.Expr("use_count - ?", 1)).Error; err != nil {
				return err
			}
		}
		if err := tx.WithContext(ctx).Where("post_id = ?", *oldPostID).Delete(&domain.PostTag{}).Error; err != nil {
			return err
		}
	}
	tagNames = uniqueStrings(tagNames)
	if len(tagNames) == 0 {
		return nil
	}
	var existing []domain.Tag
	if err := tx.WithContext(ctx).Where("name IN ?", tagNames).Select("id", "name").Find(&existing).Error; err != nil {
		return err
	}
	nameToID := map[string]int{}
	for _, tag := range existing {
		nameToID[tag.Name] = tag.ID
	}
	for _, name := range tagNames {
		if _, ok := nameToID[name]; ok {
			continue
		}
		tag := domain.Tag{Name: name}
		if err := tx.WithContext(ctx).Create(&tag).Error; err != nil {
			return err
		}
		nameToID[name] = tag.ID
	}
	postTags := make([]domain.PostTag, 0, len(tagNames))
	tagIDs := make([]int, 0, len(tagNames))
	for _, name := range tagNames {
		tagID := nameToID[name]
		tagIDs = append(tagIDs, tagID)
		postTags = append(postTags, domain.PostTag{PostID: postID, TagID: tagID})
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&postTags).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&domain.Tag{}).Where("id IN ?", uniqueInt(tagIDs)).UpdateColumn("use_count", gorm.Expr("use_count + ?", 1)).Error
}

func createFollowerNewPostNotifications(ctx context.Context, tx *gorm.DB, userID, postID int64, title string) error {
	var followers []domain.Follow
	if err := tx.WithContext(ctx).Where("following_id = ?", userID).Select("follower_id").Find(&followers).Error; err != nil {
		return err
	}
	if len(followers) == 0 {
		return nil
	}
	notifications := make([]domain.Notification, 0, len(followers))
	targetID := postID
	noticeTitle := "发布了新笔记：" + title
	if len([]rune(noticeTitle)) > 200 {
		noticeTitle = string([]rune(noticeTitle)[:200])
	}
	for _, follower := range followers {
		notifications = append(notifications, domain.Notification{UserID: follower.FollowerID, SenderID: userID, Type: 11, Title: noticeTitle, TargetID: &targetID})
	}
	return tx.WithContext(ctx).Create(&notifications).Error
}

func normalizeVisibility(value string) string {
	switch value {
	case VisibilityPrivate, VisibilityFriendsOnly:
		return value
	default:
		return VisibilityPublic
	}
}

func sortClause(sortValue string) string {
	if sortValue == "hot" {
		return "like_count DESC, view_count DESC, created_at DESC"
	}
	return "created_at DESC"
}

func popularityScore(post domain.Post) float64 {
	score := float64(post.LikeCount) + float64(post.CollectCount)*1.5 + float64(post.CommentCount)*2 + math.Log10(float64(post.ViewCount)+1)*0.5
	if post.QualityReward != nil && *post.QualityReward > 0 {
		score += 10 * (1 + math.Log10(*post.QualityReward+1))
	}
	return score
}

func recommendationScore(post domain.Post) float64 {
	return popularityScore(post)*timeDecay(post.CreatedAt, 7) + newPostBoost(post.CreatedAt)
}

func timeDecay(createdAt time.Time, halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		return 1
	}
	ageDays := math.Max(0, time.Since(createdAt).Hours()/24)
	return math.Exp(-(math.Log(2) / halfLifeDays) * ageDays)
}

func newPostBoost(createdAt time.Time) float64 {
	return 1000 * timeDecay(createdAt, 0.5)
}

func userIDs(users []domain.User) []int64 {
	out := make([]int64, 0, len(users))
	for _, user := range users {
		out = append(out, user.ID)
	}
	return out
}

func uniqueInt(values []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
