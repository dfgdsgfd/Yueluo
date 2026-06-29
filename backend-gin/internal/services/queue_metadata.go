package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"yuem-go/backend-gin/internal/config"
)

func durationMillis(start, end time.Time) any {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return nil
	}
	return end.Sub(start).Milliseconds()
}

func durationMillisInt(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func durationSeconds(start, end time.Time) any {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return nil
	}
	return end.Sub(start).Seconds()
}

func taskSortTime(task *asynq.TaskInfo) time.Time {
	for _, value := range []time.Time{taskEnqueuedTime(task), task.CompletedAt, task.LastFailedAt, task.NextProcessAt} {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func queueEmptyStats(name string) map[string]any {
	return map[string]any{
		"name":          name,
		"label":         queueLabel(name),
		"kind":          queueKind(name),
		"queueType":     queueKind(name),
		"taskTypes":     queueTaskTypes(name),
		"workerEnabled": false,
		"concurrency":   0,
		"waiting":       0,
		"pending":       0,
		"active":        0,
		"completed":     0,
		"failed":        0,
		"delayed":       0,
		"scheduled":     0,
		"retry":         0,
		"archived":      0,
		"aggregating":   0,
		"total":         0,
		"size":          0,
		"queueSize":     0,
	}
}

func queueEmptyStatsForConfig(cfg config.Config, name string) map[string]any {
	stat := queueEmptyStats(name)
	stat["workerEnabled"] = queueWorkerEnabled(cfg, name)
	stat["concurrency"] = queueConcurrency(cfg, name)
	return stat
}

func normalizeQueueStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "all", "全部":
		return "all"
	case "active":
		return "active"
	case "completed":
		return "completed"
	case "failed", "archived":
		return "failed"
	case "delayed", "scheduled":
		return "delayed"
	case "retry":
		return "retry"
	default:
		return "waiting"
	}
}

func queueStatusLabel(status string) string {
	switch normalizeQueueStatus(status) {
	case "active":
		return "active"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "delayed":
		return "delayed"
	case "retry":
		return "retry"
	case "all":
		return "all"
	default:
		return "waiting"
	}
}

func retryDelayFunc(base time.Duration) asynq.RetryDelayFunc {
	if base <= 0 {
		base = time.Second
	}
	return func(n int, _ error, _ *asynq.Task) time.Duration {
		if n < 0 {
			n = 0
		}
		if n > 6 {
			n = 6
		}
		return base * time.Duration(1<<n)
	}
}

func jsonAnyBytes(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if json.Unmarshal(raw, &out) == nil {
		return out
	}
	return string(raw)
}

func newQueueTask(taskType string, data []byte, queue string, enqueuedAt int64) *asynq.Task {
	if enqueuedAt <= 0 {
		enqueuedAt = time.Now().UnixMilli()
	}
	return asynq.NewTaskWithHeaders(taskType, data, map[string]string{
		taskHeaderEnqueuedAt: strconv.FormatInt(enqueuedAt, 10),
		taskHeaderQueueKind:  queueKind(queue),
	})
}

func taskEnqueuedTime(task *asynq.TaskInfo) time.Time {
	if task == nil {
		return time.Time{}
	}
	if value := millisFromHeaders(task.Headers); value > 0 {
		return time.UnixMilli(value)
	}
	if value := millisFromPayload(task.Payload); value > 0 {
		return time.UnixMilli(value)
	}
	for _, value := range []time.Time{task.NextProcessAt, task.CompletedAt, task.LastFailedAt} {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func taskEnqueuedTimeFromTask(task *asynq.Task) time.Time {
	if task == nil {
		return time.Time{}
	}
	if value := millisFromHeaders(task.Headers()); value > 0 {
		return time.UnixMilli(value)
	}
	if value := millisFromPayload(task.Payload()); value > 0 {
		return time.UnixMilli(value)
	}
	return time.Time{}
}

func millisFromHeaders(headers map[string]string) int64 {
	for _, key := range []string{taskHeaderEnqueuedAt, "enqueued_at", "enqueuedAt"} {
		if raw := strings.TrimSpace(headers[key]); raw != "" {
			if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
				return normalizeUnixMillis(value)
			}
			if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
				return parsed.UnixMilli()
			}
		}
	}
	return 0
}

func millisFromPayload(raw []byte) int64 {
	if len(raw) == 0 {
		return 0
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		return 0
	}
	for _, key := range []string{"enqueued_at", "enqueuedAt", "queued_at", "queuedAt"} {
		switch value := payload[key].(type) {
		case float64:
			return normalizeUnixMillis(int64(value))
		case int64:
			return normalizeUnixMillis(value)
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
				return normalizeUnixMillis(parsed)
			}
			if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value)); err == nil {
				return parsed.UnixMilli()
			}
		}
	}
	return 0
}

func normalizeUnixMillis(value int64) int64 {
	if value <= 0 {
		return 0
	}
	if value < 1_000_000_000_000 {
		return value * 1000
	}
	return value
}

func queueStatsByName(stats []map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(stats))
	for _, stat := range stats {
		name, _ := stat["name"].(string)
		if name != "" {
			out[name] = stat
		}
	}
	return out
}

func QueueStatsByNameForResponse(stats []map[string]any) map[string]map[string]any {
	return queueStatsByName(stats)
}

func queueEventMap(event queueEvent) map[string]any {
	out := map[string]any{
		"task_id":    event.TaskID,
		"id":         event.TaskID,
		"queue":      event.Queue,
		"type":       event.Type,
		"taskType":   event.Type,
		"event":      event.Event,
		"state":      event.State,
		"at":         event.At,
		"createdAt":  event.At,
		"latencyMs":  event.LatencyMS,
		"durationMs": event.DurationMS,
		"error":      emptyStringToNil(event.Error),
		"detail":     event.Detail,
	}
	return out
}

func queueEventCounts(events []map[string]any) map[string]map[string]any {
	counts := map[string]map[string]any{}
	for _, event := range events {
		queue := fmt.Sprint(event["queue"])
		if queue == "" {
			continue
		}
		stat := counts[queue]
		if stat == nil {
			stat = map[string]any{"total": 0, "lastAt": int64(0)}
			counts[queue] = stat
		}
		stat["total"] = intFromAnyDefault(stat["total"], 0) + 1
		key := fmt.Sprint(event["event"])
		if key != "" {
			stat[key] = intFromAnyDefault(stat[key], 0) + 1
		}
		at := int64FromAny(event["at"])
		if at > int64FromAny(stat["lastAt"]) {
			stat["lastAt"] = at
		}
	}
	return counts
}

func queueNameFromTask(task *asynq.Task) string {
	if task == nil {
		return ""
	}
	switch task.Type() {
	case TaskBatchNoteCreate:
		return QueueBatchNoteCreate
	case TaskVideoTranscoding:
		return QueueVideoTranscoding
	case TaskImageProtection:
		return QueueImageProtection
	case TaskAccessLogBatch, TaskSecurityAuditLogBatch:
		return QueueAuditLog
	case TaskAIPostAutoComment, TaskAICommentReply, TaskAIJobRun, TaskAIModerateContent:
		return QueueAITask
	default:
		return ""
	}
}

func queueMetadata(cfg config.Config, name string) queueMeta {
	switch name {
	case QueueBatchNoteCreate:
		return queueMeta{Kind: "批量笔记发布", TaskTypes: []string{TaskBatchNoteCreate}, Worker: true, Concurrency: cfg.Queue.Concurrency.BatchNoteCreate}
	case QueueVideoTranscoding:
		return queueMeta{Kind: "视频转码", TaskTypes: []string{TaskVideoTranscoding}, Worker: true, Concurrency: cfg.Queue.Concurrency.VideoTranscoding}
	case QueueImageProtection:
		return queueMeta{Kind: "图片压缩包", TaskTypes: []string{TaskImageProtection, TaskPostImageArchive}, Worker: true, Concurrency: cfg.Queue.Concurrency.ImageProtection}
	case QueueIPLocation:
		return queueMeta{Kind: "IP 位置刷新", TaskTypes: []string{}, Worker: false, Concurrency: cfg.Queue.Concurrency.IPLocation}
	case QueueContentAudit:
		return queueMeta{Kind: "内容审核", TaskTypes: []string{}, Worker: false, Concurrency: cfg.Queue.Concurrency.ContentAudit}
	case QueueAuditLog:
		return queueMeta{Kind: "审计日志", TaskTypes: []string{TaskAccessLogBatch, TaskSecurityAuditLogBatch}, Worker: true, Concurrency: cfg.Queue.Concurrency.AuditLog}
	case QueueGeneralTask:
		return queueMeta{Kind: "通用任务", TaskTypes: []string{}, Worker: true, Concurrency: cfg.Queue.Concurrency.GeneralTask}
	case QueueAITask:
		return queueMeta{Kind: "AI 任务", TaskTypes: []string{TaskAIPostAutoComment, TaskAICommentReply, TaskAIJobRun, TaskAIModerateContent}, Worker: true, Concurrency: cfg.Queue.Concurrency.AITask}
	case QueueBrowsingHistory:
		return queueMeta{Kind: "浏览历史", TaskTypes: []string{}, Worker: false, Concurrency: 0}
	default:
		return queueMeta{Kind: "Asynq 队列", TaskTypes: []string{}, Worker: false, Concurrency: 0}
	}
}

func queueKind(name string) string {
	return queueMetadata(config.Config{}, name).Kind
}

func queueLabel(name string) string {
	kind := queueKind(name)
	if kind == "Asynq 队列" {
		return name
	}
	return kind
}

func queueTaskTypes(name string) []string {
	return queueMetadata(config.Config{}, name).TaskTypes
}

func queueWorkerEnabled(cfg config.Config, name string) bool {
	meta := queueMetadata(cfg, name)
	return cfg.Queue.Enabled && meta.Worker
}

func queueConcurrency(cfg config.Config, name string) int {
	return queueMetadata(cfg, name).Concurrency
}

func emptyStringToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTimeMillis(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return timeMillis(value)
}

func timeMillis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano() / int64(time.Millisecond)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intFromAnyDefault(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return fallback
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	}
	return 0
}
