package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func normalizeHeaderSetting(value any) (map[string]string, bool) {
	headers := stringMapFromSetting(value)
	for key := range headers {
		if strings.TrimSpace(key) == "" {
			return nil, false
		}
	}
	return headers, true
}

func normalizeModelParametersSetting(value any) (map[string]any, bool) {
	params := jsonMapFromSetting(value)
	if params == nil {
		return nil, false
	}
	return params, true
}

func stringMapFromSetting(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, value := range typed {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	case map[string]any:
		for key, value := range typed {
			out[strings.TrimSpace(key)] = strings.TrimSpace(fmt.Sprint(value))
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return out
		}
		var parsed map[string]any
		if json.Unmarshal([]byte(typed), &parsed) == nil {
			for key, value := range parsed {
				out[strings.TrimSpace(key)] = strings.TrimSpace(fmt.Sprint(value))
			}
		}
	}
	for key, value := range out {
		if key == "" || value == "" {
			delete(out, key)
		}
	}
	return out
}

func jsonMapFromSetting(value any) map[string]any {
	out := map[string]any{}
	if value == nil {
		return out
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key != "" && value != nil {
				out[key] = value
			}
		}
	case map[string]string:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key != "" {
				out[key] = value
			}
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return out
		}
		parsed, ok := decodeJSONMapString(typed)
		if !ok {
			return nil
		}
		return jsonMapFromSetting(parsed)
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		parsed, ok := decodeJSONMapString(string(raw))
		if !ok {
			return nil
		}
		return jsonMapFromSetting(parsed)
	}
	return out
}

func decodeJSONMapString(value string) (map[string]any, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return map[string]any{}, true
	}
	if !json.Valid([]byte(value)) {
		return nil, false
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var parsed map[string]any
	if decoder.Decode(&parsed) != nil {
		return nil, false
	}
	if parsed == nil {
		return map[string]any{}, true
	}
	return parsed, true
}

func normalizeReasoningEffortSetting(value any) (string, bool) {
	text := strings.TrimSpace(fmt.Sprint(value))
	normalized := normalizeReasoningEffortValue(text)
	if normalized == "" {
		switch strings.ToLower(text) {
		case "", "default", "none", "off", "false", "0", "auto":
			return "", true
		default:
			return "", false
		}
	}
	return normalized, true
}

func normalizeReasoningEffortValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal", "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func boundedInt(value int, min int, max int, fallback int) int {
	if value < min || value > max {
		return fallback
	}
	return value
}

func boundedOptionalInt(value int, min int, max int, fallback int) int {
	if value == 0 {
		return 0
	}
	return boundedInt(value, min, max, fallback)
}

func boundedOptionalMinMaxInt(value int, min int, max int, fallback int) int {
	if value == 0 {
		return 0
	}
	return boundedInt(value, min, max, fallback)
}

func boundedOptionalMinInt(value int, min int, fallback int) int {
	if value == 0 {
		return 0
	}
	if value < min {
		return fallback
	}
	return value
}

func boundedFloat(value float64, min float64, max float64, fallback float64) float64 {
	if value < min || value > max {
		return fallback
	}
	return value
}

func settingFloat(value any, fallback float64) float64 {
	parsed, ok := floatSettingInput(value)
	if !ok {
		return fallback
	}
	return parsed
}

func intSettingInput(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		return parsed, err == nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func floatSettingInput(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := strconv.ParseFloat(typed.String(), 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func boolSettingInput(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	case float64:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	}
	return false, false
}

func int64SettingInput(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := strconv.ParseInt(typed.String(), 10, 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func autoCommentConfigFromSettings(settings *SettingsService, defaults AIAutoCommentConfig) AIAutoCommentConfig {
	if settings == nil {
		return defaults
	}
	cfg := defaults
	cfg.Enabled = settings.Bool(AISettingAutoCommentEnabled)
	cfg.BotUserID = int64(settings.Int(AISettingAutoCommentBotUserID, int(defaults.BotUserID)))
	cfg.BotUserIDMin = int64(settings.Int(AISettingAutoCommentBotUserIDMin, int(defaults.BotUserIDMin)))
	cfg.BotUserIDMax = int64(settings.Int(AISettingAutoCommentBotUserIDMax, int(defaults.BotUserIDMax)))
	cfg.TemplateKey = nonEmptyString(settings.String(AISettingAutoCommentTemplateKey), defaults.TemplateKey)
	cfg.DelaySeconds = boundedInt(settings.Int(AISettingAutoCommentDelaySeconds, defaults.DelaySeconds), 0, 3600, defaults.DelaySeconds)
	cfg.MaxImages = boundedInt(settings.Int(AISettingAutoCommentMaxImages, defaults.MaxImages), 0, 12, defaults.MaxImages)
	cfg.ImageSelectionMode = normalizeAIImageSelectionMode(settings.String(AISettingAutoCommentImageMode))
	cfg.Style = normalizeAIPromptStyle(settings.String(AISettingAutoCommentStyle), defaults.Style)
	if cfg.BotUserIDMin > cfg.BotUserIDMax {
		cfg.BotUserIDMin, cfg.BotUserIDMax = cfg.BotUserIDMax, cfg.BotUserIDMin
	}
	return cfg
}

func normalizeAutoCommentSetting(value any, current AIAutoCommentConfig) (AIAutoCommentConfig, bool) {
	rawMap, ok := value.(map[string]any)
	if !ok {
		if raw, err := json.Marshal(value); err == nil {
			_ = json.Unmarshal(raw, &rawMap)
		}
	}
	if rawMap == nil {
		return current, false
	}
	next := current
	if value, ok := firstPresent(rawMap, "enabled"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return current, false
		}
		next.Enabled = parsed
	}
	if value, ok := firstPresent(rawMap, "botUserId", "bot_user_id"); ok {
		parsed, valid := int64SettingInput(value)
		if !valid || parsed < 0 {
			return current, false
		}
		next.BotUserID = parsed
	}
	if value, ok := firstPresent(rawMap, "botUserIdMin", "bot_user_id_min"); ok {
		parsed, valid := int64SettingInput(value)
		if !valid || parsed < 0 {
			return current, false
		}
		next.BotUserIDMin = parsed
	}
	if value, ok := firstPresent(rawMap, "botUserIdMax", "bot_user_id_max"); ok {
		parsed, valid := int64SettingInput(value)
		if !valid || parsed < 0 {
			return current, false
		}
		next.BotUserIDMax = parsed
	}
	if value, ok := firstPresent(rawMap, "templateKey", "template_key"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return current, false
		}
		next.TemplateKey = text
	}
	if value, ok := firstPresent(rawMap, "delaySeconds", "delay_seconds"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 0 || parsed > 3600 {
			return current, false
		}
		next.DelaySeconds = parsed
	}
	if value, ok := firstPresent(rawMap, "maxImages", "max_images"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 0 || parsed > 12 {
			return current, false
		}
		next.MaxImages = parsed
	}
	if value, ok := firstPresent(rawMap, "imageSelectionMode", "image_selection_mode", "imageMode", "image_mode"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if !validAIImageSelectionMode(text) {
			return current, false
		}
		next.ImageSelectionMode = normalizeAIImageSelectionMode(text)
	}
	if value, ok := firstPresent(rawMap, "style"); ok {
		style := normalizeAIPromptStyle(fmt.Sprint(value), "")
		if style == "" {
			return current, false
		}
		next.Style = style
	}
	if next.BotUserIDMin > next.BotUserIDMax {
		next.BotUserIDMin, next.BotUserIDMax = next.BotUserIDMax, next.BotUserIDMin
	}
	next.ImageSelectionMode = normalizeAIImageSelectionMode(next.ImageSelectionMode)
	return next, true
}

func nonEmptyString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstPresent(input map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func chunkProgress(done int, total int) int {
	if total <= 0 {
		return 100
	}
	percent := int(float64(done) / float64(total) * 100)
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func statusFromError(err error) string {
	var aiErr AIError
	if errors.As(err, &aiErr) && aiErr.Code == "error.ai_timeout" {
		return "failed"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "canceled"
	}
	return "failed"
}

func aiErrorCode(err error) string {
	var aiErr AIError
	if errors.As(err, &aiErr) && aiErr.Code != "" {
		return aiErr.Code
	}
	if errors.Is(err, context.Canceled) {
		return "error.ai_request_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "error.ai_timeout"
	}
	return "error.ai_request_failed"
}
