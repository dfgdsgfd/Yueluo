package handlers

import (
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminPostsQualityList(c *gin.Context) {
	page, limit, offset := pageLimit(c, 20)
	sortField := safeColumn(firstNonEmpty(c.Query("sortField"), "created_at"))
	sortOrder := strings.ToUpper(firstNonEmpty(c.Query("sortOrder"), "desc"))
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}
	build := func() *gorm.DB {
		query := h.DB.WithContext(c.Request.Context()).
			Table("posts p").
			Joins("LEFT JOIN users u ON u.id = p.user_id")
		if title := strings.TrimSpace(c.Query("title")); title != "" {
			query = query.Where("p.title LIKE ?", "%"+title+"%")
		}
		if postType, ok := intFromAny(c.Query("type")); ok {
			query = query.Where("p.type = ?", postType)
		}
		if qualityLevel := strings.TrimSpace(c.Query("quality_level")); qualityLevel != "" {
			query = query.Where("p.quality_level = ?", qualityLevel)
		}
		if draft := strings.TrimSpace(c.Query("is_draft")); draft != "" {
			query = query.Where("p.is_draft = ?", draft == "1" || strings.EqualFold(draft, "true"))
		}
		if displayID := strings.TrimSpace(c.Query("user_display_id")); displayID != "" {
			query = query.Where("u.user_id LIKE ?", "%"+displayID+"%")
		}
		return query
	}
	var total int64
	if err := build().Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []postQualityRow
	err := build().Select(`p.id, p.user_id, p.title, p.content, p.type, p.view_count, p.like_count, p.collect_count, p.comment_count, p.created_at, p.is_draft, p.quality_level, p.quality_marked_at, p.quality_reward,
		u.user_id AS user_display_id, u.nickname AS nickname,
		(SELECT image_url FROM post_images WHERE post_id = p.id ORDER BY id ASC LIMIT 1) AS image_cover,
		(SELECT cover_url FROM post_videos WHERE post_id = p.id ORDER BY id ASC LIMIT 1) AS video_cover`).
		Order("p." + sortField + " " + sortOrder).Offset(offset).Limit(limit).Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.postQualityMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{"data": items, "pagination": gin.H{"page": page, "limit": limit, "total": total, "pages": totalPages(page, limit, total)}})
}

func (h NativeHandlers) adminRecommendationPostBatchCompat(c *gin.Context) {
	body := readBodyMap(c)
	ids := int64SliceFromAny(body["post_ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "post_ids 不能为空", nil)
		return
	}
	if len(ids) > 200 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "单次批量操作最大 200 篇", nil)
		return
	}

	var existing int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id IN ?", ids).Count(&existing).Error; writeDBError(c, err, "") {
		return
	}
	if existing != int64(len(ids)) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "部分帖子不存在", nil)
		return
	}

	boostScore, ok := float64FromAny(body["boost_score"])
	if !ok {
		boostScore = 0
	}
	isPinned := jsBoolean(body["is_pinned"])
	isSuppressed := jsBoolean(body["is_suppressed"])
	isActive := true
	if raw, exists := body["is_active"]; exists {
		isActive = jsBoolean(raw)
	}
	var reason any
	if text := toString(body["reason"]); text != "" {
		reason = text
	} else {
		reason = nil
	}

	count := 0
	for _, postID := range ids {
		row := map[string]any{
			"post_id":       postID,
			"boost_score":   boostScore,
			"is_pinned":     isPinned,
			"is_suppressed": isSuppressed,
			"is_active":     isActive,
			"reason":        reason,
		}
		now := time.Now()
		row["updated_at"] = now
		createRow := maps.Clone(row)
		createRow["created_at"] = now
		err := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "post_id"}},
			DoUpdates: clause.Assignments(row),
		}).Create(createRow).Error
		if err == nil {
			count++
		}
	}
	if count > 0 {
		h.bumpCacheVersions(cacheScopePosts)
	}
	writeSuccess(c, "已批量配置 "+strconv.Itoa(count)+" 篇帖子", gin.H{"count": count})
}

func (h NativeHandlers) adminRecommendationPushCompat(c *gin.Context) {
	body := readBodyMap(c)
	postID, ok := int64FromAny(body["post_id"])
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "帖子ID不能为空", nil)
		return
	}
	if !h.adminPostExists(c, postID) {
		return
	}
	boostScore, ok := float64FromAny(body["boost_score"])
	if !ok {
		boostScore = 10
	}
	reason := firstNonEmpty(toString(body["reason"]), "管理员主动推荐")
	row := map[string]any{
		"post_id":     postID,
		"boost_score": boostScore,
		"is_pinned":   true,
		"is_active":   true,
		"reason":      reason,
	}
	if targetID, ok := int64FromAny(body["target_user_id"]); ok && targetID > 0 {
		row["target_user_id"] = targetID
	} else {
		row["target_user_id"] = nil
	}
	now := time.Now()
	row["updated_at"] = now
	createRow := maps.Clone(row)
	createRow["created_at"] = now
	err := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(row),
	}).Create(createRow).Error
	if writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	updated, ok := h.loadPostRecommendConfigByPost(c, postID)
	if !ok {
		return
	}
	writeSuccess(c, "主动推荐配置已生效", h.postRecommendConfigMap(updated))
}

func (h NativeHandlers) adminPostQualityCompat(c *gin.Context) {
	postID, ok := adminPostQualityID(c)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的笔记ID", nil)
		return
	}
	body := readBodyMap(c)
	qualityLevel, hasQualityLevel := qualityLevelFromBody(body)
	if hasQualityLevel && !qualityLevelValid(qualityLevel) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的质量等级", nil)
		return
	}
	rewardAmount, hasRewardAmount, rewardValid := qualityRewardAmountFromBody(body)
	if !rewardValid {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供有效的原创激励金额", nil)
		return
	}
	if !hasQualityLevel && !hasRewardAmount {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供质量等级或原创激励金额", nil)
		return
	}

	var post domain.Post
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", postID).Take(&post).Error; writeDBError(c, err, "笔记不存在") {
		return
	}

	existingLevel := normalizedQualityLevel(post.QualityLevel)
	finalLevel := existingLevel
	if hasQualityLevel {
		finalLevel = qualityLevel
	}

	if hasQualityLevel && finalLevel != "none" && existingLevel != "none" && existingLevel != finalLevel {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "该笔记已标记过质量等级，每篇笔记仅可标记一次", nil)
		return
	}
	if !hasRewardAmount && hasQualityLevel && finalLevel != "none" {
		rewardAmount = h.qualityRewardAmount(c, finalLevel)
		hasRewardAmount = true
	}

	existingReward := qualityRewardValue(post.QualityReward)
	if hasRewardAmount && rewardAmount < existingReward {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "原创激励已入账，不能调低奖励金额", nil)
		return
	}
	finalReward := existingReward
	rewardToGrant := 0.0
	if hasRewardAmount {
		finalReward = rewardAmount
		rewardToGrant = rewardAmount - existingReward
	}

	updates := map[string]any{}
	if hasQualityLevel {
		updates["quality_level"] = finalLevel
	}
	if hasRewardAmount {
		if rewardAmount > 0 {
			updates["quality_reward"] = rewardAmount
		} else {
			updates["quality_reward"] = nil
		}
	}
	if finalLevel != "none" || finalReward > 0 {
		updates["quality_marked_at"] = time.Now()
	} else {
		updates["quality_marked_at"] = nil
	}

	post.QualityLevel = finalLevel
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Post{}).Where("id = ?", postID).Updates(updates).Error; err != nil {
			return err
		}
		return grantQualityRewardTx(tx, post, rewardToGrant, originalIncentiveRewardLabel)
	})
	if writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)

	message := "原创激励已更新"
	if rewardToGrant > 0 {
		message = "已发放" + strconv.FormatFloat(rewardToGrant, 'f', -1, 64) + "月币原创激励"
	} else if hasQualityLevel && finalLevel == "none" && finalReward <= 0 {
		message = "已清除质量标记"
	} else if hasQualityLevel && !hasRewardAmount {
		message = "质量标记已更新"
	}
	writeSuccess(c, message, gin.H{"quality_level": finalLevel, "reward_amount": finalReward, "original_incentive": finalReward > 0})
}

func (h NativeHandlers) adminPostsQualityBatchCompat(c *gin.Context) {
	body := readBodyMap(c)
	ids := int64SliceFromAny(firstPresent(body, "ids", "post_ids"))
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供要设置的笔记ID列表", nil)
		return
	}
	qualityLevel, hasQualityLevel := qualityLevelFromBody(body)
	if hasQualityLevel && !qualityLevelValid(qualityLevel) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的质量等级", nil)
		return
	}
	rewardAmount, hasRewardAmount, rewardValid := qualityRewardAmountFromBody(body)
	if !rewardValid {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供有效的原创激励金额", nil)
		return
	}
	if !hasQualityLevel && !hasRewardAmount {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供质量等级或原创激励金额", nil)
		return
	}

	if !hasRewardAmount && hasQualityLevel && qualityLevel != "none" {
		rewardAmount = h.qualityRewardAmount(c, qualityLevel)
		hasRewardAmount = true
	}
	successCount := 0
	skippedCount := 0
	totalReward := 0.0
	for _, id := range ids {
		var post domain.Post
		if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", id).Take(&post).Error; err != nil {
			continue
		}
		existingLevel := normalizedQualityLevel(post.QualityLevel)
		finalLevel := existingLevel
		if hasQualityLevel {
			finalLevel = qualityLevel
		}
		if hasQualityLevel && finalLevel != "none" && existingLevel != "none" && existingLevel != finalLevel {
			skippedCount++
			continue
		}
		existingReward := qualityRewardValue(post.QualityReward)
		if hasRewardAmount && rewardAmount < existingReward {
			skippedCount++
			continue
		}
		finalReward := existingReward
		rewardToGrant := 0.0
		if hasRewardAmount {
			finalReward = rewardAmount
			rewardToGrant = rewardAmount - existingReward
		}
		updates := map[string]any{}
		if hasQualityLevel {
			updates["quality_level"] = finalLevel
		}
		if hasRewardAmount {
			if rewardAmount > 0 {
				updates["quality_reward"] = rewardAmount
			} else {
				updates["quality_reward"] = nil
			}
		}
		if finalLevel != "none" || finalReward > 0 {
			updates["quality_marked_at"] = time.Now()
		} else {
			updates["quality_marked_at"] = nil
		}
		post.QualityLevel = finalLevel
		err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&domain.Post{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
			return grantQualityRewardTx(tx, post, rewardToGrant, originalIncentiveRewardLabel)
		})
		if err != nil {
			continue
		}
		if rewardToGrant > 0 {
			totalReward += rewardToGrant
		}
		successCount++
	}
	if successCount > 0 {
		h.bumpCacheVersions(cacheScopePosts)
	}

	message := "成功设置 " + strconv.Itoa(successCount) + " 篇笔记"
	if skippedCount > 0 {
		message += "，跳过 " + strconv.Itoa(skippedCount) + " 篇不可重复发放或已标记笔记"
	}
	if totalReward > 0 {
		message += "，共发放 " + strconv.FormatFloat(totalReward, 'f', 2, 64) + " 月币原创激励"
	}
	writeSuccess(c, message, gin.H{"success_count": successCount, "skipped_count": skippedCount, "total_reward": totalReward})
}
