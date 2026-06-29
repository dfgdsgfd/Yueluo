package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDailyErrorLogWriterRotatesAndPrunesOldFiles(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "2026-06-20-error.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	keepPath := filepath.Join(dir, "2026-06-22-error.log")
	if err := os.WriteFile(keepPath, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	otherPath := filepath.Join(dir, "manual.log")
	if err := os.WriteFile(otherPath, []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := NewDailyErrorLogWriter(dir, 7)
	writer.now = func() time.Time {
		return time.Date(2026, 6, 28, 13, 25, 51, 0, time.Local)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte("boom\n")); err != nil {
		t.Fatalf("write log: %v", err)
	}

	todayPath := filepath.Join(dir, "2026-06-28-error.log")
	data, err := os.ReadFile(todayPath)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}
	if !strings.Contains(string(data), "boom") {
		t.Fatalf("current log missing entry: %q", string(data))
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old log still exists or unexpected stat error: %v", err)
	}
	for _, path := range []string{keepPath, otherPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}
