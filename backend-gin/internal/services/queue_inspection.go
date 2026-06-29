package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"yuem-go/backend-gin/internal/config"
)

func (s *QueueService) Stats() (bool, []map[string]any, map[string]map[string]any) {
	if !s.Available() {
		names := QueueNames
		cfg := config.Config{}
		if s != nil {
			names = s.Names()
			cfg = s.cfg
		}
		stats := make([]map[string]any, 0, len(names))
		for _, name := range names {
			stats = append(stats, queueEmptyStatsForConfig(cfg, name))
		}
		return false, stats, queueStatsByName(stats)
	}
	names := s.Names()
	events := s.RecentEvents(context.Background(), "", 100)
	eventCounts := queueEventCounts(events)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		info, err := s.inspector.GetQueueInfo(name)
		if err != nil {
			out = append(out, queueEmptyStats(name))
			continue
		}
		out = append(out, map[string]any{
			"name":           name,
			"label":          queueLabel(name),
			"kind":           queueKind(name),
			"queueType":      queueKind(name),
			"taskTypes":      queueTaskTypes(name),
			"workerEnabled":  queueWorkerEnabled(s.cfg, name),
			"concurrency":    queueConcurrency(s.cfg, name),
			"waiting":        info.Pending,
			"pending":        info.Pending,
			"active":         info.Active,
			"completed":      info.Completed,
			"failed":         info.Archived,
			"failedToday":    info.Failed,
			"failedTotal":    info.FailedTotal,
			"delayed":        info.Scheduled + info.Retry,
			"scheduled":      info.Scheduled,
			"retry":          info.Retry,
			"archived":       info.Archived,
			"aggregating":    info.Aggregating,
			"total":          info.Size,
			"size":           info.Size,
			"queueSize":      info.Size,
			"paused":         info.Paused,
			"processed":      info.ProcessedTotal,
			"processedToday": info.Processed,
			"memoryUsage":    info.MemoryUsage,
			"latencyMs":      info.Latency.Milliseconds(),
			"snapshotAt":     timeMillis(info.Timestamp),
			"recentEvents":   eventCounts[name],
		})
	}
	return true, out, queueStatsByName(out)
}

func EmptyQueueStats() []map[string]any {
	out := make([]map[string]any, 0, len(QueueNames))
	for _, name := range QueueNames {
		out = append(out, queueEmptyStats(name))
	}
	return out
}

func (s *QueueService) Jobs(queue, status string, start, end int) (bool, []map[string]any, error) {
	if !s.Available() {
		return false, []map[string]any{}, nil
	}
	tasks, err := s.listTasks(queue, status, start, end)
	if err != nil {
		return true, []map[string]any{}, err
	}
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		item := taskInfoMap(task, status)
		s.attachQueueEvents(context.Background(), item)
		out = append(out, item)
	}
	return true, out, nil
}

func (s *QueueService) Job(queue, id string) (bool, map[string]any, error) {
	if !s.Available() {
		return false, nil, nil
	}
	task, err := s.inspector.GetTaskInfo(queue, id)
	if err != nil {
		return true, nil, err
	}
	item := taskInfoMap(task, "")
	s.attachQueueEvents(context.Background(), item)
	return true, item, nil
}

func (s *QueueService) Retry(queue, id string) error {
	if !s.Available() {
		return errors.New("queue service disabled")
	}
	return s.inspector.RunTask(queue, id)
}

func (s *QueueService) Clean(queue string) error {
	if !s.Available() {
		return errors.New("queue service disabled")
	}
	return s.inspector.DeleteQueue(queue, true)
}

func (s *QueueService) RecentEvents(ctx context.Context, queue string, limit int64) []map[string]any {
	if s == nil || s.redis == nil {
		return []map[string]any{}
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.redis.ZRevRange(ctx, queueEventZSetKey, 0, limit-1).Result()
	if err != nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(rows))
	for _, raw := range rows {
		var event queueEvent
		if json.Unmarshal([]byte(raw), &event) != nil {
			continue
		}
		if queue != "" && event.Queue != queue {
			continue
		}
		out = append(out, queueEventMap(event))
	}
	return out
}

func (s *QueueService) recordQueueEvent(ctx context.Context, event queueEvent) {
	if s == nil || s.redis == nil {
		return
	}
	if event.At <= 0 {
		event.At = time.Now().UnixMilli()
	}
	if event.TaskID == "" && event.Detail != nil {
		event.TaskID, _ = event.Detail["id"].(string)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	pipe := s.redis.Pipeline()
	pipe.ZAdd(ctx, queueEventZSetKey, redis.Z{Score: float64(event.At), Member: string(data)})
	trimRedisZSet(ctx, pipe, queueEventZSetKey, s.queueEventRetention(), s.queueEventMaxEntries(), time.Now())
	_, _ = pipe.Exec(ctx)
}

func (s *QueueService) queueEventRetention() time.Duration {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return time.Duration(ReadRedisMaintenanceConfig(s.settings).QueueEventRetentionHours) * time.Hour
	}
	retention := s.cfg.Observe.MetricsRetention
	if retention <= 0 {
		return 24 * time.Hour
	}
	return retention
}

func (s *QueueService) queueEventMaxEntries() int64 {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return int64(ReadRedisMaintenanceConfig(s.settings).QueueEventMaxEntries)
	}
	return defaultRedisQueueEventMaxEntries
}

func (s *QueueService) queueEventMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		taskID, _ := asynq.GetTaskID(ctx)
		queue, _ := asynq.GetQueueName(ctx)
		if queue == "" {
			queue = queueNameFromTask(task)
		}
		startedAt := time.Now()
		queuedAt := taskEnqueuedTimeFromTask(task)
		startEvent := queueEvent{
			TaskID:    taskID,
			Queue:     queue,
			Type:      task.Type(),
			Event:     "started",
			State:     "active",
			At:        startedAt.UnixMilli(),
			LatencyMS: durationMillisInt(queuedAt, startedAt),
		}
		s.recordQueueEvent(ctx, startEvent)
		err := next.ProcessTask(ctx, task)
		completedAt := time.Now()
		event := queueEvent{
			TaskID:     taskID,
			Queue:      queue,
			Type:       task.Type(),
			Event:      "completed",
			State:      "completed",
			At:         completedAt.UnixMilli(),
			LatencyMS:  startEvent.LatencyMS,
			DurationMS: completedAt.Sub(startedAt).Milliseconds(),
		}
		if err != nil {
			event.Event = "failed"
			event.State = "failed"
			event.Error = truncateText(err.Error(), 300)
		}
		s.recordQueueEvent(ctx, event)
		return err
	})
}
