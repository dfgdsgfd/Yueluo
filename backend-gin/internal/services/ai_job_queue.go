package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

func (s *QueueService) EnqueueAIJobRun(ctx context.Context, jobID string) (map[string]any, error) {
	if s == nil || s.ai == nil {
		return nil, errors.New("ai service unavailable")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, errors.New("job id is required")
	}
	payload := aiJobRunTaskPayload{JobID: jobID, EnqueuedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	taskID := fmt.Sprintf("%s:%s", TaskAIJobRun, jobID)
	cfg := s.ai.Config()
	options := []asynq.Option{
		asynq.Queue(QueueAITask),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(time.Duration(aiJobTaskTimeoutSeconds(cfg)) * time.Second),
	}
	if retention := s.completedRetention(QueueAITask, 24*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskAIJobRun, data, QueueAITask, payload.EnqueuedAt), options...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			position, eta, queueCount := s.AIJobRunQueueEstimate(ctx, jobID)
			return map[string]any{"id": taskID, "jobId": jobID, "queue": QueueAITask, "duplicate": true, "queuePosition": position, "estimatedWaitSeconds": eta, "queueCount": queueCount, "enqueuedAt": payload.EnqueuedAt}, nil
		}
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskAIJobRun,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"jobId": payload.JobID},
	})
	position, eta, queueCount := s.AIJobRunQueueEstimate(ctx, jobID)
	return map[string]any{"id": info.ID, "jobId": jobID, "queue": info.Queue, "state": info.State.String(), "queuePosition": position, "estimatedWaitSeconds": eta, "queueCount": queueCount, "enqueuedAt": payload.EnqueuedAt}, nil
}

func aiJobTaskTimeoutSeconds(cfg AIConfig) int {
	if cfg.MaxRunSeconds > 0 {
		return maxInt(30, cfg.MaxRunSeconds+60)
	}
	return maxInt(30, cfg.TimeoutSeconds*maxInt(1, cfg.Concurrency)+60)
}

func (s *QueueService) AIJobRunQueueEstimate(ctx context.Context, jobID string) (int, int, int) {
	if !s.Available() || strings.TrimSpace(jobID) == "" {
		return 0, 0, 0
	}
	active, _ := s.listTasks(QueueAITask, "active", 0, 100)
	waiting, _ := s.listTasks(QueueAITask, "waiting", 0, 100)
	retrying, _ := s.listTasks(QueueAITask, "retry", 0, 100)
	delayed, _ := s.listTasks(QueueAITask, "delayed", 0, 100)
	tasks := append(append(append([]*asynq.TaskInfo{}, active...), waiting...), retrying...)
	tasks = append(tasks, delayed...)
	queueCount := len(tasks)
	concurrency := maxInt(1, s.cfg.Queue.Concurrency.AITask)
	for index, task := range active {
		if taskAIJobRunID(task) == jobID {
			return index + 1, 0, queueCount
		}
	}
	position := len(active)
	for _, group := range [][]*asynq.TaskInfo{waiting, retrying, delayed} {
		for _, task := range group {
			position++
			if taskAIJobRunID(task) == jobID {
				eta := int(math.Ceil(float64(maxInt(0, position-concurrency)) * aiJobRunEstimatedSeconds(s)))
				return position, eta, queueCount
			}
		}
	}
	return 0, 0, queueCount
}

func (s *QueueService) AbandonAIJobRun(ctx context.Context, jobID string) map[string]any {
	result := map[string]any{"removed": false}
	if s == nil || !s.Available() {
		result["available"] = false
		return result
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return result
	}
	taskID := fmt.Sprintf("%s:%s", TaskAIJobRun, jobID)
	result["id"] = taskID
	result["jobId"] = jobID
	result["queue"] = QueueAITask
	for _, status := range []string{"waiting", "retry", "delayed"} {
		tasks, err := s.listTasks(QueueAITask, status, 0, 100)
		if err != nil {
			result["error"] = err.Error()
			continue
		}
		for _, task := range tasks {
			if task == nil || task.ID != taskID {
				continue
			}
			if err := s.inspector.DeleteTask(QueueAITask, task.ID); err != nil {
				result["error"] = err.Error()
				return result
			}
			result["removed"] = true
			result["state"] = status
			return result
		}
		select {
		case <-ctx.Done():
			result["error"] = ctx.Err().Error()
			return result
		default:
		}
	}
	return result
}

func taskAIJobRunID(task *asynq.TaskInfo) string {
	if task == nil || task.Type != TaskAIJobRun {
		return ""
	}
	payload := aiJobRunTaskPayload{}
	if json.Unmarshal(task.Payload, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.JobID)
}

func aiJobRunEstimatedSeconds(s *QueueService) float64 {
	const fallback = 20.0
	if s == nil || s.ai == nil {
		return fallback
	}
	cfg := s.ai.Config()
	speed := defaultAITokensPerSecond
	if s.ai.gate != nil {
		s.ai.gate.mu.Lock()
		if s.ai.gate.tokensPerSecond > 0 {
			speed = s.ai.gate.tokensPerSecond
		}
		s.ai.gate.mu.Unlock()
	}
	if speed <= 0 {
		speed = defaultAITokensPerSecond
	}
	return math.Max(1, float64(maxInt(1, cfg.MaxOutputTokens))/speed)
}

func (s *QueueService) processAIJobRun(ctx context.Context, task *asynq.Task) error {
	var payload aiJobRunTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if s == nil || s.ai == nil {
		return fmt.Errorf("%w: ai service unavailable", asynq.SkipRetry)
	}
	if err := s.ai.RunJobByID(ctx, payload.JobID); err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{"jobId": payload.JobID})
	_, _ = task.ResultWriter().Write(result)
	return nil
}
