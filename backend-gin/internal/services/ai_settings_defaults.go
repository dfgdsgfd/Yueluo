package services

func defaultAISettings() map[string]any {
	return map[string]any{
		AISettingEnabled:                  false,
		AISettingBaseURL:                  "https://api.openai.com/v1",
		AISettingAPIKey:                   "",
		AISettingModel:                    "gpt-5-mini",
		AISettingExtraHeaders:             map[string]any{},
		AISettingTimeoutSeconds:           60,
		AISettingMaxRunSeconds:            3600,
		AISettingChunkMaxChars:            3000,
		AISettingConcurrency:              5,
		AISettingTemperature:              0.4,
		AISettingMaxOutputTokens:          8192,
		AISettingPromptTemplates:          DefaultAIPromptTemplates(),
		AISettingShowReasoning:            false,
		AISettingThinkingParameterEnabled: false,
		AISettingThinkingEnabled:          false,
		AISettingReasoningEffort:          "",
		AISettingModelParameters:          map[string]any{},
		AISettingLogHTTPDetails:           false,
		AISettingContentFormat:            defaultAIContentFormatConfig(),

		AISettingAutoCommentEnabled:      false,
		AISettingAutoCommentBotUserID:    0,
		AISettingAutoCommentBotUserIDMin: 0,
		AISettingAutoCommentBotUserIDMax: 0,
		AISettingAutoCommentTemplateKey:  "post_auto_comment",
		AISettingAutoCommentDelaySeconds: 10,
		AISettingAutoCommentMaxImages:    4,
		AISettingAutoCommentImageMode:    AIImageSelectionOrdered,
		AISettingAutoCommentStyle:        "normal",
		AISettingCommentReply:            defaultAICommentReplyConfig(),
		AISettingModeration:              defaultAIModerationConfig(),
		AISettingPublishGeneration:       defaultAIPublishGenerationConfig(),
	}
}
