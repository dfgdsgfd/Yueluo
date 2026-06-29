package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

func TestShouldHideVideoCenterWithCountryOnlyHidesCNAfterCutoff(t *testing.T) {
	settings := services.NewSettingsService(nil, nil)
	createdAt := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)

	if !shouldHideVideoCenterWithCountry(settings, createdAt, "CN") {
		t.Fatal("expected CN user registered at cutoff to be hidden")
	}
	if shouldHideVideoCenterWithCountry(settings, createdAt, "US") {
		t.Fatal("expected non-CN user registered at cutoff to remain visible")
	}
	if shouldHideVideoCenterWithCountry(settings, createdAt, "LAN") {
		t.Fatal("expected local/non-CN network to remain visible")
	}
}

func TestShouldHideVideoCenterWithCountryKeepsUnknownConservative(t *testing.T) {
	settings := services.NewSettingsService(nil, nil)
	createdAt := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)

	if !shouldHideVideoCenterWithCountry(settings, createdAt, "") {
		t.Fatal("expected unknown country to preserve registration-time hiding")
	}
}

func TestShouldHideVideoCenterWithCountryAllowsBeforeCutoffEvenCN(t *testing.T) {
	settings := services.NewSettingsService(nil, nil)
	createdAt := time.Date(2026, 6, 11, 23, 59, 59, 0, time.UTC)

	if shouldHideVideoCenterWithCountry(settings, createdAt, "CN") {
		t.Fatal("expected users before cutoff to remain visible")
	}
}

func TestAuthConfigExposesVideoGuestRestriction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_video_restricted", true) {
		t.Fatal("failed to restrict video guest access")
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/auth-config", nil)

	NativeHandlers{Settings: settings}.authConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data struct {
			VideoCenterGuestRestricted bool `json:"videoCenterGuestRestricted"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Data.VideoCenterGuestRestricted {
		t.Fatal("expected videoCenterGuestRestricted to be true")
	}
}

func TestAuthConfigExposesLocalizedSiteProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), services.SiteTitleSetting, map[string]any{"en": "Yuem", "ja": "月見"}) ||
		!settings.Set(context.Background(), services.SiteDescriptionSetting, map[string]any{"en": "Creator feed", "ja": "創作フィード"}) ||
		!settings.Set(context.Background(), services.SiteAvatarURLSetting, "/api/file/attachments/site.png") {
		t.Fatal("failed to seed site profile settings")
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/auth-config?locale=ja", nil)

	NativeHandlers{Settings: settings}.authConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Data struct {
			SiteProfile services.SiteProfile `json:"siteProfile"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.SiteProfile.Title != "月見" {
		t.Fatalf("site title = %q, want 月見", body.Data.SiteProfile.Title)
	}
	if body.Data.SiteProfile.Description != "創作フィード" {
		t.Fatalf("site description = %q, want 創作フィード", body.Data.SiteProfile.Description)
	}
	if body.Data.SiteProfile.AvatarURL != "/api/file/attachments/site.png" {
		t.Fatalf("site avatar = %q", body.Data.SiteProfile.AvatarURL)
	}
}
