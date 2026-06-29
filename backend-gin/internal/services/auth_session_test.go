package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"yuem-go/backend-gin/internal/config"
)

func TestSessionPolicyReadsDefaultsAndSettings(t *testing.T) {
	defaultPolicy := ReadSessionPolicy(NewSettingsService(nil, nil))
	if defaultPolicy.InactiveTTL != 7*24*time.Hour {
		t.Fatalf("default inactive ttl = %s, want 168h", defaultPolicy.InactiveTTL)
	}
	if defaultPolicy.UserActiveSessionLimit != 5 {
		t.Fatalf("default user session limit = %d, want 5", defaultPolicy.UserActiveSessionLimit)
	}

	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "session_inactive_ttl_seconds", int((14 * 24 * time.Hour).Seconds()))
	_ = settings.Set(context.Background(), "redis_user_active_session_limit", 9)
	policy := ReadSessionPolicy(settings)
	if policy.InactiveTTL != 14*24*time.Hour || policy.UserActiveSessionLimit != 9 {
		t.Fatalf("custom session policy = %+v", policy)
	}
}

func TestTokenPolicyReadsSecondsAndGeneratesOpaqueRefreshToken(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "access_token_ttl_seconds", 120)
	_ = settings.Set(context.Background(), "refresh_token_active_ttl_seconds", 7200)
	_ = settings.Set(context.Background(), "refresh_token_renewal_interval_seconds", 3600)
	_ = settings.Set(context.Background(), "refresh_token_mode", RefreshTokenModeRedisOpaque)
	_ = settings.Set(context.Background(), "refresh_token_auto_renew_enabled", true)

	auth := NewAuthService(nil, nil, config.AuthConfig{
		JWTSecret:             "test-secret",
		JWTExpiresIn:          "3600",
		RefreshTokenExpiresIn: "7776000",
	}, settings)
	policy := auth.TokenPolicy()
	if policy.AccessTokenTTL != 120*time.Second ||
		policy.RefreshTokenActiveTTL != 2*time.Hour ||
		policy.RefreshTokenRenewalInterval != time.Hour ||
		policy.RefreshTokenMode != RefreshTokenModeRedisOpaque ||
		!policy.RefreshTokenAutoRenewEnabled {
		t.Fatalf("token policy = %+v", policy)
	}

	refresh, err := auth.GenerateRefreshToken(map[string]any{"userId": 42})
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if _, ok := auth.VerifyTokenClaims(refresh); ok {
		t.Fatal("opaque refresh token should not verify as a JWT")
	}
	if len(refresh) < 64 {
		t.Fatalf("opaque refresh token length = %d, want at least 64 hex chars", len(refresh))
	}
}

func TestGenerateRefreshTokenUsesJWTLegacyMode(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "refresh_token_mode", RefreshTokenModeJWTLegacy)
	_ = settings.Set(context.Background(), "refresh_token_active_ttl_seconds", 7200)
	_ = settings.Set(context.Background(), "refresh_token_auto_renew_enabled", false)
	auth := NewAuthService(nil, nil, config.AuthConfig{JWTSecret: "test-secret"}, settings)

	policy := auth.TokenPolicy()
	if policy.RefreshTokenMode != RefreshTokenModeJWTLegacy || policy.RefreshTokenAutoRenewEnabled {
		t.Fatalf("token policy switch = %+v", policy)
	}
	refresh, err := auth.GenerateRefreshToken(map[string]any{"userId": 42, "user_id": "u42"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	claims, ok := auth.VerifyTokenClaims(refresh)
	if !ok {
		t.Fatal("jwt legacy refresh token should verify")
	}
	if claims["token_type"] != "refresh" {
		t.Fatalf("token_type = %#v, want refresh", claims["token_type"])
	}
	if userID, ok := numericClaim(claims["userId"]); !ok || userID != 42 {
		t.Fatalf("userId claim = %#v", claims["userId"])
	}
}

func TestRefreshSessionAccessTokenRotatesAccessOnly(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	store := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(ctx, "refresh_token_active_ttl_seconds", 7200)
	auth := NewAuthService(nil, store, config.AuthConfig{JWTSecret: "test-secret"}, settings)

	oldAccess, err := auth.GenerateAccessToken(map[string]any{"userId": 42, "user_id": "u42"})
	if err != nil {
		t.Fatalf("GenerateAccessToken old error: %v", err)
	}
	refresh, err := auth.GenerateRefreshToken(nil)
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if !auth.CreateSession(ctx, Session{UserID: "42", Token: oldAccess, RefreshToken: refresh}, 2*time.Hour) {
		t.Fatal("CreateSession failed")
	}
	session, ok := auth.FindSessionByRefreshToken(ctx, refresh, 42)
	if !ok {
		t.Fatal("session should be found by refresh token")
	}
	newAccess, err := auth.GenerateAccessToken(map[string]any{"userId": 42, "user_id": "u42"})
	if err != nil {
		t.Fatalf("GenerateAccessToken new error: %v", err)
	}
	if !auth.RefreshSessionAccessToken(ctx, *session, newAccess, time.Now()) {
		t.Fatal("RefreshSessionAccessToken failed")
	}
	if _, ok := store.GetRaw(ctx, sessionTokenKey(oldAccess)); ok {
		t.Fatal("old access token session key should be removed")
	}
	var newAccessSession Session
	if !store.GetJSON(ctx, sessionTokenKey(newAccess), &newAccessSession) || newAccessSession.RefreshToken != refresh {
		t.Fatalf("new access token should point at same refresh session, got %+v", newAccessSession)
	}
	refreshed, ok := auth.FindSessionByRefreshToken(ctx, refresh, 42)
	if !ok || refreshed.RefreshToken != refresh || refreshed.Token != newAccess {
		t.Fatalf("refresh token should be preserved with new access token, got %+v ok=%v", refreshed, ok)
	}
}

func TestSessionShouldRemoveInactiveSession(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	session := Session{
		ID:           1,
		UserID:       "42",
		Token:        "access",
		IsActive:     true,
		ExpiresAt:    now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		CreatedAt:    now.Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano),
		LastActiveAt: now.Add(-8 * 24 * time.Hour).Format(time.RFC3339Nano),
	}
	if !sessionShouldRemove(session, now, SessionPolicy{InactiveTTL: 7 * 24 * time.Hour, UserActiveSessionLimit: 5}) {
		t.Fatal("session inactive for more than policy ttl should be removed")
	}
	session.LastActiveAt = now.Add(-6 * 24 * time.Hour).Format(time.RFC3339Nano)
	if sessionShouldRemove(session, now, SessionPolicy{InactiveTTL: 7 * 24 * time.Hour, UserActiveSessionLimit: 5}) {
		t.Fatal("recently active session should be retained")
	}
}

func TestSessionsOverLimitDeletesOldestActivity(t *testing.T) {
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	sessions := []Session{
		{ID: 1, CreatedAt: base.Add(-4 * time.Hour).Format(time.RFC3339Nano), LastActiveAt: base.Add(-4 * time.Hour).Format(time.RFC3339Nano)},
		{ID: 2, CreatedAt: base.Add(-3 * time.Hour).Format(time.RFC3339Nano), LastActiveAt: base.Add(-30 * time.Minute).Format(time.RFC3339Nano)},
		{ID: 3, CreatedAt: base.Add(-2 * time.Hour).Format(time.RFC3339Nano), LastActiveAt: base.Add(-2 * time.Hour).Format(time.RFC3339Nano)},
		{ID: 4, CreatedAt: base.Add(-1 * time.Hour).Format(time.RFC3339Nano), LastActiveAt: base.Add(-10 * time.Minute).Format(time.RFC3339Nano)},
	}
	victims := sessionsOverLimit(sessions, 2)
	if len(victims) != 2 || victims[0].ID != 1 || victims[1].ID != 3 {
		t.Fatalf("victims = %+v, want oldest active sessions 1 and 3", victims)
	}
}
