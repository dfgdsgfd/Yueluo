package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s *ObservabilityService) recordPostgresSnapshot(ctx context.Context, postgres ginLikeMap) {
	if s == nil || !s.cfg.MetricsEnabled || s.redis == nil || s.redis.Client() == nil {
		return
	}
	activity, _ := postgres["activity"].(ginLikeMap)
	locks, _ := postgres["locks"].(ginLikeMap)
	tableHealth, _ := postgres["table_health"].(ginLikeMap)
	wal, _ := postgres["wal"].(ginLikeMap)
	now := time.Now()
	snapshot := ginLikeMap{
		"ts":                  now.UnixMilli(),
		"open_connections":    postgres["open_connections"],
		"in_use":              postgres["in_use"],
		"idle":                postgres["idle"],
		"wait_count":          postgres["wait_count"],
		"wait_duration_ms":    postgres["wait_duration_ms"],
		"cache_hit_ratio":     postgres["cache_hit_ratio"],
		"xact_commit":         postgres["xact_commit"],
		"xact_rollback":       postgres["xact_rollback"],
		"active_connections":  activity["active_connections"],
		"idle_in_transaction": activity["idle_in_transaction"],
		"waiting_locks":       locks["waiting_locks"],
		"dead_tuples":         tableHealth["dead_tuples"],
		"dead_tuple_ratio":    tableHealth["dead_ratio"],
		"wal_bytes":           wal["wal_bytes"],
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	client := s.redis.Client()
	pipe := client.Pipeline()
	pipe.ZAdd(ctx, postgresMetricZSetKey, redis.Z{Score: float64(now.UnixMilli()), Member: string(data)})
	trimRedisZSet(ctx, pipe, postgresMetricZSetKey, s.MetricsRetention(), s.MetricsMaxEntriesPerKey(), now)
	_, _ = pipe.Exec(ctx)
}

func (s *ObservabilityService) recordRuntimeSnapshot(ctx context.Context, snapshot ginLikeMap) {
	if s == nil || !s.cfg.MetricsEnabled || s.redis == nil || s.redis.Client() == nil {
		return
	}
	now := time.Now()
	s.mu.Lock()
	if !s.lastRuntimeSample.IsZero() && now.Sub(s.lastRuntimeSample) < s.cfg.RuntimeSampleInterval {
		s.mu.Unlock()
		return
	}
	s.lastRuntimeSample = now
	s.mu.Unlock()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	client := s.redis.Client()
	pipe := client.Pipeline()
	pipe.ZAdd(ctx, runtimeMetricZSetKey, redis.Z{Score: float64(now.UnixMilli()), Member: string(data)})
	trimRedisZSet(ctx, pipe, runtimeMetricZSetKey, s.MetricsRetention(), s.MetricsMaxEntriesPerKey(), now)
	_, _ = pipe.Exec(ctx)
}

func (s *ObservabilityService) postgresSeries(ctx context.Context, options PerformanceOptions) []ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []ginLikeMap{}
	}
	now := time.Now()
	rows, err := s.redis.Client().ZRangeByScore(ctx, postgresMetricZSetKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10),
		Max: strconv.FormatInt(now.UnixMilli(), 10),
	}).Result()
	if err != nil {
		return []ginLikeMap{}
	}
	out := make([]ginLikeMap, 0, len(rows))
	for _, raw := range rows {
		var item ginLikeMap
		if json.Unmarshal([]byte(raw), &item) == nil {
			out = append(out, item)
		}
	}
	return out
}

func (s *ObservabilityService) runtimeSeries(ctx context.Context, options PerformanceOptions) []ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []ginLikeMap{}
	}
	now := time.Now()
	rows, err := s.redis.Client().ZRangeByScore(ctx, runtimeMetricZSetKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10),
		Max: strconv.FormatInt(now.UnixMilli(), 10),
	}).Result()
	if err != nil {
		return []ginLikeMap{}
	}
	out := make([]ginLikeMap, 0, len(rows))
	for _, raw := range rows {
		var item ginLikeMap
		if json.Unmarshal([]byte(raw), &item) == nil {
			out = append(out, item)
		}
	}
	return out
}

func requestMetricsFromRows(rows []string) []RequestMetric {
	metrics := make([]RequestMetric, 0, len(rows))
	for _, row := range rows {
		idx := len(row) - 1
		for idx >= 0 && row[idx] != '{' {
			idx--
		}
		if idx < 0 {
			continue
		}
		var metric RequestMetric
		if json.Unmarshal([]byte(row[idx:]), &metric) != nil {
			continue
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func slowQueryMetricsFromRows(rows []string) []SlowQueryMetric {
	metrics := make([]SlowQueryMetric, 0, len(rows))
	for _, row := range rows {
		idx := len(row) - 1
		for idx >= 0 && row[idx] != '{' {
			idx--
		}
		if idx < 0 {
			continue
		}
		var metric SlowQueryMetric
		if json.Unmarshal([]byte(row[idx:]), &metric) != nil {
			continue
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func (s *ObservabilityService) slowRequests(ctx context.Context, options PerformanceOptions) []ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []ginLikeMap{}
	}
	now := time.Now()
	rows, err := s.redis.Client().ZRevRangeByScore(ctx, slowRequestZSetKey, &redis.ZRangeBy{
		Min:    strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10),
		Max:    strconv.FormatInt(now.UnixMilli(), 10),
		Offset: 0,
		Count:  options.SlowLimit,
	}).Result()
	if err != nil {
		return []ginLikeMap{}
	}
	metrics := requestMetricsFromRows(rows)
	out := make([]ginLikeMap, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, ginLikeMap{
			"method":     metric.Method,
			"path":       metric.Path,
			"route":      emptyStringToNil(metric.Route),
			"status":     metric.Status,
			"latency_ms": metric.LatencyMS,
			"request_id": metric.RequestID,
			"created_at": metric.CreatedAt.Format(time.RFC3339Nano),
			"ts":         metric.CreatedAt.UnixMilli(),
		})
	}
	return out
}

func (s *ObservabilityService) slowQueries(ctx context.Context, options PerformanceOptions) []ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []ginLikeMap{}
	}
	now := time.Now()
	rows, err := s.redis.Client().ZRevRangeByScore(ctx, slowQueryZSetKey, &redis.ZRangeBy{
		Min:    strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10),
		Max:    strconv.FormatInt(now.UnixMilli(), 10),
		Offset: 0,
		Count:  options.SlowLimit,
	}).Result()
	if err != nil {
		return []ginLikeMap{}
	}
	metrics := slowQueryMetricsFromRows(rows)
	out := make([]ginLikeMap, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, ginLikeMap{
			"fingerprint": metric.Fingerprint,
			"sql":         metric.SQL,
			"latency_ms":  metric.LatencyMS,
			"rows":        metric.Rows,
			"error":       emptyStringToNil(metric.Error),
			"created_at":  metric.CreatedAt.Format(time.RFC3339Nano),
			"ts":          metric.CreatedAt.UnixMilli(),
		})
	}
	return out
}

func (s *ObservabilityService) slowQueryGroups(ctx context.Context, options PerformanceOptions, limit int) []ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []ginLikeMap{}
	}
	now := time.Now()
	rows, err := s.redis.Client().ZRevRangeByScore(ctx, slowQueryZSetKey, &redis.ZRangeBy{
		Min:    strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10),
		Max:    strconv.FormatInt(now.UnixMilli(), 10),
		Offset: 0,
		Count:  max(int64(limit*5), options.SlowLimit),
	}).Result()
	if err != nil {
		return []ginLikeMap{}
	}
	return slowQueryGroups(slowQueryMetricsFromRows(rows), limit)
}

func filterObservabilityRows(options ObservabilityEventOptions, rows []string) []ginLikeMap {
	switch options.Type {
	case "slow_queries":
		metrics := slowQueryMetricsFromRows(rows)
		out := make([]ginLikeMap, 0, len(metrics))
		for _, metric := range metrics {
			item := ginLikeMap{
				"id":          metric.Fingerprint + ":" + strconv.FormatInt(metric.CreatedAt.UnixMilli(), 10),
				"type":        "slow_query",
				"fingerprint": metric.Fingerprint,
				"sql":         metric.SQL,
				"latency_ms":  metric.LatencyMS,
				"rows":        metric.Rows,
				"error":       emptyStringToNil(metric.Error),
				"created_at":  metric.CreatedAt.Format(time.RFC3339Nano),
				"ts":          metric.CreatedAt.UnixMilli(),
			}
			if matchesObservabilityEvent(options, item) {
				out = append(out, item)
			}
		}
		return out
	default:
		metrics := requestMetricsFromRows(rows)
		out := make([]ginLikeMap, 0, len(metrics))
		for _, metric := range metrics {
			if options.Type == "errors" && metric.Status < 500 {
				continue
			}
			item := ginLikeMap{
				"id":         observabilityFirstNonEmpty(metric.RequestID, strconv.FormatInt(metric.CreatedAt.UnixNano(), 10)),
				"type":       observabilityRequestType(metric.Status >= 500),
				"method":     metric.Method,
				"path":       metric.Path,
				"route":      emptyStringToNil(metric.Route),
				"status":     metric.Status,
				"latency_ms": metric.LatencyMS,
				"request_id": metric.RequestID,
				"created_at": metric.CreatedAt.Format(time.RFC3339Nano),
				"ts":         metric.CreatedAt.UnixMilli(),
			}
			if matchesObservabilityEvent(options, item) {
				out = append(out, item)
			}
		}
		return out
	}
}

func matchesObservabilityEvent(options ObservabilityEventOptions, item ginLikeMap) bool {
	if options.Method != "" && strings.ToUpper(toStringValue(item["method"])) != options.Method {
		return false
	}
	if options.Status > 0 {
		status, _ := strconv.Atoi(toStringValue(item["status"]))
		if status != options.Status {
			return false
		}
	}
	if options.Keyword == "" {
		return true
	}
	for _, key := range []string{"method", "path", "route", "status", "request_id", "sql", "fingerprint", "error", "type"} {
		if strings.Contains(strings.ToLower(toStringValue(item[key])), options.Keyword) {
			return true
		}
	}
	return false
}

func eventPagination(page, limit int, total int64) ginLikeMap {
	pages := int64(0)
	if limit > 0 {
		pages = int64(math.Ceil(float64(total) / float64(limit)))
	}
	return ginLikeMap{
		"page":            page,
		"limit":           limit,
		"pageSize":        limit,
		"total":           total,
		"pages":           pages,
		"totalPages":      pages,
		"hasNextPage":     int64(page*limit) < total,
		"hasPreviousPage": page > 1,
	}
}

func observabilityFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func toStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case nil:
		return ""
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func observabilityRequestType(errorRequest bool) string {
	if errorRequest {
		return "error_request"
	}
	return "slow_request"
}

func requestMetricSeries(now time.Time, metrics []RequestMetric, options PerformanceOptions, slowThreshold time.Duration) []ginLikeMap {
	bucketSeconds := int(options.Bucket.Seconds())
	if bucketSeconds <= 0 {
		bucketSeconds = 300
	}
	bucketCount := int(math.Ceil(options.Window.Seconds() / float64(bucketSeconds)))
	if bucketCount <= 0 {
		bucketCount = 1
	}
	if bucketCount > 720 {
		bucketCount = 720
	}
	start := now.Add(-time.Duration(bucketCount*bucketSeconds) * time.Second)
	buckets := make([]ginLikeMap, bucketCount)
	latencyBuckets := make([][]int64, bucketCount)
	for i := range buckets {
		buckets[i] = ginLikeMap{
			"ts":             start.Add(time.Duration(i*bucketSeconds) * time.Second).UnixMilli(),
			"count":          0,
			"avg_latency_ms": float64(0),
			"p50_latency_ms": float64(0),
			"p95_latency_ms": float64(0),
			"p99_latency_ms": float64(0),
			"max_latency_ms": int64(0),
			"status_4xx":     0,
			"status_5xx":     0,
			"slow_count":     0,
			"error_rate":     float64(0),
		}
	}
	latencyTotals := make([]int64, bucketCount)
	for _, metric := range metrics {
		if metric.CreatedAt.Before(start) || metric.CreatedAt.After(now) {
			continue
		}
		index := int(metric.CreatedAt.Sub(start).Seconds()) / bucketSeconds
		if index < 0 || index >= bucketCount {
			continue
		}
		count := buckets[index]["count"].(int) + 1
		buckets[index]["count"] = count
		latencyTotals[index] += metric.LatencyMS
		latencyBuckets[index] = append(latencyBuckets[index], metric.LatencyMS)
		if metric.LatencyMS > buckets[index]["max_latency_ms"].(int64) {
			buckets[index]["max_latency_ms"] = metric.LatencyMS
		}
		if metric.Status >= 500 {
			buckets[index]["status_5xx"] = buckets[index]["status_5xx"].(int) + 1
		} else if metric.Status >= 400 {
			buckets[index]["status_4xx"] = buckets[index]["status_4xx"].(int) + 1
		}
		if slowThreshold > 0 && metric.LatencyMS >= slowThreshold.Milliseconds() {
			buckets[index]["slow_count"] = buckets[index]["slow_count"].(int) + 1
		}
	}
	for i := range buckets {
		count := buckets[i]["count"].(int)
		if count > 0 {
			buckets[i]["avg_latency_ms"] = float64(latencyTotals[i]) / float64(count)
			buckets[i]["p50_latency_ms"] = percentileLatency(latencyBuckets[i], 0.50)
			buckets[i]["p95_latency_ms"] = percentileLatency(latencyBuckets[i], 0.95)
			buckets[i]["p99_latency_ms"] = percentileLatency(latencyBuckets[i], 0.99)
			buckets[i]["error_rate"] = ratioFloat(buckets[i]["status_4xx"].(int)+buckets[i]["status_5xx"].(int), count)
		}
		buckets[i]["qps"] = float64(count) / float64(bucketSeconds)
	}
	return buckets
}

func percentileLatency(values []int64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	slices.Sort(sorted)
	index := max(int(math.Ceil(percentile*float64(len(sorted))))-1, 0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return float64(sorted[index])
}

func sanitizeSQL(raw string) string {
	sql := strings.TrimSpace(raw)
	if sql == "" {
		return ""
	}
	sql = sqlStringLiteralPattern.ReplaceAllString(sql, "?")
	sql = sqlNumberPattern.ReplaceAllString(sql, "?")
	sql = sqlWhitespacePattern.ReplaceAllString(sql, " ")
	return truncateText(sql, 1000)
}

func sqlFingerprint(sql string) string {
	normalized := strings.ToLower(sanitizeSQL(sql))
	if normalized == "" {
		return ""
	}
	sum := sha1.Sum([]byte(normalized))
	return hex.EncodeToString(sum[:8])
}

func truncateText(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
