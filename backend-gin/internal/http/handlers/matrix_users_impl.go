package handlers

import (
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	auditTypePersonal     = 1
	auditTypeBusiness     = 2
	auditStatusPending    = 0
	auditStatusOK         = 1
	verificationMaxImages = 9
)

func (h NativeHandlers) usersSearch(c *gin.Context, currentUserID int64) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请输入搜索关键词", nil)
		return
	}
	page, limit, offset := pageLimit(c, 20)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).
		Where("nickname LIKE ? OR user_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var users []domain.User
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; writeDBError(c, err, "") {
		return
	}
	items, err := h.usersDecorated(c, users, currentUserID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"users":      items,
		"keyword":    keyword,
		"pagination": matrixPagination(page, limit, total),
	})
}

func (h NativeHandlers) usersHistoryAdd(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	postID, ok := int64FromAny(firstPresent(body, "post_id", "postId", "id"))
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "笔记ID不能为空", nil)
		return
	}
	var post domain.Post
	err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND is_draft = ?", postID, false).Select("id").First(&post).Error
	if writeDBError(c, err, "笔记不存在") {
		return
	}
	now := time.Now()
	row := domain.BrowsingHistory{UserID: userID, PostID: postID, CreatedAt: now, UpdatedAt: &now}
	err = h.DB.WithContext(c.Request.Context()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{"updated_at": now}),
	}).Create(&row).Error
	if writeDBError(c, err, "") {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":         response.CodeSuccess,
		"message":      "浏览记录已保存",
		"success":      true,
		"points_award": h.awardPointsBestEffort(c, userID, repositories.PointsTaskView, postID, "浏览奖励"),
	})
}

func (h NativeHandlers) usersHistoryList(c *gin.Context, userID int64) {
	page, limit, offset := pageLimit(c, 20)
	cutoff := time.Now().Add(-48 * time.Hour)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.BrowsingHistory{}).
		Where("user_id = ? AND updated_at >= ?", userID, cutoff)
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.BrowsingHistory
	if err := query.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
		return
	}
	posts, err := h.postsForIDs(c, historyPostIDs(rows), userID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	viewedAt := map[int64]time.Time{}
	for _, row := range rows {
		if row.UpdatedAt != nil {
			viewedAt[row.PostID] = *row.UpdatedAt
		} else {
			viewedAt[row.PostID] = row.CreatedAt
		}
	}
	for _, post := range posts {
		if id, ok := int64FromAny(post["id"]); ok {
			post["viewed_at"] = viewedAt[id]
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{"posts": posts, "pagination": matrixPagination(page, limit, total)})
}

func (h NativeHandlers) usersHistoryClear(c *gin.Context, userID int64) {
	if err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", userID).Delete(&domain.BrowsingHistory{}).Error; writeDBError(c, err, "") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "浏览历史已清空", "success": true})
}

func (h NativeHandlers) usersHistoryDelete(c *gin.Context, userID int64) {
	postID, ok := int64FromAny(matrixParam(c, "postId"))
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "笔记ID不能为空", nil)
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Where("user_id = ? AND post_id = ?", userID, postID).Delete(&domain.BrowsingHistory{})
	if writeDBError(c, res.Error, "") {
		return
	}
	if res.RowsAffected == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "浏览记录不存在", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "浏览记录已删除", "success": true})
}

func (h NativeHandlers) usersPrivacyGet(c *gin.Context, userID int64) {
	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).
		Select("privacy_birthday", "privacy_age", "privacy_zodiac", "privacy_mbti", "privacy_custom_fields", "ai_auto_comment_enabled").
		Where("id = ?", userID).First(&user).Error
	if writeDBError(c, err, "用户不存在") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": matrixMsgOK, "success": true, "data": privacyMap(user)})
}

func (h NativeHandlers) usersPrivacyUpdate(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	updates := map[string]any{}
	for key, column := range map[string]string{
		"privacy_birthday": "privacy_birthday",
		"privacy_age":      "privacy_age",
		"privacy_zodiac":   "privacy_zodiac",
		"privacy_mbti":     "privacy_mbti",
	} {
		if value, exists := body[key]; exists {
			updates[column], _ = boolFromAny(value)
		}
	}
	if value, exists := body["ai_auto_comment_enabled"]; exists {
		updates["ai_auto_comment_enabled"], _ = boolFromAny(value)
	}
	if value, exists := body["privacy_custom_fields"]; exists {
		updates["privacy_custom_fields"] = jsonBytes(value)
	}
	if len(updates) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有需要更新的设置", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error; writeDBError(c, err, "") {
		return
	}
	h.usersPrivacyGet(c, userID)
}

func (h NativeHandlers) usersToolbar(c *gin.Context) {
	var rows []domain.UserToolbar
	if err := h.DB.WithContext(c.Request.Context()).Where("is_active = ?", true).Order("sort_order ASC").Find(&rows).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"id": row.ID, "name": row.Name, "icon": row.Icon, "url": row.URL, "sort_order": row.SortOrder})
	}
	writeSuccess(c, matrixMsgOK, out)
}

func (h NativeHandlers) usersVerificationStatus(c *gin.Context, userID int64) {
	var rows []domain.Audit
	err := h.DB.WithContext(c.Request.Context()).
		Where("user_id = ? AND type IN ?", userID, []int{auditTypePersonal, auditTypeBusiness}).
		Order("created_at DESC").Find(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.auditMap(row))
	}
	writeSuccess(c, matrixMsgOK, out)
}

func (h NativeHandlers) usersVerificationCreate(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	auditType, ok := intFromAny(body["type"])
	content := sanitizeMarkdownSubmittedText(toString(body["content"]))
	if !ok || content == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必填字段", nil)
		return
	}
	if auditType != auditTypePersonal && auditType != auditTypeBusiness {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的认证类型", nil)
		return
	}
	var count int64
	err := h.DB.WithContext(c.Request.Context()).Model(&domain.Audit{}).
		Where("user_id = ? AND type IN ? AND status IN ?", userID, []int{auditTypePersonal, auditTypeBusiness}, []int{auditStatusPending, auditStatusOK}).
		Count(&count).Error
	if writeDBError(c, err, "") {
		return
	}
	if count > 0 {
		response.JSON(c, http.StatusConflict, response.CodeConflict, "您已有待审核或已通过的认证申请", nil)
		return
	}
	status := auditStatusPending
	imageURLs := verificationImageURLsFromBody(body)
	row := domain.Audit{
		UserID:  userID,
		Type:    auditType,
		Content: content,
		Status:  &status,
		AuditResult: jsonBytes(gin.H{
			"verifiedName": sanitizePlainSubmittedText(toString(body["verifiedName"])),
			"imageUrls":    imageURLs,
			"images":       imageURLs,
		}),
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, "认证申请提交成功", gin.H{"id": int(row.ID)})
}

func (h NativeHandlers) usersVerificationRevoke(c *gin.Context, userID int64) {
	response.JSON(c, http.StatusForbidden, response.CodeForbidden, "认证申请不支持撤回", nil)
}

func verificationImageURLsFromBody(body map[string]any) []string {
	for _, key := range []string{"imageUrls", "image_urls", "images"} {
		if urls := verificationImageURLsFromAny(body[key]); len(urls) > 0 {
			return urls
		}
	}
	return []string{}
}

func verificationImageURLsFromAny(value any) []string {
	candidates := parseStringSlice(value)
	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		for _, item := range strings.FieldsFunc(candidate, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' || r == '，' }) {
			text := strings.TrimSpace(item)
			if text == "" || len([]rune(text)) > 2048 || seen[text] {
				continue
			}
			seen[text] = true
			out = append(out, text)
			if len(out) >= verificationMaxImages {
				return out
			}
		}
	}
	return out
}

func (h NativeHandlers) signVerificationAuditResult(value any) any {
	record, ok := value.(map[string]any)
	if !ok {
		return value
	}
	out := gin.H{}
	maps.Copy(out, record)
	for _, key := range []string{"imageUrls", "image_urls", "images"} {
		if _, exists := out[key]; exists {
			out[key] = h.signVerificationImageURLs(out[key])
		}
	}
	return out
}

func (h NativeHandlers) signVerificationImageURLs(value any) []string {
	urls := verificationImageURLsFromAny(value)
	for index, url := range urls {
		urls[index] = h.signFileURL(url)
	}
	return urls
}

func (h NativeHandlers) usersAPIKeys(c *gin.Context, userID int64) {
	var rows []domain.UserAPIKey
	err := h.DB.WithContext(c.Request.Context()).
		Where("user_id = ?", userID).
		Select("id", "name", "api_key_prefix", "is_active", "last_used_at", "created_at").
		Order("created_at DESC").Find(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"id": int(row.ID), "name": row.Name, "api_key_prefix": row.APIKeyPrefix, "is_active": row.IsActive, "last_used_at": row.LastUsedAt, "created_at": row.CreatedAt})
	}
	writeSuccess(c, "获取成功", out)
}

func (h NativeHandlers) usersAPIKeyCreate(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	name := sanitizePlainSubmittedText(toString(body["name"]))
	if name == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请输入密钥名称", nil)
		return
	}
	if len([]rune(name)) > 50 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "密钥名称不能超过50个字符", nil)
		return
	}
	var count int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.UserAPIKey{}).Where("user_id = ?", userID).Count(&count).Error; writeDBError(c, err, "") {
		return
	}
	if count >= 5 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "最多只能创建5个API密钥", nil)
		return
	}
	rawKey := "xise_" + randomHex(32)
	row := domain.UserAPIKey{UserID: userID, Name: name, APIKey: sha256Hex(rawKey), APIKeyPrefix: rawKey[:10], IsActive: true}
	if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; writeDBError(c, err, "") {
		return
	}
	writeCreated(c, "API密钥创建成功，请妥善保存，密钥仅显示一次", gin.H{"id": int(row.ID), "name": row.Name, "api_key": rawKey, "api_key_prefix": row.APIKeyPrefix, "created_at": row.CreatedAt})
}

func (h NativeHandlers) usersAPIKeyDelete(c *gin.Context, userID int64) {
	id := matrixParam(c, "id")
	res := h.DB.WithContext(c.Request.Context()).Where("id = ? AND user_id = ?", id, userID).Delete(&domain.UserAPIKey{})
	if writeDBError(c, res.Error, "") {
		return
	}
	if res.RowsAffected == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "API密钥不存在", nil)
		return
	}
	if numericID, ok := int64FromAny(id); ok {
		h.invalidateUserAPIKeyIDs(numericID)
	}
	writeSimpleSuccess(c, "API密钥已删除")
}
