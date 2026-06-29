package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminResetAllOnboarding(c *gin.Context) {
	res := h.DB.WithContext(c.Request.Context()).
		Model(&domain.User{}).
		Where("profile_completed = ?", true).
		Updates(map[string]any{"profile_completed": false})
	if writeDBError(c, res.Error, "") {
		return
	}
	writeSuccess(c, "admin.onboarding_reset_all_done", gin.H{"affected_count": res.RowsAffected})
}

func adminResetOnboardingUserID(path string) string {
	value := strings.TrimPrefix(path, "/api/admin/users/")
	value = strings.TrimSuffix(value, "/reset-onboarding")
	value = strings.Trim(value, "/")
	if unescaped, err := url.PathUnescape(value); err == nil {
		value = unescaped
	}
	return strings.TrimSpace(value)
}

func (h NativeHandlers) adminResetUserOnboarding(c *gin.Context, identifier string) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.missing_user_id", nil)
		return
	}
	user, ok := h.adminFindOnboardingUser(c, identifier)
	if !ok {
		return
	}
	wasCompleted := user.ProfileCompleted
	res := h.DB.WithContext(c.Request.Context()).
		Model(&domain.User{}).
		Where("id = ?", user.ID).
		Update("profile_completed", false)
	if writeDBError(c, res.Error, "") {
		return
	}
	if h.Cache != nil {
		h.Cache.Delete("onboarding_draft:" + strconv.FormatInt(user.ID, 10))
	}
	writeSuccess(c, "admin.onboarding_reset_user_done", gin.H{
		"affected_count":    res.RowsAffected,
		"id":                user.ID,
		"nickname":          user.Nickname,
		"profile_completed": false,
		"user_id":           user.UserID,
		"was_completed":     wasCompleted,
	})
}

func (h NativeHandlers) adminFindOnboardingUser(c *gin.Context, identifier string) (domain.User, bool) {
	var user domain.User
	query := h.DB.WithContext(c.Request.Context())
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil && id > 0 {
		query = query.Where("id = ? OR user_id = ? OR xise_id = ?", id, identifier, identifier)
	} else {
		query = query.Where("user_id = ? OR xise_id = ?", identifier, identifier)
	}
	err := query.First(&user).Error
	if err == nil {
		return user, true
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.user_not_found", nil)
		return domain.User{}, false
	}
	writeDBError(c, err, "")
	return domain.User{}, false
}
