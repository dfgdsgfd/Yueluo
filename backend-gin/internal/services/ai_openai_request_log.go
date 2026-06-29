package services

import (
	"fmt"
	"net/http"
	"strings"
)

func openAIRequestLog(targetURL string, headers http.Header, body map[string]any, includeDetails bool) map[string]any {
	out := map[string]any{
		"url":    sanitizeAIUpstreamURL(targetURL),
		"method": http.MethodPost,
	}
	if !includeDetails {
		out["detailsRecorded"] = false
		return out
	}
	out["detailsRecorded"] = true
	out["headers"] = sanitizeAIHTTPHeaders(headers)
	out["body"] = sanitizeAIHTTPRequestBody(body)
	return out
}

func sanitizeAIHTTPHeaders(headers http.Header) map[string]any {
	out := make(map[string]any, len(headers))
	for key, values := range headers {
		if aiSensitiveKey(key) {
			out[key] = "[redacted]"
			continue
		}
		if len(values) == 1 {
			out[key] = summarizeAIText(values[0], 1000)
			continue
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, summarizeAIText(value, 1000))
		}
		out[key] = items
	}
	return out
}

func sanitizeAIHTTPRequestBody(value any) any {
	return sanitizeAIHTTPRequestValue(value, 0)
}

func sanitizeAIHTTPRequestValue(value any, depth int) any {
	if depth > 12 {
		return "[max-depth]"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if aiSensitiveKey(key) {
				out[key] = "[redacted]"
				continue
			}
			if strings.EqualFold(key, "image_url") {
				out[key] = sanitizeAIImageURLBlock(item, depth+1)
				continue
			}
			out[key] = sanitizeAIHTTPRequestValue(item, depth+1)
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAIHTTPRequestValue(item, depth+1))
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAIHTTPRequestValue(item, depth+1))
		}
		return out
	case string:
		return sanitizeAIHTTPRequestString(typed)
	default:
		return typed
	}
}

func sanitizeAIImageURLBlock(value any, depth int) any {
	if block, ok := value.(map[string]any); ok {
		out := make(map[string]any, len(block))
		for key, item := range block {
			if strings.EqualFold(key, "url") {
				out[key] = sanitizeAIImageURL(fmt.Sprint(item))
				continue
			}
			out[key] = sanitizeAIHTTPRequestValue(item, depth+1)
		}
		return out
	}
	return sanitizeAIHTTPRequestValue(value, depth+1)
}

func sanitizeAIHTTPRequestString(value string) string {
	value = sanitizeAIDBText(value)
	if strings.HasPrefix(value, "data:") {
		return summarizeAIDataURL(value)
	}
	return summarizeAIText(value, 12000)
}

func sanitizeAIImageURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "data:") {
		return summarizeAIDataURL(value)
	}
	return summarizeAIText(sanitizeAIUpstreamURL(value), 4000)
}

func summarizeAIDataURL(value string) string {
	header := value
	if before, _, ok := strings.Cut(value, ","); ok {
		header = before
	}
	return fmt.Sprintf("%s,[redacted %d bytes]", summarizeAIText(header, 200), len(value))
}

func aiSensitiveKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "authorization") ||
		strings.Contains(lower, "cookie") ||
		strings.Contains(lower, "password")
}
