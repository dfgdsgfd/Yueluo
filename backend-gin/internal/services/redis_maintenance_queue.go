package services

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/hibiken/asynq"
)

func (s *QueueService) CleanupHistory(ctx context.Context, cfg RedisMaintenanceConfig) (map[string]QueueCleanupResult, error) {
	results := map[string]QueueCleanupResult{}
	if s == nil || !s.Available() {
		return results, errors.New("queue service is unavailable")
	}
	var errs error
	for _, queue := range s.Names() {
		policy := cfg.QueuePolicy(queue)
		info, err := s.inspector.GetQueueInfo(queue)
		if err != nil {
			if errors.Is(err, asynq.ErrQueueNotFound) {
				continue
			}
			errs = errors.Join(errs, err)
			continue
		}
		item := QueueCleanupResult{
			Completed: info.Completed, Archived: info.Archived,
			CompletedRetention: policy.CompletedRetentionHours,
			ArchivedRetention:  policy.ArchivedRetentionHours,
			ArchivedMax:        policy.ArchivedMaxTasks,
		}
		if policy.CompletedRetentionHours == 0 && info.Completed > 0 {
			item.CompletedDeleted, err = s.inspector.DeleteAllCompletedTasks(queue)
			errs = errors.Join(errs, err)
		} else if info.Completed > 0 {
			item.CompletedDeleted, err = s.cleanupCompletedTasks(ctx, queue, info.Completed, policy.CompletedRetentionHours)
			errs = errors.Join(errs, err)
		}
		deleted, cleanupErr := s.cleanupArchivedTasks(ctx, queue, policy)
		item.ArchivedDeleted = deleted
		errs = errors.Join(errs, cleanupErr)
		results[queue] = item
	}
	return results, errs
}

func (s *QueueService) cleanupCompletedTasks(ctx context.Context, queue string, total, retentionHours int) (int, error) {
	const (
		pageSize       = 100
		maxPagesPerRun = 500
	)
	cutoff := time.Now().Add(-time.Duration(retentionHours) * time.Hour)
	lastPage := (total + pageSize - 1) / pageSize
	firstPage := max(1, lastPage-maxPagesPerRun+1)
	deleted := 0
	var errs error
	for page := lastPage; page >= firstPage; page-- {
		tasks, err := s.inspector.ListCompletedTasks(queue, asynq.Page(page), asynq.PageSize(pageSize))
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, task := range tasks {
			if !completedTaskExpired(task, cutoff) {
				continue
			}
			if err := s.inspector.DeleteTask(queue, task.ID); err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			deleted++
			select {
			case <-ctx.Done():
				return deleted, errors.Join(errs, ctx.Err())
			default:
			}
		}
	}
	return deleted, errs
}

func completedTaskExpired(task *asynq.TaskInfo, cutoff time.Time) bool {
	return task != nil && !task.CompletedAt.IsZero() && task.CompletedAt.Before(cutoff)
}

func (s *QueueService) cleanupArchivedTasks(ctx context.Context, queue string, policy RedisQueueRetentionPolicy) (int, error) {
	var tasks []*asynq.TaskInfo
	for page := 1; page <= 100; page++ {
		items, err := s.inspector.ListArchivedTasks(queue, asynq.Page(page), asynq.PageSize(100))
		if err != nil {
			return 0, err
		}
		tasks = append(tasks, items...)
		if len(items) < 100 {
			break
		}
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].LastFailedAt.After(tasks[j].LastFailedAt)
	})
	cutoff := time.Now().Add(-time.Duration(policy.ArchivedRetentionHours) * time.Hour)
	deleteIDs := map[string]struct{}{}
	for index, task := range tasks {
		if index >= policy.ArchivedMaxTasks || (!task.LastFailedAt.IsZero() && task.LastFailedAt.Before(cutoff)) {
			deleteIDs[task.ID] = struct{}{}
		}
	}
	deleted := 0
	var errs error
	for id := range deleteIDs {
		if err := s.inspector.DeleteTask(queue, id); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		deleted++
		select {
		case <-ctx.Done():
			return deleted, errors.Join(errs, ctx.Err())
		default:
		}
	}
	return deleted, errs
}

func (s *QueueService) completedRetention(queue string, fallback time.Duration) time.Duration {
	if s == nil || s.settings == nil {
		return fallback
	}
	hours := ReadRedisMaintenanceConfig(s.settings).QueuePolicy(queue).CompletedRetentionHours
	if hours <= 0 {
		return 0
	}
	return time.Duration(hours) * time.Hour
}

func (s *ObservabilityService) SystemLogRetention() time.Duration {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return time.Duration(ReadRedisMaintenanceConfig(s.settings).SystemLogRetentionHours) * time.Hour
	}
	return s.cfg.SystemLogRetention
}

func (s *ObservabilityService) AccessLogRetention() time.Duration {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return time.Duration(ReadRedisMaintenanceConfig(s.settings).AccessLogRetentionHours) * time.Hour
	}
	return s.cfg.SystemLogRetention
}

func (s *ObservabilityService) SystemLogMaxEntries() int64 {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return int64(ReadRedisMaintenanceConfig(s.settings).SystemLogMaxEntries)
	}
	return defaultRedisSystemLogMaxEntries
}

func (s *ObservabilityService) AccessLogMaxEntries() int64 {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return int64(ReadRedisMaintenanceConfig(s.settings).AccessLogMaxEntries)
	}
	return defaultRedisAccessLogMaxEntries
}

func (s *ObservabilityService) MetricsRetention() time.Duration {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return time.Duration(ReadRedisMaintenanceConfig(s.settings).MetricsRetentionHours) * time.Hour
	}
	return s.cfg.MetricsRetention
}

func (s *ObservabilityService) MetricsMaxEntriesPerKey() int64 {
	if s == nil {
		return 0
	}
	if s.settings != nil {
		return int64(ReadRedisMaintenanceConfig(s.settings).MetricsMaxEntriesPerKey)
	}
	return defaultRedisMetricsMaxEntries
}
