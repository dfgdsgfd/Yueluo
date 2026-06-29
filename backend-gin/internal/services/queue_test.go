package services

import "testing"

func TestQueueEventCountsGroupsRecentEvents(t *testing.T) {
	events := []map[string]any{
		{"queue": QueueVideoTranscoding, "event": "enqueued", "at": int64(1000)},
		{"queue": QueueVideoTranscoding, "event": "started", "at": int64(1500)},
		{"queue": QueueVideoTranscoding, "event": "completed", "at": int64(3000)},
		{"queue": QueueBatchNoteCreate, "event": "failed", "at": int64(2500)},
	}
	counts := queueEventCounts(events)
	video := counts[QueueVideoTranscoding]
	if video["total"] != 3 || video["completed"] != 1 || video["lastAt"] != int64(3000) {
		t.Fatalf("unexpected video event counts: %#v", video)
	}
	batch := counts[QueueBatchNoteCreate]
	if batch["total"] != 1 || batch["failed"] != 1 {
		t.Fatalf("unexpected batch event counts: %#v", batch)
	}
}
