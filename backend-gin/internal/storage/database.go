package storage

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yuem-go/backend-gin/internal/config"
)

func OpenDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, nil
	}

	createBatchSize := max(cfg.CreateBatchSize, 0)
	gormConfig := &gorm.Config{
		Logger:                 gormLogger(cfg),
		PrepareStmt:            cfg.PrepareStmt,
		CreateBatchSize:        createBatchSize,
		SkipDefaultTransaction: cfg.SkipDefaultTransaction,
	}

	switch cfg.Driver {
	case "mysql":
		return openAndConfigure(mysql.Open(mysqlDSN(cfg)), gormConfig, cfg)
	case "postgres":
		return openAndConfigure(postgres.Open(postgresDSN(cfg)), gormConfig, cfg)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}

func openAndConfigure(dialector gorm.Dialector, gormConfig *gorm.Config, cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}
	sqlDB.SetMaxOpenConns(poolSize)
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = max(1, poolSize/2)
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	if cfg.IdleTimeoutMS > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.IdleTimeoutMS) * time.Millisecond)
	}
	if cfg.ConnMaxLifetimeMS > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMS) * time.Millisecond)
	}
	if cfg.AutoMigrate {
		if err := AutoMigrate(db); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
	}
	return db, nil
}

func gormLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent", "off", "none":
		return logger.Silent
	case "info":
		return logger.Info
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	default:
		return logger.Warn
	}
}

func gormLogger(cfg config.DatabaseConfig) logger.Interface {
	return gormLoggerWithWriter(cfg, os.Stdout)
}

func gormLoggerWithWriter(cfg config.DatabaseConfig, writer io.Writer) logger.Interface {
	return logger.New(
		log.New(writer, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogLevel(cfg.LogLevel),
			IgnoreRecordNotFoundError: cfg.IgnoreRecordNotFound,
			Colorful:                  true,
		},
	)
}

func mysqlDSN(cfg config.DatabaseConfig) string {
	parsed, err := url.Parse(cfg.URL)
	if err != nil || parsed.Scheme == "" {
		return cfg.URL
	}
	user := parsed.User.Username()
	password, _ := parsed.User.Password()
	host := parsed.Host
	database := strings.TrimPrefix(parsed.Path, "/")
	query := parsed.Query()
	if query.Get("charset") == "" && cfg.Charset != "" {
		query.Set("charset", cfg.Charset)
	}
	if query.Get("charset") == "" {
		query.Set("charset", "utf8mb4")
	}
	if query.Get("parseTime") == "" {
		query.Set("parseTime", "true")
	}
	if query.Get("timeout") == "" && cfg.ConnectionTimeoutMS > 0 {
		query.Set("timeout", durationParam(cfg.ConnectionTimeoutMS))
	}
	if query.Get("loc") == "" {
		query.Set("loc", "Local")
	}
	if query.Get("time_zone") == "" && cfg.Timezone != "" {
		query.Set("time_zone", cfg.Timezone)
	}

	auth := user
	if password != "" {
		auth += ":" + password
	}
	return fmt.Sprintf("%s@tcp(%s)/%s?%s", auth, host, database, query.Encode())
}

func postgresDSN(cfg config.DatabaseConfig) string {
	parsed, err := url.Parse(cfg.URL)
	if err != nil || parsed.Scheme == "" {
		return cfg.URL
	}
	query := parsed.Query()
	if schema := firstNonEmpty(query.Get("schema"), cfg.Schema); schema != "" {
		query.Del("schema")
		if query.Get("search_path") == "" {
			query.Set("search_path", schema)
		}
	}
	if query.Get("sslmode") == "" && cfg.SSLMode != "" {
		query.Set("sslmode", cfg.SSLMode)
	}
	if query.Get("connect_timeout") == "" && cfg.ConnectionTimeoutMS > 0 {
		query.Set("connect_timeout", secondsParam(cfg.ConnectionTimeoutMS))
	}
	if query.Get("statement_timeout") == "" && cfg.StatementTimeoutMS > 0 {
		query.Set("statement_timeout", fmt.Sprintf("%d", cfg.StatementTimeoutMS))
	}
	if query.Get("lock_timeout") == "" && cfg.LockTimeoutMS > 0 {
		query.Set("lock_timeout", fmt.Sprintf("%d", cfg.LockTimeoutMS))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func durationParam(ms int) string {
	if ms <= 0 {
		return "0s"
	}
	return fmt.Sprintf("%dms", ms)
}

func secondsParam(ms int) string {
	seconds := ms / 1000
	if ms%1000 != 0 {
		seconds++
	}
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
