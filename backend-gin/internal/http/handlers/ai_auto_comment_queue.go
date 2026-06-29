package handlers

import "context"

func (h NativeHandlers) enqueueAIPostAutoComment(ctx context.Context, postID int64, authorID ...int64) map[string]any {
	if h.Queue == nil || !h.Queue.AIPostAutoCommentReady() || postID == 0 {
		return nil
	}
	job, err := h.Queue.EnqueueAIPostAutoComment(ctx, postID, authorID...)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return job
}
