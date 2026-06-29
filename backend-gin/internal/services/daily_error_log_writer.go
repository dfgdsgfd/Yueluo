package services

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DailyErrorLogWriter struct {
	mu            sync.Mutex
	dir           string
	retentionDays int
	now           func() time.Time
	fileDate      string
	file          *os.File
	lastCleanup   time.Time
}

func NewDailyErrorLogWriter(dir string, retentionDays int) *DailyErrorLogWriter {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "logs"
	}
	dir = resolveLogDir(dir)
	if retentionDays <= 0 {
		retentionDays = 7
	}
	return &DailyErrorLogWriter{dir: dir, retentionDays: retentionDays, now: time.Now}
}

func (w *DailyErrorLogWriter) EnsureReady() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return err
	}
	w.cleanupLocked(w.now())
	return nil
}

func (w *DailyErrorLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := w.currentFileLocked()
	if err != nil {
		return 0, err
	}
	return file.Write(p)
}

func (w *DailyErrorLogWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *DailyErrorLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.fileDate = ""
	return err
}

func (w *DailyErrorLogWriter) currentFileLocked() (*os.File, error) {
	now := w.now()
	date := now.Format("2006-01-02")
	if w.file != nil && w.fileDate == date {
		w.cleanupLocked(now)
		return w.file, nil
	}
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return nil, err
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	path := filepath.Join(w.dir, date+"-error.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	w.file = file
	w.fileDate = date
	w.cleanupLocked(now)
	return file, nil
}

func (w *DailyErrorLogWriter) cleanupLocked(now time.Time) {
	if !w.lastCleanup.IsZero() && now.Sub(w.lastCleanup) < 24*time.Hour {
		return
	}
	w.lastCleanup = now
	if w.retentionDays <= 0 {
		return
	}
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	cutoff := dateOnlyForLog(now).AddDate(0, 0, -w.retentionDays+1)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, "-error.log") {
			continue
		}
		dateText := strings.TrimSuffix(name, "-error.log")
		date, err := time.ParseInLocation("2006-01-02", dateText, now.Location())
		if err != nil || !date.Before(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(w.dir, name))
	}
}

func dateOnlyForLog(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func resolveLogDir(dir string) string {
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	executable, err := os.Executable()
	if err != nil {
		return filepath.Clean(dir)
	}
	return filepath.Join(filepath.Dir(executable), dir)
}
