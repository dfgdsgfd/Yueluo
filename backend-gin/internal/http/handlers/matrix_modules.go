package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) AuthMatrix(c *gin.Context) {
	path := c.Request.URL.Path
	method := matrixMethod(c)
	switch {
	case method == http.MethodGet && path == "/api/auth/auth-config":
		h.authConfig(c)
	case method == http.MethodGet && path == "/api/auth/email-config":
		h.authEmailConfig(c)
	case method == http.MethodGet && path == "/api/auth/captcha":
		h.authCaptcha(c)
	case method == http.MethodGet && path == "/api/auth/check-user-id":
		h.authCheckUserID(c)
	case method == http.MethodPost && path == "/api/auth/send-email-code":
		h.authSendEmailCode(c, false)
	case method == http.MethodPost && path == "/api/auth/send-reset-code":
		h.authSendEmailCode(c, true)
	case method == http.MethodPost && path == "/api/auth/verify-reset-code":
		h.authVerifyResetCode(c)
	case method == http.MethodPost && path == "/api/auth/reset-password":
		h.authResetPassword(c)
	case method == http.MethodPost && path == "/api/auth/bind-email":
		h.authBindEmail(c)
	case method == http.MethodDelete && path == "/api/auth/unbind-email":
		h.authUnbindEmail(c)
	case method == http.MethodPost && path == "/api/auth/register":
		h.authRegister(c)
	case method == http.MethodPost && path == "/api/auth/login":
		h.authLogin(c)
	case method == http.MethodPost && path == "/api/auth/refresh":
		h.authRefresh(c)
	case method == http.MethodPost && path == "/api/auth/token":
		h.authToken(c)
	case method == http.MethodPost && path == "/api/auth/logout":
		h.authLogout(c)
	case method == http.MethodGet && path == "/api/auth/me":
		h.authMe(c)
	case method == http.MethodPost && path == "/api/auth/admin/login":
		h.authAdminLogin(c)
	case method == http.MethodGet && path == "/api/auth/admin/me":
		h.authAdminMe(c)
	case strings.HasPrefix(path, "/api/auth/admin/admins"):
		h.authAdminAdmins(c)
	case method == http.MethodGet && path == "/api/auth/oauth2/login":
		h.authOAuthLogin(c)
	case method == http.MethodGet && path == "/api/auth/oauth2/callback":
		h.authOAuthCallback(c)
	case method == http.MethodPost && path == "/api/auth/oauth2/mobile-token":
		h.authOAuthMobileToken(c)
	case method == http.MethodPost && path == "/api/auth/oauth2/app-token":
		h.authOAuthAppToken(c)
	case method == http.MethodPost && path == "/api/auth/oauth2/mobile-session":
		h.authOAuthMobileSession(c)
	case method == http.MethodPost && path == "/api/auth/file-token":
		h.authFileToken(c)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "auth route not found", nil)
	}
}

func (h NativeHandlers) UsersMatrix(c *gin.Context) {
	h.usersDispatch(c)
}

func (h NativeHandlers) IMMatrix(c *gin.Context) {
	h.imDispatch(c)
}

func (h NativeHandlers) AdminMatrix(c *gin.Context) {
	h.adminDispatch(c)
}

func (h NativeHandlers) AppMatrix(c *gin.Context) {
	h.appDispatch(c)
}
