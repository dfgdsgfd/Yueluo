package services

import (
	"net/url"
	"strings"
)

func openAIRequestAttemptLog(cfg AIConfig, params openAIRequestParams, requestLog map[string]any, status int, contentType string, summary aiOpenAIResponseSummary, err error) map[string]any {
	out := map[string]any{
		"url":               sanitizeAIUpstreamURL(openAIChatCompletionsURL(cfg.BaseURL)),
		"method":            "POST",
		"status":            status,
		"contentType":       strings.TrimSpace(contentType),
		"model":             params.model,
		"stream":            true,
		"structuredJson":    params.structuredJSON,
		"forceTextMessages": params.forceTextMessages,
		"streamDataEvents":  summary.StreamDataEvents,
		"done":              summary.Done,
		"nonStreamJson":     summary.NonStreamJSON,
		"responseSummary":   summarizeAIText(summary.RawSummary, 6000),
		"outputSummary":     summarizeAIText(summary.OutputSummary, 4000),
		"reasoningSummary":  summarizeAIText(summary.ReasoningSummary, 2000),
		"usage": map[string]any{
			"promptTokens":     summary.Usage.PromptTokens,
			"completionTokens": summary.Usage.CompletionTokens,
			"totalTokens":      summary.Usage.TotalTokens,
		},
	}
	if len(requestLog) > 0 {
		out["request"] = requestLog
	}
	if err != nil {
		out["errorCode"] = aiErrorCode(err)
		out["errorMessage"] = summarizeAIText(err.Error(), 1200)
	}
	return out
}

func sanitizeAIUpstreamURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return sanitizeAIUpstreamDetail(raw)
	}
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") {
			query.Set(key, "[redacted]")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func appendAIUpstreamAttempt(attempts *[]map[string]any, attempt map[string]any) {
	if attempts == nil {
		return
	}
	*attempts = append(*attempts, sanitizeAIDBMap(attempt))
	if len(*attempts) > 50 {
		*attempts = (*attempts)[len(*attempts)-50:]
	}
}

func appendLimitedAIText(builder *strings.Builder, value string, limit int) {
	if builder == nil || value == "" || limit <= 0 || builder.Len() >= limit {
		return
	}
	remaining := limit - builder.Len()
	if len(value) > remaining {
		value = value[:remaining]
	}
	builder.WriteString(value)
}

func firstNonEmptyAIText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func openAIChatCompletionsURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return strings.TrimRight(base, "/") + "/chat/completions"
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	return parsed.String()
}
