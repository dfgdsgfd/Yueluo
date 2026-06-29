package storage

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yuem-go/backend-gin/internal/config"
)

func TestGormLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  logger.LogLevel
	}{
		{input: "silent", want: logger.Silent},
		{input: "off", want: logger.Silent},
		{input: "info", want: logger.Info},
		{input: "error", want: logger.Error},
		{input: "warn", want: logger.Warn},
		{input: "", want: logger.Warn},
		{input: "unexpected", want: logger.Warn},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := gormLogLevel(tt.input); got != tt.want {
				t.Fatalf("gormLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGormLoggerIgnoresRecordNotFoundWhenConfigured(t *testing.T) {
	var buf bytes.Buffer
	gormLog := gormLoggerWithWriter(config.DatabaseConfig{
		LogLevel:             "error",
		IgnoreRecordNotFound: true,
	}, &buf)

	gormLog.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, gorm.ErrRecordNotFound)

	if buf.Len() != 0 {
		t.Fatalf("record-not-found log was not ignored: %q", buf.String())
	}
}

func TestGormLoggerCanLogRecordNotFoundWhenConfigured(t *testing.T) {
	var buf bytes.Buffer
	gormLog := gormLoggerWithWriter(config.DatabaseConfig{
		LogLevel:             "error",
		IgnoreRecordNotFound: false,
	}, &buf)

	gormLog.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, gorm.ErrRecordNotFound)

	if !strings.Contains(buf.String(), "record not found") {
		t.Fatalf("record-not-found log missing: %q", buf.String())
	}
}
