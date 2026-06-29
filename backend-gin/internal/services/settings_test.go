package services

import "testing"

func TestEncodeSettingValueStoresStructuredConfigAsJSON(t *testing.T) {
	got := encodeSettingValue(map[string]any{"title_weight": 8, "enabled": true})
	if got != `{"enabled":true,"title_weight":8}` && got != `{"title_weight":8,"enabled":true}` {
		t.Fatalf("encodeSettingValue() = %q, want JSON object", got)
	}
}

func TestSettingsServiceReadsDecodedDefaults(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if !settings.Set(t.Context(), "recommend_config", map[string]any{"title_weight": float64(5)}) {
		t.Fatal("Set(recommend_config) failed")
	}
	if got := settings.String("recommend_config"); got != `{"title_weight":5}` {
		t.Fatalf("String(recommend_config) = %q, want JSON", got)
	}
	if !settingBool("true") || !settingBool("1") || !settingBool(float64(1)) {
		t.Fatalf("settingBool should accept legacy truthy values")
	}
}

func TestDefaultSettingsIncludePaidContentPriceLimits(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if got := settings.Int("paid_content_points_max_price", 0); got != DefaultPaidContentPointsMaxPrice {
		t.Fatalf("paid_content_points_max_price = %d, want %d", got, DefaultPaidContentPointsMaxPrice)
	}
	if got := settings.Int("paid_content_balance_max_price", 0); got != DefaultPaidContentBalanceMaxPrice {
		t.Fatalf("paid_content_balance_max_price = %d, want %d", got, DefaultPaidContentBalanceMaxPrice)
	}
}

func TestDefaultSettingsIncludeNotificationSuppression(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if !settings.Bool("notification_interaction_suppression_enabled") {
		t.Fatal("notification interaction suppression should default to enabled")
	}
	if got := settings.Int("notification_interaction_suppression_window_seconds", 0); got != DefaultNotificationInteractionSuppressionWindowSeconds {
		t.Fatalf("notification suppression window = %d, want %d", got, DefaultNotificationInteractionSuppressionWindowSeconds)
	}
	if got := settings.Int("notification_interaction_suppression_threshold", 0); got != DefaultNotificationInteractionSuppressionThreshold {
		t.Fatalf("notification suppression threshold = %d, want %d", got, DefaultNotificationInteractionSuppressionThreshold)
	}
	if got := settingGroup("notification_interaction_suppression_threshold"); got != "notifications" {
		t.Fatalf("notification setting group = %q, want notifications", got)
	}
}

func TestDefaultSettingsIncludeFileRecycleControls(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if got := settings.Int(FileRecycleRetentionDaysKey, 0); got != DefaultFileRecycleRetentionDays {
		t.Fatalf("file recycle retention days = %d, want %d", got, DefaultFileRecycleRetentionDays)
	}
	if got := settings.Int(FileRecycleCleanupIntervalHoursKey, 0); got != DefaultFileRecycleCleanupIntervalHours {
		t.Fatalf("file recycle cleanup interval hours = %d, want %d", got, DefaultFileRecycleCleanupIntervalHours)
	}
	if got := settingGroup(FileRecycleRetentionDaysKey); got != "file_recycle" {
		t.Fatalf("file recycle setting group = %q, want file_recycle", got)
	}
}

func TestDefaultSettingsIncludeSiteProfile(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	profile := ReadSiteProfileForLocale(settings, "zh-CN")
	if profile.Title != DefaultSiteTitle {
		t.Fatalf("site title = %q, want %q", profile.Title, DefaultSiteTitle)
	}
	if profile.Description != DefaultSiteDescription {
		t.Fatalf("site description = %q, want %q", profile.Description, DefaultSiteDescription)
	}
	if profile.AvatarURL != "" {
		t.Fatalf("site avatar = %q, want empty", profile.AvatarURL)
	}
	if !LocalizedSettingKeys[SiteTitleSetting] || !LocalizedSettingKeys[SiteDescriptionSetting] {
		t.Fatal("site title and description should be localized settings")
	}
	if got := settingGroup(SiteAvatarURLSetting); got != "site" {
		t.Fatalf("site setting group = %q, want site", got)
	}

	if !settings.Set(t.Context(), SiteTitleSetting, map[string]any{"en": "Moon", "ja": "月"}) {
		t.Fatal("Set(site_title) failed")
	}
	if got := ReadSiteProfileForLocale(settings, "ja").Title; got != "月" {
		t.Fatalf("localized site title = %q, want 月", got)
	}
}

func TestSettingsServiceAppliesFileRecycleEnvDefaults(t *testing.T) {
	t.Setenv("UPLOAD_RECYCLE_RETENTION", "48h")
	t.Setenv("UPLOAD_RECYCLE_CLEANUP_INTERVAL", "30m")

	settings := NewSettingsService(nil, nil)

	if got := settings.Int(FileRecycleRetentionDaysKey, 0); got != 2 {
		t.Fatalf("file recycle retention days = %d, want 2", got)
	}
	if got := settings.Int(FileRecycleCleanupIntervalHoursKey, 0); got != 1 {
		t.Fatalf("file recycle cleanup interval hours = %d, want 1", got)
	}
}

func TestDefaultSettingsIncludePostPresentationControls(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if got := settings.Int("post_content_max_length", 0); got != 100000 {
		t.Fatalf("post_content_max_length = %d, want 100000", got)
	}
	if got := settings.String("post_resource_section_position"); got != "before_content" {
		t.Fatalf("post_resource_section_position = %q, want before_content", got)
	}
	if got := settingGroup("post_content_max_length"); got != "content" {
		t.Fatalf("post setting group = %q, want content", got)
	}
}

func TestSettingsServiceAppliesGuestAccessEnvOverrides(t *testing.T) {
	t.Setenv("GUEST_ACCESS_NOTE_RESTRICTED", "true")
	t.Setenv("GUEST_ACCESS_VIDEO_RESTRICTED", "false")

	settings := NewSettingsService(nil, nil)

	if !settings.IsNoteGuestRestricted() {
		t.Fatal("expected note guest restriction to be enabled from env")
	}
	if settings.IsVideoGuestRestricted() {
		t.Fatal("expected video guest restriction to be disabled from env")
	}
}

func TestSettingsServiceAppliesLegacyImageEnvDefaults(t *testing.T) {
	t.Setenv("WEBP_ENABLE_CONVERSION", "false")
	t.Setenv("WEBP_QUALITY", "82")
	t.Setenv("AVATAR_WEBP_QUALITY", "71")
	t.Setenv("WEBP_METHOD", "6")
	t.Setenv("WEBP_ALPHA_QUALITY", "91")
	t.Setenv("WEBP_MAX_WIDTH", "1600")
	t.Setenv("WEBP_MAX_HEIGHT", "1200")
	t.Setenv("HIDDEN_WATERMARK_ENABLED", "false")
	t.Setenv("HIDDEN_WATERMARK_PROTECTED_ONLY", "false")
	t.Setenv("HIDDEN_WATERMARK_ENGINE", "remote")
	t.Setenv("HIDDEN_WATERMARK_REMOTE_PASSWORD_WM", "77")
	t.Setenv("HIDDEN_WATERMARK_REMOTE_PASSWORD_IMG", "88")
	t.Setenv("HIDDEN_WATERMARK_REMOTE_PROFILE", "official")
	t.Setenv("IMAGE_PROTECTION_OUTPUT_MODE", "quality_webp")
	t.Setenv("IMAGE_LIBVIPS_ENABLED", "true")

	settings := NewSettingsService(nil, nil)

	if settings.Bool("image_webp_enabled") {
		t.Fatal("expected WEBP_ENABLE_CONVERSION=false")
	}
	if got := settings.Int("image_webp_quality", 0); got != 82 {
		t.Fatalf("image_webp_quality = %d, want 82", got)
	}
	if got := settings.Int("image_avatar_webp_quality", 0); got != 71 {
		t.Fatalf("image_avatar_webp_quality = %d, want 71", got)
	}
	if got := settings.Int("image_webp_method", 0); got != 6 {
		t.Fatalf("image_webp_method = %d, want 6", got)
	}
	if got := settings.Int("image_max_width", 0); got != 1600 {
		t.Fatalf("image_max_width = %d, want 1600", got)
	}
	if settings.Bool("hidden_watermark_enabled") {
		t.Fatal("expected HIDDEN_WATERMARK_ENABLED=false")
	}
	if settings.Bool("hidden_watermark_protected_only") {
		t.Fatal("expected HIDDEN_WATERMARK_PROTECTED_ONLY=false")
	}
	if got := settings.String("hidden_watermark_engine"); got != "remote" {
		t.Fatalf("hidden_watermark_engine = %q, want remote", got)
	}
	if got := settings.Int("hidden_watermark_remote_password_wm", 0); got != 77 {
		t.Fatalf("hidden_watermark_remote_password_wm = %d, want 77", got)
	}
	if got := settings.Int("hidden_watermark_remote_password_img", 0); got != 88 {
		t.Fatalf("hidden_watermark_remote_password_img = %d, want 88", got)
	}
	if got := settings.String("hidden_watermark_remote_profile"); got != "adaptive" {
		t.Fatalf("hidden_watermark_remote_profile = %q, want admin/default adaptive", got)
	}
	if got := settings.String("image_protection_output_mode"); got != "lossless_webp" {
		t.Fatalf("image_protection_output_mode = %q, want admin/default lossless_webp", got)
	}
	if got := settings.Int("image_post_max_count", 0); got != 100 {
		t.Fatalf("image_post_max_count = %d, want 100", got)
	}
	if !settings.Bool("image_archive_enabled") {
		t.Fatal("expected image_archive_enabled=true")
	}
	if got := settings.Int("image_archive_threshold", 0); got != 25 {
		t.Fatalf("image_archive_threshold = %d, want 25", got)
	}
	if !settings.Bool("image_libvips_enabled") {
		t.Fatal("expected IMAGE_LIBVIPS_ENABLED=true")
	}
}

func TestSettingsServiceAppliesOAuth2AppCallbackURLDefaults(t *testing.T) {
	t.Setenv("OAUTH2_APP_CALLBACK_URL", "legacy://auth-return")
	t.Setenv("OAUTH2_APP_CALLBACK_URLS", "xsewebfast://auth-return,yuempro://auth-return")

	settings := NewSettingsService(nil, nil)

	if got := settings.String(OAuth2AppCallbackURLsSetting); got != "xsewebfast://auth-return,yuempro://auth-return" {
		t.Fatalf("oauth2 app callback URLs = %q, want multi URL env default", got)
	}
	if got := settingGroup(OAuth2AppCallbackURLsSetting); got != "auth" {
		t.Fatalf("settingGroup(%q) = %q, want auth", OAuth2AppCallbackURLsSetting, got)
	}
}

func TestSettingGroupForRecommendation(t *testing.T) {
	if got := settingGroup("recommend_config"); got != "recommendation" {
		t.Fatalf("settingGroup(recommend_config) = %q, want recommendation", got)
	}
}

func TestSettingGroupForImageProcessing(t *testing.T) {
	for _, key := range []string{"image_webp_quality", "image_post_max_count", "image_archive_threshold", "hidden_watermark_enabled", "hidden_watermark_protected_only"} {
		if got := settingGroup(key); got != "image_processing" {
			t.Fatalf("settingGroup(%q) = %q, want image_processing", key, got)
		}
	}
}

func TestDefaultSettingsIncludeOnboardingControlsAndPointsIntro(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if !settings.Bool("onboarding_enabled") {
		t.Fatal("onboarding_enabled should default to true")
	}
	if !settings.Bool("onboarding_allow_skip") {
		t.Fatal("onboarding_allow_skip should default to true")
	}
	for _, key := range []string{
		"onboarding_points_intro_title",
		"onboarding_points_intro_summary",
		"onboarding_points_intro_detail",
	} {
		if settings.String(key) == "" {
			t.Fatalf("%s should have a default value", key)
		}
	}
	if got := settingGroup("onboarding_points_intro_detail"); got != "onboarding" {
		t.Fatalf("settingGroup(onboarding_points_intro_detail) = %q, want onboarding", got)
	}
}

func TestHiddenWatermarkUsernameDefaultsToDisabled(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	if settings.Bool("hidden_watermark_include_username") {
		t.Fatal("hidden_watermark_include_username should default to false")
	}
	if got := settings.String("hidden_watermark_engine"); got != "auto" {
		t.Fatalf("hidden_watermark_engine = %q, want auto", got)
	}
	if got := settings.Int("hidden_watermark_remote_password_wm", 0); got != 1 {
		t.Fatalf("hidden_watermark_remote_password_wm = %d, want 1", got)
	}
	if got := settings.String("hidden_watermark_remote_engine"); got != "auto" {
		t.Fatalf("hidden_watermark_remote_engine = %q, want auto", got)
	}
	if got := settings.String("hidden_watermark_remote_profile"); got != "adaptive" {
		t.Fatalf("hidden_watermark_remote_profile = %q, want adaptive", got)
	}
	if got := settings.Int("hidden_watermark_remote_timeout_seconds", 0); got != 50 {
		t.Fatalf("hidden_watermark_remote_timeout_seconds = %d, want 50", got)
	}
	if got := settings.Int("hidden_watermark_remote_operation_timeout_seconds", 0); got != 45 {
		t.Fatalf("hidden_watermark_remote_operation_timeout_seconds = %d, want 45", got)
	}
	if got := settings.Int("image_protection_max_dimension", 0); got != 2048 {
		t.Fatalf("image_protection_max_dimension = %d, want 2048", got)
	}
}
