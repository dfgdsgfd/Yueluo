package services

import (
	"context"
	"database/sql"
	"strings"

	"gorm.io/gorm"
)

func postgresDiagnostics(ctx context.Context, db *gorm.DB, stats sql.DBStats) ginLikeMap {
	return ginLikeMap{
		"pool":              postgresPoolDiagnostics(stats),
		"wait_events":       postgresWaitEventStats(ctx, db),
		"blocking_locks":    postgresBlockingLockSamples(ctx, db),
		"long_transactions": postgresLongTransactionSamples(ctx, db),
	}
}

func postgresPoolDiagnostics(stats sql.DBStats) ginLikeMap {
	usageRatio := float64(0)
	if stats.MaxOpenConnections > 0 {
		usageRatio = float64(stats.InUse) / float64(stats.MaxOpenConnections)
	}
	waitAvgMS := float64(0)
	if stats.WaitCount > 0 {
		waitAvgMS = float64(stats.WaitDuration.Milliseconds()) / float64(stats.WaitCount)
	}
	pressure := "low"
	switch {
	case usageRatio >= 0.9 || waitAvgMS >= 100:
		pressure = "high"
	case usageRatio >= 0.75 || waitAvgMS >= 10 || stats.WaitCount > 0:
		pressure = "medium"
	}
	return ginLikeMap{
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"max_open":         stats.MaxOpenConnections,
		"wait_count":       stats.WaitCount,
		"wait_duration_ms": stats.WaitDuration.Milliseconds(),
		"wait_avg_ms":      waitAvgMS,
		"usage_ratio":      usageRatio,
		"pressure":         pressure,
	}
}

type postgresWaitEventRow struct {
	WaitEventType string  `gorm:"column:wait_event_type"`
	WaitEvent     string  `gorm:"column:wait_event"`
	Count         int64   `gorm:"column:count"`
	MaxWaitMS     float64 `gorm:"column:max_wait_ms"`
}

func postgresWaitEventStats(ctx context.Context, db *gorm.DB) ginLikeMap {
	columns, err := postgresRelationColumns(ctx, db, "pg_catalog.pg_stat_activity")
	if err != nil {
		return postgresUnavailable(err)
	}
	if len(columns) == 0 {
		return postgresUnavailableMessage("pg_stat_activity is unavailable")
	}
	selectList := postgresWaitEventSelectList(columns)
	waitTypeExpr := postgresActivityColumnExpr(columns, "wait_event_type", "COALESCE(wait_event_type, 'unknown')", "'unavailable'")
	waitEventExpr := postgresActivityColumnExpr(columns, "wait_event", "COALESCE(wait_event, 'unknown')", "'unavailable'")
	query := "SELECT " + strings.Join(selectList, ", ") + `
		FROM pg_stat_activity
		WHERE datname = current_database()
			AND ` + waitEventExpr + ` NOT IN ('', 'unknown', 'unavailable')
		GROUP BY ` + waitTypeExpr + `, ` + waitEventExpr + `
		ORDER BY count DESC, max_wait_ms DESC
		LIMIT 12`
	var rows []postgresWaitEventRow
	if err := db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return postgresUnavailable(err)
	}
	items := make([]ginLikeMap, 0, len(rows))
	for _, row := range rows {
		items = append(items, ginLikeMap{
			"wait_event_type": row.WaitEventType,
			"wait_event":      row.WaitEvent,
			"count":           row.Count,
			"max_wait_ms":     row.MaxWaitMS,
		})
	}
	return ginLikeMap{"available": true, "items": items}
}

func postgresWaitEventSelectList(columns map[string]bool) []string {
	waitTypeExpr := postgresActivityColumnExpr(columns, "wait_event_type", "COALESCE(wait_event_type, 'unknown')", "'unavailable'")
	waitEventExpr := postgresActivityColumnExpr(columns, "wait_event", "COALESCE(wait_event, 'unknown')", "'unavailable'")
	maxWaitExpr := postgresActivityColumnExpr(columns, "query_start", "COALESCE(MAX(EXTRACT(EPOCH FROM (now() - query_start)) * 1000), 0)", "0")
	return []string{
		waitTypeExpr + " AS wait_event_type",
		waitEventExpr + " AS wait_event",
		"COUNT(*) AS count",
		maxWaitExpr + " AS max_wait_ms",
	}
}

func postgresActivityColumnExpr(columns map[string]bool, name string, expression string, fallback string) string {
	if columns[name] {
		return expression
	}
	return fallback
}

type postgresBlockingLockRow struct {
	BlockedPID      int64   `gorm:"column:blocked_pid"`
	BlockedUser     string  `gorm:"column:blocked_user"`
	BlockedApp      string  `gorm:"column:blocked_app"`
	BlockedState    string  `gorm:"column:blocked_state"`
	WaitEventType   string  `gorm:"column:wait_event_type"`
	WaitEvent       string  `gorm:"column:wait_event"`
	WaitMS          float64 `gorm:"column:wait_ms"`
	BlockedQuery    string  `gorm:"column:blocked_query"`
	BlockerPID      int64   `gorm:"column:blocker_pid"`
	BlockerUser     string  `gorm:"column:blocker_user"`
	BlockerApp      string  `gorm:"column:blocker_app"`
	BlockerState    string  `gorm:"column:blocker_state"`
	BlockerQueryAge float64 `gorm:"column:blocker_query_age_ms"`
	BlockerQuery    string  `gorm:"column:blocker_query"`
}

func postgresBlockingLockSamples(ctx context.Context, db *gorm.DB) ginLikeMap {
	var rows []postgresBlockingLockRow
	err := db.WithContext(ctx).Raw(`
		SELECT
			blocked.pid AS blocked_pid,
			COALESCE(blocked.usename, '') AS blocked_user,
			COALESCE(blocked.application_name, '') AS blocked_app,
			COALESCE(blocked.state, '') AS blocked_state,
			COALESCE(blocked.wait_event_type, '') AS wait_event_type,
			COALESCE(blocked.wait_event, '') AS wait_event,
			COALESCE(EXTRACT(EPOCH FROM (now() - blocked.query_start)) * 1000, 0) AS wait_ms,
			LEFT(COALESCE(blocked.query, ''), 500) AS blocked_query,
			COALESCE(blocker.pid, 0) AS blocker_pid,
			COALESCE(blocker.usename, '') AS blocker_user,
			COALESCE(blocker.application_name, '') AS blocker_app,
			COALESCE(blocker.state, '') AS blocker_state,
			COALESCE(EXTRACT(EPOCH FROM (now() - blocker.query_start)) * 1000, 0) AS blocker_query_age_ms,
			LEFT(COALESCE(blocker.query, ''), 500) AS blocker_query
		FROM pg_stat_activity blocked
		JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) AS blocker_pid(pid) ON true
		LEFT JOIN pg_stat_activity blocker ON blocker.pid = blocker_pid.pid
		WHERE blocked.datname = current_database()
		ORDER BY wait_ms DESC
		LIMIT 10`).Scan(&rows).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	items := make([]ginLikeMap, 0, len(rows))
	for _, row := range rows {
		items = append(items, ginLikeMap{
			"blocked_pid":          row.BlockedPID,
			"blocked_user":         row.BlockedUser,
			"blocked_app":          row.BlockedApp,
			"blocked_state":        row.BlockedState,
			"wait_event_type":      row.WaitEventType,
			"wait_event":           row.WaitEvent,
			"wait_ms":              row.WaitMS,
			"blocked_query":        sanitizeSQL(row.BlockedQuery),
			"blocker_pid":          row.BlockerPID,
			"blocker_user":         row.BlockerUser,
			"blocker_app":          row.BlockerApp,
			"blocker_state":        row.BlockerState,
			"blocker_query_age_ms": row.BlockerQueryAge,
			"blocker_query":        sanitizeSQL(row.BlockerQuery),
		})
	}
	return ginLikeMap{"available": true, "items": items}
}

type postgresLongTransactionRow struct {
	PID           int64   `gorm:"column:pid"`
	User          string  `gorm:"column:usename"`
	App           string  `gorm:"column:application_name"`
	State         string  `gorm:"column:state"`
	WaitEventType string  `gorm:"column:wait_event_type"`
	WaitEvent     string  `gorm:"column:wait_event"`
	XactAgeMS     float64 `gorm:"column:xact_age_ms"`
	QueryAgeMS    float64 `gorm:"column:query_age_ms"`
	Query         string  `gorm:"column:query"`
}

func postgresLongTransactionSamples(ctx context.Context, db *gorm.DB) ginLikeMap {
	var rows []postgresLongTransactionRow
	err := db.WithContext(ctx).Raw(`
		SELECT
			pid,
			COALESCE(usename, '') AS usename,
			COALESCE(application_name, '') AS application_name,
			COALESCE(state, '') AS state,
			COALESCE(wait_event_type, '') AS wait_event_type,
			COALESCE(wait_event, '') AS wait_event,
			COALESCE(EXTRACT(EPOCH FROM (now() - xact_start)) * 1000, 0) AS xact_age_ms,
			COALESCE(EXTRACT(EPOCH FROM (now() - query_start)) * 1000, 0) AS query_age_ms,
			LEFT(COALESCE(query, ''), 500) AS query
		FROM pg_stat_activity
		WHERE datname = current_database()
			AND xact_start IS NOT NULL
			AND (state = 'idle in transaction' OR now() - xact_start > interval '5 minutes')
		ORDER BY xact_age_ms DESC
		LIMIT 10`).Scan(&rows).Error
	if err != nil {
		return postgresUnavailable(err)
	}
	items := make([]ginLikeMap, 0, len(rows))
	for _, row := range rows {
		items = append(items, ginLikeMap{
			"pid":             row.PID,
			"user":            row.User,
			"application":     row.App,
			"state":           row.State,
			"wait_event_type": row.WaitEventType,
			"wait_event":      row.WaitEvent,
			"xact_age_ms":     row.XactAgeMS,
			"query_age_ms":    row.QueryAgeMS,
			"query":           sanitizeSQL(row.Query),
		})
	}
	return ginLikeMap{"available": true, "items": items}
}
