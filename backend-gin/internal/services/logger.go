package services

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"yuem-go/backend-gin/internal/config"
)

func NewLogger(cfg config.LogConfig) (*zap.Logger, func(), error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.Lock(os.Stderr),
		levelEnabler(parseZapLevel(cfg.Level, zapcore.InfoLevel)),
	)
	cores := []zapcore.Core{consoleCore}
	closers := []func(){}
	var setupErr error

	if cfg.FileEnabled {
		writer := NewDailyErrorLogWriter(cfg.FileDir, cfg.FileRetentionDays)
		if err := writer.EnsureReady(); err != nil {
			setupErr = err
		} else {
			fileCore := zapcore.NewCore(
				zapcore.NewJSONEncoder(encoderConfig),
				zapcore.AddSync(writer),
				levelEnabler(zapcore.ErrorLevel),
			)
			cores = append(cores, fileCore)
			closers = append(closers, func() { _ = writer.Close() })
		}
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	cleanup := func() {
		_ = logger.Sync()
		for _, closeFn := range closers {
			closeFn()
		}
	}
	return logger, cleanup, setupErr
}

func parseZapLevel(value string, fallback zapcore.Level) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return fallback
	}
}

func levelEnabler(min zapcore.Level) zap.LevelEnablerFunc {
	return func(level zapcore.Level) bool {
		return level >= min
	}
}
