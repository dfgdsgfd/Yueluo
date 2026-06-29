package handlers

import "context"

func (h NativeHandlers) enqueueAICommentReply(ctx context.Context, triggerCommentID int64) map[string]any {
	if h.Queue == nil || !h.Queue.AICommentReplyReady() || triggerCommentID == 0 {
		return nil
	}
	job, err := h.Queue.EnqueueAICommentReply(ctx, triggerCommentID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return job
}
