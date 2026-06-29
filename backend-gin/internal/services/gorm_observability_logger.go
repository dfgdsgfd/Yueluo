package services

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type observabilityGormLogger struct {
	next                 gormlogger.Interface
	observe              *ObservabilityService
	ignoreRecordNotFound bool
}

func NewObservabilityGormLogger(next gormlogger.Interface, observe *ObservabilityService, ignoreRecordNotFound ...bool) gormlogger.Interface {
	if next == nil {
		next = gormlogger.Default
	}
	ignore := false
	if len(ignoreRecordNotFound) > 0 {
		ignore = ignoreRecordNotFound[0]
	}
	return observabilityGormLogger{next: next, observe: observe, ignoreRecordNotFound: ignore}
}

func (l observabilityGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return observabilityGormLogger{
		next:                 l.next.LogMode(level),
		observe:              l.observe,
		ignoreRecordNotFound: l.ignoreRecordNotFound,
	}
}

func (l observabilityGormLogger) Info(ctx context.Context, message string, args ...any) {
	l.next.Info(ctx, message, args...)
}

func (l observabilityGormLogger) Warn(ctx context.Context, message string, args ...any) {
	l.next.Warn(ctx, message, args...)
}

func (l observabilityGormLogger) Error(ctx context.Context, message string, args ...any) {
	l.next.Error(ctx, message, args...)
}

func (l observabilityGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	logErr := err
	if l.ignoreRecordNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
		logErr = nil
	}
	elapsed := time.Since(begin)
	sql := ""
	rows := int64(-1)
	if fc != nil {
		sql, rows = fc()
	}
	l.next.Trace(ctx, begin, func() (string, int64) { return sql, rows }, logErr)
	if l.observe == nil || l.observe.cfg.SlowQueryThreshold <= 0 || elapsed < l.observe.cfg.SlowQueryThreshold {
		return
	}
	errText := ""
	if logErr != nil {
		errText = logErr.Error()
	}
	l.observe.RecordSlowQuery(ctx, SlowQueryMetric{
		SQL:       sql,
		LatencyMS: elapsed.Milliseconds(),
		Rows:      rows,
		Error:     errText,
		CreatedAt: time.Now(),
	})
}
