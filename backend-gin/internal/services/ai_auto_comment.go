package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

const aiAutoCommentAuditSource = "ai_auto_comment"
const aiAutoCommentInlineImageMaxBytes = 4 << 20

func (s *QueueService) AIPostAutoCommentReady() bool {
	if s == nil || s.ai == nil || !s.Available() {
		return false
	}
	cfg := s.ai.Config()
	return cfg.Enabled && strings.TrimSpace(cfg.APIKey) != "" &&
		cfg.AutoComment.Enabled && aiAutoCommentHasBotSource(cfg.AutoComment)
}

func (s *QueueService) EnqueueAIPostAutoComment(ctx context.Context, postID int64, authorID ...int64) (map[string]any, error) {
	if s == nil || s.ai == nil {
		return nil, errors.New("ai service unavailable")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	cfg := s.ai.Config()
	autoComment := cfg.AutoComment
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || !autoComment.Enabled || !aiAutoCommentHasBotSource(autoComment) {
		return nil, errors.New("ai auto comment disabled")
	}
	if postID <= 0 {
		return nil, errors.New("post id is required")
	}
	author := firstPositiveInt64(authorID...)
	if allowed, err := s.aiAutoCommentAllowedForPostAuthor(ctx, postID, author); err != nil {
		return nil, err
	} else if !allowed {
		return nil, errors.New("ai auto comment disabled by author")
	}
	payload := aiPostAutoCommentTaskPayload{
		PostID:     postID,
		AuthorID:   author,
		BotUserID:  autoComment.BotUserID,
		EnqueuedAt: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	taskID := aiPostAutoCommentTaskID(postID)
	options := []asynq.Option{
		asynq.Queue(QueueAITask),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(time.Duration(maxInt(30, cfg.TimeoutSeconds+30)) * time.Second),
	}
	if autoComment.DelaySeconds > 0 {
		options = append(options, asynq.ProcessIn(time.Duration(autoComment.DelaySeconds)*time.Second))
	}
	if retention := s.completedRetention(QueueAITask, 24*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskAIPostAutoComment, data, QueueAITask, payload.EnqueuedAt), options...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			s.recordQueueEvent(ctx, queueEvent{
				TaskID: taskID,
				Queue:  QueueAITask,
				Type:   TaskAIPostAutoComment,
				Event:  "duplicate",
				State:  "duplicate",
				At:     payload.EnqueuedAt,
				Detail: map[string]any{"postId": payload.PostID, "authorId": payload.AuthorID, "botUserId": payload.BotUserID},
			})
			return map[string]any{"id": taskID, "queue": QueueAITask, "duplicate": true, "enqueuedAt": payload.EnqueuedAt}, nil
		}
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskAIPostAutoComment,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"postId": payload.PostID, "authorId": payload.AuthorID, "botUserId": payload.BotUserID},
	})
	return map[string]any{"id": info.ID, "queue": info.Queue, "state": info.State.String(), "enqueuedAt": payload.EnqueuedAt}, nil
}

func (s *QueueService) processAIPostAutoComment(ctx context.Context, task *asynq.Task) error {
	var payload aiPostAutoCommentTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	commentID, err := s.createAIPostAutoComment(ctx, payload)
	if err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{"postId": payload.PostID, "commentId": commentID})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) createAIPostAutoComment(ctx context.Context, payload aiPostAutoCommentTaskPayload) (int64, error) {
	if s == nil || s.db == nil || s.ai == nil {
		return 0, fmt.Errorf("%w: ai auto comment dependencies unavailable", asynq.SkipRetry)
	}
	cfg := s.ai.Config()
	autoComment := cfg.AutoComment
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || !autoComment.Enabled {
		return 0, fmt.Errorf("%w: ai auto comment disabled", asynq.SkipRetry)
	}
	botUserID, err := s.resolveAIAutoCommentBotUserID(ctx, autoComment, payload.BotUserID)
	if err != nil {
		return 0, err
	}
	if botUserID <= 0 || payload.PostID <= 0 {
		return 0, fmt.Errorf("%w: invalid ai auto comment payload", asynq.SkipRetry)
	}
	if allowed, err := s.aiAutoCommentAllowedForPostAuthor(ctx, payload.PostID, payload.AuthorID); err != nil {
		return 0, err
	} else if !allowed {
		return 0, fmt.Errorf("%w: ai auto comment disabled by author", asynq.SkipRetry)
	}
	contextData, err := s.loadAIPostAutoCommentContext(ctx, payload.PostID, botUserID, autoComment.MaxImages, autoComment.ImageSelectionMode)
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
		Type:        AITaskPostAutoComment,
		Locale:      "auto",
		Input:       contextData.PromptInput,
		TemplateKey: autoComment.TemplateKey,
		Variables: map[string]any{
			"postId":                contextData.Post.ID,
			"authorId":              contextData.Author.UserID,
			"authorNickname":        contextData.Author.Nickname,
			"imageCount":            contextData.ImageCandidates,
			"imageSendSuccessCount": len(contextData.Images),
			"imageSelectionMode":    contextData.ImageSelectionMode,
			"style":                 autoComment.Style,
			"styleInstruction":      aiPromptStyleInstruction(autoComment.Style),
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
		return 0, fmt.Errorf("%w: ai auto comment was empty", asynq.SkipRetry)
	}
	result, err := repositories.NewContentRepository(s.db).CreateComment(ctx, contextData.Bot.ID, contextData.Post.ID, nil, content, repositories.CreateCommentOptions{
		AuditResult: jsonData(map[string]any{
			"source":                aiAutoCommentAuditSource,
			"taskType":              AITaskPostAutoComment,
			"postId":                contextData.Post.ID,
			"botUserId":             contextData.Bot.ID,
			"imageCandidateCount":   contextData.ImageCandidates,
			"imageSendSuccessCount": len(contextData.Images),
			"imageSelectionMode":    contextData.ImageSelectionMode,
		}),
	})
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.Comment.Comment.ID, nil
}

type aiPostAutoCommentContext struct {
	Post        domain.Post
	Author      domain.User
	Bot         domain.User
	Images      []AIImageInput
	PromptInput string

	ImageCandidates    int
	ImageSelectionMode string
}

var errAIAutoCommentSkipped = errors.New("ai auto comment skipped")

func aiAutoCommentHasBotSource(cfg AIAutoCommentConfig) bool {
	return cfg.BotUserID > 0 || (cfg.BotUserIDMin > 0 && cfg.BotUserIDMax > 0)
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (s *QueueService) aiAutoCommentAllowedForPostAuthor(ctx context.Context, postID int64, authorID int64) (bool, error) {
	if s == nil || s.db == nil || postID <= 0 {
		return false, fmt.Errorf("%w: invalid ai auto comment author check", asynq.SkipRetry)
	}
	if authorID <= 0 {
		var post domain.Post
		if err := s.db.WithContext(ctx).Select("user_id").Where("id = ?", postID).First(&post).Error; err != nil {
			return false, err
		}
		authorID = post.UserID
	}
	var user domain.User
	if err := s.db.WithContext(ctx).Select("id", "ai_auto_comment_enabled").Where("id = ?", authorID).First(&user).Error; err != nil {
		return false, err
	}
	return user.AIAutoCommentEnabled, nil
}

func (s *QueueService) resolveAIAutoCommentBotUserID(ctx context.Context, cfg AIAutoCommentConfig, payloadBotUserID int64) (int64, error) {
	if cfg.BotUserIDMin > 0 && cfg.BotUserIDMax > 0 {
		minID, maxID := cfg.BotUserIDMin, cfg.BotUserIDMax
		if minID > maxID {
			minID, maxID = maxID, minID
		}
		var user domain.User
		query := s.db.WithContext(ctx).
			Where("id BETWEEN ? AND ? AND is_active = ?", minID, maxID, true)
		if err := query.Order(aiRandomOrderExpression(s.db)).Limit(1).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, fmt.Errorf("%w: no active ai auto comment bot in uid range", asynq.SkipRetry)
			}
			return 0, err
		}
		return user.ID, nil
	}
	if cfg.BotUserID > 0 {
		return cfg.BotUserID, nil
	}
	return payloadBotUserID, nil
}

func aiRandomOrderExpression(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "RANDOM()"
	}
	switch strings.ToLower(db.Dialector.Name()) {
	case "mysql":
		return "RAND()"
	default:
		return "RANDOM()"
	}
}

func (s *QueueService) loadAIPostAutoCommentContext(ctx context.Context, postID int64, botUserID int64, maxImages int, imageSelectionMode string) (aiPostAutoCommentContext, error) {
	var out aiPostAutoCommentContext
	imageSelectionMode = normalizeAIImageSelectionMode(imageSelectionMode)
	out.ImageSelectionMode = imageSelectionMode
	if err := s.db.WithContext(ctx).Where("id = ?", botUserID).First(&out.Bot).Error; err != nil {
		return out, err
	}
	if !out.Bot.IsActive {
		return out, errAIAutoCommentSkipped
	}
	if err := s.db.WithContext(ctx).Where("id = ?", postID).First(&out.Post).Error; err != nil {
		return out, err
	}
	if out.Post.IsDraft || normalizeAIPostVisibility(out.Post.Visibility) != repositories.VisibilityPublic || out.Post.UserID == out.Bot.ID {
		return out, errAIAutoCommentSkipped
	}
	var comments []domain.Comment
	if err := s.db.WithContext(ctx).Model(&domain.Comment{}).
		Select("id", "user_id", "audit_result").
		Where("post_id = ? AND parent_id IS NULL", out.Post.ID).
		Find(&comments).Error; err != nil {
		return out, err
	}
	for _, comment := range comments {
		if comment.UserID == out.Bot.ID || commentHasAIAutoCommentMarker(comment.AuditResult) {
			return out, errAIAutoCommentSkipped
		}
	}
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
	if maxImages > 0 {
		if err := s.db.WithContext(ctx).Where("post_id = ?", out.Post.ID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
			return out, err
		}
	}
	out.ImageCandidates = len(images)
	selectedImages := selectAIImageSample(images, maxImages, imageSelectionMode)
	out.Images = s.aiImageInputs(ctx, selectedImages)
	out.PromptInput = aiPostAutoCommentPromptInput(out.Post, out.Author, category, tags, len(out.Images))
	return out, nil
}

func (s *QueueService) aiImageInputs(ctx context.Context, images []domain.PostImage) []AIImageInput {
	out := make([]AIImageInput, 0, len(images))
	for index, image := range repositories.NormalizePostImagesForAccess(images) {
		imageURL := s.aiImageURL(image.ImageURL)
		if imageURL == "" {
			continue
		}
		input := AIImageInput{
			URL: imageURL,
			Alt: fmt.Sprintf("post image %d", index+1),
		}
		if dataURL := s.aiImageDataURL(ctx, image.ImageURL); dataURL != "" {
			input.DataURL = dataURL
		}
		out = append(out, input)
	}
	return out
}

func (s *QueueService) aiImageDataURL(ctx context.Context, raw string) string {
	data, contentType, _, err := s.readProtectedImageSource(ctx, raw)
	if err != nil || len(data) == 0 || len(data) > aiAutoCommentInlineImageMaxBytes || !strings.HasPrefix(contentType, "image/") {
		return ""
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (s *QueueService) aiImageURL(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" || strings.HasPrefix(text, "data:") {
		return text
	}
	signed := strings.TrimSpace(SignFileURL(text, s.cfg))
	if signed == "" {
		return signed
	}
	if aiAbsoluteHTTPURL(signed) {
		if aiLocalHTTPBaseURL(signed) {
			if base := s.aiImagePublicBaseURL(); base != "" && !aiLocalHTTPBaseURL(base) {
				if parsed, err := url.Parse(signed); err == nil {
					return strings.TrimRight(base, "/") + parsed.RequestURI()
				}
			}
		}
		return signed
	}
	if strings.HasPrefix(signed, "/") {
		if base := s.aiImagePublicBaseURL(); base != "" {
			return strings.TrimRight(base, "/") + signed
		}
	}
	return signed
}

func (s *QueueService) aiImagePublicBaseURL() string {
	for _, candidate := range []string{
		s.cfg.API.BaseURL,
		s.cfg.Upload.LocalBase,
		s.cfg.Frontend.BaseURL,
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && aiAbsoluteHTTPURL(candidate) {
			return strings.TrimRight(candidate, "/")
		}
	}
	return ""
}

func aiAbsoluteHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func aiLocalHTTPBaseURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func aiPostAutoCommentPromptInput(post domain.Post, author domain.User, category domain.Category, tags []domain.Tag, imageCount int) string {
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		if name := strings.TrimSpace(tag.Name); name != "" {
			tagNames = append(tagNames, name)
		}
	}
	lines := []string{
		fmt.Sprintf("Post ID: %d", post.ID),
		fmt.Sprintf("Author: %s (%s)", nonEmptyString(author.Nickname, "unknown"), nonEmptyString(author.UserID, "unknown")),
		fmt.Sprintf("Title: %s", nonEmptyString(post.Title, "(untitled)")),
		fmt.Sprintf("Content:\n%s", nonEmptyString(post.Content, "(empty)")),
	}
	if category.ID != 0 {
		lines = append(lines, fmt.Sprintf("Category: %s", nonEmptyString(category.Name, fmt.Sprint(category.ID))))
	}
	if len(tagNames) > 0 {
		lines = append(lines, "Tags: "+strings.Join(tagNames, ", "))
	}
	lines = append(lines, fmt.Sprintf("Images attached: %d", imageCount))
	return strings.Join(lines, "\n\n")
}

func normalizeAIPostVisibility(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return repositories.VisibilityPublic
	}
	return value
}

func normalizeAIAutoCommentText(value string) string {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "```markdown")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > 800 {
		runes := []rune(text)
		text = strings.TrimSpace(string(runes[:800]))
	}
	return text
}

func commentHasAIAutoCommentMarker(raw []byte) bool {
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return false
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(meta["source"])) == aiAutoCommentAuditSource ||
		strings.TrimSpace(fmt.Sprint(meta["taskType"])) == AITaskPostAutoComment
}

func aiAutoCommentDisplayID(user domain.User) *string {
	display := strings.TrimSpace(user.UserID)
	if user.XiseID != nil && strings.TrimSpace(*user.XiseID) != "" {
		display = strings.TrimSpace(*user.XiseID)
	}
	if display == "" {
		display = fmt.Sprint(user.ID)
	}
	return &display
}

func aiPostAutoCommentTaskID(postID int64) string {
	return fmt.Sprintf("%s:%d", TaskAIPostAutoComment, postID)
}
