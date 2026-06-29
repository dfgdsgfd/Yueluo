package services

import (
	"context"
	"database/sql"
	"maps"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
)

const (
	systemLogStreamKey    = "observability:system_logs"
	accessLogStreamKey    = "observability:access_logs"
	requestMetricZSetKey  = "observability:request_metrics"
	postgresMetricZSetKey = "observability:postgres_metrics"
	runtimeMetricZSetKey  = "observability:runtime_metrics"
	slowRequestZSetKey    = "observability:slow_requests"
	slowQueryZSetKey      = "observability:slow_queries"
)

var (
	sqlStringLiteralPattern = regexp.MustCompile(`'([^']|'')*'`)
	sqlNumberPattern        = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	sqlWhitespacePattern    = regexp.MustCompile(`\s+`)
)

type ObservabilityService struct {
	redis             *RedisStore
	cfg               config.ObservabilityConfig
	logger            *zap.Logger
	writer            *observabilityWriter
	mu                sync.Mutex
	lastRuntimeSample time.Time
	lastNetSample     networkCounterSample
	lastDiskSample    diskCounterSample
	recentClients     map[string]time.Time
	activeRequests    atomic.Int64
	settings          *SettingsService
}

type SystemLogEvent struct {
	Type      string         `json:"type"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	ActorID   int64          `json:"actor_id,omitempty"`
	ActorType string         `json:"actor_type,omitempty"`
	Method    string         `json:"method,omitempty"`
	Path      string         `json:"path,omitempty"`
	Status    int            `json:"status,omitempty"`
	LatencyMS int64          `json:"latency_ms,omitempty"`
	IP        string         `json:"ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type RecentAccessLogEvent struct {
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Nickname  string    `json:"nickname,omitempty"`
	UserType  string    `json:"user_type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RequestMetric struct {
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Route     string    `json:"route,omitempty"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	RequestID string    `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SlowQueryMetric struct {
	Fingerprint string    `json:"fingerprint"`
	SQL         string    `json:"sql"`
	LatencyMS   int64     `json:"latency_ms"`
	Rows        int64     `json:"rows"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type PerformanceOptions struct {
	Window    time.Duration
	Bucket    time.Duration
	SlowLimit int64
}

type ObservabilityEventOptions struct {
	Type    string
	Window  time.Duration
	Page    int
	Limit   int
	Keyword string
	Method  string
	Status  int
}

type ObservabilityEventPage struct {
	Enabled    bool         `json:"enabled"`
	Type       string       `json:"type"`
	Items      []ginLikeMap `json:"items"`
	Pagination ginLikeMap   `json:"pagination"`
	Filters    ginLikeMap   `json:"filters"`
}

type networkCounterSample struct {
	Timestamp time.Time
	Sent      uint64
	Recv      uint64
}

type diskCounterSample struct {
	Timestamp  time.Time
	ReadCount  uint64
	WriteCount uint64
	ReadBytes  uint64
	WriteBytes uint64
	ReadTime   uint64
	WriteTime  uint64
	IOTime     uint64
}

func NewObservabilityService(redis *RedisStore, cfg config.ObservabilityConfig, logger *zap.Logger, settings ...*SettingsService) *ObservabilityService {
	if logger == nil {
		logger = zap.NewNop()
	}
	cfg = normalizeObservabilityConfig(cfg)
	s := &ObservabilityService{
		redis:         redis,
		cfg:           cfg,
		logger:        logger,
		recentClients: map[string]time.Time{},
	}
	if len(settings) > 0 {
		s.settings = settings[0]
	}
	s.writer = newObservabilityWriter(s)
	s.writer.start()
	return s
}

func normalizeObservabilityConfig(cfg config.ObservabilityConfig) config.ObservabilityConfig {
	if cfg.MetricsRetention <= 0 {
		cfg.MetricsRetention = 24 * time.Hour
	}
	if cfg.MetricsBucket <= 0 {
		cfg.MetricsBucket = 5 * time.Minute
	}
	if cfg.RuntimeSampleInterval <= 0 {
		cfg.RuntimeSampleInterval = 5 * time.Minute
	}
	if cfg.SlowRequestThreshold <= 0 {
		cfg.SlowRequestThreshold = time.Second
	}
	if cfg.SlowQueryThreshold <= 0 {
		cfg.SlowQueryThreshold = 500 * time.Millisecond
	}
	return cfg
}

func (s *ObservabilityService) Close() {
	if s == nil {
		return
	}
	if s.writer != nil {
		s.writer.close()
	}
}

func (s *ObservabilityService) Log(event SystemLogEvent) {
	if s == nil || !s.cfg.SystemLogEnabled {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.Level == "" {
		event.Level = "info"
	}
	if s.writer == nil || !s.writer.enqueueSystemLog(event) {
		s.logger.Warn("system log queue full", zap.String("type", event.Type), zap.String("path", event.Path))
	}
}

func (s *ObservabilityService) BeginRequest(ip string) func() {
	if s == nil {
		return func() {}
	}
	s.activeRequests.Add(1)
	return func() {
		s.activeRequests.Add(-1)
	}
}

func (s *ObservabilityService) RecordAccess(ctx context.Context, event RecentAccessLogEvent) {
	_ = ctx
	if s == nil || !s.cfg.SystemLogEnabled || s.redis == nil || s.redis.Client() == nil {
		return
	}
	if shouldSkipRecentAccessLog(event.Path) {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	s.rememberClient(event.IP, event.CreatedAt)
	if s.writer == nil || !s.writer.enqueueAccessLog(event) {
		s.logger.Warn("access log queue full", zap.String("path", event.Path))
	}
}

func (s *ObservabilityService) AccessLogs(ctx context.Context, limit int64) ([]map[string]any, error) {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []map[string]any{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	fetchLimit := min(max(limit*3, 100), 500)
	rows, err := s.redis.Client().XRevRangeN(ctx, accessLogStreamKey, "+", "-", fetchLimit).Result()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{"id": row.ID}
		maps.Copy(item, row.Values)
		if shouldSkipRecentAccessLog(toStringValue(item["path"])) {
			continue
		}
		out = append(out, item)
		if int64(len(out)) >= limit {
			break
		}
	}
	return out, nil
}

func shouldSkipRecentAccessLog(path string) bool {
	return strings.HasPrefix(strings.TrimSpace(path), "/api/admin/")
}

func (s *ObservabilityService) RecordRequest(ctx context.Context, metric RequestMetric) {
	_ = ctx
	if s == nil || !s.cfg.MetricsEnabled || s.redis == nil || s.redis.Client() == nil {
		return
	}
	if metric.CreatedAt.IsZero() {
		metric.CreatedAt = time.Now()
	}
	if s.writer == nil || !s.writer.enqueueRequestMetric(metric) {
		s.logger.Warn("request metrics queue full", zap.String("path", metric.Path))
	}
}

func (s *ObservabilityService) RecordSlowQuery(ctx context.Context, metric SlowQueryMetric) {
	_ = ctx
	if s == nil || !s.cfg.MetricsEnabled || s.redis == nil || s.redis.Client() == nil {
		return
	}
	if metric.CreatedAt.IsZero() {
		metric.CreatedAt = time.Now()
	}
	if metric.SQL != "" {
		metric.SQL = sanitizeSQL(metric.SQL)
	}
	if metric.Fingerprint == "" {
		metric.Fingerprint = sqlFingerprint(metric.SQL)
	}
	if metric.Error != "" {
		metric.Error = truncateText(metric.Error, 300)
	}
	if s.writer == nil || !s.writer.enqueueSlowQuery(metric) {
		s.logger.Warn("slow query queue full", zap.String("fingerprint", metric.Fingerprint))
	}
}

func (s *ObservabilityService) SystemLogs(ctx context.Context, limit int64, cursor string) ([]map[string]any, string, bool, error) {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return []map[string]any{}, "", false, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	max := "+"
	if cursor != "" {
		max = "(" + cursor
	}
	rows, err := s.redis.Client().XRevRangeN(ctx, systemLogStreamKey, max, "-", limit+1).Result()
	if err != nil {
		return nil, "", false, err
	}
	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{"id": row.ID}
		maps.Copy(item, row.Values)
		out = append(out, item)
	}
	nextCursor := ""
	if len(rows) > 0 {
		nextCursor = rows[len(rows)-1].ID
	}
	return out, nextCursor, hasMore, nil
}

func (s *ObservabilityService) Performance(ctx context.Context, db *gorm.DB, options PerformanceOptions) ginLikeMap {
	options = s.normalizePerformanceOptions(options)
	runtimeSnapshot := s.currentRuntimeSnapshot(ctx, time.Now())
	out := ginLikeMap{
		"window_seconds": int(options.Window.Seconds()),
		"bucket_seconds": int(options.Bucket.Seconds()),
		"runtime":        runtimeSnapshot,
		"runtime_series": s.runtimeSeries(ctx, options),
		"requests":       s.requestStats(ctx, options),
		"slow_requests":  ginLikeMap{"items": s.slowRequests(ctx, options)},
		"versions": ginLikeMap{
			"go":  runtime.Version(),
			"gin": gin.Version,
		},
	}
	out["slow_queries"] = ginLikeMap{"items": s.slowQueries(ctx, options), "groups": s.slowQueryGroups(ctx, options, 20)}
	s.recordRuntimeSnapshot(ctx, runtimeSnapshot)
	if sqlDB := sqlDBFromGorm(db); sqlDB != nil {
		stats := sqlDB.Stats()
		postgres := ginLikeMap{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"wait_duration_ms": stats.WaitDuration.Milliseconds(),
			"max_open":         stats.MaxOpenConnections,
		}
		pgStats := postgresDatabaseStats(ctx, db)
		maps.Copy(postgres, pgStats)
		postgres["activity"] = postgresActivityStats(ctx, db)
		postgres["locks"] = postgresLockStats(ctx, db)
		postgres["table_health"] = postgresTableHealth(ctx, db)
		postgres["io"] = postgresIOStats(ctx, db)
		postgres["wal"] = postgresWALStats(ctx, db)
		postgres["checkpointer"] = postgresCheckpointerStats(ctx, db)
		postgres["top_sql"] = postgresTopSQL(ctx, db)
		postgres["diagnostics"] = postgresDiagnostics(ctx, db, stats)
		out["postgresql"] = postgres
		if versions, ok := out["versions"].(ginLikeMap); ok {
			if version, ok := pgStats["server_version"]; ok {
				versions["postgresql"] = version
			}
			if versionText, ok := pgStats["version"]; ok {
				versions["postgresql_full"] = versionText
			}
		}
		s.recordPostgresSnapshot(ctx, postgres)
		out["postgresql_series"] = s.postgresSeries(ctx, options)
	}
	return out
}

func (s *ObservabilityService) Events(ctx context.Context, options ObservabilityEventOptions) ObservabilityEventPage {
	options = s.normalizeEventOptions(options)
	out := ObservabilityEventPage{
		Enabled: true,
		Type:    options.Type,
		Items:   []ginLikeMap{},
		Filters: ginLikeMap{
			"keyword": options.Keyword,
			"method":  options.Method,
			"status":  options.Status,
		},
	}
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		out.Enabled = false
		out.Pagination = eventPagination(options.Page, options.Limit, 0)
		return out
	}
	now := time.Now()
	minScore := strconv.FormatInt(now.Add(-options.Window).UnixMilli(), 10)
	maxScore := strconv.FormatInt(now.UnixMilli(), 10)
	key := requestMetricZSetKey
	switch options.Type {
	case "slow_requests":
		key = slowRequestZSetKey
	case "slow_queries":
		key = slowQueryZSetKey
	}
	rows, err := s.redis.Client().ZRevRangeByScore(ctx, key, &redis.ZRangeBy{Min: minScore, Max: maxScore}).Result()
	if err != nil {
		out.Pagination = eventPagination(options.Page, options.Limit, 0)
		return out
	}
	filtered := filterObservabilityRows(options, rows)
	total := len(filtered)
	start := (options.Page - 1) * options.Limit
	if start < total {
		end := min(start+options.Limit, total)
		out.Items = filtered[start:end]
	}
	out.Pagination = eventPagination(options.Page, options.Limit, int64(total))
	return out
}

func (s *ObservabilityService) normalizeEventOptions(options ObservabilityEventOptions) ObservabilityEventOptions {
	options.Type = strings.TrimSpace(options.Type)
	switch options.Type {
	case "errors", "slow_requests", "slow_queries":
	default:
		options.Type = "errors"
	}
	if options.Window <= 0 {
		if s != nil && s.MetricsRetention() > 0 {
			options.Window = s.MetricsRetention()
		} else {
			options.Window = 24 * time.Hour
		}
	}
	if options.Window > 7*24*time.Hour {
		options.Window = 7 * 24 * time.Hour
	}
	if options.Page <= 0 {
		options.Page = 1
	}
	if options.Limit <= 0 || options.Limit > 100 {
		options.Limit = 30
	}
	options.Keyword = strings.ToLower(strings.TrimSpace(options.Keyword))
	options.Method = strings.ToUpper(strings.TrimSpace(options.Method))
	return options
}

func (s *ObservabilityService) normalizePerformanceOptions(options PerformanceOptions) PerformanceOptions {
	if s == nil {
		return options
	}
	if options.Window <= 0 {
		options.Window = s.MetricsRetention()
	}
	if options.Window <= 0 {
		options.Window = 24 * time.Hour
	}
	if options.Window > 7*24*time.Hour {
		options.Window = 7 * 24 * time.Hour
	}
	if options.Bucket <= 0 {
		options.Bucket = s.cfg.MetricsBucket
	}
	if options.Bucket <= 0 {
		options.Bucket = 5 * time.Minute
	}
	if options.Bucket < time.Minute {
		options.Bucket = time.Minute
	}
	if options.Bucket > options.Window {
		options.Bucket = options.Window
	}
	if options.SlowLimit <= 0 || options.SlowLimit > 100 {
		options.SlowLimit = 50
	}
	return options
}

func (s *ObservabilityService) currentRuntimeSnapshot(ctx context.Context, now time.Time) ginLikeMap {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	snapshot := ginLikeMap{
		"ts":              now.UnixMilli(),
		"goroutines":      runtime.NumGoroutine(),
		"alloc_bytes":     mem.Alloc,
		"sys_bytes":       mem.Sys,
		"heap_alloc":      mem.HeapAlloc,
		"heap_sys":        mem.HeapSys,
		"heap_objects":    mem.HeapObjects,
		"gc_count":        mem.NumGC,
		"last_gc_unix_ms": int64(mem.LastGC / uint64(time.Millisecond)),
	}
	maps.Copy(snapshot, s.systemMetricsSnapshot(ctx, now))
	return snapshot
}

type ginLikeMap map[string]any

func (s *ObservabilityService) systemMetricsSnapshot(ctx context.Context, now time.Time) ginLikeMap {
	out := ginLikeMap{
		"active_requests":               int64(0),
		"unique_clients_60s":            0,
		"current_estimated_connections": int64(0),
		"process_cpu_percent":           float64(0),
		"cpu_cores_percent":             []float64{},
		"net_rx_bps":                    float64(0),
		"net_tx_bps":                    float64(0),
		"disk_io_available":             false,
		"disk_devices":                  0,
		"disk_read_count":               uint64(0),
		"disk_write_count":              uint64(0),
		"disk_read_bytes":               uint64(0),
		"disk_write_bytes":              uint64(0),
		"disk_read_latency_ms":          float64(0),
		"disk_write_latency_ms":         float64(0),
		"disk_read_ops_per_sec":         float64(0),
		"disk_write_ops_per_sec":        float64(0),
		"disk_io_util_percent":          float64(0),
	}
	if s == nil {
		return out
	}
	active, unique, estimated := s.connectionEstimate(now)
	out["active_requests"] = active
	out["unique_clients_60s"] = unique
	out["current_estimated_connections"] = estimated

	if proc, err := process.NewProcessWithContext(ctx, int32(os.Getpid())); err == nil {
		if percent, err := proc.CPUPercentWithContext(ctx); err == nil && finiteFloat(percent) {
			out["process_cpu_percent"] = percent
		}
	}
	if percents, err := cpu.PercentWithContext(ctx, 0, true); err == nil {
		cores := make([]float64, 0, len(percents))
		for index, percent := range percents {
			if finiteFloat(percent) {
				cores = append(cores, percent)
				out["cpu_core_"+strconv.Itoa(index)] = percent
			}
		}
		out["cpu_cores_percent"] = cores
	}
	if counters, err := gnet.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		sample := networkCounterSample{Timestamp: now, Sent: counters[0].BytesSent, Recv: counters[0].BytesRecv}
		rx, tx := s.networkRates(sample)
		out["net_rx_bps"] = rx
		out["net_tx_bps"] = tx
	}
	maps.Copy(out, s.diskIOSnapshot(ctx, now))
	return out
}

func (s *ObservabilityService) diskIOSnapshot(ctx context.Context, now time.Time) ginLikeMap {
	out := ginLikeMap{
		"disk_io_available":      false,
		"disk_devices":           0,
		"disk_read_count":        uint64(0),
		"disk_write_count":       uint64(0),
		"disk_read_bytes":        uint64(0),
		"disk_write_bytes":       uint64(0),
		"disk_read_latency_ms":   float64(0),
		"disk_write_latency_ms":  float64(0),
		"disk_read_ops_per_sec":  float64(0),
		"disk_write_ops_per_sec": float64(0),
		"disk_io_util_percent":   float64(0),
	}
	if s == nil {
		return out
	}
	counters, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		out["disk_io_message"] = err.Error()
		return out
	}
	if len(counters) == 0 {
		out["disk_io_message"] = "disk io counters are unavailable"
		return out
	}
	sample := diskCounterSample{Timestamp: now}
	for _, counter := range counters {
		sample.ReadCount += counter.ReadCount
		sample.WriteCount += counter.WriteCount
		sample.ReadBytes += counter.ReadBytes
		sample.WriteBytes += counter.WriteBytes
		sample.ReadTime += counter.ReadTime
		sample.WriteTime += counter.WriteTime
		sample.IOTime += counter.IoTime
	}
	readLatency, writeLatency, readOps, writeOps, ioUtil := s.diskIORates(sample)
	out["disk_io_available"] = true
	out["disk_devices"] = len(counters)
	out["disk_read_count"] = sample.ReadCount
	out["disk_write_count"] = sample.WriteCount
	out["disk_read_bytes"] = sample.ReadBytes
	out["disk_write_bytes"] = sample.WriteBytes
	out["disk_read_latency_ms"] = readLatency
	out["disk_write_latency_ms"] = writeLatency
	out["disk_read_ops_per_sec"] = readOps
	out["disk_write_ops_per_sec"] = writeOps
	out["disk_io_util_percent"] = ioUtil
	return out
}

func (s *ObservabilityService) rememberClient(ip string, now time.Time) {
	ip = strings.TrimSpace(ip)
	if s == nil || ip == "" {
		return
	}
	s.mu.Lock()
	if s.recentClients == nil {
		s.recentClients = map[string]time.Time{}
	}
	s.recentClients[ip] = now
	s.pruneRecentClientsLocked(now)
	s.mu.Unlock()
}

func (s *ObservabilityService) connectionEstimate(now time.Time) (int64, int, int64) {
	if s == nil {
		return 0, 0, 0
	}
	active := s.activeRequests.Load()
	s.mu.Lock()
	s.pruneRecentClientsLocked(now)
	unique := len(s.recentClients)
	s.mu.Unlock()
	return active, unique, active + int64(unique)
}

func (s *ObservabilityService) pruneRecentClientsLocked(now time.Time) {
	cutoff := now.Add(-60 * time.Second)
	for ip, seenAt := range s.recentClients {
		if seenAt.Before(cutoff) {
			delete(s.recentClients, ip)
		}
	}
}

func (s *ObservabilityService) networkRates(sample networkCounterSample) (float64, float64) {
	if s == nil {
		return 0, 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.lastNetSample
	s.lastNetSample = sample
	if previous.Timestamp.IsZero() || !sample.Timestamp.After(previous.Timestamp) {
		return 0, 0
	}
	seconds := sample.Timestamp.Sub(previous.Timestamp).Seconds()
	if seconds <= 0 {
		return 0, 0
	}
	rx := float64(counterDelta(sample.Recv, previous.Recv)) / seconds
	tx := float64(counterDelta(sample.Sent, previous.Sent)) / seconds
	return rx, tx
}

func (s *ObservabilityService) diskIORates(sample diskCounterSample) (float64, float64, float64, float64, float64) {
	if s == nil {
		return 0, 0, 0, 0, 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.lastDiskSample
	s.lastDiskSample = sample
	if previous.Timestamp.IsZero() || !sample.Timestamp.After(previous.Timestamp) {
		return 0, 0, 0, 0, 0
	}
	seconds := sample.Timestamp.Sub(previous.Timestamp).Seconds()
	if seconds <= 0 {
		return 0, 0, 0, 0, 0
	}
	readCount := counterDelta(sample.ReadCount, previous.ReadCount)
	writeCount := counterDelta(sample.WriteCount, previous.WriteCount)
	readLatency := float64(0)
	writeLatency := float64(0)
	if readCount > 0 {
		readLatency = float64(counterDelta(sample.ReadTime, previous.ReadTime)) / float64(readCount)
	}
	if writeCount > 0 {
		writeLatency = float64(counterDelta(sample.WriteTime, previous.WriteTime)) / float64(writeCount)
	}
	ioUtil := float64(counterDelta(sample.IOTime, previous.IOTime)) / (seconds * 1000) * 100
	if !finiteFloat(ioUtil) || ioUtil < 0 {
		ioUtil = 0
	}
	return readLatency, writeLatency, float64(readCount) / seconds, float64(writeCount) / seconds, ioUtil
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return 0
	}
	return current - previous
}

func finiteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func sqlDBFromGorm(db *gorm.DB) *sql.DB {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	return sqlDB
}
