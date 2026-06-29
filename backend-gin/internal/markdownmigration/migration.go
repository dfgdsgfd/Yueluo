package markdownmigration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/contentformat"
)

type Target struct {
	Table    string `json:"table"`
	IDColumn string `json:"idColumn"`
	Column   string `json:"column"`
	Nullable bool   `json:"nullable"`
}

type Options struct {
	Apply       bool
	Backup      bool
	BackupTable string
	BatchSize   int
	Limit       int
	Sample      int
	Tables      []string
}

type Result struct {
	StartedAt   string         `json:"started_at"`
	FinishedAt  string         `json:"finished_at"`
	Apply       bool           `json:"apply"`
	Backup      bool           `json:"backup"`
	BackupTable string         `json:"backup_table,omitempty"`
	Targets     []TargetResult `json:"targets"`
}

type TargetResult struct {
	Table     string   `json:"table"`
	Column    string   `json:"column"`
	Scanned   int      `json:"scanned"`
	Changed   int      `json:"changed"`
	Updated   int      `json:"updated"`
	BackedUp  int      `json:"backed_up"`
	Samples   []Sample `json:"samples,omitempty"`
	ErrorText string   `json:"error,omitempty"`
}

type Sample struct {
	ID     any    `json:"id"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type rowValue struct {
	ID    int64
	Value string
}

func DefaultTargets() []Target {
	return []Target{
		{Table: "posts", IDColumn: "id", Column: "content"},
		{Table: "comments", IDColumn: "id", Column: "content"},
		{Table: "im_messages", IDColumn: "id", Column: "content"},
		{Table: "users", IDColumn: "id", Column: "bio", Nullable: true},
		{Table: "feedback", IDColumn: "id", Column: "content"},
		{Table: "reports", IDColumn: "id", Column: "description", Nullable: true},
		{Table: "audit", IDColumn: "id", Column: "content"},
	}
}

func Run(ctx context.Context, db *gorm.DB, opts Options) (Result, error) {
	result := Result{
		StartedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Apply:       opts.Apply,
		Backup:      opts.Backup,
		BackupTable: backupTableName(opts.BackupTable),
	}
	if db == nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return result, fmt.Errorf("database is not configured")
	}
	opts = normalizeOptions(opts)
	targets := filterTargets(DefaultTargets(), opts.Tables)
	if opts.Apply && opts.Backup {
		if err := ensureBackupTable(ctx, db, opts.BackupTable); err != nil {
			result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return result, err
		}
	}
	for _, target := range targets {
		targetResult := migrateTarget(ctx, db, target, opts)
		result.Targets = append(result.Targets, targetResult)
	}
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func normalizeOptions(opts Options) Options {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 200
	}
	if opts.Sample < 0 {
		opts.Sample = 0
	}
	if strings.TrimSpace(opts.BackupTable) == "" {
		opts.BackupTable = "markdown_migration_backups"
	}
	return opts
}

func filterTargets(targets []Target, tables []string) []Target {
	allowed := map[string]bool{}
	for _, table := range tables {
		table = strings.TrimSpace(table)
		if table != "" {
			allowed[table] = true
		}
	}
	if len(allowed) == 0 {
		return targets
	}
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		if allowed[target.Table] || allowed[target.Table+"."+target.Column] {
			out = append(out, target)
		}
	}
	return out
}

func migrateTarget(ctx context.Context, db *gorm.DB, target Target, opts Options) TargetResult {
	result := TargetResult{Table: target.Table, Column: target.Column}
	var afterID int64
	remaining := opts.Limit
	for {
		limit := opts.BatchSize
		if remaining > 0 && remaining < limit {
			limit = remaining
		}
		rows, err := fetchRows(ctx, db, target, limit, afterID)
		if err != nil {
			result.ErrorText = err.Error()
			return result
		}
		if len(rows) == 0 {
			return result
		}
		for _, row := range rows {
			afterID = row.ID
			result.Scanned++
			next := contentformat.SanitizeMarkdown(row.Value)
			if next == row.Value {
				continue
			}
			result.Changed++
			if len(result.Samples) < opts.Sample {
				result.Samples = append(result.Samples, Sample{ID: row.ID, Before: row.Value, After: next})
			}
			if opts.Apply {
				if opts.Backup {
					if err := backupRow(ctx, db, opts.BackupTable, target, row); err != nil {
						result.ErrorText = err.Error()
						return result
					}
					result.BackedUp++
				}
				if err := updateRow(ctx, db, target, row.ID, next, target.Nullable); err != nil {
					result.ErrorText = err.Error()
					return result
				}
				result.Updated++
			}
		}
		if opts.Limit > 0 {
			remaining -= len(rows)
			if remaining <= 0 {
				return result
			}
		}
	}
}

func fetchRows(ctx context.Context, db *gorm.DB, target Target, limit int, afterID int64) ([]rowValue, error) {
	rows, err := db.WithContext(ctx).
		Table(target.Table).
		Select(quoteIdentifier(db, target.IDColumn)+" AS id, "+quoteIdentifier(db, target.Column)+" AS value").
		Where(quoteIdentifier(db, target.Column)+" IS NOT NULL AND "+quoteIdentifier(db, target.Column)+" <> ''").
		Where(quoteIdentifier(db, target.IDColumn)+" > ?", afterID).
		Order(quoteIdentifier(db, target.IDColumn) + " ASC").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rowValue{}
	for rows.Next() {
		var id int64
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			return nil, err
		}
		out = append(out, rowValue{ID: id, Value: value})
	}
	return out, rows.Err()
}

func ensureBackupTable(ctx context.Context, db *gorm.DB, table string) error {
	table = backupTableName(table)
	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id BIGINT PRIMARY KEY,
source_table TEXT NOT NULL,
source_column TEXT NOT NULL,
source_id TEXT NOT NULL,
original_value TEXT NOT NULL,
created_at TIMESTAMP NOT NULL
)`, quoteIdentifier(db, table))
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		sql = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id BIGINT AUTO_INCREMENT PRIMARY KEY,
source_table TEXT NOT NULL,
source_column TEXT NOT NULL,
source_id TEXT NOT NULL,
original_value TEXT NOT NULL,
created_at TIMESTAMP NOT NULL
		)`, quoteIdentifier(db, table))
	} else if db.Dialector != nil && db.Dialector.Name() == "postgres" {
		sql = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id BIGSERIAL PRIMARY KEY,
source_table TEXT NOT NULL,
source_column TEXT NOT NULL,
source_id TEXT NOT NULL,
original_value TEXT NOT NULL,
created_at TIMESTAMP NOT NULL
)`, quoteIdentifier(db, table))
	} else if db.Dialector == nil || db.Dialector.Name() != "postgres" {
		sql = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id INTEGER PRIMARY KEY AUTOINCREMENT,
source_table TEXT NOT NULL,
source_column TEXT NOT NULL,
source_id TEXT NOT NULL,
original_value TEXT NOT NULL,
created_at TIMESTAMP NOT NULL
)`, quoteIdentifier(db, table))
	}
	return db.WithContext(ctx).Exec(sql).Error
}

func backupRow(ctx context.Context, db *gorm.DB, table string, target Target, row rowValue) error {
	payload := map[string]any{
		"source_table":   target.Table,
		"source_column":  target.Column,
		"source_id":      fmt.Sprint(row.ID),
		"original_value": row.Value,
		"created_at":     time.Now().UTC(),
	}
	return db.WithContext(ctx).Table(backupTableName(table)).Create(payload).Error
}

func updateRow(ctx context.Context, db *gorm.DB, target Target, id int64, value string, nullable bool) error {
	var next any = value
	if nullable && strings.TrimSpace(value) == "" {
		next = nil
	}
	return db.WithContext(ctx).
		Table(target.Table).
		Where(quoteIdentifier(db, target.IDColumn)+" = ?", id).
		Update(target.Column, next).Error
}

func backupTableName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "markdown_migration_backups"
	}
	return value
}

func quoteIdentifier(db *gorm.DB, value string) string {
	value = strings.ReplaceAll(value, `"`, `""`)
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + value + `"`
}

func PrintJSON(result Result) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}
