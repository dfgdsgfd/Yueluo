package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"yuem-go/backend-gin/internal/config"
)

func TestRedisCacheGetOrLoadUsesL1WhenRedisUnavailable(t *testing.T) {
	store := &RedisStore{
		cfg: normalizeRedisConfig(config.RedisConfig{
			CacheEnabled:    true,
			CacheDefaultTTL: time.Minute,
			CacheL1TTL:      time.Minute,
		}),
		l1:  map[string]redisCacheItem{},
		now: time.Now,
	}

	loads := 0
	loader := func() (any, error) {
		loads++
		return map[string]any{"value": "fresh"}, nil
	}

	var first map[string]any
	hit, err := store.CacheGetOrLoad(context.Background(), "cache:test:l1", &first, time.Minute, loader)
	if err != nil {
		t.Fatal(err)
	}
	if hit || loads != 1 || first["value"] != "fresh" {
		t.Fatalf("first load hit=%v loads=%d value=%#v, want miss/1/fresh", hit, loads, first)
	}

	var second map[string]any
	hit, err = store.CacheGetOrLoad(context.Background(), "cache:test:l1", &second, time.Minute, loader)
	if err != nil {
		t.Fatal(err)
	}
	if !hit || loads != 1 || second["value"] != "fresh" {
		t.Fatalf("second load hit=%v loads=%d value=%#v, want hit/1/fresh", hit, loads, second)
	}
}

func TestRedisCacheBreakerSkipsRedisUntilCooldown(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	store := &RedisStore{
		cfg: normalizeRedisConfig(config.RedisConfig{
			CacheEnabled: true,
			CacheL1TTL:   time.Second,
		}),
		l1:  map[string]redisCacheItem{},
		now: func() time.Time { return now },
	}

	if store.cacheUsable() {
		t.Fatalf("store with nil redis client should not be usable")
	}

	store.client = nil
	store.noteCacheFailure()
	if store.cacheUsable() {
		t.Fatalf("cache should be in breaker cooldown")
	}

	now = now.Add(6 * time.Second)
	store.client = nil
	if store.cacheUsable() {
		t.Fatalf("nil client should remain unusable after cooldown")
	}
}

func TestCacheKeyIncludesVersionAndParts(t *testing.T) {
	if got := CacheKey("posts", 7, "recommended", 1, "type=all"); got != "cache:posts:v7:recommended:1:type=all" {
		t.Fatalf("CacheKey() = %q", got)
	}
	if got := CacheVersionKey("posts"); got != "cache:version:posts" {
		t.Fatalf("CacheVersionKey() = %q", got)
	}
}

func TestCacheKeyDigestsLongPartsWithXXH3(t *testing.T) {
	longPart := strings.Repeat("搜索词", 80)
	got := CacheKey("search", 3, longPart)
	if strings.Contains(got, longPart) {
		t.Fatalf("CacheKey() kept long raw part: %q", got)
	}
	if !strings.HasPrefix(got, "cache:search:v3:h3:") {
		t.Fatalf("CacheKey() = %q, want h3 digest part", got)
	}
}

func TestRedisCacheVersionsFallbackAndParse(t *testing.T) {
	store := &RedisStore{
		cfg: normalizeRedisConfig(config.RedisConfig{CacheEnabled: true}),
		now: time.Now,
	}
	versions := store.CacheVersions(context.Background(), "posts", "search", "posts")
	if len(versions) != 2 || versions["posts"] != 0 || versions["search"] != 0 {
		t.Fatalf("CacheVersions fallback = %#v, want zero versions for unique scopes", versions)
	}
	if got := redisInt64("42"); got != 42 {
		t.Fatalf("redisInt64 string = %d, want 42", got)
	}
	if got := redisInt64([]byte("17")); got != 17 {
		t.Fatalf("redisInt64 bytes = %d, want 17", got)
	}
}

func TestRedisL1MaxEntriesEvictsOldestExpiringItems(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	store := &RedisStore{
		cfg: normalizeRedisConfig(config.RedisConfig{
			CacheEnabled:      true,
			CacheDefaultTTL:   time.Minute,
			CacheL1TTL:        time.Minute,
			CacheL1MaxEntries: 2,
		}),
		l1:  map[string]redisCacheItem{},
		now: func() time.Time { return now },
	}

	store.setL1("cache:test:a", []byte(`{"value":"a"}`), time.Minute)
	now = now.Add(time.Millisecond)
	store.setL1("cache:test:b", []byte(`{"value":"b"}`), time.Minute)
	now = now.Add(time.Millisecond)
	store.setL1("cache:test:c", []byte(`{"value":"c"}`), time.Minute)

	if len(store.l1) != 2 {
		t.Fatalf("l1 size = %d, want 2", len(store.l1))
	}
	if _, ok := store.getL1("cache:test:a"); ok {
		t.Fatalf("oldest cache item should be evicted")
	}
	for _, key := range []string{"cache:test:b", "cache:test:c"} {
		if _, ok := store.getL1(key); !ok {
			t.Fatalf("%s should remain in l1", key)
		}
	}
}
