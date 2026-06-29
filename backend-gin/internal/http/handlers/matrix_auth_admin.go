package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/security"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) authMe(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok || !h.requireDB(c) {
		return
	}
	var row domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", user.ID).First(&row).Error; writeDBError(c, err, "用户不存在") {
		return
	}
	writeSuccess(c, matrixMsgOK, h.userPublicMap(row))
}

func (h NativeHandlers) authAdminLogin(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	ip := h.clientIP(c)
	if lock := h.adminLoginIPLockStatus(c.Request.Context(), ip); lock.Locked {
		h.writeAdminLoginLocked(c, ip, "", nil, lock)
		return
	}
	body := readBodyMap(c)
	username := toString(body["username"])
	password := toString(body["password"])
	if username == "" || password == "" {
		h.recordSecurityAudit(c, "admin_login", "password", "failure", "missing_params", http.StatusBadRequest, nil, "admin", username, nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.missing_params", nil)
		return
	}

	var admin domain.Admin
	err := h.DB.WithContext(c.Request.Context()).Where("username = ?", username).First(&admin).Error
	if errorsIsNotFound(err) {
		failure := h.recordAdminLoginIPFailure(c.Request.Context(), ip)
		if failure.Locked {
			h.writeAdminLoginLocked(c, ip, username, nil, failure)
			return
		}
		h.recordSecurityAudit(c, "admin_login", "password", "failure", "admin_not_found", http.StatusBadRequest, nil, "admin", username, gin.H{
			"failed_attempts": failure.FailedAttempts,
			"ip_scope":        true,
		})
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.admin_invalid_credentials", nil)
		return
	}
	if writeDBError(c, err, "") {
		return
	}

	if !security.VerifyPassword(password, admin.Password) {
		failure := h.recordAdminLoginIPFailure(c.Request.Context(), ip)
		adminID := int64(admin.ID)
		if failure.Locked {
			h.writeAdminLoginLocked(c, ip, admin.Username, &adminID, failure)
			return
		}
		h.recordSecurityAudit(c, "admin_login", "password", "failure", "invalid_password", http.StatusBadRequest, &adminID, "admin", admin.Username, gin.H{
			"failed_attempts": failure.FailedAttempts,
			"ip_scope":        true,
			"remaining":       max(0, adminLoginMaxFailures-failure.FailedAttempts),
		})
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.admin_invalid_credentials", gin.H{
			"remaining_attempts": max(0, adminLoginMaxFailures-failure.FailedAttempts),
		})
		return
	}

	h.clearAdminLoginIPFailures(c.Request.Context(), ip)
	access, err := h.Auth.GenerateAdminAccessToken(gin.H{"adminId": admin.ID, "username": admin.Username, "type": "admin"})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	refresh, err := h.Auth.GenerateRefreshToken(gin.H{"adminId": admin.ID, "username": admin.Username, "type": "admin"})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	adminID := int64(admin.ID)
	h.recordSecurityAudit(c, "admin_login", "password", "success", "", http.StatusOK, &adminID, "admin", admin.Username, nil)
	writeSuccess(c, "success.admin_login", gin.H{"admin": gin.H{"id": int(admin.ID), "username": admin.Username}, "tokens": h.adminTokenMap(access, refresh)})
}

func (h NativeHandlers) authAdminMe(c *gin.Context) {
	user, ok := h.requireMatrixAdmin(c)
	if !ok {
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"id": int(user.ID), "username": user.Username})
}

func (h NativeHandlers) authAdminAdmins(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	id := matrixParam(c, "id")
	method := matrixMethod(c)
	switch {
	case method == http.MethodGet && id == "":
		page, limit, offset := pageLimit(c, 20)
		var total int64
		query := h.DB.WithContext(c.Request.Context()).Model(&domain.Admin{})
		if username := c.Query("username"); username != "" {
			query = query.Where("username LIKE ?", "%"+username+"%")
		}
		if err := query.Count(&total).Error; writeDBError(c, err, "") {
			return
		}
		var rows []domain.Admin
		if err := query.Select("id", "username", "created_at").Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
			return
		}
		data := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			data = append(data, gin.H{"id": int(row.ID), "username": row.Username, "created_at": row.CreatedAt})
		}
		writeSuccess(c, matrixMsgOK, gin.H{"data": data, "pagination": matrixPagination(page, limit, total)})
	case method == http.MethodGet && id != "":
		var row domain.Admin
		if err := h.DB.WithContext(c.Request.Context()).Where("id = ? OR username = ?", id, id).Select("id", "username", "created_at").First(&row).Error; writeDBError(c, err, "管理员不存在") {
			return
		}
		writeSuccess(c, matrixMsgOK, gin.H{"id": int(row.ID), "username": row.Username, "created_at": row.CreatedAt})
	case method == http.MethodPost:
		body := readBodyMap(c)
		username := toString(body["username"])
		password := toString(body["password"])
		if username == "" || password == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "账号和密码不能为空", nil)
			return
		}
		passwordHash, err := security.HashPassword(password)
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		row := domain.Admin{Username: username, Password: passwordHash}
		if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; writeDBError(c, err, "") {
			return
		}
		writeSuccess(c, "创建管理员成功", gin.H{"id": int(row.ID)})
	case method == http.MethodPut && strings.HasSuffix(c.Request.URL.Path, "/password"):
		body := readBodyMap(c)
		password := toString(body["password"])
		if len(password) < 6 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "密码不能为空且长度不能少于6位", nil)
			return
		}
		passwordHash, err := security.HashPassword(password)
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
		err = h.DB.WithContext(c.Request.Context()).Model(&domain.Admin{}).Where("id = ?", id).Update("password", passwordHash).Error
		if writeDBError(c, err, "") {
			return
		}
		writeSimpleSuccess(c, "密码重置成功")
	case method == http.MethodPut && id != "":
		body := readBodyMap(c)
		updates := gin.H{}
		if username := toString(body["username"]); username != "" {
			updates["username"] = username
		}
		if password := toString(body["password"]); password != "" {
			passwordHash, err := security.HashPassword(password)
			if err != nil {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return
			}
			updates["password"] = passwordHash
		}
		if len(updates) == 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有需要更新的数据", nil)
			return
		}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Admin{}).Where("id = ? OR username = ?", id, id).Updates(updates).Error; writeDBError(c, err, "") {
			return
		}
		writeSimpleSuccess(c, "更新管理员信息成功")
	case method == http.MethodDelete:
		query := h.DB.WithContext(c.Request.Context()).Where("1=1")
		if id != "" {
			query = query.Where("id = ? OR username = ?", id, id)
		} else {
			ids := parseStringSlice(readBodyMap(c)["ids"])
			if len(ids) == 0 {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请选择要删除的管理员", nil)
				return
			}
			query = query.Where("id IN ? OR username IN ?", ids, ids)
		}
		if err := query.Delete(&domain.Admin{}).Error; writeDBError(c, err, "") {
			return
		}
		writeSimpleSuccess(c, "删除管理员成功")
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "route not found", nil)
	}
}

func (h NativeHandlers) authOAuthLogin(c *gin.Context) {
	h.oauthLogin(c)
}

func (h NativeHandlers) authOAuthCallback(c *gin.Context) {
	h.oauthCallback(c)
}

func (h NativeHandlers) authOAuthMobileToken(c *gin.Context) {
	if !h.Config.OAuth2.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "OAuth2登录未启用", nil)
		return
	}
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	userToken := strings.TrimSpace(toString(body["user_token"]))
	if userToken == "" {
		h.recordSecurityAudit(c, "oauth_login", "mobile_token", "failure", "missing_user_token", http.StatusBadRequest, nil, "user", "", nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth_mobile_token_missing", nil)
		return
	}
	profile, err := h.fetchOAuthUserInfo(c.Request.Context(), map[string]any{
		"access_token": userToken,
		"token_type":   firstNonEmpty(toString(body["token_type"]), "Bearer"),
	}, oauthStateEntry{})
	if err != nil {
		h.recordSecurityAudit(c, "oauth_login", "mobile_token", "failure", "invalid_user_token", http.StatusUnauthorized, nil, "user", "", nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.oauth_mobile_token_invalid", nil)
		return
	}
	oauthID, ok := int64FromAny(firstPresent(profile, "user_id", "sub", "id"))
	if !ok || oauthID <= 0 {
		h.recordSecurityAudit(c, "oauth_login", "mobile_token", "failure", "invalid_user_id", http.StatusBadRequest, nil, "user", "", nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "用户中心返回的用户ID无效", nil)
		return
	}
	username := toString(profile["username"])
	email := toString(profile["email"])
	user, isNew, ok := h.findOrCreateOAuthUser(c, oauthID, username, email)
	if !ok {
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	h.recordSecurityAudit(c, "oauth_login", "mobile_token", "success", "", http.StatusOK, &user.ID, "user", user.UserID, gin.H{"is_new_user": isNew})
	writeSuccess(c, matrixMsgOK, gin.H{"access_token": access, "refresh_token": refresh, "is_new_user": isNew, "user": h.userPublicMap(*user)})
}

func (h NativeHandlers) authFileToken(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok || h.Auth == nil {
		return
	}
	payload := gin.H{}
	if user.Type == "admin" {
		payload["adminId"] = user.ID
		payload["type"] = "admin"
	} else {
		payload["userId"] = user.ID
		payload["user_id"] = user.UserID
	}
	token, err := h.Auth.GenerateFileToken(payload)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	maxAge := int(services.FileTokenTTL.Seconds())
	secure := h.Config.Server.Env == "production"
	http.SetCookie(c.Writer, &http.Cookie{Name: "file_token", Value: token, Path: "/api/file", MaxAge: maxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
	http.SetCookie(c.Writer, &http.Cookie{Name: "file_token", Value: token, Path: "/api/pyvideo-api-proxy", MaxAge: maxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure})
	writeSuccess(c, "文件访问令牌生成成功", gin.H{"file_token": token, "expires_in": maxAge})
}
