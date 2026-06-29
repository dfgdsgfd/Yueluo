package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type AIPublishGenerationInput struct {
	Locale     string
	Title      string
	Detail     string
	Tags       []string
	NeedTitle  bool
	NeedDetail bool
	Images     []AIImageInput
}

type AIPublishGenerationResult struct {
	Enabled               bool                `json:"enabled"`
	Title                 string              `json:"title,omitempty"`
	Detail                string              `json:"detail,omitempty"`
	GeneratedTitle        bool                `json:"generatedTitle"`
	GeneratedDetail       bool                `json:"generatedDetail"`
	Skipped               map[string]string   `json:"skipped,omitempty"`
	MaxImages             int                 `json:"maxImages"`
	ImageSelectionMode    string              `json:"imageSelectionMode"`
	TitleMaxChars         int                 `json:"titleMaxChars"`
	ImageSendSuccessCount int                 `json:"imageSendSuccessCount"`
	Usage                 map[string]*AIUsage `json:"usage,omitempty"`
}

func (s *AIService) PublicPublishGenerationConfig() AIPublishGenerationConfig {
	if s == nil {
		return defaultAIPublishGenerationConfig()
	}
	cfg := s.Config()
	publish := cfg.PublishGeneration
	publish.Enabled = cfg.Enabled && publish.Enabled
	return publish
}

func (s *AIService) GeneratePublishContent(ctx context.Context, input AIPublishGenerationInput, actor AIActor) (AIPublishGenerationResult, error) {
	var out AIPublishGenerationResult
	if s == nil {
		return out, AIError{Code: "error.ai_settings_unavailable"}
	}
	cfg := s.Config()
	publish := cfg.PublishGeneration
	out.Enabled = cfg.Enabled && publish.Enabled
	out.MaxImages = publish.MaxImages
	out.ImageSelectionMode = normalizeAIImageSelectionMode(publish.ImageSelectionMode)
	out.TitleMaxChars = boundedInt(publish.TitleMaxChars, 8, 80, 40)
	out.Skipped = map[string]string{}
	out.Usage = map[string]*AIUsage{}
	if !out.Enabled {
		out.Skipped["all"] = "publish_generation_disabled"
		return out, nil
	}
	images := normalizeAIImageInputs(input.Images)
	images = selectAIImageSample(images, publish.MaxImages, publish.ImageSelectionMode)
	out.ImageSendSuccessCount = len(images)
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = "auto"
	}
	if input.NeedDetail && publish.Detail.Enabled {
		text, usage, err := s.RunText(ctx, AIRequest{
			Type:        AITaskPublishDetailGenerate,
			Locale:      locale,
			Input:       publishGenerationDetailPromptInput(input, len(images), publish.ImageSelectionMode),
			TemplateKey: publish.Detail.TemplateKey,
			Variables:   publishGenerationDetailVariables(input, len(images), publish.ImageSelectionMode),
			Images:      images,
		}, actor)
		if err != nil {
			return out, err
		}
		out.Detail = parsePublishGenerationField(text, "detail")
		out.GeneratedDetail = strings.TrimSpace(out.Detail) != ""
		if usage != nil {
			out.Usage["detail"] = usage
		}
	} else if input.NeedDetail {
		out.Skipped["detail"] = "detail_generation_disabled"
	}

	titleSource := out.Detail
	if strings.TrimSpace(titleSource) == "" {
		titleSource = input.Detail
	}
	if input.NeedTitle && publish.Title.Enabled && strings.TrimSpace(titleSource) != "" {
		titleInput := input
		titleInput.Detail = titleSource
		text, usage, err := s.RunText(ctx, AIRequest{
			Type:        AITaskPublishTitleGenerate,
			Locale:      locale,
			Input:       publishGenerationTitlePromptInput(titleInput, publish.TitleMaxChars),
			TemplateKey: publish.Title.TemplateKey,
			Variables:   publishGenerationTitleVariables(titleInput, publish.TitleMaxChars),
			Images:      nil,
		}, actor)
		if err != nil {
			return out, err
		}
		out.Title = limitPublishGenerationTitle(parsePublishGenerationField(text, "title"), publish.TitleMaxChars)
		out.GeneratedTitle = strings.TrimSpace(out.Title) != ""
		if usage != nil {
			out.Usage["title"] = usage
		}
	} else if input.NeedTitle {
		out.Skipped["title"] = "title_generation_disabled"
	}
	if len(out.Skipped) == 0 {
		out.Skipped = nil
	}
	if len(out.Usage) == 0 {
		out.Usage = nil
	}
	return out, nil
}

func parsePublishGenerationTitleDetail(raw string) (string, string) {
	text := normalizePublishGenerationOutput(raw)
	if text == "" {
		return "", ""
	}
	if title, detail, ok := parsePublishGenerationJSONTitleDetail(text); ok {
		return title, detail
	}
	return parsePublishGenerationTextTitleDetail(text)
}

func parsePublishGenerationTextTitleDetail(text string) (string, string) {
	text = normalizePublishGenerationOutput(text)
	if text == "" {
		return "", ""
	}
	title := parsePublishGenerationLabeledField(text, "title")
	detail := parsePublishGenerationLabeledField(text, "detail")
	if title != "" || detail != "" {
		return title, detail
	}
	return parsePublishGenerationPlainTitleDetail(text)
}

func publishGenerationVariables(input AIPublishGenerationInput, imageCount int) map[string]any {
	return map[string]any{
		"existingTitle":         strings.TrimSpace(input.Title),
		"existingDetail":        strings.TrimSpace(input.Detail),
		"imageSendSuccessCount": imageCount,
	}
}

func publishGenerationPromptInput(input AIPublishGenerationInput, imageCount int) string {
	return publishGenerationDetailPromptInput(input, imageCount, AIImageSelectionOrdered)
}

func publishGenerationDetailVariables(input AIPublishGenerationInput, imageCount int, imageSelectionMode string) map[string]any {
	variables := publishGenerationVariables(input, imageCount)
	variables["imageSelectionMode"] = normalizeAIImageSelectionMode(imageSelectionMode)
	return variables
}

func publishGenerationTitleVariables(input AIPublishGenerationInput, titleMaxChars int) map[string]any {
	return map[string]any{
		"existingTitle": strings.TrimSpace(input.Title),
		"detail":        strings.TrimSpace(input.Detail),
		"titleMaxChars": boundedInt(titleMaxChars, 8, 80, 40),
	}
}

func publishGenerationDetailPromptInput(input AIPublishGenerationInput, imageCount int, imageSelectionMode string) string {
	lines := []string{
		"已有标题：\n" + nonEmptyString(strings.TrimSpace(input.Title), "（空）"),
		"已有详情正文：\n" + nonEmptyString(strings.TrimSpace(input.Detail), "（空）"),
	}
	if len(input.Tags) > 0 {
		tags := make([]string, 0, len(input.Tags))
		for _, tag := range input.Tags {
			if text := strings.TrimSpace(tag); text != "" {
				tags = append(tags, text)
			}
		}
		if len(tags) > 0 {
			lines = append(lines, "标签/话题："+strings.Join(tags, ", "))
		}
	}
	lines = append(lines, "可用于分析的图片数量："+strconv.Itoa(imageCount))
	lines = append(lines, "图片取样模式："+normalizeAIImageSelectionMode(imageSelectionMode))
	return strings.Join(lines, "\n\n")
}

func publishGenerationTitlePromptInput(input AIPublishGenerationInput, titleMaxChars int) string {
	titleMaxChars = boundedInt(titleMaxChars, 8, 80, 40)
	lines := []string{
		"已有标题：\n" + nonEmptyString(strings.TrimSpace(input.Title), "（空）"),
		"最终详情正文：\n" + nonEmptyString(strings.TrimSpace(input.Detail), "（空）"),
		"标题字数上限：" + strconv.Itoa(titleMaxChars),
	}
	if len(input.Tags) > 0 {
		tags := make([]string, 0, len(input.Tags))
		for _, tag := range input.Tags {
			if text := strings.TrimSpace(tag); text != "" {
				tags = append(tags, text)
			}
		}
		if len(tags) > 0 {
			lines = append(lines, "标签/话题："+strings.Join(tags, ", "))
		}
	}
	return strings.Join(lines, "\n\n")
}

func limitPublishGenerationTitle(value string, maxChars int) string {
	text := strings.TrimSpace(value)
	maxChars = boundedInt(maxChars, 8, 80, 40)
	if len([]rune(text)) <= maxChars {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:maxChars]))
}

func parsePublishGenerationField(raw string, primary string) string {
	text := normalizePublishGenerationOutput(raw)
	if text == "" {
		return ""
	}
	if title, detail, ok := parsePublishGenerationJSONTitleDetail(text); ok {
		if primary == "title" {
			return title
		}
		return detail
	}
	if value := parsePublishGenerationLabeledField(text, primary); value != "" {
		return value
	}
	if hasPublishGenerationLabeledOutput(text) {
		return ""
	}
	return text
}

func normalizePublishGenerationOutput(raw string) string {
	text := strings.TrimSpace(extractJSONText(raw))
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.TrimSpace(text)
	if text == "" || isPublishGenerationPromptEcho(text) {
		return ""
	}
	return text
}

func parsePublishGenerationJSONTitleDetail(text string) (string, string, bool) {
	var payload any
	if json.Unmarshal([]byte(extractJSONText(text)), &payload) != nil {
		return "", "", false
	}
	title := firstPublishGenerationJSONField(payload, publishGenerationFieldAliases("title"))
	detail := firstPublishGenerationJSONField(payload, publishGenerationFieldAliases("detail"))
	if isPublishGenerationPromptEcho(title) {
		title = ""
	}
	if isPublishGenerationPromptEcho(detail) {
		detail = ""
	}
	if title == "" && detail == "" {
		if text := firstPublishGenerationJSONTextPayload(payload); text != "" {
			title, detail = parsePublishGenerationTextTitleDetail(text)
		}
	} else if detail == "" {
		if text := firstPublishGenerationJSONTextPayload(payload); text != "" {
			_, parsedDetail := parsePublishGenerationTextTitleDetail(text)
			if parsedDetail != "" {
				detail = parsedDetail
			} else if parsedTitle, _ := parsePublishGenerationTextTitleDetail(text); parsedTitle != "" && parsedTitle != title {
				detail = parsedTitle
			}
		}
	}
	return title, detail, true
}

func firstPublishGenerationJSONField(value any, keys []string) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if text := strings.TrimSpace(firstStringValue(typed[key])); text != "" {
				return text
			}
		}
		for _, wrapper := range []string{"data", "result", "output", "message"} {
			if text := firstPublishGenerationJSONField(typed[wrapper], keys); text != "" {
				return text
			}
		}
	case []any:
		for _, item := range typed {
			if text := firstPublishGenerationJSONField(item, keys); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstPublishGenerationJSONTextPayload(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, wrapper := range []string{"data", "result", "output", "message", "response", "payload", "content", "text"} {
			if text := strings.TrimSpace(firstStringValue(typed[wrapper])); text != "" {
				return text
			}
			if text := firstPublishGenerationJSONTextPayload(typed[wrapper]); text != "" {
				return text
			}
		}
	case []any:
		for _, item := range typed {
			if text := firstPublishGenerationJSONTextPayload(item); text != "" {
				return text
			}
		}
	case string:
		return strings.TrimSpace(typed)
	}
	return ""
}

func publishGenerationFieldAliases(primary string) []string {
	if primary == "title" {
		return []string{"title", "postTitle", "post_title", "headline", "name"}
	}
	return []string{"detail", "details", "body", "description", "postDetail", "post_detail"}
}

func parsePublishGenerationLabeledField(text string, primary string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	labels := publishGenerationFieldLabels(primary)
	bestStart := -1
	bestEnd := -1
	for _, label := range labels {
		for _, marker := range []string{label + "：", label + ":"} {
			if index := strings.Index(normalized, marker); index >= 0 && (bestStart < 0 || index < bestStart) {
				bestStart = index
				bestEnd = index + len(marker)
			}
		}
	}
	if bestStart < 0 {
		return ""
	}
	value := strings.TrimSpace(normalized[bestEnd:])
	for _, label := range publishGenerationOtherFieldLabels(primary) {
		for _, marker := range []string{"\n" + label + "：", "\n" + label + ":"} {
			if index := strings.Index(value, marker); index >= 0 {
				value = value[:index]
			}
		}
	}
	value = strings.TrimSpace(value)
	if isPublishGenerationPromptEcho(value) {
		return ""
	}
	return strings.Trim(value, "\"'` \t\r\n")
}

func publishGenerationFieldLabels(primary string) []string {
	if primary == "title" {
		return []string{"标题", "Title", "title"}
	}
	return []string{"详情", "正文", "Detail", "Details", "Body", "detail", "details", "body"}
}

func publishGenerationOtherFieldLabels(primary string) []string {
	if primary == "title" {
		return publishGenerationFieldLabels("detail")
	}
	return publishGenerationFieldLabels("title")
}

func parsePublishGenerationPlainTitleDetail(text string) (string, string) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	meaningful := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.Trim(line, "\"'` \t"))
		if line != "" {
			meaningful = append(meaningful, line)
		}
	}
	if len(meaningful) == 0 {
		return "", ""
	}
	title := meaningful[0]
	detail := strings.TrimSpace(strings.Join(meaningful[1:], "\n"))
	return title, detail
}

func firstStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64, float32, int, int64, bool:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}

func isPublishGenerationPromptEcho(value string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n")
	return strings.Contains(normalized, "已有标题") &&
		strings.Contains(normalized, "已有详情正文") &&
		strings.Contains(normalized, "可用于分析的图片数量")
}

func hasPublishGenerationLabeledOutput(value string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n")
	for _, label := range append(publishGenerationFieldLabels("title"), publishGenerationFieldLabels("detail")...) {
		if strings.Contains(normalized, label+"：") || strings.Contains(normalized, label+":") {
			return true
		}
	}
	return false
}
