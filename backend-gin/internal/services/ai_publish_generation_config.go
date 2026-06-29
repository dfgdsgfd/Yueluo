package services

import (
	"fmt"
	"strings"
)

const (
	AITaskPublishTitleGenerate       = "publish_title_generate"
	AITaskPublishDetailGenerate      = "publish_detail_generate"
	AITaskPublishTitleDetailGenerate = "publish_title_detail_generate"
)

func isAIPublishGenerationTask(taskType string) bool {
	switch strings.TrimSpace(taskType) {
	case AITaskPublishTitleGenerate, AITaskPublishDetailGenerate, AITaskPublishTitleDetailGenerate:
		return true
	default:
		return false
	}
}

type AIPublishGenerationConfig struct {
	Enabled            bool                            `json:"enabled"`
	Detail             AIPublishGenerationTargetConfig `json:"detail"`
	Title              AIPublishGenerationTargetConfig `json:"title"`
	Combined           AIPublishGenerationTargetConfig `json:"combined"`
	MaxImages          int                             `json:"maxImages"`
	ImageSelectionMode string                          `json:"imageSelectionMode"`
	TitleMaxChars      int                             `json:"titleMaxChars"`
}

type AIPublishGenerationTargetConfig struct {
	Enabled     bool   `json:"enabled"`
	TemplateKey string `json:"templateKey"`
}

func defaultAIPublishGenerationConfig() AIPublishGenerationConfig {
	return AIPublishGenerationConfig{
		Enabled: true,
		Detail: AIPublishGenerationTargetConfig{
			Enabled:     true,
			TemplateKey: "publish_detail_generate",
		},
		Title: AIPublishGenerationTargetConfig{
			Enabled:     true,
			TemplateKey: "publish_title_generate",
		},
		Combined: AIPublishGenerationTargetConfig{
			Enabled:     true,
			TemplateKey: "publish_title_detail_generate",
		},
		MaxImages:          3,
		ImageSelectionMode: AIImageSelectionOrdered,
		TitleMaxChars:      40,
	}
}

func publishGenerationConfigFromSettings(settings *SettingsService, defaults AIPublishGenerationConfig) AIPublishGenerationConfig {
	if settings == nil {
		return defaults
	}
	return publishGenerationConfigFromAny(settings.Get(AISettingPublishGeneration), defaults)
}

func normalizePublishGenerationSetting(value any, current AIPublishGenerationConfig) (AIPublishGenerationConfig, bool) {
	if value == nil {
		return current, false
	}
	return publishGenerationConfigFromAny(value, current), true
}

func publishGenerationConfigFromAny(value any, current AIPublishGenerationConfig) AIPublishGenerationConfig {
	raw := jsonMapFromSetting(value)
	if raw == nil {
		return current
	}
	next := current
	if value, ok := firstPresent(raw, "enabled"); ok {
		if parsed, valid := boolSettingInput(value); valid {
			next.Enabled = parsed
		}
	}
	_, hasDetailConfig := firstPresent(raw, "detail", "details")
	_, hasTitleConfig := firstPresent(raw, "title")
	_, hasCombinedConfig := firstPresent(raw, "combined", "titleDetail", "title_detail")
	if value, ok := firstPresent(raw, "detail", "details"); ok {
		next.Detail = publishGenerationTargetFromAny(value, next.Detail, "publish_detail_generate")
	}
	if value, ok := firstPresent(raw, "title"); ok {
		next.Title = publishGenerationTargetFromAny(value, next.Title, "publish_title_generate")
	}
	if value, ok := firstPresent(raw, "combined", "titleDetail", "title_detail"); ok {
		next.Combined = publishGenerationTargetFromAny(value, next.Combined, "publish_title_detail_generate")
	}
	if hasCombinedConfig && !hasDetailConfig {
		next.Detail.Enabled = next.Combined.Enabled
	}
	if hasCombinedConfig && !hasTitleConfig {
		next.Title.Enabled = next.Combined.Enabled
	}
	if value, ok := firstPresent(raw, "title", "detail", "details"); ok && strings.TrimSpace(next.Combined.TemplateKey) == "" {
		next.Combined = publishGenerationTargetFromAny(value, next.Combined, "publish_title_detail_generate")
	}
	if value, ok := firstPresent(raw, "maxImages", "max_images"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.MaxImages = boundedInt(parsed, 0, 12, next.MaxImages)
		}
	}
	if value, ok := firstPresent(raw, "imageSelectionMode", "image_selection_mode", "imageMode", "image_mode"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if validAIImageSelectionMode(text) {
			next.ImageSelectionMode = normalizeAIImageSelectionMode(text)
		}
	}
	if value, ok := firstPresent(raw, "titleMaxChars", "title_max_chars", "titleLimit", "title_limit"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.TitleMaxChars = boundedInt(parsed, 8, 80, next.TitleMaxChars)
		}
	}
	if strings.TrimSpace(next.Detail.TemplateKey) == "" {
		next.Detail.TemplateKey = "publish_detail_generate"
	}
	if strings.TrimSpace(next.Title.TemplateKey) == "" {
		next.Title.TemplateKey = "publish_title_generate"
	}
	if strings.TrimSpace(next.Combined.TemplateKey) == "" {
		next.Combined.TemplateKey = "publish_title_detail_generate"
	}
	next.ImageSelectionMode = normalizeAIImageSelectionMode(next.ImageSelectionMode)
	next.TitleMaxChars = boundedInt(next.TitleMaxChars, 8, 80, 40)
	return next
}

func publishGenerationTargetFromAny(value any, current AIPublishGenerationTargetConfig, defaultTemplate string) AIPublishGenerationTargetConfig {
	raw := jsonMapFromSetting(value)
	if raw == nil {
		return current
	}
	next := current
	if value, ok := firstPresent(raw, "enabled"); ok {
		if parsed, valid := boolSettingInput(value); valid {
			next.Enabled = parsed
		}
	}
	if value, ok := firstPresent(raw, "templateKey", "template_key"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			next.TemplateKey = text
		}
	}
	if strings.TrimSpace(next.TemplateKey) == "" {
		next.TemplateKey = defaultTemplate
	}
	return next
}
