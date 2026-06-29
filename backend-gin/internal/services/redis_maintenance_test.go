package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"yuem-go/backend-gin/internal/config"
)

func TestRedisMaintenanceConfigNormalizesBoundsAndOverrides(t *testing.T) {
	cfg := normalizeRedisMaintenanceConfig(RedisMaintenanceConfig{
		IntervalMinutes:                    1,
		AccessLogRetentionHours:            0,
		AccessLogMaxEntries:                10,
		SystemLogRetentionHours:            9999,
		SystemLogMaxEntries:                999999,
		MetricsRetentionHours:              0,
		MetricsMaxEntriesPerKey:            10,
		QueueEventRetentionHours:           0,
		QueueEventMaxEntries:               999999,
		CompletedRetentionHours:            -1,
		ArchivedRetentionHours:             0,
		ArchivedMaxTasksPerQueue:           0,
		MemoryWarningPercent:               99,
		MemoryCriticalPercent:              20,
		AccessTokenTTLSeconds:              0,
		RefreshTokenActiveTTLSeconds:       0,
		RefreshTokenRenewalIntervalSeconds: 0,
		RefreshTokenMode:                   "unknown",
		RefreshTokenAutoRenewEnabled:       false,
		SessionInactiveTTLSeconds:          0,
		UserActiveSessionLimit:             0,
		QueueOverrides: map[string]RedisQueueRetentionPolicy{
			QueueVideoTranscoding: {
				CompletedRetentionHours: 72,
				ArchivedRetentionHours:  168,
				ArchivedMaxTasks:        250,
			},
			"unknown": {
				CompletedRetentionHours: 1,
				ArchivedRetentionHours:  1,
				ArchivedMaxTasks:        1,
			},
		},
	})
	if cfg.IntervalMinutes != 5 || cfg.SystemLogRetentionHours != 2160 || cfg.MetricsRetentionHours != 1 {
		t.Fatalf("normalized retention config = %+v", cfg)
	}
	if cfg.AccessLogRetentionHours != 1 ||
		cfg.AccessLogMaxEntries != 1000 ||
		cfg.SystemLogMaxEntries != 500000 ||
		cfg.MetricsMaxEntriesPerKey != 1000 ||
		cfg.QueueEventMaxEntries != 500000 {
		t.Fatalf("normalized observability caps = %+v", cfg)
	}
	if cfg.MemoryWarningPercent != 95 || cfg.MemoryCriticalPercent != 96 {
		t.Fatalf("normalized pressure thresholds = %d/%d", cfg.MemoryWarningPercent, cfg.MemoryCriticalPercent)
	}
	if cfg.AccessTokenTTLSeconds != 60 ||
		cfg.RefreshTokenActiveTTLSeconds != 3600 ||
		cfg.RefreshTokenRenewalIntervalSeconds != 3600 ||
		cfg.SessionInactiveTTLSeconds != 3600 ||
		cfg.UserActiveSessionLimit != 1 {
		t.Fatalf("normalized token/session policy = %+v, want lower bounds", cfg)
	}
	if cfg.RefreshTokenMode != RefreshTokenModeRedisOpaque || cfg.RefreshTokenAutoRenewEnabled {
		t.Fatalf("normalized refresh switch = %+v", cfg)
	}
	if _, ok := cfg.QueueOverrides["unknown"]; ok {
		t.Fatal("unknown queue override should be removed")
	}
	policy := cfg.QueuePolicy(QueueVideoTranscoding)
	if policy.CompletedRetentionHours != 72 || policy.ArchivedRetentionHours != 168 || policy.ArchivedMaxTasks != 250 {
		t.Fatalf("video queue policy = %+v", policy)
	}
}

func TestRedisMaintenanceDefaultObservabilityCaps(t *testing.T) {
	cfg := ReadRedisMaintenanceConfig(NewSettingsService(nil, nil))
	if cfg.AccessLogRetentionHours != defaultRedisAccessLogRetentionHours ||
		cfg.AccessLogMaxEntries != defaultRedisAccessLogMaxEntries ||
		cfg.SystemLogMaxEntries != defaultRedisSystemLogMaxEntries ||
		cfg.MetricsMaxEntriesPerKey != defaultRedisMetricsMaxEntries ||
		cfg.QueueEventMaxEntries != defaultRedisQueueEventMaxEntries {
		t.Fatalf("default observability caps = %+v", cfg)
	}
}

func TestRedisMaintenanceTrimsObservabilityByRetentionAndMaxEntries(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = store.Client().Close()
	})
	client := store.Client()
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)

	addStreamEntry(t, client, accessLogStreamKey, old, 0)
	addStreamEntry(t, client, systemLogStreamKey, old, 0)
	for i := range 6 {
		at := now.Add(time.Duration(i) * time.Millisecond)
		addStreamEntry(t, client, accessLogStreamKey, at, i+1)
		addStreamEntry(t, client, systemLogStreamKey, at, i+1)
		addZSetEntry(t, client, requestMetricZSetKey, at, fmt.Sprintf("request-%d", i))
		addZSetEntry(t, client, slowRequestZSetKey, at, fmt.Sprintf("slow-request-%d", i))
		addZSetEntry(t, client, slowQueryZSetKey, at, fmt.Sprintf("slow-query-%d", i))
		addZSetEntry(t, client, postgresMetricZSetKey, at, fmt.Sprintf("postgres-%d", i))
		addZSetEntry(t, client, runtimeMetricZSetKey, at, fmt.Sprintf("runtime-%d", i))
		addZSetEntry(t, client, queueEventZSetKey, at, fmt.Sprintf("queue-%d", i))
	}
	for _, key := range []string{requestMetricZSetKey, slowRequestZSetKey, slowQueryZSetKey, postgresMetricZSetKey, runtimeMetricZSetKey, queueEventZSetKey} {
		addZSetEntry(t, client, key, old, "old-"+key)
	}

	care := NewRedisMaintenanceService(store, nil, NewSettingsService(nil, nil), nil)
	counts, err := care.trimObservability(ctx, RedisMaintenanceConfig{
		AccessLogRetentionHours:  24,
		AccessLogMaxEntries:      3,
		SystemLogRetentionHours:  24,
		SystemLogMaxEntries:      3,
		MetricsRetentionHours:    24,
		MetricsMaxEntriesPerKey:  3,
		QueueEventRetentionHours: 24,
		QueueEventMaxEntries:     3,
	})
	if err != nil {
		t.Fatalf("trimObservability error: %v", err)
	}
	if counts["access_stream_trimmed"] < 4 || counts["system_stream_trimmed"] < 4 || counts["request_metrics_deleted"] < 4 {
		t.Fatalf("trim counts = %#v, want stream and zset deletions", counts)
	}
	for _, key := range []string{accessLogStreamKey, systemLogStreamKey} {
		length, _ := client.XLen(ctx, key).Result()
		if length > 3 {
			t.Fatalf("%s length = %d, want <= 3", key, length)
		}
		assertRedisTTL(t, client, key)
	}
	for _, key := range []string{requestMetricZSetKey, slowRequestZSetKey, slowQueryZSetKey, postgresMetricZSetKey, runtimeMetricZSetKey, queueEventZSetKey} {
		length, _ := client.ZCard(ctx, key).Result()
		if length > 3 {
			t.Fatalf("%s length = %d, want <= 3", key, length)
		}
		if _, err := client.ZScore(ctx, key, "old-"+key).Result(); err != redis.Nil {
			t.Fatalf("%s old member error = %v, want redis.Nil", key, err)
		}
		assertRedisTTL(t, client, key)
	}
}

func TestRedisMaintenanceRenewsOnlyRecentlyActiveRefreshSessions(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	settings := NewSettingsService(nil, nil)
	auth := NewAuthService(nil, store, config.AuthConfig{JWTSecret: "test-secret"}, settings)

	base := time.Now().UTC().Add(-2 * time.Hour)
	activeSession := Session{
		UserID:       "42",
		Token:        "active-access",
		RefreshToken: "active-refresh",
		ExpiresAt:    base.Add(3 * time.Hour).Format(time.RFC3339Nano),
		CreatedAt:    base.Add(-time.Hour).Format(time.RFC3339Nano),
		LastActiveAt: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339Nano),
	}
	inactiveSession := Session{
		UserID:       "42",
		Token:        "inactive-access",
		RefreshToken: "inactive-refresh",
		ExpiresAt:    base.Add(3 * time.Hour).Format(time.RFC3339Nano),
		CreatedAt:    base.Add(-time.Hour).Format(time.RFC3339Nano),
		LastActiveAt: time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339Nano),
	}
	if !auth.CreateSession(ctx, activeSession, 3*time.Hour) || !auth.CreateSession(ctx, inactiveSession, 3*time.Hour) {
		t.Fatal("CreateSession failed")
	}

	care := NewRedisMaintenanceService(store, nil, settings, nil)
	counts, err := care.cleanupSessionIndexes(ctx, RedisMaintenanceConfig{
		AccessTokenTTLSeconds:              3600,
		RefreshTokenActiveTTLSeconds:       7200,
		RefreshTokenRenewalIntervalSeconds: 3600,
		RefreshTokenMode:                   RefreshTokenModeRedisOpaque,
		RefreshTokenAutoRenewEnabled:       true,
		SessionInactiveTTLSeconds:          int((24 * time.Hour).Seconds()),
		UserActiveSessionLimit:             10,
	})
	if err != nil {
		t.Fatalf("cleanupSessionIndexes error: %v", err)
	}
	if counts["refresh_sessions_renewed"] != 1 || counts["refresh_sessions_inactive"] != 1 {
		t.Fatalf("renewal counts = %#v, want one renewed and one inactive", counts)
	}
	var renewed Session
	if !store.GetJSON(ctx, sessionRefreshKey("active-refresh"), &renewed) {
		t.Fatal("active refresh session missing")
	}
	renewedExpiry := parseSessionTime(renewed.ExpiresAt)
	if ttl := time.Until(renewedExpiry); ttl < 90*time.Minute {
		t.Fatalf("active refresh ttl = %s, want close to 2h", ttl)
	}
	var inactive Session
	if !store.GetJSON(ctx, sessionRefreshKey("inactive-refresh"), &inactive) {
		t.Fatal("inactive refresh session missing")
	}
	if inactive.ExpiresAt == renewed.ExpiresAt {
		t.Fatal("inactive refresh session should not be extended")
	}
}

func TestRedisMaintenanceSkipsRefreshRenewalWhenDisabled(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	settings := NewSettingsService(nil, nil)
	auth := NewAuthService(nil, store, config.AuthConfig{JWTSecret: "test-secret"}, settings)

	base := time.Now().UTC().Add(-2 * time.Hour)
	session := Session{
		UserID:       "42",
		Token:        "active-access",
		RefreshToken: "active-refresh",
		ExpiresAt:    base.Add(3 * time.Hour).Format(time.RFC3339Nano),
		CreatedAt:    base.Add(-time.Hour).Format(time.RFC3339Nano),
		LastActiveAt: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339Nano),
	}
	if !auth.CreateSession(ctx, session, 3*time.Hour) {
		t.Fatal("CreateSession failed")
	}

	care := NewRedisMaintenanceService(store, nil, settings, nil)
	counts, err := care.cleanupSessionIndexes(ctx, RedisMaintenanceConfig{
		AccessTokenTTLSeconds:              3600,
		RefreshTokenActiveTTLSeconds:       7200,
		RefreshTokenRenewalIntervalSeconds: 3600,
		RefreshTokenMode:                   RefreshTokenModeRedisOpaque,
		RefreshTokenAutoRenewEnabled:       false,
		SessionInactiveTTLSeconds:          int((24 * time.Hour).Seconds()),
		UserActiveSessionLimit:             10,
	})
	if err != nil {
		t.Fatalf("cleanupSessionIndexes error: %v", err)
	}
	if counts["refresh_sessions_renewed"] != 0 || counts["refresh_sessions_renewal_disabled"] != 1 {
		t.Fatalf("renewal disabled counts = %#v", counts)
	}
	var unchanged Session
	if !store.GetJSON(ctx, sessionRefreshKey("active-refresh"), &unchanged) {
		t.Fatal("active refresh session missing")
	}
	if unchanged.ExpiresAt != session.ExpiresAt {
		t.Fatalf("disabled renewal changed expires_at = %s, want %s", unchanged.ExpiresAt, session.ExpiresAt)
	}
}

func TestRedisSessionIndexesHaveTTLAndCleanupStaleMembers(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = store.Client().Close()
	})
	settings := NewSettingsService(nil, nil)
	auth := NewAuthService(nil, store, config.AuthConfig{JWTSecret: "test-secret"}, settings)

	if !auth.CreateSession(ctx, Session{UserID: "42", Token: "access", RefreshToken: "refresh"}, 2*time.Hour) {
		t.Fatal("CreateSession failed")
	}
	assertRedisTTL(t, store.Client(), allSessionsKey)

	session, ok := auth.FindSessionByRefreshToken(ctx, "refresh", 42)
	if !ok {
		t.Fatal("session should be found by refresh token")
	}
	if err := store.Client().Del(ctx, allSessionsKey).Err(); err != nil {
		t.Fatalf("delete all_sessions: %v", err)
	}
	if !auth.RefreshSessionAccessToken(ctx, *session, "new-access", time.Now()) {
		t.Fatal("RefreshSessionAccessToken failed")
	}
	if _, err := store.Client().ZScore(ctx, allSessionsKey, fmt.Sprintf("%d", session.ID)).Result(); err != nil {
		t.Fatalf("refreshed session missing from all_sessions: %v", err)
	}
	assertRedisTTL(t, store.Client(), allSessionsKey)

	if err := store.Client().ZAdd(ctx, allSessionsKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: "999"}).Err(); err != nil {
		t.Fatalf("add stale all_sessions member: %v", err)
	}
	if err := store.Client().SAdd(ctx, "user_sessions:stale", "999").Err(); err != nil {
		t.Fatalf("add stale user session member: %v", err)
	}
	care := NewRedisMaintenanceService(store, nil, settings, nil)
	counts, err := care.cleanupSessionIndexes(ctx, RedisMaintenanceConfig{
		AccessTokenTTLSeconds:              3600,
		RefreshTokenActiveTTLSeconds:       7200,
		RefreshTokenRenewalIntervalSeconds: 3600,
		RefreshTokenMode:                   RefreshTokenModeRedisOpaque,
		RefreshTokenAutoRenewEnabled:       false,
		SessionInactiveTTLSeconds:          int((24 * time.Hour).Seconds()),
		UserActiveSessionLimit:             10,
	})
	if err != nil {
		t.Fatalf("cleanupSessionIndexes error: %v", err)
	}
	if counts["all_sessions_removed"] == 0 || counts["user_sessions_removed"] == 0 {
		t.Fatalf("cleanup counts = %#v, want stale all/user session cleanup", counts)
	}
	if _, err := store.Client().ZScore(ctx, allSessionsKey, "999").Result(); err != redis.Nil {
		t.Fatalf("stale all_sessions member error = %v, want redis.Nil", err)
	}
	if exists, _ := store.Client().Exists(ctx, "user_sessions:stale").Result(); exists != 0 {
		t.Fatalf("stale user_sessions key exists = %d, want 0", exists)
	}
	assertRedisTTL(t, store.Client(), allSessionsKey)
}

func TestRedisMaintenanceDefaultQueueOverrides(t *testing.T) {
	cfg := ReadRedisMaintenanceConfig(NewSettingsService(nil, nil))
	if got := cfg.QueuePolicy(QueueAuditLog).CompletedRetentionHours; got != 72 {
		t.Fatalf("audit completed retention = %d, want 72", got)
	}
	if got := cfg.QueuePolicy(QueueImageProtection).CompletedRetentionHours; got != 2 {
		t.Fatalf("image protection completed retention = %d, want 2", got)
	}
	if got := cfg.QueuePolicy(QueueBatchNoteCreate).CompletedRetentionHours; got != 24 {
		t.Fatalf("batch note completed retention = %d, want 24", got)
	}
}

func TestMaintenanceTimeDue(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	if !maintenanceTimeDue("", now) {
		t.Fatal("empty next run should be due")
	}
	if maintenanceTimeDue(now.Add(time.Minute).Format(time.RFC3339Nano), now) {
		t.Fatal("future next run should not be due")
	}
	if !maintenanceTimeDue(now.Add(-time.Minute).Format(time.RFC3339Nano), now) {
		t.Fatal("past next run should be due")
	}
}

func TestRedisKeyCategory(t *testing.T) {
	cases := map[string]string{
		"asynq:{queue}:pending":         "queues",
		"observability:system_logs":     "logs",
		"observability:request_metrics": "metrics",
		"session:id:12":                 "sessions",
		"user_sessions:42":              "sessions",
		"cache:posts:v1":                "cache",
		"settings:maintenance":          "settings",
		"admin_login:failures:127.0":    "security",
		"unclassified":                  "other",
	}
	for key, want := range cases {
		if got := redisKeyCategory(key); got != want {
			t.Fatalf("redisKeyCategory(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestRedisExpectedPermanentKeyClassification(t *testing.T) {
	expected := []string{
		sessionIDCounterKey,
		SettingsKeyPrefix + "redis_maintenance_enabled",
		CacheVersionKey("posts"),
	}
	for _, key := range expected {
		if !redisExpectedPermanentKey(key) {
			t.Fatalf("%s should be classified as expected permanent", key)
		}
	}
	for _, key := range []string{allSessionsKey, "user_sessions:42", "app_version:last_form_data", accessLogStreamKey} {
		if redisExpectedPermanentKey(key) {
			t.Fatalf("%s should not be classified as expected permanent", key)
		}
	}
}

func TestRedisInventoryCountsNoTTLKeys(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = store.Client().Close()
	})
	client := store.Client()
	if err := client.Set(ctx, sessionIDCounterKey, "1", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, SettingsKeyPrefix+"redis_maintenance_enabled", "true", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, CacheVersionKey("posts"), "1", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, "app_version:last_form_data", "{}", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, "cache:short", "{}", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	inventory := redisInventory(ctx, client, 20)
	if inventory.ExpectedPermanentKeys != 3 || inventory.NoTTLKeys != 1 {
		t.Fatalf("inventory no TTL counts = expected %d noTTL %d categories %#v", inventory.ExpectedPermanentKeys, inventory.NoTTLKeys, inventory.Categories)
	}
}

func TestCompletedTaskExpired(t *testing.T) {
	cutoff := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	if !completedTaskExpired(&asynq.TaskInfo{CompletedAt: cutoff.Add(-time.Second)}, cutoff) {
		t.Fatal("task completed before cutoff should expire")
	}
	if completedTaskExpired(&asynq.TaskInfo{CompletedAt: cutoff}, cutoff) {
		t.Fatal("task completed at cutoff should be retained")
	}
	if completedTaskExpired(&asynq.TaskInfo{}, cutoff) {
		t.Fatal("task without completion timestamp should be retained")
	}
}

func addStreamEntry(t *testing.T, client *redis.Client, key string, at time.Time, seq int) {
	t.Helper()
	id := fmt.Sprintf("%d-%d", at.UnixMilli(), seq)
	if err := client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: key,
		ID:     id,
		Values: map[string]any{"seq": seq},
	}).Err(); err != nil {
		t.Fatalf("add stream %s: %v", key, err)
	}
}

func addZSetEntry(t *testing.T, client *redis.Client, key string, at time.Time, member string) {
	t.Helper()
	if err := client.ZAdd(context.Background(), key, redis.Z{Score: float64(at.UnixMilli()), Member: member}).Err(); err != nil {
		t.Fatalf("add zset %s: %v", key, err)
	}
}

func assertRedisTTL(t *testing.T, client *redis.Client, key string) {
	t.Helper()
	ttl, err := client.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("TTL %s: %v", key, err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL %s = %s, want positive", key, ttl)
	}
}
