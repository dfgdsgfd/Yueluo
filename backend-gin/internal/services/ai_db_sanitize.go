package services

import "strings"

func sanitizeAIDBText(value string) string {
	value = strings.ToValidUTF8(value, "")
	if strings.ContainsRune(value, 0) {
		value = strings.ReplaceAll(value, "\x00", "")
	}
	return value
}

func sanitizeAIDBAny(value any) any {
	switch typed := value.(type) {
	case string:
		return sanitizeAIDBText(typed)
	case map[string]any:
		return sanitizeAIDBMap(typed)
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[sanitizeAIDBText(key)] = sanitizeAIDBText(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeAIDBAny(item))
		}
		return out
	default:
		return value
	}
}

func sanitizeAIDBMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[sanitizeAIDBText(key)] = sanitizeAIDBAny(value)
	}
	return out
}
