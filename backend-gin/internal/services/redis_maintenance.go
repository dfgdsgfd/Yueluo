package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	redisMaintenanceLockKey       = "maintenance:redis:lock"
	redisMaintenanceLockTTL       = 15 * time.Minute
	redisMaintenanceInventoryKeys = 2000

	defaultRedisAccessLogRetentionHours = 24
	defaultRedisAccessLogMaxEntries     = 50000
	defaultRedisSystemLogMaxEntries     = 20000
	defaultRedisMetricsMaxEntries       = 50000
	defaultRedisQueueEventMaxEntries    = 20000
)

var ErrRedisMaintenanceBusy = errors.New("redis maintenance is already running")

type RedisQueueRetentionPolicy struct {
	CompletedRetentionHours int `json:"completed_retention_hours"`
	ArchivedRetentionHours  int `json:"archived_retention_hours"`
	ArchivedMaxTasks        int `json:"archived_max_tasks"`
}

type RedisMaintenanceConfig struct {
	Enabled                            bool                                 `json:"enabled"`
	IntervalMinutes                    int                                  `json:"interval_minutes"`
	AccessLogRetentionHours            int                                  `json:"access_log_retention_hours"`
	AccessLogMaxEntries                int                                  `json:"access_log_max_entries"`
	SystemLogRetentionHours            int                                  `json:"system_log_retention_hours"`
	SystemLogMaxEntries                int                                  `json:"system_log_max_entries"`
	MetricsRetentionHours              int                                  `json:"metrics_retention_hours"`
	MetricsMaxEntriesPerKey            int                                  `json:"metrics_max_entries_per_key"`
	QueueEventRetentionHours           int                                  `json:"queue_event_retention_hours"`
	QueueEventMaxEntries               int                                  `json:"queue_event_max_entries"`
	CompletedRetentionHours            int                                  `json:"completed_retention_hours"`
	ArchivedRetentionHours             int                                  `json:"archived_retention_hours"`
	ArchivedMaxTasksPerQueue           int                                  `json:"archived_max_tasks_per_queue"`
	MemoryWarningPercent               int                                  `json:"memory_warning_percent"`
	MemoryCriticalPercent              int                                  `json:"memory_critical_percent"`
	AccessTokenTTLSeconds              int                                  `json:"access_token_ttl_seconds"`
	RefreshTokenActiveTTLSeconds       int                                  `json:"refresh_token_active_ttl_seconds"`
	RefreshTokenRenewalIntervalSeconds int                                  `json:"refresh_token_renewal_interval_seconds"`
	RefreshTokenMode                   string                               `json:"refresh_token_mode"`
	RefreshTokenAutoRenewEnabled       bool                                 `json:"refresh_token_auto_renew_enabled"`
	SessionInactiveTTLSeconds          int                                  `json:"session_inactive_ttl_seconds"`
	UserActiveSessionLimit             int                                  `json:"user_active_session_limit"`
	QueueOverrides                     map[string]RedisQueueRetentionPolicy `json:"queue_overrides"`
	NextRunAt                          string                               `json:"next_run_at"`
	LastRunAt                          string                               `json:"last_run_at"`
	LastResult                         any                                  `json:"last_result"`
}

type RedisMaintenanceResult struct {
	StartedAt    string                        `json:"started_at"`
	FinishedAt   string                        `json:"finished_at"`
	DurationMS   int64                         `json:"duration_ms"`
	MemoryBefore int64                         `json:"memory_before_bytes"`
	MemoryAfter  int64                         `json:"memory_after_bytes"`
	MemoryFreed  int64                         `json:"memory_freed_bytes"`
	Categories   map[string]int64              `json:"categories"`
	QueueResults map[string]QueueCleanupResult `json:"queue_results,omitempty"`
	Warnings     []string                      `json:"warnings,omitempty"`
}

type QueueCleanupResult struct {
	Completed          int `json:"completed"`
	CompletedDeleted   int `json:"completed_deleted"`
	Archived           int `json:"archived"`
	ArchivedDeleted    int `json:"archived_deleted"`
	CompletedRetention int `json:"completed_retention_hours"`
	ArchivedRetention  int `json:"archived_retention_hours"`
	ArchivedMax        int `json:"archived_max_tasks"`
}

type RedisInventoryCategory struct {
	Keys                  int64 `json:"keys"`
	MemoryBytes           int64 `json:"memory_bytes"`
	NoTTLKeys             int64 `json:"no_ttl_keys"`
	ExpectedPermanentKeys int64 `json:"expected_permanent_keys"`
}

type RedisInventory struct {
	DatabaseKeys          int64                             `json:"database_keys"`
	ScannedKeys           int                               `json:"scanned_keys"`
	Truncated             bool                              `json:"truncated"`
	NoTTLKeys             int64                             `json:"no_ttl_keys"`
	ExpectedPermanentKeys int64                             `json:"expected_permanent_keys"`
	Categories            map[string]RedisInventoryCategory `json:"categories"`
}

type RedisMaintenanceService struct {
	redis    *RedisStore
	queue    *QueueService
	settings *SettingsService
	logger   *zap.Logger
	done     chan struct{}
	close    sync.Once
	running  atomic.Bool
}

func NewRedisMaintenanceService(redisStore *RedisStore, queue *QueueService, settings *SettingsService, logger *zap.Logger) *RedisMaintenanceService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedisMaintenanceService{
		redis: redisStore, queue: queue, settings: settings, logger: logger, done: make(chan struct{}),
	}
}

func (s *RedisMaintenanceService) Start() {
	if s == nil || s.redis == nil || s.settings == nil {
		return
	}
	s.ensureNextRun()
	go s.loop()
}

func (s *RedisMaintenanceService) Close() {
	if s == nil {
		return
	}
	s.close.Do(func() { close(s.done) })
}

func (s *RedisMaintenanceService) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.runIfDue()
		}
	}
}

func (s *RedisMaintenanceService) ensureNextRun() {
	cfg := ReadRedisMaintenanceConfig(s.settings)
	if !cfg.Enabled || strings.TrimSpace(cfg.NextRunAt) != "" {
		return
	}
	next := time.Now().UTC().Add(time.Duration(cfg.IntervalMinutes) * time.Minute).Format(time.RFC3339Nano)
	_ = s.settings.Set(context.Background(), "redis_maintenance_next_run_at", next)
}

func (s *RedisMaintenanceService) runIfDue() {
	cfg := ReadRedisMaintenanceConfig(s.settings)
	if !cfg.Enabled || !maintenanceTimeDue(cfg.NextRunAt, time.Now()) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if _, err := s.Run(ctx, nil); err != nil && !errors.Is(err, ErrRedisMaintenanceBusy) {
		s.logger.Warn("automatic redis maintenance failed", zap.Error(err))
	}
}

func (s *RedisMaintenanceService) Run(ctx context.Context, categories []string) (RedisMaintenanceResult, error) {
	started := time.Now()
	result := RedisMaintenanceResult{
		StartedAt:    started.UTC().Format(time.RFC3339Nano),
		Categories:   map[string]int64{},
		QueueResults: map[string]QueueCleanupResult{},
	}
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return result, errors.New("redis is not configured")
	}
	if !s.running.CompareAndSwap(false, true) {
		return result, ErrRedisMaintenanceBusy
	}
	defer s.running.Store(false)
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	locked, err := s.redis.Client().SetNX(ctx, redisMaintenanceLockKey, token, redisMaintenanceLockTTL).Result()
	if err != nil {
		return result, err
	}
	if !locked {
		return result, ErrRedisMaintenanceBusy
	}
	defer releaseRedisMaintenanceLock(context.Background(), s.redis.Client(), token)

	result.MemoryBefore = redisUsedMemory(ctx, s.redis.Client())
	selected := normalizeMaintenanceCategories(categories)
	cfg := ReadRedisMaintenanceConfig(s.settings)

	if selected["observability"] {
		counts, trimErr := s.trimObservability(ctx, cfg)
		mergeMaintenanceCounts(result.Categories, counts)
		if trimErr != nil {
			result.Warnings = append(result.Warnings, trimErr.Error())
		}
	}
	if selected["queue_history"] && s.queue != nil {
		queueResults, cleanupErr := s.queue.CleanupHistory(ctx, cfg)
		result.QueueResults = queueResults
		for _, item := range queueResults {
			result.Categories["queue_archived_deleted"] += int64(item.ArchivedDeleted)
			result.Categories["queue_completed_deleted"] += int64(item.CompletedDeleted)
		}
		if cleanupErr != nil {
			result.Warnings = append(result.Warnings, cleanupErr.Error())
		}
	}
	if selected["sessions"] {
		counts, cleanupErr := s.cleanupSessionIndexes(ctx, cfg)
		mergeMaintenanceCounts(result.Categories, counts)
		if cleanupErr != nil {
			result.Warnings = append(result.Warnings, cleanupErr.Error())
		}
	}
	if selected["cache"] {
		deleted, cleanupErr := s.clearApplicationCache(ctx)
		result.Categories["cache_keys_deleted"] = deleted
		if cleanupErr != nil {
			result.Warnings = append(result.Warnings, cleanupErr.Error())
		}
	}

	result.MemoryAfter = redisUsedMemory(ctx, s.redis.Client())
	if result.MemoryBefore > result.MemoryAfter {
		result.MemoryFreed = result.MemoryBefore - result.MemoryAfter
	}
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	result.DurationMS = time.Since(started).Milliseconds()
	next := time.Now().UTC().Add(time.Duration(cfg.IntervalMinutes) * time.Minute).Format(time.RFC3339Nano)
	_ = SaveRedisMaintenanceRun(ctx, s.settings, next, result)
	if selected["sessions"] {
		s.logger.Info("redis session maintenance completed",
			zap.Int64("refresh_sessions_renewed", result.Categories["refresh_sessions_renewed"]),
			zap.Int64("refresh_sessions_inactive", result.Categories["refresh_sessions_inactive"]),
			zap.Int64("refresh_sessions_renewal_disabled", result.Categories["refresh_sessions_renewal_disabled"]),
			zap.Int64("sessions_deleted", result.Categories["sessions_deleted"]),
			zap.Int("refresh_token_active_ttl_seconds", cfg.RefreshTokenActiveTTLSeconds),
			zap.Int("refresh_token_renewal_interval_seconds", cfg.RefreshTokenRenewalIntervalSeconds),
			zap.String("refresh_token_mode", cfg.RefreshTokenMode),
			zap.Bool("refresh_token_auto_renew_enabled", cfg.RefreshTokenAutoRenewEnabled),
			zap.Int64("duration_ms", result.DurationMS),
			zap.Int("warnings", len(result.Warnings)),
		)
	}
	return result, nil
}

func (s *RedisMaintenanceService) Status(ctx context.Context) map[string]any {
	cfg := ReadRedisMaintenanceConfig(s.settings)
	status := map[string]any{
		"configured": false,
		"available":  false,
		"running":    s != nil && s.running.Load(),
		"config":     cfg,
		"inventory":  RedisInventory{Categories: map[string]RedisInventoryCategory{}},
	}
	if s == nil || s.redis == nil {
		return status
	}
	redisStatus := s.redis.Status(ctx, s.redis.cfg)
	status["redis"] = redisStatus
	status["configured"] = redisStatus["configured"]
	status["available"] = redisStatus["available"]
	if redisStatus["available"] == true && s.redis.Client() != nil {
		status["inventory"] = redisInventory(ctx, s.redis.Client(), redisMaintenanceInventoryKeys)
	}
	status["pressure"] = redisMemoryPressure(redisStatus, cfg)
	return status
}

func ReadRedisMaintenanceConfig(settings *SettingsService) RedisMaintenanceConfig {
	cfg := RedisMaintenanceConfig{
		Enabled:                            true,
		IntervalMinutes:                    60,
		AccessLogRetentionHours:            defaultRedisAccessLogRetentionHours,
		AccessLogMaxEntries:                defaultRedisAccessLogMaxEntries,
		SystemLogRetentionHours:            168,
		SystemLogMaxEntries:                defaultRedisSystemLogMaxEntries,
		MetricsRetentionHours:              24,
		MetricsMaxEntriesPerKey:            defaultRedisMetricsMaxEntries,
		QueueEventRetentionHours:           24,
		QueueEventMaxEntries:               defaultRedisQueueEventMaxEntries,
		CompletedRetentionHours:            24,
		ArchivedRetentionHours:             720,
		ArchivedMaxTasksPerQueue:           1000,
		MemoryWarningPercent:               75,
		MemoryCriticalPercent:              90,
		AccessTokenTTLSeconds:              int(defaultAccessTokenTTL.Seconds()),
		RefreshTokenActiveTTLSeconds:       int(defaultRefreshSessionTTL.Seconds()),
		RefreshTokenRenewalIntervalSeconds: int(defaultRefreshSessionRenewalIntervalTTL.Seconds()),
		RefreshTokenMode:                   RefreshTokenModeRedisOpaque,
		RefreshTokenAutoRenewEnabled:       true,
		SessionInactiveTTLSeconds:          int(defaultSessionIdleTTL.Seconds()),
		UserActiveSessionLimit:             defaultUserSessionLimit,
		QueueOverrides:                     map[string]RedisQueueRetentionPolicy{},
	}
	if settings == nil {
		return normalizeRedisMaintenanceConfig(cfg)
	}
	cfg.Enabled = settings.Bool("redis_maintenance_enabled")
	cfg.IntervalMinutes = settings.Int("redis_maintenance_interval_minutes", cfg.IntervalMinutes)
	cfg.AccessLogRetentionHours = settings.Int("redis_access_log_retention_hours", cfg.AccessLogRetentionHours)
	cfg.AccessLogMaxEntries = settings.Int("redis_access_log_max_entries", cfg.AccessLogMaxEntries)
	cfg.SystemLogRetentionHours = settings.Int("redis_system_log_retention_hours", cfg.SystemLogRetentionHours)
	cfg.SystemLogMaxEntries = settings.Int("redis_system_log_max_entries", cfg.SystemLogMaxEntries)
	cfg.MetricsRetentionHours = settings.Int("redis_metrics_retention_hours", cfg.MetricsRetentionHours)
	cfg.MetricsMaxEntriesPerKey = settings.Int("redis_metrics_max_entries_per_key", cfg.MetricsMaxEntriesPerKey)
	cfg.QueueEventRetentionHours = settings.Int("redis_queue_event_retention_hours", cfg.QueueEventRetentionHours)
	cfg.QueueEventMaxEntries = settings.Int("redis_queue_event_max_entries", cfg.QueueEventMaxEntries)
	cfg.CompletedRetentionHours = settings.Int("redis_completed_retention_hours", cfg.CompletedRetentionHours)
	cfg.ArchivedRetentionHours = settings.Int("redis_archived_retention_hours", cfg.ArchivedRetentionHours)
	cfg.ArchivedMaxTasksPerQueue = settings.Int("redis_archived_max_tasks_per_queue", cfg.ArchivedMaxTasksPerQueue)
	cfg.MemoryWarningPercent = settings.Int("redis_memory_warning_percent", cfg.MemoryWarningPercent)
	cfg.MemoryCriticalPercent = settings.Int("redis_memory_critical_percent", cfg.MemoryCriticalPercent)
	cfg.AccessTokenTTLSeconds = settings.Int("access_token_ttl_seconds", cfg.AccessTokenTTLSeconds)
	cfg.RefreshTokenActiveTTLSeconds = settings.Int("refresh_token_active_ttl_seconds", cfg.RefreshTokenActiveTTLSeconds)
	cfg.RefreshTokenRenewalIntervalSeconds = settings.Int("refresh_token_renewal_interval_seconds", cfg.RefreshTokenRenewalIntervalSeconds)
	cfg.RefreshTokenMode = settings.String("refresh_token_mode")
	cfg.RefreshTokenAutoRenewEnabled = settings.Bool("refresh_token_auto_renew_enabled")
	if _, exists := settings.ExplicitValue("session_inactive_ttl_seconds"); exists {
		cfg.SessionInactiveTTLSeconds = settings.Int("session_inactive_ttl_seconds", cfg.SessionInactiveTTLSeconds)
	} else {
		cfg.SessionInactiveTTLSeconds = settings.Int("redis_session_inactive_days", 7) * int((24 * time.Hour).Seconds())
	}
	cfg.UserActiveSessionLimit = settings.Int("redis_user_active_session_limit", cfg.UserActiveSessionLimit)
	cfg.NextRunAt = strings.TrimSpace(settings.String("redis_maintenance_next_run_at"))
	cfg.LastRunAt = strings.TrimSpace(settings.String("redis_maintenance_last_run_at"))
	cfg.LastResult = decodeSettingRaw(settings.String("redis_maintenance_last_result"))
	rawOverrides := settings.Get("redis_maintenance_queue_overrides")
	data, _ := json.Marshal(rawOverrides)
	if text, ok := rawOverrides.(string); ok {
		data = []byte(text)
	}
	_ = json.Unmarshal(data, &cfg.QueueOverrides)
	return normalizeRedisMaintenanceConfig(cfg)
}

func SaveRedisMaintenanceConfig(ctx context.Context, settings *SettingsService, cfg RedisMaintenanceConfig) bool {
	if settings == nil {
		return false
	}
	cfg = normalizeRedisMaintenanceConfig(cfg)
	if cfg.Enabled {
		cfg.NextRunAt = time.Now().UTC().Add(time.Duration(cfg.IntervalMinutes) * time.Minute).Format(time.RFC3339Nano)
	} else {
		cfg.NextRunAt = ""
	}
	updates := map[string]any{
		"redis_maintenance_enabled":              cfg.Enabled,
		"redis_maintenance_interval_minutes":     cfg.IntervalMinutes,
		"redis_access_log_retention_hours":       cfg.AccessLogRetentionHours,
		"redis_access_log_max_entries":           cfg.AccessLogMaxEntries,
		"redis_system_log_retention_hours":       cfg.SystemLogRetentionHours,
		"redis_system_log_max_entries":           cfg.SystemLogMaxEntries,
		"redis_metrics_retention_hours":          cfg.MetricsRetentionHours,
		"redis_metrics_max_entries_per_key":      cfg.MetricsMaxEntriesPerKey,
		"redis_queue_event_retention_hours":      cfg.QueueEventRetentionHours,
		"redis_queue_event_max_entries":          cfg.QueueEventMaxEntries,
		"redis_completed_retention_hours":        cfg.CompletedRetentionHours,
		"redis_archived_retention_hours":         cfg.ArchivedRetentionHours,
		"redis_archived_max_tasks_per_queue":     cfg.ArchivedMaxTasksPerQueue,
		"redis_memory_warning_percent":           cfg.MemoryWarningPercent,
		"redis_memory_critical_percent":          cfg.MemoryCriticalPercent,
		"access_token_ttl_seconds":               cfg.AccessTokenTTLSeconds,
		"refresh_token_active_ttl_seconds":       cfg.RefreshTokenActiveTTLSeconds,
		"refresh_token_renewal_interval_seconds": cfg.RefreshTokenRenewalIntervalSeconds,
		"refresh_token_mode":                     cfg.RefreshTokenMode,
		"refresh_token_auto_renew_enabled":       cfg.RefreshTokenAutoRenewEnabled,
		"session_inactive_ttl_seconds":           cfg.SessionInactiveTTLSeconds,
		"redis_user_active_session_limit":        cfg.UserActiveSessionLimit,
		"redis_maintenance_queue_overrides":      cfg.QueueOverrides,
		"redis_maintenance_next_run_at":          cfg.NextRunAt,
	}
	for key, value := range updates {
		if !settings.Set(ctx, key, value) {
			return false
		}
	}
	return true
}

func SaveRedisMaintenanceRun(ctx context.Context, settings *SettingsService, nextRunAt string, result RedisMaintenanceResult) bool {
	if settings == nil {
		return false
	}
	for key, value := range map[string]any{
		"redis_maintenance_last_run_at": time.Now().UTC().Format(time.RFC3339Nano),
		"redis_maintenance_next_run_at": strings.TrimSpace(nextRunAt),
		"redis_maintenance_last_result": result,
	} {
		if !settings.Set(ctx, key, value) {
			return false
		}
	}
	return true
}

func (cfg RedisMaintenanceConfig) QueuePolicy(queue string) RedisQueueRetentionPolicy {
	policy := cfg.QueuePolicyWithoutOverride()
	if override, ok := cfg.QueueOverrides[queue]; ok {
		return normalizeRedisQueuePolicy(override, policy)
	}
	return normalizeRedisQueuePolicy(policy, policy)
}

func normalizeRedisMaintenanceConfig(cfg RedisMaintenanceConfig) RedisMaintenanceConfig {
	cfg.IntervalMinutes = clampRedisInt(cfg.IntervalMinutes, 5, 1440)
	cfg.AccessLogRetentionHours = clampRedisInt(cfg.AccessLogRetentionHours, 1, 720)
	cfg.AccessLogMaxEntries = clampRedisInt(cfg.AccessLogMaxEntries, 1000, 1000000)
	cfg.SystemLogRetentionHours = clampRedisInt(cfg.SystemLogRetentionHours, 1, 2160)
	cfg.SystemLogMaxEntries = clampRedisInt(cfg.SystemLogMaxEntries, 1000, 500000)
	cfg.MetricsRetentionHours = clampRedisInt(cfg.MetricsRetentionHours, 1, 720)
	cfg.MetricsMaxEntriesPerKey = clampRedisInt(cfg.MetricsMaxEntriesPerKey, 1000, 1000000)
	cfg.QueueEventRetentionHours = clampRedisInt(cfg.QueueEventRetentionHours, 1, 720)
	cfg.QueueEventMaxEntries = clampRedisInt(cfg.QueueEventMaxEntries, 1000, 500000)
	cfg.CompletedRetentionHours = clampRedisInt(cfg.CompletedRetentionHours, 0, 720)
	cfg.ArchivedRetentionHours = clampRedisInt(cfg.ArchivedRetentionHours, 1, 2160)
	cfg.ArchivedMaxTasksPerQueue = clampRedisInt(cfg.ArchivedMaxTasksPerQueue, 1, 10000)
	cfg.MemoryWarningPercent = clampRedisInt(cfg.MemoryWarningPercent, 10, 95)
	cfg.MemoryCriticalPercent = clampRedisInt(cfg.MemoryCriticalPercent, cfg.MemoryWarningPercent+1, 100)
	cfg.AccessTokenTTLSeconds = clampRedisInt(cfg.AccessTokenTTLSeconds, int(time.Minute.Seconds()), int((24 * time.Hour).Seconds()))
	cfg.RefreshTokenActiveTTLSeconds = clampRedisInt(cfg.RefreshTokenActiveTTLSeconds, int(time.Hour.Seconds()), int((365 * 24 * time.Hour).Seconds()))
	cfg.RefreshTokenRenewalIntervalSeconds = clampRedisInt(cfg.RefreshTokenRenewalIntervalSeconds, int(time.Hour.Seconds()), int((30 * 24 * time.Hour).Seconds()))
	cfg.RefreshTokenMode = normalizeRefreshTokenMode(cfg.RefreshTokenMode)
	cfg.SessionInactiveTTLSeconds = clampRedisInt(cfg.SessionInactiveTTLSeconds, int(time.Hour.Seconds()), int((365 * 24 * time.Hour).Seconds()))
	cfg.UserActiveSessionLimit = clampRedisInt(cfg.UserActiveSessionLimit, 1, 100)
	normalized := map[string]RedisQueueRetentionPolicy{}
	for queue, policy := range cfg.QueueOverrides {
		if !containsRedisString(QueueNames, queue) {
			continue
		}
		normalized[queue] = normalizeRedisQueuePolicy(policy, cfg.QueuePolicyWithoutOverride())
	}
	cfg.QueueOverrides = normalized
	return cfg
}

func (cfg RedisMaintenanceConfig) QueuePolicyWithoutOverride() RedisQueueRetentionPolicy {
	return RedisQueueRetentionPolicy{
		CompletedRetentionHours: cfg.CompletedRetentionHours,
		ArchivedRetentionHours:  cfg.ArchivedRetentionHours,
		ArchivedMaxTasks:        cfg.ArchivedMaxTasksPerQueue,
	}
}

func normalizeRedisQueuePolicy(policy, fallback RedisQueueRetentionPolicy) RedisQueueRetentionPolicy {
	if policy.CompletedRetentionHours < 0 {
		policy.CompletedRetentionHours = fallback.CompletedRetentionHours
	}
	if policy.ArchivedRetentionHours <= 0 {
		policy.ArchivedRetentionHours = fallback.ArchivedRetentionHours
	}
	if policy.ArchivedMaxTasks <= 0 {
		policy.ArchivedMaxTasks = fallback.ArchivedMaxTasks
	}
	policy.CompletedRetentionHours = clampRedisInt(policy.CompletedRetentionHours, 0, 720)
	policy.ArchivedRetentionHours = clampRedisInt(policy.ArchivedRetentionHours, 1, 2160)
	policy.ArchivedMaxTasks = clampRedisInt(policy.ArchivedMaxTasks, 1, 10000)
	return policy
}
