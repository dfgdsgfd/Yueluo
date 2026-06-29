package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/datatypes"

	"yuem-go/backend-gin/internal/domain"
)

func (s *AIService) createGenerationLog(ctx context.Context, jobID string, req AIRequest, actor AIActor, cfg AIConfig, tmpl AITemplateConfig, totalChunks int, now time.Time) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	meta := map[string]any{
		"locale":                   req.Locale,
		"templateKey":              req.TemplateKey,
		"totalChunks":              totalChunks,
		"hasImages":                len(req.Images) > 0,
		"imageSendSuccessCount":    len(normalizeAIImageInputs(req.Images)),
		"supportsVision":           tmpl.SupportsVision,
		"templateTask":             tmpl.TaskType,
		"showReasoning":            cfg.ShowReasoning,
		"thinkingParameterEnabled": cfg.ThinkingParameterEnabled,
		"thinkingEnabled":          cfg.ThinkingEnabled,
		"reasoningEffort":          cfg.ReasoningEffort,
		"modelParameters":          sanitizeAIDBMap(cfg.ModelParameters),
	}
	addAIImageSelectionLogMetadata(meta, req)
	row := domain.AIGenerationLog{
		JobID:          jobID,
		TaskType:       req.Type,
		TemplateKey:    req.TemplateKey,
		ActorType:      nonEmptyString(actor.Type, "system"),
		ActorID:        actor.ID,
		ActorDisplayID: actor.DisplayID,
		InputSummary:   summarizeAIText(req.Input, 1200),
		Status:         "running",
		Model:          nonEmptyString(strings.TrimSpace(tmpl.Model), cfg.Model),
		BaseURL:        cfg.BaseURL,
		Metadata:       jsonData(meta),
		CreatedAt:      now,
		UpdatedAt:      &now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0
	}
	return row.ID
}

func addAIImageSelectionLogMetadata(meta map[string]any, req AIRequest) {
	if meta == nil {
		return
	}
	if value, ok := firstPresent(req.Variables, "imageSelectionMode", "image_selection_mode", "imageMode", "image_mode"); ok {
		if text := strings.TrimSpace(firstStringValue(value)); text != "" {
			meta["imageSelectionMode"] = normalizeAIImageSelectionMode(text)
		}
	}
	if value, ok := firstPresent(req.Variables, "imageCandidateCount", "imageCandidates", "imageCount"); ok {
		if parsed, valid := intSettingInput(value); valid && parsed >= 0 {
			meta["imageCandidateCount"] = parsed
		}
	}
	if value, ok := firstPresent(req.Variables, "imageSendSuccessCount", "selectedImageCount"); ok {
		if parsed, valid := intSettingInput(value); valid && parsed >= 0 {
			meta["imageSendSuccessCount"] = parsed
		}
	}
}

func (s *AIService) finishGenerationLog(ctx context.Context, id int64, status string, output string, usage AIUsage, errorCode string, errorMessage string, durationMS int64, tokensPerSecond float64, upstreamAttempts []map[string]any, fallbackChunks []map[string]any) {
	if s == nil || s.db == nil || id <= 0 {
		return
	}
	now := time.Now()
	updates := map[string]any{
		"status":            status,
		"output_summary":    summarizeAIText(output, 1200),
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":      usage.TotalTokens,
		"error_code":        errorCode,
		"error_message":     summarizeAIText(errorMessage, 1200),
		"duration_ms":       durationMS,
		"tokens_per_second": tokensPerSecond,
		"updated_at":        &now,
	}
	var row domain.AIGenerationLog
	if err := s.db.WithContext(ctx).Select("metadata").Where("id = ?", id).First(&row).Error; err == nil {
		meta := jsonMapFromLog(row.Metadata)
		meta["completedAt"] = now.UTC().Format(time.RFC3339)
		meta["durationMs"] = durationMS
		meta["tokensPerSecond"] = tokensPerSecond
		if value, ok := meta["imageSendSuccessCount"]; !ok || value == nil {
			meta["imageSendSuccessCount"] = 0
		}
		if len(upstreamAttempts) > 0 {
			meta["upstreamAttempts"] = upstreamAttempts
			last := upstreamAttempts[len(upstreamAttempts)-1]
			meta["upstreamStatus"] = last["status"]
			meta["upstreamContentType"] = last["contentType"]
			meta["upstreamResponseSummary"] = last["responseSummary"]
			meta["upstreamOutputSummary"] = last["outputSummary"]
			meta["upstreamUsage"] = last["usage"]
		}
		if len(fallbackChunks) > 0 {
			meta["fallbackChunks"] = fallbackChunks
			meta["fallbackChunkCount"] = len(fallbackChunks)
		}
		updates["metadata"] = jsonData(meta)
	}
	_ = s.db.WithContext(ctx).Model(&domain.AIGenerationLog{}).Where("id = ?", id).Updates(updates).Error
}

func jsonData(value map[string]any) datatypes.JSON {
	data, err := json.Marshal(sanitizeAIDBMap(value))
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(data)
}

func summarizeAIText(input string, limit int) string {
	input = strings.TrimSpace(sanitizeAIDBText(input))
	if limit <= 0 || utf8.RuneCountInString(input) <= limit {
		return input
	}
	runes := []rune(input)
	return string(runes[:limit])
}

func jsonMapFromLog(raw datatypes.JSON) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}
