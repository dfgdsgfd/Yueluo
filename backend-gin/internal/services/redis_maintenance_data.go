package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s *RedisMaintenanceService) trimObservability(ctx context.Context, cfg RedisMaintenanceConfig) (map[string]int64, error) {
	client := s.redis.Client()
	counts := map[string]int64{}
	var errs error
	now := time.Now()
	for _, item := range []struct {
		key       string
		retention time.Duration
		max       int64
		category  string
	}{
		{accessLogStreamKey, time.Duration(cfg.AccessLogRetentionHours) * time.Hour, int64(cfg.AccessLogMaxEntries), "access_stream_trimmed"},
		{systemLogStreamKey, time.Duration(cfg.SystemLogRetentionHours) * time.Hour, int64(cfg.SystemLogMaxEntries), "system_stream_trimmed"},
	} {
		before, _ := client.XLen(ctx, item.key).Result()
		pipe := client.Pipeline()
		trimRedisStream(ctx, pipe, item.key, item.retention, item.max, now)
		if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
			errs = errors.Join(errs, err)
			continue
		}
		after, _ := client.XLen(ctx, item.key).Result()
		if before > after {
			counts[item.category] += before - after
		}
		_ = client.Expire(ctx, item.key, item.retention+time.Hour).Err()
	}
	for _, item := range []struct {
		key       string
		retention time.Duration
		max       int64
		category  string
	}{
		{requestMetricZSetKey, time.Duration(cfg.MetricsRetentionHours) * time.Hour, int64(cfg.MetricsMaxEntriesPerKey), "request_metrics_deleted"},
		{slowRequestZSetKey, time.Duration(cfg.MetricsRetentionHours) * time.Hour, int64(cfg.MetricsMaxEntriesPerKey), "slow_requests_deleted"},
		{slowQueryZSetKey, time.Duration(cfg.MetricsRetentionHours) * time.Hour, int64(cfg.MetricsMaxEntriesPerKey), "slow_queries_deleted"},
		{postgresMetricZSetKey, time.Duration(cfg.MetricsRetentionHours) * time.Hour, int64(cfg.MetricsMaxEntriesPerKey), "postgres_metrics_deleted"},
		{runtimeMetricZSetKey, time.Duration(cfg.MetricsRetentionHours) * time.Hour, int64(cfg.MetricsMaxEntriesPerKey), "runtime_metrics_deleted"},
		{queueEventZSetKey, time.Duration(cfg.QueueEventRetentionHours) * time.Hour, int64(cfg.QueueEventMaxEntries), "queue_events_deleted"},
	} {
		before, _ := client.ZCard(ctx, item.key).Result()
		pipe := client.Pipeline()
		trimRedisZSet(ctx, pipe, item.key, item.retention, item.max, now)
		if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
			errs = errors.Join(errs, err)
			continue
		}
		after, _ := client.ZCard(ctx, item.key).Result()
		if before > after {
			counts[item.category] += before - after
		}
	}
	return counts, errs
}

func trimRedisStream(ctx context.Context, pipe redis.Pipeliner, key string, retention time.Duration, maxEntries int64, now time.Time) {
	if pipe == nil || key == "" {
		return
	}
	if retention > 0 {
		cutoff := strconv.FormatInt(now.Add(-retention).UnixMilli(), 10) + "-0"
		pipe.Do(ctx, "XTRIM", key, "MINID", "~", cutoff)
		pipe.Expire(ctx, key, retention+time.Hour)
	}
	if maxEntries > 0 {
		pipe.XTrimMaxLen(ctx, key, maxEntries)
	}
}

func trimRedisZSet(ctx context.Context, pipe redis.Pipeliner, key string, retention time.Duration, maxEntries int64, now time.Time) {
	if pipe == nil || key == "" {
		return
	}
	if retention > 0 {
		cutoff := strconv.FormatInt(now.Add(-retention).UnixMilli(), 10)
		pipe.ZRemRangeByScore(ctx, key, "0", cutoff)
		pipe.Expire(ctx, key, retention+time.Hour)
	}
	if maxEntries > 0 {
		pipe.ZRemRangeByRank(ctx, key, 0, -(maxEntries + 1))
	}
}

func (s *RedisMaintenanceService) cleanupSessionIndexes(ctx context.Context, cfg RedisMaintenanceConfig) (map[string]int64, error) {
	client := s.redis.Client()
	counts := map[string]int64{}
	var errs error
	now := time.Now()
	policy := SessionPolicy{
		InactiveTTL:            time.Duration(cfg.SessionInactiveTTLSeconds) * time.Second,
		UserActiveSessionLimit: cfg.UserActiveSessionLimit,
	}
	tokenPolicy := TokenPolicy{
		AccessTokenTTL:               time.Duration(cfg.AccessTokenTTLSeconds) * time.Second,
		RefreshTokenActiveTTL:        time.Duration(cfg.RefreshTokenActiveTTLSeconds) * time.Second,
		RefreshTokenRenewalInterval:  time.Duration(cfg.RefreshTokenRenewalIntervalSeconds) * time.Second,
		RefreshTokenMode:             normalizeRefreshTokenMode(cfg.RefreshTokenMode),
		RefreshTokenAutoRenewEnabled: cfg.RefreshTokenAutoRenewEnabled,
	}
	renewCutoff := now.Add(-tokenPolicy.RefreshTokenRenewalInterval)
	refreshRenewalEnabled := tokenPolicy.RefreshTokenAutoRenewEnabled && tokenPolicy.RefreshTokenMode == RefreshTokenModeRedisOpaque
	renewalSeen := map[int64]struct{}{}
	var cursor uint64
	for {
		values, next, err := client.ZScan(ctx, allSessionsKey, cursor, "*", 500).Result()
		if err != nil && err != redis.Nil {
			errs = errors.Join(errs, err)
			break
		}
		ids := make([]string, 0, len(values)/2)
		for i := 0; i+1 < len(values); i += 2 {
			ids = append(ids, values[i])
		}
		sessions, stale, loadErr := loadSessionsByIDs(ctx, client, ids)
		if loadErr != nil {
			errs = errors.Join(errs, loadErr)
		}
		if len(stale) > 0 {
			removed, removeErr := client.ZRem(ctx, allSessionsKey, stringSliceToInterfaces(stale)...).Result()
			counts["all_sessions_removed"] += removed
			errs = errors.Join(errs, removeErr)
		}
		for _, session := range sessions {
			if !sessionShouldRemove(session, now, policy) {
				applyRefreshRenewal(ctx, client, session, now, renewCutoff, tokenPolicy, renewalSeen, refreshRenewalEnabled, counts, &errs)
				continue
			}
			if deleteErr := deleteSessionWithClient(ctx, client, session); deleteErr != nil {
				errs = errors.Join(errs, deleteErr)
				continue
			}
			counts["sessions_deleted"]++
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	refreshAllSessionsIndexTTL(ctx, client, cfg, counts, &errs)

	cursor = 0
	for {
		keys, next, err := client.Scan(ctx, cursor, "user_sessions:*", 200).Result()
		if err != nil {
			errs = errors.Join(errs, err)
			break
		}
		for _, key := range keys {
			ids, membersErr := client.SMembers(ctx, key).Result()
			if membersErr != nil {
				errs = errors.Join(errs, membersErr)
				continue
			}
			userID := strings.TrimPrefix(key, "user_sessions:")
			sessions, stale, loadErr := loadSessionsByIDs(ctx, client, ids)
			if loadErr != nil {
				errs = errors.Join(errs, loadErr)
				continue
			}
			if len(stale) > 0 {
				removed, removeErr := client.SRem(ctx, key, stringSliceToInterfaces(stale)...).Result()
				counts["user_sessions_removed"] += removed
				errs = errors.Join(errs, removeErr)
			}
			active := make([]Session, 0, len(sessions))
			for _, session := range sessions {
				if session.UserID != userID {
					removed, removeErr := client.SRem(ctx, key, strconv.FormatInt(session.ID, 10)).Result()
					counts["user_sessions_removed"] += removed
					errs = errors.Join(errs, removeErr)
					continue
				}
				if sessionShouldRemove(session, now, policy) {
					if deleteErr := deleteSessionWithClient(ctx, client, session); deleteErr != nil {
						errs = errors.Join(errs, deleteErr)
						continue
					}
					counts["sessions_deleted"]++
					continue
				}
				applyRefreshRenewal(ctx, client, session, now, renewCutoff, tokenPolicy, renewalSeen, refreshRenewalEnabled, counts, &errs)
				active = append(active, session)
			}
			for _, session := range sessionsOverLimit(active, policy.UserActiveSessionLimit) {
				if deleteErr := deleteSessionWithClient(ctx, client, session); deleteErr != nil {
					errs = errors.Join(errs, deleteErr)
					continue
				}
				counts["user_sessions_over_limit_deleted"]++
			}
			size, _ := client.SCard(ctx, key).Result()
			if size == 0 {
				if deleted, unlinkErr := client.Unlink(ctx, key).Result(); unlinkErr == nil {
					counts["empty_user_session_keys_deleted"] += deleted
				} else {
					errs = errors.Join(errs, unlinkErr)
				}
			} else {
				_ = client.Expire(ctx, key, 31*24*time.Hour).Err()
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return counts, errs
}

func refreshAllSessionsIndexTTL(ctx context.Context, client *redis.Client, cfg RedisMaintenanceConfig, counts map[string]int64, errs *error) {
	if client == nil {
		return
	}
	size, err := client.ZCard(ctx, allSessionsKey).Result()
	if err != nil && err != redis.Nil {
		*errs = errors.Join(*errs, err)
		return
	}
	if size == 0 {
		deleted, unlinkErr := client.Unlink(ctx, allSessionsKey).Result()
		if unlinkErr != nil && unlinkErr != redis.Nil {
			*errs = errors.Join(*errs, unlinkErr)
			return
		}
		counts["empty_all_sessions_deleted"] += deleted
		return
	}
	ttl := sessionIndexTTL(time.Duration(cfg.RefreshTokenActiveTTLSeconds) * time.Second)
	if expireErr := client.Expire(ctx, allSessionsKey, ttl).Err(); expireErr != nil && expireErr != redis.Nil {
		*errs = errors.Join(*errs, expireErr)
	}
}

func applyRefreshRenewal(ctx context.Context, client *redis.Client, session Session, now time.Time, renewCutoff time.Time, policy TokenPolicy, seen map[int64]struct{}, enabled bool, counts map[string]int64, errs *error) {
	if session.ID != 0 {
		if _, ok := seen[session.ID]; ok {
			return
		}
		seen[session.ID] = struct{}{}
	}
	if !enabled {
		counts["refresh_sessions_renewal_disabled"]++
		return
	}
	renewed, inactive, renewErr := renewActiveRefreshSession(ctx, client, session, now, renewCutoff, policy)
	if renewed {
		counts["refresh_sessions_renewed"]++
	}
	if inactive {
		counts["refresh_sessions_inactive"]++
	}
	if renewErr != nil {
		*errs = errors.Join(*errs, renewErr)
	}
}

func renewActiveRefreshSessionOnce(ctx context.Context, client *redis.Client, session Session, now time.Time, renewCutoff time.Time, policy TokenPolicy, seen map[int64]struct{}) (bool, bool, error) {
	if session.ID != 0 {
		if _, ok := seen[session.ID]; ok {
			return false, false, nil
		}
		seen[session.ID] = struct{}{}
	}
	return renewActiveRefreshSession(ctx, client, session, now, renewCutoff, policy)
}

func renewActiveRefreshSession(ctx context.Context, client *redis.Client, session Session, now time.Time, renewCutoff time.Time, policy TokenPolicy) (bool, bool, error) {
	if client == nil || session.ID == 0 || session.RefreshToken == "" || policy.RefreshTokenActiveTTL <= 0 {
		return false, false, nil
	}
	lastActive := sessionActivityTime(session)
	if lastActive.IsZero() || lastActive.Before(renewCutoff) {
		return false, true, nil
	}
	nextExpiry := now.Add(policy.RefreshTokenActiveTTL)
	currentExpiry := parseSessionTime(session.ExpiresAt)
	if !currentExpiry.IsZero() && currentExpiry.After(nextExpiry.Add(-time.Second)) {
		return false, false, nil
	}
	session.ExpiresAt = nextExpiry.UTC().Format(time.RFC3339Nano)
	if session.LastActiveAt == "" {
		session.LastActiveAt = lastActive.UTC().Format(time.RFC3339Nano)
	}
	if err := saveSessionWithClient(ctx, client, session, policy.RefreshTokenActiveTTL, false); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func (s *RedisMaintenanceService) clearApplicationCache(ctx context.Context) (int64, error) {
	client := s.redis.Client()
	var cursor uint64
	var deleted int64
	for {
		keys, next, err := client.Scan(ctx, cursor, "cache:*", 500).Result()
		if err != nil {
			return deleted, err
		}
		if len(keys) > 0 {
			count, unlinkErr := client.Unlink(ctx, keys...).Result()
			deleted += count
			if unlinkErr != nil {
				return deleted, unlinkErr
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	s.redis.clearL1()
	return deleted, nil
}
