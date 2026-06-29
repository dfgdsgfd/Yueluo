package handlers

import (
	"context"

	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) enqueueVideoTranscoding(ctx context.Context, input services.VideoTranscodingInput) (map[string]any, bool) {
	if h.Queue == nil || !h.Queue.VideoTranscodingReady() {
		return nil, false
	}
	job, err := h.Queue.EnqueueVideoTranscoding(ctx, input)
	if err != nil {
		return map[string]any{"error": err.Error()}, false
	}
	return job, true
}

func (h NativeHandlers) enqueueVideoTranscodingForPost(ctx context.Context, postID int64) []map[string]any {
	if h.Queue == nil || !h.Queue.VideoTranscodingReady() || postID == 0 {
		return nil
	}
	jobs, _ := h.Queue.EnqueueVideoTranscodingForPost(ctx, postID)
	return jobs
}
