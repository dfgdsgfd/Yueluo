package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MaintenanceBypassCookie = "yuem_maintenance_bypass"

var maintenanceHexColorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

type MaintenanceState struct {
	Enabled           bool    `json:"enabled"`
	EntryCode         string  `json:"entry_code,omitempty"`
	AutoLoginUID      int64   `json:"auto_login_uid,omitempty"`
	StartedAt         string  `json:"started_at,omitempty"`
	EstimatedEndAt    string  `json:"estimated_end_at,omitempty"`
	Message           string  `json:"message,omitempty"`
	BorderVisible     bool    `json:"border_visible"`
	BorderColor       string  `json:"border_color"`
	BorderOpacity     float64 `json:"border_opacity"`
	BorderDismissible bool    `json:"border_dismissible"`
}

func ReadMaintenanceState(settings *SettingsService) MaintenanceState {
	return ReadMaintenanceStateForLocale(settings, "en")
}

func ReadMaintenanceStateForLocale(settings *SettingsService, locale string) MaintenanceState {
	if settings == nil {
		return MaintenanceState{}
	}
	return MaintenanceState{
		Enabled:           settings.Bool("maintenance_enabled"),
		EntryCode:         strings.TrimSpace(settings.String("maintenance_entry_code")),
		AutoLoginUID:      int64(settings.Int("maintenance_auto_login_uid", 0)),
		StartedAt:         strings.TrimSpace(settings.String("maintenance_started_at")),
		EstimatedEndAt:    strings.TrimSpace(settings.String("maintenance_estimated_end_at")),
		Message:           strings.TrimSpace(settings.LocalizedString("maintenance_message", locale)),
		BorderVisible:     settings.Bool("maintenance_border_visible"),
		BorderColor:       normalizeMaintenanceBorderColor(settings.String("maintenance_border_color")),
		BorderOpacity:     normalizeMaintenanceBorderOpacity(settings.Get("maintenance_border_opacity")),
		BorderDismissible: settings.Bool("maintenance_border_dismissible"),
	}
}

func normalizeMaintenanceBorderColor(value string) string {
	value = strings.TrimSpace(value)
	if !maintenanceHexColorPattern.MatchString(value) {
		return "#dc2626"
	}
	return value
}

func normalizeMaintenanceBorderOpacity(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return clampMaintenanceOpacity(typed)
	case int:
		return clampMaintenanceOpacity(float64(typed))
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return clampMaintenanceOpacity(parsed)
		}
	}
	return 1
}

func clampMaintenanceOpacity(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func MaintenanceBypassValue(entryCode, secret string, now time.Time) string {
	entryCode = strings.TrimSpace(entryCode)
	ts := strconv.FormatInt(now.Unix(), 10)
	payload := entryCode + ":" + ts
	return payload + ":" + maintenanceSignature(payload, secret)
}

func ValidMaintenanceBypass(raw, entryCode, secret string) bool {
	entryCode = strings.TrimSpace(entryCode)
	if raw == "" || entryCode == "" {
		return false
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 3 || parts[0] != entryCode {
		return false
	}
	payload := parts[0] + ":" + parts[1]
	expected := maintenanceSignature(payload, secret)
	return hmac.Equal([]byte(expected), []byte(parts[2]))
}

func maintenanceSignature(payload, secret string) string {
	if secret == "" {
		secret = "maintenance"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
