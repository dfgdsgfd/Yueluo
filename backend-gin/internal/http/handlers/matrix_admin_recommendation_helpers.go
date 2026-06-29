package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func postRecommendDataFromBody(body map[string]any, withCreateDefaults bool) map[string]any {
	row := map[string]any{}
	if v, ok := float64FromAny(body["boost_score"]); ok || withCreateDefaults {
		row["boost_score"] = v
	}
	if raw, exists := body["is_pinned"]; exists || withCreateDefaults {
		row["is_pinned"] = jsBoolean(raw)
	}
	if raw, exists := body["is_suppressed"]; exists || withCreateDefaults {
		row["is_suppressed"] = jsBoolean(raw)
	}
	if raw, exists := body["target_user_id"]; exists {
		if id, ok := int64FromAny(raw); ok && id > 0 {
			row["target_user_id"] = id
		} else {
			row["target_user_id"] = nil
		}
	}
	if raw, exists := body["reason"]; exists || withCreateDefaults {
		if text := toString(raw); text != "" {
			row["reason"] = text
		} else {
			row["reason"] = nil
		}
	}
	if raw, exists := body["is_active"]; exists {
		row["is_active"] = jsBoolean(raw)
	}
	if raw, exists := body["start_time"]; exists {
		row["start_time"] = parseTimeAny(raw)
	}
	if raw, exists := body["end_time"]; exists {
		row["end_time"] = parseTimeAny(raw)
	}
	return row
}

func recommendConfigDataFromBody(body map[string]any) map[string]any {
	row := map[string]any{}
	for _, key := range []string{"like_weight", "collect_weight", "view_weight", "category_weight", "tag_weight", "following_weight", "mutual_follow_weight", "popularity_weight", "interest_weight"} {
		if v, ok := float64FromAny(body[key]); ok {
			row[key] = v
		}
	}
	if v, ok := intFromAny(body["time_decay_half_life"]); ok {
		row["time_decay_half_life"] = v
	}
	if raw, exists := body["is_active"]; exists {
		row["is_active"] = jsBoolean(raw)
	}
	return row
}

func (h NativeHandlers) adminPostExists(c *gin.Context, postID int64) bool {
	var count int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id = ?", postID).Count(&count).Error; writeDBError(c, err, "") {
		return false
	}
	if count == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "帖子不存在", nil)
		return false
	}
	return true
}

func (h NativeHandlers) adminUserExists(c *gin.Context, userID int64) bool {
	var count int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", userID).Count(&count).Error; writeDBError(c, err, "") {
		return false
	}
	if count == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "用户不存在", nil)
		return false
	}
	return true
}

func (h NativeHandlers) loadPostRecommendConfigByPost(c *gin.Context, postID int64) (postRecommendConfigRow, bool) {
	return h.loadPostRecommendConfig(c, "prc.post_id = ?", postID)
}

func (h NativeHandlers) loadPostRecommendConfigByID(c *gin.Context, id string) (postRecommendConfigRow, bool) {
	return h.loadPostRecommendConfig(c, "prc.id = ?", id)
}

func (h NativeHandlers) loadPostRecommendConfig(c *gin.Context, where string, arg any) (postRecommendConfigRow, bool) {
	var row postRecommendConfigRow
	err := h.DB.WithContext(c.Request.Context()).
		Table("post_recommend_configs prc").
		Joins("LEFT JOIN posts p ON p.id = prc.post_id").
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Joins("LEFT JOIN users tu ON tu.id = prc.target_user_id").
		Select(`prc.id, prc.post_id, prc.boost_score, prc.is_pinned, prc.is_suppressed, prc.target_user_id, prc.reason, prc.is_active, prc.start_time, prc.end_time, prc.created_at, prc.updated_at,
			p.id AS post_id_value, p.title AS post_title, p.content AS post_content, p.type AS post_type, p.like_count AS post_like_count, p.collect_count AS post_collect_count, p.view_count AS post_view_count,
			(SELECT image_url FROM post_images pi WHERE pi.post_id = p.id ORDER BY pi.id ASC LIMIT 1) AS post_first_image_url,
			(SELECT cover_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1) AS post_first_video_cover,
			u.id AS post_user_id, u.nickname AS post_user_nickname, u.user_id AS post_user_display_id,
			tu.id AS target_user_id_value, tu.nickname AS target_user_nickname, tu.user_id AS target_user_display_id, tu.avatar AS target_user_avatar`).
		Where(where, arg).Take(&row).Error
	if writeDBError(c, err, "资源不存在") {
		return postRecommendConfigRow{}, false
	}
	return row, true
}

func (h NativeHandlers) loadRecommendConfigByUser(c *gin.Context, userID int64) (recommendConfigRow, bool) {
	return h.loadRecommendConfig(c, "rc.user_id = ?", userID)
}

func (h NativeHandlers) loadRecommendConfigByID(c *gin.Context, id string) (recommendConfigRow, bool) {
	return h.loadRecommendConfig(c, "rc.id = ?", id)
}

func (h NativeHandlers) loadRecommendConfig(c *gin.Context, where string, arg any) (recommendConfigRow, bool) {
	var row recommendConfigRow
	err := h.DB.WithContext(c.Request.Context()).
		Table("recommend_configs rc").
		Joins("LEFT JOIN users u ON u.id = rc.user_id").
		Select(`rc.id, rc.user_id, rc.like_weight, rc.collect_weight, rc.view_weight, rc.category_weight, rc.tag_weight, rc.following_weight, rc.mutual_follow_weight, rc.popularity_weight, rc.interest_weight, rc.time_decay_half_life, rc.is_active, rc.created_at, rc.updated_at,
			u.id AS user_id_value, u.user_id AS user_display_id, u.nickname AS user_nickname, u.avatar AS user_avatar`).
		Where(where, arg).Take(&row).Error
	if writeDBError(c, err, "资源不存在") {
		return recommendConfigRow{}, false
	}
	return row, true
}

func (h NativeHandlers) postRecommendConfigMap(row postRecommendConfigRow) gin.H {
	out := gin.H{
		"id":             row.ID,
		"post_id":        row.PostID,
		"boost_score":    row.BoostScore,
		"is_pinned":      row.IsPinned,
		"is_suppressed":  row.IsSuppressed,
		"target_user_id": row.TargetUserID,
		"reason":         row.Reason,
		"is_active":      row.IsActive,
		"start_time":     row.StartTime,
		"end_time":       row.EndTime,
		"created_at":     row.CreatedAt,
		"updated_at":     row.UpdatedAt,
	}
	if row.PostIDValue != nil {
		out["title"] = row.PostTitle
		out["content"] = row.PostContent
		out["type"] = row.PostType
		out["nickname"] = row.PostUserNickname
		out["user_display_id"] = row.PostUserDisplayID
		out["first_image_url"] = h.signFileURLPtr(row.PostFirstImageURL)
		out["cover_url"] = h.signFileURLPtr(row.PostFirstVideoCover)
		out["post"] = gin.H{
			"id":            *row.PostIDValue,
			"title":         row.PostTitle,
			"type":          row.PostType,
			"like_count":    row.PostLikeCount,
			"collect_count": row.PostCollectCount,
			"view_count":    row.PostViewCount,
			"user": gin.H{
				"id":       row.PostUserID,
				"nickname": row.PostUserNickname,
				"user_id":  row.PostUserDisplayID,
			},
		}
	}
	if row.TargetUserIDValue != nil {
		out["targetUser"] = gin.H{"id": *row.TargetUserIDValue, "nickname": row.TargetUserNickname, "user_id": row.TargetUserDisplayID, "avatar": h.signFileURLPtr(row.TargetUserAvatar)}
	}
	return out
}

func (h NativeHandlers) recommendConfigMap(row recommendConfigRow) gin.H {
	out := gin.H{
		"id":                   row.ID,
		"user_id":              row.UserID,
		"like_weight":          row.LikeWeight,
		"collect_weight":       row.CollectWeight,
		"view_weight":          row.ViewWeight,
		"category_weight":      row.CategoryWeight,
		"tag_weight":           row.TagWeight,
		"following_weight":     row.FollowingWeight,
		"mutual_follow_weight": row.MutualFollowWeight,
		"popularity_weight":    row.PopularityWeight,
		"interest_weight":      row.InterestWeight,
		"time_decay_half_life": row.TimeDecayHalfLife,
		"is_active":            row.IsActive,
		"created_at":           row.CreatedAt,
		"updated_at":           row.UpdatedAt,
	}
	if row.UserIDValue != nil {
		out["user"] = gin.H{"id": *row.UserIDValue, "user_id": row.UserDisplayID, "nickname": row.UserNickname, "avatar": h.signFileURLPtr(row.UserAvatar)}
	}
	return out
}

func (h NativeHandlers) postQualityMap(row postQualityRow) gin.H {
	cover := row.ImageCover
	if row.Type == 2 {
		cover = row.VideoCover
	}
	qualityLevel := row.QualityLevel
	if qualityLevel == "" {
		qualityLevel = "none"
	}
	rewardAmount := qualityRewardValue(row.QualityReward)
	return gin.H{
		"id":                 row.ID,
		"user_id":            row.UserID,
		"title":              row.Title,
		"content":            truncateRunes(row.Content, 100),
		"type":               row.Type,
		"view_count":         row.ViewCount,
		"like_count":         row.LikeCount,
		"collect_count":      row.CollectCount,
		"comment_count":      row.CommentCount,
		"created_at":         row.CreatedAt,
		"is_draft":           row.IsDraft,
		"user_display_id":    row.UserDisplayID,
		"nickname":           row.Nickname,
		"quality_level":      qualityLevel,
		"quality_marked_at":  row.QualityMarkedAt,
		"quality_reward":     row.QualityReward,
		"reward_amount":      rewardAmount,
		"original_incentive": rewardAmount > 0,
		"cover":              h.signFileURLPtr(cover),
	}
}

func jsBoolean(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case float64:
		return typed != 0
	case float32:
		return typed != 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case int32:
		return typed != 0
	case uint:
		return typed != 0
	case uint64:
		return typed != 0
	case uint32:
		return typed != 0
	default:
		return true
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func (h NativeHandlers) qualityRewardAmount(c *gin.Context, level string) float64 {
	defaults := map[string]float64{"low": 1, "medium": 3, "high": 5}
	var setting domain.PostQualityRewardSetting
	err := h.DB.WithContext(c.Request.Context()).Where("quality_level = ?", level).Take(&setting).Error
	if err == nil && setting.IsActive {
		return setting.RewardAmount
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return defaults[level]
	}
	return defaults[level]
}

func qualityLevelValid(level string) bool {
	return level == "none" || level == "low" || level == "medium" || level == "high"
}

func qualityLabel(level string) string {
	switch level {
	case "low":
		return "低质量"
	case "medium":
		return "中质量"
	case "high":
		return "高质量"
	default:
		return "无"
	}
}

const originalIncentiveRewardLabel = "原创激励"

func normalizedQualityLevel(level string) string {
	if level == "" {
		return "none"
	}
	return level
}

func qualityRewardValue(value *float64) float64 {
	if value == nil || *value <= 0 {
		return 0
	}
	return *value
}

func postHasOriginalIncentive(post domain.Post) bool {
	return qualityRewardValue(post.QualityReward) > 0
}

func qualityRewardAmountFromBody(body map[string]any) (float64, bool, bool) {
	for _, key := range []string{"quality_reward", "reward_amount"} {
		raw, exists := body[key]
		if !exists {
			continue
		}
		if toString(raw) == "" {
			return 0, false, true
		}
		amount, ok := float64FromAny(raw)
		return amount, true, ok && amount >= 0
	}
	return 0, false, true
}

func qualityLevelFromBody(body map[string]any) (string, bool) {
	raw, exists := body["quality_level"]
	if !exists {
		return "", false
	}
	return normalizedQualityLevel(toString(raw)), true
}

func adminPostQualityID(c *gin.Context) (int64, bool) {
	if postID, ok := int64FromAny(matrixParam(c, "id")); ok && postID > 0 {
		return postID, true
	}
	segments := adminSegments(c)
	if len(segments) >= 2 && (segments[0] == "posts" || segments[0] == "posts-quality") {
		if postID, ok := int64FromAny(segments[1]); ok && postID > 0 {
			return postID, true
		}
	}
	return 0, false
}

func (h NativeHandlers) grantQualityReward(c *gin.Context, post domain.Post, amount float64, label string) error {
	if amount <= 0 {
		return nil
	}
	return h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		return grantQualityRewardTx(tx, post, amount, label)
	})
}

func grantQualityRewardTx(tx *gorm.DB, post domain.Post, amount float64, label string) error {
	if amount <= 0 {
		return nil
	}
	if label == "" {
		label = originalIncentiveRewardLabel
	}
	var earnings domain.CreatorEarnings
	err := tx.Where("user_id = ?", post.UserID).Take(&earnings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		earnings = domain.CreatorEarnings{UserID: post.UserID, Balance: 0, TotalEarnings: 0, WithdrawnAmount: 0}
		if err := tx.Create(&earnings).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	newBalance := earnings.Balance + amount
	newTotal := earnings.TotalEarnings + amount
	if err := tx.Model(&domain.CreatorEarnings{}).Where("id = ?", earnings.ID).Updates(map[string]any{"balance": newBalance, "total_earnings": newTotal}).Error; err != nil {
		return err
	}
	sourceType := "post"
	reason := "笔记质量奖励: " + label
	sourceID := post.ID
	return tx.Create(&domain.CreatorEarningsLog{
		UserID:       post.UserID,
		EarningsID:   earnings.ID,
		Amount:       amount,
		BalanceAfter: newBalance,
		Type:         "quality_reward",
		SourceID:     &sourceID,
		SourceType:   &sourceType,
		Reason:       &reason,
		PlatformFee:  0,
	}).Error
}
