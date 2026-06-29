package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) MatrixRoute(sourceFile string, method string, path string, authClass string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Gin-Migration-Mode", "native")
		c.Set("matrix_materialized_path", path)
		c.Set("auth_class", authClass)
		if !h.enforceMatrixAuth(c, authClass) {
			return
		}
		switch {
		case strings.HasSuffix(sourceFile, "auth.js"):
			h.AuthMatrix(c)
		case strings.HasSuffix(sourceFile, "users.js"):
			h.UsersMatrix(c)
		case strings.HasSuffix(sourceFile, "im.js"):
			h.IMMatrix(c)
		case strings.HasSuffix(sourceFile, "admin.js"):
			h.AdminMatrix(c)
		case strings.HasSuffix(sourceFile, "app.js"):
			h.AppMatrix(c)
		case strings.HasSuffix(sourceFile, "file.js"):
			h.FileAccess(c)
		case strings.HasSuffix(sourceFile, "pyvideoProxy.js"):
			h.PyVideoProxy(c)
		default:
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "route source not registered", nil)
		}
	}
}
