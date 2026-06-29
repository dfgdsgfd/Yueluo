package services

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	observabilityWriterQueueSize = 2048
	observabilityWriterBatchSize = 64
	observabilityWriterInterval  = 200 * time.Millisecond
	observabilityWriterTimeout   = 2 * time.Second
)

type observabilityWriter struct {
	service        *ObservabilityService
	systemLogs     chan SystemLogEvent
	accessLogs     chan RecentAccessLogEvent
	requestMetrics chan RequestMetric
	slowQueries    chan SlowQueryMetric
	done           chan struct{}
	closeOnce      sync.Once
}

func newObservabilityWriter(service *ObservabilityService) *observabilityWriter {
	return &observabilityWriter{
		service:        service,
		systemLogs:     make(chan SystemLogEvent, observabilityWriterQueueSize),
		accessLogs:     make(chan RecentAccessLogEvent, observabilityWriterQueueSize),
		requestMetrics: make(chan RequestMetric, observabilityWriterQueueSize),
		slowQueries:    make(chan SlowQueryMetric, observabilityWriterQueueSize),
		done:           make(chan struct{}),
	}
}

func (w *observabilityWriter) start() {
	if w == nil {
		return
	}
	go w.run()
}

func (w *observabilityWriter) close() {
	if w == nil {
		return
	}
	w.closeOnce.Do(func() {
		close(w.done)
	})
}

func (w *observabilityWriter) enqueueSystemLog(event SystemLogEvent) bool {
	if w == nil {
		return false
	}
	select {
	case w.systemLogs <- event:
		return true
	default:
		return false
	}
}

func (w *observabilityWriter) enqueueAccessLog(event RecentAccessLogEvent) bool {
	if w == nil {
		return false
	}
	select {
	case w.accessLogs <- event:
		return true
	default:
		return false
	}
}

func (w *observabilityWriter) enqueueRequestMetric(metric RequestMetric) bool {
	if w == nil {
		return false
	}
	select {
	case w.requestMetrics <- metric:
		return true
	default:
		return false
	}
}

func (w *observabilityWriter) enqueueSlowQuery(metric SlowQueryMetric) bool {
	if w == nil {
		return false
	}
	select {
	case w.slowQueries <- metric:
		return true
	default:
		return false
	}
}

func (w *observabilityWriter) run() {
	ticker := time.NewTicker(observabilityWriterInterval)
	defer ticker.Stop()
	systemLogs := make([]SystemLogEvent, 0, observabilityWriterBatchSize)
	accessLogs := make([]RecentAccessLogEvent, 0, observabilityWriterBatchSize)
	requestMetrics := make([]RequestMetric, 0, observabilityWriterBatchSize)
	slowQueries := make([]SlowQueryMetric, 0, observabilityWriterBatchSize)

	for {
		select {
		case <-w.done:
			w.drainAvailable(&systemLogs, &accessLogs, &requestMetrics, &slowQueries)
			w.flush(&systemLogs, &accessLogs, &requestMetrics, &slowQueries)
			return
		case event := <-w.systemLogs:
			systemLogs = append(systemLogs, event)
			if len(systemLogs) >= observabilityWriterBatchSize {
				w.flushSystemLogs(&systemLogs)
			}
		case event := <-w.accessLogs:
			accessLogs = append(accessLogs, event)
			if len(accessLogs) >= observabilityWriterBatchSize {
				w.flushAccessLogs(&accessLogs)
			}
		case metric := <-w.requestMetrics:
			requestMetrics = append(requestMetrics, metric)
			if len(requestMetrics) >= observabilityWriterBatchSize {
				w.flushRequestMetrics(&requestMetrics)
			}
		case metric := <-w.slowQueries:
			slowQueries = append(slowQueries, metric)
			if len(slowQueries) >= observabilityWriterBatchSize {
				w.flushSlowQueries(&slowQueries)
			}
		case <-ticker.C:
			w.flush(&systemLogs, &accessLogs, &requestMetrics, &slowQueries)
		}
	}
}

func (w *observabilityWriter) drainAvailable(systemLogs *[]SystemLogEvent, accessLogs *[]RecentAccessLogEvent, requestMetrics *[]RequestMetric, slowQueries *[]SlowQueryMetric) {
	for {
		select {
		case event := <-w.systemLogs:
			*systemLogs = append(*systemLogs, event)
		case event := <-w.accessLogs:
			*accessLogs = append(*accessLogs, event)
		case metric := <-w.requestMetrics:
			*requestMetrics = append(*requestMetrics, metric)
		case metric := <-w.slowQueries:
			*slowQueries = append(*slowQueries, metric)
		default:
			return
		}
	}
}

func (w *observabilityWriter) flush(systemLogs *[]SystemLogEvent, accessLogs *[]RecentAccessLogEvent, requestMetrics *[]RequestMetric, slowQueries *[]SlowQueryMetric) {
	w.flushSystemLogs(systemLogs)
	w.flushAccessLogs(accessLogs)
	w.flushRequestMetrics(requestMetrics)
	w.flushSlowQueries(slowQueries)
}

func (w *observabilityWriter) flushSystemLogs(events *[]SystemLogEvent) {
	if len(*events) == 0 {
		return
	}
	if w == nil || w.service == nil || w.service.redis == nil || w.service.redis.Client() == nil {
		*events = (*events)[:0]
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), observabilityWriterTimeout)
	defer cancel()
	client := w.service.redis.Client()
	pipe := client.Pipeline()
	now := time.Now()
	for _, event := range *events {
		values := map[string]any{
			"type":       event.Type,
			"level":      event.Level,
			"message":    event.Message,
			"actor_id":   event.ActorID,
			"actor_type": event.ActorType,
			"method":     event.Method,
			"path":       event.Path,
			"status":     event.Status,
			"latency_ms": event.LatencyMS,
			"ip":         event.IP,
			"user_agent": event.UserAgent,
			"request_id": event.RequestID,
			"created_at": event.CreatedAt.Format(time.RFC3339Nano),
		}
		if len(event.Detail) > 0 {
			if data, err := json.Marshal(event.Detail); err == nil {
				values["detail"] = string(data)
			}
		}
		pipe.XAdd(ctx, &redis.XAddArgs{Stream: systemLogStreamKey, Values: values})
	}
	trimRedisStream(ctx, pipe, systemLogStreamKey, w.service.SystemLogRetention(), w.service.SystemLogMaxEntries(), now)
	_, _ = pipe.Exec(ctx)
	*events = (*events)[:0]
}

func (w *observabilityWriter) flushAccessLogs(events *[]RecentAccessLogEvent) {
	if len(*events) == 0 {
		return
	}
	if w == nil || w.service == nil || w.service.redis == nil || w.service.redis.Client() == nil {
		*events = (*events)[:0]
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), observabilityWriterTimeout)
	defer cancel()
	client := w.service.redis.Client()
	pipe := client.Pipeline()
	now := time.Now()
	for _, event := range *events {
		pipe.XAdd(ctx, &redis.XAddArgs{Stream: accessLogStreamKey, Values: map[string]any{
			"method":     event.Method,
			"path":       event.Path,
			"status":     event.Status,
			"latency_ms": event.LatencyMS,
			"ip":         event.IP,
			"user_agent": event.UserAgent,
			"request_id": event.RequestID,
			"user_id":    event.UserID,
			"username":   event.Username,
			"nickname":   event.Nickname,
			"user_type":  event.UserType,
			"created_at": event.CreatedAt.Format(time.RFC3339Nano),
		}})
	}
	trimRedisStream(ctx, pipe, accessLogStreamKey, w.service.AccessLogRetention(), w.service.AccessLogMaxEntries(), now)
	_, _ = pipe.Exec(ctx)
	*events = (*events)[:0]
}

func (w *observabilityWriter) flushRequestMetrics(metrics *[]RequestMetric) {
	if len(*metrics) == 0 {
		return
	}
	if w == nil || w.service == nil || w.service.redis == nil || w.service.redis.Client() == nil {
		*metrics = (*metrics)[:0]
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), observabilityWriterTimeout)
	defer cancel()
	client := w.service.redis.Client()
	pipe := client.Pipeline()
	now := time.Now()
	for _, metric := range *metrics {
		data, err := json.Marshal(metric)
		if err != nil {
			continue
		}
		score := float64(metric.CreatedAt.UnixMilli())
		member := strconv.FormatInt(metric.CreatedAt.UnixNano(), 10) + ":" + metric.RequestID + ":" + string(data)
		pipe.ZAdd(ctx, requestMetricZSetKey, redis.Z{Score: score, Member: member})
		if w.service.cfg.SlowRequestThreshold > 0 && metric.LatencyMS >= w.service.cfg.SlowRequestThreshold.Milliseconds() {
			pipe.ZAdd(ctx, slowRequestZSetKey, redis.Z{Score: score, Member: member})
		}
	}
	retention := w.service.MetricsRetention()
	maxEntries := w.service.MetricsMaxEntriesPerKey()
	trimRedisZSet(ctx, pipe, requestMetricZSetKey, retention, maxEntries, now)
	trimRedisZSet(ctx, pipe, slowRequestZSetKey, retention, maxEntries, now)
	_, _ = pipe.Exec(ctx)
	*metrics = (*metrics)[:0]
}

func (w *observabilityWriter) flushSlowQueries(metrics *[]SlowQueryMetric) {
	if len(*metrics) == 0 {
		return
	}
	if w == nil || w.service == nil || w.service.redis == nil || w.service.redis.Client() == nil {
		*metrics = (*metrics)[:0]
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), observabilityWriterTimeout)
	defer cancel()
	client := w.service.redis.Client()
	pipe := client.Pipeline()
	now := time.Now()
	for _, metric := range *metrics {
		data, err := json.Marshal(metric)
		if err != nil {
			continue
		}
		score := float64(metric.CreatedAt.UnixMilli())
		member := strconv.FormatInt(metric.CreatedAt.UnixNano(), 10) + ":" + metric.Fingerprint + ":" + string(data)
		pipe.ZAdd(ctx, slowQueryZSetKey, redis.Z{Score: score, Member: member})
	}
	trimRedisZSet(ctx, pipe, slowQueryZSetKey, w.service.MetricsRetention(), w.service.MetricsMaxEntriesPerKey(), now)
	_, _ = pipe.Exec(ctx)
	*metrics = (*metrics)[:0]
}
