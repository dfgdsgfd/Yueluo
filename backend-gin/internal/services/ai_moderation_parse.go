package services

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const (
	aiModerationRuleSourceExplicit = "explicit_rule"
	aiModerationRuleSourceListed   = "listed_category"
	aiModerationRuleSourceSeverity = "severity"
	aiModerationRuleSourceScore    = "confidence"
)

type aiModerationDecision struct {
	Violation bool                              `json:"violation"`
	Reason    string                            `json:"reason"`
	Action    string                            `json:"action"`
	Raw       map[string]any                    `json:"raw,omitempty"`
	Rules     map[string]aiModerationRuleResult `json:"rules"`
}

type aiModerationRuleResult struct {
	Violation   bool     `json:"violation"`
	Matched     bool     `json:"matched"`
	Reason      string   `json:"reason"`
	Severity    string   `json:"severity"`
	Confidence  float64  `json:"confidence"`
	Threshold   float64  `json:"threshold"`
	Sensitivity float64  `json:"sensitivity"`
	Action      string   `json:"action"`
	Sources     []string `json:"sources,omitempty"`
}

func parseAIModerationDecision(text string, cfg AIModerationTargetConfig, targetType string) aiModerationDecision {
	raw := decodeModerationJSONMap(text)
	decision := aiModerationDecision{Raw: raw, Rules: map[string]aiModerationRuleResult{}}
	topViolation, hasTopViolation := firstModerationBool(raw, "violation", "is_violation", "isViolation", "violated", "flagged", "unsafe", "blocked", "rejected", "should_block", "shouldBlock", "should_reject", "shouldReject")
	decision.Reason = firstModerationString(raw, "reason", "summary", "message", "explanation", "rationale")
	if action := normalizeModerationAction(firstModerationString(raw, "action", "recommended_action", "recommendedAction", "moderation_action", "moderationAction", "decision", "result"), targetType); action != "" {
		decision.Action = action
	}
	rulesRaw := firstModerationMap(raw, "rules", "categories", "category_results", "categoryResults", "rule_results", "ruleResults")
	matchedRules := 0
	for key, ruleCfg := range cfg.Rules {
		if !ruleCfg.Enabled {
			continue
		}
		itemRaw := moderationRuleMap(raw, rulesRaw, key)
		threshold := moderationConfidenceThreshold(ruleCfg.Sensitivity)
		rule := aiModerationRuleResult{
			Reason:      firstModerationString(itemRaw, "reason", "message", "explanation", "rationale"),
			Severity:    strings.ToLower(firstModerationString(itemRaw, "severity", "level", "risk", "risk_level", "riskLevel")),
			Threshold:   threshold,
			Sensitivity: boundedFloat(ruleCfg.Sensitivity, 0, 1, 0.65),
			Action:      normalizeModerationAction(ruleCfg.Action, targetType),
		}
		if parsed, ok := firstModerationBool(itemRaw, "violation", "is_violation", "isViolation", "violated", "flagged", "hit", "matched", "detected", "unsafe"); ok {
			if parsed {
				rule.Sources = append(rule.Sources, aiModerationRuleSourceExplicit)
			}
			rule.Violation = parsed
		}
		if parsed, valid := firstModerationFloat(itemRaw, "confidence", "score", "probability", "risk_score", "riskScore"); valid {
			rule.Confidence = boundedFloat(parsed, 0, 1, 0)
		}
		if !rule.Violation && moderationRuleListed(raw, key) {
			rule.Violation = true
			rule.Sources = append(rule.Sources, aiModerationRuleSourceListed)
		}
		if !rule.Violation && moderationSeverityIsViolation(rule.Severity, rule.Confidence, ruleCfg.Sensitivity) {
			rule.Violation = true
			rule.Sources = append(rule.Sources, aiModerationRuleSourceSeverity)
		}
		if !rule.Violation && rule.Confidence > 0 && rule.Confidence >= threshold {
			rule.Violation = true
			rule.Sources = append(rule.Sources, aiModerationRuleSourceScore)
		}
		rule.Matched = rule.Violation && moderationRuleEvidenceMeetsThreshold(rule, ruleCfg)
		if rule.Matched {
			matchedRules++
			decision.Violation = true
			if decision.Reason == "" {
				decision.Reason = key + ": " + rule.Reason
			}
		} else {
			rule.Violation = false
		}
		decision.Rules[key] = rule
	}
	if matchedRules == 0 {
		decision.Violation = false
	}
	if hasTopViolation && topViolation && !decision.Violation && decision.Reason == "" {
		decision.Reason = "Model set top-level violation without an enabled rule meeting the configured threshold."
	}
	return decision
}

func moderationActionForDecision(decision aiModerationDecision, cfg AIModerationTargetConfig, targetType string) string {
	if !decision.Violation {
		return AIModerationActionObserve
	}
	for key, result := range decision.Rules {
		if !result.Matched {
			continue
		}
		if rule, ok := cfg.Rules[key]; ok {
			if action := normalizeModerationAction(rule.Action, targetType); action != "" {
				return action
			}
		}
	}
	return AIModerationActionObserve
}

func moderationRuleEvidenceMeetsThreshold(rule aiModerationRuleResult, cfg AIModerationRuleConfig) bool {
	if !rule.Violation {
		return false
	}
	if rule.Confidence > 0 && rule.Confidence < moderationConfidenceThreshold(cfg.Sensitivity) {
		return false
	}
	if containsString(rule.Sources, aiModerationRuleSourceExplicit) || containsString(rule.Sources, aiModerationRuleSourceListed) {
		return true
	}
	if containsString(rule.Sources, aiModerationRuleSourceSeverity) {
		return true
	}
	return rule.Confidence > 0 && rule.Confidence >= moderationConfidenceThreshold(cfg.Sensitivity)
}

func extractJSONText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	value = strings.TrimSpace(value)
	if json.Valid([]byte(value)) {
		return value
	}
	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start >= 0 && end > start {
		candidate := strings.TrimSpace(value[start : end+1])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}
	return value
}

func boolFromModerationAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "1", "violation", "violated", "hit", "matched", "detected", "flagged", "unsafe", "block", "blocked", "reject", "rejected":
			return true, true
		case "false", "no", "0", "ok", "pass", "safe", "allow", "allowed", "approve", "approved":
			return false, true
		}
	case float64:
		return typed != 0, true
	}
	return false, false
}

func decodeModerationJSONMap(text string) map[string]any {
	raw := map[string]any{}
	_ = json.Unmarshal([]byte(extractJSONText(text)), &raw)
	if raw == nil {
		return map[string]any{}
	}
	return raw
}

func firstModerationMap(raw map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if item := jsonMapFromSetting(raw[key]); item != nil && len(item) > 0 {
			return item
		}
	}
	return map[string]any{}
}

func moderationRuleMap(raw map[string]any, rulesRaw map[string]any, key string) map[string]any {
	candidates := []any{rulesRaw[key], raw[key]}
	normalizedKey := normalizeModerationRuleKey(key)
	for mapKey, value := range rulesRaw {
		if normalizeModerationRuleKey(mapKey) == normalizedKey {
			candidates = append(candidates, value)
		}
	}
	for _, value := range candidates {
		if item := jsonMapFromSetting(value); item != nil && len(item) > 0 {
			return item
		}
		if parsed, ok := boolFromModerationAny(value); ok {
			return map[string]any{"violation": parsed}
		}
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
			return map[string]any{"reason": text}
		}
	}
	return map[string]any{}
}

func moderationRuleListed(raw map[string]any, key string) bool {
	normalizedKey := normalizeModerationRuleKey(key)
	for _, listKey := range []string{"violations", "violation_types", "violationTypes", "matched_rules", "matchedRules", "categories", "flagged_categories", "flaggedCategories"} {
		if moderationListContains(raw[listKey], normalizedKey) {
			return true
		}
	}
	category := firstModerationString(raw, "category", "type", "violation_type", "violationType")
	return normalizeModerationRuleKey(category) == normalizedKey
}

func moderationListContains(value any, normalizedKey string) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if normalizeModerationRuleKey(fmt.Sprint(item)) == normalizedKey {
				return true
			}
			if itemMap := jsonMapFromSetting(item); itemMap != nil {
				if normalizeModerationRuleKey(firstModerationString(itemMap, "type", "category", "rule", "name", "key")) == normalizedKey {
					return true
				}
			}
		}
	case []string:
		for _, item := range typed {
			if normalizeModerationRuleKey(item) == normalizedKey {
				return true
			}
		}
	case map[string]any:
		for key, item := range typed {
			if normalizeModerationRuleKey(key) == normalizedKey {
				if parsed, ok := boolFromModerationAny(item); !ok || parsed {
					return true
				}
			}
		}
	case string:
		for _, part := range strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == '\n' || r == ' '
		}) {
			if normalizeModerationRuleKey(part) == normalizedKey {
				return true
			}
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}

func normalizeModerationRuleKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	switch value {
	case "political", "politics", "sensitive_political", "politically_sensitive":
		return "political_sensitive"
	case "adult", "sexual", "sex", "pornography", "nsfw", "obscene":
		return "porn"
	case "advertising", "ad", "ads", "scam":
		return "spam"
	default:
		return value
	}
}

func firstModerationBool(raw map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if parsed, ok := boolFromModerationAny(raw[key]); ok {
			return parsed, true
		}
	}
	return false, false
}

func firstModerationString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func firstModerationFloat(raw map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if parsed, ok := floatSettingInput(raw[key]); ok {
			return parsed, true
		}
	}
	return 0, false
}

func moderationSeverityIsViolation(severity string, confidence float64, sensitivity float64) bool {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "severe", "high":
		return true
	case "medium", "moderate":
		return confidence <= 0 || confidence >= moderationConfidenceThreshold(sensitivity)
	default:
		return false
	}
}

func moderationConfidenceThreshold(sensitivity float64) float64 {
	sensitivity = boundedFloat(sensitivity, 0, 1, 0.65)
	return 0.95 - 0.45*sensitivity
}

func moderationDecisionMap(decision aiModerationDecision) map[string]any {
	return map[string]any{
		"violation": decision.Violation,
		"reason":    decision.Reason,
		"action":    decision.Action,
		"rules":     decision.Rules,
	}
}
