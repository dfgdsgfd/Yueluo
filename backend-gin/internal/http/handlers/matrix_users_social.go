package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/security"
)

func (h NativeHandlers) usersPersonality(c *gin.Context, id string, currentUserID int64) {
	user, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	writeSuccess(c, matrixMsgOK, personalityMap(user, currentUserID == user.ID))
}

func (h NativeHandlers) usersFollowStatus(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	data, err := h.followStatusData(c, currentUserID, target.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) usersFollow(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	if currentUserID == target.ID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "不能关注自己", nil)
		return
	}
	blocked, blockedBy, err := h.blockStatus(c, currentUserID, target.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if blocked {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "你已拉黑对方，无法关注", nil)
		return
	}
	if blockedBy {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "操作失败", nil)
		return
	}
	err = h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&domain.Follow{}).Where("follower_id = ? AND following_id = ?", currentUserID, target.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return repositories.ErrContentAlreadyExists
		}
		if err := tx.Create(&domain.Follow{FollowerID: currentUserID, FollowingID: target.ID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ?", currentUserID).UpdateColumn("follow_count", gorm.Expr("follow_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&domain.User{}).Where("id = ?", target.ID).UpdateColumn("fans_count", gorm.Expr("fans_count + 1")).Error
	})
	if errors.Is(err, repositories.ErrContentAlreadyExists) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "已经关注了该用户", nil)
		return
	}
	if writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeUsers, cacheScopeInteractions, cacheScopeNotifications)
	writeSimpleSuccess(c, "关注成功")
}

func (h NativeHandlers) usersUnfollow(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var follow domain.Follow
		if err := tx.Where("follower_id = ? AND following_id = ?", currentUserID, target.ID).First(&follow).Error; err != nil {
			return err
		}
		if err := tx.Delete(&follow).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ? AND follow_count > 0", currentUserID).UpdateColumn("follow_count", gorm.Expr("follow_count - 1")).Error; err != nil {
			return err
		}
		return tx.Model(&domain.User{}).Where("id = ? AND fans_count > 0", target.ID).UpdateColumn("fans_count", gorm.Expr("fans_count - 1")).Error
	})
	if writeDBError(c, err, "关注记录不存在") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeUsers, cacheScopeInteractions, cacheScopeNotifications)
	writeSimpleSuccess(c, "取消关注成功")
}

func (h NativeHandlers) usersFollowList(c *gin.Context, id string, following bool) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	page, limit, offset := pageLimit(c, 20)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.Follow{})
	field := "follower_id"
	selectField := "following_id"
	dataKey := "following"
	if !following {
		field = "following_id"
		selectField = "follower_id"
		dataKey = "followers"
	}
	query = query.Where(field+" = ?", target.ID)
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.Follow
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
		return
	}
	ids := make([]int64, 0, len(rows))
	seenAt := map[int64]time.Time{}
	for _, row := range rows {
		id := row.FollowingID
		if selectField == "follower_id" {
			id = row.FollowerID
		}
		ids = append(ids, id)
		seenAt[id] = row.CreatedAt
	}
	users, err := h.usersByIDs(c, ids)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	items, err := h.usersDecorated(c, users, currentUserID(c))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	for _, item := range items {
		if id, ok := int64FromAny(item["id"]); ok {
			item["followed_at"] = seenAt[id]
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{dataKey: items, "pagination": matrixPagination(page, limit, total)})
}

func (h NativeHandlers) usersMutualFollows(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	page, limit, offset := pageLimit(c, 20)
	var rows []domain.User
	err := h.DB.WithContext(c.Request.Context()).Raw(`
		SELECT u.* FROM users u
		JOIN follows f1 ON f1.following_id = u.id AND f1.follower_id = ?
		JOIN follows f2 ON f2.follower_id = u.id AND f2.following_id = ?
		ORDER BY u.created_at DESC LIMIT ? OFFSET ?`, target.ID, target.ID, limit, offset).Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	var total int64
	err = h.DB.WithContext(c.Request.Context()).Raw(`
		SELECT COUNT(*) FROM follows f1
		JOIN follows f2 ON f2.follower_id = f1.following_id AND f2.following_id = f1.follower_id
		WHERE f1.follower_id = ?`, target.ID).Scan(&total).Error
	if writeDBError(c, err, "") {
		return
	}
	items, err := h.usersDecorated(c, rows, currentUserID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"mutualFollows": items, "pagination": matrixPagination(page, limit, total)})
}

func (h NativeHandlers) usersBlock(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	if currentUserID == target.ID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "不能拉黑自己", nil)
		return
	}
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&domain.Blacklist{}).Where("blocker_id = ? AND blocked_id = ?", currentUserID, target.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return repositories.ErrContentAlreadyExists
		}
		if err := tx.Create(&domain.Blacklist{BlockerID: currentUserID, BlockedID: target.ID}).Error; err != nil {
			return err
		}
		return h.removeMutualFollows(tx, currentUserID, target.ID)
	})
	if errors.Is(err, repositories.ErrContentAlreadyExists) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "已经拉黑了该用户", nil)
		return
	}
	if writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeUsers, cacheScopeInteractions, cacheScopeNotifications)
	writeSimpleSuccess(c, "拉黑成功")
}

func (h NativeHandlers) usersUnblock(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Where("blocker_id = ? AND blocked_id = ?", currentUserID, target.ID).Delete(&domain.Blacklist{})
	if writeDBError(c, res.Error, "") {
		return
	}
	if res.RowsAffected == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "黑名单记录不存在", nil)
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeUsers, cacheScopeInteractions, cacheScopeNotifications)
	writeSimpleSuccess(c, "取消拉黑成功")
}

func (h NativeHandlers) usersBlockStatus(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	blocked, blockedBy, err := h.blockStatus(c, currentUserID, target.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"isBlocked": blocked, "isBlockedBy": blockedBy})
}

func (h NativeHandlers) usersPosts(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	blocked, blockedBy, err := h.blockStatus(c, currentUserID, target.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if currentUserID != target.ID && (blocked || blockedBy) {
		key := "isBlocked"
		msg := "blocked"
		if blockedBy {
			key = "isBlockedBy"
			msg = "blocked_by"
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msg, "data": gin.H{"posts": []gin.H{}, "pagination": matrixPagination(1, positiveIntQuery(c, "limit", 20), 0), key: true}})
		return
	}
	opts := repositories.PostListOptions{UserID: &target.ID, Sort: c.DefaultQuery("sort", "created_at")}
	if currentUserID == target.ID {
		if visibility := c.Query("visibility"); visibility != "" {
			opts.UserID = &target.ID
		}
	}
	h.writePostList(c, opts)
}

func (h NativeHandlers) usersCollections(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	page, limit, offset := pageLimit(c, 20)
	load := func() (gin.H, error) {
		return h.userCollectionsPayload(c, target.ID, currentUserID, page, limit, offset)
	}
	if h.Redis != nil {
		cacheKey := h.cacheKeyWithVersions(cacheScopeUsers, []string{cacheScopePosts, cacheScopeInteractions}, currentUserID, "collections", target.ID, page, limit)
		var cached gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &cached, cacheTTL(20), func() (any, error) {
			return load()
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		writeSuccess(c, matrixMsgOK, cached)
		return
	}
	data, err := load()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) userCollectionsPayload(c *gin.Context, targetID int64, currentUserID int64, page int, limit int, offset int) (gin.H, error) {
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.Collection{}).Where("user_id = ?", targetID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.Collection
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	posts, err := h.postsForIDs(c, collectionPostIDs(rows), currentUserID)
	if err != nil {
		return nil, err
	}
	collectedAt := map[int64]time.Time{}
	for _, row := range rows {
		collectedAt[row.PostID] = row.CreatedAt
	}
	for _, post := range posts {
		if id, ok := int64FromAny(post["id"]); ok {
			post["collected_at"] = collectedAt[id]
		}
		post["collected"] = true
	}
	return gin.H{"collections": posts, "pagination": matrixPagination(page, limit, total)}, nil
}

func (h NativeHandlers) usersLikes(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	page, limit, offset := pageLimit(c, 20)
	load := func() (gin.H, error) {
		return h.userLikesPayload(c, target.ID, currentUserID, page, limit, offset)
	}
	if h.Redis != nil {
		cacheKey := h.cacheKeyWithVersions(cacheScopeUsers, []string{cacheScopePosts, cacheScopeInteractions}, currentUserID, "likes", target.ID, page, limit)
		var cached gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &cached, cacheTTL(20), func() (any, error) {
			return load()
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		writeSuccess(c, matrixMsgOK, cached)
		return
	}
	data, err := load()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) userLikesPayload(c *gin.Context, targetID int64, currentUserID int64, page int, limit int, offset int) (gin.H, error) {
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.Like{}).Where("user_id = ? AND target_type = ?", targetID, 1)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.Like
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	posts, err := h.postsForIDs(c, likePostIDs(rows), currentUserID)
	if err != nil {
		return nil, err
	}
	likedAt := map[int64]time.Time{}
	for _, row := range rows {
		likedAt[row.TargetID] = row.CreatedAt
	}
	for _, post := range posts {
		if id, ok := int64FromAny(post["id"]); ok {
			post["liked_at"] = likedAt[id]
		}
		post["liked"] = true
	}
	return gin.H{"posts": posts, "pagination": matrixPagination(page, limit, total)}, nil
}

func (h NativeHandlers) usersStats(c *gin.Context, id string) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	load := func() (gin.H, error) {
		return h.userStatsPayload(c, target)
	}
	if h.Redis != nil {
		cacheKey := h.cacheKeyWithVersions(cacheScopeUsers, []string{cacheScopePosts, cacheScopeInteractions}, currentUserID(c), "stats", target.ID)
		var cached gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &cached, cacheTTL(30), func() (any, error) {
			return load()
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		writeSuccess(c, matrixMsgOK, cached)
		return
	}
	data, err := load()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) userStatsPayload(c *gin.Context, target domain.User) (gin.H, error) {
	var postCount int64
	var collectCount int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("user_id = ? AND is_draft = ?", target.ID, false).Count(&postCount).Error; err != nil {
		return nil, err
	}
	err := h.DB.WithContext(c.Request.Context()).Table("collections").
		Joins("JOIN posts ON posts.id = collections.post_id").
		Where("posts.user_id = ? AND posts.is_draft = ?", target.ID, false).
		Count(&collectCount).Error
	if err != nil {
		return nil, err
	}
	return gin.H{
		"follow_count":       target.FollowCount,
		"fans_count":         target.FansCount,
		"post_count":         postCount,
		"like_count":         target.LikeCount,
		"collect_count":      collectCount,
		"likes_and_collects": int64(target.LikeCount) + collectCount,
	}, nil
}

func (h NativeHandlers) usersPassword(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	if currentUserID != target.ID {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "只能修改自己的密码", nil)
		return
	}
	body := readBodyMap(c)
	newPassword := toString(body["newPassword"])
	if newPassword == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "新密码不能为空", nil)
		return
	}
	if len(newPassword) < 6 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "新密码长度不能少于6位", nil)
		return
	}
	if target.Password != nil {
		currentPassword := toString(body["currentPassword"])
		if currentPassword == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "当前密码不能为空", nil)
			return
		}
		if !security.VerifyPassword(currentPassword, *target.Password) {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "当前密码错误", nil)
			return
		}
	}
	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", target.ID).Update("password", passwordHash).Error; writeDBError(c, err, "") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "密码修改成功", "success": true})
}

func (h NativeHandlers) usersDetail(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	blocked, blockedBy, err := h.blockStatus(c, currentUserID, target.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	data := h.userPublicMap(target)
	isOwner := currentUserID == target.ID
	if !isOwner {
		switch target.BioAuditStatus {
		case auditStatusPending:
			data["bio"] = "正在等待审核"
		case 2:
			data["bio"] = "内容审核失败"
		}
		delete(data, "bio_audit_status")
		if !target.PrivacyBirthday {
			data["birthday"] = nil
		}
		if !target.PrivacyZodiac {
			data["zodiac_sign"] = nil
		}
		if !target.PrivacyMBTI {
			data["mbti"] = nil
		}
	}
	data["isBlocked"] = blocked
	data["isBlockedBy"] = blockedBy
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) usersUpdate(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	if currentUserID != target.ID {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "只能修改自己的资料", nil)
		return
	}
	body := readBodyMap(c)
	updates := map[string]any{}
	awardTasks := []profileAwardTask{}
	if value := sanitizePlainSubmittedText(toString(body["nickname"])); value != "" {
		updates["nickname"] = value
		if value != target.Nickname {
			awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetName, reason: "设置名称奖励"})
		}
	}
	for key, column := range map[string]string{
		"avatar":      "avatar",
		"background":  "background",
		"bio":         "bio",
		"location":    "location",
		"gender":      "gender",
		"zodiac_sign": "zodiac_sign",
		"mbti":        "mbti",
		"education":   "education",
		"major":       "major",
	} {
		if value, exists := body[key]; exists {
			text := toString(value)
			if key == "avatar" || key == "background" {
				var hashValue any
				text, hashValue = h.profileImageStorageValue(text, column)
				updates[profileImageHashColumnForStorageColumn(column)] = hashValue
			} else if key == "bio" {
				text = sanitizeMarkdownSubmittedText(text)
			} else {
				text = sanitizePlainSubmittedText(text)
			}
			updates[column] = nilIfEmpty(text)
			if text != "" {
				switch key {
				case "avatar":
					if text != stringPtrValue(target.Avatar) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetAvatar, reason: "设置头像奖励"})
					}
				case "background":
					if text != stringPtrValue(target.Background) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetBackground, reason: "设置背景奖励"})
					}
				case "bio":
					if text != stringPtrValue(target.Bio) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetSignature, reason: "设置签名奖励"})
					}
				}
			}
		}
	}
	if value, exists := body["birthday"]; exists {
		if birthday := parseTimeAny(value); birthday != nil {
			updates["birthday"] = *birthday
		} else {
			updates["birthday"] = nil
		}
	}
	if value, exists := body["interests"]; exists {
		updates["interests"] = jsonBytes(sanitizedStringSlice(value, 20, 50))
	}
	if value, exists := body["custom_fields"]; exists {
		updates["custom_fields"] = jsonBytes(sanitizedStringMap(value, 20, 50))
	}
	if _, ok := updates["bio"]; ok {
		updates["bio_audit_status"] = auditStatusOK
	}
	if len(updates) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有需要更新的数据", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", target.ID).Updates(updates).Error; writeDBError(c, err, "") {
		return
	}
	var updated domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", target.ID).First(&updated).Error; writeDBError(c, err, "") {
		return
	}
	data := h.userPublicMap(updated)
	if awards := h.awardProfileTasksBestEffort(c, target.ID, awardTasks); len(awards) > 0 {
		data["points_awards"] = awards
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "资料更新成功", "success": true, "data": data})
}
