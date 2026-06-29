package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeebo/xxh3"

	"yuem-go/backend-gin/internal/config"
)

const cacheKeyPartMaxPlainLength = 96

type RedisStore struct {
	client            *redis.Client
	cfg               config.RedisConfig
	mu                sync.RWMutex
	l1                map[string]redisCacheItem
	cacheFailureUntil time.Time
	now               func() time.Time
}

func NewRedisStore(cfg config.RedisConfig) *RedisStore {
	cfg = normalizeRedisConfig(cfg)
	options := &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		MaxRetries:   0,
	}
	if cfg.PoolSize > 0 {
		options.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		options.MinIdleConns = cfg.MinIdleConns
	}
	return &RedisStore{
		cfg:    cfg,
		client: redis.NewClient(options),
		l1:     map[string]redisCacheItem{},
		now:    time.Now,
	}
}

type redisCacheItem struct {
	raw      []byte
	expireAt time.Time
}

func normalizeRedisConfig(cfg config.RedisConfig) config.RedisConfig {
	if cfg.CacheCommandTimeout <= 0 {
		cfg.CacheCommandTimeout = 50 * time.Millisecond
	}
	if cfg.CacheDefaultTTL <= 0 {
		cfg.CacheDefaultTTL = 30 * time.Second
	}
	if cfg.CacheL1TTL < 0 {
		cfg.CacheL1TTL = 0
	}
	if cfg.CacheL1MaxEntries < 0 {
		cfg.CacheL1MaxEntries = 0
	}
	if cfg.PoolSize < 0 {
		cfg.PoolSize = 0
	}
	if cfg.MinIdleConns < 0 {
		cfg.MinIdleConns = 0
	}
	return cfg
}

func (s *RedisStore) Client() *redis.Client {
	if s == nil {
		return nil
	}
	return s.client
}

func (s *RedisStore) Available(ctx context.Context) bool {
	if s == nil || s.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	return s.client.Ping(ctx).Err() == nil
}

func (s *RedisStore) Status(ctx context.Context, cfg config.RedisConfig) map[string]any {
	status := map[string]any{
		"configured": false,
		"available":  false,
		"addr":       cfg.Addr,
		"db":         cfg.DB,
		"cache": map[string]any{
			"enabled":            cfg.CacheEnabled,
			"command_timeout_ms": cfg.CacheCommandTimeout.Milliseconds(),
			"default_ttl_sec":    cfg.CacheDefaultTTL.Seconds(),
			"l1_ttl_sec":         cfg.CacheL1TTL.Seconds(),
			"l1_max_entries":     cfg.CacheL1MaxEntries,
		},
		"checkedAt": time.Now().UnixMilli(),
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		status["message"] = "Redis address is empty"
		return status
	}
	status["configured"] = true
	if s == nil || s.client == nil {
		status["message"] = "Redis client is not initialized"
		return status
	}
	start := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	if err := s.client.Ping(pingCtx).Err(); err != nil {
		status["message"] = err.Error()
		status["pingMs"] = time.Since(start).Milliseconds()
		return status
	}
	status["available"] = true
	status["pingMs"] = time.Since(start).Milliseconds()

	infoCtx, infoCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer infoCancel()
	if info, err := s.client.Info(infoCtx, "server", "memory", "clients", "keyspace", "stats").Result(); err == nil {
		status["info"] = parseRedisInfo(info)
	}
	return status
}

func parseRedisInfo(raw string) map[string]any {
	out := map[string]any{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "redis_version", "redis_mode", "os", "arch_bits", "process_id", "tcp_port", "uptime_in_seconds",
			"used_memory", "used_memory_human", "used_memory_peak", "used_memory_peak_human", "used_memory_dataset",
			"used_memory_dataset_perc", "connected_clients", "blocked_clients", "tracking_clients", "maxmemory",
			"maxmemory_human", "maxmemory_policy", "mem_fragmentation_ratio", "expired_keys", "evicted_keys",
			"keyspace_hits", "keyspace_misses":
			out[key] = value
		default:
			if strings.HasPrefix(key, "db") {
				out[key] = value
			}
		}
	}
	return out
}

func (s *RedisStore) GetRaw(ctx context.Context, key string) (string, bool) {
	if s == nil || s.client == nil {
		return "", false
	}
	value, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return value, true
}

func (s *RedisStore) GetJSON(ctx context.Context, key string, out any) bool {
	raw, ok := s.GetRaw(ctx, key)
	if !ok {
		return false
	}
	return json.Unmarshal([]byte(raw), out) == nil
}

// GetDelJSON atomically consumes a JSON value. It is intended for one-time
// credentials such as login handoff tickets.
func (s *RedisStore) GetDelJSON(ctx context.Context, key string, out any) bool {
	if s == nil || s.client == nil || out == nil || strings.TrimSpace(key) == "" {
		return false
	}
	raw, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(raw), out) == nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value any, ttl time.Duration) bool {
	if s == nil || s.client == nil {
		return false
	}
	var serialized string
	if raw, ok := value.(string); ok {
		serialized = raw
	} else {
		data, err := json.Marshal(value)
		if err != nil {
			return false
		}
		serialized = string(data)
	}
	return s.client.Set(ctx, key, serialized, ttl).Err() == nil
}

func (s *RedisStore) Del(ctx context.Context, keys ...string) bool {
	if s == nil || s.client == nil || len(keys) == 0 {
		return false
	}
	return s.client.Del(ctx, keys...).Err() == nil
}

func (s *RedisStore) CacheGetJSON(ctx context.Context, key string, out any) bool {
	if s == nil || out == nil || strings.TrimSpace(key) == "" || !s.cfg.CacheEnabled {
		return false
	}
	if raw, ok := s.getL1(key); ok {
		return json.Unmarshal(raw, out) == nil
	}
	if !s.cacheUsable() {
		return false
	}
	cacheCtx, cancel := s.cacheContext(ctx)
	defer cancel()
	value, err := s.client.Get(cacheCtx, key).Result()
	if err != nil {
		if err != redis.Nil {
			s.noteCacheFailure()
		}
		return false
	}
	raw := []byte(value)
	if json.Unmarshal(raw, out) != nil {
		return false
	}
	s.setL1(key, raw, s.cfg.CacheL1TTL)
	return true
}

func (s *RedisStore) CacheSet(ctx context.Context, key string, value any, ttl time.Duration) bool {
	if s == nil || strings.TrimSpace(key) == "" || !s.cfg.CacheEnabled {
		return false
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return false
	}
	if ttl <= 0 {
		ttl = s.cfg.CacheDefaultTTL
	}
	l1TTL := time.Duration(0)
	if s.cfg.CacheL1TTL > 0 {
		l1TTL = minDuration(ttl, s.cfg.CacheL1TTL)
	}
	s.setL1(key, raw, l1TTL)
	if !s.cacheUsable() {
		return false
	}
	cacheCtx, cancel := s.cacheContext(ctx)
	defer cancel()
	if err := s.client.Set(cacheCtx, key, string(raw), ttl).Err(); err != nil {
		s.noteCacheFailure()
		return false
	}
	return true
}

func (s *RedisStore) CacheGetOrLoad(ctx context.Context, key string, out any, ttl time.Duration, loader func() (any, error)) (bool, error) {
	if s != nil && s.CacheGetJSON(ctx, key, out) {
		return true, nil
	}
	value, err := loader()
	if err != nil {
		return false, err
	}
	if out != nil {
		data, err := json.Marshal(value)
		if err != nil {
			return false, err
		}
		if err := json.Unmarshal(data, out); err != nil {
			return false, err
		}
	}
	if s != nil {
		_ = s.CacheSet(ctx, key, value, ttl)
	}
	return false, nil
}

func (s *RedisStore) CacheVersion(ctx context.Context, scope string) int64 {
	scope = strings.TrimSpace(scope)
	if s == nil || scope == "" || !s.cfg.CacheEnabled || !s.cacheUsable() {
		return 0
	}
	cacheCtx, cancel := s.cacheContext(ctx)
	defer cancel()
	raw, err := s.client.Get(cacheCtx, CacheVersionKey(scope)).Int64()
	if err != nil {
		if err != redis.Nil {
			s.noteCacheFailure()
		}
		return 0
	}
	return raw
}

func (s *RedisStore) CacheVersions(ctx context.Context, scopes ...string) map[string]int64 {
	versions := make(map[string]int64, len(scopes))
	keys := make([]string, 0, len(scopes))
	keyScopes := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		versions[scope] = 0
		keys = append(keys, CacheVersionKey(scope))
		keyScopes = append(keyScopes, scope)
	}
	if len(keys) == 0 || s == nil || !s.cfg.CacheEnabled || !s.cacheUsable() {
		return versions
	}
	cacheCtx, cancel := s.cacheContext(ctx)
	defer cancel()
	values, err := s.client.MGet(cacheCtx, keys...).Result()
	if err != nil {
		s.noteCacheFailure()
		return versions
	}
	for i, value := range values {
		versions[keyScopes[i]] = redisInt64(value)
	}
	return versions
}

func (s *RedisStore) BumpCacheVersion(ctx context.Context, scopes ...string) {
	if s == nil || !s.cfg.CacheEnabled {
		return
	}
	s.clearL1()
	if !s.cacheUsable() {
		return
	}
	cacheCtx, cancel := s.cacheContext(ctx)
	defer cancel()
	pipe := s.client.Pipeline()
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		pipe.Incr(cacheCtx, CacheVersionKey(scope))
	}
	if _, err := pipe.Exec(cacheCtx); err != nil && err != redis.Nil {
		s.noteCacheFailure()
	}
}

func CacheVersionKey(scope string) string {
	return "cache:version:" + strings.TrimSpace(scope)
}

func CacheKey(scope string, version int64, parts ...any) string {
	builder := strings.Builder{}
	builder.WriteString("cache:")
	builder.WriteString(strings.TrimSpace(scope))
	builder.WriteString(":v")
	builder.WriteString(fmt.Sprint(version))
	for _, part := range parts {
		builder.WriteByte(':')
		builder.WriteString(cacheKeyPart(part))
	}
	return builder.String()
}

func cacheKeyPart(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	text = strings.ReplaceAll(text, " ", "_")
	text = strings.ReplaceAll(text, "\n", "_")
	text = strings.ReplaceAll(text, "\r", "_")
	if text == "" {
		return "-"
	}
	if shouldDigestCacheKeyPart(text) {
		sum := xxh3.HashString128(text).Bytes()
		return "h3:" + hex.EncodeToString(sum[:])
	}
	return text
}

func shouldDigestCacheKeyPart(text string) bool {
	if len(text) > cacheKeyPartMaxPlainLength {
		return true
	}
	for _, ch := range text {
		if ch < 0x20 || ch == 0x7f {
			return true
		}
	}
	return false
}

func redisInt64(value any) int64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	case []byte:
		parsed, _ := strconv.ParseInt(string(typed), 10, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseInt(fmt.Sprint(typed), 10, 64)
		return parsed
	}
}

func (s *RedisStore) getL1(key string) ([]byte, bool) {
	if s == nil || s.cfg.CacheL1TTL <= 0 {
		return nil, false
	}
	now := s.nowTime()
	s.mu.RLock()
	item, ok := s.l1[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !item.expireAt.IsZero() && now.After(item.expireAt) {
		s.mu.Lock()
		delete(s.l1, key)
		s.mu.Unlock()
		return nil, false
	}
	return item.raw, true
}

func (s *RedisStore) setL1(key string, raw []byte, ttl time.Duration) {
	if s == nil || ttl <= 0 {
		return
	}
	now := s.nowTime()
	item := redisCacheItem{raw: append([]byte(nil), raw...), expireAt: now.Add(ttl)}
	s.mu.Lock()
	if s.l1 == nil {
		s.l1 = map[string]redisCacheItem{}
	}
	s.l1[key] = item
	s.pruneL1Locked(now)
	s.mu.Unlock()
}

func (s *RedisStore) pruneL1Locked(now time.Time) {
	if s == nil || s.cfg.CacheL1MaxEntries <= 0 || len(s.l1) <= s.cfg.CacheL1MaxEntries {
		return
	}
	for key, item := range s.l1 {
		if !item.expireAt.IsZero() && now.After(item.expireAt) {
			delete(s.l1, key)
		}
	}
	for len(s.l1) > s.cfg.CacheL1MaxEntries {
		var oldestKey string
		var oldestExpire time.Time
		for key, item := range s.l1 {
			if oldestKey == "" || item.expireAt.Before(oldestExpire) {
				oldestKey = key
				oldestExpire = item.expireAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(s.l1, oldestKey)
	}
}

func (s *RedisStore) clearL1() {
	s.mu.Lock()
	s.l1 = map[string]redisCacheItem{}
	s.mu.Unlock()
}

func (s *RedisStore) cacheUsable() bool {
	if s == nil || s.client == nil {
		return false
	}
	s.mu.RLock()
	until := s.cacheFailureUntil
	s.mu.RUnlock()
	return until.IsZero() || s.nowTime().After(until)
}

func (s *RedisStore) noteCacheFailure() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cacheFailureUntil = s.nowTime().Add(5 * time.Second)
	s.mu.Unlock()
}

func (s *RedisStore) cacheContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := s.cfg.CacheCommandTimeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *RedisStore) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return 0
	}
	if a < b {
		return a
	}
	return b
}
