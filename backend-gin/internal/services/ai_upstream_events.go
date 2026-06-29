package services

import (
	"net/http"
	"time"
)

type aiUpstreamEventEmitter func(string, map[string]any)

func aiStreamUpstreamEmitter(jobID string, chunkIndex int, emit func(AIStreamEvent) error) aiUpstreamEventEmitter {
	if emit == nil {
		return nil
	}
	return func(phase string, payload map[string]any) {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["phase"] = phase
		payload["at"] = time.Now().UTC().Format(time.RFC3339Nano)
		payload["chunkIndex"] = chunkIndex
		_ = emit(AIStreamEvent{
			Type:       "upstream_event",
			JobID:      jobID,
			ChunkIndex: chunkIndex,
			Stage:      phase,
			Upstream:   sanitizeAIDBMap(payload),
		})
	}
}

func emitAIUpstreamEvent(emit aiUpstreamEventEmitter, phase string, payload map[string]any) {
	if emit == nil {
		return
	}
	emit(phase, payload)
}

func aiUsageEventPayload(usage *AIUsage) map[string]any {
	if usage == nil {
		return nil
	}
	return map[string]any{
		"promptTokens":     usage.PromptTokens,
		"completionTokens": usage.CompletionTokens,
		"totalTokens":      usage.TotalTokens,
	}
}

func emitOpenAIRequestError(emit aiUpstreamEventEmitter, cfg AIConfig, params openAIRequestParams, requestLog map[string]any, err error) {
	emitAIUpstreamEvent(emit, "request_error", map[string]any{
		"url":          sanitizeAIUpstreamURL(openAIChatCompletionsURL(cfg.BaseURL)),
		"method":       "POST",
		"model":        params.model,
		"request":      requestLog,
		"errorCode":    aiErrorCode(err),
		"errorMessage": summarizeAIText(err.Error(), 1200),
	})
}

func emitOpenAIResponseStart(emit aiUpstreamEventEmitter, cfg AIConfig, params openAIRequestParams, requestLog map[string]any, resp *http.Response) {
	if resp == nil {
		return
	}
	emitAIUpstreamEvent(emit, "response_start", map[string]any{
		"url":               sanitizeAIUpstreamURL(openAIChatCompletionsURL(cfg.BaseURL)),
		"method":            "POST",
		"status":            resp.StatusCode,
		"contentType":       resp.Header.Get("Content-Type"),
		"model":             params.model,
		"structuredJson":    params.structuredJSON,
		"forceTextMessages": params.forceTextMessages,
		"request":           requestLog,
	})
}

func emitOpenAIResponseError(emit aiUpstreamEventEmitter, resp *http.Response, detail string) {
	if resp == nil {
		return
	}
	emitAIUpstreamEvent(emit, "response_error", map[string]any{
		"status":      resp.StatusCode,
		"contentType": resp.Header.Get("Content-Type"),
		"detail":      summarizeAIText(detail, 1200),
	})
}
