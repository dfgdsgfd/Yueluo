package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	aiChunkPrimaryAttempts  = 3
	aiChunkSubpartAttempts  = 2
	aiChunkFallbackMinRunes = 900
)

type aiChunkRunOptions struct {
	ChunkIndex        int
	ChunkNumber       int
	TotalChunks       int
	CurrentChunkChars int
	ProcessedChars    int
	TotalChars        int
	Percent           int
	JobID             string
	Emit              func(AIStreamEvent) error
	Usage             *AIUsage
	UpstreamAttempts  *[]map[string]any
	ShowReasoning     bool
	StagePrefix       string
}

type aiChunkRunResult struct {
	Text      string
	Reasoning string
	Fallback  map[string]any
}

func (s *AIService) runAIChunkWithFallback(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, opts aiChunkRunOptions) (aiChunkRunResult, error) {
	result, err := s.runAIChunkAttempts(ctx, cfg, tmpl, req, opts, aiChunkPrimaryAttempts)
	if err == nil {
		return result, nil
	}
	if !shouldFallbackAIChunk(req.Type, tmpl.TaskType, err) {
		return aiChunkRunResult{}, err
	}
	if split := splitAIChunkForRetry(req.Input, cfg.ChunkMaxChars); len(split) > 1 {
		result, splitErr := s.runAIChunkSubparts(ctx, cfg, tmpl, req, opts, split, err)
		if splitErr == nil {
			return result, nil
		}
		err = splitErr
	}
	return aiChunkRunResult{
		Text: sanitizeAITextOutput(req.Input, req.Type),
		Fallback: map[string]any{
			"chunk":         opts.ChunkNumber,
			"chunkIndex":    opts.ChunkIndex,
			"code":          aiErrorCode(err),
			"message":       summarizeAIText(err.Error(), 600),
			"originalChars": utf8.RuneCountInString(req.Input),
			"mode":          "original_text",
		},
	}, nil
}

func (s *AIService) runAIChunkSubparts(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, opts aiChunkRunOptions, parts []string, originalErr error) (aiChunkRunResult, error) {
	texts := make([]string, 0, len(parts))
	reasonings := make([]string, 0, len(parts))
	for index, part := range parts {
		partReq := req
		partReq.Input = part
		partReq.Variables = cloneVariables(req.Variables)
		partReq.Variables["chunkSubIndex"] = index + 1
		partReq.Variables["totalChunkSubparts"] = len(parts)
		partReq.Variables["parentChunkIndex"] = opts.ChunkNumber
		result, err := s.runAIChunkAttempts(ctx, cfg, tmpl, partReq, opts, aiChunkSubpartAttempts)
		if err != nil {
			return aiChunkRunResult{}, err
		}
		texts = append(texts, result.Text)
		if strings.TrimSpace(result.Reasoning) != "" {
			reasonings = append(reasonings, result.Reasoning)
		}
	}
	return aiChunkRunResult{
		Text:      sanitizeAITextOutput(strings.Join(texts, "\n\n"), req.Type),
		Reasoning: strings.Join(reasonings, "\n\n"),
		Fallback: map[string]any{
			"chunk":          opts.ChunkNumber,
			"chunkIndex":     opts.ChunkIndex,
			"mode":           "split_retry",
			"subparts":       len(parts),
			"originalCode":   aiErrorCode(originalErr),
			"originalReason": summarizeAIText(originalErr.Error(), 600),
		},
	}, nil
}

func (s *AIService) runAIChunkAttempts(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, opts aiChunkRunOptions, attempts int) (aiChunkRunResult, error) {
	var lastErr error
	for attempt := range attempts {
		stage := "connecting"
		if attempt > 0 {
			stage = "retrying"
		}
		stage = opts.StagePrefix + stage
		if opts.Emit != nil {
			if err := opts.Emit(AIStreamEvent{
				Type:              "progress",
				JobID:             opts.JobID,
				Percent:           opts.Percent,
				CurrentChunk:      opts.ChunkNumber,
				TotalChunks:       opts.TotalChunks,
				CurrentChunkChars: opts.CurrentChunkChars,
				ProcessedChars:    opts.ProcessedChars,
				TotalChars:        opts.TotalChars,
				Stage:             stage,
			}); err != nil {
				return aiChunkRunResult{}, err
			}
		}
		result, err := s.runAIChunkAttempt(ctx, cfg, tmpl, req, opts)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !shouldRetryAIChunkAttempt(err, attempt, attempts) {
			break
		}
		if err := waitAIChunkRetry(ctx, attempt); err != nil {
			return aiChunkRunResult{}, err
		}
	}
	if lastErr == nil {
		lastErr = AIError{Code: "error.ai_request_failed"}
	}
	return aiChunkRunResult{}, lastErr
}

func (s *AIService) runAIChunkAttempt(ctx context.Context, cfg AIConfig, tmpl AITemplateConfig, req AIRequest, opts aiChunkRunOptions) (aiChunkRunResult, error) {
	var builder strings.Builder
	var reasoningBuilder strings.Builder
	streamFilter := newAIStreamOutputFilter()
	err := s.streamOpenAIChunk(ctx, cfg, tmpl, req, func(delta string) error {
		cleanDelta := streamFilter.Process(delta)
		if cleanDelta == "" {
			return nil
		}
		builder.WriteString(cleanDelta)
		if opts.Emit != nil {
			return opts.Emit(AIStreamEvent{
				Type:       "chunk_delta",
				JobID:      opts.JobID,
				ChunkIndex: opts.ChunkIndex,
				Delta:      cleanDelta,
			})
		}
		return nil
	}, func(delta string) error {
		reasoningBuilder.WriteString(delta)
		if opts.ShowReasoning && opts.Emit != nil && strings.TrimSpace(delta) != "" {
			return opts.Emit(AIStreamEvent{
				Type:           "reasoning_delta",
				JobID:          opts.JobID,
				ChunkIndex:     opts.ChunkIndex,
				ReasoningDelta: delta,
			})
		}
		return nil
	}, opts.Usage, opts.UpstreamAttempts, aiStreamUpstreamEmitter(opts.JobID, opts.ChunkIndex, opts.Emit))
	if err != nil {
		return aiChunkRunResult{}, err
	}
	if cleanDelta := streamFilter.Flush(); cleanDelta != "" {
		builder.WriteString(cleanDelta)
		if opts.Emit != nil {
			if err := opts.Emit(AIStreamEvent{
				Type:       "chunk_delta",
				JobID:      opts.JobID,
				ChunkIndex: opts.ChunkIndex,
				Delta:      cleanDelta,
			}); err != nil {
				return aiChunkRunResult{}, err
			}
		}
	}
	text := builder.String()
	if strings.TrimSpace(text) == "" && strings.TrimSpace(req.Input) != "" && shouldEchoBlankAIChunk(req.Type, tmpl.TaskType) {
		text = req.Input
	}
	return aiChunkRunResult{
		Text:      sanitizeAITextOutput(text, req.Type),
		Reasoning: reasoningBuilder.String(),
	}, nil
}

func shouldFallbackAIChunk(requestTaskType string, templateTaskType string, err error) bool {
	return shouldRetryWithOriginalAIChunk(err) && shouldEchoBlankAIChunk(requestTaskType, templateTaskType)
}

func shouldRetryAIChunkAttempt(err error, attempt int, attempts int) bool {
	if !isRetryableAIStreamError(err) || attempt >= attempts-1 {
		return false
	}
	var aiErr AIError
	if errors.As(err, &aiErr) && aiErr.Code == "error.ai_timeout" {
		return false
	}
	if errors.As(err, &aiErr) && aiErr.Code == "error.ai_upstream_error" &&
		aiErr.UpstreamStatus >= 400 && aiErr.UpstreamStatus < 500 &&
		aiErr.UpstreamStatus != 408 && aiErr.UpstreamStatus != 409 && aiErr.UpstreamStatus != 425 && aiErr.UpstreamStatus != 429 {
		return attempt == 0
	}
	return true
}

func shouldRetryWithOriginalAIChunk(err error) bool {
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		return false
	}
	switch aiErr.Code {
	case "error.ai_upstream_unavailable", "error.ai_stream_decode_failed":
		return true
	default:
		return false
	}
}

func splitAIChunkForRetry(input string, configuredMaxChars int) []string {
	runes := utf8.RuneCountInString(input)
	if runes < aiChunkFallbackMinRunes {
		return nil
	}
	target := configuredMaxChars / 2
	if target <= 0 || target >= runes {
		target = runes / 2
	}
	if target < 500 {
		target = 500
	}
	if target >= runes {
		return nil
	}
	return SplitAITextChunks(input, target)
}

func waitAIChunkRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(300*(attempt+1)) * time.Millisecond
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return AIError{Code: "error.ai_timeout", Err: ctx.Err()}
		}
		return AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
	case <-time.After(delay):
		return nil
	}
}

func aiFinalSummary(totalChunks int, totalChars int, templateKey string, taskType string, imageCount int, fallbacks []map[string]any) map[string]any {
	summary := map[string]any{
		"totalChunks":           totalChunks,
		"totalChars":            totalChars,
		"templateKey":           templateKey,
		"type":                  taskType,
		"imageSendSuccessCount": imageCount,
	}
	if len(fallbacks) > 0 {
		summary["fallbackChunks"] = fallbacks
		summary["fallbackChunkCount"] = len(fallbacks)
	}
	return summary
}
