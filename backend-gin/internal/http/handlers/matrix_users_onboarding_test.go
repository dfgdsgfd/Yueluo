package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

func TestAvailableOnboardingFieldRuleKeepsInterestRequirementCompletable(t *testing.T) {
	required := onboardingFieldRule{Enabled: true, Required: true, Min: 3}

	empty := availableOnboardingFieldRule(required, nil)
	if empty.Enabled || empty.Required {
		t.Fatalf("empty interest options rule = %+v, want disabled and optional", empty)
	}

	clamped := availableOnboardingFieldRule(required, []string{"Bondage", "Handcuffs"})
	if !clamped.Enabled || !clamped.Required || clamped.Min != 2 {
		t.Fatalf("clamped interest rule = %+v, want enabled required min=2", clamped)
	}
}

func TestNormalizeLocalizedInterestOptionsSetting(t *testing.T) {
	got := normalizeLocalizedInterestOptionsSetting(map[string]any{
		"en":    `["Bondage", "Handcuffs", "Leg irons"]`,
		"zh-CN": "Bondage\nHandcuffs\nLeg irons",
	})

	for _, locale := range []string{"en", "zh-CN", "zh-TW"} {
		options, ok := got[locale].([]string)
		if !ok {
			t.Fatalf("%s options type = %T, want []string", locale, got[locale])
		}
		if len(options) != 3 || options[0] != "Bondage" || options[2] != "Leg irons" {
			t.Fatalf("%s options = %#v", locale, options)
		}
	}
}

func TestAdminUpdateSettingsSavesOnboardingInterestOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/system-settings", strings.NewReader(`{
		"onboarding_interest_options": {
			"en": ["Bondage", "Handcuffs", "Leg irons"],
			"zh-CN": "Bondage\nHandcuffs\nLeg irons"
		},
		"onboarding_interests_enabled": true,
		"onboarding_min_interests": 5
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.adminUpdateSettings(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !settings.Bool("onboarding_interests_enabled") {
		t.Fatal("onboarding_interests_enabled was not saved")
	}
	if got := settings.Int("onboarding_min_interests", 0); got != 5 {
		t.Fatalf("onboarding_min_interests = %d, want 5", got)
	}
	options := settings.LocalizedStringArray("onboarding_interest_options", "en")
	if len(options) != 3 || options[0] != "Bondage" || options[2] != "Leg irons" {
		t.Fatalf("saved english options = %#v", options)
	}
}
