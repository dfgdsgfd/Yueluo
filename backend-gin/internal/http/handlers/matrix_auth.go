package handlers

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/security"
	"yuem-go/backend-gin/internal/services"
)

var authUserIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

const (
	authReadableAccessCookie = "yuem_access_token"
	authHTTPAccessCookie     = "yuem_http_access_token"
	authHTTPRefreshCookie    = "yuem_http_refresh_token"
	authLegacyRefreshCookie  = "yuem_refresh_token"
)

func (h NativeHandlers) authConfig(c *gin.Context) {
	locale := localization.ResolveRequest(c.Request)
	writeSuccess(c, matrixMsgOK, gin.H{
		"emailEnabled":                     h.Config.Email.Enabled,
		"oauth2Enabled":                    h.Config.OAuth2.Enabled,
		"oauth2OnlyLogin":                  h.Config.OAuth2.OnlyOAuth2,
		"oauth2LoginUrl":                   ternaryString(h.Config.OAuth2.Enabled, h.Config.OAuth2.LoginURL, ""),
		"oauth2StartUrl":                   ternaryString(h.Config.OAuth2.Enabled, "/api/auth/oauth2/login", ""),
		"geetestEnabled":                   h.Config.Geetest.Enabled,
		"geetestCaptchaId":                 ternaryString(h.Config.Geetest.Enabled, h.Config.Geetest.CaptchaID, ""),
		"verificationCollectSensitiveInfo": false,
		"videoCenterEnabled":               h.Settings == nil || h.Settings.Bool("video_center_enabled"),
		"videoCenterAccountCutoff":         videoCenterAccountCutoffSetting(h.Settings),
		"videoCenterGuestRestricted":       h.Settings != nil && h.Settings.IsVideoGuestRestricted(),
		"videoCenterRequestCountryCode":    h.videoCenterRequestCountryCode(c),
		"siteProfile":                      services.ReadSiteProfileForLocale(h.Settings, locale),
	})
}

func videoCenterAccountCutoffSetting(settings *services.SettingsService) string {
	if settings != nil {
		return settings.String("video_center_account_cutoff")
	}
	if value, ok := services.DefaultSettings["video_center_account_cutoff"].(string); ok {
		return value
	}
	return ""
}

func (h NativeHandlers) authEmailConfig(c *gin.Context) {
	writeSuccess(c, matrixMsgOK, gin.H{"emailEnabled": h.Config.Email.Enabled})
}

func (h NativeHandlers) authCaptcha(c *gin.Context) {
	text := strings.ToUpper(randomHex(3))[:4]
	id := strconv.FormatInt(time.Now().UnixNano(), 36) + randomHex(4)
	if h.Cache != nil {
		h.Cache.Set("captcha:"+id, text, 30*time.Second)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="42"><rect width="120" height="42" fill="#f3f4f6"/><text x="18" y="28" font-size="24" font-family="monospace" fill="#111827">%s</text></svg>`, html.EscapeString(text))
	writeSuccess(c, "验证码生成成功", gin.H{"captchaId": id, "captchaSvg": svg})
}

func (h NativeHandlers) authCheckUserID(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请输入汐社号", nil)
		return
	}
	var count int64
	err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("user_id = ?", userID).Count(&count).Error
	if writeDBError(c, err, "") {
		return
	}
	message := "汐社号可用"
	if count > 0 {
		message = "汐社号已存在"
	}
	writeSuccess(c, message, gin.H{"isUnique": count == 0})
}

func (h NativeHandlers) authSendEmailCode(c *gin.Context, reset bool) {
	if !h.Config.Email.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮件功能未启用", nil)
		return
	}
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	email := strings.TrimSpace(toString(body["email"]))
	if email == "" || !strings.Contains(email, "@") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮箱格式不正确", nil)
		return
	}
	var count int64
	err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("email = ?", email).Count(&count).Error
	if writeDBError(c, err, "") {
		return
	}
	if reset && count == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeNotFound, "该邮箱未绑定任何账号", nil)
		return
	}
	if !reset && count > 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeConflict, "该邮箱已被注册", nil)
		return
	}
	code := strconv.Itoa(100000 + int(time.Now().UnixNano()%900000))
	key := "email:" + email
	if reset {
		key = "email:reset:" + email
	}
	if h.Cache != nil {
		h.Cache.Set(key, code, 10*time.Minute)
	}
	if err := h.sendEmailVerificationCode(email, code); err != nil {
		if h.Cache != nil {
			h.Cache.Delete(key)
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "验证码发送失败: "+err.Error(), nil)
		return
	}
	data := gin.H{}
	if reset {
		var user domain.User
		if err := h.DB.WithContext(c.Request.Context()).Where("email = ?", email).Select("user_id").First(&user).Error; err == nil {
			data["user_id"] = user.UserID
		}
	}
	if len(data) == 0 {
		writeSimpleSuccess(c, "验证码发送成功，请查收邮箱")
		return
	}
	writeSuccess(c, "验证码发送成功，请查收邮箱", data)
}

func (h NativeHandlers) authVerifyResetCode(c *gin.Context) {
	body := readBodyMap(c)
	email := strings.TrimSpace(toString(body["email"]))
	code := strings.TrimSpace(toString(body["emailCode"]))
	if !h.checkCachedCode("email:reset:"+email, code) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "验证码错误", nil)
		return
	}
	writeSimpleSuccess(c, "验证码验证成功")
}

func (h NativeHandlers) authResetPassword(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	email := strings.TrimSpace(toString(body["email"]))
	code := strings.TrimSpace(toString(body["emailCode"]))
	password := toString(body["newPassword"])
	if email == "" || code == "" || password == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, matrixMsgMissingParams, nil)
		return
	}
	if len(password) < 6 || len(password) > 20 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "密码长度必须在6-20位之间", nil)
		return
	}
	if !h.checkCachedCode("email:reset:"+email, code) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "验证码错误", nil)
		return
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	err = h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("email = ?", email).Update("password", passwordHash).Error
	if writeDBError(c, err, "") {
		return
	}
	if h.Cache != nil {
		h.Cache.Delete("email:reset:" + email)
	}
	writeSimpleSuccess(c, "密码重置成功，请使用新密码登录")
}

func (h NativeHandlers) authBindEmail(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok || !h.requireDB(c) {
		return
	}
	if !h.Config.Email.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮件功能未启用", nil)
		return
	}
	body := readBodyMap(c)
	email := strings.TrimSpace(toString(body["email"]))
	code := strings.TrimSpace(toString(body["emailCode"]))
	if email == "" || code == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请输入邮箱和验证码", nil)
		return
	}
	var count int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("email = ? AND id <> ?", email, user.ID).Count(&count).Error; writeDBError(c, err, "") {
		return
	}
	if count > 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeConflict, "该邮箱已被其他用户绑定", nil)
		return
	}
	if !h.checkCachedCode("email:"+email, code) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮箱验证码错误", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", user.ID).Update("email", email).Error; writeDBError(c, err, "") {
		return
	}
	if h.Cache != nil {
		h.Cache.Delete("email:" + email)
	}
	writeSuccess(c, "邮箱绑定成功", gin.H{"email": email})
}

func (h NativeHandlers) authUnbindEmail(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok || !h.requireDB(c) {
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", user.ID).Update("email", "").Error; writeDBError(c, err, "") {
		return
	}
	writeSimpleSuccess(c, "邮箱解绑成功")
}

func (h NativeHandlers) authRegister(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	userID := strings.TrimSpace(toString(body["user_id"]))
	nickname := sanitizePlainSubmittedText(toString(body["nickname"]))
	password := toString(body["password"])
	if userID == "" || nickname == "" || password == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, matrixMsgMissingParams, nil)
		return
	}
	if len(userID) < 3 || len(userID) > 15 || !authUserIDPattern.MatchString(userID) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "汐社号只能包含3-15位字母、数字和下划线", nil)
		return
	}
	if len([]rune(nickname)) > 10 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "昵称长度必须少于10位", nil)
		return
	}
	if len(password) < 6 || len(password) > 20 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "密码长度必须在6-20位之间", nil)
		return
	}
	if h.Config.Geetest.Enabled {
		for _, key := range []string{"lot_number", "captcha_output", "pass_token", "gen_time"} {
			if toString(body[key]) == "" {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少验证码参数", nil)
				return
			}
		}
	} else if !h.checkCachedCode("captcha:"+toString(body["captchaId"]), toString(body["captchaText"])) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "验证码错误", nil)
		return
	}
	email := strings.TrimSpace(toString(body["email"]))
	if email != "" && !strings.Contains(email, "@") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮箱格式不正确", nil)
		return
	}
	if h.Config.Email.Enabled {
		if email == "" || !h.checkCachedCode("email:"+email, toString(body["emailCode"])) {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "邮箱验证码错误", nil)
			return
		}
	}
	var count int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("user_id = ?", userID).Count(&count).Error; writeDBError(c, err, "") {
		return
	}
	if count > 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeConflict, "用户ID已存在", nil)
		return
	}
	if email != "" {
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("LOWER(COALESCE(email, '')) = LOWER(?)", email).Count(&count).Error; writeDBError(c, err, "") {
			return
		}
		if count > 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeConflict, "该邮箱已被注册", nil)
			return
		}
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	user := domain.User{UserID: userID, Nickname: nickname, Password: stringPtr(passwordHash), Email: &email, Avatar: stringPtr(""), Bio: stringPtr(""), Location: stringPtr("未知"), IsActive: true}
	if err := h.DB.WithContext(c.Request.Context()).Create(&user).Error; writeDBError(c, err, "") {
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	writeSuccess(c, "注册成功", gin.H{"user": h.userPublicMap(user), "tokens": h.tokenMap(access, refresh)})
}

func (h NativeHandlers) authLogin(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	identifier := strings.TrimSpace(firstNonEmpty(
		toString(body["identifier"]),
		toString(body["account"]),
		toString(body["email"]),
		toString(body["user_id"]),
	))
	password := toString(body["password"])
	if identifier == "" || password == "" {
		h.recordSecurityAudit(c, "login", "password", "failure", "missing_params", http.StatusBadRequest, nil, "user", identifier, nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, matrixMsgMissingParams, nil)
		return
	}
	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).
		Where("user_id = ? OR LOWER(COALESCE(email, '')) = LOWER(?)", identifier, identifier).
		First(&user).Error
	if errorsIsNotFound(err) {
		h.recordSecurityAudit(c, "login", "password", "failure", "user_not_found", http.StatusBadRequest, nil, "user", identifier, nil)
		response.JSON(c, http.StatusBadRequest, response.CodeNotFound, "用户不存在", nil)
		return
	}
	if writeDBError(c, err, "") {
		return
	}
	if !user.IsActive {
		h.recordSecurityAudit(c, "login", "password", "failure", "account_disabled", http.StatusForbidden, &user.ID, "user", user.UserID, nil)
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "账户已被禁用", nil)
		return
	}
	if user.Password == nil || !security.VerifyPassword(password, *user.Password) {
		h.recordSecurityAudit(c, "login", "password", "failure", "invalid_password", http.StatusBadRequest, &user.ID, "user", user.UserID, nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "密码错误", nil)
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	bodyUser := h.userPublicMap(user)
	h.recordSecurityAudit(c, "login", "password", "success", "", http.StatusOK, &user.ID, "user", user.UserID, nil)
	writeSuccess(c, "登录成功", gin.H{"user": bodyUser, "tokens": h.tokenMap(access, refresh)})
}

func (h NativeHandlers) authRefresh(c *gin.Context) {
	if h.Auth == nil {
		h.recordAuthRefreshAudit(c, "failure", "auth_unavailable", nil, "", nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	body := readBodyMap(c)
	refreshToken := toString(body["refresh_token"])
	if refreshToken == "" {
		refreshToken = h.authRefreshTokenFromRequest(c)
	}
	if refreshToken == "" {
		h.recordAuthRefreshAudit(c, "failure", "missing_refresh_token", nil, "", nil)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	if h.Auth.TokenPolicy().RefreshTokenMode == services.RefreshTokenModeJWTLegacy {
		h.authRefreshJWTLegacy(c, refreshToken)
		return
	}
	h.authRefreshRedisOpaque(c, refreshToken)
}

func (h NativeHandlers) authRefreshRedisOpaque(c *gin.Context, refreshToken string) {
	session, ok := h.Auth.FindSessionByRefreshToken(c.Request.Context(), refreshToken, 0)
	if !ok {
		h.recordAuthRefreshAudit(c, "failure", "invalid_or_expired_refresh_token", nil, "", nil)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	userID, ok := int64FromAny(session.UserID)
	if !ok || userID <= 0 {
		h.recordAuthRefreshAudit(c, "failure", "invalid_session_user", nil, "", gin.H{"session_id": session.ID})
		h.Auth.DeleteSession(c.Request.Context(), *session)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	displayID, active := h.refreshDisplayID(c, userID)
	if !active {
		h.recordAuthRefreshAudit(c, "failure", "user_inactive_or_missing", &userID, displayID, gin.H{"session_id": session.ID})
		h.Auth.DeleteSession(c.Request.Context(), *session)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	access, err := h.Auth.GenerateAccessToken(gin.H{"userId": userID, "user_id": displayID})
	if err != nil {
		h.recordAuthRefreshAudit(c, "failure", "access_token_issue_failed", &userID, displayID, gin.H{"session_id": session.ID})
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	if !h.Auth.RefreshSessionAccessToken(c.Request.Context(), *session, access, time.Now()) {
		h.recordAuthRefreshAudit(c, "failure", "session_update_failed", &userID, displayID, gin.H{"session_id": session.ID})
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	h.setAuthCookies(c, access, refreshToken)
	h.recordAuthRefreshAudit(c, "success", "", &userID, displayID, gin.H{"session_id": session.ID})
	writeSuccess(c, "令牌刷新成功", h.tokenMap(access, refreshToken))
}

func (h NativeHandlers) authRefreshJWTLegacy(c *gin.Context, refreshToken string) {
	claims, ok := h.Auth.VerifyTokenClaims(refreshToken)
	if !ok || !h.validJWTLegacyRefreshClaims(claims) {
		h.recordAuthRefreshAudit(c, "failure", "invalid_or_expired_refresh_token", nil, "", nil)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	userID, ok := int64FromAny(claims["userId"])
	if !ok || userID <= 0 {
		h.recordAuthRefreshAudit(c, "failure", "invalid_token_user", nil, "", nil)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	displayID, active := h.refreshDisplayID(c, userID)
	if !active {
		h.recordAuthRefreshAudit(c, "failure", "user_inactive_or_missing", &userID, displayID, nil)
		h.clearAuthCookies(c)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效或已过期", nil)
		return
	}
	access, err := h.Auth.GenerateAccessToken(gin.H{"userId": userID, "user_id": displayID})
	if err != nil {
		h.recordAuthRefreshAudit(c, "failure", "access_token_issue_failed", &userID, displayID, nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	newRefresh, err := h.Auth.GenerateRefreshToken(gin.H{"userId": userID, "user_id": displayID})
	if err != nil {
		h.recordAuthRefreshAudit(c, "failure", "refresh_token_issue_failed", &userID, displayID, nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
		return
	}
	session, sessionOK := h.Auth.FindSessionByRefreshToken(c.Request.Context(), refreshToken, userID)
	now := time.Now()
	metadata := gin.H{}
	if sessionOK {
		metadata["session_id"] = session.ID
		if !h.Auth.RotateSessionTokens(c.Request.Context(), *session, access, newRefresh, now) {
			h.recordAuthRefreshAudit(c, "failure", "session_update_failed", &userID, displayID, metadata)
			response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
			return
		}
	} else {
		metadata["session_recreated"] = true
		if !h.createUserSession(c, userID, access, newRefresh) {
			h.recordAuthRefreshAudit(c, "failure", "session_update_failed", &userID, displayID, metadata)
			response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "刷新令牌无效", nil)
			return
		}
	}
	h.setAuthCookies(c, access, newRefresh)
	h.recordAuthRefreshAudit(c, "success", "", &userID, displayID, metadata)
	writeSuccess(c, "令牌刷新成功", h.tokenMap(access, newRefresh))
}

func (h NativeHandlers) validJWTLegacyRefreshClaims(claims map[string]any) bool {
	if tokenType := strings.TrimSpace(toString(claims["token_type"])); tokenType != "" {
		return tokenType == "refresh"
	}
	issuedAt, hasIssuedAt := int64FromAny(claims["iat"])
	expiresAt, hasExpiresAt := int64FromAny(claims["exp"])
	if !hasIssuedAt || !hasExpiresAt || expiresAt <= issuedAt {
		return true
	}
	ttl := time.Duration(expiresAt-issuedAt) * time.Second
	accessTTL := time.Hour
	if h.Auth != nil {
		accessTTL = h.Auth.TokenPolicy().AccessTokenTTL
	}
	return ttl > accessTTL+time.Minute
}

func (h NativeHandlers) refreshDisplayID(c *gin.Context, userID int64) (string, bool) {
	fallback := strconv.FormatInt(userID, 10)
	if h.DB == nil || userID <= 0 {
		return fallback, false
	}
	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND is_active = ?", userID, true).Select("id", "user_id", "is_active").First(&user).Error
	if err != nil || strings.TrimSpace(user.UserID) == "" {
		return fallback, false
	}
	return user.UserID, true
}

func (h NativeHandlers) recordAuthRefreshAudit(c *gin.Context, outcome string, reasonCode string, actorID *int64, actorDisplayID string, metadata map[string]any) {
	if metadata == nil {
		metadata = gin.H{}
	}
	metadata["token_material_logged"] = false
	if h.Auth != nil {
		metadata["refresh_token_mode"] = h.Auth.TokenPolicy().RefreshTokenMode
	}
	status := http.StatusOK
	if outcome != "success" {
		status = http.StatusUnauthorized
	}
	h.recordSecurityAudit(c, "token", "refresh", outcome, reasonCode, status, actorID, "user", actorDisplayID, metadata)
}

func (h NativeHandlers) authToken(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	apiKey := toString(body["api_key"])
	if apiKey == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少API密钥", nil)
		return
	}
	if h.apiKeyRequestBlocked(c, userAPIKeyScope) {
		return
	}
	row, err := h.lookupUserAPIKey(c.Request.Context(), apiKey)
	if errorsIsNotFound(err) {
		h.rejectInvalidAPIKey(c, userAPIKeyScope, "API密钥无效或已被禁用")
		return
	}
	if writeDBError(c, err, "") {
		return
	}
	h.acceptAPIKey(c, userAPIKeyScope)
	var user domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND is_active = ?", row.UserID, true).First(&user).Error; writeDBError(c, err, "用户账户已被禁用") {
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	h.touchUserAPIKey(row.ID)
	writeSuccess(c, "API密钥验证成功", h.tokenMap(access, refresh))
}

func (h NativeHandlers) authLogout(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok {
		return
	}
	if h.Auth != nil {
		h.Auth.DeactivateSessionByToken(c.Request.Context(), user.Token, user.ID)
	}
	h.clearAuthCookies(c)
	h.recordSecurityAudit(c, "login", "logout", "success", "", http.StatusOK, &user.ID, user.Type, firstNonEmptyHandler(user.UserID, user.Username), nil)
	writeSimpleSuccess(c, "退出成功")
}
