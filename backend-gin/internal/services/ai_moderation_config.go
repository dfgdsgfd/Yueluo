package services

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

const (
	AIModerationTargetComment = "comment"
	AIModerationTargetPost    = "post"

	AIModerationActionObserve = "observe"
	AIModerationActionDelete  = "delete"
	AIModerationActionPrivate = "private"
)

func defaultAIModerationConfig() AIModerationConfig {
	return AIModerationConfig{
		Comment: AIModerationTargetConfig{
			Enabled:     false,
			TemplateKey: "comment_moderation",
			Rules: map[string]AIModerationRuleConfig{
				"spam":                {Enabled: true, Action: AIModerationActionObserve, Sensitivity: 0.65},
				"porn":                {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.65},
				"political_sensitive": {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.65},
			},
		},
		Post: AIModerationTargetConfig{
			Enabled:     false,
			TemplateKey: "post_moderation",
			Rules: map[string]AIModerationRuleConfig{
				"spam":                {Enabled: true, Action: AIModerationActionObserve, Sensitivity: 0.65},
				"porn":                {Enabled: true, Action: AIModerationActionPrivate, Sensitivity: 0.65},
				"political_sensitive": {Enabled: true, Action: AIModerationActionPrivate, Sensitivity: 0.65},
			},
		},
	}
}

func moderationConfigFromSettings(settings *SettingsService, defaults AIModerationConfig) AIModerationConfig {
	if settings == nil {
		return defaults
	}
	return moderationConfigFromAny(settings.Get(AISettingModeration), defaults)
}

func normalizeModerationSetting(value any, current AIModerationConfig) (AIModerationConfig, bool) {
	return moderationConfigFromAny(value, current), value != nil
}

func moderationConfigFromAny(value any, current AIModerationConfig) AIModerationConfig {
	if value == nil {
		return current
	}
	var raw map[string]any
	switch typed := value.(type) {
	case map[string]any:
		raw = typed
	case string:
		if strings.TrimSpace(typed) == "" {
			return current
		}
		_ = json.Unmarshal([]byte(typed), &raw)
	default:
		data, err := json.Marshal(value)
		if err == nil {
			_ = json.Unmarshal(data, &raw)
		}
	}
	if raw == nil {
		return current
	}
	next := current
	if value, ok := firstPresent(raw, "comment"); ok {
		next.Comment = moderationTargetConfigFromAny(value, current.Comment, AIModerationTargetComment)
	}
	if value, ok := firstPresent(raw, "post"); ok {
		next.Post = moderationTargetConfigFromAny(value, current.Post, AIModerationTargetPost)
	}
	return next
}

func moderationTargetConfigFromAny(value any, current AIModerationTargetConfig, target string) AIModerationTargetConfig {
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
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
			next.TemplateKey = text
		}
	}
	if value, ok := firstPresent(raw, "prompt"); ok {
		next.Prompt = strings.TrimSpace(fmt.Sprint(value))
	}
	if value, ok := firstPresent(raw, "rules"); ok {
		next.Rules = moderationRulesFromAny(value, next.Rules, target)
	}
	if next.TemplateKey == "" {
		if target == AIModerationTargetPost {
			next.TemplateKey = "post_moderation"
		} else {
			next.TemplateKey = "comment_moderation"
		}
	}
	return next
}

func moderationRulesFromAny(value any, current map[string]AIModerationRuleConfig, target string) map[string]AIModerationRuleConfig {
	out := map[string]AIModerationRuleConfig{}
	maps.Copy(out, current)
	raw := jsonMapFromSetting(value)
	for key, value := range raw {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		rule := out[key]
		ruleRaw := jsonMapFromSetting(value)
		if parsed, valid := boolSettingInput(ruleRaw["enabled"]); valid {
			rule.Enabled = parsed
		}
		if action := normalizeModerationAction(fmt.Sprint(ruleRaw["action"]), target); action != "" {
			rule.Action = action
		}
		if parsed, valid := floatSettingInput(ruleRaw["sensitivity"]); valid {
			rule.Sensitivity = boundedFloat(parsed, 0, 1, rule.Sensitivity)
		}
		if rule.Action == "" {
			rule.Action = AIModerationActionObserve
		}
		out[key] = rule
	}
	return out
}

func normalizeModerationAction(value string, target string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AIModerationActionObserve, "allow", "approve", "approved", "pass", "safe", "ok":
		return AIModerationActionObserve
	case AIModerationActionDelete, "remove", "reject", "rejected", "block", "blocked", "hide":
		return AIModerationActionDelete
	case AIModerationActionPrivate:
		if target == AIModerationTargetPost {
			return AIModerationActionPrivate
		}
	}
	return ""
}

func (cfg AIModerationConfig) Target(target string) AIModerationTargetConfig {
	if target == AIModerationTargetPost {
		return cfg.Post
	}
	return cfg.Comment
}
