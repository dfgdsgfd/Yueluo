package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

const aiCommentReplyAuditSource = "ai_comment_reply"
const aiCommentMentionReplyAuditSource = "ai_comment_mention_reply"
const aiCommentReplyThreadContextLimit = 8

const (
	aiCommentReplyModeThread  = "thread_reply"
	aiCommentReplyModeMention = "mention"
)

func (s *QueueService) AICommentReplyReady() bool {
	if s == nil || s.ai == nil || !s.Available() {
		return false
	}
	cfg := s.ai.Config()
	return cfg.Enabled && strings.TrimSpace(cfg.APIKey) != "" && aiCommentReplyAnyModeEnabled(cfg.CommentReply)
}

func (s *QueueService) EnqueueAICommentReply(ctx context.Context, triggerCommentID int64) (map[string]any, error) {
	if s == nil || s.ai == nil {
		return nil, errors.New("ai service unavailable")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	cfg := s.ai.Config()
	commentReply := cfg.CommentReply
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || !aiCommentReplyAnyModeEnabled(commentReply) {
		return nil, errors.New("ai comment reply disabled")
	}
	if triggerCommentID <= 0 {
		return nil, errors.New("trigger comment id is required")
	}
	payload := aiCommentReplyTaskPayload{
		TriggerCommentID: triggerCommentID,
		EnqueuedAt:       time.Now().UnixMilli(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	taskID := aiCommentReplyTaskID(triggerCommentID)
	options := []asynq.Option{
		asynq.Queue(QueueAITask),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(time.Duration(maxInt(30, cfg.TimeoutSeconds+30)) * time.Second),
	}
	if commentReply.DelaySeconds > 0 {
		options = append(options, asynq.ProcessIn(time.Duration(commentReply.DelaySeconds)*time.Second))
	}
	if retention := s.completedRetention(QueueAITask, 24*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskAICommentReply, data, QueueAITask, payload.EnqueuedAt), options...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			s.recordQueueEvent(ctx, queueEvent{
				TaskID: taskID,
				Queue:  QueueAITask,
				Type:   TaskAICommentReply,
				Event:  "duplicate",
				State:  "duplicate",
				At:     payload.EnqueuedAt,
				Detail: map[string]any{"triggerCommentId": payload.TriggerCommentID},
			})
			return map[string]any{"id": taskID, "queue": QueueAITask, "duplicate": true, "enqueuedAt": payload.EnqueuedAt}, nil
		}
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskAICommentReply,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"triggerCommentId": payload.TriggerCommentID},
	})
	return map[string]any{"id": info.ID, "queue": info.Queue, "state": info.State.String(), "enqueuedAt": payload.EnqueuedAt}, nil
}

func (s *QueueService) processAICommentReply(ctx context.Context, task *asynq.Task) error {
	var payload aiCommentReplyTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	commentID, err := s.createAICommentReply(ctx, payload)
	if err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{"triggerCommentId": payload.TriggerCommentID, "commentId": commentID})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) createAICommentReply(ctx context.Context, payload aiCommentReplyTaskPayload) (int64, error) {
	if s == nil || s.db == nil || s.ai == nil {
		return 0, fmt.Errorf("%w: ai comment reply dependencies unavailable", asynq.SkipRetry)
	}
	cfg := s.ai.Config()
	commentReply := cfg.CommentReply
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || !aiCommentReplyAnyModeEnabled(commentReply) {
		return 0, fmt.Errorf("%w: ai comment reply disabled", asynq.SkipRetry)
	}
	contextData, err := s.loadAICommentReplyContext(ctx, payload.TriggerCommentID, commentReply)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) ||
			errors.Is(err, repositories.ErrContentPostMissing) ||
			errors.Is(err, errAIAutoCommentSkipped) {
			return 0, fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		return 0, err
	}
	actorDisplayID := aiAutoCommentDisplayID(contextData.Bot)
	text, _, err := s.ai.RunText(ctx, AIRequest{
		Type:        contextData.TaskType,
		Locale:      "auto",
		Input:       contextData.PromptInput,
		TemplateKey: contextData.TemplateKey,
		Variables: map[string]any{
			"postId":                contextData.Post.ID,
			"rootAICommentId":       contextData.RootAIComment.ID,
			"triggerCommentId":      contextData.TriggerComment.ID,
			"replySequence":         contextData.ReplySequence,
			"maxReplies":            contextData.MaxReplies,
			"imageCount":            contextData.ImageCandidates,
			"imageSendSuccessCount": len(contextData.Images),
			"imageSelectionMode":    contextData.ImageSelectionMode,
			"style":                 commentReply.Style,
			"styleInstruction":      aiPromptStyleInstruction(commentReply.Style),
			"mentionName":           contextData.MentionName,
			"mentionQuery":          contextData.MentionQuery,
		},
		Images: contextData.Images,
	}, AIActor{
		Type:      "system",
		ID:        &contextData.Bot.ID,
		DisplayID: actorDisplayID,
	})
	if err != nil {
		return 0, err
	}
	content := normalizeAIAutoCommentText(text)
	if content == "" {
		return 0, fmt.Errorf("%w: ai comment reply was empty", asynq.SkipRetry)
	}
	parentID := contextData.TriggerComment.ID
	auditResult := map[string]any{
		"source":                   contextData.AuditSource,
		"taskType":                 contextData.TaskType,
		"mode":                     contextData.Mode,
		"postId":                   contextData.Post.ID,
		"triggerCommentID":         contextData.TriggerComment.ID,
		"botUserId":                contextData.Bot.ID,
		"replySequence":            contextData.ReplySequence,
		"imageCandidateCount":      contextData.ImageCandidates,
		"imageSendSuccessCount":    len(contextData.Images),
		"imageSelectionMode":       contextData.ImageSelectionMode,
		"triggerCommentAuthorUser": contextData.TriggerUser.ID,
	}
	if contextData.Mode == aiCommentReplyModeThread {
		auditResult["rootAICommentID"] = contextData.RootAIComment.ID
		auditResult["parentAICommentID"] = contextData.ParentAIComment.ID
		auditResult["maxRepliesPerAIComment"] = commentReply.MaxRepliesPerAIComment
	} else {
		auditResult["mentionName"] = contextData.MentionName
		auditResult["mentionQuery"] = contextData.MentionQuery
		auditResult["maxMentionRepliesPerPost"] = commentReply.MaxMentionRepliesPerPost
	}
	result, err := repositories.NewContentRepository(s.db).CreateComment(ctx, contextData.Bot.ID, contextData.Post.ID, &parentID, content, repositories.CreateCommentOptions{
		AuditResult: jsonData(auditResult),
	})
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.Comment.Comment.ID, nil
}

type aiCommentReplyContext struct {
	Post            domain.Post
	Author          domain.User
	Bot             domain.User
	TriggerUser     domain.User
	RootAIComment   domain.Comment
	ParentAIComment domain.Comment
	TriggerComment  domain.Comment
	Images          []AIImageInput
	PromptInput     string

	Mode               string
	TaskType           string
	TemplateKey        string
	AuditSource        string
	ReplySequence      int
	MaxReplies         int
	ImageCandidates    int
	ImageSelectionMode string
	MentionName        string
	MentionQuery       string
}

func (s *QueueService) loadAICommentReplyContext(ctx context.Context, triggerCommentID int64, cfg AICommentReplyConfig) (aiCommentReplyContext, error) {
	var out aiCommentReplyContext
	err := errAIAutoCommentSkipped
	if cfg.Enabled {
		out, err = s.loadAIThreadCommentReplyContext(ctx, triggerCommentID, cfg)
		if err == nil {
			return out, nil
		}
	}
	if cfg.MentionEnabled {
		mentionOut, mentionErr := s.loadAICommentMentionReplyContext(ctx, triggerCommentID, cfg)
		if mentionErr == nil {
			return mentionOut, nil
		}
		if !cfg.Enabled {
			return mentionOut, mentionErr
		}
	}
	return out, err
}

func (s *QueueService) loadAIThreadCommentReplyContext(ctx context.Context, triggerCommentID int64, cfg AICommentReplyConfig) (aiCommentReplyContext, error) {
	var out aiCommentReplyContext
	if !cfg.Enabled || triggerCommentID <= 0 {
		return out, errAIAutoCommentSkipped
	}
	out.Mode = aiCommentReplyModeThread
	out.TaskType = AITaskCommentReply
	out.TemplateKey = nonEmptyString(cfg.TemplateKey, "comment_reply")
	out.AuditSource = aiCommentReplyAuditSource
	out.MaxReplies = cfg.MaxRepliesPerAIComment
	cfg.ImageSelectionMode = normalizeAIImageSelectionMode(cfg.ImageSelectionMode)
	out.ImageSelectionMode = cfg.ImageSelectionMode
	if err := s.db.WithContext(ctx).Where("id = ?", triggerCommentID).First(&out.TriggerComment).Error; err != nil {
		return out, err
	}
	if out.TriggerComment.ParentID == nil || !visibleAIReplyTrigger(out.TriggerComment) || commentHasAnyAICommentMarker(out.TriggerComment.AuditResult) {
		return out, errAIAutoCommentSkipped
	}
	if err := s.db.WithContext(ctx).Where("id = ?", *out.TriggerComment.ParentID).First(&out.ParentAIComment).Error; err != nil {
		return out, err
	}
	rootID, ok := rootAICommentIDForReplyParent(out.ParentAIComment)
	if !ok {
		return out, errAIAutoCommentSkipped
	}
	if out.ParentAIComment.PostID != out.TriggerComment.PostID {
		return out, errAIAutoCommentSkipped
	}
	if out.ParentAIComment.ID == rootID {
		out.RootAIComment = out.ParentAIComment
	} else if err := s.db.WithContext(ctx).Where("id = ?", rootID).First(&out.RootAIComment).Error; err != nil {
		return out, err
	}
	if out.RootAIComment.PostID != out.TriggerComment.PostID || !commentHasAnyAICommentMarker(out.RootAIComment.AuditResult) {
		return out, errAIAutoCommentSkipped
	}
	if err := s.db.WithContext(ctx).Where("id = ?", out.RootAIComment.UserID).First(&out.Bot).Error; err != nil {
		return out, err
	}
	if !out.Bot.IsActive || out.TriggerComment.UserID == out.Bot.ID {
		return out, errAIAutoCommentSkipped
	}
	if err := s.db.WithContext(ctx).Where("id = ?", out.TriggerComment.UserID).First(&out.TriggerUser).Error; err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", out.TriggerComment.PostID).First(&out.Post).Error; err != nil {
		return out, err
	}
	if out.Post.IsDraft || normalizeAIPostVisibility(out.Post.Visibility) != repositories.VisibilityPublic || out.Post.UserID == out.Bot.ID {
		return out, errAIAutoCommentSkipped
	}
	var existing []domain.Comment
	if err := s.db.WithContext(ctx).
		Where("post_id = ?", out.Post.ID).
		Order("created_at ASC, id ASC").
		Find(&existing).Error; err != nil {
		return out, err
	}
	replyCount, triggerReplyExists := aiCommentReplyCounts(existing, out.RootAIComment.ID, out.TriggerComment.ID)
	if triggerReplyExists || replyCount >= cfg.MaxRepliesPerAIComment {
		return out, errAIAutoCommentSkipped
	}
	out.ReplySequence = replyCount + 1
	if err := s.db.WithContext(ctx).Where("id = ?", out.Post.UserID).First(&out.Author).Error; err != nil {
		return out, err
	}
	var category domain.Category
	if out.Post.CategoryID != nil {
		_ = s.db.WithContext(ctx).Where("id = ?", *out.Post.CategoryID).First(&category).Error
	}
	var tags []domain.Tag
	_ = s.db.WithContext(ctx).
		Table("tags").
		Select("tags.*").
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Where("post_tags.post_id = ?", out.Post.ID).
		Order("tags.name ASC").
		Find(&tags).Error
	var images []domain.PostImage
	if cfg.MaxImages > 0 {
		if err := s.db.WithContext(ctx).Where("post_id = ?", out.Post.ID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
			return out, err
		}
	}
	out.ImageCandidates = len(images)
	out.Images = s.aiImageInputs(ctx, selectAIImageSample(images, cfg.MaxImages, cfg.ImageSelectionMode))
	out.PromptInput = aiCommentReplyPromptInput(out, category, tags, aiCommentReplyThreadContext(existing, out.RootAIComment.ID))
	return out, nil
}

func (s *QueueService) loadAICommentMentionReplyContext(ctx context.Context, triggerCommentID int64, cfg AICommentReplyConfig) (aiCommentReplyContext, error) {
	var out aiCommentReplyContext
	if !cfg.MentionEnabled || triggerCommentID <= 0 {
		return out, errAIAutoCommentSkipped
	}
	mentionName := normalizeAICommentMentionName(cfg.MentionName)
	if mentionName == "" || cfg.MentionBotUserIDMin <= 0 || cfg.MentionBotUserIDMax <= 0 {
		return out, errAIAutoCommentSkipped
	}
	out.Mode = aiCommentReplyModeMention
	out.TaskType = AITaskCommentMentionReply
	out.TemplateKey = nonEmptyString(cfg.MentionTemplateKey, "comment_mention_reply")
	out.AuditSource = aiCommentMentionReplyAuditSource
	out.MaxReplies = cfg.MaxMentionRepliesPerPost
	out.MentionName = mentionName
	cfg.ImageSelectionMode = normalizeAIImageSelectionMode(cfg.ImageSelectionMode)
	out.ImageSelectionMode = cfg.ImageSelectionMode
	if err := s.db.WithContext(ctx).Where("id = ?", triggerCommentID).First(&out.TriggerComment).Error; err != nil {
		return out, err
	}
	if !visibleAIReplyTrigger(out.TriggerComment) || commentHasAnyAICommentMarker(out.TriggerComment.AuditResult) {
		return out, errAIAutoCommentSkipped
	}
	mentionQuery, mentioned := aiCommentMentionQuery(out.TriggerComment.Content, mentionName)
	if !mentioned {
		return out, errAIAutoCommentSkipped
	}
	out.MentionQuery = mentionQuery
	if err := s.db.WithContext(ctx).Where("id = ?", out.TriggerComment.UserID).First(&out.TriggerUser).Error; err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", out.TriggerComment.PostID).First(&out.Post).Error; err != nil {
		return out, err
	}
	if out.Post.IsDraft || normalizeAIPostVisibility(out.Post.Visibility) != repositories.VisibilityPublic {
		return out, errAIAutoCommentSkipped
	}
	if err := s.resolveAICommentMentionBot(ctx, cfg, out.Post, out.TriggerComment, &out.Bot); err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", out.Post.UserID).First(&out.Author).Error; err != nil {
		return out, err
	}
	var existing []domain.Comment
	if err := s.db.WithContext(ctx).
		Where("post_id = ?", out.Post.ID).
		Order("created_at ASC, id ASC").
		Find(&existing).Error; err != nil {
		return out, err
	}
	replyCount, triggerReplyExists := aiCommentMentionReplyCounts(existing, out.TriggerComment.ID)
	if triggerReplyExists || replyCount >= cfg.MaxMentionRepliesPerPost {
		return out, errAIAutoCommentSkipped
	}
	out.ReplySequence = replyCount + 1
	var category domain.Category
	if out.Post.CategoryID != nil {
		_ = s.db.WithContext(ctx).Where("id = ?", *out.Post.CategoryID).First(&category).Error
	}
	var tags []domain.Tag
	_ = s.db.WithContext(ctx).
		Table("tags").
		Select("tags.*").
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Where("post_tags.post_id = ?", out.Post.ID).
		Order("tags.name ASC").
		Find(&tags).Error
	var parent domain.Comment
	if out.TriggerComment.ParentID != nil {
		_ = s.db.WithContext(ctx).Where("id = ?", *out.TriggerComment.ParentID).First(&parent).Error
	}
	var images []domain.PostImage
	if cfg.MaxImages > 0 {
		if err := s.db.WithContext(ctx).Where("post_id = ?", out.Post.ID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
			return out, err
		}
	}
	out.ImageCandidates = len(images)
	out.Images = s.aiImageInputs(ctx, selectAIImageSample(images, cfg.MaxImages, cfg.ImageSelectionMode))
	out.PromptInput = aiCommentMentionReplyPromptInput(out, category, tags, parent, aiCommentRecentPostContext(existing, out.TriggerComment.ID))
	return out, nil
}

func (s *QueueService) resolveAICommentMentionBot(ctx context.Context, cfg AICommentReplyConfig, post domain.Post, trigger domain.Comment, out *domain.User) error {
	minID, maxID := cfg.MentionBotUserIDMin, cfg.MentionBotUserIDMax
	if minID > maxID {
		minID, maxID = maxID, minID
	}
	query := s.db.WithContext(ctx).
		Where("id BETWEEN ? AND ? AND is_active = ?", minID, maxID, true).
		Where("id <> ? AND id <> ?", trigger.UserID, post.UserID)
	if err := query.Order(aiRandomOrderExpression(s.db)).Limit(1).First(out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: no active ai mention bot in uid range", asynq.SkipRetry)
		}
		return err
	}
	return nil
}

func aiCommentReplyAnyModeEnabled(cfg AICommentReplyConfig) bool {
	return cfg.Enabled || cfg.MentionEnabled
}

func visibleAIReplyTrigger(comment domain.Comment) bool {
	return comment.IsPublic && comment.AuditStatus == 1 && strings.TrimSpace(comment.Content) != ""
}

func rootAICommentIDForReplyParent(parent domain.Comment) (int64, bool) {
	if commentHasAIAutoCommentMarker(parent.AuditResult) || commentHasAICommentMentionReplyMarker(parent.AuditResult) {
		return parent.ID, true
	}
	if !commentHasAICommentReplyMarker(parent.AuditResult) {
		return 0, false
	}
	rootID := aiCommentMetaInt64(aiCommentAuditMap(parent.AuditResult), "rootAICommentID", "root_ai_comment_id", "rootAiCommentId")
	return rootID, rootID > 0
}

func aiCommentReplyCounts(comments []domain.Comment, rootID int64, triggerCommentID int64) (int, bool) {
	count := 0
	triggerReplyExists := false
	for _, comment := range comments {
		meta := aiCommentAuditMap(comment.AuditResult)
		if strings.TrimSpace(fmt.Sprint(meta["source"])) != aiCommentReplyAuditSource &&
			strings.TrimSpace(fmt.Sprint(meta["taskType"])) != AITaskCommentReply {
			continue
		}
		if aiCommentMetaInt64(meta, "rootAICommentID", "root_ai_comment_id", "rootAiCommentId") != rootID {
			continue
		}
		count++
		if aiCommentMetaInt64(meta, "triggerCommentID", "trigger_comment_id", "triggerCommentId") == triggerCommentID {
			triggerReplyExists = true
		}
	}
	return count, triggerReplyExists
}

func aiCommentMentionReplyCounts(comments []domain.Comment, triggerCommentID int64) (int, bool) {
	count := 0
	triggerReplyExists := false
	for _, comment := range comments {
		meta := aiCommentAuditMap(comment.AuditResult)
		if strings.TrimSpace(fmt.Sprint(meta["source"])) != aiCommentMentionReplyAuditSource &&
			strings.TrimSpace(fmt.Sprint(meta["taskType"])) != AITaskCommentMentionReply {
			continue
		}
		count++
		if aiCommentMetaInt64(meta, "triggerCommentID", "trigger_comment_id", "triggerCommentId") == triggerCommentID {
			triggerReplyExists = true
		}
	}
	return count, triggerReplyExists
}

func aiCommentReplyPromptInput(data aiCommentReplyContext, category domain.Category, tags []domain.Tag, thread string) string {
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		if name := strings.TrimSpace(tag.Name); name != "" {
			tagNames = append(tagNames, name)
		}
	}
	lines := []string{
		fmt.Sprintf("Post ID: %d", data.Post.ID),
		fmt.Sprintf("Author: %s (%s)", nonEmptyString(data.Author.Nickname, "unknown"), nonEmptyString(data.Author.UserID, "unknown")),
		fmt.Sprintf("Title: %s", nonEmptyString(data.Post.Title, "(untitled)")),
		fmt.Sprintf("Content:\n%s", nonEmptyString(data.Post.Content, "(empty)")),
	}
	if category.ID != 0 {
		lines = append(lines, fmt.Sprintf("Category: %s", nonEmptyString(category.Name, fmt.Sprint(category.ID))))
	}
	if len(tagNames) > 0 {
		lines = append(lines, "Tags: "+strings.Join(tagNames, ", "))
	}
	lines = append(lines,
		fmt.Sprintf("Images attached: %d", len(data.Images)),
		fmt.Sprintf("Original AI comment (comment %d):\n%s", data.RootAIComment.ID, nonEmptyString(data.RootAIComment.Content, "(empty)")),
		fmt.Sprintf("User reply to answer (comment %d by %s):\n%s", data.TriggerComment.ID, nonEmptyString(data.TriggerUser.Nickname, "user"), nonEmptyString(data.TriggerComment.Content, "(empty)")),
		fmt.Sprintf("Reply sequence: %d", data.ReplySequence),
	)
	if data.ParentAIComment.ID != data.RootAIComment.ID {
		lines = append(lines, fmt.Sprintf("Parent AI reply (comment %d):\n%s", data.ParentAIComment.ID, nonEmptyString(data.ParentAIComment.Content, "(empty)")))
	}
	if strings.TrimSpace(thread) != "" {
		lines = append(lines, "Recent thread context:\n"+thread)
	}
	return strings.Join(lines, "\n\n")
}

func aiCommentMentionReplyPromptInput(data aiCommentReplyContext, category domain.Category, tags []domain.Tag, parent domain.Comment, recent string) string {
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		if name := strings.TrimSpace(tag.Name); name != "" {
			tagNames = append(tagNames, name)
		}
	}
	lines := []string{
		fmt.Sprintf("Post ID: %d", data.Post.ID),
		fmt.Sprintf("Author: %s (%s)", nonEmptyString(data.Author.Nickname, "unknown"), nonEmptyString(data.Author.UserID, "unknown")),
		fmt.Sprintf("Title: %s", nonEmptyString(data.Post.Title, "(untitled)")),
		fmt.Sprintf("Content:\n%s", nonEmptyString(data.Post.Content, "(empty)")),
	}
	if category.ID != 0 {
		lines = append(lines, fmt.Sprintf("Category: %s", nonEmptyString(category.Name, fmt.Sprint(category.ID))))
	}
	if len(tagNames) > 0 {
		lines = append(lines, "Tags: "+strings.Join(tagNames, ", "))
	}
	lines = append(lines,
		fmt.Sprintf("Images attached: %d", len(data.Images)),
		fmt.Sprintf("Configured mention name: @%s", data.MentionName),
		fmt.Sprintf("User mention comment (comment %d by %s):\n%s", data.TriggerComment.ID, nonEmptyString(data.TriggerUser.Nickname, "user"), nonEmptyString(data.TriggerComment.Content, "(empty)")),
		fmt.Sprintf("User question after removing mention:\n%s", nonEmptyString(data.MentionQuery, "(empty)")),
		fmt.Sprintf("Mention reply sequence on this post: %d", data.ReplySequence),
	)
	if parent.ID != 0 {
		actor := "user"
		if commentHasAnyAICommentMarker(parent.AuditResult) {
			actor = "ai"
		}
		lines = append(lines, fmt.Sprintf("Parent comment (comment %d by %s):\n%s", parent.ID, actor, nonEmptyString(parent.Content, "(empty)")))
	}
	if strings.TrimSpace(recent) != "" {
		lines = append(lines, "Recent public comments on this post:\n"+recent)
	}
	return strings.Join(lines, "\n\n")
}

func aiCommentReplyThreadContext(comments []domain.Comment, rootID int64) string {
	children := map[int64][]domain.Comment{}
	for _, comment := range comments {
		if comment.ParentID == nil {
			continue
		}
		children[*comment.ParentID] = append(children[*comment.ParentID], comment)
	}
	thread := make([]domain.Comment, 0, aiCommentReplyThreadContextLimit)
	queue := []int64{rootID}
	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]
		for _, child := range children[parentID] {
			if child.IsPublic && child.AuditStatus == 1 {
				thread = append(thread, child)
			}
			queue = append(queue, child.ID)
		}
	}
	if len(thread) > aiCommentReplyThreadContextLimit {
		thread = thread[len(thread)-aiCommentReplyThreadContextLimit:]
	}
	lines := make([]string, 0, len(thread))
	for _, comment := range thread {
		actor := "user"
		if commentHasAnyAICommentMarker(comment.AuditResult) {
			actor = "ai"
		}
		lines = append(lines, fmt.Sprintf("- comment %d by %s: %s", comment.ID, actor, trimAICommentContextText(comment.Content, 260)))
	}
	return strings.Join(lines, "\n")
}

func aiCommentRecentPostContext(comments []domain.Comment, triggerCommentID int64) string {
	recent := make([]domain.Comment, 0, aiCommentReplyThreadContextLimit)
	for _, comment := range comments {
		if comment.ID == triggerCommentID || !comment.IsPublic || comment.AuditStatus != 1 {
			continue
		}
		recent = append(recent, comment)
	}
	if len(recent) > aiCommentReplyThreadContextLimit {
		recent = recent[len(recent)-aiCommentReplyThreadContextLimit:]
	}
	lines := make([]string, 0, len(recent))
	for _, comment := range recent {
		actor := "user"
		if commentHasAnyAICommentMarker(comment.AuditResult) {
			actor = "ai"
		}
		lines = append(lines, fmt.Sprintf("- comment %d by %s: %s", comment.ID, actor, trimAICommentContextText(comment.Content, 260)))
	}
	return strings.Join(lines, "\n")
}

func aiCommentMentionQuery(content string, mentionName string) (string, bool) {
	name := normalizeAICommentMentionName(mentionName)
	if name == "" {
		return "", false
	}
	text := strings.TrimSpace(content)
	target := "@" + strings.ToLower(name)
	lower := strings.ToLower(text)
	for offset := 0; offset < len(lower); {
		index := strings.Index(lower[offset:], target)
		if index < 0 {
			break
		}
		start := offset + index
		end := start + len(target)
		if validAICommentMentionBoundary(text, start, end) {
			query := strings.TrimSpace(text[:start] + text[end:])
			query = strings.Trim(query, " \t\r\n,，.。:：;；!！?？")
			return query, true
		}
		offset = start + 1
	}
	return "", false
}

func validAICommentMentionBoundary(text string, start int, end int) bool {
	if start > 0 {
		prev, _ := utf8.DecodeLastRuneInString(text[:start])
		if isAICommentMentionNameRune(prev) {
			return false
		}
	}
	if end < len(text) {
		next, _ := utf8.DecodeRuneInString(text[end:])
		if isAICommentMentionNameRune(next) {
			return false
		}
	}
	return true
}

func isAICommentMentionNameRune(value rune) bool {
	return value == '_' || value == '-' || value == '.' ||
		(value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'z') ||
		(value >= 'A' && value <= 'Z')
}

func trimAICommentContextText(value string, maxRunes int) string {
	text := strings.TrimSpace(value)
	if maxRunes <= 0 || utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func commentHasAnyAICommentMarker(raw []byte) bool {
	return commentHasAIAutoCommentMarker(raw) || commentHasAICommentReplyMarker(raw) || commentHasAICommentMentionReplyMarker(raw)
}

func commentHasAICommentReplyMarker(raw []byte) bool {
	meta := aiCommentAuditMap(raw)
	return strings.TrimSpace(fmt.Sprint(meta["source"])) == aiCommentReplyAuditSource ||
		strings.TrimSpace(fmt.Sprint(meta["taskType"])) == AITaskCommentReply
}

func commentHasAICommentMentionReplyMarker(raw []byte) bool {
	meta := aiCommentAuditMap(raw)
	return strings.TrimSpace(fmt.Sprint(meta["source"])) == aiCommentMentionReplyAuditSource ||
		strings.TrimSpace(fmt.Sprint(meta["taskType"])) == AITaskCommentMentionReply
}

func aiCommentAuditMap(raw []byte) map[string]any {
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return meta
}

func aiCommentMetaInt64(meta map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := meta[key]; ok {
			switch typed := value.(type) {
			case int64:
				return typed
			case int:
				return int64(typed)
			case float64:
				return int64(typed)
			case json.Number:
				parsed, _ := typed.Int64()
				return parsed
			case string:
				var parsed int64
				_, _ = fmt.Sscan(strings.TrimSpace(typed), &parsed)
				return parsed
			}
		}
	}
	return 0
}

func aiCommentReplyTaskID(triggerCommentID int64) string {
	return fmt.Sprintf("%s:%d", TaskAICommentReply, triggerCommentID)
}
