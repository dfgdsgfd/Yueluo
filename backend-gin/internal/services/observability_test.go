package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"yuem-go/backend-gin/internal/config"
)

func TestSanitizeSQLRedactsLiteralsAndFingerprints(t *testing.T) {
	raw := "SELECT * FROM users WHERE email = 'alice@example.com' AND id = 42 AND token = 'secret-token'"
	sanitized := sanitizeSQL(raw)
	if strings.Contains(sanitized, "alice@example.com") || strings.Contains(sanitized, "secret-token") || strings.Contains(sanitized, "42") {
		t.Fatalf("sanitizeSQL leaked literal values: %q", sanitized)
	}
	if strings.Count(sanitized, "?") < 3 {
		t.Fatalf("sanitizeSQL should replace literals with placeholders, got %q", sanitized)
	}
	if sqlFingerprint(raw) != sqlFingerprint("SELECT * FROM users WHERE email = 'bob@example.com' AND id = 99 AND token = 'another'") {
		t.Fatalf("sqlFingerprint should ignore literal values")
	}
}

func TestRequestMetricSeriesUsesConfiguredBucket(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	series := requestMetricSeries(now, []RequestMetric{
		{LatencyMS: 10, Status: 200, CreatedAt: now.Add(-9 * time.Minute)},
		{LatencyMS: 90, Status: 500, CreatedAt: now.Add(-4 * time.Minute)},
	}, PerformanceOptions{Window: 10 * time.Minute, Bucket: 5 * time.Minute}, 50*time.Millisecond)
	if len(series) != 2 {
		t.Fatalf("bucket count = %d, want 2", len(series))
	}
	first, second := series[0], series[1]
	if first["count"] != 1 || second["count"] != 1 {
		t.Fatalf("unexpected bucket counts: %#v %#v", first["count"], second["count"])
	}
	if second["status_5xx"] != 1 {
		t.Fatalf("second bucket status_5xx = %#v, want 1", second["status_5xx"])
	}
	if second["p95_latency_ms"] != float64(90) {
		t.Fatalf("second p95 = %#v, want 90", second["p95_latency_ms"])
	}
	if second["slow_count"] != 1 || second["error_rate"] != float64(1) || second["max_latency_ms"] != int64(90) {
		t.Fatalf("second diagnostics = slow %#v error %#v max %#v, want 1/1/90", second["slow_count"], second["error_rate"], second["max_latency_ms"])
	}
}

func TestRequestEndpointRankingsAggregateByRoute(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	rankings := requestEndpointRankings(now, []RequestMetric{
		{Method: "GET", Path: "/api/posts/1", Route: "/api/posts/:id", Status: 200, LatencyMS: 25, CreatedAt: now.Add(-3 * time.Minute)},
		{Method: "GET", Path: "/api/posts/2", Route: "/api/posts/:id", Status: 404, LatencyMS: 120, CreatedAt: now.Add(-2 * time.Minute)},
		{Method: "POST", Path: "/api/posts", Route: "/api/posts", Status: 500, LatencyMS: 300, CreatedAt: now.Add(-1 * time.Minute)},
	}, 10*time.Minute, 100*time.Millisecond, 10)
	if len(rankings) != 2 {
		t.Fatalf("rankings length = %d, want 2: %#v", len(rankings), rankings)
	}
	first := rankings[0]
	if first["method"] != "POST" || first["status_5xx"] != 1 || first["slow_count"] != 1 {
		t.Fatalf("first ranking = %#v, want POST 5xx slow endpoint", first)
	}
	var getRoute ginLikeMap
	for _, item := range rankings {
		if item["route"] == "/api/posts/:id" {
			getRoute = item
			break
		}
	}
	if getRoute == nil || getRoute["count"] != 2 || getRoute["status_4xx"] != 1 || getRoute["error_rate"] != 0.5 {
		t.Fatalf("GET route ranking = %#v, want count 2/status_4xx 1/error 0.5", getRoute)
	}
}

func TestSlowQueryGroupsAggregateFingerprint(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	groups := slowQueryGroups([]SlowQueryMetric{
		{Fingerprint: "users-by-id", SQL: "SELECT * FROM users WHERE id = 42", LatencyMS: 800, Rows: 1, CreatedAt: now.Add(-2 * time.Minute)},
		{Fingerprint: "users-by-id", SQL: "SELECT * FROM users WHERE id = 99", LatencyMS: 1200, Rows: 1, Error: "timeout", CreatedAt: now.Add(-1 * time.Minute)},
		{Fingerprint: "posts-feed", SQL: "SELECT * FROM posts", LatencyMS: 300, Rows: 50, CreatedAt: now},
	}, 10)
	if len(groups) != 2 {
		t.Fatalf("groups length = %d, want 2: %#v", len(groups), groups)
	}
	first := groups[0]
	if first["fingerprint"] != "users-by-id" || first["count"] != 2 || first["error_count"] != 1 || first["max_latency_ms"] != int64(1200) {
		t.Fatalf("first slow query group = %#v, want users aggregate", first)
	}
	if strings.Contains(fmt.Sprint(first["sample_sql"]), "99") {
		t.Fatalf("sample SQL should be sanitized, got %#v", first["sample_sql"])
	}
}

func TestFilterObservabilityRowsSearchAndPagination(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	row := func(value any) string {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return "score:" + string(data)
	}

	requestRows := []string{
		row(RequestMetric{Method: "GET", Path: "/api/posts", Status: 500, LatencyMS: 1200, RequestID: "req-500", CreatedAt: now}),
		row(RequestMetric{Method: "POST", Path: "/api/posts", Status: 201, LatencyMS: 20, RequestID: "req-ok", CreatedAt: now}),
	}
	errors := filterObservabilityRows(ObservabilityEventOptions{Type: "errors", Method: "GET", Keyword: "posts"}, requestRows)
	if len(errors) != 1 || errors[0]["status"] != 500 || errors[0]["request_id"] != "req-500" {
		t.Fatalf("filtered errors = %#v, want only req-500", errors)
	}

	queryRows := []string{
		row(SlowQueryMetric{Fingerprint: "users-by-id", SQL: "SELECT * FROM users WHERE id = ?", LatencyMS: 900, Rows: 1, CreatedAt: now}),
		row(SlowQueryMetric{Fingerprint: "posts-feed", SQL: "SELECT * FROM posts", LatencyMS: 700, Rows: 50, CreatedAt: now}),
	}
	queries := filterObservabilityRows(ObservabilityEventOptions{Type: "slow_queries", Keyword: "users"}, queryRows)
	if len(queries) != 1 || queries[0]["fingerprint"] != "users-by-id" {
		t.Fatalf("filtered slow queries = %#v, want users-by-id", queries)
	}

	pagination := eventPagination(2, 10, 21)
	if pagination["pages"] != int64(3) || pagination["hasNextPage"] != true || pagination["hasPreviousPage"] != true {
		t.Fatalf("pagination = %#v, want page 2 of 3 with both directions", pagination)
	}
}

func TestObservabilityWriterFlushesQueuedEvents(t *testing.T) {
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = store.Client().Close()
	})
	service := NewObservabilityService(store, config.ObservabilityConfig{
		SystemLogEnabled:     true,
		SystemLogRetention:   time.Hour,
		MetricsEnabled:       true,
		MetricsRetention:     time.Hour,
		SlowRequestThreshold: 10 * time.Millisecond,
		SlowQueryThreshold:   10 * time.Millisecond,
	}, nil)
	now := time.Now()
	service.RecordRequest(context.Background(), RequestMetric{
		Method:    "GET",
		Path:      "/api/posts",
		Status:    200,
		LatencyMS: 25,
		RequestID: "req-async",
		CreatedAt: now,
	})
	service.RecordSlowQuery(context.Background(), SlowQueryMetric{
		SQL:       "SELECT * FROM users WHERE id = 42",
		LatencyMS: 30,
		Rows:      1,
		CreatedAt: now,
	})
	service.RecordAccess(context.Background(), RecentAccessLogEvent{
		Method:    "GET",
		Path:      "/api/posts",
		Status:    200,
		LatencyMS: 25,
		IP:        "10.0.0.1",
		RequestID: "req-async",
		CreatedAt: now,
	})
	service.Log(SystemLogEvent{
		Type:      "test",
		Level:     "info",
		Message:   "queued",
		RequestID: "req-async",
		CreatedAt: now,
	})
	service.Close()

	ctx := context.Background()
	waitForObservabilityWriter(t, func() (bool, string) {
		requests, _ := store.Client().ZCard(ctx, requestMetricZSetKey).Result()
		slowRequests, _ := store.Client().ZCard(ctx, slowRequestZSetKey).Result()
		slowQueries, _ := store.Client().ZCard(ctx, slowQueryZSetKey).Result()
		accessLogs, _ := store.Client().XLen(ctx, accessLogStreamKey).Result()
		systemLogs, _ := store.Client().XLen(ctx, systemLogStreamKey).Result()
		ok := requests == 1 && slowRequests == 1 && slowQueries == 1 && accessLogs == 1 && systemLogs == 1
		return ok, fmt.Sprintf("requests=%d slowRequests=%d slowQueries=%d accessLogs=%d systemLogs=%d", requests, slowRequests, slowQueries, accessLogs, systemLogs)
	})
	rows, err := store.Client().ZRange(ctx, slowQueryZSetKey, 0, -1).Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("slow query row should be sanitized, got %#v", rows)
	}
	parts := strings.SplitN(rows[0], ":", 3)
	if len(parts) != 3 {
		t.Fatalf("slow query row has unexpected format: %#v", rows)
	}
	var slowQuery map[string]any
	if err := json.Unmarshal([]byte(parts[2]), &slowQuery); err != nil {
		t.Fatalf("decode slow query row: %v", err)
	}
	sqlText := fmt.Sprint(slowQuery["sql"])
	if strings.Contains(sqlText, "42") {
		t.Fatalf("slow query SQL should be sanitized, got %q", sqlText)
	}
}

func waitForObservabilityWriter(t *testing.T, condition func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		ok, detail := condition()
		last = detail
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ok, detail := condition(); !ok {
		if detail != "" {
			last = detail
		}
		t.Fatalf("observability writer did not flush queued events: %s", last)
	}
}

func TestConnectionEstimateTracksRecentClientsAndActiveRequests(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	service := &ObservabilityService{recentClients: map[string]time.Time{}}
	finish := service.BeginRequest("10.0.0.1")
	service.rememberClient("10.0.0.1", now)
	service.rememberClient("10.0.0.2", now.Add(-30*time.Second))
	service.rememberClient("10.0.0.3", now.Add(-61*time.Second))

	active, unique, estimated := service.connectionEstimate(now)
	if active != 1 || unique != 2 || estimated != 3 {
		t.Fatalf("connectionEstimate() = active %d unique %d estimated %d, want 1/2/3", active, unique, estimated)
	}
	finish()
	active, unique, estimated = service.connectionEstimate(now)
	if active != 0 || unique != 2 || estimated != 2 {
		t.Fatalf("connectionEstimate() after finish = active %d unique %d estimated %d, want 0/2/2", active, unique, estimated)
	}
}

func TestNetworkRatesUseCounterDeltas(t *testing.T) {
	service := &ObservabilityService{}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	rx, tx := service.networkRates(networkCounterSample{Timestamp: now, Recv: 100, Sent: 200})
	if rx != 0 || tx != 0 {
		t.Fatalf("first networkRates() = %f/%f, want 0/0", rx, tx)
	}
	rx, tx = service.networkRates(networkCounterSample{Timestamp: now.Add(2 * time.Second), Recv: 500, Sent: 1000})
	if rx != 200 || tx != 400 {
		t.Fatalf("networkRates() = %f/%f, want 200/400", rx, tx)
	}
	if got := counterDelta(5, 10); got != 0 {
		t.Fatalf("counterDelta wrap guard = %d, want 0", got)
	}
}

func TestDiskIORatesUseCounterDeltas(t *testing.T) {
	service := &ObservabilityService{}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	readLatency, writeLatency, readOps, writeOps, util := service.diskIORates(diskCounterSample{
		Timestamp:  now,
		ReadCount:  100,
		WriteCount: 50,
		ReadTime:   300,
		WriteTime:  500,
		IOTime:     1000,
	})
	if readLatency != 0 || writeLatency != 0 || readOps != 0 || writeOps != 0 || util != 0 {
		t.Fatalf("first diskIORates() = %f/%f/%f/%f/%f, want all zero", readLatency, writeLatency, readOps, writeOps, util)
	}

	readLatency, writeLatency, readOps, writeOps, util = service.diskIORates(diskCounterSample{
		Timestamp:  now.Add(10 * time.Second),
		ReadCount:  120,
		WriteCount: 60,
		ReadTime:   500,
		WriteTime:  650,
		IOTime:     1500,
	})
	if readLatency != 10 || writeLatency != 15 || readOps != 2 || writeOps != 1 || util != 5 {
		t.Fatalf("diskIORates() = %f/%f/%f/%f/%f, want 10/15/2/1/5", readLatency, writeLatency, readOps, writeOps, util)
	}
}

func TestPostgresWALSelectListDefaultsRemovedColumns(t *testing.T) {
	columns := map[string]bool{
		"wal_records":      true,
		"wal_fpi":          true,
		"wal_bytes":        true,
		"wal_buffers_full": true,
	}
	got := postgresWALSelectList(columns)
	want := []string{
		"wal_records AS wal_records",
		"wal_fpi AS wal_fpi",
		"wal_bytes::float8 AS wal_bytes",
		"wal_buffers_full AS wal_buffers_full",
		"0::bigint AS wal_write",
		"0::bigint AS wal_sync",
		"0::float8 AS wal_write_ms",
		"0::float8 AS wal_sync_ms",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("postgresWALSelectList() = %#v, want %#v", got, want)
	}
}

func TestPostgresWALIOSelectListUsesIOCounters(t *testing.T) {
	columns := map[string]bool{
		"writes":     true,
		"fsyncs":     true,
		"write_time": true,
		"fsync_time": true,
	}
	got := strings.Join(postgresWALIOSelectList(columns), ", ")
	for _, fragment := range []string{
		"COALESCE(SUM(writes), 0)::bigint AS wal_write",
		"COALESCE(SUM(fsyncs), 0)::bigint AS wal_sync",
		"COALESCE(SUM(write_time), 0)::float8 AS wal_write_ms",
		"COALESCE(SUM(fsync_time), 0)::float8 AS wal_sync_ms",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("postgresWALIOSelectList() missing %q in %q", fragment, got)
		}
	}
}

func TestPostgresIOSelectListDefaultsMissingCounters(t *testing.T) {
	columns := map[string]bool{
		"backend_type": true,
		"object":       true,
		"context":      true,
		"reads":        true,
	}
	got := strings.Join(postgresIOSelectList(columns), ", ")
	if !strings.Contains(got, "0::bigint AS writes") {
		t.Fatalf("postgresIOSelectList() should default missing writes counter, got %q", got)
	}
	if activity := postgresIOActivityExpr(columns); activity != "reads" {
		t.Fatalf("postgresIOActivityExpr() = %q, want reads", activity)
	}
}

func TestPostgresWaitEventSelectListDefaultsMissingColumns(t *testing.T) {
	got := strings.Join(postgresWaitEventSelectList(map[string]bool{"wait_event": true}), ", ")
	for _, fragment := range []string{
		"'unavailable' AS wait_event_type",
		"COALESCE(wait_event, 'unknown') AS wait_event",
		"COUNT(*) AS count",
		"0 AS max_wait_ms",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("postgresWaitEventSelectList() missing %q in %q", fragment, got)
		}
	}
}

func TestPostgresCheckpointerSelectListDefaultsMissingColumns(t *testing.T) {
	got := strings.Join(postgresCheckpointerSelectList(map[string]bool{"num_timed": true}), ", ")
	for _, fragment := range []string{
		"num_timed AS num_timed",
		"0::bigint AS num_requested",
		"0::float8 AS write_time_ms",
		"0::float8 AS sync_time_ms",
		"0::bigint AS buffers_written",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("postgresCheckpointerSelectList() missing %q in %q", fragment, got)
		}
	}
}

func TestPostgresLibraryListContainsSharedPreloadEntry(t *testing.T) {
	setting := `auto_explain, "$libdir/pg_stat_statements", pg_cron.dll`
	if !postgresLibraryListContains(setting, "pg_stat_statements") {
		t.Fatalf("postgresLibraryListContains() should find pg_stat_statements in %q", setting)
	}
	if !postgresLibraryListContains(setting, "pg_cron") {
		t.Fatalf("postgresLibraryListContains() should ignore library suffixes in %q", setting)
	}
	if postgresLibraryListContains(setting, "pg_hint_plan") {
		t.Fatalf("postgresLibraryListContains() found unexpected library in %q", setting)
	}
}

func TestPostgresTopSQLSelectListSupportsLegacyTimingColumns(t *testing.T) {
	columns := map[string]bool{
		"queryid":          true,
		"calls":            true,
		"total_time":       true,
		"mean_time":        true,
		"rows":             true,
		"shared_blks_hit":  true,
		"shared_blks_read": true,
		"query":            true,
	}
	got := strings.Join(postgresTopSQLSelectList(columns), ", ")
	if !strings.Contains(got, "total_time AS total_exec_time_ms") {
		t.Fatalf("postgresTopSQLSelectList() should use legacy total_time, got %q", got)
	}
	if !strings.Contains(got, "mean_time AS mean_exec_time_ms") {
		t.Fatalf("postgresTopSQLSelectList() should use legacy mean_time, got %q", got)
	}
}
