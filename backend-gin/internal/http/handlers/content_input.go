package handlers

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/repositories"
)

func currentUserID(c *gin.Context) int64 {
	if user, ok := currentUser(c); ok {
		return user.ID
	}
	return 0
}

func (h NativeHandlers) createPostInputFromMap(raw map[string]any, userID int64) repositories.CreatePostInput {
	title, _ := raw["title"].(string)
	content, _ := raw["content"].(string)
	postType, ok := intFromAny(raw["type"])
	if !ok {
		postType = repositories.PostTypeImage
	}
	isDraft := false
	if parsed, ok := boolFromAny(raw["is_draft"]); ok {
		isDraft = parsed
	}
	visibility, _ := raw["visibility"].(string)
	return repositories.CreatePostInput{
		UserID:          userID,
		Title:           sanitizePlainSubmittedText(title),
		Content:         content,
		CategoryID:      optionalIntFromAny(raw["category_id"]),
		Type:            postType,
		Images:          h.imageInputs(raw["images"]),
		Video:           h.videoInput(raw["video"]),
		Attachment:      h.attachmentInput(raw["attachment"]),
		Tags:            stringSliceFromAny(raw["tags"]),
		IsDraft:         isDraft,
		PaymentSettings: paymentInput(raw["paymentSettings"]),
		Visibility:      visibility,
	}
}

func (h NativeHandlers) updatePostInputFromMap(raw map[string]any, userID, postID int64) repositories.UpdatePostInput {
	input := repositories.UpdatePostInput{UserID: userID, PostID: postID}
	if value, ok := raw["title"].(string); ok {
		sanitized := sanitizePlainSubmittedText(value)
		input.Title = &sanitized
	}
	if value, ok := raw["content"].(string); ok {
		input.Content = &value
	}
	if value, exists := raw["category_id"]; exists {
		input.CategoryIDSet = true
		input.CategoryID = optionalIntFromAny(value)
	}
	if value, ok := intFromAny(raw["type"]); ok {
		input.Type = &value
	}
	if value, ok := boolFromAny(raw["is_draft"]); ok {
		input.IsDraft = &value
	}
	if value, ok := raw["visibility"].(string); ok {
		input.Visibility = &value
	}
	if value, exists := raw["images"]; exists {
		input.ImagesSet = true
		input.Images = h.imageInputs(value)
	}
	if value, exists := raw["video"]; exists {
		input.VideoSet = true
		input.Video = h.videoInput(value)
	}
	if value, exists := raw["attachment"]; exists {
		input.AttachmentSet = true
		input.Attachment = h.attachmentInput(value)
	}
	if value, exists := raw["tags"]; exists {
		input.TagsSet = true
		input.Tags = stringSliceFromAny(value)
	}
	if value, exists := raw["paymentSettings"]; exists {
		input.PaymentSet = true
		input.PaymentSettings = paymentInput(value)
	}
	return input
}

func (h NativeHandlers) imageInputs(value any) []repositories.PostImageInput {
	values, ok := value.([]any)
	if !ok {
		return []repositories.PostImageInput{}
	}
	out := make([]repositories.PostImageInput, 0, len(values))
	for idx, item := range values {
		switch typed := item.(type) {
		case string:
			out = append(out, repositories.PostImageInput{URL: h.normalizeFileURLForStorage(typed), IsFreePreview: true, SortOrder: idx + 1})
		case map[string]any:
			urlValue, _ := typed["url"].(string)
			if urlValue == "" {
				urlValue, _ = typed["image_url"].(string)
			}
			urlValue = h.normalizeFileURLForStorage(urlValue)
			isFree := true
			if parsed, ok := boolFromAny(typed["isFreePreview"]); ok {
				isFree = parsed
			} else if parsed, ok := boolFromAny(typed["is_free_preview"]); ok {
				isFree = parsed
			}
			isProtected := false
			if parsed, ok := boolFromAny(firstPresent(typed, "isProtected", "is_protected")); ok {
				isProtected = parsed
			}
			sortOrder, ok := intFromAny(firstPresent(typed, "sortOrder", "sort_order"))
			if !ok || sortOrder <= 0 {
				sortOrder = idx + 1
			}
			traceToken := strings.ToLower(strings.TrimSpace(toString(firstPresent(typed, "watermarkTraceToken", "watermark_trace_token"))))
			out = append(out, repositories.PostImageInput{URL: urlValue, WatermarkTraceToken: traceToken, IsFreePreview: isFree, IsProtected: isProtected, SortOrder: sortOrder})
		}
	}
	return out
}

func imageInputCount(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []map[string]any:
		return len(typed)
	case []string:
		return len(typed)
	default:
		return 0
	}
}

func (h NativeHandlers) videoInput(value any) *repositories.PostVideoInput {
	m, ok := value.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	urlValue, _ := m["url"].(string)
	if urlValue == "" {
		urlValue, _ = m["video_url"].(string)
	}
	if urlValue == "" {
		return nil
	}
	urlValue = h.normalizeFileURLForStorage(urlValue)
	var cover *string
	if raw, ok := m["coverUrl"].(string); ok && raw != "" {
		cover = stringPtr(h.normalizeFileURLForStorage(raw))
	} else if raw, ok := m["cover_url"].(string); ok && raw != "" {
		cover = stringPtr(h.normalizeFileURLForStorage(raw))
	}
	return &repositories.PostVideoInput{URL: urlValue, CoverURL: cover}
}

func (h NativeHandlers) attachmentInput(value any) *repositories.PostAttachmentInput {
	m, ok := value.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	urlValue, _ := m["url"].(string)
	if urlValue == "" {
		urlValue, _ = m["attachment_url"].(string)
	}
	if urlValue == "" {
		return nil
	}
	urlValue = h.normalizeFileURLForStorage(urlValue)
	filename, _ := m["filename"].(string)
	filesize, _ := int64FromAny(m["filesize"])
	return &repositories.PostAttachmentInput{URL: urlValue, Filename: filename, Filesize: filesize}
}

func paymentInput(value any) *repositories.PaymentSettingsInput {
	m, ok := value.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	enabled, _ := boolFromAny(m["enabled"])
	if !enabled {
		return &repositories.PaymentSettingsInput{Enabled: false}
	}
	paymentType, _ := m["paymentType"].(string)
	if paymentType == "" {
		paymentType, _ = m["payment_type"].(string)
	}
	paymentMethod, _ := m["paymentMethod"].(string)
	if paymentMethod == "" {
		paymentMethod, _ = m["payment_method"].(string)
	}
	paymentMethod = strings.ToLower(strings.TrimSpace(paymentMethod))
	if paymentMethod == "" {
		paymentMethod = "balance"
	}
	price, _ := float64FromAny(m["price"])
	freePreview, _ := intFromAny(firstPresent(m, "freePreviewCount", "free_preview_count"))
	previewDuration, _ := intFromAny(firstPresent(m, "previewDuration", "preview_duration"))
	hideAll, _ := boolFromAny(firstPresent(m, "hideAll", "hide_all"))
	return &repositories.PaymentSettingsInput{Enabled: true, PaymentType: paymentType, PaymentMethod: paymentMethod, Price: price, FreePreviewCount: freePreview, PreviewDuration: previewDuration, HideAll: hideAll}
}

func firstPresent(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func optionalIntFromAny(value any) *int {
	parsed, ok := intFromAny(value)
	if !ok || parsed == 0 {
		return nil
	}
	return &parsed
}

func stringSliceFromAny(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			if sanitized := sanitizePlainSubmittedText(text); sanitized != "" {
				out = append(out, sanitized)
			}
		}
	}
	return out
}

func jsonRawOrNil(data []byte) any {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var value any
	if json.Unmarshal(data, &value) == nil {
		return value
	}
	return nil
}
