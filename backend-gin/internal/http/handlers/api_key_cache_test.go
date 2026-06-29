package handlers

import (
	"testing"
	"time"
)

func TestAPIKeyAuthCacheSeparatesBoundedPositiveAndNegativeEntries(t *testing.T) {
	now := time.Now()
	cache := newAPIKeyAuthCache(2, 1, 2, 2, time.Minute, time.Minute, time.Minute, 3, time.Minute, time.Minute)

	cache.StorePositive(openAPIKeyScope, "positive-1", 1, "first", now)
	cache.StorePositive(openAPIKeyScope, "positive-2", 2, "second", now)
	cache.StoreNegative(openAPIKeyScope, "negative-1", 0, now)
	cache.StoreNegative(openAPIKeyScope, "negative-2", 0, now)

	if value, state := cache.Lookup(openAPIKeyScope, "positive-1", now); state != apiKeyCachePositive || value != "first" {
		t.Fatalf("positive entry was displaced by negative cache: state=%v value=%#v", state, value)
	}
	if _, state := cache.Lookup(openAPIKeyScope, "negative-1", now); state != apiKeyCacheMiss {
		t.Fatalf("oldest negative entry state=%v, want miss after bounded eviction", state)
	}
	if _, state := cache.Lookup(openAPIKeyScope, "negative-2", now); state != apiKeyCacheNegative {
		t.Fatalf("newest negative entry state=%v, want negative hit", state)
	}
}

func TestHashAPIKeyPreservesSHA256StorageFormat(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := hashAPIKey("abc"); got != want {
		t.Fatalf("hashAPIKey() = %q, want existing SHA-256 format %q", got, want)
	}
}

func TestAPIKeyAuthCacheExpiresAndInvalidatesPositiveEntries(t *testing.T) {
	now := time.Now()
	cache := newAPIKeyAuthCache(2, 2, 2, 2, time.Second, time.Second, time.Minute, 3, time.Minute, time.Minute)

	cache.StorePositive(openAPIKeyScope, "digest", 42, "value", now)
	cache.InvalidateIdentity(openAPIKeyScope, 42)
	if _, state := cache.Lookup(openAPIKeyScope, "digest", now); state != apiKeyCacheMiss {
		t.Fatalf("invalidated positive entry state=%v, want miss", state)
	}

	cache.StoreNegative(userAPIKeyScope, "missing", 0, now)
	if _, state := cache.Lookup(userAPIKeyScope, "missing", now.Add(2*time.Second)); state != apiKeyCacheMiss {
		t.Fatalf("expired negative entry state=%v, want miss", state)
	}
}

func TestAPIKeyAuthCacheRateLimitsInvalidAttempts(t *testing.T) {
	now := time.Now()
	cache := newAPIKeyAuthCache(2, 2, 2, 2, time.Minute, time.Minute, time.Minute, 3, 5*time.Minute, time.Minute)

	if cache.RecordInvalid("client", now) {
		t.Fatal("first invalid attempt should not block")
	}
	if cache.RecordInvalid("client", now.Add(time.Second)) {
		t.Fatal("second invalid attempt should not block")
	}
	if !cache.RecordInvalid("client", now.Add(2*time.Second)) {
		t.Fatal("third invalid attempt should block")
	}
	if !cache.IsBlocked("client", now.Add(3*time.Second)) {
		t.Fatal("client should remain blocked during block TTL")
	}
	cache.ResetInvalid("client")
	if cache.IsBlocked("client", now.Add(4*time.Second)) {
		t.Fatal("successful authentication should reset invalid-attempt limiter")
	}
}

func TestAPIKeyAuthCacheDebouncesLastUsedTouch(t *testing.T) {
	now := time.Now()
	cache := newAPIKeyAuthCache(2, 2, 2, 2, time.Minute, time.Minute, time.Minute, 3, time.Minute, 10*time.Minute)

	if !cache.ReserveTouch(openAPIKeyScope, 7, now) {
		t.Fatal("first touch should be reserved")
	}
	if cache.ReserveTouch(openAPIKeyScope, 7, now.Add(time.Minute)) {
		t.Fatal("touch inside debounce interval should be skipped")
	}
	if !cache.ReserveTouch(openAPIKeyScope, 7, now.Add(11*time.Minute)) {
		t.Fatal("touch after debounce interval should be reserved")
	}

	reservedAt := now.Add(11 * time.Minute)
	cache.ReleaseTouch(openAPIKeyScope, 7, reservedAt)
	if !cache.ReserveTouch(openAPIKeyScope, 7, reservedAt.Add(time.Second)) {
		t.Fatal("failed database update should release touch reservation")
	}
}
