package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

type NativeHandlers struct {
	DB           *gorm.DB
	Config       config.Config
	Cache        *services.Cache
	Redis        *services.RedisStore
	APIKeys      *APIKeyAuthCache
	UploadPaths  *UploadPathResolver
	Queue        *services.QueueService
	Settings     *services.SettingsService
	AI           *services.AIService
	Images       *services.ImageProcessor
	Auth         *services.AuthService
	Observe      *services.ObservabilityService
	AuditLog     *services.AuditLogService
	AccessBlock  *services.AccessBlockService
	GeoIP        *services.GeoIPService
	Balance      *services.BalanceCenterService
	TempStorage  *services.TempStorageService
	UploadAssets *services.UploadAssetService
	FileRecycle  *services.FileRecycleService
	RedisCare    *services.RedisMaintenanceService

	FileAccessGroup *singleflight.Group
	PostListGroup   *singleflight.Group
	SearchGroup     *singleflight.Group
}

func (h NativeHandlers) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Auth == nil {
			response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
			c.Abort()
			return
		}
		user, failure := h.Auth.Authenticate(c.Request.Context(), h.authAccessTokenFromRequest(c))
		if failure != nil {
			response.JSON(c, failure.Status, failure.Code, failure.Message, nil)
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func markAccessOperation(c *gin.Context, behavior string, targetType string, targetID int64) {
	if c == nil {
		return
	}
	c.Set("access_behavior", behavior)
	if targetType != "" {
		c.Set("access_target_type", targetType)
	}
	if targetID > 0 {
		c.Set("access_target_id", targetID)
	}
}

func (h NativeHandlers) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Auth == nil {
			response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
			c.Abort()
			return
		}
		user, failure := h.Auth.Authenticate(c.Request.Context(), h.authAccessTokenFromRequest(c))
		if failure != nil {
			response.JSON(c, failure.Status, failure.Code, failure.Message, nil)
			c.Abort()
			return
		}
		if user.Type != "admin" {
			response.JSON(c, http.StatusForbidden, response.CodeForbidden, "权限不足", nil)
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func (h NativeHandlers) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Auth != nil {
			user := h.Auth.Optional(c.Request.Context(), h.authAccessTokenFromRequest(c))
			if user != nil {
				c.Set("user", user)
			}
		}
		c.Next()
	}
}

func (h NativeHandlers) OptionalAuthWithNoteGuestRestriction() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Auth != nil {
			user := h.Auth.Optional(c.Request.Context(), h.authAccessTokenFromRequest(c))
			if user != nil {
				c.Set("user", user)
			}
		}
		if h.Settings != nil && h.Settings.IsNoteGuestRestricted() {
			if _, ok := c.Get("user"); !ok {
				if h.allowRestrictedGuestAccess(c) {
					c.Next()
					return
				}
				response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "请登录后查看内容", nil)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func (h NativeHandlers) OptionalAuthWithVideoGuestRestriction() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.Auth != nil {
			user := h.Auth.Optional(c.Request.Context(), h.authAccessTokenFromRequest(c))
			if user != nil {
				c.Set("user", user)
			}
		}
		if h.Settings != nil && h.Settings.IsVideoGuestRestricted() {
			if _, ok := c.Get("user"); !ok {
				response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "请登录后查看内容", nil)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func (h NativeHandlers) RequireOpenAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "API密钥缺失，请在 X-API-Key 请求头中提供密钥", nil)
			c.Abort()
			return
		}
		if h.DB == nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "认证处理失败", nil)
			c.Abort()
			return
		}
		if h.apiKeyRequestBlocked(c, openAPIKeyScope) {
			c.Abort()
			return
		}
		openAPI, err := h.lookupOpenAPIKey(c.Request.Context(), apiKey)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.rejectInvalidAPIKey(c, openAPIKeyScope, "API密钥无效或已禁用")
			c.Abort()
			return
		}
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "认证处理失败", nil)
			c.Abort()
			return
		}
		h.acceptAPIKey(c, openAPIKeyScope)
		h.touchOpenAPIKey(openAPI.ID)
		c.Set("open_api", openAPI)
		c.Next()
	}
}
