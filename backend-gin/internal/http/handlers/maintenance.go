package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) MaintenanceStatus(c *gin.Context) {
	state := services.ReadMaintenanceStateForLocale(h.Settings, localization.ResolveRequest(c.Request))
	bypass := false
	if raw, err := c.Cookie(services.MaintenanceBypassCookie); err == nil {
		bypass = services.ValidMaintenanceBypass(raw, state.EntryCode, h.Config.Auth.JWTSecret)
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"enabled":            state.Enabled,
		"message":            state.Message,
		"started_at":         state.StartedAt,
		"estimated_end_at":   state.EstimatedEndAt,
		"now":                time.Now().UTC().Format(time.RFC3339Nano),
		"bypass":             bypass,
		"border_visible":     state.BorderVisible,
		"border_color":       state.BorderColor,
		"border_opacity":     state.BorderOpacity,
		"border_dismissible": state.BorderDismissible,
	})
}

func (h NativeHandlers) MaintenanceEnter(c *gin.Context) {
	state := services.ReadMaintenanceStateForLocale(h.Settings, localization.ResolveRequest(c.Request))
	body := readBodyMap(c)
	code := strings.TrimSpace(firstNonEmpty(toString(body["code"]), c.Query("code")))
	if !state.Enabled {
		writeSuccess(c, matrixMsgOK, gin.H{"enabled": false})
		return
	}
	if state.EntryCode == "" || !strings.EqualFold(code, state.EntryCode) {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "maintenance.entry_invalid", gin.H{"enabled": true})
		return
	}
	h.setMaintenanceBypassCookie(c, state.EntryCode)
	data := gin.H{
		"enabled":          true,
		"bypass":           true,
		"message":          state.Message,
		"estimated_end_at": state.EstimatedEndAt,
	}
	if state.AutoLoginUID > 0 {
		if h.DB == nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		var user domain.User
		err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND is_active = ?", state.AutoLoginUID, true).First(&user).Error
		if writeDBError(c, err, "maintenance.user_not_found") {
			return
		}
		access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
		if !ok {
			return
		}
		data["user"] = h.userPublicMap(user)
		data["tokens"] = h.tokenMap(access, refresh)
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) AdminMaintenance(c *gin.Context) {
	state := services.ReadMaintenanceStateForLocale(h.Settings, localization.ResolveRequest(c.Request))
	entryURL := "/service-mode/" + state.EntryCode
	writeSuccess(c, matrixMsgOK, gin.H{
		"enabled":            state.Enabled,
		"entry_code":         state.EntryCode,
		"entry_url":          entryURL,
		"auto_login_uid":     state.AutoLoginUID,
		"started_at":         state.StartedAt,
		"estimated_end_at":   state.EstimatedEndAt,
		"message":            state.Message,
		"border_visible":     state.BorderVisible,
		"border_color":       state.BorderColor,
		"border_opacity":     state.BorderOpacity,
		"border_dismissible": state.BorderDismissible,
	})
}

func (h NativeHandlers) AdminMaintenanceUpdate(c *gin.Context) {
	if h.Settings == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	body := readBodyMap(c)
	enabled, _ := boolFromAny(body["enabled"])
	autoUID, _ := int64FromAny(body["auto_login_uid"])
	message := strings.TrimSpace(toString(body["message"]))
	startedAt := strings.TrimSpace(toString(body["started_at"]))
	estimatedEndAt := strings.TrimSpace(toString(body["estimated_end_at"]))
	borderVisible, hasBorderVisible := boolFromAny(body["border_visible"])
	borderDismissible, hasBorderDismissible := boolFromAny(body["border_dismissible"])
	borderColor := strings.TrimSpace(toString(body["border_color"]))
	borderOpacity := maintenanceOpacityFromBody(body["border_opacity"], currentMaintenanceOpacity(h.Settings))
	current := services.ReadMaintenanceState(h.Settings)
	entryCode := strings.TrimSpace(toString(body["entry_code"]))
	if entryCode == "" {
		entryCode = current.EntryCode
	}
	if entryCode == "" {
		entryCode = randomHex(12)
	}
	if enabled && startedAt == "" {
		startedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if !hasBorderVisible {
		borderVisible = current.BorderVisible
	}
	if !hasBorderDismissible {
		borderDismissible = current.BorderDismissible
	}
	if borderColor == "" {
		borderColor = current.BorderColor
	}
	updates := map[string]any{
		"maintenance_enabled":            enabled,
		"maintenance_entry_code":         entryCode,
		"maintenance_auto_login_uid":     autoUID,
		"maintenance_started_at":         startedAt,
		"maintenance_estimated_end_at":   estimatedEndAt,
		"maintenance_message":            message,
		"maintenance_border_visible":     borderVisible,
		"maintenance_border_color":       borderColor,
		"maintenance_border_opacity":     borderOpacity,
		"maintenance_border_dismissible": borderDismissible,
	}
	for key, value := range updates {
		if !h.Settings.Set(c.Request.Context(), key, value) {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
	}
	h.AdminMaintenance(c)
}

func currentMaintenanceOpacity(settings *services.SettingsService) float64 {
	if settings == nil {
		return 1
	}
	switch value := settings.Get("maintenance_border_opacity").(type) {
	case float64:
		return clampMaintenanceOpacity(value)
	case int:
		return clampMaintenanceOpacity(float64(value))
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return clampMaintenanceOpacity(parsed)
		}
	}
	return 1
}

func maintenanceOpacityFromBody(value any, fallback float64) float64 {
	switch typed := value.(type) {
	case float64:
		return clampMaintenanceOpacity(typed)
	case int:
		return clampMaintenanceOpacity(float64(typed))
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return clampMaintenanceOpacity(parsed)
		}
	}
	return clampMaintenanceOpacity(fallback)
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

func (h NativeHandlers) AdminMaintenanceRotateEntry(c *gin.Context) {
	if h.Settings == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	code := randomHex(12)
	if !h.Settings.Set(c.Request.Context(), "maintenance_entry_code", code) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	h.AdminMaintenance(c)
}

func (h NativeHandlers) setMaintenanceBypassCookie(c *gin.Context, entryCode string) {
	value := services.MaintenanceBypassValue(entryCode, h.Config.Auth.JWTSecret, time.Now())
	secure := h.Config.Server.Env == "production"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     services.MaintenanceBypassCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   int((24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}
