package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func NewAIService(db *gorm.DB, settings *SettingsService) *AIService {
	return &AIService{
		db:           db,
		settings:     settings,
		client:       &http.Client{},
		gate:         newAIConcurrencyGate(),
		projectGates: map[string]*aiConcurrencyGate{},
		jobEvents:    newAIJobEventBroker(),
	}
}

func (s *AIService) PublicSettings() AIPublicSettings {
	cfg := s.Config()
	return AIPublicSettings{
		Enabled:                  cfg.Enabled,
		BaseURL:                  cfg.BaseURL,
		APIKeySet:                strings.TrimSpace(cfg.APIKey) != "",
		APIKeyMasked:             MaskSecret(cfg.APIKey),
		Model:                    cfg.Model,
		ExtraHeaders:             cfg.ExtraHeaders,
		TimeoutSeconds:           cfg.TimeoutSeconds,
		MaxRunSeconds:            cfg.MaxRunSeconds,
		ChunkMaxChars:            cfg.ChunkMaxChars,
		Concurrency:              cfg.Concurrency,
		Temperature:              cfg.Temperature,
		MaxOutputTokens:          cfg.MaxOutputTokens,
		Templates:                cfg.Templates,
		ShowReasoning:            cfg.ShowReasoning,
		ThinkingParameterEnabled: cfg.ThinkingParameterEnabled,
		ThinkingEnabled:          cfg.ThinkingEnabled,
		ReasoningEffort:          cfg.ReasoningEffort,
		ModelParameters:          cfg.ModelParameters,
		LogHTTPDetails:           cfg.LogHTTPDetails,
		ContentFormat:            cfg.ContentFormat,
		AutoComment:              cfg.AutoComment,
		CommentReply:             cfg.CommentReply,
		Moderation:               cfg.Moderation,
		PublishGeneration:        cfg.PublishGeneration,
		DefaultTemplates:         DefaultAIPromptTemplateDefaults(),
	}
}

func (s *AIService) Config() AIConfig {
	defaults := defaultAIConfig()
	if s == nil || s.settings == nil {
		return defaults
	}
	cfg := defaults
	cfg.Enabled = s.settings.Bool(AISettingEnabled)
	cfg.BaseURL = nonEmptyString(s.settings.String(AISettingBaseURL), defaults.BaseURL)
	cfg.APIKey = s.settings.String(AISettingAPIKey)
	cfg.Model = nonEmptyString(s.settings.String(AISettingModel), defaults.Model)
	cfg.ExtraHeaders = stringMapFromSetting(s.settings.Get(AISettingExtraHeaders))
	cfg.TimeoutSeconds = boundedInt(s.settings.Int(AISettingTimeoutSeconds, defaults.TimeoutSeconds), 1, 300, defaults.TimeoutSeconds)
	cfg.MaxRunSeconds = boundedOptionalMinMaxInt(s.settings.Int(AISettingMaxRunSeconds, defaults.MaxRunSeconds), 0, 86400, defaults.MaxRunSeconds)
	cfg.ChunkMaxChars = boundedOptionalInt(s.settings.Int(AISettingChunkMaxChars, defaults.ChunkMaxChars), 500, 20000, defaults.ChunkMaxChars)
	cfg.Concurrency = boundedInt(s.settings.Int(AISettingConcurrency, defaults.Concurrency), 1, 50, defaults.Concurrency)
	cfg.Temperature = boundedFloat(settingFloat(s.settings.Get(AISettingTemperature), defaults.Temperature), 0, 2, defaults.Temperature)
	cfg.MaxOutputTokens = boundedOptionalMinInt(s.settings.Int(AISettingMaxOutputTokens, defaults.MaxOutputTokens), 128, defaults.MaxOutputTokens)
	cfg.Templates = templatesFromSetting(s.settings.Get(AISettingPromptTemplates), defaults.Templates)
	cfg.ShowReasoning = s.settings.Bool(AISettingShowReasoning)
	cfg.ThinkingParameterEnabled = s.settings.Bool(AISettingThinkingParameterEnabled)
	cfg.ThinkingEnabled = s.settings.Bool(AISettingThinkingEnabled)
	cfg.ReasoningEffort = normalizeReasoningEffortValue(s.settings.String(AISettingReasoningEffort))
	cfg.ModelParameters = jsonMapFromSetting(s.settings.Get(AISettingModelParameters))
	cfg.LogHTTPDetails = s.settings.Bool(AISettingLogHTTPDetails)
	cfg.ContentFormat = contentFormatConfigFromSettings(s.settings, defaults.ContentFormat)
	cfg.AutoComment = autoCommentConfigFromSettings(s.settings, defaults.AutoComment)
	cfg.CommentReply = commentReplyConfigFromSettings(s.settings, defaults.CommentReply)
	cfg.Moderation = moderationConfigFromSettings(s.settings, defaults.Moderation)
	cfg.PublishGeneration = publishGenerationConfigFromSettings(s.settings, defaults.PublishGeneration)
	return cfg
}

func defaultAIConfig() AIConfig {
	return AIConfig{
		Enabled:                  false,
		BaseURL:                  "https://api.openai.com/v1",
		Model:                    "gpt-5-mini",
		ExtraHeaders:             map[string]string{},
		TimeoutSeconds:           60,
		MaxRunSeconds:            3600,
		ChunkMaxChars:            3000,
		Concurrency:              5,
		Temperature:              0.4,
		MaxOutputTokens:          8192,
		Templates:                DefaultAIPromptTemplates(),
		ShowReasoning:            false,
		ThinkingParameterEnabled: false,
		ThinkingEnabled:          false,
		ReasoningEffort:          "",
		ModelParameters:          map[string]any{},
		LogHTTPDetails:           false,
		ContentFormat:            defaultAIContentFormatConfig(),
		AutoComment: AIAutoCommentConfig{
			Enabled:            false,
			BotUserID:          0,
			BotUserIDMin:       0,
			BotUserIDMax:       0,
			TemplateKey:        "post_auto_comment",
			DelaySeconds:       10,
			MaxImages:          4,
			ImageSelectionMode: AIImageSelectionOrdered,
			Style:              "normal",
		},
		CommentReply:      defaultAICommentReplyConfig(),
		Moderation:        defaultAIModerationConfig(),
		PublishGeneration: defaultAIPublishGenerationConfig(),
	}
}

func (s *AIService) UpdateSettings(ctx context.Context, input map[string]any) error {
	if s == nil || s.settings == nil {
		return AIError{Code: "error.ai_settings_unavailable"}
	}
	cfg := s.Config()
	updates := map[string]any{}
	if value, ok := input["enabled"]; ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingEnabled] = parsed
	}
	if value, ok := firstPresent(input, "baseUrl", "base_url"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingBaseURL] = strings.TrimRight(text, "/")
	}
	if value, ok := firstPresent(input, "apiKey", "api_key"); ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && !strings.Contains(text, "••") && !strings.Contains(text, "***") {
			updates[AISettingAPIKey] = text
		}
	}
	if value, ok := firstPresent(input, "clearApiKey", "clear_api_key"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		if parsed {
			updates[AISettingAPIKey] = ""
		}
	}
	if value, ok := input["model"]; ok {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingModel] = text
	}
	if value, ok := firstPresent(input, "extraHeaders", "extra_headers"); ok {
		headers, valid := normalizeHeaderSetting(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingExtraHeaders] = headers
	}
	if value, ok := firstPresent(input, "timeoutSeconds", "timeout_seconds"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 5 || parsed > 300 {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingTimeoutSeconds] = parsed
	}
	if value, ok := firstPresent(input, "maxRunSeconds", "max_run_seconds"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 0 || parsed > 86400 {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingMaxRunSeconds] = parsed
	}
	if value, ok := firstPresent(input, "chunkMaxChars", "chunk_max_chars"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 0 || (parsed > 0 && parsed < 500) || parsed > 20000 {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingChunkMaxChars] = parsed
	}
	if value, ok := input["concurrency"]; ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 1 || parsed > 50 {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingConcurrency] = parsed
	}
	if value, ok := input["temperature"]; ok {
		parsed, valid := floatSettingInput(value)
		if !valid || parsed < 0 || parsed > 2 {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingTemperature] = parsed
	}
	if value, ok := firstPresent(input, "maxOutputTokens", "max_output_tokens"); ok {
		parsed, valid := intSettingInput(value)
		if !valid || parsed < 0 || (parsed > 0 && parsed < 128) {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingMaxOutputTokens] = parsed
	}
	if value, ok := firstPresent(input, "showReasoning", "show_reasoning"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingShowReasoning] = parsed
	}
	if value, ok := firstPresent(input, "thinkingParameterEnabled", "thinking_parameter_enabled"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingThinkingParameterEnabled] = parsed
	}
	if value, ok := firstPresent(input, "thinkingEnabled", "thinking_enabled"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingThinkingEnabled] = parsed
	}
	if value, ok := firstPresent(input, "reasoningEffort", "reasoning_effort"); ok {
		parsed, valid := normalizeReasoningEffortSetting(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingReasoningEffort] = parsed
	}
	if value, ok := firstPresent(input, "modelParameters", "model_parameters"); ok {
		parsed, valid := normalizeModelParametersSetting(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingModelParameters] = parsed
	}
	if value, ok := firstPresent(input, "logHttpDetails", "log_http_details"); ok {
		parsed, valid := boolSettingInput(value)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingLogHTTPDetails] = parsed
	}
	if value, ok := firstPresent(input, "contentFormat", "content_format"); ok {
		contentFormat, valid := normalizeContentFormatSetting(value, cfg.ContentFormat)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingContentFormat] = contentFormat
	}
	if value, ok := input["templates"]; ok {
		templates, valid := normalizeTemplatesSetting(value, cfg.Templates)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingPromptTemplates] = templates
	}
	if value, ok := firstPresent(input, "autoComment", "auto_comment"); ok {
		autoComment, valid := normalizeAutoCommentSetting(value, cfg.AutoComment)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingAutoCommentEnabled] = autoComment.Enabled
		updates[AISettingAutoCommentBotUserID] = autoComment.BotUserID
		updates[AISettingAutoCommentBotUserIDMin] = autoComment.BotUserIDMin
		updates[AISettingAutoCommentBotUserIDMax] = autoComment.BotUserIDMax
		updates[AISettingAutoCommentTemplateKey] = autoComment.TemplateKey
		updates[AISettingAutoCommentDelaySeconds] = autoComment.DelaySeconds
		updates[AISettingAutoCommentMaxImages] = autoComment.MaxImages
		updates[AISettingAutoCommentImageMode] = autoComment.ImageSelectionMode
		updates[AISettingAutoCommentStyle] = autoComment.Style
	}
	if value, ok := firstPresent(input, "commentReply", "comment_reply"); ok {
		commentReply, valid := normalizeCommentReplySetting(value, cfg.CommentReply)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingCommentReply] = commentReply
	}
	if value, ok := firstPresent(input, "moderation"); ok {
		moderation, valid := normalizeModerationSetting(value, cfg.Moderation)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingModeration] = moderation
	}
	if value, ok := firstPresent(input, "publishGeneration", "publish_generation"); ok {
		publishGeneration, valid := normalizePublishGenerationSetting(value, cfg.PublishGeneration)
		if !valid {
			return AIError{Code: "error.invalid_ai_setting"}
		}
		updates[AISettingPublishGeneration] = publishGeneration
	}
	for key, value := range updates {
		if !s.settings.Set(ctx, key, value) {
			return AIError{Code: "error.ai_settings_save_failed"}
		}
	}
	return nil
}

func (s *AIService) RunStream(ctx context.Context, req AIRequest, actor AIActor, emit func(AIStreamEvent) error) error {
	return s.runStreamWithConfig(ctx, s.Config(), req, actor, emit)
}

func (s *AIService) runStreamWithConfig(ctx context.Context, cfg AIConfig, req AIRequest, actor AIActor, emit func(AIStreamEvent) error) error {
	req = sanitizeAIRequest(req)
	if !cfg.Enabled {
		return AIError{Code: "error.ai_disabled"}
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return AIError{Code: "error.ai_api_key_missing"}
	}
	req.Type = strings.TrimSpace(req.Type)
	if req.Type == "" {
		req.Type = AITaskFormatMarkdown
	}
	if req.Locale == "" {
		req.Locale = "en"
	}
	var contentFormatTarget AIContentFormatTargetConfig
	if target, ok := contentFormatTargetForTask(cfg.ContentFormat, req.Type); ok {
		contentFormatTarget = target
		if !cfg.ContentFormat.Enabled || !target.Enabled {
			return AIError{Code: "error.ai_template_disabled"}
		}
		if strings.TrimSpace(target.TemplateKey) != "" {
			req.TemplateKey = strings.TrimSpace(target.TemplateKey)
		}
	}
	customPrompt := ""
	if value, ok := req.Variables["customPrompt"]; ok {
		customPrompt = strings.TrimSpace(fmt.Sprint(value))
	}
	if strings.TrimSpace(req.Input) == "" && len(req.Images) == 0 && !(req.Type == AITaskPostCustomGenerate && customPrompt != "") {
		return AIError{Code: "error.ai_input_required"}
	}
	req.Images = normalizeAIImageInputs(req.Images)
	templateKey := strings.TrimSpace(req.TemplateKey)
	if templateKey == "" {
		templateKey = defaultTemplateForTask(req.Type)
	}
	if contentFormatTarget.TemplateKey != "" {
		templateKey = strings.TrimSpace(contentFormatTarget.TemplateKey)
		req.TemplateKey = templateKey
	}
	tmpl, ok := cfg.Templates[templateKey]
	if !ok || !tmpl.Enabled {
		return AIError{Code: "error.ai_template_missing"}
	}
	effectiveCfg := applyAITemplateRuntimeOverrides(cfg, tmpl)
	if len(req.Images) > 0 && !tmpl.SupportsVision {
		return AIError{Code: "error.ai_template_no_vision"}
	}
	jobID := uuid.NewString()
	startedAt := time.Now()
	chunks := []string{req.Input}
	if isAITextTransformTask(req.Type) || isAITextTransformTask(tmpl.TaskType) {
		chunks = SplitAITextChunks(req.Input, cfg.ChunkMaxChars)
	}
	if len(chunks) == 0 {
		chunks = []string{req.Input}
	}
	req.TemplateKey = templateKey
	continuationPlan := aiCustomContinuationPlanForRequest(req, contentFormatTarget)
	chunkChars := make([]int, len(chunks))
	totalChars := 0
	for index, chunk := range chunks {
		chunkChars[index] = utf8.RuneCountInString(chunk)
		totalChars += chunkChars[index]
	}
	logID := s.createGenerationLog(ctx, jobID, req, actor, effectiveCfg, tmpl, len(chunks), startedAt)
	var usage AIUsage
	upstreamAttempts := []map[string]any{}
	fallbacks := []map[string]any{}
	outputs := make([]string, 0, len(chunks))
	status := "completed"
	errorCode := ""
	errorMessage := ""
	estimatedTokens := estimateAIRequestTokens(req, effectiveCfg, tmpl, chunks)
	processingStartedAt := time.Time{}
	tokensPerSecond := 0.0
	defer func() {
		duration := time.Since(startedAt).Milliseconds()
		if status == "completed" && !processingStartedAt.IsZero() {
			s.recordAIThroughput(usage, estimatedTokens, time.Since(processingStartedAt))
			tokensPerSecond = calculateAITokensPerSecond(usage, estimatedTokens, time.Since(processingStartedAt))
		}
		s.finishGenerationLog(context.Background(), logID, status, strings.Join(outputs, "\n\n"), usage, errorCode, errorMessage, duration, tokensPerSecond, upstreamAttempts, fallbacks)
	}()

	release, err := s.acquireAIConcurrency(ctx, effectiveCfg, tmpl, templateKey, jobID, len(chunks), estimatedTokens, emit)
	if err != nil {
		status = statusFromError(err)
		errorCode = aiErrorCode(err)
		errorMessage = err.Error()
		return err
	}
	defer release()
	processingStartedAt = time.Now()

	if err := emit(AIStreamEvent{Type: "progress", JobID: jobID, Percent: 0, CurrentChunk: 0, TotalChunks: len(chunks), ProcessedChars: 0, TotalChars: totalChars, Stage: "starting", EstimatedTokens: estimatedTokens}); err != nil {
		status = "canceled"
		return err
	}
	state, err := s.runAIChunksWithContinuation(ctx, effectiveCfg, tmpl, req, jobID, chunks, chunkChars, totalChars, continuationPlan, emit)
	usage = state.usage
	upstreamAttempts = state.upstreamAttempts
	fallbacks = state.fallbacks
	outputs = state.outputs
	if err != nil {
		status = statusFromError(err)
		errorCode = aiErrorCode(err)
		errorMessage = err.Error()
		return err
	}
	finalText := sanitizeAITextOutput(strings.Join(outputs, "\n\n"), req.Type)
	summary := aiFinalSummary(len(chunks), totalChars, templateKey, req.Type, len(req.Images), fallbacks)
	if continuationPlan.Enabled {
		summary["continuationEnabled"] = true
		summary["continuationMaxRounds"] = continuationPlan.MaxRounds
		summary["continuationContextChars"] = continuationPlan.ContextChars
		summary["continuationRounds"] = state.continuationRounds
	}
	return emit(AIStreamEvent{
		Type:            "final",
		JobID:           jobID,
		Text:            finalText,
		Usage:           &usage,
		TokensPerSecond: calculateAITokensPerSecond(usage, estimatedTokens, time.Since(processingStartedAt)),
		Summary:         summary,
	})
}

func (s *AIService) RunText(ctx context.Context, req AIRequest, actor AIActor) (string, *AIUsage, error) {
	return s.runTextWithConfig(ctx, s.Config(), req, actor)
}

func (s *AIService) RunTextWithTemplateOverride(ctx context.Context, req AIRequest, actor AIActor, templateKey string, tmpl AITemplateConfig) (string, *AIUsage, error) {
	cfg := s.Config()
	templateKey = strings.TrimSpace(templateKey)
	if templateKey == "" {
		templateKey = "__debug_template"
	}
	if tmpl.TaskType == "" {
		tmpl.TaskType = req.Type
	}
	if tmpl.UserPrompt == "" && tmpl.Prompt != "" {
		tmpl.UserPrompt = tmpl.Prompt
	}
	if tmpl.Prompt == "" {
		tmpl.Prompt = tmpl.UserPrompt
	}
	if tmpl.Style == "" {
		tmpl.Style = "normal"
	}
	tmpl.Enabled = true
	cfg.Templates = cloneTemplates(cfg.Templates)
	cfg.Templates[templateKey] = tmpl
	req.TemplateKey = templateKey
	return s.runTextWithConfig(ctx, cfg, req, actor)
}

func (s *AIService) runTextWithConfig(ctx context.Context, cfg AIConfig, req AIRequest, actor AIActor) (string, *AIUsage, error) {
	var finalText string
	var finalUsage *AIUsage
	err := s.runStreamWithConfig(ctx, cfg, req, actor, func(event AIStreamEvent) error {
		if event.Type == "final" {
			finalText = event.Text
			finalUsage = event.Usage
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	return finalText, finalUsage, nil
}

func isRetryableAIStreamError(err error) bool {
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		return false
	}
	switch aiErr.Code {
	case "error.ai_timeout", "error.ai_upstream_unavailable", "error.ai_upstream_error", "error.ai_stream_decode_failed":
		return true
	default:
		return false
	}
}
