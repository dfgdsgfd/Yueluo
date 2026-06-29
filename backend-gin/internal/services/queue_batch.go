package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"yuem-go/backend-gin/internal/domain"
)

func (s *QueueService) EnqueueBatchNoteCreate(ctx context.Context, notes []QueueBatchNote, userID int64, postType int, tags []string, isDraft bool) (string, []map[string]any, error) {
	batchID := fmt.Sprintf("%x", time.Now().UnixNano())
	if !s.Available() {
		return batchID, nil, errors.New("queue service disabled")
	}
	jobs := make([]map[string]any, 0, len(notes))
	for idx, note := range notes {
		payload := batchNoteTaskPayload{
			UserID:     userID,
			PostType:   postType,
			Title:      note.Title,
			Content:    note.Content,
			IsDraft:    isDraft,
			Files:      note.Files,
			Tags:       tags,
			CoverURL:   note.CoverURL,
			BatchID:    batchID,
			NoteIndex:  idx,
			TotalNotes: len(notes),
			EnqueuedAt: time.Now().UnixMilli(),
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return batchID, nil, err
		}
		options := []asynq.Option{
			asynq.Queue(QueueBatchNoteCreate),
			asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts+2)),
		}
		if retention := s.completedRetention(QueueBatchNoteCreate, 24*time.Hour); retention > 0 {
			options = append(options, asynq.Retention(retention))
		}
		info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskBatchNoteCreate, data, QueueBatchNoteCreate, payload.EnqueuedAt), options...)
		if err != nil {
			return batchID, nil, err
		}
		s.recordQueueEvent(ctx, queueEvent{
			TaskID: info.ID,
			Queue:  info.Queue,
			Type:   TaskBatchNoteCreate,
			Event:  "enqueued",
			State:  info.State.String(),
			At:     payload.EnqueuedAt,
			Detail: map[string]any{"noteIndex": idx, "batchId": batchID},
		})
		jobs = append(jobs, map[string]any{"id": info.ID, "noteIndex": idx, "queue": info.Queue, "state": info.State.String(), "enqueuedAt": payload.EnqueuedAt})
	}
	return batchID, jobs, nil
}

func (s *QueueService) BatchStatus(batchID string) (bool, map[string]any, error) {
	if !s.Available() {
		return false, map[string]any{"enabled": false}, nil
	}
	statuses := []string{"waiting", "active", "completed", "failed", "delayed", "retry"}
	result := map[string]any{"enabled": true, "batchId": batchID}
	counts := map[string]int{}
	completedJobs := []map[string]any{}
	failedJobs := []map[string]any{}
	total := 0
	for _, status := range statuses {
		tasks, err := s.listTasks(QueueBatchNoteCreate, status, 0, 100)
		if err != nil {
			return true, result, err
		}
		for _, task := range tasks {
			payload := batchNoteTaskPayload{}
			if json.Unmarshal(task.Payload, &payload) != nil || payload.BatchID != batchID {
				continue
			}
			counts[status]++
			total++
			if status == "completed" {
				completedJobs = append(completedJobs, map[string]any{"id": task.ID, "noteIndex": payload.NoteIndex, "postId": jsonAnyBytes(task.Result)})
			}
			if status == "failed" || status == "retry" {
				failedJobs = append(failedJobs, map[string]any{"id": task.ID, "noteIndex": payload.NoteIndex, "error": task.LastErr})
			}
		}
	}
	result["total"] = total
	result["waiting"] = counts["waiting"]
	result["active"] = counts["active"]
	result["completed"] = counts["completed"]
	result["failed"] = counts["failed"] + counts["retry"]
	result["completedJobs"] = completedJobs
	result["failedJobs"] = failedJobs
	return true, result, nil
}

func (s *QueueService) EnqueueImageProtectionPackage(ctx context.Context, jobID string, imageIDs []int64) (map[string]any, error) {
	return s.enqueueImagePackage(ctx, TaskImageProtection, jobID, imageIDs, "protected")
}

func (s *QueueService) EnqueuePostImageArchive(ctx context.Context, jobID string, imageIDs []int64) (map[string]any, error) {
	return s.enqueueImagePackage(ctx, TaskPostImageArchive, jobID, imageIDs, "post_archive")
}

func (s *QueueService) enqueueImagePackage(ctx context.Context, taskType, jobID string, imageIDs []int64, kind string) (map[string]any, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, errors.New("job id is required")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	payload := imageProtectionTaskPayload{JobID: jobID, ImageIDs: append([]int64(nil), imageIDs...), Kind: kind, EnqueuedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	options := []asynq.Option{
		asynq.Queue(QueueImageProtection),
		asynq.MaxRetry(0),
	}
	if retention := s.completedRetention(QueueImageProtection, 2*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(taskType, data, QueueImageProtection, payload.EnqueuedAt), options...)
	if err != nil {
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   taskType,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"jobId": jobID, "imageCount": len(imageIDs), "kind": kind},
	})
	position, eta, queueCount := s.ImageProtectionQueueEstimate(ctx, jobID)
	return map[string]any{"id": info.ID, "jobId": jobID, "queue": info.Queue, "state": info.State.String(), "queuePosition": position, "estimatedWaitSeconds": eta, "queueCount": queueCount, "enqueuedAt": payload.EnqueuedAt}, nil
}

func (s *QueueService) ImageProtectionQueueEstimate(ctx context.Context, jobID string) (int, int, int) {
	if !s.Available() || strings.TrimSpace(jobID) == "" {
		return 0, 0, 0
	}
	active, _ := s.listTasks(QueueImageProtection, "active", 0, 100)
	waiting, _ := s.listTasks(QueueImageProtection, "waiting", 0, 100)
	retrying, _ := s.listTasks(QueueImageProtection, "retry", 0, 100)
	delayed, _ := s.listTasks(QueueImageProtection, "delayed", 0, 100)
	queueCount := len(active) + len(waiting) + len(retrying) + len(delayed)
	secondsPerImage := s.imageProtectionSecondsPerImage(ctx)
	concurrency := maxInt(1, s.cfg.Queue.Concurrency.ImageProtection)
	for index, task := range active {
		payload := imageProtectionTaskPayload{}
		if json.Unmarshal(task.Payload, &payload) == nil && payload.JobID == jobID {
			return index + 1, 0, queueCount
		}
	}
	imagesAhead := imageProtectionTaskImageCount(active)
	for index, task := range waiting {
		payload := imageProtectionTaskPayload{}
		if json.Unmarshal(task.Payload, &payload) != nil {
			continue
		}
		position := len(active) + index + 1
		if payload.JobID == jobID {
			return position, int(math.Ceil(float64(imagesAhead) * secondsPerImage / float64(concurrency))), queueCount
		}
		imagesAhead += maxInt(1, len(payload.ImageIDs))
	}
	queued := append(retrying, delayed...)
	for index, task := range queued {
		payload := imageProtectionTaskPayload{}
		if json.Unmarshal(task.Payload, &payload) != nil {
			continue
		}
		position := len(active) + len(waiting) + index + 1
		if payload.JobID == jobID {
			return position, int(math.Ceil(float64(imagesAhead) * secondsPerImage / float64(concurrency))), queueCount
		}
		imagesAhead += maxInt(1, len(payload.ImageIDs))
	}
	return 0, 0, queueCount
}

func imageProtectionTaskImageCount(tasks []*asynq.TaskInfo) int {
	total := 0
	for _, task := range tasks {
		payload := imageProtectionTaskPayload{}
		if json.Unmarshal(task.Payload, &payload) == nil {
			total += maxInt(1, len(payload.ImageIDs))
		}
	}
	return total
}

func (s *QueueService) imageProtectionSecondsPerImage(ctx context.Context) float64 {
	const fallback = 20.0
	if s == nil || s.db == nil {
		return fallback
	}
	type durationRow struct {
		DurationSeconds float64 `gorm:"column:duration_seconds"`
		ImageCount      int     `gorm:"column:image_count"`
	}
	var rows []durationRow
	durationExpression := "EXTRACT(EPOCH FROM (finished_at - started_at))"
	if s.db.Dialector != nil {
		switch s.db.Dialector.Name() {
		case "sqlite":
			durationExpression = "(julianday(finished_at) - julianday(started_at)) * 86400"
		case "mysql":
			durationExpression = "TIMESTAMPDIFF(MICROSECOND, started_at, finished_at) / 1000000.0"
		}
	}
	err := s.db.WithContext(ctx).
		Model(&domain.ImageProtectionJob{}).
		Select(durationExpression+" AS duration_seconds, protected_image_count AS image_count").
		Where("status = ? AND (package_kind = ? OR package_kind = '') AND started_at IS NOT NULL AND finished_at IS NOT NULL AND protected_image_count > 0", imageProtectionStatusCompleted, "protected").
		Order("finished_at DESC").
		Limit(20).
		Scan(&rows).Error
	if err != nil {
		return fallback
	}
	totalSeconds := 0.0
	totalImages := 0
	for _, row := range rows {
		if row.DurationSeconds > 0 && row.ImageCount > 0 {
			totalSeconds += row.DurationSeconds
			totalImages += row.ImageCount
		}
	}
	if totalImages == 0 {
		return fallback
	}
	return math.Max(1, totalSeconds/float64(totalImages))
}

func (s *QueueService) listTasks(queue, status string, start, end int) ([]*asynq.TaskInfo, error) {
	if end < start {
		end = start
	}
	start = maxInt(0, start)
	size := min(maxInt(1, end-start+1), 100)
	if normalizeQueueStatus(status) == "all" {
		tasks := []*asynq.TaskInfo{}
		for _, taskStatus := range []string{"waiting", "active", "delayed", "retry", "failed", "completed"} {
			items, err := s.listTasks(queue, taskStatus, 0, end)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, items...)
		}
		sort.SliceStable(tasks, func(i, j int) bool {
			return taskSortTime(tasks[i]).After(taskSortTime(tasks[j]))
		})
		if start >= len(tasks) {
			return []*asynq.TaskInfo{}, nil
		}
		limitEnd := min(start+size, len(tasks))
		return tasks[start:limitEnd], nil
	}
	opts := []asynq.ListOption{asynq.Page(1), asynq.PageSize(start + size)}
	sliceTasks := func(tasks []*asynq.TaskInfo, err error) ([]*asynq.TaskInfo, error) {
		if err != nil {
			return nil, err
		}
		if start >= len(tasks) {
			return []*asynq.TaskInfo{}, nil
		}
		limitEnd := min(start+size, len(tasks))
		return tasks[start:limitEnd], nil
	}
	switch normalizeQueueStatus(status) {
	case "active":
		return sliceTasks(s.inspector.ListActiveTasks(queue, opts...))
	case "completed":
		return sliceTasks(s.inspector.ListCompletedTasks(queue, opts...))
	case "failed":
		return sliceTasks(s.inspector.ListArchivedTasks(queue, opts...))
	case "delayed":
		return sliceTasks(s.inspector.ListScheduledTasks(queue, opts...))
	case "retry":
		return sliceTasks(s.inspector.ListRetryTasks(queue, opts...))
	default:
		return sliceTasks(s.inspector.ListPendingTasks(queue, opts...))
	}
}
