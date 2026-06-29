package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type traceErrLogger struct {
	err error
}

func (l *traceErrLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *traceErrLogger) Info(context.Context, string, ...any) {}

func (l *traceErrLogger) Warn(context.Context, string, ...any) {}

func (l *traceErrLogger) Error(context.Context, string, ...any) {}

func (l *traceErrLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), err error) {
	if fc != nil {
		_, _ = fc()
	}
	l.err = err
}

func TestObservabilityGormLoggerIgnoresRecordNotFoundWhenConfigured(t *testing.T) {
	next := &traceErrLogger{}
	wrapped := NewObservabilityGormLogger(next, nil, true)

	wrapped.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, gorm.ErrRecordNotFound)

	if next.err != nil {
		t.Fatalf("record-not-found error was forwarded: %v", next.err)
	}
}

func TestObservabilityGormLoggerCanKeepRecordNotFoundWhenConfigured(t *testing.T) {
	next := &traceErrLogger{}
	wrapped := NewObservabilityGormLogger(next, nil, false)

	wrapped.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, gorm.ErrRecordNotFound)

	if !errors.Is(next.err, gorm.ErrRecordNotFound) {
		t.Fatalf("record-not-found error not forwarded: %v", next.err)
	}
}
