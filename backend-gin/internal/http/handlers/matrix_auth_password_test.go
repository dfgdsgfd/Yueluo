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

func TestAuthLoginAcceptsUserIDOrEmailIdentifier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPasswordAuthTestDB(t)
	passwordHash, err := security.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	email := "case.user@example.com"
	user := domain.User{
		UserID:    "case_user",
		Nickname:  "Case User",
		Email:     &email,
		Password:  &passwordHash,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	authConfig := config.AuthConfig{
		JWTSecret:             "test-secret",
		JWTExpiresIn:          "1h",
		RefreshTokenExpiresIn: "24h",
	}
	handler := NativeHandlers{
		DB:     db,
		Auth:   services.NewAuthService(db, nil, authConfig),
		Config: config.Config{Auth: authConfig},
	}

	for _, tc := range []struct {
		name string
		body gin.H
	}{
		{name: "user_id", body: gin.H{"user_id": "case_user", "password": "secret123"}},
		{name: "identifier_email", body: gin.H{"identifier": "CASE.USER@example.com", "password": "secret123"}},
		{name: "email", body: gin.H{"email": "case.user@example.com", "password": "secret123"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			handler.authLogin(ctx)

			if recorder.Code != http.StatusOK {
				t.Fatalf("authLogin status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			var envelope map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			data, ok := envelope["data"].(map[string]any)
			if !ok {
				t.Fatalf("response data = %#v", envelope["data"])
			}
			if tokens, ok := data["tokens"].(map[string]any); !ok || tokens["access_token"] == "" || tokens["refresh_token"] == "" {
				t.Fatalf("tokens missing from response: %#v", data["tokens"])
			}
			userData, ok := data["user"].(map[string]any)
			if !ok || userData["user_id"] != user.UserID {
				t.Fatalf("response user = %#v", data["user"])
			}
		})
	}
}

func TestAuthRegisterStoresEmailWhenEmailDeliveryIsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPasswordAuthTestDB(t)
	authConfig := config.AuthConfig{
		JWTSecret:             "test-secret",
		JWTExpiresIn:          "1h",
		RefreshTokenExpiresIn: "24h",
	}
	cache := services.NewCache()
	cache.Set("captcha:captcha-1", "ABCD", time.Minute)
	handler := NativeHandlers{
		DB:     db,
		Cache:  cache,
		Auth:   services.NewAuthService(db, nil, authConfig),
		Config: config.Config{Auth: authConfig},
	}
	body, _ := json.Marshal(gin.H{
		"user_id":     "email_user",
		"nickname":    "Email User",
		"email":       "new.user@example.com",
		"password":    "secret123",
		"captchaId":   "captcha-1",
		"captchaText": "abcd",
	})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.authRegister(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("authRegister status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var user domain.User
	if err := db.Where("user_id = ?", "email_user").First(&user).Error; err != nil {
		t.Fatalf("find registered user: %v", err)
	}
	if user.Email == nil || *user.Email != "new.user@example.com" {
		t.Fatalf("registered email = %#v, want new.user@example.com", user.Email)
	}
}

func newPasswordAuthTestDB(t *testing.T) *gorm.DB {
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
