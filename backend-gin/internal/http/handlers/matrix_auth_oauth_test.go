package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestOAuthCallbackURLUsesRedirectBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{Config: config.Config{OAuth2: config.OAuth2Config{
		RedirectBaseURL: "http://localhost:3000",
		CallbackPath:    "/api/auth/oauth2/callback",
	}}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "http://backend.example.test/api/auth/oauth2/login", nil)

	if got := handler.oauthCallbackURL(ctx); got != "http://localhost:3000/api/auth/oauth2/callback" {
		t.Fatalf("oauthCallbackURL() = %q", got)
	}
}

func TestOAuthCallbackURLPrefersFullRedirectURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{Config: config.Config{OAuth2: config.OAuth2Config{
		RedirectURI:     "https://xse.example.com/api/auth/oauth2/callback",
		RedirectBaseURL: "http://localhost:3000",
		CallbackPath:    "/api/auth/oauth2/callback",
	}}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "http://backend.example.test/api/auth/oauth2/login", nil)

	if got := handler.oauthCallbackURL(ctx); got != "https://xse.example.com/api/auth/oauth2/callback" {
		t.Fatalf("oauthCallbackURL() = %q", got)
	}
}

func TestOAuthCallbackURLFallsBackToRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{Config: config.Config{OAuth2: config.OAuth2Config{
		CallbackPath: "api/auth/oauth2/callback",
	}}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "http://backend.example.test/api/auth/oauth2/login", nil)
	ctx.Request.Host = "backend.example.test"
	ctx.Request.Header.Set("X-Forwarded-Proto", "https")
	ctx.Request.Header.Set("X-Forwarded-Host", "proxy.example.test")

	if got := handler.oauthCallbackURL(ctx); got != "https://proxy.example.test/api/auth/oauth2/callback" {
		t.Fatalf("oauthCallbackURL() = %q", got)
	}
}

func TestOAuthSuccessRedirectLocationDoesNotLeakTokens(t *testing.T) {
	location := oauthSuccessRedirectLocation(true)
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsed.Path != "/explore" {
		t.Fatalf("redirect path = %q, want /explore", parsed.Path)
	}
	values := parsed.Query()
	if values.Get("oauth2_login") != "success" || values.Get("is_new_user") != "true" {
		t.Fatalf("redirect query = %q", parsed.RawQuery)
	}
	for _, key := range []string{"access_token", "refresh_token", "token"} {
		if values.Has(key) {
			t.Fatalf("redirect leaked %s in query: %q", key, parsed.RawQuery)
		}
	}
}

func TestAuthAccessTokenFromRequestUsesAuthorizationBeforeCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	ctx.Request.Header.Set("Authorization", "Bearer header-token")
	ctx.Request.AddCookie(&http.Cookie{Name: authHTTPAccessCookie, Value: "cookie-token"})

	if got := handler.authAccessTokenFromRequest(ctx); got != "header-token" {
		t.Fatalf("authAccessTokenFromRequest() = %q, want header-token", got)
	}
}

func TestAuthAccessTokenFromRequestIgnoresEmptyBearerBeforeCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	ctx.Request.Header.Set("Authorization", "Bearer ")
	ctx.Request.AddCookie(&http.Cookie{Name: authHTTPAccessCookie, Value: "cookie-token"})

	if got := handler.authAccessTokenFromRequest(ctx); got != "cookie-token" {
		t.Fatalf("authAccessTokenFromRequest() = %q, want cookie-token", got)
	}
}

func TestAuthTokenFromRequestFallsBackToCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: authHTTPAccessCookie, Value: "access-cookie"})
	ctx.Request.AddCookie(&http.Cookie{Name: authHTTPRefreshCookie, Value: "refresh-cookie"})

	if got := handler.authAccessTokenFromRequest(ctx); got != "access-cookie" {
		t.Fatalf("authAccessTokenFromRequest() = %q, want access-cookie", got)
	}
	if got := handler.authRefreshTokenFromRequest(ctx); got != "refresh-cookie" {
		t.Fatalf("authRefreshTokenFromRequest() = %q, want refresh-cookie", got)
	}
}

func TestOAuthAppRedirectUsesConfiguredHTTPSCallbackWithoutTokens(t *testing.T) {
	handler := NativeHandlers{Config: config.Config{OAuth2: config.OAuth2Config{
		AppCallbackURL: "https://xse.yuelk.com/app/oauth/callback",
	}}}
	location := handler.oauthAppSuccessRedirectLocation(strings.Repeat("c", 43), strings.Repeat("s", 32), true)
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse app redirect: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "xse.yuelk.com" || parsed.Path != "/app/oauth/callback" {
		t.Fatalf("unexpected app redirect: %s", location)
	}
	if parsed.Query().Get("code") == "" || parsed.Query().Get("app_state") == "" {
		t.Fatalf("missing app handoff values: %s", location)
	}
	for _, key := range []string{"access_token", "refresh_token", "token"} {
		if parsed.Query().Has(key) {
			t.Fatalf("app redirect leaked %s: %s", key, location)
		}
	}
}

func TestOAuthAppDefaultRedirectUsesCustomSchemeWithoutTokens(t *testing.T) {
	handler := NativeHandlers{}
	location := handler.oauthAppSuccessRedirectLocation(strings.Repeat("c", 43), strings.Repeat("s", 32), false)
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse default app redirect: %v", err)
	}
	if parsed.Scheme != "xsewebfast" || parsed.Host != "oauth" || parsed.Path != "/callback" {
		t.Fatalf("unexpected default app redirect: %s", location)
	}
	if parsed.Query().Get("code") == "" || parsed.Query().Get("app_state") == "" {
		t.Fatalf("missing app handoff values: %s", location)
	}
	for _, key := range []string{"access_token", "refresh_token", "token"} {
		if parsed.Query().Has(key) {
			t.Fatalf("default app redirect leaked %s: %s", key, location)
		}
	}
}

func TestOAuthMobileBridgeStoresReturnURLAndRedirectsWithTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{
		Cache: services.NewCache(),
		Config: config.Config{
			Frontend: config.FrontendConfig{BaseURL: "https://xse.yuelk.com"},
			OAuth2: config.OAuth2Config{
				AppCallbackURL: "xsewebfast://auth-return",
				Enabled:        true,
				LoginURL:       "https://user.yuelk.com",
				ClientID:       "client-id",
			},
		},
	}
	encodedCallback := url.QueryEscape("xsewebfast://auth-return?url=https%3A%2F%2Fxse.yuelk.com%2Fmessages")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/login?app_callback="+encodedCallback+"&app_return_url=https%3A%2F%2Fxse.yuelk.com%2Fmessages", nil)

	handler.oauthLogin(ctx)
	if recorder.Code != http.StatusFound {
		t.Fatalf("oauthLogin status = %d, want 302; body=%s", recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("login redirect missing state: %s", location)
	}
	stored, ok := handler.Cache.Get("oauth2:state:" + state)
	if !ok {
		t.Fatalf("oauth state %q was not stored", state)
	}
	entry, ok := stored.(oauthStateEntry)
	if !ok {
		t.Fatalf("stored OAuth state has type %T", stored)
	}
	if entry.AppCallbackURL != "xsewebfast://auth-return?url=https%3A%2F%2Fxse.yuelk.com%2Fmessages" {
		t.Fatalf("AppCallbackURL = %q", entry.AppCallbackURL)
	}
	if entry.AppReturnURL != "https://xse.yuelk.com/messages" {
		t.Fatalf("AppReturnURL = %q", entry.AppReturnURL)
	}

	user := domain.User{ID: 42, UserID: "mobile-user", Nickname: "Mobile User", IsActive: true, CreatedAt: time.Now().UTC()}
	callbackRecorder := httptest.NewRecorder()
	callbackCtx, _ := gin.CreateTestContext(callbackRecorder)
	callbackCtx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/callback", nil)
	handler.completeOAuthMobileCallback(callbackCtx, entry, user, true)
	if callbackRecorder.Code != http.StatusFound {
		t.Fatalf("completeOAuthMobileCallback status = %d, want 302; body=%s", callbackRecorder.Code, callbackRecorder.Body.String())
	}
	callbackLocation := callbackRecorder.Header().Get("Location")
	callbackURL, err := url.Parse(callbackLocation)
	if err != nil {
		t.Fatalf("parse mobile callback redirect: %v", err)
	}
	if callbackURL.Scheme != "xsewebfast" || callbackURL.Host != "auth-return" {
		t.Fatalf("mobile callback redirect = %s", callbackLocation)
	}
	if callbackURL.Query().Get("url") != "https://xse.yuelk.com/messages" {
		t.Fatalf("mobile callback return url = %q", callbackURL.Query().Get("url"))
	}
	ticket := callbackURL.Query().Get("ticket")
	if ticket == "" {
		t.Fatalf("mobile callback missing ticket: %s", callbackLocation)
	}
	if _, ok := handler.Cache.Get("oauth2:mobile-session:" + sha256Hex(ticket)); !ok {
		t.Fatalf("mobile session ticket was not stored")
	}
	for _, key := range []string{"access_token", "refresh_token", "token", "code", "app_state"} {
		if callbackURL.Query().Has(key) {
			t.Fatalf("mobile callback leaked %s: %s", key, callbackLocation)
		}
	}
}

func TestOAuthMobileCallbackAllowlistSupportsMultipleApps(t *testing.T) {
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), services.OAuth2AppCallbackURLsSetting, strings.Join([]string{
		"xsewebfast://auth-return",
		"yuempro://auth-return",
		"xsewebbeta://oauth/callback",
	}, "\n")) {
		t.Fatal("failed to set OAuth app callback allowlist")
	}
	handler := NativeHandlers{
		Settings: settings,
		Config: config.Config{
			OAuth2: config.OAuth2Config{AppCallbackURL: "legacy://auth-return"},
		},
	}

	got, ok := handler.safeOAuthMobileCallbackURL("yuempro://auth-return?ticket=leak&url=https%3A%2F%2Fxse.yuelk.com%2Fexplore&debug=1")
	if !ok {
		t.Fatal("expected yuempro callback to be allowed")
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse normalized callback: %v", err)
	}
	if parsed.Scheme != "yuempro" || parsed.Host != "auth-return" {
		t.Fatalf("normalized callback = %q", got)
	}
	if parsed.Query().Get("debug") != "1" || parsed.Query().Has("ticket") {
		t.Fatalf("unexpected normalized callback query = %q", parsed.RawQuery)
	}
	if _, ok := handler.safeOAuthMobileCallbackURL("evil://auth-return"); ok {
		t.Fatal("unexpected evil scheme to be allowed")
	}
	if _, ok := handler.safeOAuthMobileCallbackURL("yuempro://evil-host"); ok {
		t.Fatal("unexpected mismatched host to be allowed")
	}
	if _, ok := handler.safeOAuthMobileCallbackURL("xsewebbeta://oauth/other"); ok {
		t.Fatal("unexpected mismatched path to be allowed")
	}
}

func TestNormalizeOAuth2AppCallbackURLSetting(t *testing.T) {
	normalized, ok := normalizeOAuth2AppCallbackURLSetting([]any{
		"xsewebfast://auth-return?ticket=leak",
		"yuempro://auth-return",
		"xsewebfast://auth-return?debug=1",
	})
	if !ok {
		t.Fatal("normalizeOAuth2AppCallbackURLSetting returned false")
	}
	want := "xsewebfast://auth-return\nyuempro://auth-return"
	if normalized != want {
		t.Fatalf("normalized callbacks = %q, want %q", normalized, want)
	}
	if _, ok := normalizeOAuth2AppCallbackURLSetting("https://user:pass@example.com/callback"); ok {
		t.Fatal("callback with credentials should be rejected")
	}
	if _, ok := normalizeOAuth2AppCallbackURLSetting("not-a-url"); ok {
		t.Fatal("callback without scheme/host should be rejected")
	}
}

func TestAuthOAuthMobileSessionConsumesTicketOnceAndSetsCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:oauth-mobile-session?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := domain.User{UserID: "mobile-session-user", Nickname: "Mobile Session", IsActive: true, CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	authConfig := config.AuthConfig{JWTSecret: "test-secret", JWTExpiresIn: "7d", RefreshTokenExpiresIn: "30d"}
	handler := NativeHandlers{
		DB:    db,
		Cache: services.NewCache(),
		Config: config.Config{
			Auth:   authConfig,
			OAuth2: config.OAuth2Config{Enabled: true},
		},
		Auth: services.NewAuthService(db, nil, authConfig),
	}
	ticket := "ticket-" + strings.Repeat("a", 32)
	if !handler.storeOAuthMobileSessionTicket(httptest.NewRequest(http.MethodGet, "/", nil).Context(), ticket, oauthMobileSessionTicket{
		UserID:    user.ID,
		IsNewUser: true,
		ExpiresAt: time.Now().UTC().Add(oauthMobileSessionTTL),
	}) {
		t.Fatal("storeOAuthMobileSessionTicket returned false")
	}
	body, _ := json.Marshal(gin.H{"ticket": ticket})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oauth2/mobile-session", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.authOAuthMobileSession(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("mobile-session status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, name := range []string{authReadableAccessCookie, authHTTPAccessCookie, authHTTPRefreshCookie} {
		if cookie := findSetCookie(recorder.Result().Cookies(), name); cookie == nil || cookie.Value == "" {
			t.Fatalf("missing Set-Cookie %s; cookies=%v", name, recorder.Result().Cookies())
		}
	}
	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode mobile-session response: %v", err)
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("mobile-session response data = %#v", envelope["data"])
	}
	if data["access_token"] == "" || data["refresh_token"] == "" || data["is_new_user"] != true {
		t.Fatalf("unexpected mobile-session payload: %+v", data)
	}
	userData, ok := data["user"].(map[string]any)
	if !ok || userData["user_id"] != user.UserID {
		t.Fatalf("response user = %+v", data["user"])
	}

	replayRecorder := httptest.NewRecorder()
	replayCtx, _ := gin.CreateTestContext(replayRecorder)
	replayCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oauth2/mobile-session", bytes.NewReader(body))
	replayCtx.Request.Header.Set("Content-Type", "application/json")
	handler.authOAuthMobileSession(replayCtx)
	if replayRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("replayed mobile-session status = %d, want 401; body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}

	expiredTicket := "expired-" + strings.Repeat("b", 32)
	if !handler.storeOAuthMobileSessionTicket(httptest.NewRequest(http.MethodGet, "/", nil).Context(), expiredTicket, oauthMobileSessionTicket{
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(-time.Second),
	}) {
		t.Fatal("store expired OAuth mobile session ticket returned false")
	}
	expiredBody, _ := json.Marshal(gin.H{"ticket": expiredTicket})
	expiredRecorder := httptest.NewRecorder()
	expiredCtx, _ := gin.CreateTestContext(expiredRecorder)
	expiredCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oauth2/mobile-session", bytes.NewReader(expiredBody))
	expiredCtx.Request.Header.Set("Content-Type", "application/json")
	handler.authOAuthMobileSession(expiredCtx)
	if expiredRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expired mobile-session status = %d, want 401; body=%s", expiredRecorder.Code, expiredRecorder.Body.String())
	}
}

func TestOAuthMobileReturnURLRejectsExternalOrigins(t *testing.T) {
	handler := NativeHandlers{Config: config.Config{Frontend: config.FrontendConfig{BaseURL: "https://xse.yuelk.com"}}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/login?app_return_url=https%3A%2F%2Fevil.example%2Fsteal", nil)
	if got := handler.safeOAuthMobileReturnURL(ctx, ctx.Query("app_return_url")); got != "https://xse.yuelk.com/explore" {
		t.Fatalf("safeOAuthMobileReturnURL external = %q", got)
	}
	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/login?app_return_url=%2Fmessages", nil)
	if got := handler.safeOAuthMobileReturnURL(ctx, ctx.Query("app_return_url")); got != "https://xse.yuelk.com/messages" {
		t.Fatalf("safeOAuthMobileReturnURL relative = %q", got)
	}
}

func TestOAuthAppHandoffTTLAllowsSlowBrowserReturn(t *testing.T) {
	if oauthAppHandoffTTL < 10*time.Minute {
		t.Fatalf("oauthAppHandoffTTL = %s, want at least 10m", oauthAppHandoffTTL)
	}
}

func findSetCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestConsumeOAuthAppHandoffIsPKCEBoundAndOneTime(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:oauth-app-handoff?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.OAuthAppHandoff{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := domain.User{UserID: "oauth-app-user", Nickname: "OAuth App", IsActive: true, CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	code := strings.Repeat("c", 43)
	appState := strings.Repeat("s", 32)
	verifier := strings.Repeat("v", 43)
	handoff := domain.OAuthAppHandoff{
		CodeHash:      sha256Hex(code),
		AppStateHash:  sha256Hex(appState),
		CodeChallenge: base64URLSHA256(verifier),
		UserID:        user.ID,
		ExpiresAt:     time.Now().UTC().Add(time.Minute),
		CreatedAt:     time.Now().UTC(),
	}
	if err := db.Create(&handoff).Error; err != nil {
		t.Fatalf("create handoff: %v", err)
	}

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oauth2/app-token", nil)
	handler := NativeHandlers{DB: db}
	consumed, gotUser, err := handler.consumeOAuthAppHandoff(context, code, appState, verifier)
	if err != nil {
		t.Fatalf("consume handoff: %v", err)
	}
	if consumed.ConsumedAt == nil || gotUser.ID != user.ID {
		t.Fatalf("unexpected consumed handoff=%+v user=%+v", consumed, gotUser)
	}
	if _, _, err := handler.consumeOAuthAppHandoff(context, code, appState, verifier); err == nil {
		t.Fatal("expected replayed handoff to fail")
	}
}

func TestConsumeOAuthAppHandoffRejectsWrongVerifierAndExpiry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:oauth-app-invalid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.OAuthAppHandoff{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := domain.User{UserID: "oauth-invalid-user", Nickname: "OAuth App", IsActive: true, CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	verifier := strings.Repeat("v", 43)
	rows := []domain.OAuthAppHandoff{
		{CodeHash: sha256Hex(strings.Repeat("a", 43)), AppStateHash: sha256Hex(strings.Repeat("s", 32)), CodeChallenge: base64URLSHA256(verifier), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(time.Minute), CreatedAt: time.Now().UTC()},
		{CodeHash: sha256Hex(strings.Repeat("b", 43)), AppStateHash: sha256Hex(strings.Repeat("t", 32)), CodeChallenge: base64URLSHA256(verifier), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(-time.Second), CreatedAt: time.Now().UTC()},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create handoffs: %v", err)
	}
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oauth2/app-token", nil)
	handler := NativeHandlers{DB: db}
	if _, _, err := handler.consumeOAuthAppHandoff(context, strings.Repeat("a", 43), strings.Repeat("s", 32), strings.Repeat("x", 43)); err == nil {
		t.Fatal("expected wrong verifier to fail")
	}
	if _, _, err := handler.consumeOAuthAppHandoff(context, strings.Repeat("b", 43), strings.Repeat("t", 32), verifier); err == nil {
		t.Fatal("expected expired handoff to fail")
	}
}
