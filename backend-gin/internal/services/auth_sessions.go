package services

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const allSessionsKey = "all_sessions"

func (s *AuthService) CreateSession(ctx context.Context, session Session, ttl time.Duration) bool {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return false
	}
	id, err := s.redis.Client().Incr(ctx, sessionIDCounterKey).Result()
	if err != nil {
		return false
	}
	now := time.Now()
	if ttl <= 0 {
		ttl = s.TokenPolicy().RefreshTokenActiveTTL
	}
	if session.ExpiresAt == "" {
		session.ExpiresAt = now.Add(ttl).UTC().Format(time.RFC3339Nano)
	}
	if session.CreatedAt == "" {
		session.CreatedAt = now.UTC().Format(time.RFC3339Nano)
	}
	if session.LastActiveAt == "" {
		session.LastActiveAt = now.UTC().Format(time.RFC3339Nano)
	}
	session.ID = id
	session.IsActive = true

	if !s.redis.Set(ctx, sessionTokenKey(session.Token), session, ttl) {
		return false
	}
	s.redis.Set(ctx, sessionIDKey(id), session, ttl)
	if session.RefreshToken != "" {
		s.redis.Set(ctx, sessionRefreshKey(session.RefreshToken), session, ttl)
	}
	client := s.redis.Client()
	userSessionsKey := "user_sessions:" + session.UserID
	_ = client.SAdd(ctx, userSessionsKey, strconv.FormatInt(id, 10)).Err()
	_ = client.ExpireNX(ctx, userSessionsKey, ttl+time.Hour).Err()
	_ = client.ExpireGT(ctx, userSessionsKey, ttl+time.Hour).Err()
	_ = client.ZAdd(ctx, allSessionsKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: strconv.FormatInt(id, 10),
	}).Err()
	_ = client.ExpireNX(ctx, allSessionsKey, sessionIndexTTL(ttl)).Err()
	_ = client.ExpireGT(ctx, allSessionsKey, sessionIndexTTL(ttl)).Err()
	s.EnforceUserSessionLimit(ctx, session.UserID, s.SessionPolicy())
	return true
}

func (s *AuthService) FindSessionByRefreshToken(ctx context.Context, refreshToken string, userID int64) (*Session, bool) {
	if s == nil || s.redis == nil || refreshToken == "" {
		return nil, false
	}
	var session Session
	if !s.redis.GetJSON(ctx, sessionRefreshKey(refreshToken), &session) {
		return nil, false
	}
	if !session.IsActive {
		return nil, false
	}
	if userID != 0 && session.UserID != strconv.FormatInt(userID, 10) {
		return nil, false
	}
	if sessionShouldRemove(session, time.Now(), s.SessionPolicy()) {
		s.DeleteSession(ctx, session)
		return nil, false
	}
	return &session, true
}

func (s *AuthService) RefreshSessionAccessToken(ctx context.Context, session Session, accessToken string, now time.Time) bool {
	if s == nil || s.redis == nil || s.redis.Client() == nil || session.ID == 0 || accessToken == "" {
		return false
	}
	previousToken := session.Token
	session.Token = accessToken
	session.LastActiveAt = now.UTC().Format(time.RFC3339Nano)
	ttl := sessionTTL(session, s.TokenPolicy().RefreshTokenActiveTTL)
	if err := saveSessionWithClient(ctx, s.redis.Client(), session, ttl, false); err != nil {
		return false
	}
	if previousToken != "" && previousToken != accessToken {
		_ = s.redis.Client().Del(ctx, sessionTokenKey(previousToken)).Err()
	}
	return true
}

func (s *AuthService) RotateSessionTokens(ctx context.Context, session Session, accessToken string, refreshToken string, now time.Time) bool {
	if s == nil || s.redis == nil || s.redis.Client() == nil || session.ID == 0 || accessToken == "" || refreshToken == "" {
		return false
	}
	previousToken := session.Token
	previousRefreshToken := session.RefreshToken
	policy := s.TokenPolicy()
	ttl := policy.RefreshTokenActiveTTL
	session.Token = accessToken
	session.RefreshToken = refreshToken
	session.LastActiveAt = now.UTC().Format(time.RFC3339Nano)
	session.ExpiresAt = now.Add(ttl).UTC().Format(time.RFC3339Nano)
	if err := saveSessionWithClient(ctx, s.redis.Client(), session, ttl, false); err != nil {
		return false
	}
	pipe := s.redis.Client().Pipeline()
	cleanup := 0
	if previousToken != "" && previousToken != accessToken {
		pipe.Del(ctx, sessionTokenKey(previousToken))
		cleanup++
	}
	if previousRefreshToken != "" && previousRefreshToken != refreshToken {
		pipe.Del(ctx, sessionRefreshKey(previousRefreshToken))
		cleanup++
	}
	if cleanup == 0 {
		return true
	}
	_, err := pipe.Exec(ctx)
	return err == nil
}

func (s *AuthService) DeactivateSessionByToken(ctx context.Context, token string, userID int64) bool {
	if s == nil || s.redis == nil || token == "" {
		return false
	}
	var session Session
	if !s.redis.GetJSON(ctx, sessionTokenKey(token), &session) {
		return false
	}
	if userID != 0 && session.UserID != strconv.FormatInt(userID, 10) {
		return false
	}
	return s.DeleteSession(ctx, session)
}

func (s *AuthService) DeleteSession(ctx context.Context, session Session) bool {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return false
	}
	return deleteSessionWithClient(ctx, s.redis.Client(), session) == nil
}

func (s *AuthService) TouchSession(ctx context.Context, session Session, now time.Time) bool {
	if s == nil || s.redis == nil || s.redis.Client() == nil || session.ID == 0 || session.Token == "" {
		return false
	}
	lastActive := sessionActivityTime(session)
	if !lastActive.IsZero() && now.Sub(lastActive) < sessionTouchInterval {
		return true
	}
	session.LastActiveAt = now.UTC().Format(time.RFC3339Nano)
	ttl := sessionTTL(session, s.TokenPolicy().RefreshTokenActiveTTL)
	return saveSessionWithClient(ctx, s.redis.Client(), session, ttl, false) == nil
}

func (s *AuthService) EnforceUserSessionLimit(ctx context.Context, userID string, policy SessionPolicy) int {
	if s == nil || s.redis == nil || s.redis.Client() == nil {
		return 0
	}
	return enforceUserSessionLimit(ctx, s.redis.Client(), userID, policy)
}

func sessionShouldRemove(session Session, now time.Time, policy SessionPolicy) bool {
	if !session.IsActive {
		return true
	}
	if expiresAt := parseSessionTime(session.ExpiresAt); expiresAt.IsZero() || !expiresAt.After(now) {
		return true
	}
	if policy.InactiveTTL <= 0 {
		return false
	}
	lastActive := parseSessionTime(session.LastActiveAt)
	if lastActive.IsZero() {
		return false
	}
	return !lastActive.Add(policy.InactiveTTL).After(now)
}

func sessionActivityTime(session Session) time.Time {
	if lastActiveAt := parseSessionTime(session.LastActiveAt); !lastActiveAt.IsZero() {
		return lastActiveAt
	}
	return parseSessionTime(session.CreatedAt)
}

func sessionTTL(session Session, fallback time.Duration) time.Duration {
	expiresAt := parseSessionTime(session.ExpiresAt)
	if expiresAt.IsZero() {
		return fallback
	}
	ttl := time.Until(expiresAt)
	if ttl < time.Second {
		return time.Second
	}
	return ttl
}

func parseSessionTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func saveSessionWithClient(ctx context.Context, client *redis.Client, session Session, ttl time.Duration, _ bool) error {
	if client == nil {
		return redis.Nil
	}
	if ttl <= 0 {
		ttl = defaultRefreshSessionTTL
	}
	serialized, err := json.Marshal(session)
	if err != nil {
		return err
	}
	pipe := client.Pipeline()
	if session.Token != "" {
		pipe.Set(ctx, sessionTokenKey(session.Token), serialized, ttl)
	}
	if session.ID != 0 {
		id := strconv.FormatInt(session.ID, 10)
		pipe.Set(ctx, sessionIDKey(session.ID), serialized, ttl)
		pipe.SAdd(ctx, "user_sessions:"+session.UserID, id)
		pipe.ZAdd(ctx, allSessionsKey, redis.Z{Score: sessionCreatedScore(session), Member: id})
		pipe.ExpireNX(ctx, allSessionsKey, sessionIndexTTL(ttl))
		pipe.ExpireGT(ctx, allSessionsKey, sessionIndexTTL(ttl))
	}
	if session.RefreshToken != "" {
		pipe.Set(ctx, sessionRefreshKey(session.RefreshToken), serialized, ttl)
	}
	if session.UserID != "" {
		pipe.ExpireNX(ctx, "user_sessions:"+session.UserID, ttl+time.Hour)
		pipe.ExpireGT(ctx, "user_sessions:"+session.UserID, ttl+time.Hour)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func deleteSessionWithClient(ctx context.Context, client *redis.Client, session Session) error {
	if client == nil {
		return redis.Nil
	}
	pipe := client.Pipeline()
	if session.Token != "" {
		pipe.Del(ctx, sessionTokenKey(session.Token))
	}
	if session.RefreshToken != "" {
		pipe.Del(ctx, sessionRefreshKey(session.RefreshToken))
	}
	if session.ID != 0 {
		id := strconv.FormatInt(session.ID, 10)
		pipe.Del(ctx, sessionIDKey(session.ID))
		pipe.ZRem(ctx, allSessionsKey, id)
		if session.UserID != "" {
			pipe.SRem(ctx, "user_sessions:"+session.UserID, id)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func enforceUserSessionLimit(ctx context.Context, client *redis.Client, userID string, policy SessionPolicy) int {
	userID = strings.TrimSpace(userID)
	if client == nil || userID == "" || policy.UserActiveSessionLimit <= 0 {
		return 0
	}
	ids, err := client.SMembers(ctx, "user_sessions:"+userID).Result()
	if err != nil || len(ids) == 0 {
		return 0
	}
	sessions, staleIDs, loadErr := loadSessionsByIDs(ctx, client, ids)
	removed := 0
	if len(staleIDs) > 0 {
		removed += removeSessionIDsFromIndexes(ctx, client, userID, staleIDs)
	}
	if loadErr != nil {
		return removed
	}
	now := time.Now()
	active := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		if session.UserID != userID || sessionShouldRemove(session, now, policy) {
			if deleteSessionWithClient(ctx, client, session) == nil {
				removed++
			}
			continue
		}
		active = append(active, session)
	}
	for _, session := range sessionsOverLimit(active, policy.UserActiveSessionLimit) {
		if deleteSessionWithClient(ctx, client, session) == nil {
			removed++
		}
	}
	return removed
}

func loadSessionsByIDs(ctx context.Context, client *redis.Client, ids []string) ([]Session, []string, error) {
	if client == nil || len(ids) == 0 {
		return nil, nil, nil
	}
	sessions := make([]Session, 0, len(ids))
	staleIDs := make([]string, 0)
	for start := 0; start < len(ids); start += 200 {
		end := min(start+200, len(ids))
		keys := make([]string, 0, end-start)
		for _, id := range ids[start:end] {
			keys = append(keys, "session:id:"+id)
		}
		values, err := client.MGet(ctx, keys...).Result()
		if err != nil {
			return sessions, staleIDs, err
		}
		for index, value := range values {
			session, ok := parseRedisSessionValue(value)
			if !ok {
				staleIDs = append(staleIDs, ids[start+index])
				continue
			}
			sessions = append(sessions, session)
		}
	}
	return sessions, staleIDs, nil
}

func parseRedisSessionValue(value any) (Session, bool) {
	raw, ok := value.(string)
	if !ok || raw == "" {
		return Session{}, false
	}
	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return Session{}, false
	}
	if session.ID == 0 {
		return Session{}, false
	}
	return session, true
}

func removeSessionIDsFromIndexes(ctx context.Context, client *redis.Client, userID string, ids []string) int {
	if client == nil || len(ids) == 0 {
		return 0
	}
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	pipe := client.Pipeline()
	if strings.TrimSpace(userID) != "" {
		pipe.SRem(ctx, "user_sessions:"+userID, args...)
	}
	pipe.ZRem(ctx, allSessionsKey, args...)
	_, _ = pipe.Exec(ctx)
	return len(ids)
}

func sessionsOverLimit(sessions []Session, limit int) []Session {
	if limit <= 0 || len(sessions) <= limit {
		return nil
	}
	ordered := append([]Session(nil), sessions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := sessionActivityTime(ordered[i])
		right := sessionActivityTime(ordered[j])
		if !left.Equal(right) {
			return left.Before(right)
		}
		leftCreated := parseSessionTime(ordered[i].CreatedAt)
		rightCreated := parseSessionTime(ordered[j].CreatedAt)
		if !leftCreated.Equal(rightCreated) {
			return leftCreated.Before(rightCreated)
		}
		return ordered[i].ID < ordered[j].ID
	})
	return ordered[:len(ordered)-limit]
}

func sessionCreatedScore(session Session) float64 {
	createdAt := parseSessionTime(session.CreatedAt)
	if createdAt.IsZero() {
		return float64(time.Now().UnixMilli())
	}
	return float64(createdAt.UnixMilli())
}

func sessionIndexTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		ttl = defaultRefreshSessionTTL
	}
	return ttl + time.Hour
}

func sessionTokenKey(token string) string   { return sessionKeyPrefix + "token:" + token }
func sessionIDKey(id int64) string          { return sessionKeyPrefix + "id:" + strconv.FormatInt(id, 10) }
func sessionRefreshKey(token string) string { return sessionKeyPrefix + "refresh:" + token }
