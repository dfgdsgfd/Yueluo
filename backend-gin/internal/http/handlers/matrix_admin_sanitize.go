package handlers

import "strings"

func (h NativeHandlers) adminSanitizeBody(resource adminResource, body map[string]any) map[string]any {
	out := map[string]any{}
	allowed := adminWritableColumns(resource)
	for key, value := range body {
		if !isSafeColumn(key) || key == "id" {
			continue
		}
		if allowed != nil && !allowed[key] {
			continue
		}
		if resource.Table == "posts" && key == "quality_level" && strings.TrimSpace(toString(value)) == "" {
			out[key] = "none"
			continue
		}
		if strings.HasSuffix(key, "_at") {
			if parsed := parseTimeAny(value); parsed != nil {
				out[key] = *parsed
				continue
			}
		}
		if text, ok := value.(string); ok && adminMediaURLColumn(key) {
			column := key
			if resource.Table == "users" && (key == profileImageAvatarColumn || key == profileImageBackgroundColumn) {
				var hashValue any
				text, hashValue = h.profileImageStorageValue(text, column)
				out[profileImageHashColumnForStorageColumn(column)] = hashValue
			} else {
				text = h.normalizeFileURLForStorage(text)
			}
			out[key] = text
			continue
		}
		if text, ok := value.(string); ok && adminRichTextColumn(resource, key) {
			out[key] = sanitizePostContent(text)
			continue
		}
		if text, ok := value.(string); ok && adminPlainTextColumn(key) {
			out[key] = sanitizePlainSubmittedText(text)
			continue
		}
		out[key] = value
	}
	return out
}

func adminRichTextColumn(resource adminResource, key string) bool {
	if key == "content" {
		switch resource.Name {
		case "announcements", "audit", "comments", "content-review", "feedback", "notification-templates", "posts", "system-notifications":
			return true
		}
	}
	switch resource.Name {
	case "feedback":
		return key == "admin_reply"
	case "reports":
		return key == "admin_note" || key == "description"
	case "users":
		return key == "bio"
	case "audit", "content-review":
		return key == "reason"
	default:
		return false
	}
}

func adminPlainTextColumn(key string) bool {
	switch key {
	case "app_name",
		"category_title",
		"description",
		"education",
		"gender",
		"location",
		"major",
		"mbti",
		"name",
		"nickname",
		"platform",
		"quality_level",
		"reason",
		"remark",
		"status",
		"target_type",
		"template_key",
		"title",
		"type",
		"url",
		"user_id",
		"username",
		"version_name",
		"visibility",
		"word",
		"zodiac_sign":
		return true
	default:
		return false
	}
}

func adminMediaURLColumn(key string) bool {
	return key == "avatar" ||
		key == "background" ||
		strings.HasSuffix(key, "_url") ||
		strings.HasSuffix(key, "_cover") ||
		strings.HasSuffix(key, "_image")
}
