package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
	"yuem-go/backend-gin/internal/services"
)

func TestAuthAdminLoginUsesAdminJWTExpiresIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuthAdminTestDB(t)
	passwordHash, err := security.HashPassword("secret-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := db.Create(&domain.Admin{Username: "admin", Password: passwordHash}).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	authConfig := config.AuthConfig{
		JWTSecret:             "test-secret",
		JWTExpiresIn:          "3600",
		AdminJWTExpiresIn:     "12h",
		RefreshTokenExpiresIn: "7776000",
	}
	auth := services.NewAuthService(db, nil, authConfig)
	handler := NativeHandlers{
		DB:     db,
		Auth:   auth,
		Config: config.Config{Auth: authConfig},
	}
	body, _ := json.Marshal(gin.H{"username": "admin", "password": "secret-password"})
	recorder := httptest.NewRecorder()
	requestCtx, _ := gin.CreateTestContext(recorder)
	requestCtx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/admin/login", bytes.NewReader(body))
	requestCtx.Request.Header.Set("Content-Type", "application/json")

	handler.authAdminLogin(requestCtx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("authAdminLogin status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data = %#v", envelope["data"])
	}
	tokens, ok := data["tokens"].(map[string]any)
	if !ok {
		t.Fatalf("tokens = %#v", data["tokens"])
	}
	access, _ := tokens["access_token"].(string)
	if access == "" {
		t.Fatal("admin login did not return an access token")
	}
	if got := int(tokens["expires_in"].(float64)); got != int((12 * time.Hour).Seconds()) {
		t.Fatalf("expires_in = %d, want 43200", got)
	}

	claims, ok := auth.VerifyTokenClaims(access)
	if !ok {
		t.Fatal("admin access token should verify")
	}
	issuedAt, ok := int64FromAny(claims["iat"])
	if !ok {
		t.Fatalf("iat claim = %#v", claims["iat"])
	}
	expiresAt, ok := int64FromAny(claims["exp"])
	if !ok {
		t.Fatalf("exp claim = %#v", claims["exp"])
	}
	if ttl := time.Duration(expiresAt-issuedAt) * time.Second; ttl != 12*time.Hour {
		t.Fatalf("admin token ttl = %s, want 12h", ttl)
	}
	if tokenType, _ := claims["type"].(string); tokenType != "admin" {
		t.Fatalf("token type = %#v, want admin", claims["type"])
	}
}

func newAuthAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Admin{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
