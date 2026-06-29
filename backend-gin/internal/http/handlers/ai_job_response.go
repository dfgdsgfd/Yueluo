package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) aiJobResponse(ctx context.Context, row domain.AIJob) gin.H {
	data := aiJobResponse(row)
	if queueJob := h.aiJobQueueJob(ctx, row); queueJob != nil {
		data["queueJob"] = queueJob
	}
	return data
}

func mergeAIQueueJobPayload(existing map[string]any, next map[string]any) map[string]any {
	if len(existing) == 0 {
		return next
	}
	if len(next) == 0 {
		return existing
	}
	merged := make(map[string]any, len(existing)+len(next))
	maps.Copy(merged, existing)
	for key, value := range next {
		if key == "activeJob" {
			if current, ok := merged[key].(map[string]any); ok {
				if incoming, ok := value.(map[string]any); ok {
					merged[key] = mergeAIQueueJobPayload(current, incoming)
					continue
				}
			}
		}
		merged[key] = value
	}
	return merged
}

func aiJobResponse(row domain.AIJob) gin.H {
	return gin.H{
		"id":               row.ID,
		"jobId":            row.JobID,
		"requestHash":      row.RequestHash,
		"type":             row.TaskType,
		"templateKey":      row.TemplateKey,
		"actorType":        row.ActorType,
		"actorId":          row.ActorID,
		"actorDisplayId":   row.ActorDisplayID,
		"status":           row.Status,
		"stage":            row.Stage,
		"percent":          row.Percent,
		"currentChunk":     row.CurrentChunk,
		"totalChunks":      row.TotalChunks,
		"processedChars":   row.ProcessedChars,
		"totalChars":       row.TotalChars,
		"inputSummary":     row.InputSummary,
		"output":           row.Output,
		"reasoning":        row.Reasoning,
		"errorCode":        row.ErrorCode,
		"errorMessage":     row.ErrorMessage,
		"upstreamStatus":   row.UpstreamStatus,
		"upstreamDetail":   row.UpstreamDetail,
		"promptTokens":     row.PromptTokens,
		"completionTokens": row.CompletionTokens,
		"totalTokens":      row.TotalTokens,
		"estimatedTokens":  row.EstimatedTokens,
		"tokensPerSecond":  row.TokensPerSecond,
		"metadata":         jsonValue([]byte(row.Metadata)),
		"startedAt":        row.StartedAt,
		"finishedAt":       row.FinishedAt,
		"createdAt":        row.CreatedAt,
		"updatedAt":        row.UpdatedAt,
	}
}

func (h NativeHandlers) aiJobQueueJob(ctx context.Context, row domain.AIJob) gin.H {
	var queueJob gin.H
	if metaQueue := aiJobMetadataQueueJob(row); metaQueue != nil {
		queueJob = metaQueue
	}
	if queueJob == nil {
		if row.Status != services.AIJobStatusQueued || h.Queue == nil {
			return nil
		}
		position, eta, total := h.Queue.AIJobRunQueueEstimate(ctx, row.JobID)
		if position <= 0 && total <= 0 {
			return nil
		}
		queueJob = gin.H{
			"id":                   services.TaskAIJobRun + ":" + row.JobID,
			"jobId":                row.JobID,
			"queue":                services.QueueAITask,
			"state":                "queued",
			"queuePosition":        position,
			"queueCount":           total,
			"estimatedWaitSeconds": eta,
		}
	}
	if activeJob := h.aiActiveJobForQueue(ctx, row); activeJob != nil {
		queueJob["activeJob"] = activeJob
	}
	return queueJob
}

func aiJobMetadataQueueJob(row domain.AIJob) gin.H {
	meta, _ := jsonValue([]byte(row.Metadata)).(map[string]any)
	if meta == nil {
		return nil
	}
	raw, _ := meta["queueJob"].(map[string]any)
	if raw == nil {
		return nil
	}
	position := intFromNumeric(raw["queuePosition"])
	total := intFromNumeric(raw["queueCount"])
	if total <= 0 {
		total = intFromNumeric(raw["queueTotal"])
	}
	eta := intFromNumeric(raw["estimatedWaitSeconds"])
	if eta <= 0 {
		eta = intFromNumeric(raw["etaSeconds"])
	}
	if position <= 0 && total <= 0 {
		return nil
	}
	out := gin.H{
		"id":                   firstNonEmpty(stringFromQueueMeta(raw["id"]), row.JobID),
		"jobId":                firstNonEmpty(stringFromQueueMeta(raw["jobId"]), row.JobID),
		"queue":                firstNonEmpty(stringFromQueueMeta(raw["queue"]), "ai-concurrency"),
		"state":                firstNonEmpty(stringFromQueueMeta(raw["state"]), "queued"),
		"queuePosition":        position,
		"queueCount":           total,
		"estimatedWaitSeconds": eta,
	}
	if active, _ := raw["activeJob"].(map[string]any); active != nil {
		out["activeJob"] = active
	}
	return out
}

func (h NativeHandlers) aiActiveJobForQueue(ctx context.Context, waiting domain.AIJob) gin.H {
	if h.DB == nil || waiting.Status != services.AIJobStatusQueued {
		return nil
	}
	var active domain.AIJob
	if strings.TrimSpace(waiting.TemplateKey) != "" {
		err := h.DB.WithContext(ctx).
			Where("status = ? AND id <> ? AND template_key = ?", services.AIJobStatusRunning, waiting.ID, waiting.TemplateKey).
			Order("started_at ASC, updated_at ASC").
			First(&active).Error
		if err == nil {
			return aiActiveJobPayload(active)
		}
	}
	err := h.DB.WithContext(ctx).
		Where("status = ? AND id <> ?", services.AIJobStatusRunning, waiting.ID).
		Order("started_at ASC, updated_at ASC").
		First(&active).Error
	if err != nil {
		return nil
	}
	return aiActiveJobPayload(active)
}

func aiActiveJobPayload(row domain.AIJob) gin.H {
	return gin.H{
		"jobId":           row.JobID,
		"actorId":         row.ActorID,
		"actorDisplayId":  row.ActorDisplayID,
		"generatedTokens": aiGeneratedTokensForResponse(row),
		"tokensPerSecond": row.TokensPerSecond,
	}
}

func aiGeneratedTokensForResponse(row domain.AIJob) int {
	tokens := row.CompletionTokens
	if tokens <= 0 {
		tokens = row.TotalTokens
	}
	if tokens <= 0 {
		runes := len([]rune(strings.TrimSpace(row.Output)))
		if runes > 0 {
			tokens = runes/4 + 1
		}
	}
	if tokens < 0 {
		return 0
	}
	return tokens
}

func stringFromQueueMeta(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	return text
}

func intFromNumeric(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		parsed, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		return parsed
	}
}
