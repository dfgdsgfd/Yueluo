package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) checkCachedCode(key string, code string) bool {
	if key == "" || code == "" || h.Cache == nil {
		return false
	}
	value, ok := h.Cache.Get(key)
	if !ok {
		return false
	}
	return strings.EqualFold(toString(value), strings.TrimSpace(code))
}

func (h NativeHandlers) issueUserTokens(c *gin.Context, userID int64, displayID string) (string, string, bool) {
	if h.Auth == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return "", "", false
	}
	access, err := h.Auth.GenerateAccessToken(gin.H{"userId": userID, "user_id": displayID})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return "", "", false
	}
	refresh, err := h.Auth.GenerateRefreshToken(gin.H{"userId": userID, "user_id": displayID})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return "", "", false
	}
	h.createUserSession(c, userID, access, refresh)
	h.setAuthCookies(c, access, refresh)
	return access, refresh, true
}

func (h NativeHandlers) createUserSession(c *gin.Context, userID int64, access string, refresh string) bool {
	if h.Auth == nil {
		return false
	}
	return h.Auth.CreateSession(c.Request.Context(), services.Session{
		UserID:       strconv.FormatInt(userID, 10),
		Token:        access,
		RefreshToken: refresh,
		UserAgent:    c.GetHeader("User-Agent"),
		ClientIP:     h.clientIP(c),
	}, h.Auth.TokenPolicy().RefreshTokenActiveTTL)
}

func (h NativeHandlers) authAccessTokenFromRequest(c *gin.Context) string {
	if bearer := services.ExtractBearerToken(c.GetHeader("Authorization")); bearer != "" {
		return bearer
	}
	for _, name := range []string{authHTTPAccessCookie, authReadableAccessCookie, "access_token", "token"} {
		if token, err := c.Cookie(name); err == nil && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}
	return ""
}

func (h NativeHandlers) authRefreshTokenFromRequest(c *gin.Context) string {
	for _, name := range []string{authHTTPRefreshCookie, authLegacyRefreshCookie, "refresh_token"} {
		if token, err := c.Cookie(name); err == nil && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}
	return ""
}

func (h NativeHandlers) setAuthCookies(c *gin.Context, access string, refresh string) {
	secure := h.authCookieSecure(c)
	accessMaxAge := authCookieMaxAge(h.Auth, true, h.Config.Auth.JWTExpiresIn, time.Hour)
	refreshMaxAge := authCookieMaxAge(h.Auth, false, h.Config.Auth.RefreshTokenExpiresIn, 90*24*time.Hour)
	if access != "" {
		http.SetCookie(c.Writer, &http.Cookie{Name: authReadableAccessCookie, Value: access, Path: "/", MaxAge: accessMaxAge, SameSite: http.SameSiteLaxMode, Secure: secure})
		http.SetCookie(c.Writer, &http.Cookie{Name: authHTTPAccessCookie, Value: access, Path: "/", MaxAge: accessMaxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
	if refresh != "" {
		http.SetCookie(c.Writer, &http.Cookie{Name: authHTTPRefreshCookie, Value: refresh, Path: "/", MaxAge: refreshMaxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
}

func (h NativeHandlers) clearAuthCookies(c *gin.Context) {
	secure := h.authCookieSecure(c)
	for _, name := range []string{authReadableAccessCookie, authHTTPAccessCookie, authHTTPRefreshCookie, authLegacyRefreshCookie, "access_token", "refresh_token", "token"} {
		http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: name == authHTTPAccessCookie || name == authHTTPRefreshCookie, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
}

func (h NativeHandlers) authCookieSecure(c *gin.Context) bool {
	if h.Config.Server.Env == "production" {
		return true
	}
	if c != nil && c.Request != nil {
		if c.Request.TLS != nil {
			return true
		}
		if strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	return false
}

func authCookieMaxAge(auth *services.AuthService, access bool, value string, fallback time.Duration) int {
	duration := services.ParseAuthDuration(value, fallback)
	if auth != nil {
		policy := auth.TokenPolicy()
		if access {
			duration = policy.AccessTokenTTL
		} else {
			duration = policy.RefreshTokenActiveTTL
		}
	}
	return int(duration.Seconds())
}

func parseAuthDuration(value string, fallback time.Duration) time.Duration {
	return services.ParseAuthDuration(value, fallback)
}

func (h NativeHandlers) tokenMap(access string, refresh string) gin.H {
	expiresIn := int(services.ParseAuthDuration(h.Config.Auth.JWTExpiresIn, time.Hour).Seconds())
	if h.Auth != nil {
		expiresIn = int(h.Auth.TokenPolicy().AccessTokenTTL.Seconds())
	}
	return gin.H{"access_token": access, "refresh_token": refresh, "expires_in": expiresIn}
}

func (h NativeHandlers) adminTokenMap(access string, refresh string) gin.H {
	expiresIn := int(services.ParseAuthDuration(h.Config.Auth.AdminJWTExpiresIn, services.ParseAuthDuration(h.Config.Auth.JWTExpiresIn, time.Hour)).Seconds())
	if h.Auth != nil {
		expiresIn = int(h.Auth.AdminAccessTokenTTL().Seconds())
	}
	return gin.H{"access_token": access, "refresh_token": refresh, "expires_in": expiresIn}
}

func (h NativeHandlers) findOrCreateOAuthUser(c *gin.Context, oauthID int64, username string, email string) (*domain.User, bool, bool) {
	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).Where("oauth2_id = ?", oauthID).First(&user).Error
	if err == nil {
		if !user.IsActive {
			response.JSON(c, http.StatusForbidden, response.CodeForbidden, "账号已被禁用", nil)
			return nil, false, false
		}
		return &user, false, true
	}
	if !errorsIsNotFound(err) {
		writeDBError(c, err, "")
		return nil, false, false
	}
	base := username
	if base == "" {
		base = "user_" + strconv.FormatInt(oauthID, 10)
	}
	userID := base
	for i := 0; ; i++ {
		var count int64
		_ = h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("user_id = ?", userID).Count(&count).Error
		if count == 0 {
			break
		}
		userID = base + "_" + strconv.Itoa(i+1)
	}
	nickname := username
	if nickname == "" {
		nickname = "用户" + strconv.FormatInt(oauthID, 10)
	}
	user = domain.User{UserID: userID, Nickname: nickname, Password: nil, Email: &email, Avatar: stringPtr(""), Bio: stringPtr(""), Location: stringPtr("未知"), OAuth2ID: &oauthID, IsActive: true}
	if err := h.DB.WithContext(c.Request.Context()).Create(&user).Error; writeDBError(c, err, "") {
		return nil, false, false
	}
	return &user, true, true
}

func ternaryString(cond bool, yes string, no string) string {
	if cond {
		return yes
	}
	return no
}

func errorsIsNotFound(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrRecordNotFound)
}

func absoluteFrontendRedirect(base string, values url.Values) string {
	if base == "" {
		base = "/"
	}
	if strings.Contains(base, "?") {
		return base + "&" + values.Encode()
	}
	return strings.TrimRight(base, "/") + "/explore?" + values.Encode()
}
