package services

import (
	"math"
	"slices"
	"strings"
	"time"
)

type requestEndpointAggregate struct {
	Method       string
	Route        string
	SamplePath   string
	Count        int
	TotalLatency int64
	MaxLatency   int64
	Status4xx    int
	Status5xx    int
	SlowCount    int
	LastSeen     time.Time
	Latencies    []int64
}

func requestEndpointRankings(now time.Time, metrics []RequestMetric, window time.Duration, slowThreshold time.Duration, limit int) []ginLikeMap {
	if len(metrics) == 0 || limit <= 0 {
		return []ginLikeMap{}
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	aggregates := map[string]*requestEndpointAggregate{}
	for _, metric := range metrics {
		method := strings.ToUpper(strings.TrimSpace(metric.Method))
		if method == "" {
			method = "-"
		}
		route := requestMetricRoute(metric)
		key := method + " " + route
		aggregate := aggregates[key]
		if aggregate == nil {
			aggregate = &requestEndpointAggregate{
				Method:     method,
				Route:      route,
				SamplePath: strings.TrimSpace(metric.Path),
				Latencies:  make([]int64, 0, 8),
			}
			aggregates[key] = aggregate
		}
		aggregate.Count++
		aggregate.TotalLatency += metric.LatencyMS
		aggregate.Latencies = append(aggregate.Latencies, metric.LatencyMS)
		if metric.LatencyMS > aggregate.MaxLatency {
			aggregate.MaxLatency = metric.LatencyMS
		}
		if metric.Status >= 500 {
			aggregate.Status5xx++
		} else if metric.Status >= 400 {
			aggregate.Status4xx++
		}
		if slowThreshold > 0 && metric.LatencyMS >= slowThreshold.Milliseconds() {
			aggregate.SlowCount++
		}
		if metric.CreatedAt.After(aggregate.LastSeen) {
			aggregate.LastSeen = metric.CreatedAt
			if path := strings.TrimSpace(metric.Path); path != "" {
				aggregate.SamplePath = path
			}
		}
	}
	ordered := make([]*requestEndpointAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		ordered = append(ordered, aggregate)
	}
	slices.SortFunc(ordered, func(a, b *requestEndpointAggregate) int {
		if diff := compareFloatDesc(percentileLatency(a.Latencies, 0.95), percentileLatency(b.Latencies, 0.95)); diff != 0 {
			return diff
		}
		if diff := compareIntDesc(a.Status5xx+a.Status4xx, b.Status5xx+b.Status4xx); diff != 0 {
			return diff
		}
		return compareIntDesc(a.Count, b.Count)
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	out := make([]ginLikeMap, 0, len(ordered))
	for _, aggregate := range ordered {
		avg := float64(0)
		if aggregate.Count > 0 {
			avg = float64(aggregate.TotalLatency) / float64(aggregate.Count)
		}
		item := ginLikeMap{
			"method":         aggregate.Method,
			"path":           aggregate.Route,
			"route":          aggregate.Route,
			"sample_path":    aggregate.SamplePath,
			"count":          aggregate.Count,
			"qps":            float64(aggregate.Count) / math.Max(window.Seconds(), 1),
			"avg_latency_ms": avg,
			"p50_latency_ms": percentileLatency(aggregate.Latencies, 0.50),
			"p95_latency_ms": percentileLatency(aggregate.Latencies, 0.95),
			"p99_latency_ms": percentileLatency(aggregate.Latencies, 0.99),
			"max_latency_ms": aggregate.MaxLatency,
			"status_4xx":     aggregate.Status4xx,
			"status_5xx":     aggregate.Status5xx,
			"error_rate":     ratioFloat(aggregate.Status4xx+aggregate.Status5xx, aggregate.Count),
			"slow_count":     aggregate.SlowCount,
		}
		if !aggregate.LastSeen.IsZero() {
			item["last_seen"] = aggregate.LastSeen.Format(time.RFC3339Nano)
			item["last_seen_ts"] = aggregate.LastSeen.UnixMilli()
		}
		out = append(out, item)
	}
	return out
}

func requestMetricRoute(metric RequestMetric) string {
	if route := strings.TrimSpace(metric.Route); route != "" {
		return route
	}
	if path := strings.TrimSpace(metric.Path); path != "" {
		return path
	}
	return "-"
}

type slowQueryAggregate struct {
	Fingerprint  string
	SampleSQL    string
	Count        int
	TotalLatency int64
	MaxLatency   int64
	Rows         int64
	ErrorCount   int
	LastSeen     time.Time
	Latencies    []int64
}

func slowQueryGroups(metrics []SlowQueryMetric, limit int) []ginLikeMap {
	if len(metrics) == 0 || limit <= 0 {
		return []ginLikeMap{}
	}
	aggregates := map[string]*slowQueryAggregate{}
	for _, metric := range metrics {
		fingerprint := strings.TrimSpace(metric.Fingerprint)
		if fingerprint == "" {
			fingerprint = sqlFingerprint(metric.SQL)
		}
		if fingerprint == "" {
			fingerprint = "-"
		}
		aggregate := aggregates[fingerprint]
		if aggregate == nil {
			aggregate = &slowQueryAggregate{
				Fingerprint: fingerprint,
				SampleSQL:   sanitizeSQL(metric.SQL),
				Latencies:   make([]int64, 0, 4),
			}
			aggregates[fingerprint] = aggregate
		}
		aggregate.Count++
		aggregate.TotalLatency += metric.LatencyMS
		aggregate.Latencies = append(aggregate.Latencies, metric.LatencyMS)
		if metric.LatencyMS > aggregate.MaxLatency {
			aggregate.MaxLatency = metric.LatencyMS
		}
		if metric.Rows > 0 {
			aggregate.Rows += metric.Rows
		}
		if strings.TrimSpace(metric.Error) != "" {
			aggregate.ErrorCount++
		}
		if metric.CreatedAt.After(aggregate.LastSeen) {
			aggregate.LastSeen = metric.CreatedAt
			if sql := sanitizeSQL(metric.SQL); sql != "" {
				aggregate.SampleSQL = sql
			}
		}
	}
	ordered := make([]*slowQueryAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		ordered = append(ordered, aggregate)
	}
	slices.SortFunc(ordered, func(a, b *slowQueryAggregate) int {
		if diff := compareFloatDesc(percentileLatency(a.Latencies, 0.95), percentileLatency(b.Latencies, 0.95)); diff != 0 {
			return diff
		}
		return compareIntDesc(a.Count, b.Count)
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	out := make([]ginLikeMap, 0, len(ordered))
	for _, aggregate := range ordered {
		avg := float64(0)
		if aggregate.Count > 0 {
			avg = float64(aggregate.TotalLatency) / float64(aggregate.Count)
		}
		item := ginLikeMap{
			"fingerprint":    aggregate.Fingerprint,
			"count":          aggregate.Count,
			"avg_latency_ms": avg,
			"p50_latency_ms": percentileLatency(aggregate.Latencies, 0.50),
			"p95_latency_ms": percentileLatency(aggregate.Latencies, 0.95),
			"p99_latency_ms": percentileLatency(aggregate.Latencies, 0.99),
			"max_latency_ms": aggregate.MaxLatency,
			"rows":           aggregate.Rows,
			"error_count":    aggregate.ErrorCount,
			"sample_sql":     aggregate.SampleSQL,
		}
		if !aggregate.LastSeen.IsZero() {
			item["last_seen"] = aggregate.LastSeen.Format(time.RFC3339Nano)
			item["last_seen_ts"] = aggregate.LastSeen.UnixMilli()
		}
		out = append(out, item)
	}
	return out
}

func ratioFloat(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func compareIntDesc(a, b int) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

func compareFloatDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}
