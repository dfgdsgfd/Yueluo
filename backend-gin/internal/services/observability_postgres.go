package services

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func (s *ObservabilityService) requestStats(ctx context.Context, options PerformanceOptions) ginLikeMap {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return ginLikeMap{"count": 0}
	}
	now := time.Now()
	window := options.Window
	if window <= 0 {
		window = 24 * time.Hour
	}
	min := strconv.FormatInt(now.Add(-window).UnixMilli(), 10)
	max := strconv.FormatInt(now.UnixMilli(), 10)
	rows, err := s.redis.Client().ZRangeByScore(ctx, requestMetricZSetKey, &redis.ZRangeBy{Min: min, Max: max}).Result()
	if err != nil {
		return ginLikeMap{"count": 0}
	}
	metrics := requestMetricsFromRows(rows)
	var totalLatency int64
	var maxLatency int64
	status2xx := 0
	status3xx := 0
	status4xx := 0
	status5xx := 0
	slowCount := 0
	latencies := make([]int64, 0, len(metrics))
	series := requestMetricSeries(now, metrics, options, s.cfg.SlowRequestThreshold)
	for _, metric := range metrics {
		totalLatency += metric.LatencyMS
		latencies = append(latencies, metric.LatencyMS)
		if metric.LatencyMS > maxLatency {
			maxLatency = metric.LatencyMS
		}
		switch {
		case metric.Status >= 500:
			status5xx++
		case metric.Status >= 400:
			status4xx++
		case metric.Status >= 300:
			status3xx++
		case metric.Status >= 200:
			status2xx++
		}
		if s.cfg.SlowRequestThreshold > 0 && metric.LatencyMS >= s.cfg.SlowRequestThreshold.Milliseconds() {
			slowCount++
		}
	}
	avg := float64(0)
	if len(metrics) > 0 {
		avg = float64(totalLatency) / float64(len(metrics))
	}
	errorCount := status4xx + status5xx
	return ginLikeMap{
		"window_seconds": int(window.Seconds()),
		"count":          len(metrics),
		"qps":            float64(len(metrics)) / math.Max(window.Seconds(), 1),
		"avg_latency_ms": avg,
		"p50_latency_ms": percentileLatency(latencies, 0.50),
		"p95_latency_ms": percentileLatency(latencies, 0.95),
		"p99_latency_ms": percentileLatency(latencies, 0.99),
		"max_latency_ms": maxLatency,
		"status_2xx":     status2xx,
		"status_3xx":     status3xx,
		"status_4xx":     status4xx,
		"status_5xx":     status5xx,
		"error_rate":     ratioFloat(errorCount, len(metrics)),
		"slow_count":     slowCount,
		"endpoints":      requestEndpointRankings(now, metrics, window, s.cfg.SlowRequestThreshold, 20),
		"series":         series,
	}
}

func postgresDatabaseStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		XactCommit   int64 `gorm:"column:xact_commit"`
		XactRollback int64 `gorm:"column:xact_rollback"`
		BlksRead     int64 `gorm:"column:blks_read"`
		BlksHit      int64 `gorm:"column:blks_hit"`
		TupReturned  int64 `gorm:"column:tup_returned"`
		TupFetched   int64 `gorm:"column:tup_fetched"`
		TupInserted  int64 `gorm:"column:tup_inserted"`
		TupUpdated   int64 `gorm:"column:tup_updated"`
		TupDeleted   int64 `gorm:"column:tup_deleted"`
		Deadlocks    int64 `gorm:"column:deadlocks"`
		TempBytes    int64 `gorm:"column:temp_bytes"`
		NumBackends  int64 `gorm:"column:numbackends"`
	}
	var stat row
	err := db.WithContext(ctx).Raw(`SELECT xact_commit, xact_rollback, blks_read, blks_hit, tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted, deadlocks, temp_bytes, numbackends FROM pg_stat_database WHERE datname = current_database()`).Scan(&stat).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	var serverVersion string
	_ = db.WithContext(ctx).Raw(`SHOW server_version`).Scan(&serverVersion).Error
	var fullVersion string
	_ = db.WithContext(ctx).Raw(`SELECT version()`).Scan(&fullVersion).Error
	cacheHitRatio := float64(0)
	if stat.BlksHit+stat.BlksRead > 0 {
		cacheHitRatio = float64(stat.BlksHit) / float64(stat.BlksHit+stat.BlksRead)
	}
	return ginLikeMap{
		"available":       true,
		"xact_commit":     stat.XactCommit,
		"xact_rollback":   stat.XactRollback,
		"blks_read":       stat.BlksRead,
		"blks_hit":        stat.BlksHit,
		"cache_hit_ratio": cacheHitRatio,
		"numbackends":     stat.NumBackends,
		"tup_returned":    stat.TupReturned,
		"tup_fetched":     stat.TupFetched,
		"tup_inserted":    stat.TupInserted,
		"tup_updated":     stat.TupUpdated,
		"tup_deleted":     stat.TupDeleted,
		"deadlocks":       stat.Deadlocks,
		"temp_bytes":      stat.TempBytes,
		"server_version":  serverVersion,
		"version":         fullVersion,
	}
}

func postgresActivityStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type stateRow struct {
		State string `gorm:"column:state"`
		Count int64  `gorm:"column:count"`
	}
	var states []stateRow
	err := db.WithContext(ctx).Raw(`
		SELECT COALESCE(state, 'unknown') AS state, COUNT(*) AS count
		FROM pg_stat_activity
		WHERE datname = current_database()
		GROUP BY COALESCE(state, 'unknown')`).Scan(&states).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	type summaryRow struct {
		TotalConnections    int64   `gorm:"column:total_connections"`
		ActiveConnections   int64   `gorm:"column:active_connections"`
		IdleInTransaction   int64   `gorm:"column:idle_in_transaction"`
		Waiting             int64   `gorm:"column:waiting"`
		LongTransactions    int64   `gorm:"column:long_transactions"`
		MaxTransactionAgeMS float64 `gorm:"column:max_transaction_age_ms"`
		MaxQueryAgeMS       float64 `gorm:"column:max_query_age_ms"`
	}
	var summary summaryRow
	if err := db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS total_connections,
			COUNT(*) FILTER (WHERE state = 'active') AS active_connections,
			COUNT(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_transaction,
			COUNT(*) FILTER (WHERE wait_event IS NOT NULL) AS waiting,
			COUNT(*) FILTER (WHERE xact_start IS NOT NULL AND now() - xact_start > interval '5 minutes') AS long_transactions,
			COALESCE(MAX(EXTRACT(EPOCH FROM (now() - xact_start)) * 1000), 0) AS max_transaction_age_ms,
			COALESCE(MAX(EXTRACT(EPOCH FROM (now() - query_start)) * 1000), 0) AS max_query_age_ms
		FROM pg_stat_activity
		WHERE datname = current_database()`).Scan(&summary).Error; err != nil {
		return postgresUnavailable(err)
	}
	byState := ginLikeMap{}
	for _, row := range states {
		byState[row.State] = row.Count
	}
	return ginLikeMap{
		"available":              true,
		"by_state":               byState,
		"total_connections":      summary.TotalConnections,
		"active_connections":     summary.ActiveConnections,
		"idle_in_transaction":    summary.IdleInTransaction,
		"waiting":                summary.Waiting,
		"long_transactions":      summary.LongTransactions,
		"max_transaction_age_ms": summary.MaxTransactionAgeMS,
		"max_query_age_ms":       summary.MaxQueryAgeMS,
	}
}

func postgresLockStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		TotalLocks   int64 `gorm:"column:total_locks"`
		WaitingLocks int64 `gorm:"column:waiting_locks"`
		GrantedLocks int64 `gorm:"column:granted_locks"`
	}
	var stat row
	err := db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS total_locks,
			COUNT(*) FILTER (WHERE NOT granted) AS waiting_locks,
			COUNT(*) FILTER (WHERE granted) AS granted_locks
		FROM pg_locks l
		LEFT JOIN pg_database d ON d.oid = l.database
		WHERE d.datname = current_database() OR l.database IS NULL`).Scan(&stat).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	return ginLikeMap{"available": true, "total_locks": stat.TotalLocks, "waiting_locks": stat.WaitingLocks, "granted_locks": stat.GrantedLocks}
}

func postgresTableHealth(ctx context.Context, db *gorm.DB) ginLikeMap {
	type summaryRow struct {
		LiveTuples    int64 `gorm:"column:live_tuples"`
		DeadTuples    int64 `gorm:"column:dead_tuples"`
		ManualVacuum  int64 `gorm:"column:manual_vacuum"`
		AutoVacuum    int64 `gorm:"column:auto_vacuum"`
		ManualAnalyze int64 `gorm:"column:manual_analyze"`
		AutoAnalyze   int64 `gorm:"column:auto_analyze"`
		StaleAnalyze  int64 `gorm:"column:stale_analyze"`
	}
	var summary summaryRow
	err := db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(n_live_tup), 0) AS live_tuples,
			COALESCE(SUM(n_dead_tup), 0) AS dead_tuples,
			COALESCE(SUM(vacuum_count), 0) AS manual_vacuum,
			COALESCE(SUM(autovacuum_count), 0) AS auto_vacuum,
			COALESCE(SUM(analyze_count), 0) AS manual_analyze,
			COALESCE(SUM(autoanalyze_count), 0) AS auto_analyze,
			COUNT(*) FILTER (WHERE last_analyze IS NULL AND last_autoanalyze IS NULL) AS stale_analyze
		FROM pg_stat_all_tables
		WHERE schemaname = current_schema()`).Scan(&summary).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	type tableRow struct {
		TableName       string     `gorm:"column:table_name"`
		LiveTuples      int64      `gorm:"column:live_tuples"`
		DeadTuples      int64      `gorm:"column:dead_tuples"`
		DeadRatio       float64    `gorm:"column:dead_ratio"`
		LastVacuum      *time.Time `gorm:"column:last_vacuum"`
		LastAutoVacuum  *time.Time `gorm:"column:last_autovacuum"`
		LastAnalyze     *time.Time `gorm:"column:last_analyze"`
		LastAutoAnalyze *time.Time `gorm:"column:last_autoanalyze"`
	}
	var rows []tableRow
	_ = db.WithContext(ctx).Raw(`
		SELECT relname AS table_name,
			n_live_tup AS live_tuples,
			n_dead_tup AS dead_tuples,
			CASE WHEN n_live_tup + n_dead_tup > 0 THEN n_dead_tup::float8 / (n_live_tup + n_dead_tup) ELSE 0 END AS dead_ratio,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_all_tables
		WHERE schemaname = current_schema()
		ORDER BY n_dead_tup DESC, relname ASC
		LIMIT 10`).Scan(&rows).Error
	top := make([]ginLikeMap, 0, len(rows))
	for _, row := range rows {
		top = append(top, ginLikeMap{
			"table":            row.TableName,
			"live_tuples":      row.LiveTuples,
			"dead_tuples":      row.DeadTuples,
			"dead_ratio":       row.DeadRatio,
			"last_vacuum":      timePtrRFC3339(row.LastVacuum),
			"last_autovacuum":  timePtrRFC3339(row.LastAutoVacuum),
			"last_analyze":     timePtrRFC3339(row.LastAnalyze),
			"last_autoanalyze": timePtrRFC3339(row.LastAutoAnalyze),
		})
	}
	return ginLikeMap{
		"available":      true,
		"live_tuples":    summary.LiveTuples,
		"dead_tuples":    summary.DeadTuples,
		"dead_ratio":     ratio(summary.DeadTuples, summary.LiveTuples+summary.DeadTuples),
		"manual_vacuum":  summary.ManualVacuum,
		"auto_vacuum":    summary.AutoVacuum,
		"manual_analyze": summary.ManualAnalyze,
		"auto_analyze":   summary.AutoAnalyze,
		"stale_analyze":  summary.StaleAnalyze,
		"top_dead":       top,
	}
}

func postgresIOStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		BackendType string `gorm:"column:backend_type"`
		Object      string `gorm:"column:object"`
		Context     string `gorm:"column:context"`
		Reads       int64  `gorm:"column:reads"`
		Writes      int64  `gorm:"column:writes"`
		Extends     int64  `gorm:"column:extends"`
		Hits        int64  `gorm:"column:hits"`
		Evictions   int64  `gorm:"column:evictions"`
		Fsyncs      int64  `gorm:"column:fsyncs"`
	}
	columns, err := postgresRelationColumns(ctx, db, "pg_catalog.pg_stat_io")
	if err != nil {
		return postgresUnavailable(err)
	}
	if len(columns) == 0 {
		return postgresUnavailableMessage("pg_stat_io is unavailable")
	}
	var rows []row
	query := "SELECT " + strings.Join(postgresIOSelectList(columns), ", ") + " FROM pg_catalog.pg_stat_io ORDER BY (" + postgresIOActivityExpr(columns) + ") DESC LIMIT 24"
	err = db.WithContext(ctx).Raw(query).Scan(&rows).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	items := make([]ginLikeMap, 0, len(rows))
	var reads, writes, extends, hits, evictions, fsyncs int64
	for _, row := range rows {
		reads += row.Reads
		writes += row.Writes
		extends += row.Extends
		hits += row.Hits
		evictions += row.Evictions
		fsyncs += row.Fsyncs
		items = append(items, ginLikeMap{
			"backend_type": row.BackendType,
			"object":       row.Object,
			"context":      row.Context,
			"reads":        row.Reads,
			"writes":       row.Writes,
			"extends":      row.Extends,
			"hits":         row.Hits,
			"evictions":    row.Evictions,
			"fsyncs":       row.Fsyncs,
		})
	}
	return ginLikeMap{"available": true, "reads": reads, "writes": writes, "extends": extends, "hits": hits, "evictions": evictions, "fsyncs": fsyncs, "items": items}
}

func postgresIOSelectList(columns map[string]bool) []string {
	return []string{
		postgresStatColumnExpr(columns, "backend_type", "backend_type", "''::text", "backend_type"),
		postgresStatColumnExpr(columns, "object", "object", "''::text", "object"),
		postgresStatColumnExpr(columns, "context", "context", "''::text", "context"),
		postgresStatColumnExpr(columns, "reads", "reads", "0::bigint", "reads"),
		postgresStatColumnExpr(columns, "writes", "writes", "0::bigint", "writes"),
		postgresStatColumnExpr(columns, "extends", "extends", "0::bigint", "extends"),
		postgresStatColumnExpr(columns, "hits", "hits", "0::bigint", "hits"),
		postgresStatColumnExpr(columns, "evictions", "evictions", "0::bigint", "evictions"),
		postgresStatColumnExpr(columns, "fsyncs", "fsyncs", "0::bigint", "fsyncs"),
	}
}

func postgresIOActivityExpr(columns map[string]bool) string {
	parts := make([]string, 0, 4)
	for _, column := range []string{"reads", "writes", "extends", "hits"} {
		if columns[column] {
			parts = append(parts, column)
		}
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, " + ")
}

type postgresColumnRow struct {
	Name string `gorm:"column:attname"`
}

func postgresRelationColumns(ctx context.Context, db *gorm.DB, relation string) (map[string]bool, error) {
	var rows []postgresColumnRow
	err := db.WithContext(ctx).Raw(`
		SELECT attname
		FROM pg_attribute
		WHERE attrelid = to_regclass(?)
			AND attnum > 0
			AND NOT attisdropped`, relation).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	columns := make(map[string]bool, len(rows))
	for _, row := range rows {
		columns[row.Name] = true
	}
	return columns, nil
}

func postgresStatColumnExpr(columns map[string]bool, name string, expression string, fallback string, alias string) string {
	if columns[name] {
		return expression + " AS " + alias
	}
	return fallback + " AS " + alias
}

func postgresWALSelectList(columns map[string]bool) []string {
	return []string{
		postgresStatColumnExpr(columns, "wal_records", "wal_records", "0::bigint", "wal_records"),
		postgresStatColumnExpr(columns, "wal_fpi", "wal_fpi", "0::bigint", "wal_fpi"),
		postgresStatColumnExpr(columns, "wal_bytes", "wal_bytes::float8", "0::float8", "wal_bytes"),
		postgresStatColumnExpr(columns, "wal_buffers_full", "wal_buffers_full", "0::bigint", "wal_buffers_full"),
		postgresStatColumnExpr(columns, "wal_write", "wal_write", "0::bigint", "wal_write"),
		postgresStatColumnExpr(columns, "wal_sync", "wal_sync", "0::bigint", "wal_sync"),
		postgresStatColumnExpr(columns, "wal_write_time", "wal_write_time", "0::float8", "wal_write_ms"),
		postgresStatColumnExpr(columns, "wal_sync_time", "wal_sync_time", "0::float8", "wal_sync_ms"),
	}
}

type postgresWALIOStat struct {
	WalWrite   int64   `gorm:"column:wal_write"`
	WalSync    int64   `gorm:"column:wal_sync"`
	WalWriteMS float64 `gorm:"column:wal_write_ms"`
	WalSyncMS  float64 `gorm:"column:wal_sync_ms"`
}

func postgresWALIOSelectList(columns map[string]bool) []string {
	return []string{
		postgresStatColumnExpr(columns, "writes", "COALESCE(SUM(writes), 0)::bigint", "0::bigint", "wal_write"),
		postgresStatColumnExpr(columns, "fsyncs", "COALESCE(SUM(fsyncs), 0)::bigint", "0::bigint", "wal_sync"),
		postgresStatColumnExpr(columns, "write_time", "COALESCE(SUM(write_time), 0)::float8", "0::float8", "wal_write_ms"),
		postgresStatColumnExpr(columns, "fsync_time", "COALESCE(SUM(fsync_time), 0)::float8", "0::float8", "wal_sync_ms"),
	}
}

func postgresWALIOStats(ctx context.Context, db *gorm.DB) (postgresWALIOStat, bool) {
	var stat postgresWALIOStat
	columns, err := postgresRelationColumns(ctx, db, "pg_catalog.pg_stat_io")
	if err != nil || len(columns) == 0 || !columns["object"] {
		return stat, false
	}
	query := "SELECT " + strings.Join(postgresWALIOSelectList(columns), ", ") + " FROM pg_catalog.pg_stat_io WHERE object = 'wal'"
	if err := db.WithContext(ctx).Raw(query).Scan(&stat).Error; err != nil {
		return stat, false
	}
	return stat, true
}

func postgresWALStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		WalRecords int64   `gorm:"column:wal_records"`
		WalFPI     int64   `gorm:"column:wal_fpi"`
		WalBytes   float64 `gorm:"column:wal_bytes"`
		WalBuffers int64   `gorm:"column:wal_buffers_full"`
		WalWrite   int64   `gorm:"column:wal_write"`
		WalSync    int64   `gorm:"column:wal_sync"`
		WalWriteMS float64 `gorm:"column:wal_write_ms"`
		WalSyncMS  float64 `gorm:"column:wal_sync_ms"`
	}
	columns, err := postgresRelationColumns(ctx, db, "pg_catalog.pg_stat_wal")
	if err != nil {
		return postgresUnavailable(err)
	}
	if len(columns) == 0 {
		return postgresUnavailableMessage("pg_stat_wal is unavailable")
	}
	var stat row
	query := "SELECT " + strings.Join(postgresWALSelectList(columns), ", ") + " FROM pg_catalog.pg_stat_wal"
	err = db.WithContext(ctx).Raw(query).Scan(&stat).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	if !columns["wal_write"] || !columns["wal_sync"] || !columns["wal_write_time"] || !columns["wal_sync_time"] {
		if ioStat, ok := postgresWALIOStats(ctx, db); ok {
			if !columns["wal_write"] {
				stat.WalWrite = ioStat.WalWrite
			}
			if !columns["wal_sync"] {
				stat.WalSync = ioStat.WalSync
			}
			if !columns["wal_write_time"] {
				stat.WalWriteMS = ioStat.WalWriteMS
			}
			if !columns["wal_sync_time"] {
				stat.WalSyncMS = ioStat.WalSyncMS
			}
		}
	}
	return ginLikeMap{"available": true, "wal_records": stat.WalRecords, "wal_fpi": stat.WalFPI, "wal_bytes": stat.WalBytes, "wal_buffers_full": stat.WalBuffers, "wal_write": stat.WalWrite, "wal_sync": stat.WalSync, "wal_write_ms": stat.WalWriteMS, "wal_sync_ms": stat.WalSyncMS}
}

func postgresCheckpointerStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		NumTimed       int64   `gorm:"column:num_timed"`
		NumRequested   int64   `gorm:"column:num_requested"`
		WriteTimeMS    float64 `gorm:"column:write_time_ms"`
		SyncTimeMS     float64 `gorm:"column:sync_time_ms"`
		BuffersWritten int64   `gorm:"column:buffers_written"`
	}
	columns, err := postgresRelationColumns(ctx, db, "pg_catalog.pg_stat_checkpointer")
	if err != nil {
		return postgresUnavailable(err)
	}
	if len(columns) == 0 {
		return postgresBGWriterStats(ctx, db)
	}
	var stat row
	query := "SELECT " + strings.Join(postgresCheckpointerSelectList(columns), ", ") + " FROM pg_catalog.pg_stat_checkpointer"
	err = db.WithContext(ctx).Raw(query).Scan(&stat).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	return ginLikeMap{"available": true, "source": "pg_stat_checkpointer", "num_timed": stat.NumTimed, "num_requested": stat.NumRequested, "write_time_ms": stat.WriteTimeMS, "sync_time_ms": stat.SyncTimeMS, "buffers_written": stat.BuffersWritten}
}

func postgresCheckpointerSelectList(columns map[string]bool) []string {
	return []string{
		postgresStatColumnExpr(columns, "num_timed", "num_timed", "0::bigint", "num_timed"),
		postgresStatColumnExpr(columns, "num_requested", "num_requested", "0::bigint", "num_requested"),
		postgresStatColumnExpr(columns, "write_time", "write_time", "0::float8", "write_time_ms"),
		postgresStatColumnExpr(columns, "sync_time", "sync_time", "0::float8", "sync_time_ms"),
		postgresStatColumnExpr(columns, "buffers_written", "buffers_written", "0::bigint", "buffers_written"),
	}
}

func postgresBGWriterStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	type bgRow struct {
		CheckpointsTimed    int64   `gorm:"column:checkpoints_timed"`
		CheckpointsReq      int64   `gorm:"column:checkpoints_req"`
		CheckpointWriteMS   float64 `gorm:"column:checkpoint_write_time"`
		CheckpointSyncMS    float64 `gorm:"column:checkpoint_sync_time"`
		BuffersCheckpoint   int64   `gorm:"column:buffers_checkpoint"`
		BuffersClean        int64   `gorm:"column:buffers_clean"`
		MaxWrittenClean     int64   `gorm:"column:maxwritten_clean"`
		BuffersBackend      int64   `gorm:"column:buffers_backend"`
		BuffersBackendFsync int64   `gorm:"column:buffers_backend_fsync"`
	}
	var bg bgRow
	if err := db.WithContext(ctx).Raw(`SELECT checkpoints_timed, checkpoints_req, checkpoint_write_time, checkpoint_sync_time, buffers_checkpoint, buffers_clean, maxwritten_clean, buffers_backend, buffers_backend_fsync FROM pg_stat_bgwriter`).Scan(&bg).Error; err != nil {
		return postgresUnavailable(err)
	}
	return ginLikeMap{
		"available":             true,
		"source":                "pg_stat_bgwriter",
		"num_timed":             bg.CheckpointsTimed,
		"num_requested":         bg.CheckpointsReq,
		"write_time_ms":         bg.CheckpointWriteMS,
		"sync_time_ms":          bg.CheckpointSyncMS,
		"buffers_written":       bg.BuffersCheckpoint,
		"buffers_clean":         bg.BuffersClean,
		"maxwritten_clean":      bg.MaxWrittenClean,
		"buffers_backend":       bg.BuffersBackend,
		"buffers_backend_fsync": bg.BuffersBackendFsync,
	}
}

func postgresSharedPreloadLibraryEnabled(ctx context.Context, db *gorm.DB, library string) (bool, error) {
	var setting sql.NullString
	err := db.WithContext(ctx).Raw(`SELECT current_setting('shared_preload_libraries', true)`).Scan(&setting).Error
	if err != nil {
		return false, err
	}
	return setting.Valid && postgresLibraryListContains(setting.String, library), nil
}

func postgresLibraryListContains(setting string, library string) bool {
	for item := range strings.SplitSeq(setting, ",") {
		item = strings.TrimSpace(strings.Trim(item, `"'`))
		if idx := strings.LastIndexAny(item, `/\`); idx >= 0 {
			item = item[idx+1:]
		}
		item = strings.TrimSuffix(strings.TrimSuffix(item, ".so"), ".dll")
		if strings.EqualFold(item, library) {
			return true
		}
	}
	return false
}

func postgresTopSQLTimingExpr(columns map[string]bool, preferred string, legacy string, alias string) string {
	if columns[preferred] {
		return preferred + " AS " + alias
	}
	if columns[legacy] {
		return legacy + " AS " + alias
	}
	return "0::float8 AS " + alias
}

func postgresTopSQLSelectList(columns map[string]bool) []string {
	return []string{
		postgresStatColumnExpr(columns, "queryid", "queryid::text", "''::text", "queryid"),
		postgresStatColumnExpr(columns, "calls", "calls", "0::bigint", "calls"),
		postgresTopSQLTimingExpr(columns, "total_exec_time", "total_time", "total_exec_time_ms"),
		postgresTopSQLTimingExpr(columns, "mean_exec_time", "mean_time", "mean_exec_time_ms"),
		postgresStatColumnExpr(columns, "rows", "rows", "0::bigint", "rows"),
		postgresStatColumnExpr(columns, "shared_blks_hit", "shared_blks_hit", "0::bigint", "shared_blks_hit"),
		postgresStatColumnExpr(columns, "shared_blks_read", "shared_blks_read", "0::bigint", "shared_blks_read"),
		postgresStatColumnExpr(columns, "query", `LEFT(regexp_replace(query, '\s+', ' ', 'g'), 300)`, "''::text", "query"),
	}
}

func postgresTopSQL(ctx context.Context, db *gorm.DB) ginLikeMap {
	type row struct {
		QueryID         string  `gorm:"column:queryid"`
		Calls           int64   `gorm:"column:calls"`
		TotalExecTimeMS float64 `gorm:"column:total_exec_time_ms"`
		MeanExecTimeMS  float64 `gorm:"column:mean_exec_time_ms"`
		Rows            int64   `gorm:"column:rows"`
		SharedBlksHit   int64   `gorm:"column:shared_blks_hit"`
		SharedBlksRead  int64   `gorm:"column:shared_blks_read"`
		Query           string  `gorm:"column:query"`
	}
	loaded, err := postgresSharedPreloadLibraryEnabled(ctx, db, "pg_stat_statements")
	if err != nil {
		return postgresUnavailable(err)
	}
	if !loaded {
		return postgresUnavailableMessage("pg_stat_statements is not loaded via shared_preload_libraries")
	}
	columns, err := postgresRelationColumns(ctx, db, "pg_stat_statements")
	if err != nil {
		return postgresUnavailable(err)
	}
	if len(columns) == 0 {
		return postgresUnavailableMessage("pg_stat_statements extension is not installed")
	}
	var rows []row
	query := "SELECT " + strings.Join(postgresTopSQLSelectList(columns), ", ") + " FROM pg_stat_statements"
	if columns["dbid"] {
		query += " WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())"
	}
	query += " ORDER BY total_exec_time_ms DESC LIMIT 10"
	err = db.WithContext(ctx).Raw(query).Scan(&rows).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	items := make([]ginLikeMap, 0, len(rows))
	for _, row := range rows {
		items = append(items, ginLikeMap{
			"queryid":            row.QueryID,
			"calls":              row.Calls,
			"total_exec_time_ms": row.TotalExecTimeMS,
			"mean_exec_time_ms":  row.MeanExecTimeMS,
			"rows":               row.Rows,
			"shared_blks_hit":    row.SharedBlksHit,
			"shared_blks_read":   row.SharedBlksRead,
			"query":              sanitizeSQL(row.Query),
		})
	}
	return ginLikeMap{"available": true, "items": items}
}

func postgresUnavailable(err error) ginLikeMap {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return postgresUnavailableMessage(message)
}

func postgresUnavailableMessage(message string) ginLikeMap {
	return ginLikeMap{"available": false, "message": message}
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func timePtrRFC3339(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
