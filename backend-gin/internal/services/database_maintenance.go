package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrUnsupportedVacuumDriver = errors.New("database vacuum analyze is only supported for PostgreSQL")

type DatabaseVacuumConfig struct {
	Enabled       bool     `json:"enabled"`
	Tables        []string `json:"tables"`
	IntervalHours int      `json:"interval_hours"`
	NextRunAt     string   `json:"next_run_at"`
	LastRunAt     string   `json:"last_run_at"`
	LastResult    any      `json:"last_result"`
}

type DatabaseVacuumResult struct {
	StartedAt  string                      `json:"started_at"`
	FinishedAt string                      `json:"finished_at"`
	DurationMS int64                       `json:"duration_ms"`
	Tables     []DatabaseVacuumTableResult `json:"tables"`
}

type DatabaseVacuumTableResult struct {
	Table      string `json:"table"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	DurationMS int64  `json:"duration_ms"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

type DatabaseMaintenanceService struct {
	db       *gorm.DB
	settings *SettingsService
	logger   *zap.Logger
	done     chan struct{}
	close    sync.Once
	running  atomic.Bool
}

func NewDatabaseMaintenanceService(db *gorm.DB, settings *SettingsService, logger *zap.Logger) *DatabaseMaintenanceService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DatabaseMaintenanceService{db: db, settings: settings, logger: logger, done: make(chan struct{})}
}

func (s *DatabaseMaintenanceService) Start() {
	if s == nil || s.db == nil || s.settings == nil {
		return
	}
	go s.loop()
}

func (s *DatabaseMaintenanceService) Close() {
	if s == nil {
		return
	}
	s.close.Do(func() {
		close(s.done)
	})
}

func (s *DatabaseMaintenanceService) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.runIfDue()
		}
	}
}

func (s *DatabaseMaintenanceService) runIfDue() {
	if s == nil || !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)
	cfg := ReadDatabaseVacuumConfig(s.settings)
	if !cfg.Enabled || len(cfg.Tables) == 0 || !vacuumDue(cfg.NextRunAt, time.Now()) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result, err := RunPostgresVacuumAnalyze(ctx, s.db, cfg.Tables)
	if err != nil {
		result = DatabaseVacuumResult{
			StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
			FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Tables: []DatabaseVacuumTableResult{{
				Status:  "error",
				Message: err.Error(),
			}},
		}
		s.logger.Warn("automatic vacuum analyze failed", zap.Error(err))
	}
	next := time.Now().UTC().Add(time.Duration(normalizeVacuumInterval(cfg.IntervalHours)) * time.Hour).Format(time.RFC3339Nano)
	_ = SaveDatabaseVacuumRun(ctx, s.settings, next, result)
}

func ReadDatabaseVacuumConfig(settings *SettingsService) DatabaseVacuumConfig {
	if settings == nil {
		return DatabaseVacuumConfig{IntervalHours: 24, Tables: []string{}}
	}
	return DatabaseVacuumConfig{
		Enabled:       settings.Bool("database_vacuum_enabled"),
		Tables:        normalizeVacuumTables(settings.StringArray("database_vacuum_tables")),
		IntervalHours: normalizeVacuumInterval(settings.Int("database_vacuum_interval_hours", 24)),
		NextRunAt:     strings.TrimSpace(settings.String("database_vacuum_next_run_at")),
		LastRunAt:     strings.TrimSpace(settings.String("database_vacuum_last_run_at")),
		LastResult:    decodeSettingRaw(settings.String("database_vacuum_last_result")),
	}
}

func SaveDatabaseVacuumConfig(ctx context.Context, settings *SettingsService, cfg DatabaseVacuumConfig) bool {
	if settings == nil {
		return false
	}
	cfg.Tables = normalizeVacuumTables(cfg.Tables)
	cfg.IntervalHours = normalizeVacuumInterval(cfg.IntervalHours)
	if cfg.Enabled && strings.TrimSpace(cfg.NextRunAt) == "" {
		cfg.NextRunAt = time.Now().UTC().Add(time.Duration(cfg.IntervalHours) * time.Hour).Format(time.RFC3339Nano)
	}
	updates := map[string]any{
		"database_vacuum_enabled":        cfg.Enabled,
		"database_vacuum_tables":         cfg.Tables,
		"database_vacuum_interval_hours": cfg.IntervalHours,
		"database_vacuum_next_run_at":    strings.TrimSpace(cfg.NextRunAt),
	}
	for key, value := range updates {
		if !settings.Set(ctx, key, value) {
			return false
		}
	}
	return true
}

func SaveDatabaseVacuumRun(ctx context.Context, settings *SettingsService, nextRunAt string, result DatabaseVacuumResult) bool {
	if settings == nil {
		return false
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for key, value := range map[string]any{
		"database_vacuum_last_run_at": now,
		"database_vacuum_next_run_at": strings.TrimSpace(nextRunAt),
		"database_vacuum_last_result": result,
	} {
		if !settings.Set(ctx, key, value) {
			return false
		}
	}
	return true
}

func RunPostgresVacuumAnalyze(ctx context.Context, db *gorm.DB, tables []string) (DatabaseVacuumResult, error) {
	started := time.Now()
	result := DatabaseVacuumResult{StartedAt: started.UTC().Format(time.RFC3339Nano)}
	validated, err := ValidatePostgresVacuumTables(ctx, db, tables)
	if err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		result.DurationMS = time.Since(started).Milliseconds()
		return result, err
	}
	for _, table := range validated {
		tableStarted := time.Now()
		item := DatabaseVacuumTableResult{Table: table, StartedAt: tableStarted.UTC().Format(time.RFC3339Nano)}
		err := db.WithContext(ctx).Exec("VACUUM ANALYZE " + quotePostgresIdent(table)).Error
		item.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		item.DurationMS = time.Since(tableStarted).Milliseconds()
		if err != nil {
			item.Status = "error"
			item.Message = err.Error()
			result.Tables = append(result.Tables, item)
			result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			result.DurationMS = time.Since(started).Milliseconds()
			return result, err
		}
		item.Status = "ok"
		result.Tables = append(result.Tables, item)
	}
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	result.DurationMS = time.Since(started).Milliseconds()
	return result, nil
}

func ValidatePostgresVacuumTables(ctx context.Context, db *gorm.DB, tables []string) ([]string, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil, ErrUnsupportedVacuumDriver
	}
	tables = normalizeVacuumTables(tables)
	if len(tables) == 0 {
		return nil, errors.New("at least one table is required")
	}
	available, err := PostgresCurrentSchemaTables(ctx, db)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, table := range available {
		allowed[table] = true
	}
	for _, table := range tables {
		if !allowed[table] {
			return nil, fmt.Errorf("table %q is not a base table in the current schema", table)
		}
	}
	return tables, nil
}

func PostgresCurrentSchemaTables(ctx context.Context, db *gorm.DB) ([]string, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return []string{}, ErrUnsupportedVacuumDriver
	}
	var rows []struct {
		Name string `gorm:"column:name"`
	}
	err := db.WithContext(ctx).Raw(`
		SELECT table_name AS name
		FROM information_schema.tables
		WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'
		ORDER BY table_name ASC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out, nil
}

func normalizeVacuumTables(tables []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tables))
	for _, table := range tables {
		table = strings.TrimSpace(table)
		if table == "" || seen[table] {
			continue
		}
		seen[table] = true
		out = append(out, table)
	}
	return out
}

func normalizeVacuumInterval(value int) int {
	if value <= 0 {
		return 24
	}
	if value > 24*30 {
		return 24 * 30
	}
	return value
}

func vacuumDue(raw string, now time.Time) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	next, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		next, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return true
	}
	return !next.After(now)
}

func quotePostgresIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
