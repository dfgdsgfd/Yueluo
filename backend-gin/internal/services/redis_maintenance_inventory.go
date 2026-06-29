package services

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisInventory(ctx context.Context, client *redis.Client, limit int) RedisInventory {
	inventory := RedisInventory{Categories: map[string]RedisInventoryCategory{}}
	inventory.DatabaseKeys, _ = client.DBSize(ctx).Result()
	var cursor uint64
	keys := make([]string, 0, min(limit, 1000))
	for len(keys) < limit {
		batch, next, err := client.Scan(ctx, cursor, "*", 500).Result()
		if err != nil {
			break
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(keys) > limit {
		keys = keys[:limit]
	}
	inventory.ScannedKeys = len(keys)
	inventory.Truncated = cursor != 0 || int64(len(keys)) < inventory.DatabaseKeys
	for start := 0; start < len(keys); start += 200 {
		end := min(start+200, len(keys))
		pipe := client.Pipeline()
		commands := make([]*redis.IntCmd, 0, end-start)
		ttlCommands := make([]*redis.DurationCmd, 0, end-start)
		for _, key := range keys[start:end] {
			commands = append(commands, pipe.MemoryUsage(ctx, key))
			ttlCommands = append(ttlCommands, pipe.TTL(ctx, key))
		}
		_, _ = pipe.Exec(ctx)
		for index, key := range keys[start:end] {
			bytes, _ := commands[index].Result()
			ttl, _ := ttlCommands[index].Result()
			category := redisKeyCategory(key)
			item := inventory.Categories[category]
			item.Keys++
			if bytes > 0 {
				item.MemoryBytes += bytes
			}
			if redisTTLNoExpiration(ttl) {
				if redisExpectedPermanentKey(key) {
					item.ExpectedPermanentKeys++
					inventory.ExpectedPermanentKeys++
				} else {
					item.NoTTLKeys++
					inventory.NoTTLKeys++
				}
			}
			inventory.Categories[category] = item
		}
	}
	return inventory
}

func redisKeyCategory(key string) string {
	switch {
	case strings.HasPrefix(key, "asynq:"):
		return "queues"
	case key == accessLogStreamKey || key == systemLogStreamKey:
		return "logs"
	case key == requestMetricZSetKey || key == slowRequestZSetKey || key == slowQueryZSetKey ||
		key == postgresMetricZSetKey || key == runtimeMetricZSetKey || key == queueEventZSetKey:
		return "metrics"
	case strings.HasPrefix(key, "session:") || key == allSessionsKey || strings.HasPrefix(key, "user_sessions:"):
		return "sessions"
	case strings.HasPrefix(key, "cache:"):
		return "cache"
	case strings.HasPrefix(key, SettingsKeyPrefix):
		return "settings"
	case strings.HasPrefix(key, "admin_login:"):
		return "security"
	default:
		return "other"
	}
}

func redisTTLNoExpiration(ttl time.Duration) bool {
	return ttl == -time.Second || ttl == -time.Nanosecond
}

func redisExpectedPermanentKey(key string) bool {
	return key == sessionIDCounterKey ||
		strings.HasPrefix(key, SettingsKeyPrefix) ||
		strings.HasPrefix(key, "cache:version:")
}

func redisMemoryPressure(status map[string]any, cfg RedisMaintenanceConfig) map[string]any {
	pressure := map[string]any{
		"used_bytes": int64(0), "max_bytes": int64(0), "percent": float64(0),
		"level": "unbounded", "warning": cfg.MemoryWarningPercent, "critical": cfg.MemoryCriticalPercent,
	}
	info, _ := status["info"].(map[string]any)
	used := redisInfoInt64(info["used_memory"])
	maxMemory := redisInfoInt64(info["maxmemory"])
	pressure["used_bytes"] = used
	pressure["max_bytes"] = maxMemory
	if maxMemory <= 0 {
		return pressure
	}
	percent := float64(used) * 100 / float64(maxMemory)
	pressure["percent"] = percent
	pressure["level"] = "normal"
	if percent >= float64(cfg.MemoryCriticalPercent) {
		pressure["level"] = "critical"
	} else if percent >= float64(cfg.MemoryWarningPercent) {
		pressure["level"] = "warning"
	}
	return pressure
}

func releaseRedisMaintenanceLock(ctx context.Context, client *redis.Client, token string) {
	script := redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`)
	_, _ = script.Run(ctx, client, []string{redisMaintenanceLockKey}, token).Result()
}

func missingSessionIDs(ctx context.Context, client *redis.Client, ids []string) ([]string, error) {
	stale := make([]string, 0)
	for start := 0; start < len(ids); start += 200 {
		end := min(start+200, len(ids))
		keys := make([]string, 0, end-start)
		for _, id := range ids[start:end] {
			keys = append(keys, "session:id:"+id)
		}
		values, err := client.MGet(ctx, keys...).Result()
		if err != nil {
			return stale, err
		}
		for index, value := range values {
			if value == nil {
				stale = append(stale, ids[start+index])
			}
		}
	}
	return stale, nil
}

func normalizeMaintenanceCategories(categories []string) map[string]bool {
	if len(categories) == 0 {
		return map[string]bool{"observability": true, "queue_history": true, "sessions": true}
	}
	out := map[string]bool{}
	for _, category := range categories {
		switch strings.TrimSpace(category) {
		case "observability", "queue_history", "sessions", "cache":
			out[strings.TrimSpace(category)] = true
		}
	}
	return out
}

func mergeMaintenanceCounts(target, source map[string]int64) {
	for key, value := range source {
		target[key] += value
	}
}

func maintenanceTimeDue(raw string, now time.Time) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	next, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		next, err = time.Parse(time.RFC3339, raw)
	}
	return err != nil || !now.Before(next)
}

func redisUsedMemory(ctx context.Context, client *redis.Client) int64 {
	raw, err := client.Info(ctx, "memory").Result()
	if err != nil {
		return 0
	}
	return redisInfoInt64(parseRedisInfo(raw)["used_memory"])
}

func redisInfoInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	default:
		return 0
	}
}

func stringSliceToInterfaces(values []string) []any {
	out := make([]any, len(values))
	for index, value := range values {
		out[index] = value
	}
	return out
}

func clampRedisInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func containsRedisString(values []string, target string) bool {
	return slices.Contains(values, target)
}
