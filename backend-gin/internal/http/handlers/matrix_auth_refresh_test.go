package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestAuthRefreshRotatesAccessTokenAndPreservesRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	db := newAuthRefreshTestDB(t)
	redisServer := miniredis.RunT(t)
	store := services.NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	settings := services.NewSettingsService(nil, nil)
	_ = settings.Set(ctx, "access_token_ttl_seconds", 120)
	_ = settings.Set(ctx, "refresh_token_active_ttl_seconds", 7200)
	authConfig := config.AuthConfig{JWTSecret: "test-secret", JWTExpiresIn: "3600", RefreshTokenExpiresIn: "7776000"}
	auth := services.NewAuthService(db, store, authConfig, settings)
	audit := newBufferedAuditLogForRefreshTest()
	user := domain.User{UserID: "refresh-user", Nickname: "Refresh User", IsActive: true, CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	oldAccess, err := auth.GenerateAccessToken(map[string]any{"userId": user.ID, "user_id": user.UserID})
	if err != nil {
		t.Fatalf("GenerateAccessToken old: %v", err)
	}
	refresh, err := auth.GenerateRefreshToken(nil)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if !auth.CreateSession(ctx, services.Session{UserID: int64String(user.ID), Token: oldAccess, RefreshToken: refresh}, 2*time.Hour) {
		t.Fatal("CreateSession failed")
	}

	body, _ := json.Marshal(gin.H{"refresh_token": refresh})
	recorder := httptest.NewRecorder()
	requestCtx, _ := gin.CreateTestContext(recorder)
	requestCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	requestCtx.Request.Header.Set("Content-Type", "application/json")
	handler := NativeHandlers{
		DB:       db,
		Redis:    store,
		Settings: settings,
		Auth:     auth,
		AuditLog: audit.service,
		Config:   config.Config{Auth: authConfig},
	}
	handler.authRefresh(requestCtx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("authRefresh status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data = %#v", envelope["data"])
	}
	newAccess, _ := data["access_token"].(string)
	if newAccess == "" || newAccess == oldAccess {
		t.Fatalf("access token was not rotated")
	}
	if got, _ := data["refresh_token"].(string); got != refresh {
		t.Fatalf("refresh_token = %q, want preserved token", got)
	}
	if got := int(data["expires_in"].(float64)); got != 120 {
		t.Fatalf("expires_in = %d, want 120", got)
	}
	if _, ok := store.GetRaw(ctx, "session:token:"+oldAccess); ok {
		t.Fatal("old access session key should be deleted")
	}
	var newSession services.Session
	if !store.GetJSON(ctx, "session:token:"+newAccess, &newSession) {
		t.Fatal("new access session key should exist")
	}
	if newSession.RefreshToken != refresh {
		t.Fatalf("new session refresh token = %q, want preserved token", newSession.RefreshToken)
	}
	if ok := auth.FindActiveSession(ctx, newAccess, user.ID); !ok {
		t.Fatal("new access token should authenticate through Redis session")
	}
	if ok := auth.FindActiveSession(ctx, oldAccess, user.ID); ok {
		t.Fatal("old access token should no longer authenticate")
	}
	if cookie := findSetCookie(recorder.Result().Cookies(), authHTTPRefreshCookie); cookie == nil || cookie.Value != refresh || cookie.MaxAge != 7200 {
		t.Fatalf("refresh cookie = %+v, want preserved value and dynamic max age", cookie)
	}
	event := readRefreshAuditEvent(t, audit)
	if event.Category != "token" || event.Action != "refresh" || event.Outcome != "success" {
		t.Fatalf("audit event = %+v", event)
	}
	assertRefreshAuditHasNoTokenMaterial(t, event, refresh, newAccess, oldAccess)
}

func TestAuthRefreshJWTLegacyModeRotatesRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	db := newAuthRefreshTestDB(t)
	redisServer := miniredis.RunT(t)
	store := services.NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	settings := services.NewSettingsService(nil, nil)
	_ = settings.Set(ctx, "refresh_token_mode", services.RefreshTokenModeJWTLegacy)
	_ = settings.Set(ctx, "access_token_ttl_seconds", 120)
	_ = settings.Set(ctx, "refresh_token_active_ttl_seconds", 7200)
	authConfig := config.AuthConfig{JWTSecret: "test-secret", JWTExpiresIn: "3600", RefreshTokenExpiresIn: "7776000"}
	auth := services.NewAuthService(db, store, authConfig, settings)
	audit := newBufferedAuditLogForRefreshTest()
	user := domain.User{UserID: "legacy-refresh-user", Nickname: "Legacy Refresh User", IsActive: true, CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	oldAccess, err := auth.GenerateAccessToken(map[string]any{"userId": user.ID, "user_id": user.UserID})
	if err != nil {
		t.Fatalf("GenerateAccessToken old: %v", err)
	}
	refresh, err := auth.GenerateRefreshToken(map[string]any{"userId": user.ID, "user_id": user.UserID})
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if _, ok := auth.VerifyTokenClaims(refresh); !ok {
		t.Fatal("legacy refresh token should be a JWT")
	}
	if !auth.CreateSession(ctx, services.Session{UserID: int64String(user.ID), Token: oldAccess, RefreshToken: refresh}, 2*time.Hour) {
		t.Fatal("CreateSession failed")
	}

	body, _ := json.Marshal(gin.H{"refresh_token": refresh})
	recorder := httptest.NewRecorder()
	requestCtx, _ := gin.CreateTestContext(recorder)
	requestCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	requestCtx.Request.Header.Set("Content-Type", "application/json")
	handler := NativeHandlers{
		DB:       db,
		Redis:    store,
		Settings: settings,
		Auth:     auth,
		AuditLog: audit.service,
		Config:   config.Config{Auth: authConfig},
	}
	handler.authRefresh(requestCtx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("authRefresh status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data = %#v", envelope["data"])
	}
	newAccess, _ := data["access_token"].(string)
	newRefresh, _ := data["refresh_token"].(string)
	if newAccess == "" || newAccess == oldAccess {
		t.Fatal("access token was not rotated")
	}
	if newRefresh == "" || newRefresh == refresh {
		t.Fatal("jwt legacy refresh token should be rotated")
	}
	claims, ok := auth.VerifyTokenClaims(newRefresh)
	if !ok || claims["token_type"] != "refresh" {
		t.Fatalf("new refresh claims = %#v ok=%v", claims, ok)
	}
	if _, ok := store.GetRaw(ctx, "session:refresh:"+refresh); ok {
		t.Fatal("old refresh session key should be deleted")
	}
	var newSession services.Session
	if !store.GetJSON(ctx, "session:refresh:"+newRefresh, &newSession) {
		t.Fatal("new refresh session key should exist")
	}
	if newSession.Token != newAccess {
		t.Fatalf("new session token = %q, want new access", newSession.Token)
	}
	if ok := auth.FindActiveSession(ctx, oldAccess, user.ID); ok {
		t.Fatal("old access token should no longer authenticate")
	}
	if ok := auth.FindActiveSession(ctx, newAccess, user.ID); !ok {
		t.Fatal("new access token should authenticate through Redis session")
	}
	event := readRefreshAuditEvent(t, audit)
	if event.Category != "token" || event.Action != "refresh" || event.Outcome != "success" {
		t.Fatalf("audit event = %+v", event)
	}
	if event.Metadata["refresh_token_mode"] != services.RefreshTokenModeJWTLegacy {
		t.Fatalf("audit mode = %#v", event.Metadata["refresh_token_mode"])
	}
	assertRefreshAuditHasNoTokenMaterial(t, event, refresh, newRefresh, newAccess, oldAccess)
}

func TestAuthRefreshFailureAuditDoesNotLogTokenMaterial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audit := newBufferedAuditLogForRefreshTest()
	handler := NativeHandlers{
		AuditLog: audit.service,
		Config:   config.Config{Auth: config.AuthConfig{JWTSecret: "test-secret"}},
	}
	body, _ := json.Marshal(gin.H{"refresh_token": "sensitive-refresh-token"})
	recorder := httptest.NewRecorder()
	requestCtx, _ := gin.CreateTestContext(recorder)
	requestCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	requestCtx.Request.Header.Set("Content-Type", "application/json")
	handler.authRefresh(requestCtx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("authRefresh failure status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	event := readRefreshAuditEvent(t, audit)
	if event.Outcome != "failure" || event.ReasonCode != "auth_unavailable" {
		t.Fatalf("audit event = %+v", event)
	}
	assertRefreshAuditHasNoTokenMaterial(t, event, "sensitive-refresh-token")
}

func newAuthRefreshTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

type refreshAuditCapture struct {
	service *services.AuditLogService
	events  <-chan services.SecurityAuditLogEvent
}

func newBufferedAuditLogForRefreshTest() refreshAuditCapture {
	service := &services.AuditLogService{}
	events := make(chan services.SecurityAuditLogEvent, 4)
	field := reflect.ValueOf(service).Elem().FieldByName("securityCh")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(events))
	return refreshAuditCapture{service: service, events: events}
}

func readRefreshAuditEvent(t *testing.T, audit refreshAuditCapture) services.SecurityAuditLogEvent {
	t.Helper()
	select {
	case event := <-audit.events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for refresh audit event")
	}
	return services.SecurityAuditLogEvent{}
}

func assertRefreshAuditHasNoTokenMaterial(t *testing.T, event services.SecurityAuditLogEvent, secrets ...string) {
	t.Helper()
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	if event.Metadata["token_material_logged"] != false {
		t.Fatalf("audit metadata token_material_logged = %#v", event.Metadata["token_material_logged"])
	}
	text := string(encoded)
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			t.Fatalf("audit event leaked token material %q: %s", secret, text)
		}
	}
}

func int64String(value int64) string {
	return strconv.FormatInt(value, 10)
}
