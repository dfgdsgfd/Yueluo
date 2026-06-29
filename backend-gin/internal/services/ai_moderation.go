package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/datatypes"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	aiModerationMinTimeoutSeconds = 120
	aiModerationMaxPromptRunes    = 6000
	aiModerationOutputTokens      = 768
)

type AIModerationDebugInput struct {
	TargetType   string
	Content      string
	TemplateKey  string
	SystemPrompt string
	UserPrompt   string
	Prompt       string
	Config       *AIModerationTargetConfig
}

type AIModerationDebugResult struct {
	TargetType  string         `json:"targetType"`
	TemplateKey string         `json:"templateKey"`
	PromptInput string         `json:"promptInput"`
	RawOutput   string         `json:"rawOutput"`
	Status      string         `json:"status"`
	Action      string         `json:"action"`
	Decision    map[string]any `json:"decision"`
	Usage       *AIUsage       `json:"usage,omitempty"`
}

func (s *QueueService) EnqueueAIModeration(ctx context.Context, targetType string, targetID int64, userID int64, originalVisibility string) (map[string]any, error) {
	if s == nil || s.ai == nil {
		return nil, errors.New("ai service unavailable")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	if targetID <= 0 || userID <= 0 {
		return nil, errors.New("moderation target is required")
	}
	targetType = normalizeAIModerationTarget(targetType)
	if targetType == "" {
		return nil, errors.New("unsupported moderation target")
	}
	payload := aiModerateContentTaskPayload{TargetType: targetType, TargetID: targetID, UserID: userID, OriginalVisibility: originalVisibility, EnqueuedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	taskID := fmt.Sprintf("%s:%s:%d", TaskAIModerateContent, targetType, targetID)
	cfg := s.ai.Config()
	timeoutSeconds := moderationRequestTimeoutSeconds(cfg)
	options := []asynq.Option{
		asynq.Queue(QueueAITask),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(time.Duration(timeoutSeconds+60) * time.Second),
	}
	if retention := s.completedRetention(QueueAITask, 24*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskAIModerateContent, data, QueueAITask, payload.EnqueuedAt), options...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			return map[string]any{"id": taskID, "queue": QueueAITask, "duplicate": true, "enqueuedAt": payload.EnqueuedAt}, nil
		}
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskAIModerateContent,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"targetType": payload.TargetType, "targetId": payload.TargetID, "userId": payload.UserID},
	})
	return map[string]any{"id": info.ID, "queue": info.Queue, "state": info.State.String(), "enqueuedAt": payload.EnqueuedAt}, nil
}

func (s *QueueService) processAIModerateContent(ctx context.Context, task *asynq.Task) error {
	var payload aiModerateContentTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if s == nil || s.ai == nil || s.db == nil {
		return fmt.Errorf("%w: ai moderation dependencies unavailable", asynq.SkipRetry)
	}
	if err := s.moderateContent(ctx, payload); err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{"targetType": payload.TargetType, "targetId": payload.TargetID})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) moderateContent(ctx context.Context, payload aiModerateContentTaskPayload) error {
	targetType := normalizeAIModerationTarget(payload.TargetType)
	cfg := s.ai.Config()
	targetCfg := cfg.Moderation.Target(targetType)
	if !targetCfg.Enabled {
		return fmt.Errorf("%w: ai moderation disabled", asynq.SkipRetry)
	}
	content, err := s.moderationContent(ctx, targetType, payload.TargetID)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	promptInput := buildModerationPromptInput(targetType, payload.TargetID, payload.UserID, content, targetCfg)
	text, _, err := s.ai.RunText(ctx, AIRequest{
		Type:        targetType + "_moderation",
		Locale:      "en",
		Input:       promptInput,
		TemplateKey: targetCfg.TemplateKey,
		Options:     moderationAIOptions(cfg),
		Variables: map[string]any{
			"targetType": targetType,
			"targetId":   payload.TargetID,
			"rules":      moderationRulePrompt(targetCfg),
			"prompt":     targetCfg.Prompt,
		},
	}, AIActor{Type: "system"})
	if err != nil {
		_ = s.writeModerationLog(ctx, payload, content, targetCfg, aiModerationDecision{}, "failed", AIModerationActionObserve, nil, nil, aiErrorCode(err), err.Error())
		return err
	}
	decision := parseAIModerationDecision(text, targetCfg, targetType)
	action := moderationActionForDecision(decision, targetCfg, targetType)
	status := "approved"
	if decision.Violation {
		status = "flagged"
	}
	modelJSON := jsonData(moderationModelResultMap(text, decision, targetCfg, action, status))
	categoryJSON := moderationCategoriesJSON(decision)
	if err := s.writeModerationLog(ctx, payload, content, targetCfg, decision, status, action, modelJSON, categoryJSON, "", ""); err != nil {
		return err
	}
	if err := s.applyModerationDecision(ctx, payload, decision, action); err != nil {
		return err
	}
	return nil
}

func (s *QueueService) moderationContent(ctx context.Context, targetType string, targetID int64) (string, error) {
	switch targetType {
	case AIModerationTargetPost:
		var post domain.Post
		if err := s.db.WithContext(ctx).Where("id = ?", targetID).First(&post).Error; err != nil {
			return "", err
		}
		return "Title:\n" + post.Title + "\n\nContent:\n" + post.Content, nil
	default:
		var comment domain.Comment
		if err := s.db.WithContext(ctx).Where("id = ?", targetID).First(&comment).Error; err != nil {
			return "", err
		}
		return comment.Content, nil
	}
}

func buildModerationPromptInput(targetType string, targetID int64, userID int64, content string, cfg AIModerationTargetConfig) string {
	content, truncated, originalRunes := moderationPromptContent(content)
	lines := []string{
		"Return strict JSON only.",
		"Target type: " + targetType,
		fmt.Sprintf("Target ID: %d", targetID),
		fmt.Sprintf("User ID: %d", userID),
		"Enabled rules and actions: " + moderationRulePrompt(cfg),
		"Sensitivity is 0 to 1 per enabled rule: 0 means only clear severe violations qualify; 1 means borderline risks can qualify. Return confidence 0 to 1 for every rule.",
		"Only mark a rule violation when the content matches that exact enabled rule and the evidence is strong enough for its sensitivity. Do not flag unrelated target types or disabled rules.",
		"Allowed actions are the actions configured on matched enabled rules. If no enabled rule meets sensitivity, return violation=false, action=observe, and explain the closest non-action reason.",
	}
	if truncated {
		lines = append(lines, fmt.Sprintf("Content was shortened for timeout protection. Review the provided beginning and ending carefully. Original length: %d runes.", originalRunes))
	}
	if strings.TrimSpace(cfg.Prompt) != "" {
		lines = append(lines, "Custom prompt:\n"+cfg.Prompt)
	}
	lines = append(lines, "Content:\n"+content)
	return strings.Join(lines, "\n\n")
}

func moderationPromptContent(content string) (string, bool, int) {
	runes := []rune(content)
	originalRunes := len(runes)
	if originalRunes <= aiModerationMaxPromptRunes {
		return content, false, originalRunes
	}
	headLen := aiModerationMaxPromptRunes * 2 / 3
	tailLen := aiModerationMaxPromptRunes - headLen
	omitted := originalRunes - headLen - tailLen
	shortened := string(runes[:headLen]) +
		fmt.Sprintf("\n\n[... %d runes omitted for timeout protection ...]\n\n", omitted) +
		string(runes[originalRunes-tailLen:])
	return shortened, true, originalRunes
}

func moderationRequestTimeoutSeconds(cfg AIConfig) int {
	return maxInt(aiModerationMinTimeoutSeconds, cfg.TimeoutSeconds)
}

func moderationAIOptions(cfg AIConfig) AIOptions {
	timeoutSeconds := moderationRequestTimeoutSeconds(cfg)
	maxOutputTokens := aiModerationOutputTokens
	return AIOptions{
		StructuredJSON:  boolPtr(true),
		MaxOutputTokens: &maxOutputTokens,
		TimeoutSeconds:  &timeoutSeconds,
	}
}

func moderationRulePrompt(cfg AIModerationTargetConfig) string {
	data, _ := json.Marshal(cfg.Rules)
	return string(data)
}

func (s *AIService) DebugModeration(ctx context.Context, input AIModerationDebugInput, actor AIActor) (AIModerationDebugResult, error) {
	var out AIModerationDebugResult
	if s == nil {
		return out, AIError{Code: "error.ai_settings_unavailable"}
	}
	targetType := normalizeAIModerationTarget(input.TargetType)
	if targetType == "" {
		return out, AIError{Code: "error.invalid_ai_setting"}
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return out, AIError{Code: "error.ai_input_required"}
	}
	cfg := s.Config()
	targetCfg := cfg.Moderation.Target(targetType)
	if input.Config != nil {
		targetCfg = *input.Config
	}
	if strings.TrimSpace(input.Prompt) != "" {
		targetCfg.Prompt = strings.TrimSpace(input.Prompt)
	}
	templateKey := strings.TrimSpace(input.TemplateKey)
	if templateKey == "" {
		templateKey = strings.TrimSpace(targetCfg.TemplateKey)
	}
	if templateKey == "" {
		templateKey = targetType + "_moderation"
	}
	tmpl, ok := cfg.Templates[templateKey]
	if !ok {
		return out, AIError{Code: "error.ai_template_missing"}
	}
	if strings.TrimSpace(input.SystemPrompt) != "" {
		tmpl.SystemPrompt = input.SystemPrompt
	}
	if strings.TrimSpace(input.UserPrompt) != "" {
		tmpl.UserPrompt = input.UserPrompt
		tmpl.Prompt = input.UserPrompt
	}
	promptInput := buildModerationPromptInput(targetType, 0, actorIDValue(actor.ID), content, targetCfg)
	raw, usage, err := s.RunTextWithTemplateOverride(ctx, AIRequest{
		Type:        targetType + "_moderation",
		Locale:      "en",
		Input:       promptInput,
		TemplateKey: templateKey,
		Options:     moderationAIOptions(cfg),
		Variables: map[string]any{
			"targetType": targetType,
			"rules":      moderationRulePrompt(targetCfg),
			"prompt":     targetCfg.Prompt,
		},
	}, actor, templateKey, tmpl)
	if err != nil {
		return out, err
	}
	decision := parseAIModerationDecision(raw, targetCfg, targetType)
	action := moderationActionForDecision(decision, targetCfg, targetType)
	status := "approved"
	if decision.Violation {
		status = "flagged"
	}
	return AIModerationDebugResult{
		TargetType:  targetType,
		TemplateKey: templateKey,
		PromptInput: promptInput,
		RawOutput:   raw,
		Status:      status,
		Action:      action,
		Decision:    moderationDecisionMap(decision),
		Usage:       usage,
	}, nil
}

func (s *QueueService) applyModerationDecision(ctx context.Context, payload aiModerateContentTaskPayload, decision aiModerationDecision, action string) error {
	repo := repositories.NewContentRepository(s.db)
	targetType := normalizeAIModerationTarget(payload.TargetType)
	applied := AIModerationActionObserve
	switch normalizeAIModerationTarget(payload.TargetType) {
	case AIModerationTargetPost:
		if !decision.Violation || action == AIModerationActionObserve {
			if err := repo.MarkPostAIModerated(ctx, payload.TargetID, 1); err != nil {
				return err
			}
			s.bumpAIModerationCache(ctx, targetType, applied)
			return nil
		}
		if action == AIModerationActionDelete {
			var post domain.Post
			if err := s.db.WithContext(ctx).Select("id", "user_id").Where("id = ?", payload.TargetID).First(&post).Error; err != nil {
				return err
			}
			if err := repo.DeletePost(ctx, post.UserID, post.ID); err != nil {
				return err
			}
			applied = AIModerationActionDelete
			s.bumpAIModerationCache(ctx, targetType, applied)
			return nil
		}
		if err := repo.PrivatePostAfterAIModeration(ctx, payload.TargetID); err != nil {
			return err
		}
		applied = AIModerationActionPrivate
		s.bumpAIModerationCache(ctx, targetType, applied)
		return nil
	default:
		if !decision.Violation || action == AIModerationActionObserve {
			if err := repo.MarkCommentAIModerated(ctx, payload.TargetID, 1); err != nil {
				return err
			}
			s.bumpAIModerationCache(ctx, targetType, applied)
			return nil
		}
		if action == AIModerationActionDelete {
			if err := repo.DeleteCommentAfterAIModeration(ctx, payload.TargetID); err != nil {
				return err
			}
			applied = AIModerationActionDelete
			s.bumpAIModerationCache(ctx, targetType, applied)
			return nil
		}
		if err := repo.RejectCommentAfterAIModeration(ctx, payload.TargetID); err != nil {
			return err
		}
		applied = AIModerationActionPrivate
		s.bumpAIModerationCache(ctx, targetType, applied)
		return nil
	}
}

func (s *QueueService) bumpAIModerationCache(ctx context.Context, targetType string, action string) {
	if s == nil || s.cache == nil {
		return
	}
	switch targetType {
	case AIModerationTargetPost:
		if action == AIModerationActionObserve {
			s.cache.BumpCacheVersion(ctx, "posts", "search")
			return
		}
		s.cache.BumpCacheVersion(ctx, "posts", "file_access", "comments", "search", "users", "notifications")
	default:
		if action == AIModerationActionObserve {
			s.cache.BumpCacheVersion(ctx, "posts", "comments", "search")
			return
		}
		s.cache.BumpCacheVersion(ctx, "posts", "comments", "search", "interactions", "notifications")
	}
}

func (s *QueueService) writeModerationLog(ctx context.Context, payload aiModerateContentTaskPayload, content string, cfg AIModerationTargetConfig, decision aiModerationDecision, status string, action string, modelResult datatypes.JSON, categories datatypes.JSON, errorCode string, errorMessage string) error {
	count := s.userModerationViolationCount(ctx, payload.UserID)
	if status == "flagged" {
		count++
	}
	now := time.Now()
	triggerReason := moderationTriggerReason(decision, action, errorMessage)
	row := domain.AIModerationLog{
		TargetType:         normalizeAIModerationTarget(payload.TargetType),
		TargetID:           payload.TargetID,
		UserID:             payload.UserID,
		Status:             status,
		Action:             action,
		TriggerReason:      triggerReason,
		Categories:         categories,
		ModelResult:        modelResult,
		UserViolationCount: count,
		ErrorCode:          errorCode,
		ErrorMessage:       summarizeAIText(errorMessage, 1200),
		Metadata: jsonData(map[string]any{
			"originalVisibility": payload.OriginalVisibility,
			"mode":               "post_publish_review",
			"contentSnapshot":    summarizeAIText(content, 20000),
			"contentCharCount":   len([]rune(content)),
			"triggerReason":      triggerReason,
			"enabledRules":       cfg.Rules,
			"decision":           moderationDecisionMap(decision),
		}),
		CreatedAt: now,
		UpdatedAt: &now,
	}
	if len(row.Categories) == 0 {
		row.Categories = datatypes.JSON([]byte("{}"))
	}
	if len(row.ModelResult) == 0 {
		row.ModelResult = datatypes.JSON([]byte("{}"))
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	auditType := 4
	if row.TargetType == AIModerationTargetPost {
		auditType = 3
	}
	auditStatus := 1
	if row.Status == "flagged" {
		auditStatus = 2
	}
	targetID := payload.TargetID
	reason := row.TriggerReason
	if errorMessage != "" {
		reason = errorMessage
	}
	return s.db.WithContext(ctx).Create(&domain.Audit{
		UserID:      payload.UserID,
		Type:        auditType,
		TargetID:    &targetID,
		Content:     summarizeAIText(content, 5000),
		AuditResult: row.ModelResult,
		Categories:  row.Categories,
		Reason:      &reason,
		Status:      &auditStatus,
		AuditTime:   &now,
		CreatedAt:   now,
	}).Error
}

func (s *QueueService) userModerationViolationCount(ctx context.Context, userID int64) int {
	var count int64
	_ = s.db.WithContext(ctx).Model(&domain.AIModerationLog{}).Where("user_id = ? AND status = ?", userID, "flagged").Count(&count).Error
	return int(count)
}

func normalizeAIModerationTarget(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AIModerationTargetPost, "note":
		return AIModerationTargetPost
	case AIModerationTargetComment:
		return AIModerationTargetComment
	default:
		return ""
	}
}

func actorIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func moderationDailyCap(settings *SettingsService) float64 {
	if settings == nil {
		return 50
	}
	return float64(settings.Int("points_daily_cap", 50))
}

func boolPtr(value bool) *bool {
	return &value
}
