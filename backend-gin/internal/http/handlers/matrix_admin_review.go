package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminPostQuality(c *gin.Context) {
	id := matrixParam(c, "id")
	body := readBodyMap(c)
	updates := map[string]any{"quality_marked_at": time.Now()}
	for _, key := range []string{"quality_level", "quality_reward"} {
		if value, ok := body[key]; ok {
			updates[key] = value
		}
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id = ?", id).Updates(updates)
	if writeDBError(c, res.Error, "") {
		return
	}
	writeSimpleSuccess(c, "质量标记已更新")
}

func (h NativeHandlers) adminPostsQualityBatch(c *gin.Context) {
	body := readBodyMap(c)
	ids := int64SliceFromAny(firstPresent(body, "ids", "post_ids"))
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供帖子ID列表", nil)
		return
	}
	updates := map[string]any{"quality_marked_at": time.Now()}
	for _, key := range []string{"quality_level", "quality_reward"} {
		if value, ok := body[key]; ok {
			updates[key] = value
		}
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id IN ?", ids).Updates(updates)
	if writeDBError(c, res.Error, "") {
		return
	}
	writeSimpleSuccess(c, "成功标记 "+strconv.FormatInt(res.RowsAffected, 10)+" 篇帖子")
}

func (h NativeHandlers) adminAuditAction(c *gin.Context, status int) {
	id := matrixParam(c, "id")
	updates := map[string]any{"status": status, "audit_time": time.Now()}
	if reason := toString(readBodyMap(c)["reason"]); reason != "" {
		updates["reason"] = reason
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Audit{}).Where("id = ?", id).Updates(updates)
	if writeDBError(c, res.Error, "") {
		return
	}
	writeSimpleSuccess(c, "审核状态已更新")
}

func (h NativeHandlers) adminContentReviewSettings(c *gin.Context) {
	usernameEnabled := h.Settings != nil && h.Settings.Bool("ai_username_review_enabled")
	contentEnabled := h.Settings != nil && h.Settings.Bool("ai_content_review_enabled")
	var moderation any
	if h.AI != nil {
		moderation = h.AI.PublicSettings().Moderation
	}
	writeSuccess(c, "获取设置成功", gin.H{
		"ai_auto_review":     usernameEnabled || contentEnabled,
		"ai_username_review": usernameEnabled,
		"ai_content_review":  contentEnabled,
		"moderation":         moderation,
	})
}

func (h NativeHandlers) adminUpdateContentReviewSettings(c *gin.Context) {
	body := readBodyMap(c)
	if h.Settings != nil {
		if raw, exists := body["ai_username_review"]; exists {
			if !h.Settings.Set(c.Request.Context(), "ai_username_review_enabled", jsBoolean(raw)) {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return
			}
		}
		if raw, exists := body["ai_content_review"]; exists {
			if !h.Settings.Set(c.Request.Context(), "ai_content_review_enabled", jsBoolean(raw)) {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return
			}
		}
		if raw, exists := body["ai_auto_review"]; exists {
			_, hasUsername := body["ai_username_review"]
			_, hasContent := body["ai_content_review"]
			if !hasUsername && !hasContent {
				enabled := jsBoolean(raw)
				if !h.Settings.Set(c.Request.Context(), "ai_username_review_enabled", enabled) ||
					!h.Settings.Set(c.Request.Context(), "ai_content_review_enabled", enabled) {
					response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
					return
				}
			}
		}
		if raw, exists := body["moderation"]; exists && h.AI != nil {
			if err := h.AI.UpdateSettings(c.Request.Context(), map[string]any{"moderation": raw}); err != nil {
				writeAIHTTPError(c, err)
				return
			}
		}
	}
	usernameEnabled := h.Settings != nil && h.Settings.Bool("ai_username_review_enabled")
	contentEnabled := h.Settings != nil && h.Settings.Bool("ai_content_review_enabled")
	var moderation any
	if h.AI != nil {
		moderation = h.AI.PublicSettings().Moderation
	}
	writeSuccess(c, "设置已更新", gin.H{
		"ai_auto_review":     usernameEnabled || contentEnabled,
		"ai_username_review": usernameEnabled,
		"ai_content_review":  contentEnabled,
		"moderation":         moderation,
	})
}

func (h NativeHandlers) adminContentReviewAction(c *gin.Context, status int) {
	id := matrixParam(c, "id")
	var audit domain.Audit
	err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND type IN ?", id, []int{3, 4}).Take(&audit).Error
	if writeDBError(c, err, "审核记录不存在") {
		return
	}
	now := time.Now()
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Audit{}).Where("id = ?", audit.ID).Updates(map[string]any{"status": status, "audit_time": now}).Error; writeDBError(c, err, "") {
		return
	}
	if audit.Type == 3 && audit.TargetID != nil {
		updates := map[string]any{"audit_status": status, "is_public": status == auditStatusOK}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Comment{}).Where("id = ?", *audit.TargetID).Updates(updates).Error; writeDBError(c, err, "") {
			return
		}
	}
	if status == auditStatusOK {
		writeSimpleSuccess(c, "审核通过成功")
		return
	}
	writeSimpleSuccess(c, "拒绝成功")
}

func (h NativeHandlers) adminContentReviewRetry(c *gin.Context) {
	id := matrixParam(c, "id")
	var audit domain.Audit
	err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND type IN ?", id, []int{3, 4}).Take(&audit).Error
	if writeDBError(c, err, "审核记录不存在") {
		return
	}
	if audit.RetryCount >= 5 {
		response.JSON(c, http.StatusBadRequest, response.CodeError, "已达到最大重试次数（5次）", nil)
		return
	}
	if audit.Status == nil || *audit.Status != auditStatusPending {
		response.JSON(c, http.StatusBadRequest, response.CodeError, "只有待审核状态的记录可以重试", nil)
		return
	}
	retryCount := audit.RetryCount + 1
	reason := "[AI重试审核失败 第" + strconv.Itoa(retryCount) + "次] AI服务无响应"
	updates := map[string]any{
		"risk_level":  "unknown",
		"categories":  datatypes.JSON([]byte("[]")),
		"reason":      reason,
		"status":      auditStatusPending,
		"retry_count": retryCount,
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Audit{}).Where("id = ?", audit.ID).Updates(updates).Error; writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, "AI重试完成，仍待审核", gin.H{"status": auditStatusPending, "retry_count": retryCount, "ai_result": nil})
}

func (h NativeHandlers) adminBannedWordsImport(c *gin.Context) {
	body := readBodyMap(c)
	words := parseStringSlice(body["words"])
	rows := make([]map[string]any, 0, len(words))
	for _, word := range words {
		rows = append(rows, map[string]any{"word": word, "enabled": true})
	}
	if len(rows) > 0 {
		_ = h.DB.WithContext(c.Request.Context()).Table("banned_words").Create(rows).Error
	}
	writeSuccess(c, "导入完成", gin.H{"count": len(rows)})
}

func (h NativeHandlers) adminBannedWordsExport(c *gin.Context) {
	var rows []map[string]any
	_ = h.DB.WithContext(c.Request.Context()).Table("banned_words").Find(&rows).Error
	writeSuccess(c, matrixMsgOK, rows)
}

func (h NativeHandlers) adminToggleActive(c *gin.Context, table string) {
	id := matrixParam(c, "id")
	var row map[string]any
	if err := h.DB.WithContext(c.Request.Context()).Table(table).Where("id = ?", id).Take(&row).Error; writeDBError(c, err, "资源不存在") {
		return
	}
	current, _ := boolFromAny(row["is_active"])
	res := h.DB.WithContext(c.Request.Context()).Table(table).Where("id = ?", id).Update("is_active", !current)
	if writeDBError(c, res.Error, "") {
		return
	}
	writeSuccess(c, "状态已切换", gin.H{"is_active": !current})
}

func (h NativeHandlers) adminUserEarnings(c *gin.Context) {
	userID := adminUserIDFromPath(c)
	var earnings domain.CreatorEarnings
	_ = h.DB.WithContext(c.Request.Context()).Where("user_id = ?", userID).First(&earnings).Error
	var wallet domain.UserWallet
	_ = h.DB.WithContext(c.Request.Context()).Where("user_id = ?", userID).First(&wallet).Error
	writeSuccess(c, matrixMsgOK, gin.H{"user_id": userID, "creator_earnings": earnings, "wallet": wallet})
}

func (h NativeHandlers) adminUserAdjustEarnings(c *gin.Context, add bool) {
	userID := adminUserIDFromPath(c)
	body := readBodyMap(c)
	amount, _ := float64FromAny(body["amount"])
	if !add {
		amount = -amount
	}
	err := h.DB.WithContext(c.Request.Context()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"balance": gorm.Expr("creator_earnings.balance + ?", amount), "total_earnings": gorm.Expr("creator_earnings.total_earnings + ?", positiveFloat(amount))}),
	}).Create(&domain.CreatorEarnings{UserID: userID, Balance: amount, TotalEarnings: positiveFloat(amount)}).Error
	if writeDBError(c, err, "") {
		return
	}
	writeSimpleSuccess(c, "收益已调整")
}

func (h NativeHandlers) adminMediaPublic(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "pageSize", 20), 100)
	offset := (page - 1) * limit
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.MediaLibrary{})
	if mediaType := c.Query("type"); mediaType != "" {
		query = query.Where("type = ?", mediaType)
	}
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.MediaLibrary
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"items": rows, "total": total, "page": page, "pageSize": limit})
}

func (h NativeHandlers) adminMediaUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoFile, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	url, _, errMsg, ok := h.storeLocalFilePreservingName(
		data,
		h.Config.Upload.Attachment.LocalUploadDir,
		"attachments",
		file.Filename,
	)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	writeSuccess(c, msgUploadOK, gin.H{
		"contentType":  file.Header.Get("Content-Type"),
		"originalname": file.Filename,
		"signedUrl":    h.signFileURL(url),
		"size":         file.Size,
		"url":          url,
	})
}
