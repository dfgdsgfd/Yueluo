package services

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type openAIErrorPayload struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
	Param   any    `json:"param"`
}

func openAIErrorFromPayload(raw string, payload *openAIErrorPayload) AIError {
	detail := sanitizeAIUpstreamDetail(raw)
	message := ""
	status := 0
	if payload != nil {
		message = strings.TrimSpace(payload.Message)
		status = payload.UpstreamStatus()
		if message == "" {
			message = strings.TrimSpace(fmt.Sprint(payload.Code))
		}
		if message == "" {
			message = strings.TrimSpace(payload.Type)
		}
	}
	if message == "" {
		message = detail
	}
	return AIError{
		Code:           "error.ai_upstream_error",
		Err:            fmt.Errorf("upstream returned error: %s", message),
		UpstreamStatus: status,
		UpstreamDetail: detail,
	}
}

func (e *openAIErrorPayload) UpstreamStatus() int {
	if e == nil {
		return 0
	}
	if status := statusFromOpenAIErrorCode(e.Code); status > 0 {
		return status
	}
	lower := strings.ToLower(strings.TrimSpace(e.Type + " " + e.Message))
	switch {
	case strings.Contains(lower, "rate_limit") || strings.Contains(lower, "rate limit"):
		return http.StatusTooManyRequests
	case strings.Contains(lower, "unauthorized") || strings.Contains(lower, "authentication"):
		return http.StatusUnauthorized
	case strings.Contains(lower, "forbidden") || strings.Contains(lower, "permission"):
		return http.StatusForbidden
	case strings.Contains(lower, "not_found") || strings.Contains(lower, "not found"):
		return http.StatusNotFound
	case strings.Contains(lower, "internal_server_error") || strings.Contains(lower, "internal server error"):
		return http.StatusInternalServerError
	default:
		return 0
	}
}

func statusFromOpenAIErrorCode(value any) int {
	switch typed := value.(type) {
	case float64:
		status := int(typed)
		if typed == float64(status) && status >= 400 && status <= 599 {
			return status
		}
	case int:
		if typed >= 400 && typed <= 599 {
			return typed
		}
	case string:
		status, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && status >= 400 && status <= 599 {
			return status
		}
	}
	return 0
}

func shouldRetryOpenAITextOnlyAfterStreamError(req AIRequest, err error) bool {
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		return false
	}
	if requiresOpenAIVisionAnalysis(req) {
		return false
	}
	if shouldRetryOpenAITextOnly(req, aiErr.UpstreamStatus, aiErr.UpstreamDetail) {
		return true
	}
	if len(normalizeAIImageInputs(req.Images)) == 0 {
		return false
	}
	lower := strings.ToLower(aiErr.UpstreamDetail + " " + err.Error())
	return aiErr.UpstreamStatus >= 500 && strings.Contains(lower, "internal server error")
}

func shouldRetryOpenAIWithoutStructuredJSON(params openAIRequestParams, err error) bool {
	if !params.structuredJSON {
		return false
	}
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		return false
	}
	lower := strings.ToLower(aiErr.UpstreamDetail + " " + err.Error())
	for _, marker := range []string{
		"response_format",
		"json_object",
		"structured",
		"schema",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return aiErr.UpstreamStatus >= 500 && strings.Contains(lower, "internal server error")
}

func shouldRetryOpenAIWithoutThinking(summary aiOpenAIResponseSummary, req AIRequest) bool {
	if !isAIPublishGenerationTask(req.Type) {
		return false
	}
	return strings.TrimSpace(summary.OutputSummary) == "" && strings.TrimSpace(summary.ReasoningSummary) != ""
}

func openAIWithoutThinkingParams(params openAIRequestParams) openAIRequestParams {
	params.includeThinking = true
	params.thinkingEnabled = false
	params.reasoningEffort = ""
	params.modelParameters = withoutAIThinkingModelParameters(params.modelParameters)
	return params
}

func withoutAIThinkingModelParameters(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]any, len(params))
	for key, value := range params {
		if !isAIThinkingModelParameter(key) {
			out[key] = value
		}
	}
	return out
}

func isAIThinkingModelParameter(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enable_thinking", "thinking", "thinking_enabled", "reasoning", "reasoning_effort":
		return true
	default:
		return false
	}
}

func openAIStreamOutputEmpty(summary aiOpenAIResponseSummary) bool {
	return strings.TrimSpace(summary.OutputSummary) == "" && strings.TrimSpace(summary.ReasoningSummary) == ""
}
