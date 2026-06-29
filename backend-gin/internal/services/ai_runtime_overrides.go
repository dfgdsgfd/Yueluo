package services

import "strings"

func applyAITemplateRuntimeOverrides(cfg AIConfig, tmpl AITemplateConfig) AIConfig {
	overrides := tmpl.RuntimeOverrides
	if !overrides.Enabled {
		return cfg
	}
	if overrides.ShowReasoning != nil {
		cfg.ShowReasoning = *overrides.ShowReasoning
	}
	if overrides.ThinkingParameterEnabled != nil {
		cfg.ThinkingParameterEnabled = *overrides.ThinkingParameterEnabled
	}
	if overrides.ThinkingEnabled != nil {
		cfg.ThinkingEnabled = *overrides.ThinkingEnabled
	}
	if overrides.ReasoningEffort != nil {
		cfg.ReasoningEffort = normalizeReasoningEffortValue(*overrides.ReasoningEffort)
	}
	if overrides.ModelParameters != nil {
		cfg.ModelParameters = jsonMapFromSetting(overrides.ModelParameters)
		if cfg.ModelParameters == nil {
			cfg.ModelParameters = map[string]any{}
		}
	}
	return cfg
}

func normalizeAIRuntimeOverrides(input AIRuntimeOverrides) (AIRuntimeOverrides, bool) {
	if input.ReasoningEffort != nil {
		text := strings.TrimSpace(*input.ReasoningEffort)
		normalized, valid := normalizeReasoningEffortSetting(text)
		if !valid {
			return input, false
		}
		input.ReasoningEffort = &normalized
	}
	if input.ModelParameters != nil {
		params := jsonMapFromSetting(input.ModelParameters)
		if params == nil {
			return input, false
		}
		input.ModelParameters = params
	}
	return input, true
}
