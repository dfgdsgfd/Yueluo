package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"yuem-go/backend-gin/internal/http/response"
)

const (
	adminLoginMaxFailures  = 5
	adminLoginLockDuration = 30 * time.Minute
	adminLoginFailureTTL   = 30 * time.Minute
)

var adminLoginFailureScript = redis.NewScript(`
local lock_ttl = redis.call("PTTL", KEYS[2])
if lock_ttl > 0 then
  return {-1, lock_ttl}
end

local failures = redis.call("INCR", KEYS[1])
if failures == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
end

if failures >= tonumber(ARGV[1]) then
  redis.call("SET", KEYS[2], "1", "PX", ARGV[3])
  redis.call("DEL", KEYS[1])
  return {failures, tonumber(ARGV[3])}
end

return {failures, 0}
`)

type adminLoginLockState struct {
	FailedAttempts int
	Locked         bool
	RetryAfter     time.Duration
}

type adminLoginMemoryEntry struct {
	failures    int
	windowUntil time.Time
	lockedUntil time.Time
}

type adminLoginMemoryLimiter struct {
	mu      sync.Mutex
	entries map[string]adminLoginMemoryEntry
	now     func() time.Time
}

var adminLoginIPFallback = newAdminLoginMemoryLimiter()

func newAdminLoginMemoryLimiter() *adminLoginMemoryLimiter {
	return &adminLoginMemoryLimiter{
		entries: map[string]adminLoginMemoryEntry{},
		now:     time.Now,
	}
}

func (l *adminLoginMemoryLimiter) status(key string) adminLoginLockState {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry, ok := l.entries[key]
	if !ok {
		return adminLoginLockState{}
	}
	if entry.lockedUntil.After(now) {
		return adminLoginLockState{Locked: true, RetryAfter: entry.lockedUntil.Sub(now)}
	}
	if !entry.windowUntil.After(now) {
		delete(l.entries, key)
	}
	return adminLoginLockState{}
}

func (l *adminLoginMemoryLimiter) failure(key string) adminLoginLockState {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry := l.entries[key]
	if entry.lockedUntil.After(now) {
		return adminLoginLockState{Locked: true, RetryAfter: entry.lockedUntil.Sub(now)}
	}
	if !entry.windowUntil.After(now) {
		entry = adminLoginMemoryEntry{windowUntil: now.Add(adminLoginFailureTTL)}
	}
	entry.failures++
	if entry.failures >= adminLoginMaxFailures {
		entry.lockedUntil = now.Add(adminLoginLockDuration)
	}
	l.entries[key] = entry
	return adminLoginLockState{
		FailedAttempts: entry.failures,
		Locked:         entry.lockedUntil.After(now),
		RetryAfter:     max(entry.lockedUntil.Sub(now), 0),
	}
}

func (l *adminLoginMemoryLimiter) clear(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

func (h NativeHandlers) adminLoginIPLockStatus(ctx context.Context, ip string) adminLoginLockState {
	key := adminLoginIPKey(ip)
	state := adminLoginIPFallback.status(key)
	client := h.adminLoginRedisClient()
	if client == nil {
		return state
	}

	commandCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	ttl, err := client.PTTL(commandCtx, adminLoginLockKey(key)).Result()
	if err == nil && ttl > state.RetryAfter {
		state.Locked = ttl > 0
		state.RetryAfter = ttl
	}
	return state
}

func (h NativeHandlers) recordAdminLoginIPFailure(ctx context.Context, ip string) adminLoginLockState {
	key := adminLoginIPKey(ip)
	state := adminLoginIPFallback.failure(key)
	client := h.adminLoginRedisClient()
	if client == nil {
		return state
	}

	commandCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	raw, err := adminLoginFailureScript.Run(
		commandCtx,
		client,
		[]string{adminLoginFailureKey(key), adminLoginLockKey(key)},
		adminLoginMaxFailures,
		adminLoginFailureTTL.Milliseconds(),
		adminLoginLockDuration.Milliseconds(),
	).Result()
	if err != nil {
		return state
	}
	values, ok := raw.([]any)
	if !ok || len(values) != 2 {
		return state
	}
	failures, failuresOK := redisResultInt64(values[0])
	retryMS, retryOK := redisResultInt64(values[1])
	if !failuresOK || !retryOK {
		return state
	}
	if failures > int64(state.FailedAttempts) {
		state.FailedAttempts = int(failures)
	}
	if retryMS > 0 {
		retryAfter := time.Duration(retryMS) * time.Millisecond
		state.Locked = true
		if retryAfter > state.RetryAfter {
			state.RetryAfter = retryAfter
		}
	}
	return state
}

func (h NativeHandlers) clearAdminLoginIPFailures(ctx context.Context, ip string) {
	key := adminLoginIPKey(ip)
	adminLoginIPFallback.clear(key)
	client := h.adminLoginRedisClient()
	if client == nil {
		return
	}
	commandCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	_ = client.Del(commandCtx, adminLoginFailureKey(key), adminLoginLockKey(key)).Err()
}

func (h NativeHandlers) adminLoginRedisClient() *redis.Client {
	if h.Redis == nil {
		return nil
	}
	return h.Redis.Client()
}

func (h NativeHandlers) writeAdminLoginLocked(c *gin.Context, ip string, username string, adminID *int64, lock adminLoginLockState) {
	retryAfterSeconds := max(1, int((lock.RetryAfter+time.Second-1)/time.Second))
	h.recordSecurityAudit(c, "admin_login", "password", "failure", "ip_locked", http.StatusTooManyRequests, adminID, "admin", username, gin.H{
		"failed_attempts":     lock.FailedAttempts,
		"ip_hash":             adminLoginIPKey(ip),
		"ip_scope":            true,
		"retry_after_seconds": retryAfterSeconds,
	})
	c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
	response.JSON(c, http.StatusTooManyRequests, response.CodeTooManyRequests, "error.admin_login_locked", gin.H{
		"retry_after_seconds": retryAfterSeconds,
	})
}

func adminLoginIPKey(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:16])
}

func adminLoginFailureKey(ipKey string) string {
	return fmt.Sprintf("security:admin-login:ip:%s:failures", ipKey)
}

func adminLoginLockKey(ipKey string) string {
	return fmt.Sprintf("security:admin-login:ip:%s:locked", ipKey)
}

func redisResultInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	case []byte:
		parsed, err := strconv.ParseInt(string(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
