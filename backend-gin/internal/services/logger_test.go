package services

import (
	"os"
	"path/filepath"
	"testing"

	"yuem-go/backend-gin/internal/config"
)

func TestNewLoggerKeepsConsoleWhenFileLoggerFails(t *testing.T) {
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger, cleanup, err := NewLogger(config.LogConfig{
		Level:             "error",
		FileEnabled:       true,
		FileDir:           filepath.Join(blockingFile, "logs"),
		FileRetentionDays: 7,
	})
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatal("NewLogger() error = nil, want file setup error")
	}
	if logger == nil {
		t.Fatal("NewLogger() returned nil logger when file setup failed")
	}
}
