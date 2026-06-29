package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const (
	AIJobStatusQueued    = "queued"
	AIJobStatusRunning   = "running"
	AIJobStatusCompleted = "completed"
	AIJobStatusFailed    = "failed"
	AIJobStatusCanceled  = "canceled"
)

const aiJobTokenRateWindowDuration = 10 * time.Second

type aiJobTokenRateSample struct {
	at     time.Time
	tokens int
}

type aiJobTokenRateWindow struct {
	samples []aiJobTokenRateSample
}

type AIJobCreateInput struct {
	Request     AIRequest `json:"request"`
	RequestHash string    `json:"requestHash"`
}

func (s *AIService) CreateJob(ctx context.Context, input AIJobCreateInput, actor AIActor) (domain.AIJob, error) {
	var empty domain.AIJob
	if s == nil || s.db == nil {
		return empty, AIError{Code: "error.ai_settings_unavailable"}
	}
	req := input.Request
	if strings.TrimSpace(req.Type) == "" {
		req.Type = AITaskFormatMarkdown
	}
	if req.Locale == "" {
		req.Locale = "en"
	}
	req = sanitizeAIRequest(req)
	if req.TemplateKey == "" {
		req.TemplateKey = defaultTemplateForTask(req.Type)
	}
	req.Images = normalizeAIImageInputs(req.Images)
	if strings.TrimSpace(req.Input) == "" && len(req.Images) == 0 && !hasAICustomGeneratePrompt(req) {
		return empty, AIError{Code: "error.ai_input_required"}
	}
	cfg := s.Config()
	if target, ok := contentFormatTargetForTask(cfg.ContentFormat, req.Type); ok {
		if !cfg.ContentFormat.Enabled || !target.Enabled {
			return empty, AIError{Code: "error.ai_template_disabled"}
		}
		if strings.TrimSpace(target.TemplateKey) != "" {
			req.TemplateKey = strings.TrimSpace(target.TemplateKey)
		}
	}
	requestHash := strings.TrimSpace(input.RequestHash)
	if requestHash == "" {
		requestHash = AIRequestHash(req)
	}
	if existing, ok := s.findActiveJob(ctx, requestHash, actor); ok {
		return existing, nil
	}
	tmpl := cfg.Templates[req.TemplateKey]
	chunks := []string{req.Input}
	if isAITextTransformTask(req.Type) || isAITextTransformTask(tmpl.TaskType) {
		chunks = SplitAITextChunks(req.Input, cfg.ChunkMaxChars)
	}
	if len(chunks) == 0 {
		chunks = []string{req.Input}
	}
	totalChars := 0
	for _, chunk := range chunks {
		totalChars += utf8.RuneCountInString(chunk)
	}
	rawReq, err := json.Marshal(req)
	if err != nil {
		return empty, AIError{Code: "error.ai_request_failed", Err: err}
	}
	now := time.Now()
	job := domain.AIJob{
		JobID:           uuid.NewString(),
		RequestHash:     requestHash,
		TaskType:        req.Type,
		TemplateKey:     req.TemplateKey,
		ActorType:       nonEmptyString(actor.Type, "system"),
		ActorID:         actor.ID,
		ActorDisplayID:  actor.DisplayID,
		Status:          AIJobStatusQueued,
		Stage:           "queued",
		Percent:         0,
		TotalChunks:     len(chunks),
		TotalChars:      totalChars,
		InputSummary:    summarizeAIText(req.Input, 1200),
		EstimatedTokens: estimateAIRequestTokens(req, cfg, tmpl, chunks),
		Request:         rawReq,
		Metadata:        jsonData(map[string]any{"locale": req.Locale, "templateKey": req.TemplateKey, "totalChunks": len(chunks), "totalChars": totalChars}),
		CreatedAt:       now,
		UpdatedAt:       &now,
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return empty, err
	}
	return job, nil
}

func (s *AIService) findActiveJob(ctx context.Context, requestHash string, actor AIActor) (domain.AIJob, bool) {
	var job domain.AIJob
	if requestHash == "" || s == nil || s.db == nil {
		return job, false
	}
	query := s.db.WithContext(ctx).Where("request_hash = ? AND status IN ?", requestHash, []string{AIJobStatusQueued, AIJobStatusRunning})
	query = query.Where("actor_type = ?", nonEmptyString(actor.Type, "system"))
	if actor.ID != nil {
		query = query.Where("actor_id = ?", *actor.ID)
	} else {
		query = query.Where("actor_id IS NULL")
	}
	err := query.Order("created_at DESC").First(&job).Error
	return job, err == nil
}

func (s *AIService) Job(ctx context.Context, jobID string, actor AIActor) (domain.AIJob, error) {
	var job domain.AIJob
	if s == nil || s.db == nil {
		return job, AIError{Code: "error.ai_settings_unavailable"}
	}
	query := s.db.WithContext(ctx).Where("job_id = ?", strings.TrimSpace(jobID))
	if actor.Type != "admin" {
		query = query.Where("actor_type = ?", nonEmptyString(actor.Type, "user"))
		if actor.ID != nil {
			query = query.Where("actor_id = ?", *actor.ID)
		} else {
			query = query.Where("actor_id IS NULL")
		}
	}
	if err := query.First(&job).Error; err != nil {
		return job, err
	}
	return job, nil
}

func (s *AIService) ActiveJob(ctx context.Context, requestHash string, actor AIActor) (domain.AIJob, error) {
	if job, ok := s.findActiveJob(ctx, requestHash, actor); ok {
		return job, nil
	}
	return domain.AIJob{}, gorm.ErrRecordNotFound
}

func (s *AIService) CancelJob(ctx context.Context, jobID string, actor AIActor) (domain.AIJob, error) {
	job, err := s.Job(ctx, jobID, actor)
	if err != nil {
		return job, err
	}
	if job.Status == AIJobStatusCompleted || job.Status == AIJobStatusFailed || job.Status == AIJobStatusCanceled {
		return job, nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":      AIJobStatusCanceled,
		"stage":       "canceled",
		"finished_at": &now,
		"updated_at":  &now,
		"error_code":  "error.ai_request_canceled",
		"metadata":    jsonData(s.aiJobMetadataWithoutQueue(ctx, job.ID)),
	}
	if err := s.db.WithContext(ctx).Model(&domain.AIJob{}).Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		return job, err
	}
	return s.Job(ctx, jobID, actor)
}

func (s *AIService) RunJobByID(ctx context.Context, jobID string) error {
	var job domain.AIJob
	if s == nil || s.db == nil {
		return AIError{Code: "error.ai_settings_unavailable"}
	}
	if err := s.db.WithContext(ctx).Where("job_id = ?", jobID).First(&job).Error; err != nil {
		return err
	}
	if job.Status == AIJobStatusCanceled || job.Status == AIJobStatusCompleted {
		return nil
	}
	var req AIRequest
	if err := json.Unmarshal(job.Request, &req); err != nil {
		return fmt.Errorf("%w: %v", errors.New("invalid ai job request"), err)
	}
	req = sanitizeAIRequest(req)
	actor := AIActor{Type: job.ActorType, ID: job.ActorID, DisplayID: job.ActorDisplayID}
	var output strings.Builder
	var reasoning strings.Builder
	chunkOutputs := map[int]string{}
	chunkPreviews := map[int]string{}
	var usage AIUsage
	processingStartedAt := time.Now()
	lastContentPersistedAt := time.Time{}
	tokenRateWindow := aiJobTokenRateWindow{}
	jobMetadata := jsonMapFromLog(job.Metadata)
	err := s.RunStream(ctx, req, actor, func(event AIStreamEvent) error {
		event.JobID = job.JobID
		s.enrichQueuedAIStreamEvent(ctx, job, &event)
		if event.Type != "" && s.jobEvents != nil {
			s.jobEvents.Publish(job.JobID, event)
		}
		if canceled, err := s.jobCanceled(ctx, job.ID); err != nil || canceled {
			if err != nil {
				return err
			}
			return context.Canceled
		}
		updates := map[string]any{"updated_at": time.Now()}
		finalUpdate := false
		previewReset := false
		switch event.Type {
		case "progress":
			if event.Stage == "queued" {
				updates["status"] = AIJobStatusQueued
			} else {
				updates["status"] = AIJobStatusRunning
				if job.StartedAt == nil {
					startedAt := time.Now()
					updates["started_at"] = &startedAt
					job.StartedAt = &startedAt
					tokenRateWindow.seed(startedAt)
				}
			}
			updates["stage"] = nonEmptyString(event.Stage, "running")
			updates["percent"] = event.Percent
			updates["current_chunk"] = event.CurrentChunk
			updates["total_chunks"] = event.TotalChunks
			updates["processed_chars"] = event.ProcessedChars
			updates["total_chars"] = event.TotalChars
			if event.Stage != "queued" && event.EstimatedTokens > 0 {
				updates["estimated_tokens"] = event.EstimatedTokens
			}
			if event.Stage != "queued" && event.TokensPerSecond > 0 {
				updates["tokens_per_second"] = event.TokensPerSecond
			}
			if event.Stage == "connecting" || event.Stage == "retrying" || event.Stage == "chunk_start" {
				delete(chunkPreviews, event.CurrentChunk-1)
				delete(jobMetadata, "queueJob")
				updates["metadata"] = jsonData(jobMetadata)
				output.Reset()
				output.WriteString(joinAIJobChunkOutputsWithPreviews(chunkOutputs, chunkPreviews))
				updates["output"] = output.String()
				previewReset = true
			} else if event.Stage == "queued" {
				jobMetadata["queueJob"] = aiStreamQueueJobMetadata(event)
				updates["metadata"] = jsonData(jobMetadata)
			}
		case "upstream_event":
			return nil
		case "chunk_delta":
			delta := sanitizeAIDBText(event.Delta)
			if delta == "" {
				return nil
			}
			chunkPreviews[event.ChunkIndex] += delta
			if time.Since(lastContentPersistedAt) < 150*time.Millisecond {
				return nil
			}
			output.Reset()
			output.WriteString(joinAIJobChunkOutputsWithPreviews(chunkOutputs, chunkPreviews))
			updates["output"] = output.String()
			applyAIJobLiveTokenStats(updates, output.String(), &tokenRateWindow)
		case "reasoning_delta":
			reasoning.WriteString(sanitizeAIDBText(event.ReasoningDelta))
			if time.Since(lastContentPersistedAt) < 150*time.Millisecond {
				return nil
			}
			updates["reasoning"] = reasoning.String()
		case "reasoning_done":
			event.Reasoning = sanitizeAIDBText(event.Reasoning)
			if strings.TrimSpace(event.Reasoning) != "" {
				reasoning.Reset()
				reasoning.WriteString(event.Reasoning)
				updates["reasoning"] = reasoning.String()
			}
		case "chunk_done":
			event.Text = sanitizeAIDBText(event.Text)
			chunkOutputs[event.ChunkIndex] = event.Text
			delete(chunkPreviews, event.ChunkIndex)
			output.Reset()
			output.WriteString(joinAIJobChunkOutputsWithPreviews(chunkOutputs, chunkPreviews))
			updates["output"] = output.String()
			applyAIJobLiveTokenStats(updates, output.String(), &tokenRateWindow)
		case "final":
			event.Text = sanitizeAIDBText(event.Text)
			usage = derefAIUsage(event.Usage, event.Text)
			delete(jobMetadata, "queueJob")
			updates["status"] = AIJobStatusCompleted
			updates["stage"] = "completed"
			updates["percent"] = 100
			updates["output"] = event.Text
			updates["metadata"] = jsonData(jobMetadata)
			updates["prompt_tokens"] = usage.PromptTokens
			updates["completion_tokens"] = usage.CompletionTokens
			updates["total_tokens"] = usage.TotalTokens
			updates["tokens_per_second"] = event.TokensPerSecond
			finishedAt := time.Now()
			updates["finished_at"] = &finishedAt
			finalUpdate = true
		}
		if err := s.db.WithContext(ctx).Model(&domain.AIJob{}).Where("id = ?", job.ID).Updates(sanitizeAIJobUpdates(updates)).Error; err != nil {
			if finalUpdate {
				return err
			}
			return nil
		}
		if _, ok := updates["output"]; ok && !previewReset {
			lastContentPersistedAt = time.Now()
		}
		if _, ok := updates["reasoning"]; ok {
			lastContentPersistedAt = time.Now()
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return s.markJobCanceled(context.Background(), job.ID)
		}
		return s.markJobFailed(context.Background(), job.ID, err, usage, job.EstimatedTokens, time.Since(processingStartedAt))
	}
	return nil
}

func (s *AIService) jobCanceled(ctx context.Context, id int64) (bool, error) {
	var status string
	err := s.db.WithContext(ctx).Model(&domain.AIJob{}).Select("status").Where("id = ?", id).Scan(&status).Error
	return status == AIJobStatusCanceled, err
}

func (s *AIService) aiJobMetadataWithoutQueue(ctx context.Context, id int64) map[string]any {
	var row domain.AIJob
	if s == nil || s.db == nil {
		return map[string]any{}
	}
	if err := s.db.WithContext(ctx).Select("metadata").Where("id = ?", id).First(&row).Error; err != nil {
		return map[string]any{}
	}
	metadata := jsonMapFromLog(row.Metadata)
	delete(metadata, "queueJob")
	return metadata
}

func aiStreamQueueJobMetadata(event AIStreamEvent) map[string]any {
	out := map[string]any{
		"id":                   event.JobID,
		"jobId":                event.JobID,
		"queue":                "ai-concurrency",
		"state":                "queued",
		"queuePosition":        event.QueuePosition,
		"queueCount":           event.QueueTotal,
		"estimatedWaitSeconds": event.ETASeconds,
	}
	if event.ActiveJobID != "" {
		out["activeJob"] = map[string]any{
			"jobId":           event.ActiveJobID,
			"actorId":         event.ActiveActorID,
			"actorDisplayId":  event.ActiveDisplayID,
			"generatedTokens": event.ActiveTokens,
			"tokensPerSecond": event.ActiveRate,
		}
	}
	return out
}

func (s *AIService) enrichQueuedAIStreamEvent(ctx context.Context, waitingJob domain.AIJob, event *AIStreamEvent) {
	if s == nil || s.db == nil || event == nil || event.Type != "progress" || event.Stage != "queued" {
		return
	}
	active, ok := s.activeAIJobForQueue(ctx, waitingJob)
	if !ok {
		return
	}
	event.ActiveJobID = active.JobID
	event.ActiveActorID = int64FromPointer(active.ActorID)
	event.ActiveDisplayID = stringFromPointer(active.ActorDisplayID)
	event.ActiveTokens = aiJobGeneratedTokenCountFromRow(active)
	event.ActiveRate = active.TokensPerSecond
}

func (s *AIService) activeAIJobForQueue(ctx context.Context, waitingJob domain.AIJob) (domain.AIJob, bool) {
	var active domain.AIJob
	if s == nil || s.db == nil {
		return active, false
	}
	if strings.TrimSpace(waitingJob.TemplateKey) != "" {
		err := s.db.WithContext(ctx).
			Where("status = ? AND id <> ? AND template_key = ?", AIJobStatusRunning, waitingJob.ID, waitingJob.TemplateKey).
			Order("started_at ASC, updated_at ASC").
			First(&active).Error
		if err == nil {
			return active, true
		}
	}
	err := s.db.WithContext(ctx).
		Where("status = ? AND id <> ?", AIJobStatusRunning, waitingJob.ID).
		Order("started_at ASC, updated_at ASC").
		First(&active).Error
	return active, err == nil
}

func aiJobGeneratedTokenCountFromRow(row domain.AIJob) int {
	tokens := row.CompletionTokens
	if tokens <= 0 {
		tokens = row.TotalTokens
	}
	if tokens <= 0 {
		tokens = estimateAITokens(row.Output)
	}
	return maxInt(0, tokens)
}

func int64FromPointer(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyAIJobLiveTokenStats(updates map[string]any, text string, rateWindow *aiJobTokenRateWindow) {
	generatedTokens := estimateAITokens(text)
	updates["completion_tokens"] = generatedTokens
	updates["total_tokens"] = generatedTokens
	if rateWindow == nil {
		return
	}
	if speed := rateWindow.record(time.Now(), generatedTokens); speed > 0 {
		updates["tokens_per_second"] = speed
	}
}

func (w *aiJobTokenRateWindow) seed(at time.Time) {
	if w == nil || len(w.samples) > 0 {
		return
	}
	if at.IsZero() {
		at = time.Now()
	}
	w.samples = append(w.samples, aiJobTokenRateSample{at: at, tokens: 0})
}

func (w *aiJobTokenRateWindow) record(at time.Time, tokens int) float64 {
	if w == nil || at.IsZero() {
		return 0
	}
	if tokens < 0 {
		tokens = 0
	}
	if len(w.samples) == 0 {
		w.seed(at)
	}
	cutoff := at.Add(-aiJobTokenRateWindowDuration)
	firstKept := 0
	for firstKept+1 < len(w.samples) && w.samples[firstKept+1].at.Before(cutoff) {
		firstKept++
	}
	if firstKept > 0 {
		w.samples = append([]aiJobTokenRateSample(nil), w.samples[firstKept:]...)
	}
	w.samples = append(w.samples, aiJobTokenRateSample{at: at, tokens: tokens})
	if len(w.samples) < 2 {
		return 0
	}
	oldest := w.samples[0]
	duration := at.Sub(oldest.at)
	if duration <= 0 {
		return 0
	}
	delta := tokens - oldest.tokens
	if delta <= 0 {
		return 0
	}
	return float64(delta) / duration.Seconds()
}

func (s *AIService) markJobCanceled(ctx context.Context, id int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&domain.AIJob{}).Where("id = ?", id).Updates(map[string]any{
		"status":      AIJobStatusCanceled,
		"stage":       "canceled",
		"error_code":  "error.ai_request_canceled",
		"finished_at": &now,
		"updated_at":  &now,
		"metadata":    jsonData(s.aiJobMetadataWithoutQueue(ctx, id)),
	}).Error
}

func (s *AIService) markJobFailed(ctx context.Context, id int64, err error, usage AIUsage, estimatedTokens int, duration time.Duration) error {
	now := time.Now()
	status := 0
	detail := ""
	var aiErr AIError
	if errors.As(err, &aiErr) {
		status = aiErr.UpstreamStatus
		detail = aiErr.UpstreamDetail
	}
	return s.db.WithContext(ctx).Model(&domain.AIJob{}).Where("id = ?", id).Updates(sanitizeAIJobUpdates(map[string]any{
		"status":            AIJobStatusFailed,
		"stage":             "failed",
		"error_code":        aiErrorCode(err),
		"error_message":     summarizeAIText(err.Error(), 1200),
		"upstream_status":   status,
		"upstream_detail":   summarizeAIText(detail, 1200),
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":      usage.TotalTokens,
		"tokens_per_second": calculateAITokensPerSecond(usage, estimatedTokens, duration),
		"finished_at":       &now,
		"updated_at":        &now,
		"metadata":          jsonData(s.aiJobMetadataWithoutQueue(ctx, id)),
	})).Error
}

func AIRequestHash(req AIRequest) string {
	req.Images = nil
	data, _ := json.Marshal(req)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hasAICustomGeneratePrompt(req AIRequest) bool {
	if strings.TrimSpace(req.Type) != AITaskPostCustomGenerate {
		return false
	}
	value, ok := req.Variables["customPrompt"]
	return ok && strings.TrimSpace(fmt.Sprint(value)) != ""
}

func derefAIUsage(usage *AIUsage, fallbackText string) AIUsage {
	fallbackTokens := estimateAITokens(fallbackText)
	if usage == nil {
		return AIUsage{CompletionTokens: fallbackTokens, TotalTokens: fallbackTokens}
	}
	out := *usage
	if out.CompletionTokens <= 0 && out.TotalTokens <= 0 && fallbackTokens > 0 {
		out.CompletionTokens = fallbackTokens
		out.TotalTokens = fallbackTokens
	} else if out.TotalTokens <= 0 && out.CompletionTokens > 0 {
		out.TotalTokens = out.CompletionTokens
	}
	return out
}

func joinAIJobChunkOutputs(chunks map[int]string) string {
	return joinAIJobChunkOutputsWithPreviews(chunks, nil)
}

func joinAIJobChunkOutputsWithPreviews(chunks map[int]string, previews map[int]string) string {
	if len(chunks) == 0 {
		if len(previews) == 0 {
			return ""
		}
	}
	limit := len(chunks)
	for index := 0; ; index++ {
		if _, ok := chunks[index]; ok {
			limit = maxInt(limit, index+1)
			continue
		}
		if _, ok := previews[index]; ok {
			limit = maxInt(limit, index+1)
			continue
		}
		break
	}
	if limit == 0 {
		return ""
	}
	out := make([]string, 0, limit)
	for index := 0; index < limit; index++ {
		text, ok := chunks[index]
		preview := strings.TrimSpace(previews[index])
		if ok && preview != "" {
			text = strings.TrimSpace(text + preview)
		} else if !ok {
			text = preview
		}
		out = append(out, strings.TrimSpace(text))
	}
	return strings.TrimSpace(strings.Join(out, "\n\n"))
}

func sanitizeAIJobUpdates(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = sanitizeAIDBAny(value)
	}
	return out
}
