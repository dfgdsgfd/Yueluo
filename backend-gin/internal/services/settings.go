package services

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/localization"
)

const SettingsKeyPrefix = "settings:"
const OAuth2AppCallbackURLsSetting = "oauth2_app_callback_urls"

const (
	SiteTitleSetting       = "site_title"
	SiteDescriptionSetting = "site_description"
	SiteAvatarURLSetting   = "site_avatar_url"
	DefaultSiteTitle       = "Yuem"
	DefaultSiteDescription = "Discover, create, and share moments with Yuem."
)

const (
	DefaultPaidContentBalanceMaxPrice                      = 2000
	DefaultPaidContentPointsMaxPrice                       = 50000
	DefaultFileRecycleRetentionDays                        = 30
	DefaultFileRecycleCleanupIntervalHours                 = 24
	DefaultNotificationInteractionSuppressionWindowSeconds = 600
	DefaultNotificationInteractionSuppressionThreshold     = 3
)

const (
	FileRecycleRetentionDaysKey        = "file_recycle_retention_days"
	FileRecycleCleanupIntervalHoursKey = "file_recycle_cleanup_interval_hours"

	notificationSuppressionEnabledKey   = "notification_interaction_suppression_enabled"
	notificationSuppressionWindowKey    = "notification_interaction_suppression_window_seconds"
	notificationSuppressionThresholdKey = "notification_interaction_suppression_threshold"
)

type SettingsService struct {
	db       *gorm.DB
	redis    *RedisStore
	mu       sync.Mutex
	snapshot atomic.Value
}

type SiteProfile struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	AvatarURL   string `json:"avatarUrl"`
}

type settingsSnapshot struct {
	values   map[string]any
	explicit map[string]struct{}
}

func NewSettingsService(db *gorm.DB, redis *RedisStore) *SettingsService {
	values := map[string]any{}
	maps.Copy(values, DefaultSettings)
	applySettingsEnvDefaults(values)
	applySettingsForcedEnvOverrides(values)
	s := &SettingsService{db: db, redis: redis}
	s.storeSnapshot(values, map[string]struct{}{})
	return s
}

func (s *SettingsService) Load(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	var rows []domain.SystemSetting
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.currentSnapshot()
	values := cloneSettingsMap(current.values)
	explicit := cloneExplicitSettings(current.explicit)
	for _, row := range rows {
		key := strings.TrimSpace(row.SettingKey)
		if key == "" {
			continue
		}
		values[key] = decodeSettingRaw(row.SettingValue)
		explicit[key] = struct{}{}
	}
	applySettingsForcedEnvOverrides(values)
	s.storeSnapshot(values, explicit)
}

func (s *SettingsService) Get(key string) any {
	if s == nil {
		return DefaultSettings[key]
	}
	value, ok := s.currentSnapshot().values[key]
	if ok {
		return value
	}
	return DefaultSettings[key]
}

func (s *SettingsService) Value(key string) (any, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false
	}
	if s == nil {
		value, ok := DefaultSettings[key]
		return value, ok
	}
	value, ok := s.currentSnapshot().values[key]
	if ok {
		return value, true
	}
	value, ok = DefaultSettings[key]
	return value, ok
}

func (s *SettingsService) ExplicitValue(key string) (any, bool) {
	key = strings.TrimSpace(key)
	if s == nil || key == "" {
		return nil, false
	}
	snapshot := s.currentSnapshot()
	_, explicit := snapshot.explicit[key]
	value, ok := snapshot.values[key]
	return value, explicit && ok
}

func (s *SettingsService) Set(ctx context.Context, key string, value any) bool {
	if s == nil {
		return false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if s.db != nil {
		if err := s.persist(ctx, key, value); err != nil {
			return false
		}
	}
	s.mu.Lock()
	current := s.currentSnapshot()
	values := cloneSettingsMap(current.values)
	explicit := cloneExplicitSettings(current.explicit)
	values[key] = value
	explicit[key] = struct{}{}
	s.storeSnapshot(values, explicit)
	s.mu.Unlock()
	if s.redis != nil {
		_ = s.redis.Set(ctx, SettingsKeyPrefix+key, value, 0)
		s.redis.BumpCacheVersion(ctx, "settings")
	}
	return true
}

func (s *SettingsService) All() map[string]any {
	out := map[string]any{}
	if s == nil {
		maps.Copy(out, DefaultSettings)
		return out
	}
	maps.Copy(out, s.currentSnapshot().values)
	return out
}

func (s *SettingsService) currentSnapshot() *settingsSnapshot {
	if s == nil {
		return defaultSettingsSnapshot()
	}
	if loaded := s.snapshot.Load(); loaded != nil {
		if snapshot, ok := loaded.(*settingsSnapshot); ok && snapshot != nil {
			return snapshot
		}
	}
	return defaultSettingsSnapshot()
}

func (s *SettingsService) storeSnapshot(values map[string]any, explicit map[string]struct{}) {
	if s == nil {
		return
	}
	s.snapshot.Store(&settingsSnapshot{
		values:   cloneSettingsMap(values),
		explicit: cloneExplicitSettings(explicit),
	})
}

func defaultSettingsSnapshot() *settingsSnapshot {
	values := map[string]any{}
	maps.Copy(values, DefaultSettings)
	return &settingsSnapshot{values: values, explicit: map[string]struct{}{}}
}

func cloneSettingsMap(values map[string]any) map[string]any {
	out := map[string]any{}
	if values == nil {
		maps.Copy(out, DefaultSettings)
		return out
	}
	maps.Copy(out, values)
	return out
}

func cloneExplicitSettings(values map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	maps.Copy(out, values)
	return out
}

func (s *SettingsService) Bool(key string) bool {
	return settingBool(s.Get(key))
}

func (s *SettingsService) String(key string) string {
	value := s.Get(key)
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func (s *SettingsService) Int(key string, fallback int) int {
	return settingInt(s.Get(key), fallback)
}

func settingInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed.String()))
		if err == nil {
			return parsed
		}
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func settingString(value any, fallback string) string {
	switch typed := value.(type) {
	case string:
		if text := strings.TrimSpace(typed); text != "" {
			return text
		}
	case fmt.Stringer:
		if text := strings.TrimSpace(typed.String()); text != "" {
			return text
		}
	}
	return fallback
}

func (s *SettingsService) Time(key string) (time.Time, bool) {
	return settingTime(s.Get(key))
}

func (s *SettingsService) VideoCenterAccountCutoff() (time.Time, bool) {
	return s.Time("video_center_account_cutoff")
}

func (s *SettingsService) ShouldHideVideoCenterForUser(createdAt time.Time) bool {
	cutoff, ok := s.VideoCenterAccountCutoff()
	return ok && !createdAt.IsZero() && !createdAt.Before(cutoff)
}

func (s *SettingsService) StringArray(key string) []string {
	return stringArraySettingValue(s.Get(key))
}

func (s *SettingsService) Localized(key, locale string) any {
	return localization.Value(s.Get(key), locale)
}

func (s *SettingsService) LocalizedString(key, locale string) string {
	value := s.Localized(key, locale)
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func (s *SettingsService) LocalizedStringArray(key, locale string) []string {
	return stringArraySettingValue(s.Localized(key, locale))
}

func ReadSiteProfileForLocale(settings *SettingsService, locale string) SiteProfile {
	if settings == nil {
		return SiteProfile{
			Title:       DefaultSiteTitle,
			Description: DefaultSiteDescription,
			AvatarURL:   "",
		}
	}
	return SiteProfile{
		Title:       settingString(settings.Localized(SiteTitleSetting, locale), DefaultSiteTitle),
		Description: settingString(settings.Localized(SiteDescriptionSetting, locale), DefaultSiteDescription),
		AvatarURL:   strings.TrimSpace(settings.String(SiteAvatarURLSetting)),
	}
}

func stringArraySettingValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case string:
		var parsed []string
		if json.Unmarshal([]byte(typed), &parsed) == nil {
			return parsed
		}
	}
	return []string{}
}

func (s *SettingsService) IsNoteGuestRestricted() bool { return s.Bool("guest_access_note_restricted") }
func (s *SettingsService) IsVideoGuestRestricted() bool {
	return s.Bool("guest_access_video_restricted")
}
func (s *SettingsService) IsUaBlockEnabled() bool     { return s.Bool("ua_block_enabled") }
func (s *SettingsService) UaBlockDevices() []string   { return s.StringArray("ua_block_devices") }
func (s *SettingsService) UaBlockPageHTML() string    { return s.String("ua_block_page_html") }
func (s *SettingsService) UaBlockRedirectURL() string { return s.String("ua_block_redirect_url") }
func (s *SettingsService) IsMaintenanceEnabled() bool { return s.Bool("maintenance_enabled") }

func decodeSettingRaw(raw string) any {
	var decoded any
	if json.Unmarshal([]byte(raw), &decoded) == nil {
		return decoded
	}
	return raw
}

func (s *SettingsService) persist(ctx context.Context, key string, value any) error {
	now := time.Now()
	encoded := encodeSettingValue(value)
	group := settingGroup(key)
	var row domain.SystemSetting
	err := s.db.WithContext(ctx).Where("setting_key = ?", key).Take(&row).Error
	if err == nil {
		return s.db.WithContext(ctx).Model(&domain.SystemSetting{}).Where("id = ?", row.ID).Updates(map[string]any{
			"setting_value": encoded,
			"setting_group": group,
			"updated_at":    now,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return s.db.WithContext(ctx).Create(&domain.SystemSetting{
		SettingKey:   key,
		SettingValue: encoded,
		SettingGroup: group,
		CreatedAt:    now,
		UpdatedAt:    &now,
	}).Error
}

func encodeSettingValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func settingGroup(key string) string {
	switch {
	case strings.HasPrefix(key, "post_"):
		return "content"
	case strings.HasPrefix(key, "recommend_"):
		return "recommendation"
	case strings.HasPrefix(key, "guest_access_"):
		return "access"
	case strings.HasPrefix(key, "video_center_"):
		return "video_center"
	case strings.HasPrefix(key, "ua_block_"):
		return "security"
	case strings.HasPrefix(key, "maintenance_"):
		return "maintenance"
	case strings.HasPrefix(key, "site_"):
		return "site"
	case strings.HasPrefix(key, "database_"):
		return "database"
	case strings.HasPrefix(key, "file_recycle_"):
		return "file_recycle"
	case strings.HasPrefix(key, "redis_"):
		return "redis"
	case strings.HasPrefix(key, "access_token_"), strings.HasPrefix(key, "refresh_token_"):
		return "auth"
	case strings.HasPrefix(key, "image_"), strings.HasPrefix(key, "hidden_watermark_"):
		return "image_processing"
	case strings.HasPrefix(key, "points_"):
		return "points"
	case strings.HasPrefix(key, "notification_"):
		return "notifications"
	case strings.HasPrefix(key, "onboarding_"):
		return "onboarding"
	case strings.HasPrefix(key, "oauth2_"):
		return "auth"
	case strings.HasPrefix(key, "ai_agent_"):
		return "ai"
	case strings.HasPrefix(key, "ai_"):
		return "audit"
	default:
		return "system"
	}
}

func applySettingsEnvDefaults(values map[string]any) {
	if raw, ok := os.LookupEnv("OAUTH2_APP_CALLBACK_URLS"); ok && strings.TrimSpace(raw) != "" {
		values[OAuth2AppCallbackURLsSetting] = strings.TrimSpace(raw)
	} else if raw, ok := os.LookupEnv("OAUTH2_APP_CALLBACK_URL"); ok && strings.TrimSpace(raw) != "" {
		values[OAuth2AppCallbackURLsSetting] = strings.TrimSpace(raw)
	}
	for envName, settingKey := range map[string]string{
		"JWT_EXPIRES_IN":           "access_token_ttl_seconds",
		"REFRESH_TOKEN_EXPIRES_IN": "refresh_token_active_ttl_seconds",
	} {
		raw, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		fallback := time.Duration(settingInt(DefaultSettings[settingKey], 0)) * time.Second
		if ttl := ParseAuthDuration(raw, fallback); ttl > 0 {
			values[settingKey] = int(ttl.Seconds())
		}
	}
	for envName, settingKey := range map[string]string{
		"WEBP_ENABLE_CONVERSION":          "image_webp_enabled",
		"WEBP_LOSSLESS":                   "image_webp_lossless",
		"IMAGE_LIBVIPS_ENABLED":           "image_libvips_enabled",
		"IMAGE_PROTECTION_ENABLED":        "image_protection_enabled",
		"HIDDEN_WATERMARK_ENABLED":        "hidden_watermark_enabled",
		"HIDDEN_WATERMARK_PROTECTED_ONLY": "hidden_watermark_protected_only",
	} {
		if value, ok := boolEnv(envName); ok {
			values[settingKey] = value
		}
	}
	for envName, settingKey := range map[string]string{
		"WEBP_QUALITY":        "image_webp_quality",
		"AVATAR_WEBP_QUALITY": "image_avatar_webp_quality",
		"WEBP_METHOD":         "image_webp_method",
		"WEBP_ALPHA_QUALITY":  "image_webp_alpha_quality",
		"WEBP_MAX_WIDTH":      "image_max_width",
		"WEBP_MAX_HEIGHT":     "image_max_height",
	} {
		raw, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			values[settingKey] = value
		}
	}
	if retention := durationEnvDefault("UPLOAD_RECYCLE_RETENTION", time.Duration(DefaultFileRecycleRetentionDays)*24*time.Hour); retention > 0 {
		values[FileRecycleRetentionDaysKey] = clampFileRecycleRetentionDays(int((retention + 24*time.Hour - 1) / (24 * time.Hour)))
	}
	if interval := durationEnvDefault("UPLOAD_RECYCLE_CLEANUP_INTERVAL", time.Duration(DefaultFileRecycleCleanupIntervalHours)*time.Hour); interval > 0 {
		values[FileRecycleCleanupIntervalHoursKey] = clampFileRecycleCleanupIntervalHours(int((interval + time.Hour - 1) / time.Hour))
	}
	if raw, ok := os.LookupEnv("HIDDEN_WATERMARK_ENGINE"); ok {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "auto", "local", "remote":
			values["hidden_watermark_engine"] = strings.ToLower(strings.TrimSpace(raw))
		}
	}
	for envName, settingKey := range map[string]string{
		"HIDDEN_WATERMARK_REMOTE_PASSWORD_WM":  "hidden_watermark_remote_password_wm",
		"HIDDEN_WATERMARK_REMOTE_PASSWORD_IMG": "hidden_watermark_remote_password_img",
		"HIDDEN_WATERMARK_REMOTE_CUSTOM_D1":    "hidden_watermark_remote_custom_d1",
		"HIDDEN_WATERMARK_REMOTE_CUSTOM_D2":    "hidden_watermark_remote_custom_d2",
		"IMAGE_PROTECTION_MAX_DIMENSION":       "image_protection_max_dimension",
		"IMAGE_PROTECTION_WEBP_QUALITY":        "image_protection_webp_quality",
	} {
		raw, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			values[settingKey] = value
		}
	}
}

func durationEnvDefault(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func applySettingsForcedEnvOverrides(values map[string]any) {
	for envName, settingKey := range map[string]string{
		"GUEST_ACCESS_RESTRICTED":       "guest_access_restricted",
		"GUEST_ACCESS_NOTE_RESTRICTED":  "guest_access_note_restricted",
		"GUEST_ACCESS_VIDEO_RESTRICTED": "guest_access_video_restricted",
		"GUEST_ACCESS_ADMIN_RESTRICTED": "guest_access_admin_restricted",
	} {
		if value, ok := boolEnv(envName); ok {
			values[settingKey] = value
		}
	}
}

func boolEnv(name string) (bool, bool) {
	rawValue, ok := os.LookupEnv(name)
	if !ok {
		return false, false
	}

	switch strings.ToLower(strings.TrimSpace(rawValue)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func settingBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		text := strings.TrimSpace(typed)
		var decoded any
		if json.Unmarshal([]byte(text), &decoded) == nil && decoded != typed {
			return settingBool(decoded)
		}
		return strings.EqualFold(text, "true") || text == "1"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func settingTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		if typed.IsZero() {
			return time.Time{}, false
		}
		return typed, true
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, text); err == nil {
				return parsed, true
			}
		}
	case float64:
		if typed <= 0 {
			return time.Time{}, false
		}
		if typed > 1_000_000_000_000 {
			return time.UnixMilli(int64(typed)).UTC(), true
		}
		return time.Unix(int64(typed), 0).UTC(), true
	case int64:
		if typed <= 0 {
			return time.Time{}, false
		}
		if typed > 1_000_000_000_000 {
			return time.UnixMilli(typed).UTC(), true
		}
		return time.Unix(typed, 0).UTC(), true
	case int:
		return settingTime(int64(typed))
	}
	return time.Time{}, false
}
