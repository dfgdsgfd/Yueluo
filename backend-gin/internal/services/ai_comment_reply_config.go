package services

import (
	"fmt"
	"strings"
)

func defaultAICommentReplyConfig() AICommentReplyConfig {
	return AICommentReplyConfig{
		Enabled:                  false,
		TemplateKey:              "comment_reply",
		DelaySeconds:             10,
		MaxImages:                4,
		ImageSelectionMode:       AIImageSelectionOrdered,
		Style:                    "normal",
		MaxRepliesPerAIComment:   3,
		MentionEnabled:           false,
		MentionName:              "yueai",
		MentionTemplateKey:       "comment_mention_reply",
		MentionBotUserIDMin:      0,
		MentionBotUserIDMax:      0,
		MaxMentionRepliesPerPost: 3,
	}
}

func commentReplyConfigFromSettings(settings *SettingsService, defaults AICommentReplyConfig) AICommentReplyConfig {
	if settings == nil {
		return defaults
	}
	return commentReplyConfigFromAny(settings.Get(AISettingCommentReply), defaults)
}

func normalizeCommentReplySetting(value any, current AICommentReplyConfig) (AICommentReplyConfig, bool) {
	if value == nil {
		return current, false
	}
	return commentReplyConfigFromAny(value, current), true
}

func commentReplyConfigFromAny(value any, current AICommentReplyConfig) AICommentReplyConfig {
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
	if value, ok := firstPresent(raw, "delaySeconds", "delay_seconds"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.DelaySeconds = boundedInt(parsed, 0, 3600, next.DelaySeconds)
		}
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
	if value, ok := firstPresent(raw, "style"); ok {
		if style := normalizeAIPromptStyle(fmt.Sprint(value), ""); style != "" {
			next.Style = style
		}
	}
	if value, ok := firstPresent(raw, "maxRepliesPerAIComment", "max_replies_per_ai_comment", "maxReplies", "max_replies"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.MaxRepliesPerAIComment = boundedInt(parsed, 1, 20, next.MaxRepliesPerAIComment)
		}
	}
	if value, ok := firstPresent(raw, "mentionEnabled", "mention_enabled"); ok {
		if parsed, valid := boolSettingInput(value); valid {
			next.MentionEnabled = parsed
		}
	}
	if value, ok := firstPresent(raw, "mentionName", "mention_name"); ok {
		if name := normalizeAICommentMentionName(fmt.Sprint(value)); name != "" {
			next.MentionName = name
		}
	}
	if value, ok := firstPresent(raw, "mentionTemplateKey", "mention_template_key"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			next.MentionTemplateKey = text
		}
	}
	if value, ok := firstPresent(raw, "mentionBotUserIdMin", "mention_bot_user_id_min", "mentionBotUserIDMin"); ok {
		if parsed, valid := int64SettingInput(value); valid && parsed >= 0 {
			next.MentionBotUserIDMin = parsed
		}
	}
	if value, ok := firstPresent(raw, "mentionBotUserIdMax", "mention_bot_user_id_max", "mentionBotUserIDMax"); ok {
		if parsed, valid := int64SettingInput(value); valid && parsed >= 0 {
			next.MentionBotUserIDMax = parsed
		}
	}
	if value, ok := firstPresent(raw, "maxMentionRepliesPerPost", "max_mention_replies_per_post", "maxMentionReplies", "max_mention_replies"); ok {
		if parsed, valid := intSettingInput(value); valid {
			next.MaxMentionRepliesPerPost = boundedInt(parsed, 1, 20, next.MaxMentionRepliesPerPost)
		}
	}
	if strings.TrimSpace(next.TemplateKey) == "" {
		next.TemplateKey = "comment_reply"
	}
	if strings.TrimSpace(next.MentionTemplateKey) == "" {
		next.MentionTemplateKey = "comment_mention_reply"
	}
	if next.MentionBotUserIDMin > next.MentionBotUserIDMax {
		next.MentionBotUserIDMin, next.MentionBotUserIDMax = next.MentionBotUserIDMax, next.MentionBotUserIDMin
	}
	next.DelaySeconds = boundedInt(next.DelaySeconds, 0, 3600, 10)
	next.MaxImages = boundedInt(next.MaxImages, 0, 12, 4)
	next.ImageSelectionMode = normalizeAIImageSelectionMode(next.ImageSelectionMode)
	next.Style = normalizeAIPromptStyle(next.Style, "normal")
	next.MaxRepliesPerAIComment = boundedInt(next.MaxRepliesPerAIComment, 1, 20, 3)
	next.MentionName = nonEmptyString(normalizeAICommentMentionName(next.MentionName), "yueai")
	next.MaxMentionRepliesPerPost = boundedInt(next.MaxMentionRepliesPerPost, 1, 20, 3)
	return next
}

func normalizeAICommentMentionName(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "@"))
}
