package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestDatabaseVacuumConfigDefaultsAndPersistence(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingsService(nil, nil)

	cfg := ReadDatabaseVacuumConfig(settings)
	if cfg.Enabled || cfg.IntervalHours != 24 || len(cfg.Tables) != 0 {
		t.Fatalf("default vacuum config = %#v, want disabled with 24h interval and no tables", cfg)
	}

	nextRun := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	ok := SaveDatabaseVacuumConfig(ctx, settings, DatabaseVacuumConfig{
		Enabled:       true,
		Tables:        []string{" posts ", "users", "posts", ""},
		IntervalHours: 48,
		NextRunAt:     nextRun,
	})
	if !ok {
		t.Fatalf("SaveDatabaseVacuumConfig() = false, want true")
	}
	cfg = ReadDatabaseVacuumConfig(settings)
	if !cfg.Enabled || cfg.IntervalHours != 48 || cfg.NextRunAt != nextRun {
		t.Fatalf("saved vacuum config = %#v, want enabled 48h with next run", cfg)
	}
	if want := []string{"posts", "users"}; !reflect.DeepEqual(cfg.Tables, want) {
		t.Fatalf("saved vacuum tables = %#v, want %#v", cfg.Tables, want)
	}
}

func TestDatabaseVacuumConfigSetsNextRunWhenEnabled(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingsService(nil, nil)

	ok := SaveDatabaseVacuumConfig(ctx, settings, DatabaseVacuumConfig{
		Enabled:       true,
		Tables:        []string{"posts"},
		IntervalHours: 1,
	})
	if !ok {
		t.Fatalf("SaveDatabaseVacuumConfig() = false, want true")
	}
	cfg := ReadDatabaseVacuumConfig(settings)
	if cfg.NextRunAt == "" {
		t.Fatalf("NextRunAt is empty, want automatic schedule")
	}
	if _, err := time.Parse(time.RFC3339Nano, cfg.NextRunAt); err != nil {
		t.Fatalf("NextRunAt = %q is not RFC3339Nano: %v", cfg.NextRunAt, err)
	}
}

func TestVacuumSafetyHelpers(t *testing.T) {
	if got := normalizeVacuumTables([]string{" posts ", "users", "posts", ""}); !reflect.DeepEqual(got, []string{"posts", "users"}) {
		t.Fatalf("normalizeVacuumTables() = %#v, want posts/users", got)
	}
	if got := normalizeVacuumInterval(0); got != 24 {
		t.Fatalf("normalizeVacuumInterval(0) = %d, want 24", got)
	}
	if got := normalizeVacuumInterval(24 * 31); got != 24*30 {
		t.Fatalf("normalizeVacuumInterval(max) = %d, want 720", got)
	}
	if got := quotePostgresIdent(`odd"name`); got != `"odd""name"` {
		t.Fatalf("quotePostgresIdent() = %q, want escaped identifier", got)
	}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	if !vacuumDue("", now) || !vacuumDue("not-a-date", now) || !vacuumDue(now.Add(-time.Minute).Format(time.RFC3339Nano), now) {
		t.Fatalf("vacuumDue() should treat empty, invalid, and past values as due")
	}
	if vacuumDue(now.Add(time.Hour).Format(time.RFC3339Nano), now) {
		t.Fatalf("vacuumDue() = true for future next_run_at, want false")
	}
}

func TestValidatePostgresVacuumTablesRejectsUnsupportedDriver(t *testing.T) {
	_, err := ValidatePostgresVacuumTables(context.Background(), nil, []string{"posts"})
	if !errors.Is(err, ErrUnsupportedVacuumDriver) {
		t.Fatalf("ValidatePostgresVacuumTables() error = %v, want ErrUnsupportedVacuumDriver", err)
	}
}
