package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) usersOnboardingConfig(c *gin.Context) {
	locale := localization.ResolveRequest(c.Request)
	interestOptions := parseStringSlice(nil)
	customFields := []any{}
	enabled := true
	allowSkip := true
	fields := defaultOnboardingFields()
	pointsIntro := gin.H{
		"title":        "",
		"summary":      "",
		"detail":       "",
		"result_title": "",
		"saved_text":   "",
		"wallet_label": "",
		"wallet_url":   "",
	}
	if h.Settings != nil {
		interestOptions = h.Settings.LocalizedStringArray("onboarding_interest_options", locale)
		raw := h.Settings.Get("onboarding_custom_fields")
		if parsed, ok := raw.([]any); ok {
			customFields = parsed
		} else if text := toString(raw); text != "" {
			if parsed, ok := jsonValue([]byte(text)).([]any); ok {
				customFields = parsed
			}
		}
		enabled = h.Settings.Bool("onboarding_enabled")
		allowSkip = h.Settings.Bool("onboarding_allow_skip")
		fields = h.availableOnboardingFields(interestOptions)
		pointsIntro["title"] = h.Settings.LocalizedString("onboarding_points_intro_title", locale)
		pointsIntro["summary"] = h.Settings.LocalizedString("onboarding_points_intro_summary", locale)
		pointsIntro["detail"] = h.Settings.LocalizedString("onboarding_points_intro_detail", locale)
		pointsIntro["result_title"] = h.Settings.LocalizedString("onboarding_result_title", locale)
		pointsIntro["saved_text"] = h.Settings.LocalizedString("onboarding_result_saved_text", locale)
		pointsIntro["wallet_label"] = h.Settings.LocalizedString("onboarding_points_wallet_label", locale)
		pointsIntro["wallet_url"] = h.Settings.String("onboarding_points_wallet_url")
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"interest_options": interestOptions,
		"custom_fields":    customFields,
		"enabled":          enabled,
		"allow_skip":       allowSkip,
		"fields":           fields,
		"profile_tasks":    h.onboardingProfileTasks(c),
		"points_intro":     pointsIntro,
	})
}

func defaultOnboardingFields() gin.H {
	return gin.H{
		"avatar":       gin.H{"enabled": true, "required": true},
		"background":   gin.H{"enabled": true, "required": true},
		"name":         gin.H{"enabled": true, "required": true},
		"signature":    gin.H{"enabled": true, "required": false},
		"interests":    gin.H{"enabled": false, "required": false, "min": 1},
		"customFields": gin.H{"enabled": true, "required": false},
	}
}

func (h NativeHandlers) onboardingFields() gin.H {
	if h.Settings == nil {
		return defaultOnboardingFields()
	}
	return gin.H{
		"avatar": gin.H{
			"enabled":  h.Settings.Bool("onboarding_avatar_enabled"),
			"required": h.Settings.Bool("onboarding_avatar_required"),
		},
		"background": gin.H{
			"enabled":  h.Settings.Bool("onboarding_background_enabled"),
			"required": h.Settings.Bool("onboarding_background_required"),
		},
		"name": gin.H{
			"enabled":  h.Settings.Bool("onboarding_name_enabled"),
			"required": h.Settings.Bool("onboarding_name_required"),
		},
		"signature": gin.H{
			"enabled":  h.Settings.Bool("onboarding_signature_enabled"),
			"required": h.Settings.Bool("onboarding_signature_required"),
		},
		"interests": gin.H{
			"enabled":  h.Settings.Bool("onboarding_interests_enabled"),
			"required": h.Settings.Bool("onboarding_interests_required"),
			"min":      max(h.Settings.Int("onboarding_min_interests", 1), 1),
		},
		"customFields": gin.H{"enabled": true, "required": false},
	}
}

func (h NativeHandlers) availableOnboardingFields(interestOptions []string) gin.H {
	fields := h.onboardingFields()
	fields["interests"] = availableOnboardingInterestRule(fields["interests"], interestOptions)
	return fields
}

func availableOnboardingInterestRule(raw any, interestOptions []string) gin.H {
	rule, _ := raw.(gin.H)
	enabled := onboardingBoolRuleValue(rule["enabled"], false)
	required := onboardingBoolRuleValue(rule["required"], false)
	minimum := max(onboardingIntRuleValue(rule["min"], 1), 1)
	optionCount := len(onboardingUniqueTextValues(interestOptions))
	if !enabled || optionCount == 0 {
		return gin.H{"enabled": false, "required": false, "min": minimum}
	}
	return gin.H{
		"enabled":  true,
		"required": required,
		"min":      min(minimum, optionCount),
	}
}

func onboardingBoolRuleValue(value any, fallback bool) bool {
	if parsed, ok := boolFromAny(value); ok {
		return parsed
	}
	return fallback
}

func onboardingIntRuleValue(value any, fallback int) int {
	if parsed, ok := intFromAny(value); ok {
		return parsed
	}
	return fallback
}

func onboardingUniqueTextValues(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		out = append(out, text)
	}
	return out
}

func (h NativeHandlers) onboardingProfileTasks(c *gin.Context) []gin.H {
	rows, err := h.pointsRepo().TaskConfigs(c.Request.Context(), true)
	if err != nil {
		rows = nil
	}
	byType := map[string]domain.PointsTaskConfig{}
	for _, row := range rows {
		byType[row.TaskType] = row
	}
	taskTypes := []string{
		repositories.PointsTaskSetAvatar,
		repositories.PointsTaskSetBackground,
		repositories.PointsTaskSetName,
		repositories.PointsTaskSetSignature,
	}
	out := make([]gin.H, 0, len(taskTypes))
	for _, taskType := range taskTypes {
		row, ok := byType[taskType]
		if !ok {
			out = append(out, gin.H{
				"task_type":   taskType,
				"name":        onboardingTaskName(taskType),
				"description": nil,
				"points":      0,
				"is_active":   false,
			})
			continue
		}
		out = append(out, gin.H{
			"task_type":   row.TaskType,
			"name":        row.Name,
			"description": row.Description,
			"points":      row.Points,
			"is_active":   row.IsActive,
		})
	}
	return out
}

func onboardingTaskName(taskType string) string {
	switch taskType {
	case repositories.PointsTaskSetAvatar:
		return "设置头像"
	case repositories.PointsTaskSetBackground:
		return "设置背景"
	case repositories.PointsTaskSetName:
		return "设置名称"
	case repositories.PointsTaskSetSignature:
		return "设置签名"
	default:
		return taskType
	}
}

func (h NativeHandlers) usersDraftGet(c *gin.Context, userID int64) {
	if h.Cache == nil {
		writeSuccess(c, matrixMsgOK, nil)
		return
	}
	value, _ := h.Cache.Get("onboarding_draft:" + strconv.FormatInt(userID, 10))
	writeSuccess(c, matrixMsgOK, value)
}

func (h NativeHandlers) usersDraftSave(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	draft := gin.H{
		"currentStep":  body["currentStep"],
		"gender":       body["gender"],
		"birthday":     body["birthday"],
		"interests":    body["interests"],
		"customFields": body["customFields"],
	}
	if h.Cache != nil {
		h.Cache.Set("onboarding_draft:"+strconv.FormatInt(userID, 10), draft, 7*24*time.Hour)
	}
	writeSimpleSuccess(c, matrixMsgOK)
}

func (h NativeHandlers) usersOnboardingSubmit(c *gin.Context, userID int64) {
	body := readBodyMap(c)
	if skipped, ok := boolFromAny(body["skipped"]); ok && skipped {
		if h.Settings != nil && !h.Settings.Bool("onboarding_allow_skip") {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "当前不允许跳过初始设置", nil)
			return
		}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", userID).Update("profile_completed", true).Error; writeDBError(c, err, "") {
			return
		}
		if h.Cache != nil {
			h.Cache.Delete("onboarding_draft:" + strconv.FormatInt(userID, 10))
		}
		var user domain.User
		if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", userID).First(&user).Error; writeDBError(c, err, "") {
			return
		}
		data := h.userPublicMap(user)
		data["skipped"] = true
		writeSuccess(c, "初始设置已跳过", data)
		return
	}

	var current domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", userID).First(&current).Error; writeDBError(c, err, "用户不存在") {
		return
	}
	fields := defaultOnboardingFieldRules()
	if h.Settings != nil {
		fields = h.availableOnboardingFieldRules(localization.ResolveRequest(c.Request))
	}
	if errMessage := validateOnboardingSubmission(body, current, fields, h.normalizeFileURLForStorage); errMessage != "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMessage, nil)
		return
	}
	updates := map[string]any{"profile_completed": true}
	awardTasks := []profileAwardTask{}
	if value := sanitizePlainSubmittedText(toString(body["nickname"])); fields["name"].Enabled && value != "" {
		updates["nickname"] = value
		if value != current.Nickname {
			awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetName, reason: "设置名称奖励"})
		}
	}
	for key, column := range map[string]string{
		"avatar":     "avatar",
		"background": "background",
		"bio":        "bio",
	} {
		fieldKey := key
		if key == "bio" {
			fieldKey = "signature"
		}
		if !fields[fieldKey].Enabled {
			continue
		}
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
					if text != stringPtrValue(current.Avatar) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetAvatar, reason: "设置头像奖励"})
					}
				case "background":
					if text != stringPtrValue(current.Background) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetBackground, reason: "设置背景奖励"})
					}
				case "bio":
					if text != stringPtrValue(current.Bio) {
						awardTasks = append(awardTasks, profileAwardTask{taskType: repositories.PointsTaskSetSignature, reason: "设置签名奖励"})
					}
				}
			}
		}
	}
	if _, exists := body["gender"]; exists {
		updates["gender"] = nilIfEmpty(toString(body["gender"]))
	}
	if _, exists := body["birthday"]; exists {
		if birthday := parseTimeAny(body["birthday"]); birthday != nil {
			if birthday.After(time.Now()) {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "生日不能是未来日期", nil)
				return
			}
			updates["birthday"] = *birthday
			updates["zodiac_sign"] = zodiacSign(*birthday)
		}
	}
	if _, exists := body["interests"]; exists {
		if fields["interests"].Enabled {
			updates["interests"] = jsonBytes(sanitizedStringSlice(body["interests"], 20, 50))
		}
	}
	if _, exists := body["custom_fields"]; exists {
		updates["custom_fields"] = jsonBytes(sanitizedStringMap(body["custom_fields"], 20, 50))
	}
	if _, ok := updates["bio"]; ok {
		updates["bio_audit_status"] = auditStatusOK
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error; writeDBError(c, err, "") {
		return
	}
	if h.Cache != nil {
		h.Cache.Delete("onboarding_draft:" + strconv.FormatInt(userID, 10))
	}
	var user domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", userID).First(&user).Error; writeDBError(c, err, "") {
		return
	}
	data := h.userPublicMap(user)
	if awards := h.awardProfileTasksBestEffort(c, userID, awardTasks); len(awards) > 0 {
		data["points_awards"] = awards
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "引导信息保存成功", "success": true, "data": data})
}

func validateOnboardingSubmission(body map[string]any, current domain.User, fields map[string]onboardingFieldRule, normalizeFileURL func(string) string) string {
	valueFor := func(key string, currentValue string) string {
		if raw, exists := body[key]; exists {
			if key == "avatar" || key == "background" {
				return normalizeFileURL(toString(raw))
			}
			return strings.TrimSpace(toString(raw))
		}
		return strings.TrimSpace(currentValue)
	}
	if rule := fields["name"]; rule.Enabled && rule.Required && valueFor("nickname", current.Nickname) == "" {
		return "请先设置名称"
	}
	if rule := fields["avatar"]; rule.Enabled && rule.Required && valueFor("avatar", stringPtrValue(current.Avatar)) == "" {
		return "请先设置头像"
	}
	if rule := fields["background"]; rule.Enabled && rule.Required && valueFor("background", stringPtrValue(current.Background)) == "" {
		return "请先设置背景"
	}
	if rule := fields["signature"]; rule.Enabled && rule.Required && valueFor("bio", stringPtrValue(current.Bio)) == "" {
		return "请先设置签名"
	}
	if rule := fields["interests"]; rule.Enabled && rule.Required {
		interests := onboardingInterestValues(body["interests"], current.Interests)
		if len(interests) < max(rule.Min, 1) {
			return "请先选择兴趣爱好"
		}
	}
	return ""
}

type onboardingFieldRule struct {
	Enabled  bool
	Required bool
	Min      int
}

func defaultOnboardingFieldRules() map[string]onboardingFieldRule {
	return map[string]onboardingFieldRule{
		"avatar":     {Enabled: true, Required: true},
		"background": {Enabled: true, Required: true},
		"name":       {Enabled: true, Required: true},
		"signature":  {Enabled: true, Required: false},
		"interests":  {Enabled: false, Required: false, Min: 1},
	}
}

func (h NativeHandlers) onboardingFieldRules() map[string]onboardingFieldRule {
	return map[string]onboardingFieldRule{
		"avatar": {
			Enabled:  h.Settings.Bool("onboarding_avatar_enabled"),
			Required: h.Settings.Bool("onboarding_avatar_required"),
		},
		"background": {
			Enabled:  h.Settings.Bool("onboarding_background_enabled"),
			Required: h.Settings.Bool("onboarding_background_required"),
		},
		"name": {
			Enabled:  h.Settings.Bool("onboarding_name_enabled"),
			Required: h.Settings.Bool("onboarding_name_required"),
		},
		"signature": {
			Enabled:  h.Settings.Bool("onboarding_signature_enabled"),
			Required: h.Settings.Bool("onboarding_signature_required"),
		},
		"interests": {
			Enabled:  h.Settings.Bool("onboarding_interests_enabled"),
			Required: h.Settings.Bool("onboarding_interests_required"),
			Min:      max(h.Settings.Int("onboarding_min_interests", 1), 1),
		},
	}
}

func (h NativeHandlers) availableOnboardingFieldRules(locale string) map[string]onboardingFieldRule {
	rules := h.onboardingFieldRules()
	interestOptions := h.Settings.LocalizedStringArray("onboarding_interest_options", locale)
	if rule, ok := rules["interests"]; ok {
		rules["interests"] = availableOnboardingFieldRule(rule, interestOptions)
	}
	return rules
}

func availableOnboardingFieldRule(rule onboardingFieldRule, interestOptions []string) onboardingFieldRule {
	optionCount := len(onboardingUniqueTextValues(interestOptions))
	if !rule.Enabled || optionCount == 0 {
		rule.Enabled = false
		rule.Required = false
		rule.Min = max(rule.Min, 1)
		return rule
	}
	rule.Min = min(max(rule.Min, 1), optionCount)
	return rule
}

func onboardingInterestValues(raw any, current any) []string {
	if raw == nil {
		raw = current
	}
	values := parseStringSlice(raw)
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
