package services

import (
	"fmt"
	"strings"
)

func isAIContentFormatTask(taskType string) bool {
	switch strings.TrimSpace(taskType) {
	case AITaskFormatMarkdown, AITaskPostPolish, AITaskPostCustomGenerate:
		return true
	default:
		return false
	}
}

func defaultAIContentFormatConfig() AIContentFormatConfig {
	return AIContentFormatConfig{
		Enabled: true,
		Format: AIContentFormatTargetConfig{
			Enabled:     true,
			TemplateKey: "markdown_format",
		},
		Polish: AIContentFormatTargetConfig{
			Enabled:     true,
			TemplateKey: "post_polish",
		},
		Custom: AIContentFormatTargetConfig{
			Enabled:     true,
			TemplateKey: "post_custom_generate",
			Continuation: AIContentContinuationConfig{
				Enabled:      true,
				TriggerChars: 6000,
				MaxRounds:    2,
				ContextChars: 2400,
			},
		},
	}
}

func contentFormatConfigFromSettings(settings *SettingsService, defaults AIContentFormatConfig) AIContentFormatConfig {
	if settings == nil {
		return defaults
	}
	return contentFormatConfigFromAny(settings.Get(AISettingContentFormat), defaults)
}

func normalizeContentFormatSetting(value any, current AIContentFormatConfig) (AIContentFormatConfig, bool) {
	if value == nil {
		return current, false
	}
	return contentFormatConfigFromAny(value, current), true
}

func contentFormatConfigFromAny(value any, current AIContentFormatConfig) AIContentFormatConfig {
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
	if value, ok := firstPresent(raw, "format", "markdown", "markdownFormat", "markdown_format"); ok {
		next.Format = contentFormatTargetFromAny(value, next.Format, "markdown_format")
	}
	if value, ok := firstPresent(raw, "polish", "postPolish", "post_polish"); ok {
		next.Polish = contentFormatTargetFromAny(value, next.Polish, "post_polish")
	}
	if value, ok := firstPresent(raw, "custom", "generate", "customGenerate", "post_custom_generate"); ok {
		next.Custom = contentFormatTargetFromAny(value, next.Custom, "post_custom_generate")
	}
	if strings.TrimSpace(next.Format.TemplateKey) == "" {
		next.Format.TemplateKey = "markdown_format"
	}
	if strings.TrimSpace(next.Polish.TemplateKey) == "" {
		next.Polish.TemplateKey = "post_polish"
	}
	if strings.TrimSpace(next.Custom.TemplateKey) == "" {
		next.Custom.TemplateKey = "post_custom_generate"
	}
	next.Custom.Continuation = normalizeAIContentContinuation(next.Custom.Continuation, defaultAIContentContinuationConfig())
	next.Format.Continuation = AIContentContinuationConfig{}
	next.Polish.Continuation = AIContentContinuationConfig{}
	return next
}

func contentFormatTargetFromAny(value any, current AIContentFormatTargetConfig, defaultTemplate string) AIContentFormatTargetConfig {
	raw := jsonMapFromSetting(value)
	if raw == nil {
		switch typed := value.(type) {
		case string:
			text := strings.TrimSpace(typed)
			if text != "" {
				current.TemplateKey = text
			}
		default:
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				current.TemplateKey = text
			}
		}
		if strings.TrimSpace(current.TemplateKey) == "" {
			current.TemplateKey = defaultTemplate
		}
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
	if value, ok := firstPresent(raw, "continuation", "longGeneration", "long_generation"); ok {
		next.Continuation = contentContinuationFromAny(value, next.Continuation)
	}
	if strings.TrimSpace(next.TemplateKey) == "" {
		next.TemplateKey = defaultTemplate
	}
	return next
}

func defaultAIContentContinuationConfig() AIContentContinuationConfig {
	return AIContentContinuationConfig{
		Enabled:      true,
		TriggerChars: 6000,
		MaxRounds:    2,
		ContextChars: 2400,
	}
}

func contentContinuationFromAny(value any, current AIContentContinuationConfig) AIContentContinuationConfig {
	raw := jsonMapFromSetting(value)
	if raw == nil {
		if parsed, valid := boolSettingInput(value); valid {
			current.Enabled = parsed
		}
		return normalizeAIContentContinuation(current, defaultAIContentContinuationConfig())
	}
	next := current
	if value, ok := firstPresent(raw, "enabled"); ok {
		if parsed, valid := boolSettingInput(value); valid {
			next.Enabled = parsed
		}
	}
	if value, ok := firstPresent(raw, "triggerChars", "trigger_chars", "trigger", "minChars", "min_chars"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.TriggerChars = parsed
		}
	}
	if value, ok := firstPresent(raw, "maxRounds", "max_rounds", "rounds"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.MaxRounds = parsed
		}
	}
	if value, ok := firstPresent(raw, "contextChars", "context_chars", "summaryChars", "summary_chars"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.ContextChars = parsed
		}
	}
	return normalizeAIContentContinuation(next, defaultAIContentContinuationConfig())
}

func normalizeAIContentContinuation(cfg AIContentContinuationConfig, defaults AIContentContinuationConfig) AIContentContinuationConfig {
	if cfg.TriggerChars == 0 {
		cfg.TriggerChars = defaults.TriggerChars
	}
	if cfg.MaxRounds == 0 {
		cfg.MaxRounds = defaults.MaxRounds
	}
	if cfg.ContextChars == 0 {
		cfg.ContextChars = defaults.ContextChars
	}
	cfg.TriggerChars = boundedInt(cfg.TriggerChars, 1000, 100000, defaults.TriggerChars)
	cfg.MaxRounds = boundedInt(cfg.MaxRounds, 1, 8, defaults.MaxRounds)
	cfg.ContextChars = boundedInt(cfg.ContextChars, 600, 20000, defaults.ContextChars)
	return cfg
}

func contentFormatTargetForTask(cfg AIContentFormatConfig, taskType string) (AIContentFormatTargetConfig, bool) {
	switch strings.TrimSpace(taskType) {
	case AITaskFormatMarkdown:
		return cfg.Format, true
	case AITaskPostPolish:
		return cfg.Polish, true
	case AITaskPostCustomGenerate:
		return cfg.Custom, true
	default:
		return AIContentFormatTargetConfig{}, false
	}
}
