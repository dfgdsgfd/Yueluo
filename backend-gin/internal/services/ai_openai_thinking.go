package services

import (
	"maps"
	"strings"
)

func usesNVIDIAChatTemplateThinking(baseURL, model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(baseURL + " " + model))
	return strings.Contains(normalized, "api.nvidia.com") ||
		strings.Contains(normalized, "nvidia/") ||
		strings.Contains(normalized, "nemotron")
}

func setOpenAIChatTemplateThinking(body map[string]any, enabled bool) {
	existing, _ := body["chat_template_kwargs"].(map[string]any)
	next := make(map[string]any, len(existing)+1)
	maps.Copy(next, existing)
	next["enable_thinking"] = enabled
	body["chat_template_kwargs"] = next
}
