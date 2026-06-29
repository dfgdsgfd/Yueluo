package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) adminSettings(c *gin.Context) {
	if h.Settings == nil {
		writeSuccess(c, matrixMsgOK, gin.H{})
		return
	}
	all := h.Settings.All()
	if value, ok := all[services.AISettingAPIKey]; ok {
		all[services.AISettingAPIKey] = services.MaskSecret(toString(value))
	}
	writeSuccess(c, matrixMsgOK, all)
}

func (h NativeHandlers) adminSystemSettings(c *gin.Context) {
	all := gin.H{}
	raw := gin.H{}
	if h.Settings != nil {
		raw = h.Settings.All()
		if value, ok := raw[services.AISettingAPIKey]; ok {
			raw[services.AISettingAPIKey] = services.MaskSecret(toString(value))
		}
		for key, value := range h.Settings.All() {
			if key == services.AISettingAPIKey {
				value = services.MaskSecret(toString(value))
			}
			meta := gin.H{
				"labelKey": "systemSettings." + key + ".label",
				"value":    value,
				"type":     settingTypeForKey(key, value),
			}
			if services.LocalizedSettingKeys[key] {
				meta["localized"] = true
				meta["type"] = "localized"
				meta["value"] = localization.CompleteMap(value)
			}
			if settingHint(key) != "" {
				meta["hintKey"] = "systemSettings." + key + ".hint"
			}
			all[key] = meta
		}
	}
	engine := "auto"
	if value := strings.TrimSpace(toString(raw["hidden_watermark_engine"])); value != "" {
		engine = value
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"settings": all,
		"raw":      raw,
		"watermarkRuntime": gin.H{
			"payloadBytes":      domain.ImageWatermarkPayloadBytes,
			"payloadBits":       domain.ImageWatermarkPayloadBytes * 8,
			"payloadFormat":     "short_code_v1",
			"traceTokenBytes":   domain.ImageWatermarkTraceTokenBytes,
			"engineMode":        engine,
			"remoteConfigured":  strings.TrimSpace(h.Config.WebP.HiddenWatermark.Remote.URL) != "",
			"referenceRecovery": true,
		},
	})
}

func (h NativeHandlers) adminUpdateSettings(c *gin.Context) {
	body := readBodyMap(c)
	updates := flattenSettings(body)
	for key, value := range updates {
		if isSiteSettingKey(key) {
			normalized, ok := normalizeSiteSetting(key, value)
			if !ok {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_site_setting", gin.H{"key": key})
				return
			}
			updates[key] = normalized
			continue
		}
		if strings.HasPrefix(key, "onboarding_") {
			normalized, ok := normalizeOnboardingSetting(key, value)
			if !ok {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_onboarding_setting", gin.H{"key": key})
				return
			}
			updates[key] = normalized
			continue
		}
		switch key {
		case "post_content_max_length":
			parsed, valid := intFromAny(value)
			if !valid || parsed < 1 || parsed > 1_000_000 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_post_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		case "post_resource_section_position":
			position := strings.TrimSpace(toString(value))
			if position != "before_content" && position != "after_content" {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_post_setting", gin.H{"key": key})
				return
			}
			updates[key] = position
			continue
		case services.OAuth2AppCallbackURLsSetting:
			normalized, valid := normalizeOAuth2AppCallbackURLSetting(value)
			if !valid {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_oauth2_app_callback_urls", gin.H{"key": key})
				return
			}
			updates[key] = normalized
			continue
		case services.FileRecycleRetentionDaysKey:
			parsed, valid := intFromAny(value)
			if !valid || parsed < 1 || parsed > 3650 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_file_recycle_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		case services.FileRecycleCleanupIntervalHoursKey:
			parsed, valid := intFromAny(value)
			if !valid || parsed < 1 || parsed > 24*30 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_file_recycle_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		case "notification_interaction_suppression_enabled":
			parsed, valid := boolFromAny(value)
			if !valid {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_notification_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		case "notification_interaction_suppression_window_seconds":
			parsed, valid := intFromAny(value)
			if !valid || parsed < 60 || parsed > 86400 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_notification_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		case "notification_interaction_suppression_threshold":
			parsed, valid := intFromAny(value)
			if !valid || parsed < 1 || parsed > 100 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_notification_setting", gin.H{"key": key})
				return
			}
			updates[key] = parsed
			continue
		}
		normalized, ok := normalizeImageSetting(key, value)
		if !ok {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_image_setting", gin.H{"key": key})
			return
		}
		updates[key] = normalized
	}
	maxImages := defaultMaxPostImages
	archiveThreshold := 25
	if h.Settings != nil {
		maxImages = h.Settings.Int("image_post_max_count", maxImages)
		archiveThreshold = h.Settings.Int("image_archive_threshold", archiveThreshold)
	}
	if value, ok := intFromAny(updates["image_post_max_count"]); ok {
		maxImages = value
	}
	if value, ok := intFromAny(updates["image_archive_threshold"]); ok {
		archiveThreshold = value
	}
	if archiveThreshold > maxImages {
		updates["image_archive_threshold"] = maxImages
	}
	if h.Settings != nil {
		for key, value := range updates {
			if !h.Settings.Set(c.Request.Context(), key, value) {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return
			}
		}
	}
	writeSimpleSuccess(c, "系统设置已更新")
}

func isSiteSettingKey(key string) bool {
	switch key {
	case services.SiteTitleSetting, services.SiteDescriptionSetting, services.SiteAvatarURLSetting:
		return true
	default:
		return false
	}
}

func normalizeSiteSetting(key string, value any) (any, bool) {
	switch key {
	case services.SiteTitleSetting:
		return normalizeLocalizedPlainTextSetting(value, services.DefaultSiteTitle, 80), true
	case services.SiteDescriptionSetting:
		return normalizeLocalizedPlainTextSetting(value, services.DefaultSiteDescription, 180), true
	case services.SiteAvatarURLSetting:
		text := strings.TrimSpace(toString(value))
		if text == "" {
			return "", true
		}
		if len([]rune(text)) > 2048 || strings.ContainsAny(text, "\r\n\t") {
			return nil, false
		}
		if strings.HasPrefix(text, "/") {
			return text, true
		}
		parsed, err := url.Parse(text)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, false
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			return text, true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}

func normalizeLocalizedPlainTextSetting(value any, fallback string, maxRunes int) map[string]any {
	completed := localization.CompleteMap(value)
	out := map[string]any{}
	for _, locale := range localization.Supported {
		text := sanitizePlainSubmittedText(toString(completed[locale]))
		if strings.TrimSpace(text) == "" {
			text = fallback
		}
		out[locale] = truncateSettingText(text, maxRunes)
	}
	return out
}

func truncateSettingText(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func normalizeOnboardingSetting(key string, value any) (any, bool) {
	switch key {
	case "onboarding_enabled",
		"onboarding_allow_skip",
		"onboarding_avatar_enabled",
		"onboarding_avatar_required",
		"onboarding_background_enabled",
		"onboarding_background_required",
		"onboarding_name_enabled",
		"onboarding_name_required",
		"onboarding_signature_enabled",
		"onboarding_signature_required",
		"onboarding_interests_enabled",
		"onboarding_interests_required":
		parsed, valid := boolFromAny(value)
		return parsed, valid
	case "onboarding_min_interests":
		parsed, valid := intFromAny(value)
		if !valid || parsed < 1 || parsed > 100 {
			return nil, false
		}
		return parsed, true
	case "onboarding_interest_options":
		return normalizeLocalizedInterestOptionsSetting(value), true
	case "onboarding_custom_fields":
		return normalizeStructuredSettingText(value), true
	case "onboarding_points_intro_title",
		"onboarding_points_intro_summary",
		"onboarding_points_intro_detail",
		"onboarding_result_title",
		"onboarding_result_saved_text",
		"onboarding_points_wallet_label":
		return localization.CompleteMap(value), true
	case "onboarding_points_wallet_url":
		text := strings.TrimSpace(toString(value))
		if text == "" {
			return "/wallet", true
		}
		return text, true
	default:
		return nil, false
	}
}

func normalizeLocalizedInterestOptionsSetting(value any) map[string]any {
	completed := localization.CompleteMap(value)
	out := map[string]any{}
	for _, locale := range localization.Supported {
		out[locale] = normalizeInterestOptionList(completed[locale])
	}
	return out
}

func normalizeInterestOptionList(value any) []string {
	values := parseStringSlice(value)
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text != "" {
			if parsed := jsonValue([]byte(text)); parsed == nil {
				values = strings.FieldsFunc(text, func(r rune) bool {
					return r == '\n' || r == '\r' || r == ',' || r == '，'
				})
			}
		}
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		text := sanitizePlainSubmittedText(value)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		out = append(out, text)
	}
	return out
}

func normalizeStructuredSettingText(value any) any {
	if text := strings.TrimSpace(toString(value)); text != "" {
		if parsed := jsonValue([]byte(text)); parsed != nil {
			return parsed
		}
	}
	return value
}

func (h NativeHandlers) adminStatsOverview(c *gin.Context) {
	count := func(table string) int64 {
		var total int64
		_ = h.DB.WithContext(c.Request.Context()).Table(table).Count(&total).Error
		return total
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"users":         count("users"),
		"posts":         count("posts"),
		"comments":      count("comments"),
		"reports":       count("reports"),
		"feedback":      count("feedback"),
		"announcements": count("announcements"),
	})
}

func (h NativeHandlers) adminTestUsers(c *gin.Context) {
	var users []domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("is_active = ?", true).Order("created_at DESC").Limit(20).Find(&users).Error; writeDBError(c, err, "") {
		return
	}
	out := make([]gin.H, 0, len(users))
	for _, user := range users {
		out = append(out, gin.H{"id": user.ID, "user_id": user.UserID, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar)})
	}
	writeSuccess(c, matrixMsgOK, out)
}

func (h NativeHandlers) adminRecommendationConfig(c *gin.Context) {
	if h.Settings == nil {
		writeSuccess(c, matrixMsgOK, gin.H{})
		return
	}
	raw := h.Settings.Get("recommend_config")
	if text := toString(raw); text != "" {
		if value := jsonValue([]byte(text)); value != nil {
			writeSuccess(c, matrixMsgOK, value)
			return
		}
	}
	writeSuccess(c, matrixMsgOK, raw)
}

func (h NativeHandlers) adminSaveRecommendationConfig(c *gin.Context) {
	body := readBodyMap(c)
	if h.Settings != nil {
		if !h.Settings.Set(c.Request.Context(), "recommend_config", body) {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
	}
	writeSimpleSuccess(c, "推荐配置已更新")
}

func (h NativeHandlers) adminRecommendationPostBatch(c *gin.Context) {
	body := readBodyMap(c)
	ids := int64SliceFromAny(body["post_ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "post_ids 不能为空", nil)
		return
	}
	count := 0
	for _, postID := range ids {
		row := map[string]any{"post_id": postID}
		for _, key := range []string{"boost_score", "is_pinned", "is_suppressed", "is_active", "reason"} {
			if value, ok := body[key]; ok {
				row[key] = value
			}
		}
		err := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "post_id"}},
			DoUpdates: clause.Assignments(row),
		}).Create(row).Error
		if err == nil {
			count++
		}
	}
	writeSuccess(c, "已批量配置 "+strconv.Itoa(count)+" 篇帖子", gin.H{"count": count})
}

func (h NativeHandlers) adminRecommendationPush(c *gin.Context) {
	body := readBodyMap(c)
	postID, ok := int64FromAny(body["post_id"])
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "帖子ID不能为空", nil)
		return
	}
	row := map[string]any{"post_id": postID, "is_pinned": true, "is_active": true}
	for _, key := range []string{"boost_score", "target_user_id", "reason"} {
		if value, ok := body[key]; ok {
			row[key] = value
		}
	}
	err := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(row),
	}).Create(row).Error
	if writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, "主动推荐配置已生效", row)
}
