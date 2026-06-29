package services

import (
	"fmt"
	"strings"

	"gorm.io/datatypes"
)

func moderationModelResultMap(rawText string, decision aiModerationDecision, cfg AIModerationTargetConfig, action string, status string) map[string]any {
	out := map[string]any{
		"raw":           rawText,
		"rawJson":       decision.Raw,
		"decision":      moderationDecisionMap(decision),
		"action":        action,
		"status":        status,
		"triggerReason": moderationTriggerReason(decision, action, ""),
		"enabledRules":  cfg.Rules,
	}
	if len(decision.Raw) == 0 {
		out["rawJson"] = nil
		out["rawText"] = rawText
	}
	return out
}

func moderationCategoriesJSON(decision aiModerationDecision) datatypes.JSON {
	out := map[string]any{}
	for key, result := range decision.Rules {
		if result.Matched {
			out[key] = result
		}
	}
	return jsonData(out)
}

func moderationTriggerReason(decision aiModerationDecision, action string, errorMessage string) string {
	if strings.TrimSpace(errorMessage) != "" {
		return summarizeAIText(errorMessage, 500)
	}
	parts := make([]string, 0, len(decision.Rules))
	for key, result := range decision.Rules {
		if !result.Matched {
			continue
		}
		detail := fmt.Sprintf("%s matched", key)
		if result.Action != "" {
			detail += " -> " + result.Action
		}
		if result.Confidence > 0 {
			detail += fmt.Sprintf(" (confidence %.2f >= threshold %.2f)", result.Confidence, result.Threshold)
		}
		if result.Severity != "" {
			detail += ", severity " + result.Severity
		}
		if len(result.Sources) > 0 {
			detail += ", source " + strings.Join(result.Sources, "+")
		}
		if strings.TrimSpace(result.Reason) != "" {
			detail += ": " + strings.TrimSpace(result.Reason)
		}
		parts = append(parts, detail)
	}
	if len(parts) > 0 {
		if action != "" && action != AIModerationActionObserve {
			return summarizeAIText(strings.Join(parts, "; ")+"; action "+action, 1200)
		}
		return summarizeAIText(strings.Join(parts, "; "), 1200)
	}
	if strings.TrimSpace(decision.Reason) != "" {
		return summarizeAIText("observed: "+decision.Reason, 1200)
	}
	return "ai_moderation_no_rule_match"
}
