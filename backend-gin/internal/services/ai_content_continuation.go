package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var aiRequestedOutputLengthPattern = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*(万|千|words?|tokens?|字符|个字|字|k|w)`)

type aiCustomContinuationPlan struct {
	Enabled      bool
	MaxRounds    int
	ContextChars int
}

type aiStreamRunState struct {
	usage              AIUsage
	upstreamAttempts   []map[string]any
	fallbacks          []map[string]any
	outputs            []string
	continuationRounds int
}

func aiCustomContinuationPlanForRequest(req AIRequest, target AIContentFormatTargetConfig) aiCustomContinuationPlan {
	continuation := normalizeAIContentContinuation(target.Continuation, defaultAIContentContinuationConfig())
	if strings.TrimSpace(req.Type) != AITaskPostCustomGenerate || !continuation.Enabled {
		return aiCustomContinuationPlan{}
	}
	size := utf8.RuneCountInString(req.Input)
	customPrompt := ""
	if value, ok := req.Variables["customPrompt"]; ok {
		customPrompt = strings.TrimSpace(fmt.Sprint(value))
		size += utf8.RuneCountInString(customPrompt)
	}
	if requestedChars := aiRequestedOutputChars(customPrompt); requestedChars > size {
		size = requestedChars
	}
	if size < continuation.TriggerChars {
		return aiCustomContinuationPlan{}
	}
	return aiCustomContinuationPlan{
		Enabled:      true,
		MaxRounds:    continuation.MaxRounds,
		ContextChars: continuation.ContextChars,
	}
}

func (p aiCustomContinuationPlan) totalRounds() int {
	if !p.Enabled {
		return 1
	}
	return 1 + maxAIInt(1, p.MaxRounds)
}

func (p aiCustomContinuationPlan) estimatedTokenMultiplier() int {
	if !p.Enabled {
		return 1
	}
	return p.totalRounds()
}

func (s *AIService) runAIChunksWithContinuation(
	ctx context.Context,
	cfg AIConfig,
	tmpl AITemplateConfig,
	req AIRequest,
	jobID string,
	chunks []string,
	chunkChars []int,
	totalChars int,
	plan aiCustomContinuationPlan,
	emit func(AIStreamEvent) error,
) (aiStreamRunState, error) {
	state := aiStreamRunState{outputs: make([]string, 0, len(chunks)*plan.totalRounds())}
	processedChars := 0
	for index, chunk := range chunks {
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		default:
		}
		if err := emit(AIStreamEvent{
			Type:              "progress",
			JobID:             jobID,
			Percent:           chunkProgress(index, len(chunks)),
			CurrentChunk:      index + 1,
			TotalChunks:       len(chunks),
			CurrentChunkChars: chunkChars[index],
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Stage:             "chunk_start",
		}); err != nil {
			return state, err
		}
		chunkReq := req
		chunkReq.Input = chunk
		chunkReq.Variables = aiChunkVariables(req.Variables, req.Locale, index, len(chunks))
		result, err := s.runAIChunkWithFallback(ctx, cfg, tmpl, chunkReq, aiChunkRunOptions{
			ChunkIndex:        index,
			ChunkNumber:       index + 1,
			TotalChunks:       len(chunks),
			CurrentChunkChars: chunkChars[index],
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Percent:           chunkProgress(index, len(chunks)),
			JobID:             jobID,
			Emit:              emit,
			Usage:             &state.usage,
			UpstreamAttempts:  &state.upstreamAttempts,
			ShowReasoning:     cfg.ShowReasoning,
		})
		if err != nil {
			return state, err
		}
		text := result.Text
		state.outputs = append(state.outputs, text)
		if result.Fallback != nil {
			state.fallbacks = append(state.fallbacks, result.Fallback)
		}
		if cfg.ShowReasoning && strings.TrimSpace(result.Reasoning) != "" {
			if err := emit(AIStreamEvent{Type: "reasoning_done", JobID: jobID, ChunkIndex: index, Reasoning: result.Reasoning}); err != nil {
				return state, err
			}
		}
		if err := emit(AIStreamEvent{Type: "chunk_done", JobID: jobID, ChunkIndex: index, Text: text}); err != nil {
			return state, err
		}
		if plan.Enabled {
			continuedText, err := s.runAIChunkContinuationRounds(ctx, cfg, tmpl, req, chunk, text, jobID, index, len(chunks), chunkChars[index], processedChars, totalChars, plan, emit, &state)
			if err != nil {
				return state, err
			}
			if strings.TrimSpace(continuedText) != "" {
				combined := strings.TrimSpace(text + "\n\n" + continuedText)
				state.outputs[len(state.outputs)-1] = combined
				if err := emit(AIStreamEvent{Type: "chunk_done", JobID: jobID, ChunkIndex: index, Text: combined}); err != nil {
					return state, err
				}
			}
		}
		processedChars += chunkChars[index]
		if err := emit(AIStreamEvent{
			Type:              "progress",
			JobID:             jobID,
			Percent:           chunkProgress(index+1, len(chunks)),
			CurrentChunk:      index + 1,
			TotalChunks:       len(chunks),
			CurrentChunkChars: chunkChars[index],
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Stage:             "chunk_done",
		}); err != nil {
			return state, err
		}
	}
	return state, nil
}

func (s *AIService) runAIChunkContinuationRounds(
	ctx context.Context,
	cfg AIConfig,
	tmpl AITemplateConfig,
	req AIRequest,
	chunk string,
	firstText string,
	jobID string,
	chunkIndex int,
	totalChunks int,
	currentChunkChars int,
	processedChars int,
	totalChars int,
	plan aiCustomContinuationPlan,
	emit func(AIStreamEvent) error,
	state *aiStreamRunState,
) (string, error) {
	continuations := make([]string, 0, plan.MaxRounds)
	for round := 1; round <= plan.MaxRounds; round++ {
		outputSoFar := strings.TrimSpace(strings.Join(append([]string{firstText}, continuations...), "\n\n"))
		if strings.TrimSpace(outputSoFar) == "" {
			break
		}
		if err := emit(AIStreamEvent{
			Type:              "progress",
			JobID:             jobID,
			Percent:           chunkProgress(chunkIndex, totalChunks),
			CurrentChunk:      chunkIndex + 1,
			TotalChunks:       totalChunks,
			CurrentChunkChars: currentChunkChars,
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Stage:             "continuation_start",
		}); err != nil {
			return "", err
		}
		contReq := req
		contReq.Input = aiContinuationCompressedInput(chunk, outputSoFar, plan.ContextChars)
		contReq.Variables = aiChunkVariables(req.Variables, req.Locale, chunkIndex, totalChunks)
		contReq.Variables["continuationEnabled"] = true
		contReq.Variables["continuationRound"] = round
		contReq.Variables["continuationTotalRounds"] = plan.MaxRounds
		contReq.Variables["continuationSummary"] = summarizeAIText(outputSoFar, plan.ContextChars)
		contReq.Variables["continuationTail"] = aiTextTail(outputSoFar, plan.ContextChars)
		contReq.Variables["continuationInstruction"] = aiContinuationInstruction()
		contReq.Variables["customPrompt"] = aiContinuationPrompt(req.Variables, aiContinuationInstruction())
		result, err := s.runAIChunkWithFallback(ctx, cfg, tmpl, contReq, aiChunkRunOptions{
			ChunkIndex:        chunkIndex,
			ChunkNumber:       chunkIndex + 1,
			TotalChunks:       totalChunks,
			CurrentChunkChars: currentChunkChars,
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Percent:           chunkProgress(chunkIndex, totalChunks),
			JobID:             jobID,
			Emit:              emit,
			Usage:             &state.usage,
			UpstreamAttempts:  &state.upstreamAttempts,
			ShowReasoning:     cfg.ShowReasoning,
			StagePrefix:       "continuation_",
		})
		if err != nil {
			return "", err
		}
		text := strings.TrimSpace(result.Text)
		if text == "" || aiContinuationLooksRepeated(outputSoFar, text) {
			break
		}
		continuations = append(continuations, text)
		state.continuationRounds++
		if result.Fallback != nil {
			result.Fallback["continuationRound"] = round
			state.fallbacks = append(state.fallbacks, result.Fallback)
		}
		if cfg.ShowReasoning && strings.TrimSpace(result.Reasoning) != "" {
			if err := emit(AIStreamEvent{Type: "reasoning_done", JobID: jobID, ChunkIndex: chunkIndex, Reasoning: result.Reasoning}); err != nil {
				return "", err
			}
		}
		if err := emit(AIStreamEvent{
			Type:              "progress",
			JobID:             jobID,
			Percent:           chunkProgress(chunkIndex, totalChunks),
			CurrentChunk:      chunkIndex + 1,
			TotalChunks:       totalChunks,
			CurrentChunkChars: currentChunkChars,
			ProcessedChars:    processedChars,
			TotalChars:        totalChars,
			Stage:             "continuation_done",
		}); err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(strings.Join(continuations, "\n\n")), nil
}

func aiChunkVariables(input map[string]any, locale string, index int, total int) map[string]any {
	variables := cloneVariables(input)
	variables["chunkIndex"] = index + 1
	variables["totalChunks"] = total
	variables["locale"] = locale
	variables["continuationEnabled"] = ""
	variables["continuationRound"] = ""
	variables["continuationTotalRounds"] = ""
	variables["continuationSummary"] = ""
	variables["continuationTail"] = ""
	variables["continuationInstruction"] = ""
	return variables
}

func aiContinuationPrompt(variables map[string]any, instruction string) string {
	base := ""
	if value, ok := variables["customPrompt"]; ok {
		base = strings.TrimSpace(fmt.Sprint(value))
	}
	if base == "" {
		return instruction
	}
	return strings.TrimSpace(base + "\n\n" + instruction)
}

func aiContinuationInstruction() string {
	return "Continuation control: continue from the compressed context and already generated ending. Do not restart, repeat, summarize, or mention that this is a continuation. Keep the same structure, voice, locale, and Markdown style."
}

func aiContinuationCompressedInput(source string, output string, limit int) string {
	source = summarizeAIText(source, limit)
	tail := aiTextTail(output, limit)
	if strings.TrimSpace(source) == "" {
		return tail
	}
	if strings.TrimSpace(tail) == "" {
		return source
	}
	return strings.TrimSpace("Compressed source context:\n" + source + "\n\nAlready generated ending:\n" + tail)
}

func aiTextTail(input string, limit int) string {
	input = strings.TrimSpace(sanitizeAIDBText(input))
	if limit <= 0 || utf8.RuneCountInString(input) <= limit {
		return input
	}
	runes := []rune(input)
	return string(runes[len(runes)-limit:])
}

func aiContinuationLooksRepeated(previous string, next string) bool {
	previous = strings.TrimSpace(previous)
	next = strings.TrimSpace(next)
	if next == "" {
		return true
	}
	if previous == next || strings.HasSuffix(previous, next) {
		return true
	}
	return false
}

func aiRequestedOutputChars(prompt string) int {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return 0
	}
	best := 0
	for _, match := range aiRequestedOutputLengthPattern.FindAllStringSubmatch(prompt, -1) {
		if len(match) < 3 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil || value <= 0 {
			continue
		}
		unit := strings.ToLower(strings.TrimSpace(match[2]))
		switch unit {
		case "万", "w":
			value *= 10000
		case "千", "k":
			value *= 1000
		case "word", "words":
			value *= 5
		case "token", "tokens":
			value *= 4
		}
		if int(value) > best {
			best = int(value)
		}
	}
	return best
}
