package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *AIService) streamOpenAIChunk(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, onDelta func(string) error, onReasoning func(string) error, usage *AIUsage, diagnostics *[]map[string]any, onUpstream aiUpstreamEventEmitter) error {
	idleTimeoutSeconds := cfg.TimeoutSeconds
	if req.Options.TimeoutSeconds != nil {
		idleTimeoutSeconds = boundedInt(*req.Options.TimeoutSeconds, 1, 300, cfg.TimeoutSeconds)
	}
	requestCtx := ctx
	cancel := func() {}
	if cfg.MaxRunSeconds > 0 {
		var cancelWithTimeout context.CancelFunc
		requestCtx, cancelWithTimeout = context.WithTimeout(ctx, time.Duration(cfg.MaxRunSeconds)*time.Second)
		cancel = cancelWithTimeout
	}
	defer cancel()

	temperature := cfg.Temperature
	if tmpl.Temperature > 0 {
		temperature = tmpl.Temperature
	}
	if req.Options.Temperature != nil {
		temperature = *req.Options.Temperature
	}
	maxOutputTokens := cfg.MaxOutputTokens
	if tmpl.MaxOutputTokens > 0 || tmpl.maxOutputSet {
		maxOutputTokens = tmpl.MaxOutputTokens
	}
	if req.Options.MaxOutputTokens != nil {
		maxOutputTokens = *req.Options.MaxOutputTokens
	}
	structuredJSON := tmpl.StructuredJSON
	if req.Options.StructuredJSON != nil {
		structuredJSON = *req.Options.StructuredJSON
	}
	model := nonEmptyString(strings.TrimSpace(tmpl.Model), cfg.Model)

	params := openAIRequestParams{
		model:              model,
		temperature:        temperature,
		maxOutputTokens:    maxOutputTokens,
		structuredJSON:     structuredJSON,
		includeThinking:    cfg.ThinkingParameterEnabled,
		thinkingEnabled:    cfg.ThinkingEnabled,
		reasoningEffort:    cfg.ReasoningEffort,
		modelParameters:    cfg.ModelParameters,
		forceTextMessages:  false,
		nvidiaChatTemplate: usesNVIDIAChatTemplateThinking(cfg.BaseURL, model),
	}
	resp, requestLog, err := s.doOpenAIChatCompletions(requestCtx, cfg, tmpl, req, params)
	if err != nil {
		emitOpenAIRequestError(onUpstream, cfg, params, requestLog, err)
		appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, 0, "", aiOpenAIResponseSummary{}, err))
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return AIError{Code: "error.ai_timeout", Err: err}
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
		}
		return AIError{Code: "error.ai_upstream_unavailable", Err: err}
	}
	defer resp.Body.Close()
	emitOpenAIResponseStart(onUpstream, cfg, params, requestLog, resp)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := sanitizeAIUpstreamDetail(string(data))
		emitOpenAIResponseError(onUpstream, resp, detail)
		appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), aiOpenAIResponseSummary{RawSummary: detail}, nil))
		if shouldRetryOpenAITextOnly(req, resp.StatusCode, detail) {
			resp.Body.Close()
			params.forceTextMessages = true
			resp, requestLog, err = s.doOpenAIChatCompletions(requestCtx, cfg, tmpl, req, params)
			if err != nil {
				emitOpenAIRequestError(onUpstream, cfg, params, requestLog, err)
				appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, 0, "", aiOpenAIResponseSummary{}, err))
				if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
					return AIError{Code: "error.ai_timeout", Err: err}
				}
				if errors.Is(ctx.Err(), context.Canceled) {
					return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
				}
				return AIError{Code: "error.ai_upstream_unavailable", Err: err}
			}
			defer resp.Body.Close()
			emitOpenAIResponseStart(onUpstream, cfg, params, requestLog, resp)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				summary, err := parseOpenAIStream(requestCtx, resp.Body, time.Duration(idleTimeoutSeconds)*time.Second, onDelta, onReasoning, usage, onUpstream)
				appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), summary, err))
				return err
			}
			data, _ = io.ReadAll(io.LimitReader(resp.Body, 4096))
			detail = sanitizeAIUpstreamDetail(string(data))
			emitOpenAIResponseError(onUpstream, resp, detail)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), aiOpenAIResponseSummary{RawSummary: detail}, nil))
		}
		return AIError{
			Code:           "error.ai_upstream_error",
			Err:            fmt.Errorf("upstream status %d: %s", resp.StatusCode, detail),
			UpstreamStatus: resp.StatusCode,
			UpstreamDetail: detail,
		}
	}
	summary, err := parseOpenAIStream(requestCtx, resp.Body, time.Duration(idleTimeoutSeconds)*time.Second, onDelta, onReasoning, usage, onUpstream)
	appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), summary, err))
	if err == nil && shouldRetryOpenAIWithoutThinking(summary, req) {
		resp.Body.Close()
		params = openAIWithoutThinkingParams(params)
		resp, requestLog, err = s.doOpenAIChatCompletions(requestCtx, cfg, tmpl, req, params)
		if err != nil {
			emitOpenAIRequestError(onUpstream, cfg, params, requestLog, err)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, 0, "", aiOpenAIResponseSummary{}, err))
			if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
				return AIError{Code: "error.ai_timeout", Err: err}
			}
			if errors.Is(ctx.Err(), context.Canceled) {
				return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
			}
			return AIError{Code: "error.ai_upstream_unavailable", Err: err}
		}
		defer resp.Body.Close()
		emitOpenAIResponseStart(onUpstream, cfg, params, requestLog, resp)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			detail := sanitizeAIUpstreamDetail(string(data))
			emitOpenAIResponseError(onUpstream, resp, detail)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), aiOpenAIResponseSummary{RawSummary: detail}, nil))
			return AIError{
				Code:           "error.ai_upstream_error",
				Err:            fmt.Errorf("upstream status %d: %s", resp.StatusCode, detail),
				UpstreamStatus: resp.StatusCode,
				UpstreamDetail: detail,
			}
		}
		summary, err = parseOpenAIStream(requestCtx, resp.Body, time.Duration(idleTimeoutSeconds)*time.Second, onDelta, onReasoning, usage, onUpstream)
		appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), summary, err))
	}
	if err != nil && openAIStreamOutputEmpty(summary) && shouldRetryOpenAITextOnlyAfterStreamError(req, err) {
		resp.Body.Close()
		params.forceTextMessages = true
		resp, requestLog, err = s.doOpenAIChatCompletions(requestCtx, cfg, tmpl, req, params)
		if err != nil {
			emitOpenAIRequestError(onUpstream, cfg, params, requestLog, err)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, 0, "", aiOpenAIResponseSummary{}, err))
			if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
				return AIError{Code: "error.ai_timeout", Err: err}
			}
			if errors.Is(ctx.Err(), context.Canceled) {
				return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
			}
			return AIError{Code: "error.ai_upstream_unavailable", Err: err}
		}
		defer resp.Body.Close()
		emitOpenAIResponseStart(onUpstream, cfg, params, requestLog, resp)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			detail := sanitizeAIUpstreamDetail(string(data))
			emitOpenAIResponseError(onUpstream, resp, detail)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), aiOpenAIResponseSummary{RawSummary: detail}, nil))
			return AIError{
				Code:           "error.ai_upstream_error",
				Err:            fmt.Errorf("upstream status %d: %s", resp.StatusCode, detail),
				UpstreamStatus: resp.StatusCode,
				UpstreamDetail: detail,
			}
		}
		summary, err = parseOpenAIStream(requestCtx, resp.Body, time.Duration(idleTimeoutSeconds)*time.Second, onDelta, onReasoning, usage, onUpstream)
		appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), summary, err))
	}
	if err != nil && openAIStreamOutputEmpty(summary) && shouldRetryOpenAIWithoutStructuredJSON(params, err) {
		resp.Body.Close()
		params.structuredJSON = false
		resp, requestLog, err = s.doOpenAIChatCompletions(requestCtx, cfg, tmpl, req, params)
		if err != nil {
			emitOpenAIRequestError(onUpstream, cfg, params, requestLog, err)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, 0, "", aiOpenAIResponseSummary{}, err))
			if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
				return AIError{Code: "error.ai_timeout", Err: err}
			}
			if errors.Is(ctx.Err(), context.Canceled) {
				return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
			}
			return AIError{Code: "error.ai_upstream_unavailable", Err: err}
		}
		defer resp.Body.Close()
		emitOpenAIResponseStart(onUpstream, cfg, params, requestLog, resp)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			detail := sanitizeAIUpstreamDetail(string(data))
			emitOpenAIResponseError(onUpstream, resp, detail)
			appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), aiOpenAIResponseSummary{RawSummary: detail}, nil))
			return AIError{
				Code:           "error.ai_upstream_error",
				Err:            fmt.Errorf("upstream status %d: %s", resp.StatusCode, detail),
				UpstreamStatus: resp.StatusCode,
				UpstreamDetail: detail,
			}
		}
		summary, err = parseOpenAIStream(requestCtx, resp.Body, time.Duration(idleTimeoutSeconds)*time.Second, onDelta, onReasoning, usage, onUpstream)
		appendAIUpstreamAttempt(diagnostics, openAIRequestAttemptLog(cfg, params, requestLog, resp.StatusCode, resp.Header.Get("Content-Type"), summary, err))
	}
	return err
}

type openAIRequestParams struct {
	model              string
	temperature        float64
	maxOutputTokens    int
	structuredJSON     bool
	includeThinking    bool
	thinkingEnabled    bool
	reasoningEffort    string
	modelParameters    map[string]any
	forceTextMessages  bool
	nvidiaChatTemplate bool
}

func (s *AIService) doOpenAIChatCompletions(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, params openAIRequestParams) (*http.Response, map[string]any, error) {
	body := openAIRequestBody(tmpl, req, params)
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	targetURL := openAIChatCompletionsURL(cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	for key, value := range cfg.ExtraHeaders {
		key = strings.TrimSpace(key)
		if key == "" || strings.EqualFold(key, "authorization") {
			continue
		}
		httpReq.Header.Set(key, value)
	}
	requestLog := openAIRequestLog(targetURL, httpReq.Header, body, cfg.LogHTTPDetails)
	resp, err := s.client.Do(httpReq)
	return resp, requestLog, err
}

func openAIRequestBody(tmpl AITemplateConfig, req AIRequest, params openAIRequestParams) map[string]any {
	body := map[string]any{
		"model":          params.model,
		"messages":       openAIMessagesWithMode(tmpl, req, params.forceTextMessages),
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	mergeAIModelParameters(body, params.modelParameters)
	body["temperature"] = params.temperature
	if params.maxOutputTokens > 0 {
		body["max_tokens"] = params.maxOutputTokens
	}
	if params.structuredJSON {
		body["response_format"] = map[string]any{"type": "json_object"}
	}
	if params.includeThinking {
		if params.nvidiaChatTemplate {
			setOpenAIChatTemplateThinking(body, params.thinkingEnabled)
		} else {
			body["enable_thinking"] = params.thinkingEnabled
		}
	}
	if params.reasoningEffort != "" {
		body["reasoning_effort"] = params.reasoningEffort
	}
	return body
}

func sanitizeAIUpstreamDetail(value string) string {
	value = strings.TrimSpace(sanitizeAIDBText(value))
	if value == "" {
		return ""
	}
	var payload any
	if json.Unmarshal([]byte(value), &payload) == nil {
		value = sanitizeAIUpstreamJSON(payload)
	}
	replacers := []string{"api_key", "apiKey", "authorization", "Authorization", "bearer", "Bearer", "token", "secret", "password"}
	for _, key := range replacers {
		value = strings.ReplaceAll(value, key, "[redacted]")
	}
	if len(value) > 1200 {
		value = value[:1200]
	}
	return value
}

func sanitizeAIUpstreamJSON(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") {
				typed[key] = "[redacted]"
				continue
			}
			typed[key] = sanitizeAIUpstreamJSONValue(item)
		}
		data, err := json.Marshal(typed)
		if err == nil {
			return string(data)
		}
	case []any:
		for index, item := range typed {
			typed[index] = sanitizeAIUpstreamJSONValue(item)
		}
		data, err := json.Marshal(typed)
		if err == nil {
			return string(data)
		}
	}
	data, err := json.Marshal(value)
	if err == nil {
		return string(data)
	}
	return fmt.Sprint(value)
}

func sanitizeAIUpstreamJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") {
				typed[key] = "[redacted]"
				continue
			}
			typed[key] = sanitizeAIUpstreamJSONValue(item)
		}
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = sanitizeAIUpstreamJSONValue(item)
		}
		return typed
	default:
		return typed
	}
}

func openAIMessages(tmpl AITemplateConfig, req AIRequest) []map[string]any {
	return openAIMessagesWithMode(tmpl, req, false)
}

func openAIMessagesWithMode(tmpl AITemplateConfig, req AIRequest, forceText bool) []map[string]any {
	systemPrompt := strings.TrimSpace(aiTemplateSystemPrompt(tmpl, req))
	userPrompt := aiTemplateUserPrompt(tmpl, req)
	messages := make([]map[string]any, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": systemPrompt})
	}
	images := normalizeAIImageInputs(req.Images)
	if len(images) == 0 || forceText {
		if forceText && len(images) > 0 {
			userPrompt = openAITextFallbackPrompt(userPrompt, images)
		}
		messages = append(messages, map[string]any{"role": "user", "content": userPrompt})
		return messages
	}
	content := []map[string]any{{"type": "text", "text": userPrompt}}
	for _, image := range images {
		imageURL := aiImageModelURL(image)
		content = append(content, map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": imageURL},
		})
	}
	messages = append(messages, map[string]any{"role": "user", "content": content})
	return messages
}

func openAITextFallbackPrompt(userPrompt string, images []AIImageInput) string {
	var builder strings.Builder
	builder.WriteString(userPrompt)
	builder.WriteString("\n\n图片说明：当前上游不接受 Chat Completions 多模态图片消息，本次已自动改用纯文本兼容模式。")
	builder.WriteString("请只基于用户已填写的标题、详情、标签和下列图片元数据生成内容；不要编造图片中无法从文字判断的具体画面细节。")
	builder.WriteString("\n可用图片数量：")
	builder.WriteString(fmt.Sprint(len(images)))
	for index, image := range images {
		builder.WriteString("\n- 图片 ")
		builder.WriteString(fmt.Sprint(index + 1))
		if image.Alt != "" {
			builder.WriteString("，文件/说明：")
			builder.WriteString(image.Alt)
		}
		if image.Mime != "" {
			builder.WriteString("，类型：")
			builder.WriteString(image.Mime)
		}
		if image.URL != "" {
			builder.WriteString("，来源：已上传远程图片")
		} else if image.DataURL != "" {
			builder.WriteString("，来源：内联图片数据")
		}
	}
	return builder.String()
}

func shouldRetryOpenAITextOnly(req AIRequest, status int, detail string) bool {
	if len(normalizeAIImageInputs(req.Images)) == 0 {
		return false
	}
	if requiresOpenAIVisionAnalysis(req) {
		return false
	}
	if status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
		return false
	}
	lower := strings.ToLower(detail)
	markers := []string{
		"chatcompletionrequestusermessagecontent",
		"message content",
		"content did not match",
		"invalid message content",
		"invalid type",
		"image_url",
		"image url",
		"unsupported content",
		"multi-modal",
		"multimodal",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func requiresOpenAIVisionAnalysis(req AIRequest) bool {
	if len(normalizeAIImageInputs(req.Images)) == 0 {
		return false
	}
	return req.Type == AITaskPublishTitleGenerate ||
		req.Type == AITaskPublishDetailGenerate ||
		req.Type == AITaskPublishTitleDetailGenerate
}

func normalizeAIImageInputs(images []AIImageInput) []AIImageInput {
	out := make([]AIImageInput, 0, len(images))
	for _, image := range images {
		url := strings.TrimSpace(image.URL)
		dataURL := strings.TrimSpace(image.DataURL)
		if dataURL == "" && url == "" {
			continue
		}
		out = append(out, AIImageInput{
			URL:     url,
			DataURL: dataURL,
			Mime:    strings.TrimSpace(image.Mime),
			Alt:     strings.TrimSpace(image.Alt),
		})
	}
	return out
}

func aiImageModelURL(image AIImageInput) string {
	if dataURL := strings.TrimSpace(image.DataURL); dataURL != "" {
		return dataURL
	}
	return strings.TrimSpace(image.URL)
}

func mergeAIModelParameters(body map[string]any, params map[string]any) {
	for key, value := range params {
		key = strings.TrimSpace(key)
		if key == "" || value == nil || aiModelParameterReserved(key) {
			continue
		}
		body[key] = value
	}
}

func aiModelParameterReserved(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "model", "messages", "stream", "stream_options":
		return true
	default:
		return false
	}
}

type aiOpenAIResponseSummary struct {
	RawSummary       string
	OutputSummary    string
	ReasoningSummary string
	Usage            AIUsage
	StreamDataEvents int
	Done             bool
	NonStreamJSON    bool
}

func parseOpenAIStream(ctx context.Context, body io.ReadCloser, idleTimeout time.Duration, onDelta func(string) error, onReasoning func(string) error, usage *AIUsage, onUpstream aiUpstreamEventEmitter) (aiOpenAIResponseSummary, error) {
	var summary aiOpenAIResponseSummary
	var rawPreview strings.Builder
	var nonStreamRaw strings.Builder
	var output strings.Builder
	var reasoning strings.Builder
	streamCtx, cancelIdle, touchIdle := openAIStreamIdleContext(ctx, idleTimeout, body)
	defer cancelIdle()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		touchIdle()
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			appendLimitedAIText(&nonStreamRaw, line+"\n", 1024*1024)
			appendLimitedAIText(&rawPreview, line+"\n", 6000)
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		summary.StreamDataEvents++
		appendLimitedAIText(&rawPreview, data+"\n", 6000)
		emitAIUpstreamEvent(onUpstream, "stream_data", map[string]any{
			"eventIndex": summary.StreamDataEvents,
			"data":       summarizeAIText(sanitizeAIUpstreamDetail(data), 4000),
		})
		if data == "[DONE]" {
			summary.Done = true
			summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
			summary.OutputSummary = summarizeAIText(output.String(), 4000)
			summary.ReasoningSummary = summarizeAIText(reasoning.String(), 2000)
			emitAIUpstreamEvent(onUpstream, "stream_done", map[string]any{
				"streamDataEvents": summary.StreamDataEvents,
			})
			return summary, nil
		}
		var payload openAIStreamPayload
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
			emitAIUpstreamEvent(onUpstream, "decode_error", map[string]any{
				"errorCode":    "error.ai_stream_decode_failed",
				"errorMessage": summarizeAIText(err.Error(), 1200),
			})
			return summary, AIError{Code: "error.ai_stream_decode_failed", Err: err}
		}
		if payload.Error != nil {
			summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
			summary.OutputSummary = summarizeAIText(output.String(), 4000)
			summary.ReasoningSummary = summarizeAIText(reasoning.String(), 2000)
			emitAIUpstreamEvent(onUpstream, "payload_error", map[string]any{
				"detail": summarizeAIText(sanitizeAIUpstreamDetail(data), 1200),
			})
			return summary, openAIErrorFromPayload(data, payload.Error)
		}
		if payload.Usage != nil {
			summary.Usage.PromptTokens += payload.Usage.PromptTokens
			summary.Usage.CompletionTokens += payload.Usage.CompletionTokens
			summary.Usage.TotalTokens += payload.Usage.TotalTokens
			if usage != nil {
				usage.PromptTokens += payload.Usage.PromptTokens
				usage.CompletionTokens += payload.Usage.CompletionTokens
				usage.TotalTokens += payload.Usage.TotalTokens
			}
			emitAIUpstreamEvent(onUpstream, "usage", map[string]any{
				"usage": aiUsageEventPayload(&summary.Usage),
			})
		}
		for _, choice := range payload.Choices {
			reasoningDelta := firstNonEmptyAIText(
				choice.Delta.ReasoningContent.String(),
				choice.Delta.Reasoning.String(),
				choice.Message.ReasoningContent.String(),
				choice.Message.Reasoning.String(),
			)
			if reasoningDelta != "" {
				appendLimitedAIText(&reasoning, reasoningDelta, 4000)
				emitAIUpstreamEvent(onUpstream, "decoded_reasoning", map[string]any{
					"reasoning": summarizeAIText(reasoningDelta, 1000),
				})
				if onReasoning != nil {
					if err := onReasoning(reasoningDelta); err != nil {
						return summary, err
					}
				}
			}
			delta := firstNonEmptyAIText(
				choice.Delta.Content.String(),
				choice.Message.Content.String(),
				choice.Text.String(),
			)
			if delta == "" {
				continue
			}
			appendLimitedAIText(&output, delta, 8000)
			emitAIUpstreamEvent(onUpstream, "decoded_delta", map[string]any{
				"delta": summarizeAIText(delta, 2000),
			})
			if err := onDelta(delta); err != nil {
				return summary, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
		if err := openAIStreamContextError(ctx, streamCtx); err != nil {
			emitAIUpstreamEvent(onUpstream, "stream_error", map[string]any{
				"errorCode":    aiErrorCode(err),
				"errorMessage": summarizeAIText(err.Error(), 1200),
			})
			return summary, err
		}
		emitAIUpstreamEvent(onUpstream, "decode_error", map[string]any{
			"errorCode":    "error.ai_stream_decode_failed",
			"errorMessage": summarizeAIText(err.Error(), 1200),
		})
		return summary, AIError{Code: "error.ai_stream_decode_failed", Err: err}
	}
	if err := openAIStreamContextError(ctx, streamCtx); err != nil {
		summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
		summary.OutputSummary = summarizeAIText(output.String(), 4000)
		summary.ReasoningSummary = summarizeAIText(reasoning.String(), 2000)
		emitAIUpstreamEvent(onUpstream, "stream_error", map[string]any{
			"errorCode":    aiErrorCode(err),
			"errorMessage": summarizeAIText(err.Error(), 1200),
		})
		return summary, err
	}
	if summary.StreamDataEvents == 0 && strings.TrimSpace(nonStreamRaw.String()) != "" {
		return parseOpenAINonStreamResponse(nonStreamRaw.String(), onDelta, onReasoning, usage)
	}
	summary.RawSummary = summarizeAIText(rawPreview.String(), 6000)
	summary.OutputSummary = summarizeAIText(output.String(), 4000)
	summary.ReasoningSummary = summarizeAIText(reasoning.String(), 2000)
	if summary.StreamDataEvents > 0 && !summary.Done {
		emitAIUpstreamEvent(onUpstream, "stream_error", map[string]any{
			"errorCode":    "error.ai_stream_decode_failed",
			"errorMessage": summarizeAIText(io.ErrUnexpectedEOF.Error(), 1200),
		})
		return summary, AIError{Code: "error.ai_stream_decode_failed", Err: io.ErrUnexpectedEOF}
	}
	return summary, nil
}

type openAIStreamPayload struct {
	Error   *openAIErrorPayload `json:"error"`
	Choices []struct {
		Delta struct {
			Content          aiDeltaContent `json:"content"`
			Reasoning        aiDeltaContent `json:"reasoning"`
			ReasoningContent aiDeltaContent `json:"reasoning_content"`
		} `json:"delta"`
		Message struct {
			Content          aiDeltaContent `json:"content"`
			Reasoning        aiDeltaContent `json:"reasoning"`
			ReasoningContent aiDeltaContent `json:"reasoning_content"`
		} `json:"message"`
		Text aiDeltaContent `json:"text"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAINonStreamPayload struct {
	Error   *openAIErrorPayload `json:"error"`
	Choices []struct {
		Message struct {
			Content          aiDeltaContent `json:"content"`
			Reasoning        aiDeltaContent `json:"reasoning"`
			ReasoningContent aiDeltaContent `json:"reasoning_content"`
		} `json:"message"`
		Text aiDeltaContent `json:"text"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func parseOpenAINonStreamResponse(raw string, onDelta func(string) error, onReasoning func(string) error, usage *AIUsage) (aiOpenAIResponseSummary, error) {
	summary := aiOpenAIResponseSummary{
		RawSummary:    summarizeAIText(raw, 6000),
		NonStreamJSON: true,
		Done:          true,
	}
	var payload openAINonStreamPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return summary, AIError{Code: "error.ai_stream_decode_failed", Err: err}
	}
	if payload.Error != nil {
		return summary, openAIErrorFromPayload(raw, payload.Error)
	}
	if payload.Usage != nil {
		summary.Usage = AIUsage{
			PromptTokens:     payload.Usage.PromptTokens,
			CompletionTokens: payload.Usage.CompletionTokens,
			TotalTokens:      payload.Usage.TotalTokens,
		}
		if usage != nil {
			usage.PromptTokens += payload.Usage.PromptTokens
			usage.CompletionTokens += payload.Usage.CompletionTokens
			usage.TotalTokens += payload.Usage.TotalTokens
		}
	}
	var output strings.Builder
	var reasoning strings.Builder
	for _, choice := range payload.Choices {
		reasoningDelta := firstNonEmptyAIText(
			choice.Message.ReasoningContent.String(),
			choice.Message.Reasoning.String(),
		)
		if reasoningDelta != "" {
			appendLimitedAIText(&reasoning, reasoningDelta, 4000)
			if onReasoning != nil {
				if err := onReasoning(reasoningDelta); err != nil {
					return summary, err
				}
			}
		}
		delta := firstNonEmptyAIText(choice.Message.Content.String(), choice.Text.String())
		if delta == "" {
			continue
		}
		appendLimitedAIText(&output, delta, 8000)
		if err := onDelta(delta); err != nil {
			return summary, err
		}
	}
	summary.OutputSummary = summarizeAIText(output.String(), 4000)
	summary.ReasoningSummary = summarizeAIText(reasoning.String(), 2000)
	return summary, nil
}

type aiDeltaContent string

func (c *aiDeltaContent) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*c = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*c = aiDeltaContent(text)
		return nil
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err == nil {
		for _, key := range []string{"content", "text", "summary"} {
			if value, ok := object[key].(string); ok {
				*c = aiDeltaContent(value)
				return nil
			}
		}
	}
	var list []map[string]any
	if err := json.Unmarshal(data, &list); err == nil {
		var builder strings.Builder
		for _, item := range list {
			text, _ := item["text"].(string)
			if text == "" {
				text, _ = item["content"].(string)
			}
			if text != "" {
				builder.WriteString(text)
			}
		}
		*c = aiDeltaContent(builder.String())
		return nil
	}
	return nil
}

func (c aiDeltaContent) String() string { return string(c) }
