package handlers

import (
	"context"

	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) commentAIModerationEnabled() bool {
	if h.AI == nil || h.Queue == nil || !h.Queue.Available() {
		return false
	}
	cfg := h.AI.Config()
	return cfg.Enabled && cfg.Moderation.Comment.Enabled
}

func (h NativeHandlers) postAIModerationEnabled() bool {
	if h.AI == nil || h.Queue == nil || !h.Queue.Available() {
		return false
	}
	cfg := h.AI.Config()
	return cfg.Enabled && cfg.Moderation.Post.Enabled
}

func (h NativeHandlers) enqueueAIModeration(ctx context.Context, targetType string, targetID int64, userID int64, originalVisibility string) map[string]any {
	if h.Queue == nil || targetID <= 0 || userID <= 0 {
		return nil
	}
	job, err := h.Queue.EnqueueAIModeration(ctx, targetType, targetID, userID, originalVisibility)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return job
}

var _ = services.AIModerationTargetComment
